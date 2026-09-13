# Research: Opt-in API Request Diagnostics

## Resolved questions

All design questions identified during planning are resolved below. Research used the current repository, independent transport and diagnostic-semantics reviews, and the primary Go documentation referenced below.

### 1. Central observation and coverage

**Decision:** Implement the observer in `internal/services/httpc`, below existing retry and logging wrappers: `AuthTransport → ReadRetryTransport → LogTransport → DiagnosticsTransport → base transport`. Disabled diagnostics leave the existing transport chain unchanged. A single invocation-owned collector is shared by its API and authentication clients.

**Rationale:** `httpc.New` in `internal/services/httpc/service.go` creates the shared client; `cmd/root_services.go` installs it before authenticator initialization. `cmd/cmd_cli.go` passes that client through `c8volt.WithHTTPClient` to all versioned service factories. This covers Camunda 8.7–8.10 and the existing Operate/Tasklist clients without generated code or individual command instrumentation.

**Alternatives considered:** Command/facade observation duplicates logic and hides retries; an observer outside `ReadRetryTransport` collapses attempts; changes to generated clients are unnecessary and violate the source constraints.

### 2. Authentication requests

**Decision:** Attach the same collector to the separate unauthenticated OAuth token client using the same `httpc` attachment helper as the API client, always beneath LogTransport when present. Recover the collector from the supplied API client's known transport chain; do not pass it through arbitrary global state. Preserve the token client's current timeout and absence of auth/retry wrappers. Cookie authentication continues to use the supplied shared client.

**Rationale:** `internal/services/auth/oauth2/service.go` creates its own `LogTransport` and would otherwise escape observation. Reusing the authenticated API client would risk authentication recursion. `auth/cookie/service.go` already uses the supplied client, including initialization traffic.

**Alternatives considered:** Excluding tokens leaves real workflow latency unexplained. Globally registering collectors leaks output across invocations. Wrapping the token client with the API retry/auth chain changes behavior.

### 3. Invocation wiring and stderr

**Decision:** Reuse resolved logger configuration and the existing invocation logger before authentication initialization. Capture `cmd.ErrOrStderr()` before activity wrapping and construct the normal configured logger with that writer. Pass debug/logger context into httpc; install no observer when DEBUG is filtered. Reuse `log.Debug` for emission; do not create a diagnostic writer or logger.

**Rationale:** `cmd/root.go` currently uses root stderr and can override a configured leaf destination. Effective command stderr must remain authoritative. Existing logging/activity helpers provide formatting, synchronization and terminal coexistence; quiet and configured log levels intentionally suppress DEBUG diagnostics.

**Alternatives considered:** Direct writes bypass quiet, configured formats and shared logging conventions. Raw process stderr ignores command routing; a mutable global writer breaks isolation. No command-specific runner or separate logger is needed.

### 4. Exchange boundaries and lifecycle

**Decision:** One observed exchange is one call to the instrumented underlying RoundTripper. Explicit HTTP retry attempts, service retries and redirects each reach this boundary independently. Emit exactly once on transport failure, known bodyless response, response EOF, non-EOF read error, or early Close. Never force reads or close a body for diagnostics.

**Rationale:** Generated response parsers normally read then close bodies. `internal/services/httpc/read_retry.go` closes discarded retry responses without draining them; those records must remain incomplete. `http.Transport` may internally retransmit during a single RoundTrip; its hidden sends cannot be represented as independent full-response records at this boundary. Document that limit instead of changing the transport.

**Alternatives considered:** Emitting at response headers misses body time; draining on close changes consumption; waiting for a second close after EOF delays completed records; a background timeout invents an observation and changes lifecycle responsibilities.

### 5. Timing and byte semantics

**Decision:** Use monotonic elapsed time for durations; the existing logger supplies emission timestamps in its configured format, with no duplicate start timestamp in the message. Compose `httptrace.ClientTrace` hooks with any existing trace. Capture `t0` immediately before delegated RoundTrip, `th` immediately when it returns final response headers, and `te` at observed body termination. Emit `headers=th-t0`, `body=te-th`, and `total=te-t0`; a known bodyless response ends at `th`. `GotFirstResponseByte` does not define `headers`, because it can precede final headers, including informational responses. Total ends at the observed terminal event. Preserve all completed DNS/connect/TLS phase samples; match overlapping connect attempts by network/address without emitting the address. Do not fabricate absent phases.

**Rationale:** Hooks may execute concurrently or after response termination. Protect mutable exchange state and freeze a snapshot once; ignore late changes to emitted data. Happy Eyeballs may produce concurrent connect samples, so a single overwritten start time is incorrect. Phase samples are observations, not additive portions of total elapsed time.

**Decision:** Count bytes returned by delegated request/response body reads, including bytes returned with an error. These are observed body bytes, not network/framing bytes or declared Content-Length. Automatic decompression can make response counts decoded bytes. Preserve Read/Close behavior and request replay through GetBody. Snapshot request count and completion evidence at record emission; an upload still in progress is incomplete, not a complete transfer.

**Alternatives considered:** Content-Length is declared size, not transferred evidence; payload dumps buffer/read content; replacing existing trace hooks breaks callers; first-byte timing answers a different question from the agreed final-header metric and is not emitted.

Primary references: [Go HTTP contracts](https://pkg.go.dev/net/http), [Go HTTP tracing and hook concurrency](https://pkg.go.dev/net/http/httptrace).

### 6. Safe diagnostic representation

**Decision:** Emit one compact `api #<sequence> <METHOD> <safe-path-and-query>: status=<code> error=<failure>` line followed by space-separated timing, connection and secondary metadata fields. Keep numbers beside short technical field names; quote/escape unsafe textual tokens. Use `httptrace` DNSStart/Done, ConnectStart/Done and TLSHandshakeStart/Done for `dns`, `tcp` and `tls`; only TCP network attempts qualify as `tcp`. GotConn supplies `conn=new|reused`. Use classified error categories instead of arbitrary error text. Build a new redacted record rather than dumping requests or headers. Exact fields and redaction rules are in `contracts/api-diagnostics.md`.

**Rationale:** The old `LogTransport` raw URL start line is removed. LogTransport retains activity and optionally uses `httputil.DumpRequestOut`; explicit dumps are separate from exchange diagnostics. Diagnostics must never enable those options. Error strings and free-form header descriptions can contain payloads and credentials. A safe category preserves the failure signal without trusting arbitrary strings.

**Decision:** Strip URL userinfo/fragments; classify query parameters after decoding and normalizing names, remove secret values and scrub their known encoded/decoded representations from retained fields. Preserve operational query values; fail closed for malformed or demonstrably sensitive values. Use an explicit correlation-header allowlist, parse Retry-After, and retain only validated Server-Timing metric names/durations, not free-form descriptions. Seed known secrets from configured credentials, request auth/cookie/query values and sensitive response headers (including parsed Set-Cookie values) without inspecting bodies. Collect response secrets before sanitizing response correlation fields.

**Alternatives considered:** Blanket omission of all query parameters loses required operational evidence; raw arbitrary header/error output cannot satisfy redaction. No claim is made that arbitrary business data concealed under an unrelated operational identifier can be semantically detected; the contract targets defined sensitive classes, known secret reflection, and omission of payload-bearing fields.

### 7. Resource and failure policy

**Decision:** Store only per-exchange metadata, counters and trace samples; no history, payload copies, extra requests or queue. Format the safe message after freezing state, then emit through the existing invocation logger and its synchronized writer. Writer failure is best-effort diagnostic loss and must not replace an HTTP/body error or alter command exit status; do not retry writes into other streams.

**Rationale:** This preserves operation outcomes and confines cost to actual exchanges. As with existing stderr logging, a slow writer can add output latency; no zero-overhead or nonblocking guarantee is invented. Emission timing excludes diagnostic formatting/writing time. An abandoned body with no observable terminal event cannot produce a fabricated completed record.

**Alternatives considered:** Unbounded asynchronous buffering risks memory growth and lost shutdown records; blocking operation success on diagnostic write failure contradicts behavior preservation.

### 8. Validation and documentation

**Decision:** Use existing `testx` HTTP servers, root execution patterns, and `testx.NewCmdTerminalRunner` for separate stdout/stderr with real terminal stdin. Cover `get process-definition` and `cancel process-instance`, auth variants, all supported version wiring, retries, body lifecycle, tracing, output modes and redaction. Run targeted tests before `make test`; update root metadata and README, then regenerate with `make docs-content`.

**Rationale:** Existing root, cancellation and terminal tests provide realistic invocation paths. Isolated rendering tests cannot prove existing debug resolution, bootstrap auth coverage, request counts or configured stderr routing.

**Alternatives considered:** Live mutation testing is not needed for default validation; deterministic local handlers can assert operation behavior without real resource changes. Pure mock-facade tests miss the transport entirely.
