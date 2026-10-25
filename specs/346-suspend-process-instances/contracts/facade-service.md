# Facade and Service Contract: Process-Instance Suspension

**Feature**: [../spec.md](../spec.md) | **Models**: [../data-model.md](../data-model.md)

## Public API

Add these methods to `c8volt/process/api.go`, using existing public option and key types:

```go
SuspendProcessInstance(ctx context.Context, key string, opts ...foptions.FacadeOption) (SuspensionResult, error)
SuspendProcessInstances(ctx context.Context, keys typex.Keys, wantedWorkers int, opts ...foptions.FacadeOption) (SuspensionResults, error)
```

Implement in focused `c8volt/process/suspend.go`. Map options, delegate, mechanically convert domain results, and convert errors with `ferrors.FromDomain`. Always return converted result evidence alongside a nonnil error. No pool, retry, preflight, polling, or hierarchy logic belongs here. Add models in `model.go` and mappings in `convert.go`; the composed top-level API continues to expose the process facade through its existing composition.

Options: honor `DryRun`, `NoWait`, `FailFast`, `NoWorkerLimit`, verbose/suppression, caller context, and existing timing/configuration. Discovery tenant filters do not restrict explicitly supplied keys; backend authorization remains authoritative. Force/root-expansion and no-state-check options do not alter suspension semantics and are not exposed as suspension command flags.

## Internal service surface

Add to `internal/services/processinstance/api.go` and each version's service contract:

```go
ValidateProcessInstanceSuspension() error
SuspendProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (domain.ProcessInstanceSuspensionResult, error)
```

Add the package-level collection function in `internal/services/processinstance/suspend.go`, following the existing service-level collection ownership:

```go
SuspendProcessInstances(ctx context.Context, api API, log *slog.Logger, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) (domain.ProcessInstanceSuspensionResults, error)
```

The facade's singular method delegates to the adapter's singular method. Its plural method delegates to this collection function. Keep all adapter contracts, compile-time assertions, and affected test doubles compiling together. A new independent service family is not needed.

### Capability and input ordering

1. Perform the no-I/O suspension capability guard before collection empty/preview returns and scheduling. Each singular entry point guards itself as well.
2. Normalize and validate the full key list before any mutation; deduplicate by first occurrence. Invalid collection input is not a partially executed operation.
3. Honor already-canceled contexts and validated positive timing configuration. Empty v810 collections return an initialized empty result without requests; unsupported versions still fail capability validation.
4. Dry-run returns preview items without reads, mutation, or waiting. No-wait must not change their preview status.
5. Actual collection execution initializes a keyed not-attempted item for every target, chooses workers using `toolx.DetermineNoOfWorkers`, and delegates with `pool.ExecuteSlice`. Merge completed results by index and preserve context errors even if no callback ran. Already-running work retains its actual evidence under fail-fast.

Apply call options at the owning layer and pass them to nested operations without dropping execution/suppression settings. Never use the cancellation expansion plan, root escalation, ancestry, descendants, or family methods.

## Version adapters

| Adapter | Required behavior |
| --- | --- |
| v88 | Guard and singular entry return wrapped `domain.ErrUnsupported`, including dry-run, with no generated-client calls. |
| v89 | Same unsupported contract; do not use the availability of a similarly named read state as capability proof. |
| v810 | Local guard succeeds; dry-run returns preview; actual execution submits exactly the selected key using the existing generated suspension client. |

In v810 `GenProcessInstanceClientCamunda`, add the existing generated signature for `SuspendProcessInstanceWithResponse`, including its generated request-body and optional request-editor types. Update `mockCamundaClient` and its compile-time assertion. Request body is an empty `SuspendProcessInstanceJSONRequestBody`; no new operation-reference flag or idempotency policy is introduced.

The actual wire request is `POST /process-instances/{processInstanceKey}/suspension`. Valid HTTP 204 without a call error establishes acceptance. Missing HTTP metadata, invalid response, another 2xx, or a call error accompanying a partial response does not. Convert rejected responses through existing HTTP/domain error mapping and retain per-key status/evidence.

## Submission and retry ownership

Keep submission mechanics in a focused v810 suspension file. Reuse `RetryCamundaMutation`; do not change its method/status/retry policy. The callback returns the original call error plus response/body for retry classification, while a separate evidence accumulator classifies the result and retains machine error detail. This prevents converted 429/503 errors from accidentally suppressing established retries.

Before each attempt, use existing request-rate attempt observation where needed to detect an unsent local wait, including when the HTTP timeout replaces its error. An unsent attempt must not overwrite evidence from earlier attempts. Prior uncertainty survives a later rejection; a later valid 204 can establish acceptance. Canceled backoff must return accumulated evidence, not the retry helper's zero-value return.

There is no post-failure state reconciliation. Successful no-wait ends after accepted submission. Default execution enters confirmation exactly once after acceptance. Confirmation never retries submission.

## Focused confirmation waiter

Add `waiter/suspension.go` with a narrow reader dependency based on existing `GetProcessInstance`. Return a suspension observation value and error, which the v810 adapter merges into its accepted submission result. A suitable observation value contains `HasObservation`, `LastObservedState`, `SuspendedDate`, and the confirmation disposition; it is internal and does not need a new public API.

Reuse the configured `BackoffConfig`, `NextDelay`, a context deadline bounded by caller deadline and configured timeout, and existing lookup-log suppression. Reject invalid timing before mutation through established configuration validation; no unbounded wait is allowed for public callers with incomplete config.

Follow the [state table](../data-model.md#confirmation-state-machine). Test exact matching identity; typed not-found must be checked after authorization/error classification, without raw-message heuristics. Preserve the last observation across failures. Do not change the existing generic state waiter or `State.IsTerminal`, since cancellation/deletion/expect rely on their current meanings.

Per-key polling observations and raw HTTP exchanges remain DEBUG diagnostics. Default/verbose human messages use only safe workflow facts, not wrapped read errors. Suppress nested terminal ERROR logs so the CLI owns the one aggregate failure diagnostic. A nil functional retry logger may suppress retry chatter; it must not disable the separately configured HTTP DEBUG diagnostic transport.

## Error compatibility

Preserve typed causes through wrapping/joining and facade conversion. Existing mappings apply: unsupported capability, invalid argument, not found, conflict, local precondition for authorization, timeout, unavailable, and generic/interruption behavior. Do not introduce suspension-specific numeric exits. Polling attempt exhaustion must carry an existing timeout cause; an observed incompatible terminal state carries an existing conflict cause, not a manufactured HTTP response code.

The collection returns partial reports with the original joined causes. The command uses existing aggregate classification precedence; no new precedence is introduced for mixed errors. Machine error evidence can preserve existing detail; human output must be derived separately from bounded classifications. `--no-err-codes` behavior remains the existing `ResolveExitCode` contract.

## Compatibility and test-double obligations

Public interface additions require external implementations of `process.API` to add the two methods; existing callers remain source-compatible. Internal API additions require all three adapters and concrete stubs to be updated in one compiling unit. Inspect concrete implementations in process/resource facade tests, process-instance dry-run tests, process-definition deletion tests, ops orphan-purge tests, and v810 generated-client contract tests. Embedded interfaces are not proof of behavioral coverage; add explicit suspension callbacks in new tests.

Existing cancellation, forced deletion, suspended-state/date conversion, `get pi`, and `expect pi` retain their contracts. There is no generated-client refresh, new dependency, global error schema change, or live validation requirement.
