# Ralph Memory

Feature: 346-suspend-process-instances
Started: 2026-10-25T13:42:48Z

## Codebase Patterns

- Suspension domain models are in `internal/domain/processinstance_suspension.go`; public `SuspensionResult(s)` and string enums are in `c8volt/process/model.go`. Converters copy all evidence and allocate `items: []` for empty collections. Facades return converted reports alongside `ferrors.FromDomain` errors.
- `common.NormalizeProcessInstanceSuspensionKeys` trims, checks exactly 16 ASCII digits, and deduplicates in first-input order. The collection service guards capability, validates the whole selection, initializes keyed not-attempted slots, previews locally to validate timing before pool dispatch, and merges actual indexed results without discarding partial evidence. `ctx.Err()` is joined explicitly even if the pool ran no callback.
- All version service APIs have `ValidateProcessInstanceSuspension` and `SuspendProcessInstance`; v88/v89 reject without I/O, including preview/empty collection paths. Factory selection is unchanged. v810 singular owns local checks, T001 submission, then bounded confirmation unless NoWait. Empty supported collections return no work; preview never implies eligibility or acceptance.
- `waiter.WaitForProcessInstanceSuspension` takes only a keyed reader. It validates BackoffConfig, uses NextDelay and a timeout context bounded by the caller, and retains the last matching observation. Only SUSPENDED confirms; COMPLETED/CANCELED/TERMINATED stop with conflict. Typed not-found becomes ABSENT and clears the date; authorization wins over joined absence. Unknown states normalize to UNKNOWN. Failed/mismatched reads preserve prior evidence.
- `httpc.WithoutReadRetryLogs` suppresses functional retry INFO only for the derived context. Suspension confirmation uses it to leave aggregate diagnostics to the caller, while read retry policy and configured redacted HTTP DEBUG exchanges remain intact. No service/facade human diagnostics are emitted by suspension.

- CLI construction/input validation lives in `suspend_processinstance.go`, frozen preview/approval/execution in `suspend_processinstance_batch.go`, and selected-format views/grouped diagnostics in `cmd_views_processinstance_suspend.go`. Use `collectExplicitPIAdminInputOptions`; local preview performs no reads and never supplies tenant evidence. One collection path handles all sizes.
- Suspension human results deliberately use configured stdout; plain prompts use configured stderr, and errors use the command-context logger. JSON errors attach complete results to one shared envelope before exiting. Keys-only execution means submission accepted, including unconfirmed failures; preview keys mean selection only. Quiet is orthogonal to machine results. The view explicitly succeeds for preview/empty payloads even with no-wait.
- Safe human failure labels come only from typed status/submission/observation facts and group repeated causes in one bounded ERROR. Verbose allowlists submission/confirmation/state, omitting raw machine errors and dates. Writer failure emits one bounded diagnostic, never retries rendering, and preserves an existing operation error's exit.

## Decisions

- T001–T003 are complete. The CLI, public workflow, operational guidance, generated docs, and static suspension fixture adoption are delivered. No live validation was performed; real-engine behavior remains unverified.
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

- `--no-err-codes` resolves failures to exit 0 (not generic exit 1); always preserve `ferrors.ResolveExitCode` rather than inventing a suspension exception.
- The command count is pinned in `cmd/command_contract_test.go`, `docsgen/main.go`, and `docsgen/main_test.go`: suspend adds a verb and child, taking the current count from 58 to 60 while preserving the historical assessment at 57.
- `addBackoffFlagsAndBindings` exposes hidden timeout/max-retries controls only; initial delay is configuration, not a CLI flag. Use test-owned YAML for short configured timing and test the flag override separately.

## Reusable Commands

- Targeted services/facades: `env GOENV=off GOFLAGS= go test ./internal/services/processinstance/... ./c8volt/process ./internal/services/httpc -run 'ProcessInstanceSuspension|ReadRetryTransport' -count=1` (iteration 2 passed). The waiter uses deterministic virtual-time tests.
- CLI: `env GOENV=off GOFLAGS= go test ./cmd -run 'SuspendProcessInstance' -count=1` passed in iteration 3. Tests exercise fake-server execution, both logger formats/destinations, DEBUG/verbose/quiet modes, JSON EOF/partial payloads, worker barriers, cancellation, writer failure, timeout configuration, and real macOS terminal stdin with redirected stdout. No terminal-platform skip was needed on this host.
- Affected regressions: `go test ./cmd ./docsgen -run 'SuspendProcessInstance|Cancel|DeleteProcessInstance|GetProcessInstance|ExpectProcessInstance|CommandContract|CommandCapability|CapabilityDocument|CLICommandTree|RootHelp' -count=1` found only the old capability count assertion; after updating it, its targeted check passed. All affected regressions subsequently passed in the full race run.
- Final integrated `env GOENV=off GOFLAGS= make test` passed once in iteration 3 (`go test ./... -race -count=1`; cmd took 314.165s). Before running, inspect Make's ordinary target, remove inherited C8VOLT_* and helper selectors, Make/Go overrides, and isolate config directories. No integration build tags or live helpers were enabled; untagged integration-package checks are isolated/static tests.
- `env GOENV=off GOFLAGS= make docs-content`, `bash -n integration/scripts/run-c810-suspended.sh`, touched-file gofmt, and `git diff --check` passed. Generated docs include both suspension commands and the updated command tree. Never source or execute the fixture during implementation.

## Do Not Repeat

- Do not turn acceptance into confirmation, reconcile after rejected/unknown submission, expand parents/children, or use generic wait/cancel/delete helpers for suspension.
- Do not modify generated clients, shared retry policy, existing mutation semantics or shared envelopes. The only shared retry change is opt-in context suppression of functional logs.
- Do not print/log raw per-key Error, joined errors or transport strings at normal/verbose levels. Machine evidence remains intact. CLI must derive safe grouped facts and preserve original classification/exit and one JSON envelope.
- Do not run live fixtures/integration targets or request a live target; Bash fixture adoption is static-only validation.

## Current Handoff

- Feature complete; no handoff required.
