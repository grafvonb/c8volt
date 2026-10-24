# Tasks: Optional Request Pacing and DEBUG Rate Reporting

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [CLI contract](contracts/request-rate.md), [quickstart.md](quickstart.md)

**Branch / issue**: `353-request-rate-reporting` / #353

**Organization**: Cohesive work units by user story. Each checkbox includes implementation, necessary fixtures, tests, formatting, documentation, and targeted validation. Tests are required by FR-015; write the regression assertions first within each unit, establish the expected failure, then implement and finish with passing checks. New test names include `RequestRate` for selection.

## Work-unit rules

- Follow `AGENTS.md`, `.specify/memory/constitution.md`, and, for Ralph, `specs/ralph-implementation-rules.md`. Reading context, branch checks, progress records, and commit preparation are responsibilities inside each unit, not additional tasks. Issue-associated commit subjects end with ` #353`.
- Preserve all existing mutation workflows, workers, selection, preflight, dry-run, confirmation, ordering, no-wait, result schemas, partial results and exit classes. No new resource limits, batch modes, adaptive control, generated-client edits or public facade APIs.
- Only isolated unit/generated-client fakes, test-owned HTTP servers, CLI subprocesses and terminal fixtures. Inspect targets and sanitize inherited configuration, credentials and `GOFLAGS`; never discover or test a live Camunda target. Follow [quickstart.md](quickstart.md) for the full matrix and environment isolation.
- Paths below are repository-relative; proposed new filenames are implementation targets. Reuse nearby fixtures/helpers instead of creating another framework. Add required comments and run `gofmt` on changed Go files. A passing command that selected zero tests is not evidence.
- Start with each task's targeted checks. Run `GOFLAGS='' make test` before completing a unit that changes shared runtime concurrency/dependencies; subsequent relevant changes justify another run, bookkeeping alone does not. Record actual checks and gaps in `specs/353-request-rate-reporting/progress.md` during implementation. No runtime checks are performed merely to create this task list.

## Phase 1: Setup

The repository, feature branch and planning artifacts already exist. No standalone initialization task is needed. Add the limiter dependency with its first consumer in T001.

## Phase 2: Foundational

Reuse the existing configuration resolver, shared HTTP service, retry helpers and logger. No separate scaffolding or interface task is needed; each story owns its small additions and all affected callers/test doubles together.

## Phase 3: User Story 1 — Pace a Large Command Without Changing Its Scope (P1)

**Goal:** One optional request-start allowance across discovery, preflight, mutation, confirmation and application retries, with truthful unsent local failures.

**Independent test:** Run concurrent fake requests plus representative cancel, instance-delete, definition-delete and purge commands with zero/positive rates; prove one burst-one allowance, unchanged targets/order/results and no dispatch or extra retry after a local wait failure.

- [x] T001 [US1] Deliver shared request pacing with timeout-safe retry handling in `config/http.go`, `cmd/root.go`, `cmd/root_config.go`, `internal/services/httpc/service.go`, new `internal/services/httpc/request_rate.go` and `internal/services/httpc/request_rate_error.go`, `internal/services/httpc/read_retry.go`, and `internal/services/retry.go`, including the pinned dependency in `go.mod`/`go.sum`, focused tests and operator docs.

  **Configuration and dispatch:** Add `http.request_rate` and persistent `--request-rate`, default `0`. Apply the model constraint verbatim: “Finite and nonnegative; positive values are requests/second, including fractions. Zero bypasses pacing.” Reject negative, malformed and non-finite CLI/file/profile/env values before authentication/API work; preserve explicit zero overrides through `resolverBindings` and existing overlays in `config/config.go`. Use one `golang.org/x/time v0.16.0` limiter per shared client: “Positive rate, fixed burst 1.” Place admission below `ReadRetryTransport` and above `LogTransport`/diagnostics; update both transport traversal helpers. Rate zero creates no limiter or pacing wait. Do not hold a lock over network I/O or change worker counts. Shared-client cookie bootstrap traffic participates; separate OAuth provider clients keep diagnostic sharing only, and invisible transport replays stay outside the contract.

  **Local errors and retry safety:** Respect existing request/workflow deadlines; map early insufficient-deadline admission to a typed unsent deadline failure and recheck context before delegation. Preserve cancellation/deadline identity. Exclude local failures from read retries. For positive pacing only, have `RetryCamundaMutation` attach fresh context-local observation to each callback; record admission failure in the transport and consult that observation before ordinary retry classification. Cover actual `http.Client.Timeout` error-type replacement with a live parent context; do not parse error strings or change backend timeout/429/eligible-503 budgets/backoff. At rate zero pass the original context without observation allocation; evidence must not leak between attempts/resources. “Not sent” describes only the affected attempt, never earlier attempts or the whole command.

  **Proof and docs:** Add `config/http_test.go` and `internal/services/httpc/request_rate_test.go`; extend `config/config_test.go`, `cmd/config_test.go`, `internal/services/httpc/http_read_retry_test.go` and `internal/services/retry_test.go`. Assert default/fractional/explicit-zero precedence, invalid-before-request behavior, shared concurrent delegation starts (`starts <= 1 + rate × elapsed` with documented small timing tolerance), idle burst capacity, overlapping in-flight calls, cancellation while queued, early deadline rejection, actual client-timeout wrapping, unchanged eligible backend retries and zero-rate pass-through. Exercise the flag through root command execution with isolated HTTP fakes. Update root help and `README.md` for invocation scope, overrides, rate versus concurrency and unchanged deadlines; regenerate `docs/cli/` with `make docs-content`, without promising GOAWAY elimination. Run targeted `GOFLAGS='' go test ./config -run RequestRate -count=1`, equivalent `./internal/services/httpc`, `./internal/services` and `./cmd` selections, plus affected existing retry/config/diagnostic regressions; run the shared-runtime race check from the work-unit rules.

- [x] T002 [US1] Preserve mutation outcomes and compact local-wait diagnostics through real command execution in new `cmd/request_rate_mutation_test.go`, reusing `cmd/cancel_processinstance_selector_test.go`, `cmd/delete_processinstance_selector_test.go`, `cmd/delete_processdefinition_unfinished_test.go`, `cmd/ops_purge_all_processdefinitions_unfinished_test.go`, and existing error/rendering helpers in `cmd/cmd_views_processinstance_logging.go`, `cmd/cmd_views_ops_purge_all_processdefinitions.go` and `c8volt/ferrors/errors.go` where a narrow correction is necessary.

  **Scope:** Build on T001. Use existing test-owned HTTP fixtures and separate-output subprocess/terminal runners to trace service error → facade conversion → command renderer/logger → exit. Prove request rates are shared through representative cancel, process-instance delete, process-definition delete with cleanup, and purge. Keep original scopes, nested worker settings, cancellation/deletion order, confirmation prompts and counts. Extend the existing bounded reason handling only if needed to identify an affected request as locally unsent; preserve accepted work and available partial results. Do not repair unrelated pagination/output behavior or move orchestration into commands/facades.

  **Acceptance:** Cover normal/nonempty and sparse-page cases, dry-run and empty scopes, supported no-wait/automation/auto-confirm combinations, failure and interruption. Capture stdout/stderr separately. Require exactly one JSON envelope followed by EOF, retained partial evidence and original exit code, exact established empty human output, zero-byte empty keys-only output, and quiet plus explicitly requested machine output. Dry runs never mutate; authoritative empty scopes neither prompt nor mutate; no-wait adds no confirmation. Use real terminal stdin when prompt eligibility matters and check configured/inherited stderr destinations.

  **Failure proof:** Include repeated multi-resource backend errors with distinctive URL/method/body/key markers and a local admission deadline failure. Normal/verbose output must use the configured logger with a compact bounded aggregate, grouping repeated causes and excluding raw HTTP details on both streams; quiet retains the error. Assert severity, destination and no duplicate emission, rather than just an error substring. Preserve existing DEBUG redaction under flag/config enablement and both plain/structured logging. Assertions must detect direct-stderr logger bypass and normal-level raw-error disclosure. Run `GOFLAGS='' go test ./cmd -run RequestRate -count=1` and the affected existing mutation/output/terminal regressions; run closest facade/service tests if their error mapping changes. Keep help/README/generated docs aligned in this same unit if human behavior changes.

**Checkpoint:** US1 is a usable MVP after T001 and T002. It has no dependency on rate reporting.

## Phase 4: User Story 2 — Observe Traffic at a Chosen Interval (P2)

**Goal:** Measure actual starts with or without pacing using configurable DEBUG summaries, without blocking dispatch or changing selected-format results.

**Independent test:** Controlled fake traffic and timing prove default/custom/disabled intervals, exact attempt counts and elapsed rates, effective logger filtering/routing, and prompt reporter shutdown on completion, failure or cancellation. Rate-zero reporting invokes no limiter waits.

- [ ] T003 [US2] Deliver configurable reporting and its complete invocation lifecycle in `config/http.go`, `config/config.go`, `cmd/root_config.go`, `internal/services/httpc/request_rate.go`, new `internal/services/httpc/request_rate_reporter.go`, `internal/services/httpc/service.go`, `cmd/root_services.go` and new `cmd/root_request_rate.go`, with unit/command tests in `internal/services/httpc/request_rate_reporter_test.go` and `cmd/root_request_rate_test.go`, combined acceptance in `cmd/request_rate_mutation_test.go`, overhead benchmarks in `internal/services/httpc/request_rate_benchmark_test.go`, and updated operator docs.

  **Settings and counting:** Add configuration-only `http.request_rate_report_interval`, default `5s`, with the exact constraint “Parseable, nonnegative duration; zero disables reporting. An explicit empty value is invalid.” Reuse configured-key-aware normalization so omission differs from explicit empty input; extend config/profile/env/template tests in `config/http_test.go`, `config/config_test.go` and `cmd/config_test.go`. No interval CLI flag. The dispatch counter is “Needed only for enabled reporting; increment once immediately before transport delegation.” Count after admission/final context check, including each dispatched application retry and failed response, excluding queued/unsent attempts. Zero-rate measurement skips limiter calls; when both controls are inactive retain the old transport path.

  **Windows and logger:** One atomic cumulative counter, actual monotonic snapshot times and one reporter, “One when effective DEBUG and interval > 0; stopped on cancellation or service close.” Derive count/time deltas and measured requests/second, including empty windows; guard nonpositive elapsed and account for delayed ticks and concurrent window boundaries without loss/double counting. Put all three values in one compact message through the configured command logger so plain and structured formats work. Never log per request, expose keys/URLs/bodies/credentials/raw errors in summaries, or hold a reporting lock/write logs in request dispatch. Configured DEBUG and `--debug` work independently of verbose; retain quiet precedence. Interval zero suppresses only rate summaries, not existing DEBUG diagnostics. No final partial-window report is required.

  **Lifetime:** Wire `WithInvocationContext(ctx)` from remote service setup. Add idempotent service close and one root Cobra finalizer; the current cleanup handle is “Consumed/cleared by the singleton root finalizer; no rate/count state is global.” Register service ownership immediately after creation, clean up bootstrap failures/stale handles and stop reporting on invocation cancellation. Test returned `ExecuteC` success/error, initialization failure, repeated invocations, double close and a short command with no report; do not wait for a future interval. Direct service tests own close. Do not broaden concurrent root support or refactor existing process-exit paths.

  **Proof and docs:** Reuse `cmd/root_api_diagnostics_test.go` patterns for effective `--debug`/configured DEBUG, verbose alone, quiet combinations, plain/structured formats and configured/inherited stderr. Assert all three summary values, correct severity/destination, no duplicates and no diagnostic stdout. Prove retry/failure counts, zero windows, actual elapsed rates and no dispatch waiting on reporter output. Update root metadata, configuration guidance and `README.md` examples for `5s`, custom duration, `0s`, zero-rate measurement and unchanged timeout/concurrency behavior; regenerate CLI docs in the same unit. Run targeted `RequestRate` tests in `./config`, `./internal/services/httpc`, `./internal/services` and `./cmd`, plus affected existing diagnostics/config/retry regressions. Run shared-runtime race validation after these new concurrent/lifecycle changes.

  **Combined acceptance and overhead:** Parameterize the T002 mutation fixtures with reporting enabled at zero and positive rates, and disabled interval, using controlled timing to require a summary when appropriate. Retain scope/order, dry-run/empty/no-wait behavior, partial results, stream separation, compact diagnostics and exit classes without multiplying every option into a redundant cross-product. Reuse `internal/services/httpc/diagnostics_benchmark_test.go` patterns to compare pass-through, zero-rate/reporting-off and zero-rate/reporting-on with a fake transport and controlled logger; report allocation costs and prove a blocked reporter sink cannot block dispatch. Confirm benchmark selection exists and run `GOFLAGS='' go test ./internal/services/httpc -run '^$' -bench RequestRate -benchmem`. Record measured results without an invented threshold or zero-overhead claim. Run the full race suite after the final executable changes, not before this combined coverage is complete.

**Checkpoint:** Both stories work together; reporting-only mode remains unthrottled.

## Phase 5: Polish and Cross-Cutting Concerns

No standalone bookkeeping or deferred test/documentation checkbox. T001/T002 own pacing and mutation compatibility evidence; T003 owns combined reporting, benchmarks and final race validation. Before marking the last unit complete, review generated docs and the full quickstart matrix and record actual coverage in `specs/353-request-rate-reporting/progress.md`. Do not rerun passed checks solely to finalize planning/progress records.

## Dependencies and Execution Order

```text
Existing repository → T001 → T002 (US1 / MVP) → T003 (US2 / complete)
```

T001 owns the shared transport and its retry contract. T002 proves mutation workflows against that contract. T003 reuses the transport and command fixtures rather than rebuilding them, and delivers reporting with combined-mode validation. There are no external setup blockers.

## Parallel Examples

**US1:** No independent checkbox pair: T002 consumes T001's request/error behavior. Separate read-only review of config precedence and timeout tests is possible, but is not another implementation task.

**US2:** No independent checkbox pair: reporter, counter, logger, shutdown ownership and reporting overhead/combined-mode proof must be delivered together. No tasks are marked `[P]`; shared files and real prerequisites rule out a useful parallel implementation split.

## Implementation Strategy

Deliver US1 first as the MVP, validating T001's runtime contract and T002's mutation execution behavior. Then add US2 using the same client, including combined acceptance and measurement in that unit. Ralph executes one focused unit within one story per iteration here; it must not combine work across story boundaries. Do not deploy or run live tests as part of completion.

## Requirement Ownership

| Requirements / outcomes | Owner |
|-------------------------|-------|
| FR-001–003: rate control, validation, shared allowance; SC-001 | T001 |
| FR-004–006: zero path, cancellation/deadlines, attempts/retries; SC-002–003, SC-006 | T001; reporting counts and combined overhead proof in T003 |
| FR-007–011: interval, logger, accounting, light overhead and lifetime; SC-003–004 | T003, including cross-mode/benchmark proof |
| FR-012–013: mutation behavior, output and exit preservation; SC-005 | T002, combined reporting coverage in T003 |
| FR-014: help, config guidance, README, generated CLI docs | T001 for pacing, T003 for reporting; T002 if a bounded human reason changes |
| FR-015: isolated acceptance and diagnostic matrix | Every unit; command compatibility in T002 and combined diagnostic lifecycle in T003 |

## Consolidation Review

Consolidated the fresh draft from **4 → 3 tasks**. Draft T004's combined-mode acceptance, benchmarks and completion evidence moved into T003; no work was removed. There were no checked tasks, progress records or external task-ID references to preserve. T001–T003 retain their IDs; draft T004 is superseded by T003.

- **Merged reporting and its performance/combined-mode proof:** `httpc/service.go` owns the wrapper, while `diagnostics_benchmark_test.go` and `cmd/root_api_diagnostics_test.go` already provide reusable fake/logging patterns. The former final task would reopen the same reporter and command fixtures to establish its own performance/output contract. Extending T002's fixtures and adding a focused benchmark fits the reporter unit without another production behavior or handoff.
- **Retained T001 → T002:** The strongest further merge would combine all US1 work. T001 already covers concurrency/timer admission, HTTP timeout error-type loss, two retry paths and configuration precedence. T002 adds a substantial, distinct command acceptance harness across four mutation paths, real-terminal confirmation eligibility, subprocess exits, partial envelopes and multi-resource diagnostic failures. `handleProcessInstanceMutationError`, `renderOpsPurgeAllProcessDefinitionsFailure`, and the existing definition-delete/purge terminal tests show that a transport unit test cannot prove those boundaries. Keeping these as two focused units avoids one oversized iteration; T001 still includes its own config/root/transport/retry tests and docs.
- **Retained T002 → T003:** T003 introduces a new concurrent reporter lifetime, configured logger contract and measurement-only mode. Its implementation depends on the shared pacing boundary and reuses the completed mutation fixtures. Folding this into T002 would mix two stories and exceed Ralph's permitted story scope. Shared config/service files are extended only for the newly delivered reporting behavior.
- **Kept thin wiring inside its behavior:** Flag/default/config mapping, transport traversal, retry callers/test doubles and docs stay with T001; interval mapping, service close, invocation context, root finalizer and logger tests stay with T003. No setup task, facade-only task or hypothetical parallel task is introduced.

Coverage review retained every FR-001–FR-015 and SC-001–SC-006 obligation, all isolated test/terminal/logger boundaries, and the dependency chain above. Only `tasks.md` was changed by consolidation; no application tests were run.

All tasks are initially unchecked. No implementation or runtime validation is claimed by this planning artifact.
