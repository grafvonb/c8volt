#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_DIR="${SCRIPT_DIR}/camunda-docs"

cd "$SCRIPT_DIR"

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
tag=$(git ls-remote --tags --refs "$repo" | awk -F/ '{print $3}' | sort -V | tail -n1)

git -c advice.detachedHead=false clone --depth 1 --filter=blob:none --branch "$tag" "$repo" "$TARGET_DIR"

cd "$TARGET_DIR"
git sparse-checkout init --no-cone
git sparse-checkout set '/api/*'
