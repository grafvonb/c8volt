# Ralph Memory

Feature: 258-process-definition-watch
Started: 2026-08-04T15:18:59Z

## Codebase Patterns
- `cmd/get_processdefinition.go` currently owns process-definition flags, local validation, one-shot branch routing, XML/key/search rendering, and command metadata. Existing aliases are `pd` and `pds`.
- Process-definition broad search already delegates paging to `cli.SearchProcessDefinitionsPages`; command code renders service-provided page progress but does not advance backend cursors itself.
- Current generated docs for `get process-definition` mirror command help and do not mention watch flags yet; future command metadata/help changes require `make docs-content`.
- Reusable watch mechanics now live in `toolx/watch.Run` with immediate first tick, injected sleep for tests, timeout/cancel reasons, and consecutive retry reset/exhaustion behavior.
- Process-definition watch snapshots are available through `process.API.CollectProcessDefinitionWatchSnapshot`; the facade delegates to `processdefinition.CollectProcessDefinitionWatchSnapshot`, which owns key/latest/page dispatch below `cmd`.
- `cmd/get_processdefinition.go` now routes `--watch` before XML/key/search one-shot branches, builds one facade snapshot request per command invocation, and renders each tick through `processDefinitionWatchSnapshotView`.
- Default US1 watch rendering uses compact human blocks: `snapshot N:`, existing aligned process-definition rows from `flatRowPD`, then `found: N`. Empty snapshots print only the boundary and `found: 0`.

## Decisions
- Baseline setup found no pre-existing failures in the targeted command or process-definition service/facade test slices, so `quickstart.md` did not need failure notes.
- Phase 2 introduced a public process facade method, so process API test stubs in `cmd/process_api_stub_test.go` and `c8volt/resource/client_test.go` must implement it even when tests expect it to panic.
- US1 uses a fixed `defaultGetPDWatchInterval = 1s` and `processDefinitionWatchSleep` injection only for command tests. The public `--watch-interval` flag and interval validation are intentionally still unimplemented for US2.
- US1 treats lookup errors as fatal (`Retryable` false) while clean cancel/timeout from `toolx/watch.Run` return nil; retry budget wiring and retry status output are intentionally left for US2.

## Gotchas
- Non-watch missing-selector behavior is documented as a local diagnostic; watch mode must override this only for `--watch` without selectors.
- Existing XML validation rejects `--json` and `--keys-only`, but watch must additionally reject XML, JSON, keys-only, quiet, and automation before lookup work.
- The documented `go test ./toolx/... -run 'Watch|watch' -count=1` filter only runs watch runner tests when test names include `Watch`; keep future tests aligned with that pattern.
- `resetGetProcessDefinitionCommandGlobals` now resets `flagGetPDWatch` and `flagGetPDBatchSize`; `resolveGetProcessDefinitionSearchSize` still protects command tests when the variable is zero.

## Reusable Commands
- `go test ./cmd -run 'TestGetProcessDefinition|TestCommandContract' -count=1`
- `go test ./internal/services/processdefinition/... ./c8volt/process -run 'ProcessDefinition|SearchProcessDefinitions' -count=1`
- `go test ./toolx/... -run 'Watch|watch' -count=1`
- `go test ./internal/services/processdefinition/... -run 'SearchProcessDefinitions|Watch|Snapshot' -count=1`
- `go test ./c8volt/process -run 'ProcessDefinition|Watch|Snapshot' -count=1`
- `go test ./... -count=1`

## Do Not Repeat

## Current Handoff
- Next iteration should start Phase 4 / US2 at T026-T033. Add `--watch-interval` as a positive duration flag, wire default/explicit cadence into the existing `executeGetProcessDefinitionWatch` runner options, then map existing command `backoff-max-retries` and retry status behavior without changing the completed US1 snapshot rendering contract.
