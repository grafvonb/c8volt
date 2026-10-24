# Facade and Service Contract: User-Task Completion

**Issue**: #311 | **Models**: [data-model.md](../data-model.md)

## Public facade

Add to `c8volt/task.API`:

```go
CompleteUserTasks(ctx context.Context, keys typex.Keys, variables map[string]any,
    wantedWorkers int, opts ...foptions.FacadeOption) (UserTaskCompletionResults, error)
```

This collection method handles one or many keys. Public result models and enums are defined in the data model; public signatures expose no generated or internal types. Existing `New`, `NewWithVariableUpdates`, and root client composition remain usable without another constructor.

The facade only copies/maps values and options, delegates to the area coordinator, maps all returned items even when an error accompanies them, and converts the aggregate through `ferrors.FromDomain`. Do not discard partial evidence or add pools, retries, task reads, or verification in the facade.

## Shared coordinator

Add in `internal/services/usertask/completion.go`:

```go
func CompleteUserTasks(ctx context.Context, api API, keys typex.Keys,
    variables map[string]any, wantedWorkers int, opts ...services.CallOption) (
    domain.UserTaskCompletionResults, error)
```

The coordinator owns complete validation, stable uniqueness, JSON snapshotting, worker selection, scheduling/fail-fast, ordered outcomes and error aggregation. Empty selection and no-wait are errors. Negative wantedWorkers is invalid; zero requests existing automatic selection. DryRun returns preview/not_submitted items and performs no native calls.

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

Documented HTTP 204 means acceptance; it has no success JSON payload. Validate nil/malformed responses without panicking; do not require JSON content with `common.RequirePayload`. Use existing HTTP/domain classification for errors, preserving status/evidence. Generated-client mocks must populate request URL/method as required by `httpc.HttpStatusErr`.

No search, GetUserTask, effective-variable retrieval, scope-variable mutation, assignment, legacy fallback, polling, or completion-state reconciliation is permitted in this operation or its dry-run path.

## Retry and error rules

Use `services.RetryCamundaMutation` with its existing finite policy: 429, 503 RESOURCE_EXHAUSTED, and temporary/timeout network errors. Use existing delay/Retry-After behavior. Do not add retries for 404/409/500/504, implement a custom scheduler, or change shared retry behavior. Multiple transport attempts remain one logical operation.

Track whether a native attempt began, the last response/status, and earlier ambiguous attempts in the closure. Check caller cancellation before submission. Do not lose evidence when retry-backoff cancellation returns a zero response. Last status alone cannot prove that an earlier attempt was rejected.

| Observed fact | Execution status / submission evidence |
|---|---|
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

- Every supported adapter: generated method/body, nil/empty/nested payloads, no-content success, rejected/uncertain/nil responses, cancellation and retry evidence, backend-authorized foreign-tenant keys.
- Coordinator: all input validation before requests, nil API guard, no-wait, dry-run, stable deduplication, nested-copy ownership, selected worker bounds and precedence, out-of-order completion, fail-fast in-flight survival, ordered unattempted slots and original errors.
- Facade: worker/options forwarding, defensive copying, public enum/JSON mapping, error class and partial-result preservation.
- All dependencies are isolated mocks/fake servers. No live service is needed to satisfy these checks.
