# Research: Effective User-Task Variables

## Context and evidence

Research resolves the implementation questions in [spec.md](spec.md) against the checked-in repository. Two research agents independently examined the native endpoint and command/output integration. The initial constitution review passed: this is an additive read-only workflow using existing package ownership, output contracts, and dependencies. No external dependency or unresolved clarification remains.

## 1. Retrieve backend-selected effective variables

**Decision**: Add `SearchUserTaskEffectiveVariablesPage` to the usertask adapter API. Call the matching generated `SearchUserTaskEffectiveVariablesWithResponse` on v88, v89, and v810; v87 returns the established unsupported-operation error. A service-owned loop retrieves the complete collection, and a service enrichment function attaches it to selected tasks.

**Rationale**: The checked-in generated clients expose the operation on all three supported versions. Their effective-variable request/result shapes are identical: a task key, offset pagination, optional name sorting, and optional `truncateValues`. The backend owns scope resolution. The legacy task-to-process resolver and its Tasklist fallback must remain untouched.

**Alternatives considered**: Searching process variables and resolving scopes locally would change semantics; extending Tasklist fallback would expand scope; generated-client changes are unnecessary.

**Evidence**: `internal/clients/camunda/{v88,v89,v810}/camunda/client.gen.go`, `internal/services/usertask/{api.go,v88/contract.go,v89/contract.go,v810/contract.go,v87/service.go}`.

## 2. Offset pagination, complete reads, and effective-name integrity

**Decision**: Use an offset-only page request with `From` and `Size`, fixed service page size `consts.MaxPISearchSize` (1000), and ascending name sort. Do not expose variable paging controls or use task `--limit` as a variable limit. Normalize raw item count, exact/lower-bound total metadata, and presence of further continuation evidence at the adapter boundary. Advance by raw item count, or by requested size across an empty page when metadata still requires continuation.

**Rationale**: `UserTaskEffectiveVariableSearchQueryRequest.Page` accepts `OffsetPagination`; shared response cursor fields do not make cursor requests valid. `hasMoreTotalItems` marks a capped total, not a has-next-page flag. With exact totals, stop only after the raw observed count equals the total; below it, continue through sparse pages. With capped totals, preserve the highest reported lower bound across pages and continue beyond it until an empty page without further continuation evidence after that bound is satisfied. An empty page with a nonempty end cursor is not terminal; use that fact as evidence while still advancing by offset. Reject later exact totals that contradict the retained lower bound. Reject missing/invalid required page metadata rather than interpreting omitted fields as authoritative zero. Check cancellation, negative counts, inconsistent exact counts, and int32 offset overflow. Raw counts, not deduplicated counts, drive paging.

Sort the completed collection by name. Collapse identical repeated records for the same name; if repeated names carry different effective records, fail with a malformed-response error rather than invent a scope winner. This is defensive handling of an inconsistent response, not local scope resolution.

**Alternatives considered**: Stopping on any short/empty page loses later variables; using `hasMoreTotalItems` as a next-page flag truncates exact totals; directly reusing task traversal would incorrectly introduce cursor requests. Reuse its validation patterns without generalizing it into a new framework.

**Evidence**: generated `UserTaskEffectiveVariableSearchQueryRequest`, `OffsetPagination`, and `SearchQueryPageResponse`; `internal/services/usertask/search.go`; existing variable adapters under `internal/services/variable/`.

## 3. Preserve values and truncation explicitly

**Decision**: Always send `truncateValues=false`, independently of the human display limit. Validate the generated success payload with `common.RequirePayload`, then decode the raw body in the matching usertask adapter, following the existing variable adapter's raw DTO pattern. Preserve string values and recognize `isTruncated`, falling back to `truncated` only when the first field is absent. Keep any remaining truncation flag visible. Do not add a second per-variable GET recovery workflow.

**Rationale**: Generated `VariableSearchResult`/`VariableResultBase` omit value and truncation fields on all three versions. The ordinary variable adapters already preserve them from raw JSON. The effective endpoint directly supports disabling truncation. No existing `GetVariableWithResponse` recovery pattern provides a demonstrated benefit, and its generated result has the same omissions. Malformed raw JSON or invalid required fields must produce errors rather than empty values or variables; an explicitly empty value remains valid.

**Alternatives considered**: Using generated fields alone loses values; sending the CLI character limit upstream changes JSON; silently treating a shortened value as complete breaks the spec; broad generated-model or decoder migrations add unnecessary risk.

**Evidence**: `internal/services/variable/{v88,v89,v810}/convert.go`; generated effective-variable request params and variable result types.

## 4. Reuse variable records without process-scope filtering

**Decision**: Reuse `domain.ProcessInstanceVariable` internally and expose `task.UserTaskVariable` as an alias of `process.ProcessInstanceVariable`. Add task-specific enriched item and collection types with int64 totals. Use initialized empty slices so successful empty task results have `items: []` and a task without variables has `variables: []`.

**Rationale**: The existing variable records already carry name, serialized value, variable identity, owning process, actual scope, tenant, and truncation. Reusing the record does not impose process-root scope semantics. The process facade does not import the task facade, so this alias introduces no cycle. Task-specific wrappers keep task metadata separate and avoid copying process age metadata.

**Alternatives considered**: A duplicate variable schema invites drift; moving all public variable ownership is unnecessary. Reusing `variablesForProcessInstance` is incorrect because it filters `ScopeKey == ProcessInstanceKey` and would drop task-visible local values.

**Evidence**: `internal/domain/processinstance.go`; `c8volt/process/model.go`; `internal/services/processinstance/enrichment.go`; `c8volt/process/convert.go`; `toolx.MapSlice` semantics.

## 5. Enrich only selected results, using one facade addition

**Decision**: Add only `EnrichUserTasksWithVariables(ctx, UserTasks, ...FacadeOption)` to the public task API. Keyed CLI reads retain strict `GetUserTasks` first, then enrich. Incremental search enriches already-trimmed `step.Page.Items` before rendering and prompting; collected modes enrich the final selected collection once. Empty pages skip enrichment. Per-task requests and accumulation stay in internal services, using sequential enrichment like existing process-variable enrichment.

**Rationale**: `SearchUserTasksPages` already trims to the remaining limit before invoking its visitor. The command only chooses rendering and interaction; it neither advances backend pages nor performs per-task request loops. Collected JSON needs no extra page result type or command-side cache. Sequential enrichment preserves order and avoids a new concurrency policy; existing keyed task reads still honor their worker options.

**Alternatives considered**: Enriching raw search pages overfetches excluded tasks; always enriching both page and final result duplicates requests; new enriched-search APIs or a generic enrichment framework add unnecessary surface area; a new variable worker pool is not required by this feature.

**Evidence**: `cmd/get_usertask.go`, `cmd/get_usertask_search.go`, `internal/services/usertask/search.go`, `internal/services/processinstance/enrichment.go`, `c8volt/task/client.go`.

## 6. Output modes and formatter reuse

**Decision**: Gate enrichment on `--with-vars`, non-count execution, and effective output mode not keys-only. JSON wins over keys-only. Quiet human mode retains requested retrieval/error semantics while suppressing rendering; quiet JSON still enriches. Reject negative value limits and explicit limits (including zero) without `--with-vars` before requests. Keep enrichment flags out of keyed/search-selector conflict detection.

Use a narrow shared name/value formatter with explicit value-limit input, preserving the process wrapper and its tests. Reuse existing tree branch helpers for a `vars:` section beneath each task row, omitting the section when variables are empty. Preserve structured-value compaction, rune-based limits, and existing backend/client truncation labels. Task views must propagate writer errors.

**Rationale**: Current process-variable output passes through the activity tree renderer, so its established visible convention is a `vars:` subtree. `processInstanceVariableHumanLine` currently reads a process-command global; using it directly would couple task flags to unrelated state. Existing task summary, empty output, and prompt contracts are explicit and already tested.

**Alternatives considered**: A new flat layout would diverge from current output; changing PI globals from the task command is unsafe; raw keys-only flags must not override JSON precedence; quiet is not permission to hide retrieval failures.

**Evidence**: `cmd/cmd_views_processinstance_vars.go`, `cmd/cmd_views_processinstance_activity.go`, `cmd/cmd_views_usertask.go`, `cmd/get_usertask_{output,terminal,error}_test.go`.

## 7. Validation and documentation

**Decision**: Plan targeted adapter, service, facade, command execution, real-terminal, and formatter regression checks. Update help (which currently excludes variables), capability metadata tests, all four issue examples, README, and generated CLI docs. Run full race validation for the integrated change because it extends shared service/facade interfaces and a formatter used by both task and process commands; do not run runtime tests for planning-only edits.

**Rationale**: Stub changes alone cannot prove paging or output purity. Terminal tests must capture real stdin plus separate stdout/stderr, including inherited/configured error writers. Request assertions prove absence of overfetch and mutation calls.

**Alternatives considered**: View-only tests miss execution gates; pipe-only prompt tests miss terminal eligibility; blanket tests on each documentation edit contradict constitution v2.0.0.

**Evidence**: `cmd/get_usertask_terminal_test.go`, `testx.NewCmdTerminalRunner`, `cmd/command_contract_test.go`, `Makefile`, `.specify/memory/constitution.md`.
