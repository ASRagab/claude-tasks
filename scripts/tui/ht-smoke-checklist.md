# HT/manual TUI smoke checklist

Use this when `pilotty` is unavailable in CI or local shell.

## Preconditions
- Built binary: `go build -o claude-tasks ./cmd/claude-tasks`
- Clean temp data dir: `export CLAUDE_TASKS_DATA=$(mktemp -d)`

## Steps
1. Start app:
   - `./claude-tasks`
2. Verify launch:
   - Main task list screen renders (header + empty/list state)
3. Interaction checks:
   - Press `?` (help/hints visible)
   - Press `Esc` (returns to list)
   - Press arrow keys `up/down` (cursor/movement responsive)
4. Quit path:
   - Press `q` and confirm clean exit to shell

## Evidence to capture
- Terminal screenshot or transcript for:
  - initial list screen
  - after `?`
  - final exit

## Pass criteria
- No freeze/crash
- Keystrokes are responsive
- Clean shutdown
