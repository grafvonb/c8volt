# Feature Specification: Explain Known Connection and Authentication Failures

**Feature Branch**: `381-explain-connection-failures`

**Created**: 2026-11-18

**Status**: Draft

**Input**: [GitHub issue #381](https://github.com/grafvonb/c8volt/issues/381), “fix(cli): explain known connection and authentication failures”. Operators need ordinary command errors to explain recognized causes and suggest a useful next check without requiring DEBUG.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Understand a failed connection immediately (Priority: P1)

As an operator, I want a failed command to identify a recognized connection or authentication problem and a useful next check, so I can investigate without rerunning it with DEBUG.

**Why this priority**: The original failure hides an authentication-service connection refusal behind the unhelpful class `internal`.

**Independent Test**: Execute process-definition search with an isolated authentication failure and compare its human diagnostic with the recognized cause. Exercise other supported causes with controlled failure evidence.

**Acceptance Scenarios**:

1. **Given** a process-definition search whose authentication service refuses the connection, **When** the command fails in normal human mode, **Then** its error is `search process definitions: authentication service connection refused; check the configured host and port and that the service is running`, without requiring DEBUG.
2. **Given** a recognized gateway or authentication-service timeout, DNS failure, certificate validation failure, refused or interrupted connection, invalid response, or supported service rejection, **When** a command uses shared human-error formatting, **Then** its explanation identifies the supported cause and next check, distinguishing the authentication service from the gateway when the evidence supports that distinction.
3. **Given** an authentication failure with no more specific recognized cause, **When** its human explanation is produced, **Then** it directs the operator to check authentication configuration without inventing a cause.
4. **Given** no recognized connection or authentication evidence, **When** a human explanation is produced, **Then** the existing class fallback remains unchanged, even if arbitrary error text contains phrases such as `connection refused` or `HTTP 401`.

---

### User Story 2 - Receive safe diagnostics through the configured logger (Priority: P1)

As an operator, I want useful errors to follow my logging configuration and exclude sensitive transport details, so I can collect them safely without contaminating command results.

**Why this priority**: Clearer wording must preserve diagnostic routing and confidentiality.

**Independent Test**: Execute controlled single-resource and repeated multi-resource failures with separate stdout, stderr, and configured logger captures across normal, verbose, quiet, DEBUG, and structured logging modes.

**Acceptance Scenarios**:

1. **Given** a configured or inherited diagnostic destination and plain or structured logger format, **When** a recognized failure is reported, **Then** exactly one final ERROR record follows that configuration and stdout contains no human diagnostic.
2. **Given** repeated backend failures containing distinctive credentials, URL, method, body, and resource key, **When** normal or verbose human output is selected, **Then** the existing bounded batch summary and counts are retained, with a bounded cause where shared formatting applies, and neither output stream exposes raw transport details or credentials.
3. **Given** quiet mode, **When** a recognized failure occurs, **Then** the compact ERROR diagnostic remains while informational output is suppressed.
4. **Given** DEBUG enabled through either `--debug` or configured debug logging, **When** the same controlled failure occurs, **Then** existing redacted diagnostic evidence remains available through the configured logger; verbose alone does not enable it and stdout remains free of diagnostic records.

---

### User Story 3 - Preserve operational and automation contracts (Priority: P2)

As an existing user, I want clearer human explanations without changes to execution or machine results, so scripts and established troubleshooting guidance keep working.

**Why this priority**: This correction changes an explanation rather than the meaning or execution of an operation.

**Independent Test**: Compare controlled failures with baseline expectations for machine results, exit status, request counts, retries, cancellation, request-rate waits, and command-specific guidance.

**Acceptance Scenarios**:

1. **Given** a command supporting the shared JSON contract, including one returning partial results, **When** a recognized failure occurs with JSON selected, **Then** stdout contains exactly one existing envelope followed by end of output, preserving payload, error detail, class, and outcome without additional human error rendering.
2. **Given** the same failure in human and supported machine modes, **When** the command exits, **Then** classification, exit status, and existing `--no-err-codes` behavior remain unchanged.
3. **Given** a request-rate wait failure that also carries connection evidence, **When** its human explanation is selected, **Then** request-rate guidance retains precedence and its existing distinction between unsent work and possibly submitted earlier attempts.
4. **Given** a canceled command or existing command-specific diagnostic, including connection-test guidance, **When** the failure is rendered, **Then** established cancellation semantics or command-specific guidance is preserved without misleading connection-test wording on other commands.
5. **Given** any covered command, **When** this correction explains its failure, **Then** requests, retries, confirmations, mutations, successful output, and output-mode precedence remain unchanged, with no diagnostic probe added.

### Edge Cases

- Wrapped or joined errors may carry recognized evidence alongside untrusted text; only supported structured evidence determines the explanation.
- Message text resembling a known failure without corresponding evidence retains the class fallback.
- Authentication may fail before any gateway request; advice must identify the authentication service without claiming a gateway failure or adding a gateway probe.
- Supported service rejections include rejected authentication requests, rejected credentials, denied access, missing endpoints, rate limits, and service errors. Endpoint advice must remain appropriate to the command rather than always naming the topology endpoint.
- Request-rate waiting and connection evidence may coexist; request-rate guidance takes precedence.
- Cancellation must not become an authentication or connectivity remediation problem.
- Repeated failures and partial results retain aggregate counts and original machine evidence without emitting one raw error per resource.
- JSON combined with quiet or other supported controls retains existing selection rules without extra error output.
- Existing DEBUG redaction and quiet/level precedence remain authoritative.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Shared human-error formatting MUST replace a bare class with a bounded explanation and useful next check when existing structured evidence identifies a supported connection or authentication cause. The original failure MUST produce the explanation in User Story 1, scenario 1.
- **FR-002**: Supported explanations MUST cover the causes already recognized by connection diagnostics: timeouts, DNS failures, certificate validation failures, refused or interrupted connections, invalid responses, supported service-status failures, and otherwise recognized authentication failures. Authentication-service and gateway attribution MUST follow available evidence.
- **FR-003**: Recognition MUST use typed evidence, including evidence retained through wrapping or joining, rather than arbitrary error-message text. Unrecognized failures MUST retain the existing class fallback without invented remediation.
- **FR-004**: Request-rate wait guidance MUST retain precedence and existing submission-uncertainty wording. Cancellation MUST retain established meaning and exit behavior.
- **FR-005**: Existing command prefixes, bounded aggregate summaries and counts, and command-specific guidance MUST remain intact. Shared explanations MUST agree with connection diagnostics without command-inappropriate endpoint or connection-test advice.
- **FR-006**: Human failure diagnostics MUST use the configured command-context logger at ERROR level, respecting destination, inherited configuration, level, format, activity handling, and quiet behavior. Each covered failure MUST have one final human error record and no diagnostic stdout payload.
- **FR-007**: Normal and verbose human output MUST exclude credentials, full URLs, HTTP methods, raw request/response bodies, cursors, wrapped transport chains, and per-resource raw failures. Existing redacted DEBUG diagnostics MUST remain available through flag-based and configured DEBUG activation, independently of verbose.
- **FR-008**: JSON schemas, error detail and evidence, partial results, error classification, exit codes, exit-code suppression, and output-mode precedence MUST remain unchanged. Shared-contract JSON failures MUST render exactly one envelope without duplicate human error rendering.
- **FR-009**: Requests, retries, cancellation, confirmation, mutation execution, and successful behavior MUST remain unchanged. Explaining a failure MUST add zero requests and no execution mode or configuration option.
- **FR-010**: Focused isolated checks MUST cover the original authentication refusal, supported causes, typed-versus-text recognition, configured and inherited logger routing and severity, plain and structured formats, repeated multi-resource failures, partial JSON results, redaction, quiet and verbose behavior, both DEBUG activation paths, request-rate precedence, cancellation, fallback, and command-specific compatibility. Output checks MUST capture stdout and stderr separately and detect direct diagnostic prints or raw error exposure.
- **FR-011**: Directly relevant user-facing troubleshooting guidance and affected command references MUST describe the clearer human errors consistently with the unchanged machine and execution contracts.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every supported cause scenario provides a relevant explanation and next check in ordinary human output without DEBUG; the original authentication refusal no longer ends with the unexplained class `internal`.
- **SC-002**: Every covered human failure produces exactly one final ERROR record at the configured destination and zero human diagnostic bytes on stdout, across plain and structured formats.
- **SC-003**: All seeded credentials and raw transport-detail markers are absent from both ordinary and verbose human output streams, including repeated multi-resource failures; DEBUG retains existing redaction guarantees.
- **SC-004**: Every covered automation scenario preserves the baseline machine result, partial results, classification, and exit behavior; shared-contract JSON contains exactly one envelope and no trailing result or human text.
- **SC-005**: All covered workflows preserve baseline request and retry counts, with zero extra diagnostic requests, and all request-rate, cancellation, fallback, and command-specific regression scenarios retain established semantics.

## Assumptions

- The authoritative starting point is commit `10bd90ae`, the pre-#379 baseline requested by issue #381. Broader changes from #379 are not prerequisites or part of this scope.
- Scope is limited to shared human-error formatting, necessary reuse of existing connection-diagnostic behavior, focused isolated regression coverage, and directly relevant documentation. As required by the issue, connection recognition and wording stay together in the connection helper; shared formatting preserves request-rate precedence, delegates recognized failures, and retains its class fallback. This existing-helper reuse constraint does not authorize a diagnostic redesign.
- Existing connection diagnostics and typed failure evidence define the supported cause set. No new diagnostic framework, repository-wide caller migration, transport redesign, or unrelated fix is required.
- Existing command-specific overrides remain authoritative. This feature improves callers of shared formatting rather than converting every command to a new error contract.
- Implementation validation uses isolated fakes, test-owned responses, and isolated command/subprocess execution only. A running Camunda environment, deployment, or live-resource check is not a completion requirement.
- This specification changes no runtime behavior. Planning will select proportionate checks and documentation updates under the current constitution and repository guidance.
