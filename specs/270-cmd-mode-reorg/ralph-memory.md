# Ralph Memory

Feature: 270-cmd-mode-reorg
Started: 2026-08-10T11:35:19Z

## Codebase Patterns

- For #270, `cmd` owns command construction, flags, validation, prompts, render-mode selection, stdout/stderr rendering, command metadata, and help; public facades map public inputs/errors; internal services own backend paging, traversal, frozen discovery, mutation planning, polling, retries, and worker execution.
- #254 assessment is the baseline for command ownership risk. Preserve its explicit `delete process-instance` frozen aggregate delete semantics and avoid generic ops workflow extraction unless identical safety/reporting semantics are proven.

## Decisions

- T001 found no conflict between `specs/ralph-implementation-rules.md` and `specs/270-cmd-mode-reorg/spec.md`.
- Created `specs/270-cmd-mode-reorg/ownership-followups.md` as the durable tracking artifact for included moves, deferred ownership corrections, helper removals, and validation evidence.
- T007 added `TestCommandContractFocusedModeFilesOwnLifecycleDeclarations` in `cmd/command_contract_test.go`; it parses top-level Go declarations and tracks process-definition watch lifecycle declarations in the current base-file baseline until `cmd/get_processdefinition_watch.go` exists, then requires those declarations to move there.
- T008 added `TestGetViewFilesAvoidBackendOwnership` in `cmd/cmd_views_get_test.go`; it parses `cmd_views_*.go` and fails on internal-service imports or public facade calls from renderer files. The temporary dry-run planning allowlist was removed by T041.
- T009 recorded helper caller audit notes in `ownership-followups.md`; no candidate helper is removal-ready before its planned ownership split.
- T011 added focused watch snapshot request behavior tests in `cmd/get_processdefinition_watch_test.go` without creating `cmd/get_processdefinition_watch.go`; creating the production mode file must wait for T015/T016 because the existing contract test will then require all watch lifecycle declarations to move.
- T012 added `TestGetProcessDefinitionBaseDispatchSkipsWatchLifecycle` in `cmd/get_processdefinition_test.go`; it keeps `flagGetPDWatchInterval` intentionally invalid and verifies ordinary list, key, and XML process-definition paths still bypass watch lifecycle validation and output.
- T013 extended `TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata` to pin process-definition watch discovery metadata, unsupported automation status, summary text, and the help text documenting JSON/keys-only/XML/quiet/automation rejection before lookup.
- T014 added `TestGetProcessDefinitionWatchOutputParityAssertions` in `cmd/get_processdefinition_watch_test.go`; it pins human/verbose refresh stdout parity with normal rows and local rejection of JSON, keys-only, quiet, and automation modes before watch refresh work.
- T015/T016 created `cmd/get_processdefinition_watch.go` and moved the guarded process-definition watch lifecycle declarations there. `cmd/get_processdefinition.go` now keeps process-definition command construction, flags, validation, dispatch, XML/key/search execution, and shared ordinary lookup logic.
- T017 moved process-definition watch scenarios, subprocess rejection helper, and watch harness helpers from `cmd/get_processdefinition_test.go` to `cmd/get_processdefinition_watch_test.go`. The base test file now keeps selector/filter, non-watch machine modes, base dispatch, XML/search, paging activity, and shared ordinary helpers.
- T018-T020 completed the US1 audit and checkpoints: watch lifecycle declarations remain in `cmd/get_processdefinition_watch.go`, base validation still owns incompatible watch output-mode rejection, and both watch plus non-watch process-definition targeted commands passed.
- T021 created `cmd/cmd_views_processinstance_test.go` and moved process-instance row, list, age metadata, variable enrichment, incident enrichment, activity enrichment, and process-instance incident-line tests out of `cmd/cmd_views_get_test.go`; shared flat-row, process-definition, and plain incident renderer tests remain in the old mixed file until their US2 tasks.
- T022 created `cmd/cmd_views_processdefinition_test.go`, moved the process-definition human list alignment test there, and added single-item human/JSON/keys-only, list JSON/keys-only, and watch-list parity renderer tests.
- T023 created `cmd/cmd_views_incident_test.go` and moved plain incident renderer coverage there: aligned human rows, invalid timestamp age handling, message truncation, list human/no-message/JSON/keys-only modes, and process-instance-key output.
- T024 created `cmd/cmd_views_resource_test.go` and `cmd/cmd_views_tenant_test.go`; resource lookup now has direct human/JSON/keys-only renderer coverage, and tenant list plus single-item rendering now has human/JSON/keys-only coverage.
- T025 created `cmd/cmd_views_flat_test.go`, moved the shared flat-row alignment test out of `cmd/cmd_views_get_test.go`, and added shared coverage for all-empty optional column omission plus compact single-row empty-field skipping.
- T026 created `cmd/cmd_views_flat.go`, moved `flatRow`, `formatFlatRows`, `flatColumnWidths`, `hasVisibleColumnAfter`, `compactFlatRow`, and `zeroAsMinus` into focused shared flat-row rendering ownership, and left `cmd/cmd_views_rendermode.go` focused on mode selection and shared render dispatch.
- T027 moved default process-instance list/single/total rendering, flat-row formatting, and age metadata helpers from `cmd/cmd_views_get.go` to `cmd/cmd_views_processinstance.go`; process-instance enrichment files continue to use the same shared process-instance age and row helpers.
- T028 created `cmd/cmd_views_processdefinition.go` and moved process-definition list/single/watch rendering plus flat-row formatting there; `cmd/cmd_views_get.go` now retains incident, resource, and tenant renderer declarations for later US2 moves.
- T029 created `cmd/cmd_views_incident.go` and moved incident collection rendering plus incident process-instance-key output there; `cmd/cmd_views_get.go` now retains only resource and tenant renderer declarations for T030/T031.
- T030 created `cmd/cmd_views_resource.go` and moved single-resource lookup rendering plus resource flat-row formatting there; `cmd/cmd_views_get.go` now retains only tenant renderer declarations for T031.
- T031 created `cmd/cmd_views_tenant.go`, moved tenant list/single/one-line/flat-row rendering there, and removed the now-empty mixed `cmd/cmd_views_get.go`.
- T032/T033 completed the US2 renderer ownership audit and checkpoint validation. `TestGetViewFilesAvoidBackendOwnership` and `go test ./cmd -run 'Test.*View|TestRender|Test.*JSON|Test.*KeysOnly|Test.*Flat' -count=1` passed; T041 later removed the deferred dry-run planning exception.
- T034 created `cmd/cmd_views_processinstance_dryrun_test.go` and moved process-instance dry-run preview payload, human/JSON rendering, final-state/delete-blocker messaging, and aggregate summary presentation tests out of `cmd/cancel_test.go` and `cmd/delete_test.go`. Command workflow tests for keyed execution, paging, tenant scoping, mutation guards, and subprocess scenarios remain in the cancel/delete test files for later US3 splits.
- T035 created `cmd/get_processinstance_search_test.go`, `cmd/get_processinstance_paging_test.go`, and `cmd/processinstance_mutation_progress_test.go`; moved process-instance search request-shape tests, get paging/total/progress tests, and shared cancel/delete mutation-progress tests out of the large mixed test files. No production code moved.
- T036 created `cmd/update_job_request_test.go`, `cmd/update_job_outcome_test.go`, and `cmd/update_job_plan_test.go`; `cmd/update_job_test.go` now keeps command wiring/result-view coverage plus shared job-update fake servers and assertion helpers.
- T037 created `cmd/cancel_processinstance_test.go`, `cmd/cancel_processinstance_selector_test.go`, `cmd/delete_processinstance_test.go`, and `cmd/delete_processinstance_selector_test.go`; direct-key/stdin/key-validation tests now live apart from selector/search/paged execution tests, while broad cancel/delete help and process-definition coverage stay in the original files.
- T038 created `cmd/root_config_test.go` and `cmd/root_services_test.go`; `cmd/root_test.go` now keeps root/help/flag-wiring UX tests, config resolution tests live in config ownership, and activity-indicator service/bootstrap tests live in service ownership.
- T039 created `cmd/ops_analyse_slow_process_instances_validation_test.go` and `cmd/ops_analyse_slow_process_instances_progress_test.go`; command/request-shape tests stay in the base slow-process test file, validation rejection tests live in validation ownership, and preflight/progress/channel tests live in progress ownership.
- T040 created `cmd/ops_report_test.go`, `cmd/ops_report_markdown_test.go`, and `cmd/ops_report_json_test.go`; shared preflight/report contract tests moved out of `cmd/ops_progress_test.go` and `cmd/ops_contract_test.go`, while progress mode, milestone, frozen-scope, and ETA tests remain in `cmd/ops_progress_test.go`.
- T041 moved process-instance dry-run facade planning out of `cmd/cmd_views_processinstance_dryrun.go` into `cmd/get_processinstance_paging.go`. `cmd/cmd_views_processinstance_dryrun.go` now owns only payload mapping/aggregation and terminal/JSON/key rendering for dry-run previews and summaries, and `TestGetViewFilesAvoidBackendOwnership` no longer has a dry-run planning allowlist.
- T042 audited process-instance dry-run renderer ownership: `cmd/cmd_views_processinstance_dryrun.go` imports only presentation-facing process domain models, key types, Cobra, and formatting packages; no facade calls or internal-service imports remain. `go test ./cmd -run 'Test(GetViewFilesAvoidBackendOwnership|.*DryRun)' -count=1` passed.
- T043 divided process-instance paging support by concern: `cmd/get_processinstance_search.go` owns search traversal/request construction, `cmd/get_processinstance_paging.go` owns paging/progress decisions plus shared read-search progress, `cmd/get_processinstance_total.go` owns total eligibility/counting support, and `cmd/processinstance_mutation_progress.go` owns mutation page impact/action result shapes plus direct-key dry-run planning payload construction.
- T044 split job update production ownership: `cmd/update_job.go` now owns command flags, metadata, validation dispatch, confirmation, and top-level mutation dispatch; `cmd/update_job_request.go` owns ordinary retry/timeout request parsing and JSON guardrails; `cmd/update_job_outcome.go` owns worker outcome request parsing/submission rendering; `cmd/update_job_plan.go` owns current-job lookup planning, plan construction, worker outcome plan shape, and timeout precondition checks.
- T045 created `cmd/cancel_processinstance_selector.go` for search-derived cancel execution. `cmd/cancel_processinstance.go` now owns Cobra setup, validation, mode dispatch, and direct-key cancel execution through `runCancelProcessInstanceDirect`; selector validation, search-page planning callbacks, dry-run aggregate rendering, continuation prompts, and paged cancel report rendering live in the selector file.
- T046 created `cmd/delete_processinstance_selector.go` for search-derived delete execution. `cmd/delete_processinstance.go` now owns Cobra setup, validation, mode dispatch, direct-key delete execution through `runDeleteProcessInstanceDirect`, direct-key plan/force checks, and init; selector validation, frozen search-plan aggregation, continuation prompts, dry-run aggregate rendering, search-mode mutation submission, and paged delete report rendering live in the selector file.
- T047 created `cmd/root_config.go` and `cmd/root_services.go`. `cmd/root.go` now owns root command globals, Cobra construction, persistent flag registration, top-level bootstrap flow, execution, and usage silencing; `cmd/root_config.go` owns Viper binding/defaults, config source context, config resolution, config hints, and env-key detection; `cmd/root_services.go` owns remote HTTP/auth service installation plus automation/activity indicator decisions.

## Gotchas

- `cmd/get_processdefinition.go` still contains watch flag registration and watch-specific incompatible-output validation because T016 keeps command flags and validation in the base command owner; watch timing resolution and lifecycle execution live in `cmd/get_processdefinition_watch.go`.
- Process-instance direct-key dry-run planning is now in `cmd/processinstance_mutation_progress.go`; keep future dry-run renderer edits limited to payload/view-model construction and rendering unless a later task explicitly moves presentation ownership again.
- Job update backend-state lookup remains in `cmd/update_job_plan.go` as command-side planning coordination for this mechanical split; T053 is still the planned review point for whether backend-state lookup or mutation-plan construction needs a facade/service ownership follow-up.
- Cancel and delete process-instance search-mode execution now have focused production files matching the existing selector test split. Delete search mode still freezes all selected page-level delete previews before one aggregate confirmation and mutation; do not convert it to page-by-page mutation in later cleanup.
- Root bootstrap remains command-layer ownership after T047; service installation still constructs HTTP/auth services in `cmd/root_services.go` because root command bootstrap is the existing owner for wiring those services into context, not backend workflow logic.

## Reusable Commands

- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `rg -n "^(func|type|const|var) " cmd/<file>.go`
- `go test ./cmd -run 'TestCommandContract' -count=1`
- `go test ./cmd -run 'TestCommandContract|Test.*View' -count=1`
- `go test ./cmd -run 'TestGetProcessDefinition.*Watch|TestProcessDefinition.*Watch|TestValidateGetProcessDefinitionWatch|TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata' -count=1`
- `go test ./cmd -run '^TestGetProcessDefinitionBaseDispatchSkipsWatchLifecycle$' -count=1`
- `go test ./cmd -run 'TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata|TestGetProcessDefinitionWatchOutputParityAssertions|TestValidateGetProcessDefinitionWatch|TestGetProcessDefinition.*Watch|TestProcessDefinition.*Watch' -count=1`
- `go test ./cmd -run 'TestCommandContractFocusedModeFilesOwnLifecycleDeclarations|TestGetProcessDefinition.*Watch|TestProcessDefinition.*Watch|TestValidateGetProcessDefinitionWatch|TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata|TestGetProcessDefinitionBaseDispatchSkipsWatchLifecycle' -count=1`
- `go test ./cmd -run 'TestGetProcessDefinition|TestProcessDefinitionSelectorValidationHelpContract' -count=1`
- `go test ./cmd -run 'Test(ProcessInstance|OneLinePI|ListProcessInstances|IncidentEnrichedProcessInstances|VariableEnrichedProcessInstances|ProcessInstanceVariableHumanLine|IncidentHumanLine|GetViewFilesAvoidBackendOwnership|ListProcessDefinitionsView|ListIncidentsView|FormatFlatRows|TruncateIncident)' -count=1`
- `go test ./cmd -run 'Test(ListProcessDefinitionsView|ProcessDefinitionView|ProcessDefinitionWatchView)' -count=1`
- `go test ./cmd -run 'Test(IncidentHumanLineWithMessageLimit|TruncateIncidentHumanMessage|ListIncidentsView|RenderIncidentProcessInstanceKeys|GetViewFilesAvoidBackendOwnership)' -count=1`
- `go test ./cmd -run 'Test(ResourceView|ListTenantsView|TenantView)' -count=1`
- `go test ./cmd -run 'Test.*View|Test.*JSON|Test.*KeysOnly' -count=1`
- `go test ./cmd -run 'TestFormatFlatRows|TestCompactFlatRow' -count=1`
- `go test ./cmd -run 'Test.*Flat|TestGetViewFilesAvoidBackendOwnership|Test.*View|Test.*JSON|Test.*KeysOnly' -count=1`
- `go test ./cmd -run 'TestGetProcessInstance(SearchScaffold|Search_Var|Search_Tenant|Search_HumanOutput|TotalOutput|SearchMachineOutput|PagingFlow)|TestResolvePISearchSize|TestPIContinuationProgress|Test.*ProcessInstance.*Progress' -count=1`
- `go test ./cmd -run 'Test.*(Cancel|Delete).*ProcessInstance' -count=1`
- `go test ./cmd -run 'Test(Root|ProcessInstanceHelp|TimeoutFlag|FlagParse|RetrieveAndNormalizeConfig|AutomationModeEnabled|MissingConfigHint|IndicatorEnabled)' -count=1`
- `go test ./cmd -run 'TestOpsAnalyseSlowProcessInstances|Test.*SlowProcess' -count=1`
- `go test ./cmd -run 'Test(FormatOps|OpsProgress|OpsETA|OpsWorkflowReport|ValidateOpsWorkflowReport|ResolveOpsRepairReport|OpsExecute.*Report|WriteOpsWorkflowReport|FormatOpsPurgeReportTime|WriteMarkdownReport|RenderOps.*Report)' -count=1`
- `go test ./cmd -run 'Test.*(ProcessInstance|UpdateJob|Cancel|Delete|Root|SlowProcess|Ops.*Progress|Ops.*Report|RenderOps)' -count=1`
- `go test ./cmd -run 'Test(GetViewFilesAvoidBackendOwnership|.*DryRun|.*ProcessInstance.*Plan|.*ProcessInstance.*Selector|.*Cancel.*ProcessInstance|.*Delete.*ProcessInstance|ResolveProcessInstance)' -count=1`
- `go test ./cmd -count=1`
- `git diff --check`

## Do Not Repeat

- Do not redo the setup ownership audit from scratch; use `specs/270-cmd-mode-reorg/ownership-followups.md` and only refresh notes for files a later task actually touches.

## Current Handoff
- Next iteration should continue US3 with T048 by splitting slow-process analysis command, validation, and progress declarations from `cmd/ops_analyse_slow_process_instances.go` into `cmd/ops_analyse_slow_process_instances_validation.go` and `cmd/ops_analyse_slow_process_instances_progress.go`.
