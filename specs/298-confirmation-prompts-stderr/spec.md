# Feature Specification: Confirmation Prompts on Stderr

**Feature Branch**: `298-confirmation-prompts-stderr`

**Created**: 2026-09-11

**Status**: Draft

**Input**: [GitHub issue #298](https://github.com/grafvonb/c8volt/issues/298), “fix(cli): write confirmation prompts to stderr instead of stdout”. Operators need redirected command results to remain free of confirmation text while terminal input remains interactive.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Confirm an operation while capturing results (Priority: P1)

As an operator redirecting command results to a file or pipeline, I want confirmation questions on the diagnostic stream so my captured results contain no prompt text.

**Why this priority**: Confirmation text currently contaminates results even when input remains connected to a terminal.

**Independent Test**: Run an operation requiring default-no confirmation with terminal input and separately captured stdout and stderr. Accept the operation and verify the prompt appears only on stderr while ordinary results retain their existing format on stdout.

**Acceptance Scenarios**:

1. **Given** terminal input and stdout redirected to a file or pipeline, **When** confirmation is required, **Then** the existing question and `[y/N]` choice label appear only on stderr before input is read, without logger prefixes.
2. **Given** an accepted operation that produces results, **When** it completes, **Then** stdout contains ordinary results without confirmation text and the operational outcome is unchanged.
3. **Given** a command with a configured stderr destination, **When** its confirmation is displayed, **Then** the prompt reaches that destination without leaking to stdout.
4. **Given** a default-no question, **When** the operator submits an empty answer, declines, or reaches end of input, **Then** the operation retains its existing abort behavior and the prompt remains confined to stderr.

---

### User Story 2 - Page through keys without polluting the key stream (Priority: P1)

As an operator consuming keys-only results, I want continuation questions separated from keys so I can use captured output directly.

**Why this priority**: A continuation question can otherwise become an invalid key for the next consumer.

**Independent Test**: Run a keys-only listing with multiple pages and terminal input while capturing stdout and stderr separately. Continue once, then decline at a subsequent continuation question.

**Acceptance Scenarios**:

1. **Given** keys-only output and more results available under existing paging rules, **When** a continuation question appears, **Then** stdout contains only one key per line and the question appears only on stderr.
2. **Given** a continuation question, **When** the operator accepts, **Then** paging proceeds under existing rules and subsequent stdout output remains keys only.
3. **Given** a continuation question, **When** the operator declines or reaches end of input, **Then** existing termination and exit behavior is preserved and already emitted keys contain no prompt text.

---

### User Story 3 - Retain familiar confirmation decisions (Priority: P2)

As an interactive or automation user, I want the stream correction to preserve when questions appear and how answers are interpreted, including selector recovery with default-yes confirmation.

**Why this priority**: Moving a prompt must not alter consent or disrupt established scripts.

**Independent Test**: Exercise both confirmation defaults with terminal input for accepted, declined, empty, mixed-case, whitespace-padded, and end-of-input responses. Separately exercise auto-confirm, supported automation, and non-terminal input.

**Acceptance Scenarios**:

1. **Given** selector recovery is eligible to ask its default-yes question under existing caller rules, **When** it prompts, **Then** its unchanged wording and `[Y/n]` label appear only on stderr without logger prefixes.
2. **Given** either confirmation default, **When** the operator enters `y` or `yes`, including uppercase or surrounding whitespace, **Then** the answer is accepted as before; other nonempty answers retain their existing decline behavior.
3. **Given** a default-yes question, **When** the operator submits an empty answer, **Then** the operation proceeds as before; end of input still aborts.
4. **Given** auto-confirm or supported automation skips an applicable question, **When** the command runs, **Then** neither stream receives that prompt and existing decisions and results are preserved.
5. **Given** non-terminal input or selector-recovery guards that suppress prompting, **When** execution reaches the relevant decision, **Then** existing behavior remains unchanged and no new question or input read is introduced.

### Edge Cases

- Redirecting stdout alone must not suppress a question that existing terminal-input rules would display.
- Capturing stderr separately must capture the complete prompt, including existing line breaks, choice label, and trailing spacing.
- Empty answers continue to differ from end of input: default-yes accepts an empty answer, while both defaults abort on end of input.
- Multiple paging questions must each stay off stdout, including when the user stops before all pages are retrieved.
- Auto-confirm and automation skip only the questions they already skip; unsupported automation retains its existing rejection behavior.
- Commands without a custom stderr destination use standard stderr. A caller intentionally merging stdout and stderr accepts combined output; separating such a merged destination is outside this feature.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Default-no confirmation, default-yes selector-recovery confirmation, and paging continuation questions using shared confirmation behavior MUST write all prompt text exclusively to stderr.
- **FR-002**: Confirmation prompts MUST honor the command's configured stderr destination when available, and use standard stderr otherwise.
- **FR-003**: Prompts MUST retain their exact existing wording, formatting, choice labels, and plain interactive presentation, without timestamps, severity labels, or other logger prefixes.
- **FR-004**: Command results MUST retain their existing stdout contract. In keys-only paging, stdout MUST contain only one key per line, with zero continuation text.
- **FR-005**: Both confirmation defaults MUST preserve answer normalization and decisions: `y` and `yes` accept regardless of case and surrounding whitespace; an empty answer accepts only for default-yes; other answers and end of input retain existing abort behavior.
- **FR-006**: The correction MUST preserve terminal checks, caller-side eligibility guards, auto-confirm, supported and unsupported automation behavior, non-interactive behavior, and when input is read.
- **FR-007**: Confirmation decisions, paging progression or termination, mutation authorization, exit behavior, and operational success verification MUST remain unchanged apart from prompt routing.
- **FR-008**: Acceptance validation MUST exercise terminal input with separately captured output streams for both confirmation defaults and keys-only paging. Tests limited to non-terminal input MUST NOT be considered sufficient evidence.
- **FR-009**: Validation MUST cover acceptance, decline, empty answers, end of input, configured stderr capture, and applicable prompt-skipping modes. Targeted regression checks and the full race-enabled test suite MUST pass before implementation is considered complete.
- **FR-010**: User-facing guidance and affected command references MUST consistently describe prompts on stderr and results on stdout, preserving documented confirmation and paging semantics.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In every terminal-input acceptance scenario with separately captured streams, stdout contains zero prompt characters and stderr contains the complete expected prompt.
- **SC-002**: Every line of keys-only stdout across accepted and declined paging scenarios is a result key, with no cleanup required before downstream use.
- **SC-003**: Both confirmation defaults retain 100% of existing decisions across the answer matrix, including empty answers and end of input.
- **SC-004**: All tested auto-confirm, automation, non-interactive, and suppressed selector-recovery scenarios retain their existing prompting and exit behavior.
- **SC-005**: Operators receive the same question before making the same decision in all interactive scenarios, and documentation review finds no contradictory prompt-stream guidance.

## Assumptions

- Scope is limited to routing the existing shared confirmation and selector-recovery prompts, including paging callers. No new confirmation policy or paging design is needed.
- Current answer handling, terminal checks, selector-recovery guards, and automation policies are the compatibility baseline.
- Validation depends on an environment capable of exercising terminal input; a non-terminal substitute alone cannot demonstrate the reported defect is fixed.
- Tenant logger formatting, empty-result rendering, JSON error envelopes, and general stdout/stderr refactoring are out of scope.
- No new persistent data or domain entities are introduced.
