---

description: "Dependency-ordered implementation tasks for stable tenant-aware process-definition ordering"
---

# Tasks: Stable Tenant-Aware Process-Definition Ordering

**Input**: Design documents from `specs/286-tenant-pd-ordering/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/process-definition-ordering.md`, `quickstart.md`, and `specs/ralph-implementation-rules.md`

**Tests**: Automated tests are required by FR-018 and the project constitution. Add the listed tests before their corresponding implementation; new-behavior tests should demonstrate the missing behavior, while regression tests may establish an already-preserved boundary.

**Organization**: Tasks are grouped by user story so each increment has an explicit independent test and can be validated before the next priority.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel once its stated dependencies are complete because it owns different files from concurrent tasks
- **[Story]**: Maps a task to User Story 1, 2, or 3 from `spec.md`
- Every checklist item names the exact repository file or validation artifact it affects

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish a clean implementation baseline without introducing new project structure or dependencies.

- [x] T001 Run the pre-change focused validation commands and record any baseline failures before editing, using `specs/286-tenant-pd-ordering/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisite)

**Purpose**: Define the one version-neutral ordering primitive required by every story.

**CRITICAL**: Complete this phase before any user-story implementation.

- [x] T002 Add table-driven canonical comparator tests for `<default>` placement, case-sensitive tenant/BPMN identity, versions 9 and 10, lexical keys `10` and `2`, empty/single collections, and shuffled inputs in `internal/domain/processdefinition_test.go`
- [x] T003 Implement and document the reusable canonical process-definition comparator and sort function `(tenantId ASC, bpmnProcessId ASC, version DESC, key ASC)` without changing unrelated sort helpers in `internal/domain/processdefinition.go`

**Checkpoint**: The domain package has a deterministic, reusable comparator whose focused tests pass.

---

## Phase 3: User Story 1 - Scan Definitions by Tenant and Process (Priority: P1) MVP

**Goal**: Return ordinary process-definition collections grouped by exact tenant and BPMN process, with newest versions first and opaque keys as the final tie-breaker.

**Independent Test**: Feed a shuffled collection containing at least two tenants, two BPMN process IDs, versions 9 and 10, case-only identifier differences, and tied text keys through filtered and `--all-tenants` listing paths; assert the exact canonical key sequence.

### Tests for User Story 1

- [x] T004 [P] [US1] Add page-arrival independence and final canonical collection-order tests to `internal/services/processdefinition/search_test.go`
- [x] T005 [P] [US1] Add public ordinary-search and paged-search order-preservation tests to `c8volt/process/client_test.go`
- [x] T006 [P] [US1] Add command-level filtered and `--all-tenants` canonical ordering tests to `cmd/get_processdefinition_test.go`
- [x] T007 [P] [US1] Add Camunda 8.7 request-sort and returned-order assertions for tenant/BPMN/version/key to `internal/services/processdefinition/v87/service_test.go`
- [x] T008 [P] [US1] Add Camunda 8.8 ordinary request-sort and returned-order assertions to `internal/services/processdefinition/v88/service_test.go`
- [x] T009 [P] [US1] Add Camunda 8.9 ordinary request-sort and returned-order assertions to `internal/services/processdefinition/v89/service_test.go`
- [x] T010 [P] [US1] Add Camunda 8.10 ordinary request-sort and returned-order assertions to `internal/services/processdefinition/v810/service_test.go`

### Implementation for User Story 1

- [x] T011 [P] [US1] Encode Operate 8.7 backend sorting as tenant ID ASC, BPMN process ID ASC, version DESC, and key ASC, then use the domain canonical sort for returned collections in `internal/services/processdefinition/v87/service.go`
- [x] T012 [P] [US1] Encode Camunda 8.8 ordinary backend sorting as tenant ID ASC, process definition ID ASC, version DESC, and process definition key ASC, then use the domain canonical sort for returned collections in `internal/services/processdefinition/v88/service.go`
- [x] T013 [P] [US1] Encode Camunda 8.9 ordinary backend sorting as tenant ID ASC, process definition ID ASC, version DESC, and process definition key ASC, then use the domain canonical sort for returned collections in `internal/services/processdefinition/v89/service.go`
- [x] T014 [P] [US1] Encode Camunda 8.10 ordinary backend sorting as tenant ID ASC, process definition ID ASC, version DESC, and process definition key ASC, then use the domain canonical sort for returned collections in `internal/services/processdefinition/v810/service.go`
- [x] T015 [US1] Canonically sort the accumulated ordinary result only after service-owned page traversal while preserving exact-once membership, visitor progress metadata, and existing ordinary limit semantics in `internal/services/processdefinition/search.go`

**Checkpoint**: User Story 1 passes independently for direct public search, paged facade search, filtered CLI search, and cross-tenant CLI search.

---

## Phase 4: User Story 2 - Keep Ordering Consistent Across Views (Priority: P2)

**Goal**: Preserve the canonical sequence through statistics enrichment, public conversion, human/JSON/keys-only rendering, and watch refreshes.

**Independent Test**: Run the same controlled collection with and without statistics and through human, JSON, keys-only, and two watch snapshots whose counts differ; assert an identical ordered key sequence everywhere.

### Tests for User Story 2

- [x] T016 [P] [US2] Add facade conversion tests proving canonical slice order and per-key statistics association survive public mapping in `c8volt/process/client_test.go`
- [x] T017 [P] [US2] Expand the shared renderer fixture and assert identical canonical key sequences for human, JSON, keys-only, and watch rendering in `cmd/cmd_views_processdefinition_test.go`
- [x] T018 [P] [US2] Add watch-snapshot tests proving statistics-only changes retain canonical positions and broad snapshots reuse paged collection order in `internal/services/processdefinition/search_test.go`
- [x] T019 [P] [US2] Add repeated-refresh command tests proving count changes do not move process-definition rows in `cmd/get_processdefinition_watch_test.go`

### Implementation and Version Integration for User Story 2

- [x] T020 [P] [US2] Add with-stat parity coverage and keep Camunda 8.8 statistics enrichment associated by slice index without reordering in `internal/services/processdefinition/v88/service_test.go` and `internal/services/processdefinition/v88/service.go`
- [x] T021 [P] [US2] Add with-stat parity coverage and keep Camunda 8.9 statistics enrichment associated by slice index without reordering in `internal/services/processdefinition/v89/service_test.go` and `internal/services/processdefinition/v89/service.go`
- [x] T022 [P] [US2] Add with-stat parity coverage and keep Camunda 8.10 statistics enrichment associated by slice index without reordering in `internal/services/processdefinition/v810/service_test.go` and `internal/services/processdefinition/v810/service.go`

**Checkpoint**: User Story 2 proves that existing order-preserving converters, enrichers, renderers, and watch lifecycle expose one sequence and do not require an output-schema change.

---

## Phase 5: User Story 3 - Preserve Complete and Compatible Discovery (Priority: P3)

**Goal**: Make paging and tenant-aware latest selection complete and deterministic across Camunda 8.7-8.10 without regressing direct-key or XML retrieval.

**Independent Test**: Retrieve one controlled population with page sizes 1, 2, and 1000 and with `--latest`; verify identical exact-once canonical keys, one newest definition per exact tenant/BPMN group, cross-version parity, the documented 8.7 ceiling, and unchanged direct-key/XML behavior.

### Tests for User Story 3

- [x] T023 [P] [US3] Add service tests for complete cursor/offset traversal, page sizes 1/2/1000, exact tenant/BPMN latest grouping, tied-version lexical key choice, post-reduction limiting, and latest watch paging in `internal/services/processdefinition/search_test.go`
- [x] T024 [P] [US3] Add facade tests for mapping `Latest`, complete latest results, ordered conversion, visitor behavior, and domain-error conversion in `c8volt/process/client_test.go`
- [x] T025 [P] [US3] Add Camunda 8.7 tests for canonical Operate paging, tenant-aware local latest selection, lexical key ties, and the retained 1000-definition compatibility ceiling in `internal/services/processdefinition/v87/service_test.go`
- [x] T026 [P] [US3] Add Camunda 8.8 tests for native `isLatestVersion`, tenant-then-process latest sort, continuation metadata, and multi-page latest requests in `internal/services/processdefinition/v88/service_test.go`
- [x] T027 [P] [US3] Add Camunda 8.9 tests for native `isLatestVersion`, tenant-then-process latest sort, continuation metadata, and multi-page latest requests in `internal/services/processdefinition/v89/service_test.go`
- [x] T028 [P] [US3] Add Camunda 8.10 tests for native `isLatestVersion`, tenant-then-process latest sort, continuation metadata, and multi-page latest requests in `internal/services/processdefinition/v810/service_test.go`
- [ ] T029 [P] [US3] Add CLI tests for `--latest` across tenants, selector validation, page-size invariance, unchanged key retrieval, and unchanged XML mode in `cmd/get_processdefinition_test.go` and `cmd/process_definition_selector_validation_test.go`

### Implementation for User Story 3

- [x] T030 [US3] Add the `Latest` intent to domain/public search requests and map it without changing serialized response contracts in `internal/domain/processdefinition.go`, `c8volt/process/model.go`, and `c8volt/process/convert.go`
- [x] T031 [US3] Extend service-owned traversal to collect complete latest candidates, reduce by exact `(tenantId, bpmnProcessId)` with version/key tie rules, sort canonically, apply latest limits after reduction, and route latest watch snapshots through the same path in `internal/services/processdefinition/search.go`
- [x] T032 [US3] Route `SearchProcessDefinitionsLatest` through the shared paged service request with `Latest: true`, preserving facade error conversion and result order in `c8volt/process/client.go`
- [x] T033 [P] [US3] Make the Camunda 8.7 compatibility latest wrapper group by exact tenant and BPMN IDs, resolve equal-version keys lexically, and retain its documented 1000-item bound in `internal/services/processdefinition/v87/service.go`
- [x] T034 [P] [US3] Make Camunda 8.8 latest pages use native filtering with tenant ID ASC then process definition ID ASC while leaving final normalization to the shared service in `internal/services/processdefinition/v88/service.go`
- [x] T035 [P] [US3] Make Camunda 8.9 latest pages use native filtering with tenant ID ASC then process definition ID ASC while leaving final normalization to the shared service in `internal/services/processdefinition/v89/service.go`
- [x] T036 [P] [US3] Make Camunda 8.10 latest pages use native filtering with tenant ID ASC then process definition ID ASC while leaving final normalization to the shared service in `internal/services/processdefinition/v810/service.go`
- [ ] T037 [US3] Route broad `--latest` discovery and selector validation through the canonical facade collection path while preserving direct-key, XML, progress, and watch dispatch behavior in `cmd/get_processdefinition.go`, `cmd/process_definition_selector_validation.go`, and `cmd/get_processdefinition_watch.go`

**Checkpoint**: All three user stories work independently, with complete/latest collection mechanics below `cmd` and the public facade.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Align documentation, generated artifacts, formatting, and repository-wide validation with the completed contract.

- [ ] T038 Update canonical-order wording, latest grouping, exact comparison rules, and the Camunda 8.7 compatibility note in `cmd/get_processdefinition.go` and `README.md`
- [ ] T039 Regenerate and review process-definition CLI documentation with `make docs-content`, accepting generated changes in `docs/cli/c8volt_get_process-definition.md`
- [ ] T040 Run `gofmt` on all touched Go files and execute the focused commands from `specs/286-tenant-pd-ordering/quickstart.md`
- [ ] T041 Run the repository static validation target `make vet` against the implementation described by `specs/286-tenant-pd-ordering/plan.md`
- [ ] T042 Run the required race-enabled full suite `make test` and resolve failures against `specs/286-tenant-pd-ordering/contracts/process-definition-ordering.md`
- [ ] T043 Run `git diff --check`, review command-file declaration ownership against `AGENTS.md`, and verify every acceptance item in `specs/286-tenant-pd-ordering/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependency; establishes the baseline.
- **Foundational (Phase 2)**: Depends on T001 and blocks every user story.
- **User Story 1 (Phase 3)**: Depends on the canonical comparator from T003 and establishes the MVP collection contract.
- **User Story 2 (Phase 4)**: Depends on User Story 1 because it verifies that canonical collection through existing enrichment and presentation boundaries.
- **User Story 3 (Phase 5)**: Depends on User Story 1 but not User Story 2; it may proceed in parallel with Phase 4 when file ownership is coordinated.
- **Polish (Phase 6)**: Depends on all selected user stories; T039 follows T038, and T040-T043 run in order.

### User Story Dependencies

- **US1 (P1)**: T004-T010 follow T003; T011-T014 follow their matching adapter tests; T015 follows T004 and completes the service contract.
- **US2 (P2)**: T016-T022 follow US1 and can run independently across their listed test files.
- **US3 (P3)**: T023-T029 establish the missing/regression coverage; T030 precedes T031, T032, and T037; T031 precedes T032; T033-T036 follow their matching adapter tests; T037 completes CLI integration after T032.

### Within Each User Story

- Add the story's tests before its production changes.
- Keep backend request differences in the matching `v87`, `v88`, `v89`, or `v810` service.
- Keep traversal, latest reduction, final sorting, and limit trimming in `internal/services/processdefinition/search.go`.
- Keep the public facade thin and order-preserving.
- Keep command work limited to dispatch, validation, progress, help, and presentation wiring.
- Run the story's focused tests at its checkpoint before moving to the next priority.

### Parallel Opportunities

- US1 facade, command, service, and four adapter test tasks T004-T010 can proceed in parallel after T003.
- US1 adapter implementations T011-T014 can proceed in parallel after their corresponding test tasks.
- All US2 verification tasks T016-T022 own different test files and can proceed in parallel after US1.
- US3 service, facade, command, and adapter test tasks T023-T029 can proceed in parallel after US1.
- US3 adapter implementations T033-T036 can proceed in parallel; US2 can also proceed alongside US3 when neither task edits the same test file.

---

## Parallel Example: User Story 1

```text
Task T004: Add service collection-order tests in internal/services/processdefinition/search_test.go
Task T005: Add facade order-preservation tests in c8volt/process/client_test.go
Task T006: Add CLI ordering tests in cmd/get_processdefinition_test.go
Task T007-T010: Add version-adapter request-sort tests in each v87-v810 service_test.go

After the adapter tests exist:
Task T011-T014: Implement each version adapter in parallel
```

## Parallel Example: User Story 2

```text
Task T016: Verify facade conversion and statistics association
Task T017: Verify human, JSON, keys-only, and watch renderer parity
Task T018-T019: Verify service and command watch stability
Task T020-T022: Verify with-stat parity for v88, v89, and v810 in parallel
```

## Parallel Example: User Story 3

```text
Task T023: Add shared traversal/latest tests
Task T024: Add facade latest mapping tests
Task T025-T028: Add version-specific latest tests in parallel
Task T029: Add CLI latest and regression tests

After request and traversal models are ready:
Task T033-T036: Implement version-specific latest behavior in parallel
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete T001-T003.
2. Complete T004-T015.
3. Run the User Story 1 focused domain, service, facade, adapter, and command tests.
4. Stop and validate the canonical ordinary-listing sequence independently.

### Incremental Delivery

1. **Foundation**: Establish the comparator and its edge-case tests.
2. **MVP / US1**: Stabilize ordinary backend requests and final collection order.
3. **US2**: Lock the same order through statistics, views, and watch refreshes.
4. **US3**: Complete paging and tenant-aware latest behavior across all supported versions.
5. **Polish**: Regenerate documentation and pass focused, vet, race, and diff checks.

### Ralph Execution Discipline

- Every Ralph iteration MUST read `specs/ralph-implementation-rules.md` and use `--implementation-context specs/ralph-implementation-rules.md`.
- Treat one checklist item or one tightly coupled test/implementation pair as the iteration work unit.
- Preserve unrelated working-tree changes and do not edit generated Camunda clients.
- Use Conventional Commit subjects and append `#286` as the final token when commits are created.

## Notes

- `[P]` means parallelizable only after the dependencies above are satisfied.
- Existing order-preserving code may need only regression coverage; do not refactor it without a failing contract test.
- Do not add a user-selectable sort flag, new output fields, a new dependency, or generated-client edits.
- Keep the Camunda 8.7 1000-definition ceiling explicit in code comments, tests, and documentation.
- Stop at each checkpoint and run the closest tests before proceeding.
