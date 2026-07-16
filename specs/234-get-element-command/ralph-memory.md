# Ralph Memory

Feature: 234-get-element-command
Started: 2026-07-16T11:27:27Z

## Codebase Patterns
- Placeholder Go files use repository SPDX headers and minimal package declarations; behavior should be added in the existing facade/service/cmd layers described by `plan.md`.
- Element foundational wiring now mirrors the job slice: `internal/domain/element.go` owns version-neutral models, `internal/services/element/api.go` owns service methods and version assertions, `internal/services/element/factory.go` selects v87/v88/v89, and `c8volt/element` stays a thin facade over internal services.
- Public and internal element models keep keys and timestamps as strings, matching the feature data model and existing process-instance-style JSON contracts.
- US1 direct lookup uses the generated `GetElementInstanceWithResponse` endpoint for v88/v89, maps `ElementInstanceResult` directly to `domain.Element`, and uses `common.RequirePayload` for HTTP status/malformed-payload handling.
- The keyed command path is registered as `get element --key`; search flags are present for validation and mutual exclusion, but no-key/search execution intentionally remains unimplemented for US2.

## Decisions
- Phase 1 confirmed that generated runtime element instance methods already exist in Camunda v8.8/v8.9 clients and should be used directly by later adapter tasks.
- Phase 2 added real v88/v89 service constructors with generated client validation and pending method stubs so aggregate `c8volt.New` can wire `ElementAPI` before story behavior exists.
- US1 kept v87 as explicit `domain.ErrUnsupported` behavior and replaced only the v88/v89 direct-lookup stubs; search stubs still return the pending unsupported error until US2.

## Gotchas
- Camunda v8.7 has no generated runtime element instance lookup/search methods; keep v87 behavior as explicit unsupported-operation service behavior.
- v88/v89 element search methods still return `domain.ErrUnsupported` with "service implementation is pending"; US2 must replace those stubs before unkeyed command execution is useful.
- `cmd/get_element.go` currently returns a local precondition error when no `--key` is supplied. US2 should replace that branch with search request construction and paging.

## Reusable Commands
- `rg -n "GetElementInstanceWithResponse|SearchElementInstancesWithResponse" internal/clients/camunda/v87 internal/clients/camunda/v88 internal/clients/camunda/v89`
- `go test ./c8volt/element ./internal/services/element/... ./cmd -run '^$' -count=1`
- `go test ./internal/services/element/... ./c8volt/element ./c8volt -count=1`
- `go test ./cmd -run 'TestGetElement|TestElement' -count=1`

## Do Not Repeat

## Current Handoff
- US1 direct lookup is complete and validated. Next iteration should start US2 at T024 by adding v88/v89 search service tests, facade search page/result mapping tests, and command search validation tests before replacing the pending search stubs.
