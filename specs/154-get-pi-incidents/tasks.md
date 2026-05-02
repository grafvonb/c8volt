# Tasks: Keyed Process-Instance Incident Details

**Input**: Design documents from `/specs/154-get-pi-incidents/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/get-pi-with-incidents.md

**Tests**: Required by the feature specification for validation, human output, JSON output, tenant-aware request construction, multiple keys, no-incident results, default-output preservation, search-mode incident-filter preservation, and version-specific unsupported behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare shared models and contracts without changing command behavior.

- [x] T001 [P] Review generated incident search shapes and record any field mismatch in `specs/154-get-pi-incidents/research.md`
- [x] T002 [P] Add domain incident detail model in `internal/domain/processinstance.go`
- [x] T003 [P] Add public incident detail and enriched output models in `c8volt/process/model.go`
- [x] T004 [P] Add incident-enrichment contract notes to `specs/154-get-pi-incidents/contracts/get-pi-with-incidents.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add service/facade seams and command flag validation needed by all stories.

**CRITICAL**: No user story work can begin until this phase is complete.

- [x] T005 Extend `internal/services/processinstance.API` with tenant-aware incident lookup in `internal/services/processinstance/api.go`
- [x] T006 Extend public `process.API` with process-instance incident lookup/enrichment in `c8volt/process/api.go`
- [x] T007 Add conversion helpers between domain and public incident models in `c8volt/process/convert.go`
- [x] T008 Implement facade incident lookup and keyed result enrichment helpers in `c8volt/process/client.go`
- [x] T009 Add `--with-incidents` flag storage and registration in `cmd/get_processinstance.go`
- [x] T010 Add early validation for keyed-only `--with-incidents` usage in `cmd/get_processinstance.go`
- [x] T011 [P] Add foundational facade tests for incident conversion/enrichment in `c8volt/process/client_test.go`
- [x] T012 [P] Add command validation tests for `--with-incidents` without `--key` and with search-mode filters in `cmd/get_processinstance_test.go`

**Checkpoint**: The repository can represent incident details, validate the flag scope, and call incident lookup through established service/facade seams.

---

## Phase 3: User Story 1 - Show Incident Messages for Keyed Lookup (Priority: P1) MVP

**Goal**: `c8volt get pi --key <key> --with-incidents` shows incident error messages in human-readable output.

**Independent Test**: Run keyed command tests against fixture responses with one incident, multiple incidents, and no incidents, then verify human-readable output preserves the normal row and renders incident messages as indented `incident:` lines below the matching row.

### Tests for User Story 1

- [x] T013 [P] [US1] Add `v88` service tests for process-instance incident search and error-message conversion in `internal/services/processinstance/v88/service_test.go`
- [x] T014 [P] [US1] Add `v89` service tests for process-instance incident search and error-message conversion in `internal/services/processinstance/v89/service_test.go`
- [x] T015 [P] [US1] Add command human-output test for one indented `incident:` message in `cmd/get_processinstance_test.go`
- [x] T016 [P] [US1] Add command human-output tests for multiple indented `incident:` lines and no incidents in `cmd/get_processinstance_test.go`

### Implementation for User Story 1

- [x] T017 [US1] Add generated incident search method to `internal/services/processinstance/v88/contract.go`
- [x] T018 [US1] Implement `v88` incident search request and conversion in `internal/services/processinstance/v88/service.go` and `internal/services/processinstance/v88/convert.go`
- [x] T019 [US1] Add generated incident search method to `internal/services/processinstance/v89/contract.go`
- [x] T020 [US1] Implement `v89` incident search request and conversion in `internal/services/processinstance/v89/service.go` and `internal/services/processinstance/v89/convert.go`
- [x] T021 [US1] Call facade enrichment after keyed process-instance lookup when `--with-incidents` is set in `cmd/get_processinstance.go`
- [x] T022 [US1] Add enriched human renderer for indented `incident:` message lines in `cmd/cmd_views_get.go`

**Checkpoint**: User Story 1 is independently functional and testable.

---

## Phase 4: User Story 2 - Consume Incident Details in JSON (Priority: P2)

**Goal**: `c8volt get pi --key <key> --with-incidents --json` returns stable machine-readable incident details.

**Independent Test**: Run JSON command tests and verify incident details are attached to the matching process instance.

### Tests for User Story 2

- [x] T023 [P] [US2] Add JSON command test for one key with incident details in `cmd/get_processinstance_test.go`
- [x] T024 [P] [US2] Add JSON command test for multiple keys with per-key incident association in `cmd/get_processinstance_test.go`
- [x] T025 [P] [US2] Add JSON command test for an empty incidents collection in `cmd/get_processinstance_test.go`

### Implementation for User Story 2

- [x] T026 [US2] Add enriched JSON renderer that emits `total` and per-item `incidents` in `cmd/cmd_views_get.go`
- [x] T027 [US2] Ensure enriched JSON output preserves existing command envelope behavior in `cmd/cmd_views_get.go`
- [x] T028 [US2] Ensure empty incident results render as an empty collection when enrichment was requested in `c8volt/process/client.go`

**Checkpoint**: User Stories 1 and 2 both work independently.

---

## Phase 5: User Story 3 - Protect Existing Command Semantics (Priority: P3)

**Goal**: The new flag is additive and does not change default keyed output or search-mode incident filters.

**Independent Test**: Run default-output and search-filter regression tests with the new flag omitted.

### Tests for User Story 3

- [x] T029 [P] [US3] Add regression test proving default keyed human output is unchanged without `--with-incidents` in `cmd/get_processinstance_test.go`
- [x] T030 [P] [US3] Add regression test proving default keyed JSON output is unchanged without `--with-incidents` in `cmd/get_processinstance_test.go`
- [x] T031 [P] [US3] Add regression tests preserving `--incidents-only` and `--no-incidents-only` search filters in `cmd/get_processinstance_test.go`

### Implementation for User Story 3

- [x] T032 [US3] Keep existing `listProcessInstancesView` path untouched when `--with-incidents` is omitted in `cmd/get_processinstance.go`
- [x] T033 [US3] Ensure `oneLinePI` remains the default incident-marker-only renderer in `cmd/cmd_views_get.go`
- [x] T034 [US3] Verify search-mode population and validation still use existing incident filter behavior in `cmd/get_processinstance.go`

**Checkpoint**: User Stories 1, 2, and 3 are independently functional.

---

## Phase 6: User Story 4 - Respect Tenant and Version Boundaries (Priority: P4)

**Goal**: Supported versions use tenant-aware incident search; v8.7 fails explicitly rather than falling back to tenant-unsafe behavior.

**Independent Test**: Run service tests that inspect incident request bodies and command tests for unsupported v8.7 behavior.

### Tests for User Story 4

- [ ] T035 [P] [US4] Add `v88` tenant-filter request construction assertions in `internal/services/processinstance/v88/service_test.go`
- [ ] T036 [P] [US4] Add `v89` tenant-filter request construction assertions in `internal/services/processinstance/v89/service_test.go`
- [ ] T037 [P] [US4] Add `v87` unsupported incident lookup service test in `internal/services/processinstance/v87/service_test.go`
- [ ] T038 [P] [US4] Add command unsupported-version test for `--with-incidents` on Camunda 8.7 in `cmd/get_processinstance_test.go`

### Implementation for User Story 4

- [ ] T039 [US4] Include configured tenant in `v88` incident search filters in `internal/services/processinstance/v88/service.go`
- [ ] T040 [US4] Include configured tenant in `v89` incident search filters in `internal/services/processinstance/v89/service.go`
- [ ] T041 [US4] Implement `v87` incident lookup as explicit unsupported in `internal/services/processinstance/v87/service.go`
- [ ] T042 [US4] Ensure unsupported errors map through existing facade and command handlers in `c8volt/process/client.go` and `cmd/get_processinstance.go`

**Checkpoint**: All user stories are independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, formatting, and validation across the completed feature.

- [ ] T043 [P] Update command help examples and flag description for `--with-incidents` in `cmd/get_processinstance.go`
- [ ] T044 Regenerate CLI documentation with `make docs-content`
- [ ] T045 [P] Review README process-instance examples and update `README.md` only if the new flag belongs in existing examples
- [ ] T046 [P] Run `gofmt` on changed Go files in `cmd/`, `c8volt/process/`, `internal/domain/`, and `internal/services/processinstance/`
- [ ] T047 Run targeted validation with `go test ./cmd ./c8volt/process ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1`
- [ ] T048 Run full repository validation with `make test`
- [ ] T049 Confirm `specs/154-get-pi-incidents/quickstart.md` scenarios match final command behavior

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational and is the MVP.
- **User Story 2 (Phase 4)**: Depends on User Story 1 enriched data being available.
- **User Story 3 (Phase 5)**: Depends on the new render path existing so regressions can prove it remains opt-in.
- **User Story 4 (Phase 6)**: Depends on the service method shape from User Story 1.
- **Polish (Phase 7)**: Depends on all desired user stories.

### User Story Dependencies

- **User Story 1 (P1)**: No dependency on later stories after Foundational.
- **User Story 2 (P2)**: Depends on US1 incident lookup and enrichment.
- **User Story 3 (P3)**: Can run after US1/US2 render branching exists.
- **User Story 4 (P4)**: Can be implemented alongside US1 service work but must finish before release.

### Parallel Opportunities

- Setup tasks T001-T004 can run in parallel.
- Foundational tests T011-T012 can be written alongside T005-T010 once model names are settled.
- `v88` and `v89` service tests and implementation can run in parallel for US1 and US4.
- Command human-output and JSON-output tests can run in parallel once enriched models exist.
- README review and gofmt can run in parallel after implementation stabilizes.

---

## Parallel Example: User Story 1

```text
Task: "Add v8.8 service tests for process-instance incident search in internal/services/processinstance/v88/service_test.go"
Task: "Add v8.9 service tests for process-instance incident search in internal/services/processinstance/v89/service_test.go"
Task: "Add human-output command tests in cmd/get_processinstance_test.go"
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete User Story 1 only.
3. Validate with targeted service/facade/command tests.
4. Stop if needed with a usable human-readable `--with-incidents` keyed lookup.

### Incremental Delivery

1. Add human-readable keyed enrichment.
2. Add JSON enrichment.
3. Lock down validation and default-output preservation.
4. Complete tenant/version safeguards.
5. Update docs and run full validation.

### Commit Guidance

Every commit subject for this feature must use Conventional Commits and end with `#154`, for example `feat(get-pi): add keyed incident enrichment #154`.
