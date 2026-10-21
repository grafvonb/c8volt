# Ralph Memory

Feature: 341-suspended-instance-hierarchies
Started: 2026-10-21T16:18:02Z

## Codebase Patterns

- The 8.10 `newCancellationFixture` injects a strict client; `states` supplies per-key observations and repeats the last state. Empty string means absent. `parents` supplies ancestry for selected-child cases, and `cancellationKeys()` snapshots submissions under the fixture mutex.
- Normal root cancellation reads the root three times before confirmation (precheck and family traversal). Child search results establish membership; child state lookups occur during confirmation. `waitTestConfig()` uses two attempts, a 1 ms delay, and a 25 ms timeout.
- `DryRunCancelOrDeletePlan` deduplicates ancestry roots before descendant expansion. Command overlap evidence must reach the real planner and dispatch; `TestCancelProcessInstanceCommand_SuspendedOverlap` does this through a strict test-owned HTTP server.

## Decisions

- T001 added regressions only: suspended root, both mixed trees, delayed terminal/absent child confirmation, persistent suspension, context cancellation, child force rules, opt-out reads, planner classification, and CLI overlap/dry-run. All passed on unchanged production code; no runtime correction was justified.
- Preserve existing accepted-but-unconfirmed `ProcessInstanceMutationFailure` annotations, terminal/absent semantics, and combined NoWait/NoStateCheck zero reads. No resume endpoint exists in the injected service interface; the command HTTP fixture rejects resume and separate child cancellation.
- T001 validation passed with race detection: the entire 8.10 cancellation/deletion parent, cancel command tests, dry-run planner tests, and existing suspended waiter tests. No shared runtime changes, documentation generation, older-adapter changes, or full-suite run were needed.

## Gotchas

- Command tests mutate global Cobra state; do not parallelize them. Capture streams separately and require EOF after one JSON envelope. Clear inherited `C8VOLT_*` overrides before command tests; generated test config must point only at the fixture server.
- Fakes establish client behavior, not real engine, listener, or exporter behavior. No live validation was performed or is required for implementation.
- An unrelated concurrent `AGENTS.md` addition about launching Ralph appeared during iteration 1. Preserve it outside this work-unit commit; it does not change the single-iteration scope.

## Reusable Commands

- `env GOENV=off GOFLAGS= GOWORK=off go test ./internal/services/processinstance/v810 -run 'TestService_CancelAndDeleteProcessInstance' -count=1 -race`
- `env GOENV=off GOFLAGS= GOWORK=off go test ./internal/services/processinstance ./internal/services/processinstance/waiter -run 'DryRunCancelOrDeletePlan|TestSuspendedStateWaits' -count=1 -race`
- `env GOENV=off GOFLAGS= GOWORK=off go test ./cmd -run 'TestCancelProcessInstance' -count=1 -race` with inherited `C8VOLT_*`, `SPECKIT_*`, and helper-process variables removed by a subprocess environment filter (never print credential values).
- `gofmt -w <touched Go files>` and `git diff --check`.

## Do Not Repeat

- Do not redesign cancellation or deletion to make already-passing suspension regressions fail. Keep definition discovery in #340 and direct-library overlap guarantees out of #341.
- Do not treat stubbed unique-root output as CLI deduplication evidence; retain real HTTP planning/dispatch coverage.

## Current Handoff

- Next dependency-ready work unit: T002 / US2, suspended deletion safeguards. Recheck tasks.md before selection; T001 is complete and T003 depends on T002. No blocking dependency.
- Reuse the extended cancellation fixture and the suspended planner classification test. Add meaningful deletion ordering/recovery evidence without changing descendant-first or conflict-first behavior. After fixture edits, rerun the combined 8.10 cancellation/deletion parent to protect T001.
