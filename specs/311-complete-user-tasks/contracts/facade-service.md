# Facade and Service Contract: User-Task Completion

**Issue**: #311 | **Models**: [data-model.md](../data-model.md)

## Public facade

Add to `c8volt/task.API`:

```go
CompleteUserTasks(ctx context.Context, keys typex.Keys, variables map[string]any,
    wantedWorkers int, opts ...foptions.FacadeOption) (UserTaskCompletionResults, error)
PrepareUserTaskCompletions(ctx context.Context, keys typex.Keys, variables map[string]any,
    wantedWorkers int, opts ...foptions.FacadeOption) (UserTaskCompletionPlan, error)
ExecuteUserTaskCompletions(ctx context.Context, plan UserTaskCompletionPlan) (UserTaskCompletionResults, error)
```

The collection methods handle one or many keys. Prepare returns an opaque in-memory plan with advisory ready/completed/unassigned/unchecked counts for CLI approval. Execute consumes its private frozen inputs and observations without another lookup. Complete composes the same preparation and execution for direct callers. The plan is not a machine-output payload. Public result models and enums are defined in the data model; public signatures expose no generated or internal types. Existing `New`, `NewWithVariableUpdates`, and root client composition remain usable without another constructor.

The facade maps options, delegates caller-owned inputs to the area coordinator, maps all returned items even when an error accompanies them, and converts the aggregate through `ferrors.FromDomain`. The coordinator snapshots keys and nested variables before scheduling, so the facade adds no redundant copy. Native adapters retain their own payload copy for direct callers and generated-client isolation. Do not discard partial evidence or add pools, retries, task reads, or verification in the facade.

## Shared coordinator

Add in `internal/services/usertask/completion.go`:

```go
func CompleteUserTasks(ctx context.Context, api API, keys typex.Keys,
    variables map[string]any, wantedWorkers int, opts ...services.CallOption) (
    domain.UserTaskCompletionResults, error)
```

The coordinator owns complete validation, stable uniqueness, JSON snapshotting, worker selection, preparation, scheduling/fail-fast, ordered outcomes and error aggregation. Preparation performs one native GET per unique key using a request-scoped `httpc.WithoutReadRetry` context; ordinary reads retain their existing retry policy. Matching COMPLETED observations become failed/not_submitted conflicts with no statusCode. Eligible unassigned observations and failed/malformed/mismatched reads become advisory counts; failed reads do not block a backend completion attempt. Context cancellation aborts preparation. Any preflight conflict with fail-fast prevents every submission; otherwise skip those items and submit the remaining keys without rereading. Empty selection and no-wait are errors. Negative wantedWorkers is invalid; zero requests existing automatic selection. DryRun returns preview/not_submitted items and performs no native calls.

Apply options at their owning layer and propagate relevant options to each native operation. Pass existing explicit-key options; never translate configured tenant into a task discovery filter. No variable service dependency or current-variable reads.

Use `toolx.DetermineNoOfWorkers` and `pool.ExecuteSlice`. Prepopulate all unique keys, retain per-index started/outcome/error state, and stop new submissions after observed failure when requested. Use the pool's cancellation only for scheduling; started adapter calls receive the original caller context. Aggregate real callback errors and original caller cancellation rather than returning fail-fast's synthetic context error. Worker one preserves input-order execution. Parallel results remain ordered by selection.

## Native version contracts

Add to `internal/services/usertask.API` and `v88`, `v89`, `v810` contracts:

```go
CompleteUserTask(ctx context.Context, key string, variables map[string]any,
    opts ...services.CallOption) (domain.UserTaskCompletionResult, error)
```

Extend each `GenUserTaskClientCamunda` contract with its generated client's `CompleteUserTaskWithResponse` signature. Update all affected stubs/mocks and interface assertions in the same compilable unit. Add version-specific `completion.go` files; do not hand-edit generated clients.

The native operation is `POST /user-tasks/{userTaskKey}/completion` under the existing configured v2 API base. Body mapping is:

| Input | Generated mapping | Wire meaning |
|---|---|---|
| Omitted variables | `Action=nil`, `Variables=nil` | `action:null`, `variables:null`; nullable fields accepted by vendored contract |
| Empty object | `Action=nil`, pointer to empty map | `variables:{}`; still a completion |
| Supplied object | `Action=nil`, pointer to copied object | Identical typed values for each key |

Documented HTTP 204 confirms task completion, including blocking task listeners; it has no success JSON payload. Retain accepted/accepted in the existing machine model and add no read-back or polling. Validate nil/malformed responses without panicking; do not require JSON content with `common.RequirePayload`. Use existing HTTP/domain classification for errors, preserving status/evidence. Generated-client mocks must populate request URL/method as required by `httpc.HttpStatusErr`.

No search, GetUserTask, effective-variable retrieval, scope-variable mutation, assignment, legacy fallback, polling, or completion-state reconciliation is permitted in this operation or its dry-run path.

## Retry and error rules

Use `services.RetryCamundaMutation` with its existing finite policy: 429, 503 RESOURCE_EXHAUSTED, and temporary/timeout network errors. Use existing delay/Retry-After behavior. Do not add retries for 404/409/500/504, implement a custom scheduler, or change shared retry behavior. Multiple transport attempts remain one logical operation.

Track whether a native attempt began, the last response/status, and earlier ambiguous attempts in the closure. Check caller cancellation before submission. Do not lose evidence when retry-backoff cancellation returns a zero response. Last status alone cannot prove that an earlier attempt was rejected.

| Observed fact | Execution status / submission evidence |
|---|---|
| Preflight observes completed | failed / not_submitted; conflict error, no submission statusCode |
| 204 success | accepted / accepted |
| Definitive 4xx rejection, including exhausted 429, without earlier ambiguity | failed / rejected |
| Exhausted explicit resource-exhaustion rejection without earlier ambiguity | failed / rejected |
| 504 or other inconclusive server outcome | failed / unknown, or timed_out when existing timeout classification applies |
| Transport timeout after attempt | timed_out / unknown |
| Caller cancellation after attempt | canceled / unknown unless available evidence establishes rejection; never infer rejection from cancellation alone |
| Earlier ambiguous attempt followed by non-success | final failure status / unknown |
| No native attempt due to stopped scheduling/caller cancellation | not_attempted / not_submitted |
| Nil/malformed or unexpected success response without documented acceptance | failed / unknown |

A failed operation must return a nonnil classified error even when machine evidence is available. Preserve accepted results when concurrent work fails or caller context expires. Cancellation during backoff after only explicit rejections can retain rejected submission evidence with canceled/timed_out execution; record actual evidence rather than mechanically marking all cancellation unknown.

## Diagnostics and compatibility

Use a constant retry operation label and only the configured logger for bounded verbose retry progress; pass nil for compact retry progress. Actual HTTP diagnostics stay in the existing transport and honor DEBUG/redaction independently of verbose. No command-specific HTTP instrumentation or raw-error normal logs.

Existing variable methods, reads, resolution, public variable schemas, factories, unsupported-version validation and default version remain unchanged. Native and aggregate service interface additions require corresponding test-double updates wherever those interfaces are implemented; no retired-version suite is introduced.

## Contract verification

- Every supported adapter: generated method/body, nil/empty/nested payloads, no-content success, response/body/error delegation, payload ownership, logging-option forwarding, and backend-authorized foreign-tenant keys. Shared response classification, retry, cancellation and ambiguous-outcome cases live once in `internal/services/common/usertask_completion_test.go`.
- Coordinator: all input validation before requests, nil API guard, no-wait, dry-run, stable deduplication, nested-copy ownership, selected worker bounds and precedence, out-of-order completion, fail-fast in-flight survival, ordered unattempted slots and original errors.
- Facade: worker/options forwarding, defensive copying, public enum/JSON mapping, error class and partial-result preservation.
- All dependencies are isolated mocks/fake servers. No live service is needed to satisfy these checks.
