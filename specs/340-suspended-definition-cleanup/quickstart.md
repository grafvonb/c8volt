# Quickstart: Isolated Validation for #340

This guide is for the later implementation phase. None of these runtime checks ran during planning. Validate only test-owned fakes; never run integration/smoke targets or connect to an existing Camunda service.

## Prerequisites

- Use the existing repository Go toolchain/dependencies and the feature branch.
- Read [spec.md](spec.md), [plan.md](plan.md), [data-model.md](data-model.md), and [contract](contracts/definition-cleanup.md).
- Inspect selected tests and helpers before execution. They must use generated-client mocks, service stubs, test-owned HTTP servers or isolated command/terminal fixtures. A test name containing “real” does not establish a live dependency, and a local hostname does not establish isolation.
- Remove inherited live-test configuration and build flags. Do not specify profiles, credentials, integration tags, real endpoints, or Kubernetes contexts. Do not build/start an engine or deploy BPMN.

From the repository root, this wrapper clears the harness/application environment and Go flags for an individually selected isolated command:

```sh
isolated_go_test() {
  python3 - "$@" <<'PY'
import os
import subprocess
import sys
blocked_prefixes = ('C8VOLT_', 'IT_', 'GO_WANT_')
env = {k: v for k, v in os.environ.items()
       if not k.startswith(blocked_prefixes) and k not in ('GOFLAGS', 'GOENV', 'GORACE')}
env.update(GOENV='off', GOFLAGS='', GOCACHE='/tmp/c8volt-gocache')
raise SystemExit(subprocess.run(['go', 'test', *sys.argv[1:]], env=env).returncode)
PY
}
```

This is not a substitute for inspecting the test setup. Do not add live-test tags or environment-backed helpers to the commands below.

## 1. Statistics, selector and conversion checks

After implementing the corresponding behavior, select the closest package/test patterns:

```sh
isolated_go_test ./internal/services/processdefinition/v810 -run 'TestService_(GetProcessDefinition|SearchProcessDefinitions)|Suspended|Unfinished' -count=1
isolated_go_test ./internal/services/processinstance/... -run 'Search|Suspended|Unfinished|StateFilter' -count=1
isolated_go_test ./internal/services/processdefinition/v88 ./internal/services/processdefinition/v89 -run 'TestService_(GetProcessDefinition|SearchProcessDefinitions)' -count=1
isolated_go_test ./c8volt/process ./c8volt/resource ./c8volt/ops -run 'ProcessDefinition|Suspended|Unfinished' -count=1
```

Use existing service statistics fixtures such as `expectedProcessDefinitionStats` and `mockProcessDefinitionStats`; extend their request expectations rather than replacing the harness. Add descriptive isolated tests containing Suspended/Unfinished where appropriate; verify that each selected package actually ran the intended tests rather than reporting no tests.

Expected evidence: active fields retain their values; suspended support distinguishes zero from unavailable; union requests contain both states in one predicate; ambiguous/unsupported filters fail before I/O; capped and incomplete observations cannot produce an authoritative false zero; older request sets remain unchanged; all affected facade mappings preserve additions.

## 2. Cleanup and purge sequencing

```sh
isolated_go_test ./internal/services/processdefinition -run 'DeleteProcessDefinition|CleanupProcessDefinition|Suspended|Unfinished' -count=1
isolated_go_test ./internal/services/ops -run 'TestPurgeAllProcessDefinitions|Suspended|Unfinished' -count=1
isolated_go_test ./c8volt/resource -run 'TestClient_(DeleteProcessDefinition|PreviewDeleteProcessDefinition)' -count=1
```

Existing anchors include:

- `TestPreviewDeleteProcessDefinitionsNoStateCheckUsesBatchActivity`
- `TestCleanupProcessDefinitionDeletePlanForceScopeStopsAfterCancellationFailure`
- `TestCleanupProcessDefinitionDeletePlanForceScopeStopsAfterDrainFailure`
- `TestCleanupProcessDefinitionDeletePlanForceScopeStopsAfterHistoryFailure`
- `TestCleanupProcessDefinitionDeletePlanForceScopeUsesRequestedWorkers`
- `TestDeleteProcessDefinitionResourcesWaitsForDefinitionAbsenceAfterBatchCompletion`
- `TestDeleteProcessDefinitionResourcesEmitsSubmittedCompletionFactsForNoWait`
- `TestPurgeAllProcessDefinitionsBlocksUnsafeActiveInstancesWithoutForce`
- `TestPurgeAllProcessDefinitionsForceCleanupDeduplicatesProcessInstanceRoots`
- `TestPurgeAllProcessDefinitionsMapsExecutionControlsToDeletePath`
- `TestPurgeAllProcessDefinitionsLimitStopsAfterFrozenScopeCap`
- `TestPurgeAllProcessDefinitionsRejectsUnsupportedFullHistoryVersionsBeforeDiscovery`

Extend these fake-backed setups with suspended-only and both mixed trees; preserve active controls. Assert actual request order/counts, no resume or added child cancellation, one successful root dispatch for overlaps, and no dependent deletion after failed/unconfirmed prerequisites. Include no-wait, no-state-check, genuine empty/terminal-only, read failure, advancing sparse pages, stalled/indeterminate continuation, and ACTIVE/SUSPENDED transitions. Prove additional unfinished observations do not cause cancellation expansion or false completion.

## 3. CLI output, watch, reports and controls

```sh
isolated_go_test ./cmd -run 'ProcessDefinition|OpsPurgeAllProcessDefinitions|Suspended|Unfinished' -count=1
```

Start with narrower named tests within this group when the diff affects only a specific renderer or command. Reuse `cmd_views_processdefinition_test.go`, existing watch/terminal fixtures, delete-definition command fixtures, and purge command/report tests.

Expected evidence:

- Human statistics show supported `su:` and omit unsupported data; JSON preserves active values and includes support metadata according to omission rules.
- A suspended-only change refreshes watch output; zero/support transitions remove stale data without changing watch timing or DRAINING display.
- Dry-run, force blockers, progress, human summaries, Markdown reports, JSON reports and affected-key projections describe the same unfinished workload.
- Diagnostics satisfy configured logger/stream/severity and compact-error requirements in the contract. Include repeated distinctive backend HTTP errors, normal/verbose/quiet, one-envelope JSON with EOF and original exit/partial results, and redacted DEBUG via flag and configuration. Assertions must reject direct-print logger bypasses.
- Empty/prompt-sensitive paths use real terminal stdin with separate stdout/stderr captures. Empty success has no mutation/confirmation, exact human wording, zero-byte keys, one JSON envelope, and valid quiet/automation/auto-confirm/no-wait behavior. Preserve nonempty and abort controls.

These are execution-path checks against fakes, not manual invocations of the configured CLI.

## 4. Formatting and documentation

During implementation, format touched Go files with targeted `gofmt`, update command metadata/README, and regenerate documentation using `make docs-content` after inspecting that target. Do not hand-edit generated CLI references.

For planning/documentation-only edits, run only lightweight document validation and diff review. `git diff --check` does not inspect untracked file contents, so also inspect new artifact whitespace and local links directly.

Broaden runtime validation only when the actual implementation diff, shared behavior, or unresolved failures justify it. Do not rerun passed checks merely to commit. Do not invoke `make integration-test*`, `make integration-cli-*`, live tags, smoke commands, or manual engine validation.

## Evidence to record

Record the revision, actual isolated commands/results, request/sequence and output coverage, and any remaining limitation. A passing fake-backed check is not a passing real-engine test. No backend timing, listener or exporter guarantee is added by this guide; #342 remains a separate explicitly requested workflow.
