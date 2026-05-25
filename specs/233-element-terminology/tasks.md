# Tasks: Element Terminology Standardization

**Input**: Design documents from `/specs/233-element-terminology/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are required by the feature specification and constitution. Story test tasks must be written before implementation and should fail until each story is implemented.

**Ralph Context**: Every Ralph implementation iteration for this feature MUST include `--implementation-context specs/ralph-implementation-rules.md`.

**Organization**: Tasks are grouped by user story so each story can be implemented and verified as an independent increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files or only adds tests/docs
- **[Story]**: Maps to the user story from [spec.md](./spec.md)
- Every task names exact repository paths

## Phase 1: Setup (Shared Discovery)

**Purpose**: Confirm the current public flow-node terminology surfaces before changing behavior.

- [x] T001 Inspect current incident command flags and validation in `cmd/get_incident.go`
- [x] T002 [P] Inspect current incident and process human renderers in `cmd/cmd_views_processinstance_incidents.go` and nearby `cmd/cmd_views_*.go`
- [x] T003 [P] Inspect public incident and process models/converters in `c8volt/incident/`, `c8volt/process/`, `c8volt/ops/`, and `c8volt/resource/`
- [x] T004 [P] Inspect internal domain and service mappings in `internal/domain/`, `internal/services/incident/`, and `internal/services/processinstance/`
- [x] T005 [P] Inspect ops incident filter reuse in `cmd/ops_repair_incident*.go` and `cmd/ops_purge_processinstances_with_incidents*.go`
- [x] T006 [P] Inspect command contract expectations in `cmd/command_contract_test.go`
- [x] T007 [P] Inspect documentation surfaces in `README.md`, `docs/cli/`, `docs/ops/`, and `docs/index.md`
- [x] T008 Record discovered ownership notes in `specs/233-element-terminology/progress.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish canonical naming in shared models and conversion boundaries so user-story work does not preserve old public fields.

**Critical**: No user story implementation should begin until this phase is complete.

- [x] T009 Add failing model/converter tests for canonical incident fields in `c8volt/incident/client_test.go`
- [x] T010 [P] Add failing model/converter tests for canonical process parent fields in `c8volt/process/client_test.go` or nearest existing process facade tests
- [x] T011 [P] Add failing domain/service conversion tests for canonical incident fields in `internal/services/incident/v87/`, `internal/services/incident/v88/`, and `internal/services/incident/v89/`
- [x] T012 Rename public incident filter/result fields from flow-node terms to element terms in `c8volt/incident/model.go`, `c8volt/incident/convert.go`, and `internal/domain/incident.go`
- [x] T013 Rename public process parent fields from flow-node terms to element terms in `c8volt/process/model.go`, `c8volt/process/convert.go`, `c8volt/ops/convert.go`, and `c8volt/resource/convert.go`
- [x] T014 Update incident service adapter conversions while keeping generated legacy names adapter-only in `internal/services/incident/v87/`, `internal/services/incident/v88/`, and `internal/services/incident/v89/`
- [x] T015 Run targeted compile validation for shared model changes in `c8volt/incident`, `c8volt/process`, `c8volt/ops`, `c8volt/resource`, and `internal/services/incident`

**Checkpoint**: Shared public and domain terminology is canonical, and generated-client legacy names are contained below adapter boundaries.

---

## Phase 3: User Story 1 - Filter Incidents With Element Terminology (Priority: P1) MVP

**Goal**: Incident filtering uses `--element-id` and `--element-instance-key`, while old public flags fail as unknown.

**Independent Test**: `get incident` accepts canonical filter flags, sends canonical filters through the facade/service path, and rejects legacy flags before remote work.

### Tests for User Story 1

- [x] T016 [P] [US1] Add command tests for `get incident --element-id` and `--element-instance-key` in `cmd/get_incident_test.go`
- [x] T017 [P] [US1] Add command tests proving `--flow-node-id` and `--fni-key` are unknown flags in `cmd/get_incident_test.go`
- [x] T018 [P] [US1] Add command contract tests for canonical incident flags and absence of old flags in `cmd/command_contract_test.go`
- [x] T019 [P] [US1] Add facade mapping tests for canonical incident filter fields in `c8volt/incident/client_test.go`
- [x] T020 [P] [US1] Add v8.8/v8.9 incident service tests for element filter construction or compatibility filtering in `internal/services/incident/v88/incidents_test.go` and `internal/services/incident/v89/incidents_test.go`

### Implementation for User Story 1

- [x] T021 [US1] Replace legacy incident filter flag variables and registrations with canonical flags in `cmd/get_incident.go`
- [x] T022 [US1] Update incident command validation, filter assembly, examples, and reset logic in `cmd/get_incident.go`
- [x] T023 [US1] Update command metadata expectations for canonical incident filter flags in `cmd/command_contract_test.go`
- [x] T024 [US1] Update incident facade and service filter field names used by `get incident` in `c8volt/incident/` and `internal/domain/incident.go`
- [x] T025 [US1] Update v8.8/v8.9 incident filter mapping and v8.7 compatibility mapping in `internal/services/incident/`
- [x] T026 [US1] Verify US1 with targeted tests for `cmd/get_incident_test.go`, `cmd/command_contract_test.go`, `c8volt/incident/client_test.go`, and `internal/services/incident/...`

**Checkpoint**: User Story 1 is complete when canonical incident filters work and old public flags are gone.

---

## Phase 4: User Story 2 - Show Canonical Incident Context (Priority: P2)

**Goal**: Incident JSON and human output use `elementId`, `elementInstanceKey`, `e:`, and `ei:` across affected direct, process-instance, walk, repair, and purge views.

**Independent Test**: Incident-bearing JSON and human output include canonical names/labels and exclude legacy names/labels.

### Tests for User Story 2

- [x] T027 [P] [US2] Add JSON output tests for canonical incident fields in `cmd/get_incident_test.go`
- [x] T028 [P] [US2] Add human row rendering tests for `e:` and `ei:` labels in `cmd/cmd_views_processinstance_incidents_test.go` or nearest existing view test file
- [x] T029 [P] [US2] Add `get pi --with-incidents` output tests for canonical incident context in `cmd/get_processinstance_test.go`
- [x] T030 [P] [US2] Add `walk pi --with-incidents` output tests for canonical incident context in `cmd/walk_processinstance_test.go`
- [x] T031 [P] [US2] Add ops repair/purge output regression tests for canonical incident context in `cmd/ops_repair_incident_test.go` and `cmd/ops_purge_processinstances_with_incidents_test.go`

### Implementation for User Story 2

- [x] T032 [US2] Rename incident JSON fields and converter outputs in `c8volt/incident/model.go`, `c8volt/incident/convert.go`, and `internal/domain/incident.go`
- [x] T033 [US2] Update process-instance incident detail conversions in `c8volt/process/convert.go`, `c8volt/ops/convert.go`, and `internal/domain/`
- [x] T034 [US2] Replace human labels `fn` and `fni` with `e` and `ei` in `cmd/cmd_views_processinstance_incidents.go`
- [x] T035 [US2] Update command output assertions and fixtures that consume incident JSON in `cmd/get_incident_test.go`, `cmd/get_processinstance_test.go`, `cmd/walk_processinstance_test.go`, and ops command tests
- [x] T036 [US2] Verify US2 with targeted tests for incident rendering, process-instance incident output, walk output, and ops output in `cmd/`

**Checkpoint**: User Story 2 is complete when all incident output surfaces use canonical element terminology.

---

## Phase 5: User Story 3 - Standardize Process Context Fields (Priority: P3)

**Goal**: Process-instance parent context uses `parentElementInstanceKey` everywhere in public models and JSON output.

**Independent Test**: Process-instance JSON and walk/process views expose `parentElementInstanceKey` and do not expose `parentFlowNodeInstanceKey`.

### Tests for User Story 3

- [x] T037 [P] [US3] Add process facade tests for `parentElementInstanceKey` in `c8volt/process/client_test.go` or nearest existing process facade tests
- [x] T038 [P] [US3] Add resource facade regression tests for renamed parent context in `c8volt/resource/client_test.go`
- [x] T039 [P] [US3] Add process-instance command JSON tests for `parentElementInstanceKey` in `cmd/get_processinstance_test.go`
- [x] T040 [P] [US3] Add walk command tests for canonical parent context in `cmd/walk_processinstance_test.go`
- [x] T041 [P] [US3] Add static or contract tests proving `parentFlowNodeInstanceKey` is absent from public command output contracts in `cmd/command_contract_test.go` or a focused `cmd/*_test.go`

### Implementation for User Story 3

- [x] T042 [US3] Rename public process parent fields in `c8volt/process/model.go` and `c8volt/process/convert.go`
- [x] T043 [US3] Rename parent context mappings in `c8volt/resource/convert.go` and `c8volt/ops/convert.go`
- [x] T044 [US3] Update internal domain process-instance parent context names in `internal/domain/` and affected service conversions in `internal/services/processinstance/`
- [x] T045 [US3] Update command views and JSON fixtures that render parent process context in `cmd/`
- [x] T046 [US3] Verify US3 with targeted tests for `c8volt/process`, `c8volt/resource`, `cmd/get_processinstance_test.go`, and `cmd/walk_processinstance_test.go`

**Checkpoint**: User Story 3 is complete when public process parent context uses only canonical element terminology.

---

## Phase 6: User Story 4 - Keep Legacy Names Behind Adapter Boundaries (Priority: P4)

**Goal**: Public contracts, docs, and tests contain no legacy flow-node names except generated-client and versioned adapter boundary allowances.

**Independent Test**: Repository searches and targeted tests prove old public flags, JSON fields, docs, and human labels are gone while generated clients and adapter mappings remain valid.

### Tests for User Story 4

- [x] T047 [P] [US4] Add ops repair incident flag contract tests for canonical filters in `cmd/ops_repair_incident_test.go`
- [x] T048 [P] [US4] Add ops purge incident flag contract tests for canonical filters in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [x] T049 [P] [US4] Add static regression checks for forbidden public strings in `cmd/command_contract_test.go` or a focused repository contract test file
- [x] T050 [P] [US4] Add adapter-boundary tests documenting allowed generated legacy mapping in `internal/services/incident/v87/`, `internal/services/incident/v88/`, and `internal/services/incident/v89/`
- [x] T051 [P] [US4] Add docs regression checks or update existing generated-doc assertions in `docsgen/` or `cmd/command_contract_test.go`

### Implementation for User Story 4

- [x] T052 [US4] Update ops repair incident filter flags, help, and request assembly in `cmd/ops_repair_incident*.go`
- [x] T053 [US4] Update ops purge process-instances-with-incidents filter flags, help, and request assembly in `cmd/ops_purge_processinstances_with_incidents*.go`
- [x] T054 [US4] Update README examples and incident wording in `README.md`
- [x] T055 [US4] Update source documentation for ops workflows in `docs/ops/repair-incident.md` and related non-generated docs
- [x] T056 [US4] Regenerate CLI docs and index content with `make docs-content` for `docs/cli/`, `docs/index.md`, and `docs/_site/assets/js/search-data.json`
- [x] T057 [US4] Run scoped legacy-term search and resolve public matches in `cmd`, `c8volt`, `internal/domain`, `README.md`, `docs/cli`, `docs/ops`, and `docs/index.md`
- [x] T058 [US4] Verify US4 with targeted ops, command contract, adapter-boundary, docs generation, and legacy-term checks

**Checkpoint**: User Story 4 is complete when legacy names appear only in generated clients, versioned adapter mappings, or historical spec artifacts outside the new public contract.

---

## Phase 7: Polish & Cross-Cutting Validation

**Purpose**: Final cleanup, generated docs, task/progress updates, and repository validation.

- [x] T059 [P] Run gofmt for changed Go files under `cmd/`, `c8volt/`, and `internal/`
- [x] T060 Run targeted Go tests for changed packages under `cmd/`, `c8volt/incident`, `c8volt/process`, `c8volt/ops`, `c8volt/resource`, `internal/services/incident`, and `internal/services/processinstance`
- [x] T061 Run `make docs-content` from the repository root after command metadata and README changes
- [x] T062 Run `make test` from the repository root before commit readiness
- [x] T063 [P] Review [quickstart.md](./quickstart.md) against implemented behavior and update examples if flags or output changed during implementation
- [x] T064 Update task completion and codebase pattern notes in `specs/233-element-terminology/tasks.md` and `specs/233-element-terminology/progress.md`
- [x] T065 Review `git diff` to ensure changes are scoped to issue #233 artifacts, implementation, tests, README, and generated docs

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **US1 Incident Filters (Phase 3)**: Depends on Foundational.
- **US2 Incident Output (Phase 4)**: Depends on Foundational; can start after US1 field naming settles.
- **US3 Process Parent Context (Phase 5)**: Depends on Foundational; can proceed independently of US2 once shared model names compile.
- **US4 Adapter Boundary, Ops, Docs (Phase 6)**: Depends on US1-US3 public naming changes so docs and static checks see final terminology.
- **Polish (Phase 7)**: Depends on desired stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: MVP; establishes public incident filter grammar.
- **User Story 2 (P2)**: Builds on shared incident model naming and output renderers.
- **User Story 3 (P3)**: Builds on process parent model naming and can be tested independently.
- **User Story 4 (P4)**: Final public-contract containment across ops, docs, generated outputs, and adapter boundaries.

### Parallel Opportunities

- Setup inspection tasks T002-T007 can run in parallel.
- Foundational tests T009-T011 can run in parallel before implementation.
- Story test tasks marked [P] can run in parallel within a story.
- US2 and US3 can proceed in parallel after Foundational if file conflicts are coordinated.
- Documentation and final quickstart review can run in parallel with final validation after implementation stabilizes.

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational model naming.
2. Complete US1 so canonical incident filter flags work and old flags fail.
3. Validate US1 independently before changing broader output and docs.

### Incremental Delivery

1. US1: canonical incident filters.
2. US2: canonical incident output.
3. US3: canonical process parent context.
4. US4: ops/docs/static containment.
5. Polish: generated docs, broad tests, progress updates.

### Ralph Iteration Discipline

- Every Ralph launch or iteration must include `--implementation-context specs/ralph-implementation-rules.md`.
- Each Ralph work unit should complete one story or one validation slice only.
- Do not stage or commit until relevant validation passes.
- Commit subjects must use Conventional Commits and end with `#233`.
