#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

# Purpose:
#   Guard repository-level Camunda v8.10 boundaries. The feature may add the
#   isolated v810 generated client, but stable generated clients and live
#   integration assets must remain unchanged and free of v810-specific coverage.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

PROTECTED_CLIENT_DIRS=(
  "internal/clients/camunda/v87"
  "internal/clients/camunda/v88"
  "internal/clients/camunda/v89"
)
INTEGRATION_DIR="integration"
V810_INTEGRATION_PATTERN='v810|v8\.10|8\.10|c810|camunda[[:space:]-]*8\.10'

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

assert_git_clean_for_path() {
  local path="$1"
  local label="$2"

  if ! git diff --quiet -- "$path"; then
    git diff --name-only -- "$path" >&2
    fail "$label has unstaged changes"
  fi

  if ! git diff --cached --quiet -- "$path"; then
    git diff --cached --name-only -- "$path" >&2
    fail "$label has staged changes"
  fi

  local untracked
  untracked="$(git ls-files --others --exclude-standard -- "$path")"
  if [ -n "$untracked" ]; then
    printf '%s\n' "$untracked" >&2
    fail "$label has untracked files"
  fi
}

assert_integration_has_no_v810_paths() {
  local matches

  matches="$(git ls-files --others --cached --exclude-standard -- "$INTEGRATION_DIR" | grep -Ei "$V810_INTEGRATION_PATTERN" || true)"
  if [ -n "$matches" ]; then
    printf '%s\n' "$matches" >&2
    fail "integration contains v810-specific paths"
  fi
}

assert_integration_has_no_v810_content() {
  local matches

  matches="$(rg -n -i "$V810_INTEGRATION_PATTERN" "$INTEGRATION_DIR" || true)"
  if [ -n "$matches" ]; then
    printf '%s\n' "$matches" >&2
    fail "integration contains v810-specific content"
  fi
}

main() {
  cd "$REPO_ROOT"

  local dir
  for dir in "${PROTECTED_CLIENT_DIRS[@]}"; do
    assert_git_clean_for_path "$dir" "$dir"
  done

  assert_git_clean_for_path "$INTEGRATION_DIR" "$INTEGRATION_DIR"
  assert_integration_has_no_v810_paths
  assert_integration_has_no_v810_content

  echo "ok api/tests/v810_repository_boundary_test.sh"
}

main "$@"
