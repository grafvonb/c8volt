# Facade and Service Contract

## Public facade

Add to `c8volt/task.API`:

```go
PlanUserTaskVariableUpdates(ctx context.Context, keys typex.Keys, variables map[string]any, opts ...foptions.FacadeOption) (UserTaskVariableUpdatePlan, error)
ExecuteUserTaskVariableUpdates(ctx context.Context, plan UserTaskVariableUpdatePlan, wantedWorkers int, opts ...foptions.FacadeOption) (UserTaskVariableUpdateResults, error)
```

Names describe the existing two-stage confirmation workflow; there is no persisted/exported plan command. The CLI plans, renders/prompts, then passes the same in-memory plan for execution. Facades perform only model/option conversion, delegation, output mapping, and `ferrors.FromDomain`. Preserve partial results on errors. Expose no generated client types or domain sentinels in these operation signatures.

Use `task.NewWithVariableUpdates(pdAPI, piAPI, utAPI, updateAPI, log)` alongside the unchanged `task.New`. The old constructor keeps read behavior; attempts to call new operations without configured update dependencies return a normalized local precondition error. Root facade construction provides the composed update service. Update public-interface test doubles.

## Internal usertask workflow

Define a narrow `VariableUpdateAPI` in `internal/services/usertask` with domain equivalents of the two operations. `NewVariableUpdates(utAPI, variableAPI, cfg, log)` returns the composed implementation. Existing native adapter API/factory read semantics are unchanged.

Planning owns stable key deduplication, native task lookup, all variable pages, scope resolution, comparison, sorted categories, tenant evidence and unique scope payloads. Complete all planning before returning an executable result. Reject malformed identity and inconsistent shared targets. Empty requests through the service return a no-work plan; CLI policy separately rejects missing keys.

Execution validates plan structure and consistency before requests, respects dry-run even for library callers, and never recomputes scope selection. Determine worker count with existing policy and submit unique scope payloads through `pool.ExecuteSlice`; propagate fail-fast and context cancellation. Preserve all started results and identify unstarted targets. Confirmation occurs after submission for fully accepted tasks, with existing configured bounds and cancellation; if fail-fast stops further confirmation after an error, retain `submitted` status with `confirmationStatus=skipped` for remaining accepted tasks and return the aggregate failure; never count them as confirmed.

Use the existing complete effective-variable reader for each confirmation attempt. Match all requested names against planned scopes and complete normalized values. No-wait makes zero confirmation reads. A fully unchanged run performs zero writes and zero confirmation polling. Honor existing read/mutation retry policies; no custom generic retries or recovery mutations.

## Internal variable writer

Extend `internal/services/variable.API`:

```go
UpdateScopeVariables(ctx context.Context, scopeKey string, variables map[string]any, opts ...services.CallOption) (domain.ScopeVariableUpdateResponse, error)
```

The response contains scope key, acceptance/status facts, and transport detail; it has no PI identity or confirmation claim. Extend matching adapter interfaces/assertions and mocks where necessary.

- v8.10, v8.9, v8.8: validate target and payload, use generated `CreateElementInstanceVariablesWithResponse`, explicitly set `Local=true`, and use the existing mutation retry and HTTP error normalization. Expect the documented 204 success. Handle nil/malformed responses without panic. This operation never calls a PI waiter or PI variable search.
- v8.7: explicit unsupported domain error, zero transport calls.
- Unknown configured version: existing factory error.
- Existing `UpdateProcessInstanceVariables` behavior is preserved, including its current target and wait behavior.

## Scope and error invariants

- Get native tasks, not the legacy resolver; explicit task keys are backend-authorized without discovery-tenant filtering.
- Never pass a user-task key as an element-instance key.
- Existing variable scope wins; only absent variables use task `ElementInstanceKey`.
- Truncated old values are planned changes with incomplete before-values; incomplete reads are failures, not absence.
- Shared scope writes have one outcome propagated to every dependent task. Partial errors do not discard accepted work.
- Service output contains enough facts for views; rendering adds zero remote calls.
- Invalid supplied plans fail before mutation. This validation checks internal consistency, not a new concurrency or authorization guarantee; the backend remains authoritative.

See [data-model.md](../data-model.md) for stable public fields and states.
