#!/usr/bin/env bash
#
# Installs the pinned private ai-tooling version into this repository.
#
# Usage examples:
#   ./ai/install-ai-tooling.sh
#   AI_TOOLING_REPO=/path/to/local/ai-tooling ./ai/install-ai-tooling.sh
#   AI_TOOLING_REPO=git@github.com:your-org/private-ai-tooling.git ./ai/install-ai-tooling.sh

set -euo pipefail

SCRIPT_DIR="$(CDPATH="" cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(CDPATH="" cd "$SCRIPT_DIR/.." && pwd)"
VERSION_FILE="$REPO_ROOT/.ai-tooling-version"
DEFAULT_LOCAL_AI_TOOLING_REPO="$(CDPATH="" cd "$REPO_ROOT/../.." && pwd)/ai-tooling"

if [ ! -f "$VERSION_FILE" ]; then
    echo "Missing version file: $VERSION_FILE" >&2
    exit 1
fi

if [ -n "${AI_TOOLING_REPO:-}" ]; then
    RESOLVED_AI_TOOLING_REPO="$AI_TOOLING_REPO"
elif [ -d "$DEFAULT_LOCAL_AI_TOOLING_REPO/.git" ]; then
    RESOLVED_AI_TOOLING_REPO="$DEFAULT_LOCAL_AI_TOOLING_REPO"
else
    echo "Could not resolve ai-tooling source." >&2
    echo "Tried default local checkout: $DEFAULT_LOCAL_AI_TOOLING_REPO" >&2
    echo "Set AI_TOOLING_REPO to a local ai-tooling checkout or private git URL." >&2
    exit 1
fi

TAG="$(awk -F'"' '$1 ~ /^tag = / { print $2; exit }' "$VERSION_FILE")"
if [ -z "$TAG" ]; then
    echo "Could not parse tag from $VERSION_FILE" >&2
    exit 1
fi

TMP_DIR="$(mktemp -d)"
cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

git clone --depth 1 --branch "$TAG" "$RESOLVED_AI_TOOLING_REPO" "$TMP_DIR"
"$TMP_DIR/install/sync.sh" "$REPO_ROOT"

echo "Installed ai-tooling $TAG from $RESOLVED_AI_TOOLING_REPO"
