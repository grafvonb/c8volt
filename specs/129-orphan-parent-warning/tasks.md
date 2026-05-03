# Tasks: Graceful Orphan Parent Traversal

**Input**: Design documents from `/specs/129-orphan-parent-warning/`
**Prerequisites**: [plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/plan.md) (required), [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/spec.md) (required for user stories), [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/research.md), [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/data-model.md), [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/quickstart.md), [contracts/orphan-parent-traversal.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/contracts/orphan-parent-traversal.md)

**Tests**: Automated tests are REQUIRED for this feature because the specification explicitly requires regression coverage for partial traversal behavior, strict direct-lookup non-regression, unchanged waiter semantics, and support across `v87`, `v88`, and `v89`.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Lock in the current orphan-parent failure chain, affected command flows, and regression seams before shared implementation begins.

- [x] T001 Inventory the current orphan-parent failure path across /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/dryrun.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go
- [x] T002 [P] Confirm current strict lookup and waiter boundaries in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/common/response.go
- [x] T003 [P] Confirm shared version support and traversal delegation across /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared partial-result contract and common integration seams that every story depends on.

**⚠️ CRITICAL**: No user story work should begin until this phase is complete.

- [x] T004 Define the authoritative orphan-parent warning, success, and failure contract in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/plan.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/research.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/contracts/orphan-parent-traversal.md
- [x] T005 Refactor the shared traversal API shape to represent partial results, missing ancestors, and unresolved outcomes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/api.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/api.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/walker.go
- [x] T006 [P] Update the feature data model and quickstart guidance for the finalized traversal result contract in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/data-model.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/quickstart.md
- [x] T007 [P] Add foundational facade and helper seams for structured partial traversal handling in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/common/response.go

**Checkpoint**: The repository has one shared traversal contract, shared APIs can represent partial vs unresolved outcomes, and downstream flows can build on that contract consistently.

---

## Phase 3: User Story 1 - Inspect Partial Trees Safely (Priority: P1) 🎯 MVP

**Goal**: Let operators inspect partial ancestry and family trees when a non-start ancestor is missing instead of failing the whole walk command.

**Independent Test**: Run `walk pi --parent`, `walk pi`, and `walk pi --flat` against an orphan-parent scenario and verify the commands render resolved data plus warnings when actionable results exist, while fully unresolved traversal still fails.

### Tests for User Story 1

- [x] T008 [P] [US1] Add shared walker regression tests for partial ancestry, partial family traversal, and fully unresolved failure behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker_test.go
- [x] T009 [P] [US1] Add version-aware traversal regression coverage for `v87`, `v88`, and `v89` services in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service_test.go
- [x] T010 [P] [US1] Add command rendering regressions for partial parent/family/tree output in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go

### Implementation for User Story 1

- [x] T011 [US1] Implement shared partial ancestry and family result handling in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker.go
- [x] T012 [US1] Thread the new traversal result contract through the process facade in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/walker.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go
- [x] T013 [US1] Render partial walk output and warnings for parent/family/tree modes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_processinstance.go and related view helpers under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/

**Checkpoint**: User Story 1 is independently testable: walk commands no longer fail hard on orphan parents when resolved traversal data exists.

---

## Phase 4: User Story 2 - Keep Orphan Children Actionable (Priority: P2)

**Goal**: Let cancel/delete preflight and indirect cleanup continue with resolved family keys when an ancestor is missing, while exposing missing ancestor keys through one shared contract.

**Independent Test**: Run keyed and paged cancel/delete preflight on orphan-child scenarios and verify resolved keys remain actionable, missing ancestors are surfaced structurally, and fully unresolved expansion still fails.

### Tests for User Story 2

- [x] T014 [P] [US2] Add dry-run expansion regressions for resolved roots, collected keys, and missing ancestors in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go
- [x] T015 [P] [US2] Add command preflight regressions for keyed and paged cancel/delete orphan-child scenarios in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go
- [x] T016 [P] [US2] Add indirect cleanup regression coverage for process-resource expansion paths in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/client_test.go or the closest existing resource regression seam under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/

### Implementation for User Story 2

- [x] T017 [US2] Update dependency-expansion dry-run behavior to return actionable roots/collected keys plus missing ancestors in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/dryrun.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go
- [x] T018 [US2] Consume the shared preflight warning contract in cancel/delete keyed and paged flows in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go
- [x] T019 [US2] Align indirect process-definition cleanup with the same preflight contract in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/client.go and related callers under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/

**Checkpoint**: User Story 2 is independently testable: destructive preflight stays actionable for orphan children without hiding unresolved ancestor boundaries.

---

## Phase 5: User Story 3 - Preserve Strict Single-Resource Semantics (Priority: P3)

**Goal**: Keep direct single-resource lookup and absent/deleted waiter behavior unchanged while the new traversal contract lands around them.

**Independent Test**: Verify direct `get process-instance --key` still returns the normal strict error when the target is missing and absent/deleted wait flows behave exactly as before after the traversal refactor.

### Tests for User Story 3

- [x] T020 [P] [US3] Add strict direct-lookup non-regression coverage in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service_test.go
- [x] T021 [P] [US3] Add waiter non-regression coverage for absent/deleted semantics in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter_test.go
- [x] T022 [P] [US3] Add command-level non-regression coverage for strict lookup vs traversal behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go

### Implementation for User Story 3

- [x] T023 [US3] Keep direct lookup/state-check and waiter boundaries isolated from traversal changes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/common/response.go
- [x] T024 [US3] Refresh operator-facing wording only where traversal/preflight output changes while preserving strict single-resource guidance in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go, and generated CLI help sources under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/

**Checkpoint**: User Story 3 is independently testable: the new traversal behavior is in place without weakening direct lookup or waiter contracts.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish validation, keep planning artifacts synchronized, and leave the feature ready for implementation handoff or execution.

- [x] T025 [P] Refresh final implementation and verification notes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/plan.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/quickstart.md
- [x] T026 Run focused validation with `go test ./internal/services/processinstance/... -count=1`, `go test ./c8volt/process -count=1`, and `go test ./cmd -count=1` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt
- [x] T027 Run repository validation with `make test` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile
- [x] T028 [P] Regenerate affected CLI docs with `make docs-content` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt if README or help output changed

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion and is the MVP slice.
- **User Story 2 (Phase 4)**: Depends on User Story 1 because dry-run and preflight behavior consume the shared traversal contract established there.
- **User Story 3 (Phase 5)**: Depends on User Stories 1 and 2 because strict non-regression should validate the final traversal/preflight design, not an intermediate state.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: No dependency on later stories after Foundational work is complete.
- **User Story 2 (P2)**: Depends on the shared partial traversal contract from User Story 1.
- **User Story 3 (P3)**: Depends on the settled traversal/preflight behavior from User Stories 1 and 2.

### Within Each User Story

- Add or update regression tests before considering the story complete.
- Shared helper and API changes before command or facade consumption that depends on them.
- Facade changes before command preflight or rendering cleanup that depends on them.
- Documentation regeneration only after visible behavior is stable.

### Parallel Opportunities

- `T002` and `T003` can run in parallel.
- `T006` and `T007` can run in parallel after the contract is settled.
- `T008`, `T009`, and `T010` can run in parallel.
- `T014`, `T015`, and `T016` can run in parallel.
- `T020`, `T021`, and `T022` can run in parallel.
- `T025` and `T028` can run in parallel with targeted validation once implementation behavior is stable.

---

## Parallel Example: User Story 1

```bash
# Prepare partial traversal coverage in parallel:
Task: "Add shared walker regression tests for partial ancestry, partial family traversal, and fully unresolved failure behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker_test.go"
Task: "Add version-aware traversal regression coverage for `v87`, `v88`, and `v89` services in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service_test.go"
Task: "Add command rendering regressions for partial parent/family/tree output in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go"
```

## Parallel Example: User Story 2

```bash
# Prepare preflight continuation coverage in parallel:
Task: "Add dry-run expansion regressions for resolved roots, collected keys, and missing ancestors in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go"
Task: "Add command preflight regressions for keyed and paged cancel/delete orphan-child scenarios in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go"
Task: "Add indirect cleanup regression coverage for process-resource expansion paths in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/client_test.go or the closest existing resource regression seam under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/"
```

## Parallel Example: User Story 3

```bash
# Prepare strict non-regression coverage in parallel:
Task: "Add strict direct-lookup non-regression coverage in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service_test.go"
Task: "Add waiter non-regression coverage for absent/deleted semantics in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter_test.go"
Task: "Add command-level non-regression coverage for strict lookup vs traversal behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Stop and validate partial walk rendering plus unresolved failure behavior before expanding into destructive preflight flows.

### Incremental Delivery

1. Finish Setup + Foundational once.
2. Deliver User Story 1 as the MVP for partial traversal inspection.
3. Add User Story 2 to keep orphan-child cleanup actionable.
4. Add User Story 3 to protect strict direct lookup and waiter semantics.
5. Finish with targeted validation, optional docs regeneration, and full `make test`.

### Parallel Team Strategy

1. One contributor handles Setup + Foundational work.
2. After Foundational is complete:
   - Contributor A: User Story 1 shared walker and walk rendering.
   - Contributor B: User Story 2 dry-run/preflight and resource cleanup integration.
   - Contributor C: User Story 3 strict non-regression tests and documentation cleanup once behavior stabilizes.
3. Finish with shared validation and repository-wide tests.

---

## Notes

- [P] tasks are limited to work on different files with no dependency on unfinished tasks.
- [US1], [US2], and [US3] map directly to the user stories in [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/spec.md).
- This feature’s commit subjects must keep Conventional Commit formatting and append `#129` as the final token.
- Run `make test` before committing, per repository rules.
