# Tasks: API Latency Diagnostics

**Input**: Design documents from `specs/289-api-latency-diagnostics/`
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/api-latency-cli.md](contracts/api-latency-cli.md), [quickstart.md](quickstart.md)
**Mandatory Ralph Context**: Every Ralph iteration MUST be launched with `--implementation-context specs/ralph-implementation-rules.md` and must apply that file before implementation.
**Issue Commit Rule**: Every commit subject for this feature MUST use Conventional Commits and end with `#289`.

**Tests**: Tests are required by the feature specification, repository constitution, and Ralph implementation rules. Test tasks precede their implementation tasks and must fail for the intended reason before implementation begins.

**Organization**: Tasks are grouped by user story so each increment remains independently testable. Each Ralph iteration completes only the first incomplete work unit and updates `specs/289-api-latency-diagnostics/progress.md` in the same work-unit commit.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Capture mandatory implementation context and verify the exact repository patterns before code changes.

- [x] T001 Create `specs/289-api-latency-diagnostics/progress.md` with the Ralph context path, issue `#289`, branch, validation log, and codebase-pattern sections
- [x] T002 Inspect the nearest ops analyse/execute commands, facade/service seams, `toolx/pool`, SimpleUserTask fixtures, version capabilities, progress helpers, shared report helpers, command contracts, and integration suites; record only reusable findings in `specs/289-api-latency-diagnostics/progress.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared version-neutral model, deterministic planner/statistics engine, and thin service/facade contracts required by every story.

**CRITICAL**: No user story implementation begins until this phase passes focused validation.

### Tests for Foundational Behavior

- [x] T003 [P] Add failing tests for stage ramps, minimum sample budgets, deterministic allocation, derived limits, nearest-rank percentiles, throughput, zero-baseline deltas, safe classifications, and finding order in `internal/services/ops/api_latency_test.go`
- [x] T004 [P] Add failing facade contract tests for API-latency request/result conversion, defensive slice copying, partial-result mapping, and domain error conversion in `c8volt/ops/client_test.go`

### Implementation for Foundational Behavior

- [x] T005 Define version-neutral request, plan, measurement, stage, finding, topology, ownership, visibility, cleanup, context, and result types with stable enums in `internal/domain/ops_api_latency.go`
- [x] T006 Extend the existing ops service interface with `AnalyseAPILatency` and `ExecuteAPILatencyTest` contracts in `internal/services/ops/api.go`
- [x] T007 Implement deterministic stage planning, normalized-backoff visibility attempt bounding, closed-loop accounting, safe error classification, statistics, comparisons, and finding evaluation in `internal/services/ops/api_latency.go`
- [x] T008 Define the matching public API-latency request/result models and intentional JSON tags in `c8volt/ops/model.go`
- [x] T009 Extend the public ops API with the two API-latency methods in `c8volt/ops/api.go`
- [x] T010 Implement mechanical domain/public conversions with defensive collection copying in `c8volt/ops/convert.go`
- [x] T011 Implement thin facade delegation, partial-result mapping, option propagation, and `ferrors.FromDomain` conversion in `c8volt/ops/client.go`
- [x] T012 Verify the existing `NewWithAnalysisDependencies` construction supplies all required services without new client wiring and add a regression assertion in `c8volt/client_test.go`
- [x] T013 Run `go test ./internal/services/ops ./c8volt/ops -run 'APILatency' -count=1` and record the passing foundational validation in `specs/289-api-latency-diagnostics/progress.md`

**Checkpoint**: Shared API-latency planning, models, service contracts, and facade seams are available without adding a report, worker, fixture, generated-client, or version-adapter abstraction.

---

## Phase 3: User Story 1 - Diagnose Read Latency Safely (Priority: P1) MVP

**Goal**: Deliver `c8volt ops analyse api-latency` as a strictly read-only staged comparison of topology, process-definition search/read, and process-instance search/read paths.

**Independent Test**: Run the command with default and custom valid stage plans against fakes or a reachable cluster; prove zero mutation calls, bounded workers and derived reads, correct unavailable evidence on missing/unsupported keys, useful comparative findings, and explicit read-only limitations.

### Tests for User Story 1

- [ ] T014 [P] [US1] Add failing service tests for topology/PD/PI measurement, measured-key reuse, Camunda 8.7 PI-keyed-read unavailability, empty/disappearing keys, bounded workers/counts, completed abnormal samples, cancellation, and zero mutation calls in `internal/services/ops/api_latency_test.go`
- [ ] T015 [P] [US1] Add failing facade tests for read-only progress, result, unavailable measurement, partial error, and collection conversion in `c8volt/ops/client_test.go`
- [ ] T016 [P] [US1] Add failing command tests for registration, help/examples, defaults `--count 20 --workers 4`, `-n/-w`, invalid stage budgets, keys-only rejection, read-only/full/automation metadata, tenant behavior, and no remote work after local validation in `cmd/ops_analyse_api_latency_test.go`
- [ ] T017 [US1] Add failing command tests for compact human output, one-document JSON, quiet/verbose/debug/automation progress behavior, stable stage order, findings, and mandatory read-only limitations in `cmd/ops_analyse_api_latency_test.go`

### Implementation for User Story 1

- [ ] T018 [US1] Implement the read-only closed-loop service workflow using existing cluster, process-definition, process-instance, and `toolx/pool` APIs in `internal/services/ops/api_latency_analysis.go`
- [ ] T019 [US1] Add the `ops analyse api-latency` Cobra leaf with local validation, inherited contracts, facade dispatch, and semantic activity setup in `cmd/ops_analyse_api_latency.go`
- [ ] T020 [US1] Implement compact read-only stage, finding, notice, limitation, and outcome rendering through existing human/JSON helpers in `cmd/cmd_views_ops_api_latency.go`
- [ ] T021 [US1] Centralize API-latency progress mode, rate-limited aggregate progress, and JSON/automation/quiet suppression in `cmd/ops_api_latency_progress.go`
- [ ] T022 [US1] Run `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1` and record US1 validation in `specs/289-api-latency-diagnostics/progress.md`
- [ ] T023 [US1] Execute the read-only quickstart scenarios and record the zero-mutation and bounded-evidence results in `specs/289-api-latency-diagnostics/progress.md`

**Checkpoint**: User Story 1 is a complete, independently testable production-safe MVP.

---

## Phase 4: User Story 2 - Exercise End-to-End Latency Under Bounded Load (Priority: P2)

**Goal**: Deliver a confirmed active test that preflights safety, deploys the existing SimpleUserTask fixture, measures create response and search visibility, samples reads while writes are active, stays within disclosed limits, and performs successful-path cleanup.

**Independent Test**: Run `ops execute api-latency-test` as dry-run and confirmed execution against fakes or a disposable 8.9/8.10 cluster; verify the previewed plan equals execution, no more than count instances are created, worker/derived ceilings hold, visibility is measured separately, and successful cleanup removes every owned resource.

### Tests for User Story 2

- [ ] T024 [P] [US2] Add failing service tests for run-ID generation, fixture selection, configured/observed version checks, dry-run zero mutation, active 8.7 rejection, cleanup-enabled 8.8 rejection, 8.8 no-cleanup eligibility, and 8.9/8.10 cleanup eligibility in `internal/services/ops/api_latency_test.go`
- [ ] T025 [US2] Add failing service tests for exact deployed-key creation with no visibility wait, immediate returned-key recording, stage concurrency/count bounds, overlapping-read evidence, bounded exact-key visibility polling, backpressure/timeout classification, active findings, and successful-path cleanup in `internal/services/ops/api_latency_test.go`
- [ ] T026 [P] [US2] Add failing facade tests for active plan, ownership, visibility, cleanup, progress, and partial-result conversion in `c8volt/ops/client_test.go`
- [ ] T027 [P] [US2] Add failing command tests for active flags/defaults, concrete-tenant enforcement, state-changing/full/automation metadata, dry-run preview, confirmation including `--no-cleanup`, automation/auto-confirm, JSON confirmation guardrails, and report-path preflight in `cmd/ops_execute_api_latency_test.go`
- [ ] T028 [US2] Add failing command tests for compact active preview/result output, plan/execution count parity, run identity, fixture, visibility, findings, successful cleanup, and no low-level per-key chatter in `cmd/ops_execute_api_latency_test.go`

### Implementation for User Story 2

- [ ] T029 [US2] Implement active preflight, cryptographic run-ID generation, version capability matrix, existing SimpleUserTask fixture selection, and immutable execution planning in `internal/services/ops/api_latency_execute.go`
- [ ] T030 [US2] Implement bounded active stages with `toolx/pool`, exact deployed-definition creation without exporter wait, overlapping read probes, bounded exact-key visibility polling, and actual-limit accounting in `internal/services/ops/api_latency_execute.go`
- [ ] T031 [US2] Implement successful-path exact-key process-instance then process-definition cleanup through existing owning services in `internal/services/ops/api_latency_cleanup.go`
- [ ] T032 [US2] Complete active-only ownership, visibility, cleanup, and outcome conversions in `c8volt/ops/convert.go`
- [ ] T033 [US2] Add the `ops execute api-latency-test` Cobra leaf with preview, confirmation, JSON guardrails, signal-ready facade dispatch, tenant context, and shared report-path planning in `cmd/ops_execute_api_latency.go`
- [ ] T034 [US2] Extend API-latency views with active preview, write/read/visibility stages, ownership summary, successful cleanup, and outcome rendering in `cmd/cmd_views_ops_api_latency.go`
- [ ] T035 [US2] Run `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1` and record US2 validation in `specs/289-api-latency-diagnostics/progress.md`

**Checkpoint**: User Story 2 can run and clean a bounded active diagnostic on cleanup-capable clusters without depending on reporting polish.

---

## Phase 5: User Story 3 - Preserve Ownership and Recover Safely (Priority: P2)

**Goal**: Guarantee exact-key ownership and cleanup after success, partial failure, request timeout, visibility exhaustion, and interruption, with intentional retention and cleanup failure represented distinctly.

**Independent Test**: Interrupt or fail an active fake run after returned keys are recorded; verify cleanup receives an independent bounded context, targets every and only recorded key, preserves all remaining keys/recovery commands, and returns the correct retained/partial/failed outcome.

### Tests for User Story 3

- [ ] T036 [P] [US3] Add failing service tests for concurrency-safe immediate ownership registration, stable key ordering/deduplication, exact-key-only cleanup authority, PI-before-PD order, and absence of BPMN-ID or tenant-wide cleanup discovery in `internal/services/ops/api_latency_test.go`
- [ ] T037 [US3] Add failing service tests for cleanup after stage failure, request timeout, visibility exhaustion, and canceled caller; independent cleanup timeout; partial cleanup; every remainder/recovery command; 8.8 retention; and no-cleanup distinction in `internal/services/ops/api_latency_test.go`
- [ ] T038 [P] [US3] Add failing facade tests proving partial ownership/cleanup results survive domain error conversion in `c8volt/ops/client_test.go`
- [ ] T039 [P] [US3] Add failing command tests for scoped interrupt cancellation, cleanup continuation, retained output, partial report attempts, exact recovery guidance, established nonzero error envelope, and no-cleanup confirmation in `cmd/ops_execute_api_latency_test.go`

### Implementation for User Story 3

- [ ] T040 [US3] Implement the concurrency-safe exact-key ownership registry and terminal cleanup-record accounting in `internal/services/ops/api_latency_cleanup.go`
- [ ] T041 [US3] Extend cleanup orchestration to run on success and every incomplete terminal path with `context.WithoutCancel`, a bounded completion context, exact PI-before-PD deletion, and joined partial errors in `internal/services/ops/api_latency_cleanup.go`
- [ ] T042 [US3] Implement intentional retention, unsupported-cleanup preflight blocking, remaining-resource classification, and exact-key manual recovery guidance in `internal/services/ops/api_latency_cleanup.go`
- [ ] T043 [US3] Install scoped signal-aware cancellation for the active execution window without changing global command behavior in `cmd/ops_execute_api_latency.go`
- [ ] T044 [US3] Preserve interrupted, partial, failed, and completed-retained outcomes while returning available service/facade evidence in `internal/services/ops/api_latency_execute.go` and `c8volt/ops/client.go`
- [ ] T045 [US3] Render retained resources, cleanup attempts/results, remaining exact keys, recovery guidance, and partial/interrupted outcomes compactly in `cmd/cmd_views_ops_api_latency.go`
- [ ] T046 [US3] Run cancellation/cleanup race-focused tests with `go test ./internal/services/ops ./cmd -run 'APILatency' -race -count=1` and record US3 validation in `specs/289-api-latency-diagnostics/progress.md`

**Checkpoint**: User Story 3 proves that active execution cannot silently orphan or over-delete resources it owns.

---

## Phase 6: User Story 4 - Share Reproducible Diagnostic Evidence (Priority: P3)

**Goal**: Complete compact human output, deterministic one-document JSON, shared Markdown/JSON reports, partial-result preservation, safe context, progress behavior, and exit semantics for both commands.

**Independent Test**: Run both commands in human, JSON, Markdown-report, JSON-report, partial-failure, quiet, verbose, debug, and automation modes; verify report parity, no protected values, and success for completed abnormal diagnostics versus nonzero incomplete/report/cleanup outcomes.

### Tests for User Story 4

- [ ] T047 [P] [US4] Add failing human and JSON renderer tests for stable schema/context, ordered stages/classifications/findings, read-only omission of active fields, active ownership/visibility/cleanup fields, compact wording, and five-second in-memory render budget in `cmd/ops_analyse_api_latency_test.go` and `cmd/ops_execute_api_latency_test.go`
- [ ] T048 [US4] Add failing report tests for Markdown/JSON inference, explicit override, dependent flags, `0600` files, missing parents, preserve/overwrite policy, raw JSON payload, Markdown parity, report-written line, and partial-result preservation in `cmd/ops_analyse_api_latency_test.go` and `cmd/ops_execute_api_latency_test.go`
- [ ] T049 [US4] Add failing output-safety and progress tests covering tokens, authorization headers, secrets, variables, payloads, raw response bodies, unbounded errors, JSON/automation silence, quiet failures, and verbose/debug detail in `cmd/ops_analyse_api_latency_test.go` and `cmd/ops_execute_api_latency_test.go`
- [ ] T050 [US4] Add failing subprocess tests proving completed abnormal evidence exits successfully while invalid, incomplete, report-failed, and requested-cleanup-failed runs use the established nonzero error envelope in `cmd/ops_analyse_api_latency_test.go` and `cmd/ops_execute_api_latency_test.go`

### Implementation for User Story 4

- [ ] T051 [US4] Complete shared compact human and stable command-envelope JSON rendering from the one API-latency result model in `cmd/cmd_views_ops_api_latency.go`
- [ ] T052 [US4] Implement command-specific Markdown and raw JSON report rendering by reusing `cmd/ops_report.go` and existing ops Markdown helpers from `cmd/cmd_views_ops_api_latency.go`
- [ ] T053 [US4] Wire exact shared report flag/path/write ordering, format inference, partial report attempts, and actual-mutation write mode into `cmd/ops_analyse_api_latency.go` and `cmd/ops_execute_api_latency.go`
- [ ] T054 [US4] Attach only safe build/profile/tenant/version context and fixed limitations/notices to both results in `cmd/ops_analyse_api_latency.go`, `cmd/ops_execute_api_latency.go`, and `cmd/cmd_views_ops_api_latency.go`
- [ ] T055 [US4] Map completed-abnormal results to success and incomplete/report/cleanup failures to existing command errors without a new partial stdout envelope in `cmd/ops_analyse_api_latency.go` and `cmd/ops_execute_api_latency.go`
- [ ] T056 [US4] Run `go test ./cmd -run 'APILatency|OpsWorkflowReport|CommandCapability|CapabilityDocument' -count=1` and record US4 validation in `specs/289-api-latency-diagnostics/progress.md`

**Checkpoint**: User Story 4 provides safe, reproducible evidence through existing output and report contracts.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Integrate the two leaves with repository inventories, real-state suites, documentation, formatting, and full validation.

- [ ] T057 Update command capability/family assertions and the expected inventory from 55 to 57 in `cmd/command_contract_test.go`, `cmd/capabilities_test.go`, and `integration/cli/all_commands_test.go`
- [ ] T058 [P] Extend read-only human, JSON, report, bounds, and seeded dirty-state coverage in `integration/cli/volume_ops_analyse_test.go`
- [ ] T059 [P] Extend active dry-run selected-version coverage and disposable 8.9/8.10 confirmed-cleanup evidence in `integration/cli/volume_ops_execute_test.go`
- [ ] T060 [P] Add command discoverability, safety distinction, and representative examples to `README.md` and `docs/ops/index.md`
- [ ] T061 [P] Add focused operator guidance in `docs/ops/analyse-api-latency.md` and `docs/ops/execute-api-latency-test.md`
- [ ] T062 Run `make docs-content` and review generated `docs/cli/c8volt_ops_analyse_api-latency.md`, `docs/cli/c8volt_ops_execute_api-latency-test.md`, and `docs/index.md` without hand-editing generated content
- [ ] T063 Verify documentation examples and non-tag integration contracts with `go test ./integration/cli -count=1`; record results in `specs/289-api-latency-diagnostics/progress.md`
- [ ] T064 Run `gofmt` on every touched Go file, `git diff --check`, and all focused service/facade/command tests listed in `specs/289-api-latency-diagnostics/quickstart.md`; record results in `specs/289-api-latency-diagnostics/progress.md`
- [ ] T065 Run the required race-enabled repository suite with `make test` and record final validation, known environment limits, and completion evidence in `specs/289-api-latency-diagnostics/progress.md`

**Checkpoint**: The feature is documented, regression-protected, formatted, and ready for review.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; complete T001-T002 first.
- **Foundational (Phase 2)**: Depends on Setup and blocks every user story.
- **User Story 1 (P1)**: Depends on Foundational and is the recommended MVP.
- **User Story 2 (P2)**: Depends on Foundational; it may be developed independently of US1, but both share foundational models and helpers.
- **User Story 3 (P2)**: Depends on US2 because cleanup/recovery consumes active-run ownership and execution state.
- **User Story 4 (P3)**: Depends on US1-US3 because its shared output/report model covers both modes and all active terminal states.
- **Polish (Phase 7)**: Depends on all selected stories.

### User Story Dependency Graph

```text
Setup -> Foundation -> US1 (read-only MVP)
                    -> US2 (active execution) -> US3 (cleanup/recovery)
US1 + US2 + US3 -> US4 (shared evidence/reporting) -> Polish
```

### Within Each User Story

- Write the story's tests first and verify they fail for the missing behavior.
- Add or extend domain/service behavior before facade conversion.
- Keep the facade mechanical and thin.
- Add command construction and validation before final rendering/report wiring.
- Run the narrow story validation and update `progress.md` before marking tasks complete.
- Inventory declarations in touched command files and apply the focused-file cohesion gate before validation.

### Parallel Opportunities

- T003 and T004 can be written in parallel because they touch service and facade tests.
- After foundational types exist, US1 and US2 service tests can be prepared in parallel by different contributors.
- Within US1, T014, T015, and T016 touch different files and can proceed in parallel.
- Within US2, T024, T026, and T027 touch different files and can proceed in parallel.
- Within US3, T036, T038, and T039 touch different files and can proceed in parallel.
- Within US4, renderer/report tests and service-safe-payload review can be split only when they do not edit the same command test file.
- T058-T061 are parallel after command behavior stabilizes because they touch separate integration/documentation files.

---

## Parallel Example: User Story 1

```text
Task: "Add read-only service safety and measurement tests in internal/services/ops/api_latency_test.go"
Task: "Add read-only facade conversion tests in c8volt/ops/client_test.go"
Task: "Add read-only command registration and validation tests in cmd/ops_analyse_api_latency_test.go"
```

## Parallel Example: User Story 2

```text
Task: "Add active preflight and version capability tests in internal/services/ops/api_latency_test.go"
Task: "Add active facade mapping tests in c8volt/ops/client_test.go"
Task: "Add active command contract and confirmation tests in cmd/ops_execute_api_latency_test.go"
```

## Parallel Example: User Story 3

```text
Task: "Add exact-key cleanup service tests in internal/services/ops/api_latency_test.go"
Task: "Add partial cleanup facade tests in c8volt/ops/client_test.go"
Task: "Add interrupt and retained-output command tests in cmd/ops_execute_api_latency_test.go"
```

## Parallel Example: User Story 4

```text
Task: "Add read-only report/output tests in cmd/ops_analyse_api_latency_test.go"
Task: "Add active report/output tests in cmd/ops_execute_api_latency_test.go"
Task: "Review safe aggregate result construction in internal/services/ops/api_latency.go"
```

---

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational tasks.
2. Complete User Story 1 only.
3. Stop and validate read-only behavior, especially zero mutation, worker/sample bounds, unavailable keyed evidence, safe findings, and output limitations.
4. Commit the independently useful MVP with a Conventional Commit subject ending in `#289`.

### Incremental Delivery

1. Add shared planning/models/service/facade seams.
2. Deliver read-only API latency analysis as the production-safe MVP.
3. Add bounded active execution with successful-path cleanup.
4. Harden exact ownership, interruption cleanup, retention, and recovery.
5. Complete shared human/JSON/report evidence and exit behavior.
6. Integrate inventories, volume suites, docs, generated docs, and full validation.

### Ralph Discipline

- Every iteration reads `specs/ralph-implementation-rules.md`, `spec.md`, `plan.md`, `tasks.md`, and `progress.md`, plus the relevant optional design artifacts.
- Implement only the first incomplete work unit; do not skip ahead to another story.
- Inspect nearby owners and search existing helpers before adding structures.
- Mark a task complete only after its relevant tests pass.
- Update `progress.md` with reusable discoveries and validation evidence in the same work-unit commit.
- Commit subjects must use Conventional Commits and end with `#289`.
