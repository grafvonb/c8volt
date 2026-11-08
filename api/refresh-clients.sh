#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

# Purpose:
#   Fetch upstream specs and regenerate OAuth2 and the runtime-used Camunda
#   8.8/8.9 clients, or refresh one unified API client with --target.
#
# Usage:
#   bash api/refresh-clients.sh
#   bash api/refresh-clients.sh --commit
#   bash api/refresh-clients.sh --camunda-tag 8.8.19 --camunda-docs-tag 8.8.196
#   bash api/refresh-clients.sh --target v810 --camunda-tag 8.10.9
#   bash api/refresh-clients.sh --target v89 --camunda-tag 8.9.28
#   bash api/refresh-clients.sh --target v88 --camunda-tag 8.8.46
#
# Notes:
#   This is the top-level entrypoint for the end-to-end workflow. It passes the
#   optional commit mode through to api/3-generate-clients-from-fetched-specs.sh.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

DO_COMMIT=false
CAMUNDA_TAG=""
CAMUNDA_DOCS_TAG=""
TARGET=""

usage() {
  cat <<'EOF'
Usage:
  bash api/refresh-clients.sh [--camunda-tag <tag>] [--camunda-docs-tag <tag>] [--commit]
  bash api/refresh-clients.sh --target <v88|v89|v810> --camunda-tag <tag>

Options:
  --target <target>         Generate only the named unified client. Supported: v88, v89, v810.
  --camunda-tag <tag>       Fetch a specific tag from camunda/camunda for the unified API spec.
  --camunda-docs-tag <tag>  Fetch a specific tag from camunda/camunda-docs for legacy component specs.
  --commit                  Pass commit mode through to api/3-generate-clients-from-fetched-specs.sh.
EOF
}

validate_target_path() {
  local target="$1"
  local output_dir

  case "$target" in
    *..*|*/*)
      echo "Camunda client output path escapes repository" >&2
      exit 1
      ;;
  esac

  output_dir="$REPO_ROOT/internal/clients/camunda/$target/camunda"
  case "$output_dir" in
    "$REPO_ROOT"/internal/clients/camunda/*/camunda)
      ;;
    *)
      echo "Camunda client output path escapes repository" >&2
      exit 1
      ;;
  esac
}

validate_v810_tag() {
  local tag="$1"

  if [[ ! "$tag" =~ ^8\.10([.-].*)?$ ]]; then
    echo "Invalid Camunda tag for v810: $tag" >&2
    exit 1
  fi
}

while (($# > 0)); do
  case "$1" in
    --target)
      if (($# < 2)); then
        echo "Missing value for --target" >&2
        usage >&2
        exit 1
      fi
      TARGET="$2"
      shift 2
      ;;
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

if [ -n "$TARGET" ]; then
  validate_target_path "$TARGET"

  if [ "$TARGET" != v810 ] && [ "$TARGET" != v89 ] && [ "$TARGET" != v88 ]; then
    echo "Unknown target: $TARGET" >&2
    usage >&2
    exit 1
  fi

  if [ -z "$CAMUNDA_TAG" ]; then
    echo "Missing value for --camunda-tag" >&2
    usage >&2
    exit 1
  fi

  if [ "$TARGET" = v810 ]; then
    validate_v810_tag "$CAMUNDA_TAG"
    "$SCRIPT_DIR/generate-v810-client.sh" --camunda-tag "$CAMUNDA_TAG"
  else
    bash "$SCRIPT_DIR/generate-camunda-client.sh" --target "$TARGET" --camunda-tag "$CAMUNDA_TAG"
  fi
  exit 0
fi

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
