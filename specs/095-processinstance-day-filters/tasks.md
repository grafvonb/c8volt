# Tasks: Relative Day-Based Process-Instance Date Shortcuts

**Input**: Design documents from `/specs/095-processinstance-day-filters/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are required for this feature because the specification explicitly calls for automated coverage of happy-path conversion, inclusive boundaries, configured-local-day derivation, missing `endDate` exclusion, invalid values, conflicting absolute and relative inputs, invalid derived ranges, invalid `--key` combinations, preserved direct-key behavior, and v8.7 not-implemented responses.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Align feature-specific verification guidance and command test scaffolding before shared command-helper changes begin.

- [x] T001 [P] Review and align implementation and verification notes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/095-processinstance-day-filters/quickstart.md
- [x] T002 [P] Add relative-day command test scaffolding in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared command-layer flag wiring and conversion helpers that MUST be complete before any user story is implemented.

**⚠️ CRITICAL**: No user story work should begin until this phase is complete.

- [x] T003 Wire shared relative day flag registration into /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go
- [x] T004 Implement shared relative-day parsing, local-day derivation, and mixed-filter validation helpers in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [x] T005 Update shared search-filter detection and absolute-bound population for relative day inputs in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go

**Checkpoint**: All three commands expose the new flags and share one canonical command-layer conversion and validation path.

---

## Phase 3: User Story 1 - Filter Get Results by Relative Day Offsets (Priority: P1) 🎯 MVP

**Goal**: Let users narrow `get process-instance` results with relative day-based shortcut flags that behave like the existing absolute date filters.

**Independent Test**: Run `c8volt get process-instance --start-after-days 7`, `--start-before-days 30`, and `--end-before-days 14` against a v8.8 config and verify the resulting search request behaves like the equivalent derived absolute date filters, including inclusive boundaries.

### Tests for User Story 1

- [x] T006 [US1] Add command coverage for relative start-day and end-day search requests in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go
- [x] T007 [US1] Add facade mapping coverage proving derived relative-day bounds use the canonical absolute date fields in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go

### Implementation for User Story 1

- [x] T008 [US1] Implement relative-day-aware `get process-instance` search filter composition in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [x] T009 [US1] Verify the shared process-instance filter shape continues to carry derived absolute date bounds unchanged through /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/processinstance.go

**Checkpoint**: User Story 1 is fully functional and independently testable as the MVP slice.

---

## Phase 4: User Story 2 - Use the Same Relative Filters for Search-Based Cancel and Delete (Priority: P2)

**Goal**: Let users apply the new relative day-based shortcut flags to search-driven `cancel process-instance` and `delete process-instance` workflows without changing direct key behavior.

**Independent Test**: Run `c8volt cancel process-instance --state active --start-before-days 30` and `c8volt delete process-instance --end-after-days 60 --end-before-days 7 --auto-confirm` on v8.8 and verify only matching instances are selected through the existing search-based management path.

### Tests for User Story 2

- [x] T010 [P] [US2] Add cancel-command coverage for relative-day search selection in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go
- [x] T011 [P] [US2] Add delete-command coverage for relative-day search selection in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go

### Implementation for User Story 2

- [x] T012 [US2] Implement relative-day search selection and examples in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go
- [x] T013 [US2] Implement relative-day search selection and examples in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go
- [x] T014 [US2] Confirm shared relative-day filter composition continues to flow through the existing management search path in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go

**Checkpoint**: User Story 2 works independently and preserves management-command search behavior while adding relative-day narrowing.

---

## Phase 5: User Story 3 - Receive Clear Validation and Version Responses (Priority: P3)

**Goal**: Reject invalid relative-day input, conflicting filter combinations, invalid `--key` usage, and unsupported v8.7 behavior before any search-based action occurs.

**Independent Test**: Run the affected commands with negative values, mixed absolute-plus-relative flags, invalid derived ranges, explicit `--key` plus relative day flags, relative end-day filters on instances lacking `endDate`, and any relative-day flag on v8.7, then verify each path fails or filters through the expected shared behavior.

### Tests for User Story 3

- [x] T015 [P] [US3] Add command coverage for invalid values, mixed absolute-plus-relative filters, invalid ranges, and local-day derivation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go
- [x] T016 [P] [US3] Add management-command coverage for invalid `--key` combinations and v8.7 unsupported relative-day usage in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go
- [x] T017 [P] [US3] Add service regression coverage for missing `endDate` exclusion and v8.7 not-implemented behavior with derived absolute bounds in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go

### Implementation for User Story 3

- [x] T018 [US3] Enforce invalid relative-day combinations and derived-range validation before search execution in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [x] T019 [US3] Preserve invalid `--key` plus relative-day rejection for management commands in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go
- [x] T020 [US3] Verify relative-day flows reuse the existing v8.7 rejection and v8.8 missing-`endDate` handling in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go

**Checkpoint**: User Story 3 delivers clear validation failures and preserves the existing version-specific service contract.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish user-facing documentation, generated docs, and repository-wide validation across all stories.

- [ ] T021 Update relative-day command help text and examples in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go
- [ ] T022 Update user-facing relative-day filter examples in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md
- [ ] T023 Regenerate CLI reference output for /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_get_process-instance.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_cancel_process-instance.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_delete_process-instance.md via `make docs-content` and `make docs`
- [ ] T024 [P] Refresh implemented verification steps in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/095-processinstance-day-filters/quickstart.md
- [ ] T025 Run repository validation with `make test` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion; recommended MVP starting point.
- **User Story 2 (Phase 4)**: Depends on Foundational completion and is best applied after User Story 1 because it reuses the same command-helper conversion path across management commands.
- **User Story 3 (Phase 5)**: Depends on Foundational completion and is best applied after User Stories 1 and 2 because it finalizes validation, end-date exclusion, and version-specific regression coverage across all three commands.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: No functional dependency on other stories after Foundational is done.
- **User Story 2 (P2)**: Functionally independent for testing, but implementation should mirror the completed shared conversion path proven by User Story 1.
- **User Story 3 (P3)**: Functionally independent for testing, but it extends the same shared validation and version-specific seams used by User Stories 1 and 2.

### Within Each User Story

- Add or update tests before finishing implementation for that story.
- Complete shared filter composition before treating command behavior as done.
- Keep service-level checks focused on proving the relative-day path still lands on the canonical absolute-date behavior.

### Parallel Opportunities

- `T001` and `T002` can run in parallel.
- `T010` and `T011` can run in parallel.
- `T015`, `T016`, and `T017` can run in parallel.
- `T022` and `T024` can run in parallel after command help text is stable.

---

## Parallel Example: User Story 2

```bash
# Prepare User Story 2 management-command coverage in parallel:
Task: "Add cancel-command coverage for relative-day search selection in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go"
Task: "Add delete-command coverage for relative-day search selection in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go"
```

## Parallel Example: User Story 3

```bash
# Prepare User Story 3 regression coverage in parallel:
Task: "Add command coverage for invalid values, mixed absolute-plus-relative filters, invalid ranges, and local-day derivation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go"
Task: "Add management-command coverage for invalid --key combinations and v8.7 unsupported relative-day usage in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go"
Task: "Add service regression coverage for missing endDate exclusion and v8.7 not-implemented behavior with derived absolute bounds in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. Stop and validate the `get process-instance` relative-day flow independently

### Incremental Delivery

1. Finish Setup + Foundational once.
2. Deliver User Story 1 as the MVP.
3. Add User Story 2 to extend the same relative-day behavior to management commands.
4. Add User Story 3 to finalize validation, unsupported-version handling, and end-date exclusion guarantees.
5. Finish docs regeneration and repository-wide validation.

### Parallel Team Strategy

1. One contributor handles Setup + Foundational tasks.
2. After Foundational is complete, one contributor can drive `get process-instance` while another prepares management-command coverage, as long as edits to shared helper code are coordinated.
3. Finish with shared validation, docs regeneration, and `make test` as a coordinated closeout step.

---

## Notes

- [P] tasks are limited to work on different files with no dependency on unfinished tasks.
- Each user story remains independently testable even though the implementation reuses the same shared process-instance search helpers.
- Commit messages for this feature must keep Conventional Commit formatting and append `#95` at the end of the subject line.
- Run `make test` before committing, per repository rules.
