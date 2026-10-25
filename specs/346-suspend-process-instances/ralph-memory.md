# Ralph Memory

Feature: 346-suspend-process-instances
Started: 2026-10-25T13:42:48Z

## Codebase Patterns

- Suspension domain models are in `internal/domain/processinstance_suspension.go`; public `SuspensionResult(s)` and string enums are in `c8volt/process/model.go`. Converters copy all evidence and allocate `items: []` for empty collections. Facades return converted reports alongside `ferrors.FromDomain` errors.
- `common.NormalizeProcessInstanceSuspensionKeys` trims, checks exactly 16 ASCII digits, and deduplicates in first-input order. The collection service guards capability, validates the whole selection, initializes keyed not-attempted slots, previews locally to validate timing before pool dispatch, and merges actual indexed results without discarding partial evidence. `ctx.Err()` is joined explicitly even if the pool ran no callback.
- All version service APIs have `ValidateProcessInstanceSuspension` and `SuspendProcessInstance`; v88/v89 reject without I/O, including preview/empty collection paths. Factory selection is unchanged. v810 singular owns local checks, T001 submission, then bounded confirmation unless NoWait. Empty supported collections return no work; preview never implies eligibility or acceptance.
- `waiter.WaitForProcessInstanceSuspension` takes only a keyed reader. It validates BackoffConfig, uses NextDelay and a timeout context bounded by the caller, and retains the last matching observation. Only SUSPENDED confirms; COMPLETED/CANCELED/TERMINATED stop with conflict. Typed not-found becomes ABSENT and clears the date; authorization wins over joined absence. Unknown states normalize to UNKNOWN. Failed/mismatched reads preserve prior evidence.
- `httpc.WithoutReadRetryLogs` suppresses functional retry INFO only for the derived context. Suspension confirmation uses it to leave aggregate diagnostics to the caller, while read retry policy and configured redacted HTTP DEBUG exchanges remain intact. No service/facade human diagnostics are emitted by suspension.

## Decisions

- T001 and T002 are complete. T003 remains the sole open task, with all CLI/output/terminal/docs/fixture requirements intact. No CLI command has been introduced yet, so README/generated CLI docs remain T003 work.
- Only complete HTTP metadata, valid status, empty 204 body, and no call error establish submission acceptance. Unexpected success, malformed metadata and partial responses with errors remain unknown. Preserve the last usable submission status; confirmation never overwrites it.
- Relevant explicit 4xx and structured 503 RESOURCE_EXHAUSTED establish rejection only without earlier uncertainty. Later rejection/unsent attempts cannot erase uncertainty; a subsequent valid acceptance can. Rejected/uncertain submissions never enter observation.
- Confirmation timing is validated before mutation, including default-mode preview. NoWait does not need confirmation timing or read permission. Supported empty collection calls remain no-I/O no-ops after capability/context checks.
- Added a nil generated-read-response guard to v810 GetProcessInstance so malformed confirmation reads return typed evidence rather than panic. Existing cancellation/deletion/generic waiter semantics are unchanged.
- Public `process.API` now requires both suspension methods in custom implementations. Concrete repository service and public-interface test doubles were updated, including `cmd/process_api_stub_test.go` and resource tests; legacy fixtures explicitly reject suspension.

## Gotchas

- `RetryCamundaMutation` can return a zero result on canceled backoff and nil error after exhausting 429/503 responses. Keep evidence outside its callback and separately convert terminal HTTP responses. Pass original call error/response/body to retry classification.
- Restore attempt-local request-rate admission errors via `BeginRequestRateAttempt` after actual HTTP timeout substitution. A local unsent attempt must preserve earlier submission evidence/status. Keep the shared HTTP service in caller context.
- The worker pool can leave zero slots and return no error when cancellation prevents every callback. Preserve initialized keyed slots and join caller context explicitly. Fail-fast cancels in-flight workers but must not erase accepted evidence.
- Existing read transport retries emit INFO independently of service call-option suppression. Use context-scoped `WithoutReadRetryLogs`; do not disable retries or DEBUG diagnostics to solve this. The added shared transport test proves unrelated requests retain normal logging.
- Generated `suspendedDate` parses RFC3339 timestamps. Wire fixtures must use valid timestamps; arbitrary strings are suitable only for direct domain-reader fixtures. Synctest makes deadline/backoff/worker tests deterministic; wait on ctx.Done() to avoid deadline callback races.

## Reusable Commands

- Targeted: `env GOENV=off GOFLAGS= go test ./internal/services/processinstance/... -run ProcessInstanceSuspension -count=1` and the corresponding `./c8volt/process` command. Iteration 2 also ran both with `./internal/services/httpc` and pattern `ProcessInstanceSuspension|ReadRetryTransport`; passed.
- Affected regressions/races: `env GOENV=off GOFLAGS= go test ./internal/services/processinstance/... ./internal/services/common ./internal/services/httpc ./c8volt/process ./c8volt/resource ./internal/services/processdefinition/... ./internal/services/ops -race -count=1`. All packages except the initially missing resource public-interface methods passed; after fixing those methods, resource tests passed with race enabled.
- All repository test packages were compiled with `go test ./... -run '^$' -count=1`; two missing public-interface stubs were found in cmd/resource. After repair, both packages passed the same compile-only check; all other packages had passed. This was compilation, not a full suite pass.
- Final scrubbed-environment race regression: `go test ./cmd ./c8volt/resource ./internal/services/processinstance -run 'Suspended|ProcessInstanceSuspension|Resource|Client' -race -count=1` passed with GOENV=off, empty GOFLAGS, C8VOLT_* and helper selectors removed, and GORACE=atexit_sleep_ms=0. Includes existing get/expect/suspended cancellation/deletion coverage and cancellation after preview before scheduling.
- Gofmt and `git diff --check` passed. Test entry points use mocks, in-memory transports, test-owned HTTP servers and isolated command fixtures; no live validation occurred. Full `make test` remains the integrated T003 check, not claimed as performed here.

## Do Not Repeat

- Do not turn acceptance into confirmation, reconcile after rejected/unknown submission, expand parents/children, or use generic wait/cancel/delete helpers for suspension.
- Do not modify generated clients, shared retry policy, existing mutation semantics or shared envelopes. The only shared retry change is opt-in context suppression of functional logs.
- Do not print/log raw per-key Error, joined errors or transport strings at normal/verbose levels. Machine evidence remains intact. CLI must derive safe grouped facts and preserve original classification/exit and one JSON envelope.
- Do not run live fixtures/integration targets or request a live target; Bash fixture adoption is static-only validation.

## Current Handoff

- Next dependency-ready work is T003 (Phase 5 / US3): CLI collection lifecycle, safe mode-aware output and failure diagnostics, real-terminal tests, source help/metadata, README/generated docs, and static Bash suspension-fixture adoption. Recheck tasks.md before selection. T001/T002 are complete; there are no blocking dependencies. Reuse the completed facade collection API and keep all backend mechanics in services; all CLI diagnostic/exit/envelope/PTY requirements still apply.
