# Tasks: Add Process-Instance Total-Only Output

**Input**: Design documents from `/specs/124-process-instances-total/`
**Prerequisites**: [plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/plan.md) (required), [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/spec.md) (required for user stories), [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/research.md), [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/data-model.md), [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/quickstart.md), [contracts/process-instance-total-output.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/contracts/process-instance-total-output.md)

**Tests**: Automated tests are REQUIRED for this feature because the specification explicitly requires count-only behavior coverage, zero-match coverage, lower-bound total coverage, conflicting flag validation, and non-regression of default output behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Lock in the exact count-only boundary, metadata seam, and regression anchors before shared implementation begins.

- [x] T001 Inventory the current `get process-instance` render/validation flow and count-related response seams in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_get.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service.go
- [x] T002 [P] Confirm the shared page-model and conversion seams for reported totals in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/model.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/convert.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go
- [x] T003 [P] Confirm the command, service, and docs regression anchors for `--total` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_processinstance_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_get_process-instance.md

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared count-only contract and metadata model that every user story builds on.

**⚠️ CRITICAL**: No user story work should begin until this phase is complete.

- [x] T004 Define the authoritative `--total` command contract, invalid combinations, and lower-bound total rule in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/plan.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/research.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/contracts/process-instance-total-output.md
- [x] T005 Extend the shared process-instance page model for reported totals and exact-vs-lower-bound semantics in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/processinstance.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/model.go
- [x] T006 [P] Update public/domain conversion seams and shared client coverage for the new page metadata in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/convert.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go
- [x] T007 [P] Refresh the feature data model and quickstart guidance for the finalized reported-total vocabulary in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/data-model.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/quickstart.md

**Checkpoint**: The repository has one shared reported-total contract, and command code can consume version-agnostic total metadata without version-specific branching.

---

## Phase 3: User Story 1 - Return Count Only (Priority: P1) 🎯 MVP

**Goal**: Let `get process-instance --total` return only the numeric count of matching process instances, including `0` for no matches.

**Independent Test**: Run `./c8volt get pi --total` and filtered variants against matching and no-match scenarios, and verify the command prints only a numeric result with no detail rows.

### Tests for User Story 1

- [x] T008 [P] [US1] Add command regressions for numeric-only `--total` output and zero-match behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go
- [x] T009 [P] [US1] Add shared page-metadata conversion coverage for count-only output in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go

### Implementation for User Story 1

- [x] T010 [US1] Add the `--total` flag and search-mode count-only command path in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [x] T011 [US1] Populate reported-total metadata from `v87`, `v88`, and `v89` search page responses in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service.go
- [x] T012 [US1] Keep non-`--total` detail rendering unchanged while short-circuiting count-only output before the existing list renderer in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_get.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go

**Checkpoint**: User Story 1 is independently testable: `--total` returns only a numeric count, including `0`, without changing default detail rendering.

---

## Phase 4: User Story 2 - Preserve Existing List Behavior (Priority: P2)

**Goal**: Preserve current non-`--total` behavior and make conflicting combinations fail clearly rather than producing ambiguous output.

**Independent Test**: Run default `get process-instance` flows before and after the change, and verify detail output is unchanged while `--total` rejects `--key`, `--json`, `--keys-only`, and `--with-age`.

### Tests for User Story 2

- [x] T013 [P] [US2] Add command regressions for invalid `--total` combinations and preserved default output behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_processinstance_test.go
- [x] T014 [P] [US2] Add service-level regressions proving reported-total metadata stays consistent without changing non-`--total` page behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service_test.go

### Implementation for User Story 2

- [x] T015 [US2] Enforce `--total` validation rules for `--key`, `--json`, `--keys-only`, and `--with-age` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go and any adjacent validation helpers under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/
- [x] T016 [US2] Keep command contract and output-mode metadata coherent for the new flag without introducing a new global render mode in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/command_contract.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_rendermode.go

**Checkpoint**: User Story 2 is independently testable: existing detail output remains unchanged, and unsupported `--total` combinations fail clearly.

---

## Phase 5: User Story 3 - Understand the New Flag Quickly (Priority: P3)

**Goal**: Make the new count-only behavior discoverable and accurately documented for users and automation.

**Independent Test**: Inspect `get process-instance --help`, README examples, and generated CLI docs to confirm `--total` is clearly described as returning only the number of found process instances and that lower-bound behavior is not contradicted.

### Tests for User Story 3

- [x] T017 [P] [US3] Add or update command-help regressions for the new `--total` flag text in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go

### Implementation for User Story 3

- [x] T018 [US3] Update user-facing command documentation and examples for `--total` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [x] T019 [US3] Regenerate CLI reference output with `make docs-content` so /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_get_process-instance.md matches the new flag behavior

**Checkpoint**: User Story 3 is independently testable: help text and docs clearly explain `--total` and do not conflict with shipped behavior.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish validation, synchronize planning artifacts, and leave the feature ready for implementation handoff or execution.

- [x] T020 [P] Refresh final implementation and verification notes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/plan.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/quickstart.md
- [x] T021 Run focused validation with `go test ./c8volt/process -count=1`, `go test ./internal/services/processinstance/... -count=1`, and `go test ./cmd -count=1` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt
- [x] T022 Run repository validation with `make test` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion and is the MVP slice.
- **User Story 2 (Phase 4)**: Depends on User Story 1 because validation and non-regression should build on the finalized count-only path and shared metadata seam.
- **User Story 3 (Phase 5)**: Depends on User Stories 1 and 2 because help and docs should describe the settled command contract.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: No dependency on later stories after Foundational work is complete.
- **User Story 2 (P2)**: Depends on the count-only command path and reported-total seam from User Story 1.
- **User Story 3 (P3)**: Depends on the settled behavior and validation rules from User Stories 1 and 2.

### Within Each User Story

- Add or update regression tests before considering the story complete.
- Shared page-model changes before command behavior that consumes them.
- Service metadata changes before final count-only output assertions that depend on them.
- Documentation regeneration only after command behavior and validation rules are settled.

### Parallel Opportunities

- `T002` and `T003` can run in parallel.
- `T006` and `T007` can run in parallel after the contract is settled.
- `T008` and `T009` can run in parallel.
- `T013` and `T014` can run in parallel.
- `T017` can run in parallel with `T018` once command wording stabilizes.
- `T020` can run in parallel with focused validation after implementation stabilizes.

---

## Parallel Example: User Story 1

```bash
# Prepare count-only coverage in parallel:
Task: "Add command regressions for numeric-only `--total` output and zero-match behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go"
Task: "Add shared page-metadata conversion coverage for count-only output in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go"
```

## Parallel Example: User Story 2

```bash
# Prepare non-regression coverage in parallel:
Task: "Add command regressions for invalid `--total` combinations and preserved default output behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_processinstance_test.go"
Task: "Add service-level regressions proving reported-total metadata stays consistent without changing non-`--total` page behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service_test.go"
```

## Parallel Example: User Story 3

```bash
# Prepare docs/help updates in parallel:
Task: "Add or update command-help regressions for the new `--total` flag text in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go"
Task: "Update user-facing command documentation and examples for `--total` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Stop and validate numeric-only `--total` output plus zero-match behavior before expanding into validation hardening and docs.

### Incremental Delivery

1. Finish Setup + Foundational once.
2. Deliver User Story 1 as the MVP for count-only output.
3. Add User Story 2 to preserve existing behavior and reject ambiguous combinations.
4. Add User Story 3 to make the new flag discoverable in help and docs.
5. Finish with focused validation and full `make test`.

### Parallel Team Strategy

1. One contributor handles Setup + Foundational work.
2. After Foundational is complete:
   - Contributor A: User Story 1 count-only command path and shared page metadata.
   - Contributor B: User Story 2 validation hardening and non-regression coverage.
   - Contributor C: User Story 3 help/docs updates once the command contract stabilizes.
3. Finish with shared validation and repository-wide tests.

---

## Notes

- [P] tasks are limited to work on different files with no dependency on unfinished tasks.
- [US1], [US2], and [US3] map directly to the user stories in [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/spec.md).
- This feature’s commit subjects must keep Conventional Commit formatting and append `#124` as the final token.
- Run `make test` before committing, per repository rules.
