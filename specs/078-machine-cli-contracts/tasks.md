# Tasks: Define Machine-Readable CLI Contracts

**Input**: Design documents from `/specs/078-machine-cli-contracts/`
**Prerequisites**: [plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/plan.md) (required), [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/spec.md) (required for user stories), [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/research.md), [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/data-model.md), [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/quickstart.md), [contracts/cli-machine-contract.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/contracts/cli-machine-contract.md)

**Tests**: Automated tests are REQUIRED for this feature because the specification explicitly requires representative contract coverage for every in-scope command family plus proof that JSON outcomes stay aligned with the existing exit-code contract.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g. `US1`, `US2`, `US3`)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Lock in the current command metadata, JSON rendering seams, and documentation anchors before shared contract code is introduced.

- [x] T001 Inventory the current command tree, root flags, and machine-facing render seams in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_rendermode.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_get.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/research.md
- [x] T002 [P] Inventory representative command-family payload and outcome seams in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy_processdefinition.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go
- [x] T003 [P] Confirm current automation-facing docs and generated CLI reference anchors in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/use-cases.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared machine-contract building blocks that all user stories depend on.

**⚠️ CRITICAL**: No user story work should begin until this phase is complete.

- [x] T004 Define the shared capability and result-envelope types in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/command_contract.go and align their semantics with /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/contracts/cli-machine-contract.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/data-model.md
- [x] T005 Implement shared command metadata and contract-support helpers in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/command_contract.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go
- [x] T006 [P] Implement shared result-envelope rendering helpers and outcome-mapping utilities in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_contract.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors.go
- [x] T007 [P] Add foundational regression coverage for capability metadata, outcome mapping, and exit-code alignment in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/command_contract_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors_test.go

**Checkpoint**: The repository has one shared machine-contract model, one shared result-envelope helper, and protected outcome-to-exit alignment before story-specific rollout starts.

---

## Phase 3: User Story 1 - Discover Safe Command Contracts (Priority: P1) 🎯 MVP

**Goal**: Give machine consumers one dedicated top-level discovery command that exposes command paths, flags, output modes, mutation type, and contract support status without scraping human help text.

**Independent Test**: Run `c8volt capabilities --json` and verify it returns machine-readable metadata for representative top-level and nested commands, including supported flags, output modes, read-only/state-changing classification, and `full` / `limited` / `unsupported` contract support.

### Tests for User Story 1

- [x] T008 [P] [US1] Add discovery-command regression tests for top-level and nested command metadata in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities_test.go
- [x] T009 [P] [US1] Add command-metadata coverage for flags, output modes, and support-status reporting in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/command_contract_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go

### Implementation for User Story 1

- [x] T010 [US1] Implement the dedicated top-level discovery command in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities.go and register it from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go
- [x] T011 [US1] Populate representative capability metadata for `get`, `run`, `expect`, `walk`, `deploy`, `delete`, and `cancel` command families in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities.go and the related command files under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/
- [x] T012 [US1] Mark unsupported and limited machine-contract support honestly in the discovery surface for non-rolled-out commands in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/command_contract.go

**Checkpoint**: User Story 1 is independently testable: automation can discover the command surface and support status from one top-level machine-readable command.

---

## Phase 4: User Story 2 - Receive Stable Machine Results (Priority: P2)

**Goal**: Roll out the shared machine-readable result envelope across representative command families so automation can distinguish `succeeded`, `accepted`, `invalid`, and `failed` while still receiving the existing command-family payloads.

**Independent Test**: Execute representative `get`, `run`, `expect`, `walk`, `deploy`, `delete`, and `cancel` commands in JSON mode under success, accepted-work, validation-failure, and remote-failure conditions, then verify the shared result envelope carries the correct outcome, payload, and exit-code alignment.

### Tests for User Story 2

- [x] T013 [P] [US2] Add `get` and `walk` result-envelope regression tests for confirmed successful read-only flows in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go
- [x] T014 [P] [US2] Add `run`, `deploy`, `delete`, and `cancel` regression tests for `accepted` versus `succeeded` behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go
- [x] T015 [P] [US2] Add `invalid` and `failed` envelope regression tests with exit-code alignment in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors_test.go

### Implementation for User Story 2

- [x] T016 [US2] Integrate the shared result envelope into read-only command rendering in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_rendermode.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_get.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_walk.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/version.go
- [x] T017 [US2] Integrate the shared result envelope into representative state-changing command families in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy_processdefinition.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processdefinition.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go
- [x] T018 [US2] Align `accepted`, `invalid`, and `failed` envelope behavior with repository-native `--no-wait` and `ferrors` semantics in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_contract.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_cli.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors.go

**Checkpoint**: User Story 2 is independently testable: representative commands return the shared machine envelope with stable outcomes and unchanged process-level exit behavior.

---

## Phase 5: User Story 3 - Keep Human CLI Behavior Intact (Priority: P3)

**Goal**: Preserve the current human-facing CLI structure and operator guidance while documenting the new automation contract and proving that human-oriented command behavior still works.

**Independent Test**: Review representative commands in both normal and machine-readable modes, then confirm the existing command taxonomy, human help text, and operator docs still make sense while the new automation contract is clearly documented.

### Tests for User Story 3

- [x] T019 [P] [US3] Add compatibility regression tests that prove plain-text and keys-only behavior remain intact for representative commands in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/version_test.go
- [x] T020 [P] [US3] Add discovery/help-text regression coverage for the new top-level command and unchanged CLI taxonomy in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root_test.go

### Implementation for User Story 3

- [x] T021 [US3] Update machine-contract and automation guidance in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/use-cases.md
- [x] T022 [US3] Update Cobra help text for the discovery command and affected machine-readable guidance in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go, and representative command files under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/
- [x] T023 [US3] Regenerate generated CLI reference output under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/ from the updated Cobra command metadata

**Checkpoint**: User Story 3 is independently testable: the machine contract is documented and discoverable, while the human-facing CLI remains recognizable and compatible.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish validation, refresh planning artifacts, and leave the feature ready for implementation handoff or direct execution.

- [ ] T024 [P] Refresh implementation and verification notes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/quickstart.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/research.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/plan.md after the final contract shape settles
- [ ] T025 Run focused machine-contract validation with `go test ./c8volt/ferrors -count=1` and `go test ./cmd -count=1` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt
- [ ] T026 Run documentation regeneration validation with `make docs` and `make docs-content` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile
- [ ] T027 Run repository validation with `make test` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion and is the MVP slice.
- **User Story 2 (Phase 4)**: Depends on Foundational completion and is independently testable once the shared contract helpers exist.
- **User Story 3 (Phase 5)**: Depends on User Stories 1 and 2 because documentation and compatibility validation should target the final rolled-out behavior.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: No dependency on later stories after Foundational work is complete.
- **User Story 2 (P2)**: No functional dependency on User Story 1 once shared contract helpers are in place, though implementation will likely reuse the same metadata and envelope seams.
- **User Story 3 (P3)**: Depends on the discovery command and representative envelope rollout so docs and compatibility guidance reflect real shipped behavior.

### Within Each User Story

- Add or update regression tests before considering the story complete.
- Shared metadata and helper seams before command-family rollouts that depend on them.
- Read-only and state-changing command wiring before docs updates that describe final behavior.
- Regenerated docs only after Cobra help text is final.

### Parallel Opportunities

- `T002` and `T003` can run in parallel.
- `T006` and `T007` can run in parallel after the core contract types are defined.
- `T008` and `T009` can run in parallel.
- `T013`, `T014`, and `T015` can run in parallel.
- `T019` and `T020` can run in parallel.
- `T024` can run in parallel with focused validation once implementation is stable.

---

## Parallel Example: User Story 1

```bash
# Prepare discovery coverage in parallel:
Task: "Add discovery-command regression tests for top-level and nested command metadata in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities_test.go"
Task: "Add command-metadata coverage for flags, output modes, and support-status reporting in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/command_contract_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go"
```

---

## Parallel Example: User Story 2

```bash
# Prepare representative envelope coverage in parallel:
Task: "Add get and walk result-envelope regression tests for confirmed successful read-only flows in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go"
Task: "Add run, deploy, delete, and cancel regression tests for accepted versus succeeded behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go"
Task: "Add invalid and failed envelope regression tests with exit-code alignment in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Stop and validate that machine consumers can discover the command surface through one top-level machine-readable command.

### Incremental Delivery

1. Finish Setup + Foundational once.
2. Deliver User Story 1 as the discovery MVP.
3. Add User Story 2 to roll out the shared result envelope across representative command families.
4. Add User Story 3 to lock in human-CLI compatibility and operator-facing documentation.
5. Finish with focused validation, docs regeneration, and full `make test`.

### Parallel Team Strategy

1. One contributor finalizes Setup + Foundational work.
2. After Foundational is complete:
   - Contributor A: User Story 1 discovery command and metadata coverage.
   - Contributor B: User Story 2 representative envelope rollout and outcome tests.
   - Contributor C: User Story 3 documentation/help-text work once behavior stabilizes.
3. Finish with shared validation and docs regeneration.

---

## Notes

- `[P]` tasks are limited to work on different files or isolated validation tracks.
- `[US1]`, `[US2]`, and `[US3]` map directly to the user stories in [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/spec.md).
- This feature’s commit subjects must keep Conventional Commit formatting and append `#78` as the final token.
- Run `make test` before committing, per repository rules.
