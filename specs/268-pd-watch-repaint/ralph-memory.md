# Ralph Memory

Feature: 268-pd-watch-repaint
Started: 2026-08-05T14:59:33Z

## Codebase Patterns
- Process-definition watch is command-owned in `cmd/get_processdefinition.go`: `executeGetProcessDefinitionWatch` builds a `process.ProcessDefinitionWatchSnapshotRequest`, chooses normal or explicit-admin facade options, and delegates serial looping to `toolx/watch.Run`.
- `toolx/watch.Run` is output-agnostic and synchronous: it executes one tick immediately, calls the tick function inline, resets consecutive failures on success, then sleeps via the injectable `SleepFunc`.
- Current watch result rendering is isolated in `cmd/cmd_views_get.go` as `processDefinitionWatchSnapshotView`; it prints an optional blank line, `snapshot N:`, formatted process-definition flat rows, and `found: len(snapshot.Items)`.
- Existing watch command tests in `cmd/get_processdefinition_test.go` already use separate stdout/stderr buffers through `executeGetProcessDefinitionWatchWithBackoffForTest` and deterministic sleeps through the package-level `processDefinitionWatchSleep` seam.
- Watch-related command capability/help expectations live in `cmd/command_contract_test.go`; generated CLI docs mirror command source through `make docs-content`.

## Decisions
- Iteration 1 completed Phase 1 inspection only. No command behavior was changed.

## Gotchas
- `cmd/get_processdefinition_test.go` currently asserts the stale `snapshot N:` behavior in multiple tests, including broad repeated snapshots, interval cadence, retry reset, empty-then-changed snapshots, interrupt/timeout, and stdout equality checks.
- `README.md`, `docs/cli/c8volt_get_process-definition.md`, `cmd/get_processdefinition.go`, and `cmd/command_contract_test.go` still use "snapshot" / "repeated terminal snapshots" wording that must change after implementation.
- `processDefinitionWatchSnapshotView` uses `len(snapshot.Items)` for `found:` while normal `listProcessDefinitionsView` uses the existing `listOrJSONFlat` path; US1 parity should reuse normal list rendering to avoid duplicate result-body rules.

## Reusable Commands
- `go test ./cmd -run 'TestGetProcessDefinitionWatch|TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata|TestProcessDefinitionSelectorValidationHelpContract' -count=1`
- `make docs-content`

## Do Not Repeat
- Do not add repaint behavior to `toolx/watch`; terminal repaint/status is command rendering behavior, while the watch runner should remain output-agnostic.
- Do not hand-edit generated CLI docs under `docs/cli/`; update command metadata/help and regenerate them.

## Current Handoff
- Next iteration should complete Phase 2 foundation tasks T004-T005 by extending command test seams for separate stdout/stderr capture, deterministic watch sleeps, and deterministic repaint control assertions without requiring a real terminal.
