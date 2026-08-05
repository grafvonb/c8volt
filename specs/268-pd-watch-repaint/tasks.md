# Tasks: Process Definition Watch Repaint

**Input**: Design documents from `specs/268-pd-watch-repaint/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli-watch-repaint-contract.md, quickstart.md
**Tests**: Required by the c8volt constitution for user-visible CLI behavior changes.
**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the current command surface and known failing expectations before story work begins.

- [x] T001 Inspect current process-definition watch renderer and loop boundaries in `cmd/get_processdefinition.go`, `cmd/cmd_views_get.go`, and `toolx/watch/watch.go`
- [x] T002 Inspect old watch output/help expectations in `cmd/get_processdefinition_test.go`, `cmd/command_contract_test.go`, `README.md`, and `docs/cli/c8volt_get_process-definition.md`
- [x] T003 [P] Review the repaint contract and validation scenarios in `specs/268-pd-watch-repaint/contracts/cli-watch-repaint-contract.md` and `specs/268-pd-watch-repaint/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish reusable command-level test seams needed by all stories.

**CRITICAL**: No user story work can begin until this phase is complete.

- [x] T004 Add or adjust command test helpers for separate stdout/stderr capture and deterministic watch sleeps in `cmd/get_processdefinition_test.go`
- [x] T005 Add a deterministic way to assert repaint control output without requiring a real terminal in `cmd/get_processdefinition_test.go`

**Checkpoint**: Foundation ready - user story implementation can now begin.

---

## Phase 3: User Story 1 - Watch Repaints One Live View (Priority: P1) MVP

**Goal**: Watch mode repaints one visible result view per refresh, and each refresh body matches normal non-watch human output.

**Independent Test**: Run process-definition watch mode for multiple refreshes and verify that output contains repaint behavior, normal process-definition rows and summary, and no watch-only snapshot labels.

### Tests for User Story 1

- [x] T006 [US1] Update the repeated broad watch test to expect repaint behavior and no `snapshot N:` labels in `cmd/get_processdefinition_test.go`
- [x] T007 [US1] Add a test proving a watched refresh body matches the equivalent non-watch human process-definition output in `cmd/get_processdefinition_test.go`
- [x] T008 [P] [US1] Update command metadata/help contract expectations from appended snapshots to repaint behavior in `cmd/command_contract_test.go`

### Implementation for User Story 1

- [x] T009 [US1] Replace watch-specific result-body rendering with normal process-definition list rendering in `cmd/cmd_views_get.go`
- [x] T010 [US1] Add a terminal repaint helper in `cmd/cmd_views_rendermode.go` and call it before each successful watch refresh in `cmd/get_processdefinition.go`
- [x] T011 [US1] Update `get process-definition` long help, examples, and output-mode metadata to describe repaint behavior in `cmd/get_processdefinition.go`
- [x] T012 [US1] Run `go test ./cmd -run 'TestGetProcessDefinitionWatch|TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata|TestProcessDefinitionSelectorValidationHelpContract' -count=1` and resolve failures in `cmd/get_processdefinition_test.go`, `cmd/command_contract_test.go`, `cmd/get_processdefinition.go`, and `cmd/cmd_views_get.go`

**Checkpoint**: User Story 1 is independently functional and testable as the MVP.

---

## Phase 4: User Story 2 - Slow Refreshes Are Clear Without Noise (Priority: P2)

**Goal**: Watch mode measures each refresh, warns clearly when refresh work exceeds the configured interval, suppresses repeated default warnings during a continuous slow streak, and exposes more timing detail in verbose mode.

**Independent Test**: Simulate slow and on-time refreshes and verify default warnings appear once per slow streak, reset after an on-time refresh, and remain outside the result body.

### Tests for User Story 2

- [x] T013 [US2] Add command test coverage for one default warning per continuous slow-refresh streak in `cmd/get_processdefinition_test.go`
- [x] T014 [US2] Add command test coverage that an on-time refresh resets the slow-warning streak in `cmd/get_processdefinition_test.go`
- [x] T015 [US2] Add command test coverage for verbose per-refresh timing/status outside the result body in `cmd/get_processdefinition_test.go`
- [x] T016 [P] [US2] Add watch runner test coverage that refresh ticks remain serial when a tick takes longer than the interval in `toolx/watch/watch_test.go`

### Implementation for User Story 2

- [x] T017 [US2] Measure collection-plus-render duration for each successful refresh in `cmd/get_processdefinition.go`
- [x] T018 [US2] Implement default slow-refresh warning streak state and reset behavior in `cmd/get_processdefinition.go`
- [x] T019 [US2] Add verbose refresh timing/status output outside the result body in `cmd/get_processdefinition.go`
- [x] T020 [US2] Run `go test ./cmd -run 'TestGetProcessDefinitionWatch' -count=1` and resolve failures in `cmd/get_processdefinition_test.go` and `cmd/get_processdefinition.go`

**Checkpoint**: User Story 2 is independently functional with slow-refresh behavior validated.

---

## Phase 5: User Story 3 - Watch Keeps Human-Only Boundaries (Priority: P3)

**Goal**: Incompatible machine-oriented modes remain rejected with watch, and all non-watch output modes remain unchanged.

**Independent Test**: Combine watch with each incompatible mode and verify local rejection before lookup work, then run representative non-watch mode checks for unchanged behavior.

### Tests for User Story 3

- [x] T021 [US3] Re-run and update incompatible watch mode rejection tests if wording changed in `cmd/get_processdefinition_test.go`
- [x] T022 [US3] Re-run and update non-watch machine mode compatibility tests if metadata wording changed in `cmd/get_processdefinition_test.go`
- [x] T023 [P] [US3] Update command capability output-mode notes expectations if wording changed in `cmd/command_contract_test.go`

### Implementation for User Story 3

- [x] T024 [US3] Preserve or tighten incompatible flag validation without lookup work in `cmd/get_processdefinition.go`
- [x] T025 [US3] Run `go test ./cmd -run 'TestValidateGetProcessDefinitionWatch|TestGetProcessDefinitionWatchRejectsMachineModesBeforeLookup|TestGetProcessDefinitionNonWatchMachineModesStayCompatible|TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata' -count=1` and resolve failures in `cmd/get_processdefinition_test.go`, `cmd/command_contract_test.go`, and `cmd/get_processdefinition.go`

**Checkpoint**: User Story 3 is independently functional with human-only watch boundaries preserved.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, formatting, and full validation.

- [x] T026 [P] Update README watch guidance to describe repaint behavior, normal result-body parity, slow-refresh warnings, and incompatible modes in `README.md`
- [x] T027 Regenerate generated CLI documentation from command metadata into `docs/cli/c8volt_get_process-definition.md`
- [x] T028 Run `gofmt` on touched Go files in `cmd/get_processdefinition.go`, `cmd/cmd_views_get.go`, `cmd/cmd_views_rendermode.go`, `cmd/get_processdefinition_test.go`, `cmd/command_contract_test.go`, `toolx/watch/watch.go`, and `toolx/watch/watch_test.go`
- [x] T029 Run focused validation from `specs/268-pd-watch-repaint/quickstart.md` and record any deviations in `specs/268-pd-watch-repaint/tasks.md`
- [x] T030 Run `make test` for full repository validation and record any failures or skipped checks in `specs/268-pd-watch-repaint/tasks.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Stories (Phase 3+)**: Depend on Foundational completion.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: Starts after Foundational; MVP scope and required before documentation claims repaint behavior complete.
- **User Story 2 (P2)**: Starts after Foundational; can be developed after or alongside US1, but final validation should include US1 result-body parity.
- **User Story 3 (P3)**: Starts after Foundational; can be developed alongside US1/US2 because it protects validation paths and non-watch compatibility.

### Within Each User Story

- Tests should be updated before implementation and should fail against the old behavior where practical.
- Renderer/help changes should be implemented before docs regeneration.
- Story checkpoint validation should pass before moving to the next priority in a sequential implementation.

### Parallel Opportunities

- T003 can run in parallel with T001-T002.
- T008 can run in parallel with T006-T007.
- T016 can run in parallel with T013-T015 if watch runner behavior changes.
- T023 can run in parallel with T021-T022.
- T026 can run in parallel with final Go formatting once command wording is stable.

---

## Parallel Example: User Story 1

```text
Task: "T006 [US1] Update the repeated broad watch test to expect repaint behavior and no snapshot labels in cmd/get_processdefinition_test.go"
Task: "T008 [P] [US1] Update command metadata/help contract expectations from appended snapshots to repaint behavior in cmd/command_contract_test.go"
```

## Parallel Example: User Story 2

```text
Task: "T013 [US2] Add command test coverage for one default warning per continuous slow-refresh streak in cmd/get_processdefinition_test.go"
Task: "T016 [P] [US2] Add or update serial refresh timing expectations if watch runner hooks change in toolx/watch/watch_test.go"
```

## Parallel Example: User Story 3

```text
Task: "T021 [US3] Re-run and update incompatible watch mode rejection tests if wording changed in cmd/get_processdefinition_test.go"
Task: "T023 [P] [US3] Update command capability output-mode notes expectations if wording changed in cmd/command_contract_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 and Phase 2.
2. Complete Phase 3 for repaint and result-body parity.
3. Stop and validate `go test ./cmd -run 'TestGetProcessDefinitionWatch|TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata|TestProcessDefinitionSelectorValidationHelpContract' -count=1`.
4. Demo `get process-definition --watch` in an interactive terminal if a profile is available.

### Incremental Delivery

1. Deliver US1 to remove appended snapshot blocks and align result-body output.
2. Deliver US2 to add slow-refresh measurement and warning streak behavior.
3. Deliver US3 to reassert watch/non-watch output contracts.
4. Complete polish tasks for README, generated docs, formatting, and full validation.

### Parallel Team Strategy

After Phase 2, one developer can work on US1 renderer/repaint behavior, another can work on US2 slow-refresh warning tests/status, and another can verify US3 incompatible-mode boundaries. Coordinate changes in `cmd/get_processdefinition.go` because US1, US2, and US3 all touch that file.

---

## Notes

- [P] tasks use different files or can proceed without depending on incomplete same-file edits.
- [US1], [US2], and [US3] labels map to the user stories in `specs/268-pd-watch-repaint/spec.md`.
- Keep generated Camunda clients under `internal/clients/camunda/` untouched.
- Do not hand-edit generated CLI docs under `docs/cli/` except through docs generation.
- Preserve existing non-watch command output unless a task explicitly says otherwise.
