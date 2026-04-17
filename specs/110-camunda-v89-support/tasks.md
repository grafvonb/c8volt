# Tasks: Add Camunda v8.9 Runtime Support

**Input**: Design documents from `/specs/110-camunda-v89-support/`
**Prerequisites**: [plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/plan.md) (required), [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/spec.md) (required for user stories), [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/research.md), [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/data-model.md), [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/quickstart.md), [contracts/v89-support.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/contracts/v89-support.md)

**Tests**: Automated tests are REQUIRED for this feature because the specification explicitly requires factory-selection coverage, preserved `v8.7`/`v8.8` behavior coverage, and at least one explicit `v8.9` execution test for each repository command family.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Lock in the exact `v8.9` scope, client boundary, and current verification seams before shared implementation begins.

- [x] T001 Inventory the repository-wide `v8.8` command families and current `v8.9` support boundary in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/research.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/plan.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go
- [x] T002 [P] Confirm the generated `v89` Camunda client endpoints needed for native cluster, process-definition, process-instance, and resource support in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/clients/camunda/v89/camunda/client.gen.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/research.md
- [x] T003 [P] Confirm existing factory and regression-test seams for `cluster`, `processdefinition`, `processinstance`, and `resource` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/*/factory.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/*/factory_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/*test.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared version-support contract and common service seams that all user stories build on.

**⚠️ CRITICAL**: No user story work should begin until this phase is complete.

- [ ] T004 Update the shared supported-version source of truth for `v8.9` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/toolx/version.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/version.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors.go
- [ ] T005 [P] Extend shared factory coverage and top-level client wiring expectations for `v8.9` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/cluster/factory_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/factory_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/factory_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/client.go
- [ ] T006 [P] Create the base `v89` service package scaffolds and shared contract assertions in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/cluster/v89/contract.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v89/contract.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/contract.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v89/contract.go, and their corresponding service.go files
- [ ] T007 Add the foundational version-support contract and release-gate notes to /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/contracts/v89-support.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/quickstart.md

**Checkpoint**: The repository has one explicit `v8.9` support boundary, base `v89` service packages exist, and the factories/tests are ready for story-specific implementation.

---

## Phase 3: User Story 1 - Run Existing Commands on v8.9 (Priority: P1) 🎯 MVP

**Goal**: Make every repository command family currently supported on `v8.8` execute successfully on `v8.9` with the same user-facing contract.

**Independent Test**: Configure the CLI for `v8.9` and run at least one explicit command execution path for each repository command family, confirming the same contract currently expected on `v8.8`.

### Tests for User Story 1

- [ ] T008 [P] [US1] Add native `v89` service tests for cluster topology/license behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/cluster/v89/service_test.go
- [ ] T009 [P] [US1] Add native `v89` service tests for process-definition search/get/XML/statistics behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v89/service_test.go
- [ ] T010 [P] [US1] Add native `v89` service tests for resource deploy/get/delete behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v89/service_test.go
- [ ] T011 [P] [US1] Add native `v89` service tests for process-instance create/get/search/cancel/delete/wait behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter_test.go
- [ ] T012 [P] [US1] Add explicit `v8.9` command execution tests for cluster, process-definition/resource, and process-instance command families in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go

### Implementation for User Story 1

- [ ] T013 [P] [US1] Implement native `v89` cluster and process-definition services plus factory selection in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/cluster/v89/service.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/cluster/factory.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v89/service.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/factory.go
- [ ] T014 [P] [US1] Implement native `v89` resource service plus factory selection in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v89/service.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/factory.go
- [ ] T015 [US1] Implement the native `v89` process-instance service and factory selection using only the generated `v89` Camunda client in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/contract.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory.go
- [ ] T016 [US1] Wire the new `v89` services through the shared client and process/resource facades in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/client.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/client.go

**Checkpoint**: User Story 1 is independently testable: every repository command family has a working `v8.9` path and explicit execution proof.

---

## Phase 4: User Story 2 - Keep Version Selection Predictable (Priority: P2)

**Goal**: Make `v8.9` selection explicit and reliable while preserving current `v8.7` and `v8.8` behavior and keeping temporary fallback narrow and honest.

**Independent Test**: Exercise supported, missing, invalid, and unsupported version selection paths and verify `v8.9` routes to native services while `v8.7`/`v8.8` continue behaving as before.

### Tests for User Story 2

- [ ] T017 [P] [US2] Add regression coverage for supported-version selection and preserved `v8.7`/`v8.8` behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/cluster/factory_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/factory_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/factory_test.go
- [ ] T018 [P] [US2] Add bootstrap and config regression coverage for `v8.9` support messaging and unsupported-version failures in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/app_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/bootstrap_errors_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors_test.go
- [ ] T019 [P] [US2] Add process-instance-specific regression coverage that proves final native `v8.9` paths stay on the `v89` Camunda client boundary and any temporary fallback stays documented/non-final in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/contracts/v89-support.md

### Implementation for User Story 2

- [ ] T020 [US2] Update root command version messaging and supported-version behavior for `v8.9` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/toolx/version.go
- [ ] T021 [US2] Preserve older-version behavior while extending version-aware helpers and error normalization for `v8.9` in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/app.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/errors.go
- [ ] T022 [US2] Finalize any documented transition-only fallback rules and removal conditions in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/plan.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/research.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/contracts/v89-support.md

**Checkpoint**: User Story 2 is independently testable: `v8.9` selection is predictable, preserved versions remain stable, and fallback rules are explicit and bounded.

---

## Phase 5: User Story 3 - Make v8.9 Support Verifiable and Explicit (Priority: P3)

**Goal**: Make the new `v8.9` support level reviewable and trustworthy through release-gated docs and complete validation guidance.

**Independent Test**: Review tests, README, root help, and generated docs to confirm they all present the same `v8.9` support truth and the same completion bar.

### Tests for User Story 3

- [ ] T023 [P] [US3] Add doc-facing regression coverage for updated supported-version output in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/version.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go
- [ ] T024 [P] [US3] Add or refresh final verification notes and quickstart validation guidance in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/quickstart.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/plan.md

### Implementation for User Story 3

- [ ] T025 [US3] Update user-facing version-support guidance in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go
- [ ] T026 [US3] Regenerate CLI reference output for the updated help text in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/ and sync homepage content from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md

**Checkpoint**: User Story 3 is independently testable: the runtime truth, docs, and validation guidance all agree on what `v8.9` support means.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish repository-wide validation and leave the feature artifacts aligned with the shipped result.

- [ ] T027 [P] Refresh implementation notes and final support-boundary records in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/research.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/data-model.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/contracts/v89-support.md
- [ ] T028 Run focused `v8.9` validation with `go test ./internal/services/cluster/... -count=1`, `go test ./internal/services/processdefinition/... -count=1`, `go test ./internal/services/processinstance/... -count=1`, `go test ./internal/services/resource/... -count=1`, and `go test ./cmd -count=1` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt
- [ ] T029 Run documentation regeneration and repository validation with `make docs`, `make docs-content`, and `make test` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion and is the MVP slice.
- **User Story 2 (Phase 4)**: Depends on User Story 1 because predictable version selection relies on the real `v8.9` runtime paths and factory wiring already existing.
- **User Story 3 (Phase 5)**: Depends on User Stories 1 and 2 because docs and validation must describe the settled runtime truth.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: No functional dependency on later stories after Foundational work is complete.
- **User Story 2 (P2)**: Depends on User Story 1’s native `v8.9` service paths and command parity proof.
- **User Story 3 (P3)**: Depends on the final supported-version behavior established by User Stories 1 and 2.

### Within Each User Story

- Add or update regression tests before considering the story complete.
- Service and factory behavior before command-surface cleanup that depends on it.
- Root/help text before docs regeneration.
- Documentation and spec artifacts after runtime behavior and validation expectations are stable.

### Parallel Opportunities

- `T002` and `T003` can run in parallel.
- `T005` and `T006` can run in parallel after `T004`.
- `T008`, `T009`, `T010`, `T011`, and `T012` can run in parallel.
- `T013` and `T014` can run in parallel before `T015`.
- `T017`, `T018`, and `T019` can run in parallel.
- `T023` and `T024` can run in parallel.
- `T027` can run in parallel with targeted validation once implementation behavior is stable.

---

## Parallel Example: User Story 1

```bash
# Prepare User Story 1 native v8.9 coverage in parallel:
Task: "Add native `v89` service tests for cluster topology/license behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/cluster/v89/service_test.go"
Task: "Add native `v89` service tests for process-definition search/get/XML/statistics behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v89/service_test.go"
Task: "Add native `v89` service tests for resource deploy/get/delete behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v89/service_test.go"
Task: "Add native `v89` service tests for process-instance create/get/search/cancel/delete/wait behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter_test.go"
```

## Parallel Example: User Story 2

```bash
# Prepare User Story 2 version-selection proof in parallel:
Task: "Add regression coverage for supported-version selection and preserved `v8.7`/`v8.8` behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/cluster/factory_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/factory_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/factory_test.go"
Task: "Add bootstrap and config regression coverage for `v8.9` support messaging and unsupported-version failures in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/app_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/bootstrap_errors_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors_test.go"
Task: "Add process-instance-specific regression coverage that proves final native `v8.9` paths stay on the `v89` Camunda client boundary and any temporary fallback stays documented/non-final in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/contracts/v89-support.md"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Stop and validate that every existing repository command family now has an explicit `v8.9` execution path.

### Incremental Delivery

1. Finish Setup + Foundational once.
2. Deliver User Story 1 to establish real repository-wide `v8.9` command parity.
3. Add User Story 2 to lock in predictable version selection and preserved older-version behavior.
4. Add User Story 3 to make the support claim reviewable through docs and validation guidance.
5. Finish with focused validation and full `make test`.

### Parallel Team Strategy

1. One contributor handles Setup + Foundational work.
2. After Foundational is complete:
   - Contributor A: Cluster/process-definition/resource `v89` services and tests.
   - Contributor B: Process-instance `v89` service plus walker/waiter integration.
   - Contributor C: Command execution tests, version messaging, and docs work once service behavior stabilizes.
3. Finish with shared validation and repository-wide tests.

---

## Notes

- [P] tasks are limited to work on different files with no dependency on unfinished tasks.
- [US1], [US2], and [US3] map directly to the user stories in [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/spec.md).
- This feature’s commit subjects must keep Conventional Commit formatting and append `#110` as the final token.
- Run `make test` before committing, per repository rules.
