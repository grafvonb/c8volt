#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
TEST_TMPDIR="$(mktemp -d "${TMPDIR:-/tmp}/c8volt-run-suite.XXXXXX")"

cleanup() {
  rm -rf "$TEST_TMPDIR"
}
trap cleanup EXIT

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local text="$2"
  grep -F -- "$text" "$file" >/dev/null || fail "$file does not contain: $text"
}

assert_not_contains() {
  local file="$1"
  local text="$2"
  if grep -F -- "$text" "$file" >/dev/null; then
    fail "$file unexpectedly contains: $text"
  fi
}

FAKE_CLI="$TEST_TMPDIR/fake-c8volt"
cat > "$FAKE_CLI" <<'EOF'
#!/usr/bin/env bash
set -eu
printf '%s\n' "$*" >> "${C8VOLT_FAKE_CLI_LOG:?}"
case " $* " in
  *" get cluster version "*)
    printf '%s\n' "${C8VOLT_FAKE_CLUSTER_VERSION:?}"
    ;;
  *" run pi "*)
    printf '1001\n1002\n'
    ;;
  *)
    printf 'ok\n'
    ;;
esac
EOF
chmod +x "$FAKE_CLI"

run_c810_suite() {
  local workdir="$1"
  local version="$2"
  local command_log="$3"
  C8VOLT_FAKE_CLUSTER_VERSION="$version" \
  C8VOLT_FAKE_CLI_LOG="$command_log" \
  C8VOLT_IT_BUILD=0 \
  C8VOLT_IT_BIN="$FAKE_CLI" \
  C8VOLT_IT_CONFIG="$TEST_TMPDIR/config.yaml" \
  C8VOLT_IT_PROFILE="fake-c810" \
  C8VOLT_IT_WORKDIR="$workdir" \
  C8VOLT_IT_FULL_FORCE=0 \
  C8VOLT_IT_KEEP_DATA=1 \
    "$REPO_ROOT/integration/scripts/run-c810-suite.sh"
}

accepted_log="$TEST_TMPDIR/accepted.commands"
run_c810_suite "$TEST_TMPDIR/accepted" "8.10.3" "$accepted_log" > "$TEST_TMPDIR/accepted.output" 2>&1
assert_contains "$accepted_log" "embed deploy --file processdefinitions/C810_SimpleUserTask.bpmn"
assert_contains "$TEST_TMPDIR/accepted/report.md" "target Camunda minor: 8.10"
assert_contains "$TEST_TMPDIR/accepted/report.md" "fixture prefix: C810"

for rejected_version in "8.9.9" "8.100.1"; do
  rejected_slug="${rejected_version//./-}"
  rejected_log="$TEST_TMPDIR/rejected-${rejected_slug}.commands"
  if run_c810_suite "$TEST_TMPDIR/rejected-${rejected_slug}" "$rejected_version" "$rejected_log" > "$TEST_TMPDIR/rejected-${rejected_slug}.output" 2>&1; then
    fail "C810 suite accepted mismatched version $rejected_version"
  fi
  assert_contains "$TEST_TMPDIR/rejected-${rejected_slug}/report.md" "No mutation was submitted."
  assert_not_contains "$rejected_log" "embed deploy"
  assert_not_contains "$rejected_log" "run pi"
  assert_not_contains "$rejected_log" "--auto-confirm"
done

printf 'ok integration/scripts/tests/run-suite_test.sh\n'
