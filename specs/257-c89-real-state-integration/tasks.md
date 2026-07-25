# Tasks: C89 Real-State Semantic Integration Coverage

**Input**: Design documents from `/specs/257-c89-real-state-integration/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/), [quickstart.md](./quickstart.md)

**Tests**: Integration tests are required by the feature specification. Story phases include test tasks first and each story remains independently runnable against a Camunda 8.9 profile from the default local c8volt configuration.

**Organization**: Tasks are grouped by user story to support independently runnable real-state slices.

## Phase 1: Setup

**Purpose**: Add the real-state lane without changing baseline or volume target behavior.

- [x] T001 Add `IT_REAL_STATE_TIMEOUT` and real-state Make target placeholders in `Makefile`
- [x] T002 [P] Add real-state suite overview, destructive warning, and target list in `integration/README.md`
- [x] T003 [P] Add real-state target catalog scaffolding in `integration/cli/real_state_harness_test.go`
- [x] T004 [P] Add real-state evidence structs and report writer scaffolding in `integration/cli/real_state_data_test.go`
- [x] T005 Add target-catalog validation for all planned real-state targets in `integration/cli/real_state_harness_test.go`
- [x] T006 Update runnable validation command examples in `specs/257-c89-real-state-integration/quickstart.md`

---

## Phase 2: Foundational

**Purpose**: Shared helpers that every real-state story depends on.

**Critical**: No user-story implementation should start before this phase is complete.

- [x] T007 Implement Camunda 8.9 profile selection and skip/classification helpers in `integration/cli/real_state_harness_test.go`
- [x] T008 Implement real-state family, data, progress, ops, legacy proposal report writers, and reusable JSON/keys-only stdout cleanliness assertions in `integration/cli/real_state_data_test.go`
- [x] T009 Implement suite-owned marker, resource-key, and dirty-cluster containment helpers in `integration/cli/real_state_data_test.go`
- [x] T010 Implement reusable before-state and after-state command query helpers in `integration/cli/real_state_data_test.go`
- [x] T011 Implement embedded fixture deployment and process-instance start wrappers that reuse existing helpers in `integration/cli/real_state_data_test.go`
- [x] T012 Implement legacy proposal fallback helpers for real-state command and embedded BPMN gaps in `integration/cli/real_state_proposals_test.go`
- [x] T013 Add compile-only and helper validation checks for the real-state scaffolding in `integration/cli/real_state_harness_test.go`

**Checkpoint**: Real-state harness is available, targets are listed, evidence files can be written, and all later stories can create scoped evidence.

---

## Phase 3: User Story 1 - Prove Real Job And Incident State (Priority: P1)

**Goal**: Prove non-empty real job rows, supported job mutations, and incidents with related job evidence against Camunda 8.9.

**Independent Test**: Run `make integration-cli-real-state-jobs IT_GO_TEST_FLAGS=-v` and `make integration-cli-real-state-incidents IT_GO_TEST_FLAGS=-v` against a clean or dirty disposable Camunda 8.9 profile.

### Tests For User Story 1

- [x] T014 [P] [US1] Add failing `TestRealStateJobsFamily` skeleton covering target contract and evidence names in `integration/cli/real_state_jobs_test.go`
- [x] T015 [P] [US1] Add failing `TestRealStateIncidentsFamily` skeleton covering related-job evidence expectations in `integration/cli/real_state_incidents_test.go`

### Implementation For User Story 1

- [x] T016 [US1] Wire `integration-cli-real-state-jobs` and `integration-cli-real-state-incidents` to `TestRealStateJobsFamily` and `TestRealStateIncidentsFamily` in `Makefile`
- [x] T017 [US1] Implement active job fixture setup through c8volt embedded deploy/run commands first in `integration/cli/real_state_jobs_test.go`
- [x] T018 [US1] Add non-empty `get job` JSON, JSON stdout cleanliness, and human-output assertions scoped to suite-owned keys in `integration/cli/real_state_jobs_test.go`
- [x] T019 [US1] Add `update job` dry-run, retries, timeout, fail, no-wait, and JSON stdout cleanliness scenarios with before-state and after-state evidence in `integration/cli/real_state_jobs_test.go`
- [x] T020 [US1] Implement incident fixture setup and active incident discovery scoped to suite-owned process instances in `integration/cli/real_state_incidents_test.go`
- [x] T021 [US1] Add related-job evidence checks for incident-driven repair and retry paths in `integration/cli/real_state_incidents_test.go`
- [x] T022 [US1] Record legacy command proposal gaps for job or incident states that require direct Camunda setup in `integration/cli/real_state_proposals_test.go`
- [x] T023 [US1] Record legacy embedded BPMN proposal gaps for missing job or incident fixture behavior in `integration/cli/real_state_proposals_test.go`
- [x] T024 [US1] Update job and incident rows to current statuses in `specs/257-c89-real-state-integration/coverage-matrix.md`
- [x] T025 [US1] Validate User Story 1 with `make integration-cli-real-state-jobs IT_GO_TEST_FLAGS=-v` and `make integration-cli-real-state-incidents IT_GO_TEST_FLAGS=-v` documented in `specs/257-c89-real-state-integration/quickstart.md`

**Checkpoint**: User Story 1 proves live Camunda 8.9 job and incident state, with remaining blockers to be migrated from legacy proposal evidence into spec-owned gap tracking.

---

## Phase 4: User Story 2 - Prove Listener And BPMN Error Workflows (Priority: P2)

**Goal**: Prove listener-related flags and BPMN error job behavior with real state, or record precise skipped-prerequisite runtime evidence while maintaining fixture/product gaps in specs.

**Independent Test**: Run `make integration-cli-real-state-listeners IT_GO_TEST_FLAGS=-v` and `make integration-cli-real-state-bpmn-error IT_GO_TEST_FLAGS=-v` against a Camunda 8.9 profile.

### Tests For User Story 2

- [x] T026 [P] [US2] Add failing `TestRealStateListenersFamily` skeleton for listener state and prerequisite classification in `integration/cli/real_state_listeners_test.go`
- [x] T027 [P] [US2] Add failing `TestRealStateBPMNErrorFamily` skeleton for BPMN error job state and prerequisite classification in `integration/cli/real_state_bpmn_error_test.go`

### Implementation For User Story 2

- [x] T028 [US2] Wire `integration-cli-real-state-listeners` and `integration-cli-real-state-bpmn-error` targets in `Makefile`
- [x] T029 [US2] Implement listener-capable embedded fixture discovery or missing-fixture classification in `integration/cli/real_state_listeners_test.go`
- [x] T030 [US2] Add non-empty listener evidence plus JSON/keys-only stdout cleanliness where supported for `walk process-instance --with-listeners`, `get element --with-listeners`, and `ops analyse slow-process-instances --with-listeners` in `integration/cli/real_state_listeners_test.go`
- [x] T031 [US2] Implement BPMN error-capable job fixture discovery or missing-fixture classification in `integration/cli/real_state_bpmn_error_test.go`
- [x] T032 [US2] Add `update job --throw-bpmn-error` execution, JSON stdout cleanliness, and process-state verification in `integration/cli/real_state_bpmn_error_test.go`
- [x] T033 [US2] Record legacy listener and BPMN error command setup gaps when c8volt commands cannot create the required state in `integration/cli/real_state_proposals_test.go`
- [x] T034 [US2] Record legacy listener and BPMN error embedded BPMN gaps without modifying existing embedded models in `integration/cli/real_state_proposals_test.go`
- [x] T035 [US2] Update listener and BPMN error rows to current statuses in `specs/257-c89-real-state-integration/coverage-matrix.md`
- [x] T036 [US2] Validate User Story 2 with `make integration-cli-real-state-listeners IT_GO_TEST_FLAGS=-v` and `make integration-cli-real-state-bpmn-error IT_GO_TEST_FLAGS=-v` documented in `specs/257-c89-real-state-integration/quickstart.md`

**Checkpoint**: User Story 2 either proves listener and BPMN error behavior with real state or records skipped-prerequisite/dry-run runtime evidence with spec-owned gap tracking.

---

## Phase 5: Correct Runtime Evidence And Gap Ownership (Priority: P0)

**Purpose**: Remove backlog/proposal ownership from integration test execution before adding more real-state scenarios.

**Independent Test**: Run non-integration and integration compile checks, then run the existing jobs, incidents, listeners, and BPMN error targets to confirm runtime evidence still reports truth without generating new backlog proposal files.

- [x] T037 [P0] Add repository-level integration test responsibility rules in `specs/integration-test-responsibility.md`
- [x] T038 [P0] Add feature-local real-state gap tracking in `specs/257-c89-real-state-integration/gaps.md`
- [x] T039 [P0] Remove proposal files from 257 runtime evidence contracts and update real-state evidence structs to use skipped-prerequisite and dry-run-covered outcomes in `integration/cli/real_state_data_test.go`
- [x] T040 [P0] Replace real-state proposal helpers with spec-gap validation helpers in `integration/cli/real_state_gap_validation_test.go`
- [x] T041 [P0] Rename the reserved `integration-cli-real-state-proposals` target to `integration-cli-real-state-gaps` in `Makefile`
- [x] T042 [P0] Convert `TestRealStateBPMNErrorFamily` from proposal-backed evidence to dry-run-covered plus skipped-prerequisite confirmed mutation evidence in `integration/cli/real_state_bpmn_error_test.go`
- [x] T043 [P0] Remove proposal writing from real-state jobs, incidents, and listener family tests while preserving runtime setup evidence in `integration/cli/real_state_jobs_test.go`, `integration/cli/real_state_incidents_test.go`, and `integration/cli/real_state_listeners_test.go`
- [x] T044 [P0] Update inherited 255 and 256 spec artifacts to mark runtime proposal JSON as deprecated and point future work to spec-owned gap artifacts
- [x] T045 [P0] Validate correction with `go test ./integration/cli -count=1`, integration compile checks, `make integration-cli-real-state-jobs IT_GO_TEST_FLAGS=-v`, `make integration-cli-real-state-incidents IT_GO_TEST_FLAGS=-v`, `make integration-cli-real-state-listeners IT_GO_TEST_FLAGS=-v`, and `make integration-cli-real-state-bpmn-error IT_GO_TEST_FLAGS=-v`

**Checkpoint**: Integration tests report runtime truth only; missing setup and fixture work lives in specs, not generated test evidence.

---

## Phase 6: User Story 3 - Prove Destructive Ops Semantics On Real Candidates (Priority: P3)

**Goal**: Prove real dry-run safety, confirmed destructive post-state, ops report parity, retention candidates, and mixed-target fail-fast or partial-failure behavior.

**Independent Test**: Run `make integration-cli-real-state-retention IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m` and `make integration-cli-real-state-destructive IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m` against a disposable Camunda 8.9 profile.

### Tests For User Story 3

- [x] T046 [P] [US3] Add failing `TestRealStateRetentionFamily` skeleton for completed candidate discovery and post-state evidence in `integration/cli/real_state_retention_test.go`
- [x] T047 [P] [US3] Add failing `TestRealStateDestructiveFamily` skeleton for purge, delete, cancel, resolve, repair, and mixed-target behavior in `integration/cli/real_state_destructive_test.go`

### Implementation For User Story 3

- [x] T048 [US3] Wire `integration-cli-real-state-retention` and `integration-cli-real-state-destructive` targets in `Makefile`
- [x] T049 [US3] Implement deterministic completed process-instance candidate setup for retention scenarios in `integration/cli/real_state_retention_test.go`
- [x] T050 [US3] Add `ops execute retention-policy` dry-run and confirmed execution assertions with retained, deleted, or cleanup-failed evidence in `integration/cli/real_state_retention_test.go`
- [x] T051 [US3] Implement real purge candidate setup for incident-selected process instances in `integration/cli/real_state_destructive_test.go`; process-definition and orphan purge candidates remain tracked in `gaps.md`
- [x] T052 [US3] Add dry-run non-mutation and confirmed post-state assertions for incident purge, delete, cancel, resolve command submission, and ops repair commands in `integration/cli/real_state_destructive_test.go`
- [x] T053 [US3] Implement mixed valid, missing, malformed, stale, and already-mutated target sets in `integration/cli/real_state_destructive_test.go`
- [x] T054 [US3] Add fail-fast, partial-failure accounting, and machine-output cleanliness assertions for command stdout and ops reports in `integration/cli/real_state_destructive_test.go`
- [x] T055 [US3] Reuse or extend ops report parity checks for real-state reports in `integration/cli/real_state_destructive_test.go`
- [x] T056 [US3] Update `gaps.md` for retention, purge, orphan, repair, or partial-failure prerequisites that still cannot be created through c8volt
- [x] T057 [US3] Update retention and destructive rows to current statuses in `specs/257-c89-real-state-integration/coverage-matrix.md`
- [x] T058 [US3] Validate User Story 3 with `make integration-cli-real-state-retention IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m` and `make integration-cli-real-state-destructive IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m` documented in `specs/257-c89-real-state-integration/quickstart.md`

**Checkpoint**: User Story 3 now proves destructive real-candidate semantics, dry-run safety, confirmed outcomes, mixed-target failure handling, fail-fast behavior, machine-output cleanliness, and ops report parity for the covered retention, repair, and incident-purge slices.

---

## Phase 7: User Story 4 - Keep Real-State Gaps Visible And Extensible (Priority: P4)

**Goal**: Keep `gaps.md` and the coverage matrix accurate as the real-state suite grows.

**Independent Test**: Run `make integration-cli-real-state-gaps IT_GO_TEST_FLAGS=-v` and inspect `gaps.md` plus `coverage-matrix.md`.

### Tests For User Story 4

- [x] T059 [P] [US4] Add failing `TestRealStateGapFamily` skeleton for spec-owned gap validation in `integration/cli/real_state_gap_validation_test.go`
- [x] T060 [P] [US4] Add failing coverage-matrix status validation for every priority topic in `integration/cli/real_state_gap_validation_test.go`

### Implementation For User Story 4

- [x] T061 [US4] Wire `integration-cli-real-state-gaps` target to `TestRealStateGapFamily` in `Makefile`
- [x] T062 [US4] Implement static validation that `gaps.md` includes blocked proof, affected commands, affected versions, and runtime behavior for every open prerequisite gap
- [x] T063 [US4] Implement coverage-matrix status checks for live-covered, partially live-covered, dry-run-covered, skipped-prerequisite, no-match only, and not-yet-started rows
- [x] T064 [US4] Update affected-version handling for Camunda 8.9 focus and future minor extension in `gaps.md`
- [x] T065 [US4] Update real-state gap and matrix validation instructions in `specs/257-c89-real-state-integration/quickstart.md`
- [x] T066 [US4] Validate User Story 4 with `make integration-cli-real-state-gaps IT_GO_TEST_FLAGS=-v` documented in `specs/257-c89-real-state-integration/quickstart.md`

**Checkpoint**: User Story 4 makes every remaining real-state gap visible without giving integration tests backlog ownership, and `integration-cli-real-state-gaps` validates the spec-owned gap and matrix contracts without touching Camunda.

---

## Phase 8: Polish And Cross-Cutting

**Purpose**: Final consistency, documentation, and validation across the feature.

- [x] T067 [P] Update real-state target descriptions, destructive warnings, and command help/example danger-audit notes in `integration/README.md`
- [x] T068 [P] Update implementation validation notes in `specs/257-c89-real-state-integration/quickstart.md`
- [x] T069 [P] Update final statuses and first follow-up notes in `specs/257-c89-real-state-integration/coverage-matrix.md`
- [x] T070 Run `gofmt -w integration/cli/real_state_harness_test.go integration/cli/real_state_data_test.go integration/cli/real_state_jobs_test.go integration/cli/real_state_incidents_test.go integration/cli/real_state_listeners_test.go integration/cli/real_state_bpmn_error_test.go integration/cli/real_state_retention_test.go integration/cli/real_state_destructive_test.go integration/cli/real_state_gap_validation_test.go`
- [x] T071 Run `GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1` for `integration/cli`
- [x] T072 Run `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m` for `integration/cli`
- [x] T073 Run all real-state Make targets from `Makefile` against the selected Camunda 8.9 profile
- [x] T074 Run `make test` for repository-wide validation from `Makefile`
- [x] T075 Verify whether command help/example metadata changed; if it changed, run `make docs-content` and include generated docs, otherwise record that generated CLI docs were not required in `specs/257-c89-real-state-integration/quickstart.md`
- [x] T076 Run `git diff --check -- Makefile integration/README.md integration/cli specs/257-c89-real-state-integration specs/integration-test-responsibility.md` before final review

---

## Dependencies And Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup completion and blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion and is the MVP.
- **User Story 2 (Phase 4)**: Depends on Foundational completion; can run after or beside US1 when shared helpers are stable.
- **Correction Gate (Phase 5)**: Depends on US1 and US2 because it migrates the existing proposal-evidence implementation into runtime evidence plus spec-owned gaps.
- **User Story 3 (Phase 6)**: Depends on the Correction Gate; destructive scenarios should not add more proposal-generation paths.
- **User Story 4 (Phase 7)**: Depends on the Correction Gate and should be revisited after each story changes `gaps.md` or matrix status.
- **Polish (Phase 8)**: Depends on all selected user stories for the current implementation slice.

### User Story Dependencies

- **US1**: Start first for MVP because it proves real jobs and incidents.
- **US2**: Can start after foundational helpers; listener and BPMN fixtures may produce skipped-prerequisite evidence before live coverage.
- **Correction Gate**: Must run before US3 so tests stop generating backlog proposal files.
- **US3**: Can start after the correction gate; destructive scenarios should use fixture patterns learned in US1 where useful.
- **US4**: Can start after the correction gate, then repeat after US3 updates gap and matrix evidence.

### Parallel Opportunities

- T002, T003, and T004 can run in parallel.
- T014 and T015 can run in parallel after foundational helpers exist.
- T026 and T027 can run in parallel after foundational helpers exist.
- T039 and T040 can run in parallel after the spec correction is complete.
- T046 and T047 can run in parallel after the correction gate.
- T059 and T060 can run in parallel after the correction gate.
- Documentation and matrix polish tasks T067, T068, and T069 can run in parallel.

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
Task: "T046 Add failing TestRealStateRetentionFamily skeleton in integration/cli/real_state_retention_test.go"
Task: "T047 Add failing TestRealStateDestructiveFamily skeleton in integration/cli/real_state_destructive_test.go"
```

## Parallel Example: User Story 4

```text
Task: "T059 Add failing TestRealStateGapFamily skeleton in integration/cli/real_state_gap_validation_test.go"
Task: "T060 Add failing coverage-matrix status validation in integration/cli/real_state_gap_validation_test.go"
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
3. Add listener and BPMN error runtime coverage.
4. Correct the runtime evidence and spec-owned gap boundary.
5. Add destructive real-candidate coverage.
6. Keep gap artifacts and matrix status current after every slice.

### Validation Discipline

- Run local compile and non-integration guards before destructive Camunda runs.
- Run each real-state target independently.
- Keep evidence outside `docs/`.
- Prefer c8volt commands and embedded BPMN fixtures for setup.
- Record skipped-prerequisite or dry-run runtime evidence when live proof is blocked.
- Maintain missing command setup and embedded BPMN needs in spec-owned gap artifacts, not generated test evidence.
