# Service Contract: Bulk Job Completion

## Public facade

Add matching methods to the job facade and internal service interfaces:

```text
PrepareJobCompletionBatch(ctx, request, options...) -> (preparedBatch, error)
ExecuteJobCompletionBatch(ctx, preparedBatch, wantedWorkers, options...) -> (report, error)
```

Public signatures use public models and `foptions.FacadeOption`; internal signatures use domain models and service options. Facades only copy/map inputs, delegate, map outputs and convert errors through `ferrors`. A non-nil error must not discard a returned report.

Keep existing `GetJob`, search, update and worker-outcome signatures compatible. All interface implementations and mocks must compile together. v87 returns the existing unsupported category, including empty batch requests; factory support is not removed.

Preparation is read-only. Execution rejects incomplete/invalid plans and changes in configured discovery scope. The CLI passes the exact prepared value to the same client. Plans are not durable cross-client replay objects. Library callers must treat them as immutable; services validate unique keys, selection mode, completeness and scope before any writes.

Honor `WithDryRun` inside execution as well: return a preview without invoking the submission port, even when a library caller bypasses the CLI's preview branch. Force/no-state-check options cannot bypass batch completeness, tenant/filter scope, or listener-payload safeguards. Carry effective scope once; do not silently broaden it through nested options.

## Internal ownership

`internal/services/job/completion` accepts adapter operations for key lookup, normalized page lookup and one-shot completion. It imports domain types, service options and shared helpers, not the parent job factory, facade, or generated client packages.

Adapters retain version-specific filters, response normalization, payload validation/classification and HTTP calls. Use generated `WithResponse` methods and existing payload guards. No generated code edits are required for tenant equality or job-key sorting.

### Page evidence

Normalized page facts must preserve raw traversal position, requested size, raw returned count, reported total and whether that total is exact/lower-bound, plus continuation certainty. Extend internal metadata additively where current page fields cannot represent those facts; facades map mechanically.

- Stable ascending key sort applies to bulk traversal only; preserve existing read command order unless already equivalent.
- Compute next raw offset before defensive matching and deduplication; never advance by unique selected count.
- Complete preparation only on proven exhaustion or intentional unique limit.
- Empty pages with credible later continuation are traversed. Empty/short pages with contradictory totals cannot silently establish exhaustion.
- Nonadvancing/invalid metadata fails preparation. Bound ambiguity resolution with existing backoff timeout and caller deadline; return incomplete when proof is unavailable.
- Reaching a valid target limit is not incomplete. Duplicates/out-of-tenant rows do not count toward it.
- Shared search/total callers must not silently map new indeterminate continuation to successful exhaustion. Preserve their documented behavior where evidence is conclusive; test changed edge cases explicitly.

### One-shot submission

A private adapter operation makes one generated completion call, bypassing `RetryCamundaMutation` only for this bulk workflow. Reuse existing payload construction. Existing single-key submission retains its original policy.

Classify evidence conservatively: a successful completion response is accepted; explicit backend validation/not-found/conflict/auth/backpressure rejection is rejected; network loss, timeout after dispatch or an ambiguous server/proxy response is uncertain. Do not classify every non-2xx response as proof no operation occurred. Preserve the status/diagnostic when available. Never automatically resubmit.

### Recheck and reconciliation

1. Re-read the selected key; retain missing/read errors rather than substituting another job.
2. Observed COMPLETED satisfies the target without mutation or attribution.
3. Otherwise verify original filter/tenant scope and payload compatibility. Other terminal or unknown states receive an unsatisfied disposition.
4. Known nonterminal observations (`CREATED`, `FAILED`, `TIMED_OUT`, `RETRIES_UPDATED`, `MIGRATED`) may proceed; acceptance is not guaranteed. Do not require positive retries or reset them.
5. Submit at most once. Accepted responses report confirmation skipped.
6. For rejection 404/409 and uncertain submissions, read within the existing backoff budget/caller deadline. Stop on COMPLETED or decisive terminal observation; missing/nonterminal visibility may remain stale and does not prove completion. No-wait permits one read without polling.
7. Caller cancellation ends outstanding work promptly and preserves evidence; fail-fast stops new scheduling without discarding in-flight outcomes.

No mutation uses search results as a lock. No read response proves listener business logic ran. Completing one listener may cause a later listener to appear, but it is outside this frozen selection.

## Reporting invariants

Initialize all frozen outcomes before dispatch. Merge by original index, preserve successful and failed rows, and explicitly join caller errors even if the pool reports none. Reuse existing pool limits/options. Default continues independent keys; fail-fast stops scheduling at the first final unsatisfied result.

The facade returns both partial report and error. The command uses one mode-aware renderer and an exit path that does not render again. Existing normalized error classes and explicit exit-code suppression remain authoritative.

Required tests cover prepared-plan rejection without writes, map/slice isolation, option propagation, deterministic ordering, no duplicate submissions, every unstarted target represented, adapter version parity, and accepted versus independently observed completion. See [quickstart.md](../quickstart.md) for end-to-end validation.
