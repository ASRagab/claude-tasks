package executor

import (
	"context"
	"errors"
	"github.com/ASRagab/claude-tasks/internal/db"
	"github.com/ASRagab/claude-tasks/internal/testutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestGenerateUUIDFormat(t *testing.T) {
	id, err := generateUUID()
	if err != nil {
		t.Fatalf("generate UUID: %v", err)
	}
	if !uuidRE.MatchString(id) {
		t.Fatalf("expected UUIDv4 format, got %q", id)
	}
}

func TestGenerateUUIDUniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 200)
	for i := 0; i < 200; i++ {
		id, err := generateUUID()
		if err != nil {
			t.Fatalf("generate UUID: %v", err)
		}
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate UUID generated: %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestCappedBufferTruncatesOutputAndAppendsMarker(t *testing.T) {
	buf := newCappedBuffer(5)

	n, err := buf.Write([]byte("hello world"))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if n != len("hello world") {
		t.Fatalf("expected write count %d, got %d", len("hello world"), n)
	}

	if got := buf.String(); got != "hello\n...[truncated]" {
		t.Fatalf("expected truncated output marker, got %q", got)
	}
}

func TestCappedBufferWithoutTruncationPreservesOutput(t *testing.T) {
	buf := newCappedBuffer(64)
	payload := "small output"

	n, err := buf.Write([]byte(payload))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if n != len(payload) {
		t.Fatalf("expected write count %d, got %d", len(payload), n)
	}
	if got := buf.String(); got != payload {
		t.Fatalf("expected %q, got %q", payload, got)
	}
}

func TestCappedBufferZeroLimitAlwaysTruncates(t *testing.T) {
	buf := newCappedBuffer(0)

	_, err := buf.Write([]byte("abc"))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if got := buf.String(); !strings.Contains(got, "...[truncated]") {
		t.Fatalf("expected truncation marker, got %q", got)
	}
}

func createTaskForExecutorTest(t *testing.T, database *db.DB, workingDir string) *db.Task {
	t.Helper()

	task := &db.Task{
		Name:           "executor-test",
		Prompt:         "echo test",
		CronExpr:       "0 * * * * *",
		WorkingDir:     workingDir,
		Model:          "",
		PermissionMode: db.DefaultPermissionMode,
		Enabled:        true,
	}

	if err := database.CreateTask(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	return task
}

func installFakeClaude(t *testing.T) {
	t.Helper()

	binDir := t.TempDir()
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+originalPath)

	var (
		binaryName string
		content    string
	)
	if runtime.GOOS == "windows" {
		binaryName = "claude.bat"
		content = "@echo off\r\nexit /b 0\r\n"
	} else {
		binaryName = "claude"
		content = "#!/bin/sh\nexit 0\n"
	}

	binaryPath := filepath.Join(binDir, binaryName)
	if err := os.WriteFile(binaryPath, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake claude binary: %v", err)
	}
}

func TestExecuteFailsClosedWhenUsageCheckUnavailable(t *testing.T) {
	database, _ := testutil.NewTestDB(t)
	task := createTaskForExecutorTest(t, database, t.TempDir())

	exec := &Executor{
		db:             database,
		usageClientErr: errors.New("credentials not found"),
	}

	result := exec.Execute(context.Background(), task)
	if result == nil || result.Error == nil {
		t.Fatalf("expected usage enforcement error, got %#v", result)
	}
	if !strings.Contains(result.Error.Error(), "usage threshold enforcement unavailable") {
		t.Fatalf("expected usage enforcement error, got %v", result.Error)
	}

	runs, err := database.GetTaskRuns(task.ID, 10)
	if err != nil {
		t.Fatalf("get task runs: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("expected no run records when usage check fails preflight, got %d", len(runs))
	}
}

func TestExecuteAllowsRunWhenUsageCheckDisabled(t *testing.T) {
	t.Setenv("CLAUDE_TASKS_DISABLE_USAGE_CHECK", "1")

	database, dataDir := testutil.NewTestDB(t)
	missingDir := filepath.Join(t.TempDir(), "missing")
	task := createTaskForExecutorTest(t, database, missingDir)

	installFakeClaude(t)

	exec := New(database, dataDir)
	if !exec.disableUsageCheck {
		t.Fatalf("expected usage checks to be disabled via env var")
	}

	result := exec.Execute(context.Background(), task)
	if result == nil || result.Error == nil {
		t.Fatalf("expected command execution failure for missing working directory")
	}
	if strings.Contains(result.Error.Error(), "usage threshold enforcement unavailable") {
		t.Fatalf("expected usage checks to be bypassed when disabled, got %v", result.Error)
	}
	if !strings.Contains(result.Error.Error(), "no such file or directory") {
		t.Fatalf("expected missing directory error, got %v", result.Error)
	}

	runs, err := database.GetTaskRuns(task.ID, 10)
	if err != nil {
		t.Fatalf("get task runs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected one run record when execution proceeds, got %d", len(runs))
	}
	if runs[0].Status != db.RunStatusFailed {
		t.Fatalf("expected failed status, got %s", runs[0].Status)
	}
}
