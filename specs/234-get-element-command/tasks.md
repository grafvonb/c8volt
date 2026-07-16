# Tasks: Runtime Element Instance Command

**Input**: Design documents from `/specs/234-get-element-command/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli.md, quickstart.md

**Tests**: Required by spec SC-006 and the repository constitution. Each user story includes tests before implementation tasks.

**Organization**: Tasks are grouped by user story to enable independent validation of direct lookup, search, and output modes.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with other marked tasks in the same phase because it touches different files or has no dependency on incomplete tasks.
- **[Story]**: User story label for story phases only.
- Every task includes exact repository-relative file paths.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the feature package/file structure without behavior.

- [x] T001 Create public facade package placeholders in `c8volt/element/api.go`, `c8volt/element/model.go`, `c8volt/element/client.go`, `c8volt/element/convert.go`, and `c8volt/element/client_test.go`
- [x] T002 [P] Create internal element service package placeholders in `internal/services/element/api.go`, `internal/services/element/factory.go`, `internal/services/element/v87/service.go`, `internal/services/element/v88/contract.go`, `internal/services/element/v88/convert.go`, `internal/services/element/v88/service.go`, `internal/services/element/v88/service_test.go`, `internal/services/element/v89/contract.go`, `internal/services/element/v89/convert.go`, `internal/services/element/v89/service.go`, and `internal/services/element/v89/service_test.go`
- [x] T003 [P] Create command package placeholders in `cmd/get_element.go`, `cmd/get_element_search.go`, `cmd/get_element_test.go`, `cmd/cmd_views_element.go`, and `cmd/cmd_views_element_test.go`
- [x] T004 [P] Verify generated Camunda element instance operations are present and record no generated-client edits are needed in `specs/234-get-element-command/research.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared contracts and wiring that all user stories depend on.

**CRITICAL**: No user story implementation should begin until this phase is complete.

- [x] T005 Define version-neutral element domain types, page types, reported-total types, and search query types in `internal/domain/element.go`
- [x] T006 Define the internal element service API and version assertions in `internal/services/element/api.go`
- [x] T007 Implement the element service factory for Camunda 8.7, 8.8, and 8.9 in `internal/services/element/factory.go`
- [x] T008 Implement public element facade models and JSON field tags in `c8volt/element/model.go`
- [x] T009 Implement public/internal conversion helpers in `c8volt/element/convert.go`
- [x] T010 Implement the public element facade API and thin client delegation in `c8volt/element/api.go` and `c8volt/element/client.go`
- [x] T011 Wire ElementAPI into the aggregate c8volt client in `c8volt/client.go` and `c8volt/contract.go`

**Checkpoint**: Element facade and internal service contracts compile as empty behavior and are ready for story work.

---

## Phase 3: User Story 1 - Fetch One Runtime Element Instance (Priority: P1) MVP

**Goal**: Operators can fetch one runtime element instance by element instance key, with supported 8.8/8.9 behavior and clear 8.7 unsupported behavior.

**Independent Test**: Run `c8volt get element --key <element-instance-key>` against mocked 8.8/8.9 responses and verify one matching element instance is returned; run the same command against 8.7 and verify unsupported-version failure.

### Tests for User Story 1

- [x] T012 [P] [US1] Add v87 unsupported direct lookup service tests in `internal/services/element/v87/service_test.go`
- [x] T013 [P] [US1] Add v88 direct lookup service tests for successful payload mapping and missing/not-found handling in `internal/services/element/v88/service_test.go`
- [x] T014 [P] [US1] Add v89 direct lookup service tests for successful payload mapping and missing/not-found handling in `internal/services/element/v89/service_test.go`
- [x] T015 [P] [US1] Add facade direct lookup conversion and error mapping tests in `c8volt/element/client_test.go`
- [x] T016 [P] [US1] Add command direct lookup validation tests for `--key`, invalid keys, and `--key` plus search filters in `cmd/get_element_test.go`

### Implementation for User Story 1

- [x] T017 [US1] Implement Camunda 8.7 unsupported element operations in `internal/services/element/v87/service.go`
- [x] T018 [US1] Implement v88 generated client contract and direct lookup conversion helpers in `internal/services/element/v88/contract.go` and `internal/services/element/v88/convert.go`
- [x] T019 [US1] Implement v88 direct lookup using `GetElementInstanceWithResponse` in `internal/services/element/v88/service.go`
- [x] T020 [US1] Implement v89 generated client contract and direct lookup conversion helpers in `internal/services/element/v89/contract.go` and `internal/services/element/v89/convert.go`
- [x] T021 [US1] Implement v89 direct lookup using `GetElementInstanceWithResponse` in `internal/services/element/v89/service.go`
- [x] T022 [US1] Implement `c8volt get element --key` command parsing, validation, and facade call in `cmd/get_element.go`
- [x] T023 [US1] Implement single element human/JSON/keys rendering helpers in `cmd/cmd_views_element.go`

**Checkpoint**: User Story 1 is fully functional and independently testable as the MVP.

---

## Phase 4: User Story 2 - Search Runtime Elements For Operational Inspection (Priority: P2)

**Goal**: Operators can search runtime element instances by process, BPMN element, state, type, process definition, and BPMN process identifier with AND semantics and standard paging.

**Independent Test**: Run `c8volt get element --pi-key <process-instance-key>`, combined filters, unfiltered search, `--batch-size`, and `--limit` against mocked 8.8/8.9 paged responses and verify all returned rows match the selected filters and limits.

### Tests for User Story 2

- [x] T024 [P] [US2] Add v88 search service tests for filter mapping, AND semantics, paging, limits, and reported totals in `internal/services/element/v88/service_test.go`
- [x] T025 [P] [US2] Add v89 search service tests for filter mapping, AND semantics, paging, limits, and reported totals in `internal/services/element/v89/service_test.go`
- [x] T026 [P] [US2] Add facade search page/result mapping tests in `c8volt/element/client_test.go`
- [x] T027 [P] [US2] Add command search validation tests for `--pi-key`, `--pd-key`, `--state`, `--type`, `--batch-size`, `--limit`, and unfiltered search in `cmd/get_element_test.go`

### Implementation for User Story 2

- [x] T028 [US2] Add element search request helpers and filter detection to `c8volt/element/model.go` and `internal/domain/element.go`
- [x] T029 [US2] Implement v88 search filter construction, enum normalization, and page conversion in `internal/services/element/v88/convert.go`
- [x] T030 [US2] Implement v88 paged search and collected search result behavior in `internal/services/element/v88/service.go`
- [x] T031 [US2] Implement v89 search filter construction, enum normalization, and page conversion in `internal/services/element/v89/convert.go`
- [x] T032 [US2] Implement v89 paged search and collected search result behavior in `internal/services/element/v89/service.go`
- [x] T033 [US2] Implement facade `SearchElements` and `SearchElementsPage` delegation in `c8volt/element/api.go` and `c8volt/element/client.go`
- [x] T034 [US2] Implement `c8volt get element` search flags, search request construction, key/search mutual exclusion, and local flag validation in `cmd/get_element.go`
- [x] T035 [US2] Implement element search paging, total counting, page continuation, and limit trimming in `cmd/get_element_search.go`

**Checkpoint**: User Stories 1 and 2 both work independently, with search delivering the operational inspection flow.

---

## Phase 5: User Story 3 - Consume Element Results In Standard Output Modes (Priority: P3)

**Goal**: Operators and automation can consume element lookup/search results through compact human output, JSON, keys-only, and total-only modes.

**Independent Test**: Run equivalent element searches with default output, `--json`, `--keys-only`, and `--total`, then verify each output mode matches `contracts/cli.md`, including incident markers and quiet script output.

### Tests for User Story 3

- [ ] T036 [P] [US3] Add compact human row rendering tests for active/completed elements, missing end dates, incident markers, and timestamp formatting in `cmd/cmd_views_element_test.go`
- [ ] T037 [P] [US3] Add command output tests for JSON payload shape, keys-only output, total-only output, and mode conflicts in `cmd/get_element_test.go`
- [ ] T038 [P] [US3] Add command contract tests for `get element` flags, output modes, examples, read-only mutation metadata, and automation support in `cmd/command_contract_test.go`

### Implementation for User Story 3

- [ ] T039 [US3] Implement compact list row formatting with exactly one incident marker, `inc!` or `inc!:<incidentKey>`, plus `s:`, optional `e:`, `pi:`, `pd:`, and `element:` tags in `cmd/cmd_views_element.go`
- [ ] T040 [US3] Implement JSON list payload, keys-only output, total-only output, and final `found: N` behavior in `cmd/cmd_views_element.go` and `cmd/get_element_search.go`
- [ ] T041 [US3] Register command contract metadata, automation support, read-only mutation metadata, examples, aliases if any, and help text in `cmd/get_element.go` and `cmd/command_contract.go`
- [ ] T042 [US3] Ensure normal human output omits request, cursor, backend target, and per-page lifecycle diagnostics in `cmd/get_element_search.go` and `cmd/cmd_views_element.go`

**Checkpoint**: All output modes and CLI contracts are script-safe and operator-friendly.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, generated artifacts, validation, and cleanup across all user stories.

- [ ] T043 [P] Update README usage examples and behavioral notes for `get element` in `README.md`
- [ ] T044 [P] Add or update generated documentation coverage expectations for `get element` in `docsgen/main_test.go`
- [ ] T045 Run `gofmt` on touched Go files in `cmd/`, `c8volt/element/`, `internal/domain/`, and `internal/services/element/`
- [ ] T046 Run targeted service and facade validation with `go test ./internal/services/element/... ./c8volt/element -count=1`
- [ ] T047 Run targeted command validation with `go test ./cmd -run 'TestGetElement|TestElement|TestCommandContract' -count=1`
- [ ] T048 Regenerate CLI documentation with `make docs-content`
- [ ] T049 Run full repository validation with `make test`
- [ ] T050 Verify quickstart scenarios and update expected outputs if needed in `specs/234-get-element-command/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 Setup**: No dependencies.
- **Phase 2 Foundational**: Depends on Phase 1 file/package structure.
- **Phase 3 US1**: Depends on Phase 2 and delivers the MVP.
- **Phase 4 US2**: Depends on Phase 2; recommended after US1 because it extends the same service/facade/command files.
- **Phase 5 US3**: Depends on US1 and US2 behavior being available for rendering and mode validation.
- **Phase 6 Polish**: Depends on all desired user stories.

### User Story Dependencies

- **US1 (P1)**: Requires foundational contracts only; no dependency on US2 or US3.
- **US2 (P2)**: Requires foundational contracts and can be tested independently with search mocks; recommended after US1 to reduce same-file conflicts.
- **US3 (P3)**: Requires lookup/search results from US1 and US2 to validate all output modes.

### Within Each User Story

- Write tests before implementation tasks.
- Domain and service contracts before facade calls.
- Service adapters before command integration.
- Command integration before rendering and documentation validation.

## Parallel Opportunities

- Setup tasks T002 and T003 can proceed in parallel after T001 is understood because they create separate package areas.
- Foundational conversion/model tasks T008 and T009 can proceed in parallel with internal API/factory tasks T006 and T007 after T005 defines domain names.
- US1 tests T012-T016 can be written in parallel because they target separate packages/files.
- US2 tests T024-T027 can be written in parallel because they target separate packages/files.
- US3 tests T036-T038 can be written in parallel because they target rendering, command behavior, and contract metadata separately.
- Polish docs tasks T043 and T044 can proceed in parallel with final local validation once command metadata has stabilized.

## Parallel Example: User Story 1

```text
Task: "T013 [US1] Add v88 direct lookup service tests in internal/services/element/v88/service_test.go"
Task: "T014 [US1] Add v89 direct lookup service tests in internal/services/element/v89/service_test.go"
Task: "T015 [US1] Add facade direct lookup tests in c8volt/element/client_test.go"
Task: "T016 [US1] Add command direct lookup validation tests in cmd/get_element_test.go"
```

## Parallel Example: User Story 2

```text
Task: "T024 [US2] Add v88 search service tests in internal/services/element/v88/service_test.go"
Task: "T025 [US2] Add v89 search service tests in internal/services/element/v89/service_test.go"
Task: "T026 [US2] Add facade search tests in c8volt/element/client_test.go"
Task: "T027 [US2] Add command search validation tests in cmd/get_element_test.go"
```

## Parallel Example: User Story 3

```text
Task: "T036 [US3] Add compact row rendering tests in cmd/cmd_views_element_test.go"
Task: "T037 [US3] Add command output mode tests in cmd/get_element_test.go"
Task: "T038 [US3] Add command contract tests in cmd/command_contract_test.go"
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 setup.
2. Complete Phase 2 foundational contracts and wiring.
3. Complete Phase 3 direct lookup.
4. Validate `c8volt get element --key <element-instance-key>` and unsupported Camunda 8.7 behavior.

### Incremental Delivery

1. Deliver US1 direct lookup as the MVP.
2. Add US2 search and paging without changing US1 behavior.
3. Add US3 output-mode polish and contract metadata.
4. Finish documentation, generated docs, and full validation.

### Validation Strategy

1. Run targeted package tests after each story checkpoint.
2. Run `make docs-content` once command metadata is final.
3. Run `make test` before merge, per constitution.

## Notes

- Do not hand-edit generated Camunda clients under `internal/clients/camunda`.
- Keep backend paging, total handling, and result conversion out of command rendering code where existing service/facade patterns already provide the right home.
- Preserve compact, quiet human output; keep noisy diagnostics behind existing verbose/logging behavior.
