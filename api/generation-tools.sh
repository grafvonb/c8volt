#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

# Shared tool selection for client generation. Source this file from API scripts.
GENERATION_REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GENERATION_REDOCLY_VERSION="2.76.0"

generation_redocly() {
  local actual
  command -v redocly >/dev/null 2>&1 || {
    echo "missing tool: redocly" >&2
    return 127
  }
  actual="$(redocly --version)" || return
  if [ "$actual" != "$GENERATION_REDOCLY_VERSION" ]; then
    echo "redocly version mismatch: need $GENERATION_REDOCLY_VERSION, got $actual; install with npm install --global @redocly/cli@$GENERATION_REDOCLY_VERSION" >&2
    return 1
  fi
  command redocly "$@"
}

generation_oapi_codegen() {
  command -v go >/dev/null 2>&1 || {
    echo "missing tool: go" >&2
    return 127
  }
  GOWORK=off go -C "$GENERATION_REPO_ROOT" tool oapi-codegen "$@"
}
