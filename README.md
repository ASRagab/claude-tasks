<img width="978" height="603" alt="Screenshot 2026-01-14 at 21 18 04" src="https://github.com/user-attachments/assets/476bb9c9-e4d6-4e16-8ee2-9364c6d07aa3" />

# Claude Tasks

A TUI scheduler for running Claude tasks on a cron schedule. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

> Fork of [kylemclaren/claude-tasks](https://github.com/kylemclaren/claude-tasks) with additional features for model selection, permission control, session observability, and structured logging.

![Claude Tasks TUI](https://img.shields.io/badge/TUI-BubbleTea-ff69b4)
![Go](https://img.shields.io/badge/Go-1.24+-00ADD8)

## Features

- **Cron Scheduling** - Schedule Claude tasks using 6-field cron expressions (second granularity)
- **One-off Tasks** - Run tasks immediately or schedule for a specific time
- **Model Selection** - Choose per-task model: Opus, Sonnet, or Haiku
- **Permission Modes** - Per-task permission control: Bypass, Default, Accept Edits, or Plan
- **Session Observability** - Track session IDs, view resume commands, and observe running tasks live in Terminal
- **Run History** - Tabular run history with stats (success rate, avg duration), session IDs, and output preview
- **Cron Descriptions** - Human-readable schedule descriptions (e.g., "Every hour, at 0 minutes past the hour")
- **Structured Logging** - JSON log files per task run with model, permission mode, and session metadata
- **Real-time TUI** - Terminal interface with live updates, spinners, search/filter, and responsive columns
- **Discord & Slack Webhooks** - Task results posted with rich formatting
- **Usage Tracking** - Monitor Anthropic API usage with visual progress bars and auto-skip thresholds
- **Markdown Rendering** - Task output rendered with [Glamour](https://github.com/charmbracelet/glamour)
- **Self-Update** - Upgrade to the latest version with `claude-tasks upgrade`
- **SQLite Storage** - Persistent task and run history

## Installation

### Quick Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/ASRagab/claude-tasks/main/install.sh | bash
```

This downloads the latest binary for your platform to `~/.local/bin/`.

### Build from Source

```bash
# Clone the repo
git clone https://github.com/ASRagab/claude-tasks.git
cd claude-tasks

# Build
go build -o claude-tasks ./cmd/claude-tasks

# Run
./claude-tasks
```

### Requirements

- Go 1.24+
- [Claude CLI](https://github.com/anthropics/claude-code) installed and authenticated
- SQLite (bundled via go-sqlite3)

## Usage

### CLI Commands

```bash
claude-tasks              # Launch the interactive TUI
claude-tasks daemon       # Run scheduler in foreground (for services)
claude-tasks serve        # Run HTTP API server (default port 8080)
claude-tasks serve --port 3000  # Run API on custom port
claude-tasks version      # Show version information
claude-tasks upgrade      # Upgrade to the latest version
claude-tasks help         # Show help message
```

### Keybindings

#### Task List

| Key | Action |
|-----|--------|
| `a` | Add new task |
| `e` | Edit selected task |
| `d` | Delete selected task (with confirmation) |
| `t` | Toggle task enabled/disabled |
| `r` | Run task immediately |
| `/` | Search/filter tasks |
| `Enter` | View run history |
| `s` | Settings (usage threshold) |
| `?` | Toggle help |
| `q` | Quit |

#### Run History

| Key | Action |
|-----|--------|
| `Enter` | View full run output |
| `o` | Observe running task (opens Terminal with `claude --resume`) |
| `r` | Refresh run list |
| `Esc` | Back to task list |

#### Add/Edit Form

| Key | Action |
|-----|--------|
| `Tab` | Next field |
| `Shift+Tab` | Previous field |
| `Left/Right` | Toggle options (Model, Permission Mode, Task Type) |
| `?` | Cron preset picker (in cron field) |
| `Ctrl+S` | Save |
| `Esc` | Cancel |

### Task Configuration

Each task supports:

- **Model** - Default (CLI default), Opus, Sonnet, or Haiku
- **Permission Mode** - Bypass Permissions (default for scheduled tasks), Default, Accept Edits, or Plan
- **Cron Expression** - 6-field format: `second minute hour day month weekday`
- **Working Directory** - Where Claude CLI runs
- **Webhooks** - Discord and/or Slack notification URLs

### Cron Format

Uses 6-field cron expressions: `second minute hour day month weekday`

The TUI shows human-readable descriptions as you type:

```
Cron Expression  Every hour, at 0 minutes past the hour
0 0 * * * *
```

Preset picker available with `?`:

```
0 * * * * *      # Every minute
0 0 9 * * *      # Every day at 9:00 AM
0 30 8 * * 1-5   # Weekdays at 8:30 AM
0 0 */2 * * *    # Every 2 hours
0 0 9 * * 0      # Every Sunday at 9:00 AM
```

### Session Observability

Every task execution generates a unique session ID. From the run history view:

- **Session column** shows truncated session IDs for each run
- **Run detail** shows the full session ID with a `claude --resume` command
- **`o` key** on a running task opens a new Terminal window with `claude --resume`, inheriting the task's working directory and permission mode

### Structured Logging

Each task run produces a JSON log file at `~/.claude-tasks/logs/<task_id>/`:

```json
{
  "run_id": 42,
  "task_id": 1,
  "task_name": "Daily Review",
  "model": "sonnet",
  "permission_mode": "bypassPermissions",
  "session_id": "a1b2c3d4-e5f6-...",
  "status": "completed",
  "duration_ms": 45230,
  "output": "..."
}
```

### Webhooks (Discord & Slack)

Add webhook URLs when creating a task to receive notifications:

**Discord:**
- Rich embeds with colored sidebar (green/red/yellow)
- Markdown formatting preserved
- Task status, duration, working directory

**Slack:**
- Block Kit formatting with rich layouts
- Markdown converted to Slack's mrkdwn format
- Timestamps and status fields

### Usage Threshold

Press `s` to configure the usage threshold (default: 80%). When your Anthropic API usage exceeds this threshold, scheduled tasks will be skipped to preserve quota.

The header shows real-time usage:
```
◆ Claude Tasks  5h ████░░░░░░ 42% │ 7d ██████░░░░ 61% │ ⏱ 2h15m │ ⚡ 80%
```

## Configuration

Data is stored in `~/.claude-tasks/`:
- `tasks.db` - SQLite database with tasks, runs, and settings
- `logs/` - Structured JSON log files per task run

Override the data directory:
```bash
CLAUDE_TASKS_DATA=/custom/path ./claude-tasks
```

## REST API

The `serve` command starts an HTTP server. Task and run endpoints include `model`, `permission_mode`, and `session_id` fields.

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

## Example Tasks

### Development Workflow

| Task | Schedule | Model | Prompt |
|------|----------|-------|--------|
| Daily Code Review | 6pm daily | sonnet | Review any uncommitted changes in this repo. Summarize what was worked on and flag any potential issues. |
| Morning Standup Prep | 8:30am weekdays | haiku | Analyze git log from the last 24 hours and prepare a brief standup summary of what was accomplished. |
| Dependency Audit | 9am Mondays | sonnet | Check go.mod for outdated dependencies and security vulnerabilities. Suggest updates if needed. |
| Security Scan | 9am Sundays | opus | Audit code for common security issues: SQL injection, XSS, hardcoded secrets, unsafe operations. |
| Weekly Summary | 5pm Fridays | sonnet | Generate a weekly development summary from git history. Include stats, highlights, and next steps. |

### Data & Analysis

| Task | Schedule | Model | Prompt |
|------|----------|-------|--------|
| HN Sentiment Analysis | 9am daily | sonnet | Pull the top 10 HackerNews stories and run sentiment analysis on all the comments using python and then list the posts with their analysis |
| GitHub Trending | 9am Mondays | haiku | Pull trending GitHub repos from the last week and summarize what each one does, categorizing by language/domain |
| Gold/Silver Prices | 9am daily | haiku | Fetch silver/gold prices and correlate with recent news sentiment |

## Tech Stack

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components (table, spinner, viewport, progress)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Glamour](https://github.com/charmbracelet/glamour) - Markdown rendering
- [robfig/cron](https://github.com/robfig/cron) - Cron scheduler
- [lnquy/cron](https://github.com/lnquy/cron) - Cron expression to English descriptions
- [go-sqlite3](https://github.com/mattn/go-sqlite3) - SQLite driver
- [go-chi/chi](https://github.com/go-chi/chi) - HTTP router for REST API

## License

MIT
