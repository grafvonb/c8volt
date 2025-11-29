#!/usr/bin/env zsh
set -euo pipefail

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
  echo "Install either '@redocly/openapi-cli' (openapi) or 'swagger-cli' (swagger-cli) first." >&2
  exit 1
fi

echo "Generated $V2_DIR/$OUTPUT"
