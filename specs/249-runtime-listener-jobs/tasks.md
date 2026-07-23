# Tasks: Runtime Listener Jobs Under Elements

**Input**: Design documents from `/specs/249-runtime-listener-jobs/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/), [quickstart.md](./quickstart.md)

**Tests**: Required by the c8volt constitution and quickstart validation guidance. Write story tests before implementation and verify they fail for the missing behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with other tasks in the same phase when files do not overlap.
- **[Story]**: User story traceability label for story phases only.
- Every task includes concrete repository file paths.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Orient implementation around existing repository patterns before changing shared contracts.

- [ ] T001 Review existing element, job, process activity, walk, and slow-analysis patterns in `cmd/get_element.go`, `cmd/get_job.go`, `cmd/get_processinstance_enrichment.go`, `cmd/cmd_views_processinstance_activity.go`, `cmd/walk_processinstance.go`, and `cmd/ops_analyse_slow_process_instances.go`
- [ ] T002 [P] Review facade and service boundaries for listener data in `c8volt/element/api.go`, `c8volt/process/api.go`, `c8volt/job/api.go`, `c8volt/ops/api.go`, `internal/services/processinstance/enrichment.go`, and `internal/services/job/api.go`
- [ ] T003 [P] Review generated job search capability and unsupported-version behavior in `internal/services/job/v87/service.go`, `internal/services/job/v88/service.go`, `internal/services/job/v88/convert.go`, `internal/services/job/v89/service.go`, and `internal/services/job/v89/convert.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared listener data model, ownership attachment, and rendering helpers that all user stories rely on.

**CRITICAL**: No user story work can begin until this phase is complete.

### Tests First

- [x] T004 [P] Add failing ownership and omission tests for element listener attachment in `internal/services/processinstance/enrichment_test.go`
- [x] T005 [P] Add failing public process facade conversion tests for listener-enriched elements in `c8volt/process/client_test.go`
- [x] T006 [P] Add failing listener row formatter tests for human output nesting and JSON empty-array behavior in `cmd/cmd_views_processinstance_activity_test.go`

### Implementation

- [x] T007 Add version-neutral runtime listener job fields and listener-enriched element structs in `internal/domain/job.go`, `internal/domain/element.go`, `internal/domain/processinstance_enrichment.go`, and `internal/domain/ops_slow_process_analysis.go`
- [x] T008 Add public listener job models and `listeners` fields in `c8volt/element/model.go`, `c8volt/process/model.go`, and `c8volt/ops/model.go`
- [x] T009 Implement domain-to-public listener conversion helpers in `c8volt/element/client.go`, `c8volt/process/client.go`, and `c8volt/ops/convert.go`
- [x] T010 Implement listener job grouping by process instance and element instance key in `internal/services/processinstance/enrichment.go`
- [x] T011 Extend the process facade listener-enrichment contract and root wiring in `c8volt/process/api.go`, `c8volt/process/client.go`, and `c8volt/client.go`
- [x] T012 Implement shared listener row formatting and requested-but-empty JSON support in `cmd/cmd_views_processinstance_activity.go`
- [x] T013 Run foundational tests with `go test ./internal/services/processinstance ./c8volt/process ./cmd -run 'Test.*Listener' -count=1` and record results in `specs/249-runtime-listener-jobs/quickstart.md`

**Checkpoint**: Listener jobs can be attached to element data by shared services and represented by shared human/JSON renderers.

---

## Phase 3: User Story 1 - Inspect Listener Jobs For A Specific Element (Priority: P1) MVP

**Goal**: Operators can run element-oriented commands with `--with-listeners` and see runtime listener jobs below the owning element rows.

**Independent Test**: View a known element instance and a process-instance element search with listener enrichment; listener rows appear only below matching elements, empty elements are handled cleanly, and JSON listener arrays appear only when requested.

### Tests for User Story 1

- [x] T014 [P] [US1] Add failing command contract and help tests for `get element --with-listeners` in `cmd/command_contract_test.go` and `docsgen/main_test.go`
- [x] T015 [P] [US1] Add failing keyed element listener human, JSON, empty, unmatched, and validation tests in `cmd/get_element_test.go`
- [x] T016 [US1] Add failing element search listener human, JSON, empty, unmatched, paging, total/keys-only validation, and unsupported-version tests in `cmd/get_element_test.go`
- [x] T017 [P] [US1] Add failing element facade listener enrichment tests in `c8volt/element/client_test.go`

### Implementation for User Story 1

- [x] T018 [US1] Extend the element facade API with listener enrichment methods in `c8volt/element/api.go` and `c8volt/element/client.go`
- [x] T019 [US1] Wire the element facade to job-service listener enrichment in `c8volt/client.go`
- [x] T020 [US1] Register and validate `--with-listeners` for element lookup and search modes in `cmd/get_element.go`
- [x] T021 [US1] Add listener enrichment orchestration for keyed and search element results in `cmd/get_element.go` and `cmd/get_element_search.go`
- [x] T022 [US1] Render element listener rows and JSON arrays for keyed and list element output in `cmd/cmd_views_element.go`
- [x] T023 [US1] Run US1 validation with `go test ./cmd ./c8volt/element -run 'Test(GetElement|CommandContract|GeneratedDocs).*Listener' -count=1` and record results in `specs/249-runtime-listener-jobs/quickstart.md`

**Checkpoint**: User Story 1 is independently functional and can serve as the MVP.

---

## Phase 4: User Story 2 - Correlate Listener Jobs In Process Instance Element Views (Priority: P1)

**Goal**: Operators can run `get pi --with-elements --with-listeners` and see listener jobs under matching element rows without changing default process-instance output.

**Independent Test**: Compare process-instance output with and without `--with-listeners`; only enriched output includes listener arrays and rows, while invalid missing-element-context combinations fail clearly.

### Tests for User Story 2

- [x] T024 [P] [US2] Add failing process-instance command contract and help tests for `--with-listeners` in `cmd/command_contract_test.go` and `docsgen/main_test.go`
- [x] T025 [P] [US2] Add failing `get pi --with-elements --with-listeners` human, JSON, empty-array, unmatched, unsupported-version, and unchanged-default tests in `cmd/get_processinstance_test.go`
- [x] T026 [US2] Add failing process-instance listener validation tests for missing `--with-elements` and keys-only output in `cmd/get_processinstance_test.go`

### Implementation for User Story 2

- [x] T027 [US2] Register `--with-listeners` and update process-instance help/examples in `cmd/get_processinstance.go`
- [x] T028 [US2] Add process-instance listener validation for element context and output mode in `cmd/get_processinstance_validation.go` and `cmd/get_processinstance.go`
- [x] T029 [US2] Include requested listener enrichment in shared process-instance activity collection in `cmd/get_processinstance_enrichment.go`
- [x] T030 [US2] Attach listener-enriched element data to process activity items in `cmd/cmd_views_processinstance_activity.go`
- [x] T031 [US2] Run US2 validation with `go test ./cmd -run 'TestGetProcessInstance.*Listener|TestCommandContract.*Listener|TestGenerated.*get pi' -count=1` and record results in `specs/249-runtime-listener-jobs/quickstart.md`

**Checkpoint**: User Stories 1 and 2 both work independently for element-oriented and process-instance-oriented inspection.

---

## Phase 5: User Story 3 - Preserve Walk Tree Readability With Listener Details (Priority: P2)

**Goal**: Operators can combine `walk pi --with-elements --with-listeners` with default, children, parent, and flat modes while preserving traversal structure.

**Independent Test**: Run each walk mode with listener enrichment; selected process-instance rows and tree structure match the same walk without listeners, and listener rows stay inside the owning element block.

### Tests for User Story 3

- [x] T032 [P] [US3] Add failing walk command contract and help tests for `--with-listeners` in `cmd/command_contract_test.go` and `docsgen/main_test.go`
- [x] T033 [P] [US3] Add failing default, children, parent, and flat walk listener human output tests in `cmd/walk_test.go`
- [x] T034 [US3] Add failing walk listener JSON, keys-only validation, missing `--with-elements`, unsupported-version, and unchanged-default tests in `cmd/walk_test.go`

### Implementation for User Story 3

- [x] T035 [US3] Register `--with-listeners` and update walk help/examples in `cmd/walk_processinstance.go`
- [x] T036 [US3] Add walk listener validation for `--with-elements`, keyed traversal, and keys-only output in `cmd/walk_processinstance.go`
- [x] T037 [US3] Enrich walked process instances with listener data after traversal and element enrichment in `cmd/walk_processinstance.go`
- [x] T038 [US3] Preserve listener arrays in activity traversal JSON and tree rendering in `cmd/cmd_views_walk_incidents.go` and `cmd/cmd_views_processinstance_activity.go`
- [x] T039 [US3] Run US3 validation with `go test ./cmd -run 'TestWalkProcessInstance.*Listener|TestCommandContract.*walk|TestGenerated.*walk' -count=1` and record results in `specs/249-runtime-listener-jobs/quickstart.md`

**Checkpoint**: Walk listener enrichment is independently functional across traversal modes.

---

## Phase 6: User Story 4 - Include Listener Context In Slow Process Analysis (Priority: P3)

**Goal**: Operators can run slow-process analysis with `--with-listeners` and see listener context under matching element timeline rows.

**Independent Test**: Run slow-process analysis for a process instance with element-owned listener jobs; listener rows appear under element timeline rows only when requested, and transition rows remain listener-free.

### Tests for User Story 4

- [x] T040 [P] [US4] Add failing slow-analysis command contract and help tests for `--with-listeners` in `cmd/ops_contract_test.go` and `docsgen/main_test.go`
- [x] T041 [P] [US4] Add failing slow-analysis command parse and validation tests for `--with-listeners` in `cmd/ops_analyse_slow_process_instances_test.go`
- [x] T042 [P] [US4] Add failing slow-analysis service tests for listener lookup, attachment, unmatched omission, unsupported-version behavior, and unchanged-default behavior in `internal/services/ops/slow_process_analysis_test.go`
- [x] T043 [P] [US4] Add failing slow-analysis renderer tests for human nesting, JSON listener arrays, and transition exclusion in `cmd/cmd_views_ops_slow_process_analysis_test.go`
- [x] T044 [P] [US4] Add failing public ops facade conversion tests for listener-enabled slow-analysis output in `c8volt/ops/client_test.go`

### Implementation for User Story 4

- [x] T045 [US4] Add `WithListeners` request and timeline listener fields in `internal/domain/ops_slow_process_analysis.go` and `c8volt/ops/model.go`
- [x] T046 [US4] Map slow-analysis listener request and response fields across the facade in `c8volt/ops/convert.go`
- [x] T047 [US4] Add slow-analysis listener lookup and element-timeline attachment in `internal/services/ops/slow_process_analysis.go`
- [x] T048 [US4] Register and validate `--with-listeners` for slow-process analysis in `cmd/ops_analyse_slow_process_instances.go`
- [x] T049 [US4] Render slow-analysis listener rows and JSON arrays in `cmd/cmd_views_ops_slow_process_analysis.go`
- [x] T050 [US4] Run US4 validation with `go test ./cmd ./c8volt/ops ./internal/services/ops -run 'Test.*SlowProcess.*Listener|TestOps.*Listener|TestGenerated.*slow-process' -count=1` and record results in `specs/249-runtime-listener-jobs/quickstart.md`

**Checkpoint**: Slow-process analysis listener enrichment is independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, generated CLI reference, formatting, and full validation across all stories.

- [x] T051 [P] Update README examples and behavior notes for `--with-listeners` in `README.md`
- [x] T052 [P] Update or add command metadata assertions for all in-scope commands in `cmd/command_contract_test.go`, `cmd/ops_contract_test.go`, and `docsgen/main_test.go`
- [x] T053 Regenerate CLI documentation with `make docs-content` and verify generated files under `docs/cli/`
- [x] T054 Run `gofmt` on touched Go files in `cmd/`, `c8volt/`, and `internal/`
- [x] T055 Run targeted quickstart validation from `specs/249-runtime-listener-jobs/quickstart.md`
- [x] T056 Run full repository validation with `make test`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup and blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational; this is the MVP.
- **User Story 2 (Phase 4)**: Depends on Foundational and benefits from US1 renderer/facade wiring, but remains independently testable through `get pi`.
- **User Story 3 (Phase 5)**: Depends on Foundational and uses the same activity enrichment path as US2.
- **User Story 4 (Phase 6)**: Depends on Foundational and uses the same listener job model and grouping behavior.
- **Polish (Phase 7)**: Depends on all desired stories being complete.

### User Story Dependencies

- **US1 - Element listener inspection**: Start after Foundational; recommended MVP.
- **US2 - Process-instance element listener correlation**: Start after Foundational; can proceed after shared renderer/facade tasks are stable.
- **US3 - Walk tree listener readability**: Start after Foundational; can proceed in parallel with US2 after shared activity collection supports listeners.
- **US4 - Slow-analysis listener context**: Start after Foundational; can proceed independently of walk once listener model and grouping helpers exist.

### Within Each User Story

- Write failing tests first.
- Add or extend models and service/facade contracts before command orchestration.
- Add validation before remote listener lookup.
- Add human and JSON rendering after enrichment data is available.
- Run the story-specific validation command before moving to the next story.

## Parallel Opportunities

- T002 and T003 can run in parallel after T001 starts.
- T004, T005, and T006 can run in parallel because they touch different test files.
- T014, T015, T016, and T017 can run in parallel for US1.
- T024, T025, and T026 can run in parallel for US2.
- T032, T033, and T034 can run in parallel for US3.
- T040, T041, T042, T043, and T044 can run in parallel for US4.
- T051 and T052 can run in parallel during polish.

## Parallel Example: User Story 1

```text
Task: "T014 [P] [US1] Add failing command contract and help tests for `get element --with-listeners` in `cmd/command_contract_test.go` and `docsgen/main_test.go`"
Task: "T015 [P] [US1] Add failing keyed element listener human, JSON, empty, unmatched, and validation tests in `cmd/get_element_test.go`"
Task: "T017 [P] [US1] Add failing element facade listener enrichment tests in `c8volt/element/client_test.go`"
```

## Parallel Example: User Story 3

```text
Task: "T032 [P] [US3] Add failing walk command contract and help tests for `--with-listeners` in `cmd/command_contract_test.go` and `docsgen/main_test.go`"
Task: "T033 [P] [US3] Add failing default, children, parent, and flat walk listener human output tests in `cmd/walk_test.go`"
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 and Phase 2.
2. Complete Phase 3 for `get element --with-listeners`.
3. Stop and validate US1 independently with T023.
4. Demo element-owned listener rows and JSON listener arrays.

### Incremental Delivery

1. Add US1 for direct element listener diagnostics.
2. Add US2 for `get pi --with-elements --with-listeners`.
3. Add US3 for `walk pi --with-elements --with-listeners`.
4. Add US4 for slow-process analysis listener context.
5. Finish polish, docs, generated CLI docs, and full validation.

### Validation Discipline

- Keep `--with-listeners` opt-in and verify no remote listener lookup occurs when the flag is absent or local validation fails.
- Preserve byte-stable unchanged-output fixtures for commands without listener enrichment.
- Run the closest targeted tests first, then `make test` before commit or merge.
