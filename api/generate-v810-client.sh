#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

# Purpose:
#   Generate the isolated Camunda v8.10 unified Go client from a pinned product
#   repository source tag without touching protected v8.7-v8.9 clients.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

CAMUNDA_REPOSITORY="https://github.com/camunda/camunda.git"
INITIAL_TAG="8.10.0-alpha4"
INITIAL_COMMIT="4a76f06c9df8ad0a6c64fe88b92c618c2ebf2ef6"
SOURCE_SPEC="zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml"
SOURCE_SPEC_DIR="${SOURCE_SPEC%/*}"
OUTPUT_DIR="$REPO_ROOT/internal/clients/camunda/v810/camunda"
OUTPUT_CLIENT="$OUTPUT_DIR/client.gen.go"
OUTPUT_PROVENANCE="$OUTPUT_DIR/provenance.json"
TMP_ROOT=""

MUTATION_PATHS=(
  "api/mutations/mutate-search-query-schemas.py"
  "api/mutations/mutate-search-result-schemas.py"
  "api/mutations/mutate-fix-process-instance-filter-fields.py"
  "api/mutations/mutate-fix-jobresult-discriminator.py"
  "api/mutations/mutate-fix-camunda-v2-operation-id-collisions.py"
)
MUTATION_SUFFIXES=(
  "search-query-patched"
  "search-result-patched"
  "process-instance-filter-fields-fixed"
  "jobresult-fixed"
  "opids-fixed"
)
REQUIRED_SYMBOLS=(
  "type ClientWithResponsesInterface interface"
  "func NewClientWithResponses"
  "SearchBatchOperationsWithResponse"
  "GetTopologyWithResponse"
  "SearchElementInstancesWithResponse"
  "SearchIncidentsWithResponse"
  "ResolveIncidentWithResponse"
  "SearchJobsWithResponse"
  "SearchProcessDefinitionsWithResponse"
  "SearchProcessInstancesWithResponse"
  "CancelProcessInstanceWithResponse"
  "GetResourceWithResponse"
  "DeleteResourceOpWithResponse"
  "SearchTenantsWithResponse"
  "SearchUserTasksWithResponse"
  "SearchVariablesWithResponse"
)

usage() {
  cat <<'EOF'
Usage:
  bash api/generate-v810-client.sh --camunda-tag <8.10-tag>

Options:
  --camunda-tag <tag>  Camunda 8.10 source tag to generate from.
EOF
}

cleanup() {
  if [ -n "${TMP_ROOT:-}" ]; then
    rm -rf "$TMP_ROOT"
  fi
}

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing tool: $1" >&2
    exit 127
  }
}

sha256_file() {
  local path="$1"

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$path" | awk '{print $1}'
    return 0
  fi

  shasum -a 256 "$path" | awk '{print $1}'
}

sha256_stream() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum | awk '{print $1}'
    return 0
  fi

  shasum -a 256 | awk '{print $1}'
}

checksum_tree() {
  local root="$1"

  if [ ! -d "$root" ]; then
    echo "missing protected tree: $root" >&2
    exit 1
  fi

  (
    cd "$root"
    find . -type f ! -path './.git/*' -print | LC_ALL=C sort | while IFS= read -r path; do
      printf '%s  %s\n' "$(sha256_file "$path")" "${path#./}"
    done
  ) | sha256_stream
}

fingerprint_protected_trees() {
  for version in v87 v88 v89; do
    printf '%s %s\n' "$version" "$(checksum_tree "$REPO_ROOT/internal/clients/camunda/$version")"
  done
}

validate_target_path() {
  local output_parent
  local resolved_parent

  case "$OUTPUT_DIR" in
    "$REPO_ROOT"/internal/clients/camunda/v810/camunda)
      ;;
    *)
      echo "V810 output path escapes repository" >&2
      exit 1
      ;;
  esac

  if [ -L "$OUTPUT_DIR" ]; then
    echo "V810 output path escapes repository" >&2
    exit 1
  fi

  output_parent="$(dirname "$OUTPUT_DIR")"
  if [ -e "$output_parent" ]; then
    resolved_parent="$(cd "$output_parent" && pwd -P)"
    if [ "$resolved_parent" != "$REPO_ROOT/internal/clients/camunda/v810" ]; then
      echo "V810 output path escapes repository" >&2
      exit 1
    fi
  fi
}

validate_tag() {
  local tag="$1"

  if [[ ! "$tag" =~ ^8\.10(\.[0-9]+)?$ ]] && [[ ! "$tag" =~ ^8\.10\.[0-9]+-(alpha|rc)[0-9]+$ ]]; then
    echo "Invalid Camunda tag for v810: $tag" >&2
    exit 1
  fi
}

resolve_peeled_commit() {
  local tag="$1"
  local refs
  local peeled

  refs="$(git ls-remote "$CAMUNDA_REPOSITORY" "refs/tags/$tag" "refs/tags/$tag^{}")"
  peeled="$(printf '%s\n' "$refs" | awk -v ref="refs/tags/$tag^{}" '$2 == ref {print $1; exit}')"
  if [ -z "$peeled" ]; then
    peeled="$(printf '%s\n' "$refs" | awk -v ref="refs/tags/$tag" '$2 == ref {print $1; exit}')"
  fi

  if [ -z "$peeled" ]; then
    echo "Could not resolve Camunda tag for v810: $tag" >&2
    exit 1
  fi

  printf '%s\n' "$peeled"
}

verify_commit_policy() {
  local tag="$1"
  local commit="$2"

  if [ "$tag" = "$INITIAL_TAG" ] && [ "$commit" != "$INITIAL_COMMIT" ]; then
    echo "Commit mismatch for v810 tag $tag: expected $INITIAL_COMMIT, got $commit" >&2
    exit 1
  fi
}

fetch_source_spec() {
  local tag="$1"
  local commit="$2"
  local source_repo="$TMP_ROOT/camunda"

  mkdir -p "$source_repo"
  git -C "$source_repo" init -q
  git -C "$source_repo" remote add origin "$CAMUNDA_REPOSITORY"
  git -C "$source_repo" sparse-checkout init --no-cone >/dev/null
  git -C "$source_repo" sparse-checkout set "$SOURCE_SPEC_DIR" >/dev/null
  git -C "$source_repo" fetch --depth=1 origin "refs/tags/$tag" >/dev/null
  git -C "$source_repo" checkout -q --detach "$commit"

  if [ ! -f "$source_repo/$SOURCE_SPEC" ]; then
    echo "Missing source spec after fetch: $SOURCE_SPEC" >&2
    exit 1
  fi

  printf '%s\n' "$source_repo/$SOURCE_SPEC"
}

apply_mutation() {
  local input="$1"
  local mutation_path="$2"
  local suffix="$3"
  local mutation="$REPO_ROOT/$mutation_path"
  local base="${input%.yaml}"
  local output="$base-$suffix.yaml"
  local before
  local after

  before="$(sha256_file "$input")"
  python3 "$mutation" "$input" >/dev/null

  if [ ! -f "$output" ]; then
    echo "mutation produced no expected effect: $mutation_path" >&2
    exit 1
  fi

  after="$(sha256_file "$output")"
  if [ "$before" = "$after" ]; then
    echo "mutation produced no expected effect: $mutation_path" >&2
    exit 1
  fi

  printf '%s\n' "$output"
}

prepare_spec() {
  local source="$1"
  local prepared_dir="$TMP_ROOT/prepared"
  local bundled="$prepared_dir/rest-api-bundled.yaml"
  local current

  mkdir -p "$prepared_dir"
  redocly bundle "$source" -o "$bundled" >/dev/null || exit 1
  current="$bundled"

  for i in "${!MUTATION_PATHS[@]}"; do
    current="$(apply_mutation "$current" "${MUTATION_PATHS[$i]}" "${MUTATION_SUFFIXES[$i]}")" || exit 1
  done

  printf '%s\n' "$current"
}

generate_client() {
  local prepared_spec="$1"
  local generated_dir="$TMP_ROOT/generated"
  local generated_client="$generated_dir/client.gen.go"

  mkdir -p "$generated_dir"
  "$SCRIPT_DIR/generate-go-client.sh" "$prepared_spec" "$generated_client" camunda || exit 1
  gofmt -w "$generated_client"
  printf '%s\n' "$generated_client"
}

assert_required_symbols() {
  local generated_client="$1"

  for symbol in "${REQUIRED_SYMBOLS[@]}"; do
    if ! grep -Fq "$symbol" "$generated_client"; then
      echo "generated v810 client missing required symbol: $symbol" >&2
      exit 1
    fi
  done
}

compile_generated_client() {
  local generated_client="$1"
  local compile_dir="$TMP_ROOT/compile"

  mkdir -p "$compile_dir"
  cp "$generated_client" "$compile_dir/client.gen.go"
  cat >"$compile_dir/go.mod" <<'EOF'
module c8volt-v810-generated-compile

go 1.26

require (
	github.com/oapi-codegen/nullable v1.1.0
	github.com/oapi-codegen/runtime v1.4.0
)
EOF

  (cd "$compile_dir" && GOWORK=off go mod tidy && GOWORK=off go test ./...)
}

tool_version() {
  local name="$1"

  case "$name" in
    redocly)
      redocly --version 2>&1 | awk 'NF {last=$0} END {print last}'
      ;;
    oapi-codegen)
      oapi-codegen -version 2>&1 | awk '/^v?[0-9]+[.][0-9]+[.][0-9]+/ {version=$0} END {sub(/^v/, "", version); print version}'
      ;;
  esac
}

write_provenance() {
  local tag="$1"
  local commit="$2"
  local source_hash="$3"
  local prepared_spec="$4"
  local generated_client="$5"
  local provenance="$TMP_ROOT/provenance.json"
  local transformation_json="$TMP_ROOT/transformations.json"

  python3 - "$REPO_ROOT" "${MUTATION_PATHS[@]}" >"$transformation_json" <<'PY'
import hashlib
import json
import sys
from pathlib import Path

root = Path(sys.argv[1])
items = []
for path in sys.argv[2:]:
    digest = hashlib.sha256((root / path).read_bytes()).hexdigest()
    items.append({"path": path, "sha256": digest})
print(json.dumps(items, separators=(",", ":")))
PY

  python3 - \
    "$provenance" \
    "$tag" \
    "$commit" \
    "$source_hash" \
    "$(cat "$transformation_json")" \
    "$(tool_version redocly)" \
    "$(tool_version oapi-codegen)" \
    "$(sha256_file "$prepared_spec")" \
    "$(sha256_file "$generated_client")" <<'PY'
import json
import sys
from pathlib import Path

(
    provenance_path,
    tag,
    commit,
    source_hash,
    transformations_json,
    redocly_version,
    oapi_codegen_version,
    prepared_hash,
    generated_hash,
) = sys.argv[1:]

provenance = {
    "schemaVersion": 1,
    "repository": "https://github.com/camunda/camunda.git",
    "tag": tag,
    "commit": commit,
    "sourceSpec": "zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml",
    "sourceSpecSha256": source_hash,
    "transformations": json.loads(transformations_json),
    "tools": {
        "redocly": redocly_version,
        "oapi-codegen": oapi_codegen_version,
    },
    "preparedSpecSha256": prepared_hash,
    "generatedClientSha256": generated_hash,
    "command": f"bash api/refresh-clients.sh --target v810 --camunda-tag {tag}",
}

Path(provenance_path).write_text(
    json.dumps(provenance, indent=2, separators=(",", ": ")) + "\n",
    encoding="utf-8",
)
PY

  printf '%s\n' "$provenance"
}

verify_provenance_hashes() {
  local generated_client="$1"
  local provenance="$2"

  python3 - "$generated_client" "$provenance" <<'PY'
import hashlib
import json
import sys
from pathlib import Path

generated_client = Path(sys.argv[1])
provenance = json.loads(Path(sys.argv[2]).read_text(encoding="utf-8"))
actual = hashlib.sha256(generated_client.read_bytes()).hexdigest()
if provenance.get("generatedClientSha256") != actual:
    raise SystemExit("generated client hash does not match provenance")
PY
}

verify_provenance_identity() {
  local tag="$1"
  local provenance="$2"

  python3 - "$tag" "$provenance" <<'PY'
import json
import sys
from pathlib import Path

tag, provenance_path = sys.argv[1:]
provenance = json.loads(Path(provenance_path).read_text(encoding="utf-8"))
expected_command = f"bash api/refresh-clients.sh --target v810 --camunda-tag {tag}"

if provenance.get("tag") != tag:
    raise SystemExit("provenance tag does not match requested v810 baseline")
if provenance.get("command") != expected_command:
    raise SystemExit("provenance command does not preserve v810 target identity")
command = provenance.get("command", "")
if any(identity in command for identity in ("v810alpha", "v810rc", "v810final")):
    raise SystemExit("provenance command uses a prerelease-specific v810 identity")
PY
}

publish_artifacts() {
  local generated_client="$1"
  local provenance="$2"
  local output_parent
  local staging_root
  local staged_dir
  local backup_dir=""
  local had_existing=false

  output_parent="$(dirname "$OUTPUT_DIR")"
  mkdir -p "$output_parent"
  staging_root="$(mktemp -d "$output_parent/.v810-publish.XXXXXX")"
  staged_dir="$staging_root/camunda"
  if ! mkdir -p "$staged_dir"; then
    rm -rf "$staging_root"
    exit 1
  fi

  if [ -d "$OUTPUT_DIR" ]; then
    if ! cp -R "$OUTPUT_DIR/." "$staged_dir/"; then
      echo "Failed to stage existing V810 generated artifacts" >&2
      rm -rf "$staging_root"
      exit 1
    fi
  fi
  if ! cp "$generated_client" "$staged_dir/client.gen.go"; then
    echo "Failed to stage V810 generated client" >&2
    rm -rf "$staging_root"
    exit 1
  fi
  if ! cp "$provenance" "$staged_dir/provenance.json"; then
    echo "Failed to stage V810 provenance" >&2
    rm -rf "$staging_root"
    exit 1
  fi

  if [ -e "$OUTPUT_DIR" ]; then
    backup_dir="$(mktemp -d "$output_parent/.v810-backup.XXXXXX")"
    if ! rmdir "$backup_dir"; then
      echo "Failed to prepare V810 publication backup" >&2
      rm -rf "$staging_root" "$backup_dir"
      exit 1
    fi
    had_existing=true
    if ! mv "$OUTPUT_DIR" "$backup_dir"; then
      echo "Failed to backup existing V810 generated artifacts" >&2
      rm -rf "$staging_root" "$backup_dir"
      exit 1
    fi
  fi

  if ! mv "$staged_dir" "$OUTPUT_DIR"; then
    echo "Failed to publish V810 generated artifacts" >&2
    if [ "$had_existing" = true ] && [ -e "$backup_dir" ]; then
      rm -rf "$OUTPUT_DIR"
      if ! mv "$backup_dir" "$OUTPUT_DIR"; then
        echo "Failed to restore existing V810 generated artifacts" >&2
        rm -rf "$staging_root"
        exit 1
      fi
    fi
    rm -rf "$staging_root"
    exit 1
  fi

  rm -rf "$staging_root"
  if [ "$had_existing" = true ]; then
    rm -rf "$backup_dir"
  fi
}

assert_no_removed_target_dirs() {
  for path in \
    "$REPO_ROOT/internal/clients/camunda/v810alpha" \
    "$REPO_ROOT/internal/clients/camunda/v810rc" \
    "$REPO_ROOT/internal/clients/camunda/v810final"; do
    if [ -e "$path" ]; then
      echo "Unexpected V810 prerelease output directory: $path" >&2
      exit 1
    fi
  done
}

assert_protected_unchanged() {
  local before="$1"
  local after

  after="$(fingerprint_protected_trees)"
  if [ "$before" != "$after" ]; then
    echo "Protected Camunda client tree changed during v810 generation" >&2
    exit 1
  fi
}

CAMUNDA_TAG=""
while (($# > 0)); do
  case "$1" in
    --camunda-tag)
      if (($# < 2)); then
        echo "Missing value for --camunda-tag" >&2
        usage >&2
        exit 1
      fi
      CAMUNDA_TAG="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [ -z "$CAMUNDA_TAG" ]; then
  echo "Missing value for --camunda-tag" >&2
  usage >&2
  exit 1
fi

validate_target_path
validate_tag "$CAMUNDA_TAG"

need redocly
need oapi-codegen
need python3
need go
need git

TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/c8volt-v810-client.XXXXXX")"
trap cleanup EXIT

protected_before="$(fingerprint_protected_trees)"
peeled_commit="$(resolve_peeled_commit "$CAMUNDA_TAG")" || exit 1
verify_commit_policy "$CAMUNDA_TAG" "$peeled_commit"

source_spec_path="$(fetch_source_spec "$CAMUNDA_TAG" "$peeled_commit")" || exit 1
source_spec_hash="$(sha256_file "$source_spec_path")"
prepared_spec="$(prepare_spec "$source_spec_path")" || exit 1
generated_client="$(generate_client "$prepared_spec")" || exit 1

assert_required_symbols "$generated_client"
compile_generated_client "$generated_client" >/dev/null

provenance="$(write_provenance "$CAMUNDA_TAG" "$peeled_commit" "$source_spec_hash" "$prepared_spec" "$generated_client")" || exit 1
verify_provenance_hashes "$generated_client" "$provenance"
verify_provenance_identity "$CAMUNDA_TAG" "$provenance"

assert_protected_unchanged "$protected_before"
publish_artifacts "$generated_client" "$provenance"
assert_no_removed_target_dirs
assert_protected_unchanged "$protected_before"
