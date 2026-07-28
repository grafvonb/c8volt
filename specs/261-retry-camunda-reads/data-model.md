# Data Model: Retry Transient Camunda Read Failures

This feature does not introduce persisted data. The model describes runtime concepts used to reason about behavior and tests.

## Retry Policy

Represents the bounded rules for automatic retry.

- **request_methods**: GET and HEAD only
- **attempt_budget**: finite maximum attempts
- **base_delay**: initial retry delay
- **max_delay**: upper bound for exponential backoff
- **jitter**: randomized delay addition to avoid synchronized retry bursts
- **retry_after**: optional server-provided delay override
- **log_interval**: minimum interval for repeated normal retry log lines with the same operation label

Validation rules:

- Attempt budget must be at least one.
- Delays must be non-negative.
- Policy must not include non-GET/HEAD methods for this issue.

## Retryable Request

Represents one outgoing Camunda read request that may be retried.

- **method**: HTTP method, retryable only when GET or HEAD
- **url_path**: request path used for activity wording and diagnostics
- **context**: command context used for cancellation and deadlines
- **operation_label**: compact human label derived from existing HTTP activity wording when available

Validation rules:

- Requests with methods other than GET or HEAD are non-retryable.
- Requests must preserve the same URL, headers, context, and auth behavior across attempts.
- Canceled contexts stop retry immediately.

## Retryable Failure

Represents a transient failure that can trigger another attempt.

- **transport_error**: temporary or timeout network error
- **status_code**: retryable server or throttling status
- **response_body_hint**: optional diagnostic text, for example JWK lookup timeout details
- **retry_after**: optional response header that can influence delay

Validation rules:

- Not-found responses are non-retryable.
- Invalid request, permission, and conflict responses are non-retryable.
- Mutation responses are non-retryable in the generic read layer even when status looks transient.

## Retry Attempt

Represents one retry decision after a failed attempt.

- **attempt_number**: one-based attempt number
- **reason**: compact reason such as temporary network error, rate limited, unavailable, or internal server error
- **delay**: wait before the next attempt
- **will_retry**: whether another attempt will be made

Validation rules:

- Attempt number cannot exceed the configured attempt budget.
- Delay must respect context cancellation.
- Normal operator logs should remain compact.

## Final Outcome

Represents the response or error returned to existing generated-client and service code after retry handling finishes.

- **success_response**: final successful response when a retry succeeds
- **final_failure_response**: final response when attempts are exhausted
- **final_transport_error**: final transport error when no response is available
- **body_preserved**: whether the final response body remains readable for existing error mapping

Validation rules:

- Successful retry returns the successful response to the existing caller.
- Exhausted retry returns the final failure without losing status, URL, method, or body detail.
- Cancellation returns the context cancellation result promptly.
