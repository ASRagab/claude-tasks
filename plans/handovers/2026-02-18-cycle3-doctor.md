# Cycle 3 Handoff — `claude-tasks doctor`

Date: 2026-02-18

## Checks Implemented and Severity Map

| Check | Severity | Condition for FAIL | Condition for PASS |
|-------|----------|--------------------|--------------------|
| `claude_binary` | FAIL (critical) | `claude` not found in PATH | `claude` found via `exec.LookPath` |
| `usage_credentials` | FAIL (critical) | `CLAUDE_TASKS_DISABLE_USAGE_CHECK` unset and credentials file missing | Env var truthy OR credentials file present |
| `data_dir` | FAIL (critical) | Cannot create or write to data directory | Directory exists and is writable |
| `logs_dir` | FAIL (critical) | Cannot create or write to `<data_dir>/logs/` | Directory exists and is writable |
| `database` | FAIL (critical) | Cannot open SQLite DB or acquire test lease | DB opens and lease acquire/release succeeds |
| `scheduler_lease` | WARN (non-fatal) | Lease row unreadable | Lease readable (shows holder, expiry, state) |

Any FAIL check causes the process to exit non-zero.

## Command Wiring

- `main.go` case `"doctor"` calls `runDoctor()` which instantiates `doctor.NewRunner(dataDir)` and calls `.Run()`.
- Help text (`printHelp()`) includes `claude-tasks doctor` with description.

## Exact Commands Executed

### C3-V1: Package tests
```
go test ./internal/doctor -v -timeout 30s
go test ./cmd/claude-tasks -v -timeout 30s
```
Result: All tests passed.

### C3-V2: Healthy doctor run (near-live)
```bash
TMPDATA=$(mktemp -d)
TMPBIN=$(mktemp -d)
printf '#!/bin/sh\nexit 0\n' > "$TMPBIN/claude" && chmod +x "$TMPBIN/claude"
CLAUDE_TASKS_DATA="$TMPDATA" CLAUDE_TASKS_DISABLE_USAGE_CHECK=1 PATH="$TMPBIN:$PATH" \
  ./claude-tasks doctor
```
Result: All PASS, exit code 0.

### C3-V3: Failing doctor — missing credentials
```bash
TMPDATA=$(mktemp -d)
TMPBIN=$(mktemp -d)
TMPHOME=$(mktemp -d)
printf '#!/bin/sh\nexit 0\n' > "$TMPBIN/claude" && chmod +x "$TMPBIN/claude"
CLAUDE_TASKS_DATA="$TMPDATA" CLAUDE_TASKS_DISABLE_USAGE_CHECK="" HOME="$TMPHOME" PATH="$TMPBIN:$PATH" \
  ./claude-tasks doctor
```
Result: `usage_credentials` FAIL, exit code 1.

### C3-V4: Failing doctor — missing claude binary
```bash
TMPDATA=$(mktemp -d)
EMPTYBIN=$(mktemp -d)
CLAUDE_TASKS_DATA="$TMPDATA" CLAUDE_TASKS_DISABLE_USAGE_CHECK=1 PATH="$EMPTYBIN" \
  ./claude-tasks doctor
```
Result: `claude_binary` FAIL, exit code 1.

### C3-V5: Cleanup
```
pgrep -f "ct-doctor-test/claude-tasks"
```
Result: No leftover processes.

## Sample Pass Output

```
claude-tasks doctor
Data directory: /tmp/xxx
Database path: /tmp/xxx/tasks.db
[PASS] claude_binary: found at /tmp/yyy/claude
[PASS] usage_credentials: usage check disabled via CLAUDE_TASKS_DISABLE_USAGE_CHECK
[PASS] data_dir: writable
[PASS] logs_dir: writable
[PASS] database: open and writable
[PASS] scheduler_lease: holder=doctor-xxx lease_expires_at=... (expired)
Doctor checks passed
```

## Sample Failure Output (missing credentials)

```
claude-tasks doctor
Data directory: /tmp/xxx
Database path: /tmp/xxx/tasks.db
[PASS] claude_binary: found at /tmp/yyy/claude
[FAIL] usage_credentials: usage credentials unavailable: credentials not found at ...
  hint: Login Claude CLI or set CLAUDE_TASKS_DISABLE_USAGE_CHECK=1
[PASS] data_dir: writable
[PASS] logs_dir: writable
[PASS] database: open and writable
[PASS] scheduler_lease: holder=doctor-xxx lease_expires_at=... (expired)
Error: doctor found 1 critical issue(s)
```

## Sample Failure Output (missing claude binary)

```
claude-tasks doctor
Data directory: /tmp/xxx
Database path: /tmp/xxx/tasks.db
[FAIL] claude_binary: `claude` executable not found in PATH
  hint: Install Claude CLI or prepend its bin directory to PATH
[PASS] usage_credentials: usage check disabled via CLAUDE_TASKS_DISABLE_USAGE_CHECK
[PASS] data_dir: writable
[PASS] logs_dir: writable
[PASS] database: open and writable
[PASS] scheduler_lease: holder=doctor-xxx lease_expires_at=... (expired)
Error: doctor found 1 critical issue(s)
```

## Operational Guidance

### Local use
```bash
claude-tasks doctor
```
Run before first use or after environment changes (PATH, credentials, data dir). Exit code 0 = healthy.

### CI use
```bash
CLAUDE_TASKS_DISABLE_USAGE_CHECK=1 claude-tasks doctor
```
In CI, disable usage check (no OAuth credentials) and ensure a `claude` stub is on PATH. Assert exit code 0 in pipeline.

### Troubleshooting
- **FAIL claude_binary**: Install Claude CLI or add its directory to PATH.
- **FAIL usage_credentials**: Run `claude` to authenticate, or set `CLAUDE_TASKS_DISABLE_USAGE_CHECK=1`.
- **FAIL data_dir / logs_dir**: Check filesystem permissions on `CLAUDE_TASKS_DATA` (default `~/.claude-tasks`).
- **FAIL database**: Check SQLite file permissions and ensure no corrupted lock files.
- **WARN scheduler_lease**: Informational only. Shows current lease holder if any.
