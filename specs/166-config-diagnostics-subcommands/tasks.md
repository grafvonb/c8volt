# Tasks: Config Diagnostics Subcommands

**Input**: Design documents from `/specs/166-config-diagnostics-subcommands/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are required by the feature specification and constitution. Story test tasks should be written before implementation and should fail until the story implementation is complete.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files or only adds tests/docs
- **[Story]**: Maps to the user story from [spec.md](./spec.md)
- Every task names exact repository paths

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm current command behavior and identify reusable helper seams before changing the command surface.

- [x] T001 Inspect current `config show`, config bootstrap, and topology rendering behavior in `cmd/config_show.go`, `cmd/root.go`, `cmd/cmd_cli.go`, `cmd/get_cluster_topology.go`, and `cmd/cmd_views_cluster.go`
- [x] T002 [P] Inspect existing config and cluster command tests in `cmd/config_test.go` and `cmd/get_test.go`
- [x] T003 [P] Inspect current user-facing config docs in `README.md`, `docs/index.md`, `docs/cli/c8volt_config.md`, and `docs/cli/c8volt_config_show.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Extract shared behavior needed by all stories and prevent legacy flags from drifting away from new subcommands.

**Critical**: No user story implementation should begin until this phase is complete.

- [x] T004 Add shared config validation helper in `cmd/config_show.go` that validates a `config.Config` through the existing standard error/local precondition path
- [x] T005 Add shared config template rendering helper in `cmd/config_show.go` that renders the existing blank template and returns standard rendering errors
- [x] T006 Refactor `configShowCmd` in `cmd/config_show.go` to call the shared validation and template helpers without changing `config show`, `config show --validate`, or `config show --template` behavior
- [x] T007 [P] Add command-contract expectations for the new config subcommands in `cmd/command_contract_test.go`

**Checkpoint**: Shared helpers exist and legacy `config show` behavior is still the single compatibility baseline.

---

## Phase 3: User Story 1 - Preserve Config Show Compatibility (Priority: P1)

**Goal**: Existing `config show`, `config show --validate`, and `config show --template` continue working while helper extraction happens.

**Independent Test**: Run compatibility-focused command tests and verify sanitized show output, validation behavior, and template output remain unchanged.

### Tests for User Story 1

- [x] T008 [P] [US1] Add regression tests for sanitized `config show` output and warnings in `cmd/config_test.go`
- [x] T009 [P] [US1] Add regression tests for `config show --validate` valid and invalid outcomes in `cmd/config_test.go`
- [x] T010 [P] [US1] Add regression tests for `config show --template` output in `cmd/config_test.go`

### Implementation for User Story 1

- [x] T011 [US1] Preserve `config show` sanitized output and warning behavior while using shared helpers in `cmd/config_show.go`
- [x] T012 [US1] Preserve `config show --validate` exit and error behavior while using shared helpers in `cmd/config_show.go`
- [x] T013 [US1] Preserve `config show --template` rendering and mutually exclusive flag behavior in `cmd/config_show.go`
- [x] T014 [US1] Run `go test ./cmd -run 'TestConfigShow|TestConfigHelp' -count=1` and fix regressions in `cmd/config_show.go` or `cmd/config_test.go`

**Checkpoint**: User Story 1 is independently complete when legacy config show commands pass their focused tests.

---

## Phase 4: User Story 2 - Validate Configuration Directly (Priority: P2)

**Goal**: Add `c8volt config validate` as a dedicated validation command sharing legacy validation behavior.

**Independent Test**: Run `config validate` against valid and invalid config fixtures and compare outcomes with `config show --validate`.

### Tests for User Story 2

- [x] T015 [P] [US2] Add help/discovery tests for `config validate` in `cmd/config_test.go`
- [x] T016 [P] [US2] Add valid and invalid `config validate` command tests in `cmd/config_test.go`
- [x] T017 [P] [US2] Add equivalence tests comparing `config validate` and `config show --validate` outcomes in `cmd/config_test.go`

### Implementation for User Story 2

- [x] T018 [US2] Add `configValidateCmd` under `configCmd` in `cmd/config_validate.go`
- [x] T019 [US2] Wire `config validate` to load the effective config from context and call the shared validation helper in `cmd/config_validate.go`
- [x] T020 [US2] Update `configCmd` long text and examples for `config validate` in `cmd/config.go`
- [x] T021 [US2] Run `go test ./cmd -run 'TestConfig.*Validate|TestConfigHelp|TestCommandContract' -count=1` and fix regressions in `cmd/config_validate.go`, `cmd/config.go`, or `cmd/config_test.go`

**Checkpoint**: User Story 2 is independently complete when `config validate` validates through the same observable path as the legacy flag.

---

## Phase 5: User Story 3 - Render Configuration Template Directly (Priority: P3)

**Goal**: Add `c8volt config template` as a dedicated template command sharing legacy template output.

**Independent Test**: Run `config template` and `config show --template` and verify equivalent output and exit behavior.

### Tests for User Story 3

- [x] T022 [P] [US3] Add help/discovery tests for `config template` in `cmd/config_test.go`
- [x] T023 [P] [US3] Add template output equivalence tests for `config template` and `config show --template` in `cmd/config_test.go`

### Implementation for User Story 3

- [x] T024 [US3] Add `configTemplateCmd` under `configCmd` in `cmd/config_template.go`
- [x] T025 [US3] Wire `config template` to call the shared template rendering helper in `cmd/config_template.go`
- [x] T026 [US3] Update `configCmd` long text and examples for `config template` in `cmd/config.go`
- [x] T027 [US3] Run `go test ./cmd -run 'TestConfig.*Template|TestConfigHelp|TestCommandContract' -count=1` and fix regressions in `cmd/config_template.go`, `cmd/config.go`, or `cmd/config_test.go`

**Checkpoint**: User Story 3 is independently complete when dedicated and compatibility template commands render equivalent output.

---

## Phase 6: User Story 4 - Test Configured Camunda Connection (Priority: P4)

**Goal**: Add `c8volt config test-connection` to validate local config, log config source, retrieve topology, render topology output, and warn on major/minor version mismatches.

**Independent Test**: Run command tests for invalid config, no remote call on validation failure, connection success, connection failure, config source logging, version match, patch-only difference, and major/minor mismatch warning.

### Tests for User Story 4

- [x] T028 [P] [US4] Add `config test-connection` help/discovery tests in `cmd/config_test.go`
- [x] T029 [P] [US4] Add invalid configuration test proving `config test-connection` fails before remote topology retrieval in `cmd/config_test.go`
- [x] T030 [P] [US4] Add successful topology test proving `config test-connection` logs success and prints human-readable topology output in `cmd/config_test.go`
- [x] T031 [P] [US4] Add remote connection failure test proving the standard error path and non-zero exit in `cmd/config_test.go`
- [x] T032 [P] [US4] Add loaded config file and no-file-loaded `INFO` logging tests in `cmd/config_test.go`
- [x] T033 [P] [US4] Add version comparison tests for exact/patch-only match and major/minor warning in `cmd/config_test.go`

### Implementation for User Story 4

- [x] T034 [US4] Add a config source description helper or context value in `cmd/root.go` that preserves loaded config path versus no-file-loaded state for command use
- [x] T035 [US4] Add `configTestConnectionCmd` under `configCmd` in `cmd/config_test_connection.go`
- [x] T036 [US4] Implement `config test-connection` local validation, config-source `INFO` logging, topology retrieval through `NewCli`, and standard error handling in `cmd/config_test_connection.go`
- [x] T037 [US4] Reuse `renderClusterTopologyTree` for successful topology output in `cmd/config_test_connection.go`
- [x] T038 [US4] Add major/minor version normalization and warning behavior in `cmd/config_test_connection.go`
- [x] T039 [US4] Update `configCmd` long text and examples for `config test-connection` in `cmd/config.go`
- [x] T040 [US4] Run `go test ./cmd -run 'TestConfig.*Connection|TestGetClusterTopology|TestConfigHelp|TestCommandContract' -count=1` and fix regressions in `cmd/config_test_connection.go`, `cmd/root.go`, or `cmd/config_test.go`

**Checkpoint**: User Story 4 is independently complete when `config test-connection` proves local validation and remote topology connectivity with required logging and warning semantics.

---

## Phase 7: User Story 5 - Discover The Split Commands In Help And Docs (Priority: P5)

**Goal**: Help, README, and generated CLI docs document the new command family and legacy compatibility shortcuts.

**Independent Test**: Inspect help output and generated docs for `config`, `config show`, `config validate`, `config template`, and `config test-connection`.

### Tests for User Story 5

- [x] T041 [P] [US5] Add config command help tests covering new subcommands and compatibility shortcut text in `cmd/config_test.go`
- [x] T042 [P] [US5] Add generated-doc or docs-content expectation tests where applicable in `docsgen/` or `cmd/config_test.go`

### Implementation for User Story 5

- [x] T043 [US5] Update setup and configuration examples in `README.md`
- [x] T044 [US5] Update mirrored documentation content in `docs/index.md`
- [x] T045 [US5] Regenerate CLI documentation under `docs/cli/` with the repository docs generation command
- [x] T046 [US5] Update generated site/search artifacts under `docs/_site/` if the repository docs generation command refreshes them
- [x] T047 [US5] Run `make docs-content` and fix documentation generation issues
- [x] T048 [US5] Run `go test ./cmd -run 'TestConfigHelp|TestCommandContract' -count=1` and fix help or docs-related regressions

**Checkpoint**: User Story 5 is independently complete when the new command family is discoverable in help and docs while legacy flags remain documented.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final cleanup, validation, and repository-wide proof.

- [ ] T049 [P] Run `gofmt -w cmd/config.go cmd/config_show.go cmd/config_validate.go cmd/config_template.go cmd/config_test_connection.go cmd/config_test.go cmd/root.go cmd/command_contract_test.go`
- [ ] T050 Run `go test ./cmd -count=1` and fix any command-package regressions
- [ ] T051 Run `make test` and fix any repository validation failures
- [ ] T052 [P] Review `specs/166-config-diagnostics-subcommands/quickstart.md` against implemented behavior and update if command examples changed
- [ ] T053 Verify `git diff` contains only issue #166 implementation, docs, generated docs, and Speckit artifacts before commit

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on setup and blocks all user stories.
- **US1 (Phase 3)**: Depends on foundational helper extraction.
- **US2 (Phase 4)**: Depends on foundational validation helper and can run after US1 compatibility is protected.
- **US3 (Phase 5)**: Depends on foundational template helper and can run after US1 compatibility is protected.
- **US4 (Phase 6)**: Depends on foundational validation helper and can run after US2 defines direct validation behavior.
- **US5 (Phase 7)**: Depends on the command surface from US2, US3, and US4.
- **Polish (Phase 8)**: Depends on the desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: First user-visible safety slice; protects existing behavior.
- **User Story 2 (P2)**: Depends on US1/foundational validation helper.
- **User Story 3 (P3)**: Depends on US1/foundational template helper.
- **User Story 4 (P4)**: Depends on validation behavior from US2.
- **User Story 5 (P5)**: Depends on completed command names and help text from US2-US4.

### Parallel Opportunities

- T002 and T003 can run in parallel during setup.
- T007 can run in parallel with T004-T006 once expected command paths are known.
- US1 test tasks T008-T010 can be written in parallel.
- US2 test tasks T015-T017 can be written in parallel.
- US3 test tasks T022-T023 can be written in parallel.
- US4 test tasks T028-T033 can be written in parallel.
- US5 test tasks T041-T042 can be written in parallel.
- T049 and T052 can run in parallel after implementation is complete.

## Parallel Example: User Story 4

```text
Task: "Add invalid configuration test proving `config test-connection` fails before remote topology retrieval in cmd/config_test.go"
Task: "Add successful topology test proving `config test-connection` logs success and prints human-readable topology output in cmd/config_test.go"
Task: "Add remote connection failure test proving the standard error path and non-zero exit in cmd/config_test.go"
Task: "Add loaded config file and no-file-loaded INFO logging tests in cmd/config_test.go"
Task: "Add version comparison tests for exact/patch-only match and major/minor warning in cmd/config_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete User Story 1 to protect `config show` compatibility.
3. Stop and run the US1 targeted tests before adding new commands.

### Incremental Delivery

1. Add `config validate` after the shared validation helper is protected.
2. Add `config template` after the shared template helper is protected.
3. Add `config test-connection` after validation behavior is stable.
4. Add docs/generated docs once command names and help text are final.
5. Run targeted tests after each story, then `make test` before commit.

### Commit Guidance

Use Conventional Commit subjects and append the issue number as the final token, for example:

```text
feat(config): add diagnostics subcommands #166
```
