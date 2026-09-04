# Research: Force-Cleanup Progress

## Scope and evidence

Research used the repository implementation, tests, #291 specification, #285 specification, project constitution, and `specs/ralph-implementation-rules.md`. Two read-only research agents independently traced service events and command reporting. No external dependency or backend API change is needed.

## 1. Report stages at their actual service boundaries

**Decision:** Add a typed, additive stage-entry progress event. Emit cancellation, draining, history deletion, and definition deletion entries immediately before those operations begin in the existing version-neutral process-definition workflow.

**Evidence:** `internal/services/ops/all_process_definitions_purge.go:PurgeAllProcessDefinitions` passes the request callback through `services.WithProgress` into `internal/services/processdefinition/delete.go:DeleteProcessDefinitions`. The force path calls preview, `cleanupProcessDefinitionDeletePlanForceScope`, then `DeleteProcessDefinitionResources`. Cleanup already sequences `CancelProcessInstances`, `waitForProcessDefinitionDeletePlanActiveInstancesDrained`, and `DeleteProcessInstances`. `internal/services/processinstance/bulk.go` already emits completions with phases `cancel` and `delete`; definition completions use `delete process definitions`. Draining has no entry callback, and definition deletion has no entry callback before its first completion.

**Rationale:** Service boundaries prove that a stage actually started, including before its first result. Reporting adds no remote request or new orchestration.

**Ordinary-path compatibility:** Non-force APD uses the ordinary `DeleteProcessDefinitions` worker path through `deleteProcessDefinition`, bypassing `DeleteProcessDefinitionResources`. Supply a private, per-run synchronized once-only entry hook at the first validated item's boundary immediately before `DeleteProcessDefinitionResourceAndWait`. Its total is the existing unique bulk key count. This preserves validation and scheduling and avoids repeated entries from concurrent items. If no item reaches resource deletion, no definition stage is invented. Add a non-force callback regression.

**Alternatives considered:** Routing non-force work through `DeleteProcessDefinitionResources` changes validation and probe behavior and is rejected. Inferring stages from completion events misses initial work and draining. Reusing frozen-scope counters would give a waiting stage a false work denominator; it also conflates PI root counts with its current resource label. Parsing logs would couple behavior to diagnostic wording and suppression.

## 2. Keep one command-owned workflow reporter

**Decision:** Introduce `cmd/ops_purge_all_processdefinitions_progress.go` with one coordinator owning current stage, one workflow activity, stage-local aggregates, one pacing clock, durable activation, and pending stage snapshots. Keep the base command limited to construction, confirmation, dispatch, and final output.

**Evidence:** The base command eagerly invokes `deletionProgress.Start` after confirmation. `configureOpsPurgeAllProcessDefinitionsProgress` forwards all completions to `processDefinitionDeleteSemanticProgress`, whose fixed scope only accepts definition completions. `cmd/ops_semantic_progress.go` gives each reporter its own activity and clock. Reporter maps in smoke-test and analysis adapters therefore cannot directly satisfy #291's workflow-wide pacing.

**Rationale:** Cancellation, histories, and definitions have different units. Switching activity labels must not reset counters, restart the pacing clock, or flush a finished stage prematurely.

**Alternatives considered:** Broadening the existing reporter phase filter would mix root and definition counts. Constructing independent reporters per stage duplicates activity ownership and pacing. A general workflow engine is unnecessary for this bounded rendering lifecycle.

## 3. Reuse existing progress mechanics narrowly

**Decision:** Reuse the existing output policy, renderers, durable-line printers, activity helpers, and 10-second constant. Extract the completion-to-aggregate update from `opsSemanticProgressReporter.ingestCompletionLocked` into a small pure helper shared by the existing reporter and coordinator, preserving existing single-scope behavior. Keep APD stage routing and cross-stage pending state local. Do not move waits, worker scheduling, or planning into the command or facade.

**Rationale:** Shared reduction preserves failed counts, nil affected-count semantics, and total bounds without copying them. The coordinator supplies synchronization around reductions and output. Its exact phase matching rejects unrelated or empty phases instead of adopting the generic reporter's permissive empty-phase match.

**Alternatives considered:** Copying the entire reporter creates parallel implementations of failure and count rules. Retrofitting every command with a generic multi-stage framework increases regression risk. A second shared pacing abstraction is unnecessary unless implementation demonstrates useful reuse beyond the existing fixed interval and policy.

## 4. Preserve pending stage evidence without extra transition lines

**Decision:** Start the pacing clock at the first actual mutation stage, after confirmation and planning. On matching completions, emit the current stage when at least 10 seconds have elapsed since start or the last informational milestone. Warnings activate durable progress but do not reset the informational clock. Stage changes and drain polling neither emit milestones nor reset pacing. Keep dirty snapshots for earlier stages until represented durably.

At close, if default durable progress has activated, emit at most one final diagnostic record containing every still-dirty mutation-stage aggregate in execution order, separated by semicolons and explicitly labeled as stage progress. Do not repaint activity or announce a new stage while flushing. If only one stage is dirty, retain the ordinary aggregate form. Omit waiting, unentered, and already-reported stages. Close is idempotent. Clean sub-10-second runs have no durable record.

**Rationale:** This applies the specification's workflow-wide cadence and final-flush exception without losing the last unreported cancellation/history counts. The final record is historical progress, not a claim that all work succeeded.

**Alternatives considered:** Flushing on every transition breaks the cadence. Dropping earlier snapshots loses progress. Closing independent reporters emits multiple unpaced records and can restore stale activity labels.

## 5. Distinguish planned and completed affected counts

**Decision:** Stage entry may carry a trustworthy planned affected-instance count; display it with a scope label such as `affected scope: N process instance(s)`. Keep this separate from any cumulative affected count computed from completion facts. Preserve unknown values as nil and omit unsupported cumulative totals.

**Evidence:** `processDefinitionDeleteCleanupScopeForPlan` deduplicates roots and affected keys. `processInstanceMutationAffectedCount` deliberately returns nil for multi-root expanded work without trustworthy per-root coverage. Cleanup's execution fallback to root count is not proof of full affected coverage.

**Rationale:** Operators need scope during cancellation before any completion, without mistaking that scope for work already completed.

**Alternatives considered:** Showing zero for unknown scope or dividing an aggregate among roots is misleading. Extra discovery to obtain counts changes backend requests and is outside scope.

## 6. Preserve all callback mappings and supported modes

**Decision:** Add the stage payload to `internal/domain/ops_progress.go`, `c8volt/ops/progress_model.go`, and `c8volt/foptions/options.go`; map it mechanically in `c8volt/ops/convert.go` and the foptions converter. Copy optional count pointers. Leave command results, envelopes, reports, and existing completion fields unchanged.

**Evidence:** Ops and foptions expose separate public callback models. APD advertises one-line and JSON output, full automation, and full-history deletion for supported 8.9/8.10 runtimes. It does not advertise keys-only support.

**Rationale:** An additive callback fact must survive both public paths without leaking internal types. Keys-only protection belongs in shared policy regressions and existing unsupported-mode validation, not a new APD flag.

**Alternatives considered:** Updating only ops drops stage facts for foptions consumers. Encoding stage data into command result JSON would change a contract unrelated to transient progress.

## 7. Validate the real nested sequence

**Decision:** Combine deterministic coordinator tests with command execution against a fake backend that runs the actual facade and nested services. Use barriers to inspect activity before first cancellation completion and during draining; use an injected clock for pacing. Retain service request/worker assertions.

**Evidence:** `cmd/ops_purge_all_processdefinitions_test.go` currently tests isolated definition completion callbacks. `internal/services/ops/all_process_definitions_purge_test.go:TestPurgeAllProcessDefinitionsForceCleanupDeduplicatesProcessInstanceRoots` executes real shared-root cleanup but only asserts definition completions. `internal/services/processdefinition/delete_test.go` protects requested workers, the first serial delete-history probe, and deletion verification.

**Alternatives considered:** Fabricated command callbacks alone could pass while facade conversion drops stage facts. Real cluster timing alone is nondeterministic and cannot precisely prove cadence boundaries.

## Resolution

All planning unknowns are resolved. No specification changes or constitution exceptions are required. Implementation must preserve preview rechecks, the serial first-definition request probe, polling, existing suppression, fail-fast scheduling, and operational confirmation semantics.
