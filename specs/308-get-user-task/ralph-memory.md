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
- Domain user-task paging uses task-specific closed enums with validation: reported totals are `exact`/`lower_bound`, continuation is `has_more`/`no_more`/`indeterminate`, visitor actions are `continue`/`stop`, and successful completion is `exhausted`/`limit_reached`/`visitor_stopped`.
- `UserTaskSearchPage.RawItemCount` remains distinct from selected `Items`; traversal steps and results use `int64` selected counts so later exact-count work does not narrow backend populations.
- Public task models use required JSON fields for `key`, `state`, and `processInstanceKey`; `UserTasks` always has required `total` and `items`, with nil domain collections normalized to a non-nil empty public slice. Facade converters copy task candidate slices and mechanically map page visitor actions/errors without interpreting traversal state.
- Legacy process ownership resolution performs one lookup per supplied task key and preserves input order. The v88/v89 resolver remains tenant-scoped on primary search and enforces tenant, returned task identity, and owning-process identity after Tasklist fallback; v810 enforces returned task identity, configured tenant visibility, and owning-process identity on its native resolver lookup.
- Native direct reads now use generated `GetUserTaskWithResponse` on v88/v89/v810, map every stable domain field with copied candidate slices, validate returned key/state/process-instance identity, preserve backend tenant metadata without discovery-tenant filtering, and never use search or Tasklist fallback. V87 returns `ErrUnsupported` without transport use.
- Shared native bulk reads stable-deduplicate task keys before scheduling, preserve first-input order through `pool.ExecuteSlice`, forward call options unchanged, and return nil results for any joined read or cancellation failure. Empty input returns an initialized empty slice without touching the adapter.
- Public `task.GetUserTask` delegates only to `GetNativeUserTask`; public `task.GetUserTasks` delegates to the service-owned bulk workflow. Both map facade options at the boundary, convert domain failures through `ferrors.FromDomain`, and copy domain results into stable public models without changing constructor or root embedding.

## Gotchas

- `c8volt/task` currently has no tests; its baseline command succeeds with `[no test files]`. T004 and later facade tasks add the required coverage.
- Ralph `commit.issue: auto` cannot infer an issue from the nonnumeric-leading `codex/308-get-user-task` branch, so orchestrator-validated conventional subjects for this branch must omit an issue suffix.
- The full race suite spends several minutes in `cmd`; lack of interim output is normal when the process remains active.
- `pool.ExecuteSlice` may return no pool error when a context is already canceled before non-fail-fast work is scheduled, so strict workflows must check `ctx.Err()` before treating the returned slots as success.

## Reusable Commands

- `go test ./internal/services/usertask/... -count=1`
- `go test ./internal/domain -run 'TestUserTask' -count=1`
- `go test ./internal/domain -count=1`
- `go test ./c8volt/task -count=1`
- `go test ./cmd -run 'TestGetProcessInstanceCommand_(HasUserTasks|RejectsHasUserTasks)|TestGetProcessInstanceHelp_DocumentsHasUserTasksLookup' -count=1`
- `make test`

## Do Not Repeat

- Do not change the legacy resolver getter to satisfy native keyed-read semantics; add the distinct native API required by the plan.
- Do not infer completion from an empty or short search page when continuation evidence remains.
- Follow constitution v2.0.0: documentation-only changes use diff, consistency, and local-link checks; implementation starts with the closest behavior checks. Run `make test` when shared contracts/concurrency or other concrete broad risks justify it, not before every commit. Reuse passing evidence until relevant changes invalidate it.
- Preserve the 2026-09-13 baseline log as historical evidence, not validation of the rebased implementation. The next slice selects checks for its actual changes.

## Current Handoff
- Continue US1 at T009: add keyed command subprocess contract tests before implementing the US1 command/view/input tasks T013–T017; T018 remains the story validation gate.
