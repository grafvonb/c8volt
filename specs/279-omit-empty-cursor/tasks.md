---

description: "Implementation tasks for omitting empty initial process-definition search cursors"
---

# Tasks: Omit Empty Initial Search Cursor

**Input**: Design documents from `specs/279-omit-empty-cursor/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [latest-process-definition-search.md](contracts/latest-process-definition-search.md), [quickstart.md](quickstart.md)

**Tests**: Required by FR-016, FR-017, SC-001 through SC-005, and the project constitution. Write or update the closest tests before changing each affected adapter, confirm the defect-specific assertion fails, then implement the correction.

**Organization**: Tasks are grouped by user story so the reported v8.9 workflow can be delivered as an MVP before completing cross-version parity and compatibility proof.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel after its stated dependencies because it changes different files
- **[Story]**: Maps the task to User Story 1, 2, or 3 from [spec.md](spec.md)
- Every task includes the exact repository path it inspects, changes, or validates

## Phase 1: Setup (Shared Baseline)

**Purpose**: Establish the pre-change behavior and keep the implementation inside the approved adapter boundary.

- [x] T001 Run the existing latest-search tests and confirm the empty-`after` expectations in `internal/services/processdefinition/v88/service_test.go`, `internal/services/processdefinition/v89/service_test.go`, and `internal/services/processdefinition/v810/service_test.go` before editing production code

---

## Phase 2: Foundational (Shared Paging Invariants)

**Purpose**: Lock the version-neutral continuation behavior that every affected adapter relies on.

**Critical**: Complete this phase before the user-story adapter changes.

- [x] T002 Extend `internal/services/processdefinition/search_test.go` to prove a non-empty opaque `EndCursor` is copied unchanged into the next `ProcessDefinitionPageRequest` and an empty final cursor produces no further cursor request

**Checkpoint**: Shared traversal preserves real cursors and stops without synthesizing an empty one.

---

## Phase 3: User Story 1 - Start Processes by BPMN ID Reliably (Priority: P1) — MVP

**Goal**: Remove the invalid empty initial cursor from the v8.9 latest-definition lookup used by BPMN-ID process-instance creation, while retaining pre-mutation selector validation.

**Independent Test**: Against Camunda 8 Run 8.9.17 default H2/RDBMS, deploy `C89_SimpleUserTask`, start ten instances by BPMN process ID, and observe ten returned keys with no process-definition search 500 error.

### Tests for User Story 1

- [x] T003 [US1] Replace the empty-cursor expectation in `internal/services/processdefinition/v89/service_test.go` with a failing serialized-wire assertion that the initial latest request contains `limit: 1000`, omits `after` and `from`, and retains `isLatestVersion`, tenant behavior, and stable latest sort

### Implementation for User Story 1

- [x] T004 [US1] Implement three-way page selection in `internal/services/processdefinition/v89/service.go`: real non-empty `After` uses cursor-forward pagination, initial latest uses generated `LimitPagination`, and ordinary search retains offset pagination
- [x] T005 [US1] Run the focused v8.9 adapter and selector regressions covering `internal/services/processdefinition/v89/service_test.go`, `cmd/process_definition_selector_validation_test.go`, and `cmd/run_test.go`
- [x] T006 [US1] Execute the disposable Camunda 8 Run 8.9.17 H2 workflow from `specs/279-omit-empty-cursor/quickstart.md` and verify `/tmp/c8volt-279-process-instance-keys.txt` contains exactly ten keys with no latest-search server error

**Checkpoint**: The reported Camunda 8.9.17 BPMN-ID workflow succeeds without a key or exact-version workaround and still validates all selectors before creation.

---

## Phase 4: User Story 2 - Page Latest Definitions Correctly (Priority: P1)

**Goal**: Apply the initial/continuation/final-page contract consistently to Camunda 8.8, 8.9, and 8.10.

**Independent Test**: For every affected adapter, inspect serialized requests and prove the first latest page is limit-only, a continuation page carries the exact non-empty opaque cursor and no offset, and an empty final cursor causes traversal to stop.

### Tests for User Story 2

- [x] T007 [P] [US2] Add failing initial-latest plus continuation/final-page wire assertions for Camunda 8.8 in `internal/services/processdefinition/v88/service_test.go`, including a non-empty opaque cursor that must be preserved exactly
- [x] T008 [P] [US2] Add failing initial-latest plus continuation/final-page wire assertions for Camunda 8.10 in `internal/services/processdefinition/v810/service_test.go`, including a non-empty opaque cursor that must be preserved exactly
- [x] T009 [P] [US2] Extend `internal/services/processdefinition/v89/service_test.go` with continuation and final-page assertions that complement the initial-page MVP test and prove no cursor request follows an empty response cursor

### Implementation for User Story 2

- [x] T010 [P] [US2] After T007 fails, implement the cursor/limit-only/offset branches with generated v8.8 pagination variants in `internal/services/processdefinition/v88/service.go`
- [x] T011 [P] [US2] After T008 fails, implement the cursor/limit-only/offset branches with generated v8.10 pagination variants in `internal/services/processdefinition/v810/service.go`
- [x] T012 [US2] Run all adapter and shared traversal tests under `internal/services/processdefinition/...` and confirm v8.8, v8.9, and v8.10 satisfy `specs/279-omit-empty-cursor/contracts/latest-process-definition-search.md`

**Checkpoint**: All three supported adapters implement the same latest-page state machine and preserve non-empty cursors unchanged.

---

## Phase 5: User Story 3 - Preserve Existing Process-Definition Searches (Priority: P2)

**Goal**: Prove that only empty initial latest cursors changed and every established process-definition selection contract remains stable.

**Independent Test**: Ordinary searches still serialize explicit `from` plus `limit`; latest filters, tenant scope, result limit, and sort remain unchanged; exact-version and key selectors retain existing results; CLI and output surfaces have no diff.

### Tests for User Story 3

- [x] T013 [P] [US3] Add or strengthen the ordinary offset request regression in `internal/services/processdefinition/v88/service_test.go` to require explicit `from` and `limit`, absent `after`, and unchanged filters and ordinary sort
- [x] T014 [P] [US3] Add or strengthen the ordinary offset request regression in `internal/services/processdefinition/v89/service_test.go` to require explicit `from` and `limit`, absent `after`, and unchanged filters and ordinary sort
- [x] T015 [P] [US3] Add or strengthen the ordinary offset request regression in `internal/services/processdefinition/v810/service_test.go` to require explicit `from` and `limit`, absent `after`, and unchanged filters and ordinary sort

### Compatibility Proof for User Story 3

- [x] T016 [US3] Run exact-version, key, tenant, latest-filter, no-partial-creation, and output regressions in `c8volt/process/client_test.go`, `cmd/get_processdefinition_test.go`, `cmd/process_definition_selector_validation_test.go`, and `cmd/run_test.go`
- [x] T017 [US3] Confirm the implementation introduces no changes under `internal/clients/camunda/`, `README.md`, or `docs/cli/`, documenting the internal-only decision in `specs/279-omit-empty-cursor/plan.md` if any unexpected user-facing diff appears

**Checkpoint**: Ordinary, exact-version, key-based, filter, tenant, sort, limit, CLI, and rendering behavior all remain compatible.

---

## Phase 6: Polish & Cross-Cutting Validation

**Purpose**: Format the finished change and run the required focused, repository-wide, and boundary gates.

- [x] T018 Run `gofmt` on touched files under `internal/services/processdefinition/v88/`, `internal/services/processdefinition/v89/`, `internal/services/processdefinition/v810/`, and `internal/services/processdefinition/search_test.go`
- [x] T019 Run focused tests for `internal/services/processdefinition/...`, `c8volt/process`, and `cmd`, including the process-definition and run-selector patterns from `specs/279-omit-empty-cursor/quickstart.md`
- [x] T020 Run the required race-enabled `make test` repository gate from `Makefile`
- [x] T021 Run `git diff --check` and verify protected generated-client and documentation boundaries for `internal/clients/camunda/`, `README.md`, and `docs/cli/`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; establishes the observable defect baseline.
- **Foundational (Phase 2)**: Depends on T001; blocks adapter work until shared cursor propagation is pinned.
- **User Story 1 (Phase 3)**: Depends on T002; delivers the reported v8.9 operational MVP.
- **User Story 2 (Phase 4)**: Depends on T002. Its v8.8 and v8.10 files can progress in parallel with later US1 validation, but T009 assumes the v8.9 builder from T004.
- **User Story 3 (Phase 5)**: Depends on T004, T010, and T011 so preservation tests evaluate the completed three-version correction.
- **Polish (Phase 6)**: Depends on all selected user-story tasks; T020 is mandatory before completion.

### User Story Dependencies

- **User Story 1 (P1)**: Starts after the shared foundation and independently fixes/proves the observed Camunda 8.9.17 workflow.
- **User Story 2 (P1)**: Starts after the shared foundation; v8.8 and v8.10 work is independent, while final three-version certification includes US1's v8.9 change.
- **User Story 3 (P2)**: Runs after the desired version adapters are corrected and independently proves that unrelated selection paths did not change.

### Within Each User Story

- Add or update the defect-specific test before changing its adapter and confirm the new assertion fails for the expected empty-cursor reason.
- Implement only the corresponding version-local page builder after its test is in place.
- Run the story's focused test before moving to its checkpoint.
- Keep generated clients, public facades, commands, and renderers unchanged unless a test exposes a specification conflict.

### Parallel Opportunities

- T007, T008, and T009 change three different adapter test files and can run in parallel after their dependencies.
- T010 and T011 change different adapter implementations and can run in parallel after T007 and T008 respectively.
- T013, T014, and T015 change different adapter test files and can run in parallel after the three implementations are ready.
- Operational validation T006 can run while v8.8/v8.10 work proceeds, provided T004 and T005 are complete and the disposable cluster is available.

---

## Parallel Example: User Story 2

```text
Task T007: Add v8.8 latest paging wire tests in internal/services/processdefinition/v88/service_test.go
Task T008: Add v8.10 latest paging wire tests in internal/services/processdefinition/v810/service_test.go

After each corresponding test fails:
Task T010: Implement v8.8 page selection in internal/services/processdefinition/v88/service.go
Task T011: Implement v8.10 page selection in internal/services/processdefinition/v810/service.go
```

## Parallel Example: User Story 3

```text
Task T013: Lock ordinary v8.8 offset behavior in internal/services/processdefinition/v88/service_test.go
Task T014: Lock ordinary v8.9 offset behavior in internal/services/processdefinition/v89/service_test.go
Task T015: Lock ordinary v8.10 offset behavior in internal/services/processdefinition/v810/service_test.go
```

---

## Implementation Strategy

### MVP First: User Story 1

1. Complete T001-T002 to establish the baseline and shared cursor invariant.
2. Complete T003-T004 to fix the v8.9 initial latest request.
3. Complete T005 to prove local selector compatibility.
4. Complete T006 against Camunda 8 Run 8.9.17 default H2/RDBMS.
5. Stop and review the MVP before expanding cross-version work.

### Incremental Delivery

1. **MVP**: v8.9 limit-only initial request plus ten-instance operational proof.
2. **Cross-version parity**: v8.8 and v8.10 request builders and per-version continuation/final-page proof.
3. **Compatibility lock**: ordinary offset, exact-version, key, filter, tenant, sort, CLI, output, and protected-tree regressions.
4. **Repository gate**: formatting, focused suites, `make test`, and final diff checks.

### Parallel Team Strategy

After T002:

- One worker completes the v8.9 MVP and live proof in T003-T006.
- A second worker prepares the v8.8 tests and implementation in T007/T010.
- A third worker prepares the v8.10 tests and implementation in T008/T011.
- After all adapters land, compatibility tasks T013-T015 split by version before shared validation.

## Notes

- `[P]` tasks touch different version files and have no dependency on another incomplete task beyond the explicit prerequisite stated in the task.
- Test assertions must inspect serialized JSON field absence, not only generated union accessors.
- A real cursor is opaque: do not trim, decode, normalize, or synthesize it.
- Do not edit generated Camunda clients, v8.7 adapters, public facade types, command metadata, or renderers for this feature.
- Live validation mutates cluster state; use only the disposable profile required by [quickstart.md](quickstart.md).
- Commit after each task or small logical group using Conventional Commits and issue #279 when applicable.
