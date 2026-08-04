# Ralph Memory

Feature: 258-process-definition-watch
Started: 2026-08-04T15:18:59Z

## Codebase Patterns
- `cmd/get_processdefinition.go` currently owns process-definition flags, local validation, one-shot branch routing, XML/key/search rendering, and command metadata. Existing aliases are `pd` and `pds`.
- Process-definition broad search already delegates paging to `cli.SearchProcessDefinitionsPages`; command code renders service-provided page progress but does not advance backend cursors itself.
- Current generated docs for `get process-definition` mirror command help and do not mention watch flags yet; future command metadata/help changes require `make docs-content`.

## Decisions
- Baseline setup found no pre-existing failures in the targeted command or process-definition service/facade test slices, so `quickstart.md` did not need failure notes.

## Gotchas
- Non-watch missing-selector behavior is documented as a local diagnostic; watch mode must override this only for `--watch` without selectors.
- Existing XML validation rejects `--json` and `--keys-only`, but watch must additionally reject XML, JSON, keys-only, quiet, and automation before lookup work.

## Reusable Commands
- `go test ./cmd -run 'TestGetProcessDefinition|TestCommandContract' -count=1`
- `go test ./internal/services/processdefinition/... ./c8volt/process -run 'ProcessDefinition|SearchProcessDefinitions' -count=1`

## Do Not Repeat

## Current Handoff
- Next iteration should start Phase 2 foundational work at T005-T014; add the reusable `toolx/watch` runner and process-definition snapshot model/facade/service plumbing before any US1 command watch tasks.
