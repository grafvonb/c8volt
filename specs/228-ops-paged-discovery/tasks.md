# Tasks: Ops Paged Discovery Scope

**Input**: Design documents from `specs/228-ops-paged-discovery/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/)

**Tests**: Required by the specification. Write or update targeted tests before implementation for each story.

**Implementation Context**: Every Ralph implementation iteration must read `specs/ralph-implementation-rules.md`. Ralph launch instructions must include `--implementation-context specs/ralph-implementation-rules.md`.

**Organization**: Tasks are grouped by user story so each story can be implemented and tested in a small Ralph iteration.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm local context and keep implementation bound to repository rules.

- [x] T001 Read `specs/ralph-implementation-rules.md` and verify no conflict with `specs/228-ops-paged-discovery/spec.md`
- [x] T002 [P] Review existing paged discovery patterns in `internal/services/processinstance/retention_discovery.go` and `internal/services/processinstance/orphan_discovery.go`
- [x] T003 [P] Review affected output renderers in `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`, `cmd/cmd_views_ops_repair.go`, and `cmd/cmd_views_ops_purge_all_processdefinitions.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add the shared frozen-scope status shape and page-capable service boundaries required by all stories.

**CRITICAL**: No user story work can begin until this phase is complete.

- [x] T004 Add discovery completeness fields or a shared discovery status type to `internal/domain/ops_incident_purge.go`, `internal/domain/ops_repair.go`, and `internal/domain/ops_all_process_definitions_purge.go`
- [x] T005 Add matching public facade fields and JSON tags in `c8volt/ops/model.go`
- [x] T006 Update domain-to-facade conversion for the new discovery status fields in `c8volt/ops/convert.go`
- [x] T007 Add or extend page-capable process-definition service contracts in `internal/services/processdefinition/api.go`
- [x] T008 [P] Add process-definition page request/response domain types if needed in `internal/domain/processdefinition.go`
- [x] T009 Implement v8.9 process-definition page search request and conversion in `internal/services/processdefinition/v89/service.go`
- [x] T010 Implement v8.8 process-definition page search request and conversion in `internal/services/processdefinition/v88/service.go`
- [x] T011 Implement v8.7 process-definition page search request and conversion in `internal/services/processdefinition/v87/service.go`
- [x] T012 [P] Add process-definition page service tests in `internal/services/processdefinition/v89/service_test.go`
- [x] T013 [P] Add process-definition page service tests in `internal/services/processdefinition/v88/service_test.go`
- [x] T014 [P] Add process-definition page service tests in `internal/services/processdefinition/v87/service_test.go`

**Checkpoint**: Discovery status and process-definition paging are available to service workflows.

---

## Phase 3: User Story 1 - Purge Incident-Bearing Process Instances Uses Full Scope (Priority: P1) MVP

**Goal**: `ops purge process-instances-with-incidents` and aliases freeze the full incident-derived process-instance scope by default.

**Independent Test**: Multi-page incident fixtures prove `ops purge piwi --dry-run` discovers all matching incidents and process-instance candidates unless `--limit` is supplied.

### Tests for User Story 1

- [x] T015 [P] [US1] Add multi-page incident purge service test without `--limit` in `internal/services/ops/incident_purge_test.go`
- [x] T016 [P] [US1] Add incident purge `--limit` service test proving `--batch-size` is page size only in `internal/services/ops/incident_purge_test.go`
- [x] T017 [P] [US1] Add confirmation frozen-scope reuse command test for `ops purge piwi` in `cmd/ops_purge_processinstances_with_incidents_test.go`

### Implementation for User Story 1

- [x] T018 [US1] Replace single incident search with complete-by-default paged discovery in `internal/services/ops/incident_purge.go`
- [x] T019 [US1] Populate incident purge discovery completeness and user-limited status in `internal/services/ops/incident_purge.go`
- [x] T020 [US1] Preserve frozen candidate reuse after confirmation in `internal/services/ops/incident_purge.go` and `cmd/ops_purge_processinstances_with_incidents.go`
- [x] T021 [US1] Update incident purge human, JSON, and Markdown report rendering for discovery status in `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`
- [x] T022 [US1] Run `go test ./internal/services/ops -run 'Test.*IncidentPurge' -count=1`
- [x] T023 [US1] Run `go test ./cmd -run 'TestOpsPurgeProcessInstancesWithIncidents' -count=1`

**Checkpoint**: User Story 1 is independently functional and testable.

---

## Phase 4: User Story 2 - Repair Workflows Discover All Matching Candidates (Priority: P2)

**Goal**: `ops repair incident` and `ops repair process-instance` search modes freeze all matching repair candidates by default.

**Independent Test**: Multi-page incident and incident-bearing process-instance fixtures prove repair previews include every match unless `--limit` is supplied.

### Tests for User Story 2

- [x] T024 [P] [US2] Add multi-page repair incident service test in `internal/services/ops/repair_test.go`
- [x] T025 [P] [US2] Add multi-page repair process-instance service test in `internal/services/ops/repair_test.go`
- [x] T026 [P] [US2] Add repair `--limit` service tests for incident and process-instance search modes in `internal/services/ops/repair_test.go`
- [x] T027 [P] [US2] Add frozen-scope reuse or no-second-discovery command test in `cmd/ops_repair_incident_test.go`
- [x] T028 [P] [US2] Add frozen-scope reuse or no-second-discovery command test in `cmd/ops_repair_processinstance_test.go`

### Implementation for User Story 2

- [x] T029 [US2] Replace single incident repair search with complete-by-default paged discovery in `internal/services/ops/repair.go`
- [x] T030 [US2] Replace single process-instance repair search with complete-by-default paged discovery in `internal/services/ops/repair.go`
- [x] T031 [US2] Populate repair frozen-set completeness and user-limited status in `internal/services/ops/repair.go`
- [x] T032 [US2] Update repair human, JSON, and Markdown report rendering for discovery status in `cmd/cmd_views_ops_repair.go`
- [x] T033 [US2] Run `go test ./internal/services/ops -run 'Test.*Repair' -count=1`
- [x] T034 [US2] Run `go test ./cmd -run 'TestOpsRepair(Incident|ProcessInstance)' -count=1`

**Checkpoint**: User Stories 1 and 2 work independently.

---

## Phase 5: User Story 3 - Process Definition Purge Discovers All Matching Definitions (Priority: P3)

**Goal**: `ops purge all-process-definitions` and aliases page through all matching process definitions by default.

**Independent Test**: Multi-page process-definition fixtures prove `ops purge apd --dry-run` freezes all matching definitions unless `--limit` is supplied.

### Tests for User Story 3

- [ ] T035 [P] [US3] Add all-process-definitions multi-page service test in `internal/services/ops/all_process_definitions_purge_test.go`
- [ ] T036 [P] [US3] Add all-process-definitions `--limit` service test in `internal/services/ops/all_process_definitions_purge_test.go`
- [ ] T037 [P] [US3] Add all-process-definitions command flag and confirmation reuse tests in `cmd/ops_purge_all_processdefinitions_test.go`

### Implementation for User Story 3

- [ ] T038 [US3] Add `BatchSize` and `Limit` request fields to `internal/domain/ops_all_process_definitions_purge.go` and `c8volt/ops/model.go`
- [ ] T039 [US3] Wire `BatchSize` and `Limit` through facade conversion in `c8volt/ops/convert.go`
- [ ] T040 [US3] Add `--batch-size` and `--limit` flags and validation to `cmd/ops_purge_all_processdefinitions.go`
- [ ] T041 [US3] Replace one-shot process-definition search with complete-by-default paged discovery in `internal/services/ops/all_process_definitions_purge.go`
- [ ] T042 [US3] Populate all-process-definitions discovery completeness and user-limited status in `internal/services/ops/all_process_definitions_purge.go`
- [ ] T043 [US3] Update all-process-definitions human, JSON, and Markdown report rendering for discovery status in `cmd/cmd_views_ops_purge_all_processdefinitions.go`
- [ ] T044 [US3] Run `go test ./internal/services/ops -run 'Test.*AllProcessDefinitionsPurge' -count=1`
- [ ] T045 [US3] Run `go test ./cmd -run 'TestOpsPurgeAllProcessDefinitions' -count=1`

**Checkpoint**: User Stories 1, 2, and 3 work independently.

---

## Phase 6: User Story 4 - Operators Can Audit Discovery Completeness (Priority: P4)

**Goal**: Human, JSON, Markdown report, automation, help, and docs output expose full vs. user-limited discovery consistently.

**Independent Test**: Output tests show every affected workflow reports the same frozen scope and discovery status across modes.

### Tests for User Story 4

- [ ] T046 [P] [US4] Add command contract assertions for updated discovery flag/help semantics in `cmd/command_contract_test.go`
- [ ] T047 [P] [US4] Add renderer tests for discovery complete and user-limited output in `cmd/cmd_views_ops_repair_test.go`
- [ ] T048 [P] [US4] Add docs generator expectations for affected help text in `docsgen/main_test.go`

### Implementation for User Story 4

- [ ] T049 [US4] Update affected command long help and flag descriptions in `cmd/ops_purge_processinstances_with_incidents.go`, `cmd/ops_repair_incident.go`, `cmd/ops_repair_processinstance.go`, and `cmd/ops_purge_all_processdefinitions.go`
- [ ] T050 [US4] Update ops documentation in `README.md` and affected `docs/ops/*.md` files
- [ ] T051 [US4] Run `make docs-content` to regenerate generated CLI docs after command metadata changes
- [ ] T052 [US4] Run `go test ./docsgen ./cmd -count=1`

**Checkpoint**: All user-facing and machine-facing output surfaces document and expose discovery status.

---

## Final Phase: Polish & Cross-Cutting Concerns

**Purpose**: Verify the full feature, keep generated artifacts synchronized, and capture follow-up work.

- [ ] T053 Run `go test ./internal/services/incident ./internal/services/processdefinition ./internal/services/ops ./c8volt/ops ./cmd -count=1`
- [ ] T054 Run `make test`
- [ ] T055 Review smoke-test process-definition cleanup eligibility in `internal/services/ops/smoke_test_service.go` and either include a small safe fix or record follow-up notes in `specs/228-ops-paged-discovery/quickstart.md`
- [ ] T056 Verify generated docs and working tree status with `git status --short`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup and blocks all user stories.
- **User Story 1 (P1)**: Depends on Foundational. This is the MVP.
- **User Story 2 (P2)**: Depends on Foundational and can be implemented after or alongside US1 if the shared status shape is stable.
- **User Story 3 (P3)**: Depends on Foundational process-definition page support.
- **User Story 4 (P4)**: Depends on the discovery status fields from US1-US3 so renderers and docs describe real behavior.
- **Polish**: Depends on selected user stories being complete.

### User Story Dependencies

- **US1**: Independent after Foundational; delivers the reported `piwi` defect fix.
- **US2**: Independent after Foundational; shares status model and paging approach.
- **US3**: Independent after Foundational; requires process-definition page APIs.
- **US4**: Cross-cutting output/documentation story after service behavior exists.

### Within Each User Story

- Write or update tests first and confirm they fail for the current single-page behavior.
- Implement service discovery behavior before renderer changes.
- Preserve frozen-scope reuse before broadening help/docs.
- Run targeted validation at each story checkpoint.

## Parallel Opportunities

- T002 and T003 can run in parallel.
- T012, T013, and T014 can run in parallel after process-definition page API design is clear.
- US1, US2, and US3 test-writing tasks can run in parallel after Foundational if separate workers coordinate shared model names.
- Renderer/docs tasks in US4 can run in parallel once service result shapes are stable.

## Parallel Example: User Story 2

```text
Task: "T024 [P] [US2] Add multi-page repair incident service test in internal/services/ops/repair_test.go"
Task: "T025 [P] [US2] Add multi-page repair process-instance service test in internal/services/ops/repair_test.go"
Task: "T027 [P] [US2] Add frozen-scope reuse or no-second-discovery command test in cmd/ops_repair_incident_test.go"
Task: "T028 [P] [US2] Add frozen-scope reuse or no-second-discovery command test in cmd/ops_repair_processinstance_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational tasks.
2. Complete US1 for `ops purge piwi`.
3. Run US1 targeted service and command tests.
4. Stop and validate before expanding to repair and process-definition purge.

### Incremental Delivery

1. Foundation: shared status and page-capable process-definition API.
2. US1: incident purge full discovery.
3. US2: repair full discovery.
4. US3: all-process-definitions full discovery.
5. US4: auditability, help, and docs.
6. Polish: broad validation and smoke-test follow-up decision.

## Notes

- Commit subjects must use Conventional Commits format and append `#228` as the final token.
- Do not stage or commit unless the Ralph workflow explicitly reaches its commit step and validation passes.
- Generated CLI docs must be regenerated with `make docs-content`; do not hand-edit generated docs under `docs/cli/`.
- Preserve existing Camunda 8.7, 8.8, and 8.9 compatibility behavior unless an affected command already gates a narrower version.
