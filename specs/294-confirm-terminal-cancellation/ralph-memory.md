# Ralph Memory

Feature: 294-confirm-terminal-cancellation
Started: 2026-09-10T06:42:27Z

## Codebase Patterns

- Baseline validation uses the five targeted commands in `quickstart.md`; passing output alone is insufficient when a package reports `[no tests to run]`.
- Cancellation/deletion test prefixes are `TestService_CancelProcessInstance` and `TestService_DeleteProcessInstance` in v87/v88, but the combined `TestService_CancelAndDeleteProcessInstance` in v89/v810.
- Every versioned suite has `waitTestConfig`: fixed backoff, 1ms initial delay, 2 retries, and 25ms timeout.
- v8.7 cancellation family discovery must call `walker.Family` through the existing `traversalAdapter`; the public `GetProcessInstance` remains intentionally unsupported because direct Operate lookup is not tenant-safe.
- v8.8 cancellation regressions can use one file-local Camunda double for direct state reads, parent-filter searches, and cancellation requests; the Operate client remains strict and unused.
- v8.9 and v8.10 cancellation regressions fit their combined cancellation/deletion suites and can reuse strict body-based search doubles with per-key observation sequences.
- Bulk no-op propagation is covered through `stubBulkProcessInstanceAPI`; `reporterTotals` directly verifies terminal and absent results contribute to success totals without changing `bulk.go`.
- v8.7 contract F–H coverage uses `TestService_WaitForProcessInstanceExpectation` for the real search-backed state path plus cancellation subtests for opt-out read/discovery counts and submission/error boundaries; the existing `TestService_GetProcessInstanceStateByKey/NotFound` remains the strict direct state-getter control.
- v8.8 contract F–H coverage uses `TestService_WaitForProcessInstanceExpectation` for direct-get explicit state matching plus cancellation subtests for opt-out read/discovery counts and submission/error boundaries; the existing `TestService_GetProcessInstance` and `TestService_GetProcessInstanceStateByKey` not-found cases remain strict getter controls.
- v8.9 contract F–H coverage uses `TestService_WaitForProcessInstanceExpectation` for direct-get explicit state matching plus combined lifecycle cancellation subtests for opt-out read/discovery counts and submission/error boundaries; `TestService_SearchAndLookup/GetProcessInstanceTreatsNotFoundAsNotFound` remains the strict getter control.
- v8.10 contract F–H coverage uses `TestService_WaitForProcessInstanceExpectation` for direct-get explicit state matching plus combined lifecycle cancellation subtests for opt-out read/discovery counts and submission/error boundaries; `TestService_SearchAndLookup/GetProcessInstanceTreatsNotFoundAsNotFound` remains the strict getter control.
- Shared waiter compatibility coverage uses `TestWaitForProcessInstanceExpectation_ExplicitCanceledCompatibility` to keep canceled/terminated equivalence separate from terminal cleanup acceptance while requiring incident and state to match on the same present instance.
- Command compatibility coverage uses `TestExpectProcessInstanceCommand_StateMismatchRemainsStrict` to exercise explicit canceled expectations against all four terminal observations in human and JSON subprocess modes; successful helpers exit explicitly so Go test output cannot contaminate the JSON envelope.
- Cancellation command compatibility coverage uses `TestCancelProcessInstancesWithPlan_TerminalNoOpPreservesCommandContracts` to preserve terminal no-op report fields, human/JSON outcomes, prompt and activity behavior, and all combinations of inherited no-wait/no-state-check options.
- Coverage-safe US3 adapter validation uses `TestService_(CancelProcessInstance|DeleteProcessInstance|CancelAndDeleteProcessInstance|WaitForProcessInstanceExpectation)$`; the full process-instance package run additionally includes version-specific strict not-found getter controls.
- `make docs-content` propagates README changes to `docs/index.md`, refreshes its generated build banner, and renders cancellation long help into `docs/cli/c8volt_cancel_process-instance.md`.

## Decisions

- Keep the v8.7 public getter strict and preserve family discovery semantics by routing only the cancellation family walk through its existing search-backed traversal adapter.

## Gotchas

- The original versioned filter selects no tests in v89/v810; use the coverage-safe command recorded in `quickstart.md` until test names change.
- The callback-only cleanup process-instance double cannot reproduce row A; the cleanup regression must delegate to a real v88 cancellation service through a small file-local adapter/client double.
- The initial v8.7 regression run reached the expected no-op failures and also exposed the stale direct-get family wiring; do not restore `walker.Family(ctx, s, ...)` in the v8.7 service.
- The v8.7 forced-delete regression exposed the same stale direct-get wiring in deletion scope discovery; keep that internal walk on `traversalAdapter{s}` while the public getter remains intentionally unsupported.
- The v8.8 old-state regression failed only on the intended A–D acceptance cases; its active/unknown and interruption controls already preserved existing failure behavior.
- The v8.9 old-state regression failed only on completed/disappeared family outcomes, terminal-root `Ok`, and absent-root mapping; its failure controls remained strict.
- The v8.8 forced-delete recovery fixture can extend the existing cancellation double with a strict Operate delete-response sequence; this keeps recovery waits, delete retries, and final absence reads observable in one real service path.
- The v8.9 cancellation fixture can drive strict Camunda delete-response sequences; `WithNoWait` isolates the unconditional recovery wait, while the ordinary path needs the existing cancellation-family reads before final absence verification.
- The v8.10 cancellation fixture supports the same strict delete-response sequence as v8.9, including recovery polling, retry, final absence verification, and later delete failure.
- A process-definition cleanup regression can wire a file-local Camunda/Operate double into the real v8.8 process-instance service while retaining the existing process-definition stage sequence and downstream failure controls.

## Reusable Commands

- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- Targeted baseline commands are recorded in `quickstart.md`.
- `make test` is the required pre-commit gate and passed for iterations 1, 3, 4, 5, 6, 8, 9, 10, and 12.
- `make test` passed for iterations 13–20 after the v8.7, v8.8, v8.9, v8.10, shared-waiter, command-level, and integrated contract F–H compatibility additions.
- Final validation in iteration 22 passed all targeted guide commands, the coverage-safe four-adapter command, `make docs-content`, `git diff --check`, and `make test`.

## Do Not Repeat

## Current Handoff
- Start T033: reconcile FR-001–FR-010 and SC-001–SC-005 against the recorded evidence, verify final test names and task state, then complete the terminal handoff if all checks remain satisfied.
