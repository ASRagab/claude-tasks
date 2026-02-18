package scheduler

import (
	"testing"
	"time"

	"github.com/ASRagab/claude-tasks/internal/db"
	"github.com/ASRagab/claude-tasks/internal/testutil"
)

func TestStartStopIdempotent(t *testing.T) {
	database, dataDir := testutil.NewTestDB(t)
	s := New(database, dataDir)

	if err := s.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}

	// second start should be no-op
	if err := s.Start(); err != nil {
		t.Fatalf("second start scheduler: %v", err)
	}

	s.Stop()
	// second stop should be no-op
	s.Stop()
}

func TestRestartRecreatesSyncChannel(t *testing.T) {
	database, dataDir := testutil.NewTestDB(t)
	s := New(database, dataDir)

	if err := s.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	firstStopSync := s.stopSync
	if channelClosed(firstStopSync) {
		t.Fatalf("expected first stop channel to be open")
	}

	s.Stop()
	if !channelClosed(firstStopSync) {
		t.Fatalf("expected first stop channel to be closed after stop")
	}

	if err := s.Start(); err != nil {
		t.Fatalf("restart scheduler: %v", err)
	}
	defer s.Stop()

	secondStopSync := s.stopSync
	if firstStopSync == secondStopSync {
		t.Fatalf("expected new sync stop channel on restart")
	}
	if channelClosed(secondStopSync) {
		t.Fatalf("expected restarted sync stop channel to be open")
	}
}

func TestAddTaskReturnsErrorWhenPersistingNextRunFails(t *testing.T) {
	database, dataDir := testutil.NewTestDB(t)
	s := New(database, dataDir)

	if err := s.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	defer s.Stop()

	task := &db.Task{
		Name:       "recurring",
		Prompt:     "echo hi",
		CronExpr:   "0 * * * * *",
		WorkingDir: ".",
		Enabled:    true,
	}
	if err := database.CreateTask(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := database.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	if err := s.AddTask(task); err == nil {
		t.Fatalf("expected add task to fail when db update fails")
	}
}


func TestAddAndRemoveOneOffTask(t *testing.T) {
	database, dataDir := testutil.NewTestDB(t)
	s := New(database, dataDir)

	if err := s.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	defer s.Stop()

	task := &db.Task{
		Name:       "one-off",
		Prompt:     "echo hi",
		CronExpr:   "",
		WorkingDir: ".",
		Enabled:    true,
	}
	if err := database.CreateTask(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	future := time.Now().Add(2 * time.Minute)
	task.ScheduledAt = &future
	if err := database.UpdateTask(task); err != nil {
		t.Fatalf("update task scheduled_at: %v", err)
	}

	if err := s.AddTask(task); err != nil {
		t.Fatalf("add task: %v", err)
	}

	if got := s.GetNextRunTime(task.ID); got == nil {
		t.Fatalf("expected next run time for one-off task")
	}

	s.RemoveTask(task.ID)
	if got := s.GetNextRunTime(task.ID); got != nil {
		t.Fatalf("expected no next run time after remove, got %v", got)
	}
}

func channelClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}
