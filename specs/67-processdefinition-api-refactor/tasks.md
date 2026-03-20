# Tasks: Review and Refactor Internal Service Processdefinition API Implementation

**Input**: Design documents from `/specs/67-processdefinition-api-refactor/`
**Prerequisites**: [plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/67-processdefinition-api-refactor/plan.md) (required), [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/67-processdefinition-api-refactor/spec.md) (required for user stories), [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/67-processdefinition-api-refactor/research.md), [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/67-processdefinition-api-refactor/data-model.md)

**Tests**: Automated test tasks are REQUIRED for every story and shared change in this feature.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare the processdefinition service surface, generated-client review inputs, and current test coverage for the refactor

- [ ] T001 Inspect the current processdefinition service flow and affected generated clients in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/api.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/factory.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/service.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/service.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/clients/camunda/v87/operate/client.gen.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/clients/camunda/v88/camunda/client.gen.go
- [ ] T002 Inspect baseline regression coverage in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/factory_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/service_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/service_test.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared API and helper boundaries that all stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T003 Define the preserved shared service surface and any candidate capability additions in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/api.go
- [ ] T004 [P] Prepare version-specific client contract updates needed for the refactor in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/contract.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/contract.go
- [ ] T005 [P] Prepare shared factory and dependency wiring for the refactored service surface in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/factory.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/factory_test.go

**Checkpoint**: Foundation ready - user story implementation can now begin in priority order or parallel where noted

---

## Phase 3: User Story 1 - Preserve behavior while simplifying service flows (Priority: P1) 🎯 MVP

**Goal**: Make the shared API and versioned implementations easier to read while preserving all existing processdefinition behavior

**Independent Test**: Run the processdefinition service tests and confirm search, latest, get, error handling, and version-specific statistics behavior remain unchanged after the refactor.

### Tests for User Story 1

- [ ] T006 [P] [US1] Add regression-focused factory assertions for preserved version selection and shared API wiring in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/factory_test.go
- [ ] T007 [P] [US1] Add v8.7 regression tests for preserved search, latest, get, and unsupported-statistics behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/service_test.go
- [ ] T008 [P] [US1] Add v8.8 regression tests for preserved search, latest, get, and statistics-enrichment behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/service_test.go

### Implementation for User Story 1

- [ ] T009 [US1] Refactor the shared processdefinition API declarations and common constants for clarity in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/api.go
- [ ] T010 [US1] Refactor version-selection and error-path clarity without changing behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/factory.go
- [ ] T011 [US1] Simplify duplicated request construction, response validation, and latest-result handling in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/service.go
- [ ] T012 [US1] Simplify duplicated request construction, response validation, and statistics retrieval flow in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/service.go

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Align processdefinition coverage with generated clients (Priority: P2)

**Goal**: Review the generated-client processdefinition capabilities and expose one bounded missing capability only if it fits the existing shared service surface

**Independent Test**: Compare the shared API to the generated clients, then verify that the approved capability decision is reflected in the service surface and implementation for both supported versions without breaking existing behavior.

### Tests for User Story 2

- [ ] T013 [P] [US2] Add v8.7 tests for the approved generated-client coverage decision in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/contract.go
- [ ] T014 [P] [US2] Add v8.8 tests for the approved generated-client coverage decision in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/contract.go

### Implementation for User Story 2

- [ ] T015 [US2] Update the shared service interface to either expose the approved missing capability or explicitly preserve the bounded current surface in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/api.go
- [ ] T016 [US2] Implement the approved generated-client coverage decision for v8.7 in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/service.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/contract.go
- [ ] T017 [US2] Implement the approved generated-client coverage decision for v8.8 in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/service.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/contract.go
- [ ] T018 [US2] Update processdefinition factory coverage for the final shared service surface in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/factory_test.go

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Strengthen regression protection for future maintenance (Priority: P3)

**Goal**: Ensure the refactored service is covered by focused tests that make future cleanups safe

**Independent Test**: Run targeted processdefinition tests and confirm they protect the preserved behavior paths, version differences, and any approved capability addition without relying on implementation details.

### Tests for User Story 3

- [ ] T019 [P] [US3] Expand cross-version regression cases for ordering, malformed responses, and edge-case filtering in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/service_test.go
- [ ] T020 [P] [US3] Add regression coverage for any new shared helper or service-surface edge case in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/factory_test.go

### Implementation for User Story 3

- [ ] T021 [US3] Tighten service test fixtures and helper builders for long-term maintainability in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/service_test.go
- [ ] T022 [US3] Update feature notes to record the final generated-client coverage decision and validation scope in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/67-processdefinition-api-refactor/quickstart.md

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification and conditional documentation consistency checks

- [ ] T023 [P] Run quickstart validation using the steps in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/67-processdefinition-api-refactor/quickstart.md
- [ ] T024 Run targeted processdefinition validation with `go test ./internal/services/processdefinition/... -race -count=1` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt
- [ ] T025 If User Story 2 adds a user-visible processdefinition workflow, update /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md and regenerate docs under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/
- [ ] T026 Run the repository validation command set, including `make test`, from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: Depend on Foundational phase completion
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Starts after Foundational - this is the MVP and preserves current behavior while simplifying the service flow
- **User Story 2 (P2)**: Starts after User Story 1 because the generated-client coverage decision depends on the clarified shared service surface
- **User Story 3 (P3)**: Starts after User Stories 1 and 2 because the final regression matrix depends on the settled refactor and final capability decision

### Within Each User Story

- Tests are added or updated before story sign-off
- Shared API and contract changes before version-specific service implementation
- Behavior-preserving refactors before any capability addition
- Targeted validation before repository-wide validation
- Documentation updates only if a user-visible capability is actually introduced

### Parallel Opportunities

- T004 and T005 can run in parallel once the preserved shared surface is identified
- T006, T007, and T008 can run in parallel within User Story 1
- T013 and T014 can run in parallel within User Story 2
- T019 and T020 can run in parallel within User Story 3
- T023 can run in parallel with targeted validation once implementation is complete

---

## Parallel Example: User Story 1

```bash
# Launch User Story 1 regression coverage tasks together:
Task: "Add regression-focused factory assertions for preserved version selection and shared API wiring in internal/services/processdefinition/factory_test.go"
Task: "Add v8.7 regression tests for preserved search, latest, get, and unsupported-statistics behavior in internal/services/processdefinition/v87/service_test.go"
Task: "Add v8.8 regression tests for preserved search, latest, get, and statistics-enrichment behavior in internal/services/processdefinition/v88/service_test.go"
```

---

## Parallel Example: User Story 2

```bash
# Launch generated-client coverage tests for both supported versions together:
Task: "Add v8.7 tests for the approved generated-client coverage decision in internal/services/processdefinition/v87/service_test.go and internal/services/processdefinition/v87/contract.go"
Task: "Add v8.8 tests for the approved generated-client coverage decision in internal/services/processdefinition/v88/service_test.go and internal/services/processdefinition/v88/contract.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Run the targeted processdefinition tests and confirm search, latest, get, and version-specific statistics behavior still work independently

### Incremental Delivery

1. Preserve and simplify the current service flow as the MVP
2. Add the generated-client coverage decision and validate both supported versions
3. Expand regression protection around the final service surface
4. Finish with targeted validation and `make test`

### Parallel Team Strategy

With multiple developers:

1. One developer completes Setup + Foundational work
2. After foundational work lands:
   - Developer A: User Story 1 refactor and preserved-behavior validation
   - Developer B: User Story 2 generated-client coverage tests and implementation
   - Developer C: User Story 3 regression hardening after the final service surface settles

---

## Notes

- [P] tasks touch different files or can be validated independently
- [US1], [US2], and [US3] map directly to the user stories in [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/67-processdefinition-api-refactor/spec.md)
- Keep implementation repository-native: existing internal service structure, no package renames, no new dependency layers
- The current Spec Kit prerequisite script still assumes zero-padded branch names; this task list intentionally targets the normalized `67-processdefinition-api-refactor` feature directory
