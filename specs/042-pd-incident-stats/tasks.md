# Tasks: Report Process Definition Incident Statistics

**Input**: Design documents from `/specs/042-pd-incident-stats/`
**Prerequisites**: [plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/plan.md) (required), [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/spec.md) (required for user stories), [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/research.md), [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/data-model.md), [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/quickstart.md), [contracts/process-definition-statistics.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/contracts/process-definition-statistics.md)

**Tests**: Automated tests are REQUIRED for this feature because the specification explicitly requires supported-version incident-count coverage, unsupported-version fallback coverage, and protection against regressions in surrounding stats output.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Lock in the exact stats semantics, renderer boundary, and generated-client seams before shared model or service work begins.

- [x] T001 Inventory the current `get process-definition --stat` flow in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processdefinition.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_get.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/research.md
- [x] T002 [P] Confirm the supported-version generated-client seams for incident-bearing process-instance counts in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/clients/camunda/v88/camunda/client.gen.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/clients/camunda/v89/camunda/client.gen.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/research.md
- [x] T003 [P] Confirm the existing process-definition stats model and regression anchors in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/model.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/processdefinition.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v89/service_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared stats contract and model seams that all user stories depend on.

**⚠️ CRITICAL**: No user story work should begin until this phase is complete.

- [x] T004 Define the authoritative incident-statistics contract and version matrix in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/contracts/process-definition-statistics.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/research.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/plan.md
- [x] T005 Extend the shared process-definition statistics model to represent supported incident counts versus unsupported omission in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/processdefinition.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/model.go
- [x] T006 [P] Update public-to-domain conversion and shared client coverage for the refined stats model in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/convert.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go
- [x] T007 [P] Refresh planning artifacts to reflect the finalized model vocabulary and rendering rules in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/data-model.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/quickstart.md

**Checkpoint**: The repository has one shared incident-statistics contract, and supported zero versus unsupported omission can flow through the model without renderer-side version guessing.

---

## Phase 3: User Story 1 - Show Correct Incident Counts (Priority: P1) 🎯 MVP

**Goal**: Make `get pd --stat` show the correct incident-bearing process-instance count on supported versions while keeping the surrounding stats fields unchanged.

**Independent Test**: Run `get pd --stat` on `v8.8` and `v8.9` for definitions with and without affected process instances, and verify the output shows `in:<count>` or `in:0` while `ac`, `cp`, and `cx` keep their existing meaning and formatting.

### Tests for User Story 1

- [x] T008 [P] [US1] Add `v8.8` service tests proving `WithStat` uses incident-bearing process-instance count semantics in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/service_test.go
- [x] T009 [P] [US1] Add `v8.9` service tests proving `WithStat` uses incident-bearing process-instance count semantics in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v89/service_test.go
- [x] T010 [P] [US1] Add command rendering regressions for supported non-zero and supported zero incident counts in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go

### Implementation for User Story 1

- [x] T011 [US1] Implement supported-version incident-count enrichment in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/service.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/convert.go
- [x] T012 [US1] Implement supported-version incident-count enrichment in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v89/service.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v89/convert.go
- [x] T013 [US1] Update the process-definition renderer to show `in:<count>` and `in:0` on supported versions in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_get.go

**Checkpoint**: User Story 1 is independently testable: supported versions show the clarified incident-bearing process-instance count and preserve the other stats fields.

---

## Phase 4: User Story 2 - Preserve Version-Specific Truthfulness (Priority: P2)

**Goal**: Keep unsupported versions truthful by omitting `in:` entirely while preserving the rest of the `--stat` output.

**Independent Test**: Run `get pd --stat` on `v8.7` and verify the output omits `in:` while keeping the surrounding output stable and not implying incident-count support.

### Tests for User Story 2

- [ ] T014 [P] [US2] Add `v8.7` service tests proving incident-count support remains unavailable under `WithStat` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/service_test.go
- [ ] T015 [P] [US2] Add command rendering regressions proving unsupported versions omit `in:` entirely in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go

### Implementation for User Story 2

- [ ] T016 [US2] Preserve the `v8.7` unsupported incident-count boundary in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/service.go
- [ ] T017 [US2] Update renderer behavior so unsupported stats omit `in:` without changing the other stat segments in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_get.go

**Checkpoint**: User Story 2 is independently testable: unsupported versions omit `in:` entirely and do not regress the rest of the stats output.

---

## Phase 5: User Story 3 - Verify Version Coverage With Tests (Priority: P3)

**Goal**: Make the supported and unsupported version boundaries durable through focused automated coverage and aligned user-facing docs.

**Independent Test**: Run the focused service, shared-model, and command tests plus documentation regeneration, and verify the behavior stays aligned across versions and docs.

### Tests for User Story 3

- [ ] T018 [P] [US3] Add shared facade/model coverage for the refined process-definition statistics representation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go
- [ ] T019 [P] [US3] Add or update cross-version command regressions covering supported non-zero, supported zero, and unsupported omission in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go

### Implementation for User Story 3

- [ ] T020 [US3] Update user-facing command documentation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md and the relevant command help text in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processdefinition.go
- [ ] T021 [US3] Regenerate the CLI reference in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli using /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile

**Checkpoint**: User Story 3 is independently testable: regression coverage protects the version boundary and the docs reflect the shipped behavior.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish validation, keep the feature artifacts aligned with the implemented behavior, and leave the repository ready for handoff or execution.

- [ ] T022 [P] Refresh implementation and verification notes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/plan.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/quickstart.md
- [ ] T023 Run focused validation with `go test ./c8volt/process -count=1`, `go test ./internal/services/processdefinition/... -count=1`, and `go test ./cmd -count=1` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt
- [ ] T024 Run docs regeneration validation with `make docs-content` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile
- [ ] T025 Run repository validation with `make test` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion and is the MVP slice.
- **User Story 2 (Phase 4)**: Depends on User Story 1 because unsupported rendering should be finalized against the settled shared stats model and supported-version renderer behavior.
- **User Story 3 (Phase 5)**: Depends on User Stories 1 and 2 because durable coverage and docs should reflect the final supported/unsupported contract.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: No dependency on later stories after Foundational work is complete.
- **User Story 2 (P2)**: Depends on the settled shared stats model and supported renderer behavior from User Story 1.
- **User Story 3 (P3)**: Depends on the supported and unsupported boundaries established by User Stories 1 and 2.

### Within Each User Story

- Add or update regression tests before considering the story complete.
- Shared model and conversion work before versioned service behavior that depends on it.
- Versioned service enrichment before renderer and command-level output assertions that depend on it.
- Documentation regeneration after the visible behavior is finalized.

### Parallel Opportunities

- `T002` and `T003` can run in parallel.
- `T006` and `T007` can run in parallel after the contract is settled.
- `T008`, `T009`, and `T010` can run in parallel.
- `T014` and `T015` can run in parallel.
- `T018` and `T019` can run in parallel.
- `T022` can run in parallel with focused validation after implementation stabilizes.

---

## Parallel Example: User Story 1

```bash
# Prepare supported-version count coverage in parallel:
Task: "Add `v8.8` service tests proving `WithStat` uses incident-bearing process-instance count semantics in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/service_test.go"
Task: "Add `v8.9` service tests proving `WithStat` uses incident-bearing process-instance count semantics in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v89/service_test.go"
Task: "Add command rendering regressions for supported non-zero and supported zero incident counts in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go"
```

## Parallel Example: User Story 2

```bash
# Prepare unsupported-version truthfulness coverage in parallel:
Task: "Add `v8.7` service tests proving incident-count support remains unavailable under `WithStat` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/service_test.go"
Task: "Add command rendering regressions proving unsupported versions omit `in:` entirely in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go"
```

## Parallel Example: User Story 3

```bash
# Prepare durable coverage and docs alignment in parallel:
Task: "Add shared facade/model coverage for the refined process-definition statistics representation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go"
Task: "Add or update cross-version command regressions covering supported non-zero, supported zero, and unsupported omission in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Stop and validate supported-version incident counts and supported zero rendering before expanding into unsupported-version truthfulness and documentation.

### Incremental Delivery

1. Finish Setup + Foundational once.
2. Deliver User Story 1 as the MVP for supported-version incident-count correctness.
3. Add User Story 2 to preserve exact truthfulness on unsupported versions.
4. Add User Story 3 to harden the version boundary with tests and docs.
5. Finish with focused validation, docs regeneration, and full `make test`.

### Parallel Team Strategy

1. One contributor handles Setup + Foundational work.
2. After Foundational is complete:
   - Contributor A: User Story 1 supported-version service enrichment and renderer changes.
   - Contributor B: User Story 2 unsupported-version truthfulness and command regressions.
   - Contributor C: User Story 3 shared-model coverage and documentation updates.
3. Finish with shared validation and repository-wide tests.

---

## Notes

- [P] tasks are limited to work on different files with no dependency on unfinished tasks.
- [US1], [US2], and [US3] map directly to the user stories in [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/spec.md).
- This feature’s commit subjects must keep Conventional Commit formatting and append `#42` as the final token.
- Run `make test` before committing, per repository rules.
