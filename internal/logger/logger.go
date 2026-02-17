package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ASRagab/claude-tasks/internal/db"
)

// RunLog is the structured JSON format written to log files
type RunLog struct {
	RunID      int64     `json:"run_id"`
	TaskID     int64     `json:"task_id"`
	TaskName   string    `json:"task_name"`
	Prompt     string    `json:"prompt"`
	WorkingDir string    `json:"working_dir"`
	CronExpr   string    `json:"cron_expr"`
	StartedAt  time.Time `json:"started_at"`
	EndedAt    time.Time `json:"ended_at"`
	DurationMs int64     `json:"duration_ms"`
	Status     string    `json:"status"`
	Output     string `json:"output"`
	Error      string `json:"error"`
	Model          string `json:"model,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
}

// RunLogger writes structured JSON log files for task runs
type RunLogger struct {
	baseDir string // e.g. ~/.claude-tasks/logs
}

// New creates a RunLogger that writes to dataDir/logs/
func New(dataDir string) *RunLogger {
	return &RunLogger{
		baseDir: filepath.Join(dataDir, "logs"),
	}
}

// WriteRunLog writes a structured JSON log file for a completed task run
func (l *RunLogger) WriteRunLog(task *db.Task, run *db.TaskRun) error {
	taskDir := filepath.Join(l.baseDir, fmt.Sprintf("%d", task.ID))
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	startedAt := run.StartedAt
	var endedAt time.Time
	if run.EndedAt != nil {
		endedAt = *run.EndedAt
	}
	var durationMs int64
	if !endedAt.IsZero() && !startedAt.IsZero() {
		durationMs = endedAt.Sub(startedAt).Milliseconds()
	}

	logEntry := RunLog{
		RunID:      run.ID,
		TaskID:     task.ID,
		TaskName:   task.Name,
		Prompt:     task.Prompt,
		WorkingDir: task.WorkingDir,
		CronExpr:   task.CronExpr,
		StartedAt:  startedAt,
		EndedAt:    endedAt,
		DurationMs: durationMs,
		Status:     string(run.Status),
		Output:         run.Output,
		Error:          run.Error,
		Model:          task.Model,
		PermissionMode: task.PermissionMode,
		SessionID:      run.SessionID,

	}

	data, err := json.MarshalIndent(logEntry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal log: %w", err)
	}

	timestamp := startedAt.Format("20060102T150405")
	filename := fmt.Sprintf("%d_%s_%s.json", run.ID, string(run.Status), timestamp)
	filePath := filepath.Join(taskDir, filename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("write log file: %w", err)
	}

	return nil
}
