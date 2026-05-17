#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

set -euo pipefail

required_commands=(vhs ttyd ffmpeg go)

for cmd in "${required_commands[@]}"; do
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "error: required command '${cmd}' is not available in PATH" >&2
    exit 1
  fi
done

required_env=(
  C8VOLT_VHS_C89_BASE_URL
  C8VOLT_VHS_OAUTH_TOKEN_URL
  C8VOLT_VHS_OAUTH_CLIENT_ID
  C8VOLT_VHS_OAUTH_CLIENT_SECRET
)

for name in "${required_env[@]}"; do
  if [[ -z "${!name:-}" ]]; then
    echo "error: required environment variable '${name}' is not set" >&2
    exit 1
  fi
done

echo "ok: VHS recording prerequisites are available"
