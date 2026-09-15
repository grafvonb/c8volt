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
- The v88 native effective-variable endpoint and generated shapes also match v89/v810; the keyed endpoint receives no discovery tenant predicate, so backend authorization applies while returned scope and tenant metadata remain unchanged.
- The v87 adapter exposes the same version-owned effective-variable page signature but returns `domain.ErrUnsupported` before transport use; the implementation stays in `v87/variables.go` and leaves native task reads and legacy resolver/fallback behavior unchanged.
- Complete effective-variable retrieval lives in `internal/services/usertask/variables.go`: offset advances by raw count or page size for a required empty continuation, capped totals retain their highest lower bound, normalization happens only after retrieval, and enrichment is sequential to preserve task order.
- The public task facade now exposes only `EnrichUserTasksWithVariables`: `c8volt/task/client.go` maps selected tasks and facade options once into the internal enrichment workflow, while `convert.go` initializes both empty item and variable slices and preserves the service returned-count total.
- Keyed CLI enrichment is isolated in `cmd/get_usertask_vars.go`: strict `GetUserTasks` completes first, then one eligible facade enrichment pass runs; absent opt-in, effective keys-only, total, and empty selections skip variable calls while quiet human and JSON precedence retain retrieval.
- Variable presentation is now command-independent in `cmd/cmd_views_variable_values.go`; the process-instance wrapper supplies its existing flag limit, while the baseline user-task view supplies unlimited zero and renders a `vars:` tree without process-age metadata.
- Bounded search enrichment reuses `enrichSelectedUserTasks` from `cmd/get_usertask_vars.go`: incremental human pages enrich only service-trimmed `step.Page.Items` before rendering/prompting, while collected/quiet/unattended modes enrich the final selected collection once; the final streamed summary remains count-only.
- User-task value limits remain presentation-only: `cmd/get_usertask.go` validates explicit `--var-value-limit` dependency/range before reads, and keyed/search dispatch passes the integer explicitly through task views into `variableValueHumanLine`; JSON bypasses the human formatter and process-instance flags remain independent.
- US3 command output coverage lives in `cmd/get_usertask_vars_output_test.go`; fresh task-local fixtures keep request counts isolated across default/zero/positive limits, mixed output modes, validation failures, and quiet retrieval errors.

## Decisions

- No setup divergence blocks implementation. Feature #309 extends the existing #308 command; its current help text intentionally excludes variables until the later user-story tasks update behavior and documentation.
- Preserve the Go 1.26/toolchain 1.26.2 dependency set; this feature adds no dependency.

## Gotchas

- `gh issue view 309` cannot currently validate the remote issue because the configured GitHub credentials return HTTP 401. The committed feature artifacts provide the implementation contract for this iteration.
- Generated sort constants use the package-level `camundav810.ASC` name, and generated effective-variable results omit raw value/truncation fields even when `JSON200` is populated.
- `commit.issue: auto` adds no suffix on `codex/309-user-task-variables` because the branch does not begin with a numeric prefix; iteration 8's subject was repaired accordingly before iteration 9 work.

## Reusable Commands

- Base command regression: `go test ./cmd -run '^TestGetUserTask' -count=1`
- User-task domain models: `go test ./internal/domain -run 'TestUserTaskVariable' -count=1`
- User-task variable fixture: `go test ./cmd -run '^TestGetUserTaskVariablesFixture$' -race -count=1`
- V810 effective-variable adapter: `go test ./internal/services/usertask/v810 -run '^TestService_SearchUserTaskEffectiveVariablesPage' -race -count=1`
- V89 effective-variable adapter: `go test ./internal/services/usertask/v89 -run '^TestService_SearchUserTaskEffectiveVariablesPage' -race -count=1`
- V88 effective-variable adapter: `go test ./internal/services/usertask/v88 -run '^TestService_SearchUserTaskEffectiveVariablesPage' -race -count=1`
- Complete user-task variable service workflow: `go test ./internal/services/usertask -run 'Test(SearchUserTaskEffectiveVariables|EnrichUserTasksWithVariables)' -race -count=1`
- User-task variable facade: `go test ./c8volt/task -run 'Test.*(Variable|Enrich|Convert)' -race -count=1`
- User-task value-limit/output matrix: `go test ./cmd -run 'Test(GetUserTaskVariable|VariableEnrichedUserTasksView|CommandCapabilityForCommand_UserTaskReadContract)' -race -count=1`

## Do Not Repeat

- Do not treat the base command's current variable exclusion as a conflict; removing it is explicitly deferred to T031 after the behavior exists.

## Current Handoff

- Start polish with T033 by updating README and quickstart documentation for the now-complete keyed/search/value-limit behavior; US3 T027–T032 is complete, while generated docs and integrated review/test tasks remain T034–T036.
