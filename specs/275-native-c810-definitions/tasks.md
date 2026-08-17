# Tasks: Native Camunda 8.10 Embedded Process Definitions

**Input**: Design documents from `specs/275-native-c810-definitions/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/embedded-definition-selection.md`, `quickstart.md`

**Tests**: Automated verification is required by FR-013 and FR-014. Add focused tests at the embedded-resource, shared-selector, smoke-service, and one CLI-output boundary.

**Organization**: Tasks are grouped by user story. The shared C810 resource family is created once in the foundational phase because both native selection and behavioral-parity stories depend on it.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files and has no dependency on another incomplete task in the same phase
- **[Story]**: Maps the task to User Story 1, 2, or 3
- Every task names the exact repository path it affects or validates

## Phase 1: Setup (Shared Baseline)

**Purpose**: Confirm the immutable pre-feature boundary before adding resources.

- [x] T001 Verify commit `f2658425` as the #275 fork baseline and confirm no pre-existing changes under `embedded/processdefinitions/C87_*.bpmn`, `embedded/processdefinitions/C88_*.bpmn`, `embedded/processdefinitions/C89_*.bpmn`, `integration/`, or `Makefile`

---

## Phase 2: Foundational (Shared C810 Resource Family)

**Purpose**: Add the resource family required by native selection and parity verification.

**⚠️ CRITICAL**: User-story work starts only after all eight resources exist.

- [x] T002 Create the eight C89-derived files `embedded/processdefinitions/C810_DoubleUserTask.bpmn`, `embedded/processdefinitions/C810_MultipleSubProcessesParent.bpmn`, `embedded/processdefinitions/C810_NoOpCompletion.bpmn`, `embedded/processdefinitions/C810_SimpleParent.bpmn`, `embedded/processdefinitions/C810_SimpleParentWithIncidentSubprocess.bpmn`, `embedded/processdefinitions/C810_SimpleServiceTask.bpmn`, `embedded/processdefinitions/C810_SimpleUserTask.bpmn`, and `embedded/processdefinitions/C810_SimpleUserTaskWithIncident.bpmn`; change only C89-to-C810 identity references and `8.9.0` to `8.10.0`, preserve each source `exporterVersion`, and leave all C87/C88/C89 files untouched

**Checkpoint**: The complete C810 family is available to the existing embedded filesystem.

---

## Phase 3: User Story 1 - Use Native 8.10 Embedded Definitions (Priority: P1) 🎯 MVP

**Goal**: Camunda 8.10 listing, `--all` export/deploy, and smoke selection use C810 resources instead of the temporary C89 fallback.

**Independent Test**: Configure version 8.10, assert the shared embedded selector returns exactly the eight C810 files, and assert smoke selection returns the C810 parent and its C810 deployment closure.

### Tests for User Story 1

- [x] T003 [P] [US1] Update the V810 expectation to `C810_` while retaining V87/V88/V89 mappings and unknown-version rejection in `toolx/fixture_compatibility_test.go`
- [x] T004 [P] [US1] Replace the V810-to-C89 embed-list case with an exact eight-file C810 family assertion and explicit C89 exclusion in `cmd/embed_test.go`
- [x] T005 [P] [US1] Update V810 fixture selection and add the three-resource C810 smoke deployment-closure assertion in `internal/services/ops/smoke_test_test.go`
- [x] T006 [P] [US1] Add one V810 smoke command case whose unchanged output schema reports the C810 fixture identity in `cmd/ops_execute_smoke_test_test.go`

### Implementation for User Story 1

- [x] T007 [US1] Change only the V810 case of `toolx.ProductionFixturePrefix` from `C89_` to `C810_` in `toolx/fixture_compatibility.go`; keep stable mappings and unknown-version failure unchanged
- [x] T008 [US1] Run the US1-focused tests for `toolx/fixture_compatibility_test.go`, `cmd/embed_test.go`, `internal/services/ops/smoke_test_test.go`, and `cmd/ops_execute_smoke_test_test.go`, confirming the existing consumers require no production special cases

**Checkpoint**: V810 uses C810 resources through the existing shared selector, while V89 still uses C89.

---

## Phase 4: User Story 2 - Preserve the Known Fixture Workflows (Priority: P1)

**Goal**: Prove every C810 resource is the corresponding C89 workflow with only the permitted identity and platform-version differences.

**Independent Test**: Normalize C810 identity and `8.10.0` back to the C89 values, then compare all eight files exactly and validate their inventory, XML well-formedness, version tags, ownership references, and called-process closure.

### Verification for User Story 2

- [x] T009 [US2] Add `embedded/fs_test.go` coverage that enumerates exactly eight C810 files, parses each as well-formed XML, checks C810 filenames/process names/process IDs/BPMNPlane ownership/platform version/version tag, rejects residual C89 references, resolves all called processes within the family, and proves normalized equality with the matching C89 source
- [x] T010 [US2] Run `go test ./embedded -run 'TestC810ProductionDefinitions' -count=1` against `embedded/fs_test.go` and resolve only C810 identity or parity discrepancies in `embedded/processdefinitions/C810_*.bpmn`

**Checkpoint**: All eight C810 definitions have automated structural and behavioral parity proof.

---

## Phase 5: User Story 3 - Preserve Existing Compatibility Lines (Priority: P1)

**Goal**: Preserve stable fixture selection and bytes, reject unknown versions, keep integration assets unchanged, and remove the superseded fallback from active #273 design guidance.

**Independent Test**: Stable-prefix and unknown-version cases pass; C87/C88/C89 and integration diffs against `f2658425` are empty; active #273 design artifacts describe C810; historical checked tasks are retained with a supersession note.

### Implementation and Verification for User Story 3

- [ ] T011 [P] [US3] Replace active V810-to-C89 fixture reuse with native C810 selection in `specs/273-camunda-v810-support/spec.md`, `specs/273-camunda-v810-support/plan.md`, `specs/273-camunda-v810-support/research.md`, `specs/273-camunda-v810-support/data-model.md`, `specs/273-camunda-v810-support/quickstart.md`, and `specs/273-camunda-v810-support/contracts/service-compatibility.md`
- [ ] T012 [US3] Add a #275 supersession note to `specs/273-camunda-v810-support/tasks.md` without rewriting the checked T039/T042 descriptions or changing `specs/273-camunda-v810-support/progress.md` and `specs/273-camunda-v810-support/ralph-memory.md`
- [ ] T013 [US3] Run stable and unknown mapping cases from `toolx/fixture_compatibility_test.go`, retain V89 selection assertions in `cmd/embed_test.go` and `internal/services/ops/smoke_test_test.go`, and verify empty diffs from `f2658425` for `embedded/processdefinitions/C87_*.bpmn`, `embedded/processdefinitions/C88_*.bpmn`, `embedded/processdefinitions/C89_*.bpmn`, `integration/`, and `Makefile`

**Checkpoint**: Stable runtime behavior and historical records are protected, while all active #273 guidance points to native C810 resources.

---

## Phase 6: Polish & Cross-Cutting Validation

**Purpose**: Execute the complete delivery gate and perform the final scope audit.

- [ ] T014 Run every command in `specs/275-native-c810-definitions/quickstart.md` and reconcile any mismatch in that guide with the implemented test names and outcomes
- [ ] T015 Run the repository race-enabled delivery gate from `Makefile` with `make test` and resolve all regressions within the #275 scope
- [ ] T016 Run `git diff --check` and review the complete diff for `embedded/processdefinitions/C810_*.bpmn`, `embedded/fs_test.go`, `toolx/fixture_compatibility.go`, `toolx/fixture_compatibility_test.go`, `cmd/embed_test.go`, `cmd/ops_execute_smoke_test_test.go`, `internal/services/ops/smoke_test_test.go`, and the listed #273/#275 artifacts; confirm no generated CLI documentation, new dependency, generator, stable BPMN, or integration asset changed

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: Starts immediately.
- **Phase 2 (Foundation)**: Depends on T001 and blocks all user stories.
- **Phase 3 (US1)**: Depends on T002. T003-T006 can run in parallel; T007 follows the failing selection tests; T008 completes the story.
- **Phase 4 (US2)**: Depends on T002 and may run in parallel with US1 because it changes only `embedded/fs_test.go` and validates the shared resources.
- **Phase 5 (US3)**: T011 can run after T001 in parallel with US1/US2. T012 follows T011; T013 follows US1 because it validates the final mapping.
- **Phase 6 (Polish)**: Depends on US1, US2, and US3 completion.

### User Story Dependencies

- **US1 (P1)**: Depends only on the shared C810 resources from T002.
- **US2 (P1)**: Depends only on the shared C810 resources from T002 and is independently testable through `embedded/fs_test.go`.
- **US3 (P1)**: Documentation correction can start independently; its final regression proof depends on US1's mapping change.

### Within Each User Story

- Write the focused selection tests before T007 and confirm their V810 expectations fail against the temporary C89 mapping.
- Keep asset-parity verification local to `embedded/fs_test.go`; do not add a generator or external schema dependency.
- Preserve the existing production consumers in `cmd/embed_files.go` and `internal/services/ops/smoke_test_service.go` unless a focused test exposes a real defect.
- Run the story checkpoint before proceeding to the final delivery gate.

### Parallel Opportunities

- T003, T004, T005, and T006 touch different test files and can run concurrently.
- US2 verification in T009 can run alongside US1 selection work after T002.
- T011 touches only #273 design artifacts and can run alongside code/test work.
- No BPMN creation tasks are marked parallel because cross-process references must be updated and reviewed as one coherent family.

---

## Parallel Example: User Story 1

```text
Task T003: Update V810/stable/unknown prefix expectations in toolx/fixture_compatibility_test.go
Task T004: Assert the exact V810 embedded family in cmd/embed_test.go
Task T005: Assert V810 smoke fixture and deployment closure in internal/services/ops/smoke_test_test.go
Task T006: Assert one V810 CLI smoke identity in cmd/ops_execute_smoke_test_test.go
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Complete T001-T002 to establish the protected baseline and C810 resource family.
2. Complete T003-T008 to replace the fallback through the one shared selector.
3. Stop and validate the US1 checkpoint: embed and smoke selection use C810 and retain V89-to-C89 behavior.

### Incremental Delivery

1. **Foundation**: Add the coherent eight-file C810 family.
2. **US1**: Switch native selection and prove the operator-visible fixture identity.
3. **US2**: Add exact normalized parity verification for every file.
4. **US3**: Protect stable boundaries and correct #273's active design guidance without rewriting history.
5. **Polish**: Run the quickstart, full race-enabled tests, and final scope audit.

## Notes

- Generic README and generated CLI documentation remain accurate because they do not name a version-specific fixture; no docs regeneration is required.
- `CamundaVersion.FilePrefix()` remains unchanged so live integration selection stays outside #275.
- `specs/273-camunda-v810-support/progress.md` and `ralph-memory.md` remain historical records.
- Commit after each task or cohesive task group using Conventional Commits and reference #275 where applicable.
