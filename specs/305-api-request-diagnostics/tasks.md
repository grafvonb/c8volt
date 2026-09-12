---
description: "Implementation tasks for issue #305 API request diagnostics"
---

# Tasks: Opt-in API Request Diagnostics

**Input**: Design documents from `specs/305-api-request-diagnostics/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/api-diagnostics.md](contracts/api-diagnostics.md), [quickstart.md](quickstart.md), and `.specify/memory/constitution.md`.

**Tests**: Required by the source specification and constitution. Write the behavioral tests before the corresponding implementation, confirm the intended failure, and keep implementation plus its passing tests in the same commit. Use the `TestAPIDiagnostics...` prefix for new acceptance tests so quickstart commands find them.

**Organization**: Tasks are grouped by user story. The shared foundation provides safe records before verbose/logger wiring is connected. Story-specific tests are independently runnable after their stated prerequisites. No task authorizes live mutations or generated-client edits.

## Format: `[ID] [P?] [Story] Description`

- `[P]` identifies independent files that may be worked on concurrently after the prerequisite phase is complete; exact groups appear below.
- `[US1]`, `[US2]`, `[US3]` refer to the corresponding stories in spec.md.
- All source paths are repository-relative. Run shell validation from the repository root.
- Keep all tasks unchecked until implemented and verified; generation does not count as execution.

## Path Conventions

Use existing `cmd/`, `internal/services/httpc/`, `internal/services/auth/`, `c8volt/`, `toolx/logging/`, and `testx/` ownership. Proposed new files are named explicitly below. Do not introduce another HTTP stack or change dependencies without revisiting the plan.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the working context and establish the existing behavior before changes.

- [x] T001 Confirm branch `codex/305-api-request-diagnostics` and review the feature artifacts and constitution; inspect shared HTTP/auth/bootstrap and existing terminal fixtures, run `go test ./internal/services/httpc ./internal/services/auth/... ./cmd -run 'ReadRetry|ProcessInstanceConfirmationTerminal|RootHelp' -count=1`, and record the baseline command/outcome or blockers in `specs/305-api-request-diagnostics/quickstart.md` without changing product code; if implementing through Ralph, first read `specs/ralph-implementation-rules.md` and surface any conflict.

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish safe immutable records and invocation ownership before attaching observation to existing verbose logging.

- [x] T002 Add record contract tests in `internal/services/httpc/diagnostics_record_test.go` for the field order, compact `api #` prefix, space-separated ASCII timing fields and `us` units, conditional quoting/escaping and one-line grammar and omission rules in `specs/305-api-request-diagnostics/contracts/api-diagnostics.md`; enforce “All durations and counts are nonnegative. Sequence is positive and unique per invocation.” and “Unknown fields are omitted, not filled with zero, empty strings or fabricated defaults.”
- [x] T003 Implement observation/snapshot types and deterministic safe formatting in `internal/services/httpc/diagnostics_record.go`, including optional status, reuse, total/headers/body, evidence-backed failure phase/reason, phase samples, context and separate request/response completion fields; enforce “HTTP error statuses retain their real status; transport error categories do not manufacture HTTP statuses.” and “A non-EOF body error can coexist with a real HTTP status and incomplete response evidence.”; make T002 pass.
- [x] T004 Add sanitizer unit tests in `internal/services/httpc/diagnostics_redaction_test.go` for URL userinfo/fragments, normalized/encoded/repeated query names, known-secret reflection, request and response cookies, allowed headers and typed errors; require correlation IDs “at most 256 characters”, response secrets collected before correlation formatting, and no body access.
- [x] T005 Implement centralized sanitization in `internal/services/httpc/diagnostics_redaction.go`: retain safe operational URL/query/context values; omit secret/payload-bearing or malformed metadata; seed known credentials from config and sensitive request/response metadata including parsed Set-Cookie; parse allowed Retry-After/Server-Timing fields and drop free-form descriptions; classify failure outcomes, phases and bounded safe reasons exactly as defined in the contract, preserving numeric HTTP status when a body fails; enforce “Secret matching/redaction applies before any string is serialized. Bodies and raw errors never enter a record.” and make T004 pass.
- [x] T006 Implement the invocation collector in `internal/services/httpc/diagnostics.go` with tests in `internal/services/httpc/diagnostics_test.go`: retain the existing invocation logger and resolved verbose setting, atomic sequence starting at 1, private context, state locks and once-only snapshot. Emit through `logging.InfoIfVerbose` after releasing state locks; reuse shared logger synchronization and formatting. No separate writer/logger, global destination, payload/history retention or observer when verbose is off or INFO is filtered.

**Checkpoint**: Record and redaction tests pass, collector state is invocation-owned, and no public invocation enables an unsafe/raw formatter. Run `go test ./internal/services/httpc -run 'TestAPIDiagnostics' -count=1` before proceeding.

## Phase 3: User Story 1 — Investigate a slow command (Priority: P1)

**Goal**: Enable diagnostic evidence for actual command requests with distinct final-header/body timing and separate explicit attempts.

**Independent Test**: Execute `get process-definition` through root bootstrap against a local server with controlled header/body delays; compare enabled/disabled results and request counts, and validate each emitted exchange. Authentication and redirects are exercised separately through real clients.

### Tests for User Story 1

- [x] T007 [P] [US1] Add transport contract tests in `internal/services/httpc/diagnostics_transport_test.go` for delayed final headers/body completion, informational responses that do not end headers, total=headers+body before rounding, absent headers/body on pre-header failure, known bodyless and empty responses, counted reads including `n > 0` with EOF, transport errors, supplied/custom base transports, redirects and disabled behavior; assert no extra requests/reads/closes and preserve existing trace callbacks.
- [x] T008 [P] [US1] Add OAuth diagnostic tests in `internal/services/auth/oauth2/diagnostics_test.go` for token fetch versus cache hit, one shared invocation sequence, correct timeout, no recursive authorization or new token retries, and absence of seeded client secrets/token contents.

### Implementation for User Story 1

- [x] T009 [US1] Implement transparent request/response observation wrappers in `internal/services/httpc/diagnostics_body.go`: count actual read bytes including bytes alongside errors, preserve original Read/Close values and ownership, retain GetBody/replay behavior and relevant optional body capabilities, and emit once on EOF/known bodyless response; enforce “Counts exclude headers/framing; decompressed response bytes may differ from wire transfer size.” and never infer transferred bytes from Content-Length.
- [x] T010 [US1] Implement `DiagnosticsTransport` and composed `httptrace` hooks in `internal/services/httpc/diagnostics.go`, using UTC start time and monotonic durations, final headers at delegate return, body interval from that return through observed termination, total across both intervals (body=0s when known bodyless), optional reuse and paired phase evidence; enforce “Absent hooks or unperformed phases are absent fields.” and freeze records against late callbacks; connect the body wrappers and safe formatter from T003–T009.
- [x] T011 [US1] Add `WithDiagnostics` and shared collector lookup/wrapping helpers in `internal/services/httpc/service.go`, integrating with `internal/services/httpc/round_trippers.go` so the enabled stack is Auth → ReadRetry → Log → Diagnostics → base; preserve existing logging/activity, retry/auth wrapper behavior and default transports, and pass T007 without changes to `internal/services/httpc/read_retry.go` policy.
- [x] T012 [US1] Attach the existing invocation collector to the separate unauthenticated token client in `internal/services/auth/oauth2/service.go` using the shared httpc helper, preserving its timeout and no-auth/no-retry semantics; make T008 pass and confirm cookie authentication continues using the supplied shared client.
- [x] T013 [US1] Reuse resolved verbose configuration and the existing invocation logger in `cmd/root.go` and pass diagnostic options through `cmd/root_services.go` before authenticator initialization. Preserve configuration precedence and omit unset context. Add no flag, logger or config setting, enable no debug/body dumps, and add no diagnostic logic to individual commands, facades or generated clients.
- [x] T014 [US1] Add `TestAPIDiagnosticsCommand...` basic read/bootstrap coverage in `cmd/root_api_diagnostics_test.go` using deterministic local HTTP fixtures and fresh command state: exercise `get process-definition`, before/after-command flag inheritance, token/cookie/auth-none bootstrap traffic, no-exchange/help paths, and enabled/disabled byte-identical stdout and equal request counts; verify existing verbose help and update examples in `cmd/root_test.go` as needed; test verbose-off, debug-only, quiet and restrictive INFO filtering.
- [x] T015 [US1] Run `go test ./internal/services/httpc ./internal/services/auth/oauth2 ./cmd -run 'TestAPIDiagnostics|RootHelp' -count=1`, verify tests are actually discovered, fix failures within the US1-owned files, and record the commands/outcomes in `specs/305-api-request-diagnostics/quickstart.md`.

**Checkpoint**: The read-only timing workflow is demonstrable with the safe foundation; token exchanges and explicit attempts are visible without added requests. This is the development MVP, not permission to skip the remaining required stories or release gates.

## Phase 4: User Story 2 — Use diagnostics safely in operational workflows (Priority: P1)

**Goal**: Preserve stdout, quiet semantics, prompts, mutations and effective stderr routing while collecting safe evidence.

**Independent Test**: Execute both `get process-definition` and `cancel process-instance` through real root/bootstrap paths against reset local fixtures, capture stdout/stderr separately, and compare output, exit codes, prompts and requests across supported modes.

### Tests for User Story 2

- [x] T016 [P] [US2] Extend `cmd/root_api_diagnostics_test.go` with plain/plain-time/text/JSON log formats with timestamp/source settings, normal/JSON/keys-only/quiet and quiet+machine result-output matrices for reads and cancellations, inherited root versus configured child stderr, explicit-key and selector flows, auto-confirm/automation/no-wait, dry-run where supported, sparse pages, empty scope and failures; decode exactly one JSON envelope then require EOF, assert exact human/keys output including zero-byte empty keys, and compare request bodies/counts, mutation/polling outcomes and prompt absence against disabled fixtures; assert INFO diagnostics are absent under quiet/restrictive levels and required message fields survive every log format.
- [x] T017 [P] [US2] Extend `cmd/cmd_confirmation_terminal_test.go` with `TestAPIDiagnosticsTerminal...` cases using `testx.NewCmdTerminalRunner`, real terminal stdin and separately captured streams: accepted/aborted cancellation, configured/inherited stderr, quiet/machine combinations, prompt-free empty scopes and activity coexistence; retain exact prompt wording/defaults/EOF behavior and verify aborted/empty selections submit no mutations.
- [x] T018 [P] [US2] Extend adversarial sanitizer coverage and add fuzz seeds in `internal/services/httpc/diagnostics_redaction_test.go` for signed-URL families, encoded/repeated names and values, known secrets reflected under safe query/header/context fields, Set-Cookie reflected in X-Request-ID, malformed metadata, arbitrary error payloads, control injection, and safe identifier retention; validate the output against `specs/305-api-request-diagnostics/contracts/api-diagnostics.md` without reading bodies for redaction.

### Implementation for User Story 2

- [x] T019 [US2] Preserve effective child/inherited `cmd.ErrOrStderr()` in `cmd/root.go` before activity wrapping and create the existing configured logger on that writer. Pass that logger through `cmd/root_services.go` before authentication. Respect quiet/log-level/log-format behavior, preserve stdout, leave prompts unprefixed on configured stderr, and prevent stale invocation destinations. Make T016–T017 pass without changing individual result renderers or prompt policy.
- [x] T020 [US2] Complete security integration in `internal/services/httpc/diagnostics_redaction.go` and `internal/services/auth/oauth2/diagnostics_test.go`, adding cookie-login reflection coverage in `internal/services/auth/cookie/diagnostics_test.go`; verify config/request/response secrets are available before allowed-field formatting, auth bootstrap does not leak them and arbitrary raw error/header/body output is absent; make T018 and auth security regressions pass.
- [x] T021 [US2] Run `go test ./internal/services/httpc ./internal/services/auth/... ./cmd -run 'TestAPIDiagnostics|ProcessInstanceConfirmationTerminal' -count=1`, verify supported terminal cases really execute and no mutations reach a live service, and record results or explicit platform limitations in `specs/305-api-request-diagnostics/quickstart.md`.

**Checkpoint**: The read and mutation execution matrices preserve operation behavior and clean stdout; admitted records reach intended stderr and quiet suppresses INFO diagnostics, and seeded credentials/payloads are excluded.

## Phase 5: User Story 3 — Interpret failures and partial evidence accurately (Priority: P2)

**Goal**: Preserve truthful evidence under reuse, concurrent phases, retries, timeouts, cancellation, incomplete bodies and independent invocations.

**Independent Test**: Drive controlled real and fake transports through failure/reuse/partial/concurrent cases, inspect terminal records and original call results, then run the race detector and client-version wiring checks.

### Tests for User Story 3

- [x] T022 [P] [US3] Add lifecycle edge tests in `internal/services/httpc/diagnostics_body_test.go` for early close, read and close errors, repeated close after EOF, decompressed bodies, interrupted request uploads, `n > 0` with non-EOF errors, timeout/cancellation before headers and during body reading, and an abandoned body; require “A request-body close must preserve Close behavior but is not proof the whole body was uploaded.” and no fabricated completion, extra drain, timer, read or close.
- [ ] T023 [P] [US3] Add `TestAPIDiagnostics...` trace/concurrency tests in `internal/services/httpc/diagnostics_concurrency_test.go` for reused connections, overlapping connect starts/ends, late trace callbacks, existing callback composition, reordered exchange completion, concurrent read/close observations, two collectors/writers and writer failures; enforce “Samples are ordered by their start time, not callback completion order.” and “Unpaired/in-progress samples are omitted.” with no duplicate/interleaved lines.
- [ ] T024 [P] [US3] Add retry/redirect regressions in `internal/services/httpc/diagnostics_retry_test.go` and supported-client wiring coverage in `c8volt/diagnostics_test.go`: preserve GET/HEAD and higher-level mutation attempt policies, count boundary invocations, mark undrained retry responses incomplete, and exercise representative Camunda 8.7–8.10, Operate and Tasklist paths through supplied instrumented clients without modifying generated clients or claiming visibility into transport-internal sends.

### Implementation for User Story 3

- [ ] T025 [US3] Complete phase matching and terminal-state synchronization in `internal/services/httpc/diagnostics.go`: retain all completed DNS/TCP/TLS samples with scalar or list formatting and omit non-TCP connect samples, match overlapping network/address attempts without exposing extra address fields, distinguish true zero from absent phase observations, preserve reuse evidence and existing hooks, and enforce “Samples may overlap and must not be summed into a supposed breakdown of total duration.”; pass T023 trace cases.
- [ ] T026 [US3] Complete failure/partial-transfer finalization in `internal/services/httpc/diagnostics_body.go` and safe error classification in `internal/services/httpc/diagnostics_redaction.go`, using typed errors and trace evidence for failure phase/reason (omit ambiguous phases; early Close alone is not an error), retaining actual byte counts at snapshot time, distinguishing request and response completion and preserving original Read/Close errors; enforce “Once frozen, later Close calls and trace callbacks cannot emit another record or change the snapshot.” and make T022/T024 pass.
- [ ] T027 [US3] Finish writer-failure and invocation-isolation handling in `internal/services/httpc/diagnostics.go` and reset/subprocess coverage in `cmd/root_api_diagnostics_test.go`: format a safe message and invoke the existing logger once after releasing state locks, relying on shared logging synchronization, never replace HTTP/body outcomes or retry/fall back to stdout, and verify separate subprocess invocations own separate destinations/sequences rather than concurrently executing the singleton Cobra root; pass the remaining T023 isolation cases.
- [ ] T028 [US3] Run `go test ./internal/services/httpc ./internal/services/auth/... ./c8volt ./cmd -run 'APIDiagnostics|ReadRetry|ProcessInstanceConfirmationTerminal|RootHelp' -race -count=1`, verify version/auth/concurrency cases are discovered, resolve failures in the affected files and record results in `specs/305-api-request-diagnostics/quickstart.md`.

**Checkpoint**: Failures, partial transfers and concurrent exchanges remain accurate and race-free; no measurement or lifecycle behavior is invented.

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Align operator documentation, quantify instrumentation costs and complete delivery gates.

- [ ] T029 Update `README.md` and command metadata/examples in `cmd/root.go` for existing verbose diagnostics, ASCII messages, logger formats/timestamps/source, configured stderr, quiet suppression, final-header versus total, phase overlap/reuse, observed body bytes/partial transfers, retained identifiers/redaction, synchronous writer backpressure and the transport observation boundary; align `specs/305-api-request-diagnostics/quickstart.md` with the implemented commands and test names.
- [ ] T030 Run `make docs-content` from `Makefile`, inspect regenerated `docs/cli/c8volt.md` and inherited-flag pages plus the generated README homepage for correct output guidance, and fix source metadata if needed rather than hand-editing generated CLI documents.
- [ ] T031 Add enabled/disabled small-body and streaming-body benchmarks in `internal/services/httpc/diagnostics_benchmark_test.go`, run `go test ./internal/services/httpc -run '^$' -bench APIDiagnostics -benchmem`, confirm memory is independent of payload size with no history/queue/buffering, and record measured outcomes in `specs/305-api-request-diagnostics/quickstart.md` without inventing an unsupported latency threshold.
- [ ] T032 Run gofmt on touched Go files, execute the automated validation scenarios in `specs/305-api-request-diagnostics/quickstart.md`, run full `make test` and `git diff --check` after final source/doc changes, and review the diff against FR-001–FR-011 for layering, no extra requests, no generated-client edits and preserved operation results; record actual outcomes/limitations in `specs/305-api-request-diagnostics/quickstart.md` before any Conventional Commit referencing #305.

## Dependencies & Execution Order

### Phase Dependencies

```text
T001 Setup
  → T002–T006 Foundation
    → T007–T015 US1: observable read workflow
      → T016–T021 US2: operational safety/streams
        → T022–T028 US3: partial/failure/concurrency evidence
          → T029–T032 Documentation, benchmarks and full validation
```

- Setup precedes all implementation; foundation must pass before story work.
- Within foundation, T002 → T003 → T004 → T005 → T006 provides an explicit tested record/sanitization contract before collector use.
- US1 tests T007 and T008 may be authored together after foundation; T009 → T010 → T011 → T012 → T013 → T014 → T015 is the integration order.
- US2 T016/T017/T018 may be authored together after US1. T019 follows stream tests and T020 follows adversarial tests; T021 follows both implementations. Keep edits to root/bootstrap and sanitizer/auth files sequential as listed unless file ownership is explicitly separated.
- US3 T022/T023/T024 may be authored together after US2. T025 → T026 → T027 → T028 completes the shared implementation without concurrent edits to the same files.
- Polish follows all stories. T030 depends on T029; T032 depends on T029–T031 and all preceding checks.

### User Story Dependencies

US1 provides the common transport/bootstrap path. US2 validates and completes operational integration on that path; US3 completes edge/concurrency behavior on the same shared observer. These stories have distinct independently runnable acceptance scenarios, but their implementation is deliberately sequential because they share source files. Do not claim full story-level parallelism or remove required tests to make the graph look independent.

### Parallel Opportunities

Only the following task groups are marked `[P]`; they operate on separate files after their phase prerequisites:

- US1: T007 transport tests and T008 OAuth tests.
- US2: T016 command tests, T017 terminal tests and T018 redaction tests.
- US3: T022 lifecycle tests, T023 concurrency tests and T024 retry/version tests.

Test authoring may happen concurrently; fixture-driven test execution against the global root must follow repository reset/subprocess rules. Implementation tasks editing shared files remain sequential.

## Parallel Example: User Story 1

```text
After T006:
T007 → internal/services/httpc/diagnostics_transport_test.go
T008 → internal/services/auth/oauth2/diagnostics_test.go
Then integrate T009–T015 sequentially.
```

## Parallel Example: User Story 2

```text
After T015:
T016 → cmd/root_api_diagnostics_test.go
T017 → cmd/cmd_confirmation_terminal_test.go
T018 → internal/services/httpc/diagnostics_redaction_test.go
Then complete T019–T021.
```

## Parallel Example: User Story 3

```text
After T021:
T022 → internal/services/httpc/diagnostics_body_test.go
T023 → internal/services/httpc/diagnostics_concurrency_test.go
T024 → internal/services/httpc/diagnostics_retry_test.go + c8volt/diagnostics_test.go
Then complete T025–T028.
```

## Implementation Strategy

### MVP First

Complete setup, safe foundation and US1 for a controlled read-only development demonstration. This includes redaction from the start; do not temporarily expose raw URLs/headers/errors or body dumps. US2 and US3 remain mandatory for issue #305 completion, and no release is ready until full validation/docs gates pass.

### Incremental Delivery

1. Establish safe record/collector primitives and their tests.
2. Integrate real exchange observation, OAuth coverage and the existing verbose/logger wiring; validate US1.
3. Complete effective stderr, quiet and mutation/terminal matrices; validate US2.
4. Complete lifecycle/trace/concurrency and supported-client regressions; validate US3.
5. Regenerate documentation, measure overhead and pass `make test`.

### Requirement Traceability

| Requirement | Tasks |
| --- | --- |
| FR-001 existing verbose opt-in/level filtering | T011, T013–T014 |
| FR-002 per-exchange records/separate retries | T006–T012, T024, T026 |
| FR-003 identity/context/outcomes/sizes/correlation | T002–T006, T009–T014, T020 |
| FR-004 distinct timings/observed phases | T007, T010, T023, T025 |
| FR-005 missing/partial evidence | T002–T003, T009, T022, T026 |
| FR-006 stderr/quiet/stdout/exit preservation | T014, T016–T017, T019, T027 |
| FR-007 secret exclusion/identifier retention | T004–T005, T008, T018, T020 |
| FR-008 safe errors/headers/no payloads | T004–T005, T018, T020, T026 |
| FR-009 semantic/request/response preservation | T007–T012, T016–T017, T022, T024, T026–T028 |
| FR-010 concurrency/invocation isolation | T006, T023, T025, T027–T028 |
| FR-011 help/documentation | T014, T029–T030, T032 |

SC-001/SC-002 are proven by US1 and US3; SC-003/SC-004 by US2; SC-005 by retry, mutation and concurrency coverage. T032 is the final aggregate gate.

## Notes

- Use realistic HTTP and terminal fixtures already in `testx/`; avoid a new fixture framework or live mutations.
- Preserve Spec Kit memory and existing feature artifacts. No public facade result-schema changes, additional requests, synthetic diagnostics, resource cleanup or monitoring integration belong to this feature.
- Keep commits small and use Conventional Commits with #305 in the subject. Run the closest relevant checks and full `make test` before committing, as required by the constitution; never commit deliberately failing test-only work.
- Optional extension hooks are separate actions; task generation does not execute commits or start Ralph.
