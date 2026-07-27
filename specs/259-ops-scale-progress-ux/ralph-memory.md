# Ralph Memory

Feature: 259-ops-scale-progress-ux
Started: 2026-07-27T10:17:40Z

## Codebase Patterns
- `toolx/logging.ActivitySink` supports `StartActivity`/`StopActivity`; `ActivityUpdater` adds `UpdateActivity`. `logging.UpdateActivity(ctx, msg)` is a no-op when no updater is installed, so service/command progress plumbing can be safe by default.
- `activityWriter` writes transient spinner/status output through the command stderr/activity path, clears before normal writes, truncates to terminal width, and is disabled when the writer is not an interactive terminal or root disables indicators.
- `testx/activitysink.Sink` records activity start, stop, and update messages with thread-safe snapshots for progress assertions.
- Process-instance page metadata is already normalized in `internal/domain.ProcessInstancePage`: `OverflowState`, `ReportedTotal` with exact/lower-bound kinds, `EndCursor`, and original request fields.
- Service-owned process-instance traversal lives in `internal/services/processinstance.SearchProcessInstancesPages`; it owns backend page advancement, local filtering, limit trimming, cursor/offset fallback, and visitor callbacks.
- `ops analyse slow-process-instances` currently builds facade requests in `cmd/ops_analyse_slow_process_instances.go`, delegates to `cli.AnalyseSlowProcessInstances`, and renders via command-owned view helpers.
- Slow-process discovery currently happens in `internal/services/ops/slow_process_analysis.go` through direct repeated `SearchForProcessInstancesPage` calls, then enrichment uses `pisvc.EnrichProcessInstancesWithElements` or `EnrichProcessInstancesWithElementListeners`.
- Shared ops-scale progress facts now live in `internal/domain/ops_progress.go`; public mirrors and converter helpers live in `c8volt/ops/progress_model.go` and `c8volt/ops/convert.go`.
- `cmd/ops_progress.go` owns reusable wording/gating helpers for total certainty, page count wording, frozen-scope counters, ETA gating, and stdout-safe progress channels.
- Slow-process requests now carry an optional `Progress func(ProgressEvent)` callback through the facade boundary; nil callbacks are safe. Slow-process results can carry `PreflightScope` and `FrozenScopeProgress`.
- The ops service emits snapshot frozen-scope progress events at the start and end of current slow-analysis enrichment, with later US2 tasks expected to replace or expand this with finer per-resource progress.

## Decisions
- For this feature, transient progress should reuse the existing activity context path rather than stdout or a new global writer.
- Human wording, prompts, and output-mode gating belong in `cmd`; services should emit structured facts or callbacks and continue owning traversal/enrichment mechanics.
- Preflight should peek/reuse the first process-instance page where possible; repeated discovery belongs in `internal/services/ops/slow_process_analysis.go`, not in command code.

## Gotchas
- JSON and keys-only stdout must remain clean; progress must never be printed to stdout for those modes.
- Broad preflight is required for process-definition search mode, but explicit-key slow-process mode should stay concise and skip broad preflight.
- `cmd/get_processinstance_paging.go` still has command-local process-instance paging helpers from earlier behavior; new ops-scale traversal should follow the stricter Ralph rule and keep page math in services.
- Lower-bound reported totals are useful for preflight/progress wording but cannot be treated as exact completion or mutation confirmation totals.

## Reusable Commands
- `GOCACHE=/tmp/c8volt-gocache go test ./toolx/logging ./testx/activitysink -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Progress|SlowProcess' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./c8volt/ops -run 'ClientAnalyseSlowProcessInstances' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./internal/services/ops -run 'Progress|SlowProcess' -count=1`

## Do Not Repeat
- Do not add endpoint names, cursors, request URLs, or per-resource lifecycle chatter to default human progress.
- Do not implement slow-process page-until-done loops or first-page reuse in `cmd`; keep it in internal services with tests.
- Do not hand-edit generated CLI docs; update command source and run `make docs-content` when help text changes.

## Current Handoff
- Next iteration should start US1 tasks T012-T020; build preflight-scope construction and first-page reuse on top of the existing progress models and keep broad-search mechanics in `internal/services/ops/slow_process_analysis.go`.
