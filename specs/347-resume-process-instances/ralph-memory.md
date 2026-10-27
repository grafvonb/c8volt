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

## Decisions

- Keep resumption types and lifecycle separate from suspension; reuse existing mutation retry/rate primitives without a generic mutation framework.
- Submission accepts only valid empty 204. Unknown evidence survives later rejection; last valid submission status survives transport/malformed/admission failures. Only valid acceptance resolves ambiguity.
- Every successful matching read replaces state/date evidence, including clearing an omitted date. Failed reads preserve prior evidence; typed not-found records ABSENT and clears the date. Late reads cannot confirm.
- `resumptionFailure` centralizes disposition classification while returning original error classes and machine evidence. New service paths emit no human diagnostics; configured transport DEBUG remains independent and redacted.
- T001 was completed in iteration 1; T002 completed the collection and public singular/collection APIs in iteration 2. CLI delivery and tree planning remain subsequent work.

## Gotchas

- Adding public resumption methods required explicit unsupported methods on `stubProcessAPI` in `cmd/process_api_stub_test.go` and `c8volt/resource/client_test.go`; interface-embedding fixtures still compile. Downstream public API implementers also need the two new methods.
- `toolx.MapSlice` preserves nil; the resumption converter follows the existing suspension make-and-map pattern to guarantee initialized result arrays.
- Extending internal process-instance interfaces requires all three version contracts plus stubs in process/resource facades, process-definition deletion, orphan purge, dry-run tests, and the v810 generated-client mock. These compile together now.
- NoWait exempts confirmation timing validation even without config; ordinary dry-run still validates timing. Exact preview never reads or submits.
- Do not interpret no-wait acceptance, rejected submission followed by desired state, or absence of a suspension date as confirmed resumption.
- The repository-wide `go test ./... -run '^$'` was compilation-only, without integration tags; it is not behavioral or live evidence. Race-enabled behavioral coverage was limited to the affected packages listed below.

## Reusable Commands

Run from the repository root through a sanitized Python subprocess environment: remove all inherited `C8VOLT_*` and `GO_WANT_HELPER_PROCESS`, then set `GOFLAGS=''`, `GOENV='off'`, `GOWORK='off'`. Inspect fixtures and list matching tests first.

- Focused services: `go test ./internal/services/processinstance/... -run 'Resum|Suspend|Suspension' -count=1`.
- Focused models/normalization: `go test ./internal/domain ./internal/services/common -run 'Resumption|NormalizeProcessInstance' -count=1`.
- Stub consumer regressions: `go test ./c8volt/process ./c8volt/resource ./internal/services/processdefinition ./internal/services/ops ./internal/services/processinstance -run 'Suspension|Suspend|DeleteProcessDefinition|PurgeOrphan|DryRunCancelOrDelete' -count=1`.
- Passed affected-package race suite: `go test -race ./internal/domain ./internal/services/common ./internal/services/processinstance/... ./c8volt/process ./c8volt/resource ./internal/services/processdefinition ./internal/services/ops -count=1`.
- Passed repository compilation: `go test ./... -run '^$'`.

- Iteration 2 focused checks passed: `go test ./internal/services/processinstance -run 'ResumeProcessInstances' -count=1`; `go test ./c8volt/process -run 'Resum' -count=1`; suspension regressions in those packages; `go test ./c8volt/resource -run 'DeleteProcessDefinition' -count=1`; `go test ./cmd -run '^TestSuspendProcessInstance(Selection|FailFast|Workers|Activity)$' -count=1`.
- Iteration 2 broader validation passed: `go test -race ./internal/services/processinstance ./c8volt/process ./c8volt/resource -count=1` plus repository-wide untagged compilation `go test ./... -run '^$'`. The compile-only run is not behavioral evidence.

## Do Not Repeat

- Do not run integration tags, environment-backed helpers, live probes, or real Camunda validation. Real-engine buffering, timing, and permission behavior remain unverified.
- Do not edit generated clients or regenerate CLI docs for these library-only changes; command docs belong with T003/T005.
- Do not rerun the completed broad checks solely for audit or commit edits. Revalidate when subsequent implementation changes justify it.

## Current Handoff

- Next dependency-ready work is Phase 3 US1 T003: exact-key CLI command/runner/views, diagnostic/output/prompt/activity acceptance, README/help/generated docs, and source-only integration fixture guidance. T001/T002 are complete and there are no blockers. Recheck tasks.md; T004 is independently ready after T002 but follows T003 in declared order, and T005 requires both. Stop at the selected story boundary.
