# Tasks: Add Cluster License Command

**Input**: Design documents from `/specs/63-cluster-license/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Automated test tasks are REQUIRED for every story and shared change unless the plan documents a concrete exception and manual validation path.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the shared touch points and validation surface before editing user-visible command behavior

- [ ] T001 Review cluster command entry points and help surfaces in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go`
- [ ] T002 [P] Review existing cluster license service behavior and regression coverage in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/cluster/api.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/cluster/v87/service_test.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/cluster/v88/service_test.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add the base command structure that all user-story work depends on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T003 Create the base `get cluster license` Cobra command and shared handler in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_license.go`

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Retrieve cluster license details (Priority: P1) 🎯 MVP

**Goal**: Let operators run `c8volt get cluster license` and receive the existing structured cluster license payload

**Independent Test**: Run `c8volt get cluster license` with a valid config and confirm it prints the license payload successfully without changing existing cluster read behavior

### Tests for User Story 1

- [ ] T004 [US1] Add successful `get cluster license` command coverage in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go`

### Implementation for User Story 1

- [ ] T005 [US1] Implement success-path license retrieval and JSON output in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_license.go`

**Checkpoint**: User Story 1 should now be fully functional and testable independently

---

## Phase 4: User Story 2 - Understand failures and command usage (Priority: P2)

**Goal**: Make the new command discoverable in help output and ensure its failure behavior matches the rest of the CLI

**Independent Test**: Review `get` and `get cluster` help plus a failing `get cluster license` execution and confirm the command is discoverable and returns the expected failure semantics

### Tests for User Story 2

- [ ] T006 [US2] Add help-discovery coverage for `get cluster license` in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go`
- [ ] T007 [US2] Add failing `get cluster license` exit-code coverage in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go`

### Implementation for User Story 2

- [ ] T008 [US2] Refine help text, command descriptions, and failure handling consistency in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_license.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster.go`

**Checkpoint**: User Stories 1 and 2 should both work independently

---

## Phase 5: User Story 3 - Maintain confidence through tests and docs (Priority: P3)

**Goal**: Keep user-facing documentation and command references aligned with the new command so contributors and operators can validate the behavior confidently

**Independent Test**: Review README, docs index, and generated CLI docs after regeneration to confirm `c8volt get cluster license` is documented wherever cluster read commands are surfaced

### Tests for User Story 3

- [ ] T009 [US3] Validate quickstart and command-reference expectations against `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/63-cluster-license/quickstart.md` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/63-cluster-license/contracts/cli-command-contract.md`

### Implementation for User Story 3

- [ ] T010 [P] [US3] Update cluster command examples and discovery guidance in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md`
- [ ] T011 [US3] Regenerate CLI reference output in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_get.md`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_get_cluster.md`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_get_cluster_license.md`

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and repo-wide completion checks

- [ ] T012 Run targeted command and cluster service validation from `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile` using `go test ./cmd/... -race -count=1` and `go test ./internal/services/cluster/... -race -count=1`
- [ ] T013 Run documentation and repository-wide validation from `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile` using `make docs` and `make test`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational completion
- **User Story 2 (Phase 4)**: Depends on User Story 1 command wiring being present
- **User Story 3 (Phase 5)**: Depends on shipped command/help behavior from User Stories 1 and 2
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Starts after Foundational completion and defines the MVP
- **User Story 2 (P2)**: Depends on User Story 1 because help and failure coverage assume the new command already executes
- **User Story 3 (P3)**: Depends on User Stories 1 and 2 because docs must reflect the final command surface and behavior

### Within Each User Story

- Tests must be added or updated before the story is complete
- Command wiring precedes help refinements and documentation regeneration
- User-visible documentation updates precede final validation
- Story completion should be verified before moving to the next priority

### Parallel Opportunities

- T001 and T002 can proceed in parallel during setup review
- T010 can proceed in parallel with final implementation verification once help/output behavior is stable
- Final validation tasks remain sequential because `make test` depends on the completed implementation and docs state

---

## Parallel Example: User Story 3

```bash
# Once command behavior is stable, these can be split across teammates:
Task: "Update cluster command examples and discovery guidance in README.md and docs/index.md"
Task: "Regenerate CLI reference output in docs/cli/c8volt_get.md, docs/cli/c8volt_get_cluster.md, and docs/cli/c8volt_get_cluster_license.md"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. **Stop and validate**: Confirm `c8volt get cluster license` succeeds independently

### Incremental Delivery

1. Finish Setup + Foundational to establish the new command
2. Deliver User Story 1 for the MVP command path
3. Add User Story 2 to lock down help and failure semantics
4. Add User Story 3 to align docs and generated CLI references
5. Run final validation and repo-wide checks

### Parallel Team Strategy

With multiple developers:

1. One developer completes the foundational command file in `cmd/get_cluster_license.go`
2. After that lands:
   - Developer A: success and failure tests in `cmd/get_test.go`
   - Developer B: documentation updates in `README.md` and `docs/index.md`
3. Regenerate docs and run final validation together

---

## Notes

- [P] tasks touch different files or can proceed once prerequisite behavior is stable
- [US1], [US2], and [US3] labels map directly to the clarified feature stories
- The MVP scope is User Story 1 only
- Every task includes an exact file path or concrete repo target for execution
- Final validation must include `make test` before the feature is considered complete
