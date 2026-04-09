# Tasks: Extend Process-Instance Management Date Filters

**Input**: Design documents from `/specs/093-extend-pi-date-filters/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are required for this feature because the specification explicitly calls for automated coverage of v8.8 success paths, inclusive bounds, empty search results, invalid formats, invalid ranges, invalid `--key` plus date-filter combinations, and v8.7 not-implemented behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare the feature-specific validation surface and implementation notes before command changes begin.

- [x] T001 [P] Review and align feature verification notes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/093-extend-pi-date-filters/quickstart.md
- [x] T002 [P] Add task-oriented command test scaffolding for management date-filter scenarios in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared command-layer wiring that MUST be ready before the story-specific command flows are implemented.

**⚠️ CRITICAL**: No user story work should begin until this phase is complete.

- [x] T003 Wire the shared process-instance date-search flags into /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go
- [x] T004 Reuse shared process-instance search validation for management commands in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go

**Checkpoint**: Cancel/delete expose the shared date-filter surface and validate it through the existing command-layer helpers.

---

## Phase 3: User Story 1 - Cancel by Date-Filtered Search (Priority: P1) 🎯 MVP

**Goal**: Let users cancel process instances selected through the same inclusive date-filter search behavior already available on `get process-instance`.

**Independent Test**: Run `c8volt cancel process-instance` on v8.8 with search filters plus `--start-date-*` and `--end-date-*`, and verify only matching process instances are selected for cancellation.

### Tests for User Story 1

- [ ] T005 [US1] Add cancel command coverage for v8.8 date-filtered search selection in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go
- [ ] T006 [US1] Add cancel command coverage for no-match search failure behavior with date filters in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go

### Implementation for User Story 1

- [ ] T007 [US1] Implement date-filter-aware search selection and examples in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go
- [ ] T008 [US1] Verify cancel search selection keeps using the existing shared process-instance filter path in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go

**Checkpoint**: User Story 1 is fully functional and testable on its own as the MVP slice.

---

## Phase 4: User Story 2 - Delete by Date-Filtered Search (Priority: P2)

**Goal**: Let users delete process instances selected through inclusive start/end date bounds combined with the existing delete search filters.

**Independent Test**: Run `c8volt delete process-instance` on v8.8 without explicit keys, combine state and date filters, and verify only matching process instances are selected for deletion.

### Tests for User Story 2

- [ ] T009 [US2] Add delete command coverage for v8.8 date-filtered search selection in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go
- [ ] T010 [US2] Add delete command coverage for empty selected sets with date filters in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go

### Implementation for User Story 2

- [ ] T011 [US2] Implement date-filter-aware search selection and examples in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go
- [ ] T012 [US2] Confirm delete search selection composes date filters with existing management filters in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go

**Checkpoint**: User Story 2 works independently and preserves delete search behavior while adding date-based narrowing.

---

## Phase 5: User Story 3 - Preserve Validation and Version Limits (Priority: P3)

**Goal**: Reject malformed or unsupported date-filter usage with clear failures before any cancellation or deletion occurs.

**Independent Test**: Run the cancel/delete commands with malformed dates, inverted ranges, explicit `--key` plus date filters, and v8.7 date-filtered usage, then verify that each fails through the correct validation or not-implemented path.

### Tests for User Story 3

- [ ] T013 [P] [US3] Add cancel command coverage for invalid dates, invalid ranges, `--key` plus date-filter rejection, and v8.7 unsupported behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go
- [ ] T014 [P] [US3] Add delete command coverage for invalid dates, invalid ranges, `--key` plus date-filter rejection, and v8.7 unsupported behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go

### Implementation for User Story 3

- [ ] T015 [US3] Enforce invalid `--key` plus date-filter combinations and date validation failures before management actions in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go
- [ ] T016 [US3] Verify v8.7 date-filtered management flows continue through the existing shared not-implemented service path using /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go

**Checkpoint**: User Story 3 delivers clear failures for invalid and unsupported management-command date-filter usage.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish user-visible documentation, generated docs, and final repository validation across all stories.

- [ ] T017 Update user-facing management command examples in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md
- [ ] T018 Update command help text and examples for management date filters in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go
- [ ] T019 Regenerate CLI reference output for /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_cancel_process-instance.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_delete_process-instance.md via `make docs-content` and `make docs`
- [ ] T020 [P] Refresh implemented verification steps in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/093-extend-pi-date-filters/quickstart.md
- [ ] T021 Run repository validation for the feature with `make test` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion; recommended MVP starting point.
- **User Story 2 (Phase 4)**: Depends on Foundational completion and is best applied after User Story 1 because both stories reuse the same shared search helpers and follow the same command pattern.
- **User Story 3 (Phase 5)**: Depends on Foundational completion and is best applied after User Stories 1 and 2 because it finalizes validation and unsupported-version handling across both management commands.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: No functional dependency on other stories after Foundational is done.
- **User Story 2 (P2)**: Functionally independent for testing, but implementation should mirror the completed cancel-command pattern from User Story 1.
- **User Story 3 (P3)**: Functionally independent for testing, but implementation extends the same command validation seams used by User Stories 1 and 2.

### Within Each User Story

- Add or update tests before finishing implementation for that story.
- Wire command flags and shared search selection before treating the management flow as complete.
- Finish validation behavior before closing the story as independently testable.

### Parallel Opportunities

- `T001` and `T002` can run in parallel.
- `T013` and `T014` can run in parallel.
- `T017` and `T020` can run in parallel after command behavior is stable.

---

## Parallel Example: User Story 3

```bash
# Prepare User Story 3 regression coverage in parallel:
Task: "Add cancel command coverage for invalid dates, invalid ranges, --key plus date-filter rejection, and v8.7 unsupported behavior in cmd/cancel_test.go"
Task: "Add delete command coverage for invalid dates, invalid ranges, --key plus date-filter rejection, and v8.7 unsupported behavior in cmd/delete_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. Stop and validate the cancel-command date-filter flow independently

### Incremental Delivery

1. Finish Setup + Foundational once.
2. Deliver User Story 1 as the MVP.
3. Add User Story 2 to extend the same behavior to delete.
4. Add User Story 3 to finalize validation and unsupported-version behavior.
5. Finish documentation regeneration and full repository validation.

### Parallel Team Strategy

1. One contributor handles Setup + Foundational tasks.
2. After Foundational is complete, one contributor can drive cancel while another prepares delete tests and documentation updates.
3. Finish with shared validation, docs regeneration, and repository-wide test execution.

---

## Notes

- [P] tasks are limited to work on different files with no dependency on unfinished tasks.
- Each user story remains independently testable even though implementation reuses the same shared process-instance search helpers.
- Commit messages for this feature must keep Conventional Commit formatting and append `#93` at the end of the subject line.
- Run `make test` before committing, per repository rules.
