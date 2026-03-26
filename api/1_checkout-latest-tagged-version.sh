#!/usr/bin/env bash
# Fetch the latest tagged camunda-docs snapshot into ./camunda-docs,
# preserving any existing directory by renaming it to .bak(.N), then
# sparse-checking out only /api/* for downstream API tooling.
#
# Usage:
#   bash api/1_checkout-latest-tagged-version.sh
set -euo pipefail

# Resolve paths relative to this script so it can be run from any working directory.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_DIR="${SCRIPT_DIR}/camunda-docs"

cd "$SCRIPT_DIR"

# If a previous checkout exists and is non-empty, keep it by rotating to a .bak name.
if [ -d "$TARGET_DIR" ]; then
  if [ "$(ls -A "$TARGET_DIR")" ]; then
    bak="${TARGET_DIR}.bak"
    i=1
    while [ -e "$bak" ]; do
      bak="${TARGET_DIR}.bak.$i"
      i=$((i+1))
    done
    mv "$TARGET_DIR" "$bak"
    echo "Existing directory renamed to: $bak"
  fi
fi

repo="git@github.com:camunda/camunda-docs.git"
# Pick the newest semantic-version-like tag from the remote tag list.
tag=$(git ls-remote --tags --refs "$repo" | awk -F/ '{print $3}' | sort -V | tail -n1)

# Clone only the selected tag with minimal history and blob download.
git -c advice.detachedHead=false clone --depth 1 --filter=blob:none --branch "$tag" "$repo" "$TARGET_DIR"

cd "$TARGET_DIR"
# Keep checkout minimal by materializing only the API subtree.
git sparse-checkout init --no-cone
git sparse-checkout set '/api/*'
