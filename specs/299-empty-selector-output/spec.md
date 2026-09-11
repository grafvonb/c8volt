# Feature Specification: Empty Selector Result Output

**Feature Branch**: `codex/299-empty-selector-output`

**Created**: 2026-09-11

**Status**: Draft

**Input**: [GitHub issue #299 — preserve JSON and keys-only output for empty delete/cancel results](https://github.com/grafvonb/c8volt/issues/299)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Consume empty results in automation (Priority: P1)

As an operator running scripted process-instance deletion or cancellation, I want a search with no matches to produce the selected machine-readable result so my script can recognize successful completion without special handling for human text.

**Why this priority**: Unexpected text breaks JSON consumers and can be mistaken for a process-instance key by downstream commands.

**Independent Test**: Run each selector-based command against an empty matching scope, in normal execution and dry-run, with JSON and keys-only output. Check the complete stdout and successful exit status.

**Acceptance Scenarios**:

1. **Given** no instances match the selectors, **When** either delete or cancel runs with `--json`, **Then** stdout contains exactly one valid shared result envelope identifying the command, a successful no-op outcome, and an empty command-appropriate payload, with no human summary or other text outside the envelope.
2. **Given** the same empty scope, **When** either command runs with `--dry-run --json`, **Then** stdout contains exactly one successful envelope with an empty preview payload and no suggestion that a mutation was submitted.
3. **Given** the same empty scope, **When** either command runs with `--keys-only`, with or without `--dry-run`, **Then** stdout contains zero bytes, including no blank line.
4. **Given** the same empty scope, **When** any of these executions uses auto-confirm or automation mode, **Then** it preserves the selected output contract and does not request confirmation or submit a mutation.
5. **Given** the same empty scope, **When** JSON execution uses the existing no-wait option where supported, **Then** the envelope still reports successful completion rather than an accepted or pending mutation.

---

### User Story 2 - Understand an empty result interactively (Priority: P2)

As an operator using human output, I want to see that no process instances matched, and I want quiet execution to suppress that informational message.

**Why this priority**: The existing summary is useful at a terminal, while quiet execution must avoid unwanted informational output.

**Independent Test**: Run delete and cancel against an empty matching scope with ordinary human output and quiet mode, for normal execution and dry-run, capturing stdout and stderr separately.

**Acceptance Scenarios**:

1. **Given** no instances match, **When** either command runs in ordinary human output mode, with or without dry-run, **Then** stdout contains `found: 0` exactly once and no confirmation is requested.
2. **Given** the same empty scope, **When** quiet mode is enabled, **Then** the informational empty-result message appears on neither stdout nor stderr.
3. **Given** the same empty scope, **When** quiet mode is combined with JSON or keys-only output, **Then** the selected machine-readable contract remains intact: one successful envelope for JSON or empty stdout for keys-only.

---

### User Story 3 - Preserve existing selection and execution behavior (Priority: P2)

As an operator, I want the output correction to preserve which instances are selected, when commands fail, and how nonempty operations behave.

**Why this priority**: A formatting fix must not change operational scope or report failed discovery as a successful empty result.

**Independent Test**: Compare discovery activity and exit behavior for empty scopes, then verify representative validation failures, discovery failures, nonempty selector operations, and explicit-key operations retain their existing behavior.

**Acceptance Scenarios**:

1. **Given** valid selectors return no matches, **When** either command completes, **Then** it returns the existing successful exit status without a confirmation prompt, mutation request, or additional discovery request caused by output rendering.
2. **Given** selectors fail validation or discovery fails, **When** either command runs, **Then** existing failure handling and exit-code behavior remain unchanged, without a successful empty-result envelope.
3. **Given** matching instances exist or explicit keys are supplied, **When** the command runs, **Then** its existing selection, confirmation, mutation, rendering, and exit behavior remain unchanged.

### Edge Cases

- Empty delete preview, empty delete mutation planning, and empty cancel search planning must all honor the same output-mode rules.
- Dry-run results must remain previews, even when there are no preview entries to display.
- No-wait must not turn a completed empty operation into an accepted mutation result.
- Quiet mode suppresses the informational summary without suppressing the requested JSON result.
- Auto-confirm and automation must not cause duplicate output or additional work for an empty scope.
- A discovery failure, user abort, or empty report after nonempty discovery is not automatically a successful zero-match result.
- Output redirected to a file or pipe must contain the same selected result format as terminal output.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Selector-based process-instance delete and cancel MUST render successful zero-match discovery according to the selected output mode, for both normal execution and dry-run.
- **FR-002**: JSON output MUST consist of exactly one valid shared result envelope with the command identity, a successful outcome, and an empty payload appropriate to deletion, cancellation, or the corresponding dry-run preview. It MUST contain no human empty-result message or text outside the envelope.
- **FR-003**: An empty JSON result MUST describe a completed no-op and MUST NOT claim a mutation was submitted, accepted, pending, or performed, including when no-wait is enabled.
- **FR-004**: Keys-only output MUST contain zero bytes when no instances match.
- **FR-005**: Ordinary non-quiet human output MUST retain `found: 0` exactly once on stdout. Quiet execution MUST suppress this informational message on both output streams.
- **FR-006**: Quiet mode MUST preserve explicitly selected machine-readable results, and existing output-mode precedence MUST remain unchanged.
- **FR-007**: Empty scopes MUST complete without confirmation prompts or mutation requests. Auto-confirm and automation MUST preserve the selected result format.
- **FR-008**: Rendering empty results MUST NOT introduce additional discovery requests or change discovery, validation, filtering, authorization, mutation targets, or existing exit-code behavior.
- **FR-009**: Nonempty selector results and explicit-key operations MUST retain their existing behavior. Errors and user aborts MUST NOT be reclassified as successful empty discovery.
- **FR-010**: User-facing documentation and examples MUST describe empty-result behavior consistently for human, JSON, keys-only, quiet, and dry-run execution.

### Key Entities

- **Selector scope**: The process instances matched by the command's existing filters; this feature applies when successful discovery finds zero instances.
- **Empty command result**: A completed operation with no selected instances and no mutations, represented as an empty mutation result or empty dry-run preview according to the invocation.
- **Shared result envelope**: The existing machine-readable command result containing command identity, outcome, and the command-appropriate payload, retaining applicable existing context.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Across all four combinations of command (delete/cancel) and execution type (normal/dry-run), 100% of successful zero-match JSON executions produce exactly one parseable successful result with zero reported mutations.
- **SC-002**: Across those four combinations, keys-only executions produce zero stdout bytes; ordinary human executions show exactly one `found: 0`; quiet executions show zero informational empty-result messages.
- **SC-003**: Every zero-match acceptance case completes with zero confirmation prompts, zero mutation requests, zero added discovery requests, and the existing successful exit status.
- **SC-004**: Auto-confirm, automation, quiet with machine-readable output, and supported no-wait combinations satisfy the same empty-result contracts in every applicable acceptance case.
- **SC-005**: All representative nonempty, explicit-key, validation-error, and discovery-error regression cases preserve their previous observable behavior and exit status.
- **SC-006**: Operators and script consumers can distinguish a successful empty operation from a submitted mutation using the result alone, without parsing human text or performing another discovery operation.

## Assumptions

- Scope and acceptance behavior come from issue #299; this is an output correction for successful empty selector-based deletion and cancellation only.
- Existing shared result and preview contracts define the payload shape. This feature introduces no new envelope schema or output flag.
- Quiet mode suppresses informational messages, not explicitly requested machine-readable results; existing option validation and precedence remain authoritative.
- Existing discovery and selection behavior supplies the zero-match result. No new discovery capability or backend behavior is needed.
- Prompt routing, tenant logger formatting, JSON error-envelope changes, tenant filtering, authorization changes, mutation-target changes, and general renderer refactoring are out of scope.
- Delivery validation must cover both commands, normal execution and dry-run, all specified output modes, and supported automation combinations. Targeted regression tests and the full race-enabled test suite must pass before implementation is considered complete.
