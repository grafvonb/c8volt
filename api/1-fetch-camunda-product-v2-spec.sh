#!/usr/bin/env bash
# Purpose:
#   Fetch the product-repo /v2 Camunda OpenAPI source from camunda/camunda into
#   api/camunda for local client generation.
#
# Usage:
#   bash api/1-fetch-camunda-product-v2-spec.sh
#   bash api/1-fetch-camunda-product-v2-spec.sh 8.9.0
#
# Notes:
#   When no tag is provided, the script selects the latest numeric release tag
#   and preserves any existing api/camunda checkout by rotating it to a .bak
#   directory.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_DIR="${SCRIPT_DIR}/camunda"
REPO="git@github.com:camunda/camunda.git"
SPEC_PATH='/zeebe/gateway-protocol/src/main/proto/rest-api.yaml'
TAG="${1:-}"

cd "$SCRIPT_DIR"

if [ -d "$TARGET_DIR" ] && [ "$(ls -A "$TARGET_DIR")" ]; then
  bak="${TARGET_DIR}.bak"
  i=1
  while [ -e "$bak" ]; do
    bak="${TARGET_DIR}.bak.$i"
    i=$((i + 1))
  done
  mv "$TARGET_DIR" "$bak"
  echo "Existing directory renamed to: $bak"
fi

if [ -z "$TAG" ]; then
  TAG="$(
    git ls-remote --tags --refs "$REPO" \
      | awk -F/ '{print $3}' \
      | grep -E '^[0-9]+(\.[0-9]+){1,2}$' \
      | sort -V \
      | tail -n1
  )"
fi

git -c advice.detachedHead=false clone --depth 1 --filter=blob:none --branch "$TAG" "$REPO" "$TARGET_DIR"

cd "$TARGET_DIR"
git sparse-checkout init --no-cone
git sparse-checkout set "$SPEC_PATH"

echo "Fetched Camunda v2 product spec from $REPO at tag $TAG into $TARGET_DIR"
