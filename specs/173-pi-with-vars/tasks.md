# Tasks: Process Instance Variable Output

**Input**: Design documents from `/specs/173-pi-with-vars/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are required by the feature specification and constitution. Story test tasks should be written before implementation and should fail until the story implementation is complete.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files or only adds tests/docs
- **[Story]**: Maps to the user story from [spec.md](./spec.md)
- Every task names exact repository paths

## Phase 1: Setup (Shared Discovery)

**Purpose**: Confirm the current process-instance get, generated variable client, service, facade, rendering, and documentation paths before changing behavior.

- [x] T001 Inspect keyed process-instance lookup, validation, and render dispatch in `cmd/get_processinstance.go`
- [x] T002 [P] Inspect incident enrichment facade and command patterns in `c8volt/process/client.go`, `c8volt/process/model.go`, and `cmd/cmd_views_processinstance_incidents.go`
- [x] T003 [P] Inspect v8.8/v8.9 generated `/variables/search` request and response types in `internal/clients/camunda/v88/camunda/client.gen.go` and `internal/clients/camunda/v89/camunda/client.gen.go`
- [x] T004 [P] Inspect v8.7 process-instance API surface and decide whether variable search is unsupported or available through an existing client in `internal/services/processinstance/v87/contract.go`
- [x] T005 [P] Inspect process-instance command docs and generated documentation paths in `README.md`, `docs/index.md`, `docs/cli/`, and `docsgen/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared models and service/facade contracts that every variable story can reuse.

**Critical**: No user story implementation should begin until this phase is complete.

- [x] T006 Add `ProcessInstanceVariable`, `VariableEnrichedProcessInstance`, and `VariableEnrichedProcessInstances` facade models in `c8volt/process/model.go`
- [x] T007 Add matching domain variable models in `internal/domain/processinstance.go`
- [x] T008 Add process-instance service API method signatures for searching process-instance variables in `internal/services/processinstance/api.go`, `internal/services/processinstance/v87/contract.go`, `internal/services/processinstance/v88/contract.go`, and `internal/services/processinstance/v89/contract.go`
- [x] T009 Add process facade API methods for searching and enriching process instances with variables in `c8volt/process/api.go`
- [x] T010 Add domain/facade conversion helpers for process-instance variables in `c8volt/process/convert.go`
- [x] T011 Add v8.8/v8.9 variable conversion helpers and raw value/truncation decoding support in `internal/services/processinstance/v88/convert.go` and `internal/services/processinstance/v89/convert.go`
- [x] T012 Run `GOCACHE=/tmp/c8volt-gocache go test ./c8volt/process ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -run 'Test.*Variable|Test.*API' -count=1` and fix foundational compile/test failures

**Checkpoint**: Shared variable types and contracts compile before command behavior is wired.

---

## Phase 3: User Story 1 - Inspect Process Variables (Priority: P1) MVP

**Goal**: `c8volt get pi --key <key> --with-vars` renders process-instance-level variables below each selected process-instance row.

**Independent Test**: Run keyed lookup with `--with-vars` against process-scope variables and verify row output, variable sorting, scope filtering, and per-key association.

### Tests for User Story 1

- [x] T013 [P] [US1] Add service tests for v8.8 variable search request filters `processInstanceKey=<key>` and `scopeKey=<key>` in `internal/services/processinstance/v88/service_test.go`
- [x] T014 [P] [US1] Add service tests for v8.9 variable search request filters `processInstanceKey=<key>` and `scopeKey=<key>` in `internal/services/processinstance/v89/service_test.go`
- [x] T015 [P] [US1] Add facade tests proving variable enrichment preserves process-instance order and per-key association in `c8volt/process/client_test.go`
- [x] T016 [P] [US1] Add command/view tests for human `get pi --key <key> --with-vars` output with sorted indented variable lines in `cmd/get_processinstance_test.go` and `cmd/cmd_views_get_test.go`
- [x] T017 [P] [US1] Add command/view tests proving element-scoped variables with a different `scopeKey` are excluded in `cmd/get_processinstance_test.go` or `cmd/cmd_views_get_test.go`

### Implementation for User Story 1

- [x] T018 [US1] Implement v8.8 process-instance variable search in `internal/services/processinstance/v88/service.go` or a dedicated `variables.go`
- [x] T019 [US1] Implement v8.9 process-instance variable search in `internal/services/processinstance/v89/service.go` or a dedicated `variables.go`
- [x] T020 [US1] Implement explicit v8.7 behavior for `--with-vars` in `internal/services/processinstance/v87/service.go` or a dedicated `variables.go`
- [x] T021 [US1] Implement facade-level variable search and enrichment in `c8volt/process/client.go`
- [x] T022 [US1] Add `--with-vars` flag storage, reset behavior, help text, and keyed-mode validation in `cmd/get_processinstance.go` and `cmd/get_processinstance_test.go`
- [x] T023 [US1] Add human variable-enriched renderer in `cmd/cmd_views_processinstance_vars.go`
- [x] T024 [US1] Route keyed `get pi --with-vars` through variable enrichment and rendering in `cmd/get_processinstance.go`
- [x] T025 [US1] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -run 'Test.*Var|Test.*Variable' -count=1` and fix regressions

**Checkpoint**: User Story 1 is independently complete when keyed human output shows only sorted process-scope variables under the correct rows.

---

## Phase 4: User Story 2 - Keep Large Values Usable (Priority: P2)

**Goal**: Human output compacts JSON-like values, avoids CLI shortening by default, supports `--var-value-limit <chars>`, and marks truncation source precisely.

**Independent Test**: Run keyed `--with-vars` with long values, JSON-like values, API-truncated values, and `--var-value-limit`, then verify display and markers.

### Tests for User Story 2

- [x] T026 [P] [US2] Add renderer tests for compacting JSON-like object and array values to one line in `cmd/cmd_views_get_test.go`
- [x] T027 [P] [US2] Add renderer tests proving values are not CLI-shortened when `--var-value-limit` is unset or zero in `cmd/cmd_views_get_test.go`
- [x] T028 [P] [US2] Add renderer tests for `--var-value-limit <chars>` applying character-safe human shortening and `cli-truncated` in `cmd/cmd_views_get_test.go`
- [x] T029 [P] [US2] Add renderer tests for `api-truncated` and `api-truncated,cli-truncated` labels in `cmd/cmd_views_get_test.go`
- [x] T030 [P] [US2] Add validation tests for `--var-value-limit` requiring `--with-vars` and rejecting negative values in `cmd/get_processinstance_test.go`

### Implementation for User Story 2

- [x] T031 [US2] Add `--var-value-limit` flag storage, registration, help text, and reset behavior in `cmd/get_processinstance.go`
- [x] T032 [US2] Add `--var-value-limit` validation in `cmd/get_processinstance.go`
- [x] T033 [US2] Implement human variable value compaction and optional display shortening helpers in `cmd/cmd_views_processinstance_vars.go`
- [x] T034 [US2] Render variable truncation labels as `api-truncated`, `cli-truncated`, or `api-truncated,cli-truncated` in `cmd/cmd_views_processinstance_vars.go`
- [x] T035 [US2] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*Var.*(Limit|Trunc|Compact|Validation)' -count=1` and fix regressions

**Checkpoint**: User Story 2 is independently complete when human values are one-line, full by default, optionally shortened, and precisely labeled.

---

## Phase 5: User Story 3 - Provide Enriched JSON Output (Priority: P3)

**Goal**: `c8volt get pi --key <key> --with-vars --json` returns a stable enriched process-instance and variable structure with received values intact.

**Independent Test**: Run keyed JSON output with variables and verify sorted variables, required metadata, API truncation state, and unchanged received values.

### Tests for User Story 3

- [x] T036 [P] [US3] Add command JSON test for `get pi --key <key> --with-vars --json` enriched payload shape in `cmd/get_processinstance_test.go`
- [x] T037 [P] [US3] Add JSON test proving `--var-value-limit` does not alter JSON variable values in `cmd/get_processinstance_test.go`
- [x] T038 [P] [US3] Add JSON test proving variable metadata includes name, value, variable key, process instance key, scope key, tenant ID, and API truncation state when available in `cmd/get_processinstance_test.go`
- [x] T039 [P] [US3] Add facade test proving JSON-order variables are sorted by name in `c8volt/process/client_test.go`

### Implementation for User Story 3

- [x] T040 [US3] Add JSON envelope and age metadata compatibility for variable-enriched process instances in `cmd/cmd_views_processinstance_vars.go`
- [x] T041 [US3] Route `--json --with-vars` through variable-enriched JSON rendering in `cmd/get_processinstance.go`
- [x] T042 [US3] Ensure facade/service variable models preserve received values and API truncation state in `c8volt/process/convert.go` and versioned service converters
- [x] T043 [US3] Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process -run 'Test.*Var.*JSON|TestClient_.*Var' -count=1` and fix regressions

**Checkpoint**: User Story 3 is independently complete when JSON consumers can read process instances and variables together without human-display mutation.

---

## Phase 6: Documentation & Validation

**Purpose**: Finish user-facing docs, generated docs, and repository-wide proof.

- [ ] T044 [P] Update process-instance examples and `--with-vars`/`--var-value-limit` wording in `README.md`
- [ ] T045 [P] Update site documentation source examples in `docs/index.md`
- [ ] T046 Add or update command contract/help tests for `--with-vars` and `--var-value-limit` in `cmd/command_contract_test.go` or `cmd/get_processinstance_test.go`
- [ ] T047 Run `make docs-content` and fix generated documentation issues under `docs/cli/`
- [ ] T048 Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1` and fix regressions
- [ ] T049 Run `make test` and fix repository validation failures
- [ ] T050 [P] Review [quickstart.md](./quickstart.md) against implemented behavior and update if command examples or validation commands changed
- [ ] T051 Verify `git diff` contains only issue #173 implementation, docs, generated docs, and Speckit artifacts before commit

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on setup and blocks all user stories.
- **US1 (Phase 3)**: Depends on foundational contracts and models.
- **US2 (Phase 4)**: Depends on US1 renderer and variable display path.
- **US3 (Phase 5)**: Depends on US1 enrichment path and shares models with US2.
- **Documentation & Validation (Phase 6)**: Depends on final command semantics from US1-US3.

### User Story Dependencies

- **User Story 1 (P1)**: First user-visible slice; proves keyed human variable enrichment.
- **User Story 2 (P2)**: Builds on US1 human rendering and adds display controls/labels.
- **User Story 3 (P3)**: Builds on US1 enrichment and validates machine-readable output; can be implemented after US1 independently from most US2 human formatting internals.

### Parallel Opportunities

- T002 through T005 can run in parallel during setup.
- T013 through T017 can be written in parallel for US1.
- T026 through T030 can be written in parallel for US2.
- T036 through T039 can be written in parallel for US3.
- T044 and T045 can be updated in parallel after command behavior is stable.
- T050 can run in parallel with final diff review.

## Parallel Example: User Story 1

```text
Task: "Add v8.8 variable search request filter tests in internal/services/processinstance/v88/service_test.go"
Task: "Add v8.9 variable search request filter tests in internal/services/processinstance/v89/service_test.go"
Task: "Add facade per-key association tests in c8volt/process/client_test.go"
Task: "Add command/view sorted variable output tests in cmd/get_processinstance_test.go and cmd/cmd_views_get_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete User Story 1 to deliver keyed human process-instance variable inspection.
3. Stop and validate User Story 1 independently with targeted command, facade, and service tests.

### Incremental Delivery

1. Add User Story 2 for human value display controls and truncation labels.
2. Add User Story 3 for JSON consumers.
3. Complete docs and generated CLI documentation.
4. Run targeted tests, docs generation, and `make test`.

### Commit Guidance

Every commit subject for this workflow must follow Conventional Commits and end with `#173`, for example `feat(pi): add process variable enrichment #173`.
