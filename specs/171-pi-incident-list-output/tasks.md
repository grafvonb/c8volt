# Tasks: Process Instance Incident List Output

**Input**: Design documents from `/specs/171-pi-incident-list-output/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are required by the feature specification and constitution. Story test tasks should be written before implementation and should fail until the story implementation is complete.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files or only adds tests/docs
- **[Story]**: Maps to the user story from [spec.md](./spec.md)
- Every task names exact repository paths

## Phase 1: Setup (Shared Discovery)

**Purpose**: Confirm the current process-instance get, incident enrichment, paging, and documentation paths before changing behavior.

- [x] T001 Inspect keyed `--with-incidents` validation and list/search paging flow in `cmd/get_processinstance.go`
- [x] T002 [P] Inspect incident-enriched get rendering and JSON envelope behavior in `cmd/cmd_views_processinstance_incidents.go` and `cmd/cmd_views_get_test.go`
- [x] T003 [P] Inspect walk incident rendering reuse of `incidentHumanLine` in `cmd/cmd_views_walk_incidents.go` and `cmd/walk_test.go`
- [x] T004 [P] Inspect facade enrichment association tests in `c8volt/process/client.go`, `c8volt/process/model.go`, and `c8volt/process/client_test.go`
- [x] T005 [P] Inspect process-instance command docs and generated documentation paths in `README.md`, `docs/index.md`, `docs/cli/`, and `docsgen/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared option state and rendering helpers that every story can reuse.

**Critical**: No user story implementation should begin until this phase is complete.

- [x] T006 Add `--incident-message-limit` flag storage, registration, help text, and reset behavior in `cmd/get_processinstance.go` and `cmd/get_processinstance_test.go`
- [x] T007 Add validation for `--incident-message-limit` dependency and non-negative values in `cmd/get_processinstance.go`
- [x] T008 Add human incident message truncation helper tests for unlimited, exact-limit, truncated, and multi-byte messages in `cmd/cmd_views_get_test.go`
- [x] T009 Implement reusable human incident message truncation support used by `incidentHumanLine` in `cmd/cmd_views_processinstance_incidents.go`
- [x] T010 Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestIncident|TestGetProcessInstance.*Incident|TestValidatePI' -count=1` and fix foundational regressions

**Checkpoint**: Shared flag state, validation, and truncation helpers exist before story behavior is wired.

---

## Phase 3: User Story 1 - Show Direct Incidents In List Output (Priority: P1) MVP

**Goal**: `c8volt get pi --with-incidents` works without `--key` and renders direct incidents under each matching process-instance row.

**Independent Test**: Run a list/search selector with `--with-incidents` against process instances with direct incidents and verify row output, incident association, and found count.

### Tests for User Story 1

- [x] T011 [P] [US1] Add command test for `get pi --incidents-only --with-incidents` rendering direct incident lines below matching rows in `cmd/get_processinstance_test.go`
- [x] T012 [P] [US1] Add command test proving direct incident lookup runs only for listed or limited process instances in `cmd/get_processinstance_test.go`
- [x] T013 [P] [US1] Add view test for multiple enriched process-instance rows preserving per-row incident association in `cmd/cmd_views_get_test.go`

### Implementation for User Story 1

- [x] T014 [US1] Relax `validatePIWithIncidentsUsage` to allow list/search mode while keeping `--total` invalid in `cmd/get_processinstance.go`
- [x] T015 [US1] Enrich non-incremental list/search `ProcessInstances` with incidents before rendering in `cmd/get_processinstance.go`
- [x] T016 [US1] Support incident-enriched rendering for incremental human list/search pages without changing paging prompts or found counts in `cmd/get_processinstance.go`
- [x] T017 [US1] Preserve incident lookup options, tenant handling, and per-key association through existing facade enrichment in `c8volt/process/client.go`
- [x] T018 [US1] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process -run 'Test(GetProcessInstance.*Incident|IncidentEnriched|Client_EnrichProcessInstances)' -count=1` and fix regressions

**Checkpoint**: User Story 1 is independently complete when list/search human output shows direct incidents below the correct rows and keyed behavior still passes.

---

## Phase 4: User Story 2 - Preserve Enriched JSON Behavior (Priority: P2)

**Goal**: `c8volt get pi --json --with-incidents` in list/search mode returns the existing enriched incident shape with full messages.

**Independent Test**: Run list/search `get pi --json --with-incidents` and verify the JSON payload uses the enriched shape, full messages, and default metadata.

### Tests for User Story 2

- [x] T019 [P] [US2] Add command JSON test for list/search `get pi --json --with-incidents` enriched payload shape in `cmd/get_processinstance_test.go`
- [x] T020 [P] [US2] Add command JSON test proving `--incident-message-limit` does not truncate JSON incident messages in `cmd/get_processinstance_test.go`
- [x] T021 [P] [US2] Add keyed JSON regression test showing existing `get pi --key <key> --json --with-incidents` shape remains unchanged in `cmd/get_processinstance_test.go`

### Implementation for User Story 2

- [x] T022 [US2] Route collected list/search JSON results through `incidentEnrichedProcessInstancesView` when `--json --with-incidents` is set in `cmd/get_processinstance.go`
- [x] T023 [US2] Ensure `incidentEnrichedProcessInstancesWithAgeMeta` keeps full incident messages and default process-instance age metadata in `cmd/cmd_views_processinstance_incidents.go`
- [x] T024 [US2] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestGetProcessInstance.*JSON.*Incident|TestIncidentEnrichedProcessInstancesView_JSON' -count=1` and fix regressions

**Checkpoint**: User Story 2 is independently complete when list/search JSON enrichment matches the keyed enriched shape and full messages are preserved.

---

## Phase 5: User Story 3 - Explain Indirect Incident Markers (Priority: P3)

**Goal**: Rows marked `inc!` with no direct incidents get short row notes, and list output prints one tree-inspection warning after the list.

**Independent Test**: Run list/search human output with one or more indirect markers and verify per-row notes plus one de-duplicated post-list warning.

### Tests for User Story 3

- [ ] T025 [P] [US3] Add view test for a single indirect marker row rendering a short indented note in `cmd/cmd_views_get_test.go`
- [ ] T026 [P] [US3] Add view test for multiple indirect marker rows rendering multiple short notes and one warning after the list in `cmd/cmd_views_get_test.go`
- [ ] T027 [P] [US3] Add command test proving list-mode indirect marker behavior appears after incident enrichment returns empty direct incidents in `cmd/get_processinstance_test.go`

### Implementation for User Story 3

- [ ] T028 [US3] Change `incidentEnrichedProcessInstancesView` to render row-local indirect notes and defer the tree-inspection warning until after all rows in `cmd/cmd_views_processinstance_incidents.go`
- [ ] T029 [US3] Update indirect marker note and warning text to be short per row and de-duplicated per list output in `cmd/cmd_views_processinstance_incidents.go`
- [ ] T030 [US3] Preserve `found: <n>` placement and stderr/stdout behavior for warnings according to existing rendering helpers in `cmd/cmd_views_processinstance_incidents.go`
- [ ] T031 [US3] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*Indirect.*Incident|TestIncidentEnrichedProcessInstancesView' -count=1` and fix regressions

**Checkpoint**: User Story 3 is independently complete when indirect marker rows are locally explained and list-level guidance is printed once.

---

## Phase 6: User Story 4 - Use Compact Human Incident Lines (Priority: P4)

**Goal**: Human incident lines use `inc <incident-key>:` for both get and walk, and `--incident-message-limit` truncates only human messages.

**Independent Test**: Run get and walk incident human output with long messages and verify compact prefixes, character-safe truncation, and unlimited default behavior.

### Tests for User Story 4

- [ ] T032 [P] [US4] Add get view test proving `incidentHumanLine` renders `inc <incident-key>:` instead of `incident <incident-key>:` in `cmd/cmd_views_get_test.go`
- [ ] T033 [P] [US4] Add walk command or view regression test proving `walk pi --with-incidents` uses `inc <incident-key>:` in `cmd/walk_test.go`
- [ ] T034 [P] [US4] Add command test proving `--incident-message-limit <chars>` truncates only human incident messages and appends `...` in `cmd/get_processinstance_test.go`
- [ ] T035 [P] [US4] Add command test proving default limit `0` leaves human incident messages unchanged in `cmd/get_processinstance_test.go`

### Implementation for User Story 4

- [ ] T036 [US4] Change `incidentHumanLine` to use the `inc <incident-key>:` prefix and apply human message truncation in `cmd/cmd_views_processinstance_incidents.go`
- [ ] T037 [US4] Ensure `cmd/cmd_views_walk_incidents.go` continues to call the shared incident human line helper without command-specific prefix logic
- [ ] T038 [US4] Update existing tests that assert the old `incident <incident-key>:` prefix in `cmd/` to the new `inc <incident-key>:` behavior
- [ ] T039 [US4] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test(Incident|Walk).*' -count=1` and fix regressions

**Checkpoint**: User Story 4 is independently complete when compact prefixes and human-only truncation pass for get and walk.

---

## Phase 7: User Story 5 - Validate Incident Options Safely (Priority: P5)

**Goal**: Invalid combinations and values fail clearly, help/docs describe the new behavior, and existing behavior without `--with-incidents` remains unchanged.

**Independent Test**: Run valid and invalid invocations around `--with-incidents`, `--incident-message-limit`, `--total`, keyed lookup, and list/search selectors, then inspect help and generated docs.

### Tests for User Story 5

- [ ] T040 [P] [US5] Update validation tests so `get pi --with-incidents` without `--key` is accepted in list/search mode in `cmd/get_processinstance_test.go`
- [ ] T041 [P] [US5] Add validation test for `--with-incidents` remaining invalid with `--total` in `cmd/get_processinstance_test.go`
- [ ] T042 [P] [US5] Add validation tests for `--incident-message-limit` without `--with-incidents` and negative values in `cmd/get_processinstance_test.go`
- [ ] T043 [P] [US5] Add help or command contract test for `--incident-message-limit` and updated `--with-incidents` help text in `cmd/command_contract_test.go` or `cmd/get_processinstance_test.go`
- [ ] T044 [P] [US5] Add regression test proving output without `--with-incidents` does not perform incident lookups and remains unchanged in `cmd/get_processinstance_test.go`

### Implementation for User Story 5

- [ ] T045 [US5] Update `--with-incidents` help text and examples to describe keyed and list/search enrichment in `cmd/get_processinstance.go`
- [ ] T046 [US5] Update README process-instance incident examples and wording in `README.md`
- [ ] T047 [US5] Update site documentation source examples in `docs/index.md`
- [ ] T048 [US5] Regenerate generated CLI documentation under `docs/cli/` with `make docs-content`
- [ ] T049 [US5] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestGetProcessInstance.*(WithIncidents|IncidentMessageLimit|Help|Contract|Default)' -count=1` and fix regressions

**Checkpoint**: User Story 5 is independently complete when invalid inputs fail clearly, valid list/search incident commands are accepted, and docs match behavior.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final cleanup, formatting, and repository-wide proof.

- [ ] T050 [P] Run `gofmt -w cmd/get_processinstance.go cmd/get_processinstance_test.go cmd/cmd_views_processinstance_incidents.go cmd/cmd_views_get_test.go cmd/cmd_views_walk_incidents.go cmd/walk_test.go cmd/command_contract_test.go c8volt/process/client.go c8volt/process/client_test.go c8volt/process/model.go`
- [ ] T051 Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process -count=1` and fix regressions
- [ ] T052 Run `make docs-content` and fix documentation generation issues
- [ ] T053 Run `make test` and fix repository validation failures
- [ ] T054 [P] Review [quickstart.md](./quickstart.md) against implemented behavior and update if command examples or validation commands changed
- [ ] T055 Verify `git diff` contains only issue #171 implementation, docs, generated docs, and Speckit artifacts before commit

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on setup and blocks all user stories.
- **US1 (Phase 3)**: Depends on foundational flag and helper setup.
- **US2 (Phase 4)**: Depends on US1 list/search enrichment path.
- **US3 (Phase 5)**: Depends on US1 enriched list rendering.
- **US4 (Phase 6)**: Depends on foundational truncation helper and shared rendering.
- **US5 (Phase 7)**: Depends on final command semantics from US1-US4.
- **Polish (Phase 8)**: Depends on the desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: First user-visible slice; proves list/search human direct incident rendering.
- **User Story 2 (P2)**: Builds on list/search enrichment and validates JSON shape.
- **User Story 3 (P3)**: Builds on enriched human list rendering and adds indirect marker handling.
- **User Story 4 (P4)**: Can proceed after foundational helper setup, but final regression should run after US1/US3 rendering changes.
- **User Story 5 (P5)**: Finishes validation, docs, and unchanged-default behavior once feature behavior is stable.

### Parallel Opportunities

- T002 through T005 can run in parallel during setup.
- T011 through T013 can be written in parallel for US1.
- T019 through T021 can be written in parallel for US2.
- T025 through T027 can be written in parallel for US3.
- T032 through T035 can be written in parallel for US4.
- T040 through T044 can be written in parallel for US5.
- T050 and T054 can run in parallel after implementation is complete.

## Parallel Example: User Story 1

```text
Task: "Add command test for `get pi --incidents-only --with-incidents` rendering direct incident lines below matching rows in cmd/get_processinstance_test.go"
Task: "Add command test proving direct incident lookup runs only for listed or limited process instances in cmd/get_processinstance_test.go"
Task: "Add view test for multiple enriched process-instance rows preserving per-row incident association in cmd/cmd_views_get_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete User Story 1 to deliver list/search human direct incident rendering.
3. Stop and run the US1 targeted tests before adding JSON, indirect marker, truncation, or docs behavior.

### Incremental Delivery

1. Add list/search human direct incident rendering.
2. Add list/search JSON enrichment with full messages.
3. Add indirect marker row notes and one de-duplicated warning.
4. Add compact human prefix and human-only message truncation.
5. Finish validation, help, README/docs, generated CLI docs, and unchanged-default regressions.
6. Run targeted tests after each story, then `make test` before commit.

### Commit Guidance

Use Conventional Commit subjects and append the issue number as the final token, for example:

```text
feat(get-pi): support incident details in list output #171
```
