#!/usr/bin/env bash
#
# Installs a private ai-tooling version into this repository.
#
# Usage examples:
#   ./ai/install-ai-tooling.sh
#   ./ai/install-ai-tooling.sh v0.2.0
#   AI_TOOLING_REPO=/path/to/local/ai-tooling ./ai/install-ai-tooling.sh
#   AI_TOOLING_REPO=git@github.com:your-org/private-ai-tooling.git ./ai/install-ai-tooling.sh

set -euo pipefail

if [ $# -gt 1 ]; then
    echo "Usage: $0 [tag]" >&2
    exit 1
fi

SCRIPT_DIR="$(CDPATH="" cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(CDPATH="" cd "$SCRIPT_DIR/.." && pwd)"
VERSION_FILE="$REPO_ROOT/.ai-tooling-version"
DEFAULT_LOCAL_AI_TOOLING_REPO="$(CDPATH="" cd "$REPO_ROOT/../.." && pwd)/ai-tooling"

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

PINNED_TAG=""
if [ -f "$VERSION_FILE" ]; then
    PINNED_TAG="$(awk -F'"' '$1 ~ /^tag = / { print $2; exit }' "$VERSION_FILE")"
fi

latest_tag() {
    local repo="$1"

    if [ -d "$repo/.git" ]; then
        git -C "$repo" tag --sort=-version:refname | head -n 1
        return
    fi

    git ls-remote --tags --refs "$repo" \
        | sed 's#.*refs/tags/##' \
        | sort -V -r \
        | head -n 1
}

TAG="${1:-$(latest_tag "$RESOLVED_AI_TOOLING_REPO")}"
if [ -z "$TAG" ]; then
    echo "Could not resolve an ai-tooling tag from $RESOLVED_AI_TOOLING_REPO" >&2
    exit 1
fi

TMP_DIR="$(mktemp -d)"
cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

CLONE_SOURCE="$RESOLVED_AI_TOOLING_REPO"
if [ -d "$RESOLVED_AI_TOOLING_REPO/.git" ]; then
    CLONE_SOURCE="file://$RESOLVED_AI_TOOLING_REPO"
fi

git -c advice.detachedHead=false clone --quiet --depth 1 --branch "$TAG" "$CLONE_SOURCE" "$TMP_DIR"
"$TMP_DIR/install/sync.sh" "$REPO_ROOT"

echo "Installed ai-tooling $TAG from $RESOLVED_AI_TOOLING_REPO"
if [ -n "$PINNED_TAG" ] && [ "$TAG" != "$PINNED_TAG" ]; then
    echo "Note: installed override tag $TAG while .ai-tooling-version remains pinned to $PINNED_TAG"
fi
