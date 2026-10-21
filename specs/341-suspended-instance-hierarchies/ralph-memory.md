# Ralph Memory

Feature: 341-suspended-instance-hierarchies
Started: 2026-10-21T16:18:02Z

## Codebase Patterns

- The 8.10 `newCancellationFixture` injects a strict client; `states` supplies per-key observations and repeats the last state. Empty string means absent. `parents` supplies ancestry for selected-child cases, and `cancellationKeys()` snapshots submissions under the fixture mutex.
- The fixture now records ordered `get:<key>:<state>`, `cancel:<key>`, and `delete:<key>:<status>` events; `requestEvents()` snapshots them. Empty state denotes absence. The delete callback receives key and global attempt count; `cancelError` injects a non-retryable cancellation failure.
- Normal root cancellation reads the root three times before confirmation (precheck and family traversal). Child search results establish membership; child state lookups occur during confirmation. `waitTestConfig()` uses two attempts, a 1 ms delay, and a 25 ms timeout.
- `DryRunCancelOrDeletePlan` deduplicates ancestry roots before descendant expansion. Command overlap evidence must reach the real planner and dispatch; `TestCancelProcessInstanceCommand_SuspendedOverlap` does this through a strict test-owned HTTP server.

## Decisions

- T001 added regressions only: suspended root, both mixed trees, delayed terminal/absent child confirmation, persistent suspension, context cancellation, child force rules, opt-out reads, planner classification, and CLI overlap/dry-run. All passed on unchanged production code; no runtime correction was justified.
- Preserve existing accepted-but-unconfirmed `ProcessInstanceMutationFailure` annotations, terminal/absent semantics, and combined NoWait/NoStateCheck zero reads. No resume endpoint exists in the injected service interface; the command HTTP fixture rejects resume and separate child cancellation.
- T001 validation passed with race detection: the entire 8.10 cancellation/deletion parent, cancel command tests, dry-run planner tests, and existing suspended waiter tests. No shared runtime changes, documentation generation, older-adapter changes, or full-suite run were needed.
- T002 passed against unchanged runtime code: direct-service non-force conflict, suspended child conflict escalating to root cancellation, terminal observation before retry with/without NoWait, normal final absence checks, and cancellation-error/persistent-suspension retry blocking. The real CLI planner refuses suspended root/child scope before confirmation/mutation, including NoWait; dry-run preserves one successful JSON preview.
- T002 race checks passed for the combined 8.10 cancellation/deletion parent, all delete-process-instance command tests, and existing 8.8/8.9 mutation controls. T001 already supplies suspended dry-run classification; no planner edit was needed.

- T003 corrected only cancel/delete state flag descriptions, matching get help (`suspended (Camunda 8.10)`). The new TestProcessInstanceMutationStateHelp failed for both commands before the correction and passed afterward; it also preserves the all default and s shorthand. README guidance retains root force escalation, children-first deletion, and terminal confirmation before conflict-recovery retry even with NoWait.
- T003 validation passed with race detection: all-command action help, mutation state help, suspended validation/search and 8.8/8.9 unsupported-command controls. `make docs-content` only generates local Cobra references and README-derived index; it passed and refreshed the two affected references and homepage (including build provenance). Runtime behavior and prior T001/T002 evidence remain unchanged.

## Gotchas

- Command tests mutate global Cobra state; do not parallelize them. Capture streams separately and require EOF after one JSON envelope. Clear inherited `C8VOLT_*` overrides before command tests; generated test config must point only at the fixture server.
- Fakes establish client behavior, not real engine, listener, or exporter behavior. No live validation was performed or is required for implementation.

## Reusable Commands

- `env GOENV=off GOFLAGS= GOWORK=off go test ./internal/services/processinstance/v810 -run 'TestService_CancelAndDeleteProcessInstance' -count=1 -race`
- `env GOENV=off GOFLAGS= GOWORK=off go test ./internal/services/processinstance ./internal/services/processinstance/waiter -run 'DryRunCancelOrDeletePlan|TestSuspendedStateWaits' -count=1 -race`
- `env GOENV=off GOFLAGS= GOWORK=off go test ./cmd -run 'TestCancelProcessInstance' -count=1 -race` with inherited `C8VOLT_*`, `SPECKIT_*`, and helper-process variables removed by a subprocess environment filter (never print credential values).
- `env GOENV=off GOFLAGS= GOWORK=off go test ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -run 'TestService_(CancelAndDeleteProcessInstance|CancelProcessInstance|DeleteProcessInstance)' -count=1 -race`
- `go test ./cmd -run 'TestDeleteProcessInstance' -count=1 -race` with the same sanitized environment described above.
- `gofmt -w <touched Go files>` and `git diff --check`.

## Do Not Repeat

- Do not redesign cancellation or deletion to make already-passing suspension regressions fail. Keep definition discovery in #340 and direct-library overlap guarantees out of #341.
- Do not treat stubbed unique-root output as CLI deduplication evidence; retain real HTTP planning/dispatch coverage.

## Current Handoff

- Feature complete; no handoff required.
