# Ralph Memory

Feature: 347-resume-process-instances
Started: 2026-10-27T09:01:23Z

## Codebase Patterns

- Versioned `ResumeProcessInstance` now owns exact-key preview, submission, and post-acceptance confirmation. V810 uses the existing generated resumption endpoint with `{}`; v88/v89 reject before config, key access, or I/O.
- `common.NormalizeProcessInstanceKeys` trims exactly 16 ASCII decimal digits and deduplicates first-input order. The existing suspension-named entry point delegates to it, preserving its callers and behavior.
- `waiter.WaitForProcessInstanceResumption` exposes only `GetProcessInstance`. Caller acceptance is a documented precondition. ACTIVE/COMPLETED confirms; CANCELED/TERMINATED stops with conflict; absence and unknown remain bounded observations.
- `processinstance.ResumeProcessInstances` owns capability-first all-key validation, local adapter previews before dispatch, worker selection, stable slots, and partial errors. `process.ResumeProcessInstance` wraps/extracts through the public collection method; it never bypasses collection validation.
- Public resumption conversion copies items, preserves every enum/evidence field, and initializes even nil domain items as `[]`. Accepted-but-unconfirmed failures retain both acceptance and failure facts.
- Adapter fixtures use generated response mocks or in-memory RoundTrippers; `testing/synctest` exercises deadlines, retries, and request-rate pacing without wall-clock delays or real services.

- `runResumeProcessInstances` freezes copied preview keys before one approval, then calls the collection inside an activity scope that stops before rendering. An empty preview for nonempty explicit input is a local-precondition failure, never a successful no-op.
- Resumption views use `renderHumanLine` for INFO, bounded grouped reasons for ERROR (including `cancellation observed`), and stdout only for JSON/accepted keys. Render one partial envelope before `exitAfterRenderedResult`; retain the operation error if the result writer also fails.
- `resumptionFixture` rejects discovery/preflight reads; `runResumptionTerminal` uses real PTY stdin with redirected stdout and configured/inherited stderr. Activity fixtures additionally check rejection/interruption/writer failure and copied scope during approval. Reuse these for T005 without weakening the exact-key contracts.

- `PlanProcessInstanceResumption` guards capability, all seeds and timing before discovery, then uses complete `DescendantsResult` snapshots with IgnoreTenant/detail suppression. It retains first-seen observations, skips seeds already covered by a complete earlier subtree, and preserves every explicit seed in RequestedKeys. Only SUSPENDED is eligible; unknown values normalize to UNKNOWN.
- `descendant_plan_validation.go` shares only structural validation with suspension, preserving existing suspension error wording/classes. Validate every returned member before deduplicating overlaps. Any error returns a zero plan; successful empty targets remain initialized arrays. Public mapping clones all three arrays without interpreting scope.

## Decisions

- Keep resumption types and lifecycle separate from suspension; reuse existing mutation retry/rate primitives without a generic mutation framework.
- Submission accepts only valid empty 204. Unknown evidence survives later rejection; last valid submission status survives transport/malformed/admission failures. Only valid acceptance resolves ambiguity.
- Every successful matching read replaces state/date evidence, including clearing an omitted date. Failed reads preserve prior evidence; typed not-found records ABSENT and clears the date. Late reads cannot confirm.
- `resumptionFailure` centralizes disposition classification while returning original error classes and machine evidence. New service paths emit no human diagnostics; configured transport DEBUG remains independent and redacted.
- T001/T002 delivered the lifecycle and collection/public APIs; T003 delivered exact-key CLI; iteration 4 completed T004 library tree planning. T005 alone remains for CLI tree delivery.

## Gotchas

- Adding public resumption methods required explicit unsupported methods on `stubProcessAPI` in `cmd/process_api_stub_test.go` and `c8volt/resource/client_test.go`; interface-embedding fixtures still compile. Downstream public API implementers also need the two new methods.
- `toolx.MapSlice` preserves nil; the resumption converter follows the existing suspension make-and-map pattern to guarantee initialized result arrays.
- Extending internal process-instance interfaces requires all three version contracts plus stubs in process/resource facades, process-definition deletion, orphan purge, dry-run tests, and the v810 generated-client mock. These compile together now.
- NoWait exempts confirmation timing validation even without config; ordinary dry-run still validates timing. Exact preview never reads or submits.
- Do not interpret no-wait acceptance, rejected submission followed by desired state, or absence of a suspension date as confirmed resumption.
- The repository-wide `go test ./... -run '^$'` was compilation-only, without integration tags; it is not behavioral or live evidence. Race-enabled behavioral coverage was limited to the affected packages listed below.

- Adding verbs changes the command inventory assertion in `cmd/command_contract_test.go` and the hard-coded count plus surface test in `docsgen/main.go`/`main_test.go` (now 62). Regenerate docs from command metadata; do not hand-edit generated pages.

- The public API now also requires `PlanProcessInstanceResumption`; command/resource legacy stubs explicitly return unsupported. No new internal interface methods were needed for planning. Shared fixture `planTraversal` lives in the suspension planner tests and serves both planners.

## Reusable Commands

Run from the repository root through a sanitized Python subprocess environment: remove all inherited `C8VOLT_*` and `GO_WANT_HELPER_PROCESS`, then set `GOFLAGS=''`, `GOENV='off'`, `GOWORK='off'`. Inspect fixtures and list matching tests first.

- Focused services: `go test ./internal/services/processinstance/... -run 'Resum|Suspend|Suspension' -count=1`.
- Focused models/normalization: `go test ./internal/domain ./internal/services/common -run 'Resumption|NormalizeProcessInstance' -count=1`.
- Stub consumer regressions: `go test ./c8volt/process ./c8volt/resource ./internal/services/processdefinition ./internal/services/ops ./internal/services/processinstance -run 'Suspension|Suspend|DeleteProcessDefinition|PurgeOrphan|DryRunCancelOrDelete' -count=1`.
- Passed affected-package race suite: `go test -race ./internal/domain ./internal/services/common ./internal/services/processinstance/... ./c8volt/process ./c8volt/resource ./internal/services/processdefinition ./internal/services/ops -count=1`.
- Passed repository compilation: `go test ./... -run '^$'`.

- Iteration 2 focused checks passed: `go test ./internal/services/processinstance -run 'ResumeProcessInstances' -count=1`; `go test ./c8volt/process -run 'Resum' -count=1`; suspension regressions in those packages; `go test ./c8volt/resource -run 'DeleteProcessDefinition' -count=1`; `go test ./cmd -run '^TestSuspendProcessInstance(Selection|FailFast|Workers|Activity)$' -count=1`.
- Iteration 2 broader validation passed: `go test -race ./internal/services/processinstance ./c8volt/process ./c8volt/resource -count=1` plus repository-wide untagged compilation `go test ./... -run '^$'`. The compile-only run is not behavioral evidence.

- Iteration 3 passed `go test ./cmd -run '^TestResumeProcessInstance' -count=1`, including the final empty-preview guard; the subsequently added malformed-stdin cases passed `go test ./cmd -run '^TestResumeProcessInstanceInvalidInputAndCapability$' -count=1`.
- Iteration 3 passed `go test ./cmd -run 'Contract|Metadata|Help|SuspendProcessInstance|Confirmation' -count=1`, the full `go test ./cmd -count=1`, `go test ./docsgen -count=1`, and `make docs-content`. Command-package coverage used only isolated fixtures; no PTY skips occurred.
- Iteration 3 focused race validation passed: `go test -race ./cmd -run '^TestResumeProcessInstance(Activity|Frozen|Workers|FailFast|Cancellation|Selection|FreshObservations)' -count=1`. The full repository race target was not repeated for this command-only work.

- Iteration 4 passed service planner checks: `go test ./internal/services/processinstance -run 'PlanProcessInstanceResumption|ProcessInstanceSuspensionPlan' -count=1`; public planner checks: `go test ./c8volt/process -run 'ResumptionPlan|SuspensionPlan' -count=1`, followed by the final `ResumptionPlan` slice after added no-wait/empty JSON assertions.
- Iteration 4 regressions passed: `go test ./c8volt/resource -run 'DeleteProcessDefinition' -count=1` and `go test ./cmd -run '^Test(SuspendProcessInstanceTree|ResumeProcessInstance)' -count=1`.
- Iteration 4 broader checks passed: `go test -race ./internal/services/processinstance ./internal/services/processinstance/traversal ./internal/services/processinstance/walker ./c8volt/process ./c8volt/resource -count=1` and `go test ./... -run '^$'` (compilation only). The HTTP planner fixture verifies read/search requests, the 1,000 child limit, no tenant/state pruning, child-only scope, local timing/no-wait guards, zero planning mutations and execution without rediscovery.

## Do Not Repeat

- Do not run integration tags, environment-backed helpers, live probes, or real Camunda validation. Real-engine buffering, timing, and permission behavior remain unverified.
- Do not edit generated clients or expose `--tree` until T005 delivers its complete lifecycle. T003 regenerated exact-key docs; T004 added library Go documentation only. T005 owns tree help, README and generated docs.
- Do not rerun the completed broad checks solely for audit or commit edits. Revalidate when subsequent implementation changes justify it.

## Current Handoff

- Next dependency-ready work is Phase 4 US2 T005: expose the complete tree command using the new public plan API and existing collection executor. T001–T004 are complete; no blockers. Recheck tasks.md. Clone TargetKeys before one approval; complete empty plans return before approval/activity/submission. Add frozen/empty-scope output, configured logging, terminal/activity/failure coverage and tree documentation. Reuse T003 command fixtures and T004 planner fixtures. Integration source remains unexecuted; real-engine timing, buffering and permissions remain unverified.
