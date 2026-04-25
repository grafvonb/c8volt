#!/usr/bin/env bash
#
# Purpose:
#   Bootstrap the current target repository with a fresh ai-tooling installer.
#
# Usage:
#   ./ai/bootstrap-ai-tooling.sh [--force] [tag]
#   AI_TOOLING_REPO=/path/to/local/ai-tooling ./ai/bootstrap-ai-tooling.sh
#   AI_TOOLING_REPO=git@github.com:your-org/private-ai-tooling.git ./ai/bootstrap-ai-tooling.sh
#
# Parameters:
#   --force  Forward to the installer to allow overwriting locally drifted
#            ai-tooling-managed files.
#   tag      Optional ai-tooling tag or branch to install. If omitted, the
#            latest tag from AI_TOOLING_REPO is used.
#
# Environment:
#   AI_TOOLING_REPO  Local ai-tooling checkout or remote git URL. Defaults to
#                    ../../ai-tooling relative to the target repository.
#
# Notes:
#   This script is intentionally small and stable. It fetches the requested
#   ai-tooling version, then runs that version's installer against this target
#   repository so installer changes do not require running install/sync.sh by
#   hand.

set -euo pipefail

usage() {
    cat >&2 <<'EOF'
Usage: bootstrap-ai-tooling.sh [--force] [tag]
EOF
    exit 1
}

INSTALL_ARGS=()
TAG_ARG=""

while [ $# -gt 0 ]; do
    case "$1" in
        --force)
            INSTALL_ARGS+=("$1")
            shift
            ;;
        -h|--help)
            usage
            ;;
        -*)
            echo "Unknown option: $1" >&2
            usage
            ;;
        *)
            if [ -n "$TAG_ARG" ]; then
                usage
            fi
            TAG_ARG="$1"
            INSTALL_ARGS+=("$1")
            shift
            ;;
    esac
done

SCRIPT_DIR="$(CDPATH="" cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(CDPATH="" cd "$SCRIPT_DIR/.." && pwd)"
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

TAG="${TAG_ARG:-$(latest_tag "$RESOLVED_AI_TOOLING_REPO")}"
if [ -z "$TAG" ]; then
    echo "Could not resolve an ai-tooling tag from $RESOLVED_AI_TOOLING_REPO" >&2
    exit 1
fi

if [ -z "$TAG_ARG" ]; then
    INSTALL_ARGS+=("$TAG")
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

AI_TOOLING_REPO="$RESOLVED_AI_TOOLING_REPO" \
AI_TOOLING_TARGET_REPO="$REPO_ROOT" \
    "$TMP_DIR/ai/install-ai-tooling.sh" "${INSTALL_ARGS[@]}"
