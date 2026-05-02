# Tasks: Resolve Process Instance From User Task Key

**Input**: Design documents from `/specs/152-task-key-pi-lookup/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli-get-pi-task-key.md, quickstart.md

**Tests**: Required by the feature specification for 8.8/8.9 success, 8.7 unsupported behavior, missing task handling, missing process-instance key handling, JSON output, flag conflicts, help, README, and generated CLI docs.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing. Each story is intentionally scoped small enough for one Ralph iteration.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with other tasks in the same phase when files do not overlap
- **[Story]**: Which user story the task serves
- Every task includes exact repository paths

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish the current command and service seams before behavior changes begin.

- [x] T001 Review the existing keyed-vs-search flow and validation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [x] T002 [P] Review process-instance facade and internal service contracts in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/api.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/api.go
- [x] T003 [P] Review generated native user-task client signatures for 8.8 and 8.9 in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/clients/camunda/v88/camunda/client.gen.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/clients/camunda/v89/camunda/client.gen.go
- [x] T004 [P] Review command test helpers and HTTP fixture patterns in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/process_api_stub_test.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add the shared domain and service surface that all user stories depend on.

**CRITICAL**: No user story implementation should bypass this shared seam.

- [x] T005 Add a user-task domain type exposing the owning process-instance key in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/usertask.go
- [x] T006 Add internal user-task service API, factory, and version assertions in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/usertask/api.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/usertask/factory.go
- [x] T007 Add public facade task API method for resolving a process-instance key from a user task in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/task/api.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/task/client.go
- [x] T008 Wire the user-task service into the c8volt client construction in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/client.go
- [x] T009 [P] Add test doubles or helper seams for task-key command tests in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/process_api_stub_test.go

**Checkpoint**: The repository has one native user-task resolution seam available to command code and tests.

---

## Phase 3: User Story 1 - Lookup Process Instance By User Task Key (Priority: P1) MVP

**Goal**: Resolve a user task key on Camunda 8.8/8.9 and render the owning process instance through existing keyed lookup behavior.

**Independent Test**: Run `get pi --task-key=<task-key>` against 8.8 and 8.9 fixtures and compare output to direct keyed lookup for the resolved process-instance key.

### Tests for User Story 1

- [x] T010 [P] [US1] Add 8.8 native user-task service success test resolving processInstanceKey in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/usertask/v88/service_test.go
- [x] T011 [P] [US1] Add 8.9 native user-task service success test resolving processInstanceKey in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/usertask/v89/service_test.go
- [x] T012 [P] [US1] Add command success test for `get pi --task-key` on 8.8 in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go
- [x] T013 [P] [US1] Add command success test for `get pi --task-key` on 8.9 in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go

### Implementation for User Story 1

- [x] T014 [US1] Implement 8.8 native `GetUserTaskWithResponse` resolution and conversion in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/usertask/v88/service.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/usertask/v88/convert.go
- [x] T015 [US1] Implement 8.9 native `GetUserTaskWithResponse` resolution and conversion in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/usertask/v89/service.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/usertask/v89/convert.go
- [x] T016 [US1] Add `--task-key` flag registration and happy-path branch before search mode in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [x] T017 [US1] Reuse existing process-instance keyed lookup and `listProcessInstancesView` rendering after task-key resolution in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [x] T018 [US1] Run focused US1 validation with `go test ./cmd ./c8volt/task ./internal/services/usertask/v88 ./internal/services/usertask/v89 -count=1`

**Checkpoint**: User Story 1 is the MVP and returns an owning process instance from a user task key on 8.8 and 8.9.

---

## Phase 4: User Story 2 - Reject Unsupported Or Ambiguous Task-Key Requests (Priority: P2)

**Goal**: Fail clearly for 8.7 and for all conflicting selector/search combinations.

**Independent Test**: Run `--task-key` on 8.7 and with each invalid combination; verify failure happens before task or process-instance lookup.

### Tests for User Story 2

- [x] T019 [P] [US2] Add 8.7 unsupported task-key service test in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/usertask/v87/service_test.go
- [x] T020 [P] [US2] Add command validation tests for `--task-key` with `--key` and stdin key input in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go
- [x] T021 [P] [US2] Add command validation tests for `--task-key` with process-instance search filters in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go
- [x] T022 [P] [US2] Add command validation tests for `--task-key` with `--total` and `--limit` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go

### Implementation for User Story 2

- [x] T023 [US2] Implement explicit 8.7 unsupported behavior for task-key lookup in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/usertask/v87/service.go
- [x] T024 [US2] Add task-key mutual-exclusion validation for `--key`, stdin `-`, filters, `--total`, and `--limit` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [x] T025 [US2] Ensure validation rejects conflicts before native user-task or process-instance lookup calls in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go
- [x] T026 [US2] Run focused US2 validation with `go test ./cmd ./c8volt/task ./internal/services/usertask/... -count=1`

**Checkpoint**: User Story 2 prevents unsupported and ambiguous task-key usage without fallback APIs.

---

## Phase 5: User Story 3 - Preserve Existing Single Lookup Rendering Options (Priority: P3)

**Goal**: Keep JSON, valid render flags, not-found behavior, and resolution errors aligned with direct keyed lookup.

**Independent Test**: Run task-key lookup with JSON and valid single-lookup render flags; verify missing task and missing owning key fail clearly.

### Tests for User Story 3

- [x] T027 [P] [US3] Add command test proving `get pi --task-key=<task-key> --json` matches direct keyed JSON shape in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go
- [x] T028 [P] [US3] Add command tests for `--task-key` with valid single lookup render flags such as `--with-age` and `--keys-only` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go
- [x] T029 [P] [US3] Add missing user task and missing processInstanceKey tests in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/usertask/v88/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/usertask/v89/service_test.go
- [x] T030 [P] [US3] Add command test preserving process-instance not-found behavior after task-key resolution in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go

### Implementation for User Story 3

- [x] T031 [US3] Normalize missing task and missing processInstanceKey errors through repository error conventions in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/usertask/v88/service.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/usertask/v89/service.go
- [x] T032 [US3] Preserve existing direct keyed render option flow after task-key resolution in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_processinstance.go
- [x] T033 [US3] Run focused US3 validation with `go test ./cmd ./internal/services/usertask/v88 ./internal/services/usertask/v89 -count=1`

**Checkpoint**: User Story 3 makes task-key lookup feel like direct keyed lookup after resolution.

---

## Phase 6: User Story 4 - Discover Task-Key Lookup In Help And Docs (Priority: P4)

**Goal**: Document the new flag, examples, version support, and no Tasklist/Operate fallback rule.

**Independent Test**: Inspect command help, README, and generated CLI docs for task-key examples and constraints.

### Tests for User Story 4

- [x] T034 [P] [US4] Add help-output assertions for `--task-key`, human example, JSON example, and 8.7 unsupported wording in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go
- [x] T035 [P] [US4] Add docs contract check for generated `get process-instance` docs in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docsgen/main_test.go

### Implementation for User Story 4

- [x] T036 [US4] Update command help and examples for task-key lookup in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [x] T037 [US4] Update README examples and process-instance lookup documentation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md
- [x] T038 [US4] Regenerate generated CLI docs with `make docs-content`, updating /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_get_process-instance.md
- [x] T039 [US4] Verify generated docs and README do not suggest Tasklist or Operate fallback for task-key lookup in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_get_process-instance.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md

**Checkpoint**: User Story 4 makes the new public behavior discoverable and synchronized.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final cleanup, formatting, validation, and traceability.

- [x] T040 [P] Review task-key lookup contract against implementation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/152-task-key-pi-lookup/contracts/cli-get-pi-task-key.md
- [x] T041 [P] Review quickstart commands against final behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/152-task-key-pi-lookup/quickstart.md
- [x] T042 Run `gofmt` on changed Go files in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/usertask
- [x] T043 Run targeted validation with `go test ./cmd ./c8volt/process ./c8volt/task ./internal/services/usertask/... ./internal/services/processinstance/... -count=1` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt
- [x] T044 Run final repository validation with `make test` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup and blocks all user stories.
- **US1 Task-Key Happy Path (Phase 3)**: Depends on Foundational and is the MVP.
- **US2 Unsupported/Conflict Validation (Phase 4)**: Depends on Foundational; can run after US1 or in parallel once the flag exists.
- **US3 Output/Error Parity (Phase 5)**: Depends on US1 behavior and the shared error conversion decisions.
- **US4 Help/Docs (Phase 6)**: Depends on the final command metadata from US1-US3.
- **Polish (Phase 7)**: Depends on all selected user stories.

### User Story Dependencies

- **User Story 1 (P1)**: MVP; no dependency on other stories after Foundational.
- **User Story 2 (P2)**: Requires the `--task-key` flag surface and user-task service seam.
- **User Story 3 (P3)**: Requires successful task-key resolution and error conversion.
- **User Story 4 (P4)**: Requires finalized CLI behavior and help wording.

### Parallel Opportunities

- T002-T004 can run in parallel after T001.
- T010-T013 can run in parallel because version service tests and command tests are separate enough to prepare independently.
- T019-T022 can run in parallel because validation cases are independent.
- T027-T030 can run in parallel across output and error-path tests.
- T034 and T035 can run in parallel once help wording is drafted.
- T040 and T041 can run in parallel during final review.

---

## Parallel Example: User Story 1

```text
Task: "Add 8.8 native user-task service success test resolving processInstanceKey in internal/services/usertask/v88/service_test.go"
Task: "Add 8.9 native user-task service success test resolving processInstanceKey in internal/services/usertask/v89/service_test.go"
Task: "Add command success test for get pi --task-key on 8.8 in cmd/get_processinstance_test.go"
Task: "Add command success test for get pi --task-key on 8.9 in cmd/get_processinstance_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational user-task service and facade seam.
3. Complete Phase 3: successful task-key lookup on 8.8/8.9.
4. Stop and validate that task-key lookup returns the same process instance as direct keyed lookup.

### Incremental Delivery

1. Add the service/facade resolution seam.
2. Deliver happy-path task-key lookup for 8.8/8.9.
3. Lock down unsupported and mutually exclusive usage.
4. Preserve output/error parity with direct keyed lookup.
5. Update help, README, generated docs, and run final validation.

### Commit Guidance

Use Conventional Commits and append the issue number as the final subject token, for example:

```text
feat(get): add task-key process-instance lookup #152
test(get): cover task-key selector conflicts #152
docs(get): document task-key process-instance lookup #152
```
