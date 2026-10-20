# Research: Bulk Job Completion

**Date**: 2026-10-20 | **Baseline**: v4.3.4 | **Issue**: #345

Two read-only research agents examined discovery/tenancy and completion/reporting. Findings below resolve the design questions for [plan.md](plan.md); they are not a claim that implementation or live recovery has been validated.

## 1. Tenant-safe shared discovery

**Decision**: Add configured tenant equality to `newJobSearchFilter` in v88, v89 and v810. Preserve key-only lookup behavior. Defensively verify tenant in bulk preparation and rechecks.

**Evidence**: Each `internal/services/job/v*/convert.go` omits tenant. All three generated `JobFilter` models already contain `TenantId *StringFilterProperty`. `common.EffectiveTenant`, `config/app.go` and root flags already distinguish named discovery tenant from empty/all-accessible discovery and from creation's target tenant.

**Rationale**: Fixing the shared builder also corrects `get job`, page and total searches. Explicit keys remain governed by backend authorization rather than being silently filtered by a discovery setting.

**Alternatives considered**: Client-only filtering would leave backend totals misleading and shared search unfixed. Generated-client regeneration is unnecessary. Adding a second tenant interpretation to command code would diverge from root configuration.

## 2. Complete preparation before mutation

**Decision**: Own bulk traversal in one internal coordinator using normalized pages, stable job-key sorting, first-seen deduplication and explicit preparation completion reasons.

**Evidence**: Existing `SearchJobsPages` counts raw items against limits and returns no proof distinguishing visitor stop from complete traversal. `pickJobSearchOverflowState` returns `NoMore` for an empty page before considering contradictory metadata. Tests currently encode empty termination despite lower-bound totals. Generated job search supports sorting by job key.

**Rationale**: Completing jobs while paging by offset can remove earlier records and skip later ones. Full preparation prevents this command's own mutations from shifting its selection. It cannot stop external workers from changing the index.

**Alternatives considered**: Streaming mutation is unsafe for selection completeness. Reusing the collected-search result without strengthening completion evidence is insufficient. Treating every empty page as exhaustion is false; treating every lower-bound total as proof of unlimited continuation can loop indefinitely.

**Boundary**: Preserve raw offsets before local filtering and overfull-response trimming behavior. Positive unique-target limits terminate intentionally. Ambiguous metadata gets bounded attempts to establish progress/exhaustion, then incomplete error and zero writes. Existing paging tests must deliberately change where they encode false completeness.

## 3. Coordinator ownership and imports

**Decision**: Use `internal/services/job/completion` for preparation/execution and small injected read/submission functions. Version services implement additive API methods by delegating to it.

**Evidence**: `internal/services/job/factory.go` imports every version implementation; a version importing its parent package would create a cycle. Facade conversion is already thin in `c8volt/job/client.go`.

**Rationale**: A sibling coordinator can import domain, call options and shared helpers while remaining independent of adapters. All three versions share one algorithm.

**Alternatives considered**: CLI or facade orchestration violates layering. A new cross-resource ops framework broadens scope. Copying the engine into three adapters multiplies concurrency and paging risk.

## 4. Submission evidence and reconciliation

**Decision**: Bulk uses one submission attempt, then bounded reads for ambiguous/rejected-race outcomes. Preserve existing single-key retry behavior.

**Evidence**: Adapters wrap completion in `services.RetryCamundaMutation`. `internal/services/retry.go` can retry temporary network/timeouts, 429 and resource-exhausted 503, returning only the final error/response. Existing result fields cannot represent an earlier possibly accepted attempt followed by 404.

**Rationale**: One-shot bulk transport can truthfully distinguish accepted, rejected and uncertain. An observed completed job can satisfy the target independently of who completed it. Missing data cannot establish that fact.

**Alternatives considered**: Blindly replaying completion can hide races. General retry-framework changes would affect unrelated commands. Requiring whole-process termination would expand both scope and latency.

**Policy**: No automatic mutation retry in the first bulk implementation, including backpressure rejection; report and let the operator select unresolved jobs later. Normal accepted responses skip independent confirmation. Reconciliation uses existing backoff timeout/timing, bounded by caller deadline; no-wait performs at most one read.

Official API context: [Complete job](https://docs.camunda.io/docs/apis-tools/orchestration-cluster-api-rest/specifications/complete-job/) documents success and missing/wrong-state responses. The [Camunda 8.8.0 processor](https://github.com/camunda/camunda/blob/8.8.0/zeebe/engine/src/main/java/io/camunda/zeebe/engine/processing/job/JobCompleteProcessor.java) describes listener continuation and variables restrictions. These do not prove the particular production cancellation's cause or current state.

## 5. Eligibility without inventing engine guarantees

**Decision**: Observed `COMPLETED` is a no-op; `CANCELED` and `ERROR_THROWN` are unsatisfied other-terminal outcomes. Known nonterminal searchable states may be submitted after scope checks. Unknown states are blocked conservatively. The backend remains authoritative.

**Evidence**: `validateUpdateJobPlanPreconditions` deliberately skips state checks for worker outcomes. Existing service completion has no state/retry precondition lookup. Search states include `CREATED`, `FAILED`, `TIMED_OUT`, `RETRIES_UPDATED`, `MIGRATED`, `COMPLETED`, `CANCELED`, `ERROR_THROWN`; they are observations, not locking guarantees.

**Rationale**: There is no verified basis for insisting on retries greater than zero, or for declaring CREATED jobs abandoned. Completion is not permission to reset retries or resolve incidents. The full version-specific engine eligibility matrix is not needed: rejection remains a structured outcome.

**Alternatives considered**: A positive-retry guard could incorrectly exclude targets. Completing every unknown state would suppress uncertainty. Inferring root cancellation from CANCELING ignores interrupting BPMN events.

Nonempty variables with any task-listener candidate fail batch preparation before writes; recheck protects against changed kind. Legacy single-key payload behavior is not silently rewritten.

## 6. Concurrency and complete reports

**Decision**: Reuse low-level `toolx/pool.ExecuteSlice` with initialized per-key outcomes, original indices, explicit started tracking and caller-error propagation.

**Evidence**: Pool results preserve length but unstarted entries are zero values, and caller cancellation may yield nil aggregate error. `toolx/pool/bulk.go` also imports facade options and emits generic informational logs.

**Rationale**: Every frozen key must remain accounted for after fail-fast or interruption. Index-based aggregation avoids completion-order instability.

**Alternatives considered**: A new worker pool duplicates infrastructure. Treating zero rows as successful loses work. Using a facade-oriented convenience wrapper in services mixes boundaries and output behavior.

## 7. CLI compatibility and one-envelope failures

**Decision**: Choose legacy versus collection mode from the original input form before deduplication. Keep batch lifecycle in a dedicated command file and all formatting in job views. Return report plus error and render once.

**Evidence**: Current update command requires one string key in both parsing and capability metadata. Existing completion rejects retries, returns a single result, and exits before rendering on error. Shared `exitAfterRenderedResult` already supports exiting without an additional error envelope.

**Rationale**: Filters/stdin always produce the bulk shape. Repeated keys deduplicating to one must not unexpectedly switch schema. Selector retries require mode-specific interpretation while keyed legacy rejection remains stable.

**Alternatives considered**: Making every call bulk breaks scripts. Printing per-job JSON produces multiple documents. Calling the ordinary error handler after rendering a partial report produces a second envelope.

## 8. Validation and isolated release

**Decision**: Use deterministic HTTP fixtures for 1,000-job paging/concurrency/error cases, existing real-terminal utilities for stream behavior, and a new narrowly scoped disposable cancellation-listener hierarchy for live proof.

**Evidence**: Existing version service tests and `cmd/update_job_outcome_test.go` provide request tracking. `testx/cmd_terminal_runner.go` supplies terminal stdin. Embedded user-task/parent BPMN models provide model syntax references, but no matching cancellation-listener fixture was found on this baseline.

**Rationale**: Scale proof should be reproducible without creating thousands of production instances. A separate small live scenario establishes actual listener/cancellation behavior. Integration harness internals are validation tooling, not a source of product requirements.

**Alternatives considered**: Depending on #341 fixtures couples this release to develop. Running recovery against existing tenant data is not a fixture. Planning-only runtime tests would violate the proportional-validation constitution.

`main` and `v4.3.4` resolved to the same pinned baseline during specification. Work stays on `345-bulk-job-completion`; choosing a later release number and publishing are outside this design.
