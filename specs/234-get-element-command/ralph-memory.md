# Ralph Memory

Feature: 234-get-element-command
Started: 2026-07-16T11:27:27Z

## Codebase Patterns
- Placeholder Go files use repository SPDX headers and minimal package declarations; behavior should be added in the existing facade/service/cmd layers described by `plan.md`.
- Element foundational wiring now mirrors the job slice: `internal/domain/element.go` owns version-neutral models, `internal/services/element/api.go` owns service methods and version assertions, `internal/services/element/factory.go` selects v87/v88/v89, and `c8volt/element` stays a thin facade over internal services.
- Public and internal element models keep keys and timestamps as strings, matching the feature data model and existing process-instance-style JSON contracts.

## Decisions
- Phase 1 confirmed that generated runtime element instance methods already exist in Camunda v8.8/v8.9 clients and should be used directly by later adapter tasks.
- Phase 2 added real v88/v89 service constructors with generated client validation and pending method stubs so aggregate `c8volt.New` can wire `ElementAPI` before story behavior exists.

## Gotchas
- Camunda v8.7 has no generated runtime element instance lookup/search methods; keep v87 behavior as explicit unsupported-operation service behavior.
- v88/v89 element service methods currently return `domain.ErrUnsupported` with "service implementation is pending"; US1/US2 must replace those stubs with generated-client behavior before command execution is useful.

## Reusable Commands
- `rg -n "GetElementInstanceWithResponse|SearchElementInstancesWithResponse" internal/clients/camunda/v87 internal/clients/camunda/v88 internal/clients/camunda/v89`
- `go test ./c8volt/element ./internal/services/element/... ./cmd -run '^$' -count=1`
- `go test ./internal/services/element/... ./c8volt/element ./c8volt -count=1`

## Do Not Repeat

## Current Handoff
- Phase 2 foundational contracts and aggregate facade wiring are complete. Next iteration should start US1 at T012 by adding v87 unsupported lookup tests, v88/v89 direct lookup tests, facade lookup tests, and command validation tests before replacing the pending direct-lookup stubs.
