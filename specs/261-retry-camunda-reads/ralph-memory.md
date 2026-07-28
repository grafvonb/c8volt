# Ralph Memory

Feature: 261-retry-camunda-reads
Started: 2026-07-28T05:17:54Z

## Codebase Patterns
- `internal/services/httpc.New` currently installs `LogTransport` directly; future retry installation should wrap this shared client path rather than command/facade callers.
- `WithActivitySink` depends on `unwrapLogTransport`, so every transport wrapper in the chain must be represented there for activity wiring to keep working.
- Existing mutation retry timing in `internal/services/retry.go` uses bounded attempts, exponential backoff, jitter, `Retry-After`, and context-aware sleep; `httpc` owns a separate read retry helper to avoid importing `internal/services` from `internal/services/httpc`.
- Final response/error mapping happens after generated-client transport calls through `internal/services/httpc/httpmap.go` and `internal/services/common/response.go`; exhausted read retries must return the final response with body still readable.

## Decisions
- Foundational iteration introduced `ReadRetryTransport` as a delegate-only transport with policy/timing primitives; actual retry decisions and service installation remain in US1 tasks.

## Gotchas
- Do not retry POST search or mutations in the generic read layer; feature scope is GET/HEAD only.
- Do not drain intermediate/final response bodies in retry code unless the body is restored for downstream generated-client/error mapping.

## Reusable Commands
- `go test ./internal/services/httpc -run 'ReadRetry|LogTransport' -count=1`
- `go test ./internal/services/httpc -count=1`

## Do Not Repeat
- Do not add process-instance-specific retry branches for this feature; retry behavior belongs in `internal/services/httpc`.

## Current Handoff
- Next iteration should start US1 by adding failing GET/HEAD/temporary-error retry tests in `internal/services/httpc/http_read_retry_test.go`, then implement retry decisions, backoff use, compact logging, and shared client installation without changing POST/search behavior.
