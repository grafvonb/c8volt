#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

#
# Purpose:
#   Install a private ai-tooling version into a target repository.
#
# Usage:
#   ./ai/install-ai-tooling.sh [--force] [tag]
#   AI_TOOLING_REPO=/path/to/local/ai-tooling ./ai/install-ai-tooling.sh
#   AI_TOOLING_TARGET_REPO=/path/to/consumer AI_TOOLING_REPO=/path/to/ai-tooling ./ai/install-ai-tooling.sh
#   AI_TOOLING_REPO=git@github.com:your-org/private-ai-tooling.git ./ai/install-ai-tooling.sh
#
# Parameters:
#   --force  Allow overwriting locally drifted ai-tooling-managed files.
#   tag      Optional ai-tooling tag or branch to install. If omitted, the
#            latest tag from AI_TOOLING_REPO is used.
#
# Environment:
#   AI_TOOLING_REPO          Local ai-tooling checkout or remote git URL.
#                            Defaults to ../../ai-tooling relative to the
#                            target repository.
#   AI_TOOLING_TARGET_REPO   Repository to install into. Defaults to the parent
#                            of this script's ai/ directory.
#   AI_TOOLING_ALLOW_DRIFT   Set to 1 to behave like --force.
#
# Notes:
#   The target repository must have a clean git worktree. After a successful
#   install, this script records the installed tag, writes an install manifest,
#   validates copied assets, and commits the resulting changes automatically.

set -euo pipefail

FORCE_INSTALL=0
TAG_ARG=""

while [ $# -gt 0 ]; do
    case "$1" in
        --force)
            FORCE_INSTALL=1
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [--force] [tag]" >&2
            exit 0
            ;;
        -*)
            echo "Unknown option: $1" >&2
            echo "Usage: $0 [--force] [tag]" >&2
            exit 1
            ;;
        *)
            if [ -n "$TAG_ARG" ]; then
                echo "Usage: $0 [--force] [tag]" >&2
                exit 1
            fi
            TAG_ARG="$1"
            shift
            ;;
    esac
done

if [ "${AI_TOOLING_ALLOW_DRIFT:-0}" = "1" ]; then
    FORCE_INSTALL=1
fi

SCRIPT_DIR="$(CDPATH="" cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -n "${AI_TOOLING_TARGET_REPO:-}" ]; then
    REPO_ROOT="$(CDPATH="" cd "$AI_TOOLING_TARGET_REPO" && pwd)"
else
    REPO_ROOT="$(CDPATH="" cd "$SCRIPT_DIR/.." && pwd)"
fi
DEFAULT_LOCAL_AI_TOOLING_REPO="$(CDPATH="" cd "$REPO_ROOT/../.." && pwd)/ai-tooling"
VERSION_MARKER_FILE="$REPO_ROOT/ai/installed-ai-tooling-version"
MANIFEST_FILE="$REPO_ROOT/ai/installed-ai-tooling-manifest.txt"

if [ -n "$(git -C "$REPO_ROOT" status --porcelain)" ]; then
    echo "Refusing to install ai-tooling into a dirty repository." >&2
    echo "Commit or stash existing changes first, then rerun this script." >&2
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

hash_file() {
    local path="$1"

    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$path" | awk '{print $1}'
        return
    fi

    if command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$path" | awk '{print $1}'
        return
    fi

    if command -v openssl >/dev/null 2>&1; then
        openssl dgst -sha256 -r "$path" | awk '{print $1}'
        return
    fi

    echo "No SHA-256 hashing tool available (expected sha256sum, shasum, or openssl)." >&2
    exit 1
}

append_file_if_present() {
    local path="$1"
    local list_file="$2"

    if [ -f "$path" ]; then
        printf '%s\n' "${path#"$REPO_ROOT"/}" >> "$list_file"
    fi
}

append_tree_files_if_present() {
    local path="$1"
    local list_file="$2"

    if [ -d "$path" ]; then
        find "$path" -type f | sed "s#^$REPO_ROOT/##" >> "$list_file"
    fi
}

write_install_manifest() {
    local source_repo="$1"
    local source_commit="$2"
    local tag="$3"
    local synced_at="$4"
    local list_file="$5"
    local sorted_file="$6"

    : > "$list_file"

    append_file_if_present "$REPO_ROOT/AGENTS.md" "$list_file"
    append_tree_files_if_present "$REPO_ROOT/.agents/skills" "$list_file"
    append_file_if_present "$REPO_ROOT/.specify/init-options.json" "$list_file"
    append_file_if_present "$REPO_ROOT/.specify/integration.json" "$list_file"
    append_file_if_present "$REPO_ROOT/.specify/extensions.yml" "$list_file"
    append_file_if_present "$REPO_ROOT/.specify/extension-catalogs.yml" "$list_file"
    append_tree_files_if_present "$REPO_ROOT/.specify/integrations" "$list_file"
    append_tree_files_if_present "$REPO_ROOT/.specify/workflows" "$list_file"
    append_tree_files_if_present "$REPO_ROOT/.specify/scripts" "$list_file"
    append_tree_files_if_present "$REPO_ROOT/.specify/templates" "$list_file"
    append_tree_files_if_present "$REPO_ROOT/.specify/presets" "$list_file"
    append_tree_files_if_present "$REPO_ROOT/.specify/extensions" "$list_file"
    append_file_if_present "$REPO_ROOT/.specify/memory/constitution.md" "$list_file"
    append_tree_files_if_present "$REPO_ROOT/extensions" "$list_file"
    append_tree_files_if_present "$REPO_ROOT/integrations" "$list_file"
    append_tree_files_if_present "$REPO_ROOT/presets" "$list_file"
    append_tree_files_if_present "$REPO_ROOT/workflows" "$list_file"
    append_tree_files_if_present "$REPO_ROOT/ai" "$list_file"
    append_file_if_present "$REPO_ROOT/scripts/ralph/ralph.sh" "$list_file"
    append_file_if_present "$REPO_ROOT/scripts/ralph/doctor.sh" "$list_file"
    append_file_if_present "$REPO_ROOT/scripts/ralph/prompt.md" "$list_file"

    sort -u "$list_file" > "$sorted_file"

    {
        printf 'tag=%s\n' "$tag"
        printf 'source=%s\n' "$source_repo"
        printf 'commit=%s\n' "$source_commit"
        printf 'synced_at_utc=%s\n' "$synced_at"
        printf '\n'
        printf '# path sha256\n'

        while IFS= read -r relative_path; do
            [ -n "$relative_path" ] || continue
            printf '%s %s\n' "$relative_path" "$(hash_file "$REPO_ROOT/$relative_path")"
        done < "$sorted_file"
    } > "$MANIFEST_FILE"
}

check_installed_manifest_drift() {
    local drift_file="$1"
    local relative_path=""
    local expected_hash=""
    local current_hash=""

    : > "$drift_file"

    if [ ! -f "$MANIFEST_FILE" ]; then
        return 0
    fi

    while read -r relative_path expected_hash; do
        case "$relative_path" in
            ""|\#*|tag=*|source=*|commit=*|synced_at_utc=*)
                continue
                ;;
        esac

        if [ -z "${expected_hash:-}" ]; then
            continue
        fi

        if [ ! -f "$REPO_ROOT/$relative_path" ]; then
            printf 'missing %s\n' "$relative_path" >> "$drift_file"
            continue
        fi

        current_hash="$(hash_file "$REPO_ROOT/$relative_path")"
        if [ "$current_hash" != "$expected_hash" ]; then
            printf 'modified %s\n' "$relative_path" >> "$drift_file"
        fi
    done < "$MANIFEST_FILE"

    if [ -s "$drift_file" ]; then
        if [ "$FORCE_INSTALL" -eq 1 ]; then
            echo "Warning: overwriting locally drifted ai-tooling-managed files:" >&2
            sed 's/^/  - /' "$drift_file" >&2
            return 0
        fi

        echo "Refusing to install ai-tooling because managed files drifted locally:" >&2
        sed 's/^/  - /' "$drift_file" >&2
        echo "Move reusable changes into ai-tooling, or rerun with --force / AI_TOOLING_ALLOW_DRIFT=1 to overwrite." >&2
        exit 1
    fi
}

validate_installed_artifacts() {
    python3 - "$REPO_ROOT" <<'PY'
import re
import sys
from pathlib import Path

repo_root = Path(sys.argv[1])
skills_root = repo_root / ".agents" / "skills"
extensions_root = repo_root / ".specify" / "extensions"
pattern = re.compile(r"\.specify/extensions/([A-Za-z0-9._-]+)/([A-Za-z0-9._/\-]+)")
missing: set[str] = set()

if not skills_root.is_dir():
    raise SystemExit(0)

for skill_file in skills_root.glob("*/SKILL.md"):
    try:
        content = skill_file.read_text(encoding="utf-8")
    except OSError:
        continue

    for match in pattern.finditer(content):
        extension_id = match.group(1)
        relative_path = match.group(2)
        target_path = extensions_root / extension_id / relative_path
        template_path = None

        if target_path.suffix == ".yml":
            template_path = target_path.with_name(target_path.name[:-4] + ".template.yml")

        if target_path.exists():
            continue
        if template_path is not None and template_path.exists():
            continue

        missing.add(f".specify/extensions/{extension_id}/{relative_path}")

if missing:
    sys.stderr.write("Installed ai-tooling is missing extension artifacts referenced by skills:\n")
    for item in sorted(missing):
        sys.stderr.write(f"  - {item}\n")
    raise SystemExit(1)
PY
}

TAG="${TAG_ARG:-$(latest_tag "$RESOLVED_AI_TOOLING_REPO")}"
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
check_installed_manifest_drift "$TMP_DIR/install-manifest-drift.txt"
AI_TOOLING_TARGET_REPO="$REPO_ROOT" "$TMP_DIR/ai/cleanup-ai-tooling-sync.sh" "$TMP_DIR"
"$TMP_DIR/install/sync.sh" "$REPO_ROOT"
validate_installed_artifacts

SOURCE_COMMIT="$(git -C "$TMP_DIR" rev-parse HEAD)"
SYNCED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
MANIFEST_LIST_FILE="$TMP_DIR/install-manifest-paths.txt"
MANIFEST_SORTED_FILE="$TMP_DIR/install-manifest-paths.sorted.txt"
write_install_manifest "$RESOLVED_AI_TOOLING_REPO" "$SOURCE_COMMIT" "$TAG" "$SYNCED_AT" "$MANIFEST_LIST_FILE" "$MANIFEST_SORTED_FILE"

printf '%s\n' "$TAG" > "$VERSION_MARKER_FILE"

if [ -n "$(git -C "$REPO_ROOT" status --porcelain)" ]; then
    git -C "$REPO_ROOT" add -A

    modified_files=()
    while IFS= read -r modified_file; do
        modified_files+=("$modified_file")
    done < <(git -C "$REPO_ROOT" diff --cached --name-only)
    commit_message="chore(ai): install ai-tooling $TAG"

    if [ "${#modified_files[@]}" -gt 0 ]; then
        commit_message+=$'\n\nModified files:\n'

        for modified_file in "${modified_files[@]}"; do
            commit_message+="- ${modified_file}"$'\n'
        done
    fi

    git -C "$REPO_ROOT" commit -m "$commit_message"
fi

echo "Installed ai-tooling $TAG from $RESOLVED_AI_TOOLING_REPO"
