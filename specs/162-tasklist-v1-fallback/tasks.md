# Tasks: Tasklist V1 Fallback For Task-Key Process-Instance Lookup

**Input**: Design documents from `/specs/162-tasklist-v1-fallback/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/
**Tests**: Required by the feature specification for primary-success, fallback-success, not-found, error, unsupported-version, help/docs, and validation behavior.
**Commit Rule**: Any commit subject for this work must use Conventional Commits and append `#162` as the final token.

## Phase 1: Setup

**Purpose**: Establish the exact generated-client and service surfaces before editing behavior.

- [x] T001 Review Tasklist V1 generated request/response fields in `internal/clients/camunda/v88/tasklist/client.gen.go` and `internal/clients/camunda/v89/tasklist/client.gen.go`
- [x] T002 Review current task-key lookup tests and helper server routes in `internal/services/usertask/v88/service_test.go`, `internal/services/usertask/v89/service_test.go`, and `cmd/get_processinstance_test.go`
- [x] T003 Confirm current help/docs wording that mentions no Tasklist fallback in `cmd/get_processinstance.go`, `README.md`, and generated files under `docs/`

---

## Phase 2: Foundational

**Purpose**: Add shared seams that all fallback stories depend on.

- [x] T004 Add Tasklist V1 client interfaces and constructor options for v88 fallback in `internal/services/usertask/v88/contract.go` and `internal/services/usertask/v88/service.go`
- [x] T005 Add Tasklist V1 client interfaces and constructor options for v89 fallback in `internal/services/usertask/v89/contract.go` and `internal/services/usertask/v89/service.go`
- [x] T006 Add fallback Tasklist result conversion helpers in `internal/services/usertask/v88/convert.go` and `internal/services/usertask/v89/convert.go`
- [x] T007 Add fallback request builder helpers that include task key, page size, and effective tenant when available in `internal/services/usertask/v88/service.go` and `internal/services/usertask/v89/service.go`

**Checkpoint**: Versioned user-task services can accept both primary Camunda clients and Tasklist fallback clients without changing the public command contract.

---

## Phase 3: User Story 1 - Resolve Legacy Task Keys Through Fallback (Priority: P1)

**Goal**: A primary lookup miss on Camunda 8.8 or 8.9 falls back to Tasklist V1 and resolves the owning process instance.

**Independent Test**: Service tests prove fallback returns a process-instance key after primary miss, and command tests prove final output matches existing keyed process-instance lookup.

### Tests for User Story 1

- [x] T008 [P] [US1] Add v89 service test for primary miss followed by Tasklist fallback success in `internal/services/usertask/v89/service_test.go`
- [x] T009 [P] [US1] Add v88 service test for primary miss followed by Tasklist fallback success in `internal/services/usertask/v88/service_test.go`
- [x] T010 [US1] Add command integration test for fallback-resolved process-instance output in `cmd/get_processinstance_test.go`
- [x] T011 [US1] Add command integration test for fallback-resolved JSON output matching direct keyed JSON in `cmd/get_processinstance_test.go`

### Implementation for User Story 1

- [x] T012 [US1] Implement v89 Tasklist fallback search after primary not-found in `internal/services/usertask/v89/service.go`
- [x] T013 [US1] Implement v88 Tasklist fallback search after primary not-found in `internal/services/usertask/v88/service.go`
- [x] T014 [US1] Wire v89 Tasklist client construction from `config.APIs.Tasklist.BaseURL` in `internal/services/usertask/v89/service.go`
- [x] T015 [US1] Wire v88 Tasklist client construction from `config.APIs.Tasklist.BaseURL` in `internal/services/usertask/v88/service.go`
- [x] T016 [US1] Verify fallback-resolved command behavior with `go test ./internal/services/usertask/v88 ./internal/services/usertask/v89 ./cmd -run 'HasUserTasks|GetUserTask' -count=1`

**Checkpoint**: Legacy task-key fallback works for 8.8 and 8.9 and renders through existing process-instance lookup.

---

## Phase 4: User Story 2 - Preserve Primary Lookup As First Choice (Priority: P2)

**Goal**: Modern user tasks that resolve through the primary v2 path never call Tasklist fallback.

**Independent Test**: Service and command tests fail if a fallback client is invoked after a successful primary lookup.

### Tests for User Story 2

- [x] T017 [P] [US2] Add v89 service test proving Tasklist fallback is not called when primary lookup succeeds in `internal/services/usertask/v89/service_test.go`
- [x] T018 [P] [US2] Add v88 service test proving Tasklist fallback is not called when primary lookup succeeds in `internal/services/usertask/v88/service_test.go`
- [x] T019 [US2] Add multi-task command test where one key resolves through primary lookup and one through fallback in `cmd/get_processinstance_test.go`

### Implementation for User Story 2

- [x] T020 [US2] Preserve short-circuit behavior after primary success in `internal/services/usertask/v89/service.go`
- [x] T021 [US2] Preserve short-circuit behavior after primary success in `internal/services/usertask/v88/service.go`
- [x] T022 [US2] Verify mixed primary/fallback lookup ordering with `go test ./cmd ./internal/services/usertask/v88 ./internal/services/usertask/v89 -run 'HasUserTasks|GetUserTask' -count=1`

**Checkpoint**: Primary lookup remains the first and preferred path for every supported task key.

---

## Phase 5: User Story 3 - Fail Clearly When Neither Lookup Resolves (Priority: P3)

**Goal**: Missing task keys and fallback failures produce clear, script-safe errors without masking operational failures.

**Independent Test**: Tests cover both-lookups-missing, fallback missing process-instance key, fallback server/config/auth-like failures, and 8.7 unsupported behavior.

### Tests for User Story 3

- [x] T023 [P] [US3] Add v89 service tests for both-lookups-missing, fallback missing process-instance key, and fallback server failure in `internal/services/usertask/v89/service_test.go`
- [x] T024 [P] [US3] Add v88 service tests for both-lookups-missing, fallback missing process-instance key, and fallback server failure in `internal/services/usertask/v88/service_test.go`
- [x] T025 [US3] Add command failure test for both-lookups-missing not-found output in `cmd/get_processinstance_test.go`
- [x] T026 [US3] Confirm 8.7 unsupported tests still prove no fallback attempt in `internal/services/usertask/v87/service_test.go` and `cmd/get_processinstance_test.go`

### Implementation for User Story 3

- [x] T027 [US3] Restrict fallback eligibility to primary `domain.ErrNotFound` outcomes in `internal/services/usertask/v89/service.go`
- [x] T028 [US3] Restrict fallback eligibility to primary `domain.ErrNotFound` outcomes in `internal/services/usertask/v88/service.go`
- [x] T029 [US3] Return final not-found only after both primary and fallback lookup miss in `internal/services/usertask/v89/service.go`
- [x] T030 [US3] Return final not-found only after both primary and fallback lookup miss in `internal/services/usertask/v88/service.go`
- [x] T031 [US3] Surface fallback malformed response and server failures distinctly from not found in `internal/services/usertask/v89/service.go`
- [x] T032 [US3] Surface fallback malformed response and server failures distinctly from not found in `internal/services/usertask/v88/service.go`
- [x] T033 [US3] Verify failure behavior with `go test ./internal/services/usertask/v87 ./internal/services/usertask/v88 ./internal/services/usertask/v89 ./cmd -run 'HasUserTasks|GetUserTask|Unsupported' -count=1`

**Checkpoint**: Missing tasks and operational failures are distinguishable and 8.7 remains unsupported.

---

## Phase 6: User Story 4 - Keep Version And Documentation Contract Accurate (Priority: P4)

**Goal**: Help text, README, and generated docs describe v2-first lookup with Tasklist V1 fallback and remove obsolete no-fallback wording.

**Independent Test**: Help/doc tests and generated documentation show the updated command contract.

### Tests for User Story 4

- [x] T034 [US4] Update help text test expectations for fallback wording in `cmd/get_processinstance_test.go`
- [x] T035 [P] [US4] Add or update docs content checks for fallback wording in `cmd/command_contract_test.go` or the existing generated-doc test location

### Implementation for User Story 4

- [x] T036 [US4] Update command long help and examples for `--has-user-tasks` fallback behavior in `cmd/get_processinstance.go`
- [x] T037 [US4] Update user-facing examples and caveats in `README.md`
- [x] T038 [US4] Regenerate CLI documentation with `make docs-content`
- [x] T039 [US4] Verify documentation behavior with `go test ./cmd -run 'Help|Contract|HasUserTasks' -count=1`

**Checkpoint**: User-facing documentation no longer contradicts the fallback behavior.

---

## Phase 7: Polish & Cross-Cutting Validation

**Purpose**: Final cleanup, consistency, and required repository validation.

- [x] T040 [P] Run `gofmt` on changed Go files under `internal/services/usertask/` and `cmd/`
- [x] T041 [P] Review issue traceability and active plan links in `specs/162-tasklist-v1-fallback/spec.md`, `specs/162-tasklist-v1-fallback/plan.md`, and `AGENTS.md`
- [x] T042 Run targeted validation from `specs/162-tasklist-v1-fallback/quickstart.md`
- [x] T043 Run `make test`
- [x] T044 Record implementation progress and any validation notes in `specs/162-tasklist-v1-fallback/progress.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup; blocks all user stories.
- **US1 (Phase 3)**: Depends on Foundational; MVP fallback success path.
- **US2 (Phase 4)**: Depends on Foundational; can run after or alongside US1 if write coordination avoids the same v88/v89 service files.
- **US3 (Phase 5)**: Depends on Foundational and should follow US1 fallback structure.
- **US4 (Phase 6)**: Can start after the behavioral contract is stable enough to document.
- **Polish (Phase 7)**: Depends on selected user stories and documentation updates.

### User Story Dependencies

- **US1**: First implementation slice and MVP.
- **US2**: Independent behavior check for existing primary success path; shares service files with US1.
- **US3**: Builds on fallback path from US1 to complete negative/error behavior.
- **US4**: Documents final behavior after service and command expectations are known.

### Parallel Opportunities

- T008 and T009 can run in parallel across versioned service tests.
- T017 and T018 can run in parallel across versioned service tests.
- T023 and T024 can run in parallel across versioned service tests.
- Documentation test work in T035 can run alongside README/help drafting if file ownership is coordinated.
- Final review task T041 can run in parallel with formatting T040.

## Parallel Example: User Story 1

```text
Task: "Add v89 service test for primary miss followed by Tasklist fallback success in internal/services/usertask/v89/service_test.go"
Task: "Add v88 service test for primary miss followed by Tasklist fallback success in internal/services/usertask/v88/service_test.go"
```

## Parallel Example: User Story 3

```text
Task: "Add v89 service tests for both-lookups-missing, fallback missing process-instance key, and fallback server failure in internal/services/usertask/v89/service_test.go"
Task: "Add v88 service tests for both-lookups-missing, fallback missing process-instance key, and fallback server failure in internal/services/usertask/v88/service_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational tasks.
2. Complete US1 to prove primary miss followed by fallback success for 8.8 and 8.9.
3. Stop and validate US1 with targeted service and command tests.

### Incremental Delivery

1. Add US1 fallback success.
2. Add US2 primary short-circuit regression coverage.
3. Add US3 negative/error handling.
4. Add US4 docs/help/generated docs.
5. Run targeted validation and `make test`.

### Ralph Iteration Guidance

Each Ralph iteration should complete one narrow story slice or one version-specific half of a story with tests. Avoid editing v88 and v89 service files in separate concurrent workers unless file ownership is explicit.
