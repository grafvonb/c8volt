# Tasks: Explain Known Connection and Authentication Failures

**Input**: Design artifacts in `specs/381-explain-connection-failures/`.

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/human-errors.md](contracts/human-errors.md), [quickstart.md](quickstart.md), repository `AGENTS.md`, constitution, and `specs/ralph-implementation-rules.md`.

**Tests**: Required by spec FR-010. Each checkbox includes its fixture/test changes, any production adaptation, formatting, and targeted validation. Within a work unit, write or strengthen acceptance assertions first, confirm a new regression fails for its intended reason, then implement and rerun the relevant checks. Already-passing compatibility checks need not be forced to fail. Do not leave a separate failing-test task behind.

**Organization**: One cohesive work unit per user story. US1 implements the complete shared policy once; US2 and US3 extend distinct acceptance coverage using the same implementation. Required baseline safety checks belong to US1, not a later repair task.

## Format: `[ID] [P?] [Story] Description`

Every executable task is an unchecked `Tnnn` checkbox with one story label and concrete repository-relative paths. No `[P]` marker is used: shared command/test state and dependencies require sequential execution. Numbered steps under a checkbox belong to that same work unit.

## Path Conventions

Production rendering and CLI metadata live in `cmd/`; tests stay alongside them. Only the two planned rendering files own runtime changes. `cmd/root.go` may receive concise help prose. Existing service/facade/error/transport contracts remain reference-only. Generated docs under `docs/cli/` and `docs/index.md` come from `make docs-content`.

## Work-Unit Responsibilities

- Read active context and inspect the nearest code before editing. Keep branch `381-explain-connection-failures` based on `10bd90ae`; do not import #379, rename branches, add diagnostics frameworks, change services/facades/error conversion, migrate unrelated callers, or alter execution modes.
- Preserve original errors for classification, exit resolution, and machine evidence. Human diagnostics use the configured command-context logger; stdout remains selected-format results and prompts remain plain configured stderr.
- Use only isolated typed failures, child-process fake transports, and test-owned HTTP servers. Before running a selector, inspect its dependencies, clear inherited target/profile/credential/helper variables using `isolateRequestRateEnvironment`, and prevent inherited `GOFLAGS`/build tags from enabling live tests. Fakes reject unexpected targets; never discover, provision, probe, deploy to, or mutate a running environment.
- Every new or materially changed test function needs a concise behavior comment. Do not use `t.Parallel()` with shared Cobra flags, environment, or default transport state; confine fake default-transport replacement to its subprocess.
- Trace changed failures through service → facade conversion → human renderer → logger → exit; review direct prints and raw-error interpolation at output boundaries. Never weaken logger, redaction, partial-result, or exit assertions to accept a regression.
- Run `gofmt` on touched Go files and the closest relevant checks before completing a checkbox. Record actual evidence with the work unit in `specs/381-explain-connection-failures/progress.md`; context reading and evidence recording are not separate tasks. Under Ralph, follow its commit step with a Conventional Commit subject ending in ` #381`.
- Do not run application tests for generation/consolidation of this document. During implementation, broaden beyond focused selectors only for a concrete wider diff or unresolved regression; no automatic full-suite or live-engine gate.

## Phase 1: Setup (Shared Infrastructure)

No initialization work is required. Go, Cobra, the logger, typed failure annotations, subprocess runners, and fake-server helpers already exist. Context loading and branch checks are responsibilities above rather than artificial setup tasks.

## Phase 2: Foundational (Blocking Prerequisites)

No new model, service contract, storage, dependency, or test framework is needed. Reuse `commandConnectionFailureDetails`, `httpc.FailureDetails`, configured logging, existing result envelopes, and `testx` subprocess helpers. There is no foundational checkbox blocking US1.

## Phase 3: User Story 1 - Understand a Failed Connection Immediately (Priority: P1) — MVP

**Goal**: Deliver all existing recognized causes through shared human formatting, preserving precedence, unknown fallback, and connection-test-specific advice.

**Independent Test**: A real process-definition search command with an isolated OAuth connection refusal emits the exact actionable message through its configured logger with unchanged class/exit; a formatter table covers every supported cause without using raw text.

- [x] T001 [US1] Deliver complete bounded connection/authentication explanations in `cmd/cmd_views_contract.go` and `cmd/cmd_views_config_test_connection.go`, with regression tests in `cmd/cmd_views_contract_test.go`, `cmd/get_processdefinition_error_envelope_test.go`, and `cmd/config_test_connection_diagnostics_test.go`; adapt directly affected exact expectations in `cmd/delete_processdefinition_unfinished_test.go`, update `cmd/root.go`, `README.md`, and `docs/configuration.md`, regenerate `docs/index.md` and `docs/cli/`, and pass the focused checks below.

  1. Add `TestBoundedCommandFailureReasons` before changing the formatter, using the contract's exact reasons. Cover raw/wrapped/joined and retained normalized evidence: refusal, reset/unexpected EOF, deadline/network timeout, DNS, certificate validation, malformed response, status cases including authentication 400/404, generic 401/403/404/429/5xx/default status, authentication-only failure, cancellation, unknown/class-only fallback, and misleading plain text. Include mixed request-rate plus connection evidence and current malformed-response/joined-error precedence. Test generic endpoint advice separately from connection-test topology advice.
  2. Preserve these [data-model.md](data-model.md) constraints verbatim: HTTP status “zero means absent, not a fabricated response”; connection reason “never derived from arbitrary text”; request-rate evidence “Higher-priority local unsent evidence; earlier work may have been submitted”; error class “explanation does not change class”; cancellation “Generic recognition declines; shared class fallback and existing cancellation exit semantics remain unchanged”. Preserve category constraint “preserve source precedence” and authentication constraint “distinguishes token/login failures from gateway failures” without inventing cross-branch evidence correlation.
  3. Extend the existing process-definition subprocess helper for the original OAuth token refusal, using a child-only fake default transport returning typed refusal and refusing every unexpected target. Use temporary explicit config, zero retries, and `--no-version-check` only for this controlled scenario. Count one failed authentication attempt and zero gateway requests, and assert `search process definitions: authentication service connection refused; check the configured host and port and that the service is running`. Reuse configured/inherited writers and existing log parsers; include the baseline normal/verbose/quiet, plain/structured, JSON, and DEBUG checks necessary for this output change.
  4. Extract one private `(reason, recognized)` connection-reason function in the existing connection view, retaining all recognition and wording together. Let `boundedCommandFailureReason` return the exact request-rate reason first, then delegate recognized causes, then retain its original class fallback. Preserve generic cancellation behavior and config-specific cancellation/setup/class-only fallbacks. Use only a tiny explicit caller context for topology-versus-generic gateway 404. Do not duplicate switches, parse error strings, or broaden normalization to recover lost bare transport types.
  5. Keep `handleCommandErrorWithMode`, JSON envelope creation, callers, and original error objects unchanged. Preserve connection-test exact wording. Update existing assertions only where the newly recognized human suffix legitimately changes; preserve aggregate prefixes/counts and machine evidence. Run the existing batch regression now so US1 does not leave a known failing expectation for a later story.
  6. Explain ordinary known-cause guidance without DEBUG in README and configuration diagnostics docs, add one concise root `Long` help sentence, and run `GOENV=off GOFLAGS= make docs-content`. Review generated output; do not hand-edit generated CLI Markdown or build the entire site.
  7. Run `gofmt` on touched Go files, confirm the new test selector matches, then run `GOENV=off GOFLAGS= go test ./cmd -run '^(TestBoundedCommandFailureReasons|TestConfigConnectionFailureReasons|TestConfigConnectionDiagnosticLogging|TestCommandErrorEnvelopeProcessDefinition|TestGetProcessDefinitionLifecycleSearchDiagnostics|TestDeleteProcessDefinitionUnfinishedDiagnosticLogging)$' -count=1`. Assert ERROR severity/destination and no bypass/duplicate, human stdout empty, normal/verbose redaction on both streams and logger sinks, original JSON detail with one envelope and EOF, unchanged exits, and baseline request counts. Finish with `git diff --check` and the output-boundary review.

**Checkpoint**: US1 is a usable MVP with required baseline logger safety and documentation. No production rewrite is needed for later stories.

## Phase 4: User Story 2 - Receive Safe Diagnostics Through the Configured Logger (Priority: P1)

**Goal**: Complete explicit safety coverage for repeated failures and all logging/control combinations, extending reused fixtures rather than changing shared policy again.

**Independent Test**: Test-owned repeated failures with distinctive credentials, URL, method, body, cursor, resource key, and forged-line markers yield exactly one bounded final ERROR at the selected logger destination in normal/verbose/quiet modes, while existing redacted DEBUG remains available independently of verbose.

- [x] T002 [US2] Strengthen the command diagnostic safety matrix in `cmd/get_processdefinition_error_envelope_test.go`, `cmd/config_test_connection_diagnostics_test.go`, and `cmd/delete_processdefinition_unfinished_test.go`, reusing record validation from `cmd/update_job_batch_logging_test.go`, and pass focused logger/redaction checks without adding a new production diagnostic path.

  1. Start from T001's working formatter and existing regression assertions. Identify uncovered combinations before adding cases: configured and inherited writer × plain/structured logger × ordinary/verbose/quiet × DEBUG flag/config, including quiet overriding DEBUG and ERROR remaining available at WARN level. Add only missing assertions/cases; do not replace existing fixtures or create a second formatting policy.
  2. In the multi-definition fixture, exercise repeated same-cause backend failures and partial success with distinctive unsafe evidence. Capture stdout, process stderr, and a separate configured logger destination where applicable; verify that a direct stderr bypass fails when the logger is redirected. Assert one complete final ERROR, existing bounded aggregate prefix and selected/accepted counts, no per-key dump, no duplicate final error, and zero human stdout bytes.
  3. Check forbidden raw markers on both output streams and any separate logger capture in normal and verbose human modes. Quiet retains the compact ERROR while suppressing INFO. Under both DEBUG activation paths, assert existing safe exchange evidence and redaction; verbose alone enables none. Retain the search fixture's dedicated `plain` parser because `bulkCompletionDiagnosticRecords` supports `plain-time` and `json` only.
  4. Retain JSON and partial JSON assertions within the same reused multi-resource fixture: exactly one envelope followed by EOF, original detail/evidence and class, successful/failed item counts, unchanged exit and mutation/request counts, and no extra final human ERROR. Do not scrub established machine evidence to satisfy human-mode safety assertions. Preserve prompts/activity behavior and the existing automatic confirmation settings.
  5. Run `gofmt` on touched tests and `GOENV=off GOFLAGS= go test ./cmd -run '^(TestGetProcessDefinitionLifecycleSearchDiagnostics|TestConfigConnectionDiagnosticLogging|TestDeleteProcessDefinitionUnfinishedDiagnosticLogging)$' -count=1`. Verify the assertions would reject direct diagnostic writes or a raw joined error even if it contains the operation prefix. Run `git diff --check` and record only the new coverage and actual check outcomes.

**Checkpoint**: US2 adds adversarial execution coverage over the same implementation. It must not reopen production formatting, facade normalization, or transport behavior.

## Phase 5: User Story 3 - Preserve Operational and Automation Contracts (Priority: P2)

**Goal**: Lock down execution-level compatibility for request-rate waits, cancellation, output controls, exit suppression, machine evidence, and successful results.

**Independent Test**: Existing fake-backed workflows preserve baseline request/retry counts, submission/confirmation semantics, class/exit behavior, JSON evidence and partial payloads, and successful selected-format results after the human explanation changes.

- [x] T003 [US3] Complete focused compatibility cases in `cmd/request_rate_mutation_test.go`, `cmd/get_processdefinition_error_envelope_test.go`, `cmd/config_test_connection_diagnostics_test.go`, and `cmd/root_api_diagnostics_test.go`, retaining shared precedence coverage in `cmd/cmd_views_contract_test.go` and documenting measured results in `specs/381-explain-connection-failures/progress.md`.

  1. Reuse `TestRequestRateMutationLocalFailure`, `TestRequestRateMutationInterruption`, `TestRequestRateCancellationConfirmation`, and `TestRequestRateReportingPartialFailure`. Preserve original unsent/possibly-submitted distinctions, existing cancellation and confirmation semantics, partial payload, failure exit, request/retry counts, and prompt routing. Add missing combined-evidence execution checks only where current coverage does not prove the contract; do not introduce a new lifecycle or alter existing timing/retry policy.
  2. Complete supported human/JSON/quiet+JSON and `--no-err-codes` combinations in the original refusal and existing command fixtures. Assert class/outcome/detail and partial payload against baseline expectations, one envelope plus EOF, and no second human error renderer. Retain connection-test failure policy rather than expanding its JSON support.
  3. Retain successful fake-backed read/connection scenarios and `TestAPIDiagnosticsCommandReadPreservesResultsAndRequests` to show identical selected-format results and request counts. Preserve ordinary generic cancellation fallback, exact connection-test cancellation/topology advice, unknown fallback, and command-specific diagnostic overrides. Reuse existing prompt tests with real terminal stdin when exercising confirmation; do not add a mocked terminal shortcut or change prompt eligibility.
  4. Run `gofmt` on touched tests and `GOENV=off GOFLAGS= go test ./cmd -run '^(TestRequestRateMutationLocalFailure|TestRequestRateMutationInterruption|TestRequestRateCancellationConfirmation|TestRequestRateReportingPartialFailure|TestAPIDiagnosticsCommandReadPreservesResultsAndRequests|TestCommandErrorEnvelopeProcessDefinition|TestConfigConnectionDiagnosticLogging|TestBoundedCommandFailureReasons)$' -count=1`. Also execute any newly named tests explicitly after confirming selector matches. Run `git diff --check`, review the preserved error→logger→exit path, and record limitations without claiming a live-engine pass.

**Checkpoint**: All three stories satisfy [contracts/human-errors.md](contracts/human-errors.md); remaining runtime work is zero once the three checkboxes and their validation pass.

## Phase 6: Polish & Cross-Cutting Concerns

No separate polish checkbox is needed. Documentation, generation, formatting, safety review, and progress evidence are included in their owning work units. Do not add repository-wide cleanup, benchmarks, deployments, live checks, or a redundant final test pass solely to close the feature.

## Dependencies & Execution Order

```text
Existing setup and typed-evidence/logger infrastructure
  → T001 [US1] complete shared formatter + original regression + docs
  → T002 [US2] complete adversarial logger/multi-resource coverage
  → T003 [US3] complete execution/automation compatibility coverage
```

T002 and T003 consume T001's final wording and reuse overlapping command fixtures; serialize them to avoid conflicting expectations or helper state. T003 follows T002 so its acceptance evidence uses the finalized safety matrix. The graph is acyclic. No task depends on a later task, live target, or nonexistent new interface.

### Requirement Ownership

| Requirement / criterion | Owner | Coverage |
| --- | --- | --- |
| FR-001–FR-003, SC-001 | T001 | Complete typed-cause mapping, original authentication refusal, unknown fallback |
| FR-004 | T001 + T003 | Shared request-rate/cancellation precedence plus execution compatibility |
| FR-005 | T001 + T002 + T003 | Prefix/aggregate and config-specific wording retained, batch and override checks |
| FR-006–FR-007, SC-002–SC-003 | T001 baseline; T002 complete matrix | Logger severity/destination, both streams, repeated failures, redaction, both DEBUG paths |
| FR-008, SC-004 | T001 baseline; T002 partial batch; T003 exit/mode matrix | One envelope and EOF, original machine evidence, partial results, class and exits |
| FR-009, SC-005 | T001 + T002 baseline counts; T003 compatibility | No extra requests/retries, unchanged confirmation/submission and success behavior |
| FR-010 | T001 + T002 + T003 | Required isolated unit and command-execution checks integrated with each outcome |
| FR-011 | T001 | README, configuration guidance, root metadata, generated CLI docs |
| Data model / contract | T001 shared behavior; T002/T003 acceptance | No new entity/schema/state; typed bounded facts and existing machine/error objects |

## Parallel Execution Examples

- **US1**: No parallel task example applies. Inside T001, formatter assertions precede production changes; existing command fixtures and documentation validate that same change. Keep them one compiling outcome.
- **US2**: No parallel task example applies. The destination, repeated-failure, DEBUG, and partial-result cases share subprocess/configuration helpers and the final T001 wording; run one isolated matrix.
- **US3**: No parallel task example applies. Request-rate/cancellation and output-control fixtures share environment and Cobra state; run them sequentially. Do not introduce `t.Parallel()` or extra tasks merely to enable hypothetical parallel work.

## Implementation Strategy

**MVP first**: Complete T001 with the complete supported cause set, original failure regression, required safety checks, and documentation. This is the smallest usable behavior increment; later stories do not rebuild it.

**Incremental delivery**: T002 strengthens adversarial diagnostic evidence; T003 completes operational/automation compatibility evidence. Both preserve T001's implementation and each leaves relevant checks passing. Under Ralph, execute only the current story work unit; do not start the next story or a separate polish section in the same iteration.

**Validation scope**: Follow [quickstart.md](quickstart.md) using isolated fixtures only. Task generation/consolidation validates documents, IDs, paths, links, coverage, dependencies, and progress; application tests run later with implementation. Broaden implementation testing only when concrete changes or failures require it, and report unverified real-engine behavior as a coverage limitation.

**Counts**: 3 tasks; US1: 1, US2: 1, US3: 1; setup/foundational/polish: 0; parallel markers: 0. All tasks are initially unchecked.

**Consolidation**: Mandatory review retained the three story work units with no merges; see [consolidation-review.md](consolidation-review.md). Existing mechanisms, fixtures, and command handlers are reused rather than estimated as new implementation.
