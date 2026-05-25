# Tasks: Job Ops Workflow Primitives

**Input**: Design documents from `/specs/231-job-ops-workflows/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are required by the feature specification and constitution. Story test tasks must be written before implementation and should fail until each story is implemented.

**Ralph Context**: Every Ralph implementation iteration for this feature MUST include `--implementation-context specs/ralph-implementation-rules.md`.

**Organization**: Tasks are grouped by user story so each story can be implemented and verified as an independent increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files or only adds tests/docs
- **[Story]**: Maps to the user story from [spec.md](./spec.md)
- Every task names exact repository paths

## Phase 1: Setup (Shared Discovery)

**Purpose**: Confirm the current job command, generated-client, rendering, metadata, and docs surfaces before changing behavior.

- [x] T001 Inspect current job command behavior in `cmd/get_job.go`, `cmd/update_job.go`, and `cmd/cmd_views_job.go`
- [x] T002 [P] Inspect current job facade and domain models in `c8volt/job/model.go`, `c8volt/job/client.go`, and `internal/domain/job.go`
- [x] T003 [P] Inspect current job service contracts and versioned implementations in `internal/services/job/api.go`, `internal/services/job/v87/`, `internal/services/job/v88/`, and `internal/services/job/v89/`
- [x] T004 [P] Inspect generated v8.8/v8.9 job APIs in `internal/clients/camunda/v88/camunda/client.gen.go` and `internal/clients/camunda/v89/camunda/client.gen.go`
- [x] T005 [P] Inspect comparable search/list command patterns in `cmd/get_incident.go`, `cmd/get_processinstance.go`, and `internal/services/incident/`
- [x] T006 [P] Inspect README and generated docs expectations in `README.md`, `docs/cli/c8volt_get_job.md`, and `docs/cli/c8volt_update_job.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared models and command parsing structure that all user stories rely on.

**Critical**: No user story implementation should begin until this phase is complete.

- [x] T007 Add job search query, worker outcome, and expanded job detail domain models in `internal/domain/job.go`
- [x] T008 Add matching facade request/result models for job search, worker outcomes, and mutation plans in `c8volt/job/model.go`
- [x] T009 Extend job facade and service interfaces for search and worker outcome operations in `c8volt/job/api.go` and `internal/services/job/api.go`
- [x] T010 Add compile-time conformance updates for v8.7, v8.8, and v8.9 job services in `internal/services/job/api.go`, `internal/services/job/v87/contract.go`, `internal/services/job/v88/contract.go`, and `internal/services/job/v89/contract.go`
- [x] T011 Extend command metadata tests for new get/update job flags and automation behavior in `cmd/command_contract_test.go`
- [x] T012 Run targeted compile validation for foundational surfaces in `cmd/`, `c8volt/job/`, `internal/domain/`, and `internal/services/job/`

**Checkpoint**: Shared models, interfaces, metadata expectations, and package compilation are ready for story implementation.

---

## Phase 3: User Story 1 - Preserve Keyed Job Lookup And Current Updates (Priority: P1) MVP

**Goal**: Existing `get job --key`, retry update, and timeout update behavior remains compatible.

**Independent Test**: Existing keyed lookup and retry/timeout update tests pass without behavior drift.

### Tests for User Story 1

- [x] T013 [P] [US1] Add or update keyed lookup regression tests in `cmd/get_job_test.go`
- [x] T014 [P] [US1] Add or update retry update regression tests in `cmd/update_job_test.go`
- [x] T015 [P] [US1] Add or update timeout update regression tests in `cmd/update_job_test.go`
- [x] T016 [P] [US1] Add or update facade regression tests for existing update behavior in `c8volt/job/client_test.go`
- [x] T017 [P] [US1] Add or update v8.8/v8.9 service regression tests for existing search-by-key and update requests in `internal/services/job/v88/service_test.go` and `internal/services/job/v89/service_test.go`

### Implementation for User Story 1

- [x] T018 [US1] Refactor keyed lookup validation so `--key` remains exact lookup in `cmd/get_job.go`
- [x] T019 [US1] Preserve existing retry/timeout update request parsing while preparing for new modes in `cmd/update_job.go`
- [x] T020 [US1] Preserve existing retry/timeout conversion and service delegation in `c8volt/job/client.go`
- [x] T021 [US1] Verify US1 with targeted tests for `cmd/`, `c8volt/job/`, `internal/services/job/v88/`, and `internal/services/job/v89/`

**Checkpoint**: User Story 1 is complete when current job behavior is unchanged and protected by regression tests.

---

## Phase 4: User Story 2 - Search Jobs Without A Known Key (Priority: P2)

**Goal**: `get job` lists/searches jobs when `--key` is omitted.

**Independent Test**: `get job` without `--key` supports every specified search filter and output mode.

### Tests for User Story 2

- [x] T022 [P] [US2] Add command tests for keyed-vs-search validation in `cmd/get_job_test.go`
- [x] T023 [P] [US2] Add command tests for each search flag and invalid value in `cmd/get_job_test.go`
- [x] T024 [P] [US2] Add command tests proving no `flowNode` or `fni` aliases exist for jobs in `cmd/get_job_test.go`
- [x] T025 [P] [US2] Add output rendering tests for searched job rows, JSON, and keys-only output in `cmd/cmd_views_job_test.go`
- [x] T026 [P] [US2] Add v8.8/v8.9 service tests for generated job search filter construction in `internal/services/job/v88/service_test.go` and `internal/services/job/v89/service_test.go`
- [x] T027 [P] [US2] Add facade tests for job search query mapping and pagination/limit behavior in `c8volt/job/client_test.go`

### Implementation for User Story 2

- [x] T028 [US2] Add get job search flags, validation, and mode selection in `cmd/get_job.go`
- [x] T029 [US2] Add job search facade conversion and delegation in `c8volt/job/client.go`
- [x] T030 [US2] Add domain-to-generated filter builders for state, type, process instance key, element instance key, element ID, worker, retries, kind, listener event type, and limit in `internal/services/job/v88/service.go` and `internal/services/job/v89/service.go`
- [x] T031 [US2] Add job search result conversion fields in `internal/services/job/v88/convert.go`, `internal/services/job/v89/convert.go`, `internal/domain/job.go`, and `c8volt/job/model.go`
- [x] T032 [US2] Extend job list renderers for human, JSON, and keys-only output in `cmd/cmd_views_job.go`
- [x] T033 [US2] Verify US2 with targeted tests for `cmd/get_job_test.go`, `cmd/cmd_views_job_test.go`, `c8volt/job/client_test.go`, and versioned job service tests

**Checkpoint**: User Story 2 is complete when operators can discover jobs without knowing keys.

---

## Phase 5: User Story 3 - Report Technical Job Failure (Priority: P3)

**Goal**: `update job --fail` submits technical failure outcomes through the existing mutation safety model.

**Independent Test**: `update job --fail --retries <n> --message <text>` and retry-backoff variants dry-run, validate, submit, and render correctly.

### Tests for User Story 3

- [x] T034 [P] [US3] Add command validation tests for `--fail`, retry count, retry-backoff, message, dry-run, and mutual exclusion in `cmd/update_job_test.go`
- [x] T035 [P] [US3] Add command output tests for technical failure dry-run and submitted results in `cmd/update_job_test.go`
- [x] T036 [P] [US3] Add v8.8/v8.9 service tests for `FailJobWithResponse` request construction in `internal/services/job/v88/service_test.go` and `internal/services/job/v89/service_test.go`
- [x] T037 [P] [US3] Add facade tests for technical failure request mapping and mutation error handling in `c8volt/job/client_test.go`

### Implementation for User Story 3

- [x] T038 [US3] Add technical failure flags and validation in `cmd/update_job.go`
- [x] T039 [US3] Add technical failure request/result models and conversion in `c8volt/job/model.go` and `internal/domain/job.go`
- [x] T040 [US3] Implement technical failure facade delegation in `c8volt/job/client.go`
- [x] T041 [US3] Implement v8.8/v8.9 technical failure service calls in `internal/services/job/v88/service.go` and `internal/services/job/v89/service.go`
- [x] T042 [US3] Extend job mutation plan and result rendering for technical failure in `cmd/cmd_views_job.go`
- [x] T043 [US3] Verify US3 with targeted tests for `cmd/update_job_test.go`, `c8volt/job/client_test.go`, and versioned job service tests

**Checkpoint**: User Story 3 is complete when technical failure is a safe, explicit worker outcome primitive.

---

## Phase 6: User Story 4 - Throw A Modeled BPMN Error (Priority: P4)

**Goal**: `update job --throw-bpmn-error <code>` submits modeled BPMN errors safely.

**Independent Test**: BPMN error requests validate, dry-run, submit, reject incompatible flags, and render correctly.

### Tests for User Story 4

- [x] T044 [P] [US4] Add command validation tests for BPMN error code, message, variables if supported, and mutual exclusion in `cmd/update_job_test.go`
- [x] T045 [P] [US4] Add command output tests for BPMN error dry-run and submitted results in `cmd/update_job_test.go`
- [x] T046 [P] [US4] Add v8.8/v8.9 service tests for `ThrowJobErrorWithResponse` request construction in `internal/services/job/v88/service_test.go` and `internal/services/job/v89/service_test.go`
- [x] T047 [P] [US4] Add facade tests for BPMN error request mapping and mutation error handling in `c8volt/job/client_test.go`

### Implementation for User Story 4

- [x] T048 [US4] Add BPMN error flags and validation in `cmd/update_job.go`
- [x] T049 [US4] Add BPMN error request/result models and conversion in `c8volt/job/model.go` and `internal/domain/job.go`
- [x] T050 [US4] Implement BPMN error facade delegation in `c8volt/job/client.go`
- [x] T051 [US4] Implement v8.8/v8.9 BPMN error service calls in `internal/services/job/v88/service.go` and `internal/services/job/v89/service.go`
- [x] T052 [US4] Extend job mutation plan and result rendering for BPMN error in `cmd/cmd_views_job.go`
- [x] T053 [US4] Verify US4 with targeted tests for `cmd/update_job_test.go`, `c8volt/job/client_test.go`, and versioned job service tests

**Checkpoint**: User Story 4 is complete when modeled BPMN errors are explicit and cannot be confused with technical failures.

---

## Phase 7: User Story 5 - Complete A Job With Variables (Priority: P5)

**Goal**: `update job --complete --vars <json>` completes jobs with optional variables.

**Independent Test**: Completion requests validate JSON variables, support omitted variables, dry-run, submit, reject incompatible flags, and render correctly.

### Tests for User Story 5

- [x] T054 [P] [US5] Add command validation tests for `--complete`, optional `--vars`, invalid JSON, and mutual exclusion in `cmd/update_job_test.go`
- [x] T055 [P] [US5] Add command output tests for completion dry-run and submitted results in `cmd/update_job_test.go`
- [x] T056 [P] [US5] Add v8.8/v8.9 service tests for `CompleteJobWithResponse` request construction in `internal/services/job/v88/service_test.go` and `internal/services/job/v89/service_test.go`
- [x] T057 [P] [US5] Add facade tests for completion request mapping and mutation error handling in `c8volt/job/client_test.go`

### Implementation for User Story 5

- [x] T058 [US5] Add completion flags and variable JSON validation in `cmd/update_job.go`
- [x] T059 [US5] Add completion request/result models and conversion in `c8volt/job/model.go` and `internal/domain/job.go`
- [x] T060 [US5] Implement completion facade delegation in `c8volt/job/client.go`
- [x] T061 [US5] Implement v8.8/v8.9 completion service calls in `internal/services/job/v88/service.go` and `internal/services/job/v89/service.go`
- [x] T062 [US5] Extend job mutation plan and result rendering for completion in `cmd/cmd_views_job.go`
- [x] T063 [US5] Verify US5 with targeted tests for `cmd/update_job_test.go`, `c8volt/job/client_test.go`, and versioned job service tests

**Checkpoint**: User Story 5 is complete when completion is a safe, scriptable worker outcome primitive.

---

## Phase 8: User Story 6 - Keep Job Primitives Safe, Versioned, And Documented (Priority: P6)

**Goal**: All new job behavior follows command contracts, version gates, documentation, and repository boundaries.

**Independent Test**: Command contract, docs, unsupported-version, and regression tests pass.

### Tests for User Story 6

- [x] T064 [P] [US6] Add v8.7 unsupported tests for job search and worker outcomes in `internal/services/job/v87/service_test.go`
- [x] T065 [P] [US6] Add command tests for unsupported 8.7 behavior before mutation in `cmd/update_job_test.go` and `cmd/get_job_test.go`
- [x] T066 [P] [US6] Add command contract metadata tests for new flags, mutation modes, output modes, and automation support in `cmd/command_contract_test.go`
- [x] T067 [P] [US6] Add regression tests proving process-instance and incident service APIs do not gain job behavior in `internal/services/processinstance/api.go`, `internal/services/incident/api.go`, or a focused static test file

### Implementation for User Story 6

- [x] T068 [US6] Implement v8.7 unsupported behavior for job search and worker outcomes in `internal/services/job/v87/service.go`
- [x] T069 [US6] Update command help and metadata for all new get/update job flags in `cmd/get_job.go`, `cmd/update_job.go`, and `cmd/command_contract.go`
- [x] T070 [US6] Update README examples for job search and worker outcomes in `README.md`
- [x] T071 [US6] Regenerate CLI documentation for changed commands in `docs/cli/c8volt_get_job.md` and `docs/cli/c8volt_update_job.md`
- [x] T072 [US6] Verify US6 with targeted command contract, v8.7 unsupported, docs generation, and boundary tests in `cmd/`, `internal/services/job/v87/`, and `docs/cli/`

**Checkpoint**: User Story 6 is complete when job primitives are documented, version-gated, and safe for scripts and future ops workflows.

---

## Phase 9: Polish & Cross-Cutting Validation

**Purpose**: Final cleanup, generated docs, and repository validation.

- [x] T073 [P] Run gofmt for changed Go files under `cmd/`, `c8volt/job/`, `internal/domain/`, and `internal/services/job/`
- [x] T074 Run targeted Go tests for changed packages in `cmd/`, `c8volt/job/`, `internal/domain/`, `internal/services/job/`, `internal/services/job/v87/`, `internal/services/job/v88/`, and `internal/services/job/v89/`
- [x] T075 Run `make docs-content` for README/docs command metadata changes affecting `docs/cli/`
- [x] T076 Run `make test` for repository validation from the repository root
- [x] T077 [P] Review [quickstart.md](./quickstart.md) against implemented behavior and update examples if flags or validation commands changed
- [x] T078 Review `git diff` to ensure changes are scoped to issue #231 artifacts, implementation, tests, README, and generated docs

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on setup and blocks user stories.
- **US1 (Phase 3)**: Depends on foundational models and validates compatibility baseline.
- **US2 (Phase 4)**: Depends on US1 keyed/search separation and shared query models.
- **US3 (Phase 5)**: Depends on foundational mutation-mode parsing and existing update safety model.
- **US4 (Phase 6)**: Depends on foundational worker outcome model and mutation safety model.
- **US5 (Phase 7)**: Depends on foundational worker outcome model and shared variable parsing decisions.
- **US6 (Phase 8)**: Depends on all desired command behavior being stable.
- **Polish (Phase 9)**: Depends on implementation and documentation phases.

### User Story Dependencies

- **User Story 1 (P1)**: First MVP slice; protects existing behavior.
- **User Story 2 (P2)**: Can start after foundational work and US1 compatibility validation.
- **User Story 3 (P3)**: Can start after foundational worker outcome structures are available.
- **User Story 4 (P4)**: Can start after foundational worker outcome structures are available.
- **User Story 5 (P5)**: Can start after foundational worker outcome structures are available.
- **User Story 6 (P6)**: Runs after implementation stories to align version gates, metadata, docs, and boundaries.

### Parallel Opportunities

- T002 through T006 can run in parallel during setup.
- T013 through T017 can be written in parallel for US1.
- T022 through T027 can be written in parallel for US2.
- T034 through T037 can be written in parallel for US3.
- T044 through T047 can be written in parallel for US4.
- T054 through T057 can be written in parallel for US5.
- T064 through T067 can be written in parallel for US6.
- T073 and T077 can run in parallel after implementation stabilizes.

## Parallel Example: User Story 2

```bash
# Parallel test-writing targets for job search:
Task: "T023 Add command tests for each search flag and invalid value in cmd/get_job_test.go"
Task: "T025 Add output rendering tests for searched job rows, JSON, and keys-only output in cmd/cmd_views_job_test.go"
Task: "T026 Add v8.8/v8.9 service tests for generated job search filter construction in internal/services/job/v88/service_test.go and internal/services/job/v89/service_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete User Story 1 to prove current behavior is preserved.
3. Stop and run the US1 targeted tests before expanding the command surface.

### Incremental Delivery

1. Add job search/list mode as the first new read-only increment.
2. Add technical failure as the first worker outcome mutation.
3. Add BPMN error and completion as separate increments.
4. Finish version gates, metadata, README, generated docs, and broader validation.

### Ralph Launch Guard

Do not launch Ralph unless the launch command or hook instructions include `--implementation-context specs/ralph-implementation-rules.md`. Treat this as required budget and context confirmation, not an optional note.

## Notes

- Keep each Ralph iteration to the current task or story slice only.
- Tests for a story should be written before implementation and should fail until the implementation is complete.
- Commit only after validation passes, using a Conventional Commit subject ending in `#231`.
