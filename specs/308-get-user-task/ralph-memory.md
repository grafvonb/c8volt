# Ralph Memory

Feature: 308-get-user-task
Started: 2026-09-13T11:20:06Z

## Codebase Patterns

- CLI construction, validation, metadata, and dispatch belong in `cmd`; public models and thin delegation belong in `c8volt/task`; version-neutral workflow state and mechanics belong in `internal/domain` and `internal/services/usertask`; generated-client request differences stay in `v87`, `v88`, `v89`, and `v810` adapters.
- `c8volt.API` already embeds `task.API`; extend the existing task facade and service factory rather than adding a second wiring path.
- Use `common.RequirePayload`, `common.EffectiveTenant`, `foptions` mapping, `ferrors.FromDomain`, `typex.Keys.Unique`, `toolx.DetermineNoOfWorkers`, and `toolx/pool.ExecuteSlice` before introducing helpers.

## Decisions

- Preserve legacy `GetUserTask` resolver behavior, including v88/v89 tenant-search and Tasklist fallback plus v810 tenant/identity checks. New direct reads use a separate `GetNativeUserTask` path with backend authorization, no discovery-tenant post-filter, and no Tasklist fallback.
- Keep traversal, sparse-page continuation, limits, worker scheduling, and exact-total fallback in internal services; command visitors only render or decide whether to continue.
- Camunda 8.8, 8.9, and 8.10 support the new native reads; 8.7 must return the established unsupported domain error without issuing a request.

## Gotchas

- `c8volt/task` currently has no tests; its baseline command succeeds with `[no test files]`. T004 and later facade tasks add the required coverage.
- With `commit.issue: auto`, branch `codex/308-get-user-task` has no leading numeric prefix, so Ralph commit subjects omit an issue suffix.
- The full race suite spends several minutes in `cmd`; lack of interim output is normal when the process remains active.

## Reusable Commands

- `go test ./internal/services/usertask/... -count=1`
- `go test ./c8volt/task -count=1`
- `go test ./cmd -run 'TestGetProcessInstanceCommand_(HasUserTasks|RejectsHasUserTasks)|TestGetProcessInstanceHelp_DocumentsHasUserTasksLookup' -count=1`
- `make test`

## Do Not Repeat

- Do not change the legacy resolver getter to satisfy native keyed-read semantics; add the distinct native API required by the plan.
- Do not infer completion from an empty or short search page when continuation evidence remains.

## Current Handoff
- Continue with T003 in Phase 2: add and validate the version-neutral domain task, query, page, visitor, total, and completion models without changing legacy resolver behavior.
