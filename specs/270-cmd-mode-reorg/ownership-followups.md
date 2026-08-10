# Ownership Follow-Ups: Command Mode And Concern Reorganization

## Context And Rules Review

- Reviewed `specs/ralph-implementation-rules.md` against `specs/270-cmd-mode-reorg/spec.md`; no conflict found. Both require behavior-preserving file-level reorganization, retained single `cmd` package ownership, focused mode files for distinct lifecycles, presentation-only renderer files, and service/facade ownership for backend traversal, mutation planning, polling, retries, and worker execution.
- Issue #254 assessment remains applicable as the starting baseline: `cmd` owns flags, validation, prompts, render-mode selection, stdout/stderr rendering, command metadata, and help; public facades map public inputs/errors; internal services own backend paging, traversal, frozen discovery, mutation planning, polling, retries, and worker execution.

## Included Moves

### Process-Definition Watch

- `cmd/get_processdefinition.go` currently mixes ordinary lookup construction and execution with watch-specific lifecycle behavior. Watch candidates include `processDefinitionWatchSleep`, `processDefinitionWatchNow`, `runGetProcessDefinitionWatch`, `executeGetProcessDefinitionWatch`, `processDefinitionWatchSlowWarningState`, watch refresh/retry/stop status rendering, watch interval resolution, retry classification, watch snapshot request construction, and watch output-flag validation.
- `cmd/get_processdefinition_test.go` currently contains broad command behavior plus watch-specific tests and helpers. Watch test candidates include repaint cadence, invalid watch interval validation, machine-mode rejection before lookup, retry budget behavior, slow refresh warning state, verbose refresh timing, normal row parity, keyed watch options, interrupt/timeout stop status, watch fake API, harness types, repaint controls, and watch clock helpers.
- Ordinary process-definition command ownership should keep command construction, shared flags, local flag validation, dispatch to watch/XML/key/search, XML/key lookup execution, selector validation, paged list execution, and search-size/ops progress mapping unless a later task creates a more focused paging ownership file.

### Resource Renderers

- `cmd/cmd_views_get.go` currently owns multiple resource renderers and shared flat-row layout. Candidate moves by resource:
  - Process instance: `processInstanceView`, `processInstanceTotalView`, `listProcessInstancesView`, process-instance flat-row formatting, age metadata, and age calculation should move to existing process-instance view ownership.
  - Process definition: `processDefinitionView`, `listProcessDefinitionsView`, `processDefinitionWatchView`, process-definition flat-row formatting, one-line formatting, and statistics row formatting should move to process-definition view ownership.
  - Incident: `listIncidentsView` and `renderIncidentProcessInstanceKeys` should move to incident view ownership while keeping incident detail row helpers in their existing incident/process-instance incident files.
  - Resource: `resourceView`, `resourceItemView`, one-line resource formatting, and `flatRowResource` should move to resource view ownership.
  - Tenant: `listTenantsView`, `tenantView`, one-line tenant formatting, and `flatRowTenant` should move to tenant view ownership.
  - Shared layout: `zeroAsMinus` and any generic flat-row layout helpers should live in focused flat-row rendering ownership.
- `cmd/cmd_views_get_test.go` mirrors the mixed renderer ownership. Candidate splits include flat-row layout tests, process-instance age/list JSON tests, process-definition list alignment tests, incident renderer tests, and shared get-view test helpers.
- `cmd/cmd_views_processinstance_dryrun.go` still performs dry-run planning through facade calls in `planProcessInstanceDryRunPreview` and `planProcessInstanceDryRunPreviewWithOptions`; later US3 tasks should move planning coordination out of renderer ownership and leave payload construction plus terminal/JSON/key rendering in the view file.

## Deferred Ownership Corrections

- Process-definition search paging helpers in `cmd/get_processdefinition.go` remain command-owned today. That is not part of the initial watch move, but later review should decide whether `searchProcessDefinitionsWithPaging`, `resolveGetProcessDefinitionSearchSize`, `processDefinitionOpsReportedTotal`, and `processDefinitionOpsOverflowState` belong in a focused search/paging file.
- `cancel process-instance` and `delete process-instance` already delegate search traversal and page-level dry-run planning to facade/service APIs, but direct-key execution, selector execution, prompts, force checks, result conversion, and final rendering are still colocated. US3 should split selector/search execution from direct-key execution without changing confirmation or frozen-delete semantics.
- `delete process-instance` must preserve the #254 baseline distinction: search-mode non-dry-run freezes every selected page-level delete plan before one aggregate confirmation and mutation. Do not flatten it into page-by-page mutation as part of file reorganization.
- `update job` currently combines command wiring, request parsing, worker outcome request parsing, JSON guardrails, automation checks, backend-state lookup, mutation plan construction, worker-outcome plan construction, and plan precondition validation. US3 should split request, outcome, and plan concerns, while recording any non-mechanical facade/service ownership correction separately.
- `root.go` currently combines root command wiring, config source description, resolver bindings, Viper setup, command-local config flag binding, config retrieval/normalization, automation/indicator policy, missing-config hints, and remote service installation. US3 should split configuration resolution and service installation from root command wiring.
- `ops_analyse_slow_process_instances.go` currently combines command construction, validation, request building, preflight/progress configuration, durable progress printing, filter parsing, and timestamp/duration parsing. US3 should separate validation and progress concerns.
- `ops_progress.go` currently combines progress mode selection, milestone pacing, page progress formatting, preflight scope/consequence formatting, durable line rendering, frozen-scope progress, ETA formatting, and small string normalization helpers. US3 should split mode selection, milestone pacing, and rendering/formatting into focused files.
- Ops workflow discovery must not be generalized across purge, repair, retention, smoke-test, and slow-analysis paths unless a later task proves identical safety and reporting semantics.

## Helper Removals

- No helper is confirmed dead in this setup pass.
- Before removing any helper, check production, test, subprocess-helper, example, and generated-artifact callers with `rg`, then run the nearest targeted tests and `git diff --check`.
- Initial helper audit candidates for later tasks include process-definition watch test harness helpers after test relocation, `zeroAsMinus` after flat-row ownership moves, dry-run uniqueness/formatting helpers after dry-run planning is split, and duplicated update-job parse/plan helpers after US3 file splits.

### Helper Caller Audit Notes

- Foundational T009 caller audit used `rg` across production, tests, specs, docs, and README for the current helper-removal candidates; no helper is removal-ready before the planned ownership moves.
- Process-definition watch helpers are still active in `cmd/get_processdefinition.go` and `cmd/get_processdefinition_test.go`; US1 should relocate watch helpers and tests rather than remove them.
- `zeroAsMinus` has current production callers in process-definition statistics rendering in `cmd/cmd_views_get.go`; keep it until shared flat-row and process-definition renderer ownership are split.
- Process-instance dry-run payload, uniqueness, scope-formatting, and rendering helpers are still used by cancel, delete, resolve, paging, ops workflows, and tests; US3 should split planning from rendering before reassessing any helper deletion.
- Update-job request, JSON guardrail, and plan precondition helpers are still used by `cmd/update_job.go` and `cmd/update_job_test.go`; US3 should split them by concern before reassessing duplication or deletion.

## Validation Evidence

- Setup artifact review used `rg` over `specs/254-cli-debt-refactor/assessment.md` and declaration inventories for `cmd/get_processdefinition.go`, `cmd/get_processdefinition_test.go`, `cmd/cmd_views_get.go`, `cmd/cmd_views_get_test.go`, `cmd/cmd_views_processinstance_dryrun.go`, `cmd/update_job.go`, `cmd/cancel_processinstance.go`, `cmd/delete_processinstance.go`, `cmd/root.go`, `cmd/ops_analyse_slow_process_instances.go`, and `cmd/ops_progress.go`.
- Go behavior tests were not required for this artifact-only setup work unit; no Go source or test code was changed.
