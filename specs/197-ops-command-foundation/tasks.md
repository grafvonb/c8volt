# Tasks: Ops Command Foundation

**Input**: Design documents from `/specs/197-ops-command-foundation/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/
**Tests**: Required by repository constitution and feature risk.
**Commit Rule**: Any commit subject for this feature must use Conventional Commits and end with `#197`.
**Mandatory Implementation Context**: Ralph runs MUST include `--implementation-context specs/ralph-implementation-rules.md`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files and has no dependency on incomplete tasks.
- **[Story]**: Maps work to the user story from spec.md.
- Every task includes exact repository paths.

## Phase 1: Setup (Shared Discovery)

**Purpose**: Confirm command, contract, and documentation patterns before implementation.

- [x] T001 Inspect grouping command patterns in `cmd/get.go`, `cmd/resolve.go`, and `cmd/run.go`
- [x] T002 Inspect command metadata helpers in `cmd/command_contract.go` and existing expectations in `cmd/capabilities_test.go`
- [x] T003 Inspect top-level help and generated markdown tests in `cmd/root_test.go` and `docsgen/main_test.go`
- [x] T004 Record any reusable implementation discoveries in `specs/197-ops-command-foundation/progress.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add the minimal shared command foundation that all ops grouping stories use.

**CRITICAL**: No user story work should begin until this phase is complete.

- [x] T005 Add `ops` root grouping command registration, help text, examples, aliases if warranted, and mutation metadata in `cmd/ops.go`
- [x] T006 [P] Add base ops command help tests in `cmd/ops_test.go`
- [x] T007 [P] Add ops root discovery metadata assertions in `cmd/capabilities_test.go`
- [x] T008 Update root help discovery expectations for `ops` in `cmd/root_test.go`

**Checkpoint**: The top-level ops command exists as a grouping command and is visible in help/discovery.

---

## Phase 3: User Story 1 - Discover Ops Command Family (Priority: P1) MVP

**Goal**: `c8volt ops --help` describes high-level operational workflows without requiring Camunda configuration.

**Independent Test**: Run `c8volt ops --help` without runtime configuration and verify help output succeeds without executing workflow behavior.

### Tests for User Story 1

- [x] T009 [P] [US1] Add/extend tests proving `c8volt ops --help` succeeds without runtime config in `cmd/ops_test.go`
- [x] T010 [P] [US1] Add/extend tests proving `ops` appears in root help while existing top-level commands remain discoverable in `cmd/root_test.go`

### Implementation for User Story 1

- [x] T011 [US1] Finalize `ops` help copy and grouping behavior in `cmd/ops.go`
- [x] T012 [US1] Run targeted validation with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*Ops|TestRootHelp' -count=1`

**Checkpoint**: User Story 1 is independently functional and verified.

---

## Phase 4: User Story 2 - Discover Execute Grouping Command (Priority: P2)

**Goal**: `c8volt ops execute --help` describes future predefined playbooks and performs no workflow behavior.

**Independent Test**: Run `c8volt ops execute --help` and verify no orphan cleanup, retention policy, or smoke test workflow is present.

### Tests for User Story 2

- [ ] T013 [P] [US2] Add execute grouping help tests in `cmd/ops_test.go`
- [ ] T014 [P] [US2] Add capabilities assertions for `ops execute` in `cmd/capabilities_test.go`

### Implementation for User Story 2

- [ ] T015 [US2] Add `ops execute` grouping command registration, help text, examples, and metadata in `cmd/ops_execute.go`
- [ ] T016 [US2] Ensure no concrete execute playbook commands are registered in `cmd/ops_execute.go`
- [ ] T017 [US2] Run targeted validation with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*Ops|TestCapability.*Ops' -count=1`

**Checkpoint**: User Story 2 is independently functional and verified.

---

## Phase 5: User Story 3 - Discover Repair Grouping Command (Priority: P3)

**Goal**: `c8volt ops repair --help` describes repair/remediation workflows and exposes no ambiguous top-level `--key`.

**Independent Test**: Run `c8volt ops repair --help` and verify repair help succeeds, performs no remediation, and does not show a top-level `--key`.

### Tests for User Story 3

- [ ] T018 [P] [US3] Add repair grouping help tests, including no top-level `--key`, in `cmd/ops_test.go`
- [ ] T019 [P] [US3] Add capabilities assertions for `ops repair` in `cmd/capabilities_test.go`

### Implementation for User Story 3

- [ ] T020 [US3] Add `ops repair` grouping command registration, help text, examples, and metadata in `cmd/ops_repair.go`
- [ ] T021 [US3] Ensure `cmd/ops_repair.go` defines no ambiguous top-level `--key` flag
- [ ] T022 [US3] Run targeted validation with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*Ops|TestCapability.*Ops' -count=1`

**Checkpoint**: User Story 3 is independently functional and verified.

---

## Phase 6: User Story 4 - Establish Shared Ops Workflow Contracts (Priority: P4)

**Goal**: Future ops workflows have lightweight shared conventions for workflow metadata, dry-run/report behavior, automation output, and step statuses.

**Independent Test**: Inspect shared contract definitions and verify they do not call resource services, generated clients, or concrete workflow behavior.

### Tests for User Story 4

- [ ] T023 [P] [US4] Add tests for shared ops step status values and report-format behavior in `cmd/ops_contract_test.go`
- [ ] T024 [P] [US4] Add command contract tests proving grouping commands do not claim full automation support in `cmd/capabilities_test.go`

### Implementation for User Story 4

- [ ] T025 [US4] Add lightweight shared ops workflow/report contract types and comments in `cmd/ops_contract.go`
- [ ] T026 [US4] Keep resource-specific API logic out of `cmd/ops_contract.go` and record that boundary in `specs/197-ops-command-foundation/progress.md`
- [ ] T027 [US4] Run targeted validation with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*Ops|TestCapability.*Ops' -count=1`

**Checkpoint**: User Story 4 is independently functional and verified.

---

## Phase 7: User Story 5 - Regenerate User-Facing Command Documentation (Priority: P5)

**Goal**: Generated CLI documentation includes the ops command family from source command definitions.

**Independent Test**: Run docs generation and verify generated docs contain ops pages without manual generated-doc edits.

### Tests for User Story 5

- [ ] T028 [P] [US5] Add or update docs generator expectations for ops command pages in `docsgen/main_test.go`
- [ ] T029 [P] [US5] Add README-facing command overview updates if needed in `README.md`

### Implementation for User Story 5

- [ ] T030 [US5] Regenerate CLI docs with `make docs-content`, updating generated files under `docs/cli/` and `docs/index.md`
- [ ] T031 [US5] Run docs validation with `GOCACHE=/tmp/c8volt-gocache go test ./docsgen -count=1`
- [ ] T032 [US5] Run command validation with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*Ops|TestCapability.*Ops|TestRootHelp' -count=1`

**Checkpoint**: User Story 5 is independently functional and verified.

---

## Final Phase: Validation & Handoff

**Purpose**: Prove the complete feature before commit or PR handoff.

- [ ] T033 Run full command package validation with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -count=1`
- [ ] T034 Run docs generator validation with `GOCACHE=/tmp/c8volt-gocache go test ./docsgen -count=1`
- [ ] T035 Run generated docs refresh check with `make docs-content`
- [ ] T036 Run repository validation with `make test`
- [ ] T037 Review `git diff --check` and final changed files before committing

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup and blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational.
- **User Story 2 (Phase 4)**: Depends on Foundational and can be implemented after US1 or in parallel once root command exists.
- **User Story 3 (Phase 5)**: Depends on Foundational and can be implemented after US1 or in parallel once root command exists.
- **User Story 4 (Phase 6)**: Depends on Foundational; should not add concrete workflow behavior.
- **User Story 5 (Phase 7)**: Depends on completed command source changes.
- **Validation**: Depends on all desired user stories.

### User Story Dependencies

- **US1 Discover Ops Command Family**: MVP after Foundational.
- **US2 Discover Execute Grouping Command**: Needs `ops` root command from Foundational.
- **US3 Discover Repair Grouping Command**: Needs `ops` root command from Foundational.
- **US4 Establish Shared Ops Workflow Contracts**: Can proceed after command foundation exists.
- **US5 Regenerate User-Facing Command Documentation**: Final documentation pass after source command behavior is stable.

### Parallel Opportunities

- T006, T007, and T008 can run in parallel after T005.
- T009 and T010 can run in parallel.
- T013 and T014 can run in parallel.
- T018 and T019 can run in parallel.
- T023 and T024 can run in parallel.
- T028 and T029 can run in parallel.
- T033 and T034 can run in parallel before full `make test`.

---

## Parallel Example: User Story 2

```text
Task: "Add execute grouping help tests in cmd/ops_test.go"
Task: "Add capabilities assertions for ops execute in cmd/capabilities_test.go"
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1 setup and Phase 2 foundational command registration.
2. Complete User Story 1 and validate `c8volt ops --help`.
3. Stop and verify root command discovery before adding child grouping commands.

### Incremental Delivery

1. Add `ops` root grouping command.
2. Add `ops execute` grouping command.
3. Add `ops repair` grouping command.
4. Add lightweight shared ops workflow contracts.
5. Regenerate docs and run validation.

### Ralph Iteration Discipline

- Each user story phase is small enough for one Ralph iteration.
- Before each iteration, read `specs/ralph-implementation-rules.md`.
- Mark tasks complete only after implementation and relevant validation pass.
- Include task/progress updates in the same work-unit commit.
