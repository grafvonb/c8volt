# Tasks: get incident process-instance key output

**Input**: Design documents from `/specs/198-pi-keys-only/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Included because the feature specification explicitly requires focused tests for keyed lookup, search, duplicate preservation, paging/incremental rendering, validation, docs, and delete cleanup.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Phase 1: Setup

**Purpose**: Confirm current command behavior and establish exact edit surfaces.

- [x] T001 Inspect existing incident output and validation paths in `cmd/get_incident.go`, `cmd/get_incident_search.go`, and `cmd/cmd_views_get.go`
- [x] T002 Inspect existing incident tests and docs expectations in `cmd/get_incident_test.go`, `cmd/cmd_views_get_test.go`, `cmd/command_contract_test.go`, and `docsgen/main_test.go`
- [x] T003 Inspect delete/cancel stdin dedupe behavior in `cmd/delete_processinstance.go`, `cmd/cancel_processinstance.go`, `cmd/delete_test.go`, and `cmd/cancel_test.go`

---

## Phase 2: Foundational

**Purpose**: Add shared command-local state and helpers used by every `--pi-keys-only` story.

- [x] T004 Add `flagGetIncidentPIKeysOnly` registration and help text in `cmd/get_incident.go`
- [x] T005 Add local validation in `cmd/get_incident.go` rejecting `--pi-keys-only` with `--keys-only`, `--json`, `--total`, `--error-message-limit`, or `--with-no-error-message`
- [x] T006 Add a small incident process-instance-key rendering helper in `cmd/cmd_views_get.go` that emits non-empty `ProcessInstanceKey` values and preserves duplicates

**Checkpoint**: Command parses and validates `--pi-keys-only`; rendering helper exists but stories wire it into concrete flows.

---

## Phase 3: User Story 1 - Pipe incident matches into process-instance commands (Priority: P1)

**Goal**: `get incident --pi-keys-only` emits process instance keys for keyed lookup and search/list results.

**Independent Test**: Run keyed lookup, unpaged search, duplicate-result, and paged search command tests and verify output contains only process instance keys.

### Tests for User Story 1

- [x] T007 [P] [US1] Add keyed lookup and missing-process-instance-key skip tests for `--pi-keys-only` in `cmd/get_incident_test.go`
- [x] T008 [P] [US1] Add search/list tests proving `--pi-keys-only` emits process instance keys and preserves duplicate process instance keys in `cmd/get_incident_test.go`
- [x] T009 [P] [US1] Add view-level tests for process-instance-key rendering and duplicate preservation in `cmd/cmd_views_get_test.go`
- [x] T010 [US1] Add paging/incremental rendering coverage for `--pi-keys-only` in `cmd/get_incident_test.go`

### Implementation for User Story 1

- [x] T011 [US1] Wire keyed lookup output to the process-instance-key rendering helper in `cmd/get_incident.go`
- [x] T012 [US1] Wire collected search/list output to the process-instance-key rendering helper in `cmd/cmd_views_get.go`
- [x] T013 [US1] Wire incremental search page output to process-instance-key rendering in `cmd/get_incident_search.go`
- [x] T014 [US1] Ensure `--pi-keys-only` incremental search omits `found:` summaries while preserving existing `--keys-only` summaries in `cmd/get_incident_search.go`

**Checkpoint**: User Story 1 is independently usable for pipelines and preserves duplicate process instance keys.

---

## Phase 4: User Story 2 - Avoid ambiguous output mode combinations (Priority: P2)

**Goal**: Incompatible `--pi-keys-only` combinations fail locally before remote calls.

**Independent Test**: Run validation tests combining `--pi-keys-only` with each incompatible output modifier and verify no server request is made.

### Tests for User Story 2

- [ ] T015 [P] [US2] Add validation tests for `--pi-keys-only` with `--json`, `--keys-only`, and `--total` in `cmd/get_incident_test.go`
- [ ] T016 [P] [US2] Add validation tests for `--pi-keys-only` with `--error-message-limit` and `--with-no-error-message` in `cmd/get_incident_test.go`

### Implementation for User Story 2

- [ ] T017 [US2] Finalize local mutual-exclusion diagnostics for all incompatible `--pi-keys-only` combinations in `cmd/get_incident.go`

**Checkpoint**: User Story 2 protects pipeline output mode semantics before remote calls.

---

## Phase 5: User Story 3 - Keep docs and command metadata aligned (Priority: P3)

**Goal**: Help, command metadata, README examples, and generated CLI docs describe `--pi-keys-only`.

**Independent Test**: Run help/contract/docsgen tests and inspect generated docs for the new flag and pipeline example.

### Tests for User Story 3

- [ ] T018 [P] [US3] Update help and command contract expectations for `--pi-keys-only` in `cmd/command_contract_test.go`
- [ ] T019 [P] [US3] Update generated docs tests for `--pi-keys-only` in `docsgen/main_test.go`

### Implementation for User Story 3

- [ ] T020 [US3] Update `get incident` long help and examples in `cmd/get_incident.go`
- [ ] T021 [US3] Update incident pipeline examples in `README.md`
- [ ] T022 [US3] Regenerate CLI reference docs with `make docs-content`, updating `docs/cli/c8volt_get.md`, `docs/cli/c8volt_get_incident.md`, and `docs/cli/index.md`

**Checkpoint**: User Story 3 makes the new pipeline mode discoverable and documented.

---

## Phase 6: User Story 4 - Normalize delete process-instance duplicate stdin handling (Priority: P4)

**Goal**: `delete pi` dedupes merged key input at the command boundary like `cancel pi`.

**Independent Test**: Provide duplicate stdin keys to `delete pi` and verify planning receives unique keys while duplicate `get incident --pi-keys-only` output remains unchanged.

### Tests for User Story 4

- [ ] T023 [P] [US4] Add delete duplicate stdin regression coverage in `cmd/delete_test.go`
- [ ] T024 [P] [US4] Add or update cancel duplicate stdin coverage only if needed to document existing parity in `cmd/cancel_test.go`

### Implementation for User Story 4

- [ ] T025 [US4] Update `cmd/delete_processinstance.go` to call `.Unique()` immediately after merged flag/stdin key validation
- [ ] T026 [US4] Update `delete pi --key` flag help to mention repeated flags and stdin `-` in `cmd/delete_processinstance.go`

**Checkpoint**: User Story 4 completes the small cleanup without changing incident output semantics.

---

## Phase 7: Polish & Validation

**Purpose**: Confirm behavior, docs, and repository validation.

- [ ] T027 Run targeted command tests with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestGetIncidentCommand_.*PIKeysOnly|TestListIncidentsView_.*PIKeysOnly|TestDeleteProcessInstanceCommand_.*Duplicate' -count=1`
- [ ] T028 Run broader affected validation with `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./docsgen -count=1`
- [ ] T029 Run repository validation with `make test`
- [ ] T030 Review `git diff` to confirm the change remains focused on `--pi-keys-only`, docs, tests, and the local `delete pi` dedupe cleanup

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on setup context.
- **User Story 1 (Phase 3)**: Depends on foundational flag/helper work and delivers the MVP.
- **User Story 2 (Phase 4)**: Depends on foundational validation state and can proceed after T004-T006.
- **User Story 3 (Phase 5)**: Depends on the final flag behavior from US1 and US2.
- **User Story 4 (Phase 6)**: Independent of US1-US3 except for preserving the duplicate-output contract.
- **Polish (Phase 7)**: Depends on all selected stories.

### User Story Dependencies

- **US1**: MVP; no dependency on other stories after Phase 2.
- **US2**: Can be implemented after Phase 2; should be complete before final docs.
- **US3**: Depends on final user-facing behavior from US1 and US2.
- **US4**: Can be implemented after setup; must not alter US1 duplicate output.

### Parallel Opportunities

- T001-T003 can be split as read-only inspection.
- T007-T009 can run in parallel because they cover different test surfaces.
- T015-T016 can run in parallel because they cover separate validation combinations.
- T018-T019 can run in parallel because command contract and docsgen tests are separate files.
- T023-T024 can run in parallel if cancel parity coverage is needed.

## Parallel Example: User Story 1

```bash
Task: "Add keyed lookup and missing-process-instance-key skip tests for `--pi-keys-only` in cmd/get_incident_test.go"
Task: "Add search/list tests proving `--pi-keys-only` emits process instance keys and preserves duplicate process instance keys in cmd/get_incident_test.go"
Task: "Add view-level tests for process-instance-key rendering and duplicate preservation in cmd/cmd_views_get_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete User Story 1.
3. Run targeted US1 tests.
4. Stop and verify that `get incident --pi-keys-only` can feed a process-instance command.

### Incremental Delivery

1. Deliver US1 for pipeline output.
2. Add US2 for local guardrails.
3. Add US3 for docs and command metadata.
4. Add US4 cleanup only after confirming it stays local and focused.
5. Run Phase 7 validation.
