# Tasks: Run Confirmation Observes Real Process Instance States

**Input**: Design documents from `specs/225-run-observable-keys/`
**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/), [quickstart.md](./quickstart.md)
**Issue**: [#225](https://github.com/grafvonb/c8volt/issues/225)
**Implementation Context**: Every Ralph implementation iteration must read and apply `specs/ralph-implementation-rules.md`; launch Ralph only with `--implementation-context specs/ralph-implementation-rules.md`.

**Tests**: Required by issue acceptance criteria and repository constitution.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with other marked tasks in the same phase after dependencies are met
- **[Story]**: Which user story this task belongs to
- Every task includes exact file paths

## Phase 1: Setup (Shared Context)

**Purpose**: Establish Ralph-ready context and record reusable codebase discoveries before implementation.

- [x] T001 Read `specs/ralph-implementation-rules.md`, `specs/225-run-observable-keys/spec.md`, `specs/225-run-observable-keys/plan.md`, `specs/225-run-observable-keys/tasks.md`, and `specs/225-run-observable-keys/progress.md`
- [x] T002 Record initial codebase pattern notes and current work-unit status in `specs/225-run-observable-keys/progress.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add the shared creation-confirmation concept used by all process-instance service versions.

**Critical**: No user story implementation should begin until this phase is complete.

- [x] T003 Add a documented helper for observable process-instance creation confirmation states in `internal/domain/state.go`
- [x] T004 [P] Add or update state helper tests for observable creation states in `internal/domain/state_test.go`

**Checkpoint**: Shared confirmation state semantics are available to all versioned services.

---

## Phase 3: User Story 1 - Confirm Fast Process Instance Runs (Priority: P1) MVP

**Goal**: Run-style creation succeeds when the created process instance is observable as `ACTIVE`, `COMPLETED`, `CANCELED`, or `TERMINATED`, and still fails for absent/not-found or unknown observations.

**Independent Test**: Service tests can create a process instance whose first lookup returns `COMPLETED` and verify creation succeeds without waiting for `ACTIVE`.

### Tests for User Story 1

- [ ] T005 [P] [US1] Add v8.7 creation confirmation tests for `ACTIVE`, `COMPLETED`, terminal observable state, and absent/not-found rejection in `internal/services/processinstance/v87/service_test.go`
- [ ] T006 [P] [US1] Add v8.8 creation confirmation tests for `ACTIVE`, `COMPLETED`, terminal observable state, and absent/not-found rejection in `internal/services/processinstance/v88/service_test.go`
- [ ] T007 [P] [US1] Add v8.9 creation confirmation tests for `ACTIVE`, `COMPLETED`, terminal observable state, and absent/not-found rejection in `internal/services/processinstance/v89/service_test.go`

### Implementation for User Story 1

- [ ] T008 [US1] Use the shared observable creation state set when v8.7 waits after process instance creation in `internal/services/processinstance/v87/service.go`
- [ ] T009 [US1] Use the shared observable creation state set when v8.8 waits after process instance creation in `internal/services/processinstance/v88/service.go`
- [ ] T010 [US1] Use the shared observable creation state set when v8.9 waits after process instance creation in `internal/services/processinstance/v89/service.go`
- [ ] T011 [US1] Preserve the observed state and confirmation timestamp on the returned created process instance in `internal/services/processinstance/v87/service.go`, `internal/services/processinstance/v88/service.go`, and `internal/services/processinstance/v89/service.go`
- [ ] T012 [US1] Run targeted service validation with `go test ./internal/domain ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -run 'State|CreateProcessInstance|WaitForProcessInstance' -count=1`
- [ ] T013 [US1] Update `specs/225-run-observable-keys/progress.md` with US1 implementation notes and validation results

**Checkpoint**: User Story 1 is independently functional and validated.

---

## Phase 4: User Story 2 - See The Observed Process Instance State (Priority: P2)

**Goal**: Rendered run-style process instance details expose the actual observed state in normal and JSON output.

**Independent Test**: Command/view tests can render created process instances with `COMPLETED` and `ACTIVE` states and verify both normal output and JSON payloads include the state.

### Tests for User Story 2

- [ ] T014 [P] [US2] Add `run pi` normal-output state rendering tests in `cmd/run_processinstance_test.go`
- [ ] T015 [P] [US2] Add `run pi` JSON envelope state rendering tests in `cmd/run_processinstance_test.go`
- [ ] T016 [P] [US2] Add process-instance list view state regression coverage in `cmd/cmd_views_get_test.go`

### Implementation for User Story 2

- [ ] T017 [US2] Render `run pi` non-JSON results through the existing process-instance list view in `cmd/run_processinstance.go`
- [ ] T018 [US2] Ensure `run pi` JSON output keeps the shared full-contract envelope and includes observed process instance `state` in `cmd/run_processinstance.go`
- [ ] T019 [US2] Verify `deploy --run` and `embed deploy --run` continue to use shared creation confirmation without adding separate state expectation flags in `cmd/deploy_processdefinition.go` and `cmd/embed_deploy.go`
- [ ] T020 [US2] Run targeted command validation with `go test ./cmd -run 'RunProcessInstance|ProcessInstancesView|DeployProcessDefinition|EmbedDeploy' -count=1`
- [ ] T021 [US2] Update `specs/225-run-observable-keys/progress.md` with US2 implementation notes and validation results

**Checkpoint**: User Stories 1 and 2 both work independently.

---

## Phase 5: User Story 3 - Pipe Created Keys Into Strict Expectations (Priority: P3)

**Goal**: `c8volt run pi --keys-only` prints only created process instance keys so downstream `expect pi --state <state> -` owns strict lifecycle assertions.

**Independent Test**: Command tests can run `run pi --keys-only` and verify stdout contains only one key per line; `expect pi` tests remain strict for mismatched states.

### Tests for User Story 3

- [ ] T022 [P] [US3] Add `run pi --keys-only` output tests in `cmd/run_processinstance_test.go`
- [ ] T023 [P] [US3] Add command contract/capabilities coverage for `run pi` keys-only support in `cmd/command_contract_test.go` and `cmd/capabilities_test.go`
- [ ] T024 [P] [US3] Add strict `expect pi` regression coverage for mismatched explicit state expectations in `cmd/expect_test.go`
- [ ] T025 [P] [US3] Add generated docs coverage for the `run pi --keys-only | expect pi --state <state> -` example in `docsgen/main_test.go`

### Implementation for User Story 3

- [ ] T026 [US3] Ensure `run pi --keys-only` uses the shared process-instance keys-only renderer in `cmd/run_processinstance.go`
- [ ] T027 [US3] Update `run pi` help text and examples with completed and active pipeline patterns in `cmd/run_processinstance.go`
- [ ] T028 [US3] Update README pipeline examples and wording in `README.md`
- [ ] T029 [US3] Regenerate generated CLI documentation with `make docs-content`
- [ ] T030 [US3] Run targeted validation with `go test ./cmd ./docsgen -run 'RunProcessInstance|ExpectProcessInstance|CommandContract|Capabilities|Generated' -count=1`
- [ ] T031 [US3] Update `specs/225-run-observable-keys/progress.md` with US3 implementation notes and validation results

**Checkpoint**: All user stories are independently functional.

---

## Phase 6: Polish & Cross-Cutting Validation

**Purpose**: Confirm the completed feature against repository standards.

- [ ] T032 Run broader validation with `go test ./cmd ./c8volt/process ./internal/services/processinstance/... ./docsgen -count=1`
- [ ] T033 Run repository validation with `make test`
- [ ] T034 Review generated documentation diffs in `docs/cli/` and `docs/index.md`
- [ ] T035 Update `specs/225-run-observable-keys/progress.md` with final validation results and any residual risks

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies
- **Foundational (Phase 2)**: Depends on Setup completion and blocks all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational; MVP
- **User Story 2 (Phase 4)**: Depends on US1 because rendered state depends on confirmed creation state propagation
- **User Story 3 (Phase 5)**: Depends on US2 because keys-only rendering shares the same run output path
- **Polish (Phase 6)**: Depends on all selected user stories

### User Story Dependencies

- **US1**: No story dependency after Foundational
- **US2**: Depends on US1
- **US3**: Depends on US2

### Parallel Opportunities

- T004 can run in parallel with review of versioned service code after T003 shape is decided.
- T005, T006, and T007 can run in parallel because they touch different version packages.
- T014, T015, and T016 can run in parallel because they target separate test concerns.
- T022, T023, T024, and T025 can run in parallel because they cover separate command/docs contracts.

---

## Parallel Example: User Story 1

```text
Task: "Add v8.7 creation confirmation tests for ACTIVE, COMPLETED, terminal observable state, and absent/not-found rejection in internal/services/processinstance/v87/service_test.go"
Task: "Add v8.8 creation confirmation tests for ACTIVE, COMPLETED, terminal observable state, and absent/not-found rejection in internal/services/processinstance/v88/service_test.go"
Task: "Add v8.9 creation confirmation tests for ACTIVE, COMPLETED, terminal observable state, and absent/not-found rejection in internal/services/processinstance/v89/service_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. Stop and validate service behavior before changing command rendering

### Incremental Delivery

1. Add observable-state creation confirmation
2. Add visible state rendering for `run pi`
3. Add keys-only pipeline behavior and docs
4. Run polish validation

### Ralph Iteration Guidance

- Complete only one user story phase per Ralph iteration unless the current work unit explicitly contains a setup/foundational prerequisite.
- Do not mark tasks complete until targeted validation in that phase passes.
- Commit messages must use Conventional Commits and end with `#225`.
- Every implementation prompt must include `--implementation-context specs/ralph-implementation-rules.md`.
