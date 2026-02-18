#!/usr/bin/env bash

set -euo pipefail

require_command() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "missing required command: $cmd" >&2
    return 1
  fi
}

ensure_built_binary() {
  local repo_root="$1"
  if [[ ! -x "$repo_root/claude-tasks" ]]; then
    echo "building claude-tasks binary..."
    go build -o "$repo_root/claude-tasks" "$repo_root/cmd/claude-tasks"
  fi
}

pilotty_safe_kill() {
  local session="$1"
  pilotty kill -s "$session" >/dev/null 2>&1 || true
}

pilotty_snapshot_text() {
  local session="$1"
  local output_path="$2"
  pilotty snapshot -s "$session" --format text > "$output_path"
}
