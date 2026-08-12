#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

# Purpose:
#   Guard isolated Camunda v8.10 client generation. These tests encode the
#   target, source, mutation, and publication contracts before the generator is
#   implemented.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_TMPDIR=""
COMMAND_OUTPUT=""
ORIGINAL_PATH="$PATH"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

assert_eq() {
  local want="$1"
  local got="$2"
  local label="${3:-values differ}"

  if [ "$want" != "$got" ]; then
    fail "$label: want <$want>, got <$got>"
  fi
}

assert_file_exists() {
  local path="$1"

  if [ ! -f "$path" ]; then
    fail "expected file to exist: $path"
  fi
}

assert_dir_exists() {
  local path="$1"

  if [ ! -d "$path" ]; then
    fail "expected directory to exist: $path"
  fi
}

assert_not_exists() {
  local path="$1"

  if [ -e "$path" ]; then
    fail "expected path to be absent: $path"
  fi
}

assert_path_exists() {
  local path="$1"

  if [ ! -e "$path" ]; then
    fail "expected path to exist: $path"
  fi
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local label="${3:-missing expected text}"

  if [[ "$haystack" != *"$needle"* ]]; then
    fail "$label: expected <$haystack> to contain <$needle>"
  fi
}

assert_file_contains() {
  local path="$1"
  local needle="$2"
  local label="${3:-missing expected text}"

  assert_file_exists "$path"
  assert_contains "$(cat "$path")" "$needle" "$label"
}

assert_success() {
  local label="$1"
  shift

  if ! "$@"; then
    fail "$label: command failed"
  fi
}

assert_failure() {
  local label="$1"
  shift

  if "$@"; then
    fail "$label: command succeeded unexpectedly"
  fi
}

assert_failure_output_contains() {
  local label="$1"
  local needle="$2"
  shift 2

  if "$@" >"$COMMAND_OUTPUT" 2>&1; then
    fail "$label: command succeeded unexpectedly"
  fi

  assert_file_contains "$COMMAND_OUTPUT" "$needle" "$label"
}

sha256_file() {
  local path="$1"

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$path" | awk '{print $1}'
    return 0
  fi

  shasum -a 256 "$path" | awk '{print $1}'
}

sha256_stream() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum | awk '{print $1}'
    return 0
  fi

  shasum -a 256 | awk '{print $1}'
}

checksum_tree() {
  local root="$1"

  assert_dir_exists "$root"
  (
    cd "$root"
    find . -type f ! -path './.git/*' -print | LC_ALL=C sort | while IFS= read -r path; do
      printf '%s  %s\n' "$(sha256_file "$path")" "${path#./}"
    done
  ) | sha256_stream
}

setup_tmpdir() {
  TEST_TMPDIR="$(mktemp -d "${TMPDIR:-/tmp}/c8volt-v810-generation.XXXXXX")"
  export TEST_TMPDIR
  ORIGINAL_PATH="$PATH"
}

cleanup_tmpdir() {
  if [ -n "${TEST_TMPDIR:-}" ]; then
    rm -rf "$TEST_TMPDIR"
  fi
}

reset_path() {
  export PATH="$ORIGINAL_PATH"
}

make_git_fixture_repo() {
  local name="$1"
  local repo="$TEST_TMPDIR/$name"

  mkdir -p "$repo"
  git -C "$repo" init -q
  git -C "$repo" config user.email "v810-tests@example.invalid"
  git -C "$repo" config user.name "V810 Tests"
  printf 'fixture\n' >"$repo/README.md"
  git -C "$repo" add README.md
  git -C "$repo" commit -q -m "initial fixture"
  printf '%s\n' "$repo"
}

make_temp_worktree() {
  local repo="$1"
  local name="$2"
  local worktree="$TEST_TMPDIR/$name"

  git -C "$repo" worktree add -q --detach "$worktree" HEAD
  printf '%s\n' "$worktree"
}

make_repo_worktree() {
  local name="$1"
  local worktree="$TEST_TMPDIR/$name"

  git -C "$REPO_ROOT" worktree add -q --detach "$worktree" HEAD
  printf '%s\n' "$worktree"
}

v810_output_dir() {
  local worktree="$1"

  printf '%s\n' "$worktree/internal/clients/camunda/v810/camunda"
}

assert_no_v810_publication() {
  local worktree="$1"

  assert_not_exists "$(v810_output_dir "$worktree")/client.gen.go"
  assert_not_exists "$(v810_output_dir "$worktree")/provenance.json"
  assert_not_exists "$worktree/internal/clients/camunda/v810alpha"
  assert_not_exists "$worktree/internal/clients/camunda/v810rc"
}

checksum_optional_v810_publication() {
  local worktree="$1"
  local output_dir

  output_dir="$(v810_output_dir "$worktree")"
  if [ ! -e "$output_dir" ]; then
    printf 'absent\n'
    return 0
  fi

  checksum_tree "$output_dir"
}

assert_v810_publication_unchanged() {
  local worktree="$1"
  local before="$2"
  local after

  after="$(checksum_optional_v810_publication "$worktree")"
  assert_eq "$before" "$after" "v810 publication changed after failed generation"
  assert_not_exists "$worktree/internal/clients/camunda/v810alpha"
  assert_not_exists "$worktree/internal/clients/camunda/v810rc"
}

write_existing_v810_publication() {
  local worktree="$1"
  local output_dir

  output_dir="$(v810_output_dir "$worktree")"
  mkdir -p "$output_dir"
  printf 'package camunda\n// existing client\n' >"$output_dir/client.gen.go"
  printf '{"schemaVersion":1,"existing":true}\n' >"$output_dir/provenance.json"
}

checksum_v810_publication() {
  local worktree="$1"
  local output_dir

  output_dir="$(v810_output_dir "$worktree")"
  (
    cd "$output_dir"
    sha256_file client.gen.go
    sha256_file provenance.json
  ) | sha256_stream
}

fake_tool_log() {
  printf '%s\n' "$TEST_TMPDIR/fake-tools.log"
}

install_fake_tool() {
  local name="$1"
  local exit_code="${2:-0}"
  local stdout="${3:-}"
  local bin_dir="$TEST_TMPDIR/bin"
  local tool_path="$bin_dir/$name"

  mkdir -p "$bin_dir"
  cat >"$tool_path" <<EOF
#!/usr/bin/env bash
printf '%s %s\n' "$name" "\$*" >>"\${V810_FAKE_TOOL_LOG:?}"
if [ -n "$stdout" ]; then
  printf '%s\n' "$stdout"
fi
exit "$exit_code"
EOF
  chmod +x "$tool_path"
  export V810_FAKE_TOOL_LOG
  V810_FAKE_TOOL_LOG="$(fake_tool_log)"
  export PATH="$bin_dir:$PATH"
}

install_fake_git_commit_mismatch() {
  local real_git
  local bin_dir="$TEST_TMPDIR/bin"
  local tool_path="$bin_dir/git"

  real_git="$(command -v git)"
  mkdir -p "$bin_dir"
  cat >"$tool_path" <<EOF
#!/usr/bin/env bash
for arg in "\$@"; do
  if [ "\$arg" = "ls-remote" ]; then
    printf '%s\t%s\n' "0000000000000000000000000000000000000000" "refs/tags/8.10.0-alpha4^{}"
    exit 0
  fi
done
exec "$real_git" "\$@"
EOF
  chmod +x "$tool_path"
  export PATH="$bin_dir:$PATH"
}

break_v810_mutation_effect() {
  local worktree="$1"
  local mutation="$worktree/api/mutations/mutate-search-query-schemas.py"

  printf '%s\n' '#!/usr/bin/env python3' 'import sys' 'sys.exit(0)' >"$mutation"
  chmod +x "$mutation"
}

test_temp_worktree_and_checksums() {
  local repo
  local worktree
  local before
  local after

  repo="$(make_git_fixture_repo source-repo)"
  worktree="$(make_temp_worktree "$repo" generated-worktree)"

  assert_path_exists "$worktree/.git"
  assert_file_exists "$worktree/README.md"
  before="$(checksum_tree "$worktree")"
  printf 'changed\n' >"$worktree/generated.txt"
  after="$(checksum_tree "$worktree")"

  if [ "$before" = "$after" ]; then
    fail "checksum_tree did not detect a worktree content change"
  fi
}

test_fake_tool_and_assertions() {
  local output
  local log

  install_fake_tool redocly 0 "redocly 2.0.0"
  output="$(redocly --version)"
  log="$(cat "$(fake_tool_log)")"

  assert_eq "redocly 2.0.0" "$output" "fake tool stdout"
  assert_contains "$log" "redocly --version" "fake tool invocation log"
  assert_success "true succeeds" true
  assert_failure "false fails" false
  reset_path
}

test_refresh_rejects_unknown_target_before_writes() {
  local worktree
  local before

  worktree="$(make_repo_worktree unknown-target-worktree)"
  before="$(checksum_optional_v810_publication "$worktree")"
  COMMAND_OUTPUT="$TEST_TMPDIR/unknown-target.out"

  assert_failure_output_contains \
    "unknown target is rejected" \
    "Unknown target: v999" \
    "$worktree/api/refresh-clients.sh" --target v999 --camunda-tag 8.10.0-alpha4
  assert_v810_publication_unchanged "$worktree" "$before"
}

test_refresh_rejects_missing_v810_tag_before_writes() {
  local worktree
  local before

  worktree="$(make_repo_worktree missing-tag-worktree)"
  before="$(checksum_optional_v810_publication "$worktree")"
  COMMAND_OUTPUT="$TEST_TMPDIR/missing-tag.out"

  assert_failure_output_contains \
    "v810 target requires explicit tag" \
    "Missing value for --camunda-tag" \
    "$worktree/api/refresh-clients.sh" --target v810
  assert_v810_publication_unchanged "$worktree" "$before"
}

test_refresh_rejects_non_810_source_tag_before_writes() {
  local worktree
  local before

  worktree="$(make_repo_worktree invalid-tag-worktree)"
  before="$(checksum_optional_v810_publication "$worktree")"
  COMMAND_OUTPUT="$TEST_TMPDIR/invalid-tag.out"

  assert_failure_output_contains \
    "non-8.10 source tag is rejected" \
    "Invalid Camunda tag for v810: 8.9.0" \
    "$worktree/api/refresh-clients.sh" --target v810 --camunda-tag 8.9.0
  assert_v810_publication_unchanged "$worktree" "$before"
}

test_refresh_rejects_commit_mismatch_before_writes() {
  local worktree
  local before

  worktree="$(make_repo_worktree commit-mismatch-worktree)"
  before="$(checksum_optional_v810_publication "$worktree")"
  COMMAND_OUTPUT="$TEST_TMPDIR/commit-mismatch.out"

  reset_path
  install_fake_git_commit_mismatch
  assert_failure_output_contains \
    "pinned commit mismatch is rejected" \
    "Commit mismatch for v810 tag 8.10.0-alpha4" \
    "$worktree/api/refresh-clients.sh" --target v810 --camunda-tag 8.10.0-alpha4
  assert_v810_publication_unchanged "$worktree" "$before"
  reset_path
}

test_refresh_rejects_output_escape_before_writes() {
  local worktree
  local before

  worktree="$(make_repo_worktree output-escape-worktree)"
  before="$(checksum_optional_v810_publication "$worktree")"
  COMMAND_OUTPUT="$TEST_TMPDIR/output-escape.out"

  assert_failure_output_contains \
    "escaped output path is rejected" \
    "V810 output path escapes repository" \
    "$worktree/api/refresh-clients.sh" --target ../v810 --camunda-tag 8.10.0-alpha4
  assert_v810_publication_unchanged "$worktree" "$before"
  assert_not_exists "$worktree/../escaped-v810"
}

test_refresh_names_missing_generation_tool_before_writes() {
  local worktree
  local before

  worktree="$(make_repo_worktree missing-tool-worktree)"
  before="$(checksum_optional_v810_publication "$worktree")"
  COMMAND_OUTPUT="$TEST_TMPDIR/missing-tool.out"

  assert_failure_output_contains \
    "missing redocly is reported" \
    "missing tool: redocly" \
    env PATH="/bin:/usr/bin" "$worktree/api/refresh-clients.sh" --target v810 --camunda-tag 8.10.0-alpha4
  assert_v810_publication_unchanged "$worktree" "$before"
}

test_refresh_rejects_mutation_no_op_before_publication() {
  local worktree
  local before

  worktree="$(make_repo_worktree mutation-no-op-worktree)"
  before="$(checksum_optional_v810_publication "$worktree")"
  break_v810_mutation_effect "$worktree"
  COMMAND_OUTPUT="$TEST_TMPDIR/mutation-no-op.out"

  assert_failure_output_contains \
    "mutation no-op is rejected" \
    "mutation produced no expected effect: api/mutations/mutate-search-query-schemas.py" \
    "$worktree/api/refresh-clients.sh" --target v810 --camunda-tag 8.10.0-alpha4
  assert_v810_publication_unchanged "$worktree" "$before"
}

test_refresh_failure_preserves_existing_publication_atomically() {
  local worktree
  local before
  local after

  worktree="$(make_repo_worktree atomic-publication-worktree)"
  write_existing_v810_publication "$worktree"
  before="$(checksum_v810_publication "$worktree")"
  COMMAND_OUTPUT="$TEST_TMPDIR/atomic-publication.out"

  assert_failure_output_contains \
    "failed generation leaves existing publication intact" \
    "missing tool: redocly" \
    env PATH="/bin:/usr/bin" "$worktree/api/refresh-clients.sh" --target v810 --camunda-tag 8.10.0-alpha4

  after="$(checksum_v810_publication "$worktree")"
  assert_eq "$before" "$after" "existing v810 publication changed after failed generation"
}

main() {
  setup_tmpdir
  trap cleanup_tmpdir EXIT
  cd "$REPO_ROOT"

  test_temp_worktree_and_checksums
  test_fake_tool_and_assertions
  test_refresh_rejects_unknown_target_before_writes
  test_refresh_rejects_missing_v810_tag_before_writes
  test_refresh_rejects_non_810_source_tag_before_writes
  test_refresh_rejects_commit_mismatch_before_writes
  test_refresh_rejects_output_escape_before_writes
  test_refresh_names_missing_generation_tool_before_writes
  test_refresh_rejects_mutation_no_op_before_publication
  test_refresh_failure_preserves_existing_publication_atomically

  echo "ok api/tests/v810_generation_test.sh"
}

if [[ "${BASH_SOURCE[0]}" = "$0" ]]; then
  main "$@"
fi
