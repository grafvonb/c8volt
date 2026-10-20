# Tasks: Bulk Job Completion

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [CLI contract](contracts/cli.md), [service contract](contracts/service.md), and [quickstart.md](quickstart.md).

**Branch**: `345-bulk-job-completion` | **Issue**: #345 | **PR base**: `main`

**Release baseline**: `v4.3.4`, commit `019e4c0bd865b9cf675422b4d8c97c489c36f694`. Do not merge develop, import #341, move release tags, or publish as part of these tasks.

**Tests**: Explicitly required by FR-001–FR-017 and SC-001–SC-007. Each implementation task includes its tests, formatting and targeted validation. Write the relevant regression/contract assertions first, demonstrate the failure, then implement and leave the work unit passing; do not commit a deliberately failing test-only intermediate task.

**Organization**: Nine cohesive work units. US1 owns the shared preparation/execution engine, including safeguards also exercised by US2/US3. US2 adds explicit collection CLI input to that same engine. US3 proves cross-layer reporting and terminal behavior rather than rebuilding the engine. Story labels identify acceptance ownership, not separate duplicate implementations.

## Execution Rules

- Read the feature artifacts, repository guidance and `specs/ralph-implementation-rules.md` as part of each work unit. Keep branch checks, progress updates and ordinary review inside the task, not separate tasks.
- Keep commands on public facades, facades as mechanical conversions, and paging/concurrency/reconciliation in internal services. Never hand-edit generated clients.
- Preserve compilation whenever an interface changes: include every implementation and affected test double in the same task. Locate additional implementers before editing interfaces; do not defer them to a later task.
- New/touched functions and tests need the comments required by repository guidance. Run gofmt on touched Go files and the specified targeted tests. Verify new test patterns match actual tests before accepting a run.
- User-visible changes include help/metadata, README and `make docs-content` in the same work unit. Keep final formatting in `cmd/cmd_views_job.go` and the batch lifecycle in `cmd/update_job_batch.go`.
- Respect existing options and errors, including explicit exit-code suppression. No implicit retry reset, incident resolution, process mutation or history deletion. No completion calls in any dry-run path.
- If following Ralph, complete only the current work unit, record evidence in `specs/345-bulk-job-completion/progress.md`, and commit only at the workflow's commit step. Issue-associated subjects must end with ` #345`.
- The commands below describe future implementation validation; task generation itself runs no runtime tests or Camunda operations.

## Phase 1: Setup

The repository, dependency stack, release branch and planning artifacts already exist. No initialization task is needed. Reuse existing Go/Cobra, configuration, worker-pool and test infrastructure; add no dependency or scaffolding solely for task structure.

## Phase 2: Foundational — Reliable discovery and submission evidence

**Goal**: Establish safe version-specific primitives without changing legacy completion behavior. These tasks block the user-story engine.

- [x] T001 Correct tenant scope and page-completeness evidence in `internal/services/job/v88/convert.go`, `internal/services/job/v89/convert.go`, `internal/services/job/v810/convert.go` and the matching `service.go` files, with additive raw-page metadata in `internal/domain/job.go` and mechanical mapping in `c8volt/job/model.go` / `c8volt/job/client.go` where needed. Add regressions first in each version's `service_test.go`: named tenant equality; empty/all-tenant omission; unfiltered explicit `GetJob`; correct scoped totals; raw-offset advancement before local filtering; exact versus lower-bound totals; sparse/empty pages with later continuation; ambiguous or nonadvancing metadata; and preserved overfull trimming. Shared page/collected/total callers must continue when supported or return an explicit incomplete error, never reinterpret indeterminate continuation as successful exhaustion. Enable stable key ordering for bulk use without changing ordinary read order. Update read-command scope guidance in `cmd/get_job.go` and `README.md`, regenerate CLI docs, and run `go test ./internal/services/job/... -run 'Test.*(JobSearch|SearchJobs|GetJob)' -count=1`, relevant facade/command search tests, and `git diff --check`.

- [x] T002 Add a bulk-only one-shot completion primitive in `internal/services/job/v88/completion.go`, `internal/services/job/v89/completion.go`, and `internal/services/job/v810/completion.go`, plus submission evidence in `internal/domain/job_completion.go`. Preserve this model constraint: “`CompletionSubmission` retains `status` (`not_submitted`, `accepted`, `rejected`, `uncertain`), optional HTTP status, and normalized diagnostic evidence.” Use existing generated response methods and shared payload construction without changing `SubmitJobWorkerOutcome`'s legacy retry policy. Add `completion_test.go` in each adapter proving accepted success, explicit backend rejection (including 404/409/backpressure), ambiguous HTTP/transport outcomes, and exactly one bulk submission even on timeout; preserve task-listener payload restrictions and existing single-key tests. “An uncertain submission cannot become rejected merely because a later read is missing.” Keep the primitive private to the adapter/coordinator boundary and avoid changing shared retry policy. Run `go test ./internal/services/job/... -run 'Test.*(BulkCompletion|SubmitJobWorkerOutcome)' -count=1` after formatting; no CLI documentation change is needed for this internal-only primitive.

**Checkpoint**: Tenant/page evidence and non-replayed submission evidence are available across 8.8/8.9/8.10; earlier-version job support and single-key behavior remain unchanged.

## Phase 3: User Story 1 — Complete a Selected Listener Backlog (P1)

**Goal**: Preview and execute a tenant-scoped, frozen selection of pending listeners in one command with safe outcomes from its first usable version.

**Independent test**: A controlled mixed-tenant/mixed-job backlog spanning multiple pages produces only the intended unique candidates; no writes occur until preparation finishes; one eligible confirmation precedes bounded execution. Empty, incomplete and dry-run selections submit nothing, and new sequential listeners are excluded.

- [ ] T003 [US1] Implement read-only batch preparation and its complete data contract in `internal/domain/job_completion.go` and `internal/services/job/completion/prepare.go`, expose `PrepareJobCompletionBatch` through `internal/services/job/api.go`, `c8volt/job/api.go`, `c8volt/job/model.go`, `c8volt/job/completion.go` and `c8volt/job/convert.go`, and wire all version services including explicit unsupported v87 behavior in `internal/services/job/v87/service.go`. Build both search and explicit-collection preparation now so US2 reuses it. In `prepare_test.go` and `c8volt/job/completion_test.go`, test deterministic first-seen/input order, scope/predicate verification, raw-page continuation, deduplication before the unique limit, intentional limit completion, incomplete discovery/cancellation with zero writes, empty stdin/search, all explicit reads before execution, cross-tenant authorized keys, payload restrictions, copied maps/slices and option/error propagation. Include all affected interface test doubles. Run `go test ./internal/services/job/... -run 'Test.*(BulkCompletion|JobSearch)' -count=1`, `go test ./c8volt/job -run 'Test.*BulkCompletion' -count=1`, and `go test ./... -run '^$'` for interface compilation. Bind the following data-model constraints verbatim in the implementation and its assertions:
  - Selection `mode`: “`explicit` or `search`”; `keys`: “Validated explicit string keys; input order retained before deduplication”; `filter`: “Existing normalized job-search predicates; retries remains optional, including explicit zero”; controls: “Search controls; zero limit means uncapped using existing conventions”; `inputForm`: “Single scalar, collection, stdin or filters; preserves rendering compatibility”.
  - “Keys and filter selection are mutually exclusive.” “Nonempty variables cannot accompany task-listener batch candidates.” Worker count/options remain separate from provenance.
  - Prepared batch “Completion reason: `exhausted`, `limit_reached`, or `explicit_inputs_read`.” “Preparation status: `complete` or `incomplete`; page count and observed count for diagnostics.” Preserve requested limit, bounded-selection evidence, payload, actual tenant context and normalized scope.
  - Candidate fields: “key, process-instance key, element-instance key/ID, tenant, type, kind, listener event, observed state and retries.” “Include only available facts; no fabricated lifecycle dates.”
  - “A complete batch may contain zero candidates. An incomplete batch is never executable.” “The target limit counts unique matching candidates, regardless of whether the backend will accept completion.” “Duplicate and out-of-scope rows do not consume it.”
  - Observation: “`status` (`completed`, `not_completed`, `missing`, `unreadable`, `unknown`, `skipped`), last known state, and phase (`recheck` or `reconciliation`) when applicable.” Define the final disposition partition `accepted`, `completed`, `unsatisfied`, `not_started` and reasons “`scope_changed`, `other_terminal`, `unsupported_state`, `missing`, `unreadable`, `payload_incompatible`, `rejected`, `uncertain`, `fail_fast`, `declined`, and `interrupted`”.
  - Report: “Phase/status: `preview`, `finished`, `aborted`, `preparation_failed`, or `interrupted`.” “Counts: `selected`, `accepted`, `completed`, `unsatisfied`, `notStarted`.” Preserve aggregate error separately from the payload.
  - “The new bulk payload uses `candidates` for preview and `outcomes` for execution. The applicable collection is always present and encoded as `[]` when empty; the inapplicable collection is omitted. Counts are numeric and present even when zero. Do not encode an empty collection as `null` or fabricate a per-key row.”

- [ ] T004 [US1] Implement the shared bounded execution/reconciliation workflow in `internal/services/job/completion/execute.go` and `internal/services/job/completion/reconcile.go`, exposing `ExecuteJobCompletionBatch` through `internal/services/job/api.go`, `c8volt/job/api.go` and the thin `c8volt/job/completion.go` / `convert.go` mapping with matching version delegations and unsupported v87 handling. Reuse T002's one-shot submission and the existing pool/worker-count helpers. Reject incomplete/invalid plans and changed discovery configuration; `WithDryRun` returns preview without calling submission even for direct facade callers; force/no-state-check options cannot bypass scope or payload guards. Recheck selected keys, treat observed COMPLETED as a no-op, preserve missing/read/scope/terminal/unknown outcomes, allow only the planned known nonterminal observations without inventing a positive-retry prerequisite, and reject changed listener payload compatibility. Reconcile 404/409 or uncertainty using existing backoff timeout/caller deadline; no-wait allows one read and no polling; accepted responses explicitly skip independent confirmation. Preinitialize all rows, track started work, preserve index order, stop new scheduling on fail-fast, and retain caller cancellation even when pool error is nil. In `execute_test.go`, `reconcile_test.go` and `c8volt/job/completion_test.go`, prove race outcomes, no replay, no replacement/new listener keys, finite concurrency, default continue, fail-fast/in-flight/unstarted reporting, report-plus-error mapping and no hidden mutations. Preserve “selected = accepted + completed + unsatisfied + notStarted”, “len(outcomes) = selected”, and “An interruption remains an invocation failure even if all already-returned rows are satisfied.” Run the targeted bulk-completion service/facade tests, `go test ./... -run '^$'`, and race-enabled tests for `./internal/services/job/completion`.

- [ ] T005 [US1] Enable selector completion end to end in `cmd/update_job.go`, `cmd/update_job_request.go`, `cmd/update_job_batch.go` and `cmd/cmd_views_job.go`, reusing/factoring validation from `cmd/get_job.go` without mutable cross-command flag coupling. Add all nine documented AND predicates, search controls and existing execution options; interpret explicit `--retries 0` as a selector while retaining keyed completion rejection; reject unqualified completion, control-only selection, unsupported flags and selection for unrelated mutation modes. Keep a single scalar key on the legacy path. Plan once, preview/confirm the frozen selection once through configured stderr, execute once, and render one envelope with report-plus-error before the exit-after-render path. Implement all human/verbose/JSON/quiet/empty/no-wait and abort cases from the contracts from this first CLI increment; no keys-only mode or extra paging prompt. Add request/execution/view/exit assertions in `cmd/update_job_batch_test.go`, `cmd/update_job_request_test.go`, `cmd/update_job_outcome_test.go` and `cmd/command_contract_test.go`, covering option propagation, zero writes on dry-run/incomplete/decline/empty, listener consequence text, deterministic partial results and unchanged single-key modes. Update capability metadata, help and `README.md`, run `make docs-content`, and validate with `go test ./cmd -run 'Test.*(UpdateJob|BulkCompletion|CommandContract)' -count=1` and `git diff --check`. T007/T008 add scale and terminal acceptance depth; do not defer basic safe output to them.

**Checkpoint / functional MVP**: Filter-based listener recovery is usable with frozen tenant scope, complete reporting and strict dry-run. Broader acceptance and live recovery evidence remain required before release.

## Phase 4: User Story 2 — Complete an Explicit Set of Jobs (P2)

**Goal**: Run reviewed key sets through the existing batch engine without changing ordinary single-key updates.

**Independent test**: Repeated/comma keys plus newline stdin deduplicate into one ordered set, use one confirmation decision and submit each eligible key once. Invalid/unreadable input blocks the entire preparation; one scalar key still returns the original payload.

- [ ] T006 [US2] Extend only the CLI input/routing layer in `cmd/update_job.go`, `cmd/update_job_request.go` and `cmd/update_job_batch.go` to accept repeated/comma-separated `--key` plus one positional `-`, invoking T003/T004 for every collection. Decide collection mode before deduplication; flags precede stdin; blank stdin lines are ignored but empty key/comma tokens and malformed/nonpositive/out-of-range keys fail; preserve key strings without floating-point conversion. Deliberate empty stdin succeeds empty, whereas absent selection fails. Reject explicit/filter mixtures, search-only controls on explicit input and collections for unrelated update modes. Preserve explicit keys across authorized tenants and their actual context. Extend `cmd/update_job_batch_test.go`, `cmd/update_job_request_test.go`, `cmd/update_job_outcome_test.go` and `cmd/command_contract_test.go` for duplicates reducing to one, stdin read errors, all-input validation before writes, cross-tenant keys, empty input, every legacy update mode, schema stability and automation/JSON restrictions. Update help/capabilities and `README.md`, regenerate CLI docs, and run `go test ./cmd -run 'Test.*(UpdateJob|BulkCompletion|CommandContract)' -count=1` plus `git diff --check`. Do not implement a second collection engine or relocate deduplication/paging into the facade.

**Checkpoint**: Both selector and explicit reviewed-set workflows use the same prepared/executed batch contracts; the scalar-key contract remains compatible.

## Phase 5: User Story 3 — Operate a Batch with Trustworthy Outcomes (P2)

**Goal**: Demonstrate the integrated guarantees under scale, races, partial failure and real interactive I/O. Required safeguards already ship with T003–T006; these tasks add distinct cross-layer acceptance evidence rather than test copies of helper implementation.

**Independent test**: A mixed-success 1,000-job execution accounts for every frozen target with bounded concurrency and one envelope; real terminal stdin proves one stderr confirmation and uncontaminated stdout, including redirected output and prompt-free empty results.

- [ ] T007 [P] [US3] Add cross-layer HTTP acceptance coverage in `cmd/update_job_batch_acceptance_test.go`, with a reusable fixture in `testx/job_completion_server.go` only if multiple test files genuinely share it. Exercise at least 1,000 mixed jobs over multiple pages and record request ordering, counts and peak concurrency; require complete preparation before the first mutation and no duplicate/out-of-scope submissions. Exercise sparse/duplicate/contradictory pages, limits, terminal/filter races, missing/read failures, rejection/transport uncertainty with both completed and missing reconciliation reads, fail-fast and caller interruption. Cover normal/dry-run, human/verbose/JSON/quiet plus auto-confirm/automation/no-wait combinations, including authoritative empty search/stdin and failed preparation; decode one envelope and require EOF, preserve exact empty-array/omission semantics and partial payload on nonzero exit. Parameterize supported adapter versions and retain explicit v87 unsupported checks. Include sequential-listener exclusion and listener-variable zero-write failures. Run `go test ./cmd -run 'Test.*BulkCompletionAcceptance' -count=1`; fix any discovered behavioral defect in its owning layer with a focused regression and rerun affected checks.

- [ ] T008 [P] [US3] Add genuine terminal acceptance coverage in `cmd/update_job_batch_terminal_test.go` using `testx/cmd_terminal_runner.go` and existing subprocess exit helpers. Capture stdout/stderr separately for ordinary, verbose, JSON, quiet+JSON and redirected stdout runs; verify configured/inherited stderr destinations, at most one mutation prompt, exact listener consequence and candidate count, accepted/declined/EOF behavior, and no paging prompt. Verify empty search and empty stdin finish without confirmation or mutation in normal/dry-run/no-wait cases, and auto-confirm/automation remain prompt-free. Check partial-failure JSON has one envelope and no later error envelope or logger-prefixed control text; keep explicitly requested machine output under quiet. Run `go test ./cmd -run 'Test.*BulkCompletionTerminal' -count=1`; pipe-only mocks do not satisfy this task. Correct any proven routing defect at its owning layer and revalidate affected command tests.

**Checkpoint**: SC-001–SC-005 and scripted/interactive portions of SC-007 have explicit execution-path evidence. No claim of live process termination follows from these HTTP/terminal tests.

## Phase 6: Polish & Cross-Cutting Validation

**Goal**: Prove representative listener recovery separately from submission acceptance and record release-readiness evidence without publishing.

- [ ] T009 Build and validate the disposable cancellation-listener hierarchy using new feature-local fixtures `specs/345-bulk-job-completion/fixtures/canceling-listener-parent.bpmn` and `specs/345-bulk-job-completion/fixtures/canceling-listener-child.bpmn`; update `specs/345-bulk-job-completion/quickstart.md` with exact runnable deployment/start/cancel/read/cleanup commands for those fixtures and record evidence in `specs/345-bulk-job-completion/validation.md`. Use multiple child instances, a unique listener job type with no worker, and a nonmatching control; independently observe pending listeners, dry-run nonmutation, bulk results and eventual job/process terminal states on an authorized disposable environment. Record the exact live engine versions exercised; use automated v88/v89/v810 coverage for parity and do not claim unrun live checks passed. Cleanup only recorded fixture resources. If no live environment is available, record SC-006 as unresolved and leave this task incomplete rather than substituting mocked proof. In this same delivery-validation unit, confirm help/capabilities/README/generated CLI docs match shipped behavior, run remaining quickstart checks, `make test` and `make vet` once for the changed cross-package/concurrent behavior, and inspect `git diff --check`; do not rerun successful suites solely for subsequent prose edits. Record branch/main baseline and actual feature-branch CI evidence when available; no merge, push, release number choice or publication is implied.

## Dependencies & Execution Order

```text
T001 -> T002 -> T003 -> T004 -> T005 -> T006 -> T007 -> T009
                                                 -> T008 -> T009
```

- Setup has no new work. T001/T002 are sequential because both touch version adapters and tests.
- US1 is T003–T005, after foundations. Preparation and execution split at a real read-only boundary; each introduces complete interface wiring and tests, never noncompiling consumers or placeholder success implementations.
- US2 is T006 and depends on the existing engine and selector command. Its input form and compatibility acceptance remain independently testable.
- US3 is T007/T008 after T006. They are independently executable acceptance suites with distinct HTTP/scale versus terminal/process mechanics.
- T009 follows all stories; it couples fixture creation, exact runnable guide, real observations and cleanup because those establish one recovery-validation outcome.

## Parallel Execution Examples

**US1**: No safe parallel implementation pair. Adapter wiring, contracts and command dispatch depend on preceding work; do not split the three version adapters into artificial parallel tasks.

**US2**: One cohesive command-input task; parallel edits to its parser, flags or legacy-mode tests would conflict. Test cases can be parameterized within that work unit.

**US3**: After T006, T007 can develop/run `cmd/update_job_batch_acceptance_test.go` while T008 develops/runs `cmd/update_job_batch_terminal_test.go` in independent test processes. Their primary files do not overlap. Serialize any discovered production-code fixes or shared helper changes, then rerun the affected suite. `[P]` permits this opportunity; it does not authorize an autonomous multi-agent run.

## Requirement Coverage

| Requirement | Tasks and evidence |
|---|---|
| FR-001 | T003, T006 explicit collections/stdin and shared preparation |
| FR-002 | T003, T005, T006 selector/key grammar and explicit zero retries |
| FR-003 | T001, T003, T004, T006 tenant-safe search and explicit authorization |
| FR-004 | T001, T003, T007 paging, sparse results, deduplication and limits |
| FR-005 | T003, T004, T007 complete preparation and frozen membership |
| FR-006 | T003–T005, T007 preview context, listener consequences and zero writes |
| FR-007 | T005, T008 one stderr confirmation, automation and decline |
| FR-008 | T004, T007 recheck and truthful changed/terminal/missing outcomes |
| FR-009 | T004, T007 worker bounds, fail-fast, interruption and unstarted keys |
| FR-010 | T002, T004, T007 one-shot evidence and bounded reconciliation |
| FR-011 | T004, T005, T007 count conservation, errors and preserved results |
| FR-012 | T005–T008 one envelope, stable ordering and single-key compatibility |
| FR-013 | T003–T008 authoritative empty result, quiet/JSON/no-wait behavior |
| FR-014 | T002–T006 payload restrictions and unrelated update compatibility |
| FR-015 | T003, T004, T007 no repair/deletion/rediscovery and no process-success inference |
| FR-016 | T001–T004, T007 supported adapters and v87 unsupported preservation |
| FR-017 | T001, T005, T006, T009 help, metadata, README and generated docs |
| SC-001 | T007 controlled 1,000-job multipage invocation |
| SC-002 | T003, T004, T007 exact scoped membership and request counts |
| SC-003 | T003–T008 zero-write failure/preview/decline/empty paths |
| SC-004 | T004, T007, T008 partial results and nonzero unsuccessful exit |
| SC-005 | T005, T007, T008 one prompt/envelope and unattended behavior |
| SC-006 | T009 independently observed live hierarchy recovery |
| SC-007 | T001–T008 legacy modes and three supported adapters |

## Implementation Strategy and Consolidation Review

1. Finish the two foundational primitives with existing behavior preserved.
2. Complete US1's preparation, full safe executor and selector CLI. This is the functional MVP for the incident backlog; safeguards from US3 are implemented here from the start.
3. Add US2's explicit reviewed-key input without rebuilding backend behavior.
4. Complete US3's integrated acceptance evidence and T009's live recovery/release validation before claiming the feature complete or release-ready.

No standalone branch checks, model-only scaffolding, failing test commits, per-version duplicate workflows or documentation catch-up tasks are included. Each implementation unit includes its relevant tests and documentation; final checks exist within the concrete recovery-validation outcome.

Deliberate boundaries: T001/T002 address independent discovery versus submission contracts; T003/T004 separate read-only preparation from mutation/reconciliation; T005/T006 separate selector enablement from explicit input compatibility; T007/T008 exercise different I/O mechanisms; T009 needs real engine observations and owned fixtures. Combining these adjacent units would create larger cross-layer or environment-dependent iterations without removing duplicated implementation.

Task count: **9** — setup **0**, foundational **2**, US1 **3**, US2 **1**, US3 **2**, cross-cutting **1**. No task has begun. Optional commit and Ralph execution hooks do not run automatically.
