# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
# Build
go build -o claude-tasks ./cmd/claude-tasks

# Run tests
go test -v ./...

# Run a single test
go test -v -run TestName ./internal/package/...

# Lint (requires golangci-lint)
golangci-lint run --timeout=5m

# Release build with version info and optimizations
VERSION=$(git describe --tags --always)
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
go build -ldflags="-s -w -X github.com/ASRagab/claude-tasks/internal/version.Version=$VERSION -X github.com/ASRagab/claude-tasks/internal/version.Commit=$COMMIT -X github.com/ASRagab/claude-tasks/internal/version.BuildDate=$DATE" -o claude-tasks ./cmd/claude-tasks
```

## CLI Commands

```bash
claude-tasks              # Launch the interactive TUI
claude-tasks daemon       # Run scheduler in foreground (for services)
claude-tasks serve        # Run HTTP API server (default port 8080)
claude-tasks serve --port 3000  # Run API on custom port
claude-tasks version      # Show version information
claude-tasks upgrade      # Upgrade to the latest version
claude-tasks help         # Show help message
```

## Architecture

Claude Tasks is a Go TUI application for scheduling Claude CLI tasks via cron expressions. The data is stored in SQLite at `~/.claude-tasks/tasks.db`.

### Package Structure

```
cmd/claude-tasks/main.go   Entry point - CLI commands, initializes DB, starts scheduler, launches TUI
internal/
  api/                     HTTP REST API server (chi router) for mobile/remote access
  tui/                     Bubble Tea TUI (views: list, add, edit, run history, output, settings)
  scheduler/               Cron job scheduling (robfig/cron, 6-field with seconds)
  executor/                Claude CLI subprocess execution with dynamic flags, session IDs, captures output
  db/                      SQLite models (Task, TaskRun) and CRUD operations
  logger/                  Structured JSON run logging to ~/.claude-tasks/logs/
  usage/                   Anthropic API usage tracking, threshold enforcement
  webhook/                 Discord and Slack webhook notifications
  version/                 Version info (set at build time via ldflags)
  upgrade/                 Self-update from GitHub releases
mobile/                    Expo/React Native app (connects to API server)
```

### Execution Flow

1. Scheduler triggers task based on cron expression
2. Executor checks API usage against threshold (default 80%)
3. Executor generates a UUID session ID
4. Builds CLI args dynamically: `-p`, permission mode flag, model flag, `--session-id`, prompt
5. Spawns Claude CLI in the task's working directory
6. Captures output, creates TaskRun record with session ID
7. Writes structured JSON log to `~/.claude-tasks/logs/<task_id>/`
8. Posts to Discord/Slack webhooks if configured
9. Updates next run time

### Key Dependencies

- **charmbracelet/bubbletea** - TUI framework
- **charmbracelet/bubbles** - Table, spinner, viewport, progress components
- **charmbracelet/lipgloss** - Terminal styling
- **charmbracelet/glamour** - Markdown rendering
- **go-chi/chi/v5** - HTTP router for REST API
- **robfig/cron/v3** - Cron scheduling (6-field: `second minute hour day month weekday`)
- **lnquy/cron** - Cron expression to human-readable English
- **mattn/go-sqlite3** - SQLite driver (CGO required)

### Data Storage

- Default location: `~/.claude-tasks/`
- Override with `CLAUDE_TASKS_DATA` environment variable
- Database auto-migrates on startup
- `logs/` directory contains structured JSON run logs
- `daemon.pid` file tracks running daemon process

### Operating Modes

1. **TUI Mode** (default): Interactive terminal UI with embedded scheduler
2. **Daemon Mode** (`daemon`): Headless scheduler, TUI connects as client
3. **Server Mode** (`serve`): REST API + scheduler for mobile/remote access

When a daemon is running, the TUI detects it via PID file and operates in client mode (no duplicate scheduler).

### REST API

The `serve` command starts an HTTP server with these endpoints:

```
GET    /api/v1/health              Health check
GET    /api/v1/tasks               List all tasks
POST   /api/v1/tasks               Create task (supports model, permission_mode)
GET    /api/v1/tasks/{id}          Get task by ID
PUT    /api/v1/tasks/{id}          Update task
DELETE /api/v1/tasks/{id}          Delete task
POST   /api/v1/tasks/{id}/toggle   Toggle enabled
POST   /api/v1/tasks/{id}/run      Run immediately
GET    /api/v1/tasks/{id}/runs     Get task run history (includes session_id)
GET    /api/v1/tasks/{id}/runs/latest  Get latest run
GET    /api/v1/settings            Get settings
PUT    /api/v1/settings            Update settings
GET    /api/v1/usage               Get API usage stats
```

## Mobile App

The `mobile/` directory contains an Expo/React Native app that connects to the REST API.

```bash
cd mobile
npm install
npm start         # Start Expo dev server
npm run ios       # iOS simulator
npm run android   # Android emulator
```

The app requires the API server running (`claude-tasks serve`) and configured via the setup screen.

## TUI Keybindings

### Task List
| Key | Action |
|-----|--------|
| `a` | Add task |
| `e` | Edit task |
| `d` | Delete task (with confirmation) |
| `t` | Toggle enabled |
| `r` | Run immediately |
| `Enter` | View run history |
| `s` | Settings |
| `/` | Search/filter tasks |
| `?` | Toggle help |
| `q` | Quit |

### Run History
| Key | Action |
|-----|--------|
| `Enter` | View full run output |
| `o` | Observe running task (opens Terminal with `claude --resume`) |
| `r` | Refresh |
| `Esc` | Back |

### Add/Edit Form
| Key | Action |
|-----|--------|
| `Tab`/`Shift+Tab` | Navigate fields |
| `Left/Right` | Toggle options (Model, Permission Mode, Task Type) |
| `?` | Cron preset picker (in cron field) |
| `Ctrl+S` | Save |
| `Esc` | Cancel |

## Task Configuration

Each task has:
- **Model** (`model`): `""` (default), `"opus"`, `"sonnet"`, `"haiku"` - maps to `--model` CLI flag
- **Permission Mode** (`permission_mode`): `"bypassPermissions"` (default), `"default"`, `"acceptEdits"`, `"plan"` - maps to `--dangerously-skip-permissions` or `--permission-mode` CLI flag
- **Session ID**: Auto-generated UUID per run, passed as `--session-id` to Claude CLI

Constants defined in `internal/db/models.go`:
```go
var ModelAliases    = []string{"", "opus", "sonnet", "haiku"}
var PermissionModes = []string{"bypassPermissions", "default", "acceptEdits", "plan"}
const DefaultPermissionMode = "bypassPermissions"
```

## Task Types

- **Recurring tasks**: Have a cron expression (`cron_expr`), scheduled via `robfig/cron` with 6-field format (seconds included): `second minute hour day month weekday`
- **One-off tasks**: Empty `cron_expr`, optional `scheduled_at` timestamp. Auto-disable after execution. If no `scheduled_at`, run immediately.

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `CLAUDE_TASKS_DATA` | Override data directory | `~/.claude-tasks` |

## Build Requirements

- **Go 1.24+**
- **CGO_ENABLED=1** required (SQLite driver uses CGO via `mattn/go-sqlite3`)
- Claude CLI must be installed and authenticated for task execution

## Code Patterns

- All packages under `internal/` follow single-responsibility: one primary type per package
- Database operations use raw SQL (no ORM) with `database/sql`
- Error handling wraps with `fmt.Errorf("context: %w", err)` for chain unwrapping
- Scheduler syncs from DB every 10 seconds to pick up external changes (API, another TUI)
- Executor has a 30-minute timeout per task execution
- Executor generates session IDs via `crypto/rand` (no external deps)
- Usage client reads OAuth token from `~/.claude/.credentials.json` and caches responses for 30s
- Webhook notifications support both Discord (embeds) and Slack (Block Kit) formats
- Structured JSON logs written per run to `~/.claude-tasks/logs/<task_id>/`

## Database Schema

Three tables: `tasks`, `task_runs`, `settings`. Schema auto-migrates on startup. Incremental migrations use `ALTER TABLE` with silent error handling for idempotency.

Key columns added to `tasks`: `model`, `permission_mode`
Key column added to `task_runs`: `session_id`

## CI/CD

- **CI** (`.github/workflows/ci.yml`): Build + test + lint on push/PR to `main`
- **Release** (`.github/workflows/release.yml`): Cross-platform builds (linux amd64/arm64, darwin amd64/arm64, windows amd64) on tag push `v*`, creates GitHub release with checksums
