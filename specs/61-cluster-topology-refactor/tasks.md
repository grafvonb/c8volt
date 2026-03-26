# Tasks: Refactor Cluster Topology Command

**Input**: Design documents from `/specs/61-cluster-topology-refactor/`
**Prerequisites**: [plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/61-cluster-topology-refactor/plan.md) (required), [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/61-cluster-topology-refactor/spec.md) (required for user stories), [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/61-cluster-topology-refactor/research.md), [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/61-cluster-topology-refactor/data-model.md), [contracts/cli-command-contract.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/61-cluster-topology-refactor/contracts/cli-command-contract.md)

**Tests**: Automated test tasks are REQUIRED for every story and shared change in this feature.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare the command, test, and documentation surface for the hierarchy refactor

- [ ] T001 Inspect the current topology command and affected docs in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_topology.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_get_cluster-topology.md
- [ ] T002 Create command-level test scaffolding for topology command coverage in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_topology_test.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared Cobra structure and reusable execution path that all stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T003 Implement the shared topology command execution helper in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_topology.go
- [ ] T004 Create the `get cluster` parent command wiring in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster.go
- [ ] T005 [P] Add foundational command-tree tests for `get` and `get cluster` help behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_topology_test.go

**Checkpoint**: Foundation ready - user story implementation can now begin in priority order or parallel where noted

---

## Phase 3: User Story 1 - Use the new nested command path (Priority: P1) 🎯 MVP

**Goal**: Deliver `c8volt get cluster topology` as the preferred command path with preserved topology behavior

**Independent Test**: Run `c8volt get cluster topology` and confirm it produces the same output and exit behavior as the pre-refactor `c8volt get cluster-topology` flow for both success and failure cases.

### Tests for User Story 1

- [ ] T006 [P] [US1] Add command tests for successful and failing `c8volt get cluster topology` execution in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_topology_test.go
- [ ] T007 [P] [US1] Verify topology service regression coverage still applies to the nested path in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/cluster/factory_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/cluster/v87/service_test.go

### Implementation for User Story 1

- [ ] T008 [US1] Move the preferred topology command definition to `Use: "topology"` under the new parent in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_topology.go
- [ ] T009 [US1] Register the preferred `topology` command under `get cluster` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster.go
- [ ] T010 [US1] Ensure the nested command preserves inherited flags and output behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_topology.go

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Keep the legacy command working during migration (Priority: P2)

**Goal**: Preserve `c8volt get cluster-topology` as a quiet compatibility path that routes to the same behavior as the preferred nested command

**Independent Test**: Run `c8volt get cluster-topology` after the refactor and confirm it still succeeds or fails exactly like the new command path, without runtime deprecation output.

### Tests for User Story 2

- [ ] T011 [P] [US2] Add compatibility-path tests for `c8volt get cluster-topology` behavior parity and no runtime warning in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_topology_test.go

### Implementation for User Story 2

- [ ] T012 [US2] Rework the legacy `cluster-topology` command entry to call the shared topology execution path in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_topology.go
- [ ] T013 [US2] Preserve or explicitly review existing legacy aliases for `cluster-topology` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_topology.go
- [ ] T014 [US2] Encode help-only deprecation guidance for the legacy command without changing runtime output in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_topology.go

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Understand the supported command behavior (Priority: P3)

**Goal**: Make the new hierarchy discoverable in help and documentation while documenting the legacy path as deprecated but supported

**Independent Test**: Review `c8volt get` help, `c8volt get cluster` help, README examples, and generated CLI docs to confirm the preferred path is discoverable and the legacy path is documented as deprecated.

### Tests for User Story 3

- [ ] T015 [P] [US3] Add help-output assertions for `c8volt get` and `c8volt get cluster` discoverability in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_topology_test.go

### Implementation for User Story 3

- [ ] T016 [US3] Update user-facing command examples to prefer the nested path in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md
- [ ] T017 [US3] Update generated CLI source docs to show `get cluster` and deprecate `get cluster-topology` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_get.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_get_cluster-topology.md
- [ ] T018 [US3] Generate or update nested command CLI docs for `c8volt get cluster` and `c8volt get cluster topology` under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification and repository-wide consistency checks

- [ ] T019 [P] Run quickstart validation for both command paths using the steps in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/61-cluster-topology-refactor/quickstart.md
- [ ] T020 Run targeted command and cluster regression tests with `go test ./cmd/... -race -count=1` and `go test ./internal/services/cluster/... -race -count=1`
- [ ] T021 Regenerate user-facing CLI documentation with `make docs` for /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/
- [ ] T022 Run the repository validation command set, including `make test`, for changes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/ and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: Depend on Foundational phase completion
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Starts after Foundational - this is the MVP and creates the preferred command path
- **User Story 2 (P2)**: Starts after User Story 1 because it reuses the shared topology handler and new command structure
- **User Story 3 (P3)**: Starts after User Stories 1 and 2 because docs/help must reflect the final preferred and compatibility paths

### Within Each User Story

- Tests are added or updated before story sign-off
- Shared Cobra wiring before nested or compatibility command registration
- Command behavior before documentation updates
- Generated docs before final repository validation

### Parallel Opportunities

- T005 can run in parallel with T003 and T004 once foundational design is clear
- T006 and T007 can run in parallel within User Story 1
- T016 and T017 can run in parallel within User Story 3
- T019 can run in parallel with targeted validation once implementation is complete

---

## Parallel Example: User Story 1

```bash
# Launch User Story 1 verification tasks together:
Task: "Add command tests for successful and failing c8volt get cluster topology execution in cmd/get_cluster_topology_test.go"
Task: "Verify topology service regression coverage still applies to the nested path in internal/services/cluster/factory_test.go and internal/services/cluster/v87/service_test.go"
```

---

## Parallel Example: User Story 3

```bash
# Launch documentation updates together after command behavior is stable:
Task: "Update user-facing command examples to prefer the nested path in README.md and docs/index.md"
Task: "Update generated CLI source docs to show get cluster and deprecate get cluster-topology in docs/cli/c8volt_get.md and docs/cli/c8volt_get_cluster-topology.md"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Run the new nested command tests and confirm `c8volt get cluster topology` works independently

### Incremental Delivery

1. Add the new nested command path and validate it as the MVP
2. Add the legacy compatibility path and validate parity
3. Update help and docs for discoverability and deprecation guidance
4. Finish with docs generation and `make test`

### Parallel Team Strategy

With multiple developers:

1. One developer completes Setup + Foundational work
2. After foundational work lands:
   - Developer A: User Story 1 verification and nested command refinement
   - Developer B: User Story 2 compatibility-path behavior
   - Developer C: User Story 3 documentation updates after command behavior stabilizes

---

## Notes

- [P] tasks touch different files or can be validated independently
- [US1], [US2], and [US3] map directly to the user stories in [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/61-cluster-topology-refactor/spec.md)
- Keep implementation repository-native: Cobra command composition, no new dependencies, no runtime deprecation warning
- Do not edit generated site output under `docs/_site/`; regenerate docs from source instead
