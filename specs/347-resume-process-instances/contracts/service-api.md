# Service and Public-Library Contract

## Public facade

Add to `c8volt/process.API`:

```go
ResumeProcessInstance(ctx context.Context, key string, opts ...foptions.FacadeOption) (ResumptionResult, error)
ResumeProcessInstances(ctx context.Context, keys typex.Keys, wantedWorkers int, opts ...foptions.FacadeOption) (ResumptionResults, error)
PlanProcessInstanceResumption(ctx context.Context, keys typex.Keys, opts ...foptions.FacadeOption) (ProcessInstanceResumptionPlan, error)
```

The singular method passes a singleton to the internal collection executor and mechanically extracts its result. It must preserve zero/partial result evidence when errors occur. The collection facade copies inputs, maps public options, delegates, converts models, and applies `ferrors.FromDomain`. The planner facade does the same for planning. No worker pool, discovery loop, waiter, retry, state classification, or logging orchestration belongs in the facade.

Use existing public options for dry-run, no-wait, fail-fast, worker-limit override, tenant behavior, and suppression. No separate tree executor or generic mutation API is introduced. A library caller plans, reviews `TargetKeys`, and calls the ordinary collection with those frozen keys; CLI interaction is not required for library use.

## Internal interfaces and ownership

Extend version-neutral `internal/services/processinstance.API` and all relevant versioned contracts with:

```go
ValidateProcessInstanceResumption() error
ResumeProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (domain.ProcessInstanceResumptionResult, error)
```

Area-level orchestration functions:

```go
ResumeProcessInstances(ctx context.Context, api API, log *slog.Logger, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) (domain.ProcessInstanceResumptionResults, error)
PlanProcessInstanceResumption(ctx context.Context, api API, keys typex.Keys, opts ...services.CallOption) (domain.ProcessInstanceResumptionPlan, error)
```

Match existing orchestration conventions; the logger parameter does not authorize raw joined-error emission. All interfaces, concrete implementations, and impacted stubs must compile together. Inventory process/resource facade stubs, process-definition deletion stubs, ops orphan-purge stubs, dry-run fixtures, and versioned tests when expanding interfaces.

### Capability and local validation

- V810 guard succeeds; v88/v89 return the established unsupported-capability error without I/O, even dry-run and tree planning.
- Normalize all keys before dispatch: trimmed, exactly 16 decimal digits, stable identity deduplication. Reuse a narrowly named shared validator rather than duplicate suspension's validation.
- Command missing/empty selection is an input error. Internal collection empty input may return initialized empty items without dispatch, consistent with the existing collection convention; the tree planner requires at least one seed. Public singular empty key is invalid.
- Validate timing locally before any target is mutated. No-wait retains existing timing exemptions; dry-run remains non-mutating and version guarded.
- Caller cancellation before work preserves unattempted evidence; malformed discovered keys/observations fail the whole plan.

### Adapter submission

Use `GenProcessInstanceClientCamunda.ResumeProcessInstanceWithResponse`, the generated v810 typed empty request body, and `POST /process-instances/{processInstanceKey}/resumption`. No operation reference flag is added. Only documented valid empty 204 establishes acceptance; nil/malformed/other success responses retain failure/uncertainty under established classification.

Use existing mutation retry/backoff, request-rate attempt tracking, context, HTTP error normalization, and redacted DEBUG diagnostics. Preserve cumulative unknown submission after ambiguity even if a subsequent attempt is explicitly rejected. Store last valid submission status independently of confirmation read statuses. Do not add endpoint-specific retry bypasses or alter global HTTP policy.

### Collection

Preinitialize one result per normalized key, validate local preview/timing before dispatch, then use existing worker-count policy and `pool.ExecuteSlice`. Keep results in input order. Fail-fast stops new scheduling through existing pool semantics; running targets retain their outcomes. Return partial results and joined/domain errors without logging raw causes. Singleton and tree execution enter this same path.

### Planner

Guard capability first; validate seed input and local timing; call existing `DescendantsResult` with explicit-key ignore-tenant and detail-suppression conventions. Discovery uses backend authorization, not an invented tenant-filtered selection. Validate complete descendant result, seed/root identity, key syntax, seed inclusion, missing ancestry, and matching observations. Reject explicit partial/unresolved output before mutation; do not infer a complete scope from absent reports.

Traverse all states, deduplicate overlaps in first-seen order, classify only SUSPENDED as eligible, and return a complete plan. Return zero plan plus error on any failure. Respect the existing 1,000-child-per-parent lookup bound; no pagination or repeated discovery. Planning never submits a resumption request.

### Waiter

A focused resumption reader exposes only `GetProcessInstance`; the waiter cannot submit mutations. Reuse configured backoff bounds, context, nested read-log suppression, attempt limits, and identity validation. ACTIVE/COMPLETED confirms only after accepted submission; CANCELED/TERMINATED stops with conflict. Other states and failure handling follow [data-model.md](../data-model.md). Fresh observations replace dates; no-wait performs no confirmation reads.

## Error and compatibility guarantees

- Capability, invalid-input, authorization, not-found, conflict, timeout, interruption, and transport failures retain existing domain-to-public classes and exit mappings; no ad hoc exit codes.
- Accepted but unconfirmed results survive facade conversion and final failure.
- Machine error evidence is preserved. Human summaries use allowlisted facts and bounded error categories; raw `Error` fields are not human messages.
- Existing suspension, deletion/cancellation, traversal, and inspection signatures/semantics remain unchanged, except adding resumption methods to interfaces and mechanical shared-validator reuse.
- Adding interface methods requires downstream implementers/test doubles to add methods; this is an additive concrete-client capability but an interface source-compatibility consideration. Document the new API and update repository stubs in the same work unit.
