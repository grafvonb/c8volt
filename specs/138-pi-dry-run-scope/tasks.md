# Tasks: Process-Instance Dry Run Scope Preview

**Input**: Design documents from `/specs/138-pi-dry-run-scope/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/process-instance-dry-run.md, quickstart.md

**Tests**: Required by the feature specification for keyed flows, search/paged flows, child-to-root escalation, full-family scope, partial orphan-parent resolution, structured output, docs, and non-mutation behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with other tasks in the same phase when files do not overlap
- **[Story]**: Which user story the task serves
- Every task includes exact repository paths

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish the dry-run contract and shared test support before story work begins.

- [x] T001 Review the dry-run scope contract in specs/138-pi-dry-run-scope/contracts/process-instance-dry-run.md against current helpers in c8volt/process/dryrun.go and cmd/cancel_processinstance.go
- [x] T002 [P] Extend dry-run mutation guard support in cmd/process_api_stub_test.go so unexpected cancel/delete calls fail in dry-run command tests
- [x] T003 [P] Add shared dry-run preview test fixtures or assertions in cmd/cancel_test.go for requested keys, roots, affected keys, traversal outcome, warnings, and mutationSubmitted=false
- [x] T004 [P] Add shared dry-run preview test fixtures or assertions in cmd/delete_test.go for requested keys, roots, affected keys, traversal outcome, warnings, and mutationSubmitted=false

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Create the reusable dry-run output and planning seam that all stories depend on.

**CRITICAL**: No user story implementation should bypass this shared seam.

- [x] T005 Define the shared process-instance dry-run preview payload and aggregate payload in cmd/cmd_views_processinstance_dryrun.go
- [x] T006 Implement human-readable dry-run rendering in cmd/cmd_views_processinstance_dryrun.go with requested count, root count, affected count, scope completeness, warnings, missing ancestors, and no-mutation text
- [x] T007 Implement structured dry-run rendering support in cmd/cmd_views_processinstance_dryrun.go using the repository's existing renderCommandResult conventions
- [x] T008 Refactor cancelProcessInstancesWithPlan in cmd/cancel_processinstance.go to compute a shared plan result that can be rendered or executed without duplicating dependency expansion
- [x] T009 Refactor deleteProcessInstancesWithPlan in cmd/delete_processinstance.go to compute a shared plan result that can be rendered or executed without duplicating dependency expansion
- [x] T010 [P] Add focused unit coverage for dry-run preview payload mapping in cmd/cancel_test.go
- [x] T011 [P] Add focused unit coverage for dry-run preview payload mapping in cmd/delete_test.go

**Checkpoint**: Shared dry-run payload and preflight seam are ready.

---

## Phase 3: User Story 1 - Preview Keyed Destructive Scope (Priority: P1) MVP

**Goal**: Direct-key cancel/delete dry run reports the same roots and affected family that real execution would target, without prompting or mutating.

**Independent Test**: Run keyed dry-run command tests for cancel and delete using child-to-root and full-family fixtures; verify no mutation calls occur.

### Tests for User Story 1

- [x] T012 [P] [US1] Add keyed cancel dry-run test for child-to-root escalation in cmd/cancel_test.go
- [x] T013 [P] [US1] Add keyed cancel dry-run test for full-family scope and zero mutation calls in cmd/cancel_test.go
- [x] T014 [P] [US1] Add keyed delete dry-run test for child-to-root escalation in cmd/delete_test.go
- [x] T015 [P] [US1] Add keyed delete dry-run test for full-family scope and zero mutation calls in cmd/delete_test.go

### Implementation for User Story 1

- [x] T016 [US1] Register `--dry-run` on cancel process-instance in cmd/cancel_processinstance.go
- [x] T017 [US1] Render and return keyed cancel dry-run previews before confirmation or CancelProcessInstances in cmd/cancel_processinstance.go
- [x] T018 [US1] Register `--dry-run` on delete process-instance in cmd/delete_processinstance.go
- [x] T019 [US1] Render and return keyed delete dry-run previews before confirmation, force-cancel, DeleteProcessInstances, or wait behavior in cmd/delete_processinstance.go
- [x] T020 [US1] Run focused keyed dry-run validation for cmd/cancel_test.go and cmd/delete_test.go using specs/138-pi-dry-run-scope/quickstart.md

**Checkpoint**: Keyed cancel/delete dry run is fully functional and independently testable.

---

## Phase 4: User Story 2 - Preview Search-Based and Paged Scope (Priority: P2)

**Goal**: Search-mode dry run uses the same paged selection and per-page dependency expansion as real cancel/delete preflight.

**Independent Test**: Run search/paged dry-run tests for cancel and delete across multiple pages and verify counts, roots, affected family keys, and page limit behavior match real preflight.

### Tests for User Story 2

- [x] T021 [P] [US2] Add search-based cancel dry-run test across multiple pages with aggregate structured output and nested per-page previews in cmd/cancel_test.go
- [x] T022 [P] [US2] Add search-based delete dry-run test across multiple pages with aggregate structured output and nested per-page previews in cmd/delete_test.go
- [x] T023 [P] [US2] Add search dry-run test covering `--batch-size` and `--limit` page selection behavior in cmd/cancel_test.go
- [x] T024 [P] [US2] Add search dry-run test covering `--batch-size` and `--limit` page selection behavior in cmd/delete_test.go

### Implementation for User Story 2

- [x] T025 [US2] Extend processPISearchPagesWithAction usage for cancel dry run in cmd/cancel_processinstance.go so each selected page renders dry-run scope without mutation
- [x] T026 [US2] Extend processPISearchPagesWithAction usage for delete dry run in cmd/delete_processinstance.go so each selected page renders dry-run scope without mutation
- [x] T027 [US2] Preserve existing search progress and limit-reached behavior for dry-run pages in cmd/get_processinstance.go
- [x] T028 [US2] Implement structured search dry-run output as an aggregate summary with nested per-page previews in cmd/cmd_views_processinstance_dryrun.go
- [x] T029 [US2] Run focused search/paged dry-run validation for cmd/cancel_test.go and cmd/delete_test.go using specs/138-pi-dry-run-scope/quickstart.md

**Checkpoint**: Search-based and paged dry run is independently functional for both destructive commands.

---

## Phase 5: User Story 3 - Preserve Orphan-Parent Warning Behavior (Priority: P3)

**Goal**: Dry run preserves partial orphan-parent traversal success, warning text, missing ancestor keys, and unresolved failure behavior.

**Independent Test**: Run dry-run tests against partial and unresolved orphan fixtures; verify partial previews succeed and unresolved plans fail.

### Tests for User Story 3

- [x] T030 [P] [US3] Add cancel dry-run partial orphan-parent test with warning and missing ancestor keys in cmd/cancel_test.go
- [x] T031 [P] [US3] Add delete dry-run partial orphan-parent test with warning and missing ancestor keys in cmd/delete_test.go
- [x] T032 [P] [US3] Add unresolved orphan dry-run failure test for cancel in cmd/cancel_test.go
- [x] T033 [P] [US3] Add unresolved orphan dry-run failure test for delete in cmd/delete_test.go
- [x] T034 [P] [US3] Confirm facade partial and unresolved dry-run coverage remains aligned in c8volt/process/client_test.go

### Implementation for User Story 3

- [x] T035 [US3] Ensure dry-run human output includes partial scope warning and missing ancestor keys in cmd/cmd_views_processinstance_dryrun.go
- [x] T036 [US3] Ensure dry-run structured output includes traversalOutcome, scopeComplete, warning, and missingAncestors in cmd/cmd_views_processinstance_dryrun.go
- [x] T037 [US3] Ensure cancel dry run returns unresolved expansion failures without mutation in cmd/cancel_processinstance.go
- [x] T038 [US3] Ensure delete dry run returns unresolved expansion failures without mutation in cmd/delete_processinstance.go
- [x] T039 [US3] Run focused orphan-parent dry-run validation for cmd/cancel_test.go, cmd/delete_test.go, and c8volt/process/client_test.go using specs/138-pi-dry-run-scope/quickstart.md

**Checkpoint**: Orphan-parent dry-run behavior matches existing dependency-expansion semantics.

---

## Phase 6: User Story 4 - Consume Dry-Run Results in Human and Structured Output (Priority: P4)

**Goal**: Humans and automation can inspect dry-run previews, and docs/help describe the non-mutating behavior.

**Independent Test**: Run human and structured output tests, inspect command help/docs, and verify README/generated CLI docs document `--dry-run`.

### Tests for User Story 4

- [x] T040 [P] [US4] Add human-readable cancel dry-run output assertions in cmd/cancel_test.go
- [x] T041 [P] [US4] Add structured cancel dry-run output assertions in cmd/cancel_test.go
- [x] T042 [P] [US4] Add human-readable delete dry-run output assertions in cmd/delete_test.go
- [x] T043 [P] [US4] Add structured delete dry-run output assertions in cmd/delete_test.go
- [x] T044 [P] [US4] Add help output assertions for cancel/delete `--dry-run` in cmd/cmd_processinstance_test.go

### Implementation for User Story 4

- [x] T045 [US4] Update cancel process-instance help and examples for `--dry-run` in cmd/cancel_processinstance.go
- [x] T046 [US4] Update delete process-instance help and examples for `--dry-run` in cmd/delete_processinstance.go
- [x] T047 [US4] Update README dry-run examples for destructive process-instance previews in README.md
- [x] T048 [US4] Regenerate generated CLI docs with `make docs-content`, updating docs/cli/c8volt_cancel_process-instance.md and docs/cli/c8volt_delete_process-instance.md
- [x] T049 [US4] Run focused human/structured output and help validation for cmd/cancel_test.go, cmd/delete_test.go, and cmd/cmd_processinstance_test.go using specs/138-pi-dry-run-scope/quickstart.md

**Checkpoint**: Dry-run output and documentation are complete for users and automation.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final cleanup, documentation consistency, and repository validation.

- [x] T050 [P] Review command mutation metadata and automation metadata in cmd/cancel_processinstance.go and cmd/delete_processinstance.go for dry-run accuracy
- [x] T051 [P] Review process-instance dry-run contract notes against implemented fields in specs/138-pi-dry-run-scope/contracts/process-instance-dry-run.md
- [x] T052 Run `gofmt` on changed Go files listed in cmd/cancel_processinstance.go, cmd/delete_processinstance.go, cmd/get_processinstance.go, cmd/cmd_views_processinstance_dryrun.go, cmd/process_api_stub_test.go, cmd/cancel_test.go, cmd/delete_test.go, cmd/cmd_processinstance_test.go, c8volt/process/api.go, c8volt/process/dryrun.go, and c8volt/process/client_test.go
- [x] T053 Run targeted tests listed in specs/138-pi-dry-run-scope/quickstart.md for cmd/cancel_test.go, cmd/delete_test.go, cmd/cmd_processinstance_test.go, and c8volt/process/client_test.go
- [x] T054 Run final repository validation with `make test` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies
- **Foundational (Phase 2)**: Depends on Setup completion and blocks all user stories
- **US1 Keyed Dry Run (Phase 3)**: Depends on Foundational and is the MVP
- **US2 Search/Paged Dry Run (Phase 4)**: Depends on Foundational; benefits from US1 shared rendering but remains independently testable through search-mode fixtures
- **US3 Orphan Handling (Phase 5)**: Depends on Foundational and can run after US1 or US2 tests establish dry-run paths
- **US4 Output/Docs (Phase 6)**: Depends on at least one implemented dry-run path, then finalizes help/docs across both commands
- **Polish (Phase 7)**: Depends on all selected user stories

### User Story Dependencies

- **User Story 1 (P1)**: MVP; no dependency on other stories after Foundational
- **User Story 2 (P2)**: Depends on shared payload/rendering from Foundational; can proceed in parallel with US3 after dry-run page contract is clear
- **User Story 3 (P3)**: Depends on shared payload/rendering from Foundational; can proceed in parallel with US2
- **User Story 4 (P4)**: Depends on implemented command flags and output fields from US1-US3

### Parallel Opportunities

- T002-T004 can run in parallel after T001.
- T010-T011 can run in parallel once T005-T009 are sketched.
- T012-T015 can run in parallel because cancel and delete keyed tests touch separate files.
- T021-T024 can run in parallel because search-mode tests are file-local.
- T030-T034 can run in parallel because orphan tests are separate or facade-only.
- T040-T044 can run in parallel across cancel, delete, and help tests.
- T050-T051 can run in parallel during final review.

---

## Parallel Example: User Story 1

```text
Task: "Add keyed cancel dry-run test for child-to-root escalation in cmd/cancel_test.go"
Task: "Add keyed cancel dry-run test for full-family scope and zero mutation calls in cmd/cancel_test.go"
Task: "Add keyed delete dry-run test for child-to-root escalation in cmd/delete_test.go"
Task: "Add keyed delete dry-run test for full-family scope and zero mutation calls in cmd/delete_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational dry-run payload and shared preflight seam.
3. Complete Phase 3: keyed cancel/delete dry run.
4. Validate User Story 1 independently with focused cmd/ tests.

### Incremental Delivery

1. Deliver keyed dry run first so operators can preview the riskiest direct destructive flows.
2. Add search/paged dry run using the same shared page orchestration.
3. Lock in orphan-parent partial and unresolved behavior.
4. Finish human/structured output polish, README, generated docs, and final validation.

### Commit Guidance

Use Conventional Commits and append the issue number as the final subject token, for example:

```text
feat(cmd): add keyed process-instance dry run #138
test(cmd): cover orphan dry-run previews #138
docs(cli): document process-instance dry run #138
```
