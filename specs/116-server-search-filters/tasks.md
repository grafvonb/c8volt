# Tasks: Push Supported Get Filters Into Search Requests

**Input**: Design documents from `/specs/116-server-search-filters/`
**Prerequisites**: [plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/plan.md) (required), [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/spec.md) (required for user stories), [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/research.md), [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/data-model.md), [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/quickstart.md), [contracts/process-instance-search-filters.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/contracts/process-instance-search-filters.md)

**Tests**: Automated tests are REQUIRED for this feature because the specification explicitly requires request-side filtering coverage on supported versions, client-side fallback coverage on unsupported versions, paging-behavior regressions, and explicit proof that `--orphan-children-only` stays client-side.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Lock in the exact request-pushdown boundary, version matrix, and regression seams before shared filter changes begin.

- [x] T001 Inventory the current late-filtering seam and supported-version request capabilities in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/research.md
- [x] T002 [P] Confirm the shared filter-model seams and existing mapping coverage in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/model.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/convert.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go
- [x] T003 [P] Inspect the existing request-capture and paging regression anchors in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_processinstance_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared filter contract and shared mapping seams that every user story builds on.

**⚠️ CRITICAL**: No user story work should begin until this phase is complete.

- [x] T004 Define the authoritative pushdown contract and version matrix in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/contracts/process-instance-search-filters.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/research.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/plan.md
- [x] T005 Extend the shared process-instance filter model for parent-presence and incident-presence semantics in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/model.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/processinstance.go
- [x] T006 [P] Update public-to-domain filter mapping and shared client coverage for the new filter fields in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/convert.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go
- [x] T007 [P] Refresh planning artifacts to reflect the finalized shared filter vocabulary in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/data-model.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/quickstart.md

**Checkpoint**: The repository has one shared request-capable filter contract, and versioned services can receive the new semantics without command-local branching.

---

## Phase 3: User Story 1 - Return only matching process instances per page (Priority: P1) 🎯 MVP

**Goal**: Push `roots-only`, `children-only`, `incidents-only`, and `no-incidents-only` into the search request on supported versions so fetched pages already reflect the requested filter.

**Independent Test**: Run `get process-instance` with those four flags on `v8.8` and `v8.9`, verify the request body contains the pushed-down predicates, and confirm the resulting page/continuation behavior reflects the filtered server result set rather than a broad unfiltered page.

### Tests for User Story 1

- [x] T008 [P] [US1] Add shared filter-mapping regression coverage for parent-presence and incident-presence semantics in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go
- [x] T009 [P] [US1] Add `v8.8` request-capture tests for parent-presence and incident-presence pushdown in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go
- [x] T010 [P] [US1] Add `v8.9` request-capture tests for parent-presence and incident-presence pushdown in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service_test.go
- [x] T011 [P] [US1] Add command paging regressions for supported-version filtered page behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_processinstance_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go

### Implementation for User Story 1

- [x] T012 [US1] Translate supported list-mode flags into the shared request-capable filter fields in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [x] T013 [US1] Implement `v8.8` request-side encoding for parent-presence and incident-presence filters in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/convert.go
- [x] T014 [US1] Implement `v8.9` request-side encoding for parent-presence and incident-presence filters in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/convert.go

**Checkpoint**: User Story 1 is independently testable: supported versions send the new predicates in the search request and show paging behavior that matches the filtered server result set.

---

## Phase 4: User Story 2 - Preserve behavior on unsupported versions (Priority: P2)

**Goal**: Keep `v8.7` and all unsupported filter/version combinations on the existing client-side fallback path while preserving exact current command semantics.

**Independent Test**: Exercise the same flags on `v8.7` and verify unsupported predicates are omitted from the request, the final visible results still honor the flags through local filtering, and `--orphan-children-only` remains client-side on every version.

### Tests for User Story 2

- [x] T015 [P] [US2] Add `v8.7` service tests proving unsupported pushdown predicates stay out of the request shape in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go
- [x] T016 [P] [US2] Add command regressions proving `v8.7` fallback still returns the correct final filtered results in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_processinstance_test.go
- [x] T017 [P] [US2] Add cross-version regressions proving `--orphan-children-only` remains client-side in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go

### Implementation for User Story 2

- [x] T018 [US2] Preserve version-aware fallback behavior for unsupported pushdown semantics in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/filter.go
- [x] T019 [US2] Keep `v8.7` request construction limited to supported equality filters while documenting the fallback boundary in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/research.md
- [x] T020 [US2] Keep orphan-child detection on the existing follow-up lookup path in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go

**Checkpoint**: User Story 2 is independently testable: unsupported versions and semantics preserve current results through client-side fallback, and `--orphan-children-only` still uses the follow-up lookup flow.

---

## Phase 5: User Story 3 - Apply the same audit rule across get commands (Priority: P3)

**Goal**: Complete the broader `get`-command audit and either implement any additional clearly in-scope pushdown opportunity found or record an explicit no-addition rationale.

**Independent Test**: Review the audited `get` command families, confirm any qualifying late-filter seam has been moved request-side in this feature, and verify any non-qualifying families are recorded with explicit bounded rationale rather than silent omission.

### Tests for User Story 3

- [x] T021 [P] [US3] Add or update regression coverage for any additional audited `get` command family that qualifies for the same pushdown pattern in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/ and the corresponding versioned service test files under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/

### Implementation for User Story 3

- [x] T022 [US3] Audit other `get` command families for server-capable late-filtering seams in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processdefinition.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_resource.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster.go, and any adjacent `get_*` command files under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/
- [x] T023 [US3] Record the audit outcome and either adopted follow-up pushdown cases or explicit no-addition rationale in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/research.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/plan.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/quickstart.md

**Checkpoint**: User Story 3 is independently testable: the broader `get` audit is complete, no qualifying seam was skipped silently, and any additional in-scope pushdown work is covered in this feature.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish validation, keep the feature artifacts aligned with the shipped implementation, and leave the repo ready for implementation handoff or execution.

- [ ] T024 [P] Refresh implementation and verification notes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/plan.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/quickstart.md
- [ ] T025 Run focused validation with `go test ./c8volt/process -count=1`, `go test ./internal/services/processinstance/... -count=1`, and `go test ./cmd -count=1` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt
- [ ] T026 Run repository validation with `make test` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion and is the MVP slice.
- **User Story 2 (Phase 4)**: Depends on User Story 1 because fallback behavior should be preserved against the finalized shared filter contract and supported-version pushdown behavior.
- **User Story 3 (Phase 5)**: Depends on User Stories 1 and 2 because the broader audit should use the settled supported/fallback contract.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: No dependency on later stories after Foundational work is complete.
- **User Story 2 (P2)**: Depends on the shared filter additions and supported-version request behavior from User Story 1.
- **User Story 3 (P3)**: Depends on the settled request-side versus fallback contract established by User Stories 1 and 2.

### Within Each User Story

- Add or update regression tests before considering the story complete.
- Shared filter mapping before versioned service request encoding.
- Versioned service changes before command-level paging or fallback cleanup that depends on them.
- Audit documentation only after behavior and coverage are settled.

### Parallel Opportunities

- `T002` and `T003` can run in parallel.
- `T006` and `T007` can run in parallel after the contract is settled.
- `T008`, `T009`, `T010`, and `T011` can run in parallel.
- `T015`, `T016`, and `T017` can run in parallel.
- `T021` can run in parallel with `T022` once the broader audit identifies an additional qualifying seam.
- `T024` can run in parallel with targeted validation after implementation stabilizes.

---

## Parallel Example: User Story 1

```bash
# Prepare supported-version pushdown coverage in parallel:
Task: "Add shared filter-mapping regression coverage for parent-presence and incident-presence semantics in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go"
Task: "Add `v8.8` request-capture tests for parent-presence and incident-presence pushdown in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go"
Task: "Add `v8.9` request-capture tests for parent-presence and incident-presence pushdown in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service_test.go"
Task: "Add command paging regressions for supported-version filtered page behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_processinstance_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go"
```

## Parallel Example: User Story 2

```bash
# Prepare unsupported-version fallback coverage in parallel:
Task: "Add `v8.7` service tests proving unsupported pushdown predicates stay out of the request shape in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go"
Task: "Add command regressions proving `v8.7` fallback still returns the correct final filtered results in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_processinstance_test.go"
Task: "Add cross-version regressions proving `--orphan-children-only` remains client-side in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Stop and validate supported-version request pushdown plus paging accuracy before expanding into fallback preservation and the broader audit.

### Incremental Delivery

1. Finish Setup + Foundational once.
2. Deliver User Story 1 as the MVP for supported-version request-side pushdown.
3. Add User Story 2 to preserve exact current behavior on unsupported versions and for `--orphan-children-only`.
4. Add User Story 3 to satisfy the broader `get`-command audit requirement.
5. Finish with targeted validation and full `make test`.

### Parallel Team Strategy

1. One contributor handles Setup + Foundational work.
2. After Foundational is complete:
   - Contributor A: User Story 1 shared filter and supported-version request builders.
   - Contributor B: User Story 2 fallback and orphan-child regressions.
   - Contributor C: User Story 3 broader `get`-command audit and any additional qualifying seam.
3. Finish with shared validation and repository-wide tests.

---

## Notes

- [P] tasks are limited to work on different files with no dependency on unfinished tasks.
- [US1], [US2], and [US3] map directly to the user stories in [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/spec.md).
- This feature’s commit subjects must keep Conventional Commit formatting and append `#116` as the final token.
- Run `make test` before committing, per repository rules.
