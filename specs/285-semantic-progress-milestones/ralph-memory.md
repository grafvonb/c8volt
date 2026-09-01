# Ralph Memory

Feature: 285-semantic-progress-milestones
Started: 2026-08-31T17:14:26Z

## Codebase Patterns
- Active feature artifacts live under `specs/285-semantic-progress-milestones/`; Ralph iterations for this feature must include `--implementation-context specs/ralph-implementation-rules.md`.
- Branch is `285-semantic-progress-milestones`; issue-backed commit subjects for this feature use Conventional Commits with scope `ralph` and end with `#285`.

## Decisions
- Phase 1 setup was completed as a tracking-only work unit before any source implementation.
- The canonical completion fact is `OpsCompletionProgress` / `CompletionProgress` with `submitted`, `confirmed`, and `failed` dispositions; `AffectedCount *int` distinguishes unavailable from trustworthy zero.
- Completion facts are mirrored mechanically through `c8volt/foptions` and `c8volt/ops`; command wording remains out of domain, services, and facades.
- The first command reporter scaffold is isolated in `cmd/ops_semantic_progress.go`; it owns one workflow-priority activity, mutex-protected completion aggregation, affected-count invalidation, family vocabulary, and idempotent close.
- Completion output policy stays command-local in `cmd/ops_progress_mode.go`: default human allows transient and future paced aggregate progress, verbose/debug allow per-item durable lines, quiet allows failure warnings only, and JSON/keys-only/automation stay silent.
- Process-instance bulk services now emit one completion fact per executed create/cancel/delete worker. Phases are `create`, `cancel`, and `delete`; cancel/delete facts count `process-instance tree(s)` and use `affected process instances` only when a per-worker delta is trustworthy.
- Process-instance bulk legacy ticker progress is disabled when a structured progress callback is installed, but existing frozen-scope callback events and final service summaries are unchanged.
- Process-instance direct-key, stdin-key-equivalent, and search-selected cancel/delete command paths now start a post-confirmation semantic reporter and pass a completion-only callback to the mutation facade call; planning preflight/page/frozen-scope progress remains on the existing planning callback.
- Process-instance command affected-count rendering is initially enabled only for scopes that can be proven from the frozen command impact (`one root` or `affected == roots`); the reporter still permanently invalidates affected output if any completion arrives without a trustworthy delta.
- Process-definition service deletion completion facts use phase `delete process definitions`, core resource `process definition(s)`, identity = process-definition key, and submitted/confirmed/failed disposition based on `--no-wait`, response OK, and errors; affected counts are intentionally unavailable for this scope.
- All-process-definition purge now reattaches request-owned progress to the destructive delete options so APD request progress receives process-definition deletion facts through the reused delete service path.
- Resource deployment completion facts use phase `deploy process definitions`, core resource `process definition(s)`, identity = returned process-definition key, and submitted/confirmed dispositions for no-wait acceptance versus visibility confirmation; affected counts remain unavailable.
- v8.8, v8.9, and v8.10 deployment services report no-wait completions immediately after a valid deployment response and confirmed completions from the first successful process-definition visibility lookup without extra backend requests.
- Retention, orphan, and incident-selected purge tests now prove their destructive execution paths forward process-instance `delete` completion facts from the shared bulk delete service after frozen planning; no new service implementation was needed for T012.
- Repair service workers now emit `repairing incidents` completion facts from `executeIncidentRepair` at worker return and frozen repair counters increment from the worker callback, not from the post-pool result assembly.
- Smoke-test service emits high-level completion facts for `deploying smoke-test fixture`, `starting process instances`, `walking process-instance families`, `cleaning up smoke-test process instances`, and `cleaning up smoke-test process definition`; start and cleanup phases mirror nested service completion facts into smoke-test stage phases.
- Secondary workflow assessment tests now pin bulk-start inclusion, slow-analysis discovery as transient-only, multi-key expect JSON silence, and single-key waiter polling as wait-priority rather than semantic workflow progress.
- `run process-instance --count` uses a run-specific semantic completion reporter for process-instance creation facts; the shared explicit-large-work adapter remains frozen-scope-only for walk/search-style callers.
- Reporter scope isolation is now pinned directly: completion events for a different phase do not advance aggregate counters or repaint the workflow activity.
- Basic `delete process-definition` now starts a command-owned semantic deletion reporter after confirmation and passes facade completion progress into `DeleteProcessDefinitions` without changing impact planning or final delete summaries.
- APD command progress keeps preflight/page discovery on the existing discovery renderer and routes only `delete process definitions` completion facts into a separate deletion reporter; prompted APD deletion eagerly starts after frozen confirmation, while auto-confirmed APD starts lazily on the first deletion completion.
- `deploy process-definition` and `embed deploy` now install a command-owned deployment reporter plus `WithSuppressWorkflowDetailLogs`; the reporter is lazy because deployment totals arrive in service completion facts after the upload response.
- Retention, orphan, and incident-selected purge commands now keep discovery/planning progress on the existing ops renderer and route only `delete` completion facts into a lazy process-instance deletion semantic reporter; final reports/results render after the reporter closes.
- Slow-process analysis now emits completion facts for enrichment phases `loading runtime elements` and `loading listener jobs`; search discovery/preflight remain separate and explicit-key analysis receives semantic enrichment progress without confirmation prompts.
- Multi-key `expect process-instance` now emits `expect process instances` completion facts from the waiter bulk wrappers and routes them through `cmd/expect_processinstance_progress.go`; single-key waiter polling remains wait-priority activity unless a caller explicitly uses the bulk wrapper with a progress callback.
- User Story 1 validation completed in iteration 18 with reporter, activity, process-instance, process-definition, resource, ops, run, analysis, and expect tests passing under `-race`; T024 was a validation/audit-only work unit.
- Semantic reporter durable output now uses the shared 10-second milestone cadence, activates durable progress on the first paced aggregate or immediate failure warning, tracks dirty aggregate state, and flushes exactly once on close only when activated progress has unreported completions.
- Quiet-mode semantic failure warnings now bypass logger severity filtering through `printOpsDurableLineDirect`; normal durable info/warn paths continue using the attached logger when present.

## Gotchas
- `progress.md` and `ralph-memory.md` started untracked in this worktree; include them with the coordinated task commit.
- `c8volt/foptions` previously had no test file; `c8volt/foptions/options_test.go` now covers service-to-facade progress callback mapping.
- `cmd/ops_analyse_slow_process_instances_progress_test.go` participates in broader `Progress|Activity` runs; apply output-mode globals after `resetOpsSlowProcessAnalysisTestFlags(t)` because that helper now clears shared mode flags for isolation.
- `cmd/ops_semantic_progress_test.go` now asserts all 64 concurrent reporter updates produce the exact completed-count sequence under `-race`; use `requireOpsSemanticProgressCompletedSequence` for similar aggregate-sequence checks.
- `toolx/logging/activity_test.go` has a workflow-priority regression proving lower-priority HTTP/wait updates cannot replace the visible workflow aggregate.
- `internal/services/processinstance/bulk_test.go` filters callback events by kind because structured callbacks now receive both frozen-scope and completion events.
- Delete affected counts are nil for expanded multi-root scopes when only an aggregate affected count is available; cancel can use the returned affected process-instance slice length as a trustworthy per-root delta.
- `cmd/processinstance_mutation_progress_test.go` has shared helpers for simulating process-instance completion facts and asserting workflow-priority semantic activity across cancel/delete direct and stdin-key-equivalent paths.
- In `cmd/cancel_processinstance_selector.go`, close the per-page semantic reporter immediately after each page mutation call; using `defer` inside the visitor would keep prior page activities alive until traversal completes.
- Process-definition deletion uses a serial first delete before the worker pool to catch Camunda delete-history request-shape errors; emit a failed completion for that first attempted key and no facts for the unscheduled remainder.
- APD force cleanup progress includes nested process-instance cancel/delete completion facts plus process-definition deletion facts; filter by phase `delete process definitions` in APD service tests that only care about the process-definition completion scope.
- v8.7 deployment responses do not expose process-definition keys, so resource progress tests intentionally require no completion facts rather than inventing identities from submitted filenames.
- `resourcepayload.ReportDeploymentProcessDefinitionCompletion` carries the full deployment scope total for each visible key; do not call the slice wrapper with a singleton key when the original deployment returned multiple process definitions.
- `internal/services/ops` test files can reuse `opsCompletionProgressByPhase` from `all_process_definitions_purge_test.go`; `opsIntPtr` is a package test helper for expected completion affected counts.
- Smoke-test stage completion tests should filter by the smoke-test phase name because nested process-instance/process-definition services still forward their lower-level completion facts to request-owned progress.
- Bulk-start services emit legacy frozen-scope updates after each completion; the run-specific command adapter keeps those frozen counters durable-only in verbose/debug so they cannot overwrite semantic workflow activity with affected counts.
- Expect command JSON for successful state reports omits default-valued `key`, `ok`, and `total`; tests should assert the existing envelope contract without requiring those omitted fields.
- Process-definition deletion completion reporters intentionally omit affected counts because the service facts do not provide trustworthy per-definition affected deltas.
- `cmd/delete_processdefinition_progress.go` owns process-definition deletion reporter vocabulary and both facade-level and ops-level completion callback adapters; keep discovery page formatting in APD command progress, not in that helper.
- Deployment command progress must ignore non-`deploy process definitions` completion phases before constructing the lazy reporter; otherwise v8.7 `--run` follow-up process-instance creation could open a misleading deployment activity.
- `cmd/ops_processinstance_purge_progress.go` intentionally suppresses the shared bulk-delete frozen-scope `deleting process instances` line because retention/orphan/incident purge deletion progress is now completion-driven; discovery and planning frozen scopes still render through the existing verbose path.
- Repair commands now use `cmd/ops_repair_progress.go` to route `repairing incidents` completion facts into a lazy semantic reporter; discovery, planning, and legacy frozen counters remain on the existing progress renderer.
- Smoke-test commands now use `cmd/ops_execute_smoketest_progress.go` for high-level stage completion reporters covering deploy/start/walk/cleanup; nested phases such as `create`, `delete`, `delete process definitions`, and `deploy process definitions` are intentionally ignored by the smoke-test command reporter.
- `configureOpsSlowProcessAnalysisPreflight` now returns a nullable closeable progress wrapper; command code can safely `defer progress.Close()` because the receiver handles nil.
- Slow-analysis frozen-scope tests that count raw callback events must account for both frozen-scope and completion facts, or filter by event kind.
- Expect command progress is intentionally installed only for multi-key scopes (`len(keys) > 1`) to avoid broadening single-target wait UX during the US1 slice.
- T025 added fake-clock reporter tests in `cmd/ops_semantic_progress_test.go`; they use direct stderr capture without a logger, so failure tests assert warning content rather than severity prefix. T026 now covers quiet warning visibility when logger severity would otherwise filter warn records.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./internal/domain ./c8volt/foptions ./c8volt/ops ./toolx/logging -race -count=1`
- `go test ./internal/domain ./c8volt/foptions ./c8volt/ops ./internal/services -race -count=1`
- `go test ./cmd -run 'Progress|Activity' -race -count=1`
- `go test ./cmd -run 'TestOpsSemanticProgress' -race -count=1`
- `go test ./toolx/logging -race -count=1`
- `go test ./internal/services/processinstance/... -run 'Progress|Cancel|Delete|CreateNProcessInstances' -race -count=1`
- `go test ./internal/services/processinstance -race -count=1`
- `go test ./cmd -run 'TestProcessInstanceMutationDirectAndStdinKeysUseSemanticCompletionActivity|TestCancelProcessInstanceSearchSelectedUsesSemanticCompletionActivity|TestDeleteProcessInstanceSearchSelectedUsesSemanticCompletionActivity' -race -count=1`
- `go test ./cmd -run 'ProcessInstance.*(Progress|SearchSelected|SearchProgress|WorkflowImportance|DryRun_Search|WithPlan)|CancelProcessInstanceSearch|DeleteProcessInstanceSearch' -race -count=1`
- `go test ./cmd -run 'ProcessInstance' -race -count=1`
- `go test ./internal/services/processdefinition ./internal/services/ops -run 'DeleteProcessDefinitionResources.*Completion|DeleteProcessDefinitionResourcesStopsOnDeleteHistoryRequestShapeError|PurgeAllProcessDefinitionsForceCleanupDeduplicatesProcessInstanceRoots' -race -count=1`
- `go test ./internal/services/processdefinition ./internal/services/ops -run 'Progress|Delete|PurgeAllProcessDefinitions' -race -count=1`
- `go test ./internal/services/processdefinition/... ./internal/services/ops/... -race -count=1`
- `go test ./internal/services/resource/payload ./internal/services/resource/v87 ./internal/services/resource/v88 ./internal/services/resource/v89 ./internal/services/resource/v810 -run 'Deploy|Visibility|Completion' -race -count=1`
- `go test ./internal/services/resource/... -race -count=1`
- `go test ./internal/services/ops -run 'TestRepairIncidentsEmitsWorkerCompletionFactsAtReturnPoints|TestRepairIncidentsCompletionFactsCaptureFailureDetail|TestExecuteSmokeTestEmitsStageCompletionFacts' -race -count=1`
- `go test ./internal/services/ops/... -run 'Progress|Purge|Repair|Smoke' -race -count=1`
- `go test ./internal/services/ops -race -count=1`
- `go test ./cmd ./internal/services/processinstance/waiter -run 'TestRunProcessInstanceBulkStartCompletionUsesSemanticWorkflowActivity|TestExplicitLargeWorkSharedAdapterIgnoresCompletionFacts|TestOpsAnalyseSlowProcessInstancesSearchDiscoveryStaysTransientOnly|TestExpectProcessInstanceCommand_MultiKeyStateJSONRemainsProgressFree|TestWaitForProcessInstanceState_SingleTargetPollingIsNotWorkflowProgress' -race -count=1`
- `go test ./cmd -run 'RunProcessInstance|OpsAnalyseSlowProcessInstances|ExpectProcessInstance|ExplicitLargeWork' -race -count=1`
- `go test ./internal/services/processinstance/waiter -race -count=1`
- `go test ./cmd -run 'Progress|Activity' -race -count=1`
- `go test ./cmd -run 'TestOpsSemanticProgressReporter|TestOpsDurableMilestoneCadenceIsTenSeconds|TestOpsProgressDurableMilestone' -race -count=1`
- `go test ./cmd -run 'ProcessInstance|ProcessDefinition|Purge|Retention|Orphan|Incident|Repair|Smoke|Deploy|Run|Analyse|Expect|Progress|Activity' -race -count=1`
- `go test ./cmd -run 'TestProcessDefinitionDeploySemanticProgress|TestAppendProcessDefinitionDeployProgressOptions|TestProcessDefinitionDeleteSemanticProgressRoutesFacadeCompletion|TestOpsPurgeAllProcessDefinitionsProgressKeepsDiscoverySeparate' -race -count=1`
- `go test ./cmd -run 'Deploy|ProcessDefinitionDeploy|Progress|Activity' -race -count=1`
- `go test ./cmd -run 'TestOpsRepairSemanticProgressRoutesCompletion|TestOpsExecuteSmokeTestSemanticProgressRoutesStageCompletions|TestExplicitLargeWorkProgressEventRespectsOutputModes|TestOpsRepairIncidentProgressContractPendingT068|TestOpsRepairProcessInstanceProgressContractPendingT068|TestOpsExecuteSmokeTestVerboseProgressRendersStageCounters' -race -count=1`
- `go test ./cmd -run 'Repair|Smoke|Progress|Activity' -race -count=1`
- `go test ./cmd ./internal/services/processinstance/waiter ./internal/services/ops ./internal/services/processinstance -run 'RunProcessInstance|OpsAnalyseSlowProcessInstances|ExpectProcessInstance|WaitForProcessInstances|SlowProcessAnalysis|CreateNProcessInstances' -race -count=1`
- `go test ./cmd ./internal/services/processinstance/... ./internal/services/ops -race -count=1`
- `git diff --check`

## Do Not Repeat
- Do not reintroduce semantic progress wording into services or facade converters; completion facts remain wording-free and command renderers choose verbs.

## Current Handoff
- Next iteration should continue User Story 2 at T027: add concurrent durable-write clear/redraw and nested workflow-priority arbitration tests in `toolx/logging/activity_test.go` and `testx/activitysink/activity_sink_test.go`.
