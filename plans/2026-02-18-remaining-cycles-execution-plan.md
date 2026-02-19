# Remaining Reliability Cycles — Execution Plan (Cycle 3+)

Date: 2026-02-18

This plan replaces long, monolithic validation scripts with short checkpointed execution blocks.

## Execution Protocol (applies to every remaining cycle)

### 1) Work style
- Implement in small commits/scopes per cycle.
- Never run all validations in one shell script.
- One command = one assertion.
- Every long-running command must have:
  - explicit timeout,
  - visible progress output,
  - dedicated teardown command run separately.

### 2) Checkpoint pattern
For each cycle:
1. **Implementation checkpoint**: compile + targeted tests pass.
2. **Near-live checkpoint**: real binary process path validated.
3. **Failure-mode checkpoint**: expected failure behavior validated.
4. **Teardown checkpoint**: processes/ports cleaned.
5. **Handoff checkpoint**: cycle handoff doc written with evidence.

### 3) Anti-stall constraints
- No single command expected to run >30s.
- Poll loops must print attempt counters.
- PID capture uses listener lookup (`lsof -iTCP:<port> -sTCP:LISTEN -t`), not shell job assumptions.
- Cleanup always as independent command (never buried in traps only).

---

## Cycle 3 — `claude-tasks doctor` (finish + lock)

### Current status
Partially implemented. Needs clean end-to-end validation and handoff evidence.

### Implementation tasks
1. Verify command wiring (`main.go`) and help text correctness.
2. Verify doctor checks and severity semantics:
   - `claude_binary` (FAIL on missing)
   - `usage_credentials` (PASS when disabled, FAIL when required/missing)
   - `data_dir` writable (FAIL)
   - `logs_dir` writable (FAIL)
   - `database` open+writable (FAIL)
   - `scheduler_lease` visibility (WARN/PASS, non-fatal)
3. Confirm exit behavior:
   - any FAIL -> process exits non-zero.

### Validation plan (checkpointed)

#### C3-V1: package tests
- `go test ./internal/doctor -v`
- `go test ./cmd/claude-tasks -v`

**Pass criteria:** all tests pass.

#### C3-V2: healthy doctor run (near-live)
- Build binary.
- Launch doctor in temp env with fake `claude` on PATH and `CLAUDE_TASKS_DISABLE_USAGE_CHECK=1`.

**Pass criteria:** output includes `Doctor checks passed`, exit code `0`.

#### C3-V3: failing doctor run — missing credentials
- Keep fake `claude`; unset usage bypass; HOME points to temp dir with no credentials.

**Pass criteria:** `usage_credentials` is FAIL and process exits non-zero.

#### C3-V4: failing doctor run — missing claude binary
- PATH without `claude`, usage bypass enabled.

**Pass criteria:** `claude_binary` is FAIL and process exits non-zero.

#### C3-V5: cleanup
- No leftover listeners/PIDs from doctor runs.

### Handoff doc for Cycle 3
Write: `plans/handovers/2026-02-18-cycle3-doctor.md`

Must include:
- checks implemented and severity map,
- exact commands executed,
- observed exit codes,
- sample pass output and sample failure outputs,
- operational guidance for local use and CI use.

---

## Cycle 4 — Persist preflight failures as run records + logs

### Current status
Partially implemented. Need complete verification + API/TUI visibility proof.

### Implementation tasks
1. Ensure executor preflight failures call unified path that:
   - creates failed `task_runs` record,
   - writes structured log file,
   - returns error preserving original preflight message.
2. Update/confirm tests for preflight failure behavior.
3. Ensure API `/run` async path still returns accepted while failure becomes visible in run history.

### Validation plan (checkpointed)

#### C4-V1: executor tests
- `go test ./internal/executor -v`

**Pass criteria:** includes preflight-failure test asserting run row + log file.

#### C4-V2: API tests
- `go test ./internal/api -v`

**Pass criteria:** includes run endpoint + latest run visibility for preflight failure.

#### C4-V3: near-live API execution path
1. Start `serve --scheduler=false` in temp env.
2. Create task via API.
3. Trigger `/run`.
4. Poll `/runs/latest` with attempt counters.

**Pass criteria:**
- latest run appears as `failed`,
- error contains usage preflight failure message,
- structured log file exists under `data/logs/<task_id>/`.

#### C4-V4: near-live TUI run-now path
1. Start TUI in `--scheduler=off` against same temp data.
2. Trigger run-now (`r`) through automation/manual.
3. Verify run visibility via API/DB.

**Pass criteria:** failure is visible in run history (not silent).

#### C4-V5: cleanup
- kill tracked server PID,
- verify listener port closed,
- archive relevant logs in handoff references.

### Handoff doc for Cycle 4
Write: `plans/handovers/2026-02-18-cycle4-preflight-visibility.md`

Must include:
- changed executor flow and why,
- before/after behavior table,
- evidence from unit tests and near-live path,
- any residual edge cases.

---

## Cycle 5 — Final integration gate + operator runbook

### Implementation tasks
1. Reconcile all touched files for consistency.
2. Ensure help text/docs mention new modes and doctor command.
3. Resolve any lint/test fallout across packages.

### Validation plan (checkpointed)

#### C5-V1: full test suite
- `go test -v ./...`

#### C5-V2: lint
- `golangci-lint run --timeout=5m`

#### C5-V3: multi-process near-live reliability check
1. Start two `serve --scheduler=true` processes sharing one DB.
2. Create high-frequency recurring task.
3. Observe run cadence over fixed window.

**Pass criteria:** single effective schedule cadence (no duplicate-rate burst).

#### C5-V4: productive developer workflow check
1. `serve --scheduler=false` + mobile API path check.
2. TUI `--scheduler=off` run-now check.
3. `doctor` healthy check.

**Pass criteria:** all three flows usable and deterministic.

### Final handoff doc
Write: `plans/handovers/2026-02-18-final-integration.md`

Must include:
- merged summary of cycles 1–4,
- exact validated commands,
- known limitations and follow-ups,
- recommended day-to-day startup matrix:
  - mobile/API focus,
  - TUI focus,
  - daemon focus.

---

## Ready-to-execute sequence
1. Complete Cycle 3 checkpoints C3-V1..C3-V5, then write Cycle 3 handoff.
2. Complete Cycle 4 checkpoints C4-V1..C4-V5, then write Cycle 4 handoff.
3. Complete Cycle 5 checkpoints C5-V1..C5-V4, then final integration handoff.

No cycle advances until all checkpoint pass criteria are satisfied.
