# Tasks: Job Lookup And Updates

**Input**: Design documents from `/specs/180-job-update-lookup/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are required by the feature specification and constitution. Story test tasks should be written before implementation and should fail until the story implementation is complete.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files or only adds tests/docs
- **[Story]**: Maps to the user story from [spec.md](./spec.md)
- Every task names exact repository paths

## Phase 1: Setup (Shared Discovery)

**Purpose**: Confirm existing command, service, generated-client, rendering, waiter, and docs paths before changing behavior.

- [ ] T001 Inspect get/update command registration and command metadata patterns in `cmd/get.go`, `cmd/update.go`, `cmd/get_processinstance.go`, `cmd/update_processinstance.go`, and `cmd/command_contract.go`
- [ ] T002 [P] Inspect job-related generated client methods and types in `internal/clients/camunda/v88/camunda/client.gen.go` and `internal/clients/camunda/v89/camunda/client.gen.go`
- [ ] T003 [P] Inspect versioned service package patterns in `internal/services/tenant/`, `internal/services/variable/`, and `internal/services/processinstance/`
- [ ] T004 [P] Inspect existing waiter/backoff and confirmation patterns in `internal/services/processinstance/waiter/`, `internal/services/variable/waiter/`, `c8volt/process/client.go`, and `cmd/update_processinstance.go`
- [ ] T005 [P] Inspect jobKey incident output and regressions in `cmd/cmd_views_processinstance_incidents.go`, `cmd/get_processinstance.go`, and `cmd/get_processinstance_test.go`
- [ ] T006 [P] Inspect README, generated docs, and docs generation workflow in `README.md`, `docs/`, and `docsgen/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add the shared job domain, facade, service contracts, and command shells used by every story.

**Critical**: No user story implementation should begin until this phase is complete.

- [ ] T007 Add job domain request/result models in `internal/domain/job.go` and model tests in `internal/domain/job_test.go`
- [ ] T008 Add dedicated job facade request/result models in `c8volt/job/model.go`
- [ ] T009 Add dedicated job facade interface and client shell in `c8volt/job/api.go` and `c8volt/job/client.go`
- [ ] T010 Add shared job service API and compile-time conformance expectations in `internal/services/job/api.go`
- [ ] T011 Add job service factory and factory tests in `internal/services/job/factory.go` and `internal/services/job/factory_test.go`
- [ ] T012 Add v8.7 unsupported job service shell in `internal/services/job/v87/contract.go`, `internal/services/job/v87/service.go`, and `internal/services/job/v87/service_test.go`
- [ ] T013 Add v8.8 and v8.9 job service shells with compile-time conformance in `internal/services/job/v88/contract.go`, `internal/services/job/v88/service.go`, `internal/services/job/v89/contract.go`, and `internal/services/job/v89/service.go`
- [ ] T014 Add command shells for `get job` and `update job` registration in `cmd/get_job.go` and `cmd/update_job.go`
- [ ] T015 Add command contract discovery tests for `get job` and `update job` metadata in `cmd/command_contract_test.go`
- [ ] T016 Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/job ./internal/domain ./internal/services/job ./internal/services/job/v87 -run 'Test(CommandCapability|Job|Unsupported)' -count=1` and fix foundational regressions

**Checkpoint**: The command family, dedicated facade, and service surface compile before story behavior is implemented.

---

## Phase 3: User Story 1 - Inspect A Job By Key (Priority: P1) MVP

**Goal**: `c8volt get job --key <job-key>` returns job details for a supported Camunda 8.8 or 8.9 job and reports not-found when no matching job exists.

**Independent Test**: Run `c8volt get job --key <job-key>` and verify matching job details are returned for an existing job and a not-found result is returned for a missing job.

### Tests for User Story 1

- [ ] T017 [P] [US1] Add command test for successful `get job --key <job-key>` human output in `cmd/get_job_test.go`
- [ ] T018 [P] [US1] Add command test for successful `get job --key <job-key>` JSON output in `cmd/get_job_test.go`
- [ ] T019 [P] [US1] Add command test for not-found job lookup in human and JSON modes in `cmd/get_job_test.go`
- [ ] T020 [P] [US1] Add v8.8 service test for generated job search by key in `internal/services/job/v88/service_test.go`
- [ ] T021 [P] [US1] Add v8.9 service test for generated job search by key in `internal/services/job/v89/service_test.go`
- [ ] T022 [P] [US1] Add facade lookup tests for found and not-found results in `c8volt/job/client_test.go`

### Implementation for User Story 1

- [ ] T023 [US1] Implement job detail domain and facade conversion helpers in `internal/domain/job.go` and `c8volt/job/model.go`
- [ ] T024 [US1] Implement v8.8 job search and conversion in `internal/services/job/v88/convert.go` and `internal/services/job/v88/service.go`
- [ ] T025 [US1] Implement v8.9 job search and conversion in `internal/services/job/v89/convert.go` and `internal/services/job/v89/service.go`
- [ ] T026 [US1] Implement facade lookup orchestration in `c8volt/job/client.go`
- [ ] T027 [US1] Implement `cmd/get_job.go` flag validation, service wiring, and human/JSON output for found and not-found results
- [ ] T028 [US1] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/job ./internal/services/job/v88 ./internal/services/job/v89 -run 'Test(GetJob|JobLookup|SearchJobs)' -count=1` and fix regressions

**Checkpoint**: User Story 1 is independently complete when job lookup, not-found behavior, and human/JSON output pass.

---

## Phase 4: User Story 2 - Update Job Retries With Confirmation (Priority: P2)

**Goal**: `c8volt update job --key <job-key> --retries <count>` submits the retry update and confirms the requested retry count through job lookup.

**Independent Test**: Run `c8volt update job --key <job-key> --retries 3` and verify the command reports confirmed success only after lookup observes retries `3`.

### Tests for User Story 2

- [ ] T029 [P] [US2] Add command test for `update job --key <job-key> --retries 3` submitted and confirmed output in `cmd/update_job_test.go`
- [ ] T030 [P] [US2] Add command JSON output test for confirmed retry update in `cmd/update_job_test.go`
- [ ] T031 [P] [US2] Add v8.8 service test for generated job update retries request in `internal/services/job/v88/service_test.go`
- [ ] T032 [P] [US2] Add v8.9 service test for generated job update retries request in `internal/services/job/v89/service_test.go`
- [ ] T033 [P] [US2] Add waiter test for retry confirmation success and exhaustion in `internal/services/job/waiter/waiter_test.go`
- [ ] T034 [P] [US2] Add facade test for mutation failure skipping confirmation in `c8volt/job/client_test.go`

### Implementation for User Story 2

- [ ] T035 [US2] Add job waiter implementation for retry confirmation in `internal/services/job/waiter/waiter.go`
- [ ] T036 [US2] Implement v8.8 retry update request mapping in `internal/services/job/v88/service.go`
- [ ] T037 [US2] Implement v8.9 retry update request mapping in `internal/services/job/v89/service.go`
- [ ] T038 [US2] Implement facade retry update and default confirmation flow in `c8volt/job/client.go`
- [ ] T039 [US2] Implement `cmd/update_job.go` `--retries` validation, service wiring, and confirmed human/JSON output
- [ ] T040 [US2] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/job ./internal/services/job/waiter ./internal/services/job/v88 ./internal/services/job/v89 -run 'Test(UpdateJob.*Retries|RetryConfirmation|JobUpdateRetries)' -count=1` and fix regressions

**Checkpoint**: User Story 2 is independently complete when retry updates are submitted, confirmed, rendered, and failure modes distinguish mutation and confirmation failures.

---

## Phase 5: User Story 3 - Update Job Timeout Without Deadline Confirmation (Priority: P3)

**Goal**: `c8volt update job --key <job-key> --timeout <duration>` submits timeout milliseconds and returns accepted/submitted output without deadline confirmation.

**Independent Test**: Run `c8volt update job --key <job-key> --timeout 5m` and verify the command submits milliseconds and does not claim confirmed deadline state.

### Tests for User Story 3

- [ ] T041 [P] [US3] Add command test for `update job --key <job-key> --timeout 5m` submitted output without confirmation polling in `cmd/update_job_test.go`
- [ ] T042 [P] [US3] Add command test for combined `--retries 3 --timeout 5m` confirming retries only in `cmd/update_job_test.go`
- [ ] T043 [P] [US3] Add v8.8 service test for generated job timeout milliseconds request in `internal/services/job/v88/service_test.go`
- [ ] T044 [P] [US3] Add v8.9 service test for generated job timeout milliseconds request in `internal/services/job/v89/service_test.go`
- [ ] T045 [P] [US3] Add facade test proving timeout-only updates skip deadline confirmation in `c8volt/job/client_test.go`

### Implementation for User Story 3

- [ ] T046 [US3] Implement timeout duration parsing and millisecond conversion in `cmd/update_job.go` or a repository-local helper used by the command
- [ ] T047 [US3] Implement v8.8 timeout update request mapping in `internal/services/job/v88/service.go`
- [ ] T048 [US3] Implement v8.9 timeout update request mapping in `internal/services/job/v89/service.go`
- [ ] T049 [US3] Implement timeout-only submitted result behavior and combined retries-plus-timeout retries-only confirmation in `c8volt/job/client.go`
- [ ] T050 [US3] Render timeout submitted fields without confirmed deadline claims in human and JSON output in `cmd/update_job.go`
- [ ] T051 [US3] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/job ./internal/services/job/v88 ./internal/services/job/v89 -run 'Test(UpdateJob.*Timeout|TimeoutOnly|RetriesAndTimeout)' -count=1` and fix regressions

**Checkpoint**: User Story 3 is independently complete when timeout updates submit milliseconds, combined updates confirm retries only, and output never implies deadline confirmation.

---

## Phase 6: User Story 4 - Return After Accepted Update Without Waiting (Priority: P4)

**Goal**: `--no-wait` returns submitted output immediately after mutation acceptance and skips retry confirmation.

**Independent Test**: Run `c8volt update job --key <job-key> --retries 3 --no-wait` and verify no confirmation polling occurs.

### Tests for User Story 4

- [ ] T052 [P] [US4] Add command test proving `--no-wait` skips retry confirmation for retries updates in `cmd/update_job_test.go`
- [ ] T053 [P] [US4] Add command JSON output test for no-wait submitted results in `cmd/update_job_test.go`
- [ ] T054 [P] [US4] Add facade test proving mutation errors still report failure when `--no-wait` is set in `c8volt/job/client_test.go`

### Implementation for User Story 4

- [ ] T055 [US4] Wire `--no-wait` into job update request options in `cmd/update_job.go` and `c8volt/job/model.go`
- [ ] T056 [US4] Skip retry confirmation after accepted mutation when no-wait is set in `c8volt/job/client.go`
- [ ] T057 [US4] Ensure human and JSON renderers show submitted status without implying confirmation in `cmd/update_job.go`
- [ ] T058 [US4] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/job -run 'Test(UpdateJob.*NoWait|NoWait.*Job)' -count=1` and fix regressions

**Checkpoint**: User Story 4 is independently complete when no-wait output is submitted/accepted and no retry confirmation lookup occurs.

---

## Phase 7: User Story 5 - Preserve Boundaries And Existing Behavior (Priority: P5)

**Goal**: Job behavior remains isolated from process-instance and incident services while existing incident and process-instance update behavior stays unchanged.

**Independent Test**: Run targeted regressions for `get pi --with-incidents`, `update pi --vars`, and static/API checks that job lookup/update methods are absent from process-instance and incident service APIs.

### Tests for User Story 5

- [ ] T059 [P] [US5] Add regression test proving `get pi --with-incidents` still exposes `jobKey` unchanged in `cmd/get_processinstance_test.go`
- [ ] T060 [P] [US5] Add regression test proving `update pi --vars` confirmation semantics remain unchanged in `cmd/update_processinstance_test.go`
- [ ] T061 [P] [US5] Add boundary test or static assertion that `internal/services/processinstance/api.go` and `internal/services/incident/api.go` do not expose job lookup/update/confirmation methods in `cmd/command_contract_test.go` or a focused internal test
- [ ] T062 [P] [US5] Add command/service test proving Camunda 8.7 job update fails unsupported before mutation in `cmd/update_job_test.go` or `internal/services/job/v87/service_test.go`

### Implementation for User Story 5

- [ ] T063 [US5] Ensure job service factory returns unsupported 8.7 behavior and supported 8.8/8.9 behavior in `internal/services/job/factory.go`
- [ ] T064 [US5] Keep command service wiring pointed at `c8volt/job` and `internal/services/job` without adding job methods to process-instance or incident APIs in `cmd/cmd_services.go`, `c8volt/job/client.go`, and `internal/services/job/api.go`
- [ ] T065 [US5] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/job ./internal/services/job/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -run 'Test(GetProcessInstance.*Incident|UpdateProcessInstance|UpdateJob.*Unsupported|ServiceBoundary)' -count=1` and fix regressions

**Checkpoint**: User Story 5 is independently complete when service boundaries are preserved and related existing commands keep their behavior.

---

## Phase 8: Documentation & Command Discovery

**Purpose**: Make the new commands discoverable and keep docs generated from source metadata.

- [ ] T066 [P] Add help examples and command contract metadata coverage for `get job` and `update job` in `cmd/get_job.go`, `cmd/update_job.go`, and `cmd/command_contract_test.go`
- [ ] T067 [P] Update README examples for job lookup and job updates in `README.md`
- [ ] T068 [P] Update site documentation source examples for job lookup and job updates in `docs/index.md`
- [ ] T069 Regenerate generated CLI documentation under `docs/cli/` with `make docs-content`
- [ ] T070 Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test(CommandCapability|GetJob.*Help|UpdateJob.*Help|VersionHelp)' -count=1` and fix docs/help regressions

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Final cleanup, formatting, and repository-wide proof.

- [ ] T071 [P] Run `gofmt -w cmd/get_job.go cmd/get_job_test.go cmd/update_job.go cmd/update_job_test.go cmd/command_contract_test.go cmd/get_processinstance_test.go cmd/update_processinstance_test.go c8volt/job/api.go c8volt/job/client.go c8volt/job/client_test.go c8volt/job/model.go internal/domain/job.go internal/domain/job_test.go internal/services/job/api.go internal/services/job/factory.go internal/services/job/factory_test.go internal/services/job/waiter/waiter.go internal/services/job/waiter/waiter_test.go internal/services/job/v87/contract.go internal/services/job/v87/service.go internal/services/job/v87/service_test.go internal/services/job/v88/contract.go internal/services/job/v88/convert.go internal/services/job/v88/service.go internal/services/job/v88/service_test.go internal/services/job/v89/contract.go internal/services/job/v89/convert.go internal/services/job/v89/service.go internal/services/job/v89/service_test.go`
- [ ] T072 Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/job ./internal/domain ./internal/services/job ./internal/services/job/waiter ./internal/services/job/v87 ./internal/services/job/v88 ./internal/services/job/v89 -count=1` and fix regressions
- [ ] T073 Run `make docs-content` and fix documentation generation issues
- [ ] T074 Run `make test` and fix repository validation failures
- [ ] T075 [P] Review [quickstart.md](./quickstart.md) against implemented behavior and update if command examples or validation commands changed
- [ ] T076 Verify `git diff` contains only issue #180 implementation, docs, generated docs, and Speckit artifacts before commit

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on setup and blocks all user stories.
- **US1 (Phase 3)**: Depends on foundational command/service/facade surface and delivers the MVP.
- **US2 (Phase 4)**: Depends on US1 lookup because retries confirmation uses job lookup.
- **US3 (Phase 5)**: Depends on the update request/result model from US2 but can be implemented without changing retry confirmation semantics.
- **US4 (Phase 6)**: Depends on US2 result modeling and adds no-wait behavior.
- **US5 (Phase 7)**: Depends on the final job service/command boundary from US1-US4.
- **Documentation (Phase 8)**: Depends on stable command semantics from user stories.
- **Polish (Phase 9)**: Depends on the desired user stories and docs being complete.

### User Story Dependencies

- **User Story 1 (P1)**: First user-visible slice; proves lookup, not-found behavior, output, and the lookup path needed by confirmation.
- **User Story 2 (P2)**: Builds on US1 by adding retry mutation and reliable retry confirmation.
- **User Story 3 (P3)**: Builds on US2 update request plumbing by adding timeout submission without deadline confirmation.
- **User Story 4 (P4)**: Builds on US2 result modeling by adding submitted/no-wait behavior.
- **User Story 5 (P5)**: Hardens boundaries and existing behavior after the new job command paths exist.

### Parallel Opportunities

- T002 through T006 can run in parallel during setup.
- T007 through T015 can be split across domain, facade, service, and command-contract files after discovery.
- T017 through T022 can be written in parallel for US1.
- T029 through T034 can be written in parallel for US2.
- T041 through T045 can be written in parallel for US3.
- T052 through T054 can be written in parallel for US4.
- T059 through T062 can be written in parallel for US5.
- T066 through T068 can be worked in parallel before docs regeneration.
- T071 and T075 can run in parallel after implementation is complete.

## Parallel Example: User Story 1

```text
Task: "Add command test for successful get job --key <job-key> human output in cmd/get_job_test.go"
Task: "Add v8.8 service test for generated job search by key in internal/services/job/v88/service_test.go"
Task: "Add v8.9 service test for generated job search by key in internal/services/job/v89/service_test.go"
Task: "Add facade lookup tests for found and not-found results in c8volt/job/client_test.go"
```

## Parallel Example: User Story 2

```text
Task: "Add command test for update job --key <job-key> --retries 3 submitted and confirmed output in cmd/update_job_test.go"
Task: "Add v8.8 service test for generated job update retries request in internal/services/job/v88/service_test.go"
Task: "Add v8.9 service test for generated job update retries request in internal/services/job/v89/service_test.go"
Task: "Add waiter test for retry confirmation success and exhaustion in internal/services/job/waiter/waiter_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete User Story 1 to deliver job lookup and not-found behavior.
3. Stop and run the US1 targeted tests before adding mutation behavior.

### Incremental Delivery

1. Add dedicated job domain, facade, service, and command shells.
2. Add job lookup with human/JSON output.
3. Add retry update with retries confirmation.
4. Add timeout update with submitted-only behavior and combined retries-plus-timeout retries-only confirmation.
5. Add `--no-wait` submitted output.
6. Add unsupported-version, boundary, and existing-command regression coverage.
7. Finish help, README, generated docs, formatting, targeted tests, and `make test`.

### Commit Guidance

Use Conventional Commit subjects and append the issue number as the final token, for example:

```text
feat(job): add job lookup and update commands #180
```
