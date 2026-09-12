# Quickstart Validation: Force-Cleanup Progress

## Prerequisites

Work from the repository root with its Go 1.26 toolchain available. Read [spec.md](spec.md), [data-model.md](data-model.md), [the progress contract](contracts/force-cleanup-progress.md), and `specs/ralph-implementation-rules.md` before implementation. This guide describes validation after implementation; planning alone does not make the new scenarios pass.

## 1. Focused command regressions

```sh
go test ./cmd -run 'TestOpsPurgeAllProcessDefinitions|TestOpsSemanticProgressReporter|TestNewOpsSemanticProgressReporter' -count=1
```

New coordinator and real nested command tests should retain the `TestOpsPurgeAllProcessDefinitions` prefix so this command includes them. Confirm verbose test output lists the new nested-sequence cases; a pattern matching only existing isolated callback tests is insufficient.

Expected evidence: cancellation activity before the first completed root, drain activity during a blocked wait, fresh history counters, definition counters only once definition deletion starts, no overwritten activity from nested requests. Verify normal, dry-run, declined confirmation, auto-confirmed, empty and no-cleanup paths.

Use deterministic fake clocks for 9.999s/10s boundaries and the cross-stage timeline in the contract. Assert zero timer-only milestones, immediate failure visibility, a single final record retaining earlier dirty stages, idempotent close, and silence on clean short runs. Use barriers rather than real 10-second sleeps.

## 2. Callback mappings and service invariants

```sh
go test ./internal/domain -run 'Test.*Progress' -count=1
go test ./c8volt/ops -run 'TestProgressConversions|TestClientPurgeAllProcessDefinitions' -count=1
go test ./c8volt/foptions -run 'Test.*Progress' -count=1
go test ./internal/services/processdefinition/... -run 'Test.*(DeleteProcessDefinition|CleanupProcessDefinition)' -count=1
go test ./internal/services/ops/... -run 'TestPurgeAllProcessDefinitions' -count=1
go test ./internal/services/processinstance/... -run 'Test.*(Progress|Completion|CancelProcessInstances|DeleteProcessInstances)' -count=1
```

Name new progress-conversion tests to match these prefixes. Verify nil and known-zero optional stage counts survive mapping without pointer aliasing. Extend the existing shared-root ops test to assert all stage entries and nested completion phases. Also execute non-force deletion through its ordinary worker path and assert one definition-stage entry before the first resource request; it does not traverse the force path's bulk resource helper. Preserve worker arguments, request shape, first serial deletion probe, preview rechecks, and wait/no-wait behavior. No new remote reads or mutations should appear.

## 3. Concurrent completion and mode safety

```sh
go test ./cmd ./internal/services/processdefinition/... ./internal/services/ops/... -race -run 'Test.*(PurgeAllProcessDefinitions|SemanticProgress|CleanupProcessDefinition|DeleteProcessDefinition)' -count=1
```

Use synchronized request/event collectors. Avoid parallel tests that mutate command flags, clocks, or global command trees. Check JSON and automation output (including verbose combinations), quiet failures, and existing keys-only policy/unsupported-mode coverage. Compare final result, audit report, and exit behavior independently of transient output.

## 4. Documentation and full validation

Update command help, README progress guidance, and `docs/ops/purge-all-process-definitions.md`. Run targeted `gofmt` on touched Go files, then:

```sh
make docs-content
go test ./cmd -run 'TestOpsPurgeAllProcessDefinitionsHelpDocumentsCommandShape' -count=1
make test
git diff --check
```

Review generated CLI documentation and homepage changes; do not hand-edit generated files. Confirm the docs explain force-cleanup stages and discovery-only `--batch-size`. The full test gate is required before committing an implementation work unit. Record blocked or failing checks explicitly rather than treating them as passes.

## 5. Optional terminal demonstration

The deterministic command fake-backend tests above provide the required repeatable end-to-end proof. For a visual demonstration, use an isolated supported Camunda 8.9/8.10 test environment and fixtures owned by this test run. Create at least two unique active root trees and multiple selected definitions, including nested instances, using the project's existing fixture workflow.

Build the CLI and supply a dedicated test configuration and test BPMN process ID:

```sh
make build
read -r -p 'Test config path: ' C8VOLT_PROGRESS_TEST_CONFIG
read -r -p 'Owned test BPMN process ID: ' C8VOLT_PROGRESS_TEST_BPMN_ID
: "${C8VOLT_PROGRESS_TEST_CONFIG:?required}" "${C8VOLT_PROGRESS_TEST_BPMN_ID:?required}"
./bin/c8volt --config "$C8VOLT_PROGRESS_TEST_CONFIG" ops purge all-process-definitions --bpmn-process-id "$C8VOLT_PROGRESS_TEST_BPMN_ID" --force --dry-run
./bin/c8volt --config "$C8VOLT_PROGRESS_TEST_CONFIG" ops purge all-process-definitions --bpmn-process-id "$C8VOLT_PROGRESS_TEST_BPMN_ID" --force
```

These setup commands use Bash. Review the frozen preview and confirm only the test-owned scope. The final command deletes that scope. Observe actual stages in a terminal; a fast drain may be too brief to inspect visually, which is why the barrier-controlled acceptance test is authoritative. Recreate fixtures before trying verbose or machine variants. No live backend mutation is performed as part of this planning command.
