#!/usr/bin/env bash
#
# Copies managed tooling changes from this repository back into a local
# ai-tooling checkout after validating them in the real project.
#
# Usage examples:
#   export AI_TOOLING_REPO=/path/to/local/ai-tooling
#   ./scripts/push-ai-tooling-changes.sh

set -euo pipefail

SCRIPT_DIR="$(CDPATH="" cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(CDPATH="" cd "$SCRIPT_DIR/.." && pwd)"

if [ -z "${AI_TOOLING_REPO:-}" ]; then
    echo "Set AI_TOOLING_REPO to a local ai-tooling checkout path." >&2
    exit 1
fi

if [ ! -d "$AI_TOOLING_REPO/.git" ]; then
    echo "AI_TOOLING_REPO must point to a local git checkout: $AI_TOOLING_REPO" >&2
    exit 1
fi

"$AI_TOOLING_REPO/install/sync-back.sh" "$REPO_ROOT"

echo "Copied local tooling changes into $AI_TOOLING_REPO"
