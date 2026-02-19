# Runtime Reliability Iteration Plan (2026-02-18)

## Scope
Implement and validate four reliability improvements sequentially:
1. Scheduler leadership lock with heartbeat/failover
2. Explicit scheduler startup modes across TUI/daemon/serve
3. `claude-tasks doctor` diagnostics command
4. Persist executor preflight usage failures as failed runs + structured logs

No cycle advances until all validation checks for that cycle pass.

## Validation Standards (apply every cycle)
- Unit tests: targeted package tests for changed behavior
- Smoke tests: `go test ./...` and `golangci-lint run --timeout=5m` (or package-focused lint if full lint unavailable)
- Near-live verification:
  - Build real binary (`go build -o claude-tasks ./cmd/claude-tasks`)
  - Launch process(es) with temp `CLAUDE_TASKS_DATA`
  - Attempt an actual run path via TUI-triggered or API-triggered execution
  - Verify observable behavior via DB/API/log files

## Cycle 1 — Scheduler leadership lock
### Changes
- Add DB-backed singleton lease record with owner, expiry, heartbeat metadata.
- Scheduler periodically acquires/renews lease.
- Only lease holder schedules cron/one-off jobs.
- On leadership loss: unschedule local jobs/timers.
- On leadership gain: load tasks and schedule.

### Validation
1. DB unit tests for lease acquire/renew/steal-after-expiry semantics.
2. Scheduler tests for leader/follower behavior.
3. Near-live:
   - Start two scheduler-capable processes against same DB.
   - Create one recurring task and wait one interval.
   - Confirm only one run created per tick (no duplicate runs).

### Handoff
Write compact handoff: implementation details, lease schema, failure modes, validation evidence.

## Cycle 2 — Explicit startup scheduler modes
### Changes
- Add explicit scheduler mode flags:
  - TUI: `--scheduler=auto|off|on`
  - daemon: `--scheduler=true|false`
  - serve: `--scheduler=true|false`
- Keep defaults backward-compatible where possible.
- Ensure process startup prints resolved scheduler mode and leadership state.

### Validation
1. CLI tests or integration tests for flag parsing and invalid values.
2. Main/scheduler startup smoke tests across mode combinations.
3. Near-live:
   - Run `serve --scheduler=false` and verify no scheduler leadership attempts.
   - Run TUI with `--scheduler=off`, trigger run now, confirm manual run still works.

### Handoff
Write compact handoff: mode matrix, defaults, and operator guidance.

## Cycle 3 — Doctor command
### Changes
- Add `claude-tasks doctor` command with non-zero exit on critical failures.
- Checks include:
  - `claude` binary presence
  - credentials presence OR usage check disabled
  - data dir and DB writable
  - logs dir writable
  - scheduler lease holder visibility

### Validation
1. Unit tests for check evaluators.
2. Smoke test command output formatting and exit codes.
3. Near-live:
   - Run doctor in a healthy temp environment.
   - Run doctor with intentionally broken credentials/data permissions and verify failures.

### Handoff
Write compact handoff: check catalog, exit-code contract, remediation mapping.

## Cycle 4 — Persist preflight failures as runs/logs
### Changes
- In executor preflight failure paths (usage client unavailable / threshold check errors), create failed `task_runs` records and write structured run logs.
- Ensure API/TUI surfaces these failures via existing run history.

### Validation
1. Executor unit tests asserting run row + log file exists on preflight failure.
2. API handler tests for `POST /run` followed by retrievable failed run.
3. Near-live:
   - Start server with credentials absent and usage check enabled.
   - Trigger run.
   - Confirm immediate failed run record and log file with explicit reason.

### Handoff
Write compact handoff: changed failure semantics and visibility behavior.

## Final Integration Gate
- Full test run: `go test -v ./...`
- Lint: `golangci-lint run --timeout=5m`
- Real-flow near-live:
  - Start serve + mobile-compatible API mode
  - Trigger run and inspect run history/logs
  - Start TUI and verify run initiation path still functional
- Document final operational runbook for day-to-day dev usage.
