# Tasks: Process Instance Incident Expectation

**Input**: Design documents from `/specs/170-process-incident-expect/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are required by the feature specification and constitution. Story test tasks should be written before implementation and should fail until the story implementation is complete.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files or only adds tests/docs
- **[Story]**: Maps to the user story from [spec.md](./spec.md)
- Every task names exact repository paths

## Phase 1: Setup (Shared Discovery)

**Purpose**: Confirm the current process-instance expectation flow, domain incident field, and documentation generation path before changing behavior.

- [ ] T001 Inspect current process-instance expectation command validation, stdin key handling, and read-only/automation metadata in `cmd/expect_processinstance.go`
- [ ] T002 [P] Inspect existing expect command tests and helper subprocess patterns in `cmd/expect_test.go` and `cmd/process_api_stub_test.go`
- [ ] T003 [P] Inspect process facade wait APIs and reports in `c8volt/process/api.go`, `c8volt/process/client.go`, `c8volt/process/bulk.go`, and `c8volt/process/model.go`
- [ ] T004 [P] Inspect shared process-instance waiter behavior and tests in `internal/services/processinstance/waiter/waiter.go` and `internal/services/processinstance/waiter/waiter_test.go`
- [ ] T005 [P] Inspect versioned process-instance service contracts and incident mappings in `internal/services/processinstance/api.go`, `internal/services/processinstance/v87/`, `internal/services/processinstance/v88/`, and `internal/services/processinstance/v89/`
- [ ] T006 [P] Inspect process-instance expectation docs in `README.md`, `docs/index.md`, and `docs/cli/c8volt_expect_process-instance.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared expectation representation and strict incident parsing so all stories use one validation and wait contract.

**Critical**: No user story implementation should begin until this phase is complete.

- [ ] T007 Add an incident expectation value type and strict parser accepting only `true` and `false` in `c8volt/process/model.go`
- [ ] T008 Add combined process-instance expectation request/report types for optional state and incident expectations in `c8volt/process/model.go`
- [ ] T009 Extend the public process facade API with a combined process-instance expectation wait method in `c8volt/process/api.go`
- [ ] T010 Extend the internal process-instance service API with a combined expectation wait method in `internal/services/processinstance/api.go`
- [ ] T011 Extend versioned service contracts to include the combined expectation wait method in `internal/services/processinstance/v87/contract.go`, `internal/services/processinstance/v88/contract.go`, and `internal/services/processinstance/v89/contract.go`
- [ ] T012 Add shared matcher helpers for state and incident expectations in `internal/services/processinstance/waiter/waiter.go`
- [ ] T013 Add command-level flag storage and strict validation scaffolding for `--incident` without changing runtime behavior yet in `cmd/expect_processinstance.go`

**Checkpoint**: Shared types, contracts, and strict parser exist before story behavior is wired.

---

## Phase 3: User Story 1 - Wait For Incident Presence (Priority: P1)

**Goal**: `--incident true` waits until selected process instances are present with `Incident: true`.

**Independent Test**: Run the expectation command against a selected instance whose incident marker changes to true and verify success only after the marker matches.

### Tests for User Story 1

- [ ] T014 [P] [US1] Add command test for `c8volt expect pi --key <key> --incident true` waiting until incident true in `cmd/expect_test.go`
- [ ] T015 [P] [US1] Add facade test proving incident true expectation maps through the process client in `c8volt/process/client_test.go`
- [ ] T016 [P] [US1] Add waiter test proving incident true waits across false-to-true polling in `internal/services/processinstance/waiter/waiter_test.go`

### Implementation for User Story 1

- [ ] T017 [US1] Implement combined expectation waiting for a single process instance in `internal/services/processinstance/waiter/waiter.go`
- [ ] T018 [US1] Implement combined expectation waiting for multiple process instances with existing worker controls in `internal/services/processinstance/waiter/waiter.go`
- [ ] T019 [US1] Delegate versioned services to the shared combined waiter in `internal/services/processinstance/v87/service.go`, `internal/services/processinstance/v88/service.go`, and `internal/services/processinstance/v89/service.go`
- [ ] T020 [US1] Map combined expectation wait results through the public process facade in `c8volt/process/client.go` and `c8volt/process/bulk.go`
- [ ] T021 [US1] Wire `cmd/expect_processinstance.go` to call the combined expectation wait path when `--incident true` is provided
- [ ] T022 [US1] Run `go test ./cmd ./c8volt/process ./internal/services/processinstance/waiter -run 'TestExpect|TestClient_.*Incident|TestWait.*Incident' -count=1` and fix regressions

**Checkpoint**: User Story 1 is independently complete when direct `--incident true` waits through the command, facade, and waiter tests.

---

## Phase 4: User Story 2 - Wait For Incident Absence (Priority: P2)

**Goal**: `--incident false` waits for present process instances with no incident and never treats missing instances as success.

**Independent Test**: Run incident false waits against present false and missing process-instance cases, then verify only present false succeeds.

### Tests for User Story 2

- [ ] T023 [P] [US2] Add command test for `c8volt expect pi --key <key> --incident false` succeeding for a present incident-free instance in `cmd/expect_test.go`
- [ ] T024 [P] [US2] Add waiter test proving a missing process instance does not satisfy `--incident false` in `internal/services/processinstance/waiter/waiter_test.go`
- [ ] T025 [P] [US2] Add facade test proving incident false expectation preserves present-instance semantics in `c8volt/process/client_test.go`

### Implementation for User Story 2

- [ ] T026 [US2] Update incident matcher behavior so false requires a present process instance in `internal/services/processinstance/waiter/waiter.go`
- [ ] T027 [US2] Ensure facade report mapping preserves observed incident false status in `c8volt/process/client.go` and `c8volt/process/model.go`
- [ ] T028 [US2] Run `go test ./cmd ./c8volt/process ./internal/services/processinstance/waiter -run 'TestExpect|TestClient_.*Incident|TestWait.*Incident' -count=1` and fix regressions

**Checkpoint**: User Story 2 is independently complete when `--incident false` succeeds only for present incident-free process instances.

---

## Phase 5: User Story 3 - Combine State And Incident Expectations (Priority: P3)

**Goal**: `--state` and `--incident` compose so every selected process instance must satisfy both expectations before success.

**Independent Test**: Run combined expectations where one condition matches before the other and verify success waits for both.

### Tests for User Story 3

- [ ] T029 [P] [US3] Add command test for combined `--state active --incident true` waiting until both match in `cmd/expect_test.go`
- [ ] T030 [P] [US3] Add waiter tests preserving `--state absent` and canceled/terminated compatibility with combined expectation matching in `internal/services/processinstance/waiter/waiter_test.go`
- [ ] T031 [P] [US3] Add facade test for combined state and incident expectation requests in `c8volt/process/client_test.go`

### Implementation for User Story 3

- [ ] T032 [US3] Update combined matcher logic to require all requested expectations for each selected instance in `internal/services/processinstance/waiter/waiter.go`
- [ ] T033 [US3] Preserve state-only wait delegation and reporting compatibility in `c8volt/process/client.go`, `c8volt/process/bulk.go`, and `internal/services/processinstance/waiter/waiter.go`
- [ ] T034 [US3] Update command status/log messages for incident-only and combined expectations in `cmd/expect_processinstance.go`
- [ ] T035 [US3] Run `go test ./cmd ./c8volt/process ./internal/services/processinstance/waiter -run 'TestExpect|TestClient_.*Wait|TestWaitForProcessInstanceState|TestWait.*Expectation' -count=1` and fix regressions

**Checkpoint**: User Story 3 is independently complete when combined expectations pass and state-only behavior remains stable.

---

## Phase 6: User Story 4 - Preserve Key Pipelining (Priority: P4)

**Goal**: Stdin key pipelining with `-` works when `--incident` is present and existing state-only pipelining remains covered.

**Independent Test**: Pipe keys into `c8volt expect pi --incident true -` and verify `--key` is not required.

### Tests for User Story 4

- [ ] T036 [P] [US4] Add command subprocess test for `c8volt get pi --keys-only | c8volt expect pi --incident true -` behavior in `cmd/expect_test.go`
- [ ] T037 [P] [US4] Keep or strengthen existing `expect pi --state active -` regression coverage in `cmd/expect_test.go`

### Implementation for User Story 4

- [ ] T038 [US4] Ensure `cmd/expect_processinstance.go` applies existing `readKeysIfDash` and `mergeAndValidateKeys` behavior before incident expectation waiting
- [ ] T039 [US4] Run `go test ./cmd -run 'TestExpectProcessInstanceCommand_.*Dash|TestHelperExpectProcessInstanceCommand_.*Dash' -count=1` and fix regressions

**Checkpoint**: User Story 4 is independently complete when stdin key pipelining works for incident and state expectation paths.

---

## Phase 7: User Story 5 - Validate Expectation Input And Help (Priority: P5)

**Goal**: Invalid incident values and missing expectation flags fail clearly, while help/docs document `--incident true|false`.

**Independent Test**: Run invalid invocations and inspect help/generated docs for the new flag and examples.

### Tests for User Story 5

- [ ] T040 [P] [US5] Add invalid `--incident maybe` command test through invalid-input handling in `cmd/expect_test.go`
- [ ] T041 [P] [US5] Add no-expectation flag command test for `c8volt expect pi --key <key>` in `cmd/expect_test.go`
- [ ] T042 [P] [US5] Add help/discovery test for `--incident true|false` and examples in `cmd/expect_test.go`
- [ ] T043 [P] [US5] Add command contract expectation for the new `--incident` flag in `cmd/command_contract_test.go`

### Implementation for User Story 5

- [ ] T044 [US5] Remove required `--state` marking and enforce at-least-one expectation validation in `cmd/expect_processinstance.go`
- [ ] T045 [US5] Register and document `--incident true|false` on `expectProcessInstanceCmd` in `cmd/expect_processinstance.go`
- [ ] T046 [US5] Update parent expect command examples in `cmd/expect.go`
- [ ] T047 [US5] Update process-instance expectation examples in `README.md` and `docs/index.md`
- [ ] T048 [US5] Regenerate CLI documentation under `docs/cli/` with `make docs-content`
- [ ] T049 [US5] Run `go test ./cmd -run 'TestExpect|TestCommandContract' -count=1` and fix help or validation regressions

**Checkpoint**: User Story 5 is independently complete when invalid inputs fail clearly and users can discover `--incident true|false` in help and docs.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final cleanup, validation, and repository-wide proof.

- [ ] T050 [P] Run `gofmt -w cmd/expect.go cmd/expect_processinstance.go cmd/expect_test.go cmd/command_contract_test.go cmd/process_api_stub_test.go c8volt/process/api.go c8volt/process/bulk.go c8volt/process/client.go c8volt/process/model.go c8volt/process/client_test.go internal/services/processinstance/api.go internal/services/processinstance/waiter/waiter.go internal/services/processinstance/waiter/waiter_test.go internal/services/processinstance/v87/contract.go internal/services/processinstance/v87/service.go internal/services/processinstance/v88/contract.go internal/services/processinstance/v88/service.go internal/services/processinstance/v89/contract.go internal/services/processinstance/v89/service.go`
- [ ] T051 Run `go test ./cmd ./c8volt/process ./internal/services/processinstance/waiter ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1` and fix regressions
- [ ] T052 Run `make docs-content` and fix documentation generation issues
- [ ] T053 Run `make test` and fix repository validation failures
- [ ] T054 [P] Review `specs/170-process-incident-expect/quickstart.md` against implemented behavior and update if command examples changed
- [ ] T055 Verify `git diff` contains only issue #170 implementation, docs, generated docs, and Speckit artifacts before commit

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on setup and blocks all user stories.
- **US1 (Phase 3)**: Depends on foundational types/contracts/parser.
- **US2 (Phase 4)**: Depends on US1 wait path and adds false/missing semantics.
- **US3 (Phase 5)**: Depends on US1 and US2 matchers so combined expectations can reuse both sides.
- **US4 (Phase 6)**: Depends on command wiring from US1 and validation from foundational work.
- **US5 (Phase 7)**: Depends on final command semantics from US1-US4.
- **Polish (Phase 8)**: Depends on the desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: First user-visible slice; proves `--incident true`.
- **User Story 2 (P2)**: Builds on the incident wait path and locks missing-instance semantics for false.
- **User Story 3 (P3)**: Builds on state and incident matchers.
- **User Story 4 (P4)**: Builds on command target resolution and incident wait wiring.
- **User Story 5 (P5)**: Builds on final command behavior so help/docs match implementation.

### Parallel Opportunities

- T002 through T006 can run in parallel during setup.
- T014 through T016 can be written in parallel for US1.
- T023 through T025 can be written in parallel for US2.
- T029 through T031 can be written in parallel for US3.
- T036 and T037 can be written in parallel for US4.
- T040 through T043 can be written in parallel for US5.
- T050 and T054 can run in parallel after implementation is complete.

## Parallel Example: User Story 1

```text
Task: "Add command test for `c8volt expect pi --key <key> --incident true` waiting until incident true in cmd/expect_test.go"
Task: "Add facade test proving incident true expectation maps through the process client in c8volt/process/client_test.go"
Task: "Add waiter test proving incident true waits across false-to-true polling in internal/services/processinstance/waiter/waiter_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete User Story 1 to deliver `--incident true`.
3. Stop and run the US1 targeted tests before adding false, combined, or docs behavior.

### Incremental Delivery

1. Add false/missing-instance semantics after true matching is stable.
2. Add combined state and incident matching after both incident values are covered.
3. Re-prove stdin key pipelining.
4. Finish input validation, help, README/docs, and generated CLI docs.
5. Run targeted tests after each story, then `make test` before commit.

### Commit Guidance

Use Conventional Commit subjects and append the issue number as the final token, for example:

```text
feat(expect): add process incident expectations #170
```
