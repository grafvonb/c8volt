# Ralph Memory

Feature: 268-pd-watch-repaint
Started: 2026-08-05T14:59:33Z

## Codebase Patterns
- Process-definition watch is command-owned in `cmd/get_processdefinition.go`: `executeGetProcessDefinitionWatch` builds a `process.ProcessDefinitionWatchSnapshotRequest`, chooses normal or explicit-admin facade options, and delegates serial looping to `toolx/watch.Run`.
- `toolx/watch.Run` is output-agnostic and synchronous: it executes one tick immediately, calls the tick function inline, resets consecutive failures on success, then sleeps via the injectable `SleepFunc`.
- Watch result rendering is isolated in `cmd/cmd_views_get.go`; `processDefinitionWatchView` delegates to normal process-definition list rendering so refresh bodies match non-watch human output.
- Existing watch command tests in `cmd/get_processdefinition_test.go` already use separate stdout/stderr buffers through `executeGetProcessDefinitionWatchWithBackoffForTest` and deterministic sleeps through the package-level `processDefinitionWatchSleep` seam.
- Iteration 2 added `processDefinitionWatchHarness` and `executeGetProcessDefinitionWatchHarnessForTest` in `cmd/get_processdefinition_test.go`; the older tuple-return helpers now delegate to it, and future tests can use `processDefinitionWatchRunResult` for named stdout/stderr/error fields.
- Repaint assertions can use `processDefinitionWatchRepaintControlSequenceForTest` (`ESC [ H` then `ESC [ 2 J`), `requireProcessDefinitionWatchRepaintCount`, `requireNoProcessDefinitionWatchRepaintControls`, and `stdoutWithoutRepaintControls` without requiring a real terminal.
- Slow-refresh command tests can use the `processDefinitionWatchNow` seam through `processDefinitionWatchHarness.now` and `newProcessDefinitionWatchClockForTest`; pass two timestamps per successful refresh because the command samples before collection and after rendering.
- Watch-related command capability/help expectations live in `cmd/command_contract_test.go`; generated CLI docs mirror command source through `make docs-content`.
- US1 repaint behavior is command-layer only: `renderTerminalRepaint` in `cmd/cmd_views_rendermode.go` writes `ESC [ H` + `ESC [ 2 J` to stdout before each successful process-definition watch refresh, and `processDefinitionWatchView` delegates refresh bodies to `listProcessDefinitionsView`.
- Retry/status text now says `refresh N failed`; repaint/body assertions should strip ANSI controls before comparing stdout body text.
- US2 slow-refresh status is command-layer only: `executeGetProcessDefinitionWatch` measures collection plus repaint/render duration with `processDefinitionWatchNow`, writes one default slow warning per continuous slow streak to stderr, resets the streak after an on-time refresh, and writes per-refresh timing to stderr when `flagVerbose` is true.
- `toolx/watch.Run` remains synchronous and output-agnostic; `TestWatchRunKeepsTicksSerialWhenTickExceedsInterval` protects that slow ticks finish before sleep and before the next tick starts.

## Decisions
- Iteration 1 completed Phase 1 inspection only. No command behavior was changed.
- Iteration 2 completed Phase 2 test foundation only. No command behavior was changed.
- Iteration 3 completed US1. Process-definition watch now repaints each successful refresh and renders the same body as normal non-watch list output, with command help/metadata updated from snapshot wording to refresh/repaint wording.
- Iteration 4 completed US2. Watch refresh duration is measured around collection plus render, default slow warnings are streak-based on stderr, verbose timing is emitted per refresh on stderr, and runner seriality is covered in `toolx/watch`.
- Iteration 5 completed US3. Watch remains human-only with local rejection before lookup for JSON, keys-only, XML, quiet, and automation; non-watch JSON, keys-only, XML, quiet, and automation compatibility is covered.
- Iteration 6 completed polish. README and generated CLI docs now describe repaint refreshes, normal result-body parity, slow-refresh warnings, and human-only watch boundaries; gofmt, focused quickstart validation, and `make test` passed.

## Gotchas
- `processDefinitionWatchSnapshotRequest` and `ProcessDefinitionWatchSnapshot` remain internal/facade type names even though user-facing text now says refresh/repaint.
- Verbose watch mode now intentionally writes timing status to stderr, so command tests that set `flagVerbose` should not require empty stderr.
- Command validation tests may exercise Cobra commands before a context/config is attached; automation-mode checks must tolerate a nil command context while still honoring command flags and global automation state.

## Reusable Commands
- `go test ./cmd -run 'TestGetProcessDefinitionWatch|TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata|TestProcessDefinitionSelectorValidationHelpContract' -count=1`
- `go test ./cmd -count=1`
- `go test ./toolx/watch -run 'TestWatchRunKeepsTicksSerialWhenTickExceedsInterval|TestWatchRun' -count=1`
- `go test ./cmd -run 'TestValidateGetProcessDefinitionWatch|TestGetProcessDefinitionWatchRejectsMachineModesBeforeLookup|TestGetProcessDefinitionNonWatchMachineModesStayCompatible|TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata' -count=1`
- `go test ./cmd -run 'TestGetProcessDefinitionWatch|TestValidateGetProcessDefinitionWatch|TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata|TestProcessDefinitionSelectorValidationHelpContract' -count=1`
- `go test ./toolx/... -run 'Watch|watch' -count=1`
- `make docs-content`
- `make test`

## Do Not Repeat
- Do not add repaint behavior to `toolx/watch`; terminal repaint/status is command rendering behavior, while the watch runner should remain output-agnostic.
- Do not hand-edit generated CLI docs under `docs/cli/`; update command metadata/help and regenerate them.

## Current Handoff
- Feature complete; no handoff required.
