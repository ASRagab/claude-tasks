# Handoff — Cycle 2 (Explicit Startup Scheduler Modes)

## What changed
Added explicit scheduler mode controls for each runtime entrypoint:

- **TUI**
  - New command: `claude-tasks tui`
  - New flag (works on default invocation too): `--scheduler=auto|on|off`
  - `auto` preserves prior behavior (`start scheduler only when daemon.pid not active`)
  - `on` forces local scheduler start
  - `off` disables local scheduler and keeps TUI in client/manual-run mode

- **daemon**
  - New flag: `claude-tasks daemon --scheduler=true|false`

- **serve**
  - New flag: `claude-tasks serve --scheduler=true|false`

Also added startup log lines that show effective scheduler state and leader status.

## Files touched
- `cmd/claude-tasks/main.go`
- `cmd/claude-tasks/main_test.go`

## Validation evidence
### Unit tests
- `go test ./cmd/claude-tasks -v` ✅
  - `TestParseTUISchedulerMode`
  - `TestShouldStartTUIScheduler`

### API compatibility tests
- `go test ./internal/api -v` ✅

### Near-live validation
1. Built binary: `go build -o claude-tasks ./cmd/claude-tasks` ✅
2. Started API with scheduler disabled:
   - `./claude-tasks serve --port 38200 --scheduler=false`
   - Verified log contains `Serve scheduler: disabled`
   - Verified no leadership acquisition log in serve output
3. Started TUI in scheduler-off mode via pilotty:
   - `./claude-tasks --scheduler=off`
   - Triggered run-now (`r`) on created task
   - Verified run record created and completed via API (`output: tui-run-ok`)

## Notes on earlier “stuck” signal
The repeated "stuck" came from long combined shell scripts getting cancelled in-session, not runtime deadlock. Validation is now executed in short, observable checkpoints.

## Next cycle input context
Proceed to Cycle 3 (`claude-tasks doctor`) using these mode controls as ground truth for process role separation.
