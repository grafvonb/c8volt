# Ralph Memory

Feature: 258-process-definition-watch
Started: 2026-08-04T15:18:59Z

## Codebase Patterns
- `cmd/get_processdefinition.go` currently owns process-definition flags, local validation, one-shot branch routing, XML/key/search rendering, and command metadata. Existing aliases are `pd` and `pds`.
- Process-definition broad search already delegates paging to `cli.SearchProcessDefinitionsPages`; command code renders service-provided page progress but does not advance backend cursors itself.
- Current generated docs for `get process-definition` mirror command help and do not mention watch flags yet; future command metadata/help changes require `make docs-content`.
- Reusable watch mechanics now live in `toolx/watch.Run` with immediate first tick, injected sleep for tests, timeout/cancel reasons, and consecutive retry reset/exhaustion behavior.
- Process-definition watch snapshots are available through `process.API.CollectProcessDefinitionWatchSnapshot`; the facade delegates to `processdefinition.CollectProcessDefinitionWatchSnapshot`, which owns key/latest/page dispatch below `cmd`.

## Decisions
- Baseline setup found no pre-existing failures in the targeted command or process-definition service/facade test slices, so `quickstart.md` did not need failure notes.
- Phase 2 introduced a public process facade method, so process API test stubs in `cmd/process_api_stub_test.go` and `c8volt/resource/client_test.go` must implement it even when tests expect it to panic.

## Gotchas
- Non-watch missing-selector behavior is documented as a local diagnostic; watch mode must override this only for `--watch` without selectors.
- Existing XML validation rejects `--json` and `--keys-only`, but watch must additionally reject XML, JSON, keys-only, quiet, and automation before lookup work.
- The documented `go test ./toolx/... -run 'Watch|watch' -count=1` filter only runs watch runner tests when test names include `Watch`; keep future tests aligned with that pattern.

## Reusable Commands
- `go test ./cmd -run 'TestGetProcessDefinition|TestCommandContract' -count=1`
- `go test ./internal/services/processdefinition/... ./c8volt/process -run 'ProcessDefinition|SearchProcessDefinitions' -count=1`
- `go test ./toolx/... -run 'Watch|watch' -count=1`
- `go test ./internal/services/processdefinition/... -run 'SearchProcessDefinitions|Watch|Snapshot' -count=1`
- `go test ./c8volt/process -run 'ProcessDefinition|Watch|Snapshot' -count=1`
- `go test ./... -count=1`

## Do Not Repeat

## Current Handoff
- Next iteration should start Phase 3 / US1 at T015-T025. Use `toolx/watch.Run` for command watch execution and `cli.CollectProcessDefinitionWatchSnapshot` for each tick; keep command work to flags, validation/routing, and human rendering.
