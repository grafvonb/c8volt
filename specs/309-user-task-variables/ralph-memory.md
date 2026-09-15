# Ralph Memory

Feature: 309-user-task-variables
Started: 2026-09-15T08:04:53Z

## Codebase Patterns

- The implemented #308 base command is `cmd/get_usertask.go`; it owns Cobra setup, key/search validation, top-level dispatch, aliases `user-tasks`/`ut`/`uts`, and delegates paging to `searchUserTasksWithPaging`.
- Active feature selection is duplicated intentionally: `.specify/feature.json` names `specs/309-user-task-variables`, while `AGENTS.md` names its `plan.md` under the active Speckit marker.
- User-task effective-variable paging records live beside the existing task paging records in `internal/domain/usertask.go`; the offset-only request validates `From >= 0` and `Size > 0`, while the page reuses `UserTaskReportedTotal` and `ProcessInstanceVariable`.
- Public effective user-task variables are a true alias of `process.ProcessInstanceVariable`; task-specific enriched wrappers live in `c8volt/task/model.go`, use `int64` totals, and keep explicit `items`/`variables` JSON arrays.
- Command variable tests use `newGetUserTaskVariablesServer` in `cmd/get_usertask_vars_test.go`; it combines stable native task reads, the existing search response callback shape, and task-local effective-variable page queues without changing command globals.
- The v810 effective-variable adapter lives in `internal/services/usertask/v810/variables.go`; it uses pointer-backed raw DTO fields to distinguish missing required JSON from valid empty strings, while `isTruncated` takes precedence over `truncated`.
- The v89 generated effective-variable operation is request-compatible with v810, but its explicit adapter and client-double method remain version-owned in `internal/services/usertask/v89`; raw decoding is still required because the generated success model omits value and truncation fields.

## Decisions

- No setup divergence blocks implementation. Feature #309 extends the existing #308 command; its current help text intentionally excludes variables until the later user-story tasks update behavior and documentation.
- Preserve the Go 1.26/toolchain 1.26.2 dependency set; this feature adds no dependency.

## Gotchas

- `gh issue view 309` cannot currently validate the remote issue because the configured GitHub credentials return HTTP 401. The committed feature artifacts provide the implementation contract for this iteration.
- Generated sort constants use the package-level `camundav810.ASC` name, and generated effective-variable results omit raw value/truncation fields even when `JSON200` is populated.

## Reusable Commands

- Base command regression: `go test ./cmd -run '^TestGetUserTask' -count=1`
- User-task domain models: `go test ./internal/domain -run 'TestUserTaskVariable' -count=1`
- User-task variable fixture: `go test ./cmd -run '^TestGetUserTaskVariablesFixture$' -race -count=1`
- V810 effective-variable adapter: `go test ./internal/services/usertask/v810 -run '^TestService_SearchUserTaskEffectiveVariablesPage' -race -count=1`
- V89 effective-variable adapter: `go test ./internal/services/usertask/v89 -run '^TestService_SearchUserTaskEffectiveVariablesPage' -race -count=1`

## Do Not Repeat

- Do not treat the base command's current variable exclusion as a conflict; removing it is explicitly deferred to T031 after the behavior exists.

## Current Handoff

- Continue with T007 in US1 by adding equivalent v88 effective-variable adapter tests in `internal/services/usertask/v88/variables_test.go`, including authorization of the selected task key and preservation of actual scope/tenant metadata; then implement T014 if the tests fail only for the missing adapter.
