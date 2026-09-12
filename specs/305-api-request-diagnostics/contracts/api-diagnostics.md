# CLI Contract: API Request Diagnostics

## Flag and scope

`--api-diagnostics` is a persistent boolean flag, default false, available on existing commands. It has no new configuration/environment setting. Commands with no HTTP exchanges emit no diagnostic records. Explicit diagnostics remain enabled with `--quiet`, independently of debug/verbose/log-format settings.

An observed exchange is one invocation at the shared instrumented RoundTripper boundary. HTTP retries, service retries and redirects are separate when they cross this boundary. Transport-internal retransmissions within a single call are not promised separate records. Authentication HTTP traffic is included; a token cache hit is not an HTTP exchange.

## Stream and record grammar

Records go only to the executing command's effective configured or inherited stderr, through the invocation's activity-aware writer. Existing stdout and result envelopes remain unchanged. Diagnostics are supplementary stderr text, not a new JSON result or log-format setting.

Each record is one newline-terminated line beginning `api: `, followed by space-separated `key=value` pairs in the order below. String values are always double-quoted with JSON-compatible escaping for quotes, backslashes and controls. Booleans and integer fields are unquoted; duration fields are quoted duration strings using Go duration units. Duration-list fields are quoted comma-separated duration strings, one sample per completed phase interval. Lists contain no missing samples or invented zeros.

Illustrative record (values are examples, not a timing target):

```text
api: timestamp="2026-09-12T12:00:00Z" sequence=1 method="POST" url="https://camunda.example.com/v2/process-instances/search" profile="production" tenant="customer-a" status=200 first-response="2.1s" total="2.2s" connection-reused=true request-bytes=128 request-complete=true response-bytes=256 response-complete=true
```

## Fields and semantics

| Field | Presence and meaning |
| --- | --- |
| timestamp | Required UTC RFC3339Nano exchange-start timestamp |
| sequence | Required positive invocation-local exchange sequence; completion/output order may differ |
| method | Request method, if available |
| url | Sanitized scheme, host, escaped path and safe query; no userinfo or fragment; retain path resource keys |
| profile, tenant | Resolved invocation profile/tenant when nonempty; tenant is selection context, not an assertion of resource ownership |
| status | Actual observed HTTP response status, including 4xx/5xx; absent without a response |
| error | Safe category: `canceled`, `timeout`, `dns`, `connect`, `tls`, `body-read`, `body-close` or `transport`; never arbitrary error text |
| timeout, canceled | True when established by the observed error/context at failure; absent otherwise; do not label an unrelated successful exchange from a later cancellation |
| first-response | Elapsed from boundary entry to the first response header byte, potentially an informational response; absent when no hook observed |
| total | Required elapsed from boundary entry to observed finalization, including body reading or early closure; excludes record formatting/writing |
| connection-reused | Boolean only when a connection callback supplies evidence |
| dns-durations | Completed DNS phase sample durations, ordered by start; omitted if none |
| connect-durations | Completed connection-attempt sample durations, ordered by start; may include overlapping attempts |
| tls-durations | Completed TLS handshake sample durations, ordered by start; omitted if none |
| request-bytes | Observed request-body bytes at finalization; zero only when bodyless or counting actually established |
| request-complete | Whether request body is known absent or EOF was observed by snapshot time; early Close is not completion proof |
| response-bytes | Observed response-body bytes; absent when no response-body observation exists |
| response-complete | Whether response is known bodyless or EOF was observed; false after early Close or read failure; absent if no response exists |
| request-id, correlation-id | Validated response `X-Request-ID`/`Request-ID`, `X-Correlation-ID`, respectively |
| client-request-id, client-correlation-id | Validated corresponding request headers; named separately so response values do not overwrite request values |
| retry-after | Validated response Retry-After seconds or normalized HTTP date |
| server-timing | Validated metric names and numeric durations only, separated by commas; omit free-form descriptions and unsupported parameters |

If both X-Request-ID and Request-ID exist, prefer X-Request-ID. For duplicate correlation header values, retain the first valid safe value. Header names are case-insensitive. Unsupported/invalid header values are omitted.

Total is observed elapsed workflow time at this boundary, including pauses between caller reads; it is not pure server execution or wire-transfer time. Phase durations can overlap and are not additive portions of total. Body byte counts exclude HTTP/TLS framing and may represent decoded content after automatic decompression. Declared sizes are not transferred sizes.

## Redaction policy

1. Construct a separate safe URL; never print `req.URL.String()` or raw malformed URL/error text. Remove userinfo and fragments. Preserve valid hosts, paths and operational identifiers unless they match known secrets.
2. Decode query names before classification; compare case-insensitively after separator normalization. Remove secret-bearing query entries, including authorization, auth, token/access-token/refresh-token/id-token, password/passwd, client-secret, cookie, API-key, signature/sig, and signed-URL credential/signature/security-token variants (including X-Amz and X-Goog families). Remove secret-bearing nested or encoded values when recognized. Malformed query components are omitted rather than printed raw.
3. Retain non-secret operational query parameters and repeated safe values, with deterministic key ordering. Exclude payload-bearing query values such as serialized variables/business payloads; do not dump arbitrary embedded objects. Security exclusions take precedence over identifier retention when a value is recognized as secret or payload.
4. Seed a private known-secret set from configured credentials, sensitive request header/URL userinfo/query values and sensitive response header values, including parsed Set-Cookie cookie values. Collect response secrets before sanitizing any allowed response correlation header. Apply decoded and standard URL-encoded secret matching to all retained strings, including allowed header values and profile/tenant labels; omit or replace secret occurrences with `[REDACTED]`. This set is never serialized and does not require reading a body.
5. Never record Authorization, Proxy-Authorization, cookies/Set-Cookie, token headers, passwords, client secrets or API keys. Only the correlation headers listed above are eligible. Validate correlation IDs as bounded (at most 256 characters), control-free identifier text and reject values that match known secrets or credential patterns. Validate Retry-After syntactically; parse Server-Timing and omit `desc` and unsupported free text. Omit malformed or suspicious allowed-header values.
6. Classify errors through typed/sentinel checks, not `err.Error()` string output. Unknown errors become `transport`; body read/close errors retain their phase unless timeout/cancellation is established. Do not expose payloads embedded in errors.
7. No request/response body contents or process variable values enter diagnostics. Enabling the flag never enables existing debug/body-dump logging. Existing independently enabled logging is outside this new record format.
8. Escape all string values after sanitization to prevent terminal control injection or forged extra diagnostic lines.

Validate explicit sensitive classes and known-secret reflection; do not claim semantic detection of arbitrary secrets disguised as unrelated identifiers. Unknown or malformed metadata that cannot be safely represented is omitted.

## Completion and preservation

- One record is attempted at transport failure, known bodyless response, EOF, read error or early Close, guarded against duplicate emission.
- Reused connections omit unperformed DNS/connect/TLS samples. Actual zero-duration observations may be retained.
- Incomplete response records report observed counts/duration with `response-complete=false`. Existing retry response closure does not trigger a drain.
- Counters include `n > 0` returned alongside errors. Preserve underlying Read/Close values and request replay behavior.
- No extra requests, buffering, timers, cancellation, retry-policy changes, body reads or body closes are introduced to collect diagnostics.
- Simultaneous exchanges produce indivisible lines. A failed writer cannot change request/body return values or command exit status; no fallback to stdout or write retry is added.
- A caller that never reads/closes an outstanding body has provided no completion evidence. Do not synthesize a completed record or force resource cleanup.

## Acceptance coverage

| Contract area | Required evidence |
| --- | --- |
| Flag/disabled/quiet | Root execution, inherited help, disabled baseline and quiet + diagnostics |
| Streams | Normal/JSON/keys-only, quiet combinations, configured leaf and inherited root stderr, real-terminal stdin |
| Lifecycle/timing | First byte before delayed body; EOF/partial/error/no-body; reuse and overlapping trace callbacks |
| Attempts | Retry, redirects, mutation attempts and auth requests with identical enabled/disabled request counts |
| Redaction | Encoded/repeated queries, signed URLs, headers, reflected secrets, payloads, errors and line-injection cases |
| Isolation | Parallel requests and independent invocations; no interleaved or misdirected records |
| Operational behavior | Read and cancellation paths, sparse/empty selection, confirmation/abort, auto-confirm/automation/no-wait, existing outcomes |
