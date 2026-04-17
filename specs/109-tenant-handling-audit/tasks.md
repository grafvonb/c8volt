# Tasks: Harden Tenant Handling Across Tenant-Aware Commands

**Input**: Design documents from `/specs/109-tenant-handling-audit/`
**Prerequisites**: [plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/plan.md) (required), [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/spec.md) (required for user stories), [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/research.md), [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/data-model.md), [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/quickstart.md), [contracts/tenant-handling.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/contracts/tenant-handling.md)

**Tests**: Automated tests are REQUIRED for this feature because the specification explicitly requires regression coverage across all tenant-aware command families, versioned service behavior, mixed direct-get plus search flows, and explicit flag/env/profile/base-config tenant sources.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Lock in the exact tenant-aware command surface, version support boundary, and current test seams before shared implementation begins.

- [x] T001 Inventory tenant-aware command families and shared flow seams in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/research.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/quickstart.md
- [x] T002 [P] Confirm version support and current process-instance factory boundaries in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/toolx/version.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory_test.go
- [x] T003 [P] Inspect existing tenant-related regression seams in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter_test.go, and tenant-aware command tests under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared tenant contract and service/helper seams that every user story builds on.

**⚠️ CRITICAL**: No user story work should begin until this phase is complete.

- [x] T004 Define the authoritative tenant-handling contract and unsupported-version boundaries in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/contracts/tenant-handling.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/research.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/plan.md
- [x] T005 Refactor shared process-instance service seams to support tenant-safe direct-get alternatives and explicit unsupported outcomes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/api.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/api.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go
- [x] T006 [P] Add or update shared helper support for tenant-safe filters and response normalization in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/common/filter.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/common/response.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/common/deps.go
- [x] T007 [P] Add foundational regression coverage for effective tenant source resolution in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/app_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go

**Checkpoint**: The repository has one explicit tenant contract, shared service interfaces can represent tenant-safe and unsupported outcomes, and tenant-source resolution seams are ready for story-specific work.

---

## Phase 3: User Story 1 - Keep Tenant-Scoped Lookups Safe (Priority: P1) 🎯 MVP

**Goal**: Ensure tenant-aware direct lookups return only in-tenant resources and treat supported wrong-tenant requests as the same `not found` outcome as absent resources.

**Independent Test**: Run tenant-aware direct lookup command paths with matching-tenant, wrong-tenant, default-tenant, and absent-resource keys and verify supported wrong-tenant cases look identical to `not found`.

### Tests for User Story 1

- [x] T008 [P] [US1] Add `v88` service tests for tenant-safe direct lookup and supported wrong-tenant `not found` behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go
- [x] T009 [P] [US1] Add `v87` service tests for supported direct-lookup cases and explicit unsupported outcomes where tenant-safe retrieval is impossible in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go
- [x] T010 [P] [US1] Add command regression tests for direct process-instance lookup with explicit `--tenant`, environment-derived tenant, profile-derived tenant, and base-config-derived tenant in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go

### Implementation for User Story 1

- [x] T011 [US1] Implement tenant-safe direct lookup and state lookup behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/contract.go
- [x] T012 [US1] Audit and narrow `v87` direct lookup behavior to supported tenant-safe cases plus exact unsupported outcomes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/contract.go
- [x] T013 [US1] Normalize facade and command handling for tenant-safe `not found` and unsupported outcomes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go

**Checkpoint**: User Story 1 is independently testable: direct tenant-aware lookups are safe, supported wrong-tenant requests resolve as `not found`, and unsafe `v87` segments fail explicitly.

---

## Phase 4: User Story 2 - Preserve Tenant Boundaries Through Multi-Step Flows (Priority: P2)

**Goal**: Keep tenant scope intact across walker, waiter, cancel, delete, and mixed direct-get plus search flows so no follow-up step crosses tenant boundaries.

**Independent Test**: Run walk, ancestry, descendants, wait, cancel, and delete flows with matching-tenant and wrong-tenant keys and verify mixed-flow operations either stay tenant-safe or fail at the exact unsupported segment.

### Tests for User Story 2

- [x] T014 [P] [US2] Add walker and waiter regression tests for tenant-safe ancestry, descendants, family, and state polling behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter_test.go
- [x] T015 [P] [US2] Add versioned service tests for mixed search plus direct-get flows, cancel preflight, and delete preflight behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go
- [x] T016 [P] [US2] Add command-family regression tests for `walk`, `cancel`, `delete`, and `run` tenant propagation across flag/env/profile/base-config sources in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go

### Implementation for User Story 2

- [x] T017 [US2] Implement tenant-safe traversal and direct-child lookup behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go
- [x] T018 [US2] Implement tenant-aware wait, cancel, and delete preflight handling with narrowly scoped unsupported outcomes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go
- [x] T019 [US2] Align tenant-aware command flows with the service-layer contract in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect.go

**Checkpoint**: User Story 2 is independently testable: mixed flows preserve tenant boundaries end to end, and unsupported `v87` behavior is constrained to exact unsafe operations.

---

## Phase 5: User Story 3 - Make Version Behavior Explicit and Predictable (Priority: P3)

**Goal**: Make the supported-version matrix, tenant contract, and `8.9` planning boundary explicit in tests, docs, and planning artifacts so future work stays aligned.

**Independent Test**: Review service factory behavior, versioned tests, and operator-facing guidance, then verify maintainers can identify the supported `v87`/`v88` contract, exact unsupported `v87` segments, and the current no-runtime-support status for `8.9`.

### Tests for User Story 3

- [ ] T020 [P] [US3] Add or update factory and version-matrix tests for supported runtime versions and unsupported `8.9` process-instance service behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/app_test.go
- [ ] T021 [P] [US3] Add regression coverage that proves unsupported `v87` outcomes are scoped to exact unsafe segments and do not block safe command-family behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/*test.go

### Implementation for User Story 3

- [ ] T022 [US3] Update version support and tenant contract documentation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/research.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/data-model.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/quickstart.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/contracts/tenant-handling.md
- [ ] T023 [US3] Update operator-facing tenant guidance and runtime support notes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go, and generated CLI help sources under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/
- [ ] T024 [US3] Regenerate affected CLI reference output under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/ from updated Cobra help text and docs commands

**Checkpoint**: User Story 3 is independently testable: the version-specific tenant contract is explicit, reviewable, and documented without overstating current `8.9` runtime support.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish repository-wide validation and leave the planning artifacts aligned with the shipped result.

- [ ] T025 [P] Refresh implementation and verification notes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/plan.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/quickstart.md
- [ ] T026 Run targeted tenant-handling validation with `go test ./internal/services/processinstance/... -count=1`, focused `go test ./cmd ... -count=1`, and focused `go test ./config ... -count=1` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt
- [ ] T027 Run repository validation with `make test` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion and is the MVP slice.
- **User Story 2 (Phase 4)**: Depends on User Story 1 because mixed-flow safety builds on the direct lookup contract and shared service outcomes established there.
- **User Story 3 (Phase 5)**: Depends on User Stories 1 and 2 because it documents and validates the settled version behavior and tenant contract.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: No functional dependency on later stories after Foundational work is complete.
- **User Story 2 (P2)**: Depends on User Story 1’s direct lookup contract and extends it through traversal, wait, cancel, and delete flows.
- **User Story 3 (P3)**: Depends on the final supported/unsupported behavior established by User Stories 1 and 2.

### Within Each User Story

- Add or update regression tests before considering the story complete.
- Versioned service behavior before command-surface cleanup that depends on it.
- Shared helper changes before flow-specific command changes.
- Documentation and CLI docs regeneration only after behavior and tests are stable.

### Parallel Opportunities

- `T002` and `T003` can run in parallel.
- `T006` and `T007` can run in parallel after the foundational tenant contract is settled.
- `T008`, `T009`, and `T010` can run in parallel.
- `T014`, `T015`, and `T016` can run in parallel.
- `T020` and `T021` can run in parallel.
- `T025` can run in parallel with targeted validation once implementation behavior is stable.

---

## Parallel Example: User Story 1

```bash
# Prepare User Story 1 direct-lookup coverage in parallel:
Task: "Add `v88` service tests for tenant-safe direct lookup and supported wrong-tenant `not found` behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go"
Task: "Add `v87` service tests for supported direct-lookup cases and explicit unsupported outcomes where tenant-safe retrieval is impossible in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go"
Task: "Add command regression tests for direct process-instance lookup with explicit `--tenant`, environment-derived tenant, profile-derived tenant, and base-config-derived tenant in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go"
```

## Parallel Example: User Story 2

```bash
# Prepare User Story 2 mixed-flow coverage in parallel:
Task: "Add walker and waiter regression tests for tenant-safe ancestry, descendants, family, and state polling behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter_test.go"
Task: "Add versioned service tests for mixed search plus direct-get flows, cancel preflight, and delete preflight behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go"
Task: "Add command-family regression tests for `walk`, `cancel`, `delete`, and `run` tenant propagation across flag/env/profile/base-config sources in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Stop and validate tenant-safe direct lookup behavior before expanding into traversal and mutation flows.

### Incremental Delivery

1. Finish Setup + Foundational once.
2. Deliver User Story 1 as the MVP for tenant-safe direct lookups and `not found` mismatch behavior.
3. Add User Story 2 to extend the contract through walk, wait, cancel, delete, and other mixed flows.
4. Add User Story 3 to lock in the supported-version matrix and user-facing documentation.
5. Finish with targeted validation and full `make test`.

### Parallel Team Strategy

1. One contributor handles Setup + Foundational work.
2. After Foundational is complete:
   - Contributor A: User Story 1 versioned-service and direct lookup coverage.
   - Contributor B: User Story 2 walker/waiter and command-family mixed-flow coverage.
   - Contributor C: User Story 3 documentation and version-matrix work once behavior stabilizes.
3. Finish with shared validation and repository-wide tests.

---

## Notes

- [P] tasks are limited to work on different files with no dependency on unfinished tasks.
- [US1], [US2], and [US3] map directly to the user stories in [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/spec.md).
- This feature’s commit subjects must keep Conventional Commit formatting and append `#109` as the final token.
- Run `make test` before committing, per repository rules.
