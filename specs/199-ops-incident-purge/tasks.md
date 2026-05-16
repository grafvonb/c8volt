# Tasks: Ops Purge Process Instances With Incidents

**Input**: Design documents from `specs/199-ops-incident-purge/`
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)
**Mandatory Ralph Context**: Every Ralph iteration MUST be launched with `--implementation-context specs/ralph-implementation-rules.md` and must apply that file before implementation.
**Issue Commit Rule**: Every commit subject for this feature MUST use Conventional Commits and end with `#199`.

**Tests**: Tests are required by the feature specification, repository constitution, and Ralph implementation rules.

**Organization**: Tasks are grouped by small, independently testable user stories. Each Ralph iteration should complete only the current work unit and update [progress.md](progress.md).

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Capture required implementation context and current #186/#187 ops patterns before code changes.

- [ ] T001 Record mandatory Ralph context and issue traceability in `specs/199-ops-incident-purge/progress.md`
- [ ] T002 Inspect existing #186 ops purge, #187 retention delete-plan flow, incident search, process-instance delete planning, command contract metadata, and docs generation patterns; record reusable discoveries in `specs/199-ops-incident-purge/progress.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared incident purge workflow models and service/facade seams needed by all stories.

**CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T003 [P] Define internal incident purge request/result domain models in `internal/domain/ops_incident_purge.go`
- [ ] T004 [P] Define public ops incident purge request/result models in `c8volt/ops/model.go`
- [ ] T005 [P] Extend public ops facade API for incident purge in `c8volt/ops/api.go`
- [ ] T006 Extend internal ops service interface for incident purge in `internal/services/ops/api.go`
- [ ] T007 Implement public/internal incident purge model conversions in `c8volt/ops/convert.go`
- [ ] T008 Implement thin public ops facade incident purge method in `c8volt/ops/client.go`
- [ ] T009 [P] Add foundational ops facade wiring tests for incident purge in `c8volt/ops/client_test.go`
- [ ] T010 [P] Add foundational internal ops service validation tests for incident purge in `internal/services/ops/incident_purge_test.go`
- [ ] T011 Mark Phase 2 tasks complete and record validation notes in `specs/199-ops-incident-purge/progress.md`

**Checkpoint**: Incident purge workflow model, facade, and service boundary are available for story implementation.

---

## Phase 3: User Story 1 - Register Incident Purge Command (Priority: P1) MVP

**Goal**: `c8volt ops purge process-instances-with-incidents` exists under the purge group, alias `pi-with-incidents` works, local command shape is validated, and no remote cleanup occurs.

**Independent Test**: Run command help, alias, invalid flag, and command contract tests without Camunda data.

### Tests for User Story 1

- [ ] T012 [P] [US1] Add command registration, help, and alias tests for incident purge in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [ ] T013 [P] [US1] Add unsupported display-only incident flag tests in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [ ] T014 [P] [US1] Add command contract metadata tests for state-changing and automation support in `cmd/command_contract_test.go`

### Implementation for User Story 1

- [ ] T015 [US1] Add `ops purge process-instances-with-incidents` Cobra command, alias, summary, and safe examples in `cmd/ops_purge_processinstances_with_incidents.go`
- [ ] T016 [US1] Register supported incident selection flags and delete workflow flags in `cmd/ops_purge_processinstances_with_incidents.go`
- [ ] T017 [US1] Map static flag validation failures through existing invalid-input helpers in `cmd/ops_purge_processinstances_with_incidents.go`
- [ ] T018 [US1] Set mutation, output-mode, required flag, and automation metadata in `cmd/ops_purge_processinstances_with_incidents.go`
- [ ] T019 [US1] Mark US1 tasks complete and record validation notes in `specs/199-ops-incident-purge/progress.md`

**Checkpoint**: User Story 1 is independently functional and testable.

---

## Phase 4: User Story 2 - Discover And Freeze Candidate Process Instances (Priority: P2)

**Goal**: Incident purge dry-run discovers matching incidents, extracts and freezes unique candidate process-instance keys, handles duplicates and no-target cases, and submits no delete request.

**Independent Test**: Run dry-run discovery tests against fake incident responses and verify candidate incidents, unique candidate process-instance keys, duplicates, skipped incidents, and `--limit` behavior.

### Tests for User Story 2

- [ ] T020 [P] [US2] Add incident discovery service or ops service tests for candidate extraction, duplicate detection, skipped incidents, and limit handling in `internal/services/ops/incident_purge_test.go`
- [ ] T021 [P] [US2] Add command dry-run discovery output tests in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [ ] T022 [P] [US2] Add facade conversion tests for incident discovery result fields in `c8volt/ops/client_test.go`

### Implementation for User Story 2

- [ ] T023 [US2] Reuse existing incident search semantics to discover candidate incidents in `internal/services/ops/incident_purge.go`
- [ ] T024 [US2] Extract, deduplicate, and freeze candidate process-instance keys in `internal/services/ops/incident_purge.go`
- [ ] T025 [US2] Preserve candidate incidents, duplicate candidate process instances, skipped incidents, notices, and errors in `internal/domain/ops_incident_purge.go`
- [ ] T026 [US2] Map discovery request/result through `c8volt/ops/client.go` and `c8volt/ops/convert.go`
- [ ] T027 [US2] Render compact discovery output and complete JSON discovery data in `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`
- [ ] T028 [US2] Mark US2 tasks complete and record validation notes in `specs/199-ops-incident-purge/progress.md`

**Checkpoint**: Incident discovery works independently with dry-run and mutates nothing.

---

## Phase 5: User Story 3 - Build Incident Purge Delete Plan (Priority: P3)

**Goal**: Frozen candidate process-instance keys are expanded through existing delete planning, with roots, affected scope, duplicates, expected notices, and non-final blockers reported.

**Independent Test**: Feed frozen candidate keys into planning tests and verify existing delete planning behavior is reused without delete submission.

### Tests for User Story 3

- [ ] T029 [P] [US3] Add ops service delete-plan tests for candidate keys, resolved roots, affected keys, and duplicate handling in `internal/services/ops/incident_purge_test.go`
- [ ] T030 [P] [US3] Add non-final affected instance blocking test in `internal/services/ops/incident_purge_test.go`
- [ ] T031 [P] [US3] Add command dry-run plan rendering tests in `cmd/ops_purge_processinstances_with_incidents_test.go`

### Implementation for User Story 3

- [ ] T032 [US3] Reuse existing process-instance delete planning from frozen candidate keys in `internal/services/ops/incident_purge.go`
- [ ] T033 [US3] Preserve missing ancestor, traversal warning, duplicate, final-state, non-final, and semantic notice details in `internal/domain/ops_incident_purge.go`
- [ ] T034 [US3] Map delete-plan details through `c8volt/ops/convert.go`
- [ ] T035 [US3] Render compact delete-plan human output and complete JSON output in `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`
- [ ] T036 [US3] Mark US3 tasks complete and record validation notes in `specs/199-ops-incident-purge/progress.md`

**Checkpoint**: Incident purge dry-run includes the validated delete plan and safety blockers.

---

## Phase 6: User Story 4 - Execute Confirmed Purge Through Delete Plan (Priority: P4)

**Goal**: Confirmed incident purge deletes through existing process-instance deletion behavior with established worker, wait, force, fail-fast, no-worker-limit, and automation confirmation semantics.

**Independent Test**: Run command/service tests that submit deletion through fakes and verify only resolved roots from the frozen plan are submitted.

### Tests for User Story 4

- [ ] T037 [P] [US4] Add confirmed deletion command test for exact frozen-plan root submission in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [ ] T038 [P] [US4] Add execution-control mapping tests for workers, fail-fast, no-worker-limit, no-wait, and force in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [ ] T039 [P] [US4] Add `--automation --json` without `--auto-confirm` success test for supported state-changing incident purge in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [ ] T040 [P] [US4] Add local-precondition failure subprocess tests for post-planning blockers and exit code in `cmd/ops_purge_processinstances_with_incidents_test.go`

### Implementation for User Story 4

- [ ] T041 [US4] Execute deletion through existing process-instance deletion service from `internal/services/ops/incident_purge.go`
- [ ] T042 [US4] Use `shouldImplicitlyConfirm(cmd)` for destructive confirmation decisions in `cmd/ops_purge_processinstances_with_incidents.go`
- [ ] T043 [US4] Preserve no-wait, confirmation, per-key or per-batch status, and final outcome in `internal/domain/ops_incident_purge.go`
- [ ] T044 [US4] Render deletion execution and final outcome in `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`
- [ ] T045 [US4] Mark US4 tasks complete and record validation notes in `specs/199-ops-incident-purge/progress.md`

**Checkpoint**: Confirmed incident purge deletion is independently functional and automation-compatible.

---

## Phase 7: User Story 5 - Produce Compact Output, Complete Reports, And Automation-Safe JSON (Priority: P5)

**Goal**: Human output follows the ops rhythm, JSON remains complete and deterministic, and report files preserve #186/#187 overwrite and rendering behavior.

**Independent Test**: Request human, verbose, JSON, dry-run, success, no-target, existing-file, and post-discovery failure outputs/reports.

### Tests for User Story 5

- [ ] T046 [P] [US5] Add verbose key-list output tests for incident, candidate, root, affected, and skipped keys in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [ ] T047 [P] [US5] Add deterministic `--dry-run --json` and `--automation --json` output tests in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [ ] T048 [P] [US5] Add Markdown incident purge report rendering test in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [ ] T049 [P] [US5] Add JSON incident purge report rendering test in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [ ] T050 [P] [US5] Add existing report-file preservation tests for dry-run, unconfirmed, and locally blocked runs in `cmd/ops_purge_processinstances_with_incidents_test.go`

### Implementation for User Story 5

- [ ] T051 [US5] Reuse shared ops report-file validation, format inference, overwrite safety, and file writing in `cmd/ops_purge_processinstances_with_incidents.go`
- [ ] T052 [US5] Extend report model/rendering for incident discovery, candidate set, plan, deletion, notices, errors, and outcome fields in `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`
- [ ] T053 [US5] Keep normal human output compact and gate detailed key lists behind verbose output in `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`
- [ ] T054 [US5] Print compact `report: written <path>` human output after report writes in `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`
- [ ] T055 [US5] Mark US5 tasks complete and record validation notes in `specs/199-ops-incident-purge/progress.md`

**Checkpoint**: Output and reports are independently functional for dry-run, success, and failure paths.

---

## Phase 8: User Story 6 - Preserve Documentation And Regression Contracts (Priority: P6)

**Goal**: Existing `get incident`, `delete pi`, ops, docs, and generated CLI contracts remain intact while incident purge is documented with safe examples.

**Independent Test**: Run regression tests and documentation generation checks.

### Tests for User Story 6

- [ ] T056 [P] [US6] Add regression tests for unchanged `get incident` selection and display-only behavior in `cmd/get_incident_test.go`
- [ ] T057 [P] [US6] Add regression tests for unchanged `delete pi` hierarchy planning and force/no-wait behavior in `cmd/delete_processinstance_test.go`
- [ ] T058 [P] [US6] Add docs/contract assertions for incident purge command metadata in `cmd/command_contract_test.go`

### Implementation for User Story 6

- [ ] T059 [US6] Update user-facing help examples for incident purge in `cmd/ops_purge_processinstances_with_incidents.go`
- [ ] T060 [US6] Run `make docs-content` and review generated files under `docs/cli/` and `docs/index.md`
- [ ] T061 [US6] Run targeted command tests with `go test ./cmd -run 'TestOpsPurge|TestCommandContract|TestDeleteProcessInstance|TestGetIncident' -count=1`
- [ ] T062 [US6] Run facade and service tests with `go test ./c8volt/ops ./c8volt/incident ./c8volt/process ./internal/services/ops ./internal/services/incident ./internal/services/processinstance -count=1`
- [ ] T063 [US6] Run repository validation with `make test`
- [ ] T064 [US6] Mark US6 tasks complete and record final validation notes in `specs/199-ops-incident-purge/progress.md`

**Checkpoint**: Feature is documented, regression-protected, and ready for review.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup completion and blocks all user stories.
- **User Story 1 (P1)**: Depends on Foundational phase.
- **User Story 2 (P2)**: Depends on US1 because discovery is invoked through the command/facade surface.
- **User Story 3 (P3)**: Depends on US2 because planning consumes the frozen candidate set.
- **User Story 4 (P4)**: Depends on US3 because mutation uses the validated plan.
- **User Story 5 (P5)**: Depends on US2, US3, and US4 because reports and output need discovery, plan, and execution result shapes.
- **User Story 6 (P6)**: Depends on all feature behavior that affects docs and regression contracts.

### Parallel Opportunities

- T003, T004, T005, T009, and T010 can be prepared in parallel after Setup.
- Tests within each story marked `[P]` can be drafted in parallel because they protect different layers or assertions.
- Report rendering tests in US5 can run in parallel with report-file safety tests once the incident purge report model exists.
- US6 regression tests can be drafted once the related existing command behavior is inspected, but final validation waits for all story behavior.

## Parallel Example: User Story 2

```text
Task: "Add incident discovery service or ops service tests for candidate extraction, duplicate detection, skipped incidents, and limit handling in internal/services/ops/incident_purge_test.go"
Task: "Add command dry-run discovery output tests in cmd/ops_purge_processinstances_with_incidents_test.go"
Task: "Add facade conversion tests for incident discovery result fields in c8volt/ops/client_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Phase 1 setup.
2. Complete Phase 2 foundational models and service/facade seam.
3. Complete User Story 1 command registration.
4. Complete User Story 2 discovery dry-run.
5. Stop and validate that incident candidates can be previewed without mutation.

### Incremental Delivery

1. Add command shape and metadata.
2. Add incident discovery and frozen candidate process-instance sets.
3. Add delete planning from frozen candidates.
4. Add confirmed execution.
5. Add report/output completeness and automation edge cases.
6. Add docs, generated docs, and regression validation.

### Ralph Discipline

- Each Ralph iteration must read `specs/ralph-implementation-rules.md`, `spec.md`, `plan.md`, `tasks.md`, and `progress.md`.
- Each iteration should complete only the first incomplete work unit.
- Mark tasks complete only after implementation and relevant validation pass.
- Update `progress.md` with reusable discoveries and validation results in the same work-unit commit.
- Commit subjects must use Conventional Commits and end with `#199`.
