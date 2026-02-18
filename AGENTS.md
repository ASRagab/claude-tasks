# AGENTS.md

Instructions for AI coding agents working on this repository.

## Project Overview

Claude Tasks is a Go application for scheduling and running Claude CLI tasks via cron expressions. It provides a terminal UI (TUI), a headless daemon mode, and an HTTP REST API with a companion React Native mobile app.

This is a fork of [kylemclaren/claude-tasks](https://github.com/kylemclaren/claude-tasks) with additional features: per-task model selection, permission mode control, session observability, structured logging, and cron-to-English descriptions.

## Setup

```bash
# Install dependencies (Go modules)
go mod download

# Build
go build -o claude-tasks ./cmd/claude-tasks

# Run tests
go test -v ./...

# Lint
golangci-lint run --timeout=5m
```

Requires Go 1.24+ and CGO_ENABLED=1 (the SQLite driver depends on CGO).

## Project Structure

```
cmd/claude-tasks/main.go    CLI entrypoint, routes subcommands
internal/
  api/                      REST API (chi router, JSON handlers, CORS middleware)
  db/                       SQLite database layer (models, CRUD, migrations)
  executor/                 Claude CLI subprocess runner with dynamic flags, session IDs, usage gating
  logger/                   Structured JSON run logging per task
  scheduler/                Cron scheduling (robfig/cron, 6-field with seconds)
  tui/                      Bubble Tea terminal UI (list, add, edit, run history, output, settings views)
  upgrade/                  Self-update from GitHub releases
  usage/                    Anthropic API usage tracking and threshold enforcement
  version/                  Build-time version info via ldflags
  webhook/                  Discord and Slack notification senders
mobile/                     Expo/React Native mobile app
```

## Coding Conventions

- **Language**: Go 1.24, no generics used currently
- **Error handling**: Wrap errors with `fmt.Errorf("context: %w", err)` for unwrapping
- **Database**: Raw SQL with `database/sql`, no ORM. Parameterized queries only.
- **No global state**: Dependencies injected via struct fields
- **Package scope**: Each internal package has one primary type/responsibility
- **Testing**: Standard `go test`, no external test framework
- **Formatting**: `gofmt` standard, enforced by `golangci-lint`

## Key Behaviors

- Cron uses 6-field format: `second minute hour day month weekday`
- Cron expressions are displayed as human-readable English in the TUI (via `lnquy/cron`)
- One-off tasks have empty `cron_expr` and optional `scheduled_at`; they auto-disable after running
- Scheduler polls DB every 10 seconds to pick up external changes
- Executor enforces a 30-minute timeout per task
- Executor generates a UUID session ID per run and passes `--session-id` to Claude CLI
- Executor builds CLI flags dynamically based on task's `model` and `permission_mode` fields
- Usage client reads `~/.claude/.credentials.json` for OAuth token
- Executor fails task execution if usage threshold enforcement cannot be initialized or checked
- API auth is optional via `CLAUDE_TASKS_AUTH_TOKEN` (Bearer token; health endpoint exempt)
- CORS can be restricted to a single origin via `CLAUDE_TASKS_CORS_ORIGIN`
- API run endpoint concurrency is bounded by `CLAUDE_TASKS_API_RUN_CONCURRENCY` (default 8; `0` disables)
- Usage threshold enforcement can be bypassed with `CLAUDE_TASKS_DISABLE_USAGE_CHECK=1` for non-Anthropic auth setups
- Default data directory: `~/.claude-tasks/`, overridable via `CLAUDE_TASKS_DATA`
- Structured JSON logs written to `~/.claude-tasks/logs/<task_id>/` per run

## Task Configuration Fields

| Field | Values | CLI Flag |
|-------|--------|----------|
| `model` | `""`, `"opus"`, `"sonnet"`, `"haiku"` | `--model <alias>` |
| `permission_mode` | `"bypassPermissions"`, `"default"`, `"acceptEdits"`, `"plan"` | `--dangerously-skip-permissions` or `--permission-mode <mode>` |

Constants in `internal/db/models.go`:
```go
var ModelAliases    = []string{"", "opus", "sonnet", "haiku"}
var PermissionModes = []string{"bypassPermissions", "default", "acceptEdits", "plan"}
const DefaultPermissionMode = "bypassPermissions"
```

## API Endpoints

```
GET    /api/v1/health
GET    /api/v1/tasks               (response includes model, permission_mode)
POST   /api/v1/tasks               (request accepts model, permission_mode)
GET    /api/v1/tasks/{id}
PUT    /api/v1/tasks/{id}
DELETE /api/v1/tasks/{id}
POST   /api/v1/tasks/{id}/toggle
POST   /api/v1/tasks/{id}/run
GET    /api/v1/tasks/{id}/runs     (response includes session_id)
GET    /api/v1/tasks/{id}/runs/{runID}
GET    /api/v1/tasks/{id}/runs/latest
GET    /api/v1/settings
PUT    /api/v1/settings
GET    /api/v1/usage
```

## Database

SQLite at `~/.claude-tasks/tasks.db`. Three tables: `tasks`, `task_runs`, `settings`. Schema auto-migrates on startup. Foreign keys enabled via connection string.

Key columns: `tasks.model`, `tasks.permission_mode`, `task_runs.session_id`

## Do

- Use parameterized SQL queries to prevent injection
- Wrap errors with context using `%w` verb
- Keep packages focused on a single concern
- Test with `go test -v ./...` before submitting changes
- Append new migrations (ALTER TABLE) rather than modifying existing ones

## Avoid

- ORMs or query builders -- stick with raw SQL
- Global mutable state -- pass dependencies explicitly
- Importing from `internal/` packages outside this module
- Removing or changing existing migration SQL (append new migrations instead)
- Adding external dependencies for functionality achievable with stdlib (e.g., UUID uses `crypto/rand`)
