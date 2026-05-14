# Tasks: Ops Execute Retention Policy

**Input**: Design documents from `specs/187-ops-retention-policy/`
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)
**Mandatory Ralph Context**: Every Ralph iteration MUST be launched with `--implementation-context specs/ralph-implementation-rules.md` and must apply that file before implementation.
**Issue Commit Rule**: Every commit subject for this feature MUST use Conventional Commits and end with `#187`.

**Tests**: Tests are required by the feature specification, repository constitution, and Ralph implementation rules.

**Organization**: Tasks are grouped by independently testable user story. Each Ralph iteration should complete only the current work unit and update [progress.md](progress.md).

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Capture required implementation context and current #186 ops patterns before code changes.

- [x] T001 Record mandatory Ralph context and issue traceability in `specs/187-ops-retention-policy/progress.md`
- [x] T002 Inspect existing #186 ops purge implementation, `ops execute` command group, process-instance search, process-instance delete planning, command contract metadata, and docs generation patterns; record reusable discoveries in `specs/187-ops-retention-policy/progress.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared retention workflow models and service/facade seams needed by all stories.

**CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T003 [P] Define internal retention request/result domain models in `internal/domain/ops_retention_policy.go`
- [ ] T004 [P] Define public ops retention request/result models in `c8volt/ops/model.go`
- [ ] T005 [P] Extend public ops facade API for retention policy in `c8volt/ops/api.go`
- [ ] T006 Extend internal ops service interface for retention policy in `internal/services/ops/api.go`
- [ ] T007 Implement public/internal retention model conversions in `c8volt/ops/convert.go`
- [ ] T008 Implement thin public ops facade retention method in `c8volt/ops/client.go`
- [ ] T009 [P] Add foundational ops facade wiring tests for retention policy in `c8volt/ops/client_test.go`
- [ ] T010 [P] Add foundational internal ops service tests for retention policy request validation in `internal/services/ops/retention_policy_test.go`

**Checkpoint**: Retention workflow model, facade, and service boundary are available for story implementation.

---

## Phase 3: User Story 1 - Register Retention Policy Command (Priority: P1) MVP

**Goal**: `c8volt ops execute retention-policy` exists under the execute group, validates required retention age locally, and performs no cleanup at the grouping level.

**Independent Test**: Run command help and invalid-input tests without Camunda data.

### Tests for User Story 1

- [ ] T011 [P] [US1] Add command registration and help tests for `ops execute retention-policy` in `cmd/ops_execute_retention_policy_test.go`
- [ ] T012 [P] [US1] Add invalid missing/negative/non-integer `--retention-days` subprocess tests in `cmd/ops_execute_retention_policy_test.go`
- [ ] T013 [P] [US1] Add command contract metadata tests for state-changing and automation support in `cmd/command_contract_test.go`

### Implementation for User Story 1

- [ ] T014 [US1] Add `ops execute retention-policy` Cobra command, summary, examples, and required retention flag in `cmd/ops_execute_retention_policy.go`
- [ ] T015 [US1] Wire retention-policy command into the existing execute group in `cmd/ops_execute.go`
- [ ] T016 [US1] Implement local retention flag validation and invalid-input error mapping in `cmd/ops_execute_retention_policy.go`
- [ ] T017 [US1] Set mutation, output-mode, required-flag, and automation metadata in `cmd/ops_execute_retention_policy.go`
- [ ] T018 [US1] Mark US1 tasks complete and record validation notes in `specs/187-ops-retention-policy/progress.md`

**Checkpoint**: User Story 1 is independently functional and testable.

---

## Phase 4: User Story 2 - Discover Retention Seeds (Priority: P2)

**Goal**: Retention policy dry-run discovers a frozen seed set using existing `--end-date-older-days` semantics and excludes process instances without `endDate`.

**Independent Test**: Run dry-run tests against fakes and verify discovered seed keys and derived boundary without deletion.

### Tests for User Story 2

- [ ] T019 [P] [US2] Add process-instance retention discovery service tests in `internal/services/processinstance/retention_discovery_test.go`
- [ ] T020 [P] [US2] Add ops service dry-run discovery tests for seed freezing and no delete calls in `internal/services/ops/retention_policy_test.go`
- [ ] T021 [P] [US2] Add command dry-run discovery output tests in `cmd/ops_execute_retention_policy_test.go`

### Implementation for User Story 2

- [ ] T022 [US2] Add process-instance retention discovery primitive using existing end-date older-days search semantics in `internal/services/processinstance/retention_discovery.go`
- [ ] T023 [US2] Expose retention discovery through the process-instance service interface in `internal/services/processinstance/api.go`
- [ ] T024 [US2] Implement dry-run discovery orchestration and seed freezing in `internal/services/ops/retention_policy.go`
- [ ] T025 [US2] Map dry-run discovery request and result through `c8volt/ops/client.go` and `c8volt/ops/convert.go`
- [ ] T026 [US2] Render compact human and JSON discovery output in `cmd/cmd_views_ops_execute_retention_policy.go`
- [ ] T027 [US2] Mark US2 tasks complete and record validation notes in `specs/187-ops-retention-policy/progress.md`

**Checkpoint**: Retention discovery works independently with dry-run and mutates nothing.

---

## Phase 5: User Story 3 - Apply Compatible Selection Filters (Priority: P3)

**Goal**: Compatible process-instance selection filters narrow retention discovery while explicit `--key` selection is rejected.

**Independent Test**: Run dry-run command/service tests that combine filters with retention age and verify invalid explicit key selection fails locally.

### Tests for User Story 3

- [ ] T028 [P] [US3] Add selection filter narrowing tests in `cmd/ops_execute_retention_policy_test.go`
- [ ] T029 [P] [US3] Add unsupported explicit `--key` invalid-input subprocess test in `cmd/ops_execute_retention_policy_test.go`
- [ ] T030 [P] [US3] Add service tests for normalized retention filters in `internal/services/ops/retention_policy_test.go`

### Implementation for User Story 3

- [ ] T031 [US3] Add compatible process-instance selection flags to `cmd/ops_execute_retention_policy.go`
- [ ] T032 [US3] Map selection flags into the retention request without allowing explicit keys in `cmd/ops_execute_retention_policy.go`
- [ ] T033 [US3] Apply normalized filters during retention discovery in `internal/services/processinstance/retention_discovery.go`
- [ ] T034 [US3] Include selected filters in human, JSON, and report-ready retention output in `cmd/cmd_views_ops_execute_retention_policy.go`
- [ ] T035 [US3] Mark US3 tasks complete and record validation notes in `specs/187-ops-retention-policy/progress.md`

**Checkpoint**: Retention age plus selection filters is independently testable.

---

## Phase 6: User Story 4 - Build And Validate Delete Plan (Priority: P4)

**Goal**: Discovered retention seed keys are expanded through existing delete planning, with roots, affected scope, duplicates, missing ancestors, and non-final blockers reported.

**Independent Test**: Feed discovered seed keys into planning tests and verify existing delete planning behavior is reused.

### Tests for User Story 4

- [ ] T036 [P] [US4] Add ops service delete-plan tests for child seeds, resolved roots, affected keys, and duplicates in `internal/services/ops/retention_policy_test.go`
- [ ] T037 [P] [US4] Add non-final affected instance blocking test in `internal/services/ops/retention_policy_test.go`
- [ ] T038 [P] [US4] Add command dry-run plan rendering tests in `cmd/ops_execute_retention_policy_test.go`

### Implementation for User Story 4

- [ ] T039 [US4] Reuse existing process-instance delete planning from retention seeds in `internal/services/ops/retention_policy.go`
- [ ] T040 [US4] Preserve missing ancestor, traversal warning, duplicate, final-state, and non-final details in the retention result model in `internal/domain/ops_retention_policy.go`
- [ ] T041 [US4] Map delete-plan details through `c8volt/ops/convert.go`
- [ ] T042 [US4] Render compact delete-plan human output and complete JSON output in `cmd/cmd_views_ops_execute_retention_policy.go`
- [ ] T043 [US4] Mark US4 tasks complete and record validation notes in `specs/187-ops-retention-policy/progress.md`

**Checkpoint**: Retention dry-run includes the validated delete plan and safety blockers.

---

## Phase 7: User Story 5 - Support Dry Run Output (Priority: P5)

**Goal**: `--dry-run` shows planned retention cleanup in human and JSON modes, handles no-target flows successfully, and never prompts or mutates.

**Independent Test**: Run dry-run command tests for matching, no-target, JSON, and report-path preview cases.

### Tests for User Story 5

- [ ] T044 [P] [US5] Add no-target dry-run success test in `cmd/ops_execute_retention_policy_test.go`
- [ ] T045 [P] [US5] Add `--dry-run --json` deterministic output test in `cmd/ops_execute_retention_policy_test.go`
- [ ] T046 [P] [US5] Add dry-run no prompt/no mutation test in `cmd/ops_execute_retention_policy_test.go`

### Implementation for User Story 5

- [ ] T047 [US5] Implement dry-run command orchestration through the ops facade in `cmd/ops_execute_retention_policy.go`
- [ ] T048 [US5] Add planned/skipped/final outcome status handling for dry-run and no-target flows in `internal/services/ops/retention_policy.go`
- [ ] T049 [US5] Keep detailed key lists behind verbose output while preserving complete JSON data in `cmd/cmd_views_ops_execute_retention_policy.go`
- [ ] T050 [US5] Mark US5 tasks complete and record validation notes in `specs/187-ops-retention-policy/progress.md`

**Checkpoint**: Dry-run retention cleanup is operator-usable and script-safe.

---

## Phase 8: User Story 6 - Execute Confirmed Deletion (Priority: P6)

**Goal**: Confirmed retention cleanup deletes through existing process-instance deletion behavior with established worker, wait, state-check, force, fail-fast, and automation confirmation semantics.

**Independent Test**: Run command/service tests that submit deletion through fakes and verify only resolved roots from the frozen plan are submitted.

### Tests for User Story 6

- [ ] T051 [P] [US6] Add confirmed deletion command test for exact frozen-plan root submission in `cmd/ops_execute_retention_policy_test.go`
- [ ] T052 [P] [US6] Add execution-control mapping tests for workers, fail-fast, no-wait, no-state-check, and force in `cmd/ops_execute_retention_policy_test.go`
- [ ] T053 [P] [US6] Add `--automation --json` without `--auto-confirm` success test for supported state-changing retention command in `cmd/ops_execute_retention_policy_test.go`
- [ ] T054 [P] [US6] Add local-precondition failure subprocess tests for post-planning blockers and exit code in `cmd/ops_execute_retention_policy_test.go`

### Implementation for User Story 6

- [ ] T055 [US6] Add compatible delete execution control flags and facade option mapping in `cmd/ops_execute_retention_policy.go`
- [ ] T056 [US6] Execute deletion through existing process-instance deletion service from `internal/services/ops/retention_policy.go`
- [ ] T057 [US6] Use `shouldImplicitlyConfirm(cmd)` for destructive confirmation decisions in `cmd/ops_execute_retention_policy.go`
- [ ] T058 [US6] Preserve no-wait, confirmation, per-key or per-batch status, and final outcome in `internal/domain/ops_retention_policy.go`
- [ ] T059 [US6] Render deletion execution and final outcome in `cmd/cmd_views_ops_execute_retention_policy.go`
- [ ] T060 [US6] Mark US6 tasks complete and record validation notes in `specs/187-ops-retention-policy/progress.md`

**Checkpoint**: Confirmed retention deletion is independently functional and automation-compatible.

---

## Phase 9: User Story 7 - Write Audit Reports (Priority: P7)

**Goal**: `--report-file` and `--report-format` write Markdown or JSON audit reports from one stable structured retention model, including failure paths after discovery.

**Independent Test**: Request Markdown and JSON reports for dry-run, success, no-target, existing-file safety, and post-discovery failure paths.

### Tests for User Story 7

- [ ] T061 [P] [US7] Add report format inference and validation tests for retention in `cmd/ops_contract_test.go`
- [ ] T062 [P] [US7] Add Markdown retention report rendering test in `cmd/ops_execute_retention_policy_test.go`
- [ ] T063 [P] [US7] Add JSON retention report rendering test in `cmd/ops_execute_retention_policy_test.go`
- [ ] T064 [P] [US7] Add existing report-file preservation tests for dry-run, unconfirmed, and locally blocked runs in `cmd/ops_execute_retention_policy_test.go`
- [ ] T065 [P] [US7] Add post-discovery failure report-write test in `cmd/ops_execute_retention_policy_test.go`

### Implementation for User Story 7

- [ ] T066 [US7] Reuse shared ops report-file validation, format inference, overwrite safety, and file writing in `cmd/ops_execute_retention_policy.go`
- [ ] T067 [US7] Extend report model/rendering for retention-specific discovery, plan, deletion, and outcome fields in `cmd/cmd_views_ops_execute_retention_policy.go`
- [ ] T068 [US7] Ensure reports render from the stable structured retention model in `internal/domain/ops_retention_policy.go`
- [ ] T069 [US7] Print compact `report: written <path>` human output after report writes in `cmd/cmd_views_ops_execute_retention_policy.go`
- [ ] T070 [US7] Mark US7 tasks complete and record validation notes in `specs/187-ops-retention-policy/progress.md`

**Checkpoint**: Audit reports are independently functional for dry-run, success, and failure paths.

---

## Phase 10: User Story 8 - Preserve Existing Contracts (Priority: P8)

**Goal**: Existing `get pi`, `delete pi`, ops, docs, and generated CLI contracts remain intact while retention policy is documented.

**Independent Test**: Run regression tests and documentation generation checks.

### Tests for User Story 8

- [ ] T071 [P] [US8] Add regression tests for unchanged `get pi --end-date-older-days` behavior in `cmd/get_processinstance_test.go`
- [ ] T072 [P] [US8] Add regression tests for unchanged `delete pi --keys` hierarchy planning behavior in `cmd/delete_processinstance_test.go`
- [ ] T073 [P] [US8] Add docs/contract assertions for retention command metadata in `cmd/command_contract_test.go`

### Implementation for User Story 8

- [ ] T074 [US8] Update user-facing help examples for retention policy in `cmd/ops_execute_retention_policy.go`
- [ ] T075 [US8] Run `make docs-content` and review generated files under `docs/cli/` and `docs/index.md`
- [ ] T076 [US8] Run targeted command tests with `go test ./cmd -run 'TestOps|TestCommandContract|TestDeleteProcessInstance|TestGetProcessInstance' -count=1`
- [ ] T077 [US8] Run facade and service tests with `go test ./c8volt/ops ./c8volt/process ./internal/services/ops ./internal/services/processinstance -count=1`
- [ ] T078 [US8] Run repository validation with `make test`
- [ ] T079 [US8] Mark US8 tasks complete and record final validation notes in `specs/187-ops-retention-policy/progress.md`

**Checkpoint**: Feature is documented, regression-protected, and ready for review.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup completion and blocks all user stories.
- **User Story 1 (P1)**: Depends on Foundational phase.
- **User Story 2 (P2)**: Depends on US1 because discovery is invoked through the command/facade surface.
- **User Story 3 (P3)**: Depends on US2 because filter support narrows the discovered seed set.
- **User Story 4 (P4)**: Depends on US2 and US3 because planning consumes the final discovered seed set.
- **User Story 5 (P5)**: Depends on US4 because dry-run output includes the delete plan.
- **User Story 6 (P6)**: Depends on US4 and US5 because mutation uses the validated plan and dry-run status model.
- **User Story 7 (P7)**: Depends on US5 and US6 because reports need planned, executed, no-target, and failure result shapes.
- **User Story 8 (P8)**: Depends on all feature behavior that affects docs and regression contracts.

### Parallel Opportunities

- T003, T004, T005, T009, and T010 can be prepared in parallel after Setup.
- Tests within each story marked `[P]` can be drafted in parallel because they protect different layers or assertions.
- Report rendering tests in US7 can run in parallel with report-file safety tests once the retention report model exists.
- US8 regression tests can be drafted once the related existing command behavior is inspected, but final validation waits for all story behavior.

## Parallel Example: User Story 2

```text
Task: "Add process-instance retention discovery service tests in internal/services/processinstance/retention_discovery_test.go"
Task: "Add ops service dry-run discovery tests for seed freezing and no delete calls in internal/services/ops/retention_policy_test.go"
Task: "Add command dry-run discovery output tests in cmd/ops_execute_retention_policy_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational phases.
2. Complete User Story 1.
3. Validate command registration and local retention flag errors independently.
4. Commit with a Conventional Commit subject ending in `#187`.

### Incremental Delivery

1. Add command registration and validation.
2. Add retention discovery and compatible filters.
3. Add delete planning and dry-run output.
4. Add confirmed deletion and automation behavior.
5. Add audit reports.
6. Refresh generated docs and run final validation.

### Ralph Iteration Discipline

1. Start each iteration by reading `specs/ralph-implementation-rules.md`, `specs/187-ops-retention-policy/tasks.md`, `specs/187-ops-retention-policy/plan.md`, `specs/187-ops-retention-policy/spec.md`, and `specs/187-ops-retention-policy/progress.md`.
2. Implement only the first incomplete work unit.
3. Inspect nearby code and tests before adding code.
4. Mark tasks complete only after relevant validation passes.
5. Include task/progress updates in the same work-unit commit.
6. Use Conventional Commits and append `#187` as the final subject token.
