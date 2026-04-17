# Tasks: Preserve Concise CLI Error Breadcrumbs

**Input**: Design documents from `/specs/112-error-context-dedup/`
**Prerequisites**: [plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/plan.md) (required), [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/spec.md) (required for user stories), [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/research.md), [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/data-model.md), [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/quickstart.md), [contracts/cli-error-rendering.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/contracts/cli-error-rendering.md)

**Tests**: Automated tests are REQUIRED for this feature because the specification explicitly requires representative regression coverage for each affected duplication-pattern family and preservation of existing classification and exit behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g. `US1`, `US2`, `US3`)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Lock in the audit scope, affected pattern families, and existing regression anchors before code changes begin.

- [x] T001 Refresh the implementation boundary and affected pattern-family inventory in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/plan.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/research.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/contracts/cli-error-rendering.md before code changes begin
- [x] T002 [P] Inventory current duplicated wrapper seams and target owner layers in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/
- [x] T003 [P] Confirm representative regression anchors for each duplication-pattern family in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors_test.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared rendering contract and common helper expectations that all user stories build on.

**⚠️ CRITICAL**: No user story work should begin until this phase is complete.

- [x] T004 Define the authoritative prefix-preserving dedup contract and breadcrumb-shortening rules in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/contracts/cli-error-rendering.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/data-model.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/plan.md
- [x] T005 Keep shared classification and exit behavior fixed while tightening helper expectations in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_cli.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_errors.go
- [x] T006 [P] Add foundational regression coverage for unchanged shared classification and exit behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/bootstrap_errors_test.go

**Checkpoint**: The repository has one explicit rendering contract, and shared classification/exit behavior is protected before story-specific cleanup starts.

---

## Phase 3: User Story 1 - Keep CLI failures readable (Priority: P1) 🎯 MVP

**Goal**: Remove duplicated root-failure wording from affected CLI paths while preserving the shared class prefix and a readable final message.

**Independent Test**: Run representative duplicated CLI error paths from each affected pattern family and verify the rendered error keeps the existing shared class prefix and shows the root failure detail once.

### Tests for User Story 1

- [x] T007 [P] [US1] Add traversal-family regression tests that assert preserved class prefix and single root failure detail in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker_test.go
- [x] T008 [P] [US1] Add mutation-family regression tests that assert deduplicated cancel/delete/wait failure output in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go
- [x] T009 [P] [US1] Add single-resource fetch regression tests for deduplicated CLI output in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go, and related `get_*` test files under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/

### Implementation for User Story 1

- [x] T010 [US1] Refactor process-instance lookup and traversal wrappers to keep only stage context above the root detail in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service.go
- [x] T011 [US1] Refactor process-instance mutation and wait follow-up wrappers to stop repeating keys or failure meaning in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter.go
- [x] T012 [US1] Sweep representative CLI fetch and orchestration wrappers so they no longer restate already-complete lower-layer failures in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processdefinition.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_resource.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_license.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_topology.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/client.go

**Checkpoint**: User Story 1 is independently testable: affected CLI failures are shorter and cleaner, but the shared class prefix is unchanged.

---

## Phase 4: User Story 2 - Preserve where the failure happened (Priority: P2)

**Goal**: Keep breadcrumb context useful and ordered while allowing equivalent shortening that removes noise without losing stage meaning.

**Independent Test**: Trigger multi-step failures across representative pattern families and confirm the final error still identifies the same stages in order, even where breadcrumb wording becomes shorter.

### Tests for User Story 2

- [x] T013 [P] [US2] Add helper-level tests for ordered and equivalent breadcrumb preservation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go
- [x] T014 [P] [US2] Add command-level regression tests for recognizable breadcrumb stages after shortening in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go

### Implementation for User Story 2

- [x] T015 [US2] Adjust breadcrumb wording in shared traversal and process-instance wrappers to preserve equivalent stage meaning with less noise in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service.go
- [x] T016 [US2] Align command-surface wrappers with the equivalent-breadcrumb contract in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go

**Checkpoint**: User Story 2 is independently testable: deduplicated failures still show the same identifiable call-path stages in order.

---

## Phase 5: User Story 3 - Keep failure semantics unchanged (Priority: P3)

**Goal**: Prove the repo-wide cleanup preserves existing classification, exit behavior, and shared class prefixes across matching not-found and non-not-found patterns.

**Independent Test**: Exercise representative not-found and non-not-found failure paths and confirm the same normalized class prefix and exit behavior remain intact while the wrapped detail text is cleaner.

### Tests for User Story 3

- [ ] T017 [P] [US3] Add cross-class regression tests for preserved normalized prefixes and unchanged exit behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, and representative non-not-found command test files under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/
- [ ] T018 [P] [US3] Add representative regression tests for non-not-found duplication-pattern families if the audit changes them in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_license_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_topology_test.go, or the matching command-family test files under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/

### Implementation for User Story 3

- [ ] T019 [US3] Sweep any remaining matched non-not-found duplication patterns without changing shared classification behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/client.go
- [ ] T020 [US3] Refresh the shipped rendering contract and implementation notes after the final audit in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/research.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/data-model.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/quickstart.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/contracts/cli-error-rendering.md

**Checkpoint**: User Story 3 is independently testable: the cleaned-up output preserves the same class prefixes and exit behavior across covered pattern families.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish validation, document any user-facing wording impact if needed, and leave the feature artifacts aligned with the shipped result.

- [ ] T021 [P] Review whether user-facing docs need wording updates and, only if required by the final audit, update /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md
- [ ] T022 Run focused validation with `go test ./c8volt/ferrors -count=1`, `go test ./internal/services/processinstance/... -count=1`, and `go test ./cmd -count=1` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt
- [ ] T023 Run repository validation with `make test` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user story work.
- **User Story 1 (Phase 3)**: Depends on Foundational completion and is the MVP slice.
- **User Story 2 (Phase 4)**: Depends on User Story 1 because breadcrumb cleanup builds on the deduplicated root-detail ownership established there.
- **User Story 3 (Phase 5)**: Depends on User Stories 1 and 2 because cross-class preservation and final artifact refresh should follow the settled rendering behavior.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: No dependency on later stories after Foundational work is complete.
- **User Story 2 (P2)**: Depends on User Story 1’s root-detail dedup behavior and refines breadcrumb quality without changing the core contract.
- **User Story 3 (P3)**: Depends on the final rendering behavior established by User Stories 1 and 2.

### Within Each User Story

- Add or update regression tests before considering the story complete.
- Helper/service seam updates before command-surface sweeps that depend on them.
- Artifact and documentation refresh only after code and tests stabilize.

### Parallel Opportunities

- `T002` and `T003` can run in parallel.
- `T006` can run in parallel with late Foundational review once the contract is settled.
- `T007`, `T008`, and `T009` can run in parallel.
- `T013` and `T014` can run in parallel.
- `T017` and `T018` can run in parallel.
- `T021` can run in parallel with focused validation once implementation is stable.

---

## Parallel Example: User Story 1

```bash
# Prepare representative readability regressions in parallel:
Task: "Add traversal-family regression tests that assert preserved class prefix and single root failure detail in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker_test.go"
Task: "Add mutation-family regression tests that assert deduplicated cancel/delete/wait failure output in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go"
Task: "Add single-resource fetch regression tests for deduplicated CLI output in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go, and related get_* test files under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/"
```

---

## Parallel Example: User Story 2

```bash
# Prepare breadcrumb-preservation coverage in parallel:
Task: "Add helper-level tests for ordered and equivalent breadcrumb preservation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go"
Task: "Add command-level regression tests for recognizable breadcrumb stages after shortening in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Stop and validate that representative CLI failures are cleaner while the shared class prefix remains unchanged.

### Incremental Delivery

1. Finish Setup + Foundational once.
2. Deliver User Story 1 as the MVP for readable, deduplicated CLI failures.
3. Add User Story 2 to preserve equivalent breadcrumb quality and stage recognizability.
4. Add User Story 3 to prove cross-class prefix preservation and refresh feature artifacts.
5. Finish with focused validation and full `make test`.

### Parallel Team Strategy

1. One contributor finalizes Setup + Foundational work.
2. After Foundational is complete:
   - Contributor A: User Story 1 helper/service dedup and regression work.
   - Contributor B: User Story 2 breadcrumb-quality regressions and command-surface cleanup.
   - Contributor C: User Story 3 cross-class regression work and artifact refresh once behavior stabilizes.
3. Finish with shared validation and optional doc review.

---

## Notes

- `[P]` tasks are limited to work on different files or isolated validation tracks.
- `[US1]`, `[US2]`, and `[US3]` map directly to the user stories in [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/spec.md).
- This feature’s commit subjects must keep Conventional Commit formatting and append `#112` as the final token.
- Run `make test` before committing, per repository rules.
