# Tasks: Day-Based Process Instance Date Filters

**Input**: Design documents from `/specs/090-process-instance-date-filters/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are required for this feature because the specification explicitly calls for automated coverage of v8.8 happy paths, inclusive bounds, invalid formats, invalid ranges, and v8.7 not-implemented behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare the feature-specific test harness and verification notes before shared implementation work begins.

- [x] T001 [P] Create command test coverage scaffold for date-filter scenarios in cmd/get_processinstance_test.go
- [x] T002 [P] Align feature verification commands and temp-config prerequisites in specs/090-process-instance-date-filters/quickstart.md

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared filter-model and facade plumbing that MUST be complete before any story-specific behavior is added.

**⚠️ CRITICAL**: No user story work should begin until this phase is complete.

- [x] T003 [P] Extend the public process-instance filter with start/end date bound fields in c8volt/process/model.go
- [x] T004 [P] Extend the domain process-instance filter with start/end date bound fields in internal/domain/processinstance.go
- [x] T005 Update facade-to-domain process-instance filter conversion for date bounds in c8volt/process/client.go
- [x] T006 Add shared filter-mapping coverage for the new date bound fields in c8volt/process/client_test.go

**Checkpoint**: Shared process-instance filter models and facade mappings are ready for story-specific command and service work.

---

## Phase 3: User Story 1 - Filter by Start Date (Priority: P1) 🎯 MVP

**Goal**: Let users narrow `get process-instance` results by inclusive start-date bounds on v8.8.

**Independent Test**: Run `c8volt get process-instance --start-date-after YYYY-MM-DD` and `--start-date-before YYYY-MM-DD` against a v8.8 config and verify only matching start dates are returned, including inclusive boundary matches.

### Tests for User Story 1

- [x] T007 [US1] Add command coverage for valid start-date flags and inclusive range behavior in cmd/get_processinstance_test.go
- [x] T008 [US1] Add v8.8 service coverage for inclusive start-date request mapping in internal/services/processinstance/v88/service_test.go

### Implementation for User Story 1

- [x] T009 [US1] Implement start-date flags, parsing, and search-filter population in cmd/get_processinstance.go
- [x] T010 [US1] Implement native v8.8 start-date filter translation in internal/services/processinstance/v88/service.go

**Checkpoint**: User Story 1 is fully functional and testable on its own as the MVP slice.

---

## Phase 4: User Story 2 - Filter by End Date and Combine with Existing Filters (Priority: P2)

**Goal**: Let users narrow results by inclusive end-date bounds on v8.8 while preserving existing filter behavior.

**Independent Test**: Run `c8volt get process-instance --end-date-after YYYY-MM-DD`, `--end-date-before YYYY-MM-DD`, and a combined end-date plus `--state` query on v8.8, then verify inclusive bounds, exclusion of missing `endDate`, and continued enforcement of existing filters.

### Tests for User Story 2

- [x] T011 [US2] Add command coverage for valid end-date filters combined with existing search flags in cmd/get_processinstance_test.go
- [x] T012 [US2] Add v8.8 service coverage for end-date mapping and missing `endDate` exclusion in internal/services/processinstance/v88/service_test.go

### Implementation for User Story 2

- [x] T013 [US2] Implement end-date flags and composed search-filter population in cmd/get_processinstance.go
- [x] T014 [US2] Implement native v8.8 end-date filter translation and missing `endDate` handling in internal/services/processinstance/v88/service.go

**Checkpoint**: User Story 2 works independently and preserves composition with existing process-instance filters.

---

## Phase 5: User Story 3 - Get Clear Errors for Unsupported or Invalid Input (Priority: P3)

**Goal**: Reject malformed or unsupported date-filter usage with clear command and version-specific errors.

**Independent Test**: Run `c8volt get process-instance` with malformed dates, inverted ranges, `--key` plus date filters, and any date flag on a v8.7 config, then verify early validation failures or a clear not-implemented result through the existing error model.

### Tests for User Story 3

- [x] T015 [US3] Add command coverage for invalid date formats, invalid ranges, and `--key` incompatibility in cmd/get_processinstance_test.go
- [x] T016 [US3] Add v8.7 service coverage for date-filter not-implemented behavior in internal/services/processinstance/v87/service_test.go

### Implementation for User Story 3

- [x] T017 [US3] Implement date-range validation and direct-lookup rejection for date filters in cmd/get_processinstance.go
- [x] T018 [US3] Implement v8.7 date-filter rejection through the existing service error path in internal/services/processinstance/v87/service.go

**Checkpoint**: User Story 3 delivers clear failures for invalid or unsupported inputs without changing non-date behavior.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish user-facing documentation, generated docs, and full validation across all stories.

- [x] T019 Update command help text and examples for the new date flags in cmd/get_processinstance.go
- [x] T020 Update user-facing command documentation and examples in README.md
- [x] T021 Regenerate CLI reference output for the command in docs/cli/c8volt_get_process-instance.md via `make docs-content` and `make docs`
- [x] T022 Refresh feature smoke-check steps after implementation in specs/090-process-instance-date-filters/quickstart.md
- [x] T023 Run repository validation from Makefile and stabilize affected tests in Makefile, cmd/get_processinstance_test.go, internal/services/processinstance/v87/service_test.go, and internal/services/processinstance/v88/service_test.go

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion; recommended MVP starting point.
- **User Story 2 (Phase 4)**: Depends on Foundational completion and is best applied after User Story 1 because both stories modify the same command and v8.8 service files.
- **User Story 3 (Phase 5)**: Depends on Foundational completion and is best applied after User Stories 1 and 2 because it extends the same validation flow in `cmd/get_processinstance.go`.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: No functional dependency on other stories after Foundational is done.
- **User Story 2 (P2)**: Functionally independent for testing, but implementation shares `cmd/get_processinstance.go` and `internal/services/processinstance/v88/service.go` with User Story 1.
- **User Story 3 (P3)**: Functionally independent for testing, but implementation shares command validation seams with User Stories 1 and 2.

### Within Each User Story

- Add or update tests before finishing implementation for that story.
- Update command wiring before or alongside service translation so the story is exercisable through the CLI surface.
- Complete the story’s command and service work before treating the story as done.

### Parallel Opportunities

- `T001` and `T002` can run in parallel.
- `T003` and `T004` can run in parallel.
- Once a story phase starts, its command test task and service test task can be prepared in parallel with each other:
  - `T007` with `T008`
  - `T011` with `T012`
  - `T015` with `T016`
- Polish tasks `T020` and `T022` can run in parallel after implementation stabilizes.

---

## Parallel Example: User Story 1

```bash
# Prepare User Story 1 coverage in parallel:
Task: "Add command coverage for valid start-date flags and inclusive range behavior in cmd/get_processinstance_test.go"
Task: "Add v8.8 service coverage for inclusive start-date request mapping in internal/services/processinstance/v88/service_test.go"
```

## Parallel Example: User Story 2

```bash
# Prepare User Story 2 coverage in parallel:
Task: "Add command coverage for valid end-date filters combined with existing search flags in cmd/get_processinstance_test.go"
Task: "Add v8.8 service coverage for end-date mapping and missing endDate exclusion in internal/services/processinstance/v88/service_test.go"
```

## Parallel Example: User Story 3

```bash
# Prepare User Story 3 coverage in parallel:
Task: "Add command coverage for invalid date formats, invalid ranges, and --key incompatibility in cmd/get_processinstance_test.go"
Task: "Add v8.7 service coverage for date-filter not-implemented behavior in internal/services/processinstance/v87/service_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. Stop and validate the v8.8 start-date flow independently

### Incremental Delivery

1. Finish Setup + Foundational once.
2. Deliver User Story 1 as the MVP.
3. Add User Story 2 to extend end-date behavior without regressing existing filters.
4. Add User Story 3 to finalize validation and unsupported-version handling.
5. Finish documentation regeneration and full repository validation.

### Parallel Team Strategy

1. One contributor handles Setup + Foundational tasks.
2. After Foundational is complete, contributors can prepare story-specific tests in parallel while coordinating sequential edits to shared command/service files.
3. Finish with docs regeneration and repository-wide validation as a shared closeout step.

---

## Notes

- [P] tasks are limited to work on different files with no dependency on unfinished tasks.
- Each user story remains independently testable even though implementation touches some shared files.
- Commit messages for this feature must keep Conventional Commit formatting and append `#90` at the end of the subject line.
- Run `make test` before committing, per repository rules.
