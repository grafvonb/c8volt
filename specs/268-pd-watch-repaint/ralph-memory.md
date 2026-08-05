# Ralph Memory

Feature: 268-pd-watch-repaint
Started: 2026-08-05T14:59:33Z

## Codebase Patterns
- Process-definition watch is command-owned in `cmd/get_processdefinition.go`: `executeGetProcessDefinitionWatch` builds a `process.ProcessDefinitionWatchSnapshotRequest`, chooses normal or explicit-admin facade options, and delegates serial looping to `toolx/watch.Run`.
- `toolx/watch.Run` is output-agnostic and synchronous: it executes one tick immediately, calls the tick function inline, resets consecutive failures on success, then sleeps via the injectable `SleepFunc`.
- Current watch result rendering is isolated in `cmd/cmd_views_get.go` as `processDefinitionWatchSnapshotView`; it prints an optional blank line, `snapshot N:`, formatted process-definition flat rows, and `found: len(snapshot.Items)`.
- Existing watch command tests in `cmd/get_processdefinition_test.go` already use separate stdout/stderr buffers through `executeGetProcessDefinitionWatchWithBackoffForTest` and deterministic sleeps through the package-level `processDefinitionWatchSleep` seam.
- Iteration 2 added `processDefinitionWatchHarness` and `executeGetProcessDefinitionWatchHarnessForTest` in `cmd/get_processdefinition_test.go`; the older tuple-return helpers now delegate to it, and future tests can use `processDefinitionWatchRunResult` for named stdout/stderr/error fields.
- Repaint assertions can use `processDefinitionWatchRepaintControlSequenceForTest` (`ESC [ H` then `ESC [ 2 J`), `requireProcessDefinitionWatchRepaintCount`, `requireNoProcessDefinitionWatchRepaintControls`, and `stdoutWithoutRepaintControls` without requiring a real terminal.
- Watch-related command capability/help expectations live in `cmd/command_contract_test.go`; generated CLI docs mirror command source through `make docs-content`.
- US1 repaint behavior is command-layer only: `renderTerminalRepaint` in `cmd/cmd_views_rendermode.go` writes `ESC [ H` + `ESC [ 2 J` to stdout before each successful process-definition watch refresh, and `processDefinitionWatchView` delegates refresh bodies to `listProcessDefinitionsView`.
- Retry/status text now says `refresh N failed`; repaint/body assertions should strip ANSI controls before comparing stdout body text.

## Decisions
- Iteration 1 completed Phase 1 inspection only. No command behavior was changed.
- Iteration 2 completed Phase 2 test foundation only. No command behavior was changed.
- Iteration 3 completed US1. Process-definition watch now repaints each successful refresh and renders the same body as normal non-watch list output, with command help/metadata updated from snapshot wording to refresh/repaint wording.

## Gotchas
- `README.md` and generated CLI docs under `docs/cli/` still use stale "snapshot" / "repeated terminal snapshots" wording until polish tasks T026-T027 run; do not hand-edit generated docs.
- `processDefinitionWatchSnapshotRequest` and `ProcessDefinitionWatchSnapshot` remain internal/facade type names even though user-facing text now says refresh/repaint.

## Reusable Commands
- `go test ./cmd -run 'TestGetProcessDefinitionWatch|TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata|TestProcessDefinitionSelectorValidationHelpContract' -count=1`
- `go test ./cmd -count=1`
- `make docs-content`

## Do Not Repeat
- Do not add repaint behavior to `toolx/watch`; terminal repaint/status is command rendering behavior, while the watch runner should remain output-agnostic.
- Do not hand-edit generated CLI docs under `docs/cli/`; update command metadata/help and regenerate them.

## Current Handoff
- Next iteration should complete US2 tasks T013-T020 only. Start by adding slow-refresh warning/timing tests in `cmd/get_processdefinition_test.go`; keep warnings/status on stderr and leave stdout result bodies comparable after stripping repaint controls.
