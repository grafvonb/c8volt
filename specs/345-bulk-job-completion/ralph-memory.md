# Ralph Memory

Feature: 345-bulk-job-completion
Started: 2026-10-20T10:28:15Z

## Codebase Patterns

- Job search adapters are in `internal/services/job/v88`, `v89`, and `v810`; `GetJob` builds its own key-only filter and must remain independent of discovery tenant configuration.
- Discovery filters use `common.EffectiveTenant`. v88 uses common generated filter helpers; v89/v810 use local helpers because their generated types differ. Forward-cursor `After` is a value in v88/v89 and a pointer in v810.
- `JobSearchPage` / public `job.Page` carry `RawItemCount`, `NextFrom`, and `EndCursor`; request `After` carries the forward cursor. `NextFrom` counts the consumed raw window before tenant filtering, while `RawItemCount` retains the original response count, including overfull rows. Facade mappings stay mechanical.
- Use internal `JobSearchQuery.SortByKey` for stable ascending job-key bulk discovery. Ordinary reads omit sort. The future bulk coordinator must request this explicitly.

- Bulk submission adapters expose only private `submitBulkCompletion(ctx, job domain.Job, variables map[string]any) (domain.CompletionSubmission, error)` ports. Pass the latest rechecked job when wiring the coordinator. Shared payload validation/evidence classification live in `internal/services/common/job_completion.go`; generated requests still use each adapter's existing `newCompleteJobRequestBody`.

## Decisions

- T001 enforces named discovery tenant both in generated request JSON and defensively on returned rows. Empty scope omits tenant equality. Explicit `GetJob` retains cross-tenant backend authorization.
- Prefer an advancing end cursor, otherwise advance with `NextFrom`. Overfull-response cursors are suppressed because they point beyond the consumed window; retain the existing full-response offset slicing behavior.
- Exact totals below consumed rows, negative/overflowing offsets, contradictory empty offset pages, and nonadvancing/cyclic cursors fail as `ErrMalformedResponse` with `incomplete job search` detail. Do not turn these into successful exhaustion.
- Lower-bound totals require continued reads even for short nonempty pages. An empty cursor-free page can establish exhaustion only after the lower bound has been exceeded. Empty advancing-cursor runs are bounded by normalized existing backoff timeout and any earlier caller deadline; raw progress resets this ambiguity budget.
- Scoped totals count selected pages because defensive tenant filtering makes raw totals unsafe; unscoped exact totals retain the fast path. No new shared error category or generated client edit is needed.

- T002 stores submission status, optional HTTP status, and diagnostic independently of later observations. Only known successful statuses establish acceptance; explicit validation/auth/not-found/conflict/rate-limit responses establish rejection. A 503 establishes backpressure rejection only with a structured problem title/detail containing `RESOURCE_EXHAUSTED`; generic server/proxy failures and call errors remain uncertain. Returned errors retain existing domain/context/transport identity.
- Bulk preflight rejects nonempty task-listener variables, non-JSON variables, and an already-canceled context without calling the generated client. After dispatch, call errors remain uncertain even if response status is also present. No retry or reconciliation occurs in this primitive; legacy submission remains unchanged and its 429 retry has explicit regression coverage.

## Gotchas

- `testx.TestConfig` sets tenant to `tenant`, while many legacy job-search rows omit tenant. The version-local job test constructor now explicitly defaults to unscoped discovery; tenant tests set their intended scope.
- `common.PrepareServiceDeps` does not normalize backoff settings. Normalize a copy before applying an ambiguity timeout so zero-valued library/test configuration does not expire immediately.
- Legacy filter-conversion fixtures omitted mandatory page totals; they now provide accurate totals. The old v88 lower-bound counting fixture ended empty below a reported bound of 100; it now uses a consistent bound of 1. Separate regressions require contradictory empties to fail.
- A future coordinator calling `SearchJobsPage` directly must own cross-page cursor-cycle detection, empty-page timeout, unique-candidate limits, and predicate verification. Preserve raw advancement even when all rows are filtered or deduplicated.

- Generated completion parsers discard the response on JSON/body decoding failure. The bulk primitive conservatively returns uncertainty with no HTTP status when the generated call returns no response; do not fabricate status evidence or edit generated clients to recover it.
- v810 completion JSON retains generated `businessId: null` and `leaseToken: null`; v88/v89 emit only the variables object for an empty payload. Preserve these existing shapes.
- Go vet cannot track the existing mutable deferred cancellation closure when `WithTimeout` assigns directly to `cancelAmbiguity`. Create local context/cancel values and transfer ownership to the closure-tracked variables; keep both progress reset and final cleanup. All adapters now pass the lostcancel check.

## Reusable Commands

- `go test ./internal/services/job/... -run 'Test.*(JobSearch|SearchJobs|GetJob)' -race -count=1`
- `go test ./c8volt/job ./cmd -run 'Test.*(SearchJobs|JobSearch|GetJob)' -race -count=1`
- `go test ./cmd ./c8volt/job ./internal/services/job/... ./internal/services/element/... ./internal/services/processinstance/... -race -count=1`
- `go test ./... -run '^$'`
- `make docs-content` and `git diff --check`

- `go test ./internal/services/job/... -run 'Test.*(BulkCompletion|SubmitJobWorkerOutcome)' -count=1`
- `go test ./internal/services/common ./internal/services/job/... ./c8volt/job -race -count=1`
- `go test ./cmd -run 'Test.*(UpdateJob|SubmitJob|JobSearch|SearchJobs|GetJob)' -race -count=1`
- `go vet ./internal/services/common ./internal/services/job/... ./internal/domain`

## Do Not Repeat

- Do not advance by filtered or deduplicated result count, use a truncated response's end cursor, or assume a short/empty page overrides contradictory totals.
- Do not reuse the legacy completion retry wrapper for the upcoming bulk-only one-shot primitive.
- Do not import develop/#341, move release tags, or publish; the feature remains on the v4.3.4 baseline branch.

## Current Handoff

- T001 and T002 are complete and validated; the Foundational phase is complete. Recheck selection against `tasks.md`; T003 (US1: read-only batch preparation, complete domain/public contracts, all adapter and mock wiring) is the next dependency-ready task. T004/T005 share US1 but require T003 first.
- T004 must inject the private one-shot port and preserve its original submission evidence across reconciliation; a missing later read cannot turn uncertainty into rejection or completed success. Preparation and execution must both enforce task-listener payload compatibility.
- T004–T009 remain blocked by the declared dependency chain. No story work was started in iteration 2; no external blocker exists for T003.
