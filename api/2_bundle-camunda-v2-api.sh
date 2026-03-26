#!/usr/bin/env zsh
# Bundle the Camunda v2 OpenAPI spec into a single YAML file.
#
# Defaults:
# - input:  camunda-openapi.yaml
# - output: camunda-openapi-bundled.yaml
# - dir:    camunda-docs/api/camunda/v2
#
# Usage:
#   zsh api/2_bundle-camunda-v2-api.sh
#   zsh api/2_bundle-camunda-v2-api.sh path/to/camunda-docs/api/camunda/v2
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
