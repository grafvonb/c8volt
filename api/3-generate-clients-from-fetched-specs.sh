#!/bin/bash
# Purpose:
#   Generate all checked-in Go clients from the already fetched API specs under
#   api/camunda and api/camunda-docs.
#
# Usage:
#   bash api/3-generate-clients-from-fetched-specs.sh
#   bash api/3-generate-clients-from-fetched-specs.sh --commit
#
# Notes:
#   This script expects the upstream sources to already be present locally. Use
#   api/refresh-clients.sh for the full fetch-plus-generate workflow.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$SCRIPT_DIR"

DO_COMMIT=false

usage() {
  cat <<'EOF'
Usage:
  bash api/3-generate-clients-from-fetched-specs.sh [--commit]

Options:
  --commit  Stage generated client files and create a Conventional Commit with
            the fetched upstream source tags in the commit body.
EOF
}

while (($# > 0)); do
  case "$1" in
    --commit)
      DO_COMMIT=true
      shift
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

GENERATED_FILES=(
  "internal/clients/auth/oauth2/client.gen.go"
  "internal/clients/camunda/v89/administrationsm/client.gen.go"
  "internal/clients/camunda/v89/camunda/client.gen.go"
  "internal/clients/camunda/v89/operate/client.gen.go"
  "internal/clients/camunda/v89/tasklist/client.gen.go"
  "internal/clients/camunda/v88/administrationsm/client.gen.go"
  "internal/clients/camunda/v88/camunda/client.gen.go"
  "internal/clients/camunda/v88/operate/client.gen.go"
  "internal/clients/camunda/v88/tasklist/client.gen.go"
  "internal/clients/camunda/v87/administrationsm/client.gen.go"
  "internal/clients/camunda/v87/camunda/client.gen.go"
  "internal/clients/camunda/v87/operate/client.gen.go"
  "internal/clients/camunda/v87/tasklist/client.gen.go"
  "internal/clients/camunda/v86/administrationsm/client.gen.go"
  "internal/clients/camunda/v86/camunda/client.gen.go"
  "internal/clients/camunda/v86/operate/client.gen.go"
  "internal/clients/camunda/v86/tasklist/client.gen.go"
  "internal/clients/camunda/v86/zeebe/client.gen.go"
)

git_ref_or_unknown() {
  local dir="$1"
  if [ ! -d "$dir/.git" ]; then
    echo "unknown"
    return 0
  fi

  if git -C "$dir" describe --tags --exact-match >/dev/null 2>&1; then
    git -C "$dir" describe --tags --exact-match
    return 0
  fi

  if git -C "$dir" describe --tags --always >/dev/null 2>&1; then
    git -C "$dir" describe --tags --always
    return 0
  fi

  echo "unknown"
}

resolve_camunda_v2_spec() {
  local legacy_path="./camunda/zeebe/gateway-protocol/src/main/proto/rest-api.yaml"
  local relocated_path="./camunda/zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml"

  if [ -f "$relocated_path" ]; then
    echo "$relocated_path"
    return 0
  fi

  if [ -f "$legacy_path" ]; then
    echo "$legacy_path"
    return 0
  fi

  echo "Could not find Camunda v2 OpenAPI spec in either expected location." >&2
  exit 1
}

bundle_camunda_v2_spec() {
  local spec_path="$1"

  if [[ "$spec_path" != *"/zeebe/gateway-protocol/src/main/proto/v2/"* ]]; then
    echo "$spec_path"
    return 0
  fi

  if ! command -v redocly >/dev/null 2>&1; then
    echo "missing tool: redocly" >&2
    exit 127
  fi

  local spec_dir
  local spec_name
  local bundled_path

  spec_dir="$(dirname "$spec_path")"
  spec_name="$(basename "${spec_path%.yaml}")"
  bundled_path="${spec_dir}/${spec_name}-bundled.yaml"

  redocly bundle "$spec_path" -o "$bundled_path" >/dev/null
  echo "$bundled_path"
}

CAMUNDA_V2_SPEC="$(resolve_camunda_v2_spec)"
CAMUNDA_V2_WORKING_SPEC="$(bundle_camunda_v2_spec "$CAMUNDA_V2_SPEC")"
CAMUNDA_V2_SPEC_DIR="$(dirname "$CAMUNDA_V2_WORKING_SPEC")"

# auth
./generate-go-client.sh ./auth/oauth2-openapi.json ../internal/clients/auth/oauth2/client.gen.go oauth2

# v89
./generate-go-client.sh ./camunda-docs/api/administration-sm/administration-sm-openapi.yaml ../internal/clients/camunda/v89/administrationsm/client.gen.go administrationsm
python3 ./mutations/mutate-search-query-schemas.py "$CAMUNDA_V2_WORKING_SPEC"
CAMUNDA_V2_V89_SEARCH_QUERY_PATCHED="${CAMUNDA_V2_SPEC_DIR}/$(basename "${CAMUNDA_V2_WORKING_SPEC%.yaml}")-search-query-patched.yaml"
python3 ./mutations/mutate-search-result-schemas.py "$CAMUNDA_V2_V89_SEARCH_QUERY_PATCHED"
CAMUNDA_V2_V89_SEARCH_RESULT_PATCHED="${CAMUNDA_V2_SPEC_DIR}/$(basename "${CAMUNDA_V2_WORKING_SPEC%.yaml}")-search-query-patched-search-result-patched.yaml"
python3 ./mutations/mutate-fix-process-instance-filter-fields.py "$CAMUNDA_V2_V89_SEARCH_RESULT_PATCHED"
CAMUNDA_V2_V89_PROCESS_INSTANCE_FIXED="${CAMUNDA_V2_SPEC_DIR}/$(basename "${CAMUNDA_V2_WORKING_SPEC%.yaml}")-search-query-patched-search-result-patched-process-instance-filter-fields-fixed.yaml"
python3 ./mutations/mutate-fix-jobresult-discriminator.py "$CAMUNDA_V2_V89_PROCESS_INSTANCE_FIXED"
CAMUNDA_V2_V89_JOBRESULT_FIXED="${CAMUNDA_V2_SPEC_DIR}/$(basename "${CAMUNDA_V2_WORKING_SPEC%.yaml}")-search-query-patched-search-result-patched-process-instance-filter-fields-fixed-jobresult-fixed.yaml"
python3 ./mutations/mutate-fix-camunda-v2-operation-id-collisions.py "$CAMUNDA_V2_V89_JOBRESULT_FIXED"
CAMUNDA_V2_V89_READY="${CAMUNDA_V2_SPEC_DIR}/$(basename "${CAMUNDA_V2_WORKING_SPEC%.yaml}")-search-query-patched-search-result-patched-process-instance-filter-fields-fixed-jobresult-fixed-opids-fixed.yaml"
./generate-go-client.sh "$CAMUNDA_V2_V89_READY" ../internal/clients/camunda/v89/camunda/client.gen.go camunda

python3 ./mutations/mutate-operation-ids.py ./camunda-docs/api/operate/operate-openapi.yaml
python3 ./mutations/mutate-remove-sort-values.py ./camunda-docs/api/operate/operate-openapi-oids-updated.yaml
python3 ./mutations/mutate-remove-v1-prefix.py ./camunda-docs/api/operate/operate-openapi-oids-updated-sortvalues-removed.yaml
./generate-go-client.sh ./camunda-docs/api/operate/operate-openapi-oids-updated-sortvalues-removed-v1-prefix-removed.yaml ../internal/clients/camunda/v89/operate/client.gen.go operate

python3 ./mutations/mutate-remove-v1-prefix.py ./camunda-docs/api/tasklist/tasklist-openapi.yaml
./generate-go-client.sh ./camunda-docs/api/tasklist/tasklist-openapi-v1-prefix-removed.yaml ../internal/clients/camunda/v89/tasklist/client.gen.go tasklist

# v88
./generate-go-client.sh ./camunda-docs/api/administration-sm/administration-sm-openapi.yaml ../internal/clients/camunda/v88/administrationsm/client.gen.go administrationsm

python3 ./mutations/mutate-search-query-schemas.py "$CAMUNDA_V2_WORKING_SPEC"
CAMUNDA_V2_SEARCH_QUERY_PATCHED="${CAMUNDA_V2_SPEC_DIR}/$(basename "${CAMUNDA_V2_WORKING_SPEC%.yaml}")-search-query-patched.yaml"
python3 ./mutations/mutate-search-result-schemas.py "$CAMUNDA_V2_SEARCH_QUERY_PATCHED"
CAMUNDA_V2_SEARCH_RESULT_PATCHED="${CAMUNDA_V2_SPEC_DIR}/$(basename "${CAMUNDA_V2_WORKING_SPEC%.yaml}")-search-query-patched-search-result-patched.yaml"
python3 ./mutations/mutate-fix-process-instance-filter-fields.py "$CAMUNDA_V2_SEARCH_RESULT_PATCHED"
CAMUNDA_V2_PROCESS_INSTANCE_FIXED="${CAMUNDA_V2_SPEC_DIR}/$(basename "${CAMUNDA_V2_WORKING_SPEC%.yaml}")-search-query-patched-search-result-patched-process-instance-filter-fields-fixed.yaml"
python3 ./mutations/mutate-fix-jobresult-discriminator.py "$CAMUNDA_V2_PROCESS_INSTANCE_FIXED"
CAMUNDA_V2_JOBRESULT_FIXED_SEARCH="${CAMUNDA_V2_SPEC_DIR}/$(basename "${CAMUNDA_V2_WORKING_SPEC%.yaml}")-search-query-patched-search-result-patched-process-instance-filter-fields-fixed-jobresult-fixed.yaml"
python3 ./mutations/mutate-fix-camunda-v2-operation-id-collisions.py "$CAMUNDA_V2_JOBRESULT_FIXED_SEARCH"
CAMUNDA_V2_V88_READY="${CAMUNDA_V2_SPEC_DIR}/$(basename "${CAMUNDA_V2_WORKING_SPEC%.yaml}")-search-query-patched-search-result-patched-process-instance-filter-fields-fixed-jobresult-fixed-opids-fixed.yaml"
./generate-go-client.sh "$CAMUNDA_V2_V88_READY" ../internal/clients/camunda/v88/camunda/client.gen.go camunda

python3 ./mutations/mutate-operation-ids.py ./camunda-docs/api/operate/operate-openapi.yaml
python3 ./mutations/mutate-remove-sort-values.py ./camunda-docs/api/operate/operate-openapi-oids-updated.yaml
python3 ./mutations/mutate-remove-v1-prefix.py ./camunda-docs/api/operate/operate-openapi-oids-updated-sortvalues-removed.yaml
./generate-go-client.sh ./camunda-docs/api/operate/operate-openapi-oids-updated-sortvalues-removed-v1-prefix-removed.yaml ../internal/clients/camunda/v88/operate/client.gen.go operate

python3 ./mutations/mutate-remove-v1-prefix.py ./camunda-docs/api/tasklist/tasklist-openapi.yaml
./generate-go-client.sh ./camunda-docs/api/tasklist/tasklist-openapi-v1-prefix-removed.yaml ../internal/clients/camunda/v88/tasklist/client.gen.go tasklist

# v87
./generate-go-client.sh ./camunda-docs/api/administration-sm/version-8.7/administration-sm-openapi.yaml ../internal/clients/camunda/v87/administrationsm/client.gen.go administrationsm
./generate-go-client.sh ./camunda-docs/api/camunda/version-8.7/camunda-openapi.yaml ../internal/clients/camunda/v87/camunda/client.gen.go camunda

python3 ./mutations/mutate-operation-ids.py ./camunda-docs/api/operate/version-8.7/operate-openapi.yaml
python3 ./mutations/mutate-remove-sort-values.py ./camunda-docs/api/operate/version-8.7/operate-openapi-oids-updated.yaml
python3 ./mutations/mutate-remove-v1-prefix.py ./camunda-docs/api/operate/version-8.7/operate-openapi-oids-updated-sortvalues-removed.yaml
./generate-go-client.sh ./camunda-docs/api/operate/version-8.7/operate-openapi-oids-updated-sortvalues-removed-v1-prefix-removed.yaml ../internal/clients/camunda/v87/operate/client.gen.go operate

python3 ./mutations/mutate-remove-v1-prefix.py ./camunda-docs/api/tasklist/version-8.7/tasklist-openapi.yaml
./generate-go-client.sh ./camunda-docs/api/tasklist/version-8.7/tasklist-openapi-v1-prefix-removed.yaml ../internal/clients/camunda/v87/tasklist/client.gen.go tasklist

# v86
./generate-go-client.sh ./camunda-docs/api/administration-sm/version-8.6/administration-sm-openapi.yaml ../internal/clients/camunda/v86/administrationsm/client.gen.go administrationsm
./generate-go-client.sh ./camunda-docs/api/camunda/version-8.6/camunda-openapi.yaml ../internal/clients/camunda/v86/camunda/client.gen.go camunda
python3 ./mutations/mutate-remove-v1-prefix.py ./camunda-docs/api/operate/version-8.6/operate-openapi.yaml
./generate-go-client.sh ./camunda-docs/api/operate/version-8.6/operate-openapi-v1-prefix-removed.yaml ../internal/clients/camunda/v86/operate/client.gen.go operate
python3 ./mutations/mutate-remove-v1-prefix.py ./camunda-docs/api/tasklist/version-8.6/tasklist-openapi.yaml
./generate-go-client.sh ./camunda-docs/api/tasklist/version-8.6/tasklist-openapi-v1-prefix-removed.yaml ../internal/clients/camunda/v86/tasklist/client.gen.go tasklist
./generate-go-client.sh ./camunda-docs/api/zeebe/version-8.6/zeebe-openapi.yaml ../internal/clients/camunda/v86/zeebe/client.gen.go zeebe

if [ "$DO_COMMIT" = true ]; then
  camunda_tag="$(git_ref_or_unknown "$SCRIPT_DIR/camunda")"
  camunda_docs_tag="$(git_ref_or_unknown "$SCRIPT_DIR/camunda-docs")"
  commit_subject="chore(clients): regenerate generated clients (camunda ${camunda_tag}, camunda-docs ${camunda_docs_tag})"
  commit_body=$'sources:\n'"- camunda: ${camunda_tag}"$'\n'"- camunda-docs: ${camunda_docs_tag}"

  git -C "$REPO_ROOT" add -- "${GENERATED_FILES[@]}"

  if git -C "$REPO_ROOT" diff --cached --quiet -- "${GENERATED_FILES[@]}"; then
    echo "No generated client changes to commit."
    exit 0
  fi

  git -C "$REPO_ROOT" commit -m "$commit_subject" -m "$commit_body"
fi
