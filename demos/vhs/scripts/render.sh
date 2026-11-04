#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
if [[ $# -lt 1 ]]; then
  echo "usage: $0 <tape-name> [--config <path>] [--profile <name>]" >&2
  echo >&2
  echo "available tapes:" >&2
  find "${ROOT_DIR}/demos/vhs" -maxdepth 1 -type f -name '*.tape' -exec basename {} .tape \; | sort | sed 's/^/  /' >&2
  exit 1
fi

SCENARIO="${1%.tape}"
shift
if [[ ! "${SCENARIO}" =~ ^[a-z0-9-]+$ ]]; then
  echo "error: invalid tape name" >&2
  exit 1
fi
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

make -C "${ROOT_DIR}" build
RECORDING_DIR="$("${ROOT_DIR}/demos/vhs/scripts/prepare-recording-env.sh" "${TAPE}" "$@")"
trap 'rm -rf -- "${RECORDING_DIR}"' EXIT
cd "${ROOT_DIR}"
mkdir -p "${ROOT_DIR}/docs/assets/screencasts"

vhs "${RECORDING_DIR}/recording.tape"

output_dir="${ROOT_DIR}/$(dirname "${OUTPUT_PATH}")"
output_name="$(basename "${OUTPUT_PATH}")"
output_stem="${output_name%.gif}"
find "${output_dir}" -maxdepth 1 -type f -name "${output_stem}.*" ! -name "${output_name}" -delete
