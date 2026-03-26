# Tasks: Fix Terminal Command Completion Suggestion Formatting

**Input**: Design documents from `/specs/82-tab-completion-format/`
**Prerequisites**: [plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/82-tab-completion-format/plan.md) (required), [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/82-tab-completion-format/spec.md) (required for user stories), [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/82-tab-completion-format/research.md), [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/82-tab-completion-format/data-model.md), [contracts/cli-command-contract.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/82-tab-completion-format/contracts/cli-command-contract.md), [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/82-tab-completion-format/quickstart.md)

**Tests**: Automated test tasks are REQUIRED for every story and shared change in this feature.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Inspect the existing completion surface and confirm the repo-native seams for the fix

- [ ] T001 Inspect current root-command completion behavior and utility-command treatment in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_checks.go
- [ ] T002 Inspect existing command-test helpers and help-output coverage in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/mutation_test.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared completion seam and regression harness before user-story work begins

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T003 Define the shared completion entry points and root-command metadata changes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_checks.go
- [ ] T004 [P] Add foundational completion regression helpers for root-command execution in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/mutation_test.go

**Checkpoint**: Foundation ready - user story implementation can now begin in priority order or parallel where noted

---

## Phase 3: User Story 1 - Get clean completion suggestions in the terminal (Priority: P1) 🎯 MVP

**Goal**: Ensure representative completion flows render readable user-facing suggestions instead of malformed help-like output

**Independent Test**: Trigger one representative top-level command completion and confirm the suggestion output remains readable, prompt-safe, and free of usage-style dumps.

### Tests for User Story 1

- [ ] T005 [P] [US1] Add command-level regression tests for readable top-level completion output in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go
- [ ] T006 [P] [US1] Add helper-process or root-execution regression coverage for malformed completion output handling in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/mutation_test.go

### Implementation for User Story 1

- [ ] T007 [US1] Update root command completion metadata and visibility rules to prevent usage-style dumps during completion in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go
- [ ] T008 [US1] Adjust shared utility-command or completion-command handling to keep interactive completion output readable in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_checks.go

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - See only relevant completion candidates (Priority: P2)

**Goal**: Keep normal user-facing candidates visible while filtering internal helper entries and preserving concise descriptions

**Independent Test**: Trigger one nested subcommand completion and one flag-completion flow and confirm user-facing candidates remain visible, internal helper entries do not appear, and concise descriptions render without full help text.

### Tests for User Story 2

- [ ] T009 [P] [US2] Add nested-command completion regression tests for candidate filtering in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go
- [ ] T010 [P] [US2] Add flag-completion regression tests covering readable descriptions and forbidden helper output in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/mutation_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go

### Implementation for User Story 2

- [ ] T011 [US2] Refine command/completion metadata so internal helper entries are excluded from normal suggestion output in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_checks.go
- [ ] T012 [US2] Preserve concise user-facing completion descriptions without falling back to full help text in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_processinstance.go
- [ ] T013 [US2] Verify representative command paths such as existing top-level, nested, and flag-completion flows remain repository-native after the metadata change in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_processinstance.go

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Keep completion behavior dependable across common command paths (Priority: P3)

**Goal**: Make the corrected completion behavior durable through focused regression coverage and aligned help/documentation

**Independent Test**: Run the agreed regression suite for one top-level path, one nested path, and one flag-completion path, then confirm any affected help or generated docs match the final behavior.

### Tests for User Story 3

- [ ] T014 [P] [US3] Consolidate the representative completion regression suite for top-level, nested, and flag scenarios in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/mutation_test.go

### Implementation for User Story 3

- [ ] T015 [US3] Update any user-visible root-command or completion help text affected by the final metadata changes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go
- [ ] T016 [US3] Refresh generated CLI documentation for any changed root-command help output in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt.md and related generated files under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/
- [ ] T017 [US3] Review and update completion usage guidance only if it becomes stale in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification and repository-wide consistency checks

- [ ] T018 [P] Run the quickstart validation steps in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/82-tab-completion-format/quickstart.md
- [ ] T019 Run the targeted completion regression commands from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt: `go test ./cmd -run 'Test.*Completion' -count=1`
- [ ] T020 Regenerate user-facing CLI documentation with `make docs` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt when help text changes are included
- [ ] T021 Run the repository validation command set, including `make test`, from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: Depend on Foundational phase completion
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Starts after Foundational - this is the MVP and proves clean completion rendering
- **User Story 2 (P2)**: Starts after User Story 1 because candidate filtering and description handling depend on the cleaned-up completion surface
- **User Story 3 (P3)**: Starts after User Stories 1 and 2 because the durable regression suite and docs should reflect the final completion contract

### Within Each User Story

- Regression tests are added before story sign-off
- Root-command metadata changes before representative command-path verification
- Candidate filtering before description-polish checks
- Help text updates before generated docs
- Targeted validation before repository-wide validation

### Parallel Opportunities

- T004 can run in parallel with T003 once the completion seam is identified
- T005 and T006 can run in parallel within User Story 1
- T009 and T010 can run in parallel within User Story 2
- T016 and T017 can run in parallel within User Story 3 after help text stabilizes
- T018 can run in parallel with T019 once implementation is complete

---

## Parallel Example: User Story 1

```bash
# Launch User Story 1 completion regressions together:
Task: "Add command-level regression tests for readable top-level completion output in cmd/get_test.go"
Task: "Add helper-process or root-execution regression coverage for malformed completion output handling in cmd/mutation_test.go"
```

---

## Parallel Example: User Story 2

```bash
# Launch User Story 2 regression coverage together:
Task: "Add nested-command completion regression tests for candidate filtering in cmd/get_test.go"
Task: "Add flag-completion regression tests covering readable descriptions and forbidden helper output in cmd/mutation_test.go and cmd/get_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Confirm representative top-level completion output is clean and readable

### Incremental Delivery

1. Fix the noisy completion rendering at the root-command surface
2. Add candidate filtering and concise-description preservation for representative nested and flag flows
3. Lock the behavior in with focused regressions and documentation alignment
4. Finish with targeted validation and `make test`

### Parallel Team Strategy

With multiple developers:

1. One developer completes Setup + Foundational work
2. After foundational work lands:
   - Developer A: User Story 1 rendering cleanup
   - Developer B: User Story 2 candidate filtering and description behavior
   - Developer C: User Story 3 regression hardening and docs refresh after the completion contract stabilizes

---

## Notes

- [P] tasks touch different files or can be validated independently
- [US1], [US2], and [US3] map directly to the user stories in [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/82-tab-completion-format/spec.md)
- Keep Cobra unchanged throughout implementation; all fixes must stay inside `c8volt` command/completion metadata and integration points
- Do not hand-edit generated CLI reference pages beyond what is produced from Cobra metadata; regenerate them with `make docs`
