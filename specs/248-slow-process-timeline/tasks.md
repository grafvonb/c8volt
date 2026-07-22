# Tasks: Slow Process Timeline Readability

**Input**: Design documents from `/specs/248-slow-process-timeline/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli.md, quickstart.md, specs/ralph-implementation-rules.md

**Tests**: Required by FR-022, the quickstart validation guide, and the repository constitution. Each user story includes tests before implementation tasks.

**Organization**: Tasks are grouped by user story so the default hotspot summary, full-timeline mode, and machine-output stability can be implemented and validated independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with other marked tasks in the same phase because it touches different files or has no dependency on incomplete tasks.
- **[Story]**: User story label for story phases only.
- Every task includes exact repository-relative file paths.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm feature context and current implementation boundaries before changing behavior.

- [ ] T001 Review feature artifacts and record any conflicts in `specs/248-slow-process-timeline/spec.md`, `specs/248-slow-process-timeline/plan.md`, `specs/248-slow-process-timeline/contracts/cli.md`, and `specs/ralph-implementation-rules.md`
- [ ] T002 [P] Inspect the existing slow-process command flags, examples, aliases, and validation in `cmd/ops_analyse_slow_process_instances.go` and `cmd/ops_analyse_slow_process_instances_test.go`
- [ ] T003 [P] Inspect the existing slow-process human, JSON, and keys-only renderers in `cmd/cmd_views_ops_slow_process_analysis.go` and `cmd/cmd_views_ops_slow_process_analysis_test.go`
- [ ] T004 [P] Inspect existing command metadata and docs expectations in `cmd/command_contract_test.go`, `cmd/ops_contract_test.go`, `docsgen/main_test.go`, and `README.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Prepare shared rendering concepts and prevent service/domain churn before user story work.

**CRITICAL**: No user story implementation should begin until this phase is complete.

- [ ] T005 Confirm the existing service payload remains complete before rendering by reviewing `internal/services/ops/slow_process_analysis.go`, `internal/domain/ops_slow_process_analysis.go`, and `c8volt/ops/model.go`
- [ ] T006 Add command-renderer helper scaffolding for hotspot summary row selection without changing output behavior in `cmd/cmd_views_ops_slow_process_analysis.go`
- [ ] T007 Add neutral renderer fixture builders for slow-process summary/full-timeline tests in `cmd/cmd_views_ops_slow_process_analysis_test.go`

**Checkpoint**: Foundation ready - user story implementation can begin without modifying service analysis semantics.

---

## Phase 3: User Story 1 - Read A Hotspot-Oriented Human Summary (Priority: P1) MVP

**Goal**: Default human output shows the process-instance root followed by a compact `slowest elements:` summary with completed contributors at or above 1%, active rows, incident rows, and hidden-row guidance.

**Independent Test**: Analyze a fixture result with many timeline rows and verify default human output shows the root row, selected hotspot rows, active/incident rows, and hidden summary while hiding sub-1% completed rows and noisy transition rows.

### Tests for User Story 1

- [ ] T008 [US1] Add renderer tests for default `slowest elements:` output, 1% completed element contributor inclusion, sub-1% completed element row omission, omitted-row count/kind wording, and hidden-row summary in `cmd/cmd_views_ops_slow_process_analysis_test.go`
- [ ] T009 [US1] Add renderer tests for active-row inclusion, incident-row inclusion, duplicate visibility prevention, and no misleading hidden count for empty timelines in `cmd/cmd_views_ops_slow_process_analysis_test.go`

### Implementation for User Story 1

- [ ] T010 [US1] Implement hotspot summary row selection for completed element rows at or above 1%, active element rows, incident-bearing element rows, and duplicate prevention in `cmd/cmd_views_ops_slow_process_analysis.go`
- [ ] T011 [US1] Implement default human `slowest elements:` rendering and hidden-row summary text in `cmd/cmd_views_ops_slow_process_analysis.go`
- [ ] T012 [US1] Adjust default summary row formatting to omit element instance keys except incident identity when needed in `cmd/cmd_views_ops_slow_process_analysis.go`
- [ ] T013 [US1] Run targeted renderer validation for default summary behavior with `go test ./cmd -run 'TestRenderOpsSlowProcessAnalysisResultHuman' -count=1` covering `cmd/cmd_views_ops_slow_process_analysis_test.go`

**Checkpoint**: User Story 1 delivers the MVP default operator summary and is independently testable.

---

## Phase 4: User Story 2 - Inspect The Complete Timeline On Demand (Priority: P2)

**Goal**: Operators can pass `--with-full-timeline` to restore the complete chronological human timeline, including zero-duration and sub-1% rows, without hidden-row summary text.

**Independent Test**: Run the same fixture result with and without `--with-full-timeline` and verify the flag restores existing chronological detail while the default output remains compact.

### Tests for User Story 2

- [ ] T014 [P] [US2] Add command flag tests for `--with-full-timeline` registration, aliases, and request/render parsing in `cmd/ops_analyse_slow_process_instances_test.go`
- [ ] T015 [P] [US2] Add renderer tests for full-timeline human output preserving existing `elements:` rows, element instance keys, zero-duration rows, transitions, existing detail/root filter behavior, no synthetic transitions, and no hidden summary in `cmd/cmd_views_ops_slow_process_analysis_test.go`
- [ ] T016 [P] [US2] Add command contract tests for `--with-full-timeline` help, examples, and metadata in `cmd/command_contract_test.go`

### Implementation for User Story 2

- [ ] T017 [US2] Add `--with-full-timeline` flag state, flag registration, help text, and examples in `cmd/ops_analyse_slow_process_instances.go`
- [ ] T018 [US2] Implement human renderer dispatch between default hotspot summary and full chronological timeline in `cmd/cmd_views_ops_slow_process_analysis.go`
- [ ] T019 [US2] Preserve the existing full-timeline row style by reusing chronological element and transition row formatting in `cmd/cmd_views_ops_slow_process_analysis.go`
- [ ] T020 [US2] Run targeted command and renderer validation for full-timeline behavior with `go test ./cmd -run 'TestOpsAnalyseSlowProcessInstances|TestRenderOpsSlowProcessAnalysisResultHuman|TestCommandContractOpsAnalyseSlowProcessInstances' -count=1` covering `cmd/ops_analyse_slow_process_instances_test.go`, `cmd/cmd_views_ops_slow_process_analysis_test.go`, and `cmd/command_contract_test.go`

**Checkpoint**: User Stories 1 and 2 both work independently: compact default output and explicit full-timeline output.

---

## Phase 5: User Story 3 - Preserve Script And Machine Output Contracts (Priority: P3)

**Goal**: JSON and keys-only output remain unchanged, and `--with-full-timeline` has no effect on machine-readable or pipeline-safe output modes.

**Independent Test**: Compare JSON and keys-only output with and without `--with-full-timeline` for equivalent fixture results and verify no summary-only text or fields appear.

### Tests for User Story 3

- [ ] T021 [US3] Add JSON stability tests for output with and without `--with-full-timeline`, including no hidden-row or summary-only fields, in `cmd/cmd_views_ops_slow_process_analysis_test.go`
- [ ] T022 [US3] Add keys-only stability tests for output with and without `--with-full-timeline`, including one key per line and no summary text, in `cmd/cmd_views_ops_slow_process_analysis_test.go`
- [ ] T023 [P] [US3] Add command validation tests proving `--with-full-timeline` is accepted with `--json` and `--keys-only` without changing request selection semantics in `cmd/ops_analyse_slow_process_instances_test.go`

### Implementation for User Story 3

- [ ] T024 [US3] Ensure `renderOpsSlowProcessAnalysisResult` checks JSON and keys-only modes before any human full-timeline branching in `cmd/cmd_views_ops_slow_process_analysis.go`
- [ ] T025 [US3] Remove or avoid any human-only full-timeline or hidden-row fields in `c8volt/ops/model.go`, `c8volt/ops/convert.go`, and `internal/domain/ops_slow_process_analysis.go`
- [ ] T026 [US3] Run targeted machine-output validation with `go test ./cmd -run 'TestRenderOpsSlowProcessAnalysisResult.*JSON|TestRenderOpsSlowProcessAnalysisResult.*KeysOnly|TestOpsAnalyseSlowProcessInstances' -count=1` covering `cmd/cmd_views_ops_slow_process_analysis_test.go` and `cmd/ops_analyse_slow_process_instances_test.go`

**Checkpoint**: All user stories are independently functional and machine output contracts remain stable.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, generated artifacts, formatting, and full validation across all user stories.

- [ ] T027 [P] Update README examples and behavior notes for compact default summaries and `--with-full-timeline` in `README.md`
- [ ] T028 [P] Update quickstart wording if implementation output differs from the planned examples in `specs/248-slow-process-timeline/quickstart.md`
- [ ] T029 Update docs generator expectations for the new flag and examples in `docsgen/main_test.go`
- [ ] T030 Run `gofmt` on `cmd/ops_analyse_slow_process_instances.go`, `cmd/cmd_views_ops_slow_process_analysis.go`, `cmd/ops_analyse_slow_process_instances_test.go`, `cmd/cmd_views_ops_slow_process_analysis_test.go`, `cmd/command_contract_test.go`, and `docsgen/main_test.go`
- [ ] T031 Run targeted command validation with `go test ./cmd -run 'TestRenderOpsSlowProcessAnalysisResultHuman|TestOpsAnalyseSlowProcessInstances|TestCommandContractOpsAnalyseSlowProcessInstances|TestOpsContract' -count=1` covering `cmd/`
- [ ] T032 Run docs validation and regenerate generated CLI docs with `go test ./docsgen -count=1` and `make docs-content` covering `docsgen/`, `docs/cli/`, and `docs/index.md`
- [ ] T033 Build the quickstart binary with `go build -o /tmp/c8volt-slow-timeline .` from `go.mod`
- [ ] T034 Verify feasible manual scenarios from `specs/248-slow-process-timeline/quickstart.md` against `/tmp/c8volt-slow-timeline`
- [ ] T035 Run full repository validation with `make test` from `Makefile`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 Setup**: No dependencies.
- **Phase 2 Foundational**: Depends on Phase 1 and blocks all user stories.
- **Phase 3 US1**: Depends on Phase 2 and delivers the MVP.
- **Phase 4 US2**: Depends on Phase 2; recommended after US1 to reduce same-file renderer conflicts.
- **Phase 5 US3**: Depends on US1 and US2 behavior so machine-output stability can be checked against the final human branching.
- **Phase 6 Polish**: Depends on all desired user stories.

### User Story Dependencies

- **US1 (P1)**: Can start after Foundational; no dependency on US2 or US3.
- **US2 (P2)**: Can start after Foundational, but shares renderer files with US1.
- **US3 (P3)**: Should follow US1 and US2 so stability tests cover final output-mode branching.

### Within Each User Story

- Write tests before implementation tasks and verify they fail for missing behavior.
- Command flag tests before command flag implementation.
- Renderer tests before renderer implementation.
- Story-specific targeted validation before moving to the next story.

## Parallel Opportunities

- Setup inspections T002, T003, and T004 can run in parallel because they read separate areas.
- US2 tests T014, T015, and T016 can be written in parallel because they touch different test files.
- US3 command validation test T023 can run in parallel with renderer test drafting once the desired fixture shape is agreed.
- Polish documentation tasks T027 and T028 can run in parallel after output wording stabilizes.

## Parallel Example: User Story 2

```text
Task: "T014 [US2] Add command flag tests in cmd/ops_analyse_slow_process_instances_test.go"
Task: "T015 [US2] Add full-timeline renderer tests in cmd/cmd_views_ops_slow_process_analysis_test.go"
Task: "T016 [US2] Add command contract tests in cmd/command_contract_test.go"
```

## Parallel Example: Polish

```text
Task: "T027 Update README examples and behavior notes in README.md"
Task: "T028 Update quickstart wording in specs/248-slow-process-timeline/quickstart.md"
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Stop and validate default hotspot summary output independently.

### Incremental Delivery

1. Add User Story 1 to improve default human scanability.
2. Add User Story 2 to restore complete chronological human detail on demand.
3. Add User Story 3 to prove JSON and keys-only output remain stable.
4. Finish Polish with docs regeneration and full validation.

### Ralph Iteration Strategy

- Run each work unit with `--implementation-context specs/ralph-implementation-rules.md`.
- Prefer one work unit per phase or per user story checkpoint.
- Commit subjects must use Conventional Commits and end with `#248`.

## Notes

- Keep service and domain models unchanged unless implementation proves a narrow need.
- Do not hand-edit generated CLI docs under `docs/cli/`; regenerate them with `make docs-content`.
- Keys-only output must remain one key per line and nothing else.
- JSON output must not gain hidden-row or summary-only fields.
