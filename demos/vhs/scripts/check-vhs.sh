#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

set -euo pipefail

required_commands=(vhs ttyd ffmpeg go python3)

for cmd in "${required_commands[@]}"; do
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "error: required command '${cmd}' is not available in PATH" >&2
    exit 1
  fi
done

if ! python3 -c 'import sys; sys.exit(sys.version_info < (3, 8))'; then
  echo "error: Python 3.8 or newer is required" >&2
  exit 1
fi

echo "ok: VHS recording prerequisites are available"
