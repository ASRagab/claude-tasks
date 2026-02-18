package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/ASRagab/claude-tasks/internal/db"
	"github.com/ASRagab/claude-tasks/internal/usage"
	"github.com/ASRagab/claude-tasks/internal/version"
	"github.com/go-chi/chi/v5"
	"github.com/robfig/cron/v3"
)

// HealthCheck handles GET /api/v1/health
func (s *Server) HealthCheck(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, HealthResponse{
		Status:  "ok",
		Version: version.Version,
	})
}

// ListTasks handles GET /api/v1/tasks
func (s *Server) ListTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.db.ListTasks()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to fetch tasks", err)
		return
	}

	// Get last run statuses for all tasks
	statuses, statusErr := s.db.GetLastRunStatuses()
	if statusErr != nil {
		log.Printf("api list tasks: failed to fetch last run statuses: %v", statusErr)
		statuses = make(map[int64]db.RunStatus)
	}

	response := TaskListResponse{
		Tasks: make([]TaskResponse, len(tasks)),
		Total: len(tasks),
	}

	for i, task := range tasks {
		response.Tasks[i] = s.taskToResponse(task, statuses[task.ID])
	}

	s.jsonResponse(w, http.StatusOK, response)
}

// CreateTask handles POST /api/v1/tasks
func (s *Server) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req TaskRequest
	if !s.decodeJSONBody(w, r, &req) {
		return
	}

	if err := s.validateTaskRequest(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	task := &db.Task{
		Name:           req.Name,
		Prompt:         req.Prompt,
		CronExpr:       req.CronExpr,
		WorkingDir:     req.WorkingDir,
		DiscordWebhook: req.DiscordWebhook,
		SlackWebhook:   req.SlackWebhook,
		Model:          req.Model,
		PermissionMode: req.PermissionMode,
		Enabled:        req.Enabled,
	}

	// Parse scheduled_at for one-off tasks
	if req.ScheduledAt != nil && *req.ScheduledAt != "" {
		scheduledAt, err := time.Parse(time.RFC3339, *req.ScheduledAt)
		if err != nil {
			s.errorResponse(w, http.StatusBadRequest, "Invalid scheduled_at format (use RFC3339)", err)
			return
		}
		task.ScheduledAt = &scheduledAt
	}

	if err := s.db.CreateTask(task); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to create task", err)
		return
	}

	// Schedule the task if enabled
	if task.Enabled && s.scheduler != nil {
		if err := s.scheduler.AddTask(task); err != nil {
			s.errorResponse(w, http.StatusInternalServerError, "Task created but scheduling failed", err)
			return
		}
	}

	s.jsonResponse(w, http.StatusCreated, s.taskToResponse(task, ""))
}

// GetTask handles GET /api/v1/tasks/{id}
func (s *Server) GetTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	task, err := s.db.GetTask(id)
	if err != nil {
		s.errorResponse(w, http.StatusNotFound, "Task not found", err)
		return
	}

	// Get last run status
	var status db.RunStatus
	lastRun, err := s.db.GetLatestTaskRun(id)
	if err == nil {
		status = lastRun.Status
	} else if !errors.Is(err, sql.ErrNoRows) {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to fetch latest run", err)
		return
	}

	s.jsonResponse(w, http.StatusOK, s.taskToResponse(task, status))
}

// UpdateTask handles PUT /api/v1/tasks/{id}
func (s *Server) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	task, err := s.db.GetTask(id)
	if err != nil {
		s.errorResponse(w, http.StatusNotFound, "Task not found", err)
		return
	}

	var req TaskRequest
	if !s.decodeJSONBody(w, r, &req) {
		return
	}

	if err := s.validateTaskRequest(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Update task fields
	task.Name = req.Name
	task.Prompt = req.Prompt
	task.CronExpr = req.CronExpr
	task.WorkingDir = req.WorkingDir
	task.DiscordWebhook = req.DiscordWebhook
	task.SlackWebhook = req.SlackWebhook
	task.Model = req.Model
	task.PermissionMode = req.PermissionMode
	task.Enabled = req.Enabled

	// Parse scheduled_at for one-off tasks
	if req.ScheduledAt != nil && *req.ScheduledAt != "" {
		scheduledAt, err := time.Parse(time.RFC3339, *req.ScheduledAt)
		if err != nil {
			s.errorResponse(w, http.StatusBadRequest, "Invalid scheduled_at format (use RFC3339)", err)
			return
		}
		task.ScheduledAt = &scheduledAt
	} else {
		task.ScheduledAt = nil
	}

	if err := s.db.UpdateTask(task); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to update task", err)
		return
	}

	// Update scheduler
	if s.scheduler != nil {
		if err := s.scheduler.UpdateTask(task); err != nil {
			s.errorResponse(w, http.StatusInternalServerError, "Task updated but scheduling failed", err)
			return
		}
	}

	s.jsonResponse(w, http.StatusOK, s.taskToResponse(task, ""))
}

// DeleteTask handles DELETE /api/v1/tasks/{id}
func (s *Server) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	// Check task exists
	_, err = s.db.GetTask(id)
	if err != nil {
		s.errorResponse(w, http.StatusNotFound, "Task not found", err)
		return
	}

	// Remove from scheduler first
	if s.scheduler != nil {
		s.scheduler.RemoveTask(id)
	}

	if err := s.db.DeleteTask(id); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to delete task", err)
		return
	}

	s.jsonResponse(w, http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Task deleted",
	})
}

// ToggleTask handles POST /api/v1/tasks/{id}/toggle
func (s *Server) ToggleTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	if err := s.db.ToggleTask(id); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to toggle task", err)
		return
	}

	// Get updated task
	task, err := s.db.GetTask(id)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to fetch task", err)
		return
	}

	// Update scheduler
	if s.scheduler != nil {
		if err := s.scheduler.UpdateTask(task); err != nil {
			s.errorResponse(w, http.StatusInternalServerError, "Task toggled but scheduling update failed", err)
			return
		}
	}

	s.jsonResponse(w, http.StatusOK, s.taskToResponse(task, ""))
}

// RunTask handles POST /api/v1/tasks/{id}/run
func (s *Server) RunTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	task, err := s.db.GetTask(id)
	if err != nil {
		s.errorResponse(w, http.StatusNotFound, "Task not found", err)
		return
	}

	if s.runConcurrency == 0 {
		s.errorResponse(w, http.StatusServiceUnavailable, "Task execution queue is disabled", nil)
		return
	}

	select {
	case s.runSemaphore <- struct{}{}:
		resultCh := s.executor.ExecuteAsync(task)
		go func(taskID int64) {
			result := <-resultCh
			if result != nil && result.Error != nil {
				log.Printf("api run task failed: task_id=%d err=%v", taskID, result.Error)
			}
			<-s.runSemaphore
		}(task.ID)
	default:
		s.errorResponse(w, http.StatusServiceUnavailable, "Task execution queue is full", nil)
		return
	}

	s.jsonResponse(w, http.StatusAccepted, SuccessResponse{
		Success: true,
		Message: "Task execution started",
	})
}

// GetTaskRuns handles GET /api/v1/tasks/{id}/runs
func (s *Server) GetTaskRuns(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	// Check task exists
	_, err = s.db.GetTask(id)
	if err != nil {
		s.errorResponse(w, http.StatusNotFound, "Task not found", err)
		return
	}

	// Get limit from query params, default 20
	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		l, convErr := strconv.Atoi(limitStr)
		if convErr != nil || l <= 0 {
			s.errorResponse(w, http.StatusBadRequest, "limit must be a positive integer", nil)
			return
		}
		if l > maxTaskRunsLimit {
			s.errorResponse(w, http.StatusBadRequest, "limit exceeds maximum allowed value", nil)
			return
		}
		limit = l
	}

	runs, err := s.db.GetTaskRuns(id, limit)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to fetch task runs", err)
		return
	}

	response := TaskRunsResponse{
		Runs:  make([]TaskRunResponse, len(runs)),
		Total: len(runs),
	}

	for i, run := range runs {
		response.Runs[i] = s.taskRunToResponse(run)
	}

	s.jsonResponse(w, http.StatusOK, response)
}

// GetTaskRun handles GET /api/v1/tasks/{id}/runs/{runID}
func (s *Server) GetTaskRun(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	runID, err := strconv.ParseInt(chi.URLParam(r, "runID"), 10, 64)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid run ID", err)
		return
	}

	run, err := s.db.GetTaskRun(taskID, runID)
	if err != nil {
		s.errorResponse(w, http.StatusNotFound, "Run not found", err)
		return
	}

	s.jsonResponse(w, http.StatusOK, s.taskRunToResponse(run))
}

// GetLatestTaskRun handles GET /api/v1/tasks/{id}/runs/latest
func (s *Server) GetLatestTaskRun(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	run, err := s.db.GetLatestTaskRun(id)
	if err != nil {
		s.errorResponse(w, http.StatusNotFound, "No runs found", err)
		return
	}

	s.jsonResponse(w, http.StatusOK, s.taskRunToResponse(run))
}

// GetSettings handles GET /api/v1/settings
func (s *Server) GetSettings(w http.ResponseWriter, r *http.Request) {
	threshold, err := s.db.GetUsageThreshold()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to fetch settings", err)
		return
	}

	s.jsonResponse(w, http.StatusOK, SettingsResponse{
		UsageThreshold: threshold,
	})
}

// UpdateSettings handles PUT /api/v1/settings
func (s *Server) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req SettingsRequest
	if !s.decodeJSONBody(w, r, &req) {
		return
	}

	// Validate threshold
	if req.UsageThreshold < 0 || req.UsageThreshold > 100 {
		s.errorResponse(w, http.StatusBadRequest, "Usage threshold must be between 0 and 100", nil)
		return
	}

	if err := s.db.SetUsageThreshold(req.UsageThreshold); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to update settings", err)
		return
	}

	s.jsonResponse(w, http.StatusOK, SettingsResponse(req))
}

// GetUsage handles GET /api/v1/usage
func (s *Server) GetUsage(w http.ResponseWriter, r *http.Request) {
	client, err := usage.NewClient()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Usage client not available", err)
		return
	}

	data, err := client.Fetch()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to fetch usage", err)
		return
	}

	s.jsonResponse(w, http.StatusOK, UsageResponse{
		FiveHour: UsageBucketResponse{
			Utilization: data.FiveHour.Utilization,
			ResetsAt:    data.FiveHour.ResetsAt,
		},
		SevenDay: UsageBucketResponse{
			Utilization: data.SevenDay.Utilization,
			ResetsAt:    data.SevenDay.ResetsAt,
		},
	})
}

// Helper functions

func (s *Server) taskToResponse(task *db.Task, status db.RunStatus) TaskResponse {
	resp := TaskResponse{
		ID:             task.ID,
		Name:           task.Name,
		Prompt:         task.Prompt,
		CronExpr:       task.CronExpr,
		ScheduledAt:    task.ScheduledAt,
		IsOneOff:       task.IsOneOff(),
		WorkingDir:     task.WorkingDir,
		DiscordWebhook: task.DiscordWebhook,
		SlackWebhook:   task.SlackWebhook,
		Model:          task.Model,
		PermissionMode: task.PermissionMode,
		Enabled:        task.Enabled,
		CreatedAt:      task.CreatedAt,
		UpdatedAt:      task.UpdatedAt,
		LastRunAt:      task.LastRunAt,
		NextRunAt:      task.NextRunAt,
	}
	if status != "" {
		resp.LastRunStatus = string(status)
	}
	return resp
}

func (s *Server) taskRunToResponse(run *db.TaskRun) TaskRunResponse {
	resp := TaskRunResponse{
		ID:        run.ID,
		TaskID:    run.TaskID,
		StartedAt: run.StartedAt,
		EndedAt:   run.EndedAt,
		Status:    string(run.Status),
		Output:    run.Output,
		Error:     run.Error,
		SessionID: run.SessionID,
	}
	if run.EndedAt != nil {
		durationMs := run.EndedAt.Sub(run.StartedAt).Milliseconds()
		resp.DurationMs = &durationMs
	}
	return resp
}

func (s *Server) validateTaskRequest(req *TaskRequest) error {
	if req.Name == "" {
		return errEmptyName
	}
	if req.Prompt == "" {
		return errEmptyPrompt
	}
	// CronExpr is empty for one-off tasks, non-empty for recurring
	if req.CronExpr != "" {
		// Validate cron expression if provided
		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		if _, err := parser.Parse(req.CronExpr); err != nil {
			return errInvalidCron
		}
	}
	if req.WorkingDir == "" {
		req.WorkingDir = "."
	}
	return nil
}

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("api response encode failed: status=%d err=%v", status, err)
	}
}

func (s *Server) errorResponse(w http.ResponseWriter, status int, message string, err error) {
	if err != nil {
		log.Printf("api error: status=%d message=%q err=%v", status, message, err)
	}
	resp := ErrorResponse{
		Error: message,
	}
	s.jsonResponse(w, status, resp)
}

const maxJSONBodyBytes = 1 << 20 // 1 MiB
const maxTaskRunsLimit = 200

func (s *Server) decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return false
	}

	var extra struct{}
	if err := dec.Decode(&extra); err != io.EOF {
		s.errorResponse(w, http.StatusBadRequest, "Request body must contain a single JSON object", nil)
		return false
	}

	return true
}

// Validation errors
type validationError string

func (e validationError) Error() string { return string(e) }

const (
	errEmptyName   validationError = "Name is required"
	errEmptyPrompt validationError = "Prompt is required"
	errInvalidCron validationError = "Invalid cron expression"
)
