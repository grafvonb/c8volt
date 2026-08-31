# Ralph Memory

Feature: 286-tenant-pd-ordering
Started: 2026-08-31T11:37:22Z

## Codebase Patterns
- Focused baseline commands from `specs/286-tenant-pd-ordering/quickstart.md` currently pass before implementation changes. `internal/domain` has no matching process-definition sort/order/latest tests yet, so the focused domain command reports `[no tests to run]` until T002 adds comparator coverage.

## Decisions
- Treat T001 as a validation-only setup work unit; no production code changed in iteration 1.

## Gotchas
- Shell wrapper note: zsh has special parameters named `status` and `commands`; use neutral variable names or run validation loops under `/bin/bash`.

## Reusable Commands
- `go test ./internal/domain -run 'Test.*ProcessDefinition.*(Sort|Order|Latest)' -count=1`
- `go test ./internal/services/processdefinition -run 'Test.*(SearchProcessDefinitionsPages|Latest|WatchSnapshot|Order)' -count=1`
- `go test ./internal/services/processdefinition/v87 -run 'Test.*ProcessDefinition.*(Search|Latest|Order|Sort)' -count=1`
- `go test ./internal/services/processdefinition/v88 -run 'Test.*ProcessDefinition.*(Search|Latest|Order|Sort|Stat)' -count=1`
- `go test ./internal/services/processdefinition/v89 -run 'Test.*ProcessDefinition.*(Search|Latest|Order|Sort|Stat)' -count=1`
- `go test ./internal/services/processdefinition/v810 -run 'Test.*ProcessDefinition.*(Search|Latest|Order|Sort|Stat)' -count=1`
- `go test ./c8volt/process -run 'TestClient_(SearchProcessDefinitions|SearchProcessDefinitionsLatest|SearchProcessDefinitionsPages|CollectProcessDefinitionWatchSnapshot)' -count=1`
- `go test ./cmd -run 'Test.*ProcessDefinition.*(Order|Latest|Paging|Stat|JSON|Keys|Watch|XML)|TestCommandContract' -count=1`

## Do Not Repeat
- Do not use zsh variable names `status` or `commands` in validation-loop scripts.

## Current Handoff
- Next iteration starts at T002: add table-driven canonical comparator tests in `internal/domain/processdefinition_test.go` before implementing T003.
