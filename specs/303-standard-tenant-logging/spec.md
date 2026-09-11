# Feature Specification: Standard Tenant Logging for Delete and Cancel

**Feature Branch**: `codex/303-standard-tenant-logging`

**Created**: 2026-09-11

**Status**: Draft

**Input**: [GitHub issue #303 — fix(cli): use standard logger for delete/cancel tenant messages](https://github.com/grafvonb/c8volt/issues/303)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Recognize Tenant Information and Warnings (Priority: P1)

As an operator deleting or cancelling process instances selected by filters, I want tenant context to follow the same logging conventions as other operational messages so I can distinguish information from warnings and read a consistent execution record.

**Why this priority**: Tenant context explains the scope of a destructive operation; losing warning severity makes that context harder to assess.

**Independent Test**: Exercise both delete and cancel tenant-context reporting paths with standard logging enabled and tenant context containing informational and warning messages. Verify severity, message content, ordering, and duplicate suppression.

**Acceptance Scenarios**:

1. **Given** an eligible tenant-context message without a warning marker, **When** a selector-based delete or cancel reports it, **Then** it appears as an INFO record in the standard configured log format.
2. **Given** an eligible tenant-context message marked as a warning, **When** either command reports it, **Then** it appears as a WARN record and retains its original text.
3. **Given** configured tenant, override, selection scope, and affected-tenant messages, **When** tenant context is reported, **Then** their existing wording and relative ordering are preserved.
4. **Given** tenant context has already been reported for an execution, **When** another reporting point is reached, **Then** the existing duplicate-suppression behavior is preserved.

---

### User Story 2 - Apply Configured Log Format and Level (Priority: P2)

As an operator collecting command logs, I want tenant messages to respect my selected log format and severity threshold so they can be read and filtered alongside other operational records.

**Why this priority**: Consistent formatting and filtering allow operators and log consumers to use their existing settings without special handling for tenant context.

**Independent Test**: With a logger attached to the command, exercise both tenant-context reporting paths in plain and JSON log formats at thresholds that allow both INFO and WARN, only WARN, and neither severity.

**Acceptance Scenarios**:

1. **Given** plain log formatting, **When** eligible tenant messages are reported, **Then** each uses the established plain log representation with its correct severity and unchanged message text.
2. **Given** JSON log formatting, **When** eligible tenant messages are reported, **Then** each is a valid standard structured log record containing the correct severity and unchanged message text.
3. **Given** a threshold that excludes INFO but allows WARN, **When** both message categories are eligible, **Then** only warnings are emitted.
4. **Given** a threshold that excludes INFO and WARN, **When** tenant context is reported, **Then** neither category is emitted and no raw tenant-message fallback bypasses the filter.

---

### User Story 3 - Preserve Command Output and Execution Behavior (Priority: P2)

As an operator or automation author, I want existing delete and cancel workflows to keep their result formats, tenant scope, and interaction rules so this logging correction does not change what my commands do.

**Why this priority**: Logging must remain compatible with scripts and with established safeguards around destructive operations.

**Independent Test**: Compare command execution with representative tenant context across human, JSON-result, keys-only, quiet, automation, dry-run, and verbose modes, including supported combinations. Capture result and diagnostic streams separately and verify unchanged execution decisions.

**Acceptance Scenarios**:

1. **Given** JSON results or keys-only output is requested, **When** either command completes, **Then** stdout retains its established result content with no added tenant log records; existing tenant-message eligibility remains unchanged.
2. **Given** quiet, automation, dry-run, or verbose behavior, including supported combinations with machine output, **When** either command reaches tenant reporting, **Then** existing visibility and suppression rules remain intact apart from standard log formatting and level filtering for eligible messages.
3. **Given** the same selection and operator confirmation response, **When** either command runs after this correction, **Then** tenant selection, discovery, confirmation decisions, mutation requests, and exit behavior match the existing workflow.
4. **Given** confirmation is required, **When** the operator reaches the prompt, **Then** prompt wording and stream placement remain unchanged and the prompt remains plain interactive text.

### Edge Cases

- Missing tenant context or an empty set of reportable messages produces no new log records.
- Repeated reporting opportunities retain existing deduplication, including when a log threshold suppresses the first reporting attempt.
- Tenant overrides, unfiltered scope, multiple affected tenants, and unknown target tenants retain the severity already assigned to each message; no new warning policy is introduced.
- JSON log formatting and JSON command results are separate settings; log formatting must not alter result schemas or result-stream purity.
- Quiet or otherwise suppressed reporting is not re-enabled by attaching a logger or selecting a more permissive logging threshold.
- Empty selections, dry-run previews, aborts, and failures retain their current reporting and execution semantics; logging adds no discovery or mutation work.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Selector-based process-instance delete and cancel MUST report eligible tenant-context messages through the established operational logging behavior at both progress and confirmation-context reporting points.
- **FR-002**: Messages without the existing warning marker MUST use INFO; messages with that marker MUST use WARN.
- **FR-003**: Tenant-context records MUST honor the configured plain or JSON log format and log-level filtering without raw-message output bypassing that filtering.
- **FR-004**: Message text, relative ordering, and existing duplicate-suppression behavior MUST be preserved.
- **FR-005**: Existing tenant-context output-mode guards MUST be preserved, including quiet, automation, dry-run, verbose, JSON-result, keys-only, and supported combined modes.
- **FR-006**: JSON results, keys-only stdout, and all existing result contracts MUST remain unchanged; tenant log records MUST NOT contaminate result stdout.
- **FR-007**: Tenant selection, discovery, confirmation, mutation behavior, and exit outcomes MUST remain unchanged. Interactive prompts MUST retain their established plain wording and configured diagnostic-stream destination.
- **FR-008**: Validation MUST exercise both affected reporting paths with an attached logger in plain and JSON log formats, verify INFO/WARN classification and level filtering, and include command-level compatibility coverage with separately captured output streams.
- **FR-009**: Changes MUST stay within tenant-context logging for the affected delete/cancel workflows. Other raw diagnostic messages, paging, watch, configuration warnings, tenant-reporting policy, and output-contract redesign are excluded.

### Key Entities

- **Tenant-context message**: Existing operator-facing text describing tenant configuration, overrides, selection scope, or affected tenants, with an existing warning marker and position in the reporting sequence.
- **Logging configuration**: The operator's chosen record format and severity threshold, applied to eligible tenant-context messages.
- **Reporting eligibility**: Existing rules determining whether tenant context may be shown in the current output mode and whether it has already been reported.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Across both commands and both reporting paths, 100% of eligible tenant messages have the expected INFO or WARN severity with their original text and ordering.
- **SC-002**: All plain and JSON logging acceptance cases conform to the selected format, and zero tenant records appear below the configured severity threshold.
- **SC-003**: All covered output-mode combinations preserve their existing tenant-message eligibility and result content, with zero tenant log records added to stdout and zero additional duplicate messages.
- **SC-004**: All compatibility scenarios preserve selection, discovery, confirmation, mutation, and exit outcomes; targeted regression checks and the full race-enabled test suite pass before implementation is accepted.

## Assumptions

- Issue #303 is the authoritative scope. Its named implementation touchpoints and required reuse of the existing logging helper are planning constraints available in the linked issue; this specification states their observable behavior.
- Existing standard logging conventions, warning classification, and tenant-reporting policy are authoritative; this correction introduces no new formatting standard, logging helper, or logging framework refactor.
- Plain and JSON log formats refer to diagnostic records, independently of the command result format.
- Existing behavior when no logger is attached follows the established operational logging fallback and will be assessed during planning.
- Implementation planning will determine documentation impact under the project constitution, preserving existing tenant-reporting policy and avoiding unrelated documentation changes.
