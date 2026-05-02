# Tasks: Walk Process-Instance Incident Details

**Input**: Design documents from `/specs/157-walk-pi-incidents/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/walk-pi-with-incidents.md, quickstart.md

**Tests**: Required by the feature specification for validation, human output, JSON output, tenant-aware request construction, incident lookup failure, no-incident results, default-output preservation, traversal preservation, key-only rejection, and version-specific unsupported behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare shared enriched traversal shape and review existing issue #154 incident behavior without changing command output.

- [x] T001 [P] Review issue #154 incident enrichment behavior and record any field mismatch in `specs/157-walk-pi-incidents/research.md`
- [x] T002 [P] Add incident-enriched traversal item/result public models in `c8volt/process/model.go`
- [x] T003 [P] Add walk incident enrichment contract notes to `specs/157-walk-pi-incidents/contracts/walk-pi-with-incidents.md`
- [x] T004 [P] Add fixture helpers for walked incident details in `cmd/walk_test.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add facade helpers and command flag validation needed by all user stories.

**CRITICAL**: No user story work can begin until this phase is complete.

- [x] T005 Extend public `process.API` or `process.Walker` contract with traversal incident enrichment in `c8volt/process/api.go` and `c8volt/process/walker.go`
- [x] T006 Implement traversal enrichment helper that fetches incidents for returned traversal keys in `c8volt/process/client.go`
- [x] T007 Add filtering helpers for per-key incident association in `c8volt/process/client.go`
- [x] T008 Add `--with-incidents` flag storage and registration in `cmd/walk_processinstance.go`
- [x] T009 Add early validation for keyed-only `--with-incidents` usage in `cmd/walk_processinstance.go`
- [x] T010 Add foundational facade tests for traversal incident enrichment in `c8volt/process/client_test.go`
- [x] T011 [P] Add command validation tests for `--with-incidents` without `--key` in `cmd/walk_test.go`

**Checkpoint**: The repository can represent enriched traversal results, validate the flag scope, and call incident lookup through established facade/service seams.

---

## Phase 3: User Story 1 - Show Incident Messages While Walking (Priority: P1) MVP

**Goal**: `c8volt walk pi --key <key> --with-incidents` shows incident keys and error messages in human-readable walk output.

**Independent Test**: Run keyed walk command tests against fixture responses with one incident, multiple incidents, and no incidents, then verify human-readable output preserves traversal ordering and renders incident messages under the matching process-instance rows.

### Tests for User Story 1

- [x] T012 [P] [US1] Add command human-output test for one walked process instance with one incident in `cmd/walk_test.go`
- [x] T013 [P] [US1] Add command human-output test for multiple walked instances with incidents in `cmd/walk_test.go`
- [x] T014 [P] [US1] Add command human-output test for walked instances without incidents in `cmd/walk_test.go`
- [x] T015 [P] [US1] Add facade test proving incident lookups run only for traversal result keys in `c8volt/process/client_test.go`

### Implementation for User Story 1

- [x] T016 [US1] Call facade traversal enrichment after walk fetch when `--with-incidents` is set in `cmd/walk_processinstance.go`
- [x] T017 [US1] Add enriched path renderer for indented `incident <incident-key>:` message lines in `cmd/cmd_views_walk_incidents.go`
- [x] T018 [US1] Wire parent mode human output to enriched path rendering in `cmd/walk_processinstance.go`
- [x] T019 [US1] Wire children mode human output to enriched path rendering in `cmd/walk_processinstance.go`
- [x] T020 [US1] Wire family mode human output to enriched path rendering in `cmd/walk_processinstance.go`

**Checkpoint**: User Story 1 is independently functional and testable.

---

## Phase 4: User Story 2 - Consume Walk Incident Details in JSON (Priority: P2)

**Goal**: `c8volt walk pi --key <key> --with-incidents --json` returns stable traversal JSON with per-item incident details.

**Independent Test**: Run JSON command tests and verify traversal metadata remains present while incident details attach to matching walked process instances.

### Tests for User Story 2

- [x] T021 [P] [US2] Add JSON command test for one walked item with incident details in `cmd/walk_test.go`
- [x] T022 [P] [US2] Add JSON command test for multiple walked items with per-key incident association in `cmd/walk_test.go`
- [x] T023 [P] [US2] Add JSON command test for an empty incidents collection in `cmd/walk_test.go`
- [x] T024 [P] [US2] Add JSON command test preserving traversal metadata with `--with-incidents` in `cmd/walk_test.go`

### Implementation for User Story 2

- [x] T025 [US2] Add enriched traversal JSON payload builder in `cmd/cmd_views_walk_incidents.go`
- [x] T026 [US2] Ensure enriched JSON output preserves existing shared envelope behavior in `cmd/cmd_views_walk_incidents.go`
- [x] T027 [US2] Ensure empty incident results render as an empty collection when enrichment was requested in `c8volt/process/client.go`
- [x] T028 [US2] Wire JSON mode to enriched traversal payload when `--with-incidents` is set in `cmd/walk_processinstance.go`

**Checkpoint**: User Stories 1 and 2 both work independently.

---

## Phase 5: User Story 3 - Preserve Walk Traversal Semantics (Priority: P3)

**Goal**: The new flag is additive and does not change default traversal output, ordering, tree rendering, warnings, or tenant boundaries; requested enrichment fails all-or-nothing when incident lookup fails.

**Independent Test**: Run default-output and traversal regression tests with the new flag omitted, plus explicit tests for tree output, warnings, key-only rejection, and incident lookup failure with enrichment enabled.

### Tests for User Story 3

- [x] T029 [P] [US3] Add regression test proving default children human output is unchanged without `--with-incidents` in `cmd/walk_test.go`
- [x] T030 [P] [US3] Add regression test proving default walk JSON output is unchanged without `--with-incidents` in `cmd/walk_test.go`
- [x] T031 [P] [US3] Add regression test preserving family tree layout when `--with-incidents` is omitted in `cmd/walk_test.go`
- [x] T032 [P] [US3] Add enriched tree-output test showing incident lines under the matching tree node in `cmd/walk_test.go`
- [x] T033 [P] [US3] Add partial traversal warning test with `--with-incidents` in `cmd/walk_test.go`
- [x] T034 [P] [US3] Add key-only combination rejection test in `cmd/walk_test.go`
- [x] T035 [P] [US3] Add facade test proving incident lookup failure returns an error instead of an enriched traversal in `c8volt/process/client_test.go`
- [x] T036 [P] [US3] Add command test proving incident lookup failure exits without rendering partial traversal output in `cmd/walk_test.go`

### Implementation for User Story 3

- [x] T037 [US3] Keep existing `traversalPayload` path untouched when `--with-incidents` is omitted in `cmd/cmd_views_walk.go`
- [x] T038 [US3] Keep existing `pathView` and `renderFamilyTree` behavior untouched when enrichment is omitted in `cmd/cmd_views_walk.go`
- [x] T039 [US3] Implement enriched tree renderer without changing traversal edges or node ordering in `cmd/cmd_views_walk_incidents.go`
- [x] T040 [US3] Preserve traversal warning printing after enriched parent/family output in `cmd/walk_processinstance.go`
- [x] T041 [US3] Reject `--keys-only --with-incidents` with a clear validation error in `cmd/walk_processinstance.go`
- [x] T042 [US3] Propagate incident lookup errors from traversal enrichment in `c8volt/process/client.go`
- [x] T043 [US3] Ensure `cmd/walk_processinstance.go` handles enrichment errors before any human or JSON traversal rendering occurs

**Checkpoint**: User Stories 1, 2, and 3 are independently functional.

---

## Phase 6: User Story 4 - Respect Tenant and Version Boundaries (Priority: P4)

**Goal**: Supported versions use tenant-aware incident search; v8.7 fails explicitly rather than falling back to tenant-unsafe behavior.

**Independent Test**: Run facade/service/request-construction tests and command tests that prove tenant filtering and unsupported-version behavior flow through walk enrichment.

### Tests for User Story 4

- [x] T044 [P] [US4] Add facade test passing configured options through walk incident enrichment in `c8volt/process/client_test.go`
- [x] T045 [P] [US4] Add command test proving tenant option reaches incident enrichment during walk in `cmd/walk_test.go`
- [x] T046 [P] [US4] Add v8.8 tenant-filter request assertion for reused incident search behavior in `internal/services/processinstance/v88/service_test.go`
- [x] T047 [P] [US4] Add v8.9 tenant-filter request assertion for reused incident search behavior in `internal/services/processinstance/v89/service_test.go`
- [x] T048 [P] [US4] Add command unsupported-version test for `--with-incidents` on Camunda 8.7 walk in `cmd/walk_test.go`

### Implementation for User Story 4

- [x] T049 [US4] Ensure walk enrichment uses existing facade options from `collectOptions()` in `cmd/walk_processinstance.go`
- [x] T050 [US4] Ensure v8.7 unsupported incident lookup propagates through walk command handlers in `c8volt/process/client.go` and `cmd/walk_processinstance.go`
- [x] T051 [US4] Confirm v8.8 and v8.9 incident request bodies remain process-instance-key scoped without redundant rejected filters in `internal/services/processinstance/v88/incidents.go` and `internal/services/processinstance/v89/incidents.go`

**Checkpoint**: All user stories are independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, formatting, and validation across the completed feature.

- [x] T052 [P] Update command help examples and flag description for `--with-incidents` in `cmd/walk_processinstance.go`
- [x] T053 Regenerate CLI documentation with `make docs-content` and verify `docs/cli/c8volt_walk_process-instance.md`
- [x] T054 [P] Review README walk examples and update `README.md` only if the new flag belongs in existing examples
- [x] T055 [P] Run `gofmt` on changed Go files in `cmd/`, `c8volt/process/`, and `internal/services/processinstance/`
- [x] T056 Run targeted validation with `go test ./cmd ./c8volt/process ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1`
- [x] T057 Run full repository validation with `make test` from repository root `.`
- [x] T058 Confirm `specs/157-walk-pi-incidents/quickstart.md` scenarios match final command behavior

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational and is the MVP.
- **User Story 2 (Phase 4)**: Depends on enriched traversal data being available from User Story 1.
- **User Story 3 (Phase 5)**: Depends on render branching existing so regressions can prove it remains opt-in and all-or-nothing.
- **User Story 4 (Phase 6)**: Depends on the enrichment path from User Story 1.
- **Polish (Phase 7)**: Depends on all desired user stories.

### User Story Dependencies

- **User Story 1 (P1)**: No dependency on later stories after Foundational.
- **User Story 2 (P2)**: Depends on US1 incident lookup and enrichment.
- **User Story 3 (P3)**: Can run after US1/US2 render branching exists.
- **User Story 4 (P4)**: Can be implemented alongside US1 service/facade work but must finish before release.

### Parallel Opportunities

- Setup tasks T001-T004 can run in parallel.
- Foundational tests T010-T011 can be written alongside T005-T009 once model names are settled.
- Command human-output tests T012-T014 can run in parallel.
- JSON-output tests T021-T024 can run in parallel.
- Traversal regression and failure tests T029-T036 can run in parallel.
- Tenant/version tests T044-T048 can run in parallel across command, facade, and service packages.
- README review and gofmt can run in parallel after implementation stabilizes.

---

## Parallel Example: User Story 1

```text
Task: "Add command human-output test for one walked process instance with one incident in cmd/walk_test.go"
Task: "Add command human-output test for multiple walked instances with incidents in cmd/walk_test.go"
Task: "Add facade test proving incident lookups run only for traversal result keys in c8volt/process/client_test.go"
```

---

## Parallel Example: User Story 3

```text
Task: "Add key-only combination rejection test in cmd/walk_test.go"
Task: "Add facade test proving incident lookup failure returns an error instead of an enriched traversal in c8volt/process/client_test.go"
Task: "Add command test proving incident lookup failure exits without rendering partial traversal output in cmd/walk_test.go"
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete User Story 1 only.
3. Validate with targeted facade and command tests.
4. Stop if needed with a usable human-readable `walk pi --key --with-incidents` flow.

### Incremental Delivery

1. Add human-readable keyed walk enrichment.
2. Add JSON walk enrichment.
3. Lock down traversal/default-output preservation and failure propagation.
4. Complete tenant/version safeguards.
5. Update docs and run full validation.

### Commit Guidance

Every commit subject for this feature must use Conventional Commits and end with `#157`, for example `feat(walk-pi): add keyed walk incident enrichment #157`.
