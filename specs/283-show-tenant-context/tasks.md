# Tasks: Effective Tenant Context Before Mutations

**Input**: Design documents from `/specs/283-show-tenant-context/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/tenant-context.md`, `quickstart.md`, and `specs/ralph-implementation-rules.md`

**Tests**: Automated tests are required by FR-020, the project constitution, and the Ralph implementation rules. Test tasks precede their corresponding implementation tasks.

**Organization**: Tasks are grouped by user story so each story produces a separately verifiable safety increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it changes different files and does not depend on another incomplete task in the same phase
- **[Story]**: Maps the task to a user story from `spec.md`
- Every task names the concrete repository path it changes or the progress artifact where validation is recorded

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish persistent implementation tracking for Ralph work units.

- [x] T001 Create `specs/283-show-tenant-context/progress.md` with artifact links, work-unit status, validation results, and codebase-pattern sections required by `specs/ralph-implementation-rules.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Implement the shared tenant-context value, evidence aggregation, rendering, and machine-contract plumbing used by every story.

**⚠️ CRITICAL**: No user story implementation begins until this phase passes its focused tests.

### Foundational tests

- [x] T002 [P] Add failing model and conversion tests for valid mode/filter combinations, required/omitted identifiers, copied slices, and stable JSON/YAML tags in `internal/domain/tenant_context_test.go` and `c8volt/tenant/context_test.go`
- [ ] T003 [P] Add failing accumulator tests for unique target counting, lexical tenant ordering, duplicate target suppression, unknown counts, and known/unknown merges in `internal/services/common/tenant_context_test.go`
- [ ] T004 [P] Add failing renderer and envelope tests for exact human wording, warning order, optional `tenantContext`, unchanged payload shape, quiet suppression, and keys-only silence in `cmd/cmd_views_tenant_context_test.go` and `cmd/command_contract_test.go`

### Foundational implementation

- [x] T005 Implement version-neutral and public tenant-context enums, values, validation constructors, warnings, and mechanical conversions in `internal/domain/tenant_context.go`, `c8volt/tenant/context.go`, and `c8volt/tenant/convert.go`
- [ ] T006 Implement deterministic per-key tenant evidence accumulation and snapshot merging without backend access in `internal/services/common/tenant_context.go`
- [ ] T007 Implement command-context attachment, operation-specific base-context construction, and human label/warning rendering in `cmd/cmd_tenant_context.go` and `cmd/cmd_views_tenant_context.go`
- [ ] T008 Add optional tenant context to the shared result envelope and propagate attached context through existing JSON result helpers without reshaping payloads in `cmd/command_contract.go` and `cmd/cmd_views_contract.go`
- [ ] T009 Run the focused domain, facade, accumulator, and command-contract tests and record results and reusable patterns in `specs/283-show-tenant-context/progress.md`

**Checkpoint**: Shared context can represent all four modes, aggregate deterministic evidence, render exact wording, and remain silent in protected output modes.

---

## Phase 3: User Story 1 - See Search And Selection Scope (Priority: P1) 🎯 MVP

**Goal**: Search-derived mutation previews and confirmations distinguish a named tenant filter from unfiltered discovery before execution.

**Independent Test**: Run equivalent process-instance selector previews with `tenant-a` and an empty tenant; verify `Tenant filter: tenant-a` versus `Tenant filter: none — resources from multiple tenants may be affected`, including when the unfiltered result happens to contain one tenant.

### Tests for User Story 1

- [ ] T010 [P] [US1] Add failing named/empty tenant selector and pre-confirmation ordering tests for cancel workflows in `cmd/cancel_processinstance_selector_test.go` and `cmd/cancel_processinstance_test.go`
- [ ] T011 [P] [US1] Add failing named/empty tenant selector and frozen-scope confirmation tests for delete workflows in `cmd/delete_processinstance_selector_test.go` and `cmd/delete_processinstance_test.go`
- [ ] T012 [P] [US1] Add failing human, JSON, quiet, and keys-only discovery-context tests for process-instance plan views in `cmd/cmd_views_processinstance_dryrun_test.go` and `cmd/processinstance_mutation_progress_test.go`

### Implementation for User Story 1

- [ ] T013 [US1] Attach discovery-mode tenant context to selector-based cancel planning and render it before preview or continuation confirmation in `cmd/cancel_processinstance_selector.go` and `cmd/cancel_processinstance.go`
- [ ] T014 [US1] Attach discovery-mode tenant context to selector-based delete planning and render it before the frozen-scope confirmation in `cmd/delete_processinstance_selector.go` and `cmd/delete_processinstance.go`
- [ ] T015 [US1] Carry discovery context through shared process-instance dry-run/progress views without changing candidate counts or stdout contracts in `cmd/cmd_views_processinstance_dryrun.go` and `cmd/processinstance_mutation_progress.go`
- [ ] T016 [US1] Run targeted process-instance selector and dry-run tests for the US1 independent criteria and record the commands and results in `specs/283-show-tenant-context/progress.md`

**Checkpoint**: Named and unfiltered process-instance selection scopes are independently visible and testable before mutation.

---

## Phase 4: User Story 2 - See Creation Target Tenant (Priority: P2)

**Goal**: Deploy and run commands identify the named or default creation tenant before making a backend mutation, without introducing prompts.

**Independent Test**: Exercise deploy and run with named and empty configuration; verify `Create in tenant: tenant-a` or `Create in tenant: <default>` is represented before execution in the selected output mode and no interactive prompt is added.

### Tests for User Story 2

- [ ] T017 [P] [US2] Add failing named/default creation-context, pre-call ordering, non-interactive, JSON-envelope, and quiet tests for deployment in `cmd/deploy_test.go` and `cmd/cmd_views_deploy_test.go`
- [ ] T018 [P] [US2] Add failing named/default creation-context, pre-call ordering, JSON-envelope, quiet, and keys-only tests for process-instance creation in `cmd/run_test.go` and `cmd/cmd_deploy_run_data_test.go`

### Implementation for User Story 2

- [ ] T019 [US2] Attach and render creation-mode context before process-definition deployment, including embedded deployment reuse, in `cmd/deploy_processdefinition.go` and `cmd/embed_deploy.go`
- [ ] T020 [US2] Attach and render creation-mode context before process-instance creation while preserving existing `TargetTenant()` request behavior in `cmd/run_processinstance.go`
- [ ] T021 [US2] Include creation context in deploy/run structured results without changing deployment/run payloads or key streams in `cmd/cmd_views_deploy.go` and `cmd/cmd_deploy_run_data.go`
- [ ] T022 [US2] Run targeted deploy and run tests for the US2 independent criteria and record the commands and results in `specs/283-show-tenant-context/progress.md`

**Checkpoint**: Operators and automation can identify the creation tenant before deploy/run mutation with no interaction change.

---

## Phase 5: User Story 3 - Understand Explicit-Key Tenant Behavior (Priority: P3)

**Goal**: Explicit-key operations state that the configured filter is not applied and show actual resource tenants already available in their frozen plans, including configured/resolved mismatches.

**Independent Test**: Preview explicit targets configured for `tenant-a` that resolve to `tenant-b`; verify backend-authorized behavior is unchanged, the filter is reported as not applied, and `tenant-b` is shown without a local mismatch rejection.

### Tests for User Story 3

- [ ] T023 [P] [US3] Add failing service tests proving PI dry-run plans retain known tenant evidence from existing traversal data and mark legacy key-only metadata unknown without extra requests in `internal/services/processinstance/dryrun_test.go`
- [ ] T024 [P] [US3] Add failing facade conversion tests for PI plan tenant evidence and slice-copy isolation in `c8volt/process/client_test.go` and `c8volt/process/model_test.go`
- [ ] T025 [P] [US3] Add failing explicit-key, known/unknown resource tenant, mismatch, and unchanged `IgnoreTenant` tests for PI cancel/delete/resolve/update in `cmd/cancel_processinstance_test.go`, `cmd/delete_processinstance_test.go`, `cmd/resolve_processinstance_test.go`, and `cmd/update_processinstance_test.go`
- [ ] T026 [P] [US3] Add failing explicit-key mismatch and actual-tenant tests for process-definition deletion and job update plans/results in `cmd/delete_test.go`, `cmd/update_job_plan_test.go`, `cmd/update_job_test.go`, and `cmd/update_job_outcome_test.go`

### Implementation for User Story 3

- [ ] T027 [US3] Preserve resolved PI tenant evidence in domain dry-run plans and aggregate traversal-chain targets before metadata is discarded in `internal/domain/processinstance_traversal.go` and `internal/services/processinstance/dryrun.go`
- [ ] T028 [US3] Expose PI plan tenant evidence through public models and mechanical facade conversion in `c8volt/process/api.go`, `c8volt/process/dryrun.go`, and `c8volt/process/convert.go`
- [ ] T029 [US3] Attach explicit-key semantics and resolved tenant evidence to PI cancel, delete, and resolve previews and confirmations without changing `IgnoreTenant` in `cmd/cancel_processinstance.go`, `cmd/delete_processinstance.go`, and `cmd/resolve_processinstance.go`
- [ ] T030 [US3] Preserve available variable tenant metadata in the PI update plan and render explicit-key context without enrichment calls in `cmd/update_processinstance_variables.go`, `cmd/update_processinstance.go`, and `cmd/cmd_views_processinstance_update.go`
- [ ] T031 [US3] Aggregate existing process-definition plan-item tenant IDs and render explicit-key context while preserving backend-authorized keys in `c8volt/resource/model.go`, `cmd/delete_processdefinition.go`, and `cmd/cmd_views_processdefinition.go`
- [ ] T032 [US3] Derive actual tenant evidence from the already-loaded current job and render it in update preview, confirmation, and result views in `cmd/update_job_plan.go`, `cmd/update_job.go`, `cmd/update_job_outcome.go`, and `cmd/cmd_views_job.go`
- [ ] T033 [US3] Run targeted service, facade, PI, process-definition, and job explicit-key tests for the US3 independent criteria and record the results in `specs/283-show-tenant-context/progress.md`

**Checkpoint**: Explicit-key paths truthfully separate configured context from actual tenant evidence and preserve backend authorization.

---

## Phase 6: User Story 4 - Receive Cross-Tenant Mutation Warnings (Priority: P4)

**Goal**: Frozen mutation plans warn prominently and deterministically when known targets span tenants and warn non-blockingly when any target tenant is unknown.

**Independent Test**: Resolve plans containing repeated `tenant-a`/`tenant-b` values and unknown targets; verify one sorted cross-tenant warning and one unknown warning appear in preview and confirmation, while a single-known-tenant plan has no cross-tenant warning.

### Tests for User Story 4

- [ ] T034 [P] [US4] Add failing PI plan/page merge tests for stable tenant deduplication, unique unknown counting, cross-plus-unknown warnings, and no enrichment calls in `internal/services/processinstance/dryrun_test.go` and `cmd/processinstance_mutation_progress_test.go`
- [ ] T035 [P] [US4] Add failing process-definition impact tests for duplicate tenants, multi-tenant cancellation subplans, unknown items, and compact confirmation warning order in `internal/services/processdefinition/delete_test.go` and `cmd/delete_test.go`

### Implementation for User Story 4

- [ ] T036 [US4] Merge tenant evidence by unique affected key across PI mutation pages and confirmation boundaries in `internal/services/processinstance/dryrun.go`, `internal/services/processinstance/traversal/result.go`, and `cmd/processinstance_mutation_progress.go`
- [ ] T037 [US4] Render sorted resolved tenant summaries plus coexisting cross-tenant and unknown warnings in dry-run output and destructive confirmations in `cmd/cmd_views_processinstance_dryrun.go`, `cmd/cancel_processinstance.go`, `cmd/delete_processinstance.go`, and `cmd/resolve_processinstance.go`
- [ ] T038 [US4] Merge process-definition item and nested cancellation-plan evidence and render the same warning contract in `internal/services/processdefinition/delete.go`, `c8volt/resource/convert.go`, and `cmd/delete_processdefinition.go`
- [ ] T039 [US4] Run targeted cross-tenant and unknown-metadata service/command tests for the US4 independent criteria and record the results in `specs/283-show-tenant-context/progress.md`

**Checkpoint**: Cross-tenant and unknown-target risks are deterministic, prominent, and non-blocking across representative frozen mutation plans.

---

## Phase 7: User Story 5 - Trust Tenant Context Across Safety Surfaces (Priority: P5)

**Goal**: Configuration diagnostics, all named mutation families, operations preflight, final structured output, and audit reports use the same tenant semantics without breaking JSON, YAML, quiet, keys-only, or report contracts.

**Independent Test**: Exercise representative configuration, PI, process-definition, job, retention, purge, repair, smoke-test, deploy, and run workflows across named, empty, explicit, cross-tenant, and unknown cases; verify consistent context before mutation and truthful final/audit evidence.

### Tests for User Story 5

- [ ] T040 [P] [US5] Add failing named/empty configuration-context tests for validate, sanitized show YAML, and test-connection human/JSON output in `cmd/config_test.go` and `config/config_test.go`
- [ ] T041 [P] [US5] Add failing internal/public ops model and conversion tests for the common nested object, copied slices, deprecated `tenantId`, and unfiltered omission in `internal/domain/ops_progress_test.go`, `c8volt/ops/client_test.go`, and `c8volt/ops/model_test.go`
- [ ] T042 [P] [US5] Add failing no-extra-call tenant aggregation tests for retention, all-definition purge, orphan purge, incident purge, repair, and smoke test services in `internal/services/ops/retention_policy_test.go`, `internal/services/ops/all_process_definitions_purge_test.go`, `internal/services/ops/orphan_purge_test.go`, `internal/services/ops/incident_purge_test.go`, `internal/services/ops/repair_test.go`, and `internal/services/ops/smoke_test_test.go`
- [ ] T043 [P] [US5] Add failing preflight, confirmation, JSON/Markdown report, unfiltered-not-default, quiet, and keys-only tests across ops workflows in `cmd/ops_execute_retention_policy_test.go`, `cmd/ops_purge_all_processdefinitions_test.go`, `cmd/ops_purge_orphan_processinstances_test.go`, `cmd/ops_purge_processinstances_with_incidents_test.go`, `cmd/ops_repair_incident_test.go`, `cmd/ops_repair_processinstance_test.go`, `cmd/ops_execute_smoke_test_test.go`, `cmd/ops_report_json_test.go`, and `cmd/ops_report_markdown_test.go`

### Implementation for User Story 5

- [ ] T044 [US5] Add configuration-mode context to validation and connection diagnostics and embed the common object in sanitized config YAML without corrupting the document in `cmd/config_diagnostics.go`, `cmd/config_show.go`, `cmd/config_validate.go`, `cmd/config_test_connection.go`, `cmd/cmd_views_config_test_connection.go`, and `config/config.go`
- [ ] T045 [US5] Add tenant context to ops preflight/report domain and public models and map it mechanically across the facade boundary in `internal/domain/ops_progress.go`, `internal/domain/ops_retention_policy.go`, `internal/domain/ops_all_process_definitions_purge.go`, `internal/domain/ops_orphan_purge.go`, `internal/domain/ops_incident_purge.go`, `internal/domain/ops_repair.go`, `internal/domain/ops_smoke_test_model.go`, `c8volt/ops/progress_model.go`, `c8volt/ops/model.go`, and `c8volt/ops/convert.go`
- [ ] T046 [US5] Aggregate tenant evidence from already-frozen ops workflow plans/results without additional backend requests in `internal/services/ops/retention_policy.go`, `internal/services/ops/all_process_definitions_purge.go`, `internal/services/ops/orphan_purge.go`, `internal/services/ops/incident_purge.go`, `internal/services/ops/repair.go`, and `internal/services/ops/smoke_test_service.go`
- [ ] T047 [US5] Attach discovery, explicit-key, or creation semantics to ops preflight and report enrichment while removing unfiltered `ViewTenant()` misuse in `cmd/ops_progress_render.go`, `cmd/ops_execute_retention_policy.go`, `cmd/ops_purge_all_processdefinitions.go`, `cmd/ops_purge_orphan_processinstances.go`, `cmd/ops_purge_processinstances_with_incidents.go`, `cmd/ops_repair.go`, `cmd/ops_repair_incident.go`, `cmd/ops_repair_processinstance.go`, and `cmd/ops_execute_smoketest.go`
- [ ] T048 [US5] Render the common context and warnings in affected ops human summaries and JSON/Markdown audits while retaining truthful optional legacy fields and v1 schema identifiers in `cmd/cmd_views_ops_execute_retention_policy.go`, `cmd/cmd_views_ops_purge_all_processdefinitions.go`, `cmd/cmd_views_ops_purge_orphan_processinstances.go`, `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`, `cmd/cmd_views_ops_repair.go`, `cmd/cmd_views_ops_execute_smoketest.go`, `cmd/ops_report_purge_all_processdefinitions.go`, `cmd/ops_report_purge_processinstances_with_incidents.go`, and `cmd/ops_report_repair.go`
- [ ] T049 [US5] Extend representative cross-family machine-contract assertions for one JSON document, common object placement, unchanged payloads, quiet suppression, and exact keys-only stdout in `cmd/cmd_json_assertions_test.go`, `cmd/command_contract_test.go`, and `cmd/ops_contract_test.go`
- [ ] T050 [US5] Verify Camunda 8.7 normalization versus 8.8–8.10 unfiltered behavior through configuration and service tests without modifying generated clients in `cmd/root_config_test.go`, `config/config_test.go`, and `internal/services/common/tenant_context_test.go`
- [ ] T051 [US5] Run the quickstart cross-family acceptance commands and record output-contract and no-extra-backend-call results in `specs/283-show-tenant-context/progress.md`

**Checkpoint**: Every required safety surface reports one consistent, operation-specific tenant meaning and preserves its established machine/report contracts.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Align documentation, generated artifacts, code ownership, and repository-wide validation.

- [ ] T052 [P] Update affected Cobra `Long` text and examples with named discovery, unfiltered discovery, default creation, explicit-key, cross-tenant, and unknown semantics in `cmd/config_show.go`, `cmd/config_validate.go`, `cmd/config_test_connection.go`, `cmd/cancel_processinstance.go`, `cmd/delete_processinstance.go`, `cmd/resolve_processinstance.go`, `cmd/update_processinstance.go`, `cmd/delete_processdefinition.go`, `cmd/update_job.go`, `cmd/deploy_processdefinition.go`, `cmd/run_processinstance.go`, `cmd/ops_execute_retention_policy.go`, `cmd/ops_execute_smoketest.go`, `cmd/ops_purge_all_processdefinitions.go`, `cmd/ops_purge_orphan_processinstances.go`, `cmd/ops_purge_processinstances_with_incidents.go`, `cmd/ops_repair_incident.go`, and `cmd/ops_repair_processinstance.go`
- [ ] T053 [P] Update operator guidance for tenant semantics and protected output modes in `README.md`, `docs/ops/execute-retention-policy.md`, `docs/ops/execute-smoke-test.md`, `docs/ops/purge-all-process-definitions.md`, `docs/ops/purge-orphan-process-instances.md`, `docs/ops/purge-process-instances-with-incidents.md`, `docs/ops/repair-incident.md`, and `docs/ops/repair-process-instance.md`
- [ ] T054 Regenerate CLI documentation with `make docs-content`, review affected pages in `docs/cli/c8volt_config_show.md`, `docs/cli/c8volt_config_validate.md`, `docs/cli/c8volt_config_test-connection.md`, `docs/cli/c8volt_cancel_process-instance.md`, `docs/cli/c8volt_delete_process-instance.md`, `docs/cli/c8volt_delete_process-definition.md`, `docs/cli/c8volt_resolve_process-instance.md`, `docs/cli/c8volt_update_process-instance.md`, `docs/cli/c8volt_update_job.md`, `docs/cli/c8volt_deploy_process-definition.md`, `docs/cli/c8volt_run_process-instance.md`, `docs/cli/c8volt_ops_execute_retention-policy.md`, `docs/cli/c8volt_ops_execute_smoke-test.md`, `docs/cli/c8volt_ops_purge_all-process-definitions.md`, `docs/cli/c8volt_ops_purge_orphan-process-instances.md`, `docs/cli/c8volt_ops_purge_process-instances-with-incidents.md`, `docs/cli/c8volt_ops_repair_incident.md`, `docs/cli/c8volt_ops_repair_process-instance.md`, and `docs/index.md`, then record the command in `specs/283-show-tenant-context/progress.md`
- [ ] T055 Inventory new and modified declarations for command-file cohesion, verify no generated client or tenant-selection request changed, run `gofmt`/`git diff --check`, and record the audit in `specs/283-show-tenant-context/progress.md`
- [ ] T056 Run `make vet` and constitution-required `make test`, resolve all failures, and record final validation and completion status in `specs/283-show-tenant-context/progress.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on T001 and blocks every user story.
- **User Story 1 (Phase 3)**: Depends on Phase 2; this is the MVP.
- **User Story 2 (Phase 4)**: Depends on Phase 2 and can proceed independently of US1.
- **User Story 3 (Phase 5)**: Depends on Phase 2 and can proceed independently of US1/US2.
- **User Story 4 (Phase 6)**: Depends on the resolved-plan evidence introduced by US3 (T027–T032).
- **User Story 5 (Phase 7)**: Depends on the shared contract and completed representative semantics from US1–US4 so it can enforce cross-surface consistency.
- **Polish (Phase 8)**: Depends on all desired user stories; T054 depends on T052, and T056 follows T054–T055.

### User Story Dependency Graph

```text
Setup → Foundation ─┬→ US1 (P1, MVP) ─────────┐
                    ├→ US2 (P2) ──────────────┤
                    └→ US3 (P3) → US4 (P4) ──┼→ US5 (P5) → Polish
                                              ┘
```

### Within Each User Story

- Write the listed tests first and confirm they fail for the expected missing behavior.
- Add domain fields before service aggregation, then facade conversion, then command rendering.
- Use only resource data already present in the frozen plan; never add enrichment calls for tenant reporting.
- Render context before mutation/confirmation and reuse the frozen value in final/audit output.
- Run the story's targeted validation before marking its checkpoint complete.
- Update `specs/283-show-tenant-context/progress.md` and commit only through the Ralph work-unit flow using a Conventional Commit subject ending in `#283`.

### Parallel Opportunities

- T002–T004 can be authored concurrently before the shared implementation.
- T010–T012, T017–T018, T023–T026, T034–T035, and T040–T043 are independent test-file groups within their respective phases.
- After Phase 2, US1, US2, and US3 can be developed concurrently by separate owners.
- T052 and T053 can proceed concurrently after the implementation stories complete.

---

## Parallel Examples

### User Story 1

```text
Task T010: cancel selector scope tests
Task T011: delete selector scope tests
Task T012: shared dry-run/output-mode tests
```

### User Story 2

```text
Task T017: deploy creation-target tests
Task T018: run creation-target tests
```

### User Story 3

```text
Task T023: PI service evidence tests
Task T024: PI facade conversion tests
Task T025: direct PI command tests
Task T026: process-definition and job tests
```

### User Story 4

```text
Task T034: PI cross/unknown aggregation tests
Task T035: process-definition cross/unknown impact tests
```

### User Story 5

```text
Task T040: configuration diagnostic/YAML tests
Task T041: ops model/conversion tests
Task T042: ops service aggregation tests
Task T043: ops command/report tests
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Complete T001–T009.
2. Complete T010–T016.
3. Stop and run the US1 independent test with named and empty tenant configuration.
4. Review that no filter/default ambiguity or stdout regression remains before expanding scope.

### Incremental Delivery

1. **Foundation**: shared typed context, deterministic evidence, rendering, and envelope support.
2. **US1**: search/selection scope safeguard (MVP).
3. **US2**: creation target safeguard.
4. **US3**: explicit-key authorization explanation and actual tenant evidence.
5. **US4**: cross-tenant and unknown-target warnings.
6. **US5**: configuration/ops/report consistency and compatibility proof.
7. **Polish**: documentation generation, ownership audit, vet, and full race-enabled tests.

### Ralph Execution

Run Ralph only with:

```text
--implementation-context specs/ralph-implementation-rules.md
```

Each iteration completes only the current work unit, updates `tasks.md` and `progress.md`, runs the closest validation, and commits with a Conventional Commit subject ending in `#283`.

## Notes

- `[P]` marks only file-disjoint work that can run without an incomplete same-phase dependency.
- `tenantContext` is additive; do not wrap or reshape existing JSON payloads.
- `config show` is the only YAML-producing command in scope; do not add a global YAML mode.
- Quiet and keys-only contracts are regression gates, not best-effort behavior.
- Do not use `ViewTenant()` to describe unfiltered discovery.
- Do not add local tenant mismatch rejection or an all-tenants flag.
- Do not fetch tenant metadata solely for reporting.
- Do not hand-edit generated Camunda clients.
