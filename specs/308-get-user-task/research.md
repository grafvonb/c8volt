# Research: Get User Tasks

**Date**: 2026-09-13
**Scope**: Issue #308 and the checked-in Camunda contracts. Research is based on this repository revision, not assumptions about newer upstream releases.

## 1. Preserve the legacy resolver; add native reads

**Decision**: Keep `internal/services/usertask.API.GetUserTask` and `ResolveProcessInstanceKeys` behavior intact. Add `GetNativeUserTask` and `SearchUserTasksPage` to the existing adapter contracts. Public `task.GetUserTask` will delegate to the new native getter. The public resolver methods continue to use the legacy getter.

**Rationale**: In v88/v89 the existing getter performs tenant-scoped search and can fall back to Tasklist; v810 performs native GET with tenant and identity checks. Replacing it would change `get pi --has-user-tasks`. The new command requires direct backend-authorized keys without discovery tenant filtering and no Tasklist fallback.

**Alternatives considered**: Reusing or changing the old getter would couple two different tenant and fallback contracts. Creating another service area would duplicate established ownership.

**Evidence**: `internal/services/usertask/{api.go,workflow.go}`, `internal/services/usertask/{v88,v89,v810}/service.go`, `c8volt/task/{api.go,client.go}`, `cmd/get_processinstance_user_tasks.go`.

## 2. Generated clients already cover the feature

**Decision**: Use native `GetUserTaskWithResponse` and `SearchUserTasksWithResponse` in v88, v89, and v810. v87 implements the new methods with the established unsupported domain error. Keep the factory and default Camunda version unchanged.

**Rationale**: All three generated clients already contain the endpoints and the requested fields. v88/v89 process-instance key, process-definition key, and BPMN process ID filters use scalar pointers; v810 uses generated filter unions. Assignment, candidate, and tenant equality are represented through the matching generated string-filter types. BPMN process ID maps to the backend `processDefinitionId` property.

**Alternatives considered**: Generated-code edits, client regeneration, legacy endpoint fallback, and one generated-type abstraction spanning all versions are unnecessary.

**Evidence**: `internal/clients/camunda/{v88,v89,v810}/camunda/client.gen.go` (`UserTaskResult`, `UserTaskFilter`, `UserTaskSearchQuery`, `SearchQueryPageRequest`, `SearchQueryPageResponse`); `internal/services/usertask/factory.go`.

## 3. State validation and public task shape

**Decision**: Expose only the version-neutral fields needed for task identity, assignment, candidates, process association, and tenant. Preserve domain `Key`, `ProcessInstanceKey`, and `TenantId` and add fields without changing legacy conversion semantics. Normalize optional name and assignee to empty strings, with intentional public omission tags. Copy candidate slices at the facade boundary.

**Rationale**: The nine checked-in states are identical across 8.8, 8.9, and 8.10: `ASSIGNING`, `CANCELED`, `CANCELING`, `COMPLETED`, `COMPLETING`, `CREATED`, `CREATING`, `FAILED`, and `UPDATING`. `assigned` is not a state. `all` is a CLI selector that becomes an absent state filter. A shared parser may serve the current set, with adapter contract tests pinning each version's membership. These are read states, not transitions this feature performs.

**Alternatives considered**: Exporting generated structs would expose version-specific additions and fields outside scope. Adding dates, forms, variables, custom headers, or v810 business IDs is unnecessary.

**Evidence**: `internal/domain/usertask.go`, `internal/services/usertask/*/convert.go`, generated `UserTaskStateEnum` definitions; public model and converter patterns in `c8volt/element`.

## 4. Shared service-owned traversal and bulk retrieval

**Decision**: Add service-owned free functions beside the existing resolver: strict bulk retrieval, collected search, page-visitor search, and exact count. Version adapters expose native GET and one normalized page only. The facade maps inputs, visitor values/actions, outputs, and errors mechanically. CLI visitors decide only whether to render or ask for continuation.

**Rationale**: `usertask/workflow.go` already uses a service-owned free function. This avoids repeating traversal across three adapters or placing backend loops in commands/facades. Bulk reads use `typex.Keys.Unique`, `toolx.DetermineNoOfWorkers`, and `toolx/pool.ExecuteSlice` to preserve order and options without custom concurrency.

**Alternatives considered**: A new workflow constructor/framework is unnecessary. Copying command-owned PI total traversal violates current layering rules. `common.RunBulk` is available but does not directly provide the same fail-fast and worker-limit option path as `pool.ExecuteSlice`.

**Evidence**: `internal/services/usertask/workflow.go`, `internal/services/processinstance/bulk.go`, `toolx/pool`, `internal/services/element/v88/service.go`, `specs/ralph-implementation-rules.md`.

## 5. Sparse pages, cursor progress, and exact counts

**Decision**: Normalize `TotalItems` (int64), `HasMoreTotalItems` (bool), and nullable start/end cursors in adapters. Prefer forward cursor requests after the first limit request. Use service-owned offset fallback when no cursor is available and metadata still indicates continuation. Empty or short pages do not terminate a traversal while continuation evidence remains. Repeated cursors, impossible metadata, or offset overflow fail instead of silently yielding incomplete success.

**Rationale**: An end cursor can be present on the final nonempty page. An exact total consistent with observed results can establish completion; otherwise a terminal probe may be necessary. A capped total is a lower bound, never a completion threshold or, by itself, proof that another page exists. An empty page without an advancing cursor after the known population lower bound has been traversed can establish terminal exhaustion; do not keep issuing offset requests solely because the global cap flag remains true. Before the known bound is traversed, empty pages with remaining population evidence require progress rather than completion. On an empty offset page with more results indicated, advance by requested page size; on a nonempty page advance by raw backend item count. Apply user-limit trimming separately from backend progress. Do not reuse element/PI empty-page short circuits blindly.

For exact counting, use a trustworthy exact backend total immediately. Otherwise count raw matching pages using int64 and continue until authoritative exhaustion. The count path retains only page data and traversal state, not an accumulated task collection. It has no interactive visitor and no result limit. Context cancellation or failure returns an error with no numeric result.

**Alternatives considered**: Stopping at `len(items)==0`, a short page, or the reported cap loses matches. Retaining every task to count wastes memory. A new snapshot or retry policy exceeds scope; existing HTTP transport, context, and error behavior remain authoritative.

**Evidence**: Generated `SearchQueryPageRequest` union (limit/offset/forward cursor), `SearchQueryPageResponse`; `internal/domain/element.go`; `internal/services/element/v88/service.go`; `cmd/get_processinstance_total.go` as behavioral reference only.

## 6. Key input and prompt compatibility

**Decision**: Use existing optional-dash argument validation, key parsing, error classification, and flag-before-stdin ordering. Add implicit piped-stdin reading only for the new command: read nonterminal stdin, or explicit `-`; never read terminal stdin merely to discover keys. Preserve explicit-dash input errors: terminal stdin or an empty explicit `-` stream is invalid. An empty implicit nonterminal stream contributes no keys, leaving flags or ordinary search to select the mode. Preserve malformed-input errors, blank-line skipping, and the existing 10 MiB scanner ceiling; validate flag keys as well as stdin keys against the existing 16-digit convention. Do not change `get pi` input behavior.

**Rationale**: Issue #308 explicitly makes `-` optional, but the current `readKeysIfDash` reads only when `-` is present. Reuse requires a narrowly scoped opt-in reader, not a claim that implicit reading already exists. Paging prompt eligibility follows existing terminal-stdin checks even with redirected stdout, with the existing auto-confirm, automation, limit, and machine-mode restrictions.

**Alternatives considered**: Requiring `-` misses the issue requirement. Reading terminal stdin would block normal search. Changing all get commands would enlarge scope.

**Evidence**: `cmd/cmd_stdin.go`, `cmd/cmd_cli.go`, `cmd/get_processinstance.go`, `cmd/get_element_search.go`, `cmd/cmd_stdin_error_envelope_test.go`.

## 7. Output and metadata

**Decision**: Use a stable `task.UserTasks` payload (`total`, `items`) for both single/multiple keyed CLI lookup and collected search. `total` is the number returned, not the backend matching total. Empty collections use `items: []`. JSON uses the existing successful envelope; errors use the shared error renderer. Human lists use flat rows and `found: N`; empty human search uses exactly `found: 0\n`. Select JSON before keys-only before human, then apply quiet-human suppression. Numeric total remains a requested result even with quiet.

**Rationale**: Existing `renderOutputLine` does not suppress human output just because quiet logging is set. The new view must enforce the spec's quiet-empty contract explicitly. JSON and keys-only cannot be suppressed or contaminated. Count mode follows `get pi` conflicts with JSON, keys-only, and limit. New metadata marks the command read-only with full shared-contract and automation support.

**Alternatives considered**: A raw generated payload, different schemas for key cardinality, human messages emitted inside discovery, and a new JSON count envelope would complicate automation or depart from the agreed spec.

**Evidence**: `cmd/command_contract.go`, `cmd/cmd_views_rendermode.go`, `cmd/cmd_views_element.go`, `cmd/get_processinstance_validation.go`, `cmd/root.go`.

## Research closure

All planning questions are resolved from the issue, specification, checked-in contracts, and local patterns. No new product decision or dependency is required. Implementation must prove the behaviors with tests; these findings do not claim that the feature already exists.
