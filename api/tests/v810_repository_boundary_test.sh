#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

# Purpose:
#   Guard repository-level Camunda v8.10 boundaries by fingerprinting the
#   stable generated clients before and after an optional validation command.
#   Intentional integration-harness edits are outside this generator boundary.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

PROTECTED_CLIENT_DIRS=(
  "internal/clients/camunda/v88"
  "internal/clients/camunda/v89"
)

fail() {
  echo "FAIL: $*" >&2
  exit 1
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

fingerprint_protected_clients() {
  local dir

  for dir in "${PROTECTED_CLIENT_DIRS[@]}"; do
    if [ ! -d "$dir" ]; then
      fail "missing protected client directory: $dir"
    fi
    (
      cd "$dir"
      find . -type f -print | LC_ALL=C sort | while IFS= read -r path; do
        printf '%s  %s/%s\n' "$(sha256_file "$path")" "$dir" "${path#./}"
      done
    )
  done | sha256_stream
}

main() {
  cd "$REPO_ROOT"

  if [ "${1:-}" = "--" ]; then
    shift
  fi

  local before
  local after
  local command_status=0
  before="$(fingerprint_protected_clients)"

  if (($# > 0)); then
    set +e
    "$@"
    command_status=$?
    set -e
  fi

  after="$(fingerprint_protected_clients)"
  if [ "$before" != "$after" ]; then
    fail "protected v88/v89 client fingerprint changed"
  fi

  if [ "$command_status" -ne 0 ]; then
    return "$command_status"
  fi

  echo "ok api/tests/v810_repository_boundary_test.sh"
}

main "$@"
