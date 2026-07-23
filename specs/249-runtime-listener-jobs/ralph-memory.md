# Ralph Memory

Feature: 249-runtime-listener-jobs
Started: 2026-07-23T09:30:41Z

## Codebase Patterns
- `cmd/get_element.go` has a single global flag block, local validation in `validateGetElementFlags`/`validateGetElementFlagValues`, command request construction in `newGetElementSearchRequest`, then keyed `cli.GetElement` or paged search plus `elementView`/`elementsView`. `--total` already rejects JSON and keys-only locally before client creation.
- `cmd/get_job.go` already exposes listener-source filters: `--kind`, `--listener-event-type`, `--pi-key`, `--element-instance-key`, and `--element-id`. Valid listener job kinds are `EXECUTION_LISTENER` and `TASK_LISTENER`; event values include `START`, `END`, `CREATING`, `COMPLETING`, and related Camunda values.
- `cmd/get_processinstance_enrichment.go` is the shared command orchestration point for process-instance activity enrichment. Activity wrappers call facade enrichment even for zero selected process instances so requested-empty JSON behavior stays consistent.
- `cmd/cmd_views_processinstance_activity.go` preserves requested enrichment with nil versus empty slices. `processInstanceActivityItem.MarshalJSON` includes `elements` only when non-nil, and `mergeProcessInstanceActivity` converts missing requested sections to empty slices. Human section order is `vars:`, `incidents:`, then `elements:`.
- Runtime element row formatting for process-instance activity is centralized in `formatProcessInstanceElementRows` and `flatRowProcessInstanceElementWithTimezone`; listener nesting is handled by `formatProcessInstanceElementTreeLines` plus `formatProcessInstanceElementListenerRows`. Standalone `get element` row formatting is still in `cmd/cmd_views_element.go`.
- `cmd/walk_processinstance.go` enriches walked instances after traversal by converting traversal output with `processInstancesFromTraversal`, then renders combined activity via `activityItemsFromTraversal`, `activityTraversalPayload`, and `walkActivityView`. Existing validation rejects `--with-elements` without `--key` and with `--keys-only`.
- `cmd/ops_analyse_slow_process_instances.go` keeps slow-analysis flag parsing in command-local helpers, maps validated inputs into `ops.SlowProcessAnalysisRequest`, and delegates all analysis mechanics to the ops facade/service.
- `internal/services/processinstance/enrichment.go` owns enrichment attachment mechanics. `EnrichProcessInstancesWithElements` searches elements once per selected process instance, filters returned elements by `ProcessInstanceKey`, and sorts stable by `StartDate` then `ElementInstanceKey`. `EnrichProcessInstancesWithElementListeners` additionally searches jobs twice per selected process instance, once for `EXECUTION_LISTENER` and once for `TASK_LISTENER`, then attaches matched listener jobs by `ElementInstanceKey`.
- Public process facade wiring now uses `process.NewWithElementListeners(pdAPI, piAPI, incAPI, eAPI, jAPI, log)` from `c8volt/client.go`; `NewWithElements` remains as a compatibility constructor with no job service dependency.
- Public facades are thin: process facade methods map public models to domain, delegate to `internal/services/processinstance`, convert with existing helpers, and wrap errors with `ferrors.FromDomain`.
- `internal/services/job.API` already exposes `SearchJobs` and `SearchJobsPage`; v8.8 and v8.9 build generated filters for process instance key, element instance key, kind, listener event type, job type, element ID, worker, retries, and state. v8.7 returns `ErrUnsupported` with messages like `search jobs requires Camunda 8.8 or newer`.
- Domain `Job` already carries the operator-relevant listener fields needed by the feature: key, kind, listener event type, type, state, retries, worker, deadline, process instance key, element instance key, element ID, tenant, error code, and error message.
- Domain/public listener fields use `*[]RuntimeListenerJob`: nil means listeners were not requested; a non-nil empty slice means listener enrichment was requested and no matched listener jobs were found.

## Decisions
- Keep listener ownership and omission logic in internal services rather than CLI renderers. Command code should request enrichment and render attached data only.
- Reuse existing requested-empty semantics by making listener slices nil when not requested and empty when requested but no jobs matched.
- Listener enrichment deliberately omits jobs whose process instance key, listener kind, or element instance key cannot be matched to the selected process-instance elements.

## Gotchas
- Phase 1 tasks T001-T003 are review-only. Ralph rules say not to mark review-only tasks complete when they produce no substantive project change, so they remain unchecked after this setup audit.
- `get element --with-listeners` must reject `--keys-only` and `--total` before client creation; `get element --total` already rejects JSON and keys-only but has no listener-specific validation yet.
- Process and walk listener enrichment must preserve explicit-key admin input behavior by passing the same option sets currently used for element enrichment.
- v8.8/v8.9 job search can return listener jobs, but v8.7 unsupported behavior must be allowed to flow through the normal facade/domain error conversion path.
- `EnrichProcessInstancesWithElementListeners` performs listener job lookup even when a selected process instance has zero returned elements, so requested listener lookup errors such as v8.7 unsupported still surface.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./internal/services/processinstance ./c8volt/process ./cmd -run 'Test.*Listener' -count=1`
- `go test ./cmd ./c8volt/element -run 'Test(GetElement|CommandContract|GeneratedDocs).*Listener' -count=1`

## Do Not Repeat
- Do not add per-element listener job lookups in command renderers; listener lookup should happen once per selected process instance and attach by element instance key in service/facade paths.
- Do not hand-edit generated Camunda clients for this feature; the existing v8.8/v8.9 job search surface already has the required listener filters.

## Current Handoff
- Phase 2 foundational shared listener enrichment is complete and validated. T001-T003 remain unchecked because they are review-only. Next iteration should start the first incomplete user story, US1, with T014-T017 tests for `get element --with-listeners`, then wire the element facade and command flags through the shared listener models/renderers.
