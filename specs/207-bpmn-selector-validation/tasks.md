# Tasks: BPMN Selector Validation for Operational Commands

**Input**: Design documents from `/specs/207-bpmn-selector-validation/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are required by the feature specification and constitution. Story test tasks should be written before implementation and should fail until the story implementation is complete.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files or only adds tests/docs
- **[Story]**: Maps to the user story from [spec.md](./spec.md)
- Every task names exact repository paths

## Phase 1: Setup (Shared Discovery)

**Purpose**: Confirm the current command paths, selector registration, shared validator behavior, paging boundaries, and documentation surfaces before changing behavior.

- [x] T001 Audit every direct `--bpmn-process-id` registration in `cmd/get_processinstance_filtering.go`, `cmd/run_processinstance.go`, `cmd/cancel_processinstance.go`, `cmd/delete_processinstance.go`, `cmd/get_incident.go`, `cmd/get_processdefinition.go`, and `cmd/delete_processdefinition.go`
- [x] T002 [P] Inspect shared selector validation behavior and reusable gaps in `cmd/process_definition_selector_validation.go` and `cmd/process_definition_selector_validation_test.go`
- [x] T003 [P] Inspect process-instance search-selected mutation paths in `cmd/cancel_processinstance.go`, `cmd/delete_processinstance.go`, `cmd/get_processinstance_paging.go`, and `cmd/get_processinstance_search.go`
- [x] T004 [P] Inspect incident search filters, paging, totals, and output modes in `cmd/get_incident.go`, `cmd/get_incident_search.go`, and `cmd/get_incident_test.go`
- [x] T005 [P] Inspect direct process-definition search/delete behavior and tests in `cmd/get_processdefinition.go`, `cmd/get_processdefinition_test.go`, `cmd/delete_processdefinition.go`, and `cmd/delete_test.go`
- [x] T006 [P] Inspect README and generated documentation surfaces for affected command wording in `README.md`, `docs/index.md`, and `docs/cli/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Generalize the existing selector validation helper so all aligned direct BPMN selector commands can share diagnostics, prompt policy, and test fixtures.

**Critical**: No user story implementation should begin until this phase is complete.

- [x] T007 Extend or add shared single-selector request construction in `cmd/process_definition_selector_validation.go` for non-PI commands that directly accept one BPMN process ID
- [x] T008 Add reusable validation entry points that return local precondition errors without recovery prompts when command mode forbids prompting in `cmd/process_definition_selector_validation.go`
- [x] T009 [P] Add helper tests for single-selector request construction, version/tag narrowing, and no-prompt modes in `cmd/process_definition_selector_validation_test.go`
- [x] T010 [P] Add or update command test stubs to prove selector validation happens before resource paging by failing if search methods are called in `cmd/process_api_stub_test.go` or existing command test stubs
- [x] T011 Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*ProcessDefinitionSelector' -count=1` and fix foundational compile/test failures

**Checkpoint**: Shared validation can be reused by process-instance, incident, and direct process-definition commands without changing keyed or non-BPMN flows.

---

## Phase 3: User Story 1 - Block no-op mutations from missing BPMN selectors (Priority: P1) MVP

**Goal**: `cancel pi` and `delete pi` validate direct BPMN selectors before process-instance search paging, dry-run planning, confirmation, or mutation.

**Independent Test**: Run `cancel pi -b <missing>` and `delete pi -b <missing>` in command tests and verify both fail with the shared missing visible process-definition diagnostic before any process-instance search or mutation planning.

### Tests for User Story 1

- [x] T012 [P] [US1] Add `cancel pi --bpmn-process-id <missing>` test proving validation fails before process-instance search paging in `cmd/cancel_test.go`
- [x] T013 [P] [US1] Add `delete pi --bpmn-process-id <missing>` test proving validation fails before process-instance search paging or delete planning in `cmd/delete_test.go`
- [x] T014 [P] [US1] Add valid visible selector with zero matching process instances tests preserving existing no-op/empty behavior in `cmd/cancel_test.go` and `cmd/delete_test.go`
- [x] T015 [P] [US1] Add machine/non-interactive mode tests for `--json`, `--automation`, and key-only-equivalent output where applicable in `cmd/cancel_test.go` and `cmd/delete_test.go`

### Implementation for User Story 1

- [x] T016 [US1] Invoke shared BPMN selector validation before `processPISearchPagesWithAction` in the search-selected path of `cmd/cancel_processinstance.go`
- [x] T017 [US1] Invoke shared BPMN selector validation before `deleteProcessInstanceSearchPages` in the search-selected path of `cmd/delete_processinstance.go`
- [x] T018 [US1] Preserve keyed, stdin key, non-BPMN search, dry-run, auto-confirm, and valid `found: 0` behavior in `cmd/cancel_processinstance.go` and `cmd/delete_processinstance.go`
- [x] T019 [US1] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test(Cancel|Delete).*Bpmn|Test(Cancel|Delete).*ProcessDefinitionSelector' -count=1` and fix regressions

**Checkpoint**: User Story 1 is independently complete when direct mutating PI commands no longer turn missing BPMN selectors into successful no-ops.

---

## Phase 4: User Story 2 - Validate incident searches by BPMN selector (Priority: P2)

**Goal**: `get incident -b <missing>` validates the BPMN selector before incident search paging, totals, key-only output, or process-instance-key-only output.

**Independent Test**: Run `get incident -b <missing>` in command tests and verify the shared selector diagnostic appears before any incident search call.

### Tests for User Story 2

- [x] T020 [P] [US2] Add `get incident --bpmn-process-id <missing>` test proving validation fails before incident search paging in `cmd/get_incident_test.go`
- [x] T021 [P] [US2] Add visible selector with zero matching incidents test preserving empty incident output in `cmd/get_incident_test.go`
- [x] T022 [P] [US2] Add `--total`, `--keys-only`, `--pi-keys-only`, `--json`, and `--automation` no-prompt validation tests where compatible in `cmd/get_incident_test.go`
- [x] T023 [P] [US2] Add version/tag/tenant selector context coverage for incident BPMN validation in `cmd/get_incident_test.go` or `cmd/process_definition_selector_validation_test.go`

### Implementation for User Story 2

- [x] T024 [US2] Build and invoke shared BPMN selector validation before `searchIncidentsTotal` and `searchIncidentsWithPaging` when `flagGetIncidentBpmnProcessID` is set in `cmd/get_incident.go`
- [x] T025 [US2] Preserve keyed incident mode, `--pd-key`, non-BPMN incident filters, paging continuation, totals, and key-only rendering in `cmd/get_incident.go` and `cmd/get_incident_search.go`
- [x] T026 [US2] Update `get incident` help text to describe BPMN selector validation without changing unrelated incident examples in `cmd/get_incident.go`
- [x] T027 [US2] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestGetIncident.*(Bpmn|Selector|Total|KeysOnly|Automation|JSON)' -count=1` and fix regressions

**Checkpoint**: User Story 2 is independently complete when incident search distinguishes missing BPMN selectors from legitimate empty incident results.

---

## Phase 5: User Story 3 - Audit direct process-definition selectors (Priority: P3)

**Goal**: `get pd -b` and `delete pd -b` have explicit missing-selector behavior, with the preferred shared diagnostic when no visible process definition matches.

**Independent Test**: Run `get pd -b <missing>` and `delete pd -b <missing>` command tests and verify each outcome is explicit, documented, and occurs before delete impact planning or mutation.

### Tests for User Story 3

- [x] T028 [P] [US3] Add `get pd --bpmn-process-id <missing>` test for explicit missing-selector behavior in `cmd/get_processdefinition_test.go`
- [x] T029 [P] [US3] Add `delete pd --bpmn-process-id <missing>` test proving failure before delete impact planning in `cmd/delete_test.go`
- [x] T030 [P] [US3] Add valid visible selector tests preserving existing `get pd` listing and `delete pd` preview/confirmation behavior in `cmd/get_processdefinition_test.go` and `cmd/delete_test.go`

### Implementation for User Story 3

- [x] T031 [US3] Align direct BPMN search misses in `runSearchProcessDefinitions` with the explicit selector diagnostic when `flagGetPDBpmnProcessId` is set in `cmd/get_processdefinition.go`
- [x] T032 [US3] Align direct BPMN delete misses before impact planning when `flagDeletePDBpmnProcessId` is set in `cmd/delete_processdefinition.go`
- [x] T033 [US3] Preserve broad `get pd`, keyed `get pd`, keyed/stdin `delete pd`, `--latest`, version, tag, and XML compatibility behavior in `cmd/get_processdefinition.go` and `cmd/delete_processdefinition.go`
- [x] T034 [US3] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test(Get|Delete)ProcessDefinition.*(Bpmn|Selector|Missing|Latest)' -count=1` and fix regressions

**Checkpoint**: User Story 3 is independently complete when direct process-definition commands have tested missing BPMN selector semantics.

---

## Phase 6: User Story 4 - Preserve pipelines and documented command contracts (Priority: P4)

**Goal**: Pipeline workflows keep validation upstream, machine modes stay non-interactive, and help/docs/contract output match the final command behavior.

**Independent Test**: Verify pipeline tests, command contract tests, generated docs, and quickstart scenarios after user stories 1-3 are complete.

### Tests and Documentation for User Story 4

- [ ] T035 [P] [US4] Add or update pipeline-boundary tests proving `get pi -b <missing> --keys-only | cancel pi -` validates upstream while downstream keyed commands remain key-only in `cmd/get_processinstance_test.go`, `cmd/cancel_test.go`, or `cmd/delete_test.go`
- [ ] T036 [P] [US4] Add or update command contract tests for aligned command help, mutation metadata, automation notes, and output mode expectations in `cmd/command_contract_test.go`
- [ ] T037 [P] [US4] Update user-facing selector guidance and examples in `README.md`
- [ ] T038 [P] [US4] Update docs source guidance and examples in `docs/index.md` and relevant files under `docs/ops/`
- [ ] T039 [US4] Run `make docs-content` and review generated updates under `docs/cli/`
- [ ] T040 [US4] Review [quickstart.md](./quickstart.md) against the implemented behavior and adjust validation commands if paths or test names changed

### Final Validation

- [ ] T041 Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process ./c8volt/incident ./internal/services/processdefinition ./internal/services/incident -count=1` and fix regressions
- [ ] T042 Run `make test` and fix repository validation failures
- [ ] T043 Verify `rg -n "found: 0|no visible process definition|bpmn-process-id" README.md docs cmd` shows documentation and help wording consistent with the implemented behavior
- [ ] T044 Verify `git diff` contains only issue #207 implementation, docs, generated docs, and Speckit artifacts before commit

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on setup and blocks all user stories.
- **US1 (Phase 3)**: Depends on shared validation helpers and delivers the MVP.
- **US2 (Phase 4)**: Depends on shared validation helpers and can proceed after US1 proves command-level reuse.
- **US3 (Phase 5)**: Depends on shared diagnostics and direct process-definition audit decisions.
- **US4 (Phase 6)**: Depends on final command behavior from US1-US3.

### User Story Dependencies

- **User Story 1 (P1)**: First because it fixes the confirmed mutating PI gaps.
- **User Story 2 (P2)**: Next because incident search has the same selector ambiguity and feeds operational pipelines.
- **User Story 3 (P3)**: Follows because direct process-definition commands need explicit audit/diagnostic alignment.
- **User Story 4 (P4)**: Last because docs and command contracts should reflect final behavior.

### Parallel Opportunities

- T002 through T006 can run in parallel during setup.
- T009 and T010 can run in parallel during foundational test work.
- T012 through T015 can be written in parallel for US1.
- T020 through T023 can be written in parallel for US2.
- T028 through T030 can be written in parallel for US3.
- T035 through T038 can be done in parallel after command semantics stabilize.

## Parallel Example: User Story 2

```text
Task: "Add get incident --bpmn-process-id <missing> test proving validation fails before incident search paging in cmd/get_incident_test.go"
Task: "Add visible selector with zero matching incidents test preserving empty incident output in cmd/get_incident_test.go"
Task: "Add --total, --keys-only, --pi-keys-only, --json, and --automation no-prompt validation tests where compatible in cmd/get_incident_test.go"
Task: "Add version/tag/tenant selector context coverage for incident BPMN validation in cmd/get_incident_test.go or cmd/process_definition_selector_validation_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete User Story 1 to fix the confirmed mutating process-instance gaps.
3. Stop and validate User Story 1 independently with targeted command tests.

### Incremental Delivery

1. Add User Story 2 for incident search selector validation.
2. Add User Story 3 for direct process-definition selector audit and explicit missing behavior.
3. Add User Story 4 for pipeline boundary proof, docs, generated docs, and final validation.
4. Run targeted tests, docs generation, and `make test`.

### Commit Guidance

Every commit subject for this workflow must follow Conventional Commits and end with `#207`, for example `fix(cli): validate BPMN selectors consistently #207`.
