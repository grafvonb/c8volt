---

description: "Dependency-ordered implementation tasks for the all-tenants tenant override"
---

# Tasks: All-Tenants Tenant Override

**Input**: Design documents from `/specs/282-all-tenants-override/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/all-tenants.md](./contracts/all-tenants.md), [quickstart.md](./quickstart.md)

**Tests**: Tests are required by the feature specification and constitution. Write each story's test tasks before its implementation tasks and confirm the new assertions fail for the intended reason.

**Ralph Context**: Every Ralph implementation iteration for this feature MUST include `--implementation-context specs/ralph-implementation-rules.md` and complete only the current work unit.

**Organization**: Tasks are grouped by user story so each behavior can be implemented and verified as an independent increment. Commit subjects for issue-backed work must use Conventional Commits and end with `#282`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files and does not depend on another incomplete task in the same phase
- **[Story]**: Maps to the user story from [spec.md](./spec.md)
- Every task names exact repository paths

## Phase 1: Setup (Shared Discovery)

**Purpose**: Reconfirm current ownership and record the implementation baseline before changing the root command.

- [x] T001 Inspect root flag declaration, configuration resolution, and private tenant provenance in `cmd/root.go`, `cmd/root_config.go`, `cmd/cmd_tenant_context.go`, and `cmd/cmd_views_tenant_context.go`
- [x] T002 [P] Inventory concrete tenant destination call sites and pre-run side effects in `cmd/deploy_processdefinition.go`, `cmd/embed_deploy.go`, `cmd/run_processinstance.go`, and `cmd/ops_execute_smoketest.go`
- [x] T003 [P] Inspect command annotation, capability serialization, and human capability rendering patterns in `cmd/command_contract.go`, `cmd/capabilities.go`, `cmd/command_contract_test.go`, and `cmd/capabilities_test.go`
- [x] T004 [P] Inspect inherited root-flag parsing and generated documentation ownership in `integration/cli/examples_test.go`, `docsgen/`, `README.md`, and `docs/ops/`
- [x] T005 Record confirmed ownership, the four-command destination inventory, and reusable #283 warning patterns in `specs/282-all-tenants-override/progress.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the inherited flag and shared command-support classification used by all stories.

**Critical**: No user story implementation begins until the root flag and support resolver compile and their contract tests pass.

- [x] T006 [P] Add failing tests for root and subcommand placement, boolean parsing, default false, and explicit `--all-tenants=false` in `cmd/root_test.go`
- [x] T007 [P] Add failing tests for default `accepted` support and explicit `rejected_concrete_destination` annotation resolution in `cmd/command_contract_test.go`
- [x] T008 Register the command-line-only root persistent boolean `--all-tenants` flag without a Viper/config/environment binding in `cmd/root.go`
- [x] T009 Add the documented all-tenants support type, annotation setter, and single defaulting resolver in `cmd/command_contract.go`
- [x] T010 Run the focused foundational tests covering `cmd/root_test.go` and `cmd/command_contract_test.go`

**Checkpoint**: The inherited flag parses safely and one command annotation resolver can be consumed by runtime validation and capabilities; the flag does not yet alter tenant resolution.

---

## Phase 3: User Story 1 - Search Across Visible Tenants Explicitly (Priority: P1) MVP

**Goal**: Active `--all-tenants` clears any configured tenant after normalization, reuses the existing broadening warning, and leaves requests bounded by backend authorization.

**Independent Test**: Configure named tenants through base configuration, profile, and environment; run representative discovery commands with the flag before and after the subcommand; verify request bodies omit tenant filtering, human output shows the exact once-only warning in order, structured/protected output stays clean, and direct-key authorization behavior is unchanged.

### Tests for User Story 1

- [x] T011 [P] [US1] Add failing base/profile/environment, already-empty, explicit-false, absent-flag, and Camunda 8.7 post-normalization override tests in `cmd/root_config_test.go`
- [x] T012 [P] [US1] Add failing exact-warning, configured-tenant ordering, once-only, already-unfiltered, and existing `--tenant ""` regression tests in `cmd/cmd_views_tenant_context_test.go`
- [x] T013 [P] [US1] Add failing human, JSON, and YAML tenant-context isolation tests for active all-tenants in `cmd/config_test.go`
- [x] T014 [P] [US1] Add failing representative read/search request tests across Camunda 8.7, 8.8, 8.9, and 8.10 plus quiet/total-only/keys-only assertions in `cmd/get_processinstance_test.go` and `cmd/get_test.go`
- [x] T015 [P] [US1] Add failing durable-progress tests proving the all-tenants warning appears once before effective scope without leaking to protected output in `cmd/processinstance_mutation_progress_test.go` and `cmd/ops_progress_test.go`
- [x] T016 [US1] Extend private tenant override provenance with an all-tenants origin and exact broadening warning while preserving public `tenant.Context` in `cmd/cmd_tenant_context.go`
- [x] T017 [US1] Apply active all-tenants after `retrieveAndNormalizeConfig` by capturing the resolved configured tenant and setting the effective tenant to empty in `cmd/root_config.go`
- [x] T018 [US1] Invoke the post-normalization override before configuration enters command context or service installation in `cmd/root.go`
- [x] T019 [US1] Run focused US1 tests for `cmd/root_config_test.go`, `cmd/cmd_views_tenant_context_test.go`, `cmd/config_test.go`, `cmd/get_processinstance_test.go`, `cmd/get_test.go`, `cmd/processinstance_mutation_progress_test.go`, and `cmd/ops_progress_test.go`

**Checkpoint**: US1 is complete when all configured sources, including Camunda 8.7 normalization, resolve to the existing unfiltered request semantics and all output/authorization regressions pass.

---

## Phase 4: User Story 2 - Prevent Ambiguous Tenant Overrides (Priority: P2)

**Goal**: Active `--all-tenants` and any explicitly changed `--tenant`, including empty, fail as invalid input before command work.

**Independent Test**: Invoke a harmless command with named and empty explicit tenant values plus all-tenants and verify invalid-input exit 2 occurs before config loading or requests; verify explicit false, flag absence, and configured non-command-line tenants remain valid.

### Tests for User Story 2

- [ ] T020 [P] [US2] Add failing named/empty explicit tenant conflict, explicit-false, absent-flag, and configured-source acceptance tests in `cmd/root_test.go`
- [ ] T021 [P] [US2] Add failing subprocess assertions for invalid-input class, exit 2, silenced usage, and conflict precedence over missing configuration in `cmd/bootstrap_errors_test.go`

### Implementation for User Story 2

- [ ] T022 [US2] Add early root tenant-choice validation using `Flag.Changed`, `mutuallyExclusiveFlagsf`, and `silenceUsageForError` before configuration or service work in `cmd/root.go`
- [ ] T023 [US2] Add regression coverage that flag state resets between in-process executions without changing existing tenant precedence in `cmd/root_test.go` and `cmd/root_config_test.go`
- [ ] T024 [US2] Run focused US2 tests for `cmd/root_test.go`, `cmd/bootstrap_errors_test.go`, and `cmd/root_config_test.go`

**Checkpoint**: US2 is complete when every explicit tenant conflict fails locally and all invocations without an active conflict retain established behavior.

---

## Phase 5: User Story 3 - Protect Concrete Tenant Destinations (Priority: P3)

**Goal**: All commands that create or deploy into one tenant reject all-tenants before file, stdin, prompt, activity, report, dry-run plan, or remote work.

**Independent Test**: Run all four concrete-destination commands with active all-tenants, including smoke-test dry-run, and verify invalid input with zero side effects; run their existing named/default tenant cases without the flag and verify unchanged targeting.

### Tests for User Story 3

- [ ] T025 [P] [US3] Add failing deployment rejection tests using nonexistent/stdin inputs and request/activity spies in `cmd/deploy_test.go`
- [ ] T026 [P] [US3] Add failing embedded deployment and optional-run rejection tests proving no embedded file or request work in `cmd/embed_test.go`
- [ ] T027 [P] [US3] Add failing process-instance run rejection tests proving no stdin, prompt, activity, or creation request in `cmd/run_test.go`
- [ ] T028 [P] [US3] Add failing smoke-test normal/dry-run rejection tests proving no report, plan, prompt, activity, or request work in `cmd/ops_execute_smoke_test_test.go`

### Implementation for User Story 3

- [ ] T029 [P] [US3] Mark `deploy process-definition` as `rejected_concrete_destination` in `cmd/deploy_processdefinition.go`
- [ ] T030 [P] [US3] Mark `embed deploy` as `rejected_concrete_destination` in `cmd/embed_deploy.go`
- [ ] T031 [P] [US3] Mark `run process-instance` as `rejected_concrete_destination` in `cmd/run_processinstance.go`
- [ ] T032 [P] [US3] Mark `ops execute smoke-test` as `rejected_concrete_destination` in `cmd/ops_execute_smoketest.go`
- [ ] T033 [US3] Extend early root validation to reject active all-tenants through the shared command support resolver before any destination work in `cmd/root.go`
- [ ] T034 [US3] Add a future-safety inventory assertion tying all current concrete-destination leaves to rejection metadata in `cmd/command_contract_test.go`
- [ ] T035 [US3] Run focused US3 tests for `cmd/deploy_test.go`, `cmd/embed_test.go`, `cmd/run_test.go`, `cmd/ops_execute_smoke_test_test.go`, and `cmd/command_contract_test.go`

**Checkpoint**: US3 is complete when all four destination leaves reject before side effects and their flag-absent named/default destination behavior remains unchanged.

---

## Phase 6: User Story 4 - Discover and Automate the New Contract (Priority: P4)

**Goal**: Root/subcommand help, capabilities, examples, and generated/operator documentation agree on availability, visibility bounds, conflict behavior, and destination rejection.

**Independent Test**: Inspect root and representative command help, human and JSON capabilities, generated CLI pages, README, and operator guidance; verify every surface distinguishes `accepted` from `rejected_concrete_destination` and examples parse with the inherited boolean flag.

### Tests for User Story 4

- [ ] T036 [P] [US4] Add failing JSON and human capability tests for additive `allTenantsSupport` values and unchanged document version `v1` in `cmd/command_contract_test.go` and `cmd/capabilities_test.go`
- [ ] T037 [P] [US4] Add failing root, applicable-command, and four destination-command help assertions in `cmd/root_test.go`, `cmd/deploy_test.go`, `cmd/embed_test.go`, `cmd/run_test.go`, and `cmd/ops_execute_smoke_test_test.go`
- [ ] T038 [P] [US4] Add failing inherited boolean root-flag example recognition without value consumption in `integration/cli/examples_test.go`
- [ ] T039 [P] [US4] Add failing generated-page assertions for all-tenants syntax and destination restrictions in `docsgen/main_test.go`

### Implementation for User Story 4

- [ ] T040 [US4] Add `AllTenantsSupport` to `CommandCapability`, populate it from the shared resolver, and keep capability version `v1` in `cmd/command_contract.go`
- [ ] T041 [US4] Include all-tenants support state in the compact human capability summary in `cmd/capabilities.go`
- [ ] T042 [US4] Finalize root flag help/examples and concrete-destination restriction text in `cmd/root.go`, `cmd/deploy_processdefinition.go`, `cmd/embed_deploy.go`, `cmd/run_processinstance.go`, and `cmd/ops_execute_smoketest.go`
- [ ] T043 [US4] Register `all-tenants` as a non-value-consuming inherited root flag in `integration/cli/examples_test.go`
- [ ] T044 [P] [US4] Add supported syntax, exact warning, visibility boundary, mutual exclusion, destination restriction, and direct-key behavior to `README.md`
- [ ] T045 [P] [US4] Update all-tenants safety guidance in `docs/ops/index.md`, `docs/ops/analyse-slow-process-instances.md`, `docs/ops/execute-retention-policy.md`, `docs/ops/execute-smoke-test.md`, `docs/ops/purge-all-process-definitions.md`, `docs/ops/purge-orphan-process-instances.md`, `docs/ops/purge-process-instances-with-incidents.md`, `docs/ops/repair-incident.md`, and `docs/ops/repair-process-instance.md`
- [ ] T046 [US4] Regenerate `docs/cli/` and `docs/index.md` from command metadata and `README.md` with `make docs-content`
- [ ] T047 [US4] Run and record the 30-second discoverability review required by SC-007, including participant count and success rate, in `specs/282-all-tenants-override/progress.md`
- [ ] T048 [US4] Run focused US4 tests for `cmd/command_contract_test.go`, `cmd/capabilities_test.go`, `cmd/root_test.go`, the four destination test files, `integration/cli/examples_test.go`, and `docsgen/main_test.go`

**Checkpoint**: US4 is complete when executable behavior, machine capability metadata, help, generated docs, operator docs, and usability evidence all agree.

---

## Phase 7: Polish & Cross-Cutting Validation

**Purpose**: Enforce code ownership, documentation consistency, and repository-wide quality before implementation completion.

- [ ] T049 Inventory new declarations, add required intent comments, and run `gofmt` on touched files in `cmd/root.go`, `cmd/root_config.go`, `cmd/cmd_tenant_context.go`, `cmd/command_contract.go`, `cmd/capabilities.go`, the four destination command files, their `cmd/*_test.go` files, `integration/cli/examples_test.go`, and `docsgen/main_test.go`
- [ ] T050 Run targeted package tests for `./cmd`, `./integration/cli`, and `./docsgen` using the focused patterns from `specs/282-all-tenants-override/quickstart.md`
- [ ] T051 Run `make docs-content` and verify a second generation produces no unexplained drift in `README.md`, `docs/cli/`, `docs/index.md`, and `docs/ops/`
- [ ] T052 Run the `vet` target in `Makefile` with `make vet` and resolve all feature-related failures in touched Go files
- [ ] T053 Run the constitution-required full race-enabled `test` target in `Makefile` with `make test`
- [ ] T054 Review `git diff --check` and the full diff to confirm no changes under `c8volt/`, `internal/services/`, or `internal/clients/`, no public tenant-context provenance, and no authorization-bypass option
- [ ] T055 Update completed checkboxes and reusable codebase patterns in `specs/282-all-tenants-override/tasks.md` and `specs/282-all-tenants-override/progress.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; begins with repository inspection and creates `progress.md`.
- **Foundational (Phase 2)**: Depends on Setup and blocks all stories by establishing the inherited flag and shared support resolver.
- **US1 (Phase 3)**: Depends on Foundational; delivers the MVP tenant override and warning behavior.
- **US2 (Phase 4)**: Depends on Foundational and can proceed independently of US1 implementation after the flag exists, though priority order is recommended.
- **US3 (Phase 5)**: Depends on Foundational and can proceed independently of US1/US2 by using the shared flag and support resolver.
- **US4 (Phase 6)**: Depends on Foundational and the final US3 command classifications; documentation should wait for US1-US3 behavior to stabilize.
- **Polish (Phase 7)**: Depends on all desired user stories being complete.

### User Story Dependency Graph

```text
Setup -> Foundational -> US1 (MVP)
                    ├-> US2
                    └-> US3 -> US4
US1 + US2 + US3 + US4 -> Polish
```

### Within Each User Story

- Write the listed tests first and confirm they fail for the missing feature behavior.
- Implement only the owner-layer behavior named by the current task.
- Keep the flag command-line-only and keep backend/facade/service layers unchanged.
- Run the phase's focused verification before marking its checkpoint complete.
- Update `tasks.md` and `progress.md` in the same validated Ralph work-unit commit.

### Parallel Opportunities

- Setup inspections T002-T004 can run in parallel after T001 establishes context.
- Foundational tests T006-T007 can run in parallel; implementations T008-T009 then touch separate owner files.
- US1 test tasks T011-T015 touch separate test surfaces and can run in parallel before T016-T018.
- US2 test tasks T020-T021 can run in parallel.
- US3 tests T025-T028 and command annotations T029-T032 can each run in parallel within their respective groups.
- US4 tests T036-T039 and documentation source tasks T044-T045 can run in parallel when their prerequisites are satisfied.

## Parallel Examples

### User Story 1

```text
Task: "T011 add configuration-source and version override tests in cmd/root_config_test.go"
Task: "T012 add exact warning and ordering tests in cmd/cmd_views_tenant_context_test.go"
Task: "T013 add structured output isolation tests in cmd/config_test.go"
Task: "T014 add representative request and protected-mode tests in cmd/get_processinstance_test.go and cmd/get_test.go"
Task: "T015 add durable warning tests in cmd/processinstance_mutation_progress_test.go and cmd/ops_progress_test.go"
```

### User Story 2

```text
Task: "T020 add tenant-choice conflict and unchanged-behavior tests in cmd/root_test.go"
Task: "T021 add subprocess error classification and early-failure tests in cmd/bootstrap_errors_test.go"
```

### User Story 3

```text
Task: "T025 test deploy rejection in cmd/deploy_test.go"
Task: "T026 test embed rejection in cmd/embed_test.go"
Task: "T027 test run rejection in cmd/run_test.go"
Task: "T028 test smoke-test and dry-run rejection in cmd/ops_execute_smoke_test_test.go"
```

### User Story 4

```text
Task: "T036 test capability metadata in cmd/command_contract_test.go and cmd/capabilities_test.go"
Task: "T038 test inherited boolean example parsing in integration/cli/examples_test.go"
Task: "T039 test generated CLI pages in docsgen/main_test.go"
Task: "T044 update README.md"
Task: "T045 update the listed docs/ops/*.md playbooks"
```

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational.
2. Complete US1 so operators can explicitly clear configured tenant filtering across authenticated visibility.
3. Stop and run the US1 focused validation matrix before adding conflict, destination, or capability increments.

### Incremental Delivery

1. **US1**: Central override, exact warning, request/output/auth regression coverage.
2. **US2**: Ambiguous explicit tenant choices fail before work.
3. **US3**: Concrete destination operations fail before side effects.
4. **US4**: Capability/help/docs/usability surfaces expose the complete safety contract.
5. **Polish**: Full docs, vet, race suite, scope audit, and progress completion.

### Parallel Team Strategy

After Foundational is complete, separate contributors can prepare US1, US2, and US3 failing tests in their non-overlapping test files. Coordinate changes to `cmd/root.go` sequentially: US1 override wiring, then US2 conflict validation, then US3 destination validation. Begin final US4 documentation only after those runtime semantics and support annotations settle.

## Notes

- `[P]` tasks use distinct files or are read-only inspections with no dependency on an incomplete sibling task.
- Tests are mandatory and precede implementation within each story.
- Every new or materially modified function/test requires an intent-focused comment under `specs/ralph-implementation-rules.md`.
- Do not bind `--all-tenants` through Viper, add a durable config key, call `WithIgnoreTenant`, enumerate tenants client-side, or edit generated Camunda clients.
- Do not hand-edit `docs/cli/*` or `docs/index.md`; update source metadata/README and regenerate them.
- Do not mark tasks complete until their focused validation passes.
