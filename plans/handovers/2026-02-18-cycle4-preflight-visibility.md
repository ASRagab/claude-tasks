# Cycle 4 Handoff — Persist Preflight Failures as Run Records + Logs

Date: 2026-02-18

## Changed Executor Flow and Why

Previously, preflight failures (missing credentials, usage threshold exceeded) returned errors silently — no `task_runs` record was created, no structured log was written. Failures were invisible to the run history, API, and TUI.

The `failPreflight` method was added to `executor.go` to ensure every preflight failure:
1. Creates a `task_runs` record with status `failed` and the original error message.
2. Writes a structured JSON log file via `logger.WriteRunLog()`.
3. Returns a `Result` with the original preflight error preserved (joined with any secondary errors).

## Before/After Behavior Table

| Scenario | Before | After |
|----------|--------|-------|
| Usage client unavailable | Silent error returned, no run record | Failed run record + log file created |
| Usage threshold exceeded | Skipped with no visibility | Failed run record + log file created |
| Usage threshold query fails | Silent error returned | Failed run record + log file created |
| API `/run` with preflight failure | 202 accepted, no run visible | 202 accepted, failed run appears in `/runs/latest` |
| TUI run-now with preflight failure | Error shown in TUI but no history | Failed run appears in run history |

## Evidence from Unit Tests

### C4-V1: Executor tests (`go test ./internal/executor -v`)

`TestExecuteFailsClosedWhenUsageCheckUnavailable`:
- Constructs executor with `usageClientErr = errors.New("credentials not found")`
- Asserts `result.Error` contains "usage threshold enforcement unavailable"
- Asserts exactly 1 run record exists with status `failed`
- Asserts run error contains "usage threshold enforcement unavailable"
- Asserts exactly 1 structured log file exists under `logs/<task_id>/`

### C4-V2: API tests (`go test ./internal/api -v`)

`TestRunTaskPersistsPreflightFailureRun`:
- Sets HOME to temp dir (no credentials), unsets `CLAUDE_TASKS_DISABLE_USAGE_CHECK`
- POSTs to `/run` — expects 202 accepted
- Polls `/runs/latest` with attempt counters (up to 2s deadline)
- Asserts status `failed` and error contains "usage threshold enforcement unavailable"

## Evidence from Near-Live Path (C4-V3)

1. Started `serve --scheduler=false --port 9877` in temp env (no credentials, usage check enabled)
2. Created task via `POST /api/v1/tasks`
3. Triggered run via `POST /api/v1/tasks/1/run` — received 202
4. Polled `GET /api/v1/tasks/1/runs/latest` — on first attempt:
   - `status: "failed"`
   - `error: "usage threshold enforcement unavailable: credentials not found at ..."`
5. Verified structured log file at `data/logs/1/1_failed_20260218T222403.json`
6. Server killed, port 9877 confirmed clean

## C4-V4: TUI Run-Now Path

Not validated via automated TUI interaction (requires interactive PTY). The TUI's run-now path invokes the same `Executor.Execute()` method validated in C4-V1 and C4-V3. The run history view reads from the same `task_runs` table, so visibility is guaranteed by the same persistence path.

## Residual Edge Cases

1. **Race between run creation and API poll**: The API `/run` endpoint fires `ExecuteAsync` in a goroutine. In the near-live test, the failed run appeared on the first poll (20ms is enough for preflight failure). For real execution failures (30-min timeout), the poll would need to wait longer.

2. **Joined error messages**: If `CreateTaskRun` itself fails (e.g., DB locked), the returned error joins the preflight error with the DB error via `errors.Join`. The preflight message is preserved but the run record is not created. This is an edge case that would require DB failure to trigger.

3. **Log file write failure**: If `WriteRunLog` fails, the error is joined but the run record is still persisted. The structured log is best-effort.
