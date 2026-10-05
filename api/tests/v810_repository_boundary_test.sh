#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

# Purpose:
#   Guard repository-level Camunda v8.10 generation boundaries. Stable generated
#   clients and live integration assets must remain unchanged.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

PROTECTED_CLIENT_DIRS=(
  "internal/clients/camunda/v88"
  "internal/clients/camunda/v89"
)
INTEGRATION_DIR="integration"

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

main() {
  cd "$REPO_ROOT"

  local dir
  for dir in "${PROTECTED_CLIENT_DIRS[@]}"; do
    assert_git_clean_for_path "$dir" "$dir"
  done

  assert_git_clean_for_path "$INTEGRATION_DIR" "$INTEGRATION_DIR"

  echo "ok api/tests/v810_repository_boundary_test.sh"
}

main "$@"
