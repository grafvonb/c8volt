# Ralph Memory

Feature: 261-retry-camunda-reads
Started: 2026-07-28T05:17:54Z

## Codebase Patterns
- `internal/services/httpc.New` installs `ReadRetryTransport` in the shared client path around the existing `LogTransport` chain rather than in command/facade callers.
- `WithActivitySink` depends on `unwrapLogTransport`, so every transport wrapper in the chain must be represented there for activity wiring to keep working.
- Existing mutation retry timing in `internal/services/retry.go` uses bounded attempts, exponential backoff, jitter, `Retry-After`, and context-aware sleep; `httpc` owns a separate read retry helper to avoid importing `internal/services` from `internal/services/httpc`.
- Final response/error mapping happens after generated-client transport calls through `internal/services/httpc/httpmap.go` and `internal/services/common/response.go`; exhausted read retries must return the final response with body still readable.
- Focused shared-client install coverage is `TestNewInstallsReadRetryTransport`.
- US2 coverage in `http_read_retry_test.go` confirms semantic 400/401/403/404/409 responses are single-call outcomes, exhausted retries return the final response object with headers/request/body readable, and canceled contexts interrupt retry sleep before another attempt.
- US3 coverage in `http_read_retry_test.go` confirms POST search plus DELETE/PATCH/PUT/non-search POST transient responses are single-call outcomes; the existing `isReadRetryMethod` GET/HEAD-only gate required no production change.
- `README.md` documents compact transient GET/HEAD retry behavior in the Ops-scale progress area; generated CLI docs remain unchanged because no command metadata changed.

## Decisions
- Foundational iteration introduced `ReadRetryTransport` with policy/timing primitives; US1 completed retry decisions and shared service installation.
- US1 retries only GET/HEAD temporary or timeout transport errors and HTTP 429/500/502/503/504; intermediate retry responses are closed, while the final response is returned untouched for later error mapping.
- Read retry logs use `httpActivityMessage` wording and `slog.Info` messages shaped as `Camunda read failed <operation>; <reason>; retrying in <delay>`, rate-limited per operation label by the transport.
- US2 required no production changes beyond the US1 implementation; the existing retry loop already treats non-transient responses as final, returns the last exhausted retry response without closing it, and propagates context cancellation from `sleepReadRetry`.
- US3 required no production changes beyond the existing method gate; mutation retry regression coverage in `internal/services/retry_test.go` remains the owner for resource-exhausted mutation retries.

## Gotchas
- Do not retry POST search or mutations in the generic read layer; feature scope is GET/HEAD only.
- Do not drain intermediate/final response bodies in retry code unless the body is restored for downstream generated-client/error mapping.
- `InstallAuthEditor` can wrap an already-created `ReadRetryTransport` when auth is installed after `httpc.New`; `unwrapLogTransport` already handles `AuthTransport -> ReadRetryTransport -> LogTransport`.

## Reusable Commands
- `go test ./internal/services/httpc -run 'ReadRetry|LogTransport' -count=1`
- `go test ./internal/services/httpc -count=1`
- `go test ./internal/services/httpc -run 'ReadRetry' -count=1`
- `go test ./internal/services/httpc ./internal/services -run 'ReadRetry|RetryCamundaMutation' -count=1`
- `go test ./internal/services/httpc -count=1`
- `go test ./internal/services -run 'RetryCamundaMutation' -count=1`
- `go test ./cmd ./internal/services/processinstance/... -run 'Orphan|Retry' -count=1`
- `make test`

## Do Not Repeat
- Do not add process-instance-specific retry branches for this feature; retry behavior belongs in `internal/services/httpc`.

## Current Handoff
- Feature complete; no handoff required.
