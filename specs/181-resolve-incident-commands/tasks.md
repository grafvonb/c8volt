# Tasks: Resolve Incident Commands

**Input**: Design documents from `/specs/181-resolve-incident-commands/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/
**Tests**: Required by repository constitution and feature risk.
**Commit Rule**: Any commit subject for this feature must use Conventional Commits and end with `#181`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files and has no dependency on incomplete tasks.
- **[Story]**: Maps work to the user story from spec.md.
- Every task includes exact repository paths.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm generated client surfaces and command patterns before implementation.

- [x] T001 Inspect generated incident resolution methods in `internal/clients/camunda/v88/camunda/client.gen.go` and `internal/clients/camunda/v89/camunda/client.gen.go`
- [x] T002 Inspect current mutation command patterns in `cmd/update_job.go`, `cmd/cancel_processinstance.go`, and `cmd/update_processinstance.go`
- [x] T003 Inspect existing incident lookup tests in `internal/services/incident/v87/incidents.go`, `internal/services/incident/v88/incidents.go`, `internal/services/incident/v89/incidents.go`, `c8volt/process/client_test.go`, and `cmd/get_processinstance_test.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared service and facade foundations needed by every resolve command.

**CRITICAL**: No user story command work should begin until this phase is complete.

- [x] T004 Add incident resolution domain result fields or reuse existing domain models in `internal/domain/processinstance.go`
- [x] T005 Extend the incident service API with incident resolution and state lookup methods in `internal/services/incident/api.go`
- [x] T006 [P] Add v8.7 unsupported resolution tests in `internal/services/incident/v87/incidents_test.go`
- [x] T007 [P] Add v8.8 resolution service tests in `internal/services/incident/v88/incidents_test.go`
- [x] T008 [P] Add v8.9 resolution service tests in `internal/services/incident/v89/incidents_test.go`
- [x] T009 Implement unsupported v8.7 incident resolution behavior in `internal/services/incident/v87/incidents.go` and `internal/services/incident/v87/contract.go`
- [x] T010 Implement v8.8 incident resolution calls in `internal/services/incident/v88/incidents.go` and `internal/services/incident/v88/contract.go`
- [x] T011 Implement v8.9 incident resolution calls in `internal/services/incident/v89/incidents.go` and `internal/services/incident/v89/contract.go`
- [x] T012 Add incident service factory/API compile checks and version tests in `internal/services/incident/factory_test.go`
- [x] T013 Add process facade resolution plan/result models, dry-run status, mutation-submission status, and totals helpers in `c8volt/process/model.go`
- [x] T014 Extend the process facade API with incident and process-instance resolution methods in `c8volt/process/api.go`
- [x] T015 Implement versioned-service lookup polling for post-mutation confirmation, following the `update pi --vars` waiter pattern, in `internal/services/incident`
- [x] T016 Implement facade resolution orchestration and wait result mapping in `c8volt/process/client.go` and `c8volt/process/bulk.go`
- [x] T017 Add facade tests for direct incident resolution, process-instance discovery, dry-run plans, lookup-polling confirmation, partial failures, `--no-wait`, and unsupported errors in `c8volt/process/client_test.go`

**Checkpoint**: Incident service and facade can resolve incidents without CLI command wiring.

---

## Phase 3: User Story 1 - Resolve Known Incidents (Priority: P1)

**Goal**: Resolve explicit incident keys through `c8volt resolve incident` and `c8volt resolve inc`.

**Independent Test**: Running `c8volt resolve incident --key <incident-key>` resolves one known incident and reports a per-target result; repeated flags and stdin deduplicate keys.

### Tests for User Story 1

- [x] T018 [P] [US1] Add command tests for incident key flags, stdin `-`, duplicate keys, and invalid keys in `cmd/resolve_incident_test.go`
- [x] T019 [P] [US1] Add human and JSON view tests for incident resolution results and non-dry-run payloads in `cmd/cmd_views_resolve_test.go`
- [x] T020 [P] [US1] Add command contract expectations for `resolve incident`, alias `inc`, and mutation metadata in `cmd/command_contract_test.go`

### Implementation for User Story 1

- [x] T021 [US1] Add the `resolve` root command with shared backoff and state-changing metadata in `cmd/resolve.go`
- [x] T022 [US1] Add `resolve incident` command parsing, aliases, flags, stdin key merge, validation, automation support, and worker flags in `cmd/resolve_incident.go`
- [x] T023 [US1] Add incident resolution human and JSON rendering in `cmd/cmd_views_resolve.go`
- [x] T024 [US1] Wire `resolve incident` to the process facade and preserve per-target failures in `cmd/resolve_incident.go`

**Checkpoint**: User Story 1 is fully functional and independently testable.

---

## Phase 4: User Story 2 - Resolve Process Instance Incidents (Priority: P2)

**Goal**: Resolve incidents discovered for selected process instance keys through `c8volt resolve process-instance` and `c8volt resolve pi`.

**Independent Test**: Running `c8volt resolve pi --key <process-instance-key>` discovers active incidents at command start, resolves only that set, and reports a no-op when no incidents are active.

### Tests for User Story 2

- [x] T025 [P] [US2] Add command tests for process-instance key flags, stdin `-`, duplicate keys, no active incidents, and lookup failures in `cmd/resolve_processinstance_test.go`
- [x] T026 [P] [US2] Add facade tests proving process-instance resolution uses incident lookup and never adds incident methods to `internal/services/processinstance` in `c8volt/process/client_test.go`
- [x] T027 [P] [US2] Add command contract expectations for `resolve process-instance` and alias `pi` in `cmd/command_contract_test.go`

### Implementation for User Story 2

- [x] T028 [US2] Add `resolve process-instance` command parsing, aliases, flags, stdin key merge, validation, automation support, and worker flags in `cmd/resolve_processinstance.go`
- [x] T029 [US2] Implement process-instance resolution command orchestration in `cmd/resolve_processinstance.go`
- [x] T030 [US2] Complete process-instance resolution result rendering for no-op, success, partial failure, and JSON output in `cmd/cmd_views_resolve.go`

**Checkpoint**: User Story 2 is functional without changing existing get or update process-instance behavior.

---

## Phase 5: User Story 3 - Preview Resolution Plan (Priority: P3)

**Goal**: Support `--dry-run` for both resolve commands with lookup-backed plans and no mutation.

**Independent Test**: `resolve incident --dry-run` and `resolve pi --dry-run` render planned incident resolution targets, submit no mutation, and `--json --dry-run` returns a stable plan payload.

### Tests for User Story 3

- [x] T031 [P] [US3] Add command dry-run tests for direct incident keys, stdin keys, duplicate keys, and no submitted mutation in `cmd/resolve_incident_test.go`
- [x] T032 [P] [US3] Add command dry-run tests for process-instance incident discovery, no active incidents, and no submitted mutation in `cmd/resolve_processinstance_test.go`
- [x] T033 [P] [US3] Add JSON dry-run and `--json --verbose` rejection tests in `cmd/resolve_incident_test.go`, `cmd/resolve_processinstance_test.go`, and `cmd/cmd_views_resolve_test.go`
- [x] T034 [P] [US3] Add command contract expectations proving `resolve incident` and `resolve process-instance` expose `--dry-run` in `cmd/command_contract_test.go`

### Implementation for User Story 3

- [x] T035 [US3] Add `--dry-run` parsing and request propagation in `cmd/resolve_incident.go` and `cmd/resolve_processinstance.go`
- [x] T036 [US3] Implement lookup-backed dry-run plan construction in `c8volt/process/client.go` and `c8volt/process/bulk.go`
- [x] T037 [US3] Implement compact human dry-run rendering and stable JSON dry-run payloads in `cmd/cmd_views_resolve.go`
- [x] T038 [US3] Reject `--json --verbose` for resolve commands before lookup or mutation in `cmd/resolve_incident.go` and `cmd/resolve_processinstance.go`

**Checkpoint**: Dry-run plans are independently complete for both resolve command paths.

---

## Phase 6: User Story 4 - Control Waiting and Failure Reporting (Priority: P4)

**Goal**: Make default waiting, `--no-wait`, timeout, retry exhaustion, and partial failures explicit.

**Independent Test**: Default commands wait for confirmation, `--no-wait` returns submitted results, one failed target does not hide other successes, and confirmation failures are reported per target.

### Tests for User Story 4

- [x] T039 [P] [US4] Add command tests for `--no-wait`, fail-fast, workers, no-worker-limit, timeout, and retry exhaustion in `cmd/resolve_incident_test.go` and `cmd/resolve_processinstance_test.go`
- [x] T040 [P] [US4] Add facade bulk tests for worker fan-out, fail-fast, no-worker-limit, and partial failure totals in `c8volt/process/client_test.go`

### Implementation for User Story 4

- [x] T041 [US4] Thread existing backoff, timeout, retry, worker, fail-fast, no-worker-limit, and no-wait options through resolve commands in `cmd/resolve_incident.go` and `cmd/resolve_processinstance.go`
- [x] T042 [US4] Ensure facade result totals and command exit behavior surface partial failures without suppressing successful target output in `c8volt/process/model.go`, `c8volt/process/client.go`, and `cmd/cmd_views_resolve.go`

**Checkpoint**: Waiting and failure behavior is explicit across incident and process-instance resolution.

---

## Phase 7: User Story 5 - Preserve Existing Workflows (Priority: P5)

**Goal**: Keep command contracts, docs, generated docs, and existing process-instance workflows stable.

**Independent Test**: Existing process-instance get/update tests still pass, capability output includes the new command family, and generated CLI docs match behavior.

### Tests for User Story 5

- [x] T043 [P] [US5] Add regression tests that `get process-instance --with-incidents`, `get pi --with-incidents`, and `update pi --vars` remain unchanged in `cmd/get_processinstance_test.go` and `cmd/update_processinstance_test.go`
- [x] T044 [P] [US5] Update docs generation tests for resolve command docs in `docsgen/main_test.go`
- [x] T045 [P] [US5] Update capabilities tests for resolve metadata in `cmd/capabilities_test.go`

### Implementation for User Story 5

- [x] T046 [US5] Update README examples and command overview for resolve workflows, including dry-run examples, in `README.md`
- [x] T047 [US5] Regenerate CLI reference markdown with `make docs-content`, updating `docs/cli/c8volt_resolve.md`, `docs/cli/c8volt_resolve_incident.md`, `docs/cli/c8volt_resolve_process-instance.md`, and `docs/cli/index.md`
- [x] T048 [US5] Verify no incident lookup or resolution methods were added to `internal/services/processinstance/factory.go`, `internal/services/processinstance/v87/contract.go`, `internal/services/processinstance/v88/contract.go`, or `internal/services/processinstance/v89/contract.go`

**Checkpoint**: User-facing documentation and existing workflows remain consistent.

---

## Final Phase: Validation & Handoff

**Purpose**: Prove the complete feature before commit or PR handoff.

- [x] T049 Run targeted service validation with `GOCACHE=/tmp/c8volt-gocache go test ./internal/services/incident/... -count=1`
- [x] T050 Run targeted facade validation with `GOCACHE=/tmp/c8volt-gocache go test ./c8volt/process -count=1`
- [x] T051 Run targeted command validation with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestResolve|TestCommandContract|TestCapabilities|TestGetProcessInstance|TestUpdatePI' -count=1`
- [x] T052 Run docs validation with `GOCACHE=/tmp/c8volt-gocache go test ./docsgen -count=1`
- [x] T053 Run repository validation with `make test`
- [x] T054 Review `git diff --check` and final changed files before committing

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup and blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational.
- **User Story 2 (Phase 4)**: Depends on Foundational and may reuse US1 rendering helpers.
- **User Story 3 (Phase 5)**: Depends on US1 and US2 command paths.
- **User Story 4 (Phase 6)**: Depends on US1, US2, and US3 command paths.
- **User Story 5 (Phase 7)**: Depends on command behavior being stable enough for docs and capability output.
- **Validation**: Depends on all desired user stories.

### User Story Dependencies

- **US1 Resolve Known Incidents**: MVP after Foundational.
- **US2 Resolve Process Instance Incidents**: Depends on incident lookup and resolution foundations; can be implemented after or alongside US1 once shared rendering is stable.
- **US3 Preview Resolution Plan**: Depends on both command paths and their lookup behavior.
- **US4 Control Waiting and Failure Reporting**: Depends on both command paths and dry-run not bypassing safety behavior.
- **US5 Preserve Existing Workflows**: Final compatibility and documentation pass.

### Parallel Opportunities

- T006, T007, and T008 can run in parallel after T005.
- T017, T018, and T019 can run in parallel.
- T024, T025, and T026 can run in parallel.
- T031, T032, T033, and T034 can run in parallel.
- T039 and T040 can run in parallel.
- T043, T044, and T045 can run in parallel.
- Final validations T049 through T052 can run in parallel before T053.

## Parallel Example: User Story 1

```text
Task: "Add command tests for incident key flags, stdin `-`, duplicate keys, and invalid keys in cmd/resolve_incident_test.go"
Task: "Add human and JSON view tests for incident resolution results in cmd/cmd_views_resolve_test.go"
Task: "Add command contract expectations for resolve incident and alias inc in cmd/command_contract_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational phases.
2. Complete User Story 1.
3. Validate direct incident resolution with targeted service, facade, and command tests.

### Incremental Delivery

1. Add direct incident resolution.
2. Add process-instance incident discovery and resolution.
3. Add dry-run planning and stable JSON plans.
4. Harden waiting, partial failure, and automation behavior.
5. Update docs and verify existing workflows.

### Ralph Iteration Guidance

Each Ralph iteration should select the next unchecked task or a tightly coupled pair from the same phase. Avoid mixing service foundation work with documentation or broad validation in the same iteration unless all prior story tasks are complete.
