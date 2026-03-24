# Tasks: Review and Refactor CLI Error Code Usage

**Input**: Design documents from `/specs/19-cli-error-model/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Automated tests are required by the specification for invalid-input, local-precondition, unsupported-operation, internal, and remote-failure paths. Subprocess-based CLI tests should be used wherever `os.Exit` behavior must be asserted.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g. `US1`, `US2`, `US3`)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Capture the implementation baseline and prepare the feature artifacts for execution.

- [ ] T001 Review and refresh the planning artifacts in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/19-cli-error-model/plan.md`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/19-cli-error-model/research.md`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/19-cli-error-model/contracts/cli-command-contract.md` before code changes begin
- [ ] T002 [P] Identify and group existing failure-path coverage in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go` and adjacent `cmd/*_test.go` files so the command-family regression plan is explicit before implementation

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared failure-classification model and common command-path wiring that all user stories depend on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T003 Define the shared CLI error classes, normalized mapping helpers, and exit-code policy in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors.go`
- [ ] T004 [P] Extend shared exit-code and sentinel coverage in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/exitcode/exitcode.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/errors.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/errors.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_errors.go` only where the shared model requires explicit repository-native mappings
- [ ] T005 Route root pre-run and shared CLI bootstrap failures through the normalized error model in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_cli.go`
- [ ] T006 [P] Add focused shared-error regression tests for classification and `--no-err-codes` override behavior in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors_test.go`

**Checkpoint**: Shared classification and root wiring are ready; command-family work can now proceed by user story

---

## Phase 3: User Story 1 - Understand failures quickly as an operator (Priority: P1) 🎯 MVP

**Goal**: Make operator-facing failures consistent and understandable across commands by unifying message structure and classification for local, unsupported, malformed, and remote errors.

**Independent Test**: Run representative failing commands from read-only and state-changing families and confirm the stderr output consistently distinguishes caller mistakes, local setup issues, unsupported behavior, and remote-system failures.

### Tests for User Story 1 ⚠️

- [ ] T007 [P] [US1] Add subprocess CLI tests for representative read-only command failures in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go`
- [ ] T008 [P] [US1] Add subprocess CLI tests for representative state-changing and expectation command failures in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy_test.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect_test.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go`

### Implementation for User Story 1

- [ ] T009 [US1] Normalize operator-facing error wrapping in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processdefinition.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_resource.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_license.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_topology.go`
- [ ] T010 [US1] Normalize operator-facing validation and local-precondition failures in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_processinstance.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processdefinition.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_processinstance.go`
- [ ] T011 [P] [US1] Normalize operator-facing failure handling in utility and embed command paths in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/config_show.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/embed_list.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/embed_export.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/embed_deploy.go`

**Checkpoint**: Operators now see consistent failure classes and messages across representative commands

---

## Phase 4: User Story 2 - React correctly in automation and AI flows (Priority: P2)

**Goal**: Make exit-code behavior and machine-facing failure semantics stable enough for scripts and AI agents to distinguish invalid input, unsupported behavior, retryable failures, and permanent faults.

**Independent Test**: Execute representative failing commands with and without `--no-err-codes` and confirm exit codes and failure semantics stay consistent across command families.

### Tests for User Story 2 ⚠️

- [ ] T012 [P] [US2] Add subprocess CLI tests for exit-code stability and `--no-err-codes` behavior in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root_test.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/config_show_test.go`
- [ ] T013 [P] [US2] Add subprocess CLI tests for unsupported-version, timeout, unavailable, and conflict mappings in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect_test.go`

### Implementation for User Story 2

- [ ] T014 [US2] Apply the shared exit-code mapping to representative unsupported, timeout, unavailable, conflict, and malformed-response paths in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/` command handlers that currently fall back to generic errors
- [ ] T015 [US2] Sweep remaining command families and nested command entry points under `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/` so all existing commands use the same machine-facing failure semantics instead of family-specific ad hoc mapping
- [ ] T016 [US2] Update the public scripting and error-code documentation in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md` to reflect the shipped failure model and `--no-err-codes` behavior

**Checkpoint**: Scripts and AI callers can rely on one shared failure contract across the CLI

---

## Phase 5: User Story 3 - Maintain one coherent error model across command families (Priority: P3)

**Goal**: Leave the codebase with one maintainable failure-model boundary that future command work can extend without reintroducing drift.

**Independent Test**: Review the central classifier and command call sites and confirm that new command work would extend the shared layer instead of inventing per-command error behavior.

### Tests for User Story 3 ⚠️

- [ ] T017 [P] [US3] Add or expand maintainability-focused regression tests for shared command-validation sentinels and cross-family mappings in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_errors_test.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors_test.go`, and any newly added command-family test files under `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/`

### Implementation for User Story 3

- [ ] T018 [US3] Refactor repeated error-wrapping patterns into repository-native shared helpers in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_cli.go` without changing command structure
- [ ] T019 [US3] Update feature-facing implementation notes and contracts in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/19-cli-error-model/contracts/cli-command-contract.md`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/19-cli-error-model/quickstart.md`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/19-cli-error-model/plan.md` if the final implementation meaningfully narrows or refines the shipped model

**Checkpoint**: The shared error model is centralized enough for future maintenance without per-command drift

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and cross-story cleanup

- [ ] T020 [P] Run targeted validation from `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/19-cli-error-model/quickstart.md` with `go test ./cmd/... ./c8volt/ferrors/... -count=1`
- [ ] T021 Run full repository validation with `make test` from `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt`
- [ ] T022 [P] Regenerate CLI reference docs with `make docs` from `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt` if any Cobra help text or flag descriptions changed during implementation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies
- **Foundational (Phase 2)**: Depends on Setup completion and blocks all user story work
- **User Story 1 (Phase 3)**: Depends on Foundational completion
- **User Story 2 (Phase 4)**: Depends on User Story 1 because the shared operator-facing model must exist before machine-facing compatibility is verified across all commands
- **User Story 3 (Phase 5)**: Depends on User Stories 1 and 2 because it consolidates the shipped model and final maintainability shape
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: First deliverable and MVP; establishes the consistent operator-facing failure model
- **User Story 2 (P2)**: Builds on US1 by locking machine-facing exit semantics and documentation
- **User Story 3 (P3)**: Final consolidation pass after the public contract is implemented

### Within Each User Story

- Tests should be written or expanded before implementation changes for the same story
- Shared classifier or sentinel changes must land before sweeping command handlers that depend on them
- Documentation updates follow the implementation they describe

### Parallel Opportunities

- `T002`, `T004`, and `T006` can run in parallel during early work
- `T007` and `T008` can run in parallel because they target different command-family test files
- `T012` and `T013` can run in parallel because they cover different exit-semantics slices
- `T020` and `T022` can run in parallel once implementation is complete

---

## Parallel Example: User Story 1

```bash
# Launch representative failure-path test work in parallel:
Task: "T007 Add subprocess CLI tests for representative read-only command failures in cmd/get_test.go"
Task: "T008 Add subprocess CLI tests for representative state-changing and expectation command failures in cmd/run_test.go, cmd/deploy_test.go, cmd/cancel_test.go, cmd/delete_test.go, cmd/expect_test.go, and cmd/walk_test.go"

# Launch implementation sweeps that touch separate command groups:
Task: "T009 Normalize operator-facing error wrapping in read-only get command files"
Task: "T011 Normalize operator-facing failure handling in utility and embed command files"
```

---

## Parallel Example: User Story 2

```bash
# Launch exit-semantics coverage in parallel:
Task: "T012 Add subprocess CLI tests for exit-code stability and --no-err-codes behavior in cmd/root_test.go, cmd/get_test.go, and cmd/config_show_test.go"
Task: "T013 Add subprocess CLI tests for unsupported-version, timeout, unavailable, and conflict mappings in cmd/run_test.go, cmd/cancel_test.go, cmd/delete_test.go, and cmd/expect_test.go"
```

---

## Parallel Example: User Story 3

```bash
# Launch maintainability validation alongside doc refinement:
Task: "T017 Add or expand maintainability-focused regression tests for shared command-validation sentinels and cross-family mappings"
Task: "T019 Update feature-facing implementation notes and contracts if the final implementation refines the shipped model"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. Stop and validate the consistent operator-facing failure model before widening scope

### Incremental Delivery

1. Build the shared classifier and root wiring first
2. Land the operator-facing consistency pass across command families
3. Add machine-facing exit-code guarantees and documentation updates
4. Finish with maintainability cleanup and final validation

### Parallel Team Strategy

1. One contributor establishes the shared classifier and root wiring
2. Once Phase 2 completes, split command-family test and implementation sweeps by file groups
3. Consolidate machine-facing exit semantics and docs after the operator-facing pass is stable

---

## Notes

- All tasks use the required checklist format with task IDs and exact file paths
- `[P]` tasks are limited to work that can proceed on different files or isolated validation tracks
- User stories remain independently testable even though the feature eventually spans the full command surface
- `make test` is mandatory before the feature is complete
