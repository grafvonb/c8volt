# Research: Update User-Task Variables

## 1. Reuse effective reads and scope identity

**Decision**: Use `GetNativeUserTask` and `SearchUserTaskEffectiveVariables` for planning. Preserve backend-selected names, `ScopeKey`, tenant evidence, and `APITruncated`; use `ElementInstanceKey` only for absent names.

**Rationale**: `internal/services/usertask/variables.go` already traverses sparse/capped pages, validates metadata and normalizes effective names. Native keyed reads avoid discovery-tenant filtering; the legacy `GetUserTask` resolver has different tenant semantics and is not the correct read path.

**Alternatives considered**: Resolving a task to its PI would lose task-local and intermediate scopes. Reimplementing effective selection or traversal would duplicate established behavior.

## 2. Minimal variable-write extension

**Decision**: Extend `internal/services/variable.API` with `UpdateScopeVariables(ctx, scopeKey, variables, opts...)`, returning a scope-neutral acceptance response. Each supported adapter calls its existing generated `CreateElementInstanceVariablesWithResponse` with explicit `local=true`. This primitive submits only; it performs no PI lookup or PI confirmation.

**Rationale**: Existing adapter contracts in v88/v89/v810 already expose this generated method. `internal/services/variable/v810/variables.go` currently omits `Local` and invokes a PI-specific waiter; passing task scope keys to that method would reuse incorrect semantics. `SetVariableRequest` in each supported generated client supports local scope writes. v87 remains explicitly unsupported for this feature.

**Alternatives considered**: User-task attribute PATCH does not update variables. Copying generated transport behavior into the task command or facade breaks ownership. Changing the existing PI method globally adds unnecessary regression risk.

## 3. Workflow ownership and dependency wiring

**Decision**: Keep a focused composed update service in `internal/services/usertask`, owning existing task reads, the variable writer, configured backoff, and logging. Expose planning and execution through an update-specific contract in that same area. Wire it once from `c8volt/client.go`, with `task.NewWithVariableUpdates` preserving the existing `task.New` signature for read-only consumers.

**Rationale**: The existing usertask `API` is a low-level version adapter interface with compile-time assertions against all version packages. Polling and cross-resource planning belong above those adapters but below the facade. Nearby `c8volt/process/client.go` already uses compatible `NewWith...` construction. Legacy construction must return a normal precondition error for unconfigured update methods, never panic or silently succeed.

**Alternatives considered**: Putting worker/poll loops in the facade violates current rules. Adding workflow methods to every read adapter duplicates orchestration. A new top-level service area or general workflow engine is unnecessary.

## 4. Plan and execute fixed scope targets

**Decision**: Complete planning for every selected task before allowing any mutation. Store per-task plans and a stable list of unique scope targets, each containing sorted variable names and only planned changes. Order scopes by first encounter in first-input task order. Merge requests for the same scope/name and group different names in one scope payload. Retain all task associations.

**Rationale**: A single input payload gives shared targets the same requested value. This supports one logical write per target and consistent confirmation counts. Planning errors prevent all writes. Execution uses that plan without rediscovery or rescoping; concurrent changes remain possible, as with PI updates.

**Alternatives considered**: Calling a single-task updater once per task duplicates inherited writes. Executing original keys after preview would discard frozen target decisions. Atomic rollback is outside scope.

**Integrity rules**: Reject conflicting tenant/identity evidence or conflicting shared requested values before writes; do not choose an arbitrary target. Truncated current values count as changes with explicitly incomplete before-values, never equality; confirmation requires a complete matching value at the planned scope. Same name at different scopes is valid and remains separate.

## 5. Confirmation and partial outcomes

**Decision**: Submit unique scopes with existing `pool.ExecuteSlice`, worker policy, cancellation and `RetryCamundaMutation`. Then confirm tasks whose required writes were accepted through complete effective-variable reads, requiring each requested name, intended scope, and complete value to match. Use a focused task waiter with the existing PI backoff timing/timeout/retry conventions. No-wait skips confirmation only.

**Rationale**: The existing variable waiter filters `ProcessInstanceKey == key && ScopeKey == key`; it cannot confirm inherited and element-local UT variables. The general `toolx/poller` also has separate defaults and process-stderr output. Neither should be reused blindly. Keep task-specific comparison in the owning service; extract a shared timing helper only if two concrete consumers benefit without changing PI behavior.

**Partial handling**: A scope acceptance/failure is propagated to every associated task. Within a task, expose each target outcome so one accepted write and one failed write cannot become a misleading all-or-nothing result. Fail-fast stops new scope submissions; retain accepted writes and all unfinished target facts. Confirmation errors cannot undo acceptance. Return results plus the normalized error and render them once.

## 6. CLI and output reuse

**Decision**: Reuse PI keys/payload/flag conventions and shared envelope helpers. Explicitly dispatch dry-run, no-op, accepted, confirmed, and failed outputs from service facts. Extract shared payload parsing only where both PI and UT call it. Keep plan rendering in the UT view and human summaries compact, with full details under verbose.

**Rationale**: `cmd/update_processinstance_variables.go` provides the payload baseline but its backend planning belongs in services under current rules. `renderCommandResult` infers accepted status from `--no-wait`; no-op/dry-run must instead explicitly render success. `handleCommandError` exits after emitting an envelope; do not call it after rendering a partial-result envelope.

**Alternatives considered**: Copying the PI human-only no-op branch breaks the new command's machine contract. Adding scope controls, search flags, or a shared schema revision would expand the requested feature.

## 7. Verification and documentation

**Decision**: Planning receives document/link/diff checks only. Implementation starts with affected service/facade/command tests and adds real-terminal stream tests. Finish with `make docs-content` and issue-required `make test` because the feature changes shared service interfaces, facade construction, and concurrent execution.

**Rationale**: This follows constitution v2.0.0 and issue #326. Generated clients need no edits or refresh for the observed supported request shape.

**Open questions**: None. Research uses checked-in code and generated contracts; live backend behavior is to be verified by the implementation validation guide, not claimed here.
