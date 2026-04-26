#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

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
#   directory. After fetching, the nested .git metadata is removed so the
#   working tree can be tracked by the parent repository.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_DIR="${SCRIPT_DIR}/camunda"
REPO="git@github.com:camunda/camunda.git"
SPEC_PATH='/zeebe/gateway-protocol/src/main/proto'
TAG="${1:-}"
COMMIT_SHA=""

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
    git ls-remote --tags "$REPO" \
      | awk '
          $2 ~ /^refs\/tags\/[0-9]+(\.[0-9]+){1,2}(\^\{\})?$/ {
            ref = $2
            sub(/^refs\/tags\//, "", ref)
            sub(/\^\{\}$/, "", ref)
            print ref
          }
        ' \
      | sort -uV \
      | tail -n1
  )"
fi

COMMIT_SHA="$(
  git ls-remote --tags "$REPO" "refs/tags/${TAG}^{}" "refs/tags/${TAG}" \
    | awk 'NR==1 { print $1 }'
)"

if [ -z "$COMMIT_SHA" ]; then
  echo "Could not resolve commit for tag: $TAG" >&2
  exit 1
fi

git init "$TARGET_DIR" >/dev/null
git -C "$TARGET_DIR" remote add origin "$REPO"
git -C "$TARGET_DIR" sparse-checkout init --no-cone
git -C "$TARGET_DIR" sparse-checkout set "$SPEC_PATH"
git -C "$TARGET_DIR" fetch --depth 1 --filter=blob:none origin "$COMMIT_SHA"
git -C "$TARGET_DIR" checkout --detach FETCH_HEAD >/dev/null

cd "$TARGET_DIR"
rm -rf "$TARGET_DIR/.git"

echo "Fetched Camunda v2 product spec from $REPO at tag $TAG into $TARGET_DIR"
