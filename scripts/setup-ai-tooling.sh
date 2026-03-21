#!/usr/bin/env bash
#
# Installs the pinned private ai-tooling version into this repository.
#
# Usage examples:
#   export AI_TOOLING_REPO=/path/to/local/ai-tooling
#   ./scripts/setup-ai-tooling.sh
#
#   export AI_TOOLING_REPO=git@github.com:your-org/private-ai-tooling.git
#   ./scripts/setup-ai-tooling.sh

set -euo pipefail

SCRIPT_DIR="$(CDPATH="" cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(CDPATH="" cd "$SCRIPT_DIR/.." && pwd)"
VERSION_FILE="$REPO_ROOT/.ai-tooling-version"

if [ ! -f "$VERSION_FILE" ]; then
    echo "Missing version file: $VERSION_FILE" >&2
    exit 1
fi

if [ -z "${AI_TOOLING_REPO:-}" ]; then
    echo "Set AI_TOOLING_REPO to the private ai-tooling git URL or local path." >&2
    exit 1
fi

TAG="$(sed -n 's/^tag = \"\\([^\"]*\\)\"$/\\1/p' "$VERSION_FILE")"
if [ -z "$TAG" ]; then
    echo "Could not parse tag from $VERSION_FILE" >&2
    exit 1
fi

TMP_DIR="$(mktemp -d)"
cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

git clone --depth 1 --branch "$TAG" "$AI_TOOLING_REPO" "$TMP_DIR"
"$TMP_DIR/install/sync.sh" "$REPO_ROOT"

echo "Installed ai-tooling $TAG from $AI_TOOLING_REPO"
