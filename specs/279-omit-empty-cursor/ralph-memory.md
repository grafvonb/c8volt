# Ralph Memory

Feature: 279-omit-empty-cursor
Started: 2026-08-26T19:53:00Z

## Codebase Patterns
- Existing latest-definition adapter tests are `TestService_SearchProcessDefinitionsLatestForcesLatest` in `internal/services/processdefinition/v88`, `v89`, and `v810`.
- Shared process-definition traversal coverage lives in `TestSearchProcessDefinitionsPagesUsesCursorTraversal`; it now asserts exact opaque cursor propagation into the next `ProcessDefinitionPageRequest` and two-page termination when the final page has an empty cursor.

## Decisions

## Gotchas
- Baseline before implementation: v8.8 asserts `camundav88.EndCursor("")` on the latest request page, while v8.9 and v8.10 assert `body.Page.After` is present and equals `""`.

## Reusable Commands
- `GOCACHE=/tmp/c8volt-gocache go test ./internal/services/processdefinition/v88 ./internal/services/processdefinition/v89 ./internal/services/processdefinition/v810 -run TestService_SearchProcessDefinitionsLatestForcesLatest -count=1`

## Do Not Repeat

## Current Handoff
- Next iteration should start with T003 in `internal/services/processdefinition/v89/service_test.go`: replace the empty-cursor latest-search expectation with serialized-wire assertions for initial latest requests.
