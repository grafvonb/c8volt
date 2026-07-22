# Tasks: Walk PI Elements

**Input**: Design documents from `/specs/251-walk-pi-elements/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/walk-pi-elements-cli.md, quickstart.md

**Tests**: Required by the c8volt constitution and quickstart validation guidance. Write/adjust targeted tests before implementation where each task says so.

**Organization**: Tasks are grouped by user story so each story can be implemented and validated as an independent increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with other marked tasks in the same phase because it touches different files or only reads context
- **[Story]**: User story label for story phases only
- Each task names exact repository paths

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish the current behavior and implementation references before changing command behavior.

- [x] T001 Review existing `get pi --with-elements` behavior and helpers in `cmd/get_processinstance.go`, `cmd/get_processinstance_enrichment.go`, and `cmd/cmd_views_processinstance_activity.go`
- [x] T002 Review existing walk enrichment behavior and tests in `cmd/walk_processinstance.go`, `cmd/cmd_views_walk_incidents.go`, and `cmd/walk_test.go`
- [x] T003 [P] Review public and internal element enrichment contracts in `c8volt/process/api.go`, `c8volt/process/model.go`, `c8volt/process/client.go`, and `internal/services/processinstance/enrichment.go`
- [x] T004 [P] Review command metadata expectations in `cmd/command_contract.go` and `cmd/command_contract_test.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Prepare shared walk activity plumbing that all story slices depend on.

**CRITICAL**: No user story work can begin until this phase is complete.

- [x] T005 Add a walk-specific element enrichment flag variable and reset coverage location in `cmd/walk_processinstance.go` and `cmd/walk_test.go`
- [x] T006 Extend `activityItemsFromTraversal` to accept element-enriched process instances and populate `processInstanceActivityItem.Elements` in `cmd/cmd_views_processinstance_activity.go`
- [x] T007 Route `activityPathView`, `renderActivityFamilyTree`, `formatMustActivityLinesWithTimezone`, and `writeProcessInstanceActivityLinesWithTimezone` through element-aware formatting in `cmd/cmd_views_walk_incidents.go`
- [x] T008 Run `gofmt` on `cmd/walk_processinstance.go`, `cmd/cmd_views_processinstance_activity.go`, and `cmd/cmd_views_walk_incidents.go`

**Checkpoint**: Walk activity items can carry elements and render element detail sections without yet exposing the full user-facing contract.

---

## Phase 3: User Story 1 - Inspect Runtime Elements During Process Walk (Priority: P1) MVP

**Goal**: Operators can run default family `walk pi --key <key> --with-elements` and see runtime element rows under the owning process-instance rows.

**Independent Test**: Run a family walk with `--with-elements` and verify that each walked process instance keeps its traversal position while element rows appear only below their owner.

### Tests for User Story 1

- [ ] T009 [P] [US1] Add failing command capability assertions for `walk pi --with-elements` in `cmd/command_contract_test.go`
- [ ] T010 [US1] Add failing help assertions and family human output test with root and child process instances plus `elements:` sections in `cmd/walk_test.go`
- [ ] T011 [US1] Add failing empty-elements ownership test ensuring rows with no elements stay visible without placeholder element rows in `cmd/walk_test.go`
- [ ] T012 [US1] Add failing element incident marker rendering coverage for walked element rows in `cmd/walk_test.go`

### Implementation for User Story 1

- [ ] T013 [US1] Register `--with-elements`, update long help and examples, and expose command capability metadata in `cmd/walk_processinstance.go` and `cmd/command_contract.go`
- [ ] T014 [US1] Invoke `EnrichProcessInstancesWithElements` after traversal using existing admin input options and element activity helpers in `cmd/walk_processinstance.go`
- [ ] T015 [US1] Merge element enrichments into walked activity items while preserving traversal key order and owner mapping in `cmd/cmd_views_processinstance_activity.go`
- [ ] T016 [US1] Render default family tree `elements:` sections without nesting child process instances under detail sections in `cmd/cmd_views_walk_incidents.go`
- [ ] T017 [US1] Run targeted US1 tests covering `cmd/walk_test.go` and `cmd/command_contract_test.go` with `go test ./cmd -run 'TestWalkHelp|TestCommandCapabilityForCommand|TestWalkProcessInstanceCommand_.*Elements' -count=1`

**Checkpoint**: User Story 1 is functional and testable as the MVP.

---

## Phase 4: User Story 2 - Preserve Traversal Modes With Elements (Priority: P2)

**Goal**: Operators can combine `--with-elements` with children, parent, and flat walk modes without changing selected process instances or path readability.

**Independent Test**: Run children, parent, and flat walks with and without `--with-elements`; selected keys and ordering must match, and enriched output must remain readable.

### Tests for User Story 2

- [ ] T018 [US2] Add failing `--children --with-elements` human output test preserving descendant selection and owner-specific elements in `cmd/walk_test.go`
- [ ] T019 [US2] Add failing `--parent --with-elements` human output test preserving ancestry order and owner-specific elements in `cmd/walk_test.go`
- [ ] T020 [US2] Add failing `--flat --with-elements` human output test preserving flat separators and element sections in `cmd/walk_test.go`
- [ ] T021 [US2] Add failing unchanged-default regression test proving no element lookup runs without `--with-elements` in `cmd/walk_test.go`

### Implementation for User Story 2

- [ ] T022 [P] [US2] Ensure children and parent mode enrichment uses `processInstancesFromTraversal` order in `cmd/walk_processinstance.go`
- [ ] T023 [P] [US2] Ensure flat family mode preserves path separators while writing element detail sections in `cmd/cmd_views_walk_incidents.go`
- [ ] T024 [US2] Preserve traversal warnings and missing-ancestor warning rendering when elements are requested in `cmd/walk_processinstance.go` and `cmd/cmd_views_walk_incidents.go`
- [ ] T025 [US2] Run targeted US2 tests covering `cmd/walk_test.go` with `go test ./cmd -run 'TestWalkProcessInstanceCommand_.*WithElements|TestWalkProcessInstanceCommand_Default.*Without' -count=1`

**Checkpoint**: User Stories 1 and 2 both work independently.

---

## Phase 5: User Story 3 - Use Elements In Scripted Output Safely (Priority: P3)

**Goal**: Script authors get stable JSON output with per-item elements, combined enrichments, and clear validation failures for invalid output modes or unsupported Camunda versions.

**Independent Test**: Run JSON and invalid flag combinations; valid JSON preserves traversal metadata and invalid requests fail before remote enrichment.

### Tests for User Story 3

- [ ] T026 [US3] Add failing JSON output test for `walk pi --key <key> --with-elements` preserving traversal metadata and per-item `elements` in `cmd/walk_test.go`
- [ ] T027 [US3] Add failing combined `--with-vars --with-incidents --with-elements` human and JSON output tests preserving section order and arrays in `cmd/walk_test.go`
- [ ] T028 [US3] Add failing `--keys-only --with-elements` validation test proving element lookup is not called in `cmd/walk_test.go`
- [ ] T029 [US3] Add failing Camunda 8.7 unsupported element-search test for `walk pi --with-elements` in `cmd/walk_test.go`
- [ ] T030 [US3] Add failing element lookup failure test proving no partial success output is rendered in `cmd/walk_test.go`

### Implementation for User Story 3

- [ ] T031 [US3] Add `validateWalkPIWithElementsUsage` for `--key` and `--keys-only` constraints in `cmd/walk_processinstance.go`
- [ ] T032 [US3] Use `activityTraversalPayload` for JSON whenever variables, incidents, or elements are combined in `cmd/walk_processinstance.go` and `cmd/cmd_views_walk_incidents.go`
- [ ] T033 [US3] Wrap element enrichment failures with command context and stop before rendering any success output in `cmd/walk_processinstance.go`
- [ ] T034 [US3] Preserve Camunda 8.7 unsupported capability propagation from existing element enrichment in `cmd/walk_processinstance.go`
- [ ] T035 [US3] Run targeted US3 tests covering `cmd/walk_test.go` with `go test ./cmd -run 'TestWalkProcessInstanceCommand_.*Elements|TestWalkProcessInstanceCommand_RejectsKeysOnlyWithElements|TestWalkProcessInstanceCommand_WithElementsUnsupportedV87' -count=1`

**Checkpoint**: All user stories are independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish documentation, generated references, formatting, and validation across the feature.

- [ ] T036 [P] Update walk process-instance guidance and examples in `README.md`
- [ ] T037 Update generated CLI documentation by running `make docs-content` and reviewing `docs/cli/c8volt_walk_process-instance.md`, `docs/cli/c8volt_walk.md`, and `docs/index.md`
- [ ] T038 Run `gofmt` on all touched Go files in `cmd/`
- [ ] T039 Run targeted command validation covering `cmd/walk_test.go` and `cmd/command_contract_test.go` with `go test ./cmd -run 'TestWalkProcessInstanceCommand|TestWalkHelp|TestCommandCapabilityForCommand' -count=1`
- [ ] T040 Run full repository validation through `Makefile` with `make test`
- [ ] T041 Update task checkboxes in `specs/251-walk-pi-elements/tasks.md` as work completes

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational; delivers the MVP.
- **User Story 2 (Phase 4)**: Depends on Foundational and benefits from US1 rendering plumbing; can be tested independently once the flag exists.
- **User Story 3 (Phase 5)**: Depends on Foundational and JSON/activity plumbing from US1; validates script-safe behavior.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **US1**: Starts after Phase 2; no dependency on US2 or US3.
- **US2**: Starts after Phase 2; uses the same element enrichment path created for US1.
- **US3**: Starts after Phase 2; depends on activity payload and validation hooks touched by US1.

### Within Each User Story

- Write failing tests before implementation tasks in the same story.
- Register flags and validation before relying on command behavior in manual validation.
- Merge/enrich data before rendering human or JSON output.
- Run each story's targeted test command before advancing to the next story.

---

## Parallel Opportunities

- T003 and T004 can run in parallel during setup.
- Within US1, T009 can run in parallel with T010-T012 because it touches `cmd/command_contract_test.go` while the other tests focus on `cmd/walk_test.go`.
- Within US2, T022 and T023 can run in parallel after tests are written because they split orchestration and flat rendering across `cmd/walk_processinstance.go` and `cmd/cmd_views_walk_incidents.go`.
- Documentation task T036 can run in parallel with late validation once command behavior and wording are settled.
- Separate contributors can draft US2 and US3 tests in `cmd/walk_test.go` only with coordination because they touch the same file; do not mark them parallel in the checklist.

---

## Parallel Example: User Story 1

```bash
Task: "Add failing command capability assertions for `walk pi --with-elements` in `cmd/command_contract_test.go`"
Task: "Add failing help assertions and family human output test with root and child process instances plus `elements:` sections in `cmd/walk_test.go`"
```

---

## Parallel Example: User Story 2

```bash
Task: "Ensure children and parent mode enrichment uses `processInstancesFromTraversal` order in `cmd/walk_processinstance.go`"
Task: "Ensure flat family mode preserves path separators while writing element detail sections in `cmd/cmd_views_walk_incidents.go`"
```

---

## Parallel Example: User Story 3

US3 intentionally has no intra-story `[P]` tasks because validation, JSON routing, failure handling, and unsupported-version propagation all converge on `cmd/walk_processinstance.go` and must be sequenced after the failing tests in `cmd/walk_test.go`.

---

## Parallel Example: Polish

```bash
Task: "Update walk process-instance guidance and examples in `README.md`"
Task: "Run targeted command validation with `go test ./cmd -run 'TestWalkProcessInstanceCommand|TestWalkHelp|TestCommandCapabilityForCommand' -count=1`"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 and Phase 2.
2. Add tests T009-T012 and confirm they fail.
3. Implement T013-T016.
4. Run T017 and manually inspect `walk pi --key <key> --with-elements`.
5. Stop and validate MVP output before expanding traversal modes.

### Incremental Delivery

1. Complete Setup and Foundational work.
2. Deliver US1 for default family diagnostics.
3. Deliver US2 for parent, children, and flat traversal variants.
4. Deliver US3 for JSON, invalid combinations, unsupported versions, and failure behavior.
5. Finish docs, generated references, and full validation.

### Validation Discipline

- Use targeted `go test ./cmd -run ... -count=1` after each story.
- Run `make docs-content` after command help or examples change.
- Run `make test` before commit or merge.

## Notes

- `[P]` tasks are parallelizable because they avoid write conflicts with other same-phase tasks.
- `[US1]`, `[US2]`, and `[US3]` labels map directly to the prioritized user stories in `spec.md`.
- Keep command-layer code version-neutral; do not call generated Camunda clients or version-specific services directly from `cmd/`.
- Do not add element filters, listener enrichment, job enrichment, metrics enrichment, traversal-selection changes, parent/child discovery changes, or missing-ancestor behavior changes.
