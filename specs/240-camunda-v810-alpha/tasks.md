# Tasks: Experimental Camunda 8.10 Alpha Support

**Input**: Design documents from `/specs/240-camunda-v810-alpha/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Tests are required by the feature specification and project constitution. Add focused tests before each corresponding implementation and run the repository-wide race-enabled test gate before completion.

**Organization**: Tasks are grouped by user story so each operator outcome can be implemented and validated independently after the shared generated-client foundation is complete.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files and has no dependency on another incomplete task in the same phase.
- **[Story]**: Maps work to a user story in `spec.md`; setup, foundational, and final tasks have no story label.
- Every task includes exact repository-relative file paths.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Capture the pre-feature state, repository rules, and pinned upstream prerequisites before generation or runtime changes begin.

- [ ] T001 Read `AGENTS.md`, `specs/ralph-implementation-rules.md`, `specs/240-camunda-v810-alpha/spec.md`, `specs/240-camunda-v810-alpha/plan.md`, `specs/240-camunda-v810-alpha/research.md`, `specs/240-camunda-v810-alpha/data-model.md`, `specs/240-camunda-v810-alpha/quickstart.md`, `specs/240-camunda-v810-alpha/contracts/generation-contract.md`, `specs/240-camunda-v810-alpha/contracts/service-compatibility.md`, and `specs/240-camunda-v810-alpha/contracts/version-selection.md`, then record any conflict or the no-conflict result in `specs/240-camunda-v810-alpha/progress.md`
- [ ] T002 Record the current default version, supported-version output, stable factory coverage, and fingerprints of `internal/clients/camunda/v87/`, `internal/clients/camunda/v88/`, and `internal/clients/camunda/v89/` in `specs/240-camunda-v810-alpha/progress.md`
- [ ] T003 Verify Go, Redocly, Python/PyYAML, and oapi-codegen prerequisites plus the `8.10.0-alpha4` tag-to-commit resolution, then record exact tool versions and commit `4a76f06c9df8ad0a6c64fe88b92c618c2ebf2ef6` in `specs/240-camunda-v810-alpha/progress.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish an isolated, reproducible alpha generated-client boundary required by every runtime story.

**CRITICAL**: No user story implementation can begin until the pinned alpha client and its generation guardrails pass.

- [ ] T004 Add failing target/tag/output mismatch, pre-write failure, and stable-tree protection tests for alpha generation in `api/refresh_clients_test.go`
- [ ] T005 [P] Add failing provenance schema, exact-tag/commit, ordered-mutation, generator-version, canonical-command, and deterministic-regeneration tests in `api/provenance_test.go`
- [ ] T006 Add `--target all|v810alpha` parsing and alpha-only orchestration that skips docs/auth/stable generation while preserving no-target behavior in `api/refresh-clients.sh`
- [ ] T007 Extend pinned product-spec fetching to accept an isolated temporary destination and emit resolved tag/commit metadata without relying on retained nested Git state in `api/1-fetch-camunda-product-v2-spec.sh`
- [ ] T008 Implement alpha-only bundle/mutation/generation through temporary files, validate the required tuple, fingerprint v8.7-v8.9 generated trees, and atomically publish only alpha outputs in `api/3-generate-clients-from-fetched-specs.sh`
- [ ] T009 Generate `internal/clients/camunda/v810alpha/camunda/client.gen.go` and `internal/clients/camunda/v810alpha/camunda/provenance.json` from the exact pinned source using the canonical command in `specs/240-camunda-v810-alpha/contracts/generation-contract.md`
- [ ] T010 Run `go test ./api -count=1`, compile the generated alpha package, regenerate a second time, verify no second-run diff or v8.7-v8.9 fingerprint change, and record evidence in `specs/240-camunda-v810-alpha/progress.md`

**Checkpoint**: The alpha client is reproducible, isolated, provenance-complete, and safe for version-specific adapters.

---

## Phase 3: User Story 1 - Select the Experimental Runtime Explicitly (Priority: P1) MVP

**Goal**: Operators can explicitly select `8.10-alpha`, reject stable-looking or unknown 8.10 inputs, retain the existing default, and execute representative workflows through native alpha adapters.

**Independent Test**: Configure `8.10-alpha`, construct the full c8volt API, execute representative generated-client-backed paths, verify canonical identity and output, reject plain `8.10` and full prerelease tags, and confirm omission still selects v8.8.

### Tests for User Story 1

> Write these tests first and confirm their new alpha expectations fail before implementation.

- [ ] T011 [P] [US1] Add canonical identity, accepted aliases, rejected stable-looking inputs, supported/implemented grouping, unchanged default, and explicit `C89_` prefix tests in `toolx/version_test.go` and `config/app_test.go`
- [ ] T012 [P] [US1] Add root help, hidden version-flag usage, human version output, additive JSON metadata, and 8.10 gateway/config matching tests in `cmd/root_test.go`, `cmd/version_test.go`, `cmd/bootstrap_errors_test.go`, and `cmd/config_test.go`
- [ ] T013 [P] [US1] Add `V810Alpha` factory-selection and unknown-version regressions for batch operations, cluster, and tenants in `internal/services/batchoperation/factory_test.go`, `internal/services/cluster/factory_test.go`, and `internal/services/tenant/factory_test.go`
- [ ] T014 [P] [US1] Add `V810Alpha` factory-selection and generated-type behavior tests for elements, incidents, and jobs in `internal/services/element/factory_test.go`, `internal/services/incident/factory_test.go`, `internal/services/job/factory_test.go`, `internal/services/element/v810alpha/service_test.go`, `internal/services/incident/v810alpha/service_test.go`, and `internal/services/job/v810alpha/service_test.go`
- [ ] T015 [P] [US1] Add `V810Alpha` factory-selection and generated-type behavior tests for process definitions, process instances, and resources in `internal/services/processdefinition/factory_test.go`, `internal/services/processinstance/factory_test.go`, `internal/services/resource/factory_test.go`, `internal/services/processdefinition/v810alpha/service_test.go`, `internal/services/processinstance/v810alpha/service_test.go`, and `internal/services/resource/v810alpha/service_test.go`
- [ ] T016 [P] [US1] Add `V810Alpha` factory-selection and generated-type behavior tests for user tasks and variables, including direct keyed user-task lookup, tenant validation, and no Tasklist fallback, in `internal/services/usertask/factory_test.go`, `internal/services/variable/factory_test.go`, `internal/services/usertask/v810alpha/service_test.go`, and `internal/services/variable/v810alpha/service_test.go`

### Implementation for User Story 1

- [ ] T017 [US1] Add `V810Alpha`, explicit alpha aliases, stable/experimental support collections, canonical string rendering, implemented discovery, and the explicit C89 fixture mapping in `toolx/version.go`
- [ ] T018 [US1] Preserve the v8.8 default and make configured `8.10-alpha` match an observed 8.10 gateway while retaining mismatch warnings for other major/minor versions in `config/app.go` and the config diagnostic implementation under `cmd/config.go`
- [ ] T019 [US1] Add separate stable/experimental build metadata and additive JSON fields while preserving existing envelope and field types in `cmd/version.go`, `cmd/root.go`, and `docsgen/main.go`
- [ ] T020 [P] [US1] Implement the native alpha batch-operation adapter and register its API assertion/factory case in `internal/services/batchoperation/v810alpha/`, `internal/services/batchoperation/api.go`, and `internal/services/batchoperation/factory.go`
- [ ] T021 [P] [US1] Implement the native alpha cluster adapter and register its API assertion/factory constructor in `internal/services/cluster/v810alpha/`, `internal/services/cluster/api.go`, and `internal/services/cluster/factory.go`
- [ ] T022 [P] [US1] Implement the native alpha element adapter and register its API assertion/factory case in `internal/services/element/v810alpha/`, `internal/services/element/api.go`, and `internal/services/element/factory.go`
- [ ] T023 [P] [US1] Implement the native alpha incident adapter and register its API assertion/factory case in `internal/services/incident/v810alpha/`, `internal/services/incident/api.go`, and `internal/services/incident/factory.go`
- [ ] T024 [P] [US1] Implement the native alpha job adapter and register its API assertion/factory case in `internal/services/job/v810alpha/`, `internal/services/job/api.go`, and `internal/services/job/factory.go`
- [ ] T025 [P] [US1] Implement the native alpha process-definition adapter and register its API assertion/factory case in `internal/services/processdefinition/v810alpha/`, `internal/services/processdefinition/api.go`, and `internal/services/processdefinition/factory.go`
- [ ] T026 [P] [US1] Implement the native alpha process-instance adapter and register its API assertion/factory case in `internal/services/processinstance/v810alpha/`, `internal/services/processinstance/api.go`, and `internal/services/processinstance/factory.go`
- [ ] T027 [P] [US1] Implement the native alpha resource adapter with existing deployment confirmation behavior and register its API assertion/factory case in `internal/services/resource/v810alpha/`, `internal/services/resource/api.go`, and `internal/services/resource/factory.go`
- [ ] T028 [P] [US1] Implement the native alpha tenant adapter and register its API assertion/factory case in `internal/services/tenant/v810alpha/`, `internal/services/tenant/api.go`, and `internal/services/tenant/factory.go`
- [ ] T029 [P] [US1] Implement the unified-only alpha user-task adapter using direct keyed retrieval and tenant-result validation, with no Tasklist client construction or fallback, in `internal/services/usertask/v810alpha/`, `internal/services/usertask/api.go`, and `internal/services/usertask/factory.go`
- [ ] T030 [P] [US1] Implement the native alpha variable adapter and register its API assertion/factory case in `internal/services/variable/v810alpha/`, `internal/services/variable/api.go`, and `internal/services/variable/factory.go`
- [ ] T031 [US1] Add full-client construction and representative alpha wiring coverage through all factories in `c8volt/client_test.go` without adding version branching to `c8volt/client.go`
- [ ] T032 [US1] Run `go test ./toolx ./config ./c8volt ./cmd ./internal/services/... -run 'V810|810Alpha|CamundaVersion|Factory|Client' -count=1` and record the MVP result in `specs/240-camunda-v810-alpha/progress.md`

**Checkpoint**: `8.10-alpha` is explicitly selectable, the default remains v8.8, and the complete client constructs with native alpha adapters.

---

## Phase 4: User Story 2 - Preserve Stable Runtime Behavior (Priority: P1)

**Goal**: Adding the alpha target leaves Camunda 8.7, 8.8, and 8.9 behavior, generated clients, fixture selection, defaults, and stable documentation claims unchanged.

**Independent Test**: Re-run stable version, factory, fixture, command, and documentation tests; compare generated-tree fingerprints with the Phase 1 baseline; and verify stable and experimental support are displayed separately.

### Tests for User Story 2

- [ ] T033 [P] [US2] Strengthen stable-order, stable-alias, default-version, and v8.7-v8.9 factory regression assertions alongside alpha cases in `toolx/version_test.go`, `config/app_test.go`, `internal/services/batchoperation/factory_test.go`, `internal/services/cluster/factory_test.go`, `internal/services/element/factory_test.go`, `internal/services/incident/factory_test.go`, `internal/services/job/factory_test.go`, `internal/services/processdefinition/factory_test.go`, `internal/services/processinstance/factory_test.go`, `internal/services/resource/factory_test.go`, `internal/services/tenant/factory_test.go`, `internal/services/usertask/factory_test.go`, and `internal/services/variable/factory_test.go`
- [ ] T034 [P] [US2] Add production and integration fixture-selection tests proving v8.7-v8.9 retain their prefixes and alpha deliberately resolves to C89 fixtures in `cmd/embed_test.go`, `integration/cli/deploy_embed_run_test.go`, and `integration/cli/harness_test.go`
- [ ] T035 [P] [US2] Add documentation metadata tests that prohibit undifferentiated stable/alpha claims and preserve stable-version wording in `cmd/root_test.go`, `cmd/version_test.go`, `cmd/command_contract_test.go`, and `docsgen/main_test.go`

### Implementation for User Story 2

- [ ] T036 [US2] Extend integration profile/version and embedded-fixture selection for the experimental target without changing stable profile lists in `integration/cli/harness_test.go` and `integration/cli/deploy_embed_run_test.go`
- [ ] T037 [US2] Compare current v8.7-v8.9 generated-tree fingerprints and diffs with the Phase 1 baseline, fail on collateral changes, and record the result in `specs/240-camunda-v810-alpha/progress.md`
- [ ] T038 [US2] Update authored stable/experimental support wording and existing command descriptions that currently stop at v8.9 in `README.md`, `cmd/root.go`, `cmd/get_processinstance.go`, `cmd/get_job.go`, `cmd/update.go`, `cmd/update_job.go`, and `cmd/update_processinstance.go`
- [ ] T039 [US2] Regenerate docs with `make docs-content`, then verify `docs/index.md`, `docs/cli/c8volt.md`, `docs/cli/c8volt_version.md`, `docs/cli/c8volt_get_process-instance.md`, `docs/cli/c8volt_get_job.md`, `docs/cli/c8volt_update.md`, `docs/cli/c8volt_update_job.md`, `docs/cli/c8volt_update_process-instance.md`, `docs/cli/c8volt_delete_process-definition.md`, and `docs/cli/c8volt_ops_purge_all-process-definitions.md` label `8.10-alpha` as experimental and pinned to `8.10.0-alpha4`
- [ ] T040 [US2] Run stable-version service, command, embed, config, and docs regression suites plus `git diff -- internal/clients/camunda/v87 internal/clients/camunda/v88 internal/clients/camunda/v89`, then record results in `specs/240-camunda-v810-alpha/progress.md`

**Checkpoint**: Stable runtime behavior and artifacts are unchanged, while alpha is visibly and explicitly experimental.

---

## Phase 5: User Story 3 - Get Clear Capability Outcomes (Priority: P2)

**Goal**: Every version-aware command family has a verified alpha path or a clear pre-mutation unsupported result, with no older generated client or removed component API hidden behind alpha selection.

**Independent Test**: Exercise representative alpha behavior for all eleven service families, verify history-safe deletion works for v8.9 and alpha, prove unsupported mutations stop before requests, and audit alpha source imports for a strict unified-client boundary.

### Tests for User Story 3

- [ ] T041 [P] [US3] Add named history-safe process-definition deletion capability tests for v8.7, v8.8, v8.9, alpha, unknown, and future stable-looking values in `toolx/camunda_capabilities_test.go`
- [ ] T042 [P] [US3] Add direct process-definition deletion and all-process-definitions purge tests proving v8.9 and alpha are accepted while v8.7/v8.8 fail before mutation in `cmd/delete_test.go`, `cmd/ops_purge_all_processdefinitions_test.go`, and `internal/services/ops/all_process_definitions_purge_test.go`
- [ ] T043 [P] [US3] Add a source-boundary regression test rejecting older Camunda generated-client imports and all Operate, Tasklist, or Administration SM imports under alpha adapters in `internal/services/version_client_boundary_test.go`
- [ ] T044 [P] [US3] Add alpha resource eventual-consistency tests for transient not-found retry, bounded failure, deployment visibility confirmation, and unchanged stable behavior in `internal/services/resource/v810alpha/service_test.go`
- [ ] T045 [P] [US3] Add representative alpha command execution tests across cluster, batch operations, elements, incidents, jobs, definitions, instances, resources, tenants, user tasks, and variables in `cmd/root_services_test.go`, `cmd/get_test.go`, `cmd/get_element_test.go`, `cmd/get_incident_test.go`, `cmd/get_job_test.go`, `cmd/get_processdefinition_test.go`, `cmd/get_processinstance_test.go`, `cmd/deploy_test.go`, `cmd/get_tenant_test.go`, and `cmd/capabilities_test.go` using `testx` fake servers

### Implementation for User Story 3

- [ ] T046 [US3] Implement named capability predicates without semantic-version ordering in `toolx/camunda_capabilities.go` and replace exact-v8.9 guards in `cmd/delete_processdefinition.go` and `internal/services/ops/all_process_definitions_purge.go`
- [ ] T047 [US3] Add bounded alpha resource read retry using existing service retry helpers while preserving deployment polling and operational confirmation in `internal/services/resource/v810alpha/service.go`
- [ ] T048 [US3] Update “8.9 or newer” command/help behavior to include alpha only where the named capability supports it in `cmd/delete_processdefinition.go` and `cmd/ops_purge_all_processdefinitions.go`
- [ ] T049 [US3] Run all eleven versioned service suites and the source-boundary audit with `go test ./internal/services/... -count=1`, then record each compatibility-matrix outcome in `specs/240-camunda-v810-alpha/progress.md`
- [ ] T050 [US3] Run representative alpha command tests with `go test ./cmd -run 'V810|810Alpha|DeleteProcessDefinition|AllProcessDefinitions|Get.*(Cluster|Job|Incident|Element|Tenant|Variable|Process)' -count=1` and record unsupported pre-mutation evidence in `specs/240-camunda-v810-alpha/progress.md`

**Checkpoint**: Every version-aware family has an explicit alpha outcome and no alpha path silently crosses into an older or removed client contract.

---

## Phase 6: User Story 4 - Reproduce and Audit Alpha Support (Priority: P3)

**Goal**: Maintainers can reproduce the alpha client from the pinned source, audit all provenance fields, and run a focused live smoke against Camunda 8.10.0-alpha4.

**Independent Test**: Regenerate twice from the canonical command, validate provenance and stable-tree isolation, then produce a smoke report containing authentication, topology, one read, one confirmed mutation, and one deterministic unsupported result.

### Tests for User Story 4

- [ ] T051 [P] [US4] Add integration harness tests for alpha profile selection, 8.10 gateway verification, C89 fixture reuse, report fields, and deterministic unsupported probing in `integration/cli/harness_test.go` and `integration/cli/deploy_embed_run_test.go`
- [ ] T052 [P] [US4] Add smoke-script argument, missing-environment, expected-version, failure-reporting, and no-secret-output tests in `integration/cli/c810alpha_smoke_test.go`

### Implementation for User Story 4

- [ ] T053 [US4] Document the canonical alpha generation command, exact provenance, prerequisites, stable-tree guarantees, and failure behavior in `api/README.md`
- [ ] T054 [US4] Implement the focused authentication/topology/read/confirmed-mutation/unsupported-result workflow and durable report in `integration/scripts/run-c810alpha-smoke.sh` using C89-compatible embedded fixtures
- [ ] T055 [US4] Run `integration/scripts/run-c810alpha-smoke.sh` against Camunda `8.10.0-alpha4`, record the report path and five outcomes in `specs/240-camunda-v810-alpha/progress.md`, or record the unavailable environment as an explicit release blocker
- [ ] T056 [US4] Re-run the canonical generation command, validate all seven provenance fields and exact tag/commit resolution, prove a clean second generation and unchanged stable fingerprints, and record the audit in `specs/240-camunda-v810-alpha/progress.md`

**Checkpoint**: Alpha support is reproducible and auditable, and live smoke evidence is either passing or explicitly blocks release readiness.

---

## Final Phase: Polish & Cross-Cutting Concerns

**Purpose**: Synchronize project guidance and generated docs, run all release gates, and close every measurable outcome with evidence.

- [ ] T057 [P] Run `gofmt` on all touched Go files under `api/`, `toolx/`, `config/`, `cmd/`, `c8volt/`, `internal/services/`, and `integration/`, then record the result in `specs/240-camunda-v810-alpha/progress.md`
- [ ] T058 [P] Run `git diff --check` and validate all JSON provenance files parse successfully, then record the result in `specs/240-camunda-v810-alpha/progress.md`
- [ ] T059 Update durable architecture and implementation guidance for the new `v810alpha` client/service boundary in `AGENTS.md` and `specs/ralph-implementation-rules.md` without adding temporary issue details
- [ ] T060 Run `make docs-content`, verify only generated documentation changes expected from authored metadata, and record the docs result in `specs/240-camunda-v810-alpha/progress.md`
- [ ] T061 Run the focused validation sequence from `specs/240-camunda-v810-alpha/quickstart.md` and record every pass, failure, and skipped external check in `specs/240-camunda-v810-alpha/progress.md`
- [ ] T062 Run the repository gate `make test` and record the race-enabled result in `specs/240-camunda-v810-alpha/progress.md`
- [ ] T063 Recompute fingerprints for `internal/clients/camunda/v87/`, `internal/clients/camunda/v88/`, and `internal/clients/camunda/v89/`, run the alpha import audit in `internal/services/version_client_boundary_test.go`, and record proof of no stable generated-client changes in `specs/240-camunda-v810-alpha/progress.md`
- [ ] T064 Map FR-001 through FR-022 and SC-001 through SC-010 to final automated/live evidence, document any unresolved blocker, and complete the handoff in `specs/240-camunda-v810-alpha/progress.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup and blocks all user stories because every alpha adapter requires the generated client.
- **User Story 1 (Phase 3)**: Depends on Foundational and is the MVP runtime slice.
- **User Story 2 (Phase 4)**: Depends on Foundational; recommended after US1 because version/help files overlap, but stable regression tests can be prepared in parallel.
- **User Story 3 (Phase 5)**: Depends on US1 native adapters; capability tests can start after Foundational.
- **User Story 4 (Phase 6)**: Depends on Foundational for generation auditing and on US1/US3 for a meaningful live smoke.
- **Polish (Final Phase)**: Depends on all desired user stories.

### User Story Dependencies

- **US1 Select the Experimental Runtime Explicitly**: Starts after Phase 2; no dependency on US2 or US4. It supplies native adapters required by US3.
- **US2 Preserve Stable Runtime Behavior**: Starts after Phase 2; its tests are independent, but authored version/help updates should merge after US1 to avoid same-file conflicts.
- **US3 Get Clear Capability Outcomes**: Requires US1 adapters for representative service and command execution; named capability work is otherwise independent.
- **US4 Reproduce and Audit Alpha Support**: Generation audit can start after Phase 2; live smoke requires US1 and US3 completion.

### Within Each User Story

- Add focused tests and confirm new expectations fail before implementation.
- Complete shared identity/client foundations before service adapters.
- Implement version-local conversion and behavior before factory selection is treated as complete.
- Run the story checkpoint and record evidence before advancing sequentially.
- Preserve public facade and command contracts unless the specification explicitly adds metadata.

## Parallel Opportunities

- T004 and T005 can run in parallel before shared script implementation.
- T011 through T016 can run in parallel because they target separate test ownership areas.
- T020 through T030 can run in parallel after T017-T019 and the generated client are stable; coordinate only shared `api.go` assertions if necessary.
- T033 through T035 can run in parallel after US1 test expectations settle.
- T041 through T045 can run in parallel because they target capability, boundary, resource, and command test surfaces separately.
- T051 and T052 can run in parallel before smoke-script implementation.
- T057 and T058 can run in parallel before final docs and repository validation.

## Parallel Example: User Story 1

```text
Task: "T020 [P] [US1] Implement the native alpha batch-operation adapter in internal/services/batchoperation/v810alpha/"
Task: "T021 [P] [US1] Implement the native alpha cluster adapter in internal/services/cluster/v810alpha/"
Task: "T024 [P] [US1] Implement the native alpha job adapter in internal/services/job/v810alpha/"
Task: "T028 [P] [US1] Implement the native alpha tenant adapter in internal/services/tenant/v810alpha/"
```

## Parallel Example: User Story 2

```text
Task: "T033 [P] [US2] Strengthen stable version and factory regression assertions in toolx/, config/, and internal/services/"
Task: "T034 [P] [US2] Add stable and alpha fixture-selection tests in cmd/ and integration/cli/"
Task: "T035 [P] [US2] Add stable/experimental documentation metadata tests in cmd/ and docsgen/"
```

## Parallel Example: User Story 3

```text
Task: "T041 [P] [US3] Add named capability tests in toolx/camunda_capabilities_test.go"
Task: "T043 [P] [US3] Add alpha generated-client source-boundary tests in internal/services/version_client_boundary_test.go"
Task: "T044 [P] [US3] Add alpha resource eventual-consistency tests in internal/services/resource/v810alpha/service_test.go"
```

## Parallel Example: User Story 4

```text
Task: "T051 [P] [US4] Add alpha profile, gateway, fixture, and report integration tests in integration/cli/"
Task: "T052 [P] [US4] Add smoke-script contract tests in integration/cli/c810alpha_smoke_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Complete Setup and the isolated generated-client foundation.
2. Add runtime identity, native alpha adapters, and full-client construction.
3. Stop and run the US1 checkpoint from T032.
4. Demonstrate canonical alpha selection, rejection of plain `8.10`, unchanged v8.8 default, and representative native alpha execution.

### Incremental Delivery

1. Deliver Phase 2 to establish reproducible alpha generation without runtime claims.
2. Deliver US1 as the selectable native-alpha MVP.
3. Deliver US2 to lock down stable-version non-regression and truthful support wording.
4. Deliver US3 to centralize capabilities and prove every versioned family has a safe alpha outcome.
5. Deliver US4 to close provenance and live-smoke release evidence.
6. Complete final formatting, documentation, architecture guidance, full tests, and traceability.

### Parallel Team Strategy

After the generated client is stable, service adapters in T020-T030 can be split across maintainers because each owns a separate package. A second stream can prepare US2 regression/docs tests, while a third prepares capability and source-boundary tests. Coordinate edits to `toolx/version.go`, `cmd/root.go`, `cmd/version.go`, factory `api.go` files, and `specs/240-camunda-v810-alpha/progress.md`.

---

## Notes

- Generated code in `internal/clients/camunda/v810alpha/camunda/client.gen.go` must come only from the guarded generation workflow.
- Do not hand-edit generated CLI documentation under `docs/cli/`; regenerate with `make docs-content`.
- Do not modify v8.7-v8.9 generated clients while producing alpha support.
- Keep `CurrentCamundaVersion` on `V88`.
- Keep alpha adapters free of v8.7-v8.9 generated imports and removed Operate, Tasklist, or Administration SM clients.
- A missing live alpha environment is an explicit release blocker, not a silent skip.
- Commit only after the current task or cohesive task group passes its closest validation.
