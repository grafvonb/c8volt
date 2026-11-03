# Task Consolidation Review: Standardize Flag Intent

**Date**: 2026-11-03
**Feature**: [spec.md](spec.md) · [plan.md](plan.md) · [final tasks](tasks.md)
**Status**: COMPLETED — 10 proposed unchecked tasks consolidated to 7 unchecked work units. No completed or partial implementation progress existed to preserve; this run changed planning documents only.

## Reviewed Inputs and Code

- Feature spec, plan, research, data model, CLI contract and quickstart; repository AGENTS guidance, constitution v2.0.0 and Ralph implementation rules.
- `cmd/update_variables_payload.go`: `readUpdateVariablesFromFlags` owns variable-source presence, exclusions and labels; callers share a single canonical-name change.
- `cmd/ops_repair_variables.go`: `parseOpsRepairVariablesFromFlags` delegates through the process variable parser; repair variable flags must move with update helper callers, rather than remain a later separate repair task.
- `cmd/update_usertask_assignment_input.go`: `resolveUpdateUserTaskAssignmentMode` and `parseUpdateUserTaskAssignmentInput` share assignment/variable exclusions, force and completion routing. `cmd/update_usertask_variables.go` reads assignment presence for JSON confirmation. These are adapters of existing assignment behavior, not new planners.
- `cmd/run_processinstance.go`: BID presence and variable input already map into creation requests. `cmd/ops_repair_incident.go` and `cmd/ops_repair_processinstance.go` already validate requested retries and optional job timeouts. Remaining changes are spelling/presence adaptation, not new workflows.
- `cmd/update_job_request.go`: `parseUpdateJobRequest` dispatches existing outcomes and checks retry/timeout presence; `cmd/update_job_batch.go` retains separate completion filtering. `cmd/update_job_outcome.go` holds shared action/exclusion checks. Retry/timeout and error action changes share files but belong to distinct explicit stories.
- `cmd/get_processinstance_validation.go`: `useInvalidInputFlagErrors` classifies pre-run failures. `cmd/bootstrap_errors.go`: `handleBootstrapError` uses an early exit boundary and `bootstrapFailureContext` may lack configured context. T001 therefore contains actual new diagnostic preparation work, not routine setup.
- `cmd/command_contract_test.go`, `cmd/capabilities_test.go`, `cmd/completion_test.go`, `docsgen/main_test.go`: existing introspection and generated-surface test infrastructure can host the full inventory in the last consumer; a separate final test/docs review unit is unnecessary.
- `Makefile`: docs-content is local docsgen; test is the race suite. Neither application tests nor generation was run for these planning changes.

## Applied Merges and ID Mapping

| Original IDs | Final ID | Cohesive outcome / eliminated handoff |
| --- | --- | --- |
| T001 | T001 | Local-only diagnostic context/recovery, shared error integration and isolated regression tests. |
| T002 + T003 | T002 | Canonical variable/assignment input policy across shared update/repair callers. One pass through task mode/exclusion/JSON guardrails, one fixture adaptation and docs refresh. All original assignment, variable/file, numeric, BID, force, prompt, logging and terminal obligations remain. |
| T004 | T003 | Job retry/timeout/inline-variable separation and effective HTTP binding coverage. |
| T005 + T006 | T004 | Canonical creation and remaining repair submitted values. Both reuse existing payload constructors and migration support; no new state machine is introduced. Retain creation exact BID/count/version behavior and repair zero/timeout/report/tenant/terminal semantics, with one migration fixture/docs pass. |
| T007 | T005 | Strictly valueless worker error action plus separate code/message/backoff, with old-code rejection and payload verification. |
| T008 | T006 | Production configured-output/migration contract acceptance and necessary root/error-path integration fixes, reusing T001 rather than rebuilding it. |
| T009 + T010 | T007 | Complete canonical query/enrichment/suppression surface and full inventory when the last consumer exists. Full-surface metadata/completion/docsgen assertions, static invocation audits and actual check evidence belong here, not a later routine review. |

All original requirements, exact-zero/opaque/numeric rules, levels/destinations, JSON EOF/partial evidence, terminal/empty-scope checks, isolated test boundary, metadata/fixtures/docs ownership and conditional broader validation remain explicit in tasks or their mandatory execution rules. Test-first ordering is inside each merged unit, before behavior changes. Regenerate documentation before assertions that read generated references.

## Retained Boundaries: Combined Outcomes Considered

### T001 + first canonical consumer T002

**Concrete combined outcome considered**: Build token-aware diagnostic recovery and local bootstrap/error rendering, then simultaneously migrate shared variable/file and assignment inputs for element/process/task/repair, preserving exact decoding/force/BID exclusions and every execution mode.

**Why retained**: This would combine two substantial validation surfaces in the first iteration: new lexical recovery after parse failure (short/inherited/attached flags, legacy value arity, terminator and opaque-value handling) plus new local-only configured bootstrap/single-envelope exit handling, and all payload/assignment regressions across existing write workflows. `bootstrapFailureContext` lacks configured state on early failure, while `readUpdateVariablesFromFlags` and `parseUpdateUserTaskAssignmentInput` govern a separate valid-execution dispatch path. A failing first-consumer case could reflect either parser recovery, early context setup, numeric decoding or dispatch. The combined implementation and required pre-run/valid-run matrix is impractical for one short iteration after accounting for reused helpers; keep the new shared mechanism with synthetic/isolated leaf execution coverage, then its existing-workflow adaptations. This is not a context-reading or scaffolding-only prerequisite. Eliminating the boundary saves one handoff but does not eliminate either new error-preparation algorithm or payload/mode validation.

### T002 + T003 + T004: all US1 submitted-value renames

**Concrete combined outcome considered**: Change all shared update/repair variable/assignment paths, job retry/HTTP-timeout semantics and run/repair initialization in one work unit with all fixtures and docs.

**Why retained**: Job retry/timeout input introduces an independent selection-versus-mutation split and fixes actual Viper inherited binding (`cmd/root_config.go` binds `http.timeout` from the resolved timeout flag), requiring config/profile/env/position permutations and captured payload evidence. The other combined consumer already spans exact task-number decoding, assignment force/prompt/JSON exclusions and shared repair variable behavior. Creation adds BID/version/count semantics and repair preserves zero-skip/report behavior. This complete cross-product plus partial-output and terminal checks is impractical for one iteration even with shared helpers. Three units keep those obligations executable; no services or shared interfaces are temporarily broken. The creation/repair work itself was merged because it is existing mapping adaptation.

### Strongest rejected merge: T003 + T005 (nonadjacent job changes)

**Concrete combined outcome drafted**: “Complete all job write grammar in one pass through update_job/request/outcome/batch: retry predicates versus submitted retries, independent job/HTTP timeouts, variables, strictly valueless BPMN action, code/message/backoff; include effective-config, legacy Boolean-code, payload, output and terminal matrices.”

**Evidence preventing merge**: US1 owns selection/submitted-value distinction; US2 explicitly owns error-action/value separation. `specs/ralph-implementation-rules.md`, Mandatory Scope, requires: “Complete only the current Ralph work unit. Do not start another user story or polish section in the same iteration.” Combining these outcomes would require completing two named stories in one iteration. Retain this explicit execution boundary, while T005 reuses T003's storage/presence and timeout fixes. No new shared parser is built twice. The intervening T004 has no job dependency; the graph remains acyclic with T005 depending on T003.

### T001 + T006: shared support with full production acceptance

A combined outcome would build the early error mechanism and certify canonical production writes with the complete logging/partial-result/debug/terminal matrix. The latter depends on T002–T005 already migrating those commands; moving it into T001 either creates a dependency cycle or absorbs the intervening stories into one iteration, violating story scope. T001 includes necessary direct mechanism tests immediately; T006 contributes production cross-path acceptance and necessary integration fixes, not postponed tests of an unverified helper.

### T005 + T006 and T006 + T007

These candidate merges cross explicit US2/US4 and US4/US3 story boundaries under the same Ralph rule. T006 remains the production diagnostic contract owner; T007 adds the remaining query consumers and full inventory, without rebuilding reporting support. All per-command tests and docs already belong with their implementation; T006 does not excuse incomplete logging/output tests in T002–T005.

## Second-Pass Review

After the three merges, challenged every remaining adjacent and nonadjacent boundary. No additional merge can fit the practical first-mechanism/consumer or US1 complete-matrix limit above without crossing explicit story scope or creating the support/production-acceptance dependency cycle. No tasks were added for speculative parallelism. Final phase headings retain template structure, but setup and polish have no independent checkboxes.

## Final Graph, Coverage and Progress

- Counts: 7 total, all unchecked; foundational 1; US1 3; US2 1; US4 1; US3 1. No P markers or isolated documentation/setup chores.
- Order: T001 → T002 → T003 → T004 → T005 → T006 → T007 by default. Essential edges are recorded in tasks; all point forward. Shared README/metadata/generated artifacts justify serial integration, not extra tasks.
- US1 primary outcome: exact submitted values and independent job HTTP configuration; US2: error action/code/failure payload semantics; US4: configured logging/machine-output and legacy rejection; US3: identical query enrichment/suppression information and final canonical-surface agreement.
- Requirement coverage: every FR-001–FR-014 and SC-001–SC-006 has an owner in the task coverage table; shared behavior is implemented once, query-specific acceptance is added with T007. No backend/public schema change or live gate was added.
- No progress was marked complete. No application implementation, tests, commit, publication or Ralph launch occurred.

## Checks

Read back both saved files. Validate seven sequential unique checklist IDs, per-story labels, file paths, forward dependencies and references, original-to-final coverage mapping, links and whitespace; check no unresolved placeholders and no new live-environment gate. Run `git diff --check`. Proposed Go checks are future work-unit instructions only; application tests not run for planning-only changes. No limitations identified beyond unperformed implementation/runtime checks, which belong to execution.
