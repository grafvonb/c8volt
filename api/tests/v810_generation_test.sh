#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

# Purpose:
#   Provide reusable shell-test helpers for isolated Camunda v8.10 client
#   generation. Later tests extend this harness with target, provenance,
#   determinism, and publication cases.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_TMPDIR=""

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
}

cleanup_tmpdir() {
  if [ -n "${TEST_TMPDIR:-}" ]; then
    rm -rf "$TEST_TMPDIR"
  fi
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
}

main() {
  setup_tmpdir
  trap cleanup_tmpdir EXIT
  cd "$REPO_ROOT"

  test_temp_worktree_and_checksums
  test_fake_tool_and_assertions

  echo "ok api/tests/v810_generation_test.sh"
}

if [[ "${BASH_SOURCE[0]}" = "$0" ]]; then
  main "$@"
fi
