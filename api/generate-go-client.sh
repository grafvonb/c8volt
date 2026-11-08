#!/bin/bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

# Purpose:
#   Generate a single Go client file from one OpenAPI specification using
#   oapi-codegen.
#
# Usage:
#   bash api/generate-go-client.sh <spec> <output> <package>
#
# Notes:
#   This is a low-level helper used by the higher-level generation workflow
#   scripts in this directory.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/generation-tools.sh"

src="${1:-${SRC:-}}"
out="${2:-${OUT:-}}"
pkg="${3:-${PKG:-}}"

[[ "$src" = /* || "$src" = https://* || "$src" = http://* ]] || src="$PWD/$src"
[[ "$out" = /* ]] || out="$PWD/$out"

generation_oapi_codegen -config "$SCRIPT_DIR/oapi-codegen.yaml" -generate types,client -package "$pkg" -o "$out" "$src"
