# Tasks: CLI Debt Refactor

**Input**: Design documents from `specs/254-cli-debt-refactor/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/), [quickstart.md](./quickstart.md)

**Tests**: Required by FR-013, the quickstart validation guide, and the repository constitution. Write or update targeted tests before behavior changes in each story.

**Implementation Context**: Every Ralph implementation iteration must read `specs/ralph-implementation-rules.md`. Ralph launch instructions must include `--implementation-context specs/ralph-implementation-rules.md`.

**Organization**: Tasks are grouped by user story so each story can be implemented and validated independently after the assessment baseline is complete.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files and has no dependency on incomplete tasks.
- **[Story]**: Maps work to the user story from `spec.md`; only story phases include story labels.
- Every task includes exact repository-relative file paths.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm project rules, active artifacts, and existing local patterns before implementation.

- [x] T001 Read `specs/ralph-implementation-rules.md` and verify no conflict with `specs/254-cli-debt-refactor/spec.md`
- [x] T002 [P] Review basic paging implementations in `cmd/get_job_search.go`, `cmd/get_element_search.go`, `cmd/get_incident_search.go`, `cmd/get_processinstance_search.go`, and record findings in `specs/254-cli-debt-refactor/assessment.md`
- [x] T003 [P] Review process-instance mutation paging in `cmd/get_processinstance_paging.go`, `cmd/cancel_processinstance.go`, `cmd/delete_processinstance.go`, and record findings in `specs/254-cli-debt-refactor/assessment.md`
- [x] T004 [P] Review high-level ops workflow ownership in `internal/services/ops/`, `cmd/ops_*.go`, and record findings in `specs/254-cli-debt-refactor/assessment.md`
- [x] T005 [P] Review command output, activity, and capability helpers in `cmd/root.go`, `cmd/command_contract.go`, `toolx/logging/activity.go`, and record findings in `specs/254-cli-debt-refactor/assessment.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Create the baseline assessment and validation scaffolding required before any refactor slice starts.

**CRITICAL**: No user story work can begin until this phase is complete.

- [x] T006 Create the command behavior assessment structure with all required columns in `specs/254-cli-debt-refactor/assessment.md`
- [x] T007 Populate all 55 command-node classifications in `specs/254-cli-debt-refactor/assessment.md`
- [x] T008 Add high-risk workflow and duplicated-mechanics findings to `specs/254-cli-debt-refactor/assessment.md`
- [x] T009 Add intentional ops-workflow differences and non-goals to `specs/254-cli-debt-refactor/assessment.md`
- [x] T010 [P] Add command tree count and assessment completeness assertions in `cmd/command_contract_test.go`
- [x] T011 [P] Add assessment artifact validation expectations in `docsgen/main_test.go`
- [x] T012 Run `go test ./cmd -run 'TestCommandContract|TestCapability' -count=1` and record the result in `specs/254-cli-debt-refactor/quickstart.md`

**Checkpoint**: All command nodes are classified and refactor slices can be selected without rediscovering command ownership.

---

## Phase 3: User Story 1 - Consistent Output During Large CLI Workflows (Priority: P1) MVP

**Goal**: Operators and automation get consistent progress, prompts, summaries, and clean machine output during long-running and paged workflows.

**Independent Test**: Representative paged read and bulk mutation commands pass human, verbose, quiet, JSON, keys-only, automation, and no-indicator output checks without corrupting machine output.

### Tests for User Story 1

- [x] T013 [P] [US1] Add paged search verbose progress tests for job and element output in `cmd/get_job_test.go` and `cmd/get_element_test.go`
- [x] T014 [P] [US1] Add paged search verbose progress tests for incident and process-instance output in `cmd/get_incident_test.go` and `cmd/get_processinstance_test.go`
- [x] T015 [P] [US1] Add machine-output cleanliness tests for JSON, keys-only, quiet, automation, and no-indicator modes in `cmd/cmd_json_assertions_test.go`
- [x] T016 [P] [US1] Add ops discovery summary renderer tests for complete vs user-limited discovery in `cmd/cmd_views_ops_repair_test.go`, `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`, and `cmd/cmd_views_ops_purge_all_processdefinitions.go`
- [x] T017 [P] [US1] Add shared activity indicator enablement tests for quiet, automation, no-indicator, and JSON log constraints in `toolx/logging/activity_test.go` and `cmd/root_test.go`

### Implementation for User Story 1

- [x] T018 [US1] Define the CLI progress policy artifact in `specs/254-cli-debt-refactor/progress-policy.md`
- [x] T019 [US1] Align command-level progress helper behavior with `specs/254-cli-debt-refactor/progress-policy.md` in `cmd/get_processinstance_paging.go`
- [x] T020 [US1] Align job search progress, prompt, incremental rendering, and found-summary behavior with the policy in `cmd/get_job_search.go`
- [x] T021 [US1] Align element search progress, prompt, incremental rendering, and found-summary behavior with the policy in `cmd/get_element_search.go`
- [x] T022 [US1] Align incident search progress, prompt, process-instance-key output, and found-summary behavior with the policy in `cmd/get_incident_search.go`
- [x] T023 [US1] Align process-instance search progress, prompt, machine-output silence, and warning-stop behavior with the policy in `cmd/get_processinstance_search.go`
- [x] T024 [US1] Align ops discovery summaries with the policy in `cmd/cmd_views_ops_processinstance_scope.go`, `cmd/cmd_views_ops_repair.go`, and `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`
- [x] T025 [US1] Run `go test ./cmd -run 'TestGet(Job|Element|Incident|ProcessInstance)|Test.*JSON|Test.*KeysOnly|Test.*Automation|Test.*NoIndicator|Test.*Prompt' -count=1` and record the result in `specs/254-cli-debt-refactor/quickstart.md`

**Checkpoint**: User Story 1 is independently functional as the MVP and machine output remains clean.

---

## Phase 4: User Story 2 - Clear Ownership of Paging and Discovery Behavior (Priority: P2)

**Goal**: Command code owns CLI behavior while backend paging, discovery, query strategy, local filtering, and mutation planning move below `cmd` where appropriate.

**Independent Test**: The assessment and tests show command-owned paging debt is reduced for basic searches and process-instance mutation planning without changing user-visible behavior.

### Tests for User Story 2

- [x] T026 [P] [US2] Add service/facade page-collection tests for job search in `c8volt/job/client_test.go` and `internal/services/job/v88/service_test.go`
- [x] T027 [P] [US2] Add service/facade page-collection tests for element search in `c8volt/element/client_test.go` and `internal/services/element/v88/service_test.go`
- [ ] T028 [P] [US2] Add service/facade page-collection tests for incident search in `c8volt/incident/client_test.go` and `internal/services/incident/v88/incidents_test.go`
- [ ] T029 [P] [US2] Add service/facade page-collection and local-filtering tests for process-instance search in `c8volt/process/client_test.go` and `internal/services/processinstance/v88/service_test.go`
- [ ] T030 [P] [US2] Add cancel/delete discovery planning tests for process-instance mutation paging in `c8volt/process/client_test.go` and `cmd/delete_test.go`

### Implementation for User Story 2

- [ ] T031 [US2] Add version-neutral paged search result contracts for job discovery in `c8volt/job/model.go` and `internal/domain/job.go`
- [ ] T032 [US2] Move job page walking, total fallback, and limit trimming below command ownership in `c8volt/job/client.go` and `internal/services/job/api.go`
- [ ] T033 [US2] Reduce command-owned job paging to rendering and prompts in `cmd/get_job_search.go`
- [ ] T034 [US2] Add version-neutral paged search result contracts for element discovery in `c8volt/element/model.go` and `internal/domain/element.go`
- [ ] T035 [US2] Move element page walking, total fallback, and limit trimming below command ownership in `c8volt/element/client.go` and `internal/services/element/api.go`
- [ ] T036 [US2] Reduce command-owned element paging to rendering and prompts in `cmd/get_element_search.go`
- [ ] T037 [US2] Add version-neutral paged search result contracts for incident discovery in `c8volt/incident/model.go` and `internal/domain/incident.go`
- [ ] T038 [US2] Move incident page walking, cursor handling, process-instance-key projection, and local filtering below command ownership in `c8volt/incident/client.go` and `internal/services/incident/api.go`
- [ ] T039 [US2] Reduce command-owned incident paging to rendering and prompts in `cmd/get_incident_search.go`
- [ ] T040 [US2] Move process-instance query strategy, page traversal, total fallback, and local compatibility filtering below command ownership in `c8volt/process/client.go` and `internal/services/processinstance/api.go`
- [ ] T041 [US2] Reduce command-owned process-instance search to validation, rendering, prompts, and mode selection in `cmd/get_processinstance_search.go`
- [ ] T042 [US2] Move process-instance cancel/delete discovery and mutation planning traversal below command ownership in `c8volt/process/client.go` and `internal/services/processinstance/dryrun.go`
- [ ] T043 [US2] Reduce command-owned cancel/delete paging to confirmation, rendering, and final outcome handling in `cmd/cancel_processinstance.go` and `cmd/delete_processinstance.go`
- [ ] T044 [US2] Update ownership and before/after notes in `specs/254-cli-debt-refactor/assessment.md`
- [ ] T045 [US2] Run `go test ./cmd ./c8volt/job ./c8volt/element ./c8volt/incident ./c8volt/process -count=1` and record the result in `specs/254-cli-debt-refactor/quickstart.md`

**Checkpoint**: User Story 2 is independently functional and command-owned paging debt is reduced without behavior drift.

---

## Phase 5: User Story 3 - Faster High-Volume Operations Without Safety Loss (Priority: P3)

**Goal**: High-volume searches, discovery, enrichment, planning, and mutations preserve or improve throughput while respecting safety and operator controls.

**Independent Test**: Fake-latency, benchmark-style, or targeted smoke validation proves changed high-volume workflows are not slower without an accepted tradeoff.

### Tests for User Story 3

- [ ] T046 [P] [US3] Add fake-latency tests for process-instance search enrichment and incident detail lookup in `c8volt/process/client_test.go`
- [ ] T047 [P] [US3] Add fake-latency tests for cancel/delete planning and dependency traversal in `internal/services/processinstance/dryrun_test.go`
- [ ] T048 [P] [US3] Add high-volume ops repair and purge tests for bounded worker behavior in `internal/services/ops/repair_test.go` and `internal/services/ops/incident_purge_test.go`
- [ ] T049 [P] [US3] Add high-volume slow-process analysis and retention policy tests for serial hot paths in `internal/services/ops/slow_process_analysis_test.go` and `internal/services/ops/retention_policy_test.go`
- [ ] T050 [P] [US3] Add worker-control pass-through tests for `--workers`, `--fail-fast`, and `--no-worker-limit` in `cmd/cancel_test.go`, `cmd/delete_test.go`, and `cmd/ops_repair_incident_test.go`

### Implementation for User Story 3

- [ ] T051 [US3] Add high-volume performance characterization results for process-instance search, enrichment, cancel/delete planning, repair, purge, retention, slow-process analysis, job search, element search, and incident search to `specs/254-cli-debt-refactor/assessment.md`
- [ ] T052 [US3] Use bounded concurrency for independent process-instance enrichment or incident-detail lookup where safe in `internal/services/processinstance/enrichment.go`
- [ ] T053 [US3] Use bounded concurrency for independent cancel/delete dependency planning where safe in `internal/services/processinstance/dryrun.go`
- [ ] T054 [US3] Use bounded concurrency for independent ops repair planning or execution where safe in `internal/services/ops/repair.go`
- [ ] T055 [US3] Use bounded concurrency for independent purge, retention, and slow-process analysis steps where safe in `internal/services/ops/incident_purge.go`, `internal/services/ops/retention_policy.go`, and `internal/services/ops/slow_process_analysis.go`
- [ ] T056 [US3] Preserve deterministic result ordering and worker-limit controls in `toolx/pool/pool.go` and affected service call sites under `internal/services/`
- [ ] T057 [US3] Record any accepted performance tradeoffs or intentionally retained serial paths in `specs/254-cli-debt-refactor/assessment.md`
- [ ] T058 [US3] Run `go test ./cmd ./internal/services/ops ./c8volt/process -run 'Test.*(Latency|Concurrent|Performance|HighVolume|Workers|Cancel|Delete|Repair|Purge|Retention|Slow)' -count=1` and record the result in `specs/254-cli-debt-refactor/quickstart.md`

**Checkpoint**: User Story 3 is independently functional and high-volume workflows are characterized with preserved safety controls.

---

## Phase 6: User Story 4 - Help and Documentation Match Behavior (Priority: P4)

**Goal**: Help text, generated CLI docs, README/docs examples, and `capabilities --json` describe actual `--batch-size`, `--limit`, progress, automation, and output-contract behavior.

**Independent Test**: Generated docs, command contract tests, and capability output match the changed command behavior for every affected command.

### Tests for User Story 4

- [ ] T059 [P] [US4] Add command contract assertions for updated `--batch-size`, `--limit`, output modes, automation support, and aliases in `cmd/command_contract_test.go`
- [ ] T060 [P] [US4] Add capabilities document assertions for affected commands in `cmd/capabilities_test.go`
- [ ] T061 [P] [US4] Add generated docs expectations for changed command help and examples in `docsgen/main_test.go`
- [ ] T062 [P] [US4] Add README/docs example expectations for user-facing workflow text in `docsgen/main_test.go`

### Implementation for User Story 4

- [ ] T063 [US4] Update basic command help text and examples for paging, limits, and output modes in `cmd/get_job.go`, `cmd/get_element.go`, `cmd/get_incident.go`, and `cmd/get_processinstance.go`
- [ ] T064 [US4] Update mutation command help text and examples for paging, limits, workers, fail-fast, and confirmation in `cmd/cancel_processinstance.go` and `cmd/delete_processinstance.go`
- [ ] T065 [US4] Update ops command help text and examples for discovery summaries, limits, and worker behavior in `cmd/ops_analyse_slow_process_instances.go`, `cmd/ops_execute_retention_policy.go`, `cmd/ops_purge_processinstances_with_incidents.go`, `cmd/ops_purge_all_processdefinitions.go`, `cmd/ops_repair_incident.go`, and `cmd/ops_repair_processinstance.go`
- [ ] T066 [US4] Update command capability metadata for changed commands in `cmd/command_contract.go` and affected `cmd/*.go` command registrations
- [ ] T067 [US4] Update README and operator documentation examples for changed behavior in `README.md`, `docs/use-cases.md`, and `docs/camunda-cli.md`
- [ ] T068 [US4] Regenerate generated CLI docs with `make docs-content` and verify generated files under `docs/cli/`
- [ ] T069 [US4] Run `go test ./cmd ./docsgen -count=1` and record the result in `specs/254-cli-debt-refactor/quickstart.md`

**Checkpoint**: User Story 4 is independently functional and documentation surfaces match command behavior.

---

## Final Phase: Polish & Cross-Cutting Concerns

**Purpose**: Validate the complete feature, keep artifacts synchronized, and prepare for handoff.

- [ ] T070 [P] Run `gofmt` on touched Go files in `cmd/`, `c8volt/`, `internal/services/`, `internal/domain/`, and `toolx/`, then record the result in `specs/254-cli-debt-refactor/quickstart.md`
- [ ] T071 [P] Verify `specs/254-cli-debt-refactor/assessment.md` covers SC-001 through SC-007 and update any missing evidence in `specs/254-cli-debt-refactor/assessment.md`
- [ ] T072 Run focused validation with `go test ./cmd ./c8volt/job ./c8volt/element ./c8volt/incident ./c8volt/process ./internal/services/ops -count=1` and record the result in `specs/254-cli-debt-refactor/quickstart.md`
- [ ] T073 Run generated docs validation with `make docs-content` and `go test ./docsgen -count=1`, then record the result in `specs/254-cli-debt-refactor/quickstart.md`
- [ ] T074 Run full repository validation with `make test` and record the result in `specs/254-cli-debt-refactor/quickstart.md`
- [ ] T075 Review `git diff --check` and final changed files, then record handoff notes in `specs/254-cli-debt-refactor/assessment.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup and blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational and is the MVP.
- **User Story 2 (Phase 4)**: Depends on Foundational; recommended after US1 so output policy is stable before paging ownership changes.
- **User Story 3 (Phase 5)**: Depends on Foundational; recommended after US2 so performance work measures the new ownership boundaries.
- **User Story 4 (Phase 6)**: Depends on the selected behavior changes from US1-US3.
- **Polish**: Depends on all desired user stories.

### User Story Dependencies

- **US1 Consistent Output**: Can start after Foundational; no dependency on US2-US4.
- **US2 Ownership Refactor**: Can start after Foundational; should use US1 progress policy if US1 is in scope.
- **US3 Performance**: Can start after Foundational but should run after the relevant US2 refactor slice for accurate before/after validation.
- **US4 Help and Docs**: Should run after changed behavior and command metadata are stable.

### Within Each User Story

- Write or update tests first and confirm they fail for the current missing or inconsistent behavior.
- Move service/facade mechanics before simplifying command ownership.
- Preserve command rendering, prompts, and confirmation behavior before changing help or docs.
- Run targeted validation at each checkpoint before moving to the next story.

## Parallel Opportunities

- T002, T003, T004, and T005 can run in parallel after T001.
- T010 and T011 can run in parallel after T006-T009 establish the assessment artifact.
- US1 tests T013-T017 can run in parallel because they target separate files.
- US2 tests T026-T030 can run in parallel because they target separate facade/service/command areas.
- US3 tests T046-T050 can run in parallel across process, ops, and command packages.
- US4 tests T059-T062 can run in parallel across command contract, capabilities, docsgen, and README/docs expectations.
- Final checks T070 and T071 can run in parallel before broader validation.

## Parallel Example: User Story 1

```text
Task: "T013 [P] [US1] Add paged search verbose progress tests for job and element output in cmd/get_job_test.go and cmd/get_element_test.go"
Task: "T014 [P] [US1] Add paged search verbose progress tests for incident and process-instance output in cmd/get_incident_test.go and cmd/get_processinstance_test.go"
Task: "T015 [P] [US1] Add machine-output cleanliness tests for JSON, keys-only, quiet, automation, and no-indicator modes in cmd/cmd_json_assertions_test.go"
Task: "T017 [P] [US1] Add shared activity indicator enablement tests in toolx/logging/activity_test.go and cmd/root_test.go"
```

## Parallel Example: User Story 2

```text
Task: "T026 [P] [US2] Add service/facade page-collection tests for job search in c8volt/job/client_test.go and internal/services/job/v88/service_test.go"
Task: "T027 [P] [US2] Add service/facade page-collection tests for element search in c8volt/element/client_test.go and internal/services/element/v88/service_test.go"
Task: "T028 [P] [US2] Add service/facade page-collection tests for incident search in c8volt/incident/client_test.go and internal/services/incident/v88/incidents_test.go"
Task: "T029 [P] [US2] Add service/facade page-collection and local-filtering tests for process-instance search in c8volt/process/client_test.go and internal/services/processinstance/v88/service_test.go"
```

## Parallel Example: User Story 3

```text
Task: "T046 [P] [US3] Add fake-latency tests for process-instance search enrichment and incident detail lookup in c8volt/process/client_test.go"
Task: "T047 [P] [US3] Add fake-latency tests for cancel/delete planning and dependency traversal in internal/services/processinstance/dryrun_test.go"
Task: "T048 [P] [US3] Add high-volume ops repair and purge tests in internal/services/ops/repair_test.go and internal/services/ops/incident_purge_test.go"
Task: "T049 [P] [US3] Add high-volume slow-process analysis and retention policy tests in internal/services/ops/slow_process_analysis_test.go and internal/services/ops/retention_policy_test.go"
```

## Parallel Example: User Story 4

```text
Task: "T059 [P] [US4] Add command contract assertions in cmd/command_contract_test.go"
Task: "T060 [P] [US4] Add capabilities document assertions in cmd/capabilities_test.go"
Task: "T061 [P] [US4] Add generated docs expectations in docsgen/main_test.go"
Task: "T062 [P] [US4] Add README/docs example expectations in docsgen/main_test.go"
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 setup.
2. Complete Phase 2 assessment and validation scaffold.
3. Complete Phase 3 output/progress policy and tests.
4. Stop and validate US1 independently before paging ownership or performance refactors.

### Incremental Delivery

1. Deliver the assessment baseline.
2. Deliver US1 output/progress consistency as MVP.
3. Deliver US2 ownership refactor one command family at a time.
4. Deliver US3 high-volume performance improvements only where measured and safe.
5. Deliver US4 docs/capability updates after behavior stabilizes.
6. Finish with quickstart and full repository validation.

### Ralph Iteration Discipline

- Each implementation iteration should complete one task or one tightly related task group.
- Before each iteration, read `specs/ralph-implementation-rules.md` plus this feature's `spec.md`, `plan.md`, `tasks.md`, and `assessment.md` if it exists.
- Mark tasks complete only after implementation and relevant validation pass.
- Commit subjects must follow Conventional Commits and end with `#254`.

## Notes

- Do not hand-edit generated Camunda clients under `internal/clients/camunda`.
- Do not create a universal generic pager; extract only mechanics with identical ownership and behavior.
- Keep JSON, keys-only, quiet, and automation output free from progress and prompt noise.
- Keep command help and generated docs synchronized through source metadata and `make docs-content`.
