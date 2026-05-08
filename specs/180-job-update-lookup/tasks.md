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

- [x] T001 Inspect get/update command registration and command metadata patterns in `cmd/get.go`, `cmd/update.go`, `cmd/get_processinstance.go`, `cmd/update_processinstance.go`, and `cmd/command_contract.go`
- [x] T002 [P] Inspect job-related generated client methods and types in `internal/clients/camunda/v88/camunda/client.gen.go` and `internal/clients/camunda/v89/camunda/client.gen.go`
- [x] T003 [P] Inspect versioned service package patterns in `internal/services/tenant/`, `internal/services/variable/`, and `internal/services/processinstance/`
- [x] T004 [P] Inspect existing waiter/backoff and confirmation patterns in `internal/services/processinstance/waiter/`, `internal/services/variable/waiter/`, `c8volt/process/client.go`, and `cmd/update_processinstance.go`
- [x] T005 [P] Inspect jobKey incident output and regressions in `cmd/cmd_views_processinstance_incidents.go`, `cmd/get_processinstance.go`, and `cmd/get_processinstance_test.go`
- [x] T006 [P] Inspect README, generated docs, and docs generation workflow in `README.md`, `docs/`, and `docsgen/`
- [x] T077 [P] Inspect current variable update dry-run, plan, confirmation, JSON, and no-op behavior in `cmd/update_processinstance.go`, `cmd/update_processinstance_payload.go`, `cmd/update_processinstance_plan.go`, and `cmd/update_processinstance_test.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add the shared job domain, facade, service contracts, and command shells used by every story.

**Critical**: No user story implementation should begin until this phase is complete.

- [x] T007 Add job domain request/result models in `internal/domain/job.go` and model tests in `internal/domain/job_test.go`
- [x] T008 Add dedicated job facade request/result models in `c8volt/job/model.go`
- [x] T009 Add dedicated job facade interface and client shell in `c8volt/job/api.go` and `c8volt/job/client.go`
- [x] T010 Add shared job service API and compile-time conformance expectations in `internal/services/job/api.go`
- [x] T011 Add job service factory and factory tests in `internal/services/job/factory.go` and `internal/services/job/factory_test.go`
- [x] T012 Add v8.7 unsupported job service shell in `internal/services/job/v87/contract.go`, `internal/services/job/v87/service.go`, and `internal/services/job/v87/service_test.go`
- [x] T013 Add v8.8 and v8.9 job service shells with compile-time conformance in `internal/services/job/v88/contract.go`, `internal/services/job/v88/service.go`, `internal/services/job/v89/contract.go`, and `internal/services/job/v89/service.go`
- [x] T014 Add command shells for `get job` and `update job` registration in `cmd/get_job.go` and `cmd/update_job.go`
- [x] T015 Add command contract discovery tests for `get job` and `update job` metadata in `cmd/command_contract_test.go`
- [x] T016 Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/job ./internal/domain ./internal/services/job ./internal/services/job/v87 -run 'Test(CommandCapability|Job|Unsupported)' -count=1` and fix foundational regressions
- [x] T078 Add job update plan models for current job state, requested retries, requested timeout, material-change classification, dry-run status, and mutation-submitted status in `c8volt/job/model.go` or the command-local model file chosen during discovery
- [x] T079 Add command contract tests proving `update job` exposes `--dry-run`, marks the command state-changing, and keeps mutation metadata automation-compatible in `cmd/command_contract_test.go`
- [x] T080 Add command validation tests proving `update job --json --verbose` is rejected before lookup or mutation in `cmd/update_job_test.go`
- [x] T081 Add command validation tests proving non-dry-run `--json update job` requires `--auto-confirm` or automation mode before lookup or mutation in `cmd/update_job_test.go`

**Checkpoint**: The command family, dedicated facade, service surface, and update planning contract compile before story behavior is implemented.

---

## Phase 3: User Story 1 - Inspect A Job By Key (Priority: P1) MVP

**Goal**: `c8volt get job --key <job-key>` returns job details for a supported Camunda 8.8 or 8.9 job and reports not-found when no matching job exists.

**Independent Test**: Run `c8volt get job --key <job-key>` and verify matching job details are returned for an existing job and a not-found result is returned for a missing job.

### Tests for User Story 1

- [x] T017 [P] [US1] Add command test for successful `get job --key <job-key>` human output in `cmd/get_job_test.go`
- [x] T018 [P] [US1] Add command test for successful `get job --key <job-key>` JSON output in `cmd/get_job_test.go`
- [x] T019 [P] [US1] Add command test for not-found job lookup in human and JSON modes in `cmd/get_job_test.go`
- [x] T020 [P] [US1] Add v8.8 service test for generated job search by key in `internal/services/job/v88/service_test.go`
- [x] T021 [P] [US1] Add v8.9 service test for generated job search by key in `internal/services/job/v89/service_test.go`
- [x] T022 [P] [US1] Add facade lookup tests for found and not-found results in `c8volt/job/client_test.go`

### Implementation for User Story 1

- [x] T023 [US1] Implement job detail domain and facade conversion helpers in `internal/domain/job.go` and `c8volt/job/model.go`
- [x] T024 [US1] Implement v8.8 job search and conversion in `internal/services/job/v88/convert.go` and `internal/services/job/v88/service.go`
- [x] T025 [US1] Implement v8.9 job search and conversion in `internal/services/job/v89/convert.go` and `internal/services/job/v89/service.go`
- [x] T026 [US1] Implement facade lookup orchestration in `c8volt/job/client.go`
- [x] T027 [US1] Implement `cmd/get_job.go` flag validation, service wiring, and human/JSON output for found and not-found results
- [x] T028 [US1] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/job ./internal/services/job/v88 ./internal/services/job/v89 -run 'Test(GetJob|JobLookup|SearchJobs)' -count=1` and fix regressions

**Checkpoint**: User Story 1 is independently complete when job lookup, not-found behavior, and human/JSON output pass.

---

## Phase 4: User Story 2 - Update Job Retries With Confirmation (Priority: P2)

**Goal**: `c8volt update job --key <job-key> --retries <count>` submits the retry update and confirms the requested retry count through job lookup.

**Independent Test**: Run `c8volt update job --key <job-key> --retries 3` and verify the command reports confirmed success only after lookup observes retries `3`.

### Tests for User Story 2

- [x] T029 [P] [US2] Add command test for `update job --key <job-key> --retries 3` submitted and confirmed output in `cmd/update_job_test.go`
- [x] T030 [P] [US2] Add command JSON output test for confirmed retry update in `cmd/update_job_test.go`
- [x] T031 [P] [US2] Add v8.8 service test for generated job update retries request in `internal/services/job/v88/service_test.go`
- [x] T032 [P] [US2] Add v8.9 service test for generated job update retries request in `internal/services/job/v89/service_test.go`
- [x] T033 [P] [US2] Add waiter test for retry confirmation success and exhaustion in `internal/services/job/waiter/waiter_test.go`
- [x] T034 [P] [US2] Add facade test for mutation failure skipping confirmation in `c8volt/job/client_test.go`
- [x] T082 [P] [US2] Add command test proving `update job --key <job-key> --retries 3 --dry-run` loads current job state, renders retry before/after, and submits no mutation in `cmd/update_job_test.go`
- [x] T083 [P] [US2] Add command test proving retry-only no-op requests report nothing to update and skip prompt and mutation in `cmd/update_job_test.go`
- [x] T084 [P] [US2] Add command test proving material interactive retry updates render a compact plan and require confirmation before mutation in `cmd/update_job_test.go`
- [x] T085 [P] [US2] Add command JSON dry-run test proving the full retry update plan payload is stable without verbose output in `cmd/update_job_test.go`

### Implementation for User Story 2

- [x] T035 [US2] Add job waiter implementation for retry confirmation in `internal/services/job/waiter/waiter.go`
- [x] T036 [US2] Implement v8.8 retry update request mapping in `internal/services/job/v88/service.go`
- [x] T037 [US2] Implement v8.9 retry update request mapping in `internal/services/job/v89/service.go`
- [x] T038 [US2] Implement facade retry update and default confirmation flow in `c8volt/job/client.go`
- [x] T039 [US2] Implement `cmd/update_job.go` `--retries` validation, service wiring, and confirmed human/JSON output
- [x] T040 [US2] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/job ./internal/services/job/waiter ./internal/services/job/v88 ./internal/services/job/v89 -run 'Test(UpdateJob.*Retries|RetryConfirmation|JobUpdateRetries)' -count=1` and fix regressions
- [x] T086 [US2] Implement retry plan construction from current job lookup state in `cmd/update_job.go` and the selected plan model file
- [x] T087 [US2] Implement `--dry-run` retry rendering and JSON payload without submitting mutation in `cmd/update_job.go` and `cmd/cmd_views_job.go`
- [x] T088 [US2] Implement retry-only no-op detection that skips prompt and mutation in `cmd/update_job.go`
- [x] T089 [US2] Implement interactive confirmation gate for material retry updates, reusing existing command confirmation helpers in `cmd/update_job.go`
- [x] T090 [US2] Implement JSON guardrails for retry updates, including rejection of `--json --verbose` and non-dry-run JSON mutations without auto-confirm or automation, in `cmd/update_job.go`

**Checkpoint**: User Story 2 is independently complete when retry updates are planned, dry-run rendered, no-op safe, confirmation-gated, submitted, confirmed, rendered, and failure modes distinguish mutation and confirmation failures.

---

## Phase 5: User Story 3 - Update Job Timeout Without Deadline Confirmation (Priority: P3)

**Goal**: `c8volt update job --key <job-key> --timeout <duration>` submits timeout milliseconds and returns accepted/submitted output without deadline confirmation.

**Independent Test**: Run `c8volt update job --key <job-key> --timeout 5m` and verify the command submits milliseconds and does not claim confirmed deadline state.

### Tests for User Story 3

- [x] T041 [P] [US3] Add command test for `update job --key <job-key> --timeout 5m` submitted output without confirmation polling in `cmd/update_job_test.go`
- [x] T042 [P] [US3] Add command test for combined `--retries 3 --timeout 5m` confirming retries only in `cmd/update_job_test.go`
- [x] T043 [P] [US3] Add v8.8 service test for generated job timeout milliseconds request in `internal/services/job/v88/service_test.go`
- [x] T044 [P] [US3] Add v8.9 service test for generated job timeout milliseconds request in `internal/services/job/v89/service_test.go`
- [x] T045 [P] [US3] Add facade test proving timeout-only updates skip deadline confirmation in `c8volt/job/client_test.go`
- [x] T091 [P] [US3] Add command test proving timeout dry-run reports timeout submission intent, performs no deadline comparison, and submits no mutation in `cmd/update_job_test.go`
- [x] T092 [P] [US3] Add command test proving combined retries-plus-timeout dry-run includes retry classification and timeout submission intent in `cmd/update_job_test.go`

### Implementation for User Story 3

- [x] T046 [US3] Implement timeout duration parsing and millisecond conversion in `cmd/update_job.go` or a repository-local helper used by the command
- [x] T047 [US3] Implement v8.8 timeout update request mapping in `internal/services/job/v88/service.go`
- [x] T048 [US3] Implement v8.9 timeout update request mapping in `internal/services/job/v89/service.go`
- [x] T049 [US3] Implement timeout-only submitted result behavior and combined retries-plus-timeout retries-only confirmation in `c8volt/job/client.go`
- [x] T050 [US3] Render timeout submitted fields without confirmed deadline claims in human and JSON output in `cmd/update_job.go`
- [x] T051 [US3] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/job ./internal/services/job/v88 ./internal/services/job/v89 -run 'Test(UpdateJob.*Timeout|TimeoutOnly|RetriesAndTimeout)' -count=1` and fix regressions
- [x] T093 [US3] Extend job update planning to mark timeout requests as material submission intent without deadline equality checks in `cmd/update_job.go`
- [x] T094 [US3] Extend dry-run and JSON renderers to show timeout submission intent without confirmed deadline claims in `cmd/update_job.go` and `cmd/cmd_views_job.go`

**Checkpoint**: User Story 3 is independently complete when timeout updates plan submission intent, dry-run without mutation, submit milliseconds, combined updates confirm retries only, and output never implies deadline confirmation.

---

## Phase 6: User Story 5 - Return After Accepted Update Without Waiting (Priority: P4)

**Goal**: `--no-wait` returns submitted output immediately after mutation acceptance and skips retry confirmation.

**Independent Test**: Run `c8volt update job --key <job-key> --retries 3 --no-wait` and verify no confirmation polling occurs.

### Tests for User Story 5

- [x] T052 [P] [US5] Add command test proving `--no-wait` skips retry confirmation for retries updates in `cmd/update_job_test.go`
- [x] T053 [P] [US5] Add command JSON output test for no-wait submitted results in `cmd/update_job_test.go`
- [x] T054 [P] [US5] Add facade test proving mutation errors still report failure when `--no-wait` is set in `c8volt/job/client_test.go`
- [x] T095 [P] [US5] Add command test proving `--no-wait` still uses the local confirmation gate for material interactive updates and only skips post-mutation polling in `cmd/update_job_test.go`

### Implementation for User Story 5

- [x] T055 [US5] Wire `--no-wait` into job update request options in `cmd/update_job.go` and `c8volt/job/model.go` without bypassing dry-run planning or the local confirmation gate
- [x] T056 [US5] Skip retry confirmation after accepted mutation when no-wait is set in `c8volt/job/client.go`
- [x] T057 [US5] Ensure human and JSON renderers show submitted status without implying confirmation in `cmd/update_job.go`
- [x] T058 [US5] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/job -run 'Test(UpdateJob.*NoWait|NoWait.*Job)' -count=1` and fix regressions

**Checkpoint**: User Story 5 is independently complete when no-wait output is submitted/accepted and no retry confirmation lookup occurs.

---

## Phase 7: User Story 6 - Preserve Boundaries And Existing Behavior (Priority: P5)

**Goal**: Job behavior remains isolated from process-instance and incident services while existing incident and process-instance update behavior stays unchanged.

**Independent Test**: Run targeted regressions for `get pi --with-incidents`, `update pi --vars`, and static/API checks that job lookup/update methods are absent from process-instance and incident service APIs.

### Tests for User Story 6

- [ ] T059 [P] [US6] Add regression test proving `get pi --with-incidents` still exposes `jobKey` unchanged in `cmd/get_processinstance_test.go`
- [ ] T060 [P] [US6] Add regression test proving `update pi --vars` planning, dry-run, and confirmation semantics remain unchanged in `cmd/update_processinstance_test.go`
- [ ] T061 [P] [US6] Add boundary test or static assertion that `internal/services/processinstance/api.go` and `internal/services/incident/api.go` do not expose job lookup/update/confirmation methods in `cmd/command_contract_test.go` or a focused internal test
- [ ] T062 [P] [US6] Add command/service test proving Camunda 8.7 job update fails unsupported before mutation in `cmd/update_job_test.go` or `internal/services/job/v87/service_test.go`

### Implementation for User Story 6

- [ ] T063 [US6] Ensure job service factory returns unsupported 8.7 behavior and supported 8.8/8.9 behavior in `internal/services/job/factory.go`
- [ ] T064 [US6] Keep command service wiring pointed at `c8volt/job` and `internal/services/job` without adding job methods to process-instance or incident APIs in `cmd/cmd_services.go`, `c8volt/job/client.go`, and `internal/services/job/api.go`
- [ ] T065 [US6] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/job ./internal/services/job/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -run 'Test(GetProcessInstance.*Incident|UpdateProcessInstance|UpdateJob.*Unsupported|ServiceBoundary)' -count=1` and fix regressions

**Checkpoint**: User Story 6 is independently complete when service boundaries are preserved and related existing commands keep their behavior.

---

## Phase 8: Documentation & Command Discovery

**Purpose**: Make the new commands discoverable and keep docs generated from source metadata.

- [ ] T066 [P] Add help examples and command contract metadata coverage for `get job` and `update job`, including `--dry-run`, `--no-wait`, `--auto-confirm`, and JSON guardrails, in `cmd/get_job.go`, `cmd/update_job.go`, and `cmd/command_contract_test.go`
- [ ] T067 [P] Update README examples for job lookup, job update dry-run, confirmed updates, and no-wait updates in `README.md`
- [ ] T068 [P] Update site documentation source examples for job lookup, job update dry-run, confirmed updates, and no-wait updates in `docs/index.md`
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
- **US2 (Phase 4)**: Depends on US1 lookup because retries confirmation and retry planning use job lookup.
- **US3 (Phase 5)**: Depends on the update request/result model from US2 but can be implemented without changing retry confirmation semantics.
- **US4 (Cross-cutting)**: Depends on US1 lookup and US2 update planning; its dry-run, no-op, JSON, and confirmation-gate tasks are included in US2 and US3 so the mutation path is never implemented without the safety contract.
- **US5 (Phase 6)**: Depends on US2 result modeling and adds no-wait behavior.
- **US6 (Phase 7)**: Depends on the final job service/command boundary from US1-US5.
- **Documentation (Phase 8)**: Depends on stable command semantics from user stories.
- **Polish (Phase 9)**: Depends on the desired user stories and docs being complete.

### User Story Dependencies

- **User Story 1 (P1)**: First user-visible slice; proves lookup, not-found behavior, output, and the lookup path needed by confirmation.
- **User Story 2 (P2)**: Builds on US1 by adding retry mutation and reliable retry confirmation.
- **User Story 3 (P3)**: Builds on US2 update request plumbing by adding timeout submission without deadline confirmation.
- **User Story 4 (P3)**: Cross-cuts US2 and US3 by adding dry-run planning, no-op detection, JSON guardrails, and confirmation gating before mutation.
- **User Story 5 (P4)**: Builds on US2 result modeling by adding submitted/no-wait behavior.
- **User Story 6 (P5)**: Hardens boundaries and existing behavior after the new job command paths exist.

### Parallel Opportunities

- T002 through T006 can run in parallel during setup.
- T007 through T015 can be split across domain, facade, service, and command-contract files after discovery.
- T017 through T022 can be written in parallel for US1.
- T029 through T034 can be written in parallel for US2.
- T041 through T045 can be written in parallel for US3.
- T052 through T054 and T095 can be written in parallel for US5.
- T059 through T062 can be written in parallel for US6.
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
3. Add update planning, `--dry-run`, retry no-op handling, JSON guardrails, and interactive confirmation before any retry mutation.
4. Add retry update with retries confirmation.
5. Add timeout planning with submitted intent, timeout mutation with submitted-only behavior, and combined retries-plus-timeout retries-only confirmation.
6. Add `--no-wait` submitted output without bypassing the local confirmation gate.
7. Add unsupported-version, boundary, and existing-command regression coverage.
8. Finish help, README, generated docs, formatting, targeted tests, and `make test`.

### Commit Guidance

Use Conventional Commit subjects and append the issue number as the final token, for example:

```text
feat(job): add job lookup and update commands #180
```
