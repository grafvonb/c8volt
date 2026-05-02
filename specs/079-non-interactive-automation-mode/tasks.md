# Tasks: Define Non-Interactive Automation Mode

**Input**: Design documents from `/specs/079-non-interactive-automation-mode/`
**Prerequisites**: [plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/079-non-interactive-automation-mode/plan.md) (required), [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/079-non-interactive-automation-mode/spec.md) (required for user stories), [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/079-non-interactive-automation-mode/research.md), [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/079-non-interactive-automation-mode/data-model.md), [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/079-non-interactive-automation-mode/quickstart.md), [contracts/cli-automation-mode-contract.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/079-non-interactive-automation-mode/contracts/cli-automation-mode-contract.md)

**Tests**: Automated test tasks are REQUIRED for this feature because the specification explicitly requires targeted coverage for representative prompting and state-changing commands plus preserved human-mode behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g. `US1`, `US2`, `US3`)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the repo-native seams that the automation-mode rollout will extend.

- [x] T001 Inventory the current root flags, config bindings, and machine-facing guidance in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md
- [x] T002 [P] Inventory the current prompt, paging, and result-envelope seams in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_cli.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_contract.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared automation-mode model and discovery metadata before story work begins.

**⚠️ CRITICAL**: No user story work should begin until this phase is complete.

- [x] T003 Define the root automation flag, effective-mode helper, and config binding in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go
- [x] T004 Extend shared command metadata and discovery models for automation support in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/command_contract.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities.go
- [x] T005 [P] Add foundational regression coverage for the root automation flag and automation-support discovery metadata in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/command_contract_test.go

**Checkpoint**: The repository has one shared automation-mode flag and one truthful discovery model before representative command rollout starts.

---

## Phase 3: User Story 1 - Run Commands Safely Without Prompts (Priority: P1) 🎯 MVP

**Goal**: Let automation callers run representative commands under one dedicated flag without hanging on confirmations or paging continuation, while unsupported commands fail explicitly.

**Independent Test**: Run representative state-changing and paged commands with the automation flag and verify they either proceed without waiting for terminal input or fail immediately with an actionable unsupported-command error.

### Tests for User Story 1

- [x] T006 [P] [US1] Add regression tests for automation-mode confirmation and unsupported-command rejection in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect_test.go
- [x] T007 [P] [US1] Add regression tests for automation-mode paging continuation behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go

### Implementation for User Story 1

- [x] T008 [US1] Implement shared automation-mode prompt handling in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_cli.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [x] T009 [US1] Wire implicit automation-mode confirmation into representative state-changing commands in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processdefinition.go
- [x] T010 [US1] Implement explicit automation-mode support and rejection behavior for representative read and observe commands in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_processinstance.go

**Checkpoint**: User Story 1 is independently testable: the automation flag prevents blocking on supported prompt flows and rejects unsupported command paths explicitly.

---

## Phase 4: User Story 2 - Combine Automation Mode With Machine Output (Priority: P2)

**Goal**: Make automation mode work cleanly with the existing machine-readable contract so JSON results stay parseable and `--no-wait` still means accepted-but-not-yet-complete work.

**Independent Test**: Run representative read and write commands with `--automation --json`, then verify stdout carries only the machine-readable result and automation plus `--no-wait` yields `accepted` instead of implied completion.

### Tests for User Story 2

- [x] T011 [P] [US2] Add regression tests for clean stdout and shared-envelope behavior in automation JSON mode in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities_test.go
- [x] T012 [P] [US2] Add regression tests for automation-mode `accepted` outcomes with `--no-wait` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go

### Implementation for User Story 2

- [x] T013 [US2] Extend shared result-envelope and output-channel helpers for automation mode in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_contract.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_rendermode.go
- [x] T014 [US2] Wire representative automation JSON behavior into discovery and read flows in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_processinstance.go
- [x] T015 [US2] Wire automation-mode `--no-wait` accepted-result behavior into representative state-changing commands in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy_processdefinition.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go

**Checkpoint**: User Story 2 is independently testable: automation callers can combine the new flag with JSON and `--no-wait` while keeping deterministic machine-readable semantics.

---

## Phase 5: User Story 3 - Preserve Human CLI Workflows (Priority: P3)

**Goal**: Keep current human-oriented command behavior intact while documenting the new automation contract as an intentional extension of the existing CLI.

**Independent Test**: Compare representative commands with and without the automation flag and confirm current human-friendly help, prompt flows, and output remain available outside automation mode.

### Tests for User Story 3

- [x] T016 [P] [US3] Add regression tests proving human-mode output and prompt behavior stay intact without the automation flag in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go
- [x] T017 [P] [US3] Add discovery/help-text regression coverage for the documented automation contract in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root_test.go

### Implementation for User Story 3

- [x] T018 [US3] Update root and representative command help text for the automation contract in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go
- [x] T019 [US3] Update user-facing automation guidance in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md
- [x] T020 [US3] Regenerate CLI reference documentation under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/ from the updated Cobra metadata

**Checkpoint**: User Story 3 is independently testable: the automation contract is clearly documented and the human CLI still behaves as before when automation mode is not enabled.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish validation and refresh the feature artifacts around the final implementation.

- [x] T021 [P] Refresh implementation and validation notes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/079-non-interactive-automation-mode/plan.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/079-non-interactive-automation-mode/research.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/079-non-interactive-automation-mode/quickstart.md after the final contract shape lands
- [x] T022 Run focused automation-mode regression coverage with `go test ./cmd -count=1` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt
- [x] T023 Run documentation regeneration with `make docs` and `make docs-content` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt
- [x] T024 Run repository validation with `make test` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion and is the MVP slice.
- **User Story 2 (Phase 4)**: Depends on Foundational completion and benefits from User Story 1’s prompt-support rollout.
- **User Story 3 (Phase 5)**: Depends on User Stories 1 and 2 so the docs and compatibility checks describe final behavior.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: No dependency on later stories after Foundational work is complete.
- **User Story 2 (P2)**: Depends on the shared automation-mode foundation and should build on the supported command set established in US1.
- **User Story 3 (P3)**: Depends on the shipped automation behavior from US1 and US2 so docs and human-mode compatibility checks match reality.

### Within Each User Story

- Add or update regression tests before calling the story complete.
- Shared helpers and metadata before command-specific wiring.
- Runtime behavior before docs regeneration.
- Focused command validation before repository-wide `make test`.

### Parallel Opportunities

- `T002` can run in parallel with `T001`.
- `T005` can run in parallel after `T003` and `T004`.
- `T006` and `T007` can run in parallel.
- `T011` and `T012` can run in parallel.
- `T016` and `T017` can run in parallel.
- `T021` can run in parallel with focused validation once implementation stabilizes.

---

## Parallel Example: User Story 1

```bash
# Prepare automation prompt coverage in parallel:
Task: "Add regression tests for automation-mode confirmation and unsupported-command rejection in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect_test.go"
Task: "Add regression tests for automation-mode paging continuation behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go"
```

---

## Parallel Example: User Story 2

```bash
# Prepare automation JSON coverage in parallel:
Task: "Add regression tests for clean stdout and shared-envelope behavior in automation JSON mode in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities_test.go"
Task: "Add regression tests for automation-mode no-wait accepted outcomes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Stop and validate that supported commands no longer block in automation mode and unsupported commands fail explicitly.

### Incremental Delivery

1. Finish Setup + Foundational once.
2. Deliver User Story 1 as the safe non-interactive execution MVP.
3. Add User Story 2 to make automation mode work predictably with JSON and `--no-wait`.
4. Add User Story 3 to lock in documentation and preserved human workflows.
5. Finish with focused validation, docs regeneration, and repository-wide `make test`.

### Parallel Team Strategy

1. One contributor finalizes Setup + Foundational work.
2. After Foundational is complete:
   - Contributor A: User Story 1 prompt and unsupported-command behavior.
   - Contributor B: User Story 2 JSON and accepted-outcome behavior.
   - Contributor C: User Story 3 help text and docs once runtime behavior stabilizes.
3. Finish with shared validation and docs regeneration.

---

## Notes

- `[P]` tasks are limited to work on different files or isolated validation tracks.
- `[US1]`, `[US2]`, and `[US3]` map directly to the user stories in [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/079-non-interactive-automation-mode/spec.md).
- This feature’s commit subjects must keep Conventional Commit formatting and append `#79` as the final token.
- Run `make test` before committing, per repository rules.
