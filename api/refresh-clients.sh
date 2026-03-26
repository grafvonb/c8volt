#!/usr/bin/env bash
# Purpose:
#   Run the full client refresh workflow: fetch upstream specs and regenerate
#   all checked-in Go clients.
#
# Usage:
#   bash api/refresh-clients.sh
#   bash api/refresh-clients.sh --commit
#   bash api/refresh-clients.sh --camunda-tag 8.8.19 --camunda-docs-tag 8.8.196
#
# Notes:
#   This is the top-level entrypoint for the end-to-end workflow. It passes the
#   optional commit mode through to api/3-generate-clients-from-fetched-specs.sh.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

DO_COMMIT=false
CAMUNDA_TAG=""
CAMUNDA_DOCS_TAG=""

usage() {
  cat <<'EOF'
Usage:
  bash api/refresh-clients.sh [--camunda-tag <tag>] [--camunda-docs-tag <tag>] [--commit]

Options:
  --camunda-tag <tag>       Fetch a specific tag from camunda/camunda for the /v2 spec.
  --camunda-docs-tag <tag>  Fetch a specific tag from camunda/camunda-docs for legacy component specs.
  --commit                  Pass commit mode through to api/3-generate-clients-from-fetched-specs.sh.
EOF
}

while (($# > 0)); do
  case "$1" in
    --camunda-tag)
      if (($# < 2)); then
        echo "Missing value for --camunda-tag" >&2
        usage >&2
        exit 1
      fi
      CAMUNDA_TAG="$2"
      shift 2
      ;;
    --camunda-docs-tag)
      if (($# < 2)); then
        echo "Missing value for --camunda-docs-tag" >&2
        usage >&2
        exit 1
      fi
      CAMUNDA_DOCS_TAG="$2"
      shift 2
      ;;
    --commit)
      DO_COMMIT=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [ -n "$CAMUNDA_TAG" ]; then
  "$SCRIPT_DIR/1-fetch-camunda-product-v2-spec.sh" "$CAMUNDA_TAG"
else
  "$SCRIPT_DIR/1-fetch-camunda-product-v2-spec.sh"
fi

if [ -n "$CAMUNDA_DOCS_TAG" ]; then
  "$SCRIPT_DIR/1-fetch-camunda-docs-api-specs.sh" "$CAMUNDA_DOCS_TAG"
else
  "$SCRIPT_DIR/1-fetch-camunda-docs-api-specs.sh"
fi

if [ "$DO_COMMIT" = true ]; then
  "$SCRIPT_DIR/3-generate-clients-from-fetched-specs.sh" --commit
else
  "$SCRIPT_DIR/3-generate-clients-from-fetched-specs.sh"
fi
