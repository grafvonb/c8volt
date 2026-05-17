# Tasks: Ops Execute Smoke Test

**Input**: Design documents from `specs/188-ops-smoke-test/`
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)
**Mandatory Ralph Context**: Every Ralph iteration MUST be launched with `--implementation-context specs/ralph-implementation-rules.md` and must apply that file before implementation.
**Issue Commit Rule**: Every commit subject for this feature MUST use Conventional Commits and end with `#188`.

**Tests**: Tests are required by the feature specification, repository constitution, and Ralph implementation rules.

**Organization**: Tasks are grouped by small, independently testable user stories. Each Ralph iteration should complete only the current work unit and update [progress.md](progress.md).

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Capture required implementation context and current ops workflow patterns before code changes.

- [x] T001 Record mandatory Ralph context and issue traceability in `specs/188-ops-smoke-test/progress.md`
- [x] T002 Inspect existing #186/#187/#199 ops workflows, `ops execute` command group, report helpers, process-instance run/walk/delete behavior, process-definition delete behavior, embedded fixtures, command contract metadata, and docs generation patterns; record reusable discoveries in `specs/188-ops-smoke-test/progress.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add smoke-test request/result models and service/facade seams needed by all stories.

**CRITICAL**: No user story work can begin until this phase is complete.

- [x] T003 [P] Define internal smoke-test request/result domain models in `internal/domain/ops_smoke_test.go`
- [x] T004 [P] Define public ops smoke-test request/result models in `c8volt/ops/model.go`
- [x] T005 [P] Extend public ops facade API for smoke-test execution in `c8volt/ops/api.go`
- [x] T006 Extend internal ops service interface for smoke-test execution in `internal/services/ops/api.go`
- [x] T007 Implement public/internal smoke-test model conversions in `c8volt/ops/convert.go`
- [x] T008 Implement thin public ops facade smoke-test method in `c8volt/ops/client.go`
- [x] T009 [P] Add foundational ops facade wiring tests for smoke-test execution in `c8volt/ops/client_test.go`
- [x] T010 [P] Add foundational internal ops service validation tests for smoke-test request shape in `internal/services/ops/smoke_test_test.go`
- [x] T011 Mark Phase 2 tasks complete and record validation notes in `specs/188-ops-smoke-test/progress.md`

**Checkpoint**: Smoke-test workflow model, facade, and service boundary are available for story implementation.

---

## Phase 3: User Story 1 - Register Smoke Test Command Surface (Priority: P1) MVP

**Goal**: `c8volt ops execute smoke-test` exists under the execute group, validates local command shape, and performs no workflow at the grouping level.

**Independent Test**: Run command help, invalid flag, and command contract tests without Camunda data.

### Tests for User Story 1

- [x] T012 [P] [US1] Add command registration and help tests for `ops execute smoke-test` in `cmd/ops_execute_smoke_test_test.go`
- [x] T013 [P] [US1] Add invalid `--count`, invalid `--report-format`, and missing dependent report flag subprocess tests in `cmd/ops_execute_smoke_test_test.go`
- [x] T014 [P] [US1] Add command contract metadata tests for smoke-test state-changing and automation support in `cmd/command_contract_test.go`

### Implementation for User Story 1

- [x] T015 [US1] Add `ops execute smoke-test` Cobra command, summary, examples, and count/report flags in `cmd/ops_execute_smoke_test.go`
- [x] T016 [US1] Wire smoke-test command into the existing execute group in `cmd/ops_execute.go`
- [x] T017 [US1] Implement local count and report flag validation with invalid-input error mapping in `cmd/ops_execute_smoke_test.go`
- [x] T018 [US1] Set mutation, output-mode, report, count, worker, cleanup, and automation metadata in `cmd/ops_execute_smoke_test.go`
- [x] T019 [US1] Mark US1 tasks complete and record validation notes in `specs/188-ops-smoke-test/progress.md`

**Checkpoint**: User Story 1 is independently functional and testable.

---

## Phase 4: User Story 2 - Dry-Run Smoke Test Planning (Priority: P2)

**Goal**: `--dry-run` validates configuration, read-only connectivity, fixture availability, concurrency settings, cleanup intent, and report options without mutation.

**Independent Test**: Run dry-run command/service tests verifying planned steps, read-only behavior, JSON output, and optional dry-run report generation.

### Tests for User Story 2

- [x] T020 [P] [US2] Add ops service dry-run planning tests for no mutation and planned steps in `internal/services/ops/smoke_test_test.go`
- [x] T021 [P] [US2] Add command dry-run human and JSON output tests in `cmd/ops_execute_smoke_test_test.go`
- [x] T022 [P] [US2] Add dry-run report-file behavior tests in `cmd/ops_execute_smoke_test_test.go`

### Implementation for User Story 2

- [x] T023 [US2] Implement smoke-test planning and dry-run status handling in `internal/services/ops/smoke_test.go`
- [x] T024 [US2] Reuse config test-connection effective behavior or add the required lower-level primitive in the owning package before calling it from `internal/services/ops/smoke_test.go`
- [x] T025 [US2] Map dry-run request/result through `c8volt/ops/client.go` and `c8volt/ops/convert.go`
- [x] T026 [US2] Render compact dry-run human output and complete JSON result data in `cmd/cmd_views_ops_execute_smoke_test.go`
- [x] T027 [US2] Mark US2 tasks complete and record validation notes in `specs/188-ops-smoke-test/progress.md`

**Checkpoint**: Dry-run smoke-test planning is operator-usable and mutates nothing.

---

## Phase 5: User Story 3 - Select And Deploy Version-Matched Fixture (Priority: P3)

**Goal**: The workflow selects the matching embedded multiple-subprocess fixture and deploys it through existing resource behavior.

**Independent Test**: Run fixture selection, missing fixture, deployment call, tenant propagation, and report-field tests against fakes.

### Tests for User Story 3

- [x] T028 [P] [US3] Add fixture selection tests for Camunda 8.7, 8.8, and 8.9 in `internal/services/ops/smoke_test_test.go`
- [x] T029 [P] [US3] Add missing fixture failure-before-mutation tests in `internal/services/ops/smoke_test_test.go`
- [x] T030 [P] [US3] Add deployment result mapping tests in `c8volt/ops/client_test.go`
- [x] T031 [P] [US3] Add command deployment output tests in `cmd/ops_execute_smoke_test_test.go`

### Implementation for User Story 3

- [x] T032 [US3] Implement version-matched embedded smoke-test fixture selection in `internal/services/ops/smoke_test.go`
- [x] T033 [US3] Expose or reuse embedded fixture deployment behavior through the owning resource facade/service in `c8volt/resource` or `internal/services/resource`
- [x] T034 [US3] Deploy the selected fixture through lower-level resource behavior from `internal/services/ops/smoke_test.go`
- [x] T035 [US3] Preserve fixture file, BPMN process ID, deployed process-definition key, deployed version, tenant id, and deployment status in `internal/domain/ops_smoke_test.go`
- [x] T036 [US3] Render deployment details in `cmd/cmd_views_ops_execute_smoke_test.go`
- [x] T037 [US3] Mark US3 tasks complete and record validation notes in `specs/188-ops-smoke-test/progress.md`

**Checkpoint**: The smoke-test workflow can select and deploy the correct fixture independently.

---

## Phase 6: User Story 4 - Start And Walk Created Instances (Priority: P4)

**Goal**: The workflow starts one or more process instances from the deployed definition and walks each created process-instance family.

**Independent Test**: Run count, shorthand, worker, fail-fast, deployed-key preference, and traversal tests against fake process services.

### Tests for User Story 4

- [x] T038 [P] [US4] Add process-instance creation count and `-n` shorthand command tests in `cmd/ops_execute_smoke_test_test.go`
- [x] T039 [P] [US4] Add service tests for deployed process-definition key preference in `internal/services/ops/smoke_test_test.go`
- [x] T040 [P] [US4] Add worker, fail-fast, and no-worker-limit mapping tests in `cmd/ops_execute_smoke_test_test.go`
- [x] T041 [P] [US4] Add traversal summary tests in `internal/services/ops/smoke_test_test.go`

### Implementation for User Story 4

- [x] T042 [US4] Reuse or add the owning process-instance primitive for running by deployed process-definition key in `internal/services/processinstance/api.go`
- [x] T043 [US4] Start the requested count through existing process-instance creation behavior from `internal/services/ops/smoke_test.go`
- [x] T044 [US4] Walk each created process-instance family through existing traversal behavior from `internal/services/ops/smoke_test.go`
- [x] T045 [US4] Preserve requested count, created count, created keys, per-instance run status, walk status, and traversal summaries in `internal/domain/ops_smoke_test.go`
- [x] T046 [US4] Render created keys, run status, walk status, and traversal summaries in `cmd/cmd_views_ops_execute_smoke_test.go`
- [x] T047 [US4] Mark US4 tasks complete and record validation notes in `specs/188-ops-smoke-test/progress.md`

**Checkpoint**: The core deployment-run-walk proof works independently.

---

## Phase 7: User Story 5 - Cleanup Created Resources Safely (Priority: P5)

**Goal**: Default cleanup deletes created process instances through existing delete behavior and deletes the deployed process definition only when no unrelated instances exist.

**Independent Test**: Run cleanup planning, confirmation, no-wait, unrelated-instance blocker, and process-definition cleanup tests against fakes.

### Tests for User Story 5

- [ ] T048 [P] [US5] Add process-instance cleanup tests proving existing delete planning/deletion is reused in `internal/services/ops/smoke_test_test.go`
- [ ] T049 [P] [US5] Add process-definition cleanup eligibility tests for unrelated-instance blockers in `internal/services/ops/smoke_test_test.go`
- [ ] T050 [P] [US5] Add command confirmation and no-wait cleanup tests in `cmd/ops_execute_smoke_test_test.go`
- [ ] T051 [P] [US5] Add local-precondition failure subprocess tests for unsafe cleanup blockers and exit code in `cmd/ops_execute_smoke_test_test.go`

### Implementation for User Story 5

- [ ] T052 [US5] Reuse existing process-instance delete planning/deletion behavior for created keys in `internal/services/ops/smoke_test.go`
- [ ] T053 [US5] Add or reuse the owning process-definition cleanup eligibility primitive in `internal/services/processdefinition` or `internal/services/processinstance`
- [ ] T054 [US5] Delete the deployed process definition only when cleanup eligibility confirms no unrelated instances in `internal/services/ops/smoke_test.go`
- [ ] T055 [US5] Use `shouldImplicitlyConfirm(cmd)` for destructive confirmation decisions in `cmd/ops_execute_smoke_test.go`
- [ ] T056 [US5] Preserve cleanup eligibility, cleanup statuses, no-wait, confirmation, blockers, errors, and outcome in `internal/domain/ops_smoke_test.go`
- [ ] T057 [US5] Render cleanup submitted, confirmed, blocked, skipped, failed, and passed states in `cmd/cmd_views_ops_execute_smoke_test.go`
- [ ] T058 [US5] Mark US5 tasks complete and record validation notes in `specs/188-ops-smoke-test/progress.md`

**Checkpoint**: Default cleanup is independently functional and safety-checked.

---

## Phase 8: User Story 6 - Support No-Cleanup Retention Mode (Priority: P6)

**Goal**: `--no-cleanup` leaves created process instances and deployed definition in place and clearly reports retained resources.

**Independent Test**: Run no-cleanup command/service tests verifying no delete calls, retained key output, JSON/report completeness, and automation behavior.

### Tests for User Story 6

- [ ] T059 [P] [US6] Add no-cleanup service tests proving process-instance and process-definition delete calls are skipped in `internal/services/ops/smoke_test_test.go`
- [ ] T060 [P] [US6] Add no-cleanup human and JSON output tests in `cmd/ops_execute_smoke_test_test.go`
- [ ] T061 [P] [US6] Add `--automation --no-cleanup` without destructive confirmation tests in `cmd/ops_execute_smoke_test_test.go`

### Implementation for User Story 6

- [ ] T062 [US6] Implement no-cleanup branching and retained resource result fields in `internal/services/ops/smoke_test.go`
- [ ] T063 [US6] Preserve retained created keys and deployed definition metadata in `internal/domain/ops_smoke_test.go`
- [ ] T064 [US6] Render no-cleanup retained resource output in `cmd/cmd_views_ops_execute_smoke_test.go`
- [ ] T065 [US6] Mark US6 tasks complete and record validation notes in `specs/188-ops-smoke-test/progress.md`

**Checkpoint**: No-cleanup mode is independently functional and automation-compatible.

---

## Phase 9: User Story 7 - Produce Audit Reports And Stable Output (Priority: P7)

**Goal**: Human output, deterministic JSON, and Markdown/JSON audit reports render from one stable structured smoke-test report model.

**Independent Test**: Request human, verbose, JSON, Markdown report, JSON report, failure-after-planning, existing-file safety, and report-format inference cases.

### Tests for User Story 7

- [ ] T066 [P] [US7] Add report format inference and validation tests for smoke-test in `cmd/ops_contract_test.go`
- [ ] T067 [P] [US7] Add Markdown smoke-test report rendering test in `cmd/ops_execute_smoke_test_test.go`
- [ ] T068 [P] [US7] Add JSON smoke-test report rendering test in `cmd/ops_execute_smoke_test_test.go`
- [ ] T069 [P] [US7] Add existing report-file preservation tests for dry-run, unconfirmed, and locally blocked runs in `cmd/ops_execute_smoke_test_test.go`
- [ ] T070 [P] [US7] Add `--automation --json` deterministic stdout test in `cmd/ops_execute_smoke_test_test.go`

### Implementation for User Story 7

- [ ] T071 [US7] Reuse shared ops report-file validation, format inference, overwrite safety, and file writing in `cmd/ops_execute_smoke_test.go`
- [ ] T072 [US7] Implement smoke-test Markdown and JSON report renderers in `cmd/cmd_views_ops_execute_smoke_test.go`
- [ ] T073 [US7] Ensure reports render from the stable structured smoke-test model in `internal/domain/ops_smoke_test.go`
- [ ] T074 [US7] Keep normal human output compact while preserving complete JSON/report data in `cmd/cmd_views_ops_execute_smoke_test.go`
- [ ] T075 [US7] Print compact `report: written <path>` human output after report writes in `cmd/cmd_views_ops_execute_smoke_test.go`
- [ ] T076 [US7] Mark US7 tasks complete and record validation notes in `specs/188-ops-smoke-test/progress.md`

**Checkpoint**: Output and reports are independently functional for dry-run, success, cleanup-skipped, and failure paths.

---

## Phase 10: User Story 8 - Preserve Documentation And Existing Behavior (Priority: P8)

**Goal**: Existing lower-level commands remain unchanged while smoke-test docs and examples are generated from source metadata.

**Independent Test**: Run regression tests and documentation generation checks.

### Tests for User Story 8

- [ ] T077 [P] [US8] Add regression tests for unchanged `config test-connection` behavior in `cmd/config_test.go`
- [ ] T078 [P] [US8] Add regression tests for unchanged `embed deploy` fixture deployment behavior in `cmd/embed_test.go`
- [ ] T079 [P] [US8] Add regression tests for unchanged `run pi`, `walk pi`, `delete pi`, and `delete pd` behavior in `cmd/run_test.go`, `cmd/walk_test.go`, and `cmd/delete_test.go`
- [ ] T080 [P] [US8] Add docs/contract assertions for smoke-test command metadata in `cmd/command_contract_test.go`

### Implementation for User Story 8

- [ ] T081 [US8] Update user-facing help examples for smoke-test in `cmd/ops_execute_smoke_test.go`
- [ ] T082 [US8] Run `make docs-content` and review generated files under `docs/cli/` and `docs/index.md`
- [ ] T083 [US8] Run targeted command tests with `go test ./cmd -run 'TestOpsExecuteSmokeTest|TestCommandCapability|TestConfigTestConnection|TestDeploy|TestRun|TestWalk|TestDelete' -count=1`
- [ ] T084 [US8] Run facade and service tests with `go test ./c8volt/ops ./c8volt/process ./c8volt/resource ./internal/services/ops ./internal/services/processinstance ./internal/services/processdefinition ./internal/services/resource -count=1`
- [ ] T085 [US8] Run repository validation with `make test`
- [ ] T086 [US8] Mark US8 tasks complete and record final validation notes in `specs/188-ops-smoke-test/progress.md`

**Checkpoint**: Feature is documented, regression-protected, and ready for review.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup completion and blocks all user stories.
- **User Story 1 (P1)**: Depends on Foundational phase.
- **User Story 2 (P2)**: Depends on US1 because dry-run is invoked through the command/facade surface.
- **User Story 3 (P3)**: Depends on US2 because deployment uses the validated plan and fixture metadata.
- **User Story 4 (P4)**: Depends on US3 because instance creation should use the deployed definition.
- **User Story 5 (P5)**: Depends on US4 because cleanup consumes created process-instance keys and deployed definition metadata.
- **User Story 6 (P6)**: Depends on US4 and can be implemented before or after US5 if it cleanly skips cleanup paths.
- **User Story 7 (P7)**: Depends on US2 through US6 because reports need planned, deployed, run/walk, cleanup, and cleanup-skipped result shapes.
- **User Story 8 (P8)**: Depends on all feature behavior that affects docs and regression contracts.

### Parallel Opportunities

- T003, T004, T005, T009, and T010 can be prepared in parallel after Setup.
- Tests within each story marked `[P]` can be drafted in parallel because they protect different layers or assertions.
- Fixture-selection tests in US3 can run in parallel with deployment mapping tests after foundational models exist.
- Report rendering tests in US7 can run in parallel with report-file safety tests once the smoke-test report model exists.
- US8 regression tests can be drafted once related existing command behavior is inspected, but final validation waits for all story behavior.

## Parallel Example: User Story 4

```text
Task: "Add process-instance creation count and -n shorthand command tests in cmd/ops_execute_smoke_test_test.go"
Task: "Add service tests for deployed process-definition key preference in internal/services/ops/smoke_test_test.go"
Task: "Add traversal summary tests in internal/services/ops/smoke_test_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Phase 2 foundational models and service/facade seam.
2. Complete User Story 1 command registration.
3. Complete User Story 2 dry-run planning.
4. Stop and validate that smoke-test planning is visible and mutation-free.
5. Commit with a Conventional Commit subject ending in `#188`.

### Incremental Delivery

1. Add command shape and metadata.
2. Add dry-run planning and fixture selection.
3. Add deployment.
4. Add process-instance creation and traversal.
5. Add default cleanup and no-cleanup.
6. Add report/output completeness and automation edge cases.
7. Add docs, generated docs, and regression validation.

### Ralph Discipline

- Each Ralph iteration must read `specs/ralph-implementation-rules.md`, `spec.md`, `plan.md`, `tasks.md`, and `progress.md`.
- Each iteration should complete only the first incomplete work unit.
- Mark tasks complete only after implementation and relevant validation pass.
- Update `progress.md` with reusable discoveries and validation results in the same work-unit commit.
- Commit subjects must use Conventional Commits and end with `#188`.
