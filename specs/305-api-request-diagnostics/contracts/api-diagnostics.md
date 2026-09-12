# CLI Contract: API Request Diagnostics

## Flag and scope

Use the existing `--verbose` opt-in and invocation `*slog.Logger` at INFO level through the established verbose logging helper. There is no new flag, configuration key or separate diagnostic output mode. Quiet and configured levels filter records normally, including failure diagnostics; do not promote them to ERROR to bypass filtering. Without verbose, emit no diagnostic records even if debug is enabled. Commands with no HTTP exchanges emit none.

An observed exchange is one invocation at the shared instrumented RoundTripper boundary. HTTP retries, service retries and redirects are separate when they cross this boundary. Transport-internal retransmissions within a single call are not promised separate records. Authentication HTTP traffic is included; a token cache hit is not an HTTP exchange.

## Stream and record grammar

Records go through the existing invocation logger and activity-aware writer to the executing command's effective configured or inherited `cmd.ErrOrStderr()`. Existing stdout and result envelopes remain unchanged. Prompts remain plain text on configured stderr without logger prefixes. Never write diagnostics directly to process stderr/stdout or a separate collector-owned output sink.

Each admitted record is emitted once as an INFO log message using existing logging features. The logger owns timestamp, level, source and plain/plain-time/text/JSON framing. The following grammar defines the message, not the entire serialized log line. Keep stable space-separated field order; omit unavailable fields.

```text
api #<sequence> <METHOD> <safe-path-and-query>: status=<code> error=<failure> total=<duration> headers=<duration> body=<duration> phase=<phase> reason=<reason> conn=<new|reused> dns=<duration> tcp=<duration> tls=<duration> <secondary metadata>
```

The identity and `total` are required; other fields depend on observations. Emit status only when a response exists, error only on failure, and both when a body fails after headers. Unknown method or target uses `?`, never raw malformed input. Text tokens containing whitespace, controls, quotes or backslashes use JSON-compatible quoted escaping. Grammar punctuation and duration units are ASCII; use `us` instead of the Unicode microsecond symbol, with no color or alignment-padding dependency. Single completed phase samples are scalars (`tcp=24ms`); multiple samples use brackets (`tcp=[12ms,24ms]`) in start order without summing overlap. Integers and booleans are unquoted.

Core log examples (secondary metadata omitted here for readability; values are illustrative):

```text
api #45 GET /v2/topology: status=200 total=180ms headers=170ms body=10ms dns=8ms tcp=24ms tls=61ms
api #46 POST /v2/process-instances/search: status=200 total=2.24s headers=2.1s body=140ms conn=reused
api #47 GET /v2/topology: error=TIMEOUT total=30s phase=response-headers conn=reused
api #48 GET /v2/topology: error=CONNECT_ERROR total=12ms phase=connect reason=connection-refused
api #49 GET /v2/topology: status=200 error=TIMEOUT total=2s headers=100ms body=1.9s phase=body conn=reused response-complete=false
```

A complete bodyless record with secondary metadata:

```text
api #50 HEAD /v2/topology: status=200 total=20ms headers=20ms body=0s conn=reused host=camunda.example.com profile=production tenant=customer-a request-bytes=0 request-complete=true response-bytes=0 response-complete=true
```

## Fields and semantics

| Field | Presence and meaning / low-level source |
| --- | --- |
| #sequence | Required positive invocation-local atomic sequence; completion/output order may differ |
| METHOD, safe-path-and-query | Request method and separately sanitized escaped URL path/query, retaining safe resource identifiers; host appears in metadata |
| status | Actual returned HTTP status, including 4xx/5xx; absent without a response |
| error | Classified failure, whether before headers or alongside an actual status after headers |
| total | Delegate RoundTrip start (`t0`) to observed terminal event (`te`); excludes formatting/writing |
| headers | `th-t0`, where `th` is captured immediately when delegate returns final response headers; omitted without a response |
| body | `te-th`: final headers through EOF, non-EOF read error or early Close; known bodyless response has `te=th`, `body=0s` |
| phase | Observed failure phase: `dns`, `connect`, `tls`, `request-write`, `response-headers`, `body`; omit when attribution is unavailable or ambiguous |
| reason | Optional safe bounded enum: `connection-refused`, `connection-reset`, `unexpected-eof`, `certificate-invalid`, `read-failed`, `close-failed`; typed/sentinel evidence only |
| conn | `new` or `reused` from httptrace GotConn.Reused; absent without callback evidence |
| dns | Paired httptrace DNSStart/DNSDone durations |
| tcp | Paired ConnectStart/ConnectDone durations for TCP networks only; pair by network/address internally |
| tls | Paired TLSHandshakeStart/TLSHandshakeDone durations |
| timestamp | Provided by the existing logger at emission using its configured format; no duplicate exchange-start timestamp in the message |
| host | Sanitized URL authority including explicit port, when available; no userinfo |
| profile, tenant | Resolved invocation context when nonempty; tenant is selection context, not resource ownership |
| request-bytes, request-complete | Actual observed request-body bytes and whether known bodyless or EOF observed at snapshot time; early Close is not completion proof |
| response-bytes, response-complete | Actual observed response-body bytes and known bodyless/EOF evidence; incomplete is false, absent without a response |
| request-id, correlation-id | Validated response X-Request-ID/Request-ID and X-Correlation-ID |
| client-request-id, client-correlation-id | Validated corresponding request headers, kept separate from response values |
| retry-after | Validated response seconds or normalized HTTP date |
| server-timing | Validated metric names and numeric durations only, comma-separated; exclude descriptions and unsupported parameters |

Secondary message metadata follows the table order from host onward. The collector keeps monotonic exchange-start time for durations; logger time describes emission. Byte counts are bytes, not declared Content-Length. Omit unknown counts; zero requires actual counting or known absence of a body.

Failure classifications are `CANCELED`, `TIMEOUT`, `DNS_ERROR`, `CONNECT_ERROR`, `TLS_ERROR`, `BODY_ERROR`, or `TRANSPORT_ERROR`. Cancellation/timeout classification takes precedence when established by the failure. HTTP error status alone is not a transport error. Preserve an actual status alongside a body failure; routine early Close only sets incomplete evidence, not BODY_ERROR.

Derive phase from typed error evidence, observed body failure, or unambiguous trace lifecycle: paired DNS/connect/TLS callbacks, WroteRequest and GotConn. A failed write can establish request-write; a successful WroteRequest with no returned final headers and no conflicting active setup evidence can establish response-headers. Do not simply use the last callback when attempts overlap, or infer a phase when a custom transport supplies no evidence. Later context cancellation must not relabel a successful exchange.

If both X-Request-ID and Request-ID exist, prefer X-Request-ID. For duplicate correlation header values, retain the first valid safe value. Header names are case-insensitive. Unsupported/invalid header values are omitted.

All elapsed measurements use monotonic time. Before display rounding, total=headers+body when headers exist. Headers includes connection acquisition/setup, request sending and waiting for final headers; it is not server processing time. GotFirstResponseByte is deliberately not used for this boundary, since informational responses or partial headers can precede final headers. Body includes pauses between caller reads and is not pure wire-transfer time. DNS/TCP/TLS durations are already within the observed exchange; they can overlap and are not additive portions of total. Body byte counts exclude HTTP/TLS framing and may represent decoded content after automatic decompression. Declared sizes are not transferred sizes.

## Redaction policy

1. Construct a separate safe URL; never print `req.URL.String()` or raw malformed URL/error text. Remove userinfo and fragments. Preserve valid hosts, paths and operational identifiers unless they match known secrets.
2. Decode query names before classification; compare case-insensitively after separator normalization. Remove secret-bearing query entries, including authorization, auth, token/access-token/refresh-token/id-token, password/passwd, client-secret, cookie, API-key, signature/sig, and signed-URL credential/signature/security-token variants (including X-Amz and X-Goog families). Remove secret-bearing nested or encoded values when recognized. Malformed query components are omitted rather than printed raw.
3. Retain non-secret operational query parameters and repeated safe values, with deterministic key ordering. Exclude payload-bearing query values such as serialized variables/business payloads; do not dump arbitrary embedded objects. Security exclusions take precedence over identifier retention when a value is recognized as secret or payload.
4. Seed a private known-secret set from configured credentials, sensitive request header/URL userinfo/query values and sensitive response header values, including parsed Set-Cookie cookie values. Collect response secrets before sanitizing any allowed response correlation header. Apply decoded and standard URL-encoded secret matching to all retained strings, including allowed header values and profile/tenant labels; omit or replace secret occurrences with `[REDACTED]`. This set is never serialized and does not require reading a body.
5. Never record Authorization, Proxy-Authorization, cookies/Set-Cookie, token headers, passwords, client secrets or API keys. Only the correlation headers listed above are eligible. Validate correlation IDs as bounded (at most 256 characters), control-free identifier text and reject values that match known secrets or credential patterns. Validate Retry-After syntactically; parse Server-Timing and omit `desc` and unsupported free text. Omit malformed or suspicious allowed-header values.
6. Classify errors through typed/sentinel checks, not `err.Error()` string output. Unknown transport errors become `TRANSPORT_ERROR`; body errors become `BODY_ERROR` unless timeout/cancellation is established. Preserve phase separately and omit an unsupported reason. Do not expose payloads embedded in errors.
7. No request/response body contents or process variable values enter diagnostics. Verbose diagnostics never enable existing debug/body-dump logging. Existing independently enabled logging is outside this new record format.
8. Apply the token escaping rules to all strings after sanitization to prevent terminal control injection or forged extra diagnostic lines.

Validate explicit sensitive classes and known-secret reflection; do not claim semantic detection of arbitrary secrets disguised as unrelated identifiers. Unknown or malformed metadata that cannot be safely represented is omitted.

## Completion and preservation

- When verbose and INFO are enabled, one logger emission is attempted at transport failure, known bodyless response, EOF, read error or early Close, guarded against duplicate emission.
- Reused connections omit unperformed DNS/connect/TLS samples. Actual zero-duration observations may be retained.
- Incomplete response records report observed counts/duration with `response-complete=false`. Existing retry response closure does not trigger a drain.
- Counters include `n > 0` returned alongside errors. Preserve underlying Read/Close values and request replay behavior.
- No extra requests, buffering, timers, cancellation, retry-policy changes, body reads or body closes are introduced to collect diagnostics.
- Simultaneous exchanges use the existing synchronized logging writer and produce indivisible records. A failed writer cannot change request/body return values or command exit status; no fallback to stdout or write retry is added.
- A caller that never reads/closes an outstanding body has provided no completion evidence. Do not synthesize a completed record or force resource cleanup.

## Acceptance coverage

| Contract area | Required evidence |
| --- | --- |
| Flag/disabled/quiet | Existing verbose gating, disabled/debug-only baseline, quiet suppression, configured levels and all existing log formats |
| Streams | Normal/JSON/keys-only, quiet combinations, configured leaf and inherited root stderr, real-terminal stdin |
| Lifecycle/timing | Final headers after informational responses and before delayed body; total=headers+body; failure-phase evidence, EOF/partial/error/no-body; reuse and overlapping trace callbacks |
| Attempts | Retry, redirects, mutation attempts and auth requests with identical enabled/disabled request counts |
| Redaction | Encoded/repeated queries, signed URLs, headers, reflected secrets, payloads, errors and line-injection cases |
| Isolation | Parallel requests and independent invocations; no interleaved or misdirected records |
| Operational behavior | Read and cancellation paths, sparse/empty selection, confirmation/abort, auto-confirm/automation/no-wait, existing outcomes |

## Existing logger integration

Build the safe compact content as the message passed to the existing verbose INFO helper. PlainHandler ignores structured attributes, so do not place required diagnostic fields exclusively in slog attributes or redesign the shared handler. In JSON log format, the same message is in the logger's existing `msg` field; this is stderr logging, separate from the stdout command JSON envelope. Default plain-time example:

```text
12:34:56.789 INFO api #45 GET /v2/topology: status=200 total=180ms headers=170ms body=10ms dns=8ms tcp=24ms tls=61ms
```
