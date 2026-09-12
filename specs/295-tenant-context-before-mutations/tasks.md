---
description: "Dependency-ordered implementation tasks for issue #295"
---

# Tasks: Tenant Context Before Ops Mutations

**Input**: Design artifacts in `specs/295-tenant-context-before-mutations/`.

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contract](contracts/tenant-reporting.md), and [quickstart.md](quickstart.md).

**Tests**: Explicitly required by FR-012 and the constitution. Tests precede the behavior they verify; execute relevant checks within each work unit, not only at the final phase. Do not commit an intentionally failing test-only unit.

**Organization**: Shared prerequisites, then US1 (P1), US2 (P1), US3 (P2), then final validation. This is an existing Go repository; no scaffolding, new dependency, migration, or generated-client change is needed.

## Format and Path Conventions

Every task uses `- [ ] TNNN [P?] [USn?] Description`. Paths are relative to the repository root. `[P]` indicates independent files within a ready dependency wave, not permission to skip prerequisites. Story tasks carry their spec story ID; shared and polish tasks do not.

## Execution Rules

Read `specs/ralph-implementation-rules.md` before implementing. Any Ralph launch must include `--implementation-context specs/ralph-implementation-rules.md`. Keep work units bounded, document changed functions/tests as required, reuse `testx` helpers, synchronize concurrent observations, and avoid parallel tests that mutate shared command globals. Preserve operational verification and current backend behavior. Before any implementation commit, run `make test`; use Conventional Commits with #295 and include task/progress evidence for completed work. Documentation changes must accompany the corresponding user-visible work before it is committed or released; T034 is the final all-command documentation sweep, not permission to defer required guidance from earlier work units.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the existing project context and establish implementation evidence; no new project scaffolding or dependencies are required.

- [x] T001 Read `AGENTS.md`, `specs/ralph-implementation-rules.md`, and all feature artifacts; create `specs/295-tenant-context-before-mutations/progress.md` recording the current branch, owner-layer map, baseline targeted test results from `quickstart.md`, and any blockers. Preserve existing uncommitted work and do not change branches.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Provide the shared event contract and stage-aware rendering needed by all stories. Complete this phase before story work. Write regression tests before their implementation; keep intentionally failing test work together with the corresponding fix before marking the work unit complete.

- [x] T002 [P] Add progress conversion contract tests in `c8volt/ops/client_test.go` for the new optional tenant-scope payload, nil callbacks, empty evidence versus absent payload, both mapping directions, and slice/target isolation: “Payload slices are copied at the facade boundary; callback consumers cannot modify service-owned plan evidence.”
- [x] T003 [P] Add staged-renderer regressions in `cmd/cmd_views_tenant_context_test.go` for selection followed by affected context, duplicate callbacks, complete versus partial final rendering, protected channels, fresh command execution state, and unchanged legacy creation/full-context rendering.
- [x] T004 Add `tenant_scope` kind and optional `TenantScope` payload to `internal/domain/ops_progress.go` and `c8volt/ops/progress_model.go`, preserving existing event meanings: “The event is an additive member of the existing progress union; other payloads are absent for this kind.” and “A nonnil payload with empty evidence explicitly indicates a successfully established empty scope.” Keep evidence typed to existing domain/public models and callbacks excluded from serialized requests.
- [x] T005 Map the new progress payload mechanically in both directions in `c8volt/ops/convert.go`, reuse evidence conversion/copy helpers, preserve nil and empty payload distinctions, and make the conversion tests pass. Preserve: “This event is not appended to final CLI results or audit report schemas.”
- [x] T006 Implement staged ops emission state and semantic selection/affected line partitioning in `cmd/cmd_tenant_context.go`, `cmd/cmd_views_tenant_context.go`, and `cmd/ops_tenant_context.go`; implement “Fields: `selectionRendered` and `affectedRendered` booleans, initialized false per execution.” Keep attached context immutable, mark only permitted stages, treat successful empty scope as complete without affected lines, and preserve unrelated legacy rendering. Reuse `opsProgressChannelForMode` and existing warning grammar rather than text matching.
- [x] T007 Run the foundational facade and tenant-renderer tests from `specs/295-tenant-context-before-mutations/quickstart.md`; record exact commands/results in `specs/295-tenant-context-before-mutations/progress.md` and resolve failures before story integration.

**Checkpoint**: Event mapping and stage-aware rendering are tested; no service or command path has been forced into an extra discovery pass.

---

## Phase 3: User Story 1 - See tenant scope before auto-confirmed work (Priority: P1)

**Goal**: All six commands display selection before discovery and affected evidence before the earliest mutation, with auto-confirm skipping the question only.

**Independent Test**: Run each real command handler with a fake backend and auto-confirm; observe selection at discovery/activity entry and affected context at the first mutation. Include APD alias, repair variables, and request counters to prove no additional discovery.

**Tests first**: T008–T012 establish service and command regressions before T013–T020 wire behavior.

- [x] T008 [P] [US1] Add service ordering tests in `internal/services/ops/all_process_definitions_purge_test.go` and `internal/services/ops/incident_purge_test.go` asserting one scope event from validated delete-plan evidence before cancellation/deletion, no partial event after failed planning, and unchanged discovery counts, targets, force gates, dry-run, and empty outcomes.
- [x] T009 [P] [US1] Add equivalent service ordering and no-extra-request tests in `internal/services/ops/orphan_purge_test.go` and `internal/services/ops/retention_policy_test.go`, including dependency-expanded evidence and existing preview/force/no-work gates.
- [x] T010 [P] [US1] Add repair notification tests in `internal/services/ops/repair_test.go` for explicit/search incident repair and explicit/search process-instance repair; assert “Synchronous callback completion precedes the first mutation, including repair variable updates.” Cover empty frozen sets, discovery failures, nil callbacks, and unchanged variable/job/incident mutations.
- [x] T011 [P] [US1] Add real-command auto-confirm timing regressions in `cmd/ops_purge_all_processdefinitions_test.go`, `cmd/ops_purge_orphan_processinstances_test.go`, `cmd/ops_purge_processinstances_with_incidents_test.go`, and `cmd/ops_execute_retention_policy_test.go`; include `apd`, named/unfiltered selection, supported explicit keys, selection before activity/first discovery, affected output before the first mutation, no prompt, and baseline request-count/target equality.
- [x] T012 [P] [US1] Add real-command auto-confirm timing regressions in `cmd/ops_repair_incident_test.go` and `cmd/ops_repair_processinstance_test.go` for both keyed/search paths, observing the variable update as first mutation when requested; include named/unfiltered selection, explicit-key not-applied semantics, no prompt, and unchanged backend targets/request counts.
- [x] T013 [US1] Add a shared synchronous tenant-scope emission helper in `internal/services/ops/tenant_evidence.go` using existing evidence snapshots and progress callbacks; nil callback is a no-op, no retrieval occurs, and callback consumers must not modify the stored plan. Preserve “Existing plan/impact validation must succeed before emission; partial failure evidence is not published as validated.”
- [x] T014 [P] [US1] Emit validated `DeletePlan.TenantEvidence` in `internal/services/ops/all_process_definitions_purge.go` at the successful preview/execution boundaries identified in `research.md`, before cancellation/deletion; preserve dry-run/force/error/empty branches and all existing discovery and cleanup behavior.
- [x] T015 [P] [US1] Emit validated `DeletePlan.TenantEvidence` in `internal/services/ops/incident_purge.go` before mutation and on successful preview/empty scope paths; retain existing force checks, impact validation, frozen candidate reuse, and deletion behavior.
- [x] T016 [P] [US1] Emit validated `DeletionPlan.TenantEvidence` and `DeletePlan.TenantEvidence` in `internal/services/ops/orphan_purge.go` and `internal/services/ops/retention_policy.go`, respectively; publish after successful expansion/impact validation without moving force gates or adding traversal, before each first mutation and on applicable successful previews.
- [x] T017 [P] [US1] Emit `FrozenSet.TenantEvidence` in `internal/services/ops/repair.go` from `repairExplicitIncidents`, `repairFilteredIncidents`, and `finishProcessInstanceIncidentRepair`, after complete frozen scope and before dry-run return or `executeRepairVariableUpdates`; preserve existing no-active-incident, failure, and execution paths.
- [x] T018 [US1] Compose tenant-event handling with existing progress callbacks in `cmd/ops_purge_all_processdefinitions_progress.go`, `cmd/ops_processinstance_purge_progress.go`, and `cmd/ops_repair_progress.go`; attach normalized full evidence and route affected lines via `cmd/ops_tenant_context.go` before generic progress processing, without replacing existing callback behavior or rate-limiting this one-time boundary.
- [x] T019 [US1] Initialize staged selection context in `cmd/ops_purge_all_processdefinitions.go`, `cmd/ops_purge_orphan_processinstances.go`, `cmd/ops_purge_processinstances_with_incidents.go`, and `cmd/ops_execute_retention_policy.go` after local/report-path validation and before activity wrappers or service calls; derive explicit/discovery mode from existing request semantics and preserve final report-context attachment.
- [x] T020 [US1] Initialize staged selection context in `cmd/ops_repair_incident.go` and `cmd/ops_repair_processinstance.go` after local/report-path validation and request-mode resolution, before discovery/activity; preserve explicit/search distinctions and existing command dispatch, variable inputs, and final report attachment.
- [x] T021 [US1] Run all US1 service, facade, and six-command timing tests using the targeted commands in `specs/295-tenant-context-before-mutations/quickstart.md`; record discovery-count, earliest-mutation, and selected-target evidence in `specs/295-tenant-context-before-mutations/progress.md` and resolve regressions.

**Checkpoint / MVP**: Auto-confirm tenant visibility works for all six commands. Existing mode protections and backend behavior must still pass; subsequent phases add complete interactive and audit acceptance coverage.

---

## Phase 4: User Story 2 - Review complete context once before confirming (Priority: P1)

**Goal**: Interactive execution presents all applicable context before the question and suppresses duplicate tenant output through final rendering.

**Independent Test**: Use accepted/declined input with single/multiple/unknown/empty scopes; inspect context at the prompt and count each applicable line once across the invocation.

**Tests first**: T022–T024 before T025–T027. US2 builds on US1 notification wiring; its acceptance scenarios are independently runnable.

- [x] T022 [P] [US2] Add interactive accept/decline ordering and duplicate-callback tests to `cmd/ops_purge_all_processdefinitions_test.go`, `cmd/ops_purge_orphan_processinstances_test.go`, `cmd/ops_purge_processinstances_with_incidents_test.go`, and `cmd/ops_execute_retention_policy_test.go`; assert context precedes the prompt, decline mutates nothing, planned-key reuse is unchanged, and final output repeats no tenant summaries.
- [x] T023 [P] [US2] Add interactive keyed/search acceptance tests in `cmd/ops_repair_incident_test.go` and `cmd/ops_repair_processinstance_test.go` checking complete tenant context at confirmation, no mutations on decline, no repeated warning after confirmed execution, and unchanged repair counts.
- [x] T024 [P] [US2] Extend `cmd/cmd_views_tenant_context_test.go` and create `cmd/ops_tenant_context_test.go` with single/multiple/default/unknown-only/known-plus-unknown/multiple-plus-unknown/duplicate/empty evidence cases and all override transitions. Assert: “Missing metadata stays unknown; configured tenant never fills it in.”; “Empty validated evidence means zero known affected tenants and zero unknown targets, not discovery failure.”; “Negative unknown counts remain normalized by existing rules.”; “Actual default-tenant evidence uses the established representation; an empty selection filter does not imply the default tenant.”; “More than one distinct known tenant sets cross-tenant semantics; unknown metadata may coexist with that condition.” Include fresh invocation state and fallback rendering of only unrendered applicable stages.
- [x] T025 [US2] Replace hardcoded human-channel confirmation printing with idempotent staged reporting in `cmd/ops_purge_all_processdefinitions.go`, `cmd/ops_purge_orphan_processinstances.go`, `cmd/ops_purge_processinstances_with_incidents.go`, and `cmd/ops_execute_retention_policy.go`; retain planned-result fallback attachment, frozen candidate reuse, force/no-work gates, prompt wording, and real mode-derived channels.
- [x] T026 [US2] Integrate idempotent pre-prompt reporting in `cmd/ops_repair_incident.go` and `cmd/ops_repair_processinstance.go`, retaining current preflight policy, frozen-request reuse, and no-prompt dry-run/auto-confirm behavior; remove hardcoded human visibility bypasses.
- [x] T027 [US2] Complete stage-aware final suppression in `cmd/cmd_views_tenant_context.go` and `cmd/ops_tenant_context.go` so repeated preview/execution events and final `renderAttachedTenantContext` calls emit each applicable tenant summary/warning once, at existing severity, without pruning attached evidence or changing unrelated workflow headers/results. Verify absent/equal overrides stay silent and explicit-key selection never implies a tenant filter.
- [x] T028 [US2] Run US2 interactive and renderer tests, plus US1 timing regressions, using `specs/295-tenant-context-before-mutations/quickstart.md`; record prompt-time output, warning occurrence counts, and zero mutations on decline in `specs/295-tenant-context-before-mutations/progress.md`.

**Checkpoint**: Interactive and auto-confirm paths share reporting order and retain their distinct confirmation policies.

---

## Phase 5: User Story 3 - Preserve automation output and audit evidence (Priority: P2)

**Goal**: Preserve protected modes, structured results, audit evidence, and operator guidance.

**Independent Test**: Parse structured output and generated JSON/Markdown audit files after equivalent human/protected executions; compare tenant evidence and verify no new human preflight text enters either protected stream.

**Tests first**: T029–T031 before T032–T034. Existing protection remains mandatory during earlier phases.

- [x] T029 [P] [US3] Add command-mode regressions in `cmd/ops_contract_test.go` using all six actual command handlers: JSON, quiet, automation, verbose/debug, and other advertised modes. Verify no new tenant chatter on either protected stream, stable envelopes, existing implicit confirmation/exit behavior and quiet failure diagnostics; keep unsupported modes rejected and supported keys-only output one key per line.
- [x] T030 [P] [US3] Extend `cmd/ops_report_json_test.go` and `cmd/ops_report_markdown_test.go` to render early tenant context before serializing complete reports; cover named, unfiltered, explicit-key, multiple and unknown evidence, existing warning fields, cloned context independence, and truthful legacy `tenantId` without adding override-provenance fields.
- [x] T031 [P] [US3] Add real-command audit-file and failure-path regressions in `cmd/ops_purge_all_processdefinitions_test.go`, `cmd/ops_purge_orphan_processinstances_test.go`, `cmd/ops_purge_processinstances_with_incidents_test.go`, `cmd/ops_execute_retention_policy_test.go`, `cmd/ops_repair_incident_test.go`, and `cmd/ops_repair_processinstance_test.go`; distribute dry-run, empty, invalid input, discovery/impact failure, force-blocked, mutation failure, and report-preservation scenarios across existing fixtures while requiring complete applicable audit tenant evidence for all six commands.
- [x] T032 [US3] Verify and correct any staged-output bypass found by T029 in `cmd/ops_tenant_context.go` and `cmd/cmd_views_tenant_context.go`, consistently using `cmd/ops_progress_mode.go` policy without broadening supported modes or suppressing existing operational failure diagnostics; keep context attachment independent from human emission flags.
- [x] T033 [US3] Preserve complete audit serialization and legacy tenant semantics in `cmd/ops_tenant_context.go` and all six result/report attachment call sites in `cmd/ops_purge_all_processdefinitions.go`, `cmd/ops_purge_orphan_processinstances.go`, `cmd/ops_purge_processinstances_with_incidents.go`, `cmd/ops_execute_retention_policy.go`, `cmd/ops_repair_incident.go`, and `cmd/ops_repair_processinstance.go`; make T030–T031 pass without removing tenant evidence, changing schemas, or treating partial failure evidence as validated human scope. Avoid production edits where existing behavior already passes.
- [x] T034 [US3] Update operator guidance in `README.md` and command help/source metadata in `cmd/ops_purge_all_processdefinitions.go`, `cmd/ops_purge_orphan_processinstances.go`, `cmd/ops_purge_processinstances_with_incidents.go`, `cmd/ops_execute_retention_policy.go`, `cmd/ops_repair_incident.go`, and `cmd/ops_repair_processinstance.go` to explain selection before discovery, affected tenants before mutation, and auto-confirm skipping only the question. Run `make docs-content` to regenerate `docs/cli/` and `docs/index.md`; do not hand-edit generated references.
- [x] T035 [US3] Run protected-mode, audit, failure, and help regressions from `specs/295-tenant-context-before-mutations/quickstart.md`; record results and review source/generated guidance consistency in `specs/295-tenant-context-before-mutations/progress.md`.

**Checkpoint**: All three stories are implemented and independently verifiable, with full evidence retained and user guidance aligned.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Complete the feature-level validation and repository governance checks.

- [x] T036 Audit changed files against layer and command-cohesion rules in `AGENTS.md` and `specs/ralph-implementation-rules.md`; ensure reporting lifecycle stays in focused support/progress files, facades only map, and no command/service changes add retrieval or alter mutation mechanics. Apply `gofmt` to touched Go files and record the review in `specs/295-tenant-context-before-mutations/progress.md`.
- [x] T037 Execute the complete guide in `specs/295-tenant-context-before-mutations/quickstart.md`, including targeted packages followed by `make test` (`go test ./... -race -count=1`) and `git diff --check`; resolve failures and record exact commands/results in `specs/295-tenant-context-before-mutations/progress.md`. Do not claim completion or commit with failing required checks.
- [x] T038 Review every FR-001–FR-014 and SC-001–SC-007 against the coverage matrix in `specs/295-tenant-context-before-mutations/contracts/tenant-reporting.md`; record concrete test names and outcomes in `specs/295-tenant-context-before-mutations/progress.md`, update `specs/295-tenant-context-before-mutations/tasks.md` only for verified work, and leave unresolved work unchecked.

---

## Dependencies & Execution Order

### Phase and Story Dependencies

```text
Setup T001
  → Foundation T002–T007
  → US1 T008–T021 (MVP)
  → US2 T022–T028
  → US3 T029–T035
  → Polish T036–T038
```

US2 reuses US1 service notifications and command integration. US3 validates those completed reporting paths and both confirmation modes. These are explicit implementation dependencies, while each story has its own independently runnable acceptance criteria. Do not claim the stories can be implemented concurrently in the same command files.

### Task Dependency Waves

- T002 and T003 can run together after T001. T004 follows T002; T005 follows T004. T006 follows T003. T005 and T006 touch different files and can be developed concurrently after their individual prerequisites. T007 waits for both.
- T008–T012 are independent regression-writing tasks after T007. T013 follows their contract agreement. After T013, T014–T017 can run together in separate service files; each must satisfy its preceding service tests. T018 then composes callback handling; T019 and T020 wire commands after T018. T021 gates US1 completion.
- T022–T024 can run together after T021. T025 follows T022; T026 follows T023; T027 follows T024 and prompt integration. T028 verifies the integrated result.
- T029–T031 can run together after T028, provided T029 keeps mode tests in `cmd/ops_contract_test.go` and does not edit the six fixture files owned by T031. T032 follows T029; T033 follows T030–T032; T034 follows completed reporting/audit integration; T035 gates US3 completion.
- T036–T038 run sequentially after T035. Any fixes discovered during validation require rerunning the relevant checks.

### Parallel Example: User Story 1

After foundation, develop T008 (APD/incident-purge service regressions) alongside T009 (orphan/retention service regressions) and T010 (repair service regressions). After T013, implement T014, T015, T016, and T017 in disjoint service files. Shared progress callback integration T018 waits for these producers.

### Parallel Example: User Story 2

After US1, develop T022 (purge/retention interactive command tests), T023 (repair interactive command tests), and T024 (shared renderer/evidence tests) together. Complete shared renderer and confirmation integration before running the story checkpoint.

### Parallel Example: User Story 3

After US2, develop T029 (`cmd/ops_contract_test.go` mode tests), T030 (JSON/Markdown report unit tests), and T031 (six command audit/lifecycle fixtures) together. Serialize production fixes and documentation when they touch the same command/support files.

## Requirement Coverage

| Requirements / outcomes | Primary tasks |
| --- | --- |
| FR-001, FR-002, FR-004, FR-006; SC-001 | T008–T021: all six commands, early selection, earliest-mutation event and output |
| FR-003; selection cases in FR-012 | T011–T012, T019–T020, T024: named/unfiltered/explicit keys and all override transitions |
| FR-005, FR-007; SC-002–SC-004 | T003, T006, T022–T028: prompt order, severity, duplicate/unknown/default/empty evidence |
| FR-008; SC-005 | T002, T005, T030–T033: copied evidence and complete JSON/Markdown reports |
| FR-009; SC-005 | T003, T006, T029, T032, T035: protected modes, streams, prompts, exit behavior |
| FR-010, FR-014; SC-006 | T008–T017, T021, T036: existing evidence only, unchanged request counts and mutation targets |
| FR-011 | T008–T012, T024, T031: empty, failed validation/discovery, force gates, dry-run and failure semantics |
| FR-012 | All service, facade, command, interactive, mode, and audit test tasks; final traceability T038 |
| FR-013; SC-007 | T034–T035: README, source help, regenerated CLI references |

The broad acceptance matrix uses representative combinations, while all six commands must have auto-confirm ordering, interactive ordering, protected-mode, and audit coverage. Both repair selection modes and each distinct service notification placement require coverage. Test actual mutation boundaries using safe observations; final output substring ordering alone is insufficient.

## Implementation Strategy

### MVP First

Complete setup, foundation, and US1 through T021. Demonstrate all six auto-confirm commands against deterministic fake backends. The MVP must preserve existing protected output and mutation behavior; US3 adds comprehensive regression proof, not deferred correctness. Apply relevant documentation and full-suite commit gates even for an MVP checkpoint.

### Incremental Delivery

Add US2 to complete interactive context and uniqueness acceptance, then US3 to complete mode/audit/documentation coverage. Finish cross-cutting validation and requirement traceability. Keep related failing tests and their implementation in the same completed work unit. Work on the current issue branch; do not launch implementation, create additional branches, or commit merely because task generation has completed.

## Notes

No task is pre-completed. No implementation or runtime validation is claimed by this document. Optional commit and Ralph hooks are offered separately; neither is required to generate this task list.
