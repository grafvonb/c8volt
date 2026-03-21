# Tasks: Review and Refactor Internal Service Resource API Implementation

**Input**: Design documents from `/specs/71-resource-api-refactor/`
**Prerequisites**: [plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/plan.md), [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/spec.md), [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/research.md), [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/data-model.md), [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/quickstart.md)

**Tests**: Automated test tasks are REQUIRED for every story and shared change in this feature.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., `US1`, `US2`, `US3`)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Capture the exact reviewed service surface, candidate missing capability, and current regression surface before code changes

- [ ] T001 Review the current resource service flow and generated client operations in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/api.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/factory.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/service.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/contract.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/contract.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/clients/camunda/v87/camunda/client.gen.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/clients/camunda/v88/camunda/client.gen.go`
- [ ] T002 Review the current resource regression surface in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/factory_test.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service_test.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared API, contract, and validation boundaries that all stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T003 Define the preserved shared service surface and any candidate capability additions in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/api.go`
- [ ] T004 [P] Prepare version-specific generated-client contract updates needed for the refactor in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/contract.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/contract.go`
- [ ] T005 [P] Expand factory coverage for preserved version routing and unknown-version behavior in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/factory_test.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in priority order or parallel where noted

---

## Phase 3: User Story 1 - Preserve resource service behavior while simplifying maintenance (Priority: P1) 🎯 MVP

**Goal**: Make the shared API and versioned implementations easier to read while preserving all existing resource deploy and delete behavior

**Independent Test**: Run the resource service tests and confirm deploy, delete, malformed-response handling, and version-specific confirmation behavior remain unchanged after the refactor.

### Tests for User Story 1

- [ ] T006 [P] [US1] Add v8.7 regression tests for deploy, delete, and malformed-response behavior in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/service_test.go`
- [ ] T007 [P] [US1] Expand v8.8 regression tests for deploy, wait/no-wait behavior, delete, and malformed-response behavior in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service_test.go`

### Implementation for User Story 1

- [ ] T008 [US1] Refactor shared API declarations and factory readability without changing behavior in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/api.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/factory.go`
- [ ] T009 [P] [US1] Simplify v8.7 multipart request construction, payload validation, and delete-path handling in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/service.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/convert.go`
- [ ] T010 [P] [US1] Simplify v8.8 multipart request construction, payload validation, deployment confirmation flow, and delete-path handling in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/convert.go`
- [ ] T011 [US1] Reconcile shared response-validation and helper usage with repository-native patterns in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/common/response.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/service.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service.go`

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Review generated-client-backed resource coverage (Priority: P2)

**Goal**: Review the generated-client resource capabilities and expose one bounded missing capability only if it fits the existing shared service surface

**Independent Test**: Compare the shared API to the generated clients, then verify that the approved capability decision is reflected in the service surface and implementation for both supported versions without breaking existing behavior.

### Tests for User Story 2

- [ ] T012 [P] [US2] Add v8.7 tests for the approved generated-client coverage decision in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/service_test.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/contract.go`
- [ ] T013 [P] [US2] Add v8.8 tests for the approved generated-client coverage decision in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service_test.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/contract.go`

### Implementation for User Story 2

- [ ] T014 [US2] Record the final generated-resource capability review and add-or-defer rationale in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/research.md`
- [ ] T015 [US2] Update the shared resource service interface to either expose the approved missing capability or explicitly preserve the bounded current surface in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/api.go`
- [ ] T016 [US2] Implement the approved generated-client coverage decision for v8.7 in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/service.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/contract.go`
- [ ] T017 [US2] Implement the approved generated-client coverage decision for v8.8 in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/contract.go`
- [ ] T018 [US2] Update the resource data-shape notes for the final service surface in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/data-model.md`

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Strengthen regression confidence for future changes (Priority: P3)

**Goal**: Ensure the refactored service is covered by focused tests and explicit validation/documentation decisions that make future cleanups safe

**Independent Test**: Run targeted resource tests and confirm they protect the preserved behavior paths, version differences, and any approved capability addition without relying on implementation details.

### Tests for User Story 3

- [ ] T019 [P] [US3] Expand cross-version regression cases for edge-case delete behavior, empty-success payloads, and confirmation-path failures in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/service_test.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service_test.go`
- [ ] T020 [P] [US3] Add final shared-surface regression coverage for the settled resource API in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/factory_test.go`

### Implementation for User Story 3

- [ ] T021 [US3] Tighten resource service test fixtures and helper builders for long-term maintainability in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/service_test.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service_test.go`
- [ ] T022 [US3] Update the final validation sequence and documentation-impact decision in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/quickstart.md`
- [ ] T023 [US3] If User Story 2 adds a user-visible resource workflow, update `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md` and regenerate docs under `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/`

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification and closeout across all stories

- [ ] T024 [P] Run quickstart validation using the steps in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/quickstart.md`
- [ ] T025 Run targeted resource validation with `go test ./internal/services/resource/... -race -count=1` from `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt`
- [ ] T026 Run the repository validation command set, including `make test`, from `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: Depend on Foundational phase completion
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Starts after Foundational - this is the MVP and preserves current behavior while simplifying the resource service flow
- **User Story 2 (P2)**: Starts after User Story 1 because the generated-client coverage decision depends on the clarified and stabilized shared service surface
- **User Story 3 (P3)**: Starts after User Stories 1 and 2 because the final regression matrix and documentation decision depend on the settled refactor and final capability choice

### Within Each User Story

- Tests are added or updated before story sign-off
- Shared API and contract changes before version-specific service implementation
- Behavior-preserving refactors before any capability addition
- Targeted validation before repository-wide validation
- Documentation updates only if a user-visible capability is actually introduced

### Parallel Opportunities

- `T004` and `T005` can run in parallel once the preserved shared surface is identified
- `T006` and `T007` can run in parallel within User Story 1
- `T009` and `T010` can run in parallel after `T008`
- `T012` and `T013` can run in parallel within User Story 2
- `T019` and `T020` can run in parallel within User Story 3
- `T024` can run in parallel with `T025` once implementation is complete

---

## Parallel Example: User Story 1

```bash
# Launch User Story 1 regression coverage tasks together:
Task: "Add v8.7 regression tests for deploy, delete, and malformed-response behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/service_test.go"
Task: "Expand v8.8 regression tests for deploy, wait/no-wait behavior, delete, and malformed-response behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service_test.go"

# Launch version-specific refactors together after the shared API/factory cleanup:
Task: "Simplify v8.7 multipart request construction, payload validation, and delete-path handling in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/service.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/convert.go"
Task: "Simplify v8.8 multipart request construction, payload validation, deployment confirmation flow, and delete-path handling in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/convert.go"
```

---

## Parallel Example: User Story 2

```bash
# Launch generated-client coverage tests for both supported versions together:
Task: "Add v8.7 tests for the approved generated-client coverage decision in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/contract.go"
Task: "Add v8.8 tests for the approved generated-client coverage decision in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/contract.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Run the targeted resource tests and confirm deploy, delete, and confirmation behavior still work independently

### Incremental Delivery

1. Preserve and simplify the current resource service flow as the MVP
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
- [US1], [US2], and [US3] map directly to the user stories in [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/spec.md)
- Keep implementation repository-native: existing internal service structure, no package renames, no new dependency layers
- Suggested MVP scope: Phase 3 / User Story 1
