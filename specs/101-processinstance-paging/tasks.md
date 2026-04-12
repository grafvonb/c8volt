# Tasks: Version-Aware Process-Instance Paging and Overflow Handling

**Input**: Design documents from `/specs/101-processinstance-paging/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are required for this feature because the specification explicitly calls for automated coverage of default paging, shared config defaults, `--count` override behavior, overflow detection, prompt-driven continuation, auto-confirm continuation, exact-boundary non-overflow, cross-page processing, partial completion, warning-stop behavior, and version-specific handling for `8.7` and `8.8`.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare paging-focused verification guidance and reusable test capture seams before shared implementation work begins.

- [x] T001 [P] Review and align paging verification notes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/101-processinstance-paging/quickstart.md
- [x] T002 [P] Add shared paging-oriented command test capture helpers in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_processinstance_test.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared config, API, and command orchestration seams that MUST exist before any story-specific paging flow can be implemented.

**⚠️ CRITICAL**: No user story work should begin until this phase is complete.

- [x] T003 Add the shared process-instance paging default config field and normalization in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/app.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config.go
- [x] T004 Bind the shared process-instance paging default into the Cobra/Viper bootstrap in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go
- [x] T005 Extend paging-related process-instance facade and service contracts in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/api.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/api.go
- [x] T006 Create shared page-size resolution, continuation-state, and progress-summary helpers in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go

**Checkpoint**: The repository has one shared page-size config path and one shared command/service seam for paging metadata and continuation decisions.

---

## Phase 3: User Story 1 - Page Through Matching Process Instances (Priority: P1) 🎯 MVP

**Goal**: Let `c8volt get process-instance` page through large result sets without silent truncation, while reporting page size, current-page count, cumulative count, and continuation state.

**Independent Test**: Run `c8volt get process-instance` on a v8.8 config with more matches than fit in one page, verify the command uses the shared default or `--count` override, reports current-page and cumulative counts, and prompts or auto-continues according to `--auto-confirm`.

### Tests for User Story 1

- [ ] T007 [P] [US1] Add `get process-instance` command coverage for shared default page size, `--count` overrides, prompt flow, and auto-confirm flow in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go
- [ ] T008 [P] [US1] Add facade regression coverage for process-instance paging metadata propagation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go
- [ ] T009 [P] [US1] Add v8.8 service coverage for native page metadata and exact-boundary non-overflow behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go

### Implementation for User Story 1

- [ ] T010 [US1] Implement v8.8 process-instance search metadata extraction and overflow signaling in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go
- [ ] T011 [US1] Implement facade mapping for paged process-instance search results in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go
- [ ] T012 [US1] Implement shared `get process-instance` paging orchestration, continuation prompts, and progress output in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [ ] T013 [US1] Update `get process-instance` examples and help text for paging behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go

**Checkpoint**: User Story 1 is independently functional and testable as the MVP slice for paging-aware read-only search.

---

## Phase 4: User Story 2 - Continue Search-Based Cancel and Delete Safely Across Pages (Priority: P2)

**Goal**: Let search-based `cancel process-instance` and `delete process-instance` process one page at a time with the same continuation model as `get`, while preserving direct-key behavior.

**Independent Test**: Run search-based `cancel process-instance` and `delete process-instance` against data sets larger than one page, verify the commands process one page at a time, prompt or auto-continue according to `--auto-confirm`, and preserve existing direct `--key` behavior outside paging mode.

### Tests for User Story 2

- [ ] T014 [P] [US2] Add paging regression coverage for search-based cancellation, prompt flow, and auto-confirm continuation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go
- [ ] T015 [P] [US2] Add paging regression coverage for search-based deletion, prompt flow, and auto-confirm continuation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go

### Implementation for User Story 2

- [ ] T016 [US2] Implement shared search-page processing for cancellation flows in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go
- [ ] T017 [US2] Implement shared search-page processing for deletion flows in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go
- [ ] T018 [US2] Preserve direct-key bypass and align paging-aware examples in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go

**Checkpoint**: User Story 2 works independently and applies the same page-by-page continuation model to search-based write commands without altering direct-key workflows.

---

## Phase 5: User Story 3 - Receive Version-Aware Overflow Handling and Clear Operator Feedback (Priority: P3)

**Goal**: Deliver explicit version-aware overflow behavior for `8.7` and `8.8`, including partial-completion summaries, warning-stop behavior, and consistent operator-facing output across all affected commands.

**Independent Test**: Run the affected commands on `8.7` and `8.8` fixtures that exercise exact-boundary pages, pages with additional matches, user-declined continuation, and indeterminate fallback behavior; verify the commands report current-page and cumulative counts, stop cleanly on partial completion, and warn when overflow cannot be proven.

### Tests for User Story 3

- [ ] T019 [P] [US3] Add command coverage for partial-completion summaries, cumulative counts, and warning-stop behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go
- [ ] T020 [P] [US3] Add v8.7 service regression coverage for fallback overflow detection and indeterminate-warning behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go
- [ ] T021 [P] [US3] Add cross-version paging metadata contract coverage in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go

### Implementation for User Story 3

- [ ] T022 [US3] Implement v8.7 fallback overflow detection and indeterminate-warning signaling in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go
- [ ] T023 [US3] Implement partial-completion and warning-stop summaries in the shared command paging helpers in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [ ] T024 [US3] Align cross-command paging output wording and continuation-state reporting in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go

**Checkpoint**: User Story 3 makes version differences explicit, keeps operator messaging consistent, and closes the remaining safety gaps around partial completion and indeterminate overflow.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish shared documentation, generated docs, and repository-wide validation across all stories.

- [ ] T025 Update user-facing paging behavior and config guidance in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md
- [ ] T026 Regenerate CLI reference output for /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_get_process-instance.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_cancel_process-instance.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_delete_process-instance.md via `make docs-content` and `make docs`
- [ ] T027 [P] Refresh implemented verification notes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/101-processinstance-paging/quickstart.md
- [ ] T028 Run repository validation with `make test` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion; recommended MVP starting point.
- **User Story 2 (Phase 4)**: Depends on Foundational completion and is best applied after User Story 1 because it reuses the same paging orchestration seam for search-mode commands.
- **User Story 3 (Phase 5)**: Depends on Foundational completion and is best applied after User Stories 1 and 2 because it finalizes version-aware overflow handling and cross-command summary behavior.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: No functional dependency on other stories after the Foundational phase is complete.
- **User Story 2 (P2)**: Functionally independent for testing, but implementation should reuse the paging orchestration proven by User Story 1.
- **User Story 3 (P3)**: Functionally independent for testing, but implementation closes version/fallback behavior across the shared orchestration introduced by User Stories 1 and 2.

### Within Each User Story

- Add or update regression tests before considering the story complete.
- Complete service/facade metadata plumbing before finishing command behavior that depends on it.
- Treat current-page counts, cumulative counts, and continuation state as part of the story’s acceptance surface, not as optional polish.

### Parallel Opportunities

- `T001` and `T002` can run in parallel.
- `T007`, `T008`, and `T009` can run in parallel.
- `T014` and `T015` can run in parallel.
- `T019`, `T020`, and `T021` can run in parallel.
- `T025` and `T027` can run in parallel after command help text and output wording are stable.

---

## Parallel Example: User Story 1

```bash
# Prepare User Story 1 coverage in parallel:
Task: "Add get process-instance command coverage for shared default page size, --count overrides, prompt flow, and auto-confirm flow in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go"
Task: "Add facade regression coverage for process-instance paging metadata propagation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go"
Task: "Add v8.8 service coverage for native page metadata and exact-boundary non-overflow behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go"
```

## Parallel Example: User Story 3

```bash
# Prepare User Story 3 regression coverage in parallel:
Task: "Add command coverage for partial-completion summaries, cumulative counts, and warning-stop behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go"
Task: "Add v8.7 service regression coverage for fallback overflow detection and indeterminate-warning behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go"
Task: "Add cross-version paging metadata contract coverage in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. Stop and validate `get process-instance` paging independently before expanding to write commands

### Incremental Delivery

1. Finish Setup + Foundational once.
2. Deliver User Story 1 as the MVP for paging-aware search.
3. Add User Story 2 to extend the same continuation model to search-based cancel/delete.
4. Add User Story 3 to finalize version-aware fallback behavior and operator summaries.
5. Finish docs regeneration and repository-wide validation.

### Parallel Team Strategy

1. One contributor handles Setup + Foundational tasks.
2. After Foundational is complete, one contributor can drive `get process-instance` while another prepares search-based write-command regression coverage.
3. Finish with version-specific service work, shared summary alignment, documentation regeneration, and `make test`.

---

## Notes

- [P] tasks are limited to work on different files with no dependency on unfinished tasks.
- Each user story remains independently testable even though implementation reuses shared command/facade/service paging seams.
- Commit messages for this feature must keep Conventional Commit formatting and append `#101` at the end of the subject line.
- Run `make test` before committing, per repository rules.
