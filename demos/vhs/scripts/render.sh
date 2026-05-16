#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SCENARIO="${1:-fast-start}"
TAPE="${ROOT_DIR}/demos/vhs/${SCENARIO}.tape"

if [[ ! -f "${TAPE}" ]]; then
  echo "error: tape not found: ${TAPE}" >&2
  exit 1
fi

"${ROOT_DIR}/demos/vhs/scripts/check-vhs.sh"

cd "${ROOT_DIR}"
make build
"${ROOT_DIR}/demos/vhs/scripts/prepare-recording-env.sh"
mkdir -p "${ROOT_DIR}/docs/assets/screencasts"

vhs "${TAPE}"

