# Tasks: Validate Process Definition Selectors for Process-Instance Commands

**Input**: Design documents from `/specs/175-validate-pd-selectors/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are required by the feature specification and constitution. Story test tasks should be written before implementation and should fail until the story implementation is complete.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files or only adds tests/docs
- **[Story]**: Maps to the user story from [spec.md](./spec.md)
- Every task names exact repository paths

## Phase 1: Setup (Shared Discovery)

**Purpose**: Confirm current command paths, selector mapping, prompt policy, process-definition facade behavior, and documentation surfaces before changing behavior.

- [x] T001 Inspect shared process-instance BPMN/version/tag flags and search filter construction in `cmd/get_processinstance.go`
- [x] T002 [P] Inspect process-instance search/mutation paging paths in `cmd/get_processinstance.go`, `cmd/cancel_processinstance.go`, and `cmd/delete_processinstance.go`
- [x] T003 [P] Inspect `run pi` BPMN process ID and version validation in `cmd/run_processinstance.go`
- [x] T004 [P] Inspect process-definition search facade and conversion behavior in `c8volt/process/api.go`, `c8volt/process/client.go`, `c8volt/process/convert.go`, and `c8volt/process/model.go`
- [x] T005 [P] Inspect process-definition list rendering and tests in `cmd/get_processdefinition.go`, `cmd/cmd_views_get.go`, and `cmd/get_test.go`
- [x] T006 [P] Inspect automation, JSON, keys-only, and prompt helpers in `cmd/` before adding the listing offer

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared selector validation types/helpers and facade coverage that every story can reuse.

**Critical**: No user story implementation should begin until this phase is complete.

- [x] T007 Add shared process-definition selector validation request/result types and helper skeletons in `cmd/get_processinstance.go` or a new `cmd/process_definition_selector_validation.go`
- [x] T008 Add helper logic that maps process-instance BPMN/version/tag flags to `process.ProcessDefinitionFilter` in `cmd/get_processinstance.go` or `cmd/process_definition_selector_validation.go`
- [x] T009 Add helper logic that validates one or more BPMN process IDs through `process.API.SearchProcessDefinitions` or `SearchProcessDefinitionsLatest` in `cmd/process_definition_selector_validation.go`
- [x] T010 Add reusable missing-selector diagnostic formatting and no-prompt error behavior in `cmd/process_definition_selector_validation.go`
- [x] T011 [P] Add process facade tests proving `SearchProcessDefinitions` receives BPMN process ID, version, and version tag filters in `c8volt/process/client_test.go`
- [x] T012 [P] Add command unit tests for selector-to-filter construction and missing selector formatting in `cmd/get_processinstance_test.go` or a new `cmd/process_definition_selector_validation_test.go`
- [x] T013 Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process -run 'Test.*ProcessDefinitionSelector|TestClient_SearchProcessDefinitions' -count=1` and fix foundational compile/test failures

**Checkpoint**: Shared validation compiles, maps selector fields correctly, and can report missing selectors without touching process-instance work.

---

## Phase 3: User Story 1 - Detect Missing Selectors Before Empty Results (Priority: P1) MVP

**Goal**: `get pi --bpmn-process-id <missing>` validates visible process definitions before process-instance search and no longer reports only `found: 0`.

**Independent Test**: Run `c8volt get pi --bpmn-process-id <missing>` and verify c8volt reports that no visible process definition matches before any process-instance empty-result output.

### Tests for User Story 1

- [x] T014 [P] [US1] Add command test for `get pi --bpmn-process-id <missing>` failing before process-instance search in `cmd/get_processinstance_test.go`
- [x] T015 [P] [US1] Add command test for visible process definition with zero matching process instances preserving `found: 0` in `cmd/get_processinstance_test.go`
- [x] T016 [P] [US1] Add command test proving `--pd-version`, `--pd-version-tag`, and tenant options are included in validation context in `cmd/get_processinstance_test.go`

### Implementation for User Story 1

- [x] T017 [US1] Invoke shared selector validation before `get pi` process-instance search when `--bpmn-process-id` is set in `cmd/get_processinstance.go`
- [x] T018 [US1] Ensure successful validation allows existing `searchProcessInstancesWithPaging` and `found: 0` behavior to continue unchanged in `cmd/get_processinstance.go`
- [x] T019 [US1] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestGetProcessInstance.*(Bpmn|Selector|Found0)' -count=1` and fix regressions

**Checkpoint**: User Story 1 is independently complete when `get pi` separates missing process definitions from valid empty process-instance results.

---

## Phase 4: User Story 2 - Provide Safe Human and Automation Diagnostics (Priority: P2)

**Goal**: Selector validation failures render helpful human diagnostics with an optional visible-definition listing prompt, while automation-oriented modes fail without prompting.

**Independent Test**: Exercise missing selectors in human interactive output and in `--json`, `--automation`, `--keys-only`, and non-TTY modes; verify only prompt-eligible human output offers visible process-definition listing.

### Tests for User Story 2

- [ ] T020 [P] [US2] Add human-output diagnostic tests for a single missing selector in `cmd/get_processinstance_test.go` or `cmd/process_definition_selector_validation_test.go`
- [ ] T021 [P] [US2] Add human-output diagnostic tests for multiple missing selectors in `cmd/process_definition_selector_validation_test.go`
- [ ] T022 [P] [US2] Add tests proving `--json`, `--automation`, `--keys-only`, and non-TTY execution do not prompt in `cmd/get_processinstance_test.go`
- [ ] T023 [P] [US2] Add test proving accepted interactive listing uses existing process-definition list rendering in `cmd/process_definition_selector_validation_test.go` or `cmd/get_test.go`

### Implementation for User Story 2

- [ ] T024 [US2] Implement prompt eligibility checks for visible-definition listing in `cmd/process_definition_selector_validation.go`
- [ ] T025 [US2] Implement interactive listing through existing process-definition search/list view helpers in `cmd/process_definition_selector_validation.go` and `cmd/get_processdefinition.go`
- [ ] T026 [US2] Ensure structured and automation modes return clear errors without prompt text in `cmd/process_definition_selector_validation.go`
- [ ] T027 [US2] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*Selector.*(Diagnostic|Prompt|JSON|Automation|KeysOnly|TTY)' -count=1` and fix regressions

**Checkpoint**: User Story 2 is independently complete when humans get recovery guidance and scripts never block.

---

## Phase 5: User Story 3 - Guard Mutating Process-Instance Commands (Priority: P3)

**Goal**: `cancel pi` and `delete pi` validate `--bpmn-process-id` selectors before search-selected mutation planning and never treat a missing process definition as a successful no-op.

**Independent Test**: Run `cancel pi --bpmn-process-id <missing>` and `delete pi --bpmn-process-id <missing>` in non-interactive mode and verify each fails before mutation planning.

### Tests for User Story 3

- [ ] T028 [P] [US3] Add `cancel pi --bpmn-process-id <missing>` validation test in `cmd/cancel_test.go`
- [ ] T029 [P] [US3] Add `delete pi --bpmn-process-id <missing>` validation test in `cmd/delete_test.go`
- [ ] T030 [P] [US3] Add tests proving visible process definition with zero matching process instances preserves existing searched no-op behavior in `cmd/cancel_test.go` and `cmd/delete_test.go`

### Implementation for User Story 3

- [ ] T031 [US3] Invoke shared selector validation before `cancel pi` search-selected paging when `--bpmn-process-id` is set in `cmd/cancel_processinstance.go`
- [ ] T032 [US3] Invoke shared selector validation before `delete pi` search-selected paging when `--bpmn-process-id` is set in `cmd/delete_processinstance.go`
- [ ] T033 [US3] Verify keyed `--key` cancellation/deletion paths and non-BPMN searches remain unchanged in `cmd/cancel_processinstance.go` and `cmd/delete_processinstance.go`
- [ ] T034 [US3] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test(Cancel|Delete).*ProcessDefinitionSelector|Test(Cancel|Delete).*Bpmn' -count=1` and fix regressions

**Checkpoint**: User Story 3 is independently complete when mutating process-instance commands fail before mutation for missing BPMN selectors.

---

## Phase 6: User Story 4 - Prevent Partial Multi-ID Starts (Priority: P4)

**Goal**: `run pi` validates every BPMN process ID before creating process instances so mixed valid/missing requests cannot partially start work.

**Independent Test**: Run `run pi --bpmn-process-id <existing> --bpmn-process-id <missing>` and verify no process instances are created and the diagnostic lists the missing ID.

### Tests for User Story 4

- [ ] T035 [P] [US4] Add `run pi` test proving multiple BPMN IDs are all validated before any create request in `cmd/run_test.go`
- [ ] T036 [P] [US4] Add `run pi` test proving multiple missing BPMN IDs are all listed in the diagnostic in `cmd/run_test.go`
- [ ] T037 [P] [US4] Add `run pi` test proving all-visible BPMN IDs preserve existing create behavior in `cmd/run_test.go`
- [ ] T038 [P] [US4] Add `run pi --pd-version` validation test for single BPMN ID exact-version selector behavior in `cmd/run_test.go`

### Implementation for User Story 4

- [ ] T039 [US4] Build a run-specific selector validation request from `flagRunPIProcessDefinitionBpmnProcessIds` and `flagRunPIProcessDefinitionVersion` in `cmd/run_processinstance.go`
- [ ] T040 [US4] Invoke shared selector validation before constructing or submitting `ProcessInstanceData` for BPMN process ID starts in `cmd/run_processinstance.go`
- [ ] T041 [US4] Use latest-definition validation for `run pi` when `--pd-version` is absent and exact-version validation when it is present in `cmd/run_processinstance.go`
- [ ] T042 [US4] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestRunProcessInstance.*(Bpmn|Selector|Partial|Version)' -count=1` and fix regressions

**Checkpoint**: User Story 4 is independently complete when `run pi` has all-or-nothing BPMN selector validation.

---

## Phase 7: Documentation & Validation

**Purpose**: Finish user-facing docs, generated docs, and repository-wide proof.

- [ ] T043 [P] Update process-instance examples and missing BPMN selector wording in `README.md`
- [ ] T044 [P] Update site documentation source examples in `docs/index.md`
- [ ] T045 Add or update command contract/help tests for selector diagnostics and prompt policy in `cmd/command_contract_test.go` or `cmd/get_processinstance_test.go`
- [ ] T046 Run `make docs-content` and fix generated documentation issues under `docs/cli/`
- [ ] T047 Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process ./internal/services/processdefinition/v87 ./internal/services/processdefinition/v88 ./internal/services/processdefinition/v89 -count=1` and fix regressions
- [ ] T048 Run `make test` and fix repository validation failures
- [ ] T049 [P] Review [quickstart.md](./quickstart.md) against implemented behavior and update if command examples or validation commands changed
- [ ] T050 Verify `git diff` contains only issue #175 implementation, docs, generated docs, and Speckit artifacts before commit

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on setup and blocks all user stories.
- **US1 (Phase 3)**: Depends on foundational selector validation and delivers the MVP.
- **US2 (Phase 4)**: Depends on foundational diagnostics and may be completed after or alongside US1 implementation details.
- **US3 (Phase 5)**: Depends on US1 validation behavior and shared diagnostics.
- **US4 (Phase 6)**: Depends on foundational multi-ID validation support and shared diagnostics.
- **Documentation & Validation (Phase 7)**: Depends on final command semantics from US1-US4.

### User Story Dependencies

- **User Story 1 (P1)**: First user-visible slice; proves missing selector versus valid empty result behavior for `get pi`.
- **User Story 2 (P2)**: Builds on shared failure diagnostics and prompt policy; supports all later command paths.
- **User Story 3 (P3)**: Extends shared validation to cancel/delete mutation workflows.
- **User Story 4 (P4)**: Extends shared validation to run/start workflows with all-or-nothing multi-ID behavior.

### Parallel Opportunities

- T002 through T006 can run in parallel during setup.
- T011 and T012 can be written in parallel during foundational work.
- T014 through T016 can be written in parallel for US1.
- T020 through T023 can be written in parallel for US2.
- T028 through T030 can be written in parallel for US3.
- T035 through T038 can be written in parallel for US4.
- T043 and T044 can be updated in parallel after command behavior is stable.
- T049 can run in parallel with final diff review.

## Parallel Example: User Story 1

```text
Task: "Add command test for get pi --bpmn-process-id <missing> failing before process-instance search in cmd/get_processinstance_test.go"
Task: "Add command test for visible process definition with zero matching process instances preserving found: 0 in cmd/get_processinstance_test.go"
Task: "Add command test proving version, version tag, and tenant selector context in cmd/get_processinstance_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete User Story 1 to deliver corrected `get pi` behavior.
3. Stop and validate User Story 1 independently with targeted command and facade tests.

### Incremental Delivery

1. Add User Story 2 for human recovery prompts and no-prompt automation behavior.
2. Add User Story 3 for cancel/delete mutation guards.
3. Add User Story 4 for all-or-nothing `run pi` validation.
4. Complete docs and generated CLI documentation.
5. Run targeted tests, docs generation, and `make test`.

### Commit Guidance

Every commit subject for this workflow must follow Conventional Commits and end with `#175`, for example `fix(pi): validate process definition selectors #175`.
