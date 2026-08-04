# Ralph Memory

Feature: 258-process-definition-watch
Started: 2026-08-04T15:18:59Z

## Codebase Patterns
- `cmd/get_processdefinition.go` currently owns process-definition flags, local validation, one-shot branch routing, XML/key/search rendering, and command metadata. Existing aliases are `pd` and `pds`.
- Process-definition broad search already delegates paging to `cli.SearchProcessDefinitionsPages`; command code renders service-provided page progress but does not advance backend cursors itself.
- Current generated docs for `get process-definition` mirror command help and do not mention watch flags yet; future command metadata/help changes require `make docs-content`.
- Reusable watch mechanics now live in `toolx/watch.Run` with immediate first tick, injected sleep for tests, timeout/cancel reasons, and consecutive retry reset/exhaustion behavior.
- Process-definition watch snapshots are available through `process.API.CollectProcessDefinitionWatchSnapshot`; the facade delegates to `processdefinition.CollectProcessDefinitionWatchSnapshot`, which owns key/latest/page dispatch below `cmd`.
- `cmd/get_processdefinition.go` now routes `--watch` before XML/key/search one-shot branches, validates `--watch-interval`, builds one facade snapshot request per command invocation, and renders each tick through `processDefinitionWatchSnapshotView`.
- Default US1 watch rendering uses compact human blocks: `snapshot N:`, existing aligned process-definition rows from `flatRowPD`, then `found: N`. Empty snapshots print only the boundary and `found: 0`.
- US2 watch status messages are written to `cmd.ErrOrStderr()`: retry notices, timeout stop notices, and retry-exhaustion stop notices stay away from result stdout.
- US3 watch validation is command-local in `validateGetProcessDefinitionWatchOutputFlags`: `--watch` rejects JSON, keys-only, XML, quiet, and automation combinations before process-definition lookup work.
- Process-definition command output metadata now explicitly advertises one-line, JSON, and keys-only modes; the notes document that watch uses terminal snapshots and rejects finite machine-oriented output combinations.
- README and generated CLI docs now document process-definition watch examples, default `1s` interval, broad unselected watch behavior, human-only output rejections, `--batch-size`, timeout, and retry reset behavior.

## Decisions
- Baseline setup found no pre-existing failures in the targeted command or process-definition service/facade test slices, so `quickstart.md` did not need failure notes.
- Phase 2 introduced a public process facade method, so process API test stubs in `cmd/process_api_stub_test.go` and `c8volt/resource/client_test.go` must implement it even when tests expect it to panic.
- `--watch-interval` is stored as a duration string through `toolx.NewDurationStringValue`, defaults to `1s`, and is parsed/validated in `validateGetProcessDefinitionFlags` only when `--watch` is active.
- Watch retry tolerance uses the existing command backoff config: `cfg.App.Backoff.Timeout` maps to watch timeout and `cfg.App.Backoff.MaxRetries` maps to consecutive retry budget. The command default remains `0` for unlimited retries.
- Process-definition watch retries only shared `ferrors` timeout and unavailable classes; validation, unsupported, not-found, conflict, local precondition, and unknown/internal errors stay fatal.
- Incompatibility errors use `forbiddenFlagCombinationf("--watch cannot be combined with ...; watch prints repeated terminal snapshots")` so JSON mode returns the shared envelope and other modes use the standard invalid-args CLI error path.
- `--watch-interval` flag usage should not include a literal `(default 1s)` because Cobra appends the default automatically in generated docs.

## Gotchas
- Non-watch missing-selector behavior is documented as a local diagnostic; watch mode must override this only for `--watch` without selectors.
- Existing XML validation rejects `--json` and `--keys-only`, but watch must additionally reject XML, JSON, keys-only, quiet, and automation before lookup work.
- The documented `go test ./toolx/... -run 'Watch|watch' -count=1` filter only runs watch runner tests when test names include `Watch`; keep future tests aligned with that pattern.
- `resetGetProcessDefinitionCommandGlobals` now resets watch flags, process-definition batch size, and root output globals touched by process-definition tests (`flagViewAsJson`, `flagViewKeysOnly`, `flagQuiet`, `flagVerbose`, `flagDebug`, `flagCmdAutomation`); `resolveGetProcessDefinitionSearchSize` still protects command tests when the variable is zero.

## Reusable Commands
- `go test ./cmd -run 'TestGetProcessDefinition|TestCommandContract' -count=1`
- `go test ./cmd -count=1`
- `go test ./internal/services/processdefinition/... ./c8volt/process -run 'ProcessDefinition|SearchProcessDefinitions' -count=1`
- `go test ./toolx/... -run 'Watch|watch' -count=1`
- `go test ./internal/services/processdefinition/... -run 'SearchProcessDefinitions|Watch|Snapshot' -count=1`
- `go test ./c8volt/process -run 'ProcessDefinition|Watch|Snapshot' -count=1`
- `go test ./... -count=1`
- `make docs-content`
- `make test`

## Do Not Repeat

## Current Handoff
- Feature complete; no handoff required.
