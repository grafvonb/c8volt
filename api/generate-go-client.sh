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

need() { command -v "$1" >/dev/null 2>&1 || { echo "missing tool: $1" >&2; exit 127; }; }
need oapi-codegen

src="${1:-${SRC:-}}"
out="${2:-${OUT:-}}"
pkg="${3:-${PKG:-}}"

oapi-codegen -generate types,client -package "$pkg" -o "$out" "$src"
