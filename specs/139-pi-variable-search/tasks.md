# Tasks: Native Process Instance Variable Search

**Input**: Design documents from `/specs/139-pi-variable-search/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are required by the feature specification and constitution. Story test tasks must be written before implementation and should fail until each story is implemented.

**Ralph Context**: Every Ralph implementation iteration for this feature MUST include `--implementation-context specs/ralph-implementation-rules.md`.

**Organization**: Tasks are grouped by user story so each story can be implemented and verified as an independent increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files or only adds tests/docs
- **[Story]**: Maps to the user story from [spec.md](./spec.md)
- Every task names exact repository paths

## Phase 1: Setup (Shared Discovery)

**Purpose**: Confirm current process-instance search ownership, generated request shapes, and docs surfaces before changing behavior.

- [x] T001 Inspect current `get pi` search flags, validation, and filter population in `cmd/get_processinstance.go` and adjacent `cmd/get_processinstance_*.go`
- [x] T002 [P] Inspect process facade filter mapping in `c8volt/process/api.go`, `c8volt/process/model.go`, `c8volt/process/convert.go`, and `c8volt/process/client.go`
- [x] T003 [P] Inspect current domain process-instance filters in `internal/domain/processinstance.go`
- [x] T004 [P] Inspect process-instance service interfaces and versioned search request construction in `internal/services/processinstance/api.go`, `internal/services/processinstance/v87/`, `internal/services/processinstance/v88/`, and `internal/services/processinstance/v89/`
- [x] T005 [P] Inspect generated Camunda variable/process-instance search request types in `internal/clients/camunda/v88/` and `internal/clients/camunda/v89/`
- [x] T006 [P] Inspect existing variable display and update parsing patterns in `cmd/get_processinstance*.go`, `cmd/update_processinstance*.go`, and `internal/services/processinstance/variables.go`
- [x] T007 [P] Inspect command contract and docs generation expectations in `cmd/command_contract_test.go`, `README.md`, `docsgen/`, and `docs/cli/`
- [x] T008 Record discovered ownership notes in `specs/139-pi-variable-search/progress.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared variable-filter representation and parser test scaffolding used by every user story.

**Critical**: No user story implementation should begin until this phase is complete.

- [x] T009 Add failing parser tests for variable clause splitting, quoting, arrays, and operator normalization in a focused `cmd/get_processinstance_variable_filter_test.go`
- [x] T010 [P] Add failing domain filter string and validation tests for variable filters in `internal/domain/processinstance_test.go`
- [x] T011 [P] Add failing process facade mapping tests for variable filters in `c8volt/process/client_test.go`
- [x] T012 Add version-neutral variable filter clause and filter set fields in `internal/domain/processinstance.go`
- [x] T013 Add public process variable filter models and conversions in `c8volt/process/model.go` and `c8volt/process/convert.go`
- [x] T014 Add command parser scaffolding for `--var-exists`, `--var`, and `--var-like` in a focused `cmd/get_processinstance_variable_filter.go`
- [x] T015 Wire parsed variable filters into `populatePISearchFilterOpts` and search validation in the appropriate `cmd/get_processinstance_*.go` files
- [x] T016 Run targeted compile and parser validation for `cmd`, `c8volt/process`, and `internal/domain`

**Checkpoint**: Shared variable filter grammar and model plumbing compile, parser tests can drive user-story implementation, and no remote behavior is required yet.

---

## Phase 3: User Story 1 - Find Instances By Variable Existence (Priority: P1) MVP

**Goal**: `get pi --var-exists` filters process instances by required variable existence on supported native versions.

**Independent Test**: `get pi --var-exists customerId` and `get pi --var-exists payload,email` apply existence clauses and preserve existing search behavior.

### Tests for User Story 1

- [x] T017 [P] [US1] Add command parser and validation tests for `--var-exists customerId` and `--var-exists payload,email` in `cmd/get_processinstance_variable_filter_test.go`
- [x] T018 [P] [US1] Add command execution tests for `get pi --var-exists` request flow in `cmd/get_processinstance_test.go`
- [x] T019 [P] [US1] Add v8.8 native request construction tests for `$exists=true` filters in `internal/services/processinstance/v88/service_test.go`
- [x] T020 [P] [US1] Add v8.9 native request construction tests for `$exists=true` filters in `internal/services/processinstance/v89/service_test.go`

### Implementation for User Story 1

- [x] T021 [US1] Register `--var-exists` flag and help text in `cmd/get_processinstance.go`
- [x] T022 [US1] Implement `--var-exists` parsing and validation in `cmd/get_processinstance_variable_filter.go`
- [x] T023 [US1] Map existence clauses through process facade and domain filters in `c8volt/process/convert.go` and `internal/domain/processinstance.go`
- [x] T024 [US1] Implement native existence request mapping for Camunda 8.8 in `internal/services/processinstance/v88/service.go` or a focused v88 filter file
- [x] T025 [US1] Implement native existence request mapping for Camunda 8.9 in `internal/services/processinstance/v89/service.go` or a focused v89 filter file
- [x] T026 [US1] Verify US1 with targeted tests for `cmd`, `c8volt/process`, `internal/domain`, `internal/services/processinstance/v88`, and `internal/services/processinstance/v89`

**Checkpoint**: User Story 1 is complete when existence filters work natively on 8.8/8.9 and existing searches without variable filters still pass.

---

## Phase 4: User Story 2 - Find Instances By Variable Equality (Priority: P2)

**Goal**: `get pi --var name=value` supports equality shorthand, repeated flags, and comma-separated clauses without splitting quoted commas.

**Independent Test**: `get pi --var 'status="approved"'` and `get pi --var 'status="canceled",payload="payload"'` apply equality clauses together.

### Tests for User Story 2

- [x] T027 [P] [US2] Add parser tests for equality shorthand, repeated `--var`, and quoted comma values in `cmd/get_processinstance_variable_filter_test.go`
- [x] T028 [P] [US2] Add command execution tests for equality filters in `cmd/get_processinstance_test.go`
- [x] T029 [P] [US2] Add v8.8 native request tests for `$eq` equality filters in `internal/services/processinstance/v88/service_test.go`
- [x] T030 [P] [US2] Add v8.9 native request tests for `$eq` equality filters in `internal/services/processinstance/v89/service_test.go`

### Implementation for User Story 2

- [x] T031 [US2] Register `--var` flag and equality examples in `cmd/get_processinstance.go`
- [x] T032 [US2] Implement `name=value` equality shorthand parsing in `cmd/get_processinstance_variable_filter.go`
- [x] T033 [US2] Preserve quoted values and comma-containing values in parser logic in `cmd/get_processinstance_variable_filter.go`
- [x] T034 [US2] Map equality clauses through process facade and domain filters in `c8volt/process/convert.go` and `internal/domain/processinstance.go`
- [x] T035 [US2] Implement native equality request mapping for Camunda 8.8 in `internal/services/processinstance/v88/`
- [x] T036 [US2] Implement native equality request mapping for Camunda 8.9 in `internal/services/processinstance/v89/`
- [x] T037 [US2] Verify US2 with targeted tests for `cmd`, `c8volt/process`, `internal/domain`, `internal/services/processinstance/v88`, and `internal/services/processinstance/v89`

**Checkpoint**: User Story 2 is complete when equality shorthand composes with existence filters and parser behavior is stable for quoted commas.

---

## Phase 5: User Story 3 - Search With Like Patterns (Priority: P3)

**Goal**: `get pi --var-like name=pattern` supports native wildcard patterns without adding implicit wildcards.

**Independent Test**: `get pi --var-like 'email=*@example.com'` preserves `*`, `?`, and escaped wildcard semantics in native search criteria.

### Tests for User Story 3

- [x] T038 [P] [US3] Add parser tests for `--var-like`, `*`, `?`, and escaped wildcard values in `cmd/get_processinstance_variable_filter_test.go`
- [x] T039 [P] [US3] Add command execution tests for like filters in `cmd/get_processinstance_test.go`
- [x] T040 [P] [US3] Add v8.8 native request tests for `$like` filters in `internal/services/processinstance/v88/service_test.go`
- [x] T041 [P] [US3] Add v8.9 native request tests for `$like` filters in `internal/services/processinstance/v89/service_test.go`

### Implementation for User Story 3

- [x] T042 [US3] Register `--var-like` flag and wildcard examples in `cmd/get_processinstance.go`
- [x] T043 [US3] Implement `--var-like` shorthand parsing in `cmd/get_processinstance_variable_filter.go`
- [x] T044 [US3] Preserve wildcard and escaped wildcard values in parser logic in `cmd/get_processinstance_variable_filter.go`
- [x] T045 [US3] Map like clauses through process facade and domain filters in `c8volt/process/convert.go` and `internal/domain/processinstance.go`
- [x] T046 [US3] Implement native like request mapping for Camunda 8.8 in `internal/services/processinstance/v88/`
- [x] T047 [US3] Implement native like request mapping for Camunda 8.9 in `internal/services/processinstance/v89/`
- [x] T048 [US3] Verify US3 with targeted tests for `cmd`, `internal/services/processinstance/v88`, and `internal/services/processinstance/v89`

**Checkpoint**: User Story 3 is complete when like filters preserve native wildcard behavior and compose with earlier variable filters.

---

## Phase 6: User Story 4 - Use Advanced Native Operators (Priority: P4)

**Goal**: `get pi --var name.$operator=value` supports `$eq`, `$neq`, `$exists`, `$in`, `$notIn`, `$like`, and `$notin` alias normalization.

**Independent Test**: Each advanced operator serializes into native search criteria, arrays are preserved, and invalid operators fail locally.

### Tests for User Story 4

- [x] T049 [P] [US4] Add parser tests for `$eq`, `$neq`, `$exists`, `$in`, `$notIn`, `$like`, and `$notin` in `cmd/get_processinstance_variable_filter_test.go`
- [x] T050 [P] [US4] Add parser tests for invalid operators, malformed booleans, and malformed arrays in `cmd/get_processinstance_variable_filter_test.go`
- [x] T051 [P] [US4] Add command execution tests for advanced operators in `cmd/get_processinstance_test.go`
- [x] T052 [P] [US4] Add v8.8 native request tests for advanced operators in `internal/services/processinstance/v88/service_test.go`
- [x] T053 [P] [US4] Add v8.9 native request tests for advanced operators in `internal/services/processinstance/v89/service_test.go`

### Implementation for User Story 4

- [x] T054 [US4] Add advanced operator parsing and `$notin` normalization in `cmd/get_processinstance_variable_filter.go`
- [x] T055 [US4] Add local validation for operator value shape in `cmd/get_processinstance_variable_filter.go`
- [x] T056 [US4] Extend domain/facade variable filter conversion for advanced operators in `internal/domain/processinstance.go` and `c8volt/process/convert.go`
- [x] T057 [US4] Implement native advanced operator mapping for Camunda 8.8 in `internal/services/processinstance/v88/`
- [x] T058 [US4] Implement native advanced operator mapping for Camunda 8.9 in `internal/services/processinstance/v89/`
- [x] T059 [US4] Verify US4 with targeted parser, command, and versioned service tests

**Checkpoint**: User Story 4 is complete when every requested advanced operator works or fails locally according to the contract.

---

## Phase 7: User Story 5 - Preserve Version And Tenant Contracts (Priority: P5)

**Goal**: Variable-search flags are native and supported on 8.8/8.9, explicitly unsupported on 8.7, and compatible with existing tenant-aware process-instance searches.

**Independent Test**: The same variable-search command succeeds through native request paths on 8.8/8.9, fails explicitly on 8.7, and preserves tenant filter behavior.

### Tests for User Story 5

- [x] T060 [P] [US5] Add 8.7 unsupported command tests for each new variable-search flag in `cmd/get_processinstance_test.go`
- [x] T061 [P] [US5] Add v8.7 service unsupported tests for variable filters in `internal/services/processinstance/v87/service_test.go`
- [x] T062 [P] [US5] Add tenant preservation tests for variable filters in `cmd/get_processinstance_test.go` or `internal/services/processinstance/v88/service_test.go`
- [x] T063 [P] [US5] Add regression tests proving existing 8.7 `get pi` searches without variable filters still behave as before in `cmd/get_processinstance_test.go`

### Implementation for User Story 5

- [x] T064 [US5] Add local version support validation for variable-search flags in `cmd/get_processinstance*.go`
- [x] T065 [US5] Add explicit 8.7 unsupported handling for domain filters with variable clauses in `internal/services/processinstance/v87/`
- [x] T066 [US5] Preserve tenant filter composition with variable filters in `internal/services/processinstance/v88/` and `internal/services/processinstance/v89/`
- [x] T067 [US5] Ensure no Operate fallback is used for variable-search paths in `internal/services/processinstance/v87/`, `v88/`, and `v89/`
- [x] T068 [US5] Verify US5 with targeted command and versioned service tests

**Checkpoint**: User Story 5 is complete when version and tenant behavior is explicit, tested, and free of Operate-backed variable-search fallback.

---

## Phase 8: User Story 6 - Understand The User-Facing Contract (Priority: P6)

**Goal**: Help text, command metadata, README examples, generated docs, and scopeKey wording explain the new variable-search contract.

**Independent Test**: Users can discover existence, equality, like, advanced operators, quoting, arrays, wildcard escaping, and `scopeKey` semantics from help/docs.

### Tests for User Story 6

- [ ] T069 [P] [US6] Add command contract tests for `--var-exists`, `--var`, and `--var-like` metadata in `cmd/command_contract_test.go`
- [ ] T070 [P] [US6] Add help/example regression tests for variable-search flags in `cmd/get_processinstance_test.go`
- [ ] T071 [P] [US6] Add docs or generated-content regression checks for variable-search examples in `docsgen/` or the nearest existing docs test path

### Implementation for User Story 6

- [ ] T072 [US6] Update `get pi` long help and examples for variable-search syntax in `cmd/get_processinstance.go`
- [ ] T073 [US6] Update command contract metadata expectations for variable-search flags in `cmd/command_contract_test.go`
- [ ] T074 [US6] Update README examples and user-facing guidance in `README.md`
- [ ] T075 [US6] Regenerate CLI docs and index content with `make docs-content` for `docs/cli/`, `docs/index.md`, and related generated docs assets
- [ ] T076 [US6] Verify `scopeKey` wording in `cmd/get_processinstance.go`, `README.md`, and `docs/cli/` describes direct definition scope only
- [ ] T077 [US6] Verify US6 with targeted command contract, help, docs-generation, and documentation checks

**Checkpoint**: User Story 6 is complete when user-facing docs and metadata fully describe the feature and match executable behavior.

---

## Phase 9: Polish & Cross-Cutting Validation

**Purpose**: Final cleanup, generated docs, task/progress updates, and repository validation.

- [ ] T078 [P] Run gofmt for changed Go files under `cmd/`, `c8volt/process/`, `internal/domain/`, and `internal/services/processinstance/`
- [ ] T079 Run targeted Go tests for changed packages under `cmd`, `c8volt/process`, `internal/domain`, and `internal/services/processinstance`
- [ ] T080 Run `make docs-content` from the repository root after command metadata and README changes
- [ ] T081 Run `make test` from the repository root before commit readiness
- [ ] T082 [P] Review [quickstart.md](./quickstart.md) against implemented behavior and update examples if flags or output changed during implementation
- [ ] T083 Update task completion and codebase pattern notes in `specs/139-pi-variable-search/tasks.md` and `specs/139-pi-variable-search/progress.md`
- [ ] T084 Review `git diff` to ensure changes are scoped to issue #139 artifacts, implementation, tests, README, and generated docs

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **US1 Existence (Phase 3)**: Depends on Foundational.
- **US2 Equality (Phase 4)**: Depends on Foundational and reuses US1 parser/model plumbing.
- **US3 Like (Phase 5)**: Depends on Foundational and can proceed after shared parser/model plumbing is stable.
- **US4 Advanced Operators (Phase 6)**: Depends on Foundational and benefits from US2/US3 parser behavior.
- **US5 Version/Tenant (Phase 7)**: Depends on at least one working variable-filter path and must complete before final docs claim support.
- **US6 Docs/Metadata (Phase 8)**: Depends on final flag names and behavior from US1-US5.
- **Polish (Phase 9)**: Depends on desired stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: MVP; establishes native existence search.
- **User Story 2 (P2)**: Builds on shared variable filter plumbing to add equality shorthand.
- **User Story 3 (P3)**: Adds like shorthand and wildcard behavior.
- **User Story 4 (P4)**: Adds advanced operators and invalid-input coverage.
- **User Story 5 (P5)**: Confirms version gates, tenant behavior, and no Operate fallback.
- **User Story 6 (P6)**: Finalizes help, docs, and metadata after behavior settles.

### Parallel Opportunities

- Setup inspection tasks T002-T007 can run in parallel.
- Foundational tests T009-T011 can run in parallel before implementation.
- Story test tasks marked [P] can run in parallel within a story.
- US2 and US3 can proceed in parallel after Foundational if parser file conflicts are coordinated.
- Documentation and final quickstart review can run in parallel with final validation after implementation stabilizes.

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational parser/model plumbing.
2. Complete US1 so `--var-exists` works on 8.8/8.9.
3. Validate US1 independently before adding value operators.

### Incremental Delivery

1. US1: variable existence filters.
2. US2: equality shorthand and comma parsing.
3. US3: like shorthand and wildcard preservation.
4. US4: advanced native operators and invalid-input behavior.
5. US5: version/tenant contract and no fallback.
6. US6: docs, help, and command metadata.
7. Polish: generated docs, broad tests, progress updates.

### Ralph Iteration Discipline

- Every Ralph launch or iteration must include `--implementation-context specs/ralph-implementation-rules.md`.
- Each Ralph work unit should complete one story or one validation slice only.
- Do not stage or commit until relevant validation passes.
- Commit subjects must use Conventional Commits and end with `#139`.
