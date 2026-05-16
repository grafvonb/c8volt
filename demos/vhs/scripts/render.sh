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

OUTPUT_PATH="$(awk '$1 == "Output" { print $2; exit }' "${TAPE}")"
if [[ -z "${OUTPUT_PATH}" ]]; then
  echo "error: tape does not define an Output: ${TAPE}" >&2
  exit 1
fi
if [[ "${OUTPUT_PATH}" != *.gif ]]; then
  echo "error: VHS recordings must produce GIF output only: ${OUTPUT_PATH}" >&2
  exit 1
fi

"${ROOT_DIR}/demos/vhs/scripts/check-vhs.sh"

cd "${ROOT_DIR}"
make build
"${ROOT_DIR}/demos/vhs/scripts/prepare-recording-env.sh"
mkdir -p "${ROOT_DIR}/docs/assets/screencasts"

vhs "${TAPE}"

output_dir="${ROOT_DIR}/$(dirname "${OUTPUT_PATH}")"
output_name="$(basename "${OUTPUT_PATH}")"
output_stem="${output_name%.gif}"
find "${output_dir}" -maxdepth 1 -type f -name "${output_stem}.*" ! -name "${output_name}" -delete
