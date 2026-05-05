# Feature Specification: Process Instance Incident List Output

**Feature Branch**: `171-pi-incident-list-output`  
**Created**: 2026-05-05  
**Status**: Draft  
**Input**: User description: "GitHub issue #171: feat(get-pi): support incident details in process-instance list output"

## GitHub Issue Traceability

- **Issue Number**: 171
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/171
- **Issue Title**: feat(get-pi): support incident details in process-instance list output

## Clarifications

### Session 2026-05-05

- No critical ambiguities detected worth formal clarification. The GitHub issue defines the command surface, list/search selectors, direct and indirect incident rendering, JSON behavior, truncation rules, validation errors, paging preservation, and required automated coverage.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Show Direct Incidents In List Output (Priority: P1)

As a CLI user searching process instances, I want `c8volt get pi --with-incidents` to work without `--key` so that list and search results can show incident details directly under each matching process instance.

**Why this priority**: List/search incident enrichment is the core requested behavior and delivers the main user value independently of formatting refinements.

**Independent Test**: Run `c8volt get pi --with-incidents` with a list/search selector that returns a process instance with direct incidents, then verify the normal process-instance row remains unchanged and each direct incident appears as an indented line immediately below its matching row.

**Acceptance Scenarios**:

1. **Given** a process-instance list selector matches one instance with direct incidents, **When** the user runs `c8volt get pi --with-incidents`, **Then** the output shows the unchanged process-instance row followed by each direct incident indented below that row.
2. **Given** multiple matching process instances have different incident details, **When** the user runs a list/search command with `--with-incidents`, **Then** each incident line appears below the process instance it belongs to.
3. **Given** list/search output is limited or paged, **When** `--with-incidents` is enabled, **Then** incident rendering applies only to the listed process instances and does not change paging or `--limit` behavior.

---

### User Story 2 - Preserve Enriched JSON Behavior (Priority: P2)

As an automation user consuming JSON, I want list/search mode with `--json --with-incidents` to return enriched incident data using the existing keyed lookup shape so that scripts can consume full incident details consistently.

**Why this priority**: JSON behavior is part of the requested command contract and must be consistent with keyed lookup before formatting-only refinements are layered on.

**Independent Test**: Run keyed and list/search `get pi --json --with-incidents` commands against process instances with direct incidents, then verify list/search results include the same enriched incident shape and full messages used by keyed lookup.

**Acceptance Scenarios**:

1. **Given** a process instance has direct incident details, **When** the user runs `c8volt get pi --json --with-incidents` in list/search mode, **Then** the returned JSON includes the existing enriched incident structure with full incident messages.
2. **Given** a user requests JSON with incident messages longer than any human output limit, **When** `--json --with-incidents` is used, **Then** the JSON keeps full incident messages.
3. **Given** keyed lookup already supports `--json --with-incidents`, **When** the feature is complete, **Then** keyed lookup remains functionally unchanged.

---

### User Story 3 - Explain Indirect Incident Markers (Priority: P3)

As a CLI user investigating process instances marked with incidents, I want list output to distinguish direct incident details from markers that may refer to incidents deeper in the process-instance tree so that I know when to inspect with `walk pi`.

**Why this priority**: Without this behavior, rows marked `inc!` but lacking direct incidents look broken or misleading in list output.

**Independent Test**: Run list/search output where at least one listed process instance is marked `inc!` or `incident: true` but direct incident lookup returns no incidents, then verify the row has a short indented note and the list emits a single de-duplicated tree-inspection warning.

**Acceptance Scenarios**:

1. **Given** a listed process instance is marked with an incident but direct incident lookup returns no incidents, **When** human output renders the row, **Then** a short indented note appears below that row.
2. **Given** multiple listed process instances have indirect incident markers without direct incidents, **When** human output completes, **Then** the tree-inspection warning is printed at most once for the list.
3. **Given** the warning is printed, **When** a user reads it, **Then** it explains that the incident may exist in the process-instance tree and can be inspected with `walk pi --key <key> --with-incidents`.

---

### User Story 4 - Use Compact Human Incident Lines (Priority: P4)

As a CLI user reading human output, I want incident detail lines to use the compact `inc <incident-key>:` prefix and optionally shorten long messages so that incident output stays readable in lists.

**Why this priority**: The prefix and truncation behavior improves readability after core enrichment is available.

**Independent Test**: Run `get pi --with-incidents` and `walk pi --with-incidents` in human-readable mode with direct incidents and long messages, then verify both commands use `inc <incident-key>:` and `--incident-message-limit <chars>` truncates only human incident messages with an appended `...`.

**Acceptance Scenarios**:

1. **Given** human-readable incident output is rendered by `get pi --with-incidents`, **When** an incident line is shown, **Then** it uses `inc <incident-key>: <error-message>`.
2. **Given** human-readable incident output is rendered by `walk pi --with-incidents`, **When** an incident line is shown, **Then** it uses `inc <incident-key>: <error-message>`.
3. **Given** `--incident-message-limit <chars>` is set and an incident message exceeds the limit, **When** human output renders the incident line, **Then** the message portion is character-safe truncated and ends with `...`.
4. **Given** the message limit is `0`, **When** human output renders incident lines, **Then** messages are not truncated.

---

### User Story 5 - Validate Incident Options Safely (Priority: P5)

As a CLI user, I want invalid flag combinations and values to fail clearly so that existing workflows remain predictable while the new incident-list behavior is added.

**Why this priority**: Validation protects existing command behavior and script compatibility after the main output capabilities are defined.

**Independent Test**: Run valid and invalid `get pi` invocations around `--with-incidents`, `--incident-message-limit`, `--total`, keyed lookup, and list/search selectors, then verify accepted cases work and invalid cases fail with clear errors.

**Acceptance Scenarios**:

1. **Given** the user runs list/search `get pi --with-incidents` without `--key`, **When** validation runs, **Then** the command is accepted.
2. **Given** the user runs keyed `get pi --key <key> --with-incidents`, **When** validation runs, **Then** the command remains accepted and preserves existing behavior.
3. **Given** the user combines `--with-incidents` with `--total`, **When** validation runs, **Then** the command fails clearly.
4. **Given** the user passes `--incident-message-limit` without `--with-incidents`, **When** validation runs, **Then** the command fails clearly.
5. **Given** the user passes a negative `--incident-message-limit`, **When** validation runs, **Then** the command fails clearly.

### Edge Cases

- List/search selectors include `--incidents-only`, `--state active`, and `--bpmn-process-id <id>` with `--with-incidents`.
- Matching process instances with no direct incidents and no incident marker render unchanged.
- Matching process instances with multiple direct incidents render all direct incidents below the correct row.
- Matching process instances marked `inc!` with no direct incident details render a short note below the affected row.
- The tree-inspection warning for indirect incident markers is de-duplicated per list output.
- `--with-incidents` remains invalid with `--total`.
- `--incident-message-limit` is invalid without `--with-incidents`.
- Negative incident message limits are rejected.
- Default incident message limit `0` means unlimited.
- Human truncation applies to the incident error message, not the full rendered line.
- Human truncation is character-safe and appends `...` only when truncation is applied.
- JSON output keeps full incident messages even when a human message limit is provided.
- Existing behavior without `--with-incidents` remains unchanged.
- Keyed `get pi --with-incidents` remains functionally unchanged.
- Paging and `--limit` determine the process instances to display before incident details are rendered.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow `c8volt get pi --with-incidents` in list/search mode without requiring `--key`.
- **FR-002**: The system MUST continue allowing keyed `c8volt get pi --key <key> --with-incidents`.
- **FR-003**: Human list/search output MUST render the normal process-instance row unchanged before any incident detail lines.
- **FR-004**: Human list/search output MUST render each direct incident as an indented line immediately below its matching process instance.
- **FR-005**: Human incident detail lines MUST use the prefix `inc <incident-key>:` for `get pi --with-incidents`.
- **FR-006**: Human incident detail lines MUST use the prefix `inc <incident-key>:` for `walk pi --with-incidents`.
- **FR-007**: Incident details MUST preserve per-process-instance association in list/search output.
- **FR-008**: `c8volt get pi --json --with-incidents` in list/search mode MUST return the existing enriched incident shape used by keyed lookup.
- **FR-009**: JSON output with `--with-incidents` MUST keep full incident messages by default.
- **FR-010**: The system MUST add `--incident-message-limit <chars>` for human-readable incident output.
- **FR-011**: The default incident message limit MUST be `0`, meaning unlimited.
- **FR-012**: `--incident-message-limit` MUST require `--with-incidents`.
- **FR-013**: `--incident-message-limit` MUST reject negative values with a clear validation error.
- **FR-014**: Human output MUST truncate only the incident error message when the message exceeds the configured limit.
- **FR-015**: Human output truncation MUST be character-safe and append `...` when applied.
- **FR-016**: `--incident-message-limit` MUST NOT truncate JSON incident messages.
- **FR-017**: If a listed process instance is marked `inc!` or `incident: true` but direct incident lookup returns no incidents, human output MUST render a short indented note below that row.
- **FR-018**: If at least one listed process instance has an indirect incident marker without direct incident details, human list output MUST print at most one warning after the list.
- **FR-019**: The de-duplicated warning MUST explain that the incident may exist in the process-instance tree and can be inspected with `walk pi --key <key> --with-incidents`.
- **FR-020**: `--with-incidents` MUST remain invalid with `--total`.
- **FR-021**: Existing output and behavior without `--with-incidents` MUST remain unchanged.
- **FR-022**: Existing keyed `get pi --with-incidents` behavior MUST remain functionally unchanged except for the human prefix shortening.
- **FR-023**: Paging and `--limit` behavior MUST remain compatible with incident rendering.
- **FR-024**: Automated tests MUST cover list-mode incident rendering.
- **FR-025**: Automated tests MUST cover the `inc <incident-key>:` prefix for `get pi --with-incidents`.
- **FR-026**: Automated tests MUST cover the `inc <incident-key>:` prefix for `walk pi --with-incidents`.
- **FR-027**: Automated tests MUST cover indirect incident marker notes and the de-duplicated warning.
- **FR-028**: Automated tests MUST cover human incident message truncation.
- **FR-029**: Automated tests MUST cover JSON incident enrichment with full messages.
- **FR-030**: Automated tests MUST cover validation errors for `--total`, missing `--with-incidents`, and negative limits.

### Key Entities *(include if feature involves data)*

- **Process Instance Result**: A process-instance row selected by keyed lookup or list/search selectors, including visible lifecycle and incident marker fields.
- **Direct Incident**: An incident record directly associated with a listed process instance and rendered beneath that specific process-instance row.
- **Indirect Incident Marker**: A process-instance incident indicator where direct incident lookup returns no details, implying the incident may exist deeper in the process-instance tree.
- **Incident Message Limit**: The optional human-output character limit applied only to incident error messages.
- **Incident Output Association**: The relationship between each rendered incident detail or indirect-note line and the process-instance row it belongs to.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated tests show `c8volt get pi --with-incidents` works without `--key` for list/search selectors.
- **SC-002**: Automated tests show direct incident lines render under the matching process-instance rows and preserve row-to-incident association.
- **SC-003**: Automated tests show keyed `get pi --with-incidents` remains functionally unchanged.
- **SC-004**: Automated tests show human incident lines use `inc <incident-key>:` for both `get pi --with-incidents` and `walk pi --with-incidents`.
- **SC-005**: Automated tests show rows marked `inc!` with no direct incident details include a short note.
- **SC-006**: Automated tests show list output emits no more than one tree-inspection warning for indirect incident markers.
- **SC-007**: Automated tests show `--incident-message-limit <chars>` truncates long human incident messages and appends `...`.
- **SC-008**: Automated tests show JSON output keeps full incident messages.
- **SC-009**: Automated tests show `--with-incidents` remains invalid with `--total`.
- **SC-010**: Automated tests show `--incident-message-limit` is invalid without `--with-incidents` and rejects negative values.
- **SC-011**: Automated tests show paging and `--limit` behavior remain compatible with incident rendering.
- **SC-012**: Relevant process-instance command tests and walk incident output tests pass.

## Assumptions

- The affected command surfaces remain `c8volt get process-instance`, `c8volt get pi`, `c8volt walk process-instance`, and `c8volt walk pi`.
- Existing keyed incident enrichment is the source of truth for incident lookup shape and association.
- Existing list/search selectors continue to define which process instances are included before incident detail rendering is applied.
- The direct incident lookup can return an empty list for a process instance that still carries an incident marker.
- The indirect incident note should be short per row, while the longer tree-inspection guidance belongs in the de-duplicated list warning.
- Documentation and help text should be updated through the repository's existing command metadata and generation path where applicable.
