# Ralph Memory

Feature: 347-resume-process-instances
Started: 2026-10-27T09:01:23Z

## Codebase Patterns

- Versioned `ResumeProcessInstance` now owns exact-key preview, submission, and post-acceptance confirmation. V810 uses the existing generated resumption endpoint with `{}`; v88/v89 reject before config, key access, or I/O.
- `common.NormalizeProcessInstanceKeys` trims exactly 16 ASCII decimal digits and deduplicates first-input order. The existing suspension-named entry point delegates to it, preserving its callers and behavior.
- `waiter.WaitForProcessInstanceResumption` exposes only `GetProcessInstance`. Caller acceptance is a documented precondition. ACTIVE/COMPLETED confirms; CANCELED/TERMINATED stops with conflict; absence and unknown remain bounded observations.
- Adapter fixtures use generated response mocks or in-memory RoundTrippers; `testing/synctest` exercises deadlines, retries, and request-rate pacing without wall-clock delays or real services.

## Decisions

- Keep resumption types and lifecycle separate from suspension; reuse existing mutation retry/rate primitives without a generic mutation framework.
- Submission accepts only valid empty 204. Unknown evidence survives later rejection; last valid submission status survives transport/malformed/admission failures. Only valid acceptance resolves ambiguity.
- Every successful matching read replaces state/date evidence, including clearing an omitted date. Failed reads preserve prior evidence; typed not-found records ABSENT and clears the date. Late reads cannot confirm.
- `resumptionFailure` centralizes disposition classification while returning original error classes and machine evidence. New service paths emit no human diagnostics; configured transport DEBUG remains independent and redacted.
- T001 is the sole completed work unit in iteration 1. Public collection/facade APIs and CLI delivery remain subsequent work.

## Gotchas

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

## Do Not Repeat

- Do not run integration tags, environment-backed helpers, live probes, or real Camunda validation. Real-engine buffering, timing, and permission behavior remain unverified.
- Do not edit generated clients or regenerate CLI docs for this internal-only change; command docs belong with T003/T005.
- Do not rerun the completed broad checks solely for audit or commit edits. Revalidate when subsequent implementation changes justify it.

## Current Handoff

- Next dependency-ready work is Phase 2 T002: ordered internal collection execution and thin public singular/collection APIs, mechanical conversion, all affected public stubs, and isolated worker/partial-result tests. T001 is complete; there are no blocking dependencies. Recheck tasks.md before selection. T003–T005 remain open and outside iteration 1.
