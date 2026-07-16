# Tasks: Process Instance Element Enrichment

**Input**: Design documents from `/specs/235-get-pi-elements/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli.md, quickstart.md, specs/ralph-implementation-rules.md

**Tests**: Required by the feature specification, success criteria, quickstart, and repository constitution. Each user story includes tests before implementation tasks.

**Organization**: Tasks are grouped by user story to enable independent validation of keyed enrichment, list/search enrichment, and combined output modes.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with other marked tasks in the same phase because it touches different files or has no dependency on incomplete tasks.
- **[Story]**: User story label for story phases only.
- Every task includes exact repository-relative file paths.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the feature context and prepare shared implementation touchpoints without behavior changes.

- [x] T001 Review the feature artifacts and record any implementation conflicts in `specs/235-get-pi-elements/plan.md`, `specs/235-get-pi-elements/spec.md`, `specs/235-get-pi-elements/contracts/cli.md`, and `specs/ralph-implementation-rules.md`
- [x] T002 [P] Inspect existing process-instance enrichment patterns in `cmd/get_processinstance.go`, `cmd/get_processinstance_enrichment.go`, `cmd/cmd_views_processinstance_activity.go`, and `cmd/process_api_stub_test.go`
- [x] T003 [P] Inspect existing element facade/service contracts for reuse in `c8volt/element/api.go`, `c8volt/element/model.go`, `internal/services/element/api.go`, and `internal/domain/element.go`
- [x] T004 [P] Inspect existing process facade and internal enrichment contracts in `c8volt/process/api.go`, `c8volt/process/client.go`, `c8volt/process/model.go`, `c8volt/process/convert.go`, and `internal/services/processinstance/enrichment.go`
- [x] T005 Confirm Ralph launch instructions include `--implementation-context specs/ralph-implementation-rules.md` in `specs/235-get-pi-elements/plan.md` and `specs/235-get-pi-elements/tasks.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared models, facade contracts, service interfaces, and command test plumbing needed by all user stories.

**CRITICAL**: No user story implementation should begin until this phase is complete.

- [x] T006 Define element-enriched process-instance domain types in `internal/domain/processinstance_enrichment.go`
- [x] T007 Define public process element enrichment models and JSON field tags in `c8volt/process/model.go`
- [x] T008 [P] Add public/internal conversion helpers for attached runtime elements in `c8volt/process/convert.go`
- [x] T009 Add `EnrichProcessInstancesWithElements` to the process facade API in `c8volt/process/api.go`
- [x] T010 Add an element service dependency to the process facade client in `c8volt/process/client.go`
- [x] T011 Wire the element service into the process facade construction in `c8volt/client.go`
- [x] T012 Extend the process API command stub with element enrichment support in `cmd/process_api_stub_test.go`
- [x] T013 Add `flagGetPIWithElements` reset and command-level plumbing alongside existing enrichment flags in `cmd/get_processinstance.go` and `cmd/get_processinstance_test.go`

**Checkpoint**: Shared contracts compile and command tests can stub element enrichment before story behavior is implemented.

---

## Phase 3: User Story 1 - Inspect Elements For One Process Instance (Priority: P1) MVP

**Goal**: Operators can fetch one known process instance with its runtime element instances by running `c8volt get pi --key <key> --with-elements`.

**Independent Test**: Run keyed process-instance command tests and service/facade enrichment tests with one selected process instance, matching elements, active elements without end dates, and incident markers.

### Tests for User Story 1

- [ ] T014 [P] [US1] Add service tests for element attachment, per-key filtering, process-instance order, element sorting, and search error propagation in `internal/services/processinstance/enrichment_test.go`
- [ ] T015 [P] [US1] Add process facade tests for element enrichment conversion and error mapping in `c8volt/process/client_test.go`
- [ ] T016 [P] [US1] Add keyed command validation tests for `--with-elements`, `--with-elements --total`, `--keys-only --with-elements`, keyed search-filter conflicts, and Camunda 8.7 unsupported behavior in `cmd/get_processinstance_test.go`
- [ ] T017 [P] [US1] Add keyed activity rendering tests for `elements:`, active rows without `e:`, incident markers, aligned element columns, and no `element:<id>` suffix in `cmd/cmd_views_get_test.go`

### Implementation for User Story 1

- [ ] T018 [US1] Implement `elementSearcher` and `EnrichProcessInstancesWithElements` in `internal/services/processinstance/enrichment.go`
- [ ] T019 [US1] Implement process facade element enrichment delegation and conversion in `c8volt/process/client.go` and `c8volt/process/convert.go`
- [ ] T020 [US1] Implement the command activity wrapper for element enrichment in `cmd/get_processinstance_enrichment.go`
- [ ] T021 [US1] Add the `--with-elements` flag, help text, examples, and keyed validation in `cmd/get_processinstance.go` and `cmd/get_processinstance_validation.go`
- [ ] T022 [US1] Invoke element enrichment for keyed `get pi --key <key> --with-elements` in `cmd/get_processinstance.go`
- [ ] T023 [US1] Render nested runtime element rows under `elements:` in `cmd/cmd_views_processinstance_activity.go`

**Checkpoint**: User Story 1 is fully functional and independently testable as the MVP.

---

## Phase 4: User Story 2 - Attach Elements To Process Instance Search Results (Priority: P2)

**Goal**: Operators can attach elements to every selected process instance in bounded list/search output without changing process-instance paging, limits, prompts, or filters.

**Independent Test**: Run process-instance search command tests using `--with-elements`, `--limit`, `--batch-size`, active-state filters, and BPMN process filters, then verify selected process-instance counts and prompts are unchanged by element row counts.

### Tests for User Story 2

- [ ] T024 [P] [US2] Add list/search command tests for `--state active --limit 5 --with-elements` and BPMN process selector enrichment in `cmd/get_processinstance_test.go`
- [ ] T025 [P] [US2] Add incremental paging tests proving `--batch-size`, prompts, and `found: N` remain process-instance scoped with `--with-elements` in `cmd/get_processinstance_test.go`
- [ ] T026 [P] [US2] Add bounded JSON search tests proving process-instance limits are preserved while attached elements are included in `cmd/get_processinstance_test.go`
- [ ] T027 [P] [US2] Add command activity tests for repeated or looped BPMN elements rendering as separate rows in `cmd/cmd_views_get_test.go`

### Implementation for User Story 2

- [ ] T028 [US2] Apply element enrichment to bounded list/search result aggregation in `cmd/get_processinstance.go`
- [ ] T029 [US2] Apply element enrichment to incremental one-line/page rendering paths in `cmd/get_processinstance_search.go`
- [ ] T030 [US2] Preserve process-instance page counts, limits, and continuation prompts while rendering enriched rows in `cmd/get_processinstance_paging.go` and `cmd/get_processinstance_search.go`
- [ ] T031 [US2] Ensure process-instance filters remain authoritative and element-specific filters are not added in `cmd/get_processinstance.go` and `cmd/get_processinstance_validation.go`
- [ ] T032 [US2] Surface reused element-service Camunda 8.7 unsupported errors as command failures in `cmd/get_processinstance.go` and `c8volt/process/client.go`

**Checkpoint**: User Stories 1 and 2 both work independently, with list/search preserving process-instance selection semantics.

---

## Phase 5: User Story 3 - Combine Elements With Existing Enrichment (Priority: P3)

**Goal**: Operators can combine `--with-elements` with `--with-vars` and `--with-incidents` while human and JSON output remain stable and script-safe.

**Independent Test**: Run keyed and bounded process-instance output tests with `--with-vars`, `--with-incidents`, and `--with-elements` in combination, then verify section order, no duplicate sections, and JSON payload fields.

### Tests for User Story 3

- [ ] T033 [P] [US3] Add combined human output tests for `vars:`, `incidents:`, and `elements:` section order in `cmd/cmd_views_get_test.go`
- [ ] T034 [P] [US3] Add combined JSON payload tests for `variables`, `incidents`, and `elements` fields in `cmd/cmd_views_get_test.go`
- [ ] T035 [P] [US3] Add command integration tests for keyed combined enrichment and bounded list/search combined enrichment in `cmd/get_processinstance_test.go`
- [ ] T036 [P] [US3] Add command contract tests for the `--with-elements` flag, help metadata, output modes, examples, and read-only capability in `cmd/command_contract_test.go`

### Implementation for User Story 3

- [ ] T037 [US3] Extend process-instance activity view models with attached elements for human and JSON output in `cmd/cmd_views_processinstance_activity.go`
- [ ] T038 [US3] Replace incident/variable-only merge logic with combined activity merge support for variables, incidents, and elements in `cmd/cmd_views_processinstance_activity.go`
- [ ] T039 [US3] Orchestrate all requested enrichment combinations in keyed mode without duplicate facade calls in `cmd/get_processinstance.go`
- [ ] T040 [US3] Orchestrate all requested enrichment combinations in list/search mode without duplicate facade calls in `cmd/get_processinstance.go` and `cmd/get_processinstance_search.go`
- [ ] T041 [US3] Ensure JSON output uses the shared command envelope and includes attached element fields from `c8volt/process/model.go` in `cmd/cmd_views_processinstance_activity.go`
- [ ] T042 [US3] Keep keys-only invalid for element enrichment while preserving existing keys-only behavior without `--with-elements` in `cmd/get_processinstance_validation.go`

**Checkpoint**: All enrichment combinations are independently functional in human and JSON output.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, generated artifacts, validation, and cleanup across all user stories.

- [ ] T043 [P] Update README process-instance examples and behavior notes for `--with-elements` in `README.md`
- [ ] T044 [P] Update docs generator or metadata expectations for `--with-elements` command documentation in `docsgen/main_test.go` and `cmd/command_contract_test.go`
- [ ] T045 [P] Update quickstart examples if implementation output wording changes in `specs/235-get-pi-elements/quickstart.md`
- [ ] T046 Run `gofmt` on touched Go files in `cmd/`, `c8volt/process/`, `c8volt/client.go`, `internal/domain/`, and `internal/services/processinstance/`
- [ ] T047 Run targeted service and facade validation for element enrichment tests in `internal/services/processinstance/enrichment_test.go` and `c8volt/process/client_test.go`
- [ ] T048 Run targeted command validation for element enrichment tests in `cmd/get_processinstance_test.go`, `cmd/cmd_views_get_test.go`, and `cmd/command_contract_test.go`
- [ ] T049 Run generated documentation validation and regenerate CLI docs with `docsgen/main_test.go` and `Makefile`
- [ ] T050 Run full repository validation with `make test` using `Makefile`
- [ ] T051 Verify all manual scenarios from `specs/235-get-pi-elements/quickstart.md` against the built binary `/tmp/c8volt-get-pi-elements`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 Setup**: No dependencies.
- **Phase 2 Foundational**: Depends on Phase 1 context review and blocks all user stories.
- **Phase 3 US1**: Depends on Phase 2 and delivers the MVP.
- **Phase 4 US2**: Depends on Phase 2; recommended after US1 because it extends the same command paths.
- **Phase 5 US3**: Depends on US1 and US2 behavior for complete combined output validation.
- **Phase 6 Polish**: Depends on all desired user stories.

### User Story Dependencies

- **US1 (P1)**: Requires foundational models, facade wiring, and command stub support; no dependency on US2 or US3.
- **US2 (P2)**: Requires foundational models and service enrichment; can be tested independently with search/list mocks, but should follow US1 to reduce same-file conflicts.
- **US3 (P3)**: Requires the individual enrichment paths from US1 and US2 to validate all combinations.

### Within Each User Story

- Write tests before implementation tasks and verify they fail for the missing behavior.
- Domain and public models before facade delegation.
- Service enrichment before command orchestration.
- Command orchestration before renderer and documentation validation.

## Parallel Opportunities

- Setup inspections T002, T003, and T004 can run in parallel because they read separate source areas.
- Foundational conversion task T008 can run in parallel with command stub task T012 after model names are agreed.
- US1 tests T014-T017 can be written in parallel because they target service, facade, validation, and rendering files separately.
- US2 tests T024-T027 can be written in parallel because they cover separate command scenarios and renderer behavior.
- US3 tests T033-T036 can be written in parallel because they target human output, JSON output, command integration, and contract metadata separately.
- Polish documentation tasks T043-T045 can run in parallel once command wording stabilizes.

## Parallel Example: User Story 1

```text
Task: "T014 [US1] Add service tests for element attachment in internal/services/processinstance/enrichment_test.go"
Task: "T015 [US1] Add process facade tests in c8volt/process/client_test.go"
Task: "T016 [US1] Add keyed command validation tests in cmd/get_processinstance_test.go"
Task: "T017 [US1] Add keyed activity rendering tests in cmd/cmd_views_get_test.go"
```

## Parallel Example: User Story 2

```text
Task: "T024 [US2] Add list/search command tests in cmd/get_processinstance_test.go"
Task: "T026 [US2] Add bounded JSON search tests in cmd/get_processinstance_test.go"
Task: "T027 [US2] Add repeated element rendering tests in cmd/cmd_views_get_test.go"
```

## Parallel Example: User Story 3

```text
Task: "T033 [US3] Add combined human output tests in cmd/cmd_views_get_test.go"
Task: "T034 [US3] Add combined JSON payload tests in cmd/cmd_views_get_test.go"
Task: "T035 [US3] Add command integration tests in cmd/get_processinstance_test.go"
Task: "T036 [US3] Add command contract tests in cmd/command_contract_test.go"
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 setup.
2. Complete Phase 2 foundational contracts and wiring.
3. Complete Phase 3 keyed element enrichment.
4. Validate keyed `c8volt get pi --key <process-instance-key> --with-elements` and Camunda 8.7 unsupported behavior.

### Incremental Delivery

1. Deliver US1 keyed enrichment as the MVP.
2. Add US2 bounded list/search enrichment without changing process-instance paging semantics.
3. Add US3 combined enrichment output and JSON contract support.
4. Finish documentation, generated docs, quickstart validation, and full repository validation.

### Ralph Execution

1. Launch Ralph only after confirming it receives `--implementation-context specs/ralph-implementation-rules.md`.
2. Keep each Ralph iteration to one story or validation slice from this task file.
3. Do not stage or commit until that iteration's validation passes.

### Validation Strategy

1. Run targeted package tests after each story checkpoint.
2. Run `make docs-content` once command metadata is final.
3. Run `make test` before merge, per constitution.

## Notes

- Reuse `c8volt/element` and `internal/services/element`; do not duplicate runtime element lookup mechanics in `cmd`.
- Keep `cmd` responsible only for flags, validation, facade calls, and rendering.
- Keep `--limit`, `--batch-size`, prompts, and totals scoped to process instances.
- Do not add process-instance element filter flags; direct element filtering belongs to `get element` / `get ei`.
- Do not hand-edit generated Camunda clients under `internal/clients/camunda`.
