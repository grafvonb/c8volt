# Ralph Memory

Feature: 234-get-element-command
Started: 2026-07-16T11:27:27Z

## Codebase Patterns
- Placeholder Go files use repository SPDX headers and minimal package declarations; behavior should be added in the existing facade/service/cmd layers described by `plan.md`.

## Decisions
- Phase 1 confirmed that generated runtime element instance methods already exist in Camunda v8.8/v8.9 clients and should be used directly by later adapter tasks.

## Gotchas
- Camunda v8.7 has no generated runtime element instance lookup/search methods; keep v87 behavior as explicit unsupported-operation service behavior.

## Reusable Commands
- `rg -n "GetElementInstanceWithResponse|SearchElementInstancesWithResponse" internal/clients/camunda/v87 internal/clients/camunda/v88 internal/clients/camunda/v89`
- `go test ./c8volt/element ./internal/services/element/... ./cmd -run '^$' -count=1`

## Do Not Repeat

## Current Handoff
- Phase 1 setup is complete. Next iteration should start Phase 2 foundational contracts at T005, defining version-neutral element domain types before service/facade wiring.
