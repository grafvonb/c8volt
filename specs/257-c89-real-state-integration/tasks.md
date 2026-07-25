# Tasks: C89 Real-State Semantic Integration Coverage

**Input**: Design documents from `/specs/257-c89-real-state-integration/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/), [quickstart.md](./quickstart.md)

**Tests**: Integration tests are required by the feature specification. Story phases include test tasks first and each story remains independently runnable against a Camunda 8.9 profile from the default local c8volt configuration.

**Organization**: Tasks are grouped by user story to support independently runnable real-state slices.

## Phase 1: Setup

**Purpose**: Add the real-state lane without changing baseline or volume target behavior.

- [ ] T001 Add `IT_REAL_STATE_TIMEOUT` and real-state Make target placeholders in `Makefile`
- [ ] T002 [P] Add real-state suite overview, destructive warning, and target list in `integration/README.md`
- [ ] T003 [P] Add real-state target catalog scaffolding in `integration/cli/real_state_harness_test.go`
- [ ] T004 [P] Add real-state evidence structs and report writer scaffolding in `integration/cli/real_state_data_test.go`
- [ ] T005 Add target-catalog validation for all planned real-state targets in `integration/cli/real_state_harness_test.go`
- [ ] T006 Update runnable validation command examples in `specs/257-c89-real-state-integration/quickstart.md`

---

## Phase 2: Foundational

**Purpose**: Shared helpers that every real-state story depends on.

**Critical**: No user-story implementation should start before this phase is complete.

- [ ] T007 Implement Camunda 8.9 profile selection and skip/classification helpers in `integration/cli/real_state_harness_test.go`
- [ ] T008 Implement real-state family, data, progress, ops, proposal report writers, and reusable JSON/keys-only stdout cleanliness assertions in `integration/cli/real_state_data_test.go`
- [ ] T009 Implement suite-owned marker, resource-key, and dirty-cluster containment helpers in `integration/cli/real_state_data_test.go`
- [ ] T010 Implement reusable before-state and after-state command query helpers in `integration/cli/real_state_data_test.go`
- [ ] T011 Implement embedded fixture deployment and process-instance start wrappers that reuse existing helpers in `integration/cli/real_state_data_test.go`
- [ ] T012 Implement proposal fallback helpers for real-state command and embedded BPMN gaps in `integration/cli/real_state_proposals_test.go`
- [ ] T013 Add compile-only and helper validation checks for the real-state scaffolding in `integration/cli/real_state_harness_test.go`

**Checkpoint**: Real-state harness is available, targets are listed, evidence files can be written, and all later stories can create scoped evidence.

---

## Phase 3: User Story 1 - Prove Real Job And Incident State (Priority: P1)

**Goal**: Prove non-empty real job rows, supported job mutations, and incidents with related job evidence against Camunda 8.9.

**Independent Test**: Run `make integration-cli-real-state-jobs IT_GO_TEST_FLAGS=-v` and `make integration-cli-real-state-incidents IT_GO_TEST_FLAGS=-v` against a clean or dirty disposable Camunda 8.9 profile.

### Tests For User Story 1

- [ ] T014 [P] [US1] Add failing `TestRealStateJobsFamily` skeleton covering target contract and evidence names in `integration/cli/real_state_jobs_test.go`
- [ ] T015 [P] [US1] Add failing `TestRealStateIncidentsFamily` skeleton covering related-job evidence expectations in `integration/cli/real_state_incidents_test.go`

### Implementation For User Story 1

- [ ] T016 [US1] Wire `integration-cli-real-state-jobs` and `integration-cli-real-state-incidents` to `TestRealStateJobsFamily` and `TestRealStateIncidentsFamily` in `Makefile`
- [ ] T017 [US1] Implement active job fixture setup through c8volt embedded deploy/run commands first in `integration/cli/real_state_jobs_test.go`
- [ ] T018 [US1] Add non-empty `get job` JSON, JSON stdout cleanliness, and human-output assertions scoped to suite-owned keys in `integration/cli/real_state_jobs_test.go`
- [ ] T019 [US1] Add `update job` dry-run, retries, timeout, fail, no-wait, and JSON stdout cleanliness scenarios with before-state and after-state evidence in `integration/cli/real_state_jobs_test.go`
- [ ] T020 [US1] Implement incident fixture setup and active incident discovery scoped to suite-owned process instances in `integration/cli/real_state_incidents_test.go`
- [ ] T021 [US1] Add related-job evidence checks for incident-driven repair and retry paths in `integration/cli/real_state_incidents_test.go`
- [ ] T022 [US1] Record command proposals for job or incident states that require direct Camunda setup in `integration/cli/real_state_proposals_test.go`
- [ ] T023 [US1] Record embedded BPMN proposals for missing job or incident fixture behavior in `integration/cli/real_state_proposals_test.go`
- [ ] T024 [US1] Update job and incident rows to current statuses in `specs/257-c89-real-state-integration/coverage-matrix.md`
- [ ] T025 [US1] Validate User Story 1 with `make integration-cli-real-state-jobs IT_GO_TEST_FLAGS=-v` and `make integration-cli-real-state-incidents IT_GO_TEST_FLAGS=-v` documented in `specs/257-c89-real-state-integration/quickstart.md`

**Checkpoint**: User Story 1 proves live Camunda 8.9 job and incident state, or records explicit proposal-backed blockers.

---

## Phase 4: User Story 2 - Prove Listener And BPMN Error Workflows (Priority: P2)

**Goal**: Prove listener-related flags and BPMN error job behavior with real state, or record precise fixture/product proposals.

**Independent Test**: Run `make integration-cli-real-state-listeners IT_GO_TEST_FLAGS=-v` and `make integration-cli-real-state-bpmn-error IT_GO_TEST_FLAGS=-v` against a Camunda 8.9 profile.

### Tests For User Story 2

- [ ] T026 [P] [US2] Add failing `TestRealStateListenersFamily` skeleton for listener state and proposal fallback in `integration/cli/real_state_listeners_test.go`
- [ ] T027 [P] [US2] Add failing `TestRealStateBPMNErrorFamily` skeleton for BPMN error job state and proposal fallback in `integration/cli/real_state_bpmn_error_test.go`

### Implementation For User Story 2

- [ ] T028 [US2] Wire `integration-cli-real-state-listeners` and `integration-cli-real-state-bpmn-error` targets in `Makefile`
- [ ] T029 [US2] Implement listener-capable embedded fixture discovery or missing-fixture classification in `integration/cli/real_state_listeners_test.go`
- [ ] T030 [US2] Add non-empty listener evidence plus JSON/keys-only stdout cleanliness where supported for `walk process-instance --with-listeners`, `get element --with-listeners`, and `ops analyse slow-process-instances --with-listeners` in `integration/cli/real_state_listeners_test.go`
- [ ] T031 [US2] Implement BPMN error-capable job fixture discovery or missing-fixture classification in `integration/cli/real_state_bpmn_error_test.go`
- [ ] T032 [US2] Add `update job --throw-bpmn-error` execution, JSON stdout cleanliness, and process-state verification in `integration/cli/real_state_bpmn_error_test.go`
- [ ] T033 [US2] Record listener and BPMN error command setup proposals when c8volt commands cannot create the required state in `integration/cli/real_state_proposals_test.go`
- [ ] T034 [US2] Record listener and BPMN error embedded BPMN proposals without modifying existing embedded models in `integration/cli/real_state_proposals_test.go`
- [ ] T035 [US2] Update listener and BPMN error rows to current statuses in `specs/257-c89-real-state-integration/coverage-matrix.md`
- [ ] T036 [US2] Validate User Story 2 with `make integration-cli-real-state-listeners IT_GO_TEST_FLAGS=-v` and `make integration-cli-real-state-bpmn-error IT_GO_TEST_FLAGS=-v` documented in `specs/257-c89-real-state-integration/quickstart.md`

**Checkpoint**: User Story 2 either proves listener and BPMN error behavior with real state or produces actionable proposal evidence.

---

## Phase 5: User Story 3 - Prove Destructive Ops Semantics On Real Candidates (Priority: P3)

**Goal**: Prove real dry-run safety, confirmed destructive post-state, ops report parity, retention candidates, and mixed-target fail-fast or partial-failure behavior.

**Independent Test**: Run `make integration-cli-real-state-retention IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m` and `make integration-cli-real-state-destructive IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m` against a disposable Camunda 8.9 profile.

### Tests For User Story 3

- [ ] T037 [P] [US3] Add failing `TestRealStateRetentionFamily` skeleton for completed candidate discovery and post-state evidence in `integration/cli/real_state_retention_test.go`
- [ ] T038 [P] [US3] Add failing `TestRealStateDestructiveFamily` skeleton for purge, delete, cancel, resolve, repair, and mixed-target behavior in `integration/cli/real_state_destructive_test.go`

### Implementation For User Story 3

- [ ] T039 [US3] Wire `integration-cli-real-state-retention` and `integration-cli-real-state-destructive` targets in `Makefile`
- [ ] T040 [US3] Implement deterministic completed process-instance candidate setup for retention scenarios in `integration/cli/real_state_retention_test.go`
- [ ] T041 [US3] Add `ops execute retention-policy` dry-run and confirmed execution assertions with retained, deleted, or cleanup-failed evidence in `integration/cli/real_state_retention_test.go`
- [ ] T042 [US3] Implement real purge candidate setup for incidents, process instances, and process definitions in `integration/cli/real_state_destructive_test.go`
- [ ] T043 [US3] Add dry-run non-mutation and confirmed post-state assertions for purge, delete, cancel, expect-resolve, and ops repair commands in `integration/cli/real_state_destructive_test.go`
- [ ] T044 [US3] Implement mixed valid, missing, malformed, stale, and already-mutated target sets in `integration/cli/real_state_destructive_test.go`
- [ ] T045 [US3] Add fail-fast, partial-failure accounting, and machine-output cleanliness assertions for command stdout and ops reports in `integration/cli/real_state_destructive_test.go`
- [ ] T046 [US3] Reuse or extend ops report parity checks for real-state reports in `integration/cli/real_state_destructive_test.go`
- [ ] T047 [US3] Record command and embedded BPMN proposals for retention, purge, orphan, repair, or partial-failure states that cannot be created through c8volt in `integration/cli/real_state_proposals_test.go`
- [ ] T048 [US3] Update retention and destructive rows to current statuses in `specs/257-c89-real-state-integration/coverage-matrix.md`
- [ ] T049 [US3] Validate User Story 3 with `make integration-cli-real-state-retention IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m` and `make integration-cli-real-state-destructive IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m` documented in `specs/257-c89-real-state-integration/quickstart.md`

**Checkpoint**: User Story 3 proves destructive real-candidate semantics, dry-run safety, confirmed outcomes, and partial failure reporting.

---

## Phase 6: User Story 4 - Keep Real-State Gaps Visible And Extensible (Priority: P4)

**Goal**: Keep proposal evidence and the coverage matrix accurate as the real-state suite grows.

**Independent Test**: Run `make integration-cli-real-state-proposals IT_GO_TEST_FLAGS=-v` and inspect aggregate proposal evidence plus `coverage-matrix.md`.

### Tests For User Story 4

- [ ] T050 [P] [US4] Add failing `TestRealStateProposalFamily` skeleton for aggregate command and embedded BPMN proposal evidence in `integration/cli/real_state_proposals_test.go`
- [ ] T051 [P] [US4] Add failing coverage-matrix status validation for every priority topic in `integration/cli/real_state_proposals_test.go`

### Implementation For User Story 4

- [ ] T052 [US4] Wire `integration-cli-real-state-proposals` target to `TestRealStateProposalFamily` in `Makefile`
- [ ] T053 [US4] Implement aggregate proposal evidence checks for baseline, volume, ops repair, and real-state gaps in `integration/cli/real_state_proposals_test.go`
- [ ] T054 [US4] Implement coverage-matrix status checks for live-covered, partially live-covered, no-match only, proposal-backed, and not-yet-started rows in `integration/cli/real_state_proposals_test.go`
- [ ] T055 [US4] Update proposal affected-version handling for Camunda 8.9 focus and future minor extension in `integration/cli/real_state_proposals_test.go`
- [ ] T056 [US4] Update real-state proposal and matrix validation instructions in `specs/257-c89-real-state-integration/quickstart.md`
- [ ] T057 [US4] Validate User Story 4 with `make integration-cli-real-state-proposals IT_GO_TEST_FLAGS=-v` documented in `specs/257-c89-real-state-integration/quickstart.md`

**Checkpoint**: User Story 4 makes every remaining real-state gap visible and keeps future Camunda minor extension points explicit.

---

## Phase 7: Polish And Cross-Cutting

**Purpose**: Final consistency, documentation, and validation across the feature.

- [ ] T058 [P] Update real-state target descriptions, destructive warnings, and command help/example danger-audit notes in `integration/README.md`
- [ ] T059 [P] Update implementation validation notes in `specs/257-c89-real-state-integration/quickstart.md`
- [ ] T060 [P] Update final statuses and first follow-up notes in `specs/257-c89-real-state-integration/coverage-matrix.md`
- [ ] T061 Run `gofmt -w integration/cli/real_state_harness_test.go integration/cli/real_state_data_test.go integration/cli/real_state_jobs_test.go integration/cli/real_state_incidents_test.go integration/cli/real_state_listeners_test.go integration/cli/real_state_bpmn_error_test.go integration/cli/real_state_retention_test.go integration/cli/real_state_destructive_test.go integration/cli/real_state_proposals_test.go`
- [ ] T062 Run `GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1` for `integration/cli`
- [ ] T063 Run `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m` for `integration/cli`
- [ ] T064 Run all real-state Make targets from `Makefile` against the selected Camunda 8.9 profile
- [ ] T065 Run `make test` for repository-wide validation from `Makefile`
- [ ] T066 Verify whether command help/example metadata changed; if it changed, run `make docs-content` and include generated docs, otherwise record that generated CLI docs were not required in `specs/257-c89-real-state-integration/quickstart.md`
- [ ] T067 Run `git diff --check -- Makefile integration/README.md integration/cli specs/257-c89-real-state-integration` before final review

---

## Dependencies And Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup completion and blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion and is the MVP.
- **User Story 2 (Phase 4)**: Depends on Foundational completion; can run after or beside US1 when shared helpers are stable.
- **User Story 3 (Phase 5)**: Depends on Foundational completion; benefits from US1 fixture learnings but remains independently testable.
- **User Story 4 (Phase 6)**: Depends on Foundational completion; should be revisited after each story adds proposal or matrix changes.
- **Polish (Phase 7)**: Depends on all selected user stories for the current implementation slice.

### User Story Dependencies

- **US1**: Start first for MVP because it proves real jobs and incidents.
- **US2**: Can start after foundational helpers; listener and BPMN fixtures may produce proposals before live coverage.
- **US3**: Can start after foundational helpers; destructive scenarios should use fixture patterns learned in US1 where useful.
- **US4**: Can start early for proposal target wiring, then repeat after US1, US2, and US3 update gap evidence.

### Parallel Opportunities

- T002, T003, and T004 can run in parallel.
- T014 and T015 can run in parallel after foundational helpers exist.
- T026 and T027 can run in parallel after foundational helpers exist.
- T037 and T038 can run in parallel after foundational helpers exist.
- T050 and T051 can run in parallel after foundational helpers exist.
- Documentation and matrix polish tasks T058, T059, and T060 can run in parallel.

---

## Parallel Example: User Story 1

```text
Task: "T014 Add failing TestRealStateJobsFamily skeleton in integration/cli/real_state_jobs_test.go"
Task: "T015 Add failing TestRealStateIncidentsFamily skeleton in integration/cli/real_state_incidents_test.go"
```

## Parallel Example: User Story 2

```text
Task: "T026 Add failing TestRealStateListenersFamily skeleton in integration/cli/real_state_listeners_test.go"
Task: "T027 Add failing TestRealStateBPMNErrorFamily skeleton in integration/cli/real_state_bpmn_error_test.go"
```

## Parallel Example: User Story 3

```text
Task: "T037 Add failing TestRealStateRetentionFamily skeleton in integration/cli/real_state_retention_test.go"
Task: "T038 Add failing TestRealStateDestructiveFamily skeleton in integration/cli/real_state_destructive_test.go"
```

## Parallel Example: User Story 4

```text
Task: "T050 Add failing TestRealStateProposalFamily skeleton in integration/cli/real_state_proposals_test.go"
Task: "T051 Add failing coverage-matrix status validation in integration/cli/real_state_proposals_test.go"
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete Phase 3 for User Story 1.
3. Stop and validate `integration-cli-real-state-jobs` and `integration-cli-real-state-incidents`.
4. Commit the MVP before starting listener, BPMN error, retention, or broader destructive paths.

### Incremental Delivery

1. Add real-state target infrastructure.
2. Add real jobs and incidents.
3. Add listener and BPMN error coverage or proposal evidence.
4. Add destructive real-candidate coverage.
5. Keep proposal aggregation and matrix status current after every slice.

### Validation Discipline

- Run local compile and non-integration guards before destructive Camunda runs.
- Run each real-state target independently.
- Keep evidence outside `docs/`.
- Prefer c8volt commands and embedded BPMN fixtures for setup.
- Record proposal evidence whenever direct Camunda API setup or missing embedded BPMN behavior is required.
