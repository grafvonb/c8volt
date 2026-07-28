# Contract: HTTP GET/HEAD Read Retry

## Scope

This contract describes behavior visible to c8volt commands that use the shared Camunda HTTP client.

In scope:

- Camunda-facing GET requests
- Camunda-facing HEAD requests
- Transient transport failures for those methods
- Transient server or throttling responses for those methods

Out of scope:

- POST search requests
- Other read-like non-GET/HEAD requests
- DELETE, PATCH, PUT, and mutation-style POST requests
- Partial-result or continue-on-error batch semantics
- New command flags or output schemas

## Retryable Outcomes

A GET/HEAD request may be retried when the outcome is transient:

- temporary or timeout network error
- 429 Too Many Requests
- 500 Internal Server Error
- 502 Bad Gateway
- 503 Service Unavailable
- 504 Gateway Timeout
- server error body indicating temporary platform dependency failure, including JWK lookup timeout wording

## Non-Retryable Outcomes

The generic read retry layer must not retry:

- 400 Bad Request
- 401 Unauthorized
- 403 Forbidden
- 404 Not Found
- 409 Conflict
- any non-GET/HEAD request
- any canceled request context

## Operator Output

When a retry is attempted in normal human output, c8volt may emit one compact log line per retry attempt, subject to the same rate-limiting style as existing Camunda retry logs.

Example shape:

```text
INFO Camunda read failed loading process instance; 500 Internal Server Error; retrying in 725ms
```

The exact operation label should reuse existing activity wording when available.

## Machine Output

The retry layer must not write transient retry information to stdout.

- JSON output remains one valid JSON document.
- Keys-only output remains one key per line.
- Quiet mode remains quiet except for final errors.
- Automation mode remains deterministic.

## Final Error Contract

If all attempts fail, c8volt must preserve the final diagnostic behavior used today:

- final status code is available to existing error mapping
- final method and URL remain available to existing error messages
- final response body remains readable by existing generated-client/service code
- final transport error remains available when no response exists

## Cancellation Contract

If the command context is canceled while waiting between attempts, retry stops promptly and returns the cancellation result instead of continuing to call Camunda.
