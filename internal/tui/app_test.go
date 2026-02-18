package tui

import (
	"testing"
	"time"

	"github.com/ASRagab/claude-tasks/internal/db"
	"github.com/ASRagab/claude-tasks/internal/testutil"
)

func newTestModel(t *testing.T) Model {
	t.Helper()
	database, dataDir := testutil.NewTestDB(t)
	return NewModel(database, nil, true, dataDir)
}

func TestPollRefreshGuardQueuesWhenInFlight(t *testing.T) {
	m := newTestModel(t)
	m.refreshInFlight = false
	m.refreshPending = false

	if cmd := m.requestTaskRefresh(); cmd == nil {
		t.Fatalf("expected refresh command on first request")
	}
	if !m.refreshInFlight {
		t.Fatalf("expected refreshInFlight to be true after first request")
	}

	if cmd := m.requestTaskRefresh(); cmd != nil {
		t.Fatalf("expected no refresh command while request is in flight")
	}
	if !m.refreshPending {
		t.Fatalf("expected refreshPending to be true while request is in flight")
	}
}

func TestPollPendingRefreshIsScheduledAfterLoadCompletes(t *testing.T) {
	m := newTestModel(t)
	m.refreshInFlight = true
	m.refreshPending = true

	updatedModel, _ := m.Update(tasksLoadedMsg{
		tasks:    []*db.Task{},
		running:  map[int64]bool{},
		nextRuns: map[int64]time.Time{},
		statuses: map[int64]db.RunStatus{},
	})
	updated := updatedModel.(Model)

	if updated.refreshPending {
		t.Fatalf("expected pending refresh to be cleared")
	}
	if !updated.refreshInFlight {
		t.Fatalf("expected follow-up refresh to be scheduled")
	}
}

func TestPollUsageGuardQueuesWhenInFlight(t *testing.T) {
	m := newTestModel(t)
	m.usageInFlight = false
	m.usagePending = false

	if cmd := m.requestUsageRefresh(); cmd == nil {
		t.Fatalf("expected usage refresh command on first request")
	}
	if !m.usageInFlight {
		t.Fatalf("expected usageInFlight to be true after first request")
	}

	if cmd := m.requestUsageRefresh(); cmd != nil {
		t.Fatalf("expected no usage refresh command while request is in flight")
	}
	if !m.usagePending {
		t.Fatalf("expected usagePending to be true while request is in flight")
	}

	updatedModel, _ := m.Update(usageUpdatedMsg{data: nil, err: nil})
	updated := updatedModel.(Model)
	if updated.usagePending {
		t.Fatalf("expected usagePending to be cleared after usage update")
	}
	if !updated.usageInFlight {
		t.Fatalf("expected queued usage refresh to be scheduled")
	}
}

func TestStatusBatchLoadDerivesRunningTasksWithoutPerTaskQueries(t *testing.T) {
	database, dataDir := testutil.NewTestDB(t)
	m := NewModel(database, nil, true, dataDir)

	taskRunning := &db.Task{
		Name:       "running",
		Prompt:     "echo run",
		CronExpr:   "0 * * * * *",
		WorkingDir: ".",
		Enabled:    true,
	}
	if err := database.CreateTask(taskRunning); err != nil {
		t.Fatalf("create running task: %v", err)
	}

	nextRun := time.Now().Add(5 * time.Minute).UTC().Truncate(time.Second)
	taskRunning.NextRunAt = &nextRun
	if err := database.UpdateTask(taskRunning); err != nil {
		t.Fatalf("update running task next run: %v", err)
	}

	taskCompleted := &db.Task{
		Name:       "completed",
		Prompt:     "echo done",
		CronExpr:   "0 * * * * *",
		WorkingDir: ".",
		Enabled:    true,
	}
	if err := database.CreateTask(taskCompleted); err != nil {
		t.Fatalf("create completed task: %v", err)
	}

	started := time.Now().UTC().Truncate(time.Second)
	ended := started.Add(2 * time.Second)

	if err := database.CreateTaskRun(&db.TaskRun{
		TaskID:    taskRunning.ID,
		StartedAt: started,
		Status:    db.RunStatusRunning,
		Output:    "",
		Error:     "",
	}); err != nil {
		t.Fatalf("create running task run: %v", err)
	}

	if err := database.CreateTaskRun(&db.TaskRun{
		TaskID:    taskCompleted.ID,
		StartedAt: started,
		EndedAt:   &ended,
		Status:    db.RunStatusCompleted,
		Output:    "done",
		Error:     "",
	}); err != nil {
		t.Fatalf("create completed task run: %v", err)
	}

	msg := m.loadTasks()().(tasksLoadedMsg)
	if msg.err != nil {
		t.Fatalf("unexpected load error: %v", msg.err)
	}

	if len(msg.tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(msg.tasks))
	}
	if !msg.running[taskRunning.ID] {
		t.Fatalf("expected running task %d to be marked running", taskRunning.ID)
	}
	if msg.running[taskCompleted.ID] {
		t.Fatalf("expected completed task %d to not be marked running", taskCompleted.ID)
	}
	if msg.statuses[taskRunning.ID] != db.RunStatusRunning {
		t.Fatalf("expected running status for task %d", taskRunning.ID)
	}
	if msg.statuses[taskCompleted.ID] != db.RunStatusCompleted {
		t.Fatalf("expected completed status for task %d", taskCompleted.ID)
	}
	if got, ok := msg.nextRuns[taskRunning.ID]; !ok || !got.Equal(nextRun) {
		t.Fatalf("expected next run %v for task %d, got %v (present=%v)", nextRun, taskRunning.ID, got, ok)
	}
}
