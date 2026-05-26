#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="${BIN:-$ROOT_DIR/c8volt}"
CONFIG="${CONFIG:-$ROOT_DIR/config.yaml}"
FAST_BPMN="${FAST_BPMN:-C89_NoOpCompletion_Process}"
ACTIVE_BPMN="${ACTIVE_BPMN:-C89_SimpleUserTask_Process}"
FAST_FIXTURE="${FAST_FIXTURE:-processdefinitions/C89_NoOpCompletionProcess.bpmn}"
ACTIVE_FIXTURE="${ACTIVE_FIXTURE:-processdefinitions/C89_SimpleUserTaskProcess.bpmn}"

active_keys=()

log() {
  printf '[issue-225-smoke] %s\n' "$*"
}

fail() {
  printf '[issue-225-smoke] ERROR: %s\n' "$*" >&2
  exit 1
}

require_tool() {
  command -v "$1" >/dev/null 2>&1 || fail "required tool '$1' is not available"
}

run_c8volt() {
  "$BIN" --config "$CONFIG" "$@"
}

cleanup() {
  if [[ ${#active_keys[@]} -eq 0 ]]; then
    return
  fi
  log "cleanup: canceling active smoke process instance(s): ${active_keys[*]}"
  printf '%s\n' "${active_keys[@]}" | run_c8volt cancel pi --auto-confirm --no-wait - >/dev/null || true
}
trap cleanup EXIT

require_tool jq

[[ -x "$BIN" ]] || fail "binary is not executable: $BIN"
[[ -f "$CONFIG" ]] || fail "config file not found: $CONFIG"

log "using binary: $BIN"
log "using config: $CONFIG"
log "deploying embedded fixtures"
run_c8volt embed deploy --file "$FAST_FIXTURE" >/dev/null
run_c8volt embed deploy --file "$ACTIVE_FIXTURE" >/dev/null

log "smoke 1: fast completion succeeds and normal output shows COMPLETED"
normal_output="$(run_c8volt run pi -b "$FAST_BPMN")"
printf '%s\n' "$normal_output" | grep -q 'COMPLETED' ||
  fail "normal run output did not include COMPLETED; output was: $normal_output"

log "smoke 2: JSON output carries observed COMPLETED state"
json_output="$(run_c8volt --json run pi -b "$FAST_BPMN")"
json_state="$(printf '%s\n' "$json_output" | jq -r '.payload.items[0].state // empty')"
[[ "$json_state" == "COMPLETED" ]] ||
  fail "JSON run state was '$json_state', expected COMPLETED; output was: $json_output"

log "smoke 3: keys-only fast completion output pipes into strict completed expectation"
completed_key="$(run_c8volt run pi -b "$FAST_BPMN" --keys-only)"
[[ "$completed_key" =~ ^[0-9]+$ ]] ||
  fail "keys-only completed output was not a single numeric key: $completed_key"
printf '%s\n' "$completed_key" | run_c8volt expect pi --state completed - >/dev/null

log "smoke 4: keys-only long-running output pipes into strict active expectation"
active_key="$(run_c8volt run pi -b "$ACTIVE_BPMN" --keys-only)"
[[ "$active_key" =~ ^[0-9]+$ ]] ||
  fail "keys-only active output was not a single numeric key: $active_key"
active_keys+=("$active_key")
printf '%s\n' "$active_key" | run_c8volt expect pi --state active - >/dev/null

log "all issue #225 smoke checks passed"
