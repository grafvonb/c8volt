# Tasks: Ops Repair Workflows

**Input**: Design documents from `specs/183-ops-repair-workflows/`
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)
**Mandatory Ralph Context**: Every Ralph iteration MUST be launched with `--implementation-context specs/ralph-implementation-rules.md` and must apply that file before implementation.
**Issue Commit Rule**: Every commit subject for this feature MUST use Conventional Commits and end with `#183`.

**Tests**: Tests are required by the feature specification, repository constitution, and Ralph implementation rules.

**Organization**: Tasks are grouped by independently testable user story. Each Ralph iteration should complete only the current work unit, update [progress.md](progress.md), and stop after the relevant validation for that work unit passes.

## Phase 1: Setup (Shared Discovery)

**Purpose**: Confirm current codebase patterns and record reusable discoveries before code changes.

- [x] T001 Inspect existing ops repair grouping behavior in `cmd/ops_repair.go` and record target-specific command constraints in `specs/183-ops-repair-workflows/progress.md`
- [x] T002 Inspect incident command/filter patterns in `cmd/get_incident.go`, `cmd/get_incident_search.go`, `internal/services/incident/api.go`, and `internal/domain/incident.go`
- [x] T003 Inspect process-instance search and variable update patterns in `cmd/get_processinstance*.go`, `cmd/update_processinstance_variables.go`, and `internal/services/processinstance/variables.go`
- [x] T004 Inspect job lookup/update patterns in `cmd/update_job.go`, `c8volt/job`, and `internal/services/job`
- [x] T005 Inspect ops report and automation patterns in `cmd/ops_contract.go`, `cmd/ops_purge_processinstances_with_incidents.go`, and `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`
- [x] T006 Record mandatory Ralph context, issue traceability, and discovered implementation patterns in `specs/183-ops-repair-workflows/progress.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add the shared repair model, service dependency wiring, and report status vocabulary required by every repair target.

**CRITICAL**: No user story work can begin until this phase is complete.

- [x] T007 [P] Add internal repair request/result/domain models and `not_applicable` workflow status in `internal/domain/ops_repair.go`
- [x] T008 [P] Add public repair request/result models and `not_applicable` workflow status mapping in `c8volt/ops/model.go`
- [x] T009 Extend internal ops service API and constructors to accept job service dependency in `internal/services/ops/api.go`
- [x] T010 Extend public ops facade API with repair workflow methods in `c8volt/ops/api.go`
- [x] T011 Implement repair model conversions in `c8volt/ops/convert.go`
- [x] T012 Implement thin public repair facade methods in `c8volt/ops/client.go`
- [x] T013 Wire job service into ops facade construction in `c8volt/client.go` and `internal/services/ops/api.go`
- [x] T014 [P] Add foundational ops repair model/conversion tests in `c8volt/ops/client_test.go`
- [x] T015 [P] Add internal repair workflow constructor/dependency tests in `internal/services/ops/repair_test.go`
- [x] T016 Mark foundational tasks complete and record validation notes in `specs/183-ops-repair-workflows/progress.md`

**Checkpoint**: Repair request/result models and service dependencies exist without concrete target behavior.

---

## Phase 3: User Story 1 - Repair Explicit Incidents (Priority: P1) MVP

**Goal**: `c8volt ops repair incident --key ...` and stdin key input repair explicit job-backed and non-job incidents, freeze targets before mutation, and report per-step outcomes.

**Independent Test**: Command and service tests use fake incident/job behavior to verify exact-key repair, no generated-client access from command code, job-backed retry defaulting, non-job `not_applicable` job steps, and resolution confirmation.

### Tests for User Story 1

- [x] T017 [P] [US1] Add command tests for `ops repair incident --help`, no top-level parent `--key`, explicit `--key`, stdin `-`, and invalid key failures in `cmd/ops_repair_incident_test.go`
- [x] T018 [P] [US1] Add internal service tests for frozen explicit incident keys and mixed job-backed/non-job repair planning in `internal/services/ops/repair_test.go`
- [x] T019 [P] [US1] Add facade tests for explicit incident repair request conversion and error mapping in `c8volt/ops/client_test.go`
- [x] T020 [P] [US1] Add command contract metadata tests for `ops repair incident` in `cmd/command_contract_test.go`

### Implementation for User Story 1

- [x] T021 [US1] Add `ops repair incident` Cobra command, aliases if appropriate, explicit key flags, stdin key handling, and validation in `cmd/ops_repair_incident.go`
- [x] T022 [US1] Implement explicit incident repair planning and target freezing in `internal/services/ops/repair.go`
- [x] T023 [US1] Implement per-incident job applicability and default retry planning using job service primitives in `internal/services/ops/repair.go`
- [x] T024 [US1] Implement incident resolution and confirmation delegation through incident service primitives in `internal/services/ops/repair.go`
- [x] T025 [US1] Add human and JSON rendering for explicit incident repair results in `cmd/cmd_views_ops_repair.go`
- [x] T026 [US1] Set mutation, contract, output-mode, and automation metadata for `ops repair incident` in `cmd/ops_repair_incident.go`
- [x] T027 [US1] Run targeted validation with `go test ./cmd ./c8volt/ops ./internal/services/ops -run 'TestOpsRepairIncident|TestCommandContract' -count=1`
- [x] T028 [US1] Mark US1 tasks complete and record validation notes in `specs/183-ops-repair-workflows/progress.md`

**Checkpoint**: Explicit incident repair is independently functional and testable.

---

## Phase 4: User Story 2 - Discover Incidents With Filters (Priority: P2)

**Goal**: `c8volt ops repair incident` discovers incidents through native incident filters, freezes the result set, supports dry-run previews, and rejects keyed-plus-filter combinations before mutation.

**Independent Test**: Fake server/service tests verify paged filtered discovery, immutable target sets, dry-run no-mutation behavior, and local validation failures.

### Tests for User Story 2

- [x] T029 [P] [US2] Add command tests for incident filter flags, keyed-plus-filter rejection, batch-size/limit validation, and dry-run output in `cmd/ops_repair_incident_test.go`
- [x] T030 [P] [US2] Add internal service tests for filtered incident discovery, frozen target sets, and no expansion to newly created incidents in `internal/services/ops/repair_test.go`
- [x] T031 [P] [US2] Add command rendering tests for dry-run incident repair rows and JSON in `cmd/cmd_views_ops_repair_test.go`

### Implementation for User Story 2

- [x] T032 [US2] Reuse `get incident` filter parsing and validation patterns for repair incident search mode in `cmd/ops_repair_incident.go`
- [x] T033 [US2] Implement incident-filter discovery and frozen repair set construction in `internal/services/ops/repair.go`
- [x] T034 [US2] Implement dry-run behavior that discovers and validates without variable, job, or incident mutations in `internal/services/ops/repair.go`
- [x] T035 [US2] Render dry-run discovery filters, frozen keys, job applicability, retry/timeout requests, and resolution targets in `cmd/cmd_views_ops_repair.go`
- [x] T036 [US2] Run targeted validation with `go test ./cmd ./internal/services/ops -run 'TestOpsRepairIncident' -count=1`
- [x] T037 [US2] Mark US2 tasks complete and record validation notes in `specs/183-ops-repair-workflows/progress.md`

**Checkpoint**: Incident search repair and dry-run previews are independently functional.

---

## Phase 5: User Story 3 - Repair Incidents From Process Instances (Priority: P3)

**Goal**: `c8volt ops repair process-instance` selects explicit or filtered process instances, discovers active incidents for them, freezes deduped incident keys, and repairs those incidents.

**Independent Test**: Command and service tests verify explicit PI keys, stdin keys, incident-bearing search selectors, duplicate incident dedupe, and deterministic output.

### Tests for User Story 3

- [x] T038 [P] [US3] Add command tests for `ops repair process-instance --help`, explicit keys, stdin `-`, invalid keys, and keyed-plus-filter rejection in `cmd/ops_repair_processinstance_test.go`
- [x] T039 [P] [US3] Add command tests requiring `--incidents-only` or `--direct-incidents-only` in process-instance search mode in `cmd/ops_repair_processinstance_test.go`
- [x] T040 [P] [US3] Add internal service tests for process-instance selection, incident discovery, deduped incident keys, and deterministic reporting in `internal/services/ops/repair_test.go`
- [x] T041 [P] [US3] Add command contract metadata tests for `ops repair process-instance` in `cmd/command_contract_test.go`

### Implementation for User Story 3

- [x] T042 [US3] Add `ops repair process-instance` Cobra command, aliases if appropriate, explicit key flags, stdin key handling, PI filter flags, and validation in `cmd/ops_repair_processinstance.go`
- [x] T043 [US3] Reuse `get pi` search filter validation and incident-bearing selector behavior in `cmd/ops_repair_processinstance.go`
- [x] T044 [US3] Implement process-instance selection and active incident discovery in `internal/services/ops/repair.go`
- [x] T045 [US3] Deduplicate incident keys and route process-instance repair through the shared incident repair workflow in `internal/services/ops/repair.go`
- [x] T046 [US3] Render process-instance repair discovery, frozen incidents, duplicate handling, and final result in `cmd/cmd_views_ops_repair.go`
- [x] T047 [US3] Set mutation, contract, output-mode, and automation metadata for `ops repair process-instance` in `cmd/ops_repair_processinstance.go`
- [x] T048 [US3] Run targeted validation with `go test ./cmd ./internal/services/ops -run 'TestOpsRepairProcessInstance|TestCommandContract' -count=1`
- [x] T049 [US3] Mark US3 tasks complete and record validation notes in `specs/183-ops-repair-workflows/progress.md`

**Checkpoint**: Process-instance-selected incident repair is independently functional.

---

## Phase 6: User Story 4 - Apply Shared Variable Updates Safely (Priority: P4)

**Goal**: `--vars` and `--vars-file` update each unique process-instance scope once, confirm requested variables, and block dependent incident resolution when a variable update fails.

**Independent Test**: Tests verify parsing parity with `update pi`, dedupe by scope, normalized JSON confirmation, and blocked dependent incidents.

### Tests for User Story 4

- [x] T050 [P] [US4] Add command tests for `--vars`, `--vars-file`, parse failures, and variable-scope output in `cmd/ops_repair_incident_test.go` and `cmd/ops_repair_processinstance_test.go`
- [x] T051 [P] [US4] Add internal service tests for variable scope dedupe, normalized confirmation, and blocked dependent incidents in `internal/services/ops/repair_test.go`
- [x] T052 [P] [US4] Add process-instance service or facade tests for any missing variable confirmation primitive in `internal/services/processinstance` or `c8volt/process`

### Implementation for User Story 4

- [x] T053 [US4] Reuse existing process-instance variable parsing and validation for repair variable flags in `cmd/ops_repair_incident.go` and `cmd/ops_repair_processinstance.go`
- [x] T054 [US4] Add or extend process-instance service primitives for requested variable confirmation in `internal/services/processinstance/variables.go`
- [x] T055 [US4] Implement deduped variable scope update planning and execution in `internal/services/ops/repair.go`
- [x] T056 [US4] Block incident resolution for incidents dependent on failed variable scopes in `internal/services/ops/repair.go`
- [x] T057 [US4] Render variable scopes, requested names, normalized confirmation status, and blocked incidents in `cmd/cmd_views_ops_repair.go`
- [x] T058 [US4] Run targeted validation with `go test ./cmd ./internal/services/ops ./internal/services/processinstance -run 'TestOpsRepair.*Var|Test.*Variable' -count=1`
- [x] T059 [US4] Mark US4 tasks complete and record validation notes in `specs/183-ops-repair-workflows/progress.md`

**Checkpoint**: Variable repair is independently functional and safely gates resolution.

---

## Phase 7: User Story 5 - Preview Repair Without Mutation (Priority: P5)

**Goal**: `--dry-run` for both repair targets returns complete frozen target and applicability information while submitting zero mutations.

**Independent Test**: Tests prove discovery occurs, mutation services are not called, and dry-run output/report content is complete.

### Tests for User Story 5

- [x] T060 [P] [US5] Add dry-run no-mutation service tests covering variables, jobs, and incident resolution in `internal/services/ops/repair_test.go`
- [x] T061 [P] [US5] Add dry-run command tests for both repair targets with report options in `cmd/ops_repair_incident_test.go` and `cmd/ops_repair_processinstance_test.go`
- [x] T062 [P] [US5] Add dry-run rendering tests for planned report path/format and mixed job applicability in `cmd/cmd_views_ops_repair_test.go`

### Implementation for User Story 5

- [x] T063 [US5] Normalize dry-run planning behavior across explicit incident, incident search, explicit process-instance, and process-instance search modes in `internal/services/ops/repair.go`
- [x] T064 [US5] Ensure command planning validates report paths without implying mutations in `cmd/ops_repair_incident.go` and `cmd/ops_repair_processinstance.go`
- [x] T065 [US5] Ensure dry-run JSON includes frozen targets, variable scopes, job keys, applicability, retry/timeout requests, and resolution targets in `cmd/cmd_views_ops_repair.go`
- [x] T066 [US5] Run targeted validation with `go test ./cmd ./internal/services/ops -run 'TestOpsRepair.*DryRun' -count=1`
- [x] T067 [US5] Mark US5 tasks complete and record validation notes in `specs/183-ops-repair-workflows/progress.md`

**Checkpoint**: Dry-run behavior is complete and mutation-free for both targets.

---

## Phase 8: User Story 6 - Produce Audited Repair Reports (Priority: P6)

**Goal**: `--report-file` and `--report-format` write Markdown or JSON audit reports from one stable structured repair model for success and failure paths after discovery.

**Independent Test**: Request Markdown and JSON reports and verify discovery, frozen targets, step statuses, notices, errors, and final outcome.

### Tests for User Story 6

- [x] T068 [P] [US6] Add report format inference and validation tests for repair reports in `cmd/ops_contract_test.go`
- [x] T069 [P] [US6] Add Markdown report rendering tests for repair success and planned dry-run in `cmd/cmd_views_ops_repair_test.go`
- [x] T070 [P] [US6] Add JSON report rendering tests for repair success, partial failure, and post-discovery failure in `cmd/cmd_views_ops_repair_test.go`
- [x] T071 [P] [US6] Add command tests proving report files are written for failure after discovery in `cmd/ops_repair_incident_test.go`

### Implementation for User Story 6

- [x] T072 [US6] Add repair audit report model and shared report conversion in `cmd/ops_contract.go` or `cmd/cmd_views_ops_repair.go`
- [x] T073 [US6] Implement Markdown and JSON repair report renderers in `cmd/cmd_views_ops_repair.go`
- [x] T074 [US6] Add report-file/report-format flags and report write path for incident repair in `cmd/ops_repair_incident.go`
- [x] T075 [US6] Add report-file/report-format flags and report write path for process-instance repair in `cmd/ops_repair_processinstance.go`
- [x] T076 [US6] Ensure failure paths after discovery preserve report data before surfacing command errors in both repair command files
- [x] T077 [US6] Run targeted validation with `go test ./cmd -run 'TestOpsRepair.*Report|TestOpsWorkflowReport' -count=1`
- [x] T078 [US6] Mark US6 tasks complete and record validation notes in `specs/183-ops-repair-workflows/progress.md`

**Checkpoint**: Repair audit reports are independently functional for dry-run, success, partial failure, and failure paths.

---

## Final Phase: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, generated docs, full validation, and final feature readiness.

- [ ] T079 [P] Update repair command examples and user-facing help text in `cmd/ops_repair.go`, `cmd/ops_repair_incident.go`, and `cmd/ops_repair_processinstance.go`
- [ ] T080 [P] Update README-facing repair examples or confirm no README update is needed in `README.md`
- [ ] T081 Run `gofmt` on touched Go files under `cmd/`, `c8volt/`, and `internal/services/`
- [ ] T082 Run `make docs-content` and review generated files under `docs/cli/` and `docs/index.md`
- [ ] T083 Run command package validation with `go test ./cmd -count=1` for files under `cmd/`
- [ ] T084 Run facade and service validation with `go test ./c8volt/ops ./internal/services/ops ./internal/services/incident ./internal/services/processinstance ./internal/services/job -count=1`
- [ ] T085 Run race validation for worker-sensitive command paths with `go test ./cmd -race -run 'TestOpsRepair' -count=1` for `cmd/`
- [ ] T086 Run repository validation with `make test` from `Makefile`
- [ ] T087 Review `git diff --check` and final changed files before committing under `cmd/`, `c8volt/`, `internal/services/`, `README.md`, `docs/`, and `specs/183-ops-repair-workflows/`
- [ ] T088 Update final implementation notes and completed validation in `specs/183-ops-repair-workflows/progress.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup and blocks all user stories.
- **User Story 1 (P1)**: Depends on Foundational.
- **User Story 2 (P2)**: Depends on US1 because search-mode incident repair shares the explicit incident repair workflow.
- **User Story 3 (P3)**: Depends on US1 and benefits from US2 discovery/freeze behavior.
- **User Story 4 (P4)**: Depends on US1 because variable repair gates incident resolution.
- **User Story 5 (P5)**: Depends on US1, US2, US3, and US4 to provide complete dry-run content for every repair path.
- **User Story 6 (P6)**: Depends on structured results from US1 through US5.
- **Polish**: Depends on all desired user stories.

### User Story Dependencies

- **US1 Repair Explicit Incidents**: MVP after Foundational.
- **US2 Discover Incidents With Filters**: Builds on the incident repair workflow from US1.
- **US3 Repair Incidents From Process Instances**: Uses the shared incident repair workflow and process-instance discovery.
- **US4 Apply Shared Variable Updates Safely**: Adds variable mutation and confirmation before incident resolution.
- **US5 Preview Repair Without Mutation**: Completes dry-run parity after all mutation-capable paths exist.
- **US6 Produce Audited Repair Reports**: Renders the stable structured result for all paths.

### Parallel Opportunities

- T001 through T005 can be inspected in parallel before T006 records discoveries.
- T007 and T008 can run in parallel because they touch internal and public models.
- Tests marked `[P]` in each story can be drafted in parallel with service tests before implementation.
- Report rendering tests in US6 can run in parallel with report flag tests once the repair report model exists.

## Parallel Example: User Story 1

```text
Task: "Add command tests for ops repair incident --help, explicit --key, stdin -, and invalid key failures in cmd/ops_repair_incident_test.go"
Task: "Add internal service tests for frozen explicit incident keys and mixed job-backed/non-job repair planning in internal/services/ops/repair_test.go"
Task: "Add facade tests for explicit incident repair request conversion and error mapping in c8volt/ops/client_test.go"
Task: "Add command contract metadata tests for ops repair incident in cmd/command_contract_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational phases.
2. Complete User Story 1.
3. Validate explicit incident repair independently.
4. Commit with a Conventional Commit subject ending in `#183`.

### Incremental Delivery

1. Add explicit incident repair.
2. Add incident filtered discovery.
3. Add process-instance selected repair.
4. Add variable update safety.
5. Complete dry-run parity.
6. Add audit reports.
7. Refresh generated docs and run final validation.

### Ralph Iteration Discipline

1. Start each iteration by reading `specs/ralph-implementation-rules.md`, `specs/183-ops-repair-workflows/tasks.md`, `specs/183-ops-repair-workflows/plan.md`, `specs/183-ops-repair-workflows/spec.md`, and `specs/183-ops-repair-workflows/progress.md`.
2. Implement only the first incomplete work unit.
3. Inspect nearby code and tests before adding code.
4. Mark tasks complete only after implementation and relevant validation pass.
5. Include task/progress updates in the same work-unit commit.
6. Use Conventional Commits and append `#183` as the final subject token.
