# Tasks: Ops Purge Orphan Process Instances

**Input**: Design documents from `specs/186-ops-orphan-cleanup/`  
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)  
**Mandatory Ralph Context**: Every Ralph iteration MUST be launched with `--implementation-context specs/ralph-implementation-rules.md` and must apply that file before implementation.  
**Issue Commit Rule**: Every commit subject for this feature MUST use Conventional Commits and end with `#186`.

**Tests**: Tests are required by the feature specification, repository constitution, and Ralph implementation rules.

**Organization**: Tasks are grouped by independently testable user story. Each Ralph iteration should complete only the current work unit and update [progress.md](progress.md).

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Capture required implementation context and codebase patterns before code changes.

- [x] T001 Record mandatory Ralph context and issue traceability in `specs/186-ops-orphan-cleanup/progress.md`
- [x] T002 Inspect existing ops foundation from issue #197, process-instance search, process-instance delete, command contract, and docs generation patterns; record reusable discoveries in `specs/186-ops-orphan-cleanup/progress.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add the workflow model and facade/service wiring needed by all stories.

**CRITICAL**: No user story work can begin until this phase is complete.

- [x] T003 [P] Define internal orphan-purge request/result domain models in `internal/domain/ops_orphan_purge.go`
- [x] T004 [P] Define public ops facade request/result models in `c8volt/ops/model.go`
- [x] T005 [P] Define public ops facade API in `c8volt/ops/api.go`
- [x] T006 Define internal ops service interface and constructor in `internal/services/ops/api.go`
- [x] T007 Implement public ops facade conversions in `c8volt/ops/convert.go`
- [x] T008 Implement thin public ops facade client in `c8volt/ops/client.go`
- [x] T009 Wire ops facade creation and API embedding in `c8volt/client.go` and `c8volt/contract.go`
- [x] T010 [P] Add foundational ops facade wiring tests in `c8volt/ops/client_test.go`
- [x] T011 [P] Add foundational internal ops service tests in `internal/services/ops/orphan_purge_test.go`

**Checkpoint**: Ops workflow model, facade, and service boundary are available for story implementation.

---

## Phase 3: User Story 1 - Preview Orphan Cleanup Safely (Priority: P1) MVP

**Goal**: `c8volt ops purge orphan-process-instances --dry-run` discovers orphan child process-instance keys, validates the delete plan, reports the plan, and mutates nothing.

**Independent Test**: Run command tests against fake process-instance behavior and verify discovered keys are reported while no delete request is sent.

### Tests for User Story 1

- [x] T012 [P] [US1] Add dry-run command tests for discovered orphan keys and no delete request in `cmd/ops_purge_orphan_processinstances_test.go`
- [x] T013 [P] [US1] Add no-target dry-run command test in `cmd/ops_purge_orphan_processinstances_test.go`
- [x] T014 [P] [US1] Add compatible filter narrowing test in `cmd/ops_purge_orphan_processinstances_test.go`
- [x] T015 [P] [US1] Add orphan discovery and delete-plan service tests in `internal/services/ops/orphan_purge_test.go`

### Implementation for User Story 1

- [x] T016 [US1] Add process-instance orphan discovery primitive or reuse wrapper in `internal/services/processinstance/orphan_discovery.go`
- [x] T017 [US1] Implement dry-run discovery and plan orchestration in `internal/services/ops/orphan_purge.go`
- [x] T018 [US1] Map dry-run ops facade inputs and outputs in `c8volt/ops/client.go`
- [x] T019 [US1] Add `ops purge` grouping command in `cmd/ops_purge.go` and `ops purge orphan-process-instances` Cobra command, dry-run flag handling, and compatible selection flags in `cmd/ops_purge_orphan_processinstances.go`
- [x] T020 [US1] Add human and JSON rendering for dry-run purge results in `cmd/cmd_views_ops_purge_orphan_processinstances.go`
- [x] T021 [US1] Mark US1 tasks complete and record validation notes in `specs/186-ops-orphan-cleanup/progress.md`

**Checkpoint**: User Story 1 is independently functional and testable.

---

## Phase 4: User Story 2 - Run Confirmed Orphan Purge (Priority: P2)

**Goal**: `c8volt ops purge orphan-process-instances --auto-confirm` deletes exactly the keys discovered at command start through existing process-instance deletion behavior.

**Independent Test**: Run command/service tests that mutate fake process instances and verify only the initial discovered keys are deleted.

### Tests for User Story 2

- [x] T022 [P] [US2] Add confirmed cleanup command test for exact discovered-key deletion in `cmd/ops_purge_orphan_processinstances_test.go`
- [x] T023 [P] [US2] Add immutable discovered-set service test in `internal/services/ops/orphan_purge_test.go`
- [x] T024 [P] [US2] Add no-target confirmed cleanup test in `cmd/ops_purge_orphan_processinstances_test.go`
- [x] T025 [P] [US2] Add process-instance delete delegation test in `c8volt/ops/client_test.go`

### Implementation for User Story 2

- [x] T026 [US2] Implement confirmed delete orchestration through existing process-instance delete behavior in `internal/services/ops/orphan_purge.go`
- [x] T027 [US2] Reuse destructive confirmation and delete-plan validation from process-instance command helpers in `cmd/ops_purge_orphan_processinstances.go`
- [x] T028 [US2] Render deletion execution and final outcome in `cmd/cmd_views_ops_purge_orphan_processinstances.go`
- [x] T029 [US2] Mark US2 tasks complete and record validation notes in `specs/186-ops-orphan-cleanup/progress.md`

**Checkpoint**: User Stories 1 and 2 both work independently.

---

## Phase 5: User Story 3 - Run Cleanup In Automation (Priority: P3)

**Goal**: Automation mode is non-interactive, requires explicit auto-confirm for mutation, and produces deterministic JSON stdout.

**Independent Test**: Execute automation command tests and verify stdout, exit behavior, and contract metadata.

### Tests for User Story 3

- [ ] T030 [P] [US3] Add automation-without-auto-confirm pre-mutation failure test in `cmd/ops_purge_orphan_processinstances_test.go`
- [ ] T031 [P] [US3] Add `--automation --json --auto-confirm` deterministic stdout test in `cmd/ops_purge_orphan_processinstances_test.go`
- [ ] T032 [P] [US3] Add command contract metadata test for state-changing and automation support in `cmd/command_contract_test.go`

### Implementation for User Story 3

- [ ] T033 [US3] Add automation validation and pre-mutation guard in `cmd/ops_purge_orphan_processinstances.go`
- [ ] T034 [US3] Set mutation, contract, output-mode, and automation metadata in `cmd/ops_purge_orphan_processinstances.go`
- [ ] T035 [US3] Ensure JSON rendering uses existing shared result envelope in `cmd/cmd_views_ops_purge_orphan_processinstances.go`
- [ ] T036 [US3] Mark US3 tasks complete and record validation notes in `specs/186-ops-orphan-cleanup/progress.md`

**Checkpoint**: Automation behavior is independently testable and script-safe.

---

## Phase 6: User Story 4 - Produce Audit Reports (Priority: P4)

**Goal**: `--report-file` and `--report-format` write Markdown or JSON audit reports from one stable structured cleanup model.

**Independent Test**: Request Markdown and JSON reports and verify content for dry-run, success, and post-discovery failure.

### Tests for User Story 4

- [ ] T037 [P] [US4] Add report format inference and validation tests in `cmd/ops_contract_test.go`
- [ ] T038 [P] [US4] Add Markdown report rendering test in `cmd/ops_purge_orphan_processinstances_test.go`
- [ ] T039 [P] [US4] Add JSON report rendering test in `cmd/ops_purge_orphan_processinstances_test.go`
- [ ] T040 [P] [US4] Add post-discovery failure report-write test in `cmd/ops_purge_orphan_processinstances_test.go`

### Implementation for User Story 4

- [ ] T041 [US4] Extend stable ops audit report model in `cmd/ops_contract.go`
- [ ] T042 [US4] Implement Markdown and JSON report renderers in `cmd/cmd_views_ops_purge_orphan_processinstances.go`
- [ ] T043 [US4] Implement `--report-file` and `--report-format` command flags and write path in `cmd/ops_purge_orphan_processinstances.go`
- [ ] T044 [US4] Mark US4 tasks complete and record validation notes in `specs/186-ops-orphan-cleanup/progress.md`

**Checkpoint**: Audit reports are independently functional for dry-run, success, and failure paths.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, validation, cleanup, and final feature readiness.

- [ ] T045 [P] Update command examples and user-facing help text in `cmd/ops_purge.go` and `cmd/ops_purge_orphan_processinstances.go`
- [ ] T046 Run `make docs-content` and review generated files under `docs/cli/` and `docs/index.md`
- [ ] T047 Run targeted command tests for `cmd/` with `go test ./cmd -run 'TestOps|TestCommandContract' -count=1`
- [ ] T048 Run facade and service tests for `c8volt/ops`, `internal/services/ops`, and `internal/services/processinstance`
- [ ] T049 Run repository validation through `Makefile` with `make test`
- [ ] T050 Update final implementation notes and completed validation in `specs/186-ops-orphan-cleanup/progress.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup completion and blocks all user stories.
- **User Story 1 (P1)**: Depends on Foundational phase.
- **User Story 2 (P2)**: Depends on US1 because confirmed cleanup uses the discovered dry-run plan.
- **User Story 3 (P3)**: Depends on US2 because automation guards destructive execution.
- **User Story 4 (P4)**: Depends on US1 and can be implemented after the structured cleanup result exists; failure/success reporting is easier after US2.
- **Polish**: Depends on all desired user stories.

### Parallel Opportunities

- T003, T004, T005, T010, and T011 can be prepared in parallel after Setup.
- Tests within each story marked `[P]` can be drafted in parallel because they touch different assertions or layers.
- Report rendering tests in US4 can run in parallel with command flag tests once the audit model exists.

## Parallel Example: User Story 1

```text
Task: "Add dry-run command tests for discovered orphan keys and no delete request in cmd/ops_purge_orphan_processinstances_test.go"
Task: "Add orphan discovery and delete-plan service tests in internal/services/ops/orphan_purge_test.go"
Task: "Add compatible filter narrowing test in cmd/ops_purge_orphan_processinstances_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational phases.
2. Complete User Story 1.
3. Validate dry-run discovery and planning independently.
4. Commit with a Conventional Commit subject ending in `#186`.

### Incremental Delivery

1. Add dry-run planning.
2. Add confirmed deletion.
3. Add automation-safe JSON behavior.
4. Add audit reports.
5. Refresh generated docs and run final validation.

### Ralph Iteration Discipline

1. Start each iteration by reading `specs/ralph-implementation-rules.md`, `specs/186-ops-orphan-cleanup/tasks.md`, `specs/186-ops-orphan-cleanup/plan.md`, `specs/186-ops-orphan-cleanup/spec.md`, and `specs/186-ops-orphan-cleanup/progress.md`.
2. Implement only the first incomplete work unit.
3. Inspect nearby code and tests before adding code.
4. Mark tasks complete only after relevant validation passes.
5. Include task/progress updates in the same work-unit commit.
6. Use Conventional Commits and append `#186` as the final subject token.
