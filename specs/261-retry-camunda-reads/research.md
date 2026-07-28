# Research: Retry Transient Camunda Read Failures

## Decision: Implement retry behavior at the shared HTTP client layer

**Rationale**: `cmd/root.go` creates one `httpc.Service`, `c8volt/client.go` passes its `*http.Client` into all public facades and internal services, and generated Camunda clients use that client for requests. A retrying transport in this path covers command families generically without per-command code.

**Alternatives considered**: Adding retries inside process-instance orphan detection was rejected because it solves only the observed command. Adding retries inside every versioned service was rejected because it duplicates behavior across v87/v88/v89 adapters. Adding partial-result handling was rejected because the customer problem is a transient read failure, not a requirement to continue after persistent bad resources.

## Decision: Limit this issue to GET and HEAD requests

**Rationale**: The clarification session selected GET/HEAD only. This directly solves the observed parent process-instance lookup failure and avoids request-body replay complexity for search requests. It also prevents accidental mutation replay at the generic HTTP layer.

**Alternatives considered**: Retrying replayable search requests was rejected for this issue because it adds body replay and endpoint classification work. Retrying all read-intent requests was rejected because intent cannot be inferred safely at the HTTP method level.

## Decision: Mirror existing Camunda retry timing style

**Rationale**: `internal/services/retry.go` already establishes repository-native retry behavior: bounded attempts, exponential backoff, jitter, `Retry-After` support, context-aware sleep, and rate-limited compact log lines. Reusing this behavior keeps operator experience consistent.

**Alternatives considered**: Immediate retry loops were rejected because they can amplify platform pressure. A new configurable retry subsystem was rejected as overdesigned for the current customer issue. Silent retries were rejected because operators benefit from knowing why a long read command paused.

## Decision: Retry transient read failures before final HTTP error mapping

**Rationale**: `internal/services/httpc/httpmap.go` maps final HTTP responses to domain errors, and service helpers rely on the final response body for useful diagnostics. The retry layer must run before that mapping and preserve the final response body when it gives up.

**Alternatives considered**: Retrying after domain error mapping was rejected because every caller would need to preserve response/status/body context. Dropping the response body after inspection was rejected because it would degrade existing error messages.

## Decision: Treat semantic client outcomes as final

**Rationale**: Commands such as orphan detection depend on not-found responses to identify missing parents. Invalid request, permission, not-found, and conflict outcomes represent command or authorization semantics and should not be delayed by retry loops.

**Alternatives considered**: Retrying all server and client failures was rejected because it would hide meaningful operator feedback and slow down deterministic failures.

## Decision: Keep retry logs compact and off machine stdout

**Rationale**: c8volt already protects JSON, keys-only, quiet, and automation output contracts. Retry information should use the existing logger/activity path and follow the established `Camunda ...; retrying in ...` style without adding structured output fields in this issue.

**Alternatives considered**: Printing retry messages to stdout was rejected because it breaks machine output. Adding progress records to JSON was rejected because the feature is internal resilience, not a new output schema.
