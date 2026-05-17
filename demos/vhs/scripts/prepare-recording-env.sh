#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SANDBOX_DIR="${C8VOLT_VHS_SANDBOX_DIR:-/tmp/c8volt-vhs}"
BIN_DIR="${SANDBOX_DIR}/bin"
HOME_DIR="${SANDBOX_DIR}/home"
CONFIG_PATH="${BIN_DIR}/config.yaml"

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "error: required environment variable '${name}' is not set" >&2
    exit 1
  fi
}

yaml_quote() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/ }"
  printf '"%s"' "${value}"
}

write_scope_if_set() {
  local key="$1"
  local value="$2"
  if [[ -n "${value}" ]]; then
    printf '      %s: ' "${key}"
    yaml_quote "${value}"
    printf '\n'
  fi
}

require_env C8VOLT_VHS_C89_BASE_URL
require_env C8VOLT_VHS_OAUTH_TOKEN_URL
require_env C8VOLT_VHS_OAUTH_CLIENT_ID
require_env C8VOLT_VHS_OAUTH_CLIENT_SECRET

if [[ ! -x "${ROOT_DIR}/bin/c8volt" ]]; then
  echo "error: ${ROOT_DIR}/bin/c8volt does not exist; run 'make build' first" >&2
  exit 1
fi

rm -rf "${SANDBOX_DIR}"
mkdir -p "${BIN_DIR}" "${HOME_DIR}/.config"
chmod 700 "${SANDBOX_DIR}" "${HOME_DIR}"

cp "${ROOT_DIR}/bin/c8volt" "${BIN_DIR}/c8volt"
chmod 700 "${BIN_DIR}/c8volt"

{
  printf 'active_profile: c89\n\n'
  printf 'auth:\n'
  printf '  mode: oauth2\n'
  printf '  oauth2:\n'
  printf '    token_url: '
  yaml_quote "${C8VOLT_VHS_OAUTH_TOKEN_URL}"
  printf '\n'
  printf '    client_id: '
  yaml_quote "${C8VOLT_VHS_OAUTH_CLIENT_ID}"
  printf '\n'
  printf '    client_secret: '
  yaml_quote "${C8VOLT_VHS_OAUTH_CLIENT_SECRET}"
  printf '\n'
  if [[ -n "${C8VOLT_VHS_OAUTH_CAMUNDA_SCOPE:-}${C8VOLT_VHS_OAUTH_OPERATE_SCOPE:-}${C8VOLT_VHS_OAUTH_TASKLIST_SCOPE:-}" ]]; then
    printf '    scopes:\n'
    write_scope_if_set camunda_api "${C8VOLT_VHS_OAUTH_CAMUNDA_SCOPE:-}"
    write_scope_if_set operate_api "${C8VOLT_VHS_OAUTH_OPERATE_SCOPE:-}"
    write_scope_if_set tasklist_api "${C8VOLT_VHS_OAUTH_TASKLIST_SCOPE:-}"
  fi
  printf '\n'
  printf 'profiles:\n'
  printf '  c89:\n'
  printf '    app:\n'
  printf '      camunda_version: "8.9"\n'
  printf '    apis:\n'
  printf '      camunda_api:\n'
  printf '        base_url: '
  yaml_quote "${C8VOLT_VHS_C89_BASE_URL}"
  printf '\n\n'
  printf 'log:\n'
  printf '  format: plain-time\n'
  printf '  level: info\n'
} >"${CONFIG_PATH}"

chmod 600 "${CONFIG_PATH}"

echo "ok: prepared VHS recording sandbox at ${SANDBOX_DIR}"
