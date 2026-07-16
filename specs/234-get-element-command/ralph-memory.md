# Ralph Memory

Feature: 234-get-element-command
Started: 2026-07-16T11:27:27Z

## Codebase Patterns
- Placeholder Go files use repository SPDX headers and minimal package declarations; behavior should be added in the existing facade/service/cmd layers described by `plan.md`.
- Element foundational wiring now mirrors the job slice: `internal/domain/element.go` owns version-neutral models, `internal/services/element/api.go` owns service methods and version assertions, `internal/services/element/factory.go` selects v87/v88/v89, and `c8volt/element` stays a thin facade over internal services.
- Public and internal element models keep keys and timestamps as strings, matching the feature data model and existing process-instance-style JSON contracts.
- US1 direct lookup uses the generated `GetElementInstanceWithResponse` endpoint for v88/v89, maps `ElementInstanceResult` directly to `domain.Element`, and uses `common.RequirePayload` for HTTP status/malformed-payload handling.
- US2 search uses generated v88/v89 `SearchElementInstancesWithResponse`, typed generated filters, offset pagination, reported-total metadata, and command-side page iteration modeled after `get job`.
- `get element` now allows no-key search, unfiltered search, AND-combined filters, `--batch-size`, `--limit`, and `--total`; US3 still owns final output-mode polish and command contract metadata.

## Decisions
- Phase 1 confirmed that generated runtime element instance methods already exist in Camunda v8.8/v8.9 clients and should be used directly by later adapter tasks.
- Phase 2 added real v88/v89 service constructors with generated client validation and pending method stubs so aggregate `c8volt.New` can wire `ElementAPI` before story behavior exists.
- US1 kept v87 as explicit `domain.ErrUnsupported` behavior and replaced only the v88/v89 direct-lookup stubs; search stubs still return the pending unsupported error until US2.
- US2 maps `--bpmn-process-id` to generated `processDefinitionId`, because Camunda's element search filter names the BPMN process identifier field that way.
- Element `SearchResult.Total` is the bounded collected count; exact/lower-bound backend totals stay on page metadata for command `--total` and future callers.

## Gotchas
- Camunda v8.7 has no generated runtime element instance lookup/search methods; keep v87 behavior as explicit unsupported-operation service behavior.
- Generated element `ElementInstanceKey` search filter fields are direct key pointers, while `State` uses a union filter wrapper; do not copy the job key union pattern for element key filters.
- US3 should avoid assuming command contract metadata is complete: `get element` is read-only, but full contract/automation metadata and docs examples remain task T041/T048 work.

## Reusable Commands
- `rg -n "GetElementInstanceWithResponse|SearchElementInstancesWithResponse" internal/clients/camunda/v87 internal/clients/camunda/v88 internal/clients/camunda/v89`
- `go test ./c8volt/element ./internal/services/element/... ./cmd -run '^$' -count=1`
- `go test ./internal/services/element/... ./c8volt/element ./c8volt -count=1`
- `go test ./cmd -run 'TestGetElement|TestElement' -count=1`
- `go test ./internal/services/element/... ./c8volt/element -count=1`

## Do Not Repeat

## Current Handoff
- US2 search is complete and validated. Next iteration should start US3 at T036 by adding compact row rendering/output-mode/command-contract tests before polishing `cmd/cmd_views_element.go`, JSON/keys/total behavior, and `get element` contract metadata.
