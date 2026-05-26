#!/usr/bin/env bash

set -u

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
LIB="$ROOT_DIR/integration/scripts/lib/suite-lib.sh"
# shellcheck source=integration/scripts/lib/suite-lib.sh
. "$LIB"

IT_SUITE_NAME="${IT_SUITE_NAME:-c89}"
IT_TARGET_VERSION="${IT_TARGET_VERSION:-8.9}"
IT_FIXTURE_PREFIX="${IT_FIXTURE_PREFIX:-C89}"
IT_CONFIG="${C8VOLT_IT_CONFIG:-./config.yaml}"
IT_PROFILE="${C8VOLT_IT_PROFILE:-}"
IT_BUILD="${C8VOLT_IT_BUILD:-1}"
IT_BIN="${C8VOLT_IT_BIN:-/tmp/c8volt-it-${IT_SUITE_NAME}}"
IT_VOLUME_COUNT="${C8VOLT_IT_VOLUME_COUNT:-12}"
IT_FULL_FORCE="${C8VOLT_IT_FULL_FORCE:-1}"
IT_KEEP_DATA="${C8VOLT_IT_KEEP_DATA:-0}"
IT_WORKDIR="${C8VOLT_IT_WORKDIR:-/tmp/c8volt-it-${IT_SUITE_NAME}-$(date -u +%Y%m%dT%H%M%SZ)}"
IT_REPORT="$IT_WORKDIR/report.md"
IT_PROGRESS="$IT_WORKDIR/progress.tsv"
IT_RUN_ID="${C8VOLT_IT_RUN_ID:-it-${IT_SUITE_NAME}-$(date -u +%Y%m%dT%H%M%SZ)-$$}"

it_init_report

cd "$ROOT_DIR" || exit 2

common_args=(--config "$IT_CONFIG")
if [ -n "$IT_PROFILE" ]; then
  common_args+=(--profile "$IT_PROFILE")
fi

c8() {
  "$IT_BIN" "${common_args[@]}" "$@"
}

if [ "$IT_BUILD" != "0" ]; then
  it_run_case "Build" "go-build" "local" "pass" go build -o "$IT_BIN" .
else
  it_run_case "Build" "use-existing-binary" "local" "pass" test -x "$IT_BIN"
fi

it_run_case "Preflight" "cli-version" "read-only" "pass" c8 version
it_run_case "Preflight" "config-validate" "read-only" "pass" c8 config validate
it_run_case "Preflight" "config-show-json" "read-only" "pass" c8 config show --json
it_run_case "Preflight" "config-test-connection" "read-only" "pass" c8 config test-connection
it_run_case "Preflight" "config-test-connection-json" "read-only" "pass" c8 config test-connection --json
it_run_case "Preflight" "cluster-version" "read-only" "pass" c8 get cluster version
if ! it_require_output_contains "Preflight" "cluster-version" "$IT_TARGET_VERSION"; then
  printf "Target version gate failed. See %s\n" "$IT_REPORT"
  it_finish_report
  exit 2
fi
it_run_case "Preflight" "cluster-topology" "read-only" "pass" c8 get cluster topology
it_run_case "Preflight" "capabilities-json" "read-only" "pass" c8 capabilities --json

it_run_case "Command Inventory" "root-help" "read-only" "pass" c8 --help
it_run_case "Command Inventory" "get-help" "read-only" "pass" c8 get --help
it_run_case "Command Inventory" "run-help" "read-only" "pass" c8 run --help
it_run_case "Command Inventory" "cancel-help" "read-only" "pass" c8 cancel --help
it_run_case "Command Inventory" "delete-help" "read-only" "pass" c8 delete --help
it_run_case "Command Inventory" "update-help" "read-only" "pass" c8 update --help
it_run_case "Command Inventory" "resolve-help" "read-only" "pass" c8 resolve --help
it_run_case "Command Inventory" "ops-help" "read-only" "pass" c8 ops --help
it_run_case "Command Inventory" "ops-execute-help" "read-only" "pass" c8 ops execute --help
it_run_case "Command Inventory" "ops-purge-help" "read-only" "pass" c8 ops purge --help
it_run_case "Command Inventory" "ops-repair-help" "read-only" "pass" c8 ops repair --help

simple_user_task="${IT_FIXTURE_PREFIX}_SimpleUserTask"
service_task="${IT_FIXTURE_PREFIX}_SimpleServiceTask"
incident_process="${IT_FIXTURE_PREFIX}_SimpleUserTaskWithIncident"
multi_subprocess="${IT_FIXTURE_PREFIX}_MultipleSubProcessesParent"
pd_fixture="processdefinitions/${simple_user_task}.bpmn"

it_run_case "Fixture Setup" "embed-list" "read-only" "pass" c8 embed list
it_run_case "Fixture Setup" "deploy-user-task-fixture" "mutation" "pass" c8 embed deploy --file "$pd_fixture"

vars_json="{\"releaseRunId\":\"${IT_RUN_ID}\",\"suite\":\"${IT_SUITE_NAME}\",\"customerTier\":\"bronze\",\"status\":\"pending\",\"payload\":\"integration-volume\"}"
keys_file="$IT_WORKDIR/data/user-task.keys"
it_run_capture "Data Generation" "run-volume-user-task-processes" "mutation" "pass" "$keys_file" c8 run pi -b "$simple_user_task" -n "$IT_VOLUME_COUNT" --workers 4 --vars "$vars_json" --keys-only

first_key="$(it_first_key "$keys_file" || true)"
key_count="$(awk '/^[0-9]+$/ { count++ } END { print count + 0 }' "$keys_file" 2>/dev/null || printf "0")"
{
  printf "\n## Generated Data\n\n"
  printf -- '- run id: `%s`\n' "$IT_RUN_ID"
  printf -- '- user-task process id: `%s`\n' "$simple_user_task"
  printf -- '- requested volume count: `%s`\n' "$IT_VOLUME_COUNT"
  printf -- '- captured process-instance keys: `%s`\n' "$key_count"
  printf -- '- key file: `%s`\n' "$keys_file"
  printf -- '- dirty-cluster full force: `%s`\n' "$IT_FULL_FORCE"
} >> "$IT_REPORT"

it_run_case "Read Workflows" "get-pd-latest" "read-only" "pass" c8 get pd --bpmn-process-id "$simple_user_task" --latest
it_run_case "Read Workflows" "get-pi-active-limited" "read-only" "pass" c8 get pi --bpmn-process-id "$simple_user_task" --state active --limit 5
it_run_case "Read Workflows" "get-pi-total" "read-only" "pass" c8 get pi --bpmn-process-id "$simple_user_task" --state active --total
it_run_case "Read Workflows" "get-pi-with-vars-limited" "read-only" "pass" c8 get pi --bpmn-process-id "$simple_user_task" --state active --with-vars --var-value-limit 120 --limit 5
it_run_case "Read Workflows" "get-pi-var-exists" "read-only" "pass" c8 get pi --var-exists releaseRunId --limit 5
it_run_case "Read Workflows" "get-pi-var-eq" "read-only" "pass" c8 get pi --var "releaseRunId=\"${IT_RUN_ID}\"" --limit 5
it_run_case "Read Workflows" "get-incident-active-limited" "read-only" "optional" c8 get incident --state active --limit 5
it_run_case "Read Workflows" "get-job-limited" "read-only" "optional" c8 get job --limit 5

if [ -n "$first_key" ]; then
  update_key_args=()
  while IFS= read -r key_arg; do
    update_key_args+=("$key_arg")
  done < <(it_key_args_from_file "$keys_file" 3)
  update_vars_json="{\"releaseRunId\":\"${IT_RUN_ID}\",\"customerTier\":\"gold\",\"status\":\"updated\"}"
  it_run_case "Keyed Workflows" "get-first-pi" "read-only" "pass" c8 get pi --key "$first_key"
  it_run_case "Keyed Workflows" "get-first-pi-with-vars" "read-only" "pass" c8 get pi --key "$first_key" --with-vars --var-value-limit 120
  it_run_case "Keyed Workflows" "walk-first-pi" "read-only" "pass" c8 walk pi --key "$first_key"
  it_run_case "Mutating Workflows" "update-pi-dry-run" "dry-run" "pass" c8 update pi "${update_key_args[@]}" --vars "$update_vars_json" --dry-run
  it_run_case "Mutating Workflows" "update-pi-real" "mutation" "pass" c8 --auto-confirm update pi "${update_key_args[@]}" --vars "$update_vars_json"
  it_run_case "Mutating Workflows" "update-pi-post-check" "read-only" "pass" c8 get pi --key "$first_key" --with-vars --var-value-limit 120
  it_run_case "Mutating Workflows" "cancel-pi-dry-run" "dry-run" "pass" c8 cancel pi --key "$first_key" --dry-run
  it_run_case "Mutating Workflows" "cancel-pi-real" "mutation" "pass" c8 --auto-confirm cancel pi --key "$first_key"
  it_run_case "Mutating Workflows" "cancel-pi-post-check" "read-only" "pass" c8 expect pi --key "$first_key" --state canceled
  it_run_case "Mutating Workflows" "delete-pi-dry-run" "dry-run" "pass" c8 delete pi --key "$first_key" --dry-run
  it_run_case "Mutating Workflows" "delete-pi-real" "mutation" "pass" c8 --auto-confirm delete pi --key "$first_key"
  it_run_case "Mutating Workflows" "delete-pi-post-check" "read-only" "pass" c8 expect pi --key "$first_key" --state absent
else
  it_skip_case "Keyed Workflows" "first-key-dependent-cases" "mixed" "no process-instance key captured"
  it_skip_case "Mutating Workflows" "keyed-mutation-cases" "mutation" "no process-instance key captured"
fi

it_run_case "Ops Workflows" "smoke-test-dry-run" "dry-run" "pass" c8 ops execute smoke-test --count 3 --dry-run --report-file "$IT_WORKDIR/reports/smoke-test-dry-run.md"
it_run_case "Ops Workflows" "smoke-test-real" "mutation" "pass" c8 --auto-confirm ops execute smoke-test --count 3 --report-file "$IT_WORKDIR/reports/smoke-test-real.md"
it_run_case "Ops Workflows" "retention-policy-dry-run" "dry-run" "pass" c8 ops execute retention-policy --retention-days 90 --dry-run
it_run_case "Ops Workflows" "purge-orphan-pi-dry-run" "dry-run" "pass" c8 ops purge orphan-process-instances --dry-run
it_run_case "Ops Workflows" "purge-pi-with-incidents-dry-run" "dry-run" "pass" c8 ops purge process-instances-with-incidents --dry-run
if [ "$IT_FULL_FORCE" = "1" ]; then
  it_run_case "Ops Workflows" "purge-orphan-pi-real-full-force" "mutation" "optional" c8 --auto-confirm ops purge orphan-process-instances --force --report-file "$IT_WORKDIR/reports/orphan-pi-purge-real-${IT_RUN_ID}.md"
  it_run_case "Ops Workflows" "purge-pi-with-incidents-real-full-force" "mutation" "optional" c8 --auto-confirm ops purge process-instances-with-incidents --force --report-file "$IT_WORKDIR/reports/pi-with-incidents-purge-real-${IT_RUN_ID}.md"
else
  it_skip_case "Ops Workflows" "purge-orphan-pi-real-full-force" "mutation" "C8VOLT_IT_FULL_FORCE!=1"
  it_skip_case "Ops Workflows" "purge-pi-with-incidents-real-full-force" "mutation" "C8VOLT_IT_FULL_FORCE!=1"
fi
it_run_case "Ops Workflows" "repair-process-instance-search-dry-run" "dry-run" "optional" c8 ops repair process-instance --bpmn-process-id "$incident_process" --limit 5 --dry-run
it_run_case "Ops Workflows" "repair-incident-search-dry-run" "dry-run" "optional" c8 ops repair incident --limit 5 --dry-run

if [ "$IT_TARGET_VERSION" = "8.8" ]; then
  it_run_case "Version Gates" "delete-pd-c88-unsupported-full-force" "expected unsupported" "fail" c8 delete pd --bpmn-process-id "$simple_user_task" --latest --force --auto-confirm
  it_require_output_contains "Version Gates" "delete-pd-c88-unsupported-full-force" "unsupported|requires Camunda 8\\.9" || IT_FAILS=$((IT_FAILS + 1))
  it_run_case "Version Gates" "purge-all-pd-c88-unsupported-dry-run-broad" "expected unsupported" "fail" c8 ops purge all-process-definitions --dry-run
  it_require_output_contains "Version Gates" "purge-all-pd-c88-unsupported-dry-run-broad" "unsupported|requires Camunda 8\\.9" || IT_FAILS=$((IT_FAILS + 1))
  if [ "$IT_FULL_FORCE" = "1" ]; then
    it_run_case "Version Gates" "purge-all-pd-c88-unsupported-real-full-force" "expected unsupported" "fail" c8 --auto-confirm ops purge all-process-definitions --force --report-file "$IT_WORKDIR/reports/all-pd-purge-c88-unsupported-real-${IT_RUN_ID}.md"
    it_require_output_contains "Version Gates" "purge-all-pd-c88-unsupported-real-full-force" "unsupported|requires Camunda 8\\.9" || IT_FAILS=$((IT_FAILS + 1))
  else
    it_skip_case "Version Gates" "purge-all-pd-c88-unsupported-real-full-force" "expected unsupported" "C8VOLT_IT_FULL_FORCE!=1"
  fi
else
  pd_cleanup_report="$IT_WORKDIR/reports/all-pd-purge-${IT_RUN_ID}.md"
  it_run_case "C89 Destructive Workflows" "purge-all-pd-dry-run-scoped" "dry-run" "pass" c8 ops purge all-process-definitions --bpmn-process-id "$simple_user_task" --latest --dry-run --report-file "$pd_cleanup_report"
  if [ "$IT_FULL_FORCE" = "1" ]; then
    it_run_case "C89 Destructive Workflows" "purge-all-pd-real-scoped" "mutation" "optional" c8 --auto-confirm ops purge all-process-definitions --bpmn-process-id "$simple_user_task" --latest --force --report-file "$IT_WORKDIR/reports/all-pd-purge-real-${IT_RUN_ID}.md"
    it_run_case "C89 Destructive Workflows" "purge-all-pd-dry-run-broad" "dry-run" "pass" c8 ops purge all-process-definitions --dry-run --report-file "$IT_WORKDIR/reports/all-pd-purge-broad-dry-run-${IT_RUN_ID}.md"
    it_run_case "C89 Destructive Workflows" "purge-all-pd-real-broad-full-force" "mutation" "pass" c8 --auto-confirm ops purge all-process-definitions --force --report-file "$IT_WORKDIR/reports/all-pd-purge-broad-real-${IT_RUN_ID}.md"
  else
    it_skip_case "C89 Destructive Workflows" "purge-all-pd-real-scoped" "mutation" "C8VOLT_IT_FULL_FORCE!=1"
    it_skip_case "C89 Destructive Workflows" "purge-all-pd-dry-run-broad" "dry-run" "C8VOLT_IT_FULL_FORCE!=1"
    it_skip_case "C89 Destructive Workflows" "purge-all-pd-real-broad-full-force" "mutation" "C8VOLT_IT_FULL_FORCE!=1"
  fi
fi

if [ "$IT_KEEP_DATA" = "0" ]; then
  if [ "$IT_FULL_FORCE" = "1" ]; then
    it_run_case "Cleanup" "cancel-dirty-active-pis-full-force" "mutation" "optional" c8 --auto-confirm cancel pi --state active --limit "$IT_VOLUME_COUNT" --force
    it_run_case "Cleanup" "delete-dirty-canceled-pis-full-force" "mutation" "optional" c8 --auto-confirm delete pi --state canceled --limit "$IT_VOLUME_COUNT" --force
  else
    it_run_case "Cleanup" "cancel-fixture-active-pis" "mutation" "optional" c8 --auto-confirm cancel pi --bpmn-process-id "$simple_user_task" --state active --limit "$IT_VOLUME_COUNT"
    it_run_case "Cleanup" "delete-fixture-canceled-pis" "mutation" "optional" c8 --auto-confirm delete pi --bpmn-process-id "$simple_user_task" --state canceled --limit "$IT_VOLUME_COUNT"
  fi
else
  it_skip_case "Cleanup" "run-owned-data" "mutation" "C8VOLT_IT_KEEP_DATA=1"
fi

it_finish_report

printf "\nReport: %s\n" "$IT_REPORT"
printf "Progress: %s\n" "$IT_PROGRESS"

if [ "$IT_FAILS" -gt 0 ] || [ "$IT_EXPECTED_FAIL_MISSES" -gt 0 ]; then
  exit 1
fi
