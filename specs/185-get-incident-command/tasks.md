# Tasks: Get Incident Command

**Input**: Design documents from `/specs/185-get-incident-command/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/
**Tests**: Required by repository constitution and feature risk.
**Commit Rule**: Any commit subject for this feature must use Conventional Commits and end with `#185`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files and has no dependency on incomplete tasks.
- **[Story]**: Maps work to the user story from spec.md.
- Every task includes exact repository paths.

## Phase 1: Setup (Shared Discovery)

**Purpose**: Confirm existing generated client, incident service, facade, and renderer surfaces before implementation.

- [x] T001 Inspect top-level incident generated client search and lookup methods in `internal/clients/camunda/v88/camunda/client.gen.go` and `internal/clients/camunda/v89/camunda/client.gen.go`
- [x] T002 Inspect current incident service methods and version behavior in `internal/services/incident/api.go`, `internal/services/incident/v87/incidents.go`, `internal/services/incident/v88/incidents.go`, and `internal/services/incident/v89/incidents.go`
- [x] T003 Inspect existing process-instance incident validation and rendering in `cmd/get_processinstance_validation.go`, `cmd/cmd_views_processinstance_incidents.go`, and `cmd/get_processinstance_test.go`
- [x] T004 Inspect existing list, paging, limit, keys-only, total, and JSON conventions in `cmd/get_processinstance.go`, `cmd/get_processinstance_paging.go`, `cmd/get_processinstance_total.go`, and `cmd/cmd_views_get.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared incident query, filter, service, facade, and view foundations needed by every story.

**CRITICAL**: No user story command work should begin until this phase is complete.

- [x] T005 Add incident query/filter facade models and result metadata in `c8volt/process/model.go`
- [x] T006 Extend the process facade API for keyed incident lookup and incident search/list in `c8volt/process/api.go`
- [x] T007 Extend conversion helpers for incident query/filter/result values in `c8volt/process/convert.go`
- [x] T008 Extend the incident service API with top-level incident search/list support in `internal/services/incident/api.go`
- [x] T009 [P] Add v8.7 unsupported incident search/list tests in `internal/services/incident/v87/incidents_test.go`
- [x] T010 [P] Add v8.8 incident search/list compatibility tests in `internal/services/incident/v88/incidents_test.go`
- [x] T011 [P] Add v8.9 incident search/list server-filter tests in `internal/services/incident/v89/incidents_test.go`
- [x] T012 Implement v8.7 unsupported incident search/list behavior in `internal/services/incident/v87/incidents.go` and `internal/services/incident/v87/contract.go`
- [x] T013 Implement v8.8 incident search/list compatibility path in `internal/services/incident/v88/incidents.go`, `internal/services/incident/v88/convert.go`, and `internal/services/incident/v88/contract.go`
- [x] T014 Implement v8.9 incident search/list server-side filters in `internal/services/incident/v89/incidents.go`, `internal/services/incident/v89/convert.go`, and `internal/services/incident/v89/contract.go`
- [x] T015 Add factory/API compile and version selection tests for incident search/list in `internal/services/incident/factory_test.go`
- [x] T016 Add shared incident row formatting helpers reusable by process-instance and incident output in `cmd/cmd_views_processinstance_incidents.go`
- [x] T017 Add facade tests for incident query validation, service option mapping, result metadata, and unsupported-version propagation in `c8volt/process/client_test.go`
- [x] T018 Implement facade orchestration for keyed lookup and incident search/list in `c8volt/process/client.go`

**Checkpoint**: Incident service and facade can return incident details for keyed lookup and search/list without CLI command wiring.

---

## Phase 3: User Story 1 - Fetch Known Incidents (Priority: P1) MVP

**Goal**: Fetch explicit incident keys through `c8volt get incident`, `get incidents`, and `get inc`.

**Independent Test**: Running `c8volt get incident --key <incident-key>` fetches one known incident; repeated flags and stdin deduplicate keys; human, JSON, and keys-only output work.

### Tests for User Story 1

- [x] T019 [P] [US1] Add command tests for `get incident --key`, repeated `--key`, stdin `-`, deduplication, missing keys, and invalid keys in `cmd/get_incident_test.go`
- [x] T020 [P] [US1] Add human, JSON, and keys-only incident view tests in `cmd/cmd_views_get_test.go`
- [x] T021 [P] [US1] Add command contract expectations for `get incident`, aliases `incidents` and `inc`, and inherited get flags in `cmd/command_contract_test.go`

### Implementation for User Story 1

- [x] T022 [US1] Register `get incident` with aliases, examples, flags, and help text in `cmd/get_incident.go` and wire it from `cmd/get.go`
- [x] T023 [US1] Implement keyed lookup parsing, stdin `-` handling, key merge, validation, and facade invocation in `cmd/get_incident.go`
- [x] T024 [US1] Implement incident human, JSON, and keys-only rendering in `cmd/cmd_views_get.go` and `cmd/cmd_views_processinstance_incidents.go`
- [x] T025 [US1] Ensure keyed lookup not-found and partial lookup failures preserve existing get command exit/output conventions in `cmd/get_incident.go`

**Checkpoint**: User Story 1 is fully functional and independently testable.

---

## Phase 4: User Story 2 - Search Incidents By Core Fields (Priority: P2)

**Goal**: Search/list incidents by state, error type, process context, and flow-node context.

**Independent Test**: Running `get incident` without keys and with each core filter returns only matching incidents while preserving pagination/list conventions.

### Tests for User Story 2

- [x] T026 [P] [US2] Add command tests for default active state, `--state all`, invalid states, and state output in `cmd/get_incident_test.go`
- [x] T027 [P] [US2] Add command tests for case-insensitive `--error-type` validation and generated valid-value messages in `cmd/get_incident_test.go`
- [x] T028 [P] [US2] Add command tests for process instance, root process instance, process definition, flow node, and flow node instance filters in `cmd/get_incident_test.go`
- [x] T029 [P] [US2] Add facade/service tests proving server-safe filter options are passed through in `c8volt/process/client_test.go` and `internal/services/incident/v89/incidents_test.go`

### Implementation for User Story 2

- [x] T030 [US2] Add search/list flags and validation for state, error type, process context, and flow-node context in `cmd/get_incident.go`
- [x] T031 [US2] Reuse `internal/services/incidentfilter` for error type normalization and valid-value help text in `cmd/get_incident.go`
- [x] T032 [US2] Map validated search filters to facade/service options in `c8volt/process/client.go` and `internal/services/calloption.go`
- [x] T033 [US2] Preserve existing get pagination, limit, interactive, auto-confirm, and non-interactive behavior for incident search in `cmd/get_incident.go`

**Checkpoint**: User Story 2 is functional without relying on process-instance incident view extender flags.

---

## Phase 5: User Story 3 - Search Incident Messages Safely (Priority: P3)

**Goal**: Search incident error messages with case-insensitive substring semantics across all relevant pages.

**Independent Test**: Searching by mixed-case message substrings across multiple backend pages returns all matching incidents up to the explicit limit.

### Tests for User Story 3

- [ ] T034 [P] [US3] Add command tests for case-insensitive `--error-message` matching in `cmd/get_incident_test.go`
- [ ] T035 [P] [US3] Add service/facade tests proving local message filtering pages beyond the first page in `c8volt/process/client_test.go` and `internal/services/incident/v88/incidents_test.go`
- [ ] T036 [P] [US3] Add v8.8 compatibility tests proving known broken scoped `filter` request shapes are not sent in `internal/services/incident/v88/incidents_test.go`

### Implementation for User Story 3

- [ ] T037 [US3] Add `--error-message` parsing and validation in `cmd/get_incident.go`
- [ ] T038 [US3] Reuse existing case-insensitive message matching helper behavior from `internal/services/incidentfilter/incidentfilter.go`
- [ ] T039 [US3] Implement local post-filter pagination for message filtering in `c8volt/process/client.go` and `internal/services/incident/v88/incidents.go`
- [ ] T040 [US3] Ensure explicit command limits stop local filtering only after enough matching results are found or search is exhausted in `c8volt/process/client.go`

**Checkpoint**: User Story 3 is independently complete for message filtering correctness.

---

## Phase 6: User Story 4 - Filter By Creation Time (Priority: P4)

**Goal**: Filter incidents by `creationTime` using after and before bounds.

**Independent Test**: Valid date/timestamp bounds filter incidents by creation time, combined bounds form a window, and invalid values fail before remote calls.

### Tests for User Story 4

- [ ] T041 [P] [US4] Add command tests for `--creation-time-after`, `--creation-time-before`, combined time windows, and invalid date values in `cmd/get_incident_test.go`
- [ ] T042 [P] [US4] Add v8.9 service tests for creation-time request shape in `internal/services/incident/v89/incidents_test.go`
- [ ] T043 [P] [US4] Add local fallback creation-time filtering tests where needed in `c8volt/process/client_test.go`

### Implementation for User Story 4

- [ ] T044 [US4] Add creation-time flags using existing date parsing conventions in `cmd/get_incident.go`
- [ ] T045 [US4] Add creation-time call options and conversion in `internal/services/calloption.go`, `internal/services/incident/v88/convert.go`, and `internal/services/incident/v89/convert.go`
- [ ] T046 [US4] Apply creation-time bounds through safe server-side filters or local fallback in `internal/services/incident/v88/incidents.go`, `internal/services/incident/v89/incidents.go`, and `c8volt/process/client.go`

**Checkpoint**: User Story 4 is independently complete for time-window searches.

---

## Phase 7: User Story 5 - Render Incident Lists And Counts (Priority: P5)

**Goal**: Render incident human rows, JSON, keys-only, and exact totals.

**Independent Test**: The same filtered result set can be rendered as human rows, JSON, keys-only, and exact numeric totals, with full JSON messages and optional human truncation.

### Tests for User Story 5

- [ ] T047 [P] [US5] Add human row tests for tenant, state, error type, creation time, process context, flow-node context, job key `n/a`, message, and age in `cmd/cmd_views_get_test.go`
- [ ] T048 [P] [US5] Add JSON output tests proving full `errorMessage` and `creationTime` are preserved in `cmd/get_incident_test.go`
- [ ] T049 [P] [US5] Add keys-only and exact `--total` command tests, including local-filter totals, in `cmd/get_incident_test.go`
- [ ] T050 [P] [US5] Add validation tests rejecting `--total --json`, `--total --keys-only`, and `--error-message-limit` with non-human output in `cmd/get_incident_test.go`

### Implementation for User Story 5

- [ ] T051 [US5] Add incident age calculation and missing or unparsable `creationTime` handling in `cmd/cmd_views_processinstance_incidents.go`
- [ ] T052 [US5] Add `--error-message-limit` handling for human incident output in `cmd/get_incident.go` and `cmd/cmd_views_processinstance_incidents.go`
- [ ] T053 [US5] Implement exact `--total` output after all local filters in `cmd/get_incident.go` and `c8volt/process/client.go`
- [ ] T054 [US5] Ensure JSON output preserves full incident fields without human truncation in `cmd/cmd_views_get.go`

**Checkpoint**: User Story 5 is complete for script and operator output modes.

---

## Phase 8: User Story 6 - Preserve Command Contracts (Priority: P6)

**Goal**: Keep command validation, docs, generated docs, unsupported-version behavior, and existing process-instance incident workflows stable.

**Independent Test**: Help text, command contracts, docs, unsupported versions, and existing `get pi` incident regression tests pass.

### Tests for User Story 6

- [ ] T055 [P] [US6] Add regression tests proving `get pi --with-incidents`, `--incidents-only`, `--direct-incidents-only`, and `--no-incidents-only` behavior is unchanged in `cmd/get_processinstance_test.go`
- [ ] T056 [P] [US6] Add docs generation tests or update expectations for `get incident` docs in `docsgen/main_test.go`
- [ ] T057 [P] [US6] Add unsupported-version behavior tests for `get incident` in `cmd/get_incident_test.go` and `internal/services/incident/v87/incidents_test.go`

### Implementation for User Story 6

- [ ] T058 [US6] Update README examples and command overview for `get incident` in `README.md`
- [ ] T059 [US6] Regenerate CLI reference markdown with `make docs-content`, updating `docs/cli/c8volt_get.md`, `docs/cli/c8volt_get_incident.md`, and `docs/cli/index.md`
- [ ] T060 [US6] Verify no `--with-incidents`, `--incidents-only`, or `--direct-incidents-only` flags were added to `get incident` in `cmd/get_incident.go`
- [ ] T061 [US6] Verify incident search logic remains out of `internal/services/processinstance/` and inside `internal/services/incident/`

**Checkpoint**: User-facing documentation and existing workflows remain consistent.

---

## Final Phase: Validation & Handoff

**Purpose**: Prove the complete feature before commit or PR handoff.

- [ ] T062 Run targeted service validation with `GOCACHE=/tmp/c8volt-gocache go test ./internal/services/incident/... -count=1`
- [ ] T063 Run targeted facade validation with `GOCACHE=/tmp/c8volt-gocache go test ./c8volt/process -count=1`
- [ ] T064 Run targeted command validation with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestGetIncident|TestCommandContract|TestGetProcessInstance' -count=1`
- [ ] T065 Run docs validation with `GOCACHE=/tmp/c8volt-gocache go test ./docsgen -count=1`
- [ ] T066 Run repository validation with `make test`
- [ ] T067 Review `git diff --check` and final changed files before committing

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup and blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational.
- **User Story 2 (Phase 4)**: Depends on Foundational and may reuse US1 command/view helpers.
- **User Story 3 (Phase 5)**: Depends on search/list foundation from US2.
- **User Story 4 (Phase 6)**: Depends on search/list foundation from US2.
- **User Story 5 (Phase 7)**: Depends on US1 through US4 result shapes.
- **User Story 6 (Phase 8)**: Depends on stable command behavior for docs and regression checks.
- **Validation**: Depends on all desired user stories.

### User Story Dependencies

- **US1 Fetch Known Incidents**: MVP after Foundational.
- **US2 Search Incidents By Core Fields**: Adds search/list mode after foundation.
- **US3 Search Incident Messages Safely**: Builds on search/list and compatibility paths.
- **US4 Filter By Creation Time**: Builds on search/list and date parsing conventions.
- **US5 Render Incident Lists And Counts**: Builds on result shapes from lookup and search.
- **US6 Preserve Command Contracts**: Final compatibility and documentation pass.

### Parallel Opportunities

- T009, T010, and T011 can run in parallel after T008.
- T019, T020, and T021 can run in parallel.
- T026, T027, T028, and T029 can run in parallel.
- T034, T035, and T036 can run in parallel.
- T041, T042, and T043 can run in parallel.
- T047, T048, T049, and T050 can run in parallel.
- T055, T056, and T057 can run in parallel.
- Final validations T062 through T065 can run in parallel before T066.

## Parallel Example: User Story 1

```text
Task: "Add command tests for get incident --key, repeated --key, stdin -, deduplication, missing keys, and invalid keys in cmd/get_incident_test.go"
Task: "Add human, JSON, and keys-only incident view tests in cmd/cmd_views_get_test.go"
Task: "Add command contract expectations for get incident, aliases incidents and inc, and inherited get flags in cmd/command_contract_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational phases.
2. Complete User Story 1.
3. Validate direct incident lookup with targeted service, facade, and command tests.

### Incremental Delivery

1. Add direct incident lookup.
2. Add server-safe incident search filters.
3. Add message filtering and v8.8 compatibility handling.
4. Add creation-time filters.
5. Complete output modes and totals.
6. Update docs and verify existing process-instance incident workflows.

### Ralph Iteration Guidance

Each Ralph iteration should select the next unchecked task or a tightly coupled pair from the same phase. Avoid mixing service foundation work with documentation or broad validation in the same iteration unless all prior story tasks are complete.
