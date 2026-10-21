# Research: Pragmatic Suspended Instance Regression Scope

**Date**: 2026-10-21

**Inspected baseline**: `ff74ebe817007b0de3b0b4cb5242e390cd31b41b`; branch `341-suspended-instance-hierarchies`.

**Method**: Read-only inspection of c8volt source/tests, Camunda documentation, and Camunda 8.10.4 engine source/tests. No local runtime tests, upstream tests, or live environment checks executed. Earlier broad planning included two read-only research agents; this revision uses the subsequent reassessment and supersedes those earlier design decisions, not their historical existence.

## 1. Preserve root cancellation

**Decision**: Keep existing root resolution, force restrictions, and ordinary cancellation. Do not resume instances or submit a separate cancellation for each child.

**Rationale**: Camunda permits cancellation while suspended. Its pinned tests explicitly cover cancellation of a suspended root and cascaded termination of a suspended call-activity child. c8volt already treats SUSPENDED as nonterminal and submits normal root cancellation.

**Alternatives considered**: A new shared mutation lifecycle, manual child cancellation, and a resume workaround add no demonstrated suspension benefit.

Evidence:

- [Camunda suspension documentation](https://docs.camunda.io/docs/components/concepts/process-instance-suspension/).
- [8.10.4 ProcessInstanceSuspensionGateTest](https://github.com/camunda/camunda/blob/8.10.4/zeebe/engine/src/test/java/io/camunda/zeebe/engine/processing/processinstance/ProcessInstanceSuspensionGateTest.java), including `shouldCancelProcessInstanceWhileSuspended` and `shouldClearSuspensionStateForCallActivityChildInstanceOnCascadedTermination`.
- `internal/domain/state.go`: SUSPENDED is nonterminal.
- `internal/services/processinstance/v810/service.go`: `CancelProcessInstance` uses ancestry/force escalation and the existing cancellation endpoint.

## 2. Test existing overlap handling, do not redesign it

**Decision**: Add focused evidence for the normal CLI planning path only.

**Rationale**: `DryRunCancelOrDeletePlan` already deduplicates resolved roots, and CLI cancellation dispatches those roots. Direct bulk library calls deduplicate literal inputs and have different boundaries. This is a preexisting distinction, not evidence that suspension causes duplicate engine execution. Camunda rejects cancellation already in progress.

**Alternatives considered**: A shared invocation ledger and a universal deduplication guarantee would widen the task without a suspension reproducer.

Evidence: `internal/services/processinstance/dryrun.go`, `bulk.go`, `cmd/cancel_processinstance.go`, and the [8.10.4 cancellation processor](https://github.com/camunda/camunda/blob/8.10.4/zeebe/engine/src/main/java/io/camunda/zeebe/engine/processing/processinstance/ProcessInstanceCancelProcessor.java).

## 3. Preserve deletion order and opt-outs

**Decision**: Verify suspended instances using descendant-first deletion and existing conflict-triggered cancellation recovery. Preserve terminal waiting before the recovery retry, including with NoWait. Retain existing NoStateCheck/NoWait request boundaries and terminal/absent behavior.

**Rationale**: The current implementation already waits for terminal/absent states before retrying a conflicted forced deletion. Command help explicitly documents that the entire scope is not canceled before any deletion. The earlier whole-tree pre-cancel proposal changed working behavior without a demonstrated suspension need.

**Alternatives considered**: New preflight deletion gates, mandatory child-absence confirmation, changed absence retries, and additional opt-out reads are outside this scope.

Evidence: `internal/services/processinstance/v810/service.go:DeleteProcessInstance`, `cmd/delete_processinstance.go`, and existing `TestService_CancelAndDeleteProcessInstance` cases, including `CancelPreservesOptOutReadAndSubmissionBoundaries` and force recovery with NoWait.

## 4. Fill focused coverage gaps

**Decision**: Extend mutation tests with suspended roots, both mixed-tree combinations, persistent suspended observations, and force recovery with and without NoWait. Keep existing timeout/error representations and retry helpers.

**Rationale**: Existing tests cover SUSPENDED parsing, searches, creation visibility, and generic waits. Inspected 8.10 cancel/delete tests cover terminal-family and failure behavior but lack explicit suspended mutation scenarios. A state remaining suspended must not satisfy terminal confirmation. Listener-related delay can be represented by controlled observations without implementing listener orchestration or claiming an actual listener diagnosis.

**Alternatives considered**: New outcome types, public mutation fields, and retry reconciliation are unnecessary unless a separate concrete defect justifies them.

Evidence: `internal/services/processinstance/v810/service_test.go`, `internal/services/processinstance/waiter/waiter_test.go:TestSuspendedStateWaits`, and `cmd/processinstance_suspended_test.go`. Camunda also tests suspended cancellation with user-task canceling listeners in [UserTaskSuspensionGateTest](https://github.com/camunda/camunda/blob/8.10.4/zeebe/engine/src/test/java/io/camunda/zeebe/engine/processing/usertask/UserTaskSuspensionGateTest.java).

## 5. Keep the concrete definition gap in #340

**Decision**: Definition counts, ACTIVE-plus-SUSPENDED discovery, and purge orchestration remain in #340. Reuse existing instance operations; do not require a broad #341 framework first.

**Rationale**: `completeProcessDefinitionDeleteImpact` returns early when active count is zero; `listActiveProcessInstancesForDefinition` filters only ACTIVE. The 8.10 statistics adapter does not count SUSPENDED. Camunda documents that ACTIVE searches exclude suspended instances. This is a concrete selection/counting gap, unlike the speculative cancellation redesign. It is not evidence that Camunda would delete unfinished execution.

**Alternatives considered**: Moving definition discovery into #341 confuses ownership and delays the demonstrated fix.

Evidence: `internal/services/processdefinition/delete.go`, `internal/services/processdefinition/v810/service.go`, [issue #340](https://github.com/grafvonb/c8volt/issues/340), and the suspension documentation above.

## 6. Small documentation correction; isolate unrelated findings

**Decision**: Correct cancel/delete state help to include suspended and 8.10 applicability. No runtime semantics change is intended.

**Rationale**: Shared validation recognizes suspended, but both command-specific state flag descriptions omit it.

**Alternatives considered**: Broad output/error redesign is not required to correct selector help.

General observations from the earlier investigation, such as direct-library overlap, direct-child page limits, and ignored unsuccessful response values, may warrant separate reproductions and issues. They are not accepted suspension defects or mandatory #341 work. Retention already excludes nonterminal instances and needs no behavior change.

## Validation boundary

Use isolated fake clients/HTTP servers and controlled observations. No environment discovery or live mutation is an implementation gate. Upstream tests are source evidence, not tests executed by this workflow. The earlier Camunda 8.10.4 live-validation proposal is retained here only as historical context for a separately requested workflow; generated-client provenance remains unchanged. No unresolved design question requires a new architecture or user decision.
