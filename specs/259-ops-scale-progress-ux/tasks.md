# Tasks: Ops-Scale Preflight And Progress UX

**Input**: Design documents from `specs/259-ops-scale-progress-ux/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/cli-progress-contract.md](contracts/cli-progress-contract.md), [quickstart.md](quickstart.md)

**Tests**: Required by FR-031 and FR-032. Test tasks are included before implementation tasks for each story.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files or only reads context
- **[Story]**: Maps to user stories from [spec.md](spec.md)
- Every task names exact repository paths

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm implementation context and existing local contracts before code changes begin.

- [x] T001 Read and apply Ralph implementation rules in `specs/ralph-implementation-rules.md` before any Ralph-driven implementation work
- [x] T002 Review feature artifacts `specs/259-ops-scale-progress-ux/spec.md`, `specs/259-ops-scale-progress-ux/plan.md`, `specs/259-ops-scale-progress-ux/data-model.md`, and `specs/259-ops-scale-progress-ux/contracts/cli-progress-contract.md`
- [x] T003 [P] Review existing activity writer and activity sink behavior in `toolx/logging/activity.go`, `toolx/logging/activity_test.go`, and `testx/activitysink/activity_sink.go`
- [x] T004 [P] Review existing process-instance paging and reported-total contracts in `cmd/get_processinstance_paging.go`, `internal/domain/processinstance.go`, and `internal/services/processinstance/search.go`
- [x] T005 [P] Review the current slow-process analysis discovery and enrichment flow in `cmd/ops_analyse_slow_process_instances.go`, `internal/services/ops/slow_process_analysis.go`, and `c8volt/ops/model.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared progress/preflight primitives that MUST exist before user-story implementation.

**CRITICAL**: No user story work can begin until this phase is complete.

- [x] T006 [P] Add shared progress domain types for total certainty, preflight scope, page progress, frozen-scope progress, consequence summary, and ETA samples in `internal/domain/ops_progress.go`
- [x] T007 [P] Add public facade progress types and domain mapping helpers in `c8volt/ops/progress_model.go`
- [x] T008 [P] Add activity/progress formatter unit tests for total certainty, page count wording, frozen counters, quiet gating, and ETA gating in `cmd/ops_progress_test.go`
- [x] T009 Implement shared command progress formatting and output-mode gating helpers in `cmd/ops_progress.go`
- [x] T010 Add slow-process request/result progress fields and facade mapping tests in `c8volt/ops/model.go` and `c8volt/ops/client_test.go`
- [x] T011 Add service-level progress callback plumbing and no-op safety tests in `internal/services/ops/slow_process_analysis_test.go`

**Checkpoint**: Shared progress/preflight types and command formatting helpers are available for story work.

---

## Phase 3: User Story 1 - See Scope Before Expensive Work Starts (Priority: P1) MVP

**Goal**: Broad high-volume commands show scope, count certainty, page context, and consequences before expensive work begins, with the first page reused when possible.

**Independent Test**: Run a high-volume command with a broad BPMN process selector and verify preflight appears before expensive processing, labels total certainty correctly, and does not refetch the first page.

### Tests for User Story 1

- [x] T012 [P] [US1] Add command tests for slow-process preflight wording with exact, lower-bound, and unknown totals in `cmd/ops_analyse_slow_process_instances_test.go`
- [x] T013 [P] [US1] Add service tests proving slow-process process-definition search reuses the first preflight page and does not repeat the initial search in `internal/services/ops/slow_process_analysis_test.go`
- [x] T014 [P] [US1] Add formatter contract tests for consequence summaries and broad-selector confirmation text in `cmd/ops_progress_test.go`

### Implementation for User Story 1

- [x] T015 [US1] Implement preflight-scope construction from process-instance page metadata in `internal/services/ops/slow_process_analysis.go`
- [x] T016 [US1] Refactor slow-process discovery to peek and reuse the first page during process-definition search in `internal/services/ops/slow_process_analysis.go`
- [x] T017 [US1] Map slow-process preflight and discovery metadata through the public facade in `c8volt/ops/model.go`
- [x] T018 [US1] Render slow-process preflight and consequence text through shared command helpers in `cmd/ops_analyse_slow_process_instances.go`
- [x] T019 [US1] Add interactive preflight confirmation for broad slow-process search while preserving auto-confirm and automation behavior in `cmd/ops_analyse_slow_process_instances.go`
- [x] T020 [US1] Verify explicit-key slow-process mode skips broad preflight and stays concise in `cmd/ops_analyse_slow_process_instances.go`

**Checkpoint**: User Story 1 is independently functional for slow-process process-definition search.

---

## Phase 4: User Story 2 - Track Long Work By Phase And Exact Counters (Priority: P2)

**Goal**: Long workflows show operator-facing phases, page progress, and exact done/total counters after the final scope is frozen.

**Independent Test**: Run `ops analyse slow-process-instances` with multi-page fake data and verify discovery page progress plus exact runtime-element enrichment progress without `--debug`.

### Tests for User Story 2

- [x] T021 [P] [US2] Add activity-sink tests for slow-process discovery phase updates and frozen-scope enrichment counters in `internal/services/ops/slow_process_analysis_test.go`
- [x] T022 [P] [US2] Add command tests proving default human slow-process search emits meaningful activity/progress without `--debug` in `cmd/ops_analyse_slow_process_instances_test.go`
- [x] T023 [P] [US2] Add page-progress formatting tests for known, lower-bound, and unknown page counts in `cmd/ops_progress_test.go`

### Implementation for User Story 2

- [x] T024 [US2] Emit page progress events during slow-process process-definition discovery in `internal/services/ops/slow_process_analysis.go`
- [x] T025 [US2] Emit frozen-scope progress events while loading runtime elements for slow-process analysis in `internal/services/ops/slow_process_analysis.go`
- [x] T026 [US2] Emit listener-job progress events when `--with-listeners` is used in `internal/services/ops/slow_process_analysis.go`
- [x] T027 [US2] Route slow-process progress events to `logging.UpdateActivity` with operator-facing phase names in `cmd/ops_analyse_slow_process_instances.go`
- [x] T028 [US2] Add durable verbose progress lines for slow-process discovery and enrichment without leaking endpoint details in `cmd/ops_analyse_slow_process_instances.go`
- [x] T029 [US2] Preserve final slow-process result ordering, counts, and warnings after progress integration in `cmd/cmd_views_ops_slow_process_analysis.go`

**Checkpoint**: User Stories 1 and 2 both work independently for the proof command.

---

## Phase 5: User Story 3 - Preserve Script-Safe Output Contracts (Priority: P3)

**Goal**: Progress never corrupts JSON, keys-only, quiet, or automation output.

**Independent Test**: Run affected command tests across human, verbose, quiet, JSON, keys-only, and automation modes and verify progress appears only on allowed stderr/activity channels.

### Tests for User Story 3

- [x] T030 [P] [US3] Add JSON stdout safety tests for slow-process preflight and progress in `cmd/ops_analyse_slow_process_instances_test.go`
- [x] T031 [P] [US3] Add keys-only stdout safety tests for slow-process preflight and progress in `cmd/ops_analyse_slow_process_instances_test.go`
- [x] T032 [P] [US3] Add quiet and automation mode progress-suppression tests in `cmd/ops_analyse_slow_process_instances_test.go`
- [x] T033 [P] [US3] Add command contract assertions for progress/output mode guarantees in `cmd/command_contract_test.go`

### Implementation for User Story 3

- [x] T034 [US3] Enforce shared progress channel gating for JSON, keys-only, quiet, automation, and default human modes in `cmd/ops_progress.go`
- [x] T035 [US3] Apply shared progress channel gating to slow-process command execution in `cmd/ops_analyse_slow_process_instances.go`
- [x] T036 [US3] Expose auditable preflight/frozen-scope metadata in slow-process JSON result fields without adding transient progress records in `c8volt/ops/model.go`
- [x] T037 [US3] Ensure keys-only slow-process output remains one key per line after progress integration in `cmd/cmd_views_ops_slow_process_analysis.go`

**Checkpoint**: Script-safe output contracts are preserved for the proof command.

---

## Phase 6: User Story 4 - Estimate Time Remaining Responsibly (Priority: P4)

**Goal**: Long phases show approximate rate and ETA only after enough samples exist.

**Independent Test**: Use controlled timing tests to verify ETA is omitted before the sample threshold and appears with approximate wording after enough completed work.

### Tests for User Story 4

- [x] T038 [P] [US4] Add ETA sample-window unit tests for minimum samples, unknown totals, exact totals, and approximate wording in `cmd/ops_progress_test.go`
- [x] T039 [P] [US4] Add controlled slow-process enrichment timing tests for ETA appearance and omission in `internal/services/ops/slow_process_analysis_test.go`

### Implementation for User Story 4

- [x] T040 [US4] Implement ETA sample-window calculation in `internal/domain/ops_progress.go`
- [x] T041 [US4] Add command formatter support for elapsed time, throughput, percent complete, and approximate remaining time in `cmd/ops_progress.go`
- [x] T042 [US4] Attach ETA sample updates to slow-process frozen-scope enrichment progress in `internal/services/ops/slow_process_analysis.go`
- [x] T043 [US4] Render slow-process ETA only when the shared gating rules allow it in `cmd/ops_analyse_slow_process_instances.go`

**Checkpoint**: ETA behavior is available and responsibly gated for the proof command.

---

## Phase 7: Coverage Rollout Inventory (Cross-Story Expansion)

**Purpose**: Assess all high-volume command families and create bounded follow-up implementation tasks or slices from the shared contract.

- [x] T044 [P] Add high-volume command coverage matrix for process instances, incidents, jobs, elements, process definitions, ops purge, ops repair, retention, and smoke flows in `specs/259-ops-scale-progress-ux/coverage.md`
- [x] T045 [P] Audit basic `get process-instance`, `get incident`, `get job`, and `get element` paging behavior against `specs/259-ops-scale-progress-ux/contracts/cli-progress-contract.md` and record gaps in `specs/259-ops-scale-progress-ux/coverage.md`
- [x] T046 [P] Audit `cancel process-instance`, `delete process-instance`, `walk process-instance`, and bulk `run process-instance` flows against `specs/259-ops-scale-progress-ux/contracts/cli-progress-contract.md` and record gaps in `specs/259-ops-scale-progress-ux/coverage.md`
- [x] T047 [P] Audit `ops execute retention-policy`, `ops purge orphan-process-instances`, `ops purge process-instances-with-incidents`, `ops purge all-process-definitions`, `ops repair incident`, and `ops repair process-instance` against `specs/259-ops-scale-progress-ux/contracts/cli-progress-contract.md` and record gaps in `specs/259-ops-scale-progress-ux/coverage.md`
- [x] T048 Create follow-up tasks or implementation slices for command families not completed by the proof workflow in `specs/259-ops-scale-progress-ux/tasks.md`

### Follow-up Implementation Slices From Coverage Inventory

- [x] T058 [P] Add basic inspection command tests for shared preflight/page progress and machine-output safety in `cmd/get_processinstance_test.go`, `cmd/get_incident_test.go`, `cmd/get_job_test.go`, `cmd/get_element_test.go`, and `cmd/ops_progress_test.go`
- [x] T059 Implement shared preflight/page progress routing for `get process-instance`, `get incident`, `get job`, and `get element` in `cmd/get_processinstance_search.go`, `cmd/get_incident_search.go`, `cmd/get_job_search.go`, and `cmd/get_element_search.go`
- [x] T060 Add frozen enrichment progress for basic process-instance and element listener enrichment in `internal/services/processinstance`, `internal/services/element`, `c8volt/process`, `c8volt/element`, and the matching command render paths
- [x] T061 [P] Add process-definition progress tests for broad listing and all-process-definition purge discovery in `cmd/get_processdefinition_test.go`, `cmd/ops_purge_all_processdefinitions_test.go`, and `internal/services/ops/all_process_definitions_purge_test.go`
- [x] T062 Implement shared process-definition preflight/page progress for `get process-definition` and `ops purge all-process-definitions` in `cmd/get_processdefinition.go`, `cmd/ops_purge_all_processdefinitions.go`, `c8volt/process`, and `internal/services/ops/all_process_definitions_purge.go`
- [x] T063 [P] Add process-instance mutation progress tests for destructive preflight, planning counters, mutation counters, and JSON/quiet/automation safety in `cmd/cancel_processinstance_test.go`, `cmd/delete_processinstance_test.go`, and `internal/services/processinstance`
- [x] T064 Implement shared destructive preflight and frozen planning/mutation progress for `cancel process-instance` and `delete process-instance` in `cmd/cancel_processinstance.go`, `cmd/delete_processinstance.go`, `c8volt/process`, and `internal/services/processinstance`
- [ ] T065 [P] Add ops purge and retention progress tests for candidate discovery, frozen delete planning, deletion counters, reports, and output-mode safety in `cmd/ops_execute_retention_policy_test.go`, `cmd/ops_purge_orphan_processinstances_test.go`, and `cmd/ops_purge_processinstances_with_incidents_test.go`
- [ ] T066 Implement shared preflight/page/frozen progress for retention, orphan purge, and incident-based purge workflows in `cmd/ops_execute_retention_policy.go`, `cmd/ops_purge_orphan_processinstances.go`, `cmd/ops_purge_processinstances_with_incidents.go`, and `internal/services/ops`
- [ ] T067 [P] Add ops repair progress tests for incident search, process-instance search, keyed bulk repair counters, confirmation prompts, and output-mode safety in `cmd/ops_repair_incident_test.go`, `cmd/ops_repair_processinstance_test.go`, and `internal/services/ops/repair_test.go`
- [ ] T068 Implement shared preflight and frozen repair progress for `ops repair incident` and `ops repair process-instance` in `cmd/ops_repair_incident.go`, `cmd/ops_repair_processinstance.go`, `c8volt/ops`, and `internal/services/ops/repair.go`
- [ ] T069 Add explicit-large-work progress for `walk process-instance`, bulk `run process-instance`, and `ops execute smoke-test` in `cmd/walk_processinstance.go`, `cmd/run_processinstance.go`, `cmd/ops_execute_smoketest.go`, `c8volt/process`, `c8volt/ops`, and matching service tests

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, validation, and final repository hygiene.

- [ ] T049 [P] Update slow-process help text and command examples for preflight, progress, total certainty, and `--batch-size` versus `--limit` in `cmd/ops_analyse_slow_process_instances.go`
- [ ] T050 [P] Update command contract tests for help/documentation wording in `cmd/command_contract_test.go`
- [ ] T051 Update README operational notes for ops-scale preflight and progress behavior in `README.md`
- [ ] T052 Regenerate generated CLI documentation with `make docs-content` and verify updated files under `docs/cli/`
- [ ] T053 Run focused activity tests with `GOCACHE=/tmp/c8volt-gocache go test ./toolx/logging ./testx/activitysink -count=1` for `toolx/logging/activity.go` and `testx/activitysink/activity_sink.go`
- [ ] T054 Run focused service tests with `GOCACHE=/tmp/c8volt-gocache go test ./internal/services/processinstance ./internal/services/ops -run 'Progress|Preflight|SlowProcess|SearchProcessInstances' -count=1` for `internal/services/processinstance/search.go` and `internal/services/ops/slow_process_analysis.go`
- [ ] T055 Run focused facade tests with `GOCACHE=/tmp/c8volt-gocache go test ./c8volt/process ./c8volt/ops -run 'Progress|Preflight|SlowProcess|SearchProcessInstances' -count=1` for `c8volt/process/model.go` and `c8volt/ops/model.go`
- [ ] T056 Run focused command tests with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'SlowProcess|Progress|Activity|OutputMode|KeysOnly|JSON|Automation' -count=1` for `cmd/ops_analyse_slow_process_instances.go` and `cmd/ops_progress.go`
- [ ] T057 Run full repository validation with `make test` using `Makefile`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational; recommended MVP.
- **User Story 2 (Phase 4)**: Depends on Foundational; can begin after shared progress primitives exist, but should integrate cleanly with US1 preflight work.
- **User Story 3 (Phase 5)**: Depends on Foundational; should be validated before shipping any human progress change.
- **User Story 4 (Phase 6)**: Depends on Foundational and benefits from US2 counters.
- **Coverage Rollout Inventory (Phase 7)**: Can begin after Foundational and should be finalized after proof workflow behavior is clear.
- **Polish (Phase 8)**: Depends on desired user stories and coverage inventory being complete.

### User Story Dependencies

- **US1 See Scope Before Expensive Work Starts**: MVP; no dependency on other stories after Foundational.
- **US2 Track Long Work By Phase And Exact Counters**: Can start after Foundational; expected to reuse preflight/discovery facts from US1 when both are implemented.
- **US3 Preserve Script-Safe Output Contracts**: Can start after Foundational; must pass before any progress behavior is considered complete.
- **US4 Estimate Time Remaining Responsibly**: Depends on exact progress counters from US2 for useful ETA.

### Within Each User Story

- Tests must be written first and should fail before implementation.
- Domain/facade shape changes before service callbacks.
- Service progress facts before command rendering.
- Command rendering before docs and generated CLI documentation.
- Story complete before broad rollout to additional command families.

### Parallel Opportunities

- T003, T004, and T005 are parallel context review tasks.
- T006, T007, and T008 can begin in parallel, then T009 through T011 complete the foundation.
- US1 tests T012 through T014 can be written in parallel.
- US2 tests T021 through T023 can be written in parallel.
- US3 tests T030 through T033 can be written in parallel.
- US4 tests T038 and T039 can be written in parallel.
- Coverage audits T044 through T047 can run in parallel after the shared contract is stable.

---

## Parallel Example: User Story 1

```bash
Task: "T012 [US1] Add command tests for slow-process preflight wording in cmd/ops_analyse_slow_process_instances_test.go"
Task: "T013 [US1] Add service tests proving first-page reuse in internal/services/ops/slow_process_analysis_test.go"
Task: "T014 [US1] Add formatter contract tests in cmd/ops_progress_test.go"
```

---

## Parallel Example: User Story 2

```bash
Task: "T021 [US2] Add activity-sink tests in internal/services/ops/slow_process_analysis_test.go"
Task: "T022 [US2] Add command activity/progress tests in cmd/ops_analyse_slow_process_instances_test.go"
Task: "T023 [US2] Add page-progress formatting tests in cmd/ops_progress_test.go"
```

---

## Parallel Example: User Story 3

```bash
Task: "T030 [US3] Add JSON stdout safety tests in cmd/ops_analyse_slow_process_instances_test.go"
Task: "T031 [US3] Add keys-only stdout safety tests in cmd/ops_analyse_slow_process_instances_test.go"
Task: "T032 [US3] Add quiet and automation suppression tests in cmd/ops_analyse_slow_process_instances_test.go"
Task: "T033 [US3] Add command contract assertions in cmd/command_contract_test.go"
```

---

## Parallel Example: User Story 4

```bash
Task: "T038 [US4] Add ETA sample-window unit tests in cmd/ops_progress_test.go"
Task: "T039 [US4] Add controlled enrichment timing tests in internal/services/ops/slow_process_analysis_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 setup and Phase 2 foundation.
2. Complete Phase 3 for `ops analyse slow-process-instances` preflight.
3. Stop and validate US1 independently with the command and service tests.
4. Demonstrate that broad `-b <bpmnProcessId>` no longer appears hung before expensive work begins.

### Incremental Delivery

1. Foundation: shared progress/preflight model and command gating helpers.
2. US1: preflight scope and first-page reuse for slow-process analysis.
3. US2: page and frozen-scope activity updates for the proof command.
4. US3: script-safe output-mode guarantees.
5. US4: responsible ETA.
6. Coverage inventory and follow-up slices for the remaining command families.

### Ralph Execution Guidance

Ralph-driven implementation must include:

```text
--implementation-context specs/ralph-implementation-rules.md
```

Recommended first Ralph work unit: complete T001 through T014, then stop before implementation beyond failing tests unless explicitly instructed to continue.

---

## Notes

- Keep default human progress compact and operator-facing.
- Keep endpoint names, request bodies, cursors, and low-level request lifecycle detail behind debug or verbose diagnostics.
- Do not add transient progress to JSON stdout or keys-only stdout.
- Preserve existing mutation confirmation and frozen-scope semantics.
- Run `gofmt` on touched Go files before validation.
