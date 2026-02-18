package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ASRagab/claude-tasks/internal/db"
	"github.com/ASRagab/claude-tasks/internal/testutil"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	database, dataDir := testutil.NewTestDB(t)
	return NewServer(database, nil, dataDir)
}

func TestCreateTaskAndListTasks(t *testing.T) {
	srv := newTestServer(t)

	createReq := TaskRequest{
		Name:       "backup",
		Prompt:     "echo hello",
		CronExpr:   "0 * * * * *",
		WorkingDir: ".",
		Enabled:    true,
	}

	rr := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/api/v1/tasks", createReq)
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	created := testutil.DecodeJSON[TaskResponse](t, rr)
	if created.ID == 0 {
		t.Fatalf("expected created task id to be set")
	}
	if created.Name != createReq.Name {
		t.Fatalf("expected name %q, got %q", createReq.Name, created.Name)
	}

	listRR := httptest.NewRecorder()
	listReq := testutil.JSONRequest(t, http.MethodGet, "/api/v1/tasks", nil)
	srv.Router().ServeHTTP(listRR, listReq)

	if listRR.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, listRR.Code, listRR.Body.String())
	}

	list := testutil.DecodeJSON[TaskListResponse](t, listRR)
	if list.Total != 1 {
		t.Fatalf("expected total=1, got %d", list.Total)
	}
	if len(list.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(list.Tasks))
	}
}

func TestGetTaskReturns404ForMissingTask(t *testing.T) {
	srv := newTestServer(t)

	rr := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/v1/tasks/99999", nil)
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d: %s", http.StatusNotFound, rr.Code, rr.Body.String())
	}

	errResp := testutil.DecodeJSON[ErrorResponse](t, rr)
	if errResp.Error == "" {
		t.Fatalf("expected error message in response")
	}
}

func TestErrorResponseDoesNotLeakInternalDetails(t *testing.T) {
	srv := newTestServer(t)

	rr := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/v1/tasks/not-a-number", nil)
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}

	errResp := testutil.DecodeJSON[ErrorResponse](t, rr)
	if errResp.Details != "" {
		t.Fatalf("expected error details to be redacted, got %q", errResp.Details)
	}
}

func TestCreateTaskRejectsUnknownFields(t *testing.T) {
	srv := newTestServer(t)

	body := `{"name":"backup","prompt":"echo hi","cron_expr":"0 * * * * *","working_dir":".","enabled":true,"unexpected":"nope"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestCreateTaskRejectsTrailingJSONPayload(t *testing.T) {
	srv := newTestServer(t)

	body := `{"name":"backup","prompt":"echo hi","cron_expr":"0 * * * * *","working_dir":".","enabled":true}{"name":"extra"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestGetTaskRunsRejectsExcessiveLimit(t *testing.T) {
	srv := newTestServer(t)
	task := &db.Task{
		Name:       "backup",
		Prompt:     "echo hi",
		CronExpr:   "0 * * * * *",
		WorkingDir: ".",
		Enabled:    true,
	}
	if err := srv.db.CreateTask(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	rr := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d/runs?limit=999999", task.ID), nil)
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestRunTaskReturns503WhenRunQueueDisabled(t *testing.T) {
	t.Setenv("CLAUDE_TASKS_API_RUN_CONCURRENCY", "0")
	srv := newTestServer(t)
	task := &db.Task{
		Name:       "backup",
		Prompt:     "echo hi",
		CronExpr:   "0 * * * * *",
		WorkingDir: ".",
		Enabled:    true,
	}
	if err := srv.db.CreateTask(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	rr := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/tasks/%d/run", task.ID), nil)
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected %d, got %d: %s", http.StatusServiceUnavailable, rr.Code, rr.Body.String())
	}
}

func TestGetTaskRunReturnsSpecificRun(t *testing.T) {
	srv := newTestServer(t)
	task := &db.Task{
		Name:       "backup",
		Prompt:     "echo hi",
		CronExpr:   "0 * * * * *",
		WorkingDir: ".",
		Enabled:    true,
	}
	if err := srv.db.CreateTask(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	run := &db.TaskRun{
		TaskID:    task.ID,
		StartedAt: time.Date(2026, time.February, 17, 0, 0, 0, 0, time.UTC),
		Status:    db.RunStatusCompleted,
		Output:    "done",
	}
	if err := srv.db.CreateTaskRun(run); err != nil {
		t.Fatalf("create task run: %v", err)
	}

	rr := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d/runs/%d", task.ID, run.ID), nil)
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	resp := testutil.DecodeJSON[TaskRunResponse](t, rr)
	if resp.ID != run.ID {
		t.Fatalf("expected run id %d, got %d", run.ID, resp.ID)
	}
	if resp.Output != "done" {
		t.Fatalf("expected output %q, got %q", "done", resp.Output)
	}
}




