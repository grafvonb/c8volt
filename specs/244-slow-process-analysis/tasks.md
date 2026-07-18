# Tasks: Slow Process Instance Analysis

**Input**: Design documents from `/specs/244-slow-process-analysis/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli.md, quickstart.md, specs/ralph-implementation-rules.md

**Tests**: Required by the feature specification, success criteria, quickstart, and repository constitution. Each user story includes tests before implementation tasks.

**Organization**: Tasks are grouped by user story to enable independent validation of keyed analysis, process-definition discovery, timeline filtering, and output modes.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with other marked tasks in the same phase because it touches different files or has no dependency on incomplete tasks.
- **[Story]**: User story label for story phases only.
- Every task includes exact repository-relative file paths.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm feature context and prepare shared implementation touchpoints without behavior changes.

- [x] T001 Review feature artifacts and record any implementation conflicts in `specs/244-slow-process-analysis/spec.md`, `specs/244-slow-process-analysis/plan.md`, `specs/244-slow-process-analysis/contracts/cli.md`, and `specs/ralph-implementation-rules.md`
- [x] T002 [P] Inspect existing ops command and contract patterns in `cmd/ops.go`, `cmd/ops_contract.go`, `cmd/ops_contract_test.go`, and `cmd/command_contract_test.go`
- [x] T003 [P] Inspect existing ops facade and service patterns in `c8volt/ops/api.go`, `c8volt/ops/client.go`, `c8volt/ops/model.go`, and `internal/services/ops/api.go`
- [x] T004 [P] Inspect reusable process-instance and element service contracts in `internal/services/processinstance/api.go`, `internal/services/element/api.go`, `c8volt/process/api.go`, and `c8volt/element/api.go`
- [x] T005 Confirm Ralph launch instructions include `--implementation-context specs/ralph-implementation-rules.md` in `specs/244-slow-process-analysis/plan.md` and `specs/244-slow-process-analysis/tasks.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared analysis models, API contracts, dependency wiring, and command test scaffolding needed by all user stories.

**CRITICAL**: No user story implementation should begin until this phase is complete.

- [x] T006 Define version-neutral slow analysis request, result, duration, timeline, comparison, and filter domain types in `internal/domain/ops_slow_process_analysis.go`
- [x] T007 Define public slow analysis request/result models with stable JSON field tags in `c8volt/ops/model.go`
- [x] T008 [P] Add public/domain conversion helpers for slow analysis request and result models in `c8volt/ops/convert.go`
- [x] T009 Add `AnalyseSlowProcessInstances` to the public ops facade API in `c8volt/ops/api.go`
- [x] T010 Add `AnalyseSlowProcessInstances` to the internal ops service API in `internal/services/ops/api.go`
- [x] T011 Add process-instance and runtime element dependencies needed by slow analysis orchestration in `internal/services/ops/api.go` and `c8volt/client.go`
- [x] T012 Add command-level request parsing structures and test stubs for slow analysis in `cmd/ops_analyse_slow_process_instances.go` and `cmd/ops_test.go`
- [x] T013 Add slow analysis command metadata expectations for read-only behavior, output modes, and automation suitability in `cmd/ops_contract_test.go` and `cmd/command_contract_test.go`
- [x] T014 Add reusable slow analysis fixture builders for process instances, elements, and timestamps in `internal/services/ops/slow_process_analysis_test.go`

**Checkpoint**: Shared contracts compile and tests can stub the analysis flow before user story behavior is implemented.

---

## Phase 3: User Story 1 - Analyze Known Process Instances By Key (Priority: P1) MVP

**Goal**: Operators can analyze one or more known process instance keys from flags, stdin, or both, with deduplication and deterministic root ordering.

**Independent Test**: Analyze one known key, repeated `--key` values, stdin keys, and mixed flag/stdin keys; verify deduplication, validation, sorting, read-only behavior, and Camunda 8.7 unsupported behavior.

### Tests for User Story 1

- [ ] T015 [P] [US1] Add internal service tests for explicit-key deduplication, lookup, missing-key and unauthorized-key failures, unavailable-duration ordering, and Camunda 8.7 unsupported behavior in `internal/services/ops/slow_process_analysis_test.go`
- [ ] T016 [P] [US1] Add ops facade tests for explicit-key request conversion, service delegation, result conversion, and error mapping in `c8volt/ops/client_test.go`
- [ ] T017 [P] [US1] Add command validation tests for repeated `--key`, `-k`, stdin `-`, mixed keys, empty stdin, invalid keys, extra positional args, and mutually exclusive selectors in `cmd/ops_analyse_slow_process_instances_test.go`
- [ ] T018 [P] [US1] Add keyed human rendering tests for root rows, `dur:`, unavailable duration placement, and final process-instance count in `cmd/cmd_views_ops_slow_process_analysis_test.go`

### Implementation for User Story 1

- [ ] T019 [US1] Implement explicit-key request validation and stdin key merging in `cmd/ops_analyse_slow_process_instances.go`
- [ ] T020 [US1] Implement public ops facade delegation for keyed slow analysis in `c8volt/ops/client.go` and `c8volt/ops/convert.go`
- [ ] T021 [US1] Implement explicit-key selection, deduplication, tenant-safe lookup, Camunda 8.7 unsupported handling, captured analysis time, and root duration sorting in `internal/services/ops/slow_process_analysis.go`
- [ ] T022 [US1] Register `ops analyse slow-process-instances` and `ops analyze slow-process-instances` with `--key`/`-k`, stdin `-`, read-only metadata, help text, and examples in `cmd/ops_analyse_slow_process_instances.go`
- [ ] T023 [US1] Render keyed analysis root rows and final process-instance count in `cmd/cmd_views_ops_slow_process_analysis.go`

**Checkpoint**: User Story 1 is fully functional and independently testable as the MVP.

---

## Phase 4: User Story 2 - Discover Slow Runs For One Process Definition (Priority: P2)

**Goal**: Operators can select process instances for one process definition with search filters, freeze the selected set, and receive successful empty analyses when no process instances match.

**Independent Test**: Select by exactly one process-definition selector, apply process-instance filters and discovery controls, verify frozen selection behavior, and verify empty human/JSON/keys-only results.

### Tests for User Story 2

- [ ] T024 [P] [US2] Add internal service tests for `--bpmn-process-id`, `--pd-key`, state values `active`/`completed`/`canceled`/`terminated`/`all`, precise timestamp and `YYYY-MM-DD` date filters, no-incidents filters, batch size, limit, frozen selection, and empty search success in `internal/services/ops/slow_process_analysis_test.go`
- [ ] T025 [P] [US2] Add command validation tests for required selector, mutually exclusive `--bpmn-process-id` and `--pd-key`, search filters rejected in explicit-key mode, and `--incidents-only` not accepted or advertised in `cmd/ops_analyse_slow_process_instances_test.go`
- [ ] T026 [P] [US2] Add command integration tests for process-definition search with `--state all`, precise timestamp and `YYYY-MM-DD` date filters, `--batch-size`, `--limit`, and empty results in `cmd/ops_analyse_slow_process_instances_test.go`
- [ ] T027 [P] [US2] Add empty-result rendering tests for human count, JSON empty items, and keys-only silence in `cmd/cmd_views_ops_slow_process_analysis_test.go`

### Implementation for User Story 2

- [ ] T028 [US2] Implement process-definition selector and process-instance search filter parsing in `cmd/ops_analyse_slow_process_instances.go`
- [ ] T029 [US2] Implement process-definition search discovery, paging controls, limits, empty success, and frozen selected-set construction in `internal/services/ops/slow_process_analysis.go`
- [ ] T030 [US2] Map process-definition search requests and discovery result metadata through the ops facade in `c8volt/ops/model.go`, `c8volt/ops/convert.go`, and `c8volt/ops/client.go`
- [ ] T031 [US2] Render empty analysis results consistently for human, JSON, and keys-only modes in `cmd/cmd_views_ops_slow_process_analysis.go`
- [ ] T032 [US2] Ensure `--batch-size` and `--limit` affect process-instance discovery only and never truncate explicit keys or timeline details in `internal/services/ops/slow_process_analysis.go`

**Checkpoint**: User Stories 1 and 2 both work independently, with search mode preserving process-instance selection semantics.

---

## Phase 5: User Story 3 - Inspect Timelines, Transitions, And Slow Details (Priority: P3)

**Goal**: Operators can inspect chronological element rows, adjacent transition timings, duration calculations, and detail filters without causality claims or synthetic transitions.

**Independent Test**: Analyze process instances with completed, active, terminated, overlapping, missing-timestamp, and incident-bearing elements; apply detail filters and verify complete-timeline calculations remain stable.

### Tests for User Story 3

- [ ] T033 [P] [US3] Add internal service tests for runtime element ordering, active/completed/terminated element durations, missing timestamps, incident markers, and captured analysis time reuse in `internal/services/ops/slow_process_analysis_test.go`
- [ ] T034 [P] [US3] Add internal service tests for adjacent transition timings, overlap rejection, missing timestamp gaps, no synthetic bridging, and chronological-only semantics in `internal/services/ops/slow_process_analysis_test.go`
- [ ] T035 [P] [US3] Add internal service tests for `--element-id`, `--type`, `--element-state`, and `--duration-after` detail filtering after complete timeline calculations in `internal/services/ops/slow_process_analysis_test.go`
- [ ] T036 [P] [US3] Add renderer tests for `elements:` sections, compact element rows, `A -> B: duration`, no `between:`/`transition:` prefix, and `PI:<percentage>` placement in `cmd/cmd_views_ops_slow_process_analysis_test.go`
- [ ] T037 [P] [US3] Add command tests for detail filter parsing and invalid duration values in `cmd/ops_analyse_slow_process_instances_test.go`

### Implementation for User Story 3

- [ ] T038 [US3] Implement runtime element lookup coordination for every selected process instance in `internal/services/ops/slow_process_analysis.go`
- [ ] T039 [US3] Implement element duration calculation and chronological element timeline construction in `internal/services/ops/slow_process_analysis.go`
- [ ] T040 [US3] Implement adjacent transition timing calculation with overlap, missing timestamp, and no-bridging rules in `internal/services/ops/slow_process_analysis.go`
- [ ] T041 [US3] Implement detail filter parsing and duration value validation in `cmd/ops_analyse_slow_process_instances.go`
- [ ] T042 [US3] Implement post-calculation detail filtering for elements and transitions in `internal/services/ops/slow_process_analysis.go`
- [ ] T043 [US3] Render element rows, transition rows, incident markers, durations, and process-duration shares in `cmd/cmd_views_ops_slow_process_analysis.go`

**Checkpoint**: User Stories 1, 2, and 3 provide complete timeline analysis in human output.

---

## Phase 6: User Story 4 - Consume Stable Human, JSON, And Keys-Only Results (Priority: P4)

**Goal**: Operators and automation authors can consume stable slow-analysis results in human, JSON, and keys-only modes, including relative-duration indicators.

**Independent Test**: Run equivalent selections in human, JSON, and keys-only modes; verify stable root ordering, explicit JSON fields, comparison indicators, sample counts, and final counts.

### Tests for User Story 4

- [ ] T044 [P] [US4] Add internal service tests for process, element, and transition comparison scopes, percentile calculation, tie handling, minimum sample count, and ten-cell bar inputs in `internal/services/ops/slow_process_analysis_test.go`
- [ ] T045 [P] [US4] Add JSON rendering tests for captured analysis time, root durations, timeline entries, endpoints, comparison sample counts, relative percentiles, and process-duration shares in `cmd/cmd_views_ops_slow_process_analysis_test.go`
- [ ] T046 [P] [US4] Add keys-only rendering tests for unique root keys, longest-to-shortest ordering, unavailable-duration ordering, empty output, and detail-filter independence in `cmd/cmd_views_ops_slow_process_analysis_test.go`
- [ ] T047 [P] [US4] Add command contract tests for both command spellings, output modes, read-only metadata, automation compatibility, and help examples in `cmd/ops_contract_test.go` and `cmd/command_contract_test.go`
- [ ] T048 [P] [US4] Add docs metadata tests for generated CLI docs covering slow-analysis commands and flags in `docsgen/main_test.go`

### Implementation for User Story 4

- [ ] T049 [US4] Implement relative percentile, comparison sample count, ten-cell bar, and `PI:<percentage>` calculations in `internal/services/ops/slow_process_analysis.go`
- [ ] T050 [US4] Implement JSON payload structs and shared envelope rendering for slow analysis in `cmd/cmd_views_ops_slow_process_analysis.go`
- [ ] T051 [US4] Implement keys-only output for unique process-instance keys in result ordering in `cmd/ops_analyse_slow_process_instances.go` and `cmd/cmd_views_ops_slow_process_analysis.go`
- [ ] T052 [US4] Finalize command metadata, aliases, examples, output mode declarations, and docs-visible flag descriptions in `cmd/ops_analyse_slow_process_instances.go`
- [ ] T053 [US4] Update README examples and behavior notes for slow process-instance analysis in `README.md`

**Checkpoint**: All user stories are independently functional across supported output modes.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, generated artifacts, validation, and cleanup across all user stories.

- [ ] T054 [P] Update quickstart examples if implementation output wording changes in `specs/244-slow-process-analysis/quickstart.md`
- [ ] T055 Run `gofmt` on `cmd/ops_analyse_slow_process_instances.go`, `cmd/cmd_views_ops_slow_process_analysis.go`, `c8volt/ops/api.go`, `c8volt/ops/client.go`, `c8volt/ops/convert.go`, `c8volt/ops/model.go`, `c8volt/client.go`, `internal/domain/ops_slow_process_analysis.go`, and `internal/services/ops/slow_process_analysis.go`
- [ ] T056 Run targeted internal service validation for `internal/services/ops/slow_process_analysis_test.go` with `go test ./internal/services/ops -run 'TestSlowProcessAnalysis' -count=1`
- [ ] T057 Run targeted facade validation for `c8volt/ops/client_test.go` with `go test ./c8volt/ops -run 'TestClient_.*SlowProcessAnalysis' -count=1`
- [ ] T058 Run targeted command validation for `cmd/ops_analyse_slow_process_instances_test.go`, `cmd/cmd_views_ops_slow_process_analysis_test.go`, `cmd/ops_contract_test.go`, and `cmd/command_contract_test.go` with `go test ./cmd -run 'Test.*SlowProcessAnalysis|TestOps.*SlowProcess|TestCommandContract|TestOpsContract' -count=1`
- [ ] T059 Run docs validation and regenerate generated CLI docs for `docsgen/main_test.go`, `docs/cli/`, and `docs/index.md` with `go test ./docsgen -count=1` and `make docs-content`
- [ ] T060 Build the quickstart binary from `go.mod` with `go build -o /tmp/c8volt-slow-pi-analysis .`
- [ ] T061 Verify manual scenarios from `specs/244-slow-process-analysis/quickstart.md` against `/tmp/c8volt-slow-pi-analysis`
- [ ] T062 Run full repository validation through `Makefile` with `make test`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 Setup**: No dependencies.
- **Phase 2 Foundational**: Depends on Phase 1 context review and blocks all user stories.
- **Phase 3 US1**: Depends on Phase 2 and delivers the MVP.
- **Phase 4 US2**: Depends on Phase 2; recommended after US1 because it extends the same command and service flow.
- **Phase 5 US3**: Depends on US1 and US2 because timeline details attach to selected process-instance roots.
- **Phase 6 US4**: Depends on US1-US3 data being available for stable output and comparison indicators.
- **Phase 7 Polish**: Depends on all desired user stories.

### User Story Dependencies

- **US1 (P1)**: Requires foundational models, facade wiring, service API, and command scaffolding; no dependency on other stories.
- **US2 (P2)**: Requires foundational service API and command scaffolding; can be tested independently with search fixtures, but should follow US1 to reduce same-file conflicts.
- **US3 (P3)**: Requires selected process-instance roots from US1/US2 before element timeline and transition analysis can be meaningful.
- **US4 (P4)**: Requires analysis result fields from US1-US3 before JSON, keys-only, and comparison output can be finalized.

### Within Each User Story

- Write tests before implementation tasks and verify they fail for missing behavior.
- Domain/public models before facade conversion.
- Service orchestration before command rendering.
- Command validation before remote service calls.
- Renderer and output contracts before documentation regeneration.

## Parallel Opportunities

- Setup inspections T002, T003, and T004 can run in parallel because they read separate source areas.
- Foundational conversion task T008 can run in parallel after model names are defined.
- US1 tests T015-T018 can be written in parallel because they target service, facade, validation, and rendering files separately.
- US2 tests T024-T027 can be written in parallel because they target service, validation, integration, and rendering behavior.
- US3 tests T033-T037 can be written in parallel because they cover service calculations, renderer output, and command parsing.
- US4 tests T044-T048 can be written in parallel because they target service comparison math, output rendering, contract metadata, and docs metadata.
- Polish quickstart update T054 can run in parallel with validation once command output wording stabilizes.

## Parallel Example: User Story 1

```text
Task: "T015 [US1] Add internal service tests in internal/services/ops/slow_process_analysis_test.go"
Task: "T016 [US1] Add ops facade tests in c8volt/ops/client_test.go"
Task: "T017 [US1] Add command validation tests in cmd/ops_analyse_slow_process_instances_test.go"
Task: "T018 [US1] Add keyed human rendering tests in cmd/cmd_views_ops_slow_process_analysis_test.go"
```

## Parallel Example: User Story 2

```text
Task: "T024 [US2] Add process-definition search service tests in internal/services/ops/slow_process_analysis_test.go"
Task: "T025 [US2] Add selector validation tests in cmd/ops_analyse_slow_process_instances_test.go"
Task: "T027 [US2] Add empty-result rendering tests in cmd/cmd_views_ops_slow_process_analysis_test.go"
```

## Parallel Example: User Story 3

```text
Task: "T033 [US3] Add element duration tests in internal/services/ops/slow_process_analysis_test.go"
Task: "T034 [US3] Add transition timing tests in internal/services/ops/slow_process_analysis_test.go"
Task: "T036 [US3] Add compact timeline renderer tests in cmd/cmd_views_ops_slow_process_analysis_test.go"
Task: "T037 [US3] Add detail filter parsing tests in cmd/ops_analyse_slow_process_instances_test.go"
```

## Parallel Example: User Story 4

```text
Task: "T044 [US4] Add comparison indicator tests in internal/services/ops/slow_process_analysis_test.go"
Task: "T045 [US4] Add JSON rendering tests in cmd/cmd_views_ops_slow_process_analysis_test.go"
Task: "T046 [US4] Add keys-only rendering tests in cmd/cmd_views_ops_slow_process_analysis_test.go"
Task: "T047 [US4] Add command contract tests in cmd/ops_contract_test.go and cmd/command_contract_test.go"
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 setup.
2. Complete Phase 2 foundational contracts and wiring.
3. Complete Phase 3 keyed slow-analysis workflow.
4. Validate `c8volt ops analyse slow-process-instances --key <process-instance-key>` and Camunda 8.7 unsupported behavior.

### Incremental Delivery

1. Deliver US1 keyed analysis as the MVP.
2. Add US2 process-definition discovery and empty-result behavior.
3. Add US3 element timeline, transition timing, and detail filtering.
4. Add US4 stable JSON, keys-only, relative indicators, docs, and output contracts.
5. Finish generated docs, quickstart validation, and full repository validation.

### Ralph Execution

1. Launch Ralph only after confirming it receives `--implementation-context specs/ralph-implementation-rules.md`.
2. Keep each Ralph iteration to one story or validation slice from this task file.
3. Do not stage or commit until that iteration's validation passes.
