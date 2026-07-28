# Feature Specification: Retry Transient Camunda Read Failures

**Feature Branch**: `261-retry-camunda-reads`

**Created**: 2026-07-28

**Status**: Draft

**Input**: User description: "GitHub issue #261: fix(http): retry transient Camunda read failures. Read-heavy commands such as `c8volt get pi --orphan-children-only --total --tenant <tenant>` can fail completely when Camunda or its platform dependencies return a transient server-side error for a single read request, for example a 500 response while resolving a parent process instance because the platform could not retrieve a JWK set before timing out. The requested fix should be pragmatic, central, generic for commands using the shared c8volt request path, similar to existing Camunda refusal/throttling retries, and should not introduce partial-results semantics, command-specific retry code, progress UX redesign, or unsafe mutation retries."

## Clarifications

### Session 2026-07-28

- Q: Which request scope should this issue retry? -> A: Retry only safe GET/HEAD reads for this issue.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Continue After Transient Read Failure (Priority: P1)

As an operator running a read-heavy c8volt command against a busy Camunda cluster, I want a transient platform read failure to be retried automatically so that one temporary backend issue does not abort the whole operation.

**Why this priority**: This directly addresses the observed customer problem where a single temporary server-side read failure stopped an otherwise valid diagnostic command.

**Independent Test**: Can be tested by simulating a read request that fails once with a transient server-side timeout and then succeeds; the command must continue and report the requested result.

**Acceptance Scenarios**:

1. **Given** a command is resolving many process-instance-related resources, **When** one read request receives a transient server-side timeout response and a later attempt succeeds, **Then** the command completes without surfacing the first transient failure as the final outcome.
2. **Given** orphan process-instance detection is checking parent process instances, **When** a parent lookup temporarily fails because the platform cannot complete token key lookup in time, **Then** c8volt retries the read and continues if the retry succeeds.

---

### User Story 2 - Preserve Business Errors (Priority: P2)

As an operator, I want expected business or selection outcomes to remain unchanged so that retry behavior does not hide real command results.

**Why this priority**: Some commands intentionally depend on semantic responses such as "not found" to determine the correct result.

**Independent Test**: Can be tested by simulating non-transient read outcomes and verifying that c8volt does not retry or reinterpret them.

**Acceptance Scenarios**:

1. **Given** orphan detection checks a missing parent resource, **When** the read returns a not-found result, **Then** c8volt treats it as the expected orphan signal instead of retrying it.
2. **Given** a read request is rejected because of invalid input, missing permissions, or a conflict, **When** the response is non-transient, **Then** c8volt reports the existing error behavior without retrying until timeout.

---

### User Story 3 - Keep Mutations Safe (Priority: P3)

As an operator using destructive or state-changing commands, I want generic retry behavior to avoid replaying unsafe operations so that automatic retries do not accidentally duplicate mutations.

**Why this priority**: The customer issue is read-side reliability; extending generic retries to mutations would increase operational risk and duplicate existing mutation-specific retry behavior.

**Independent Test**: Can be tested by simulating transient failures for state-changing requests and verifying that the generic read retry behavior does not replay them.

**Acceptance Scenarios**:

1. **Given** a destructive command issues a state-changing request, **When** that request receives a transient server-side response, **Then** the generic read retry behavior does not replay the mutation.
2. **Given** existing mutation retry behavior already handles a known Camunda refusal or throttling case, **When** that case occurs, **Then** the existing mutation-specific behavior remains the mechanism responsible for retrying it.

### Edge Cases

- A retryable read keeps failing until the retry budget is exhausted; c8volt must report the final failure clearly and preserve the original diagnostic context.
- The user cancels the command while retries are waiting; c8volt must stop promptly instead of continuing background retry attempts.
- A retryable response includes a body that existing error handling needs; c8volt must preserve the final response details for current error mapping and diagnostics.
- A large batch command encounters several transient read failures; retry logging must remain compact and must not flood normal operator output.
- A read-like search request fails transiently; c8volt must not retry it as part of this issue because the scope is limited to safe GET/HEAD reads.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: c8volt MUST automatically retry transient failures for safe GET and HEAD read operations used by Camunda-facing commands.
- **FR-002**: c8volt MUST allow a read-heavy command to continue when a transient read failure is followed by a successful retry.
- **FR-003**: c8volt MUST treat server-side timeout and temporary availability failures as retryable read failures.
- **FR-004**: c8volt MUST keep expected non-transient outcomes, including not-found results, invalid requests, permission failures, and conflicts, final without retrying them as transient failures.
- **FR-005**: c8volt MUST avoid generic automatic retries for unsafe state-changing operations.
- **FR-006**: c8volt MUST preserve the existing final error details when all retry attempts are exhausted.
- **FR-007**: c8volt MUST stop retrying promptly when the user cancels the command or the command context ends.
- **FR-008**: c8volt MUST emit compact retry information that follows established c8volt operator logging style.
- **FR-009**: c8volt MUST preserve existing command output contracts for human, keys-only, and structured modes except for the additional compact retry information when a retry actually happens.
- **FR-010**: c8volt MUST keep the solution generic for commands using the shared Camunda GET/HEAD request path, without adding command-specific retry branches for the observed command.
- **FR-011**: c8volt MUST NOT retry search or other read-like non-GET/HEAD requests as part of this issue.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In a simulated read-heavy command where one safe GET or HEAD read fails once with a transient server-side timeout and then succeeds, the command completes successfully in 100% of test runs.
- **SC-007**: In tests covering search or other non-GET/HEAD requests, c8volt performs zero generic read-layer retries.
- **SC-002**: In tests covering not-found, invalid request, permission, and conflict responses, c8volt performs zero generic retries for those outcomes.
- **SC-003**: In tests covering unsafe state-changing requests, c8volt performs zero generic read-layer retries.
- **SC-004**: When retry attempts are exhausted, the final error shown to the operator retains the response status and diagnostic detail needed to troubleshoot the failure.
- **SC-005**: Retry logging for repeated transient read failures remains compact enough that one affected request produces no more than one normal operator-facing retry line per retry attempt.
- **SC-006**: Existing command tests for process-instance querying, orphan detection, and mutation retry behavior continue to pass.

## Assumptions

- The customer problem is caused by transient Camunda or platform dependency failures during read requests, not by invalid user input.
- Safe GET and HEAD retries are acceptable because they do not change Camunda state.
- Existing mutation-specific retry behavior remains the preferred place for state-changing retry semantics.
- Partial results, degraded totals, and continue-on-error batch semantics are intentionally outside this issue.
- Search and other read-like non-GET/HEAD retry behavior is intentionally deferred.
- The implementation should prefer the existing shared request and logging patterns already used by c8volt.
