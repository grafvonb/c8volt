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
- Slow-process requests now carry optional `Progress func(ProgressEvent)` and `ConfirmPreflight func(PreflightScope) error` callbacks through the facade boundary; nil callbacks are safe. Slow-process results can carry `PreflightScope` and `FrozenScopeProgress`.
- Slow-process process-definition search now peeks the first `SearchForProcessInstancesPage`, emits/stores preflight metadata from its reported-total/page metadata, confirms through the command callback when configured, and then consumes that same first page before following `EndCursor`/offset continuation.
- `cmd/ops_analyse_slow_process_instances.go` configures preflight rendering only for process-definition search mode. Human/verbose/debug modes can print compact preflight lines to stderr and prompt; JSON, keys-only, quiet, and automation modes skip the prompt and keep stdout clean. Explicit-key mode leaves progress and confirmation callbacks nil.
- The ops service emits slow-process discovery page progress after each collected page, using the first preflight page's page-count certainty as the best available traversal count metadata.
- Slow-process enrichment now emits exact frozen-scope progress at start and after each process-instance enrichment. `--with-listeners` uses the `loading listener jobs` phase; default enrichment uses `loading runtime elements`.
- `cmd/ops_analyse_slow_process_instances.go` routes slow-process page/frozen progress to `logging.UpdateActivity` for default human mode and writes durable page/counter lines only for verbose/debug progress channels. Preflight remains durable in allowed human modes from US1.
- Slow-process machine-mode safety is covered through callback-level tests: JSON, keys-only, quiet, and automation channels suppress preflight/progress stdout, stderr, prompts, and transient activity. Debug mode now reaches the shared progress channel through `flagDebug`.
- Slow-process JSON result rendering carries auditable `preflightScope` and `frozenScopeProgress` metadata, while transient callback events remain excluded from JSON and keys-only output.
- Slow-process output-mode capability metadata now documents that JSON stdout remains a single document and keys-only stdout remains one process-instance key per line without progress/preflight text.
- ETA timing now uses `internal/domain.NewOpsETASampleWindow` with `OpsDefaultETAMinimumSamples=3` and `OpsMinimumTimingElapsed=1s`; elapsed/rate/remaining stay absent for sub-threshold fast phases and ETA requires exact frozen totals plus remaining work.
- Slow-process enrichment updates frozen-scope progress with elapsed/rate/ETA after each root once timing is useful. Completion events keep elapsed/rate but omit stale ETA when done equals total.
- `cmd/ops_progress.go` renders frozen progress with percent complete only when elapsed timing is present and total is exact/nonzero; approximate throughput/remaining use `~` wording. Standalone ETA events are ignored unless `opsETAAllowed` passes.
- Phase 7 inventory lives in `specs/259-ops-scale-progress-ux/coverage.md`. It records current coverage and gaps for basic get commands, process-definition inspection/purge, process-instance cancel/delete/walk/run, smoke, retention, purge, and repair workflows.
- Basic get commands (`get process-instance`, `get incident`, `get job`, `get element`) already have service-owned page traversal and stdout-safe incremental/JSON behavior, but still use older command progress summaries instead of shared ops preflight/page/frozen progress.
- Ops purge/retention/repair workflows already own discovery, freeze, planning, mutation, and audit-report semantics in services and command wrappers; follow-up progress should add shared callbacks/events without moving those backend mechanics into `cmd`.
- `ops purge orphan-process-instances` has an existing service-level orphan discovery progress callback shape in `internal/services/processinstance/orphan_discovery.go`; it is not yet mapped to `OpsProgressEvent` or command progress-channel gating.
- `run process-instance` has an operator-provided `--count`, and `internal/services/processinstance/bulk.go` has older periodic bulk progress logging. Treat large-count run progress as exact frozen work and preserve keys-only/JSON stdout contracts.

## Decisions
- For this feature, transient progress should reuse the existing activity context path rather than stdout or a new global writer.
- Human wording, prompts, and output-mode gating belong in `cmd`; services should emit structured facts or callbacks and continue owning traversal/enrichment mechanics.
- Preflight should peek/reuse the first process-instance page where possible; repeated discovery belongs in `internal/services/ops/slow_process_analysis.go`, not in command code.

## Gotchas
- JSON and keys-only stdout must remain clean; progress must never be printed to stdout for those modes.
- Broad preflight is required for process-definition search mode, but explicit-key slow-process mode should stay concise and skip broad preflight.
- `cmd/get_processinstance_paging.go` still has command-local process-instance paging helpers from earlier behavior; new ops-scale traversal should follow the stricter Ralph rule and keep page math in services.
- Lower-bound reported totals are useful for preflight/progress wording but cannot be treated as exact completion or mutation confirmation totals.
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd -count=1` currently trips an unrelated date-sensitive `TestGetProcessInstancePagingFlow` assertion that rejects the current "126 days ago" fixture text; use focused affected command patterns unless that broader test is updated.
- Output-mode contract notes participate in exact `OutputModeContract` comparisons; update both command metadata tests and ops-specific contract tests when adding notes.
- Fast service tests may see zero elapsed/rate by design because timing facts are suppressed below one second; use the `slowProcessAnalysisNow` test clock in `internal/services/ops` for deterministic ETA samples instead of sleeping.

## Reusable Commands
- `GOCACHE=/tmp/c8volt-gocache go test ./toolx/logging ./testx/activitysink -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Progress|SlowProcess' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'OpsAnalyseSlowProcessInstances|OpsProgress|CommandContractOpsAnalyseSlowProcessInstances' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./c8volt/ops -run 'ClientAnalyseSlowProcessInstances' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./internal/services/ops -run 'Preflight|Progress|SlowProcess' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./internal/domain -run 'OpsETA|OpsProgress|ETASample' -count=1`

## Do Not Repeat
- Do not add endpoint names, cursors, request URLs, or per-resource lifecycle chatter to default human progress.
- Do not implement slow-process page-until-done loops or first-page reuse in `cmd`; keep it in internal services with tests.
- Do not hand-edit generated CLI docs; update command source and run `make docs-content` when help text changes.

## Current Handoff
- Next iteration should start the generated follow-up implementation slices at T058, beginning with basic inspection command tests. Preserve coverage.md as the rollout source, keep backend traversal in services/facades, and preserve US3/US4 guarantees: machine stdout stays clean and ETA only appears after exact-scope timing is useful.
