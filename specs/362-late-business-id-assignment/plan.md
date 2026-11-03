# Implementation Plan: Late Business-ID Assignment and Atomic Job Completion

**Branch**: `362-late-business-id-assignment` | **Date**: 2026-11-02 | **Spec**: [spec.md](spec.md)

**Input**: `specs/362-late-business-id-assignment/spec.md`

## Summary

Add `--set-bid` to exact-key process assignment and existing job completion. Use the already generated native 8.10 assignment and atomic completion requests. Preserve selection semantics, frozen approval payloads, ordered partial evidence, and omitted-input compatibility. Treat BID-bearing mutation as one-shot: timeout or lost response cannot trigger replay or be converted to assignment success by observing a completed job.

## Technical Context

**Language/Version**: Go 1.26.0, toolchain go1.26.2 (`go.mod`).

**Primary Dependencies**: Existing Cobra/pflag/Viper CLI, generated Camunda adapters, standard library, `toolx/pool`, call options and backoff helpers. No new dependency or generated-client change expected.

**Storage**: Invocation-local copied plans and reports; native BID persists in Camunda. No new local persistence.

**Testing**: Go tests, testify, generated-client doubles, test-owned HTTP servers, isolated CLI/subprocess and real-terminal fixtures. Planning-only changes receive document checks.

**Target Platform**: Existing supported CLI/library platforms; native assignment on Camunda 8.10 and later supported adapters; explicit rejection on 8.8/8.9.

**Project Type**: CLI plus public Go facade library.

**Performance Goals**: One submission per distinct target, bounded existing worker policy, preserved batch/limit semantics, and no extra discovery from rendering. Observe accepted direct assignments within existing configured backoff rather than introducing a new latency target.

**Constraints**: Validate all input before mutation; freeze selection and payload before approval; preserve exact strings and JSON shapes; never redirect to roots; no replay of ambiguous assignment; backend eligibility authoritative; no live-environment implementation validation.

**Scale/Scope**: Two existing command families; scalar, collection and stdin direct assignment; scalar, explicit collection, stdin and selector completion; all adapter policies and corresponding public paths. No other mutation gains assignment.

## Constitution Check

*Pre-research and post-design gates both pass.*

| Principle / repository gate | Before research | After design |
| --- | --- | --- |
| Operational proof | Separate intent, submission and observation | Direct normal execution observes accepted assignment; no-wait labels skipped observation; atomic acknowledgement is accepted evidence; uncertainty remains unknown |
| CLI-first, script-safe | Extend existing commands | Preserve scalar/bulk shape, mode restrictions, tenant discovery, exact zero-retry selectors, output precedence and original error class |
| Proportionate validation | Documents only in planning | Isolated targeted tests with each behavior unit; one full race suite justified by shared coordinator/interface/concurrency changes |
| Documentation matches behavior | README and generated docs required | Change help/examples and regenerate with `make docs-content` in delivery units |
| Small compatible changes | Reuse current layers | Thin facades; service-owned planning/pools/waits; existing native transport and optional serialization; no generic framework |
| Logger / prompt contracts | Required on all outcomes | Command-execution tests for configured destinations, compact failures, INFO suppression, redacted DEBUG, one envelope and real terminal prompts |
| Implementation testing boundary | No live targets | All validation uses owned fakes; real-engine eligibility is a documented coverage limitation |

No unjustified gate failure or unresolved design clarification remains. The post-design checks apply to the contracts and validation guide below as well as this plan.

## Project Structure

### Documentation (this feature)

```text
specs/362-late-business-id-assignment/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    ├── cli.md
    └── public-api.md
```

`tasks.md` is the next phase's output and is not created by this command.

### Source Code (repository root)

```text
cmd/
  update_processinstance.go                  # flags and assignment dispatch
  update_processinstance_assignment.go       # new assignment command lifecycle
  cmd_views_processinstance_assignment.go    # new final views / aggregate formatting
  update_job{,_request,_outcome,_batch,_plan}.go
  cmd_views_job{,_logging}.go
c8volt/
  api.go                                    # facade aggregation / affected doubles
  process/{api,client,model,convert}.go
  process/business_id_assignment.go          # new thin plan/execute delegation
  job/{api,client,model,convert,completion}.go
internal/domain/
  business_id.go                            # separate assignment validator
  process_instance_business_id_assignment.go # new version-neutral contracts
  job.go
  job_completion.go
internal/services/
  processinstance/
    api.go
    business_id_assignment.go               # new collection coordinator
    waiter/                                 # bounded exact-key observation
    v88/ v89/ v810/                          # explicit policy, native adapter
  job/
    completion/{prepare,execution_plan,execute,reconcile}.go
    v88/ v89/ v810/                          # all ports/policies updated together
  common/job_completion.go                   # existing submission evidence classifier
README.md
```

**Structure Decision**: Keep ownership within existing packages. New process assignment has enough lifecycle declarations to require a dedicated mode file. Job changes extend its existing focused files; isolate any new sizeable assignment-only lifecycle instead of expanding base command construction. Views own final formatting. Generated clients remain read-only.

## Phase 0 — Research Decisions

[research.md](research.md) resolves transport support, validator semantics, owner layers, copied preparation, replay policy, completion attribution, serialization, and validation. Source inspection found two concrete hazards: scalar completion's retry helper and bulk reconciliation's completed-is-satisfied rule. Both need conditional assignment-aware changes while preserving plain completion behavior.

## Phase 1 — Design

### Input and adapter policy

Introduce an optional requested assignment pointer named `SetBusinessID` in public and domain job inputs, separate from observed `BusinessId`. New process assignment requests require an explicit value. Assignment-specific validation delegates UTF-8/UTF-16 counting to the existing validator and rejects whitespace-only input without trimming accepted strings. Validate CLI, public-service entry points, and prepared execution so callers cannot bypass policy with a forged batch. Leave creation and search semantics unchanged.

Version adapters expose explicit capability/value validation and exact-key assignment. v88/v89 return the established unsupported error without requests; v810 sends the generated native instruction. Do not infer eligibility from previews or add configuration probes for uniqueness.

### Direct process assignment

Expose `PlanProcessInstanceBusinessIDAssignments` and `ExecuteProcessInstanceBusinessIDAssignments` on the process facade and aggregate API. Match existing `typex.Keys`, `foptions`, workers, conversions, and `ferrors` conventions. Keep interface implementations, embedded aggregate APIs, constructors and test doubles compiling in the same unit.

The service normalizes the full key list, deduplicates by process key in stable order, copies the requested value, and optionally captures exact-instance facts for preview. Treat absent/failed planning reads as errors rather than silently shrinking scope. A completed explicit empty stdin scope returns a successful empty plan; no-key invocation without deliberate stdin remains existing invalid input. Do not search variables, traverse families, redirect a child, or drop an ineligible candidate silently.

The command dispatches assignment before mandatory variable parsing and rejects assignment/selector/payload conflicts. Freeze and validate the entire plan before confirmation; a public plan is invocation-local data, not an authorization token. Execution revalidates readiness, keys, payload and adapter policy, copies it, uses the existing worker policy and fail-fast conventions, and submits once per distinct target. Exact-key authorization stays unchanged. No mutation or observation occurs for a dry-run or completed empty plan.

For known acceptance, normal execution polls the exact instance until its observed BID matches, or returns accepted-but-unconfirmed evidence with a failure. No-wait skips confirmation reads. A rejection is a failure even if identical BID is later visible. Lost-response submission stays unknown with the original error; later matching observations cannot prove attribution. Observation never triggers mutation replay. Preserve ordered skipped/not-started slots under fail-fast and interruption.

### Atomic job completion

Extend existing worker request, completion request, prepared batch, update plan, report and result conversions with copied requested intent and optional assignment evidence. Keep job/candidate snapshots unchanged. Both public/domain boundaries and preparation/execution snapshots copy BID pointers and nested variables. Scalar BID-bearing plans must freeze the approved payload too; execution must use that snapshot rather than a caller-mutated request or stale plan attachment.

Extend the coordinator's private submit port with a typed completion payload holding variables and optional assignment. Update v88/v89/v810 ports, mocks and callers together. In v810 populate generated body BusinessId and continue using `completeJobWithResponse`, which already omits nil optional fields and preserves other fields. Preserve existing task-listener variable restrictions and lease-token serialization behavior; this feature does not introduce new lease-token inputs.

For scalar BID-bearing completion, bypass `RetryCamundaMutation` and classify a single response using existing submission evidence conventions. Non-BID scalar retry behavior stays unchanged. Bulk remains one-shot. Preserve `--retries 0` as an optional exact selector for keyless completion, full tenant-scoped discovery, complete frozen membership, and job-key deduplication. Never deduplicate distinct jobs by process key.

Make pre-submit completed-state handling and reconciliation assignment-aware. A completed job before submission cannot satisfy an assignment request. After uncertain or rejected submission, recording COMPLETED is valid completion evidence but cannot turn assignment into satisfied or clear the original error. Accepted native atomic completion can report assignment accepted without pretending its old snapshot is refreshed. Do not add mandatory process lookups as a shortcut to attribution.

### Output and error flow

Use optional `requestedBusinessId` and assignment evidence fields; observed snapshot fields retain their meaning. The [public contract](contracts/public-api.md) defines transitions. Preserve scalar results for scalar input and bulk results for collection input, even after deduplication. Preserve existing keys-only behavior, including rejection for bulk job completion; the feature adds no unsupported mode.

Trace failures: native adapter returns original classified error and evidence → coordinator preserves ordered partial results → facade maps both result and error → view emits selected-format result once → configured logger emits one bounded aggregate human error where applicable → `exitAfterRenderedResult` preserves original exit classification. Scalar BID-bearing failures currently lose returned evidence; add the result-preserving branch. Do not render another envelope afterward. Do not use cancellation-specific process failure text for assignment.

Human mutation/dry-run summaries and verbose facts use `renderHumanLine`/configured logger. Prompts are plain configured stderr. Normal and verbose failures group bounded known causes, never raw generated/transport errors; redacted DEBUG remains independent of verbose. Empty views preserve established payload/null conventions and quiet+machine precedence without claiming accepted work.

### Compatibility and documentation

All assignment fields are optional in existing job shapes and omitted when unused. New public interface methods require external custom implementations to add methods; document that source impact and additive struct-field implications. Existing selection, variable updates, plain completion, creation-time whitespace behavior and unsupported mode combinations remain unchanged.

Update source metadata, README and examples for selection versus assignment, eligibility, uniqueness disabled, exact strings, atomicity, conflicts including identical values, dry-run races, observed snapshots, no replay and no-wait evidence. Regenerate CLI docs rather than hand-editing them. Implementation examples remain descriptive; implementation validation never executes them against a live engine.

## Validation Strategy

[quickstart.md](quickstart.md) defines isolated runnable checks and scenario expectations. Planning validation checks paths, links, template completion, requirement coverage and whitespace; no runtime suite is needed for this documentation-only turn.

Implementation starts with targeted tests in changed packages, formatted touched Go files, and affected docs generation. After all behavior units pass, run one full race-enabled `make test` because this changes shared completion coordination, interface implementations and concurrent batch execution. Verify targets and inherited settings before running; integration-tagged/live suites are excluded.

Require exact wire capture and omission; assignment Unicode boundaries; public validation/unsupported versions; prepared pointer/map isolation; two jobs in one process; preexisting completed jobs; native rejection/uniqueness/identical reassignment; exactly-one lost-response submissions; completed-state uncertainty; frozen selection, sparse pages and discovery failures; dry-run/no-wait/no-op/partial outcomes. Exercise actual command execution, not only views.

Output tests must detect direct-print replacement and raw aggregate leakage. Capture configured/inherited destinations in plain and structured formats, WARN/quiet INFO suppression, repeated distinctive HTTP errors, DEBUG flag and configured level with redaction, stdout purity, one-envelope+EOF, original exit code, supported keys semantics, and real terminal confirmation/abort/EOF and prompt-free empty selections. Retain representative non-assignment regressions.

## Work-Unit Guidance for Task Generation

Build collection behavior once. Proposed cohesive outcomes, not tasks:

1. Direct assignment end to end (US1): assignment validator, domain/public/service interfaces and all adapters/doubles, collection planner/executor/observation, CLI lifecycle/views, required diagnostics/empty/terminal tests and documentation. If too large for one iteration, split a compiling service/facade slice from CLI integration, with the same primary story and an independently verifiable contract.
2. Atomic completion propagation (US2): all model/converter/clone/port changes together, both scalar and bulk one-shot submission, assignment-aware reconciliation and original-error preservation, wire/immutability/uncertainty tests, CLI mode handling and docs. Split service delivery from CLI integration only if practical iteration size requires it; do not separate evidence correctness from submission.
3. Cross-cutting output coverage (US3) and validation compatibility (US4) belong inside each relevant behavior unit, not deferred cleanup. Broader regression verification can be one final independent validation outcome after both workflows exist.

Task generation must consolidate adjacent coupled work, retain every acceptance scenario, keep interfaces and implementations compiling together, and respect Ralph's single-primary-story iteration rule. Routine reading/branch checks/progress are responsibilities, not standalone tasks. No implementation task needs live-engine validation.

## Complexity Tracking

No constitution exception is required. A focused process coordinator and optional independent assignment evidence are necessary to avoid redirecting scope or confusing completion state with one-time assignment. Existing pools, prepared batches, adapters, serialization and logger helpers are simpler than a new generic mutation framework or persistent approval registry.
