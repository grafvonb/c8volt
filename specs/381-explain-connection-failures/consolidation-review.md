# Task Consolidation Review: Explain Known Connection and Authentication Failures

**Date**: 2026-11-18

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Tasks**: [tasks.md](tasks.md)

**Result**: NO MERGES — reviewed 3 tasks, retained 3 tasks, all unchecked.

## Review Inputs and Existing Mechanisms

Read the feature specification, plan, research, transient evidence model, human-error contract, quickstart, constitution, repository guidance, and Ralph scope/validation rules. There is no completed or partially completed task evidence to preserve; implementation has not started.

| Inspected code / rule | Evidence and effect on task boundaries |
| --- | --- |
| `cmd/cmd_views_contract.go`: `boundedCommandFailureReason`, `handleCommandErrorWithMode`, `resultEnvelopeForError` | Shared formatter currently returns a class plus request-rate evidence; logger routing, JSON-first rendering, error identity, and exits already exist. Only shared reason delegation is new; all stay in T001 rather than separate framework/wiring tasks. |
| `cmd/cmd_views_config_test_connection.go`: `configConnectionFailureMessage` | Existing typed-cause/status wording is reusable. Extraction, generic delegation, config cancellation/fallback/topology preservation, and their exact tests form one outcome in T001. |
| `cmd/cmd_errors.go`: `commandConnectionFailureDetails`; `internal/services/httpc/failure.go`: `FailureDetails` | Typed fact extraction already exists. No transport implementation task, new domain model, or string parser is warranted. |
| `c8volt/ferrors/errors.go`: `wrap` | Existing auth/status/admission annotations preserve causes; bare transport evidence may be discarded. T001 keeps that boundary and tests fallback; no cross-layer normalization refactor is introduced. |
| `cmd/get_processdefinition.go`: `runSearchProcessDefinitions` | Caller already supplies an operation prefix and bounded reason. Original refusal requires an isolated test fixture and formatter correction, not a new command handler or caller migration. |
| `cmd/get_processdefinition_error_envelope_test.go`: `TestGetProcessDefinitionLifecycleSearchDiagnostics`, helper, and `assertProcessDefinitionErrorMatrix` | Plain/structured, configured/inherited logging, JSON, raw-marker exclusion, request counts, and exit suppression infrastructure exist. T001 adds original refusal and adapts exact expectations; T002/T003 extend only missing story acceptance, not another harness. |
| `cmd/delete_processdefinition_unfinished_test.go`: `TestDeleteProcessDefinitionUnfinishedDiagnosticLogging`; `cmd/cmd_views_resource.go`: `formatProcessDefinitionDeleteFailure` | Repeated failures, counts, partial results, DEBUG/quiet, mutation counts, and one-envelope checks already exist. T001 owns changed human expectations and passing baseline checks; T002 strengthens adversarial/destination assertions within that same fixture. No new batch workflow is planned. |
| `cmd/update_job_batch_logging_test.go`: `bulkCompletionDiagnosticRecords` | Whole-record validation rejects direct writes; supports plain-time/JSON, not plain. Existing parser reuse remains in T002; no logging test framework task is needed. |
| `cmd/request_rate_mutation_test.go`: `isolateRequestRateEnvironment`, local-failure/interruption/confirmation/partial-reporting tests | Admission, cancellation, isolated env, timing control, and subprocess mechanics are available. T003 adds only missing compatibility combinations and uses existing mechanics. |
| `cmd/config_test_connection_diagnostics_test.go` and `cmd/root_api_diagnostics_test.go` | Specialized advice, auth flow, successful output, requests, logger filtering, and DEBUG evidence already have fixtures. Preserve/reuse them in the owning tasks. |
| `AGENTS.md`, Task Planning For Ralph | Implementation, fixtures, tests, formatting, docs, and validation stay together. No setup/context/progress/polish-only tasks are added. |
| `specs/ralph-implementation-rules.md`, Mandatory Scope and Iteration Discipline | Explicit rule: “Complete only the current Ralph work unit. Do not start another user story or polish section in the same iteration.” Retain separate story execution boundaries; do not justify them merely by file names or generated headings. |
| `speckit-tasks` Task Generation Rules | Explicitly requires task organization by user story and one story phase per story. The three labels map to the specification's three actual acceptance journeys. |

## Adjacent and Nonadjacent Merge Review

Every boundary was treated as removable initially. No new model/service/algorithm/lifecycle work was inferred from existing helpers. The core formatter and necessary fixtures/tests/docs are already combined in T001; splitting extraction from its first consumer or splitting failing tests from implementation would add a useless handoff.

### Strongest candidate: T001 + T002

**Draft combined outcome**: Deliver the complete recognized-cause formatter, exact original OAuth refusal reproduction, all normal/verbose/quiet/DEBUG and destination/format assertions, repeated-resource redaction and partial JSON evidence, help/docs generation, formatting, and targeted validation in one unit.

This would remove a revisit to the search/config/deletion test files and one fixture handoff. Reused mechanisms make it technically plausible; there is no claim that code volume alone makes it impractical. It is retained as two work units because it would execute two separate specified stories in one Ralph iteration, contrary to the explicit Mandatory Scope rule quoted above and the tasks skill's story organization requirement. T001 already includes mandatory baseline logger, redaction, batch, and machine-contract checks; T002 is not a deferred safety repair. T002 adds concrete adversarial coverage and missing WARN/quiet/DEBUG/destination combinations rather than a second production policy.

### T002 + T003

**Draft combined outcome**: Complete the full adversarial logger matrix and request-rate/cancellation/output-control/exit-suppression/success compatibility acceptance using existing fake-backed subprocess fixtures.

This would avoid another overlap in search/config fixtures and partial-JSON assertions. The same explicit single-story execution rule prevents combining US2 and US3 into one Ralph unit. T002's distinct outcome is complete safe diagnostic collection; T003's is operation/automation invariance, including original-refusal quiet+JSON and exit-suppression combinations. Keep these targeted and reuse existing checks; do not add duplicate assertions merely to fill a task.

### Nonadjacent T001 + T003

**Draft combined outcome**: Ship recognized-cause formatting together with all operation/automation compatibility cases and documentation, leaving US2 adversarial matrix work separate.

This would remove some error-evidence fixture repetition but combine US1/US3 across the intervening US2 dependency and violate the same single-story scope. Baseline precedence, class, JSON, and count checks already belong to T001; T003 only completes execution-level cases. No hidden prerequisite or normalization change is deferred from T001 to T003.

### Setup, foundations, and polish

No merge is needed because none has an executable checkbox. Reading, branch checks, progress, formatting, exact-expectation adaptation, documentation, and targeted checks are included in their owning units. There is no separate interfaces/generated-client/service work, no hypothetical parallelism, and no completed progress that could mask unfinished work.

## Saved Result and Coverage Preservation

- Counts remain 3 → 3; US1: 1, US2: 1, US3: 1. Setup/foundational/polish counts are zero.
- ID mapping is unchanged: T001 → T001, T002 → T002, T003 → T003; all remain unchecked and no evidence is claimed as implemented.
- Dependency order remains T001 → T002 → T003, acyclic, with no `[P]` markers or speculative parallel tasks.
- T001 owns the complete cause policy, shared/helper reuse, original refusal, config-specific advice, baseline routing/redaction/machine compatibility, all directly changed exact expectations, documentation, and targeted validation.
- T002 owns missing adversarial/destination/severity/logging combinations and repeated-failure evidence; T003 owns missing execution/output/exit compatibility combinations. Both reuse T001 without rewriting it.
- Every FR-001–FR-011 and SC-001–SC-005 remains mapped in tasks.md. Data-model constraints are quoted in T001; no schema or persistence work is added.
- Tests remain first within each work unit, followed by needed implementation/adaptation and passing targeted checks. No unit is deliberately left uncompilable or with a known failing related check.
- During review, clarified the verbatim category/authentication constraints and added the review link in tasks.md. These are not merges. Only tasks.md and this review were written by the consolidation hook.

## Checks and Limitations

Read back both saved files; validate checkbox format/IDs/story labels, counts, unchanged progress, requirement ownership, dependency references, paths/local links, Markdown fences, placeholder removal, and whitespace. Review existing code/test symbols without running application tests. The user requested task generation, so no implementation, commit, publication, Ralph launch, test execution, or live service access is part of this review.

No planning blocker identified. Runtime behavior and live-engine behavior remain unverified; planned isolated implementation checks are required later. Optional downstream hooks are handled by the parent tasks workflow after this review completes.
