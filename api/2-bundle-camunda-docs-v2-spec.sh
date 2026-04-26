#!/usr/bin/env zsh
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

# Purpose:
#   Bundle the legacy docs-repo /v2 Camunda OpenAPI spec into a single YAML
#   file.
#
# Usage:
#   zsh api/2-bundle-camunda-docs-v2-spec.sh
#   zsh api/2-bundle-camunda-docs-v2-spec.sh path/to/camunda-docs/api/camunda/v2
#
# Notes:
#   This script operates on the docs-repo split YAML files and writes
#   camunda-openapi-bundled.yaml in the target directory.
set -euo pipefail

# Optional first arg overrides the default v2 API directory.
V2_DIR="${1:-camunda-docs/api/camunda/v2}"
INPUT="camunda-openapi.yaml"
OUTPUT="camunda-openapi-bundled.yaml"

cd "$V2_DIR"

if command -v openapi >/dev/null 2>&1; then
  # Redocly OpenAPI CLI
  openapi bundle "$INPUT" -o "$OUTPUT" --ext yaml
elif command -v swagger-cli >/dev/null 2>&1; then
  # Fallback: swagger-cli
  swagger-cli bundle "$INPUT" -o "$OUTPUT" -t yaml
else
  # Fail fast with explicit install guidance when no bundler is available.
  echo "Install either '@redocly/openapi-cli' (openapi) or 'swagger-cli' (swagger-cli) first." >&2
  exit 1
fi

echo "Generated $V2_DIR/$OUTPUT"
