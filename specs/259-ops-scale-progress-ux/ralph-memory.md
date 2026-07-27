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
- Basic get commands (`get process-instance`, `get incident`, `get job`, `get element`) have service-owned page traversal, stdout-safe incremental/JSON behavior, shared ops preflight/page progress routing for plain search paths, and frozen enrichment progress for process-instance runtime-element/listener enrichment and element listener enrichment.
- Public facade options now include `foptions.WithProgress(func(foptions.ProgressEvent))`, mapped to `services.WithProgress(func(domain.OpsProgressEvent))`; this avoids a `c8volt/process` -> `c8volt/ops` import cycle while still using the shared internal ops progress model.
- `internal/services/processinstance.EnrichProcessInstancesWithElements` emits exact `loading runtime elements` frozen-scope events at 0 and after each process instance; `EnrichProcessInstancesWithElementListeners` emits `loading listener jobs` over the same process-instance total.
- `internal/services/element.EnrichElementWithListeners` and `EnrichSearchElementsWithListeners` emit `loading listener jobs` frozen-scope events over the process-instance keys that require listener job lookup, not the element row count.
- `cmd/get_processinstance_enrichment.go`, `cmd/get_element.go`, and `cmd/get_element_search.go` append the progress option only through command-owned wrappers. They reuse `formatOpsFrozenScopeProgress` and `opsProgressChannelForMode`: default human uses transient activity updates, verbose/debug can write durable stderr, and JSON/keys-only/quiet/automation stay stdout-safe.
- Ops purge/retention/repair workflows already own discovery, freeze, planning, mutation, and audit-report semantics in services and command wrappers; follow-up progress should add shared callbacks/events without moving those backend mechanics into `cmd`.
- `ops purge orphan-process-instances` has an existing service-level orphan discovery progress callback shape in `internal/services/processinstance/orphan_discovery.go`; it is not yet mapped to `OpsProgressEvent` or command progress-channel gating.
- `run process-instance` has an operator-provided `--count`, and `internal/services/processinstance/bulk.go` has older periodic bulk progress logging. Treat large-count run progress as exact frozen work and preserve keys-only/JSON stdout contracts.
- Basic inspection command tests now include separated stdout/stderr helpers for incident, job, and element search, plus process-instance coverage through the existing helper. These tests assert that future shared progress/preflight text stays out of JSON and keys-only stdout.
- `cmd/ops_progress_test.go` now locks shared preflight resource labels for process instances, incidents, jobs, and elements; reuse `formatOpsPreflightScope`/`formatOpsPageProgress` wording when implementing basic get progress routing.
- Basic inspection searches now route first-page preflight and page discovery through `printBasicSearchOpsProgress` in `cmd/get_processinstance_paging.go`. Default human mode updates activity only; verbose/debug write durable stderr; JSON, keys-only, quiet, and automation suppress progress text through `opsProgressChannelForMode`.
- `get process-instance` progress only trusts backend `ReportedTotal` when `canUsePIReportedTotal()` allows it. Relationship/incident local-filter modes intentionally show unknown totals while still reporting page/seen progress.
- `get incident`, `get job`, and `get element` convert their facade `ReportedTotal` and overflow metadata into `ops.TotalCertainty`/`ops.OverflowState` in their search command files before using the shared command progress renderer.
- Process-definition rollout now has service-owned traversal in `internal/services/processdefinition.SearchProcessDefinitionsPages`, facade mapping through `c8volt/process.SearchProcessDefinitionsPages`, and command-side progress routing for broad `get process-definition`.
- `get process-definition` broad non-latest listing accepts `--batch-size`, traverses all pages below the facade, emits shared preflight/page progress through `printBasicSearchOpsProgress`, and keeps JSON/keys-only stdout progress-free.
- `ops purge all-process-definitions` requests now carry optional progress callbacks; internal discovery emits process-definition preflight and page events from page metadata, and the command renders them through the shared ops progress channel while machine modes stay suppressed.
- `get process-definition` machine-output safety is already active: JSON and keys-only broad listings must keep stdout progress-free and stderr empty while returning the expected process-definition payload/keys.
- The former T063 process-instance mutation progress contract tests are now active in the repo's actual grouped files (`cmd/cancel_test.go`, `cmd/delete_test.go`, and `internal/services/processinstance/dryrun_test.go`). They define expected destructive preflight, discovery page, frozen planning, frozen mutation, and JSON/quiet/automation safety assertions.
- Process-instance mutation planning now emits shared `OpsProgressEvent` facts from `internal/services/processinstance.PlanProcessInstanceMutationPages`: destructive preflight from first page metadata, page discovery progress, and exact frozen planning counters. The facade option bridge in `c8volt/foptions` now preserves preflight/page/frozen fields without importing `c8volt/ops`.
- Bulk `CancelProcessInstances` and `DeleteProcessInstances` now emit exact frozen mutation counters through `services.WithProgress`; the older interval-based activity/logging remains in place for long-running root operations.
- `cmd/processinstance_mutation_progress.go` owns cancel/delete progress rendering. It rewrites the generic planning phase into operation-specific `planning process-instance cancel/delete scope`, routes durable lines only for verbose/debug, updates transient activity when allowed, and suppresses JSON, quiet, and automation progress output.
- T065 added pending T066 command contract tests in `cmd/ops_execute_retention_policy_test.go`, `cmd/ops_purge_orphan_processinstances_test.go`, and `cmd/ops_purge_processinstances_with_incidents_test.go`. They currently skip through `pendingOpsPurgeRetentionProgressT066`; T066 should remove that skip after implementing shared progress routing.
- Retention, orphan purge, and incident-based purge requests now carry optional `Progress func(ProgressEvent)` callbacks through `c8volt/ops` into `internal/services/ops`; nil callbacks remain safe and facade `WithProgress` can fill unset request callbacks.
- T066 progress routing uses `cmd/ops_processinstance_purge_progress.go`: preflight and page events reuse shared ops formatters, frozen planning/deletion counters use the no-comma process-instance mutation counter style, and JSON/quiet/automation remain suppressed by `opsProgressChannelForMode`.
- `internal/services/ops/process_instance_purge_progress.go` emits process-instance purge preflight/page/frozen facts for retention, orphan purge, and incident purge. Orphan preflight intentionally derives cheap first-page scope from existing `OrphanDiscoveryProgress` fields instead of expanding that callback with full page payloads, preserving existing exact progress assertions in `internal/services/processinstance`.
- `DeleteProcessInstances` already emits exact `deleting process instances` frozen counters through `services.WithProgress`; T066 passes each workflow request callback into delete opts so deletion counters appear for confirmed purge/retention runs.
- Human result renderers for retention, orphan purge, and incident purge now write final result lines directly to command stdout even when `--verbose` installs a logger; progress remains on stderr/activity.
- Retention and incident-purge audit reports now include `deleteRequested` to match orphan purge report semantics and T066 report contracts.
- T067 added pending T068 repair progress tests in `cmd/ops_repair_incident_test.go`, `cmd/ops_repair_processinstance_test.go`, and `internal/services/ops/repair_test.go`. They currently skip through `pendingOpsRepairProgressT068` and `pendingRepairProgressT068`; T068 should remove those skips after implementing shared progress routing.
- Repair progress tests define incident-search and process-instance-search preflight/page events, process-instance active-incident lookup counters, frozen planning counters, keyed bulk repair counters, confirmation prompt assertions, and JSON/quiet/automation progress silence.
- `ops repair incident` and `ops repair process-instance` already use the public facade with `collectOptions()`, so T068 can pass `foptions.WithProgress` from command wrappers while keeping service-owned discovery and mutation mechanics in `internal/services/ops/repair.go`.
- Repair requests now carry optional `Progress func(ProgressEvent)` callbacks through `c8volt/ops` into `internal/services/ops`; nil callbacks and `services.WithProgress` fallback remain safe.
- T068 progress routing uses `cmd/ops_repair_progress.go`: preflight/page events reuse shared ops formatters, frozen planning/loading/repair counters use the no-comma mutation counter style, and JSON/quiet/automation modes stay silent.
- `internal/services/ops/repair_progress.go` emits repair search preflight/page facts from first-page metadata, planning counters during dry-run planning, process-instance active-incident lookup counters, and exact `repairing incidents` counters during keyed mutation.
- Repair human result renderers now write final result/report lines directly to command stdout even when `--verbose` installs a logger; progress remains on stderr/activity.
- T069 explicit large-work progress uses `cmd/ops_explicit_large_work_progress.go`: default human updates transient activity, verbose/debug write exact frozen counters to stderr, and JSON/keys-only/quiet/automation remain progress-free.
- Bulk `run process-instance --count` now emits exact `starting process instances` counters from `internal/services/processinstance.CreateNProcessInstances`, mapped through `c8volt/process` facade progress options.
- `walk process-instance` now passes progress options through ancestry/descendants/family traversal and incident/variable enrichment. Traversal result builders emit exact completed walk-scope counters once the immutable result set is known; element/listener enrichment still uses the existing enrichment progress wrappers.
- `ops execute smoke-test` requests now carry optional progress callbacks through `c8volt/ops`; the service emits stage-level deployment and family-walk counters, preserves progress through nested create/delete cleanup options, and suppresses lower-level family traversal progress inside smoke-test walk loops so smoke output stays stage-oriented.

## Decisions
- For this feature, transient progress should reuse the existing activity context path rather than stdout or a new global writer.
- Human wording, prompts, and output-mode gating belong in `cmd`; services should emit structured facts or callbacks and continue owning traversal/enrichment mechanics.
- Preflight should peek/reuse the first process-instance page where possible; repeated discovery belongs in `internal/services/ops/slow_process_analysis.go`, not in command code.

## Gotchas
- JSON and keys-only stdout must remain clean; progress must never be printed to stdout for those modes.
- Broad preflight is required for process-definition search mode, but explicit-key slow-process mode should stay concise and skip broad preflight.
- `cmd/get_processinstance_paging.go` still has command-local process-instance paging helpers from earlier behavior; new ops-scale traversal should follow the stricter Ralph rule and keep page math in services.
- Lower-bound reported totals are useful for preflight/progress wording but cannot be treated as exact completion or mutation confirmation totals.
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd -count=1` passed during T059 after narrowing a process-instance paging assertion that accidentally matched relative age text.
- Output-mode contract notes participate in exact `OutputModeContract` comparisons; update both command metadata tests and ops-specific contract tests when adding notes.
- Fast service tests may see zero elapsed/rate by design because timing facts are suppressed below one second; use the `slowProcessAnalysisNow` test clock in `internal/services/ops` for deterministic ETA samples instead of sleeping.

## Reusable Commands
- `GOCACHE=/tmp/c8volt-gocache go test ./toolx/logging ./testx/activitysink -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Progress|SlowProcess' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'OpsAnalyseSlowProcessInstances|OpsProgress|CommandContractOpsAnalyseSlowProcessInstances' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./internal/services/processinstance ./internal/services/element -run 'Enrich.*Progress|EnrichProcessInstancesWithElements|EnrichProcessInstancesWithElementListeners|EnrichSearchElementsWithListeners|EnrichElementWithListeners' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./c8volt/process ./c8volt/element -run 'Progress|EnrichProcessInstancesWithElements|SearchElementsWithListeners' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./c8volt/ops -run 'ClientAnalyseSlowProcessInstances' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./internal/services/ops -run 'Preflight|Progress|SlowProcess' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./internal/domain -run 'OpsETA|OpsProgress|ETASample' -count=1`

## Do Not Repeat
- Do not add endpoint names, cursors, request URLs, or per-resource lifecycle chatter to default human progress.
- Do not implement slow-process page-until-done loops or first-page reuse in `cmd`; keep it in internal services with tests.
- Do not hand-edit generated CLI docs; update command source and run `make docs-content` when help text changes.

## Current Handoff
- Next iteration should start Phase 8 polish with T049/T050 help and command contract wording for the completed ops-scale progress behavior.
