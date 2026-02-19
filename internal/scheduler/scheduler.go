package scheduler

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ASRagab/claude-tasks/internal/db"
	"github.com/ASRagab/claude-tasks/internal/executor"
	"github.com/robfig/cron/v3"
)

// Scheduler manages cron jobs for tasks
type Scheduler struct {
	cron                *cron.Cron
	db                  *db.DB
	executor            *executor.Executor
	jobs                map[int64]cron.EntryID
	cronExprs           map[int64]string      // Track cron expressions to detect changes
	oneOffTimers        map[int64]*time.Timer // Track one-off task timers
	mu                  sync.RWMutex
	running             bool
	stopSync            chan struct{}
	syncDone            chan struct{}
	leaseHolderID       string
	leaseTTL            time.Duration
	leaseRenewInterval  time.Duration
	schedulerLeadership bool
}

// New creates a new scheduler
func New(database *db.DB, dataDir string) *Scheduler {
	return &Scheduler{
		cron:                cron.New(cron.WithSeconds()),
		db:                  database,
		executor:            executor.New(database, dataDir),
		jobs:                make(map[int64]cron.EntryID),
		cronExprs:           make(map[int64]string),
		oneOffTimers:        make(map[int64]*time.Timer),
		stopSync:            make(chan struct{}),
		leaseHolderID:       fmt.Sprintf("scheduler-%d-%d", os.Getpid(), time.Now().UnixNano()),
		leaseTTL:            15 * time.Second,
		leaseRenewInterval:  5 * time.Second,
		schedulerLeadership: false,
	}
}

// Start starts the scheduler and leadership maintenance loops.
func (s *Scheduler) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}

	s.stopSync = make(chan struct{})
	s.syncDone = make(chan struct{})
	s.running = true
	s.cron.Start()
	stopSync := s.stopSync
	syncDone := s.syncDone
	s.mu.Unlock()

	s.refreshLeadership()
	s.SyncTasks()

	// Start background loop to maintain leadership and pick up DB changes.
	go s.syncLoop(stopSync, syncDone)

	return nil
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	wasLeader := s.schedulerLeadership
	holderID := s.leaseHolderID
	s.schedulerLeadership = false
	s.clearSchedulesLocked()

	stopSync := s.stopSync
	syncDone := s.syncDone
	s.stopSync = nil
	s.syncDone = nil
	s.mu.Unlock()

	// Stop sync loop
	if stopSync != nil {
		close(stopSync)
	}
	if syncDone != nil {
		<-syncDone
	}

	if wasLeader {
		if err := s.db.ReleaseSchedulerLease(holderID); err != nil {
			fmt.Printf("Failed to release scheduler lease: %v\n", err)
		}
	}

	ctx := s.cron.Stop()
	<-ctx.Done()
}

// AddTask schedules a new task when this process is the current scheduler leader.
func (s *Scheduler) AddTask(task *db.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.schedulerLeadership {
		return nil
	}
	return s.scheduleTaskLocked(task)
}

// RemoveTask removes a task from the local scheduler state.
func (s *Scheduler) RemoveTask(taskID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.removeTaskLocked(taskID)
}

// UpdateTask updates a task's schedule
func (s *Scheduler) UpdateTask(task *db.Task) error {
	s.RemoveTask(task.ID)
	if task.Enabled {
		return s.AddTask(task)
	}
	return nil
}

// IsLeader reports whether this scheduler currently owns the scheduling lease.
func (s *Scheduler) IsLeader() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.schedulerLeadership
}

// GetNextRunTime returns the next scheduled run time for a task
func (s *Scheduler) GetNextRunTime(taskID int64) *time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check cron jobs
	if entryID, ok := s.jobs[taskID]; ok {
		entry := s.cron.Entry(entryID)
		if !entry.Next.IsZero() {
			return &entry.Next
		}
	}

	// Check one-off tasks (return from DB since timer doesn't expose time)
	if _, ok := s.oneOffTimers[taskID]; ok {
		task, err := s.db.GetTask(taskID)
		if err == nil && task.NextRunAt != nil {
			return task.NextRunAt
		}
	}

	return nil
}

// GetAllNextRunTimes returns next run times for all scheduled tasks
func (s *Scheduler) GetAllNextRunTimes() map[int64]time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[int64]time.Time)

	// Get cron job next runs
	for taskID, entryID := range s.jobs {
		entry := s.cron.Entry(entryID)
		if !entry.Next.IsZero() {
			result[taskID] = entry.Next
		}
	}

	// Get one-off task next runs from DB
	for taskID := range s.oneOffTimers {
		task, err := s.db.GetTask(taskID)
		if err == nil && task.NextRunAt != nil {
			result[taskID] = *task.NextRunAt
		}
	}

	return result
}

func (s *Scheduler) scheduleTaskLocked(task *db.Task) error {
	// Route one-off tasks to separate handler
	if task.IsOneOff() {
		return s.scheduleOneOffTaskLocked(task)
	}

	// Remove existing job if any
	if entryID, ok := s.jobs[task.ID]; ok {
		s.cron.Remove(entryID)
		delete(s.jobs, task.ID)
	}

	// Create a copy of task ID for the closure
	taskID := task.ID

	entryID, err := s.cron.AddFunc(task.CronExpr, func() {
		s.mu.RLock()
		isLeader := s.schedulerLeadership
		s.mu.RUnlock()
		if !isLeader {
			return
		}

		// Get fresh task data from DB
		freshTask, err := s.db.GetTask(taskID)
		if err != nil {
			fmt.Printf("Failed to get task %d: %v\n", taskID, err)
			return
		}
		if !freshTask.Enabled {
			return
		}
		go func(runTask *db.Task) {
			result := <-s.executor.ExecuteAsync(runTask)
			if result != nil && result.Error != nil {
				fmt.Printf("Failed to execute task %d: %v\n", taskID, result.Error)
			}
		}(freshTask)

		// Update next run time in DB after execution
		s.mu.RLock()
		if eid, ok := s.jobs[taskID]; ok {
			entry := s.cron.Entry(eid)
			if !entry.Next.IsZero() {
				freshTask.NextRunAt = &entry.Next
				if err := s.db.UpdateTask(freshTask); err != nil {
					fmt.Printf("Failed to update task %d next run time: %v\n", taskID, err)
				}
			}
		}
		s.mu.RUnlock()
	})
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	s.jobs[task.ID] = entryID
	s.cronExprs[task.ID] = task.CronExpr

	// Update next run time in DB
	entry := s.cron.Entry(entryID)
	if !entry.Next.IsZero() {
		task.NextRunAt = &entry.Next
		if err := s.db.UpdateTask(task); err != nil {
			s.cron.Remove(entryID)
			delete(s.jobs, task.ID)
			delete(s.cronExprs, task.ID)
			return fmt.Errorf("failed to persist next run time: %w", err)
		}
	}

	return nil
}

// scheduleOneOffTaskLocked schedules a one-off task
func (s *Scheduler) scheduleOneOffTaskLocked(task *db.Task) error {
	// Cancel existing timer if any
	if timer, ok := s.oneOffTimers[task.ID]; ok {
		timer.Stop()
		delete(s.oneOffTimers, task.ID)
	}

	taskID := task.ID

	// If no scheduled time, run immediately
	if task.ScheduledAt == nil {
		go s.executeOneOff(taskID)
		return nil
	}

	delay := time.Until(*task.ScheduledAt)
	if delay <= 0 {
		// Scheduled time has passed, run immediately
		go s.executeOneOff(taskID)
		return nil
	}

	// Schedule for future execution
	timer := time.AfterFunc(delay, func() {
		s.executeOneOff(taskID)
	})
	s.oneOffTimers[task.ID] = timer

	// Update NextRunAt in DB
	task.NextRunAt = task.ScheduledAt
	if err := s.db.UpdateTask(task); err != nil {
		timer.Stop()
		delete(s.oneOffTimers, task.ID)
		return fmt.Errorf("failed to persist one-off next run time: %w", err)
	}

	return nil
}

// executeOneOff runs a one-off task and disables it afterward
func (s *Scheduler) executeOneOff(taskID int64) {
	defer func() {
		s.mu.Lock()
		delete(s.oneOffTimers, taskID)
		s.mu.Unlock()
	}()

	s.mu.RLock()
	isLeader := s.schedulerLeadership
	s.mu.RUnlock()
	if !isLeader {
		return
	}

	// Get fresh task data
	task, err := s.db.GetTask(taskID)
	if err != nil {
		fmt.Printf("Failed to get one-off task %d: %v\n", taskID, err)
		return
	}
	if !task.Enabled {
		return
	}

	// Execute the task
	result := <-s.executor.ExecuteAsync(task)
	if result != nil && result.Error != nil {
		fmt.Printf("Failed to execute one-off task %d: %v\n", taskID, result.Error)
	}

	// Auto-disable the task after execution
	task.Enabled = false
	task.NextRunAt = nil
	if err := s.db.UpdateTask(task); err != nil {
		fmt.Printf("Failed to disable one-off task %d after execution: %v\n", taskID, err)
	}

}

// RunTaskNow executes a task immediately
func (s *Scheduler) RunTaskNow(taskID int64) error {
	task, err := s.db.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	go func() {
		result := <-s.executor.ExecuteAsync(task)
		if result != nil && result.Error != nil {
			fmt.Printf("Failed to execute task %d: %v\n", taskID, result.Error)
		}
	}()

	return nil
}

// syncLoop periodically renews leadership and syncs tasks from DB.
func (s *Scheduler) syncLoop(stopSync <-chan struct{}, syncDone chan<- struct{}) {
	leadershipTicker := time.NewTicker(s.leaseRenewInterval)
	syncTicker := time.NewTicker(10 * time.Second)
	defer leadershipTicker.Stop()
	defer syncTicker.Stop()
	defer close(syncDone)

	for {
		select {
		case <-stopSync:
			return
		case <-leadershipTicker.C:
			s.refreshLeadership()
		case <-syncTicker.C:
			s.refreshLeadership()
			s.SyncTasks()
		}
	}
}

func (s *Scheduler) refreshLeadership() {
	acquired, lease, err := s.db.TryAcquireSchedulerLease(s.leaseHolderID, s.leaseTTL)
	if err != nil {
		fmt.Printf("Failed to refresh scheduler lease: %v\n", err)
		return
	}

	var gainedLeadership bool
	var lostLeadership bool
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}

	switch {
	case acquired && !s.schedulerLeadership:
		s.schedulerLeadership = true
		gainedLeadership = true
	case !acquired && s.schedulerLeadership:
		s.schedulerLeadership = false
		s.clearSchedulesLocked()
		lostLeadership = true
	}
	s.mu.Unlock()

	if gainedLeadership {
		fmt.Printf("Scheduler leadership acquired: holder=%s\n", s.leaseHolderID)
		s.SyncTasks()
		return
	}
	if lostLeadership {
		holder := "unknown"
		if lease != nil && lease.HolderID != "" {
			holder = lease.HolderID
		}
		fmt.Printf("Scheduler leadership lost: holder=%s active_holder=%s\n", s.leaseHolderID, holder)
	}
}

// SyncTasks reloads tasks from DB and updates scheduler.
func (s *Scheduler) SyncTasks() {
	s.mu.RLock()
	isLeader := s.schedulerLeadership
	s.mu.RUnlock()
	if !isLeader {
		return
	}

	tasks, err := s.db.ListTasks()
	if err != nil {
		fmt.Printf("Failed to sync tasks from DB: %v\n", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.schedulerLeadership {
		return
	}

	// Build set of current task IDs in DB.
	dbTaskIDs := make(map[int64]bool)
	for _, task := range tasks {
		dbTaskIDs[task.ID] = true
	}

	// Remove cron jobs for tasks that no longer exist.
	for taskID := range s.jobs {
		if !dbTaskIDs[taskID] {
			s.removeTaskLocked(taskID)
		}
	}

	// Remove one-off timers for tasks that no longer exist.
	for taskID := range s.oneOffTimers {
		if !dbTaskIDs[taskID] {
			s.removeTaskLocked(taskID)
		}
	}

	// Add/update tasks.
	for _, task := range tasks {
		_, hasCronJob := s.jobs[task.ID]
		_, hasOneOffTimer := s.oneOffTimers[task.ID]
		isScheduled := hasCronJob || hasOneOffTimer
		oldCronExpr := s.cronExprs[task.ID]

		if task.Enabled && !isScheduled {
			// Task should be scheduled but isn't.
			if err := s.scheduleTaskLocked(task); err != nil {
				fmt.Printf("Failed to schedule task %d during sync: %v\n", task.ID, err)
			}
		} else if !task.Enabled && isScheduled {
			// Task shouldn't be scheduled but is - remove it.
			s.removeTaskLocked(task.ID)
		} else if task.Enabled && hasCronJob && task.IsOneOff() {
			// Task was converted from recurring to one-off, reschedule.
			s.removeTaskLocked(task.ID)
			if err := s.scheduleTaskLocked(task); err != nil {
				fmt.Printf("Failed to reschedule task %d during sync: %v\n", task.ID, err)
			}
		} else if task.Enabled && hasOneOffTimer && !task.IsOneOff() {
			// Task was converted from one-off to recurring, reschedule.
			s.removeTaskLocked(task.ID)
			if err := s.scheduleTaskLocked(task); err != nil {
				fmt.Printf("Failed to reschedule task %d during sync: %v\n", task.ID, err)
			}
		} else if task.Enabled && hasCronJob && task.CronExpr != oldCronExpr {
			// Cron expression changed, reschedule.
			if err := s.scheduleTaskLocked(task); err != nil {
				fmt.Printf("Failed to reschedule task %d during sync: %v\n", task.ID, err)
			}
		}
	}
}

func (s *Scheduler) removeTaskLocked(taskID int64) {
	if entryID, ok := s.jobs[taskID]; ok {
		s.cron.Remove(entryID)
		delete(s.jobs, taskID)
		delete(s.cronExprs, taskID)
	}

	if timer, ok := s.oneOffTimers[taskID]; ok {
		timer.Stop()
		delete(s.oneOffTimers, taskID)
	}
}

func (s *Scheduler) clearSchedulesLocked() {
	for taskID := range s.jobs {
		s.removeTaskLocked(taskID)
	}
	for taskID := range s.oneOffTimers {
		s.removeTaskLocked(taskID)
	}
}
