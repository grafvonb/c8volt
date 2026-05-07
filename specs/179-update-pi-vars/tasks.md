# Tasks: Process Instance Variable Updates

**Input**: Design documents from `/specs/179-update-pi-vars/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are required by the feature specification and constitution. Story test tasks should be written before implementation and should fail until the story implementation is complete.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files or only adds tests/docs
- **[Story]**: Maps to the user story from [spec.md](./spec.md)
- Every task names exact repository paths

## Phase 1: Setup (Shared Discovery)

**Purpose**: Confirm existing command, metadata, variable lookup, worker, waiter, and docs paths before changing behavior.

- [x] T001 Inspect root command registration and mutation metadata patterns in `cmd/run.go`, `cmd/delete.go`, `cmd/deploy_processdefinition.go`, and `cmd/command_contract.go`
- [x] T002 [P] Inspect process-instance key/stdin, worker, fail-fast, and no-worker-limit behavior in `cmd/delete_processinstance.go`, `cmd/expect_processinstance.go`, and related tests in `cmd/delete_test.go` and `cmd/expect_test.go`
- [x] T003 [P] Inspect process-instance variable lookup and rendering paths in `cmd/get_processinstance.go`, `cmd/get_processinstance_enrichment.go`, `cmd/cmd_views_processinstance_activity.go`, `cmd/cmd_views_processinstance_vars.go`, and `cmd/get_processinstance_test.go`
- [x] T004 [P] Inspect process facade models and service interface patterns in `c8volt/process/api.go`, `c8volt/process/client.go`, `c8volt/process/model.go`, and `internal/services/processinstance/api.go`
- [x] T005 [P] Inspect generated v8.8/v8.9 element-instance variable update client methods and v8.7 unsupported patterns in `internal/clients/camunda/v88/camunda/client.gen.go`, `internal/clients/camunda/v89/camunda/client.gen.go`, and `internal/services/processinstance/v87/service.go`
- [x] T006 [P] Inspect README, generated docs, and docs generation workflow in `README.md`, `docs/`, and `docsgen/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add the shared command shell, domain/facade/service contracts, and result model used by every story.

**Critical**: No user story implementation should begin until this phase is complete.

- [x] T007 Add the `update` root command with aliases, examples, backoff bindings, and state-changing metadata in `cmd/update.go`
- [x] T008 Add process-instance variable update request/result domain models in `internal/domain/processinstance.go`
- [x] T009 Add facade-level process-instance variable update request/result models in `c8volt/process/model.go`
- [x] T010 Extend process facade interface and client stubs for update/confirmation orchestration in `c8volt/process/api.go` and `c8volt/process/client.go`
- [x] T011 Extend process-instance service API and compile-time implementation assertions for variable update support in `internal/services/processinstance/api.go`
- [x] T012 Add unsupported Camunda 8.7 update method behavior to `internal/services/processinstance/v87/contract.go` and `internal/services/processinstance/v87/service.go`
- [x] T013 Add command contract discovery tests for the new update root and process-instance metadata in `cmd/command_contract_test.go`
- [x] T014 Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process ./internal/services/processinstance/v87 -run 'Test(CommandCapability|Update|Unsupported)' -count=1` and fix foundational regressions

**Checkpoint**: The command family, shared data types, and service/facade surface compile before story behavior is implemented.

---

## Phase 3: User Story 1 - Update Variables For One Process Instance (Priority: P1) MVP

**Goal**: `c8volt update process-instance` and `update pi` update one process instance and confirm requested variables through the existing variable lookup path.

**Independent Test**: Run `c8volt update pi --key <key> --vars '{"foo":"bar"}'` and verify mutation plus confirmed visibility through the `get pi --with-vars` lookup path.

### Tests for User Story 1

- [x] T015 [P] [US1] Add command test for `update pi --key <key> --vars '{"foo":"bar"}'` submitting the v8.8 update request and confirming through variable lookup in `cmd/update_processinstance_test.go`
- [x] T016 [P] [US1] Add command test proving `update process-instance` and `update pi` behave identically for a single key in `cmd/update_processinstance_test.go`
- [x] T017 [P] [US1] Add v8.8 service test for `PUT /v2/element-instances/{elementInstanceKey}/variables` using the process instance key in `internal/services/processinstance/v88/service_test.go`
- [x] T018 [P] [US1] Add v8.9 service test for `PUT /v2/element-instances/{elementInstanceKey}/variables` using the process instance key in `internal/services/processinstance/v89/service_test.go`
- [x] T019 [P] [US1] Add facade test for normalized JSON confirmation comparing requested values to returned process-instance variables in `c8volt/process/client_test.go`

### Implementation for User Story 1

- [x] T020 [US1] Add `cmd/update_processinstance.go` with command registration, `--key`, `--vars`, `--no-wait`, worker flags, automation support, and state-changing metadata
- [x] T021 [US1] Parse and validate single-key `--vars` JSON object input before mutation in `cmd/update_processinstance.go`
- [x] T022 [US1] Implement v8.8 variable update service call in `internal/services/processinstance/v88/variables.go`
- [x] T023 [US1] Implement v8.9 variable update service call in `internal/services/processinstance/v89/variables.go`
- [x] T024 [US1] Implement facade update and default confirmation flow using existing variable lookup and normalized JSON comparison in `c8volt/process/client.go`
- [x] T025 [US1] Render single-key confirmed update results for human and JSON output in `cmd/update_processinstance.go` and existing command view helpers as needed
- [x] T026 [US1] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -run 'Test(UpdateProcessInstance|UpdatePI|ElementInstanceVariables|VariableConfirmation)' -count=1` and fix regressions

**Checkpoint**: User Story 1 is independently complete when one-key update, alias behavior, confirmation, and human/JSON output pass.

---

## Phase 4: User Story 2 - Update Multiple Selected Process Instances (Priority: P2)

**Goal**: The command applies the same variable map to every unique key from repeated `--key` flags and stdin `-`, reporting each key independently.

**Independent Test**: Run the update command with duplicate flag and stdin keys, then verify every unique process instance is updated once and reported independently.

### Tests for User Story 2

- [x] T027 [P] [US2] Add command test for multiple repeated `--key` values applying one `--vars` payload to each unique key in `cmd/update_processinstance_test.go`
- [x] T028 [P] [US2] Add command test for stdin `-` keys merged and deduplicated with `--key` values in `cmd/update_processinstance_test.go`
- [x] T029 [P] [US2] Add facade test for multi-key update respecting worker count and fail-fast options in `c8volt/process/client_test.go`

### Implementation for User Story 2

- [x] T030 [US2] Reuse existing stdin key parsing, validation, merge, and deduplication behavior for update targets in `cmd/update_processinstance.go`
- [x] T031 [US2] Apply the same parsed variable map to every unique target key through facade/service calls in `c8volt/process/client.go`
- [x] T032 [US2] Reuse existing worker, `--workers`, `--fail-fast`, and `--no-worker-limit` option mapping for multi-key updates in `cmd/update_processinstance.go` and `c8volt/process/client.go`
- [x] T033 [US2] Render multi-key human and JSON results with independent per-key statuses in `cmd/update_processinstance.go`
- [x] T034 [US2] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process -run 'Test(UpdateProcessInstance.*(Multiple|Stdin|Dedup|Workers|FailFast))' -count=1` and fix regressions

**Checkpoint**: User Story 2 is independently complete when repeated keys, stdin keys, deduplication, workers, and per-key output pass.

---

## Phase 5: User Story 3 - Return Accepted Output Without Waiting (Priority: P3)

**Goal**: `--no-wait` returns after accepted mutation requests and clearly reports submitted status without confirmation polling.

**Independent Test**: Run `c8volt update pi --key <key> --vars '{"foo":"bar"}' --no-wait` and verify variable lookup is not called.

### Tests for User Story 3

- [x] T035 [P] [US3] Add command test proving `--no-wait` returns accepted/submitted output without variable confirmation lookup in `cmd/update_processinstance_test.go`
- [x] T036 [P] [US3] Add JSON output test for `--no-wait` submitted results in `cmd/update_processinstance_test.go`
- [x] T037 [P] [US3] Add facade test proving mutation errors still report per-key failure when `--no-wait` is set in `c8volt/process/client_test.go`

### Implementation for User Story 3

- [x] T038 [US3] Wire `--no-wait` into update request options and skip confirmation after accepted mutation in `cmd/update_processinstance.go` and `c8volt/process/client.go`
- [x] T039 [US3] Distinguish submitted, confirmed, mutation-failed, and confirmation-failed statuses in result models in `c8volt/process/model.go`
- [x] T040 [US3] Ensure human and JSON renderers show submitted status without implying read-model confirmation in `cmd/update_processinstance.go`
- [x] T041 [US3] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process -run 'Test(UpdateProcessInstance.*NoWait|NoWait.*Update)' -count=1` and fix regressions

**Checkpoint**: User Story 3 is independently complete when no-wait output is accepted/submitted and no confirmation polling occurs.

---

## Phase 6: User Story 4 - Reject Invalid Or Unsupported Updates (Priority: P4)

**Goal**: Invalid payloads, missing targets, unsupported Camunda 8.7, and confirmation timeouts fail clearly before unsafe or misleading success.

**Independent Test**: Run malformed JSON, non-object JSON, missing `--vars`, missing key/stdin, Camunda 8.7, and forced confirmation-timeout scenarios and verify clear failures.

### Tests for User Story 4

- [x] T042 [P] [US4] Add command validation tests for missing `--vars`, malformed JSON, and non-object JSON in `cmd/update_processinstance_test.go`
- [x] T043 [P] [US4] Add command validation tests for missing `--key` and missing stdin input via `-` in `cmd/update_processinstance_test.go`
- [x] T044 [P] [US4] Add v8.7 unsupported-version command or service test proving mutation is not attempted in `cmd/update_processinstance_test.go` or `internal/services/processinstance/v87/service_test.go`
- [x] T045 [P] [US4] Add facade or command test for confirmation timeout/retry exhaustion reporting confirmation failure for the affected key in `c8volt/process/client_test.go` or `cmd/update_processinstance_test.go`
- [x] T046 [P] [US4] Add regression tests proving existing `run --vars` and `get process-instance --with-vars` behavior remains unchanged in `cmd/run_test.go` and `cmd/get_processinstance_test.go`

### Implementation for User Story 4

- [x] T047 [US4] Reject missing `--vars`, malformed JSON, and non-object JSON before creating a CLI client mutation request in `cmd/update_processinstance.go`
- [x] T048 [US4] Reject missing target keys through existing key target validation behavior in `cmd/update_processinstance.go`
- [x] T049 [US4] Ensure v8.7 returns unsupported-version errors before mutation through `internal/services/processinstance/v87/service.go` and facade error mapping in `c8volt/process/client.go`
- [x] T050 [US4] Report confirmation timeout or retry exhaustion as per-key confirmation failure in `c8volt/process/client.go`
- [x] T051 [US4] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process ./internal/services/processinstance/v87 -run 'Test(UpdateProcessInstance.*(Invalid|Missing|Unsupported|Timeout)|RunProcessInstance.*Vars|GetProcessInstance.*WithVars)' -count=1` and fix regressions

**Checkpoint**: User Story 4 is independently complete when invalid/unsupported/timeout paths fail clearly and existing variable commands still pass.

---

## Phase 7: Documentation & Command Discovery

**Purpose**: Make the new command discoverable and keep docs generated from source metadata.

- [x] T052 [P] Add help examples and command contract metadata coverage for `update`, `update process-instance`, and `update pi` in `cmd/update.go`, `cmd/update_processinstance.go`, and `cmd/command_contract_test.go`
- [x] T053 [P] Update README examples and automation guidance for process-instance variable updates in `README.md`
- [x] T054 [P] Update site documentation source examples for process-instance variable updates in `docs/index.md`
- [x] T055 Regenerate generated CLI documentation under `docs/cli/` with `make docs-content`
- [x] T056 Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test(CommandCapability|UpdateProcessInstance.*Help|VersionHelp)' -count=1` and fix docs/help regressions

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final cleanup, formatting, and repository-wide proof.

- [x] T057 [P] Run `gofmt -w cmd/update.go cmd/update_processinstance.go cmd/update_processinstance_test.go cmd/command_contract_test.go cmd/run_test.go cmd/get_processinstance_test.go c8volt/process/api.go c8volt/process/client.go c8volt/process/client_test.go c8volt/process/model.go internal/domain/processinstance.go internal/domain/processinstance_test.go internal/services/processinstance/api.go internal/services/processinstance/v87/contract.go internal/services/processinstance/v87/service.go internal/services/processinstance/v87/service_test.go internal/services/processinstance/v88/contract.go internal/services/processinstance/v88/variables.go internal/services/processinstance/v88/service_test.go internal/services/processinstance/v89/contract.go internal/services/processinstance/v89/variables.go internal/services/processinstance/v89/service_test.go`
- [x] T058 Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1` and fix regressions
- [x] T059 Run `make docs-content` and fix documentation generation issues
- [x] T060 Run `make test` and fix repository validation failures
- [x] T061 [P] Review [quickstart.md](./quickstart.md) against implemented behavior and update if command examples or validation commands changed
- [x] T062 Verify `git diff` contains only issue #179 implementation, docs, generated docs, and Speckit artifacts before commit

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on setup and blocks all user stories.
- **US1 (Phase 3)**: Depends on foundational command/service/facade surface and delivers the MVP.
- **US2 (Phase 4)**: Depends on US1 update flow and extends target selection/bulk reporting.
- **US3 (Phase 5)**: Depends on US1 mutation results and adds no-wait semantics.
- **US4 (Phase 6)**: Depends on final validation/update behavior from US1-US3.
- **Documentation (Phase 7)**: Depends on stable command semantics from user stories.
- **Polish (Phase 8)**: Depends on the desired user stories and docs being complete.

### User Story Dependencies

- **User Story 1 (P1)**: First user-visible slice; proves command registration, single-key mutation, confirmation, alias behavior, and output.
- **User Story 2 (P2)**: Builds on US1 by adding multi-key/stdin target handling and worker-controlled batch behavior.
- **User Story 3 (P3)**: Builds on US1 result modeling by adding submitted/no-wait behavior.
- **User Story 4 (P4)**: Hardens invalid input, unsupported versions, timeout paths, and regressions after command behavior exists.

### Parallel Opportunities

- T002 through T006 can run in parallel during setup.
- T008 through T013 can be split across domain/facade/service/command-contract files after T007 lands.
- T015 through T019 can be written in parallel for US1.
- T027 through T029 can be written in parallel for US2.
- T035 through T037 can be written in parallel for US3.
- T042 through T046 can be written in parallel for US4.
- T052 through T054 can be worked in parallel before docs regeneration.
- T057 and T061 can run in parallel after implementation is complete.

## Parallel Example: User Story 1

```text
Task: "Add command test for `update pi --key <key> --vars '{\"foo\":\"bar\"}'` submitting the v8.8 update request and confirming through variable lookup in cmd/update_processinstance_test.go"
Task: "Add v8.8 service test for `PUT /v2/element-instances/{elementInstanceKey}/variables` using the process instance key in internal/services/processinstance/v88/service_test.go"
Task: "Add v8.9 service test for `PUT /v2/element-instances/{elementInstanceKey}/variables` using the process instance key in internal/services/processinstance/v89/service_test.go"
Task: "Add facade test for normalized JSON confirmation comparing requested values to returned process-instance variables in c8volt/process/client_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete User Story 1 to deliver a single-key confirmed update through both command names.
3. Stop and run the US1 targeted tests before adding multi-key, no-wait, or validation hardening.

### Incremental Delivery

1. Add the update command shell and service/facade contracts.
2. Add single-key update with default confirmation.
3. Add multi-key and stdin target support.
4. Add `--no-wait` submitted output.
5. Add invalid input, unsupported version, timeout, and regression coverage.
6. Finish help, README, generated docs, formatting, targeted tests, and `make test`.

### Commit Guidance

Use Conventional Commit subjects and append the issue number as the final token, for example:

```text
feat(update): add process-instance variable updates #179
```
