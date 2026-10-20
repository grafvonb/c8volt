# Ralph Memory

Feature: 345-bulk-job-completion
Started: 2026-10-20T10:28:15Z

## Codebase Patterns

- Job search adapters are in `internal/services/job/v88`, `v89`, and `v810`; `GetJob` builds its own key-only filter and must remain independent of discovery tenant configuration.
- Discovery filters use `common.EffectiveTenant`. v88 uses common generated filter helpers; v89/v810 use local helpers because their generated types differ. Forward-cursor `After` is a value in v88/v89 and a pointer in v810.
- `JobSearchPage` / public `job.Page` carry `RawItemCount`, `NextFrom`, and `EndCursor`; request `After` carries the forward cursor. `NextFrom` counts the consumed raw window before tenant filtering, while `RawItemCount` retains the original response count, including overfull rows. Facade mappings stay mechanical.
- Use internal `JobSearchQuery.SortByKey` for stable ascending job-key bulk discovery. Ordinary reads omit sort. The bulk preparation coordinator requests this explicitly while leaving the raw adapter limit uncapped.

- Bulk submission adapters expose only private `submitBulkCompletion(ctx, job domain.Job, variables map[string]any) (domain.CompletionSubmission, error)` ports. Pass the latest rechecked job when wiring the coordinator. Shared payload validation/evidence classification live in `internal/services/common/job_completion.go`; generated requests still use each adapter's existing `newCompleteJobRequestBody`.

- `internal/services/job/completion.Prepare` accepts only `ReadPorts` (Get/Search), configured tenant and backoff. v88/v89/v810 delegate; v87 rejects even empty stdin. Facade, version API interfaces, and job/element/process test doubles are wired together.
- Batch models are distinct public/domain types. `CompletionFilter` contains only the nine predicates; controls and input form live in `CompletionSelection`. Public `CompletionBatchReport.MarshalJSON` emits candidates for preview/preparation_failed and outcomes for execution, enforcing empty arrays and omitting the other collection.

- `completion.Execute` owns execution through `ExecutionPorts` (key-only Get and one-shot Submit), with plan snapshot/invariant checks in `execution_plan.go` and bounded observation in `reconcile.go`. Adapters and all API/test doubles expose ExecuteJobCompletionBatch; facade conversions copy plans, reports, nested variables and optional HTTP status.

- Selector CLI routing is `parseUpdateJobCommandRequest` -> `parseUpdateJobBatchRequest` -> `runUpdateJobBatch`. New selector values use command-local pflags, with pure `validateJobSearchPredicates` shared with reads; never assign `flagGetJob*` to build an update request. Scalar requests still use the legacy parser/runner.

- Explicit CLI collection parsing stays in `cmd/update_job_batch.go`; `--key` is a stringArray to preserve repeated occurrences and raw comma tokens. Dispatch chooses collection mode before deduplication, then passes flag-before-stdin order to the existing service. Scalar requests still use the legacy parser.

- T007 HTTP acceptance lives in `cmd/update_job_batch_acceptance_test.go`; its fixture is file-local and reuses existing listener row/envelope helpers. It records raw search offsets/cursors, key-only reads, one-shot writes and peak overlapping requests without serializing handlers. Subprocesses exercise real exit status and separate streams.
- Cancellation acceptance uses a test-only server handshake to cancel the actual Cobra context during a known discovery or mutation request; avoid timing-based sleeps. A four-request barrier demonstrates real HTTP concurrency independently of scheduling speed.

## Decisions

- T001 enforces named discovery tenant both in generated request JSON and defensively on returned rows. Empty scope omits tenant equality. Explicit `GetJob` retains cross-tenant backend authorization.
- Prefer an advancing end cursor, otherwise advance with `NextFrom`. Overfull-response cursors are suppressed because they point beyond the consumed window; retain the existing full-response offset slicing behavior.
- Exact totals below consumed rows, negative/overflowing offsets, contradictory empty offset pages, and nonadvancing/cyclic cursors fail as `ErrMalformedResponse` with `incomplete job search` detail. Do not turn these into successful exhaustion.
- Lower-bound totals require continued reads even for short nonempty pages. An empty cursor-free page can establish exhaustion only after the lower bound has been exceeded. Empty advancing-cursor runs are bounded by normalized existing backoff timeout and any earlier caller deadline; raw progress resets this ambiguity budget.
- Scoped totals count selected pages because defensive tenant filtering makes raw totals unsafe; unscoped exact totals retain the fast path. No new shared error category or generated client edit is needed.

- T002 stores submission status, optional HTTP status, and diagnostic independently of later observations. Only known successful statuses establish acceptance; explicit validation/auth/not-found/conflict/rate-limit responses establish rejection. A 503 establishes backpressure rejection only with a structured problem title/detail containing `RESOURCE_EXHAUSTED`; generic server/proxy failures and call errors remain uncertain. Returned errors retain existing domain/context/transport identity.
- Bulk preflight rejects nonempty task-listener variables, non-JSON variables, and an already-canceled context without calling the generated client. After dispatch, call errors remain uncertain even if response status is also present. No retry or reconciliation occurs in this primitive; legacy submission remains unchanged and its 429 retry has explicit regression coverage.

- T003 preserves first-seen matching candidates (including terminal states), deduplicates before the unique cap, and marks completion only as exhausted, limit_reached, or explicit_inputs_read. Any read/metadata/payload/cancellation failure retains incomplete diagnostic candidates and clears completion reason. Partial observations are never executable.
- Explicit decimal keys and predicate keys are validated as positive signed 64-bit values, then canonicalized without floating-point conversion; normalized input order and duplicates remain in provenance, while unique candidate reads preserve first-input order. Discovery tenant is recorded for all plans but only constrains search membership. Actual selected tenant IDs retain first-seen order.
- `toolx.CopyJSONMap` snapshots nested JSON containers at the request/prepared facade boundaries and in service preparation; numbers normalize to `json.Number` to preserve large integer precision. Invalid/cyclic/non-JSON payloads fail before reads.

- T004 validates normalized selection, unique canonical keys, explicit input order, candidate scope/tenant evidence, preparation reason/count/limit consistency and unchanged configured discovery tenant before remote work. Direct `WithDryRun` returns a candidate preview without execution reads or writes. Force/no-state-check/ignore-tenant cannot bypass these invariants or current scope/payload checks.
- T004 rechecks only frozen keys. COMPLETED is satisfied without submission; changed scope, incompatible payload, other terminal and unknown states remain unsatisfied. Known nonterminal states may submit with zero retries. Accepted submissions skip confirmation; only uncertain or 404/409 rejection evidence triggers bounded key-only reconciliation. Preserve submission evidence even when later reads are missing; observed completion clears the aggregate failure for that target without attributing who completed it.
- T004 reuses `DetermineNoOfWorkers` and `pool.ExecuteSlice`. Callback results carry explicit started state and errors; overlay only started results onto preinitialized rows. Pool cancellation stops fail-fast scheduling, while started work uses the original caller context. Join callback errors and caller errors, excluding pool-internal fail-fast cancellation. Every execution retains the frozen order and disposition count partition.

- Batch command results use `renderOutputLine` and explicit quiet handling, preserving stdout for selected-format results. `jobCompletionBatchView` renders one shared envelope with report-plus-error; the caller uses `exitAfterRenderedResult` on errors, including suppressed exit codes. No-wait becomes accepted only when report accepted count is positive. Prompts use configured stderr and include frozen count, filters, scope and listener consequence.

- T006 validates all collection/stdin tokens as positive signed-int64 decimal strings before preparation, preserving duplicates and spellings for service-owned canonicalization. Blank stdin lines are ignored; intentional empty stdin remains a batch. Explicit inputs reject predicates/search controls and every unrelated mutation mode.
- Explicit batch rendering attaches `explicit_keys` tenant context with actual tenant evidence, names explicit selection in summaries/prompts, and omits search-page detail. Empty explicit results say `no selected jobs`; scalar payloads and selector wording remain compatible.

## Gotchas

- `testx.TestConfig` sets tenant to `tenant`, while many legacy job-search rows omit tenant. The version-local job test constructor now explicitly defaults to unscoped discovery; tenant tests set their intended scope.
- `common.PrepareServiceDeps` does not normalize backoff settings. Normalize a copy before applying an ambiguity timeout so zero-valued library/test configuration does not expire immediately.
- Legacy filter-conversion fixtures omitted mandatory page totals; they now provide accurate totals. The old v88 lower-bound counting fixture ended empty below a reported bound of 100; it now uses a consistent bound of 1. Separate regressions require contradictory empties to fail.
- The completion coordinator owns cross-page cursor-cycle detection, empty-page timeout, unique-candidate limits, and all nine predicate checks. Preserve raw advancement even when all rows are filtered or deduplicated.

- Generated completion parsers discard the response on JSON/body decoding failure. The bulk primitive conservatively returns uncertainty with no HTTP status when the generated call returns no response; do not fabricate status evidence or edit generated clients to recover it.
- v810 completion JSON retains generated `businessId: null` and `leaseToken: null`; v88/v89 emit only the variables object for an empty payload. Preserve these existing shapes.
- Go vet cannot track the existing mutable deferred cancellation closure when `WithTimeout` assigns directly to `cancelAmbiguity`. Create local context/cancel values and transfer ownership to the closure-tracked variables; keep both progress reset and final cleanup. All adapters now pass the lostcancel check.

- Generated equality filters serialize as scalar strings (`tenantId: "tenant"`, `jobKey: "..."`), not `$eq` objects in these adapters. Cross-version HTTP preparation fixtures assert the actual wire form and count all non-search calls as unexpected mutations.
- Prepared candidate JSON contains only observed context (including worker and retries), with no synthetic lifecycle dates. The report does not embed the prepared plan, avoiding duplicate candidate collections on execution. Aggregate errors remain separate return values.

- `pool.ExecuteSlice` cancels its derived context on fail-fast, including in-flight requests if that context is passed to adapters. Use its context only at dispatch admission; keep started requests on the caller context and an atomic stop flag to close the scheduling gap. Caller cancellation still stops started requests and marks the invocation interrupted even if every returned row was accepted.
- Reconciliation normalizes a copy of existing backoff configuration and uses its timeout, delays and max retries. No-wait performs at most one bounded read. Preserve the last known state on missing/unreadable observations; reject lookup responses for another key. Internal reconciliation timeout is an unsatisfied finished batch; caller timeout/cancellation makes the batch interrupted.

- `renderHumanLine` uses the root logger on stderr; it is not a stdout result helper. Tenant context human lines include override provenance, so exact output assertions must account for configured tenant and the explicit override. The bulk view reuses those lines on stdout.
- Output capability metadata alone does not reject inherited `--keys-only`; batch parsing must reject the effective keys-only mode before preparation. Preserve JSON precedence. Existing `localPreconditionError` preserves its class but not `ErrCmdAborted` identity, so test the established class/message contract.
- Existing CLI predicate-key validation requires 16 decimal digits. Reuse it for get/update selector consistency; internal batch APIs separately validate positive int64 keys. Zero `--limit` is explicitly valid for bulk selection even though `get job --limit 0` is rejected.

- Global CLI metadata tests require recurring shorthand consistency (`--state` uses `-s`) and action-focused help without `stdout`/`stderr` wording. Include `TestCommandCapability` and `TestAllCommandHelp` in targeted metadata checks; `Test.*CommandContract` alone misses them.

- pflag `GetStringArray` round-trips values through CSV and loses a lone empty string. Read `Lookup("key").Value.(pflag.SliceValue).GetSlice()` to reject `--key ''`, including combined stdin. The command dispatcher resets its legacy scalar value because it is no longer directly bound to the key flag.
- Existing `readKeysIfDash` rejects empty stdin and `validateKeys` requires 16 digits, so they cannot implement the completion collection contract unchanged. The batch-local scanner uses `cmd.InOrStdin()` and the existing scanner buffer bound, retains reader errors, and leaves shared stdin behavior untouched.
- Empty-stdin terminal tests use a test-only stderr readiness marker before command execution so `CmdTerminalRunner` can send real canonical EOF. Assert that marker is the entire stderr stream; no product prompt is added.

- Generated key-only search filters may contain nullable `deadline` and `isDenied` fields encoded as null. These are not extra predicates; HTTP assertions should reject additional non-null predicates while permitting generated null fields.

## Reusable Commands

- `go test ./cmd -list 'Test.*BulkCompletionAcceptance'` and `go test ./cmd -run 'Test.*BulkCompletionAcceptance' -count=1`
- `GORACE=atexit_sleep_ms=0 go test ./cmd -run 'TestBulkCompletionAcceptanceScale' -race -count=1 -v` records per-adapter counts and concurrency.

- `go test ./internal/services/job/... -run 'Test.*(JobSearch|SearchJobs|GetJob)' -race -count=1`
- `go test ./c8volt/job ./cmd -run 'Test.*(SearchJobs|JobSearch|GetJob)' -race -count=1`
- `go test ./cmd ./c8volt/job ./internal/services/job/... ./internal/services/element/... ./internal/services/processinstance/... -race -count=1`
- `go test ./... -run '^$'`
- `make docs-content` and `git diff --check`

- `go test ./internal/services/job/... -run 'Test.*(BulkCompletion|SubmitJobWorkerOutcome)' -count=1`
- `go test ./internal/services/common ./internal/services/job/... ./c8volt/job -race -count=1`
- `go test ./cmd -run 'Test.*(UpdateJob|SubmitJob|JobSearch|SearchJobs|GetJob)' -race -count=1`
- `go vet ./internal/services/common ./internal/services/job/... ./internal/domain`

- `go test ./internal/services/job/... -run 'Test.*(BulkCompletion|JobSearch)' -count=1`
- `go test ./c8volt/job -run 'Test.*BulkCompletion' -count=1`
- `go test ./internal/services/job/... ./c8volt/job ./c8volt/element ./c8volt/process ./toolx -race -count=1`
- `go vet ./internal/services/job/... ./c8volt/job ./internal/domain ./toolx`

- `go test ./internal/services/job/... ./c8volt/job -run 'Test.*BulkCompletion' -count=1`
- `go test ./internal/services/job/completion -race -count=1`
- `go test ./internal/services/job/... ./c8volt/job ./c8volt/element ./c8volt/process -race -count=1`
- `go vet ./internal/services/job/... ./c8volt/job ./c8volt/element ./c8volt/process`

- `go test ./cmd -run 'Test.*(UpdateJob|BulkCompletion|CommandContract|CommandCapability|AllCommandHelp)' -count=1`
- `GORACE=atexit_sleep_ms=0 go test ./cmd -race -count=1` (race detector enabled; avoids the default exit delay for each helper subprocess)
- `go vet ./cmd`

## Do Not Repeat

- Do not advance by filtered or deduplicated result count, use a truncated response's end cursor, or assume a short/empty page overrides contradictory totals.
- Do not reuse the legacy completion retry wrapper for the upcoming bulk-only one-shot primitive.
- Do not import develop/#341, move release tags, or publish; the feature remains on the v4.3.4 baseline branch.

## Current Handoff

- T007 is complete and validated; US3 remains partial. Recheck `tasks.md`: T008 (genuine terminal acceptance) is the next dependency-ready task. T009 depends on T008 and remains blocked; its live SC-006 recovery proof is still required. No external blocker exists for T008.
- T007 adds HTTP acceptance only, with no production or generated-document changes. Each supported adapter previews 1,000 candidates without key lookups or writes, then accounts for 1,000 mixed outcomes from 1,102 raw rows over 12 pages: 1,300 key reads, 500 single submissions, peak four concurrent HTTP requests, 200 accepted / 300 observed completed / 500 unsatisfied / zero unstarted. Wrong-tenant/type rows and duplicates are excluded.
- Additional acceptance covers sparse cursor/offset traversal, unique limits, contradictory/cyclic/error pages, rejection and real lost-response reconciliation (completed versus missing), polling versus no-wait, fail-fast, interrupted preparation/execution, terminal/filter/tenant races, sequential-listener exclusion, listener-variable zero-write failure and v87 unsupported empty input. The output matrix verifies normal/dry-run human/verbose/JSON/quiet, automation/auto-confirm/no-wait, authoritative empty search/stdin, incomplete preparation, exact empty human output, array omission rules and one envelope with EOF on nonzero exits.
- Passed acceptance test discovery and required targeted suite (12.014s), `go vet ./cmd`, full command race suite (`GORACE=atexit_sleep_ms=0 go test ./cmd -race -count=1`, 144.660s), and the final scale count assertion under race (2.563s); gofmt and diff checks passed. The first acceptance run exposed only an overly strict fixture assertion about generated null filter fields; no product defect was found. T008 owns the deeper terminal destination/interaction matrix; T009 owns delivery-wide tests and live recovery evidence. No live mutation, push or publication occurred.
