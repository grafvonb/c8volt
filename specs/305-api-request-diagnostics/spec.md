# Feature Specification: Opt-in API Request Diagnostics

**Feature Branch**: `codex/305-api-request-diagnostics`

**Created**: 2026-09-12

**Status**: Draft

**Input**: [GitHub issue #305](https://github.com/grafvonb/c8volt/issues/305), “feat(cli): add opt-in API request diagnostics”

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Investigate a slow command (Priority: P1)

As an operator, I can enable `--api-diagnostics` on an existing API-backed command and identify the actual requests that contribute to its runtime, so I can correlate evidence with Camunda logs and infrastructure metrics.

**Why this priority**: Request-level evidence is the core capability needed to investigate slow or timed-out operations.

**Independent Test**: Run an existing read command against controlled responses with different response-start and body-transfer delays. Verify that its diagnostics distinguish these delays and identify the requests while its result remains unchanged.

**Acceptance Scenarios**:

1. **Given** an existing API-backed command, **When** the operator adds `--api-diagnostics`, **Then** each observed HTTP exchange produces one diagnostic record on configured stderr, with available request identity, context, outcome, timing and size information.
2. **Given** a response whose headers arrive before its body finishes, **When** the body completes, **Then** first-response timing measures the interval from exchange start to the first response byte, and total duration measures the interval from exchange start through body reading completion.
3. **Given** a workflow that retries a request, **When** diagnostics are enabled, **Then** each attempt appears as a separate exchange with an invocation-local sequence identifier.
4. **Given** a command run without the flag, **When** it completes or fails, **Then** it emits no API diagnostic records and retains existing behavior and output.

---

### User Story 2 - Use diagnostics safely in operational workflows (Priority: P1)

As an operator running commands interactively or in automation, I can collect useful diagnostic evidence without exposing credentials or contaminating command results.

**Why this priority**: Diagnostics must be safe to enable on real workflows and compatible with existing consumers of command output.

**Independent Test**: Exercise one existing read command and one existing mutation command with seeded secrets and operational identifiers, capturing stdout and stderr separately across supported output modes.

**Acceptance Scenarios**:

1. **Given** normal, JSON or keys-only output, **When** diagnostics are enabled, **Then** stdout and exit behavior match the same invocation without diagnostics; records appear only on configured or inherited stderr.
2. **Given** `--quiet --api-diagnostics`, **When** exchanges occur, **Then** diagnostics remain visible on stderr while established quiet and explicitly selected result-output behavior is preserved.
3. **Given** credentials, secret query parameters, sensitive header values or secrets in errors, **When** records are emitted, **Then** those secrets are absent, while safe hosts, paths, resource keys, profile identities, tenant IDs, non-secret query parameters and correlation IDs remain visible.
4. **Given** request and response bodies contain business data, **When** diagnostics are enabled, **Then** body contents and process variable values never appear in records.
5. **Given** an existing mutation workflow, **When** diagnostics are enabled, **Then** its requests, confirmation behavior, operational verification and outcome remain unchanged, with no extra requests or mutations.

---

### User Story 3 - Interpret failures and partial evidence accurately (Priority: P2)

As an operator, I can distinguish failed, canceled, incomplete and concurrent exchanges without mistaking missing measurements for completed work.

**Why this priority**: Accurate limits on the evidence prevent misleading investigations under the conditions most likely to need diagnostics.

**Independent Test**: Exercise reused and fresh connections, transport failures, timeouts, cancellation, early body closure and concurrent requests; compare each record with the controlled exchange outcome.

**Acceptance Scenarios**:

1. **Given** a reused connection with no DNS lookup or TLS handshake, **When** its record is emitted, **Then** connection reuse is reported and unperformed phase timings are omitted rather than reported as zero.
2. **Given** a transport failure, timeout or cancellation, **When** the exchange terminates, **Then** the record identifies the observed failure and available timing information without inventing a response status or unavailable measurements.
3. **Given** a response body is closed early or fails during reading, **When** its record is emitted, **Then** observed duration and transferred size are identified as incomplete rather than a complete transfer.
4. **Given** concurrent exchanges, **When** their records are emitted, **Then** every record is complete, distinguishable by sequence and free of interleaved text.
5. **Given** separate invocations with different context and stderr destinations, **When** both collect diagnostics, **Then** each invocation receives only its own records and context.

### Edge Cases

- An invocation that performs no HTTP exchanges emits no exchange records and performs no diagnostic requests.
- Failed connections may have no response status or first-response timing; only observed data is included.
- Empty response bodies can complete successfully with zero observed body bytes; unknown sizes remain absent.
- Early closure, body-read errors, timeout and cancellation after a response starts preserve partial evidence and an incomplete-transfer indication.
- Retries and concurrent exchanges remain distinguishable even when they share the same method and destination; sequence identifies exchanges, not a promise of completion order.
- URL credentials, encoded secret query values and sensitive content reflected in errors or allowed headers require the same protection as direct credentials.
- Unavailable profile, tenant, correlation or phase information is omitted, never fabricated.
- Redirected stdout, inherited stderr, quiet mode and existing interactive controls retain their established behavior.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Every existing API-backed command MUST accept the global opt-in `--api-diagnostics` flag. Diagnostics MUST be disabled by default.
- **FR-002**: An enabled invocation MUST emit one complete diagnostic record per observed HTTP exchange, including separate records for retry attempts, with an invocation-local sequence identifier.
- **FR-003**: Records MUST include, when available, request timestamp, method, host, path and path resource identifiers, active profile and tenant, non-secret query parameters, response status or transport error, timeout or cancellation, connection reuse, request and response sizes, and safe correlation information including request ID, Retry-After and Server-Timing.
- **FR-004**: Records MUST distinguish elapsed time to the first response byte from total elapsed exchange time including response-body reading, and include DNS, connection and TLS handshake durations only when those phases were performed and observed.
- **FR-005**: Unavailable information MUST be omitted. A body closed before completion or interrupted by failure MUST report only observed duration and size and explicitly indicate an incomplete transfer.
- **FR-006**: Diagnostics MUST use only the current command's configured or inherited stderr destination, remain effective with explicit `--quiet`, and leave normal, JSON and keys-only stdout and existing exit behavior unchanged.
- **FR-007**: Records MUST retain safe operational identifiers while excluding authorization headers, access and refresh tokens, client secrets, passwords, cookies, API keys, embedded URL credentials, signed URL secrets and other secret-bearing query values.
- **FR-008**: The same redaction rules MUST protect error messages and allowed header values. Arbitrary headers, request or response body contents, process variable values and business payloads MUST never be recorded.
- **FR-009**: Enabling diagnostics MUST preserve request semantics, retry behavior, timeouts, cancellation, response consumption, confirmation behavior and operational outcome verification. It MUST issue no additional requests and MUST NOT buffer or consume payloads solely for diagnostics.
- **FR-010**: Concurrent exchanges MUST produce complete records without interleaved text. Diagnostic collection, context and output MUST remain scoped to the current invocation without leakage between invocations.
- **FR-011**: Command help and user documentation MUST explain opt-in usage, output destination, quiet-mode interaction, timing meanings, incomplete transfers, retained identifiers and excluded sensitive information.

### Key Entities *(include if feature involves data)*

- **Invocation context**: The active command run, its profile and tenant when available, diagnostic enablement and stderr destination.
- **HTTP exchange**: One actual request attempt and its observed response or failure, including lifecycle timing and transferred sizes; retries are distinct exchanges.
- **Diagnostic record**: A redacted, indivisible representation of one exchange, associated with its invocation and sequence, with explicit limits where transfer evidence is incomplete.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In controlled successful, failed, timed-out, canceled and retried workflows, 100% of observed terminated exchanges produce exactly one distinguishable record when enabled and zero records when disabled.
- **SC-002**: Operators can distinguish response-start delay from body-transfer delay, connection reuse from newly observed connection phases, and complete from incomplete transfers in every corresponding acceptance scenario without consulting payload contents.
- **SC-003**: For one existing read command and one existing mutation command, enabling diagnostics produces byte-for-byte equivalent stdout and identical exit outcomes across supported normal, JSON, keys-only and quiet combinations, while records reach the selected stderr destination.
- **SC-004**: All seeded credentials, secrets and payload contents are absent from diagnostic output, while all available safe operational identifiers in the redaction acceptance cases remain usable for correlation.
- **SC-005**: Enabled and disabled comparison runs issue identical request attempts and preserve retry and response-consumption behavior; concurrent and separate-invocation cases produce zero interleaved or misdirected records.

## Assumptions

- The intended users are operators investigating actual command workflows; this feature provides evidence, not automatic root-cause conclusions.
- Sizes describe observed transfer evidence; unknown values are omitted, and partial values are not presented as complete payload sizes. Precise field naming and size-unit conventions will be documented during planning.
- Sequence numbers distinguish exchanges within one invocation. Cross-invocation persistence and retry-group aggregation are not required.
- The issue's example record illustrates content rather than a mandated machine-readable schema; command result models remain unchanged.
- Planning must preserve the source issue's explicit delivery constraints: implement observation once at the shared HTTP transport boundary, use no mutable global output destination, and add no diagnostic-specific logic to individual commands or generated Camunda clients. These are implementation constraints, not new user-facing capabilities.
- Delivery validation must cover timing, reuse, concurrency, failures, retries, cancellation, redaction and output separation, including real execution paths for an existing read and mutation command. Command metadata and relevant user documentation must be updated and CLI documentation regenerated with `make docs-content`.
- Dedicated latency-analysis commands, synthetic or repeated diagnostic requests, aggregate statistics or percentiles, automatic analysis, report files, history, resource creation or cleanup, monitoring integrations and command result-model changes are outside scope.
