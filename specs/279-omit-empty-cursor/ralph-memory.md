# Ralph Memory

Feature: 279-omit-empty-cursor
Started: 2026-08-26T19:53:00Z

## Codebase Patterns
- Existing latest-definition adapter tests are `TestService_SearchProcessDefinitionsLatestForcesLatest` in `internal/services/processdefinition/v88`, `v89`, and `v810`.
- Shared process-definition traversal coverage lives in `TestSearchProcessDefinitionsPagesUsesCursorTraversal`; it now asserts exact opaque cursor propagation into the next `ProcessDefinitionPageRequest` and two-page termination when the final page has an empty cursor.
- v8.9 latest initial-page wire assertions use `decodeProcessDefinitionSearchPageFields` to inspect raw serialized page keys because generated union accessors do not prove forbidden fields are absent.

## Decisions
- v8.9 process-definition paging now branches in adapter code: non-empty `After` uses `CursorForwardPagination`, initial latest uses generated `LimitPagination`, and ordinary searches keep `OffsetPagination`.

## Gotchas
- Baseline still remains in v8.8 and v8.10: their latest-search tests assert an empty initial cursor until US2 updates those adapters.
- Local profile `kind-camunda-platform-local-c89` failed the T006 read-only connection/version check with OAuth `unauthorized_client` / invalid client credentials against `localhost:18080`; do not mark T006 complete until a disposable Camunda 8 Run 8.9.17 H2/RDBMS profile authenticates and creates exactly ten keys.
- Local profile `c89local` authenticates but currently points at gateway `8.10.0-alpha4`, so it does not satisfy the 8.9.17 H2/RDBMS live-proof requirement.
- T006 passed by downloading official `camunda8-run-8.9.17-darwin-aarch64.zip`, starting extracted C8 Run on port `18089`, using `auth.mode: none`, and verifying `/tmp/c8volt-279-process-instance-keys.txt` contains exactly ten keys.

## Reusable Commands
- `GOCACHE=/tmp/c8volt-gocache go test ./internal/services/processdefinition/v88 ./internal/services/processdefinition/v89 ./internal/services/processdefinition/v810 -run TestService_SearchProcessDefinitionsLatestForcesLatest -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./internal/services/processdefinition/v89 -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'ProcessDefinition|RunProcessInstance|Selector' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go build -o /tmp/c8volt-279 .`
- `/tmp/c8volt-279 --config /tmp/c8volt-279-c8run.yaml --json config test-connection`
- `/tmp/c8volt-279 --config /tmp/c8volt-279-c8run.yaml get cluster version`
- `/tmp/c8volt-279 --config /tmp/c8volt-279-c8run.yaml embed deploy --file processdefinitions/C89_SimpleUserTask.bpmn`
- `/tmp/c8volt-279 --config /tmp/c8volt-279-c8run.yaml --keys-only run process-instance --bpmn-process-id C89_SimpleUserTask --count 10 --workers 4 | tee /tmp/c8volt-279-process-instance-keys.txt`

## Do Not Repeat

## Current Handoff
- Next iteration should start User Story 2 with T007: add the failing v8.8 initial-latest plus continuation/final-page wire assertions in `internal/services/processdefinition/v88/service_test.go`, including exact preservation of a non-empty opaque cursor.
