# Tasks: All-Command Integration Suite

**Input**: Design documents from `/specs/255-all-command-integration/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: This feature is an integration test suite. Story test tasks are required and should be written before implementation in each slice.

**Organization**: Tasks are grouped by user story so each slice can be implemented and validated independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files or only adds isolated coverage
- **[Story]**: Maps to user stories from [spec.md](./spec.md)
- Every task names exact repository paths

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish the isolated suite package and shared harness boundaries.

- [ ] T001 Create `integration/cli/` package with build-tagged `integration/cli/harness_test.go`
- [ ] T002 Add package-level suite configuration types and default environment names in `integration/cli/harness_test.go`
- [ ] T003 Add binary build-once support in `integration/cli/harness_test.go` using the repository root as build input and a temporary output path
- [ ] T004 Add subprocess command runner with stdout/stderr capture in `integration/cli/harness_test.go`
- [ ] T005 Add evidence workdir initialization and path helpers in `integration/cli/harness_test.go`
- [ ] T006 [P] Add `integration/cli/testdata/.gitkeep` if a stable empty testdata directory is needed for future fixtures
- [ ] T007 [P] Update `integration/README.md` with the future `go test -tags=integration ./integration/cli` entry point

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build shared manifest, profile, evidence, and safety primitives required by every story.

**Critical**: No command-family user story should begin until this phase is complete.

- [ ] T008 Add command inventory structs matching `capabilities --json` in `integration/cli/all_commands_test.go`
- [ ] T009 Add coverage manifest structs for command paths, aliases, flags, output modes, version expectations, and destructive classification in `integration/cli/all_commands_test.go`
- [ ] T010 Add initial manifest entries for all 55 command nodes in `integration/cli/all_commands_test.go`
- [ ] T011 Add profile selection and default-config guardrails in `integration/cli/harness_test.go`
- [ ] T012 Add profile connectivity/version gate helpers in `integration/cli/harness_test.go`
- [ ] T013 Add run marker generation and JSON variable payload helpers in `integration/cli/harness_test.go`
- [ ] T014 Add evidence record writer for command outputs and exit codes in `integration/cli/harness_test.go`
- [ ] T015 Add proposal record writers for command and embedded BPMN gaps in `integration/cli/harness_test.go`
- [ ] T016 Add dirty-cluster-safe assertion helpers in `integration/cli/harness_test.go`
- [ ] T017 Add common JSON, keys-only, and human-output assertion helpers in `integration/cli/harness_test.go`
- [ ] T018 Run `go test -tags=integration ./integration/cli -run '^$' -count=1` to validate package compilation before user-story work

**Checkpoint**: The suite package compiles, can build the CLI, can run subprocesses, can write evidence, and has a complete manifest skeleton.

---

## Phase 3: User Story 1 - Discover Complete Command Coverage (Priority: P1) MVP

**Goal**: The suite fails when the live command inventory and manifest diverge.

**Independent Test**: Run the inventory check and verify every live command path has a coverage entry.

### Tests for User Story 1

- [ ] T019 [P] [US1] Add failing inventory-count test for the 55 current command nodes in `integration/cli/all_commands_test.go`
- [ ] T020 [P] [US1] Add failing missing-path and stale-path manifest tests in `integration/cli/all_commands_test.go`
- [ ] T021 [P] [US1] Add failing missing-flag coverage test for leaf commands in `integration/cli/all_commands_test.go`

### Implementation for User Story 1

- [ ] T022 [US1] Implement live `capabilities --json` discovery and flattening in `integration/cli/all_commands_test.go`
- [ ] T023 [US1] Implement manifest comparison for missing paths, stale paths, missing flags, aliases, and output modes in `integration/cli/all_commands_test.go`
- [ ] T024 [US1] Write `inventory.json` and `coverage.json` evidence for inventory checks in `integration/cli/all_commands_test.go`
- [ ] T025 [US1] Verify MVP with `go test -tags=integration ./integration/cli -run TestCommandInventory -count=1 -timeout=10m`

**Checkpoint**: User Story 1 is complete when command inventory drift fails loudly before any cluster mutation.

---

## Phase 4: User Story 2 - Validate Real Local Profiles (Priority: P2)

**Goal**: The suite uses only the operator's default local c8volt configuration and validates profiles before destructive work.

**Independent Test**: Run profile gates against selected local profiles and verify readiness evidence.

### Tests for User Story 2

- [ ] T026 [P] [US2] Add failing test that rejects explicit generated config usage in `integration/cli/config_test.go`
- [ ] T027 [P] [US2] Add failing profile connectivity/version evidence test in `integration/cli/config_test.go`
- [ ] T028 [P] [US2] Add failing read-only smoke test for `version`, `capabilities`, `config validate`, and `config test-connection` in `integration/cli/config_test.go`

### Implementation for User Story 2

- [ ] T029 [US2] Implement profile discovery/selection from default local config behavior in `integration/cli/harness_test.go`
- [ ] T030 [US2] Implement version gate execution for selected profiles in `integration/cli/config_test.go`
- [ ] T031 [US2] Implement read-only smoke scenarios and evidence capture in `integration/cli/config_test.go`
- [ ] T032 [US2] Write `profiles.json` evidence with reachable, expected, and actual version fields in `integration/cli/config_test.go`
- [ ] T033 [US2] Verify with `go test -tags=integration ./integration/cli -run 'TestProfiles|TestReadOnlySmoke' -count=1 -timeout=10m`

**Checkpoint**: User Story 2 is complete when destructive family tests are gated behind proven disposable profile readiness.

---

## Phase 5: User Story 3 - Seed And Reuse Disposable Cluster Data (Priority: P3)

**Goal**: The suite can prepare data on clean clusters and coexist with unrelated dirty-cluster data.

**Independent Test**: Run seeded-data setup and confirm run-owned evidence without exact global count assumptions.

### Tests for User Story 3

- [ ] T034 [P] [US3] Add failing embedded fixture discovery test in `integration/cli/deploy_embed_run_test.go`
- [ ] T035 [P] [US3] Add failing deploy-and-run seeded process data test in `integration/cli/deploy_embed_run_test.go`
- [ ] T036 [P] [US3] Add failing dirty-cluster assertion test using unrelated search results in `integration/cli/get_test.go`

### Implementation for User Story 3

- [ ] T037 [US3] Implement `embed list` and version-matched embedded fixture selection in `integration/cli/deploy_embed_run_test.go`
- [ ] T038 [US3] Implement `embed deploy` or `deploy process-definition` seeded definition setup in `integration/cli/deploy_embed_run_test.go`
- [ ] T039 [US3] Implement `run process-instance` seeded instance creation with run marker variables in `integration/cli/deploy_embed_run_test.go`
- [ ] T040 [US3] Persist seeded process definition keys, process instance keys, and resource IDs under evidence `data/` in `integration/cli/harness_test.go`
- [ ] T041 [US3] Implement best-effort cleanup tracking without requiring cleanup success in `integration/cli/harness_test.go`
- [ ] T042 [US3] Verify with `go test -tags=integration ./integration/cli -run TestSeededData -count=1 -timeout=20m`

**Checkpoint**: User Story 3 is complete when clean and dirty clusters can both provide usable command targets.

---

## Phase 6: User Story 4 - Exercise Command Families And Flags (Priority: P4)

**Goal**: Every command family records scenario evidence for flags, aliases, output modes, version behavior, validation behavior, and destructive paths.

**Independent Test**: Run a single command-family group and confirm manifest coverage for that group is satisfied.

### Tests for User Story 4

- [ ] T043 [P] [US4] Add failing `get` family coverage tests in `integration/cli/get_test.go`
- [ ] T044 [P] [US4] Add failing `deploy`, `embed`, and `run` family coverage tests in `integration/cli/deploy_embed_run_test.go`
- [ ] T045 [P] [US4] Add failing `update` family coverage tests in `integration/cli/update_test.go`
- [ ] T046 [P] [US4] Add failing `cancel` family coverage tests in `integration/cli/cancel_test.go`
- [ ] T047 [P] [US4] Add failing `delete` family coverage tests in `integration/cli/delete_test.go`
- [ ] T048 [P] [US4] Add failing `expect` and `resolve` family coverage tests in `integration/cli/expect_resolve_test.go`
- [ ] T049 [P] [US4] Add failing `walk` family coverage tests in `integration/cli/walk_test.go`
- [ ] T050 [P] [US4] Add failing `ops analyse` family coverage tests in `integration/cli/ops_analyse_test.go`
- [ ] T051 [P] [US4] Add failing `ops execute` family coverage tests in `integration/cli/ops_execute_test.go`
- [ ] T052 [P] [US4] Add failing `ops purge` family coverage tests in `integration/cli/ops_purge_test.go`
- [ ] T053 [P] [US4] Add failing `ops repair` family coverage tests in `integration/cli/ops_repair_test.go`

### Implementation for User Story 4

- [ ] T054 [US4] Implement `get` command scenarios for cluster, process-definition, process-instance, resource, incident, job, element, and tenant commands in `integration/cli/get_test.go`
- [ ] T055 [US4] Implement `deploy`, `embed`, and `run` scenarios including aliases, required selectors, variables, count, no-wait, and output checks in `integration/cli/deploy_embed_run_test.go`
- [ ] T056 [US4] Implement `update process-instance` and `update job` scenarios including dry-run, worker outcome, variables, and validation paths in `integration/cli/update_test.go`
- [ ] T057 [US4] Implement `cancel process-instance` scenarios including key, filter, dry-run, force, workers, no-wait, and validation paths in `integration/cli/cancel_test.go`
- [ ] T058 [US4] Implement `delete process-instance` and `delete process-definition` scenarios including key/filter selectors, dry-run, force, latest/version flags, and validation paths in `integration/cli/delete_test.go`
- [ ] T059 [US4] Implement `expect process-instance`, `resolve incident`, and `resolve process-instance` scenarios including stdin keys, dry-run, no-wait, and state checks in `integration/cli/expect_resolve_test.go`
- [ ] T060 [US4] Implement `walk process-instance` scenarios for parent, children, flat, with-vars, with-incidents, with-elements, and with-listeners proposal fallback in `integration/cli/walk_test.go`
- [ ] T061 [US4] Implement `ops analyse slow-process-instances` scenarios including key, filters, durations, timeline, listeners, json, and keys-only behavior in `integration/cli/ops_analyse_test.go`
- [ ] T062 [US4] Implement `ops execute smoke-test` and `ops execute retention-policy` scenarios including dry-run, reports, count, workers, no-wait, and confirmed execution in `integration/cli/ops_execute_test.go`
- [ ] T063 [US4] Implement `ops purge` scenarios for all-process-definitions, orphan-process-instances, and process-instances-with-incidents including dry-run, reports, filters, workers, and confirmed execution in `integration/cli/ops_purge_test.go`
- [ ] T064 [US4] Implement `ops repair incident` and `ops repair process-instance` scenarios including keys, search filters, vars, retries, timeout, reports, dry-run, and confirmed execution in `integration/cli/ops_repair_test.go`
- [ ] T065 [US4] Implement family-level manifest satisfaction checks after each command-family scenario in `integration/cli/all_commands_test.go`
- [ ] T066 [US4] Verify family slices with targeted `go test -tags=integration ./integration/cli -run 'TestGetFamily|TestWalkFamily|TestOps' -count=1 -timeout=60m`

**Checkpoint**: User Story 4 is complete when each command family can be run independently and the aggregate manifest marks all flags/outputs/version expectations covered.

---

## Phase 7: User Story 5 - Report Setup Gaps As Product Proposals (Priority: P5)

**Goal**: Direct API setup and missing embedded model needs are captured as proposal records.

**Independent Test**: Run scenarios that require unavailable setup support and verify proposal output is generated.

### Tests for User Story 5

- [ ] T067 [P] [US5] Add failing command proposal report test in `integration/cli/harness_test.go`
- [ ] T068 [P] [US5] Add failing embedded BPMN proposal report test in `integration/cli/harness_test.go`
- [ ] T069 [P] [US5] Add failing empty-proposal JSON array test for no-gap runs in `integration/cli/harness_test.go`

### Implementation for User Story 5

- [ ] T070 [US5] Implement direct Camunda setup fallback registration in `integration/cli/harness_test.go`
- [ ] T071 [US5] Implement missing embedded BPMN proposal registration in `integration/cli/harness_test.go`
- [ ] T072 [US5] Wire proposal recording into listener, BPMN error, variable-shape, duration, retention, and incident/job-state gap scenarios in `integration/cli/walk_test.go`, `integration/cli/update_test.go`, `integration/cli/ops_analyse_test.go`, and `integration/cli/ops_execute_test.go`
- [ ] T073 [US5] Verify proposal outputs with `go test -tags=integration ./integration/cli -run TestProposalReports -count=1 -timeout=10m`

**Checkpoint**: User Story 5 is complete when setup gaps are visible, actionable, and separated from product behavior changes.

---

## Phase 8: User Story 6 - Validate Help And Example Trustworthiness (Priority: P6)

**Goal**: Help and generated CLI examples are executable or produce actionable failures, and destructive examples are warned.

**Independent Test**: Run example validation and inspect `examples.json`.

### Tests for User Story 6

- [ ] T074 [P] [US6] Add failing help example extraction test in `integration/cli/examples_test.go`
- [ ] T075 [P] [US6] Add failing generated CLI docs example extraction test in `integration/cli/examples_test.go`
- [ ] T076 [P] [US6] Add failing placeholder substitution test in `integration/cli/examples_test.go`
- [ ] T077 [P] [US6] Add failing destructive-warning detection test in `integration/cli/examples_test.go`

### Implementation for User Story 6

- [ ] T078 [US6] Implement command help example extraction and normalization in `integration/cli/examples_test.go`
- [ ] T079 [US6] Implement generated `docs/cli/*.md` example extraction without editing generated docs in `integration/cli/examples_test.go`
- [ ] T080 [US6] Implement placeholder substitution from seeded data and embedded fixture evidence in `integration/cli/examples_test.go`
- [ ] T081 [US6] Implement read-only and disposable-target example execution in `integration/cli/examples_test.go`
- [ ] T082 [US6] Implement destructive-warning validation and source-location reporting in `integration/cli/examples_test.go`
- [ ] T083 [US6] Write `examples.json` evidence with pass/fail/source-location details in `integration/cli/examples_test.go`
- [ ] T084 [US6] Verify with `go test -tags=integration ./integration/cli -run TestExamples -count=1 -timeout=20m`

**Checkpoint**: User Story 6 is complete when examples are validated and unsafe mutating examples cannot pass without warnings.

---

## Phase 9: Polish & Cross-Cutting Validation

**Purpose**: Final validation, evidence review, and integration documentation alignment.

- [ ] T085 [P] Review `integration/assets/all-command-go-integration-rules.md` against implemented suite behavior and update only if suite scope changed
- [ ] T086 [P] Update `integration/assets/command-matrix.md` with the final all-command coverage map
- [ ] T087 [P] Update `integration/README.md` with final run commands, environment variables, and evidence layout
- [ ] T088 Run `gofmt` on all Go files under `integration/cli/`
- [ ] T089 Run `go test -tags=integration ./integration/cli -run TestCommandInventory -count=1 -timeout=10m`
- [ ] T090 Run `go test -tags=integration ./integration/cli -count=1 -timeout=60m` against disposable local profiles
- [ ] T091 Run `go test ./integration/cli -count=1` without the integration tag and verify the package is excluded or harmless
- [ ] T092 Run `make test` to confirm normal unit validation is unaffected
- [ ] T093 Review generated evidence to ensure nothing is written under `docs/`
- [ ] T094 Review `git diff` to ensure changes are isolated to `integration/`, `specs/255-all-command-integration/`, and accepted harness-only files

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup and blocks all user stories.
- **US1 Inventory (Phase 3)**: Depends on Foundational and is the MVP.
- **US2 Profiles (Phase 4)**: Depends on Foundational; should complete before destructive stories.
- **US3 Seeded Data (Phase 5)**: Depends on US2 for safe profile gates.
- **US4 Command Families (Phase 6)**: Depends on US1, US2, and US3.
- **US5 Proposal Reports (Phase 7)**: Can begin after Foundational, but final wiring depends on US4 gap scenarios.
- **US6 Examples (Phase 8)**: Depends on US3 for dynamic substitution and benefits from US4 command coverage.
- **Polish (Phase 9)**: Depends on desired stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: MVP after Foundational.
- **User Story 2 (P2)**: Required before destructive execution.
- **User Story 3 (P3)**: Required before most command-family scenarios.
- **User Story 4 (P4)**: Main broad coverage; depends on seeded data.
- **User Story 5 (P5)**: Proposal infrastructure can start early; full value arrives with gap scenarios.
- **User Story 6 (P6)**: Should run after seeded data and command-family coverage exist.

### Parallel Opportunities

- Setup documentation and harness helpers marked [P] can run in parallel.
- Foundational manifest/profile/evidence helpers can be split by file once `harness_test.go` basics exist.
- Command-family tests in Phase 6 can run in parallel after seeded data support exists.
- Proposal and example validation work can proceed in parallel with later command-family implementation.

---

## Parallel Example: User Story 4

```text
Task: "Add failing get family coverage tests in integration/cli/get_test.go"
Task: "Add failing update family coverage tests in integration/cli/update_test.go"
Task: "Add failing ops execute family coverage tests in integration/cli/ops_execute_test.go"
Task: "Add failing ops repair family coverage tests in integration/cli/ops_repair_test.go"
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete User Story 1 only.
3. Validate `TestCommandInventory`.
4. Stop before mutating clusters.

### Incremental Delivery

1. Add profile gates and read-only smoke checks.
2. Add seeded data creation.
3. Add one command family at a time.
4. Add proposal reporting wherever direct setup or fixture gaps appear.
5. Add example validation last.

### Safety Rhythm

Always validate inventory and profile readiness before destructive scenarios. Run command-family slices against disposable profiles before running the full suite.

## Notes

- Optional Speckit git hooks were not invoked during task generation.
- Optional Ralph-after-tasks hook must not be launched unless the user explicitly asks to implement the suite.
- Normal Ralph implementation rules still apply if this feature is later implemented by Ralph; the integration rules file is feature-specific context, not global context.
