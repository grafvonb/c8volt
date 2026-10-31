# Task Consolidation Review: Advisory Camunda Version Check

**Reviewed**: 2026-10-30
**Feature**: [spec.md](spec.md) · [plan.md](plan.md) · [tasks.md](tasks.md)
**Trigger**: Mandatory `speckit.task-consolidation.consolidate` after-tasks hook.
**Before / after**: 3 / 3 tasks.
**Outcome**: No task changes. The existing three cohesive units are retained after code-grounded review. This file records the review; it does not establish implementation completion.

## Inputs and progress

Read the feature specification, plan, research, data model, CLI contract, quickstart, resolved task template, repository instructions, applicable Ralph work-unit constraints, and constitution. The fresh task draft has T001–T003, all unchecked; no feature progress record exists. No feature-local external T001–T003 references or started-task evidence required renumbering. No IDs were removed or reassigned.

## Code evidence inspected

| Evidence | Consequence for decomposition |
| --- | --- |
| `internal/services/common/deps.go`: `ServiceDeps`, `PrepareServiceDeps` | A single concrete HTTP client currently flows to all adapters; the additive Do capability and its consumers must change together. |
| `internal/services/cluster/api.go`, `factory.go`, `cluster/common/topology.go` | Existing neutral topology source and versioned factories can serve the complete checker; topology already imports httpc, so checker/gate dependency direction must be explicit. |
| `internal/services/httpc/service.go`: `New` and `Client` | `http.Client.Timeout` is already configured before transport execution. Pre-Do checking is essential to avoid stealing an operation's timeout; it is not a cosmetic wrapper split. |
| `internal/services/httpc/read_retry.go`: `WithoutReadRetry`, `ReadRetryTransport.RoundTrip` | Reuse the existing advisory retry bypass, while preserving operational retries. Separate repeated retry implementations are unnecessary. |
| `internal/services/httpc/request_rate_error.go`: `BeginRequestRateAttempt`, `RequestRateObservation` | Probe admission evidence needs a fresh context observation, distinct from a mutation's observation. |
| `internal/services/httpc/request_rate_test.go`: `TestRequestRateClientTimeoutRetainsObservation` | Real HTTP timeout behavior can replace a transport error; helper-only deadline checks cannot replace the actual wiring regression. |
| 33 `internal/services/<area>/{v88,v89,v810}/service.go` files | Confirmed 36 generated `WithHTTPClient(deps.HTTPClient)` uses; keep them together, including v88 Operate and v88/v89 Tasklist. Do not create one task per version. |
| `internal/services/processinstance/v810/service.go` and corresponding v88/v89 constructors | Nested variable-service construction still requires the concrete client; broad replacement would break the intended contract. |
| `internal/services/common/usertask_submission.go`: `submitUserTask` | Submission status and sticky ambiguity evidence are existing workflow contracts, not outcomes the advisory may replace. |
| `cmd/root_services.go`: `installRemoteCommandServices` | Auth initialization precedes invocation publication. Flag activation and checker installation must preserve this order, including direct connection-test installation. |
| `cmd/config_diagnostics.go`, `cmd/config_test_connection.go`, `cmd/config_test.go` | Legacy parser/classification is permissive and JSON warnings are asserted. Human deduplication/safe display must remain separate from machine warning evidence. |
| `cmd/root_api_diagnostics_test.go`: `TestAPIDiagnosticsCommandReadPreservesResultsAndRequests` | Strict endpoint expectations and one-request/one-diagnostic assertions require intentional matching-probe fixture updates with activation. |
| `cmd/root_request_rate_test.go`: `TestRequestRateCommandReporting` | Request counts, counter text, configured/inherited destinations, and invocation cleanup are observable, so activation tests cannot be reduced to renderer tests. |
| `cmd/update_job_batch_logging_test.go`: `TestBulkCompletionDiagnosticLogging` | Existing repeated-failure, partial-result, log-format, configured-destination, quiet/debug and exit-code fixtures provide meaningful coverage to extend, not duplicate wholesale. |

## Adjacent boundaries evaluated

### T001 → T002: complete checker versus shared operational wiring

This was the strongest plausible merge. It would eliminate the handoff from the checker/gate capability to `PrepareServiceDeps` and allow full adapter tests immediately.

**Decision: retain.** T001 already includes all its model/parser/interface/bypass/wrapper dependencies and failure handling; it is not a parser-only or interface-only task. Its substantial new behavior is one-attempt concurrency, callback publication, independently cancelable waiters, short child deadlines, request-body ownership, recursion bypass, safe failure classification, and private pacing evidence. T002 then must prove those properties survive actual `http.Client.Timeout`, existing retries and request-rate error replacement, legacy clients, and nested service construction. The cited timeout-observation regression demonstrates why this second proof is materially different from unit checker tests. Combining state-machine development with the complete adapter and real-client evidence matrix would make one iteration responsible for both protocol design and broad regression integration. The boundary keeps a stable executable checker as the input to a distinct, meaningful integration outcome; it is not justified merely by separate packages or file counts.

T002 is not split further: `ServiceDeps.APIClient`, all generated-client arguments, nested concrete-client retention, doubles, and actual wiring tests form one compiling contract. No adapter-per-version, facade-signature, or temporary-enable task is introduced.

### T002 → T003: integrated request path versus default CLI activation

A merge would remove a context-installation handoff and might appear to complete one end-to-end feature sooner.

**Decision: retain.** Actual shared-client coverage in T002 must coexist with standalone no-gate compatibility. Activating the flag/default in T003 changes strict command handlers, `api #` counters, quiet/debug and configured logger paths, connection-test warning deduplication, partial-result exits, and real-terminal eligibility. `root_api_diagnostics_test.go`, `root_request_rate_test.go`, and `config_test.go` contain concrete current assertions that would change immediately. Combining those execution/output regressions and docs generation with T002's timeout/auth/pacing/adapter proof would produce an oversized combined validation obligation. T003 remains substantial enough to own all user-visible behavior and its required fixes/tests together.

T003 is not split into a renderer task, flag task, global-enable task, later diagnostic hardening, or docs-only task. Such splits would either create thin mapping work, temporary activation, known missing acceptance, or repeated fixture updates. It reuses existing test harnesses and representative matrices rather than a full command × version × logging cross-product. This keeps the activation unit focused despite its mandatory acceptance breadth.

## Nonadjacent shared behavior and additional merge candidates

- **T001 and T003 parsing/outcomes**: One bounded advisory parser and one typed outcome are produced in T001, reused for rendering in T003. Legacy connection-test machine classification stays intact. No duplicate parser or outcome mechanics are scheduled; merging would also pull in the intervening T002 wiring and both adjacent boundaries' complexity.
- **Comparison and failure policy**: T001 includes both from the start. US2 does not rebuild a successful-check-only prototype. Story ownership expresses acceptance emphasis, not duplicated implementation.
- **Service and thin facade**: No public facade method/signature change is required. There is no facade-only task to merge; all generated-client wiring and required doubles are already together in T002.
- **Validation/frozen intent/preview**: This feature adds no mutation planning workflow. Local no-op, dry-run, confirmation, and output guardrail checks belong to T003; no separate setup/input-validation task is justified.
- **Single versus collection**: One invocation gate supports both from T001. T002/T003 prove sharing across workers/pages/polling; no single-item implementation followed by a bulk rewrite is scheduled.
- **Implementation plus tests/formatting/docs**: Every task owns tests and gofmt. T003 owns help, README, generated CLI docs, and final shared race validation with activation. No deferred failing-test or polish task remains.
- **Hypothetical parallelism**: No `[P]` marker is justified. The examples describe only safe validation concurrency after code is ready; global Cobra/environment tests remain sequential.

## Coverage and consistency checks performed

- Reviewed all 18 story acceptance scenarios against task detail: US1 comparison/version/invocation cases; US2 denied/timeout/malformed/partial/cancellation cases; US3 flag/local/discovery/output/logger/debug/docs cases all have owners.
- Verified all FR-001–FR-015 and SC-001–SC-006 appear in the ownership matrix and corresponding acceptance clauses. Full command acceptance is explicitly gated on T003; earlier internal checkpoints are not falsely described as a shippable CLI MVP.
- Programmatically checked all eight outcome/release model field constraints are quoted verbatim in T001, with probe/client/gate invariants carried into T001/T002.
- Confirmed three unchecked checklist rows, sequential T001–T003 IDs, one primary label each for US1/US2/US3, concrete paths, and no extra checkboxes or `[P]` flags. T002 depends on T001; T003 depends on T002; no forward dependency is needed to compile T001.
- Confirmed the enumerated constructor files exist and contain 36 targeted generated-client arguments. New implementation/test paths are explicitly prospective.
- Validated local artifact links, whitespace, placeholder absence, and the explicit no-live-test boundary. No feature progress needs preservation beyond the unchanged unchecked draft; no completed task was rewritten.
- No standalone consolidation-review section existed in tasks.md, so no review relocation occurred. No superseded-ID map is needed.

## Limitations and actual changes

Only this review file was added during consolidation; tasks.md was retained unchanged. Task generation and review are documentation-only work. Application tests, the docs generator, and live-engine checks were not run; planned test names still need implementation. No unresolved scope or dependency issue was identified, and no claim is made that the future implementation already compiles or passes its tests.
