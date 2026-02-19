# Handoff — Cycle 1 (Scheduler Leadership Lease)

## What changed
- Added DB table: `scheduler_leases` (singleton row `id=1`) in migration path.
- Added lease APIs in `internal/db/lease.go`:
  - `TryAcquireSchedulerLease(holderID, ttl)`
  - `GetSchedulerLease()`
  - `ReleaseSchedulerLease(holderID)`
- Scheduler now maintains leadership lease:
  - New fields: holder id, TTL, renew interval, leadership flag
  - `Start()` boots cron + leadership loop, then syncs only if leader
  - `syncLoop()` periodically renews leadership and syncs tasks
  - leadership loss clears local scheduled jobs/timers
  - leadership gain triggers immediate sync
- Scheduling behavior gated by leadership:
  - `AddTask` is no-op for followers
  - `SyncTasks` returns early for followers
  - cron callback and one-off execution re-check leadership before execution
- Added helper methods:
  - `IsLeader()`
  - internal `removeTaskLocked`, `clearSchedulesLocked`

## Files touched
- `internal/db/db.go`
- `internal/db/lease.go`
- `internal/db/lease_test.go`
- `internal/scheduler/scheduler.go`
- `internal/scheduler/scheduler_test.go`

## Validation evidence
### Unit/package tests
- `go test ./internal/db -v` ✅
- `go test ./internal/scheduler -v` ✅
  - includes new tests:
    - single leader at a time
    - failover after leader stop

### Near-live run test
- Built binary: `go build -o claude-tasks ./cmd/claude-tasks` ✅
- Launched two `serve` processes against same `CLAUDE_TASKS_DATA` on ports 38180/38181.
- Created recurring task (`*/2 * * * * *`) with stub `claude` binary.
- Observed only one process claiming leadership in logs (`serve2.log` showed leadership acquired).
- Run timestamps from API were stable at ~2.0s intervals (no duplicate cadence).
- Latest sample from `/tasks/1/runs` showed one completed run per tick.

## Operational notes
- Lease row is self-healing: contenders can take over after expiry.
- If leader process dies, follower should take over after `leaseTTL`.
- Current defaults: TTL 15s, renew interval 5s.

## Next cycle input context
- Cycle 2 will add explicit scheduler startup modes for TUI/daemon/serve.
- Leadership lock is now available and should be treated as source of truth for active scheduler execution.
