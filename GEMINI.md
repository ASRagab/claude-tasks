# GEMINI.md

Context for Gemini when working with the claude-tasks codebase.

## What This Project Does

Claude Tasks schedules and runs Claude CLI commands on a cron schedule. It has three operating modes:

1. **TUI** (default) -- Interactive terminal interface built with Bubble Tea
2. **Daemon** (`claude-tasks daemon`) -- Headless background scheduler
3. **Server** (`claude-tasks serve`) -- HTTP REST API for remote/mobile access

Data is stored in SQLite at `~/.claude-tasks/tasks.db`. Structured run logs are written to `~/.claude-tasks/logs/`.

This is a fork of [kylemclaren/claude-tasks](https://github.com/kylemclaren/claude-tasks) with additional features.

## Fork Additions

- **Model selection** -- Per-task model choice (opus, sonnet, haiku, or default)
- **Permission modes** -- Per-task permission control (bypass, default, acceptEdits, plan)
- **Session observability** -- UUID session IDs per run, resume commands, live observe via Terminal
- **Structured logging** -- JSON log files per run with model, permissions, and session metadata
- **Cron descriptions** -- Human-readable English descriptions of cron expressions in the TUI
- **Run history view** -- Tabular run list with success rate, avg duration, session IDs
- **Responsive columns** -- Dynamic column widths in run history based on terminal size

## Build and Test

```bash
go build -o claude-tasks ./cmd/claude-tasks   # Build binary
go test -v ./...                               # Run all tests
golangci-lint run --timeout=5m                 # Lint
```

**Requirements**: Go 1.24+, CGO_ENABLED=1 (SQLite driver needs CGO)

## Source Layout

| Directory | Purpose |
|-----------|---------|
| `cmd/claude-tasks/` | CLI entrypoint and subcommand routing |
| `internal/api/` | REST API server using chi router |
| `internal/db/` | SQLite models (`Task`, `TaskRun`) and CRUD |
| `internal/executor/` | Builds CLI args dynamically (model, permission mode, session ID), runs Claude subprocess |
| `internal/logger/` | Structured JSON run logging |
| `internal/scheduler/` | Cron job management via `robfig/cron/v3` (6-field with seconds) |
| `internal/tui/` | Bubble Tea TUI with multiple views (list, add/edit, run history, output, settings) |
| `internal/usage/` | Anthropic API usage tracking, threshold gating |
| `internal/webhook/` | Discord and Slack notification delivery |
| `internal/version/` | Build-time version info (ldflags) |
| `internal/upgrade/` | Self-update from GitHub releases |
| `mobile/` | Expo/React Native companion app |

## Important Patterns

- **Two task types**: Recurring (has `cron_expr`) and one-off (empty `cron_expr`, optional `scheduled_at`). One-off tasks auto-disable after execution.
- **Per-task configuration**: Each task has `model` and `permission_mode` fields that control Claude CLI flags.
- **Session tracking**: Executor generates a UUID v4 session ID (via `crypto/rand`) per run, passes `--session-id` to Claude CLI, and stores it in `task_runs.session_id`.
- **Dynamic CLI args**: Executor builds args list based on task config -- `--dangerously-skip-permissions` or `--permission-mode <mode>`, `--model <alias>`, `--session-id <uuid>`.
- **Usage gating**: Before each execution, the executor checks Anthropic API usage against a configurable threshold (default 80%). Skips if over threshold.
- **DB sync**: The scheduler polls the database every 10 seconds so changes from the API or another TUI instance are picked up.
- **Error wrapping**: All errors use `fmt.Errorf("context: %w", err)`.
- **No ORM**: Raw SQL with `database/sql` and parameterized queries throughout.
- **Auto-migration**: Database schema applies on startup; incremental migrations use `ALTER TABLE` with silent error handling.
- **Structured logging**: JSON log files written to `~/.claude-tasks/logs/<task_id>/` with run metadata.

## REST API

Base path: `/api/v1`

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/tasks` | List all tasks (includes model, permission_mode) |
| POST | `/tasks` | Create task (accepts model, permission_mode) |
| GET | `/tasks/{id}` | Get task |
| PUT | `/tasks/{id}` | Update task |
| DELETE | `/tasks/{id}` | Delete task |
| POST | `/tasks/{id}/toggle` | Toggle enabled |
| POST | `/tasks/{id}/run` | Run immediately |
| GET | `/tasks/{id}/runs` | Run history (includes session_id) |
| GET | `/tasks/{id}/runs/latest` | Latest run |
| GET | `/settings` | Get settings |
| PUT | `/settings` | Update settings |
| GET | `/usage` | API usage stats |

## Key Dependencies

| Package | Role |
|---------|------|
| `charmbracelet/bubbletea` | TUI framework |
| `charmbracelet/bubbles` | TUI components (table, spinner, viewport, etc.) |
| `charmbracelet/lipgloss` | Terminal styling |
| `charmbracelet/glamour` | Markdown rendering in terminal |
| `go-chi/chi/v5` | HTTP router |
| `robfig/cron/v3` | Cron scheduling |
| `lnquy/cron` | Cron expression to English descriptions |
| `mattn/go-sqlite3` | SQLite driver (CGO) |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CLAUDE_TASKS_DATA` | `~/.claude-tasks` | Data directory for database, logs, and PID file |

## CI/CD

- CI runs on push/PR to `main`: build, test, lint
- Release builds cross-platform binaries (linux/darwin amd64+arm64, windows amd64) on tag push (`v*`)
- Release artifacts include SHA256 checksums

## TUI Keybindings

**Task list**: `a` add, `e` edit, `d` delete, `t` toggle, `r` run now, `Enter` run history, `s` settings, `/` search, `?` help, `q` quit

**Run history**: `Enter` view detail, `o` observe running task, `r` refresh, `Esc` back

**Add/Edit form**: `Tab`/`Shift+Tab` navigate, `Left/Right` toggle (model, permissions, type), `?` cron presets, `Ctrl+S` save, `Esc` cancel
