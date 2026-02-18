#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
source "$SCRIPT_DIR/pilotty-lib.sh"

SESSION_NAME="claude-tasks-smoke"
TMP_DIR="$(mktemp -d)"
SNAP_DIR="$TMP_DIR/snapshots"
mkdir -p "$SNAP_DIR"

cleanup() {
  pilotty_safe_kill "$SESSION_NAME"
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

require_command pilotty
require_command go
ensure_built_binary "$REPO_ROOT"

echo "running TUI smoke with data dir: $TMP_DIR"

pilotty spawn --name "$SESSION_NAME" env CLAUDE_TASKS_DATA="$TMP_DIR" "$REPO_ROOT/claude-tasks"

if ! pilotty wait-for -s "$SESSION_NAME" "claude" -t 12000 >/dev/null 2>&1; then
  pilotty wait-for -s "$SESSION_NAME" "task" -t 12000 >/dev/null
fi

pilotty_snapshot_text "$SESSION_NAME" "$SNAP_DIR/01-initial.txt"

# Basic navigation/input smoke
pilotty key -s "$SESSION_NAME" "down"
pilotty key -s "$SESSION_NAME" "up"
pilotty key -s "$SESSION_NAME" "?"
pilotty_snapshot_text "$SESSION_NAME" "$SNAP_DIR/02-after-help-key.txt"
pilotty key -s "$SESSION_NAME" "Escape"

# Quit TUI
pilotty key -s "$SESSION_NAME" "q"
sleep 1
pilotty_safe_kill "$SESSION_NAME"

echo "pilotty smoke complete"
echo "snapshots: $SNAP_DIR"
