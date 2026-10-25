# Ralph Memory

Feature: 346-suspend-process-instances
Started: 2026-10-25T13:42:48Z

## Codebase Patterns

- Domain suspension models and typed status/submission/confirmation enums live in `internal/domain/processinstance_suspension.go`. All result fields have the planned JSON tags; collection callers must initialize `Items` to an empty slice.
- The private `(*v810.Service).submitProcessInstanceSuspension(ctx, key)` in `suspension_submission.go` sends the exact key and empty generated body. It returns submission evidence only, always `confirmation=not_requested`, with no reads, normalization, preview, capability guard, or public operation. T002 owns those surrounding responsibilities.
- `services.RetryCamundaMutation` retains its existing eight-attempt policy. Keep evidence outside its callback; pass original response/body/call error for HTTP retry classification. Convert terminal HTTP errors separately with `httpc.HttpStatusErr` so 429/503 retries are not suppressed.
- For local admission failures, a fresh `httpc.BeginRequestRateAttempt` observation survives real `http.Client.Timeout` substitution. Join its typed error with the original call error before retry classification; do not modify prior submission/status evidence. The shared HTTP service must be in the caller context for observation.

## Decisions

- T001 completed as the sole Phase 2 foundation work unit. No public service/facade/CLI suspension entry point was exposed. T002 and T003 remain unchanged and open.
- Only complete HTTP metadata, a valid status, empty 204 body, and no call error establish acceptance. Other success codes, malformed metadata and partial responses with errors remain unknown. Last usable submission HTTP status survives later missing/malformed responses and unsent attempts.
- Relevant explicit 4xx and structured 503 RESOURCE_EXHAUSTED problems establish rejection only without earlier uncertainty. Generic server/proxy failures remain unknown; only subsequent valid acceptance replaces uncertainty.
- Submission uses a nil functional retry logger and never logs returned errors. Configured HTTP DEBUG diagnostics remain independent. `Error` contains machine evidence only; T003 must derive human aggregate reasons from typed facts and preserve existing error classes/exits.
- Internal-only groundwork requires no CLI docs regeneration. Validation used injected generated clients and in-memory HTTP transports only; no real-engine behavior was tested.

## Gotchas

- The common retry helper can return a zero result on canceled backoff, and nil error after exhausting HTTP 429/503 responses. Use the accumulated result and separately converted last response error.
- A local unsent retry must preserve prior rejected or unknown evidence, not reset it to not_submitted. Restoring only the returned timeout error loses the local admission classification and can trigger unwanted retries.
- Existing completion helpers provide useful patterns but are not suspension workflows. Generic process-instance waiters and cancellation traversal cannot satisfy suspension's confirmation semantics.
- `testing/synctest` makes unchanged retry delays and HTTP timeout substitution deterministic and fast. When asserting an expired context, wait on `ctx.Done()`; sleeping exactly to the deadline can race its callback.

## Reusable Commands

- Targeted submission checks: `env GOENV=off GOFLAGS= go test ./internal/services/processinstance/v810 -run 'ProcessInstanceSuspensionSubmission' -count=1`.
- Adapter regressions/race check: `env GOENV=off GOFLAGS= go test ./internal/services/processinstance/v810 -race -count=1`.
- Both checks passed in iteration 1 after an initial compiling-stub red test and correction of a virtual-time test fixture. Tests cover protocol, retry exhaustion, cancellation, request-rate admission, actual HTTP timeout substitution, generated wire mapping, preserved status and plain/JSON configured DEBUG diagnostics.
- Use `gofmt` on touched Go files and `git diff --check`. Keep `GOENV=off GOFLAGS=` and inspect test entry points before subsequent validation; do not run integration scripts or live targets.

## Do Not Repeat

- Do not turn submission acceptance into observed suspension or perform reconciliation after rejected/unknown submission.
- Do not modify generated clients, shared retry policy, existing cancellation/deletion semantics, or shared envelopes for this feature.
- Do not emit raw report errors or retry chatter at normal/verbose levels. Command diagnostics and exit/envelope checks are still required in T003.

## Current Handoff

- Next dependency-ready work is T002 (Phase 3 / US1): complete service/facade single and collection workflow, capability guards, input normalization, preview, bounded confirmation waiter, no-wait, all affected interfaces/test doubles and isolated validation. Recheck `tasks.md` before selection. T001 is complete; there are no blocking dependencies. T003 remains dependent on T002.
