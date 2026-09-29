# Tasks: Update User-Task Variables

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [CLI contract](contracts/cli.md), [service contract](contracts/facade-service.md), [quickstart.md](quickstart.md); issue #326.

**Execution approach**: Eight substantive work units, each including its implementation, fixtures, tests, and targeted validation. Build one collection-based workflow from the start; a single task is a collection of one. Do not implement a temporary single-task workflow and then rebuild it for bulk use.

The three specification stories remain acceptance criteria. Story labels below identify the primary delivery concern for Ralph's selection; shared infrastructure is implemented once under US1, command integration under US2, and full output/operational acceptance under US3. These are implementation checkpoints, not independently released commands. Full CLI acceptance requires T006–T007. Keep the existing Ralph rule of at most one story section per iteration; several validated tasks in that section may be completed together.

## Prerequisites and Completion Rules

These are part of every work unit, not additional tasks or iterations:

- Read the active feature artifacts, constitution, `AGENTS.md`, and `specs/ralph-implementation-rules.md`; verify issue #326 selection and stop on a genuine conflict. Reuse the existing areas, helpers, and dependencies.
- Read/update `ralph-memory.md` and append `progress.md` through the Ralph workflow. Do not create separate setup, fixture, model-only, test-only, or bookkeeping commits.
- Add behavioral tests with the implementation they verify. Create fixtures where first needed, using existing `testx` support. Do not introduce a shared fixture abstraction without concrete reuse.
- Every completed task must compile and pass its closest meaningful checks. Add required function/test comments, format touched Go files, and check the diff in the same work unit. Red tests are temporary within a work unit, not completed tasks.
- Keep all version implementations and affected interface doubles compiling in the same interface-changing work unit. Do not leave placeholder success implementations for later tasks.
- Preserve scope, payload, output and error contracts. Do not add flags, task mutations, search-based selection, dependencies, generic frameworks, or unrelated PI refactoring.
- Reuse passing validation unless relevant changes invalidate it. Run command-global/terminal tests serially. Runtime tests apply to implementation, not this task-list edit.

## User Story 1 — Scope-Safe Variable Updates: Service and Facade

**Goal**: Implement the complete reusable backend once, including collection inputs and shared targets needed by US2.

**Validation checkpoint**: Service/facade tests prove local, inherited and new-variable scope behavior for one or several tasks, with truthful confirmation and partial results. CLI wiring follows in T005; this checkpoint does not claim a shipped command.

- [ ] T001 [US1] Implement the scope-write capability and all version adapters together: add the minimal acceptance response in `internal/domain/usertask_update.go`, extend `internal/services/variable/api.go`, and implement `UpdateScopeVariables` in `internal/services/variable/v810/variables.go`, `internal/services/variable/v89/variables.go`, `internal/services/variable/v88/variables.go`, and `internal/services/variable/v87/variables.go`. Use generated scope requests with explicit `local=true`, existing mutation retries and HTTP error normalization; v8.7 returns unsupported without transport calls. Add adapter tests in each matching `scope_variables_test.go`, update affected doubles/assertions, and run targeted adapter plus PI-write regressions. Verify exact key/body, 204 handling, invalid input, nil/HTTP/transport errors and zero PI reads/waits; preserve existing PI mutation behavior and generated clients.

- [ ] T002 [US1] Implement complete planning with its domain models and tests in `internal/domain/usertask_update.go`, `internal/services/usertask/update.go`, and `internal/services/usertask/update_test.go`. Reuse native task lookup and complete effective-variable traversal; resolve existing names to returned scopes and missing names to the task element scope. Include unique input order, sorted categories, tenant evidence, grouped scope payloads, shared task associations and cross-task deduplication from the outset. Enforce the model constraints: “Logical identity is `(ScopeKey, Name)`”; “Variables: nonempty map of planned additions/changes for this scope”; “A plan is executable only after all task and variable reads succeeded”; “Complete normalized values establish equality; truncated values never do”; “Null is a value, not deletion”; “Invalid or absent required scope identity is a planning error”; “MutationSubmitted: false for every plan/preview/no-op.” Test sparse/complete paging, local/intermediate/root scopes, inherited equality, empty payload/service input, truncated before-values, shared names at same/different scopes, conflicting identity/tenant evidence and later discovery failure with zero writes. Variable counts remain task-variable observations; physical target counts do not replace them. Introduce the composed service now and finish its execution contract in T003 without publishing an unimplemented execution method.

- [ ] T003 [US1] Implement execution, confirmation and partial outcomes together in `internal/services/usertask/update.go` and `internal/services/usertask/update_wait.go`, with tests in `internal/services/usertask/update_test.go`, `internal/services/usertask/update_bulk_test.go`, and `internal/services/usertask/update_wait_test.go`. Validate supplied plans before requests, honor library dry-run, and execute frozen unique scope payloads through existing pool/worker/fail-fast controls; preserve started outcomes and mark unscheduled targets. Confirm through complete effective-variable reads using existing backoff bounds and cancellation, matching name, intended scope and complete value. No-wait/no-op makes zero confirmation reads; planning failure/dry-run/no-op makes zero writes. Preserve “No task variable confirmation takes place inside the scope writer” and “No rollback and no rescoping after confirmation.” Test shared failure fan-out, one-task partial acceptance, worker bounds, cancellation, task disappearance/authorization loss, mismatched scope despite equal value, truncation and timeout. Use model states `confirmed`, `submitted`, `mutation_failed`, `confirmation_failed`, `unchanged`, `skipped`; acceptance is true if any required scope write was accepted, never a success predicate. Unchanged uses skipped confirmation and false acceptance; fail-fast confirmation stops retain submitted/skipped-confirmation facts plus aggregate failure. Return results with errors, retain input order, and run focused service tests with race coverage for concurrent paths.

- [ ] T004 [US1] Expose and wire the thin facade in `c8volt/task/api.go`, `c8volt/task/model.go`, `c8volt/task/convert.go`, `c8volt/task/client.go`, and `c8volt/client.go`; implement `PlanUserTaskVariableUpdates`, `ExecuteUserTaskVariableUpdates`, and compatible `NewWithVariableUpdates` construction per the service contract. Keep configuration, planning, pools and polling in the internal service; copy mutable values at boundaries, map facade options, preserve partial results through `ferrors.FromDomain`, and update affected doubles. Add conversion/delegation/wiring tests in `c8volt/task/update_test.go` and `c8volt/client_test.go`, covering nested mutation isolation, all result states, normalized errors, retained read behavior and a precondition error for update calls through an unconfigured legacy constructor; run the focused facade and affected constructor checks.

## User Story 2 — Explicit-Key Command and Bulk Integration

**Goal**: Connect the existing command grammar to the already complete collection-based workflow.

**Validation checkpoint**: Command-path tests prove inline/file/stdin equivalence, explicit-key authorization, fixed confirmation scope and shared-write execution. Do not rebuild service mechanics here. Final presentation acceptance is T006.

- [ ] T005 [US2] Implement CLI construction, shared payload parsing, confirmation and dispatch in `cmd/update_usertask.go`, `cmd/update_usertask_variables.go`, and `cmd/update_variables_payload.go`, updating only the shared parsing calls in `cmd/update_processinstance_variables.go`. Register canonical/alias names, PI-equivalent key/payload/worker/dry-run/no-wait flags and state-changing/full-contract/full-automation metadata. Validate local errors before requests; retain JSON/verbose and unattended restrictions, explicit-key tenant behavior, existing prompt eligibility and configured stderr. Delegate the original frozen plan, preserve partial results and keep loops out of the command. Add tests in `cmd/update_usertask_test.go`, `cmd/update_usertask_bulk_test.go`, and `cmd/command_contract_test.go` for repeated/comma/stdin keys, deduplication, missing keys, payload/worker errors, file parity, aliases, counts, tenant context and shared writes; retain PI parsing regressions in `cmd/update_processinstance_test.go`. Keep this work unit compiling with the existing shared rendering primitives; introduce only the minimal working view entry points in `cmd/cmd_views_usertask_update.go` needed for dispatch, with no placeholder success or misleading no-op. T006 completes and locks down these views. Run focused command, metadata and PI parsing checks.

## User Story 3 — Complete Output, Automation and Delivery

**Goal**: Finish the output contract in one pass, prove terminal/unattended behavior, and deliver documentation and integrated validation.

**Validation checkpoint**: All three specification stories pass end to end across supported versions, including pure stdout, shared-scope results, no-op and failure cases.

- [ ] T006 [US3] Complete all views and their contract tests in `cmd/cmd_views_usertask_update.go`, `cmd/cmd_views_usertask_update_test.go`, and `cmd/update_usertask_output_test.go`; make only required dispatch adjustments in `cmd/update_usertask.go`. Cover single/bulk human plans and results, inherited scope evidence, tenant context, exact PI-style wording/tokens/counts, stable JSON fields and omission/null rules, keys-only changed-task keys, quiet+machine precedence and JSON-over-keys precedence. Dry-run/no-op uses succeeded with no submission even under no-wait; accepted writes use accepted; partial failures retain payload in one normalized error envelope and exit without rendering again. Require one decoded envelope then EOF, zero-byte no-work keys, truthful skipped/failed/unchanged counts, writer-failure handling and zero remote calls from rendering. Exercise normal/dry-run/no-op/no-wait with human/JSON/keys/quiet and compatible auto-confirm/automation options; run focused views/output and PI output regressions.

- [ ] T007 [US3] Complete end-to-end terminal, automation, version and error coverage in `cmd/update_usertask_terminal_test.go`, `cmd/update_usertask_error_test.go`, and the existing UT command/bulk/metadata test files; include fixes in the owning command/service files in this same work unit. Use real `testx.NewCmdTerminalRunner` stdin with separate stdout/stderr to prove yes/no/default/EOF, configured/inherited stderr, redirected stdout, unchanged prompt wording/eligibility, prompt-free dry-run/no-op/automation/auto-confirm, supported no-wait combinations and zero writes on abort. Exercise 8.8/8.9/8.10 equivalence, zero-mutation unsupported 8.7, unauthorized/missing tasks, malformed discovery, truncated confirmation, partial acceptance and exit codes. Recheck shared-target per-task outcomes and request counts through command execution; keep functional detail under verbose and HTTP diagnostics under DEBUG. Run focused command/terminal/version checks and only relevant invalidated service/PI regressions; record actual evidence and gaps.

- [ ] T008 [US3] Finish documentation and integrated review in `README.md`, `cmd/update.go`, `cmd/update_usertask.go`, and `specs/326-update-user-task-vars/quickstart.md`; align final examples/test patterns, run `make docs-content`, and inspect `docs/cli/c8volt_update_user-task.md`, parent pages and `docs/index.md` without hand-editing generated content. Review the full diff against the feature contracts and `specs/ralph-implementation-rules.md`: thin facades, service-owned mechanics, focused files/comments, frozen scope behavior, unchanged PI behavior and no generated-client edits. Fix related findings, format touched Go files, run `git diff --check` and the issue-required `make test` for this cross-package concurrent runtime change. Record results and optional live-validation gaps in `specs/326-update-user-task-vars/progress.md`, retain durable discoveries in `ralph-memory.md`, and complete the coordinated Ralph task/memory/progress commit only after checks pass.

## Dependencies and Ralph Execution

```text
US1: T001 → T002 → T003 → T004
US2: T005
US3: T006 → T007 → T008
```

- No separate setup/foundation/check-only iterations are needed. T001 includes its response type and all adapter implementations; T002 includes the rest of the models and planning fixtures; subsequent tasks create their own tests and fixtures.
- Single-task and bulk planning/execution are complete in T002–T003. T005 integrates both through one command. T006 implements their shared output contract together; T007 verifies full execution without another implementation phase for bulk.
- Each task is independently compilable and testable at its owner layer. The command/view boundary in T005–T006 permits a minimal working view, not undefined functions or false-success stubs.
- Ralph may complete several validated tasks within the current story section in one iteration. Eight tasks is not a promised iteration count; partial work and failures still follow the installed Ralph rules. Do not stop solely because one checkbox was completed if another task in the same section can be validated coherently.
- No task is marked `[P]`: the eight units deliberately have dependencies and shared files. Independent version adapter edits inside T001 can be batched, but all adapters and doubles must compile together. No parallel command-global test execution.
- Keep the three original user stories as acceptance scenarios; this ordering does not add stories, remove requirements, or imply separate feature releases. The usable end-to-end checkpoint is T006, acceptance closes at T007, and delivery closes at T008.

## Acceptance and Requirement Coverage

| Specification story | Required proof | Primary tasks |
|---|---|---|
| US1: preview/update one task | Local/inherited/new scopes; unchanged values; preview/abort/no-op; exact scope confirmation | T001–T006, T007 |
| US2: multiple explicit tasks | Flag/stdin/file parity; one shared target write; stable order/counts; partial and fail-fast outcomes; actual tenant context | T002–T005, T006–T007 |
| US3: automation/output | Prompt-free unattended paths; accepted versus confirmed; one JSON envelope; pure keys; real terminal stderr; versions | T001, T003, T005–T008 |

| Requirements | Tasks |
|---|---|
| FR-001 grammar/aliases | T005, T007, T008 |
| FR-002 explicit key input | T002, T005, T007 |
| FR-003 payload rules | T002, T005, T006 |
| FR-004 effective scope/local additions | T001–T004, T007 |
| FR-005 plan categories/equality | T002, T006 |
| FR-006 shared scope deduplication | T002, T003, T005–T007 |
| FR-007 confirmation/dry-run/no-op | T002, T003, T005–T007 |
| FR-008 scope confirmation/no-wait | T003, T006, T007 |
| FR-009 concurrency/errors/partial results | T003, T004, T006, T007 |
| FR-010 output modes | T005–T007 |
| FR-011 tenant/authorization | T002, T005–T007 |
| FR-012 versions | T001, T007 |
| FR-013 architecture/reuse/PI compatibility | T001–T008 |
| FR-014 documentation/metadata | T005, T007, T008 |

## Validation Notes

Follow the existing [quickstart](quickstart.md). Select actual test names and verify patterns do not pass with zero matching tests. Each work unit includes its own relevant checks; do not repeat suites merely to commit or create a validation-only iteration. Keep the final issue-required race suite in T008.

This revision consolidates the previous 32 tasks into eight implementation units. All tasks remain unchecked; implementation has not started. Only documentation checks apply to this revision.
