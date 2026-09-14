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
- The keyed `get user-task` command owns only input, validation, dispatch, metadata, and rendering. Its focused stdin reader consumes explicit dash input or nonterminal implicit input, preserves the shared 10 MiB scanner ceiling, allows an empty implicit stream to reach future search, and never consumes terminal stdin to infer keys.
- User-task command output uses one collection payload for every keyed cardinality, JSON-before-keys-before-human precedence, quiet suppression only for human output, aligned contract-order rows with element-ID name fallback, and writer errors propagated from human/key lines.
- Keyed-only help must label search/filter/limit/total flags as reserved until US2 implements them. Keep the established `get` short summary stable because command help, shell-completion, and docsgen tests treat it as a compatibility string; add new resource discoverability in the parent long text, examples, and generated command tree.
- Native search adapters use limit pagination for an initial position, offset pagination only when `From` is nonzero, and forward-cursor pagination when `After` is set. V810 process-instance, process-definition-key, and process-definition-ID selectors are generated equality unions; V88/V89 use scalar pointers while assignment, candidate, state, and tenant predicates remain equality unions.
- One-page search adapters always preserve raw item count separately, map exact versus capped totals, treat an advancing cursor on capped results as continuation, and leave capped no-cursor exhaustion indeterminate for the shared traversal. Search results validate key, state, and owning-process identity before crossing the adapter boundary.
- Shared user-task traversal prefers unseen advancing cursors, otherwise advances offsets by raw count for nonempty pages or requested size for sparse empty pages. It validates request echoes, counts, totals, continuation, cursor cycles, and arithmetic before returning a typed exhausted, limit-reached, or visitor-stopped result.
- Exact user-task counts return trustworthy exact metadata immediately. Capped counts reuse the same validated walker with int64 raw progress, no caller limit or visitor, and no accumulated task collection; an empty indeterminate probe ends traversal only after the known lower bound has been observed.
- Public user-task search methods map requests, options, visitor steps/actions, result metadata, and errors mechanically around the service-owned collected, paged, and exact-total workflows; facade conversion preserves independently owned candidate slices and non-nil empty collections.
- Command search validation normalizes only the closed nine-state lifecycle set, maps `all` to no predicate, trims identifiers without case-folding assignee/candidate values, and keeps the 1000 page-size and positive explicit-limit boundary in `cmd/get_usertask.go`.
- User-task paging uses `cmd/get_usertask_search.go`: human and keys modes stream selected pages, JSON/quiet/auto-confirm collect one bounded result, sparse and indeterminate pages continue without prompting, and only authoritative `has_more` pages with items are eligible for the shared stderr prompt.
- Combined output acceptance belongs in `cmd/get_usertask_output_test.go`: exercise real command subprocesses, decode exactly one JSON envelope through EOF, compare human/keys/empty/total bytes exactly, and prove each render path adds no backend read. Debug diagnostics stay on stderr while verbose keys-only output remains clean.
- Real-terminal user-task paging belongs in `cmd/get_usertask_terminal_test.go`: drive the actual command with `testx.NewCmdTerminalRunner`, keep prompt text on configured or inherited stderr, and explicitly auto-continue collected JSON as well as automation/auto-confirm modes.

## Gotchas

- `c8volt/task` currently has no tests; its baseline command succeeds with `[no test files]`. T004 and later facade tasks add the required coverage.
- Ralph `commit.issue: auto` cannot infer an issue from the nonnumeric-leading `codex/308-get-user-task` branch, so orchestrator-validated conventional subjects for this branch must omit an issue suffix.
- The full race suite spends several minutes in `cmd`; lack of interim output is normal when the process remains active.
- `pool.ExecuteSlice` may return no pool error when a context is already canceled before non-fail-fast work is scheduled, so strict workflows must check `ctx.Err()` before treating the returned slots as success.
- Adding a canonical command requires updating `specs/254-cli-debt-refactor/assessment.md`; `TestCapabilityDocumentForRoot_CoversCLIDebtAssessment` compares the live capability inventory against that historical assessment table.
- The canonical `get user-task` node raises the generated command-tree and CLI-debt assessment inventory from 55 to 56; keep the assessment prose and `docsgen/main_test.go` count guards synchronized with its row.

## Reusable Commands

- `go test ./internal/services/usertask/... -count=1`
- `go test ./internal/services/usertask -run 'TestSearchUserTasks' -count=10`
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
- Continue US3 at T036: add failure-after-streaming and legacy resolver regression coverage; use it to finish and validate T037 error integration before proceeding to later US3 tasks.
