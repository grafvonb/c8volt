#!/usr/bin/env bash

# Focused live checks; invoke explicitly against a disposable Camunda target.
# Reuse the shell suite reporter and checked-in C810 BPMN fixtures.
set +x
set -uo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck source=integration/scripts/lib/suite-lib.sh
. "$ROOT_DIR/integration/scripts/lib/suite-lib.sh"

IT_SUITE_NAME=c810-suspended
IT_TARGET_VERSION=8.10
IT_FIXTURE_PREFIX=C810
IT_CONFIG="${C8VOLT_IT_CONFIG:-./config.yaml}"
IT_PROFILE="${C8VOLT_IT_PROFILE:-kind-camunda-platform-local-c810}"
IT_BIN="${C8VOLT_IT_BIN:-/tmp/c8volt-it-c810-suspended}"
IT_FULL_FORCE=0
IT_WORKDIR="${C8VOLT_IT_WORKDIR:-$(mktemp -d /tmp/c8volt-it-c810-suspended.XXXXXX)}"
IT_REPORT="$IT_WORKDIR/report.md"
IT_PROGRESS="$IT_WORKDIR/progress.tsv"
IT_RUN_ID="c810-suspended-$(date -u +%Y%m%dT%H%M%SZ)-$$"
IT_API_URL="${C8VOLT_IT_CAMUNDA_URL:-}"

# Keep all CLI calls on the same explicitly selected API.
if [[ ! "$IT_API_URL" =~ ^https?://[^/?#@]+(/[^?#]*)?/v2/?$ ]]; then
  printf 'Set C8VOLT_IT_CAMUNDA_URL to the target Camunda REST URL ending in /v2.\n' >&2
  exit 2
fi
IT_API_URL="${IT_API_URL%/}"
export C8VOLT_APIS_CAMUNDA_API_BASE_URL="$IT_API_URL"
for dependency in jq; do
  command -v "$dependency" >/dev/null || exit 2
done
cd "$ROOT_DIR" || exit 2
it_init_report
owned_roots="$IT_WORKDIR/data/owned-roots.keys"
: > "$owned_roots"

# Preserve the established shell config/profile convention and bound waits.
c8() {
  "$IT_BIN" --config "$IT_CONFIG" --profile "$IT_PROFILE" --timeout 15s --no-err-codes=false "$@"
}

# Snapshot assertions parse one envelope, never a loose match on an error string.
# A missing instance must be an empty successful result or an explicit not_found.
snapshot() {
  local key="$1" state="$2" label="$3" rc=0
  local file="$IT_WORKDIR/data/${label}-${key}.json"
  c8 get pi --key "$key" --json > "$file" 2> "$file.stderr" || rc=$?
  if [ "$state" = ABSENT ]; then
    jq -e -s 'length == 1 and (.[0] |
      (.outcome == "failed" and .class == "not_found") or
      (.outcome == "succeeded" and (.payload | type) == "object" and
        (.payload.total // 0) == 0 and ((.payload.items // []) | length) == 0))' "$file"
  else
    [ "$rc" -eq 0 ] || return "$rc"
    jq -e -s --arg key "$key" --arg state "$state" 'length == 1 and (.[0] |
      .outcome == "succeeded" and (.payload.items | length) == 1 and
      (.payload.items[0] | .key == $key and
        (if $state == "TERMINAL" then (.state == "CANCELED" or .state == "TERMINATED") else .state == $state end) and
        (if $state == "SUSPENDED" then ((.suspendedDate // "") | length) > 0 else true end)))' "$file"
  fi
}

# Suspension setup uses the public command and requires observed confirmation.
suspend_fixture() {
  local key="$1"
  local file="$IT_WORKDIR/data/suspend-${key}.json"
  c8 suspend pi --key "$key" --auto-confirm --backoff-timeout 60s --json \
    > "$file" 2> "$file.stderr" || return 1
  jq -e -s --arg key "$key" 'length == 1 and (.[0] |
    .outcome == "succeeded" and (.payload.items | length) == 1 and
    (.payload.items[0] | .key == $key and .status == "confirmed" and
      .submission == "accepted" and .confirmation == "confirmed" and
      .hasObservation == true and .lastObservedState == "SUSPENDED"))' "$file"
}

# Cleanup is scoped to created roots, independent of scenario acceptance. Shared
# embedded definitions are retained; a cleanup failure never hides a test failure.
finish() {
  local result="$1" root
  trap - EXIT
  while IFS= read -r root; do
    if ! snapshot "$root" ABSENT cleanup >/dev/null 2>&1; then
      it_run_case Cleanup "delete-root-$root" mutation pass \
        c8 delete pi --key "$root" --force --automation --backoff-timeout 60s --json || result=1
      it_run_case Cleanup "root-absent-$root" read-only pass \
        c8 expect pi --key "$root" --state absent --backoff-timeout 60s --json || result=1
    fi
  done < "$owned_roots"
  it_finish_report
  printf '\nReport: %s\nProgress: %s\n' "$IT_REPORT" "$IT_PROGRESS"
  if [ "$IT_FAILS" -gt 0 ] || [ "$IT_EXPECTED_FAIL_MISSES" -gt 0 ]; then result=1; fi
  exit "$result"
}
trap 'finish $?' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# Wait for the existing four-member parent/mid/two-leaf shape before suspension.
capture_tree() {
  local root="$1" file="$2" attempt
  for ((attempt=0; attempt<30; attempt++)); do
    c8 walk pi --key "$root" --flat --json > "$file" 2> "$file.stderr" || return 1
    if jq -e -s --arg root "$root" 'length == 1 and (.[0] |
      .outcome == "succeeded" and .payload.rootKey == $root and
      (.payload.keys | length) == 4 and (.payload.keys | unique | length) == 4 and
      ([.payload.items[].key] | sort) == (.payload.keys | sort) and
      ([.payload.items[] | select(.key == $root and .bpmnProcessId == "C810_MultipleSubProcessesParent")] | length) == 1 and
      ([.payload.items[] | select(.bpmnProcessId == "C810_SimpleParent")] | length) == 1 and
      ([.payload.items[] | select(.bpmnProcessId == "C810_SimpleUserTask")] | length) == 2)' "$file" >/dev/null; then
      return 0
    fi
    sleep 2
  done
  return 1
}

# Prove the whole mixed tree, both before mutation and after non-force refusal.
check_states() {
  local group="$1" keys="$2" suspended="$3" phase="$4" key state
  while IFS= read -r key; do
    state=ACTIVE
    if [ "$key" = "$suspended" ]; then state=SUSPENDED; fi
    it_run_case "$group" "$phase-$key" read-only pass snapshot "$key" "$state" "$group-$phase" || return 1
  done < "$keys"
}

# Each case owns a fresh root. Child selections exercise the existing root
# resolution; --no-wait checks acceptance separately from eventual completion.
run_scenario() {
  local operation="$1" shape="$2" mode="$3" group model root selected suspended key rc=0
  group="$operation-$shape-$mode"
  local prefix="$IT_WORKDIR/data/$group"
  model=C810_SimpleUserTask
  if [ "$shape" != suspended-root ]; then model=C810_MultipleSubProcessesParent; fi
  it_run_capture "$group" create mutation pass "$prefix-created.keys" \
    c8 run pi --bpmn-process-id "$model" --count 1 --keys-only \
      --vars "{\"releaseRunId\":\"$IT_RUN_ID\",\"scenario\":\"$group\"}" || rc=$?
  # Save any returned keys before asserting success so partial creation is owned.
  awk '/^[0-9]+$/ { print }' "$prefix-created.keys" >> "$owned_roots"
  [ "$rc" -eq 0 ] || return 1
  it_run_case "$group" created-one-key read-only pass \
    awk 'NF { if ($0 !~ /^[0-9]+$/) exit 1; n++ } END { if (n != 1) exit 1 }' "$prefix-created.keys" || return 1
  root="$(it_first_key "$prefix-created.keys")"
  selected="$root"
  printf '%s\n' "$root" > "$prefix-family.keys"
  if [ "$shape" != suspended-root ]; then
    it_run_case "$group" discover-family read-only pass capture_tree "$root" "$prefix-tree.json" || return 1
    jq -r '.payload.keys[]' "$prefix-tree.json" > "$prefix-family.keys" || return 1
    selected="$(jq -r '.payload.items[] | select(.bpmnProcessId == "C810_SimpleParent") | .key' "$prefix-tree.json")"
  fi
  suspended="$root"
  if [ "$shape" = active-root-suspended-child ]; then suspended="$selected"; fi
  it_run_case "$group" suspend-fixture mutation pass suspend_fixture "$suspended" || return 1
  it_run_case "$group" await-suspended read-only pass \
    c8 expect pi --key "$suspended" --state suspended --backoff-timeout 60s --json || return 1
  check_states "$group" "$prefix-family.keys" "$suspended" before || return 1

  local args=("$operation" pi --key "$selected" --automation --json --backoff-timeout 60s)
  local outcome=succeeded
  if [ "$mode" = no-wait ]; then args+=(--no-wait); outcome=accepted; fi
  if [ "$operation" = delete ]; then
    rc=0
    it_run_capture "$group" without-force mutation fail "$prefix-refused.json" c8 "${args[@]}" || rc=$?
    it_run_case "$group" refusal-exit read-only pass test "$rc" -eq 1 || return 1
    it_run_case "$group" refusal-contract read-only pass jq -e -s 'length == 1 and (.[0] |
      .outcome == "failed" and .class == "local_precondition" and
      (.detail.message | contains("not in a final state") and contains("no delete request was submitted")))' "$prefix-refused.json" || return 1
    check_states "$group" "$prefix-family.keys" "$suspended" after-refusal || return 1
    args+=(--force)
  fi
  it_run_capture "$group" mutate mutation pass "$prefix-result.json" c8 "${args[@]}" || return 1
  it_run_case "$group" result-contract read-only pass jq -e -s --arg root "$root" --arg outcome "$outcome" \
    'length == 1 and (.[0] | .outcome == $outcome and (.payload.items | length) == 1 and
      (.payload.items[0] | .key == $root and .ok == true))' "$prefix-result.json" || return 1
  while IFS= read -r key; do
    if [ "$mode" = no-wait ]; then
      local states=(--state absent)
      if [ "$operation" = cancel ]; then states=(--state canceled --state terminated); fi
      it_run_case "$group" "eventual-$key" read-only pass \
        c8 expect pi --key "$key" "${states[@]}" --backoff-timeout 60s --json || return 1
    fi
    local final_state=ABSENT
    if [ "$operation" = cancel ]; then final_state=TERMINAL; fi
    it_run_case "$group" "final-$key" read-only pass snapshot "$key" "$final_state" "$group-final" || return 1
  done < "$prefix-family.keys"
}

if [ "${C8VOLT_IT_BUILD:-1}" = 1 ]; then
  it_run_case Build go-build local pass go build -o "$IT_BIN" . || exit 2
else
  it_run_case Build existing-binary local pass test -x "$IT_BIN" || exit 2
fi
it_run_capture Preflight cluster-version read-only pass "$IT_WORKDIR/data/version.txt" c8 get cluster version || exit 2
it_require_exact_minor Preflight cluster-version "$IT_TARGET_VERSION" || exit 2
it_run_case Preflight suspension-patch read-only pass \
  test "$(tr -d '\r\n' < "$IT_WORKDIR/data/version.txt")" = 8.10.4 || exit 2
printf '\nSuspension setup and tested mutations use c8volt; suspension requires observed confirmation.\n' >> "$IT_REPORT"
for fixture in C810_SimpleUserTask C810_SimpleParent C810_MultipleSubProcessesParent; do
  it_run_case Setup "deploy-$fixture" mutation pass c8 embed deploy --file "processdefinitions/$fixture.bpmn" || exit 2
done
for operation in cancel delete; do
  for shape in suspended-root active-root-suspended-child suspended-root-active-child; do
    for mode in wait no-wait; do
      run_scenario "$operation" "$shape" "$mode" || exit 1
    done
  done
done
