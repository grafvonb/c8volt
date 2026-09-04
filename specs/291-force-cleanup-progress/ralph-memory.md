# Ralph Memory

Feature: 291-force-cleanup-progress
Started: 2026-09-04T10:01:49Z

## Codebase Patterns
- APD command flow: `cmd/ops_purge_all_processdefinitions.go` builds `ops.AllProcessDefinitionsPurgeRequest`, installs a progress callback through `configureOpsPurgeAllProcessDefinitionsProgress`, runs a dry-run planning call for interactive confirmation, freezes discovered keys, then executes through `cli.PurgeAllProcessDefinitions`.
- Current APD progress callback handles preflight/page facts locally and forwards only completion facts into `processDefinitionDeleteSemanticProgress`; that reporter is fixed to `delete process definitions` and starts eagerly after confirmation when frozen keys exist.
- Service callback chain for real deletion is command `request.Progress` -> public facade conversion (`c8volt/ops/convert.go` and `c8volt/foptions/options.go`) -> `internal/services/ops.PurgeAllProcessDefinitions` -> `pdsvc.DeleteProcessDefinitions` -> PI cancel/delete completion callbacks and PD resource delete completion callbacks.
- Existing force cleanup path in `internal/services/processdefinition/delete.go`: preview with workers, `cleanupProcessDefinitionDeletePlanForceScope`, `pisvc.CancelProcessInstances`, `waitForProcessDefinitionDeletePlanActiveInstancesDrained`, `pisvc.DeleteProcessInstances`, then `DeleteProcessDefinitionResources`.
- Foundation now exposes `OpsProgressEventKindStage` / `OpsStageProgress` in `internal/domain`, mirrored as public `StageProgress` in both `c8volt/ops` and `c8volt/foptions`; optional `Total` and `PlannedAffectedCount` use `*int`, where nil means unavailable and known zero is preserved.
- `c8volt/ops` converts stage events both directions and copies optional count pointers with `toolx.CopyPtr`; `c8volt/foptions` maps service stage events one-way into public callbacks with explicit pointer copies.
- `cmd/ops_semantic_progress.go` now has pure reducer `applyOpsSemanticCompletionToAggregate`, preserving the old reporter rules for total adoption/bounds, failure counts, dirty acceptance, and affected coverage invalidation.
- `internal/services/processdefinition/delete.go` now emits typed stage entries at service-owned boundaries: cancel with unique-root total and trustworthy planned affected scope, drain without totals, PI history delete with unique-root total, and definition delete before the first preplanned resource request.
- Ordinary non-force `DeleteProcessDefinitions` still uses the existing worker path and now passes a private `sync.Once` hook into `deleteProcessDefinition`, reporting one definition stage after item validation and immediately before the first resource delete.
- `internal/services/ops/all_process_definitions_purge_test.go` now asserts the APD service mutation sequence by filtering stage/completion events away from discovery and nested FrozenScope facts; force cleanup expects cancel -> drain -> PI history delete -> PD delete, with unique-root totals and planned affected scope.
- `cmd/ops_purge_all_processdefinitions_progress.go` now owns a dormant APD stage coordinator for real execution: `Start` opens one generic workflow activity, stage entries update the current activity, cancellation/history/definition aggregates remain stage-local, draining has no denominator, unentered/unknown/empty completion phases are ignored, and concurrent callbacks are serialized through the T007 reducer.
- APD coordinator rendering keeps planned affected scope separate as `affected scope: N process instance(s)` and only shows completed affected totals after trusted completion counts arrive; known zero is preserved while nil is omitted.
- Real APD execution now uses `opsPurgeAllProcessDefinitionsProgress` as the sole workflow activity owner. Preview/dry-run still uses the legacy outer activity wrapper; real execution starts the coordinator, routes stage/completion callbacks through it, and closes it before final rendering.
- APD coordinator durable pacing now anchors at the first recognized stage entry, not generic real-execution start, and stage transitions/drain time do not reset or flush the clock. Default close emits one historical line for dirty mutation-stage aggregates in execution order, using ordinary aggregate wording for a single dirty stage and `stage progress: ...; ...` for multiple dirty stages.
- Verbose/debug APD coordinator output remains item-outcome based and does not emit paced aggregate lines or a close-time aggregate flush.
- Real command acceptance coverage can install services manually with an activity sink in context, then run the Cobra command path through the actual facade and services. Root bootstrap replaces the terminal activity writer, so assertions must observe the sink-backed command context used for the installed service factory.
- Fake nested APD command servers must distinguish process-definition stat active searches (`sort` by `processInstanceKey`) from active-instance list searches (`sort` by `processDefinitionName`/`version`); confusing them prevents the drain loop from reaching zero.
- APD stage completions are ignored until an explicit stage entry has been observed. Tests that assert definition milestones must enter the definition stage before sending completion callbacks.
- First-root cancellation activity can be asserted through activity sink history; drain synchronization still needs a blocking active-stat response so the wait-stage activity can be observed before release.
- Nested APD command tests can reuse `newOpsPurgeAllProcessDefinitionsNestedServer` for output assertions by closing `state.drainRelease` before subprocess execution; failure switches on the shared state now produce deterministic cancel/history/definition HTTP 500 responses for default warning coverage.
- In verbose/debug command output, APD human results and semantic progress both route to stderr; stdout stays empty for one-line diagnostic-mode subprocess assertions. `--no-wait` keeps nested cancel/history/definition item outcomes as `submitted`; waited execution renders cancel as `cancelled` and history/definition deletion as `deleted`.
- US3 command compatibility coverage now exercises the real nested force-cleanup server in JSON, JSON+verbose, and automation+JSON+verbose modes; nested stage text must stay out of stdout/stderr while the final JSON deletion envelope remains unchanged.
- Quiet nested force-cleanup failure coverage expects the immediate failed-item warning to remain visible, while no later unentered drain/history/definition stage progress or final `stage progress:` line appears.
- `TestOpsPurgeAllProcessDefinitionsCommandHelper` supports `C8VOLT_TEST_ALL_PD_PURGE_DECLINE=1` to write the captured prompt and then abort confirmation before mutation.
- Empty APD command coverage uses `newOpsPurgeAllProcessDefinitionsEmptyServer`, which returns zero definitions and fails on mutation requests; human dry-run output may route through stderr in subprocess logger context.
- Process-definition service US3 failure coverage now asserts cleanup stops at the last entered stage for cancellation failure, drain backend error, drain deadline/cancellation interruption, and history failure; nested PI calls preserve caller `NoWait`/`FailFast` plus APD-added affected count and suppression options.
- `DeleteProcessDefinitionResources` US3 coverage now asserts the definition stage entry precedes the first serial request, Camunda `deleteHistory` request-shape rejection submits only the first definition, and fail-fast after the serial probe emits no completion for unscheduled definitions while still returning the pool-sized result slice with zero-value unscheduled entries.
- APD facade boundary coverage in `c8volt/ops/client_test.go` now proves request-level stage progress forwards through `PurgeAllProcessDefinitions`, nil request/options callbacks stay nil, frozen keys/options/tenant evidence/report mapping remain intact, and domain validation errors normalize through `ferrors`.
- US3 compatibility verification passed without production repair: command/facade/service focused suites and race-enabled command/service matrix all accept the current progress coordinator, APD wiring, and ops conversion boundaries.
- Authored APD documentation now describes the four forced cleanup stages, root-tree versus definition counts, planned affected scope, completion-driven/default pacing, verbose item outcomes, quiet/machine suppression, and discovery-only `--batch-size`. Generated CLI docs remain untouched until T029.
- APD command help now matches implemented force-cleanup progress: it names cancellation, draining, history deletion, and definition deletion stages; distinguishes root-tree counts from process-definition counts; keeps `--batch-size` discovery-only; and pins the wording in `TestOpsPurgeAllProcessDefinitionsHelpDocumentsCommandShape`.
- `make docs-content` regenerated the APD CLI page and docs index from the command metadata/README; generated docs now carry the forced-cleanup stage sequence, root-tree versus definition counts, planned affected-scope distinction, and updated build stamp.

## Decisions
- Iteration 1 was setup/evidence only. No production code or generated docs changed.
- Iteration 2 completed Phase 2 foundational event, mapping, and reducer work. No service emission, APD coordinator, user-facing command output, docs, or generated docs changed yet.
- Iteration 3 completed the service stage-entry slice for US1. The APD command still ignores stage entries until the coordinator/wiring tasks are implemented.
- Iteration 6 completed US1 stage visibility end to end: real nested command callbacks now show cancellation, drain, history deletion, and definition deletion through the APD coordinator.
- Iteration 7 completed the coordinator-owned US2 timing/final-record slice. Nested command output tests and the full US2 validation record remain open.
- Iteration 8 completed US2 command-level output evidence and validation: default cancel/history/definition failures warn once with owning-stage aggregates; verbose/debug output emits one item outcome per nested stage completion for submitted and confirmed paths with no duplicate aggregate lines.
- Iteration 12 completed US3 verification and compatibility evidence. No progress-specific regression was exposed, so no code repair was needed in the coordinator, APD command wiring, or ops conversion boundary.
- Iteration 13 completed T027 authored documentation only. Command metadata/help, generated CLI docs, cohesion review, focused/full validation, and terminal handoff remain open.
- Iteration 14 completed T028 command metadata/help only. Generated CLI docs remain untouched until T029.
- Iteration 15 completed T029 generated CLI documentation refresh. Cohesion review, focused/race validation, full `make test`, `git diff --check`, and terminal handoff remain open.
- Actual checkout is `develop`; `.specify/feature.json` still points at `specs/291-force-cleanup-progress`; `AGENTS.md` still points at `specs/291-force-cleanup-progress/plan.md`.

## Gotchas
- `ralph-memory.md` and `progress.md` were already untracked at iteration start; they are now part of the coordinated setup commit.
- Commit policy supplied by the orchestrator is conventional with scope `ralph` and `issue: auto`; branch `develop` has no leading numeric prefix.
- The feature tasks mention appending `#291`, but the resolved orchestrator policy says infer issue suffix only from a leading numeric branch prefix. Use the resolved policy unless a later orchestrator defect says otherwise.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./cmd -run 'TestOpsPurgeAllProcessDefinitions|TestOpsSemanticProgressReporter|TestNewOpsSemanticProgressReporter' -count=1`
- `go test ./internal/domain -run 'Test.*Progress' -count=1`
- `go test ./c8volt/ops -run 'TestProgressConversions|TestClientPurgeAllProcessDefinitions' -count=1`
- `go test ./c8volt/foptions -run 'Test.*Progress' -count=1`
- `go test ./internal/services/processdefinition/... -run 'Test.*(DeleteProcessDefinition|CleanupProcessDefinition)' -count=1`
- `go test ./internal/services/ops/... -run 'TestPurgeAllProcessDefinitions' -count=1`
- `go test ./internal/services/processinstance/... -run 'Test.*(Progress|Completion|CancelProcessInstances|DeleteProcessInstances)' -count=1`

## Do Not Repeat
- Do not treat `FrozenScope` as a stage entry or completion source for #291; stage visibility requires a new typed stage event.
- Do not reroute the ordinary non-force worker path through `DeleteProcessDefinitionResources`; the plan requires a private once-only entry hook in the existing `deleteProcessDefinition` path.

## Current Handoff
- Continue Phase 6 polish with T030: review touched declarations in `cmd/ops_purge_all_processdefinitions.go`, `cmd/ops_purge_all_processdefinitions_progress.go`, `cmd/ops_semantic_progress.go`, and service/facade counterparts for ownership, comments, bounded state, and dependencies; run targeted `gofmt` if edits are needed.
