#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

# Retain the existing 8.10 entry point; the shared driver owns generation.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
for argument in "$@"; do
  if [ "$argument" = --target ]; then
    echo "Unknown argument: --target" >&2
    exit 1
  fi
done
exec bash "$SCRIPT_DIR/generate-camunda-client.sh" --target v810 "$@"
