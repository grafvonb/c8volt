# Tasks: Review and Refactor Cluster Service

**Input**: Design documents from `/specs/058-review-and-refactor-internal-service-cluster-api-implementation/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `quickstart.md`

**Tests**: Automated test tasks are required for every story and shared change for this feature.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g. `US1`, `US2`, `US3`)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Capture the exact reviewed service surface and validation scope before code changes

- [ ] T001 Update `specs/058-review-and-refactor-internal-service-cluster-api-implementation/research.md` with the reviewed supported-version cluster capability matrix and the planned add-or-defer decision criteria
- [ ] T002 Update `specs/058-review-and-refactor-internal-service-cluster-api-implementation/quickstart.md` with the exact validation sequence for cluster unit tests, integration tests, and full `make test`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish shared guardrails that all user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T003 Expand baseline factory coverage in `internal/services/cluster/factory_test.go` to lock supported-version selection and unknown-version failure behavior before refactoring
- [ ] T004 [P] Align the reviewed cluster service surface in `internal/services/cluster/api.go`, `internal/services/cluster/v87/contract.go`, and `internal/services/cluster/v88/contract.go`

**Checkpoint**: Factory behavior and shared service surface are fixed, so story work can proceed safely

---

## Phase 3: User Story 1 - Safer Cluster Service Maintenance (Priority: P1) 🎯 MVP

**Goal**: Reduce duplication and improve readability in the existing cluster topology service without changing observable behavior

**Independent Test**: Review the cluster service area and confirm that duplicated decision paths are reduced, version-specific responsibilities are clearer, and all existing cluster behavior still passes automated tests

### Tests for User Story 1

- [ ] T005 [P] [US1] Extend topology success and error coverage in `internal/services/cluster/v87/service_test.go`
- [ ] T006 [P] [US1] Extend topology success and error coverage in `internal/services/cluster/v88/service_test.go`

### Implementation for User Story 1

- [ ] T007 [US1] Refactor shared cluster service entry-point behavior in `internal/services/cluster/api.go` and `internal/services/cluster/factory.go`
- [ ] T008 [P] [US1] Refactor v87 cluster topology construction and response handling in `internal/services/cluster/v87/service.go` and `internal/services/cluster/v87/convert.go`
- [ ] T009 [P] [US1] Refactor v88 cluster topology construction and response handling in `internal/services/cluster/v88/service.go` and `internal/services/cluster/v88/convert.go`
- [ ] T010 [US1] Reconcile shared error handling and remove avoidable duplication across `internal/services/cluster/factory.go`, `internal/services/cluster/v87/service.go`, and `internal/services/cluster/v88/service.go`

**Checkpoint**: User Story 1 is complete when topology behavior is preserved and the cluster service is measurably easier to follow

---

## Phase 4: User Story 2 - Verified Client Coverage Review (Priority: P2)

**Goal**: Confirm whether the service layer fully covers the supported cluster operations and add one low-risk missing capability only if it fits the existing boundaries

**Independent Test**: Compare available cluster capabilities with the service surface and confirm the coverage review is documented, with any adopted additions covered by tests and behavior notes

### Tests for User Story 2

- [ ] T011 [P] [US2] Add reviewed service-surface coverage assertions in `internal/services/cluster/v87/service_test.go` and `internal/services/cluster/v88/service_test.go`
- [ ] T012 [P] [US2] Add shared service-surface regression coverage in `internal/services/cluster/factory_test.go`

### Implementation for User Story 2

- [ ] T013 [US2] Record the final supported-cluster-capability review and add-or-defer rationale in `specs/058-review-and-refactor-internal-service-cluster-api-implementation/research.md`
- [ ] T014 [US2] Update the shared cluster service surface for any approved capability in `internal/services/cluster/api.go`, `internal/services/cluster/v87/contract.go`, and `internal/services/cluster/v88/contract.go`
- [ ] T015 [P] [US2] Implement the approved capability or codify the no-addition decision in `internal/services/cluster/v87/service.go` and `internal/services/cluster/v88/service.go`

**Checkpoint**: User Story 2 is complete when the supported-version capability review is explicit and any accepted addition is fully covered without changing module boundaries

---

## Phase 5: User Story 3 - Clear Validation and Documentation Expectations (Priority: P3)

**Goal**: Make the verification and documentation outcome explicit so the refactor can be merged confidently

**Independent Test**: Verify that changed behavior, retained behavior, and any new capability all have corresponding automated tests and any affected user-facing documentation is updated in the same change

### Tests for User Story 3

- [ ] T016 [P] [US3] Update fake-server integration coverage in `testx/integration87/cluster_test.go` and `testx/integration88/cluster_test.go`

### Implementation for User Story 3

- [ ] T017 [US3] Validate user-visible impact and update `cmd/get_cluster_topology.go`, `README.md`, and `docs/cli/c8volt_get_cluster-topology.md` if the final capability review introduces a CLI-visible change
- [ ] T018 [US3] Record final verification expectations and documentation impact in `specs/058-review-and-refactor-internal-service-cluster-api-implementation/quickstart.md`

**Checkpoint**: User Story 3 is complete when verification expectations are explicit and any required documentation parity work is finished

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification and closeout across all stories

- [ ] T019 [P] Run the quickstart validation sequence documented in `specs/058-review-and-refactor-internal-service-cluster-api-implementation/quickstart.md`
- [ ] T020 Run the repository validation command set from `Makefile`, including `make test`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1: Setup**: No dependencies, can start immediately
- **Phase 2: Foundational**: Depends on Phase 1 and blocks all story work
- **Phase 3: User Story 1**: Depends on Phase 2 and delivers the MVP
- **Phase 4: User Story 2**: Depends on Phase 2 and should build on the stabilized service surface from US1
- **Phase 5: User Story 3**: Depends on the completed implementation and test updates from US1 and US2
- **Phase 6: Polish**: Depends on all selected stories being complete

### User Story Dependencies

- **US1**: Starts after Foundational and has no dependency on later stories
- **US2**: Starts after Foundational and may reuse the refactored service shape from US1
- **US3**: Starts after US1 and US2 because it validates their final behavior and documentation impact

### Within Each User Story

- Test tasks should be completed before story sign-off
- Shared surface changes come before version-specific implementation updates
- Version-specific implementations come before final reconciliation or documentation tasks
- Story validation completes before moving to the next dependent story

### Parallel Opportunities

- `T004` can run in parallel with `T003` once the setup notes are in place
- `T005` and `T006` can run in parallel
- `T008` and `T009` can run in parallel after `T007`
- `T011` and `T012` can run in parallel
- `T015` can proceed once `T014` defines the final service-surface decision
- `T016` can run in parallel with documentation-impact review work

---

## Parallel Example: User Story 1

```bash
# Launch User Story 1 test updates together:
Task: "Extend topology success and error coverage in internal/services/cluster/v87/service_test.go"
Task: "Extend topology success and error coverage in internal/services/cluster/v88/service_test.go"

# Launch version-specific refactors together after the shared entry-point update:
Task: "Refactor v87 cluster topology construction and response handling in internal/services/cluster/v87/service.go and internal/services/cluster/v87/convert.go"
Task: "Refactor v88 cluster topology construction and response handling in internal/services/cluster/v88/service.go and internal/services/cluster/v88/convert.go"
```

---

## Parallel Example: User Story 2

```bash
# Launch service-surface regression tests together:
Task: "Add reviewed service-surface coverage assertions in internal/services/cluster/v87/service_test.go and internal/services/cluster/v88/service_test.go"
Task: "Add shared service-surface regression coverage in internal/services/cluster/factory_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. Stop and validate the preserved topology behavior

### Incremental Delivery

1. Deliver US1 to lock in the maintainability refactor without behavior change
2. Deliver US2 to close the generated-client coverage review and any approved low-risk addition
3. Deliver US3 to finalize integration validation and documentation parity
4. Run the full validation sequence in Phase 6

### Parallel Team Strategy

1. One contributor completes Setup and Foundational tasks
2. Once Phase 2 is done, test updates and version-specific refactors can split across contributors
3. US3 closes out the combined verification and documentation work after the code paths settle

---

## Notes

- [P] tasks touch different files or can proceed after a clearly defined dependency
- Every task includes an exact file path to keep execution unambiguous
- `README.md` and CLI docs are conditional updates, but the validation task must explicitly confirm whether they changed
- Suggested MVP scope: Phase 3 / User Story 1
