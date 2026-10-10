# Tasks: Complete Smoke-Test Definition Cleanup

**Input**: Design documents in `specs/333-fix-smoke-test-cleanup/`.
**Branch**: `codex/333-fix-smoke-test-cleanup` | **Issue**: #333
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contract](contracts/smoke-test.md), [quickstart.md](quickstart.md).
**Tests**: Regression coverage is part of each implementation work unit, as required by repository guidance and the design. No separate test-only preparation tasks or TDD mandate.
**Format**: `- [ ] Tnnn [USn] Description with repository-relative file paths`. No task is marked `[P]` because shared files and producer/consumer dependencies require sequential execution.

## Phase 1: Setup (Shared Infrastructure)

No initialization task is needed: the repository, dependencies, feature branch, and design artifacts already exist. Reading feature context, checking the branch, and preserving existing changes are responsibilities of every work unit. Use `specs/ralph-implementation-rules.md` for Ralph execution. The published issue remains unverified; the specification records the conversation draft as its source.

## Phase 2: Foundational (Blocking Prerequisites)

No independent infrastructure is needed. Shared model changes belong to T001 in the earliest consuming story, together with converters and compilation-preserving test doubles. Do not create unrelated setup or model scaffolding phases that Ralph cannot execute as a story.

## Phase 3: User Story 1 — Clean Up the Complete Deployment (Priority: P1)

**Goal**: Remove every eligible deployed fixture definition and expose complete cleanup evidence without touching unrelated definitions.
**Independent test**: A three-definition deployment in each supported fixture family produces exactly three distinct-key deletions, leaves no created fixture resources after confirmation, and reports all three results. Older versions and other tenants remain untouched.

- [x] T001 [US1] Implement complete deployment tracking and safe collection cleanup across models, facade, and service, including the regression coverage below. Add the complete collection contract in `internal/domain/ops_smoke_test_model.go` and `c8volt/ops/model.go`, mechanical mappings in `c8volt/ops/convert.go`, and mapping/serialization coverage in `c8volt/ops/client_test.go` and `c8volt/ops/model_test.go`; introduce the deployed-definition identity, per-definition cleanup item, disposition, and optional summary described in `specs/333-fix-smoke-test-cleanup/data-model.md`, including all collection fields and existing singular projections. Preserve `ops.smoke-test.v1` and JSON tags `processDefinitions,omitempty`, `definitionScopeComplete,omitempty`, `definitionResults,omitempty`, and `definitionSummary,omitempty`. Enforce the model constraints “Existing singular deployment fields continue describing the main runnable fixture.”, “Existing singular eligibility, cleanup, and retained-definition fields remain main-fixture projections.”, and “A nil summary permits legacy render fallback only; freshly executed deployments populate summary and scope completeness explicitly.” Keep affected test doubles compiling, add Go comments, run gofmt on touched files, and validate with `go test ./c8volt/ops -run 'Test.*SmokeTest' -count=1`.

  In the same work unit, implement complete deployment capture and collection cleanup in `internal/services/ops/smoke_test_service.go` and focused `internal/services/ops/smoke_test_cleanup.go`, with regression fixtures in `internal/services/ops/smoke_test_test.go`; select the main fixture by exact BPMN ID, retain all returned evidence, initialize cleanup records before early returns, and after successful existing instance cleanup evaluate each exact key through existing paged eligibility and single-key deletion helpers. Build the collection state machine once, including blockers, error attribution, no-cleanup/no-wait, fail-fast/cancellation, and deterministic continuation; preserve nested options and existing force/not-found semantics without widening generic deletion APIs. Apply the quoted identity/state/count constraints below. Cover three returned definitions for every supported fixture family, one deletion per deduplicated key, preservation of unrelated versions/tenants, malformed identities, empty responses, and positive confirmation evidence; update existing one-definition stubs to realistic deployment responses where required. Include any affected smoke-test command fixtures in `cmd/ops_execute_smoke_test_test.go` to keep current tests passing. Add comments, gofmt, and run `go test ./internal/services/ops -run 'Test.*SmokeTest' -count=1` plus affected command tests if fixtures changed.

  T001 carries these exact data-model constraints:
  - “Deduplicate nonempty keys in first-seen order.”
  - “Conflicting nonempty identity/tenant metadata for a duplicate is an evidence error.”
  - “Missing tenant information stays unknown rather than being fabricated; retain existing deployment tenant fallback rules.”
  - “Missing key entries retain available identity evidence for reporting but are never mutation targets.”
  - “Validate that the expected fixture identities are represented; fixture inventory supplies validation, never replacement keys.”
  - “Results follow the deployment collection order. Initialize entries before cleanup so early returns preserve resource accounting.”
  - “State transitions: unattempted -> retained for no-cleanup or safety blocker; unattempted -> failed for eligibility/identity error; eligible attempt -> submitted after acceptance without wait; eligible attempt -> confirmed only after successful confirmation; eligible attempt -> failed on submission or confirmation error.”
  - “A failed confirmation can retain `Submitted=true` while `Confirmed=false`. Never overwrite completed entries on later errors.”
  - “Counts are mutually exclusive disposition counts; `Submitted` counts unconfirmed submitted entries, not every historical submission. Total equals the result-entry count.”
  - “Identity errors must be represented and cannot disappear from the total.”
  - “If an expected fixture is absent from the response entirely, record the scope error without inventing a deployed resource entry; scope incompleteness independently prevents confirmed aggregate status. A zero total never overrides a scope error.”
  - “Status precedence: failed for errors or unexpected unattempted work; skipped for explicit no-cleanup or safety retention without errors; submitted when otherwise successful but confirmation is skipped; confirmed only when the entire valid nonempty scope is confirmed.”
  - “Mixed skipped/submitted collections retain their separate counts. Dry-run has no actual deployment/result entries and no fabricated submitted resources.”
  - “Aggregate `Errors` includes dependency errors. The new summary, not the legacy projection, drives the overall cleanup stage and final outcome. Process-instance cleanup retains its existing model and behavior.”
  - “One deployment has many definition entries; each entry has exactly one cleanup result once deployment evidence is available. Report and result share the same collection facts through existing construction/mapping.”
  - “No-cleanup preserves incomplete evidence and must not imply a complete inventory when evidence is missing.”

- [x] T002 [US1] Integrate deployment-wide human and audit output in `cmd/cmd_views_ops_execute_smoketest.go`, update descriptive metadata in `cmd/ops_execute_smoketest.go`, and document complete cleanup in `README.md`; prefer the new aggregate whenever present, retain singleton fallback only for legacy values, show truthful counts and basic mixed-state summaries, and include all definition identities/outcomes in JSON and Markdown reports. Enforce “No renderer performs discovery.” Preserve existing envelope/error behavior, singular fields, omission/null rules, stage ordering, compact default output, and prompt wording. Add execution-path and report regression coverage in `cmd/ops_execute_smoke_test_test.go` and relevant metadata assertions in `cmd/command_contract_test.go`, including exact successful cleanup text, complete report collections, legacy fallback, and zero renderer requests. Add comments, gofmt, run `go test ./cmd -run 'Test.*SmokeTest' -count=1` plus any affected metadata test names, and regenerate CLI docs through `make docs-content` without hand-editing `docs/cli/`.

**Checkpoint**: US1 is a working collection implementation with safety guards and truthful base output; it is the MVP demonstration scope, not permission to omit later failure and control acceptance coverage.

## Phase 4: User Story 2 — Preserve Unsafe Definitions and Explain Partial Cleanup (Priority: P1)

**Goal**: Preserve precise failure/retention evidence across service, facade, and user reports while keeping unrelated workloads safe.
**Independent test**: A blocked dependency leaves eligible siblings removable; a later deletion/confirmation error preserves earlier successes and produces partial failure with all failed/unattempted identities accounted for.

- [x] T003 [US2] Complete fault-path evidence and actionable mixed-result reporting in `internal/services/ops/smoke_test_cleanup.go`, `internal/services/ops/smoke_test_service.go`, and `cmd/cmd_views_ops_execute_smoketest.go`, reusing the T001 collection algorithm; ensure available blocker/error reasons reach verbose and audit output, earlier-stage failure leaves known definitions unattempted, errors override skips, and a main-fixture success never hides dependency failure. Add end-to-end service/facade/command fault fixtures in `internal/services/ops/smoke_test_test.go`, `c8volt/ops/client_test.go`, and `cmd/ops_execute_smoke_test_test.go` for blocked dependency plus successful siblings, sparse-page eligibility, eligibility error, rejected submission, accepted response followed by wait error, pre-delete not-found, empty response, incomplete/conflicting identities, and cancellation/fail-fast with preserved prior results. Verify no mutation for blocked/invalid keys and no name-based fallback; use the existing real paged helper in at least one dependency safety case. Extend `README.md` and regenerate docs with `make docs-content` if explaining partial outcomes changes command metadata; keep default output compact and detailed resource evidence in verbose/audit output. Add comments, gofmt, and run targeted `Test.*SmokeTest` tests in the three affected packages plus `go test ./internal/services/processdefinition -run 'TestFindUnrelatedProcessInstancesForDefinition' -count=1`.

**Checkpoint**: Safety retention maps to `passed_cleanup_skipped`; errors after deployment map to `partially_failed` with existing error/exit policy. Every output format agrees on the same per-definition facts.

## Phase 5: User Story 3 — Preserve Existing Execution Controls (Priority: P2)

**Goal**: Make collection cleanup respect execution controls and keep progress and command streams accurate.
**Independent test**: No-cleanup submits zero cleanup mutations; dry-run submits zero mutations; no-wait accounts for all accepted definition requests without saying removed; unattended and interactive execution preserve their existing contracts.

- [ ] T004 [US3] Integrate execution controls and collection progress as one command-contract work unit. Integrate collection-level semantic progress in `internal/services/ops/smoke_test_cleanup.go` and `cmd/ops_execute_smoketest_progress.go`, with service progress assertions in `internal/services/ops/smoke_test_test.go` and activity assertions in `cmd/ops_execute_smoketest_progress_test.go`; freeze one total for the distinct tracked definition scope, suppress nested 1/1 stage resets, and emit actual confirmed/submitted/failed/skipped facts without mislabeling retained or unattempted resources. Preserve existing progress ownership, verbose routing, worker limits forwarded to nested operations, and DEBUG-only HTTP diagnostics; do not add a new worker pool or CLI lifecycle. Add comments, gofmt, and run `go test ./internal/services/ops -run 'Test.*SmokeTest' -count=1` and `go test ./cmd -run 'Test.*SmokeTest.*Progress' -count=1`.

  In the same work unit, finish command execution-control integration in `cmd/ops_execute_smoketest.go`, `cmd/cmd_views_ops_execute_smoketest.go`, and any required option propagation in `internal/services/ops/smoke_test_service.go`, with three-definition fake-server/subprocess coverage in `cmd/ops_execute_smoke_test_test.go`; exercise no-cleanup, no-wait, dry-run, count/workers, fail-fast, supported automation/auto-confirm combinations, human/verbose/JSON/quiet and quiet+JSON, plus existing keys-only support or rejection without introducing a new mode. Capture stdout/stderr separately, decode exactly one JSON envelope followed by EOF, assert mutation counts and no false removed/confirmed claims, and verify confirmation with real terminal stdin and configured/inherited stderr destinations where interactive execution is eligible. Preserve aborts, EOF/default answers, exit classification, and report-write failures. Align `README.md`, command metadata, and `specs/333-fix-smoke-test-cleanup/quickstart.md` with the resulting controls; regenerate CLI docs using `make docs-content`. Add comments, gofmt, run targeted `Test.*SmokeTest` tests for changed command/service packages and affected command-contract tests, and finish with `git diff --check`.

**Checkpoint**: All three stories satisfy their acceptance scenarios and preserve existing control semantics. No-wait can retain the existing `passed` outcome only with explicit submitted/unconfirmed cleanup evidence.

## Final Phase: Polish & Cross-Cutting Concerns

No standalone polish task: documentation, formatting, relevant checks, and progress tracking are included in each work unit. Before marking a unit complete, review its diff for layer ownership and command cohesion, record checks and material gaps, and preserve unrelated changes. For Ralph, update feature progress/memory as required by its workflow and commit validated work using Conventional Commits with subject suffix ` #333`.

The full race suite is conditional on shared-runtime impact or unresolved regressions under the constitution; do not run it merely to complete a checklist. Live quickstart checks need a disposable configured environment and are not prerequisites for documentation-only task generation. Report any live-validation gap without claiming cluster verification.

## Dependencies & Execution Order

```text
US1: T001 (models + service) -> T002 (output + docs)
                                          |
US2:                                    T003 (failure evidence)
                                          |
US3:                                    T004 (controls + progress)
```

The collection algorithm is built once in T001, including required safety and flag semantics. T003 completes fault-path integration and evidence; T004 completes command/control integration. Neither introduces another cleanup loop. Ralph must stay within one story per iteration; it may complete a validated subset of that story's tasks.

## Parallel Execution Examples

No parallel implementation tasks: each unit consumes the preceding result and shares files. Within US1, service and facade checks may run concurrently after T001 edits stabilize. Within US2 and US3, independent package checks may likewise run concurrently once edits are complete. Do not separate tests from the behavior they validate solely to enable parallel work.

## Implementation Strategy and Consolidation Review

Reduced from six tasks to four:

- Combined the former model/converter task with deployment capture and cleanup. These types have one immediate consumer, so implementing and validating them together avoids a scaffolding-only work unit.
- Combined progress and execution-control integration. Both describe the same command lifecycle and share service/command fixtures, so one unit avoids reopening those files and repeating validation.
- Kept output/docs separate from the service change: report formatting, compatibility, and generated documentation are independently verifiable and would make the service unit unnecessarily large.
- Kept failure integration as US2 to respect Ralph's single-story scope and retain a focused check of resource safety and truthful error reporting. Core safety behavior is already mandatory in T001; T003 does not defer it.

US1 remains the MVP demonstration scope. All four tasks are required to finish #333. Every task retains its implementation, fixtures, regression tests, formatting, and targeted checks. No requirements or meaningful tests were removed. Run each relevant check once after the combined edits stabilize; broaden or repeat only when later changes or failures warrant it.

## Requirement Coverage

| Requirements | Tasks | Acceptance evidence |
| --- | --- | --- |
| FR-001, FR-011, FR-012 | T001, T003 | Complete inventory, supported versions, malformed/missing identities |
| FR-002, FR-003, FR-004 | T001, T003 | Exact-key cleanup, tenant/version isolation, blocked/sparse-page safety |
| FR-005, FR-006 | T001, T002, T003, T004 | Dispositions, aggregate precedence, partial failures, explicit no-wait |
| FR-007 | T001, T002, T003, T004 | Public mapping, human/JSON/Markdown consistency, progress and streams |
| FR-008, FR-009, FR-010 | T001, T004 | No-cleanup, no-wait, dry-run, automation and output contracts |
| FR-013 | T002, T003, T004 | README, command metadata, regenerated docs, quickstart |

All task markers remain unchecked until their implementation and relevant checks succeed.
