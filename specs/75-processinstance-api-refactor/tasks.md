# Tasks: Review and Refactor Internal Service Processinstance API Implementation

**Input**: Design documents from `/specs/75-processinstance-api-refactor/`
**Prerequisites**: [plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/plan.md), [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/spec.md), [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/research.md), [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/data-model.md), [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/quickstart.md)

**Tests**: Automated test tasks are REQUIRED for every story and shared change in this feature.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., `US1`, `US2`, `US3`)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Capture the exact reviewed service surface, helper boundaries, and generated-client coverage candidates before code changes

- [ ] T001 Review the current processinstance service flow and helper boundaries in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/api.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go`
- [ ] T002 Review the supported generated processinstance client operations and current contract surface in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/contract.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/contract.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/clients/camunda/v87/camunda/client.gen.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/clients/camunda/v87/operate/client.gen.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/clients/camunda/v88/camunda/client.gen.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/clients/camunda/v88/operate/client.gen.go`
- [ ] T003 Review the current processinstance regression surface and validation path in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory_test.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/quickstart.md`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared API, contract, and validation boundaries that all stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T004 Define the preserved shared processinstance service surface and the rule for any partial-version capability addition in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/api.go`
- [ ] T005 [P] Prepare version-specific generated-client contract updates needed for the refactor and any approved capability review outcome in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/contract.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/contract.go`
- [ ] T006 [P] Expand factory coverage for preserved version routing and unknown-version behavior in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory_test.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in priority order or parallel where noted

---

## Phase 3: User Story 1 - Preserve process instance behavior while simplifying maintenance (Priority: P1) 🎯 MVP

**Goal**: Make the shared API, helper usage, and versioned implementations easier to read while preserving all current processinstance behavior

**Independent Test**: Run the processinstance service tests and confirm create, get, search, wait, walk, cancel, delete, malformed-response handling, and version-specific semantics remain unchanged after the refactor.

### Tests for User Story 1

- [ ] T007 [P] [US1] Add v8.7 regression tests for create, get, search, cancel, delete, and malformed-response behavior in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go`
- [ ] T008 [P] [US1] Add v8.8 regression tests for create, get, search, cancel, delete, and malformed-response behavior in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go`
- [ ] T009 [P] [US1] Add helper-level regression tests for polling and traversal invariants in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter_test.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker_test.go`

### Implementation for User Story 1

- [ ] T010 [US1] Refactor shared API declarations and factory readability without changing behavior in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/api.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory.go`
- [ ] T011 [P] [US1] Simplify v8.7 service control flow, payload validation, and helper interaction paths in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/bulk.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/convert.go`
- [ ] T012 [P] [US1] Simplify v8.8 service control flow, payload validation, and helper interaction paths in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/bulk.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/convert.go`
- [ ] T013 [US1] Reconcile shared response-validation and helper usage with repository-native patterns in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/common/response.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go`
- [ ] T014 [US1] Simplify waiter and walker call sites without changing their semantics in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go`

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Review generated-client-backed processinstance coverage (Priority: P2)

**Goal**: Review the generated-client processinstance capabilities and expose one bounded missing capability only if it fits the existing shared service surface

**Independent Test**: Compare the shared API to the generated clients, then verify that the approved capability decision is reflected in the service surface and implementation for both supported versions, including a defined unsupported-version path if support differs.

### Tests for User Story 2

- [ ] T015 [P] [US2] Add v8.7 tests for the approved generated-client coverage decision in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/contract.go`
- [ ] T016 [P] [US2] Add v8.8 tests for the approved generated-client coverage decision in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/contract.go`

### Implementation for User Story 2

- [ ] T017 [US2] Record the final generated-processinstance capability review and add-or-defer rationale in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/research.md`
- [ ] T018 [US2] Update the shared processinstance service interface to either expose the approved missing capability or explicitly preserve the bounded current surface in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/api.go`
- [ ] T019 [US2] Implement the approved generated-client coverage decision for v8.7, including unsupported-version behavior if applicable, in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/contract.go`
- [ ] T020 [US2] Implement the approved generated-client coverage decision for v8.8, including unsupported-version behavior if applicable, in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/contract.go`
- [ ] T021 [US2] Update the processinstance data-shape notes for the final service surface in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/data-model.md`

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Strengthen regression confidence for future maintenance (Priority: P3)

**Goal**: Ensure the refactored service is covered by focused tests and explicit validation or documentation decisions that make future cleanups safe

**Independent Test**: Run targeted processinstance tests and confirm they protect preserved behavior, helper semantics, partial-version capability behavior, and documentation-impact decisions without relying on implementation details.

### Tests for User Story 3

- [ ] T022 [P] [US3] Expand cross-version regression cases for edge-case state transitions, empty-success payloads, unsupported-version capability behavior, and delete or cancel follow-up failures in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go`
- [ ] T023 [P] [US3] Add final shared-surface regression coverage for the settled processinstance API and factory behavior in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory_test.go`

### Implementation for User Story 3

- [ ] T024 [US3] Tighten processinstance service test fixtures and helper builders for long-term maintainability in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter_test.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker_test.go`
- [ ] T025 [US3] Update the final validation sequence and documentation-impact decision in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/quickstart.md`
- [ ] T026 [US3] If User Story 2 adds a user-visible processinstance workflow, update `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md` and regenerate docs under `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/`

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification and closeout across all stories

- [ ] T027 [P] Run quickstart validation using the steps in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/quickstart.md`
- [ ] T028 Run targeted processinstance validation with `go test ./internal/services/processinstance/... -race -count=1` from `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt`
- [ ] T029 Run the repository validation command set, including `make test`, from `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: Depend on Foundational phase completion
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Starts after Foundational - this is the MVP and preserves current behavior while simplifying the processinstance service flow
- **User Story 2 (P2)**: Starts after User Story 1 because the generated-client coverage decision depends on the clarified and stabilized shared service surface
- **User Story 3 (P3)**: Starts after User Stories 1 and 2 because the final regression matrix and documentation decision depend on the settled refactor and final capability choice

### Within Each User Story

- Tests are added or updated before story sign-off
- Shared API and contract changes before version-specific service implementation
- Behavior-preserving refactors before any capability addition
- Targeted validation before repository-wide validation
- Documentation updates only if a user-visible capability is actually introduced

### Parallel Opportunities

- `T005` and `T006` can run in parallel once the preserved shared surface is identified
- `T007`, `T008`, and `T009` can run in parallel within User Story 1
- `T011` and `T012` can run in parallel after `T010`
- `T015` and `T016` can run in parallel within User Story 2
- `T022` and `T023` can run in parallel within User Story 3
- `T027` can run in parallel with `T028` once implementation is complete

---

## Parallel Example: User Story 1

```bash
# Launch User Story 1 regression coverage tasks together:
Task: "Add v8.7 regression tests for create, get, search, cancel, delete, and malformed-response behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go"
Task: "Add v8.8 regression tests for create, get, search, cancel, delete, and malformed-response behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go"
Task: "Add helper-level regression tests for polling and traversal invariants in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker_test.go"

# Launch version-specific refactors together after the shared API/factory cleanup:
Task: "Simplify v8.7 service control flow, payload validation, and helper interaction paths in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/bulk.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/convert.go"
Task: "Simplify v8.8 service control flow, payload validation, and helper interaction paths in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/bulk.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/convert.go"
```

---

## Parallel Example: User Story 2

```bash
# Launch generated-client coverage tests for both supported versions together:
Task: "Add v8.7 tests for the approved generated-client coverage decision in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/contract.go"
Task: "Add v8.8 tests for the approved generated-client coverage decision in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/contract.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Run the targeted processinstance tests and confirm create, wait, walk, cancel, and delete behavior still work independently

### Incremental Delivery

1. Preserve and simplify the current processinstance service flow as the MVP
2. Add the generated-client coverage decision and validate both supported versions
3. Expand regression protection and finalize the documentation decision
4. Finish with targeted validation and `make test`

### Parallel Team Strategy

With multiple contributors:

1. One contributor completes Setup + Foundational work
2. After foundational work lands:
   - Contributor A: User Story 1 refactor and preserved-behavior validation
   - Contributor B: User Story 2 generated-client coverage tests and implementation
   - Contributor C: User Story 3 regression hardening after the final service surface settles

---

## Notes

- [P] tasks touch different files or can be validated independently
- [US1], [US2], and [US3] map directly to the user stories in [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/spec.md)
- Keep implementation repository-native: existing internal service structure, helper packages, no package renames, and no new dependency layers
- Suggested MVP scope: Phase 3 / User Story 1
