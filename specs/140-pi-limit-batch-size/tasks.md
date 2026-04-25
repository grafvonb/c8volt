# Tasks: Process-Instance Limit and Batch Size Flags

**Input**: Design documents from `/specs/140-pi-limit-batch-size/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are required because the specification changes CLI flags, validation, paging stop semantics, destructive command behavior, generated help, and user-facing docs.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish the shared flag contract and test seams used by all stories.

- [x] T001 Review existing process-instance paging helpers and identify shared limit insertion points in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [x] T002 [P] Review command test helpers for multi-page process-instance fixtures in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_processinstance_test.go
- [x] T003 [P] Review affected docs references for `--count`, `--batch-size`, and `--limit` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared flag registration, validation, and internal state that MUST exist before user-story behavior is implemented.

**CRITICAL**: No user story work should begin until this phase is complete.

- [x] T004 Replace affected `--count` registrations with `--batch-size` while preserving `-n` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go
- [x] T005 Add shared `--limit` / `-l` flag storage and registration for affected process-instance commands in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go
- [x] T006 Add validation for positive `--limit`, `--limit` with `--key`, `--limit` with `--total`, and updated `--batch-size` flag checks in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [x] T007 Add command tests for removed `--count`, invalid `--limit`, and `--limit` with `--key` across get/cancel/delete in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go

**Checkpoint**: The public flag surface is renamed, limit validation is enforced, and invalid legacy/combination cases are tested.

---

## Phase 3: User Story 1 - Limit Search Results Across Pages (Priority: P1) MVP

**Goal**: Cap total matched process instances across pages for `get`, search-based `cancel`, and search-based `delete`.

**Independent Test**: Run each affected command against a multi-page fixture with `--limit` and verify no more than the configured total is returned or processed.

### Tests for User Story 1

- [ ] T008 [P] [US1] Add `get process-instance` multi-page `--limit` tests in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go
- [ ] T009 [P] [US1] Add search-based `cancel process-instance` multi-page `--limit` tests in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go
- [ ] T010 [P] [US1] Add search-based `delete process-instance` multi-page `--limit` tests in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go

### Implementation for User Story 1

- [ ] T011 [US1] Add remaining-limit calculation and page truncation helpers in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [ ] T012 [US1] Apply limited page results to `get process-instance` aggregation and incremental rendering in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [ ] T013 [US1] Apply limited page keys to search-based cancel/delete page actions in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go

**Checkpoint**: User Story 1 is independently functional and testable as the MVP slice.

---

## Phase 4: User Story 2 - Distinguish Batch Size From Total Limit (Priority: P2)

**Goal**: Ensure `--batch-size` and `--limit` work independently and together.

**Independent Test**: Run affected commands with `--batch-size`, `-n`, `--limit`, and combined `--batch-size --limit`; verify per-page fetch size and total processed count differ as expected.

### Tests for User Story 2

- [ ] T014 [P] [US2] Add `get process-instance` tests for `--batch-size`, `-n`, and combined `--batch-size --limit` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go
- [ ] T015 [P] [US2] Add search-based cancel/delete tests for combined `--batch-size --limit` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go

### Implementation for User Story 2

- [ ] T016 [US2] Update page-size resolution to use `--batch-size` flag change detection and preserve shared config/default behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [ ] T017 [US2] Update examples and worker help text to use batch-size terminology in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go

**Checkpoint**: User Story 2 proves the renamed page-size flag is independent from the total limit.

---

## Phase 5: User Story 3 - Reject Ambiguous or Invalid Flag Combinations (Priority: P3)

**Goal**: Make migration and invalid usage failures clear and consistent.

**Independent Test**: Run affected commands with removed `--count`, invalid `--limit`, and direct key plus limit; verify standard invalid-arguments behavior.

### Tests for User Story 3

- [ ] T018 [P] [US3] Add explicit help/parse regression coverage proving `--count` is absent or rejected on affected command paths in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go
- [ ] T019 [P] [US3] Add regression coverage confirming `--total` with `--limit` is rejected as mutually exclusive in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go

### Implementation for User Story 3

- [ ] T020 [US3] Ensure `--limit` validation is search-mode-only and rejects `--total` combinations in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [ ] T021 [US3] Align standard invalid-argument handling for removed and invalid flags without adding `--count` aliases in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go

**Checkpoint**: User Story 3 makes invalid and legacy usage behavior explicit and tested.

---

## Phase 6: User Story 4 - Discover the New Semantics in Help and Docs (Priority: P4)

**Goal**: Keep help, README examples, and generated CLI docs synchronized with the new flag semantics.

**Independent Test**: Inspect help and generated docs to verify affected command paths use `--batch-size`/`-n` and `--limit`/`-l`, and no longer document `--count`.

### Tests for User Story 4

- [ ] T022 [P] [US4] Add or update help-output assertions for batch-size and limit descriptions in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go

### Implementation for User Story 4

- [ ] T023 [US4] Update README process-instance paging examples and explanation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md
- [ ] T024 [US4] Regenerate generated CLI docs with `make docs-content` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile
- [ ] T025 [US4] Verify generated docs for affected commands in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_get_process-instance.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_cancel_process-instance.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_delete_process-instance.md

**Checkpoint**: User Story 4 makes the new public behavior discoverable and synchronized.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final quality, validation, and traceability.

- [ ] T026 [P] Update implementation notes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/140-pi-limit-batch-size/quickstart.md if final commands or tests differ from the planned names
- [ ] T027 Run focused validation with `go test ./cmd -count=1` against /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd
- [ ] T028 Run repository validation with `make test` using /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion and is the MVP.
- **User Story 2 (Phase 4)**: Depends on Foundational completion and should follow or coordinate with User Story 1.
- **User Story 3 (Phase 5)**: Depends on Foundational completion and is safest after User Stories 1 and 2 establish the main flow.
- **User Story 4 (Phase 6)**: Depends on final command metadata from User Stories 1-3.
- **Polish (Phase 7)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: No dependency on other stories after Foundational.
- **User Story 2 (P2)**: Uses the same flag surface and should not change limit semantics from User Story 1.
- **User Story 3 (P3)**: Validates migration and edge behavior after the core flag behavior exists.
- **User Story 4 (P4)**: Documents the final command behavior after code and help text stabilize.

### Parallel Opportunities

- T002 and T003 can run in parallel.
- T008, T009, and T010 can run in parallel.
- T014 and T015 can run in parallel.
- T018 and T019 can run in parallel.
- T022 and T023 can run in parallel once command wording is stable.
- T026 can run in parallel with final validation preparation.

---

## Parallel Example: User Story 1

```bash
Task: "Add get process-instance multi-page --limit tests in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go"
Task: "Add search-based cancel process-instance multi-page --limit tests in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go"
Task: "Add search-based delete process-instance multi-page --limit tests in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go"
```

## Parallel Example: User Story 4

```bash
Task: "Add or update help-output assertions in affected command tests"
Task: "Update README process-instance paging examples and explanation"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Stop and validate that all affected commands cap total matched process instances across pages.

### Incremental Delivery

1. Establish the new public flag surface and validation.
2. Deliver total limit behavior for all affected commands.
3. Confirm batch-size/page-size semantics and combined usage.
4. Lock down invalid/migration behavior.
5. Update docs and run final validation.

### Parallel Team Strategy

1. One contributor handles shared flag/validation work.
2. Separate contributors can prepare get/cancel/delete limit tests in parallel.
3. Documentation can start once command help wording is stable.

---

## Notes

- [P] tasks are limited to work on different files with no dependency on unfinished tasks.
- Commit subjects for this feature must keep Conventional Commit formatting and append `#140` as the final token.
- Run `go test ./cmd -count=1` before broader `make test`.
