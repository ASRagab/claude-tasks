package executor

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ASRagab/claude-tasks/internal/db"
	"github.com/ASRagab/claude-tasks/internal/logger"
	"github.com/ASRagab/claude-tasks/internal/usage"
	"github.com/ASRagab/claude-tasks/internal/webhook"
)

// Executor runs Claude CLI tasks
type Executor struct {
	db                *db.DB
	logger            *logger.RunLogger
	discord           *webhook.Discord
	slack             *webhook.Slack
	usageClient       *usage.Client
	usageClientErr    error
	disableUsageCheck bool
}

const maxCapturedOutputBytes = 256 * 1024

type cappedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func newCappedBuffer(limit int) *cappedBuffer {
	return &cappedBuffer{limit: limit}
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	if c.limit <= 0 {
		c.truncated = len(p) > 0
		return len(p), nil
	}

	remaining := c.limit - c.buf.Len()
	if remaining <= 0 {
		c.truncated = len(p) > 0 || c.truncated
		return len(p), nil
	}

	if len(p) > remaining {
		if _, err := c.buf.Write(p[:remaining]); err != nil {
			return 0, err
		}
		c.truncated = true
		return len(p), nil
	}

	return c.buf.Write(p)
}

func (c *cappedBuffer) String() string {
	content := c.buf.String()
	if !c.truncated {
		return content
	}
	if content == "" {
		return "...[truncated]"
	}
	return content + "\n...[truncated]"
}

// New creates a new executor
func New(database *db.DB, dataDir string) *Executor {
	disableUsageCheck := isTruthy(strings.TrimSpace(os.Getenv("CLAUDE_TASKS_DISABLE_USAGE_CHECK")))

	var usageClient *usage.Client
	var usageClientErr error
	if !disableUsageCheck {
		usageClient, usageClientErr = usage.NewClient()
	}

	return &Executor{
		db:                database,
		logger:            logger.New(dataDir),
		discord:           webhook.NewDiscord(),
		slack:             webhook.NewSlack(),
		usageClient:       usageClient,
		usageClientErr:    usageClientErr,
		disableUsageCheck: disableUsageCheck,
	}
}

// Result represents the result of a task execution
type Result struct {
	Output     string
	Error      error
	Duration   time.Duration
	Skipped    bool
	SkipReason string
}

func generateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func isTruthy(value string) bool {
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func (e *Executor) failPreflight(task *db.Task, startedAt time.Time, preflightErr error) *Result {
	endTime := time.Now()
	run := &db.TaskRun{
		TaskID:    task.ID,
		StartedAt: startedAt,
		EndedAt:   &endTime,
		Status:    db.RunStatusFailed,
		Error:     preflightErr.Error(),
	}

	if err := e.db.CreateTaskRun(run); err != nil {
		return &Result{
			Error:    errors.Join(preflightErr, fmt.Errorf("failed to create preflight run record: %w", err)),
			Duration: endTime.Sub(startedAt),
		}
	}

	var logErr error
	if e.logger != nil {
		logErr = e.logger.WriteRunLog(task, run)
	}

	return &Result{
		Error:    errors.Join(preflightErr, logErr),
		Duration: endTime.Sub(startedAt),
	}
}

// Execute runs a Claude CLI command for the given task
func (e *Executor) Execute(ctx context.Context, task *db.Task) *Result {
	startTime := time.Now()

	if !e.disableUsageCheck {
		// Check usage threshold before running
		if e.usageClient == nil {
			preflightErr := fmt.Errorf("usage threshold enforcement unavailable")
			if e.usageClientErr != nil {
				preflightErr = fmt.Errorf("usage threshold enforcement unavailable: %w", e.usageClientErr)
			}
			return e.failPreflight(task, startTime, preflightErr)
		}

		threshold, thresholdErr := e.db.GetUsageThreshold()
		if thresholdErr != nil {
			return e.failPreflight(task, startTime, fmt.Errorf("failed to enforce usage threshold: %w", thresholdErr))
		}

		ok, usageData, checkErr := e.usageClient.CheckThreshold(threshold)
		if checkErr != nil {
			return e.failPreflight(task, startTime, fmt.Errorf("failed to enforce usage threshold: %w", checkErr))
		}

		if !ok {
			// Usage is above threshold, skip the task
			skipReason := fmt.Sprintf("Usage above threshold (%.0f%%): 5h=%.0f%%, 7d=%.0f%%. Resets in %s",
				threshold,
				usageData.FiveHour.Utilization,
				usageData.SevenDay.Utilization,
				usageData.FormatTimeUntilReset())

			// Create a skipped run record
			run := &db.TaskRun{
				TaskID:    task.ID,
				StartedAt: startTime,
				Status:    db.RunStatusFailed,
				Error:     skipReason,
			}
			endTime := time.Now()
			run.EndedAt = &endTime
			if err := e.db.CreateTaskRun(run); err != nil {
				return &Result{Error: fmt.Errorf("failed to create skipped run record: %w", err)}
			}

			var logErr error
			if e.logger != nil {
				logErr = e.logger.WriteRunLog(task, run)
			}

			return &Result{
				Skipped:    true,
				SkipReason: skipReason,
				Duration:   time.Since(startTime),
				Error:      logErr,
			}
		}
	}

	// Generate session ID and build CLI args
	sessionID, err := generateUUID()
	if err != nil {
		return &Result{Error: err}
	}
	args := []string{"-p"}

	permMode := task.PermissionMode
	if permMode == "" {
		permMode = db.DefaultPermissionMode
	}
	if permMode == "bypassPermissions" {
		args = append(args, "--dangerously-skip-permissions")
	} else if permMode != "default" {
		args = append(args, "--permission-mode", permMode)
	}

	if task.Model != "" {
		args = append(args, "--model", task.Model)
	}

	args = append(args, "--session-id", sessionID)
	args = append(args, task.Prompt)

	// Create task run record
	run := &db.TaskRun{
		TaskID:    task.ID,
		StartedAt: startTime,
		Status:    db.RunStatusRunning,
		SessionID: sessionID,
	}
	if err := e.db.CreateTaskRun(run); err != nil {
		return &Result{Error: fmt.Errorf("failed to create run record: %w", err)}
	}

	// Build and execute command
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = task.WorkingDir

	stdout := newCappedBuffer(maxCapturedOutputBytes)
	stderr := newCappedBuffer(maxCapturedOutputBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	execErr := cmd.Run()
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Update run record
	run.EndedAt = &endTime
	run.Output = stdout.String()
	if execErr != nil {
		run.Status = db.RunStatusFailed
		run.Error = fmt.Sprintf("%s\n%s", execErr.Error(), stderr.String())
	} else {
		run.Status = db.RunStatusCompleted
	}

	var postRunErrs []error
	if err := e.db.UpdateTaskRun(run); err != nil {
		postRunErrs = append(postRunErrs, fmt.Errorf("failed to update run record: %w", err))
	}

	if e.logger != nil {
		if err := e.logger.WriteRunLog(task, run); err != nil {
			postRunErrs = append(postRunErrs, fmt.Errorf("failed to write run log: %w", err))
		}
	}

	// Update task's last run time
	task.LastRunAt = &endTime
	if err := e.db.UpdateTask(task); err != nil {
		postRunErrs = append(postRunErrs, fmt.Errorf("failed to update task last run time: %w", err))
	}

	// Send webhook notifications if configured
	if task.DiscordWebhook != "" {
		if err := e.discord.SendResult(task.DiscordWebhook, task, run); err != nil {
			postRunErrs = append(postRunErrs, fmt.Errorf("failed to send discord webhook: %w", err))
		}
	}
	if task.SlackWebhook != "" {
		if err := e.slack.SendResult(task.SlackWebhook, task, run); err != nil {
			postRunErrs = append(postRunErrs, fmt.Errorf("failed to send slack webhook: %w", err))
		}
	}

	result := &Result{
		Output:   stdout.String(),
		Duration: duration,
	}

	var resultErrs []error
	if execErr != nil {
		resultErrs = append(resultErrs, fmt.Errorf("%s: %s", execErr.Error(), stderr.String()))
	}
	resultErrs = append(resultErrs, postRunErrs...)
	result.Error = errors.Join(resultErrs...)

	return result
}

// ExecuteAsync runs a task asynchronously
func (e *Executor) ExecuteAsync(task *db.Task) <-chan *Result {
	ch := make(chan *Result, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		ch <- e.Execute(ctx, task)
		close(ch)
	}()
	return ch
}
