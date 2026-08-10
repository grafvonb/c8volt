# Tasks: Command Mode And Concern Reorganization

**Input**: Design documents from `specs/270-cmd-mode-reorg/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/), [quickstart.md](./quickstart.md)

**Tests**: Required by FR-015, SC-005, SC-006, SC-007, SC-008, SC-009, the quickstart validation guide, and the repository constitution. Add or update targeted behavior-preservation tests before each move when the current coverage does not already protect the slice.

**Implementation Context**: Every Ralph implementation iteration must read `specs/ralph-implementation-rules.md`. Ralph launch instructions must include `--implementation-context specs/ralph-implementation-rules.md`.

**Organization**: Tasks are grouped by user story so each story can be implemented and validated independently after the foundational baseline is complete.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files and has no dependency on incomplete tasks.
- **[Story]**: Maps work to the user story from `spec.md`; only story phases include story labels.
- Every task includes exact repository-relative file paths.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm project rules, current command layout, and baseline artifacts before implementation.

- [x] T001 Read `specs/ralph-implementation-rules.md` and verify no conflict with `specs/270-cmd-mode-reorg/spec.md`
- [x] T002 [P] Review the #254 ownership baseline in `specs/254-cli-debt-refactor/assessment.md` and record applicable #270 starting notes in `specs/270-cmd-mode-reorg/ownership-followups.md`
- [x] T003 [P] Review process-definition watch ownership in `cmd/get_processdefinition.go` and `cmd/get_processdefinition_test.go`, then record the current split candidates in `specs/270-cmd-mode-reorg/ownership-followups.md`
- [x] T004 [P] Review renderer ownership in `cmd/cmd_views_get.go`, `cmd/cmd_views_get_test.go`, and `cmd/cmd_views_processinstance_dryrun.go`, then record non-rendering candidates in `specs/270-cmd-mode-reorg/ownership-followups.md`
- [x] T005 [P] Review large workflow ownership in `cmd/update_job.go`, `cmd/cancel_processinstance.go`, `cmd/delete_processinstance.go`, `cmd/root.go`, `cmd/ops_analyse_slow_process_instances.go`, and `cmd/ops_progress.go`, then record split candidates in `specs/270-cmd-mode-reorg/ownership-followups.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish behavior-preservation guardrails and tracking before any user story moves source files.

**CRITICAL**: No user story work can begin until this phase is complete.

- [x] T006 Create the ownership follow-up artifact with sections for included moves, deferred ownership corrections, helper removals, and validation evidence in `specs/270-cmd-mode-reorg/ownership-followups.md`
- [x] T007 [P] Add command-file cohesion contract checks or extend existing command metadata expectations for focused mode files in `cmd/command_contract_test.go`
- [x] T008 [P] Add renderer ownership regression checks that fail if view files call public facades or internal services in `cmd/cmd_views_get_test.go`
- [x] T009 [P] Add helper caller audit notes for candidate dead helpers in `specs/270-cmd-mode-reorg/ownership-followups.md`
- [x] T010 Run `go test ./cmd -run 'TestCommandContract|Test.*View' -count=1` and record the baseline result in `specs/270-cmd-mode-reorg/quickstart.md`

**Checkpoint**: Foundation ready - user story implementation can now begin.

---

## Phase 3: User Story 1 - Maintainers Can Navigate Command Modes Safely (Priority: P1) MVP

**Goal**: Distinct command modes and lifecycles have focused ownership while base command files remain focused on construction, flags, validation, dispatch, and ordinary execution.

**Independent Test**: Reorganize one command mode at a time, review moved ownership boundaries, and run targeted tests confirming user-facing behavior is unchanged.

### Tests for User Story 1

- [x] T011 [US1] Move or add process-definition watch behavior tests beside the watch mode in `cmd/get_processdefinition_watch_test.go`
- [x] T012 [P] [US1] Add base process-definition command behavior tests that exclude watch lifecycle concerns in `cmd/get_processdefinition_test.go`
- [x] T013 [P] [US1] Add process-definition watch metadata and incompatible-mode contract assertions in `cmd/command_contract_test.go`
- [x] T014 [P] [US1] Add process-definition watch output parity assertions for human, verbose, JSON rejection, keys-only rejection, quiet rejection, and automation rejection in `cmd/get_processdefinition_watch_test.go`

### Implementation for User Story 1

- [x] T015 [US1] Move process-definition watch execution, state, timing, retry, slow-refresh status, stop status, and snapshot request construction from `cmd/get_processdefinition.go` to `cmd/get_processdefinition_watch.go`
- [x] T016 [US1] Keep process-definition command construction, flags, validation, metadata, and ordinary lookup execution focused in `cmd/get_processdefinition.go`
- [x] T017 [US1] Move process-definition watch test helpers and watch-specific scenarios from `cmd/get_processdefinition_test.go` to `cmd/get_processdefinition_watch_test.go`
- [x] T018 [US1] Audit process-definition watch moved code for unchanged output text, prompts, validation errors, exit behavior, and backend request semantics in `cmd/get_processdefinition_watch.go`
- [x] T019 [US1] Run `go test ./cmd -run 'TestGetProcessDefinition.*Watch|TestProcessDefinition.*Watch|TestValidateGetProcessDefinitionWatch|TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata' -count=1` and record the result in `specs/270-cmd-mode-reorg/quickstart.md`
- [x] T020 [US1] Run `go test ./cmd -run 'TestGetProcessDefinition|TestProcessDefinitionSelectorValidationHelpContract' -count=1` and record the non-watch compatibility result in `specs/270-cmd-mode-reorg/quickstart.md`

**Checkpoint**: User Story 1 is independently functional as the MVP.

---

## Phase 4: User Story 2 - Renderers Stay Focused On Presentation (Priority: P2)

**Goal**: Rendering ownership is organized by resource and shared presentation concern, with no backend orchestration or facade calls inside renderer files.

**Independent Test**: Review each reorganized renderer area and verify output tests pass without changes to output text, fields, ordering, or machine-readable contracts.

### Tests for User Story 2

- [x] T021 [US2] Move process-instance renderer tests from `cmd/cmd_views_get_test.go` to `cmd/cmd_views_processinstance_test.go`
- [x] T022 [P] [US2] Add process-definition renderer tests for human, JSON, keys-only, and watch list parity in `cmd/cmd_views_processdefinition_test.go`
- [x] T023 [P] [US2] Add incident renderer tests for human, JSON, keys-only, and process-instance-key output in `cmd/cmd_views_incident_test.go`
- [x] T024 [P] [US2] Add resource and tenant renderer tests for human, JSON, and keys-only output in `cmd/cmd_views_resource_test.go` and `cmd/cmd_views_tenant_test.go`
- [x] T025 [P] [US2] Add shared flat-row layout tests in `cmd/cmd_views_flat_test.go`

### Implementation for User Story 2

- [x] T026 [US2] Move shared flat-row layout helpers from `cmd/cmd_views_get.go` to `cmd/cmd_views_flat.go`
- [x] T027 [US2] Move process-instance rendering declarations from `cmd/cmd_views_get.go` to existing process-instance view files under `cmd/cmd_views_processinstance.go`, `cmd/cmd_views_processinstance_incidents.go`, `cmd/cmd_views_processinstance_vars.go`, and `cmd/cmd_views_processinstance_activity.go`
- [x] T028 [US2] Move process-definition rendering declarations from `cmd/cmd_views_get.go` to `cmd/cmd_views_processdefinition.go`
- [x] T029 [US2] Move incident rendering declarations from `cmd/cmd_views_get.go` to `cmd/cmd_views_incident.go`
- [x] T030 [US2] Move resource rendering declarations from `cmd/cmd_views_get.go` to `cmd/cmd_views_resource.go`
- [x] T031 [US2] Move tenant rendering declarations from `cmd/cmd_views_get.go` to `cmd/cmd_views_tenant.go`
- [x] T032 [US2] Remove facade calls, backend orchestration, traversal, polling, mutation planning, or workflow execution from renderer ownership and record any deferred corrections in `specs/270-cmd-mode-reorg/ownership-followups.md`
- [x] T033 [US2] Run `go test ./cmd -run 'Test.*View|TestRender|Test.*JSON|Test.*KeysOnly|Test.*Flat' -count=1` and record the renderer compatibility result in `specs/270-cmd-mode-reorg/quickstart.md`

**Checkpoint**: User Story 2 is independently functional with presentation-only renderer ownership.

---

## Phase 5: User Story 3 - Complex Command Workflows Have Reviewable Ownership (Priority: P3)

**Goal**: Planning, selection, direct-key execution, worker outcomes, progress, and reports are separated by concern so future workflow fixes remain small, testable, and reviewable.

**Independent Test**: Complete each reorganization slice independently, review focused ownership, and run affected command, workflow, and report tests before moving to the next slice.

### Tests for User Story 3

- [x] T034 [US3] Split or add process-instance dry-run presentation tests in `cmd/cmd_views_processinstance_dryrun_test.go`
- [x] T035 [P] [US3] Split process-instance search, paging, progress, and mutation-result tests into focused files under `cmd/get_processinstance_search_test.go`, `cmd/get_processinstance_paging_test.go`, and `cmd/processinstance_mutation_progress_test.go`
- [x] T036 [P] [US3] Split job update tests by command wiring, request parsing, worker outcome, and planning concern in `cmd/update_job_test.go`, `cmd/update_job_request_test.go`, `cmd/update_job_outcome_test.go`, and `cmd/update_job_plan_test.go`
- [x] T037 [P] [US3] Split process-instance cancel and delete direct-key versus selector execution tests in `cmd/cancel_processinstance_test.go`, `cmd/cancel_processinstance_selector_test.go`, `cmd/delete_processinstance_test.go`, and `cmd/delete_processinstance_selector_test.go`
- [x] T038 [P] [US3] Split root command wiring, configuration resolution, and service installation tests in `cmd/root_test.go`, `cmd/root_config_test.go`, and `cmd/root_services_test.go`
- [x] T039 [P] [US3] Split slow-process analysis command, validation, and progress tests in `cmd/ops_analyse_slow_process_instances_test.go`, `cmd/ops_analyse_slow_process_instances_validation_test.go`, and `cmd/ops_analyse_slow_process_instances_progress_test.go`
- [x] T040 [P] [US3] Split ops progress and report serialization tests in `cmd/ops_progress_test.go`, `cmd/ops_report_test.go`, `cmd/ops_report_markdown_test.go`, and `cmd/ops_report_json_test.go`

### Implementation for User Story 3

- [x] T041 [US3] Move process-instance dry-run facade calls and planning construction out of `cmd/cmd_views_processinstance_dryrun.go` into focused command or support ownership in `cmd/get_processinstance_paging.go` and record non-mechanical follow-ups in `specs/270-cmd-mode-reorg/ownership-followups.md`
- [x] T042 [US3] Keep process-instance dry-run payload and terminal rendering presentation-only in `cmd/cmd_views_processinstance_dryrun.go`
- [x] T043 [US3] Divide process-instance paging support by search request construction, paging progress, shared search progress, and mutation-result ownership across `cmd/get_processinstance_search.go`, `cmd/get_processinstance_paging.go`, `cmd/get_processinstance_total.go`, and `cmd/processinstance_mutation_progress.go`
- [x] T044 [US3] Split job update command wiring, request parsing, worker-outcome handling, and planning declarations from `cmd/update_job.go` into `cmd/update_job_request.go`, `cmd/update_job_outcome.go`, and `cmd/update_job_plan.go`
- [x] T045 [US3] Separate selector/search execution from direct-key execution for process-instance cancel in `cmd/cancel_processinstance.go` and `cmd/cancel_processinstance_selector.go`
- [ ] T046 [US3] Separate selector/search execution from direct-key execution for process-instance delete in `cmd/delete_processinstance.go` and `cmd/delete_processinstance_selector.go`
- [ ] T047 [US3] Split root command wiring, configuration resolution, and service installation from `cmd/root.go` into `cmd/root_config.go` and `cmd/root_services.go`
- [ ] T048 [US3] Split slow-process analysis command, validation, and progress declarations from `cmd/ops_analyse_slow_process_instances.go` into `cmd/ops_analyse_slow_process_instances_validation.go` and `cmd/ops_analyse_slow_process_instances_progress.go`
- [ ] T049 [US3] Separate ops progress mode selection, milestone pacing, formatting, and rendering from `cmd/ops_progress.go` into focused progress files under `cmd/ops_progress_mode.go`, `cmd/ops_progress_milestones.go`, and `cmd/ops_progress_render.go`
- [ ] T050 [US3] Move shared report-file and Markdown helpers into focused ops report files under `cmd/ops_report.go` and `cmd/ops_report_markdown.go`
- [ ] T051 [US3] Separate terminal rendering from JSON and Markdown report serialization for affected ops workflows in `cmd/cmd_views_ops_repair.go`, `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`, `cmd/cmd_views_ops_purge_all_processdefinitions.go`, and `cmd/cmd_views_ops_slow_process_analysis.go`
- [ ] T052 [US3] Confirm candidate dead helpers have no production, test, subprocess, example, or generated-artifact callers before removing them from affected `cmd/*.go` files and recording evidence in `specs/270-cmd-mode-reorg/ownership-followups.md`
- [ ] T053 [US3] Review job update planning, backend-state lookup, mutation-plan construction, process-instance orphan filtering, and limit ownership for follow-up scope in `specs/270-cmd-mode-reorg/ownership-followups.md`
- [ ] T054 [US3] Run `go test ./cmd -run 'Test.*(ProcessInstance|UpdateJob|Cancel|Delete|Root|SlowProcess|Ops.*Progress|Ops.*Report|RenderOps)' -count=1` and record the workflow compatibility result in `specs/270-cmd-mode-reorg/quickstart.md`
- [ ] T055 [US3] Run `go test ./cmd -count=1` after all workflow splits and record the command package result in `specs/270-cmd-mode-reorg/quickstart.md`

**Checkpoint**: User Story 3 is independently functional with large workflow ownership split by concern.

---

## Final Phase: Polish & Cross-Cutting Concerns

**Purpose**: Validate the complete feature, keep artifacts synchronized, and prepare for handoff.

- [ ] T056 [P] Run `gofmt` on touched Go files in `cmd/`, `c8volt/`, `internal/services/`, `internal/domain/`, and `toolx/`, then record the result in `specs/270-cmd-mode-reorg/quickstart.md`
- [ ] T057 [P] Run `git diff --check` and record the result in `specs/270-cmd-mode-reorg/quickstart.md`
- [ ] T058 Verify generated CLI documentation has no unintended diff after `make docs-content` by checking `docs/cli/`, `docs/`, and `README.md`, then record the result in `specs/270-cmd-mode-reorg/quickstart.md`
- [ ] T059 Run focused validation from `specs/270-cmd-mode-reorg/quickstart.md` and record any failures or skipped checks in `specs/270-cmd-mode-reorg/quickstart.md`
- [ ] T060 Run full repository validation with `make test` and record the result in `specs/270-cmd-mode-reorg/quickstart.md`
- [ ] T061 Verify SC-001 through SC-010 evidence is recorded in `specs/270-cmd-mode-reorg/ownership-followups.md`
- [ ] T062 Review final changed files and deferred ownership corrections, then add handoff notes to `specs/270-cmd-mode-reorg/ownership-followups.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup and blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational and is the MVP.
- **User Story 2 (Phase 4)**: Depends on Foundational; recommended after US1 so watch output and shared rendering boundaries are stable.
- **User Story 3 (Phase 5)**: Depends on Foundational; recommended after US1 and US2 so workflow moves do not mix with watch or renderer ownership churn.
- **Polish**: Depends on all desired user stories.

### User Story Dependencies

- **US1 Maintainers Can Navigate Command Modes Safely**: Can start after Foundational; no dependency on US2 or US3.
- **US2 Renderers Stay Focused On Presentation**: Can start after Foundational; should run after US1 if process-definition watch still depends on shared process-definition rendering.
- **US3 Complex Command Workflows Have Reviewable Ownership**: Can start after Foundational; should run after relevant renderer ownership is stable for workflows that emit reports or dry-run views.

### Within Each User Story

- Add or update targeted tests before moving the corresponding production code.
- Move files and declarations before making any ownership correction in the same area.
- Preserve behavior first; record non-mechanical ownership corrections as follow-ups unless they are required for a safe move.
- Run the story checkpoint validation before starting the next priority in a sequential implementation.

## Parallel Opportunities

- T002, T003, T004, and T005 can run in parallel after T001.
- T007, T008, and T009 can run in parallel after T006 creates the tracking artifact.
- US1 tests T012, T013, and T014 can run in parallel with T011 when split targets are known.
- US2 tests T022, T023, T024, and T025 can run in parallel because they target separate renderer files.
- US3 tests T035 through T040 can run in parallel because they target separate workflow areas.
- Final checks T056 and T057 can run in parallel before docs and full validation.

## Parallel Example: User Story 1

```text
Task: "T012 [P] [US1] Add base process-definition command behavior tests that exclude watch lifecycle concerns in cmd/get_processdefinition_test.go"
Task: "T013 [P] [US1] Add process-definition watch metadata and incompatible-mode contract assertions in cmd/command_contract_test.go"
Task: "T014 [P] [US1] Add process-definition watch output parity assertions in cmd/get_processdefinition_watch_test.go"
```

## Parallel Example: User Story 2

```text
Task: "T022 [P] [US2] Add process-definition renderer tests for human, JSON, keys-only, and watch list parity in cmd/cmd_views_processdefinition_test.go"
Task: "T023 [P] [US2] Add incident renderer tests for human, JSON, keys-only, and process-instance-key output in cmd/cmd_views_incident_test.go"
Task: "T024 [P] [US2] Add resource and tenant renderer tests in cmd/cmd_views_resource_test.go and cmd/cmd_views_tenant_test.go"
Task: "T025 [P] [US2] Add shared flat-row layout tests in cmd/cmd_views_flat_test.go"
```

## Parallel Example: User Story 3

```text
Task: "T036 [P] [US3] Split job update tests by command wiring, request parsing, worker outcome, and planning concern in cmd/update_job_test.go, cmd/update_job_request_test.go, cmd/update_job_outcome_test.go, and cmd/update_job_plan_test.go"
Task: "T038 [P] [US3] Split root command wiring, configuration resolution, and service installation tests in cmd/root_test.go, cmd/root_config_test.go, and cmd/root_services_test.go"
Task: "T040 [P] [US3] Split ops progress and report serialization tests in cmd/ops_progress_test.go, cmd/ops_report_test.go, cmd/ops_report_markdown_test.go, and cmd/ops_report_json_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 and Phase 2.
2. Complete Phase 3 for process-definition watch and base command separation.
3. Stop and validate `go test ./cmd -run 'TestGetProcessDefinition.*Watch|TestProcessDefinition.*Watch|TestValidateGetProcessDefinitionWatch|TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata' -count=1`.
4. Confirm `cmd/get_processdefinition.go` and `cmd/get_processdefinition_watch.go` clearly separate ordinary lookup and watch lifecycle ownership.

### Incremental Delivery

1. Deliver US1 to establish focused command mode ownership.
2. Deliver US2 to make renderer ownership presentation-only and resource-owned.
3. Deliver US3 to split larger workflows and record deferred ownership corrections.
4. Complete polish tasks for formatting, generated-doc checks, full validation, and handoff evidence.

### Parallel Team Strategy

After Phase 2, one developer can work on US1 watch ownership, another can prepare US2 renderer tests and moves, and another can prepare US3 workflow test splits. Coordinate changes that touch `cmd/cmd_views_get.go`, `cmd/get_processdefinition.go`, and process-instance mutation files because those are likely to overlap.

---

## Notes

- [P] tasks use different files or can proceed without depending on incomplete same-file edits.
- [US1], [US2], and [US3] labels map to the user stories in `specs/270-cmd-mode-reorg/spec.md`.
- Keep generated Camunda clients under `internal/clients/camunda/` untouched.
- Do not hand-edit generated CLI docs under `docs/cli/`; regenerate them with `make docs-content` only if required.
- Preserve existing user-visible command behavior unless a deferred follow-up explicitly plans a behavior change.
