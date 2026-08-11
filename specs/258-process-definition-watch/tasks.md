# Tasks: Process Definition Watch Mode

**Input**: Design documents from `specs/258-process-definition-watch/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli-watch-contract.md, quickstart.md

**Tests**: Included because the feature specification requires automated tests for watch cadence, retry behavior, paging, human-only output contracts, incompatible output validation, non-watch compatibility, documentation, and validation.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- Single Go CLI project rooted at repository root
- Command code under `cmd/`
- Public facade code under `c8volt/process/`
- Version-neutral service/domain code under `internal/services/processdefinition/` and `internal/domain/`
- Shared reusable helpers under `toolx/`
- Generated CLI docs under `docs/cli/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm current feature context and protect the existing process-definition command surface before implementation.

- [x] T001 Review `specs/258-process-definition-watch/spec.md`, `specs/258-process-definition-watch/plan.md`, `specs/258-process-definition-watch/contracts/cli-watch-contract.md`, and `specs/ralph-implementation-rules.md` before changing code
- [x] T002 [P] Run baseline targeted command tests with `go test ./cmd -run 'TestGetProcessDefinition|TestCommandContract' -count=1` and record any pre-existing failures in `specs/258-process-definition-watch/quickstart.md`
- [x] T003 [P] Run baseline process-definition service/facade tests with `go test ./internal/services/processdefinition/... ./c8volt/process -run 'ProcessDefinition|SearchProcessDefinitions' -count=1` and record any pre-existing failures in `specs/258-process-definition-watch/quickstart.md`
- [x] T004 [P] Inspect existing process-definition docs in `README.md`, `cmd/get_processdefinition.go`, `cmd/command_contract.go`, `docsgen/main_test.go`, and `docs/cli/c8volt_get_process-definition.md` for current examples and output contract wording

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared watch mechanics and process-definition snapshot data shapes that all watch stories depend on.

**CRITICAL**: No user story work can begin until this phase is complete.

- [x] T005 [P] Add unit tests for immediate first tick, positive interval sleeping, context cancellation, timeout termination, retry reset, and retry exhaustion in `toolx/watch/watch_test.go`
- [x] T006 Implement reusable watch session runner types and fixed-interval loop in `toolx/watch/watch.go`
- [x] T007 [P] Add process-definition watch snapshot/request model tests in `c8volt/process/model_test.go`
- [x] T008 Add public process-definition watch request/result structs in `c8volt/process/model.go`
- [x] T009 Add matching version-neutral process-definition watch request/result structs in `internal/domain/processdefinition.go`
- [x] T010 Wire process-definition watch request/result conversion helpers in `c8volt/process/convert.go`
- [x] T011 [P] Add service tests for complete paged snapshot collection and latest/key dispatch in `internal/services/processdefinition/search_test.go`
- [x] T012 Implement process-definition snapshot collection helper in `internal/services/processdefinition/search.go`
- [x] T013 Add facade tests for process-definition watch snapshot delegation and error conversion in `c8volt/process/client_test.go`
- [x] T014 Expose process-definition watch snapshot facade method in `c8volt/process/api.go` and implement it in `c8volt/process/client.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in priority order or parallel by story.

---

## Phase 3: User Story 1 - Watch Process Definitions Until Visible (Priority: P1) MVP

**Goal**: Operators can run watch mode for process-definition lookup, get an immediate complete snapshot, watch all definitions when no selector is provided, receive empty snapshots for explicit selectors, and stop cleanly on interrupt or timeout.

**Independent Test**: Run `c8volt get process-definition --watch` against controlled fake process-definition responses, verify immediate repeated complete snapshots, broad missing-selector behavior, explicit-selector empty snapshots, and clean interrupt/timeout termination.

### Tests for User Story 1

- [x] T015 [P] [US1] Add command tests for `--watch` flag registration, immediate first snapshot, repeated snapshots, and broad missing-selector all-definition lookup in `cmd/get_processdefinition_test.go`
- [x] T016 [P] [US1] Add command tests for explicit BPMN/latest empty snapshots and changed population between snapshots in `cmd/get_processdefinition_test.go`
- [x] T017 [P] [US1] Add command tests for interrupt and timeout termination without lookup-failure wording in `cmd/get_processdefinition_test.go`
- [x] T018 [P] [US1] Add command contract tests for `--watch` metadata and alias coverage in `cmd/command_contract_test.go`

### Implementation for User Story 1

- [x] T019 [US1] Add `flagGetPDWatch` and register `--watch` on `getProcessDefinitionCmd` in `cmd/get_processdefinition.go`
- [x] T020 [US1] Route `runGetProcessDefinition` into a watch execution path before XML/key/search one-shot branches in `cmd/get_processdefinition.go`
- [x] T021 [US1] Implement watch snapshot collection for broad, BPMN/latest, and key selectors via the process facade in `cmd/get_processdefinition.go`
- [x] T022 [US1] Render default human watch snapshots with compact human-only snapshot boundaries while reusing existing process-definition rows in `cmd/cmd_views_get.go`
- [x] T023 [US1] Preserve existing non-watch selector validation and one-shot output behavior in `cmd/get_processdefinition.go`
- [x] T024 [US1] Update command capability metadata for watch support in `cmd/command_contract.go`
- [x] T025 [US1] Run `go test ./toolx/... ./internal/services/processdefinition/... ./c8volt/process ./cmd -run 'Watch|watch|TestGetProcessDefinition|TestCommandContract' -count=1`

**Checkpoint**: User Story 1 is fully functional and independently testable as the MVP.

---

## Phase 4: User Story 2 - Control Watch Cadence And Retry Tolerance (Priority: P2)

**Goal**: Operators can rely on a default 1-second interval, override it with positive durations, reject invalid intervals before lookup work, and use existing command retry defaults with reset-after-success semantics.

**Independent Test**: Run watch mode with omitted, valid, invalid, zero, and negative intervals plus transient failure sequences; verify cadence, validation, retry continuation, retry reset, and retry exhaustion.

### Tests for User Story 2

- [x] T026 [P] [US2] Add command tests for default `1s` interval and valid `--watch-interval 2s` cadence wiring in `cmd/get_processdefinition_test.go`
- [x] T027 [P] [US2] Add command tests rejecting invalid, zero, and negative `--watch-interval` values before lookup work in `cmd/get_processdefinition_test.go`
- [x] T028 [P] [US2] Add command tests for existing retry default usage, retry budget reset after success, and retry exhaustion exit behavior in `cmd/get_processdefinition_test.go`

### Implementation for User Story 2

- [x] T029 [US2] Add `flagGetPDWatchInterval` and register `--watch-interval` with default `1s` duration parsing in `cmd/get_processdefinition.go`
- [x] T030 [US2] Extend `validateGetProcessDefinitionFlags` to reject invalid, zero, and negative watch intervals before lookup work in `cmd/get_processdefinition.go`
- [x] T031 [US2] Map existing command backoff timeout and max-retry config into `toolx/watch` runner options in `cmd/get_processdefinition.go`
- [x] T032 [US2] Render retry, timeout, and retry-exhaustion status away from result stdout in `cmd/get_processdefinition.go`
- [x] T033 [US2] Run `go test ./toolx/... -run 'Watch|watch' -count=1` and `go test ./cmd -run 'TestGetProcessDefinition' -count=1`

**Checkpoint**: User Stories 1 and 2 work independently and together.

---

## Phase 5: User Story 3 - Preserve Script-Safe Watch Output (Priority: P3)

**Goal**: Automation authors keep simple script-safe contracts because watch mode rejects JSON, keys-only, XML, quiet, and automation combinations while default human and verbose watch output remain useful.

**Independent Test**: Run watch with default human and verbose modes, then verify JSON, keys-only, XML, quiet, and automation combinations fail before lookup work and non-watch machine modes remain unchanged.

### Tests for User Story 3

- [x] T034 [P] [US3] Add JSON, keys-only, XML, quiet, and automation watch rejection tests that prove no lookup work starts in `cmd/get_processdefinition_test.go`
- [x] T035 [P] [US3] Add non-watch JSON, keys-only, quiet, and automation regression tests for unchanged process-definition output contracts in `cmd/get_processdefinition_test.go`
- [x] T036 [P] [US3] Add verbose and default human watch output contract tests in `cmd/get_processdefinition_test.go`
- [x] T037 [P] [US3] Add key/stat compatibility tests for human watch mode in `cmd/get_processdefinition_test.go`
- [x] T038 [P] [US3] Add docsgen or command contract tests for human-only watch output documentation in `docsgen/main_test.go` or `cmd/command_contract_test.go`

### Implementation for User Story 3

- [x] T039 [US3] Implement human-only watch output validation for JSON, keys-only, XML, quiet, and automation combinations in `cmd/get_processdefinition.go`
- [x] T040 [US3] Preserve non-watch JSON, keys-only, quiet, and automation behavior while watch validation is enabled in `cmd/get_processdefinition.go`
- [x] T041 [US3] Gate default human, verbose, debug, and progress/status channels according to `contracts/cli-watch-contract.md` in `cmd/get_processdefinition.go` and `cmd/cmd_views_get.go`
- [x] T042 [US3] Update command long help, examples, output mode metadata, and watch flag descriptions to say watch is human-output only in `cmd/get_processdefinition.go` and `cmd/command_contract.go`
- [x] T043 [US3] Run `go test ./cmd -run 'TestGetProcessDefinition|TestCommandContract' -count=1`

**Checkpoint**: All user stories are independently functional, human-watch-only, and machine-output safe.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, formatting, and full validation required before commit readiness.

- [x] T044 [P] Update README process-definition examples and watch behavior notes in `README.md`
- [x] T045 Regenerate generated CLI docs with `make docs-content` for `docs/cli/c8volt_get_process-definition.md`
- [x] T046 [P] Review generated process-definition docs for watch flags and output contracts in `docs/cli/c8volt_get_process-definition.md`
- [x] T047 Run `gofmt -w toolx/watch cmd/get_processdefinition.go cmd/cmd_views_get.go cmd/command_contract.go c8volt/process internal/domain/processdefinition.go internal/services/processdefinition/search.go`
- [x] T048 Run focused quickstart validation commands from `specs/258-process-definition-watch/quickstart.md`
- [x] T049 Run full repository validation with `make test` using `Makefile`
- [x] T050 Update final validation notes in `specs/258-process-definition-watch/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion - blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion and is the MVP.
- **User Story 2 (Phase 4)**: Depends on Foundational completion; may be developed after US1 or alongside US1 after shared watch runner interfaces stabilize.
- **User Story 3 (Phase 5)**: Depends on Foundational completion and should integrate after the basic US1 watch path exists.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **US1 (P1)**: Required MVP. Establishes command watch execution and complete snapshots.
- **US2 (P2)**: Builds on shared watch runner and command watch path to add interval/retry controls.
- **US3 (P3)**: Builds on command watch rendering to reject incompatible output modes and preserve non-watch machine-output contracts.

### Within Each User Story

- Tests should be written first and fail before implementation.
- Shared models and service/facade snapshot collection precede command integration.
- Command flag validation precedes output rendering.
- Story-specific validation runs before moving to the next story.

### Parallel Opportunities

- T002, T003, and T004 can run in parallel during setup.
- T005, T007, and T011 can run in parallel before foundational implementation.
- T015, T016, T017, and T018 can run in parallel for US1 tests.
- T026, T027, and T028 can run in parallel for US2 tests.
- T034, T035, T036, T037, and T038 can run in parallel for US3 tests.
- T044 and T046 can run in parallel after docs are generated, while T047 can run before validation.

---

## Parallel Example: User Story 1

```bash
Task: "T015 [P] [US1] Add command tests for --watch flag registration, immediate first snapshot, repeated snapshots, and broad missing-selector all-definition lookup in cmd/get_processdefinition_test.go"
Task: "T016 [P] [US1] Add command tests for explicit BPMN/latest empty snapshots and changed population between snapshots in cmd/get_processdefinition_test.go"
Task: "T017 [P] [US1] Add command tests for interrupt and timeout termination without lookup-failure wording in cmd/get_processdefinition_test.go"
Task: "T018 [P] [US1] Add command contract tests for --watch metadata and alias coverage in cmd/command_contract_test.go"
```

## Parallel Example: User Story 2

```bash
Task: "T026 [P] [US2] Add command tests for default 1s interval and valid --watch-interval 2s cadence wiring in cmd/get_processdefinition_test.go"
Task: "T027 [P] [US2] Add command tests rejecting invalid, zero, and negative --watch-interval values before lookup work in cmd/get_processdefinition_test.go"
Task: "T028 [P] [US2] Add command tests for existing retry default usage, retry budget reset after success, and retry exhaustion exit behavior in cmd/get_processdefinition_test.go"
```

## Parallel Example: User Story 3

```bash
Task: "T034 [P] [US3] Add JSON, keys-only, XML, quiet, and automation watch rejection tests that prove no lookup work starts in cmd/get_processdefinition_test.go"
Task: "T035 [P] [US3] Add non-watch JSON, keys-only, quiet, and automation regression tests for unchanged process-definition output contracts in cmd/get_processdefinition_test.go"
Task: "T036 [P] [US3] Add verbose and default human watch output contract tests in cmd/get_processdefinition_test.go"
Task: "T037 [P] [US3] Add key/stat compatibility tests for human watch mode in cmd/get_processdefinition_test.go"
Task: "T038 [P] [US3] Add docsgen or command contract tests for human-only watch output documentation in docsgen/main_test.go or cmd/command_contract_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 setup.
2. Complete Phase 2 foundational watch runner and process-definition snapshot plumbing.
3. Complete Phase 3 User Story 1.
4. Stop and validate US1 with focused `toolx`, service/facade, and command tests.
5. Demo `c8volt get process-definition --watch` against fake or configured Camunda data.

### Incremental Delivery

1. Foundation -> shared runner and process-definition snapshot method.
2. US1 -> immediately useful watch snapshots and broad process-definition observation.
3. US2 -> interval validation and retry/timeout controls.
4. US3 -> human-only watch validation plus non-watch machine-output guarantees.
5. Polish -> docs, generated CLI docs, quickstart notes, and full validation.

### Validation Discipline

- Keep command changes close to `cmd/get_processdefinition.go` and `cmd/cmd_views_get.go`.
- Keep backend mechanics out of `cmd`; use `c8volt/process` and `internal/services/processdefinition`.
- Do not hand-edit generated Camunda clients.
- Run targeted validation after each story and `make test` before commit readiness.
