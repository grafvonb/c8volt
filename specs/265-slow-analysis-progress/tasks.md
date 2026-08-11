# Tasks: Slow Analysis Progress After Confirmation

**Input**: Design documents from `/specs/265-slow-analysis-progress/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/slow-analysis-progress-contract.md, quickstart.md

**Tests**: Required by the feature specification and repository constitution. Write focused tests before implementation tasks where behavior changes.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story the task supports
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm active feature context and nearby implementation surface before changing code.

- [x] T001 Review `specs/265-slow-analysis-progress/spec.md`, `specs/265-slow-analysis-progress/plan.md`, `specs/265-slow-analysis-progress/research.md`, `specs/265-slow-analysis-progress/data-model.md`, `specs/265-slow-analysis-progress/contracts/slow-analysis-progress-contract.md`, and `specs/ralph-implementation-rules.md`
- [x] T002 Inspect existing progress ownership in `cmd/ops_progress.go`, `cmd/ops_analyse_slow_process_instances.go`, `internal/services/ops/slow_process_analysis.go`, `internal/domain/ops_progress.go`, and `toolx/logging/activity.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Define shared command progress pacing tests and behavior before user-story-specific wiring.

**CRITICAL**: No user story implementation should begin until shared progress pacing expectations are captured.

- [x] T003 [P] Add shared milestone pacing state and boundary tests in `cmd/ops_progress_test.go` for no elapsed time, no forward progress, elapsed plus page progress, elapsed plus frozen-scope progress, and timer-only suppression
- [x] T004 [P] Add shared output-mode gating tests in `cmd/ops_progress_test.go` proving default human can use sparse durable milestones while JSON, keys-only, quiet, and automation modes suppress them
- [x] T005 Implement shared milestone pacing state, named elapsed threshold constant, progress signature comparison, and durable milestone decision helpers in `cmd/ops_progress.go`
- [x] T006 Run `go test ./cmd -run 'TestOps.*Progress|TestFormatOps.*Progress|TestOpsETA' -count=1` using validation guidance in `specs/265-slow-analysis-progress/quickstart.md` and record any failures before story work continues

**Checkpoint**: Shared milestone policy is test-covered and available for slow-process analysis.

---

## Phase 3: User Story 1 - See Progress After Confirmation (Priority: P1) MVP

**Goal**: Default human broad slow-process analysis visibly reports progress after confirmation using shared elapsed-plus-progress durable milestone pacing.

**Independent Test**: Simulate a broad slow-process analysis after preflight confirmation and verify default human output gets high-level activity plus sparse durable stderr milestones without touching stdout.

### Tests for User Story 1

- [x] T007 [P] [US1] Add slow-analysis default human post-confirmation milestone tests in `cmd/ops_analyse_slow_process_instances_test.go` covering page progress after elapsed silence and frozen-scope progress after elapsed silence
- [x] T008 [P] [US1] Add slow-analysis workflow activity preservation assertions in `cmd/ops_analyse_slow_process_instances_test.go` proving progress updates use `logging.ActivityImportanceWorkflow` while nested runtime work can occur
- [x] T009 [P] [US1] Add or adjust service progress event tests in `internal/services/ops/slow_process_analysis_test.go` only if implementation changes structured event emission for discovery or frozen-scope progress

### Implementation for User Story 1

- [x] T010 [US1] Wire shared milestone pacing state into `configureOpsSlowProcessAnalysisPreflight` and `printOpsSlowProcessAnalysisProgress` usage in `cmd/ops_analyse_slow_process_instances.go`
- [x] T011 [US1] Ensure default human slow-analysis progress continues updating transient workflow activity while sparse durable milestones print only when shared pacing allows in `cmd/ops_analyse_slow_process_instances.go`
- [x] T012 [US1] Keep internal services emitting structured progress only and remove any accidental command-rendering policy from `internal/services/ops/slow_process_analysis.go`
- [x] T013 [US1] Run `go test ./cmd -run 'TestOpsAnalyseSlowProcessInstances.*Progress|TestOpsAnalyseSlowProcessInstances.*Preflight' -count=1` for `cmd/ops_analyse_slow_process_instances_test.go`
- [x] T014 [US1] Run `go test ./internal/services/ops -run 'TestSlowProcessAnalysis.*Progress|TestSlowProcessAnalysis.*Preflight' -count=1` for `internal/services/ops/slow_process_analysis_test.go` if T009 or T012 touched service progress behavior

**Checkpoint**: User Story 1 is independently functional and satisfies the MVP.

---

## Phase 4: User Story 2 - Preserve Quiet Machine Output (Priority: P2)

**Goal**: JSON, keys-only, quiet, and automation modes remain free of human progress text while slow-analysis milestone behavior exists for default human mode.

**Independent Test**: Run or simulate broad slow-process progress events in every machine-oriented mode and verify stdout and stderr remain clean according to each contract.

### Tests for User Story 2

- [x] T015 [P] [US2] Extend slow-analysis JSON and keys-only progress silence tests in `cmd/ops_analyse_slow_process_instances_test.go` so paced durable milestones cannot leak to stdout or stderr
- [x] T016 [P] [US2] Extend slow-analysis quiet and automation progress silence tests in `cmd/ops_analyse_slow_process_instances_test.go` so paced durable milestones are suppressed
- [x] T017 [P] [US2] Add command contract regression coverage in `cmd/command_contract_test.go` only if command metadata or help text changes

### Implementation for User Story 2

- [x] T018 [US2] Verify `opsProgressChannelForMode` and slow-analysis milestone wiring in `cmd/ops_progress.go` and `cmd/ops_analyse_slow_process_instances.go` keep `StdoutAllowed` false and durable human milestones disabled for JSON, keys-only, quiet, and automation
- [x] T019 [US2] Run `go test ./cmd -run 'TestOpsAnalyseSlowProcessInstances.*JSON|TestOpsAnalyseSlowProcessInstances.*KeysOnly|TestOpsAnalyseSlowProcessInstances.*Quiet|TestOpsAnalyseSlowProcessInstances.*Automation|TestOpsProgressChannelForMode' -count=1` for `cmd/ops_analyse_slow_process_instances_test.go` and `cmd/ops_progress_test.go`

**Checkpoint**: User Stories 1 and 2 both work independently without corrupting machine output.

---

## Phase 5: User Story 3 - Keep Detailed Diagnostics Available (Priority: P3)

**Goal**: Verbose and debug modes keep durable detailed progress, while default human mode remains compact and milestone-paced.

**Independent Test**: Simulate slow-analysis progress in verbose and debug modes and verify detailed durable progress remains visible on stderr without endpoint/cursor chatter in default human milestones.

### Tests for User Story 3

- [x] T020 [P] [US3] Add debug-mode durable progress coverage in `cmd/ops_analyse_slow_process_instances_test.go` matching existing verbose coverage
- [x] T021 [P] [US3] Add compact default-human milestone wording assertions in `cmd/ops_analyse_slow_process_instances_test.go` proving milestones exclude endpoint names, request details, cursors, and per-resource lifecycle detail

### Implementation for User Story 3

- [x] T022 [US3] Preserve `opsSlowProcessAnalysisDurableProgressAllowed` verbose/debug behavior while adding default-human milestone pacing in `cmd/ops_analyse_slow_process_instances.go`
- [x] T023 [US3] Verify shared formatters in `cmd/ops_progress.go` continue producing compact operator-facing discovery and frozen-scope text for default human milestones
- [x] T024 [US3] Run `go test ./cmd -run 'TestOpsAnalyseSlowProcessInstances.*Verbose|TestOpsAnalyseSlowProcessInstances.*Debug|TestFormatOps.*Progress' -count=1` for `cmd/ops_analyse_slow_process_instances_test.go` and `cmd/ops_progress_test.go`

**Checkpoint**: All user stories are independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validate docs, formatting, and repository-wide behavior.

- [x] T025 Run `gofmt -w cmd/ops_progress.go cmd/ops_progress_test.go cmd/ops_analyse_slow_process_instances.go cmd/ops_analyse_slow_process_instances_test.go internal/services/ops/slow_process_analysis.go internal/services/ops/slow_process_analysis_test.go cmd/command_contract_test.go` for touched Go files
- [x] T026 Review `README.md`, `docs/index.md`, `docs/cli/c8volt_ops_analyse_slow-process-instances.md`, and `cmd/ops_analyse_slow_process_instances.go` help text for documentation impact; update source help or README only if shipped wording changes
- [x] T027 Run `make docs-content` via `Makefile` if T026 changes command help or generated CLI documentation inputs
- [x] T028 Run `go test ./cmd -count=1` for the `cmd/` package
- [x] T029 Run `make test` via `Makefile`
- [x] T030 Update `specs/265-slow-analysis-progress/quickstart.md` or implementation progress notes only if validation commands or documentation decisions differ from the planned path

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; starts immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational; MVP.
- **User Story 2 (Phase 4)**: Depends on Foundational and should run after US1 wiring exists so machine-output tests exercise the real milestone path.
- **User Story 3 (Phase 5)**: Depends on Foundational and can run after US1 wiring exists.
- **Polish (Phase 6)**: Depends on all desired user stories.

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational; no dependency on US2 or US3.
- **User Story 2 (P2)**: Depends on shared milestone wiring from US1 to prove machine-output safety against the implemented path.
- **User Story 3 (P3)**: Depends on shared milestone wiring from US1 to prove verbose/debug behavior remains intact.

### Within Each User Story

- Write or adjust tests before implementation.
- Keep shared pacing behavior in `cmd/ops_progress.go`.
- Keep slow-analysis wiring in `cmd/ops_analyse_slow_process_instances.go`.
- Keep services limited to structured progress event emission.
- Validate each story independently before moving to the next priority.

### Parallel Opportunities

- T003 and T004 can run in parallel because both are test additions in the same file but cover separate shared policy concerns; coordinate if editing concurrently.
- T007, T008, and T009 can run in parallel if service progress behavior is touched.
- T015, T016, and T017 can run in parallel if command metadata changes are isolated.
- T020 and T021 can run in parallel after US1 wiring is in place.
- Documentation review T026 can begin while targeted validation runs, but docs generation T027 depends on any help/doc edits.

---

## Parallel Example: User Story 1

```bash
Task: "Add slow-analysis default human post-confirmation milestone tests in cmd/ops_analyse_slow_process_instances_test.go"
Task: "Add slow-analysis workflow activity preservation assertions in cmd/ops_analyse_slow_process_instances_test.go"
Task: "Add or adjust service progress event tests in internal/services/ops/slow_process_analysis_test.go only if service progress behavior changes"
```

---

## Parallel Example: User Story 2

```bash
Task: "Extend slow-analysis JSON and keys-only progress silence tests in cmd/ops_analyse_slow_process_instances_test.go"
Task: "Extend slow-analysis quiet and automation progress silence tests in cmd/ops_analyse_slow_process_instances_test.go"
Task: "Add command contract regression coverage in cmd/command_contract_test.go only if command metadata or help text changes"
```

---

## Parallel Example: User Story 3

```bash
Task: "Add debug-mode durable progress coverage in cmd/ops_analyse_slow_process_instances_test.go"
Task: "Add compact default-human milestone wording assertions in cmd/ops_analyse_slow_process_instances_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 and Phase 2.
2. Complete Phase 3 for default human post-confirmation progress.
3. Stop and validate with T013 and T014 when applicable.
4. Demonstrate that broad slow-analysis progress can emit sparse durable milestones after confirmation without stdout leakage.

### Incremental Delivery

1. Add shared milestone pacing and tests.
2. Add US1 slow-analysis default human milestone wiring and tests.
3. Add US2 machine-output suppression tests against the new path.
4. Add US3 verbose/debug preservation tests and any required adjustments.
5. Finish with docs review, formatting, targeted tests, and full `make test`.

### Notes

- [P] tasks indicate possible parallel work, but same-file edits still require coordination.
- Do not move milestone pacing into internal services.
- Do not add timer-only "still working" output.
- Do not change slow-process analysis selection semantics or result ranking.
- Commit only after validation passes and the active workflow requests a commit.
