# Feature Specification: Preserve Concise CLI Error Breadcrumbs

**Feature Branch**: `112-error-context-dedup`  
**Created**: 2026-04-17  
**Status**: Draft  
**Input**: User description: "GitHub issue #112: refactor(errors): preserve call-path context while removing duplicated error message content"

## GitHub Issue Traceability

- **Issue Number**: 112
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/112
- **Issue Title**: refactor(errors): preserve call-path context while removing duplicated error message content

## Clarifications

### Session 2026-04-17

- Q: Which scope should this feature lock in if investigation finds similar duplication elsewhere? → A: Fix every duplicated CLI error path found in the repo during investigation.
- Q: When a not-found-style CLI failure is cleaned up, what should the final rendered shape preserve? → A: Keep the existing shared class prefix such as `resource not found:` and only deduplicate the trailing wrapped details.
- Q: How broad should the required automated coverage be for the repo-wide investigation scope? → A: Add or update tests for representative duplicated paths in each affected error-pattern family.
- Q: If investigation finds the same duplication pattern on non-not-found failures, how should this feature treat them? → A: Apply the same rule: preserve the existing shared class prefix for that error class and deduplicate only the wrapped details.
- Q: If a breadcrumb label itself is verbose or partly repetitive, how much wording change should this feature allow? → A: Allow breadcrumb labels to be shortened if their meaning stays equivalent and the stage remains identifiable.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Keep CLI failures readable (Priority: P1)

As a CLI operator, I want failure output to retain short breadcrumb context without repeating the same root message so that I can understand what failed quickly.

**Why this priority**: The issue is visible directly in user-facing command output, and noisy failures make real troubleshooting slower even when the command is otherwise classifying the error correctly.

**Independent Test**: Run representative duplicated CLI error paths from each affected error-pattern family, including the originally reported process-instance flow and any other duplicated patterns discovered during investigation, and verify the rendered errors keep the stage breadcrumbs while showing the underlying resource failure only once.

**Acceptance Scenarios**:

1. **Given** a duplicated CLI error path fails while composing user-facing output, **When** the CLI renders the error, **Then** it keeps the existing shared error-class prefix, includes concise breadcrumbs for the failing stages, and shows only one final expression of the underlying resource failure.
2. **Given** an affected lookup already has a concise resource-specific failure, **When** higher layers wrap that failure, **Then** they do not restate the same identifier or the same failure meaning in additional prose, and any shortened breadcrumb labels still identify the same failure stages.

---

### User Story 2 - Preserve where the failure happened (Priority: P2)

As a maintainer, I want intermediate layers to keep short call-path context so that user-facing errors still show the rough failure path without turning into repeated sentences.

**Why this priority**: Removing duplication should not erase the contextual breadcrumbs that make multi-step command failures diagnosable.

**Independent Test**: Trigger failures through multiple duplicated CLI error paths discovered during investigation, then confirm the final message still names the relevant stages in order while avoiding repeated root-cause wording.

**Acceptance Scenarios**:

1. **Given** a multi-step command fails after several internal lookup stages, **When** the final error is composed, **Then** the rendered output preserves the sequence of concise breadcrumb labels needed to identify the failing stage.
2. **Given** a lower layer returns a resource-specific failure, **When** an intermediate layer adds context, **Then** the added wrapper contributes only stage context rather than another fully formatted failure sentence, and any breadcrumb wording changes preserve equivalent stage meaning.

---

### User Story 3 - Keep failure semantics unchanged (Priority: P3)

As a maintainer, I want the refactor to improve message composition without changing classification or exit behavior so that existing CLI failure handling remains stable.

**Why this priority**: The feature is intentionally about message composition, not about changing error classes, normalization, or exit-code mapping.

**Independent Test**: Exercise the affected command paths and shared error handling expectations, then verify the user-facing message is cleaner while the normalized error class and exit behavior remain the same.

**Acceptance Scenarios**:

1. **Given** an affected command fails with a known shared error class, **When** the message composition is refactored, **Then** the command still resolves to the same user-facing error class, keeps that class prefix, and preserves the same exit behavior as before.
2. **Given** automated coverage inspects the affected command path, **When** the refactor is complete, **Then** tests confirm both preserved breadcrumb context and reduced duplication in the final rendered error.

### Edge Cases

- A multi-step command may add several contextual layers, but each layer must contribute only stage context and must not restate the same root failure message.
- If the failing resource identifier appears in a breadcrumb label, the final composed message must avoid repeating the same identifier again as part of duplicated failure prose where the breadcrumb already provides that context.
- Different command paths may fail through ancestry, direct lookup, or other multi-step stages, and all duplicated flows found during investigation must follow the same concise composition rule.
- When the root failure is already minimal, additional wrappers must not expand it into a longer sentence that repeats the same meaning.
- The rendered output must remain clear for users even when the failure passes through shared normalization or classification layers after the contextual wrappers are applied.
- If a breadcrumb label is shortened to reduce noise, the shortened label must still make the same failure stage recognizable to users and maintainers.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST preserve concise breadcrumb context that indicates the sequence of relevant stages leading to an affected CLI failure.
- **FR-002**: The system MUST ensure the root failure message appears only once in the final rendered error where a single concise root message is available.
- **FR-003**: The system MUST avoid repeating the same resource identifier across multiple error layers when the repetition does not add new context.
- **FR-004**: The system MUST avoid repeating equivalent "not found" wording across multiple wrapped layers of the same failure.
- **FR-005**: Intermediate layers MUST add only concise stage context when wrapping an already formatted underlying failure.
- **FR-005a**: Breadcrumb labels MAY be shortened to reduce redundancy only when their meaning remains equivalent and the failing stage remains identifiable.
- **FR-006**: The system MUST review duplicated CLI error propagation paths across the repository, including the originally reported process-instance walk, ancestry, and direct get chain, for redundant message expansion.
- **FR-007**: The system MUST keep existing shared error classification, normalization, and exit-code behavior unchanged for the affected failure paths.
- **FR-007a**: For not-found-style failures, the system MUST preserve the existing shared class prefix in the final rendered CLI error while deduplicating only the wrapped detail content that follows it.
- **FR-007b**: When the same duplication pattern appears on other shared error classes, the system MUST preserve the existing shared class prefix for those classes and deduplicate only the wrapped detail content that follows it.
- **FR-008**: The system MUST preserve the externally observable failure semantics of the affected commands aside from the reduction in duplicate message content.
- **FR-009**: The system MUST provide automated regression coverage for representative duplicated error paths in each affected error-pattern family found during investigation, verifying both preserved breadcrumb context and removal of duplicated failure wording.

### Key Entities *(include if feature involves data)*

- **Breadcrumb Context**: Short stage labels that show the rough call path of a failing command without rephrasing the root failure.
- **Root Failure Message**: The final concise resource-specific failure that should appear once in the rendered CLI error.
- **Rendered CLI Error**: The composed user-facing error string shown after contextual wrapping and shared error handling are applied, including the existing shared class prefix where that prefix already defines the error class.
- **Affected Error Path**: Any CLI error propagation flow that duplicates resource identifiers or repeated failure wording while composing the final user-facing message.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated coverage for representative duplicated CLI error paths in each affected error-pattern family shows the final rendered errors preserve breadcrumb labels identifying the failing stages.
- **SC-002**: Automated coverage for those representative paths shows the root resource failure text is rendered once instead of being restated across multiple layers.
- **SC-003**: Automated coverage confirms repeated identifiers and repeated failure phrasing are removed from the final user-facing message for each covered pattern family.
- **SC-004**: Automated coverage confirms the affected failure paths still resolve through the same shared classification and exit-code behavior as before the refactor.
- **SC-004a**: Automated coverage confirms that not-found-style failures keep the existing shared class prefix while the trailing wrapped details are deduplicated.
- **SC-004b**: Automated coverage for any covered non-not-found pattern family confirms the existing shared class prefix is preserved while duplicated wrapped detail text is removed.
- **SC-004c**: Automated coverage confirms any shortened breadcrumb labels still identify the same failure stages as the pre-refactor paths for the covered pattern families.

## Assumptions

- The feature is limited to improving how affected errors are composed and rendered, not redefining which scenarios are classified as not found or any other shared error class.
- Existing shared class prefixes remain part of the intended user-facing contract for the affected failure classes and should be preserved while deduplicating lower-level detail text.
- The same prefix-preservation rule should apply consistently across shared error classes when the duplication pattern is otherwise the same.
- Breadcrumb wording may become shorter, but it should not become ambiguous or hide which stage of the call path failed.
- Concise breadcrumb labels such as ancestry or lookup stage names remain valuable user-facing context and should be preserved.
- The originally reported process-instance walk, ancestry, and related get flows are the starting point, but the feature scope expands to any duplicated CLI error path discovered during investigation.
- Regression coverage should prove the intended behavior for each affected duplication pattern family through representative paths rather than requiring one test per every discovered command path.
- Downstream implementation work for this feature must keep Conventional Commit formatting and append `#112` as the final token of every commit subject.
