# Feature Specification: Preserve JSON Error Envelopes

**Feature Branch**: `codex/301-preserve-json-errors`

**Created**: 2026-09-11

**Status**: Draft

**Input**: [GitHub issue #301](https://github.com/grafvonb/c8volt/issues/301), “fix(cli): preserve JSON error envelopes in command handlers”. Automation callers need structured failure results from commands that already advertise full machine-contract support.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Receive actionable validation errors in automation (Priority: P1)

As an automation author, I want invalid search options and invalid piped keys to produce the promised structured error result so my script can identify and explain a correctable input problem.

**Why this priority**: An exit status and text diagnostic alone leave callers without the structured result promised by the command.

**Independent Test**: Execute process-instance deletion with invalid search options and each affected full-contract command with invalid stdin keys, capturing stdout, stderr, and exit status separately.

**Acceptance Scenarios**:

1. **Given** process-instance deletion with `--json` and search options that fail validation during command execution, **When** validation fails, **Then** stdout contains exactly one established shared error envelope with outcome `invalid`, the existing validation classification, and the precise normalized error detail.
2. **Given** an affected full-contract command with `--json` and a malformed stdin key, **When** key validation fails, **Then** the single envelope identifies the invoked command and preserves the existing invalid-line and index detail and corrective guidance.
3. **Given** an affected full-contract command receiving human-readable stdin beginning with `filter: `, **When** key validation fails, **Then** the single envelope preserves the specific guidance to use `--keys-only`, without adding a second generic key error.
4. **Given** any corrected validation failure in JSON mode, **When** the command exits, **Then** stdout contains no text outside the envelope, stderr does not repeat the rendered failure, and the existing exit status is preserved.

---

### User Story 2 - Consume structured cluster lookup failures (Priority: P1)

As an operator running scripted cluster checks, I want failed topology, version, and license lookups to return structured errors so automation can interpret failures consistently.

**Why this priority**: These commands already promise structured results, but runtime failures can omit them entirely.

**Independent Test**: Cause a runtime failure separately in cluster topology, version, and license retrieval with JSON output selected, and inspect both streams and exit status.

**Acceptance Scenarios**:

1. **Given** each of the three cluster retrieval commands with `--json`, **When** retrieval fails, **Then** stdout contains exactly one shared error envelope with the correct command identity and the outcome, classification, and normalized detail selected by existing error behavior.
2. **Given** a runtime error with contextual detail, **When** it is rendered, **Then** the detail retains its precise meaning without duplicated message prefixes or a second diagnostic for the same failure.
3. **Given** another command execution failure discovered to bypass the shared result contract, **When** that command already advertises full contract support, **Then** the same single-envelope behavior applies without changing its existing error meaning or exit status.

---

### User Story 3 - Preserve established operator and script behavior (Priority: P2)

As an existing user, I want the correction to preserve human diagnostics, exit controls, successful results, and commands outside the shared contract so existing workflows keep working.

**Why this priority**: Repairing a promised output contract must not introduce unrelated compatibility changes.

**Independent Test**: Repeat corrected failures in human mode and with `--no-err-codes` both enabled and disabled; exercise successful commands and representative callers without full contract support.

**Acceptance Scenarios**:

1. **Given** a corrected failure in ordinary human mode, **When** the command exits, **Then** the existing error appears on stderr and stdout contains no result text.
2. **Given** a corrected validation or runtime failure in JSON or human mode, **When** `--no-err-codes` is enabled, **Then** existing exit-code suppression is preserved while the selected error output still describes the failure accurately.
3. **Given** the same failure without `--no-err-codes`, **When** the command exits, **Then** the existing classified exit code is preserved.
4. **Given** successful execution or a command without full shared-contract support, **When** the command runs, **Then** its output, diagnostics, and exit behavior remain unchanged, including when it shares stdin-key validation with affected commands.
5. **Given** a supported combination of JSON with quiet, automation, or another output-selection flag, **When** a corrected failure occurs, **Then** existing mode precedence and eligibility determine the output, and explicitly selected JSON is preserved wherever the established contract requires it.

### Edge Cases

- Stdin can contain human-readable filter text or a malformed key after valid keys; each must preserve its existing specific validation message.
- A shared stdin-key validation path can serve commands with different contract support; correcting one caller must not expand another caller's contract.
- A failure remains a failure in its structured result even when `--no-err-codes` suppresses the nonzero exit status.
- Wrapped runtime errors retain their existing classification and normalized detail rather than being reclassified as input errors.
- Separate stdout and stderr capture must reveal duplicate diagnostics or trailing output after the single envelope.
- Successful commands and already-correct error paths must not acquire an additional envelope.
- Bootstrap failures and argument or flag parsing failures before command execution remain outside this correction.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Affected commands that already advertise full shared-contract support MUST emit the established shared error envelope on stdout when JSON mode is selected and validation or runtime execution fails.
- **FR-002**: Coverage MUST include process-instance deletion search-option validation, cluster topology/version/license retrieval failures, and stdin-key validation failures for every affected full-contract caller. Other execution failures with the same contract omission MUST be identified and corrected only within existing full-contract support.
- **FR-003**: Each corrected JSON failure MUST produce exactly one valid envelope, with no surrounding human text or additional result, and MUST NOT duplicate the failure diagnostic on stderr.
- **FR-004**: Caller-correctable validation failures MUST retain the existing `invalid` outcome and validation classification. Runtime failures MUST retain the outcome and classification selected by established error behavior.
- **FR-005**: Error results MUST preserve the invoked command identity and existing precise normalized message, including specific stdin guidance and offending-line/index detail where applicable, without repeated message content.
- **FR-006**: Ordinary human error reporting MUST remain on stderr with no added stdout result text. Existing message normalization and human diagnostic behavior MUST be preserved.
- **FR-007**: Existing classified exit codes and `--no-err-codes` behavior MUST remain unchanged in both JSON and human modes. Suppressing an exit code MUST NOT convert a failure result into success.
- **FR-008**: Successful results, already-correct error results, commands without full shared-contract support, and existing output-mode precedence and automation behavior MUST remain unchanged.
- **FR-009**: The correction MUST preserve existing envelope schemas, error classes, and exit-code policy and MUST NOT broaden command contract support or unrelated error behavior.
- **FR-010**: Acceptance validation MUST execute each corrected command error path with stdout and stderr captured separately, verify one decoded envelope followed by end of output, and check precise details, classifications, exit codes, human output, and supported exit-code suppression. Shared validation coverage MUST include both specific stdin failure forms and affected callers.
- **FR-011**: Targeted regression checks and the full race-enabled test suite MUST pass before implementation is considered complete. Regression evidence MUST cover successful execution and commands outside full-contract support.
- **FR-012**: User-facing guidance and affected command references MUST match the corrected failure-output behavior and retained compatibility boundaries.

### Key Entities *(include if feature involves data)*

- **Command error result**: The existing structured failure record containing command identity, outcome, classification, and normalized diagnostic detail. Its established shape and meaning remain unchanged.
- **Command contract support**: The command's existing declaration of machine-readable behavior, which determines eligibility for the shared envelope and is not expanded by this feature.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In 100% of corrected JSON validation and runtime failure scenarios, callers can consume exactly one structured failure result directly, with zero cleanup of surrounding text and zero duplicate diagnostics.
- **SC-002**: Every corrected input-error scenario identifies the failure as invalid and retains the existing actionable detail; every corrected runtime scenario retains its established failure meaning.
- **SC-003**: All corrected failure scenarios retain their expected exit status with exit-code suppression both enabled and disabled, without changing the reported failure into success.
- **SC-004**: Every ordinary human-mode regression scenario reports its error on stderr with zero stdout result bytes.
- **SC-005**: All tested successful executions, already-correct failures, and commands outside full-contract support retain their established output and exit behavior.

## Assumptions

- The current shared error contract and existing classification, normalization, and exit policies are authoritative; this feature repairs missing delivery of that contract.
- Scope is bounded to command execution failures that bypass a contract the command already supports. The known paths in issue #301 are the minimum coverage, not permission for repository-wide error redesign.
- Planning will inventory affected full-contract callers and define local corrections using the established error-handling behavior. No new error framework is needed.
- Bootstrap failures, pre-execution argument/flag parsing errors, prompt routing, empty-result rendering, tenant logger formatting, and repository-wide refactoring are out of scope.
- Validation assumes writable output destinations; introducing a new policy for output-write failures is outside this issue.
- This feature relies on existing command contract declarations and introduces no new persistent data.
