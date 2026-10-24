# Research: Request Pacing and DEBUG Rate Reporting

**Feature**: 353-request-rate-reporting | **Date**: 2026-10-24

Scope and acceptance requirements are in [spec.md](spec.md). Research used the existing configuration, HTTP, retry, logging, and command lifecycle code, with two focused read-only reviews. No live environment was inspected or exercised.

## 1. Shared request boundary

**Decision:** Add one invocation-owned transport in `internal/services/httpc`, below `ReadRetryTransport` and above existing request logging/diagnostics. The usual chain becomes `AuthTransport → ReadRetryTransport → RequestRateTransport → LogTransport → DiagnosticsTransport → underlying transport`. Update the existing transport traversal helpers for the new wrapper. Count immediately after successful admission and a final context check, immediately before delegating to the existing transport chain.

**Rationale:** `cmd/root_services.go` constructs the shared client and `cmd/cmd_cli.go` passes it through the facades. This covers discovery, preflight, mutations, confirmation reads, and nested service requests without modifying their workflows. Read retries pass through the wrapper on each attempt; service-level mutation retries also re-enter it. Waiting outside logging/diagnostics avoids recording a queued request as a dispatched network attempt.

**Alternatives considered:** Per-command/per-worker limiters would multiply the allowance and duplicate wiring. Wrapping outside read retries would undercount attempts. Changing worker pools, generated clients, or facade APIs is unnecessary. Cookie bootstrap traffic uses the shared client and crosses the same boundary; standalone identity-provider clients keep their existing diagnostic sharing without acquiring this limiter or counter. Transport-internal replays remain outside the observable application boundary.

## 2. Limiter and the zero-rate path

**Decision:** Use one `golang.org/x/time/rate.Limiter` with burst 1 and context-aware admission for positive rates. Pin `golang.org/x/time v0.16.0`, compatible with the repository's Go 1.26/toolchain 1.26.2. Zero bypasses limiter creation and waiting. With both features disabled, retain the existing transport path; measurement-only mode adds a counter increment, not a limiter call. Never hold a reporting lock or perform logger writes on the request path.

**Rationale:** The maintained limiter supports concurrent callers, fractional rates, cancellation, and deadline-aware waits. It avoids custom token scheduling and reservation cancellation code. Its [documented behavior](https://pkg.go.dev/golang.org/x/time@v0.16.0/rate) and [module requirement](https://raw.githubusercontent.com/golang/time/v0.16.0/go.mod) match this use. Application dispatch timing is subject to scheduler/timer precision; tests must record delegation starts and declare a small timing tolerance, not claim wire-level scheduling guarantees.

**Alternatives considered:** A home-grown timer/queue adds cancellation and fairness maintenance. Per-worker sleeps fail the shared-rate requirement. An always-active limiter would impose unnecessary work on the default path. This is the only new production dependency proposed.

## 3. Local waiting and existing retries

**Decision:** Return a small typed local-wait error that preserves cancellation/deadline identity and says the affected request was not sent. Normalize the limiter's early insufficient-deadline result to this deadline error even when the context has not yet expired. Explicitly exclude this error from read retries. For mutation retries, create a fresh, concurrency-safe context observation for each call of the existing retry callback; the pacing transport records a local-wait failure there. After the callback, check that observation before normal retry classification and return its typed cause without another attempt.

**Rationale:** `internal/services/retry.go` retries timeout-like network errors. Go's [`http.Client` implementation](https://cs.opensource.google/go/go/+/refs/tags/go1.26.2:src/net/http/client.go) can replace a transport error with a timeout wrapper when `Client.Timeout` expires, retaining text but losing the original error type. A typed error alone therefore does not reliably prevent a local wait from becoming a mutation retry. The per-attempt context observation survives request context cloning and supplies proof that this attempt was never delegated. It is reset between attempts and is not invocation-global state.

**Alternatives considered:** Parsing error strings is brittle. Moving the limiter outside `http.Client` would exclude waiting from its timeout. Replacing the HTTP timeout mechanism or changing general network retry eligibility is broader than this issue. The narrow observation helper belongs in `httpc`; the parent services package already uses `httpc`, so no new architectural layer or dependency cycle is needed.

**Boundary:** Preserve timeout/cancellation classification through service errors, facade conversion, and command rendering. Use the configured logger and existing compact aggregate-error helpers for human diagnostics. Earlier attempts or other resources might already have been sent; “not sent” applies only to the recorded attempt. Backend timeouts, 429/503 handling, retry budgets, and backoff remain unchanged.

Create per-attempt observations only when the invocation's HTTP service has positive pacing enabled; the helper is a no-op otherwise. Reuse the service already carried in context to determine this. Thus the default mutation path does not gain context allocation or synchronization solely for disabled pacing.

## 4. Configuration resolution

**Decision:** Add `HTTP.RequestRate float64` and `HTTP.RequestRateReportInterval string`, matching the existing HTTP duration convention. Register defaults 0 and `5s`; bind only `--request-rate` through the existing root resolver. Reuse reflection-based profile/environment overlays and changed-flag precedence. Extend HTTP normalization with the existing configured-key pattern so omission gets a default but explicitly empty interval input is rejected.

**Rationale:** `cmd/root_config.go` and `config/config.go` already distinguish explicit zero from omission. The new values do not need a separate configuration system. Validate finite nonnegative rate and nonnegative parsed duration before authentication or API work. Verify malformed numeric source values cannot silently decode as zero.

**Alternatives considered:** A custom interval flag, special per-mutation settings, and zero-as-missing normalization would add surface area or break the agreed override behavior.

## 5. Counting, reporting, and lifetime

**Decision:** When effective DEBUG and a positive interval enable reporting, maintain an atomic cumulative dispatch counter and one reporter goroutine. Each tick snapshots the counter and current monotonic time; differences from the previous snapshot produce count, elapsed duration, and requests/second. Emit a compact complete message through the supplied logger, usable in plain and structured formats. Empty windows report zero. Stop without a final partial-window summary.

**Rationale:** Atomic snapshots avoid lost/double-counted increments at boundaries; a start concurrent with a snapshot belongs to exactly one adjacent window. Actual elapsed time handles delayed ticks. No request waits for a reporter or log sink. Existing effective logger resolution already covers configured DEBUG, `--debug`, quiet precedence, and the command's inherited stderr destination.

**Lifecycle decision:** Give the HTTP service an idempotent close operation. Pass the root invocation context through a small `WithInvocationContext(ctx)` option; `httpc.New` currently has no context parameter, so cancellation must be wired explicitly. Register one Cobra finalizer for the existing singleton root and keep only the current service cleanup handle there. Install it immediately after service creation, consume/clear it on finalization, and close any stale handle before a new invocation. This covers returned success, bootstrap errors, and execution errors without per-command wiring or accumulating finalizers. Explicit process exits already terminate goroutines; do not refactor exit handling for this feature. Direct service users/tests own close. Stop the ticker and terminate reporting promptly on close or invocation cancellation; do not wait for the next tick. Concurrent execution of the singleton Cobra root is not a new supported mode.

**Alternatives considered:** Reporting in each request would add output contention. A persistent post-run hook alone misses returned errors; an `Execute` defer alone misses direct `ExecuteC` use. A general cleanup framework or edits across all mutation commands are unnecessary.

## 6. Validation and scope control

**Decision:** Use isolated round-tripper fakes, test-owned HTTP servers, command subprocesses, and terminal fixtures. Reuse `cmd/root_api_diagnostics_test.go`, config precedence tests, retry tests, and mutation command fixtures. Measure zero-rate overhead with focused comparative benchmarks; do not impose an invented throughput threshold. Run the full race suite after focused implementation checks because shared concurrent runtime behavior and a dependency change have broad impact. Planning itself needs only document checks.

**Rationale:** These tests can prove dispatch counts, pacing, local timeout handling, report lifetime, logger routing, and unchanged command contracts without touching Camunda. Full HTTP-client timeout tests are essential; testing only `RoundTrip` misses the wrapper-loss issue.

**Alternatives considered:** Live tests and deliberate production backpressure are outside the implementation boundary. A new benchmark/telemetry platform, batching modes, adaptive control, and unrelated mutation repairs are outside scope. All planning unknowns are resolved; no clarification is required.
