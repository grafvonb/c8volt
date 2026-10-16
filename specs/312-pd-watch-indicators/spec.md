# Feature Specification: Process Definition Watch Change Indicators

**Feature Branch**: `312-pd-watch-indicators`

**Created**: 2026-10-16

**Status**: Draft

**Input**: [GitHub issue #312](https://github.com/grafvonb/c8volt/issues/312): add row change indicators to `get pd --watch`.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Spot changes between refreshes (Priority: P1)

As an operator watching a long process-definition list, I want changed and newly appearing definitions identified against the last successful refresh so I can find relevant activity without remembering the previous screen, including after temporary refresh failures.

**Why this priority**: Identifying changes is the core purpose of the feature.

**Independent Test**: Observe a sequence of successful refreshes containing unchanged, updated, added, removed, and reordered definitions and verify the markers and summary against the preceding successful refresh, including failed/partial refreshes, recovery, and empty results.

**Acceptance Scenarios**:

1. **Given** a newly started watch session, **When** its first refresh succeeds, **Then** all rows have blank markers and the summary reports zero updated, new, and removed definitions.
2. **Given** a successful baseline, **When** an existing definition changes one or more displayed values, **Then** its row has `*` and it contributes exactly one to the updated count.
3. **Given** a successful baseline, **When** a previously absent key appears, **Then** its row has `+` and contributes exactly one to the new count, without also counting as updated.
4. **Given** a successful baseline, **When** a key disappears from the next successful result, **Then** its row is absent and the summary counts it as removed.
5. **Given** unchanged definitions returned in a different order or with different alignment caused by another row, **When** the view refreshes, **Then** no false changes are marked and existing stable row ordering is preserved.
6. **Given** one updated, one new, and no removed definitions, **When** the list is displayed, **Then** the summary above it reads `HH:MM:SS.mmm INFO 1 updated, 1 new, 0 removed`.
7. **Given** rows marked on a previous refresh, **When** the next successful refresh has the same displayed values and keys, **Then** all markers clear and all change counts are zero.

8. **Given** an established baseline, **When** one or more refreshes fail and a later refresh succeeds, **Then** markers and counts compare the recovered result with that baseline, ignoring failed or partial results.
9. **Given** no successful refresh yet, **When** initial refreshes fail before one succeeds, **Then** that first successful result establishes an unmarked baseline with zero change counts.
10. **Given** an established nonempty baseline, **When** a successful refresh is empty, **Then** all prior definitions are counted as removed and no stale rows remain; a subsequent unchanged empty refresh reports zero changes.

### User Story 2 - Read indicators with or without styling (Priority: P2)

As an operator, I want a compact, self-explanatory watch view that highlights changed values where supported while remaining understandable without terminal styling.

**Why this priority**: Styling improves scanning, while text markers make the information accessible in all supported watch environments.

**Independent Test**: Compare equivalent watch refreshes with styling enabled and unavailable, then exercise ordinary process-definition output and consult command documentation.

**Acceptance Scenarios**:

1. **Given** an existing row with changed displayed values and terminal styling available, **When** it is rendered, **Then** changed values are bold and the row retains its `*` marker; unchanged values are not newly highlighted.
2. **Given** styling unavailable or disabled by existing behavior, **When** changes are rendered, **Then** the trailing markers and timestamped summary convey the same classifications and counts without bold styling.
3. **Given** ordinary `get pd` use, **When** any supported output mode is selected, **Then** its existing output is preserved without watch markers or the change summary.
4. **Given** an operator consulting command documentation, **When** they read the watch guidance, **Then** it explains markers, bold values, summary counts, first-refresh behavior, and comparison with the previous successful refresh without requiring additional flags.

### Edge Cases

- A successful empty first result establishes a valid baseline; definitions appearing afterward are new.
- A removed key that reappears after a successful refresh without it is new relative to that refresh.
- Multiple changed values in one row count as one updated definition.
- Changes to undisplayed information, spacing, styling, or row position do not count as value changes.
- Displayed optional values appearing or disappearing count as changes for an existing key, including optional statistics.
- A failed refresh with partial discovery must not cause additions or removals to be inferred.
- A fresh watch session does not inherit change history from an earlier session.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Watch output MUST append a marker after the last displayed value: `*` for an existing key with changed displayed values, `+` for a key absent from the preceding successful refresh, and no suffix otherwise.
- **FR-002**: The first successful refresh of each watch session MUST establish the baseline with blank markers and zero change counts, including when the result is empty.
- **FR-003**: Comparisons MUST match definitions by key and compare displayed values independently of row position, padding, alignment, and styling. Changes to undisplayed information MUST NOT produce indicators.
- **FR-004**: The watch view MUST preserve existing stable row ordering; change categories MUST NOT reorder rows.
- **FR-005**: Each successful refresh MUST log a timestamped INFO summary to configured stderr after repaint and before the rows on stdout in the form `HH:MM:SS.mmm INFO <updated> updated, <new> new, <removed> removed`. Counts MUST represent definitions, with new and updated categories mutually exclusive.
- **FR-006**: Removed definitions MUST contribute to the summary but MUST NOT remain as stale rows in the current list.
- **FR-007**: Markers, highlights, and counts MUST describe only changes since the preceding successful refresh. An unchanged successful refresh MUST clear prior indicators and report zero counts.
- **FR-008**: Failed or incomplete refreshes MUST NOT replace the baseline or report changes derived from partial results. Existing failure handling and retry behavior MUST remain intact.
- **FR-009**: When terminal styling is available under the watch output styling policy, changed displayed values of existing definitions MUST be bold. Without styling, markers and the summary MUST remain sufficient to identify changed and new rows and the number removed.
- **FR-010**: Indicators MUST apply only to `get pd --watch`. Ordinary process-definition output, supported output modes, watch eligibility restrictions, selectors, and refresh controls MUST retain their existing behavior. No new flags or configurable highlight duration may be introduced.
- **FR-011**: Command documentation and relevant user guidance MUST explain the indicators and their refresh semantics, including an example summary.
- **FR-012**: Acceptance validation MUST cover baseline, unchanged, updated, added, removed, reordered, and failed-refresh sequences, as well as empty results, marker clearing, styling availability, and ordinary-output compatibility.

### Key Entities

- **Displayed definition**: A process definition identified by its key and the values currently visible to the operator, including optional displayed statistics.
- **Successful refresh baseline**: The most recent complete successful result in the current watch session, including an empty result, against which the next successful result is compared.
- **Refresh change summary**: Counts of updated, new, and removed definitions for one comparison, without an inline marker legend.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In every defined acceptance sequence, 100% of current rows receive the expected marker and all three summary counts exactly match the observed definition changes.
- **SC-002**: Reordering, alignment changes, and changes to undisplayed information produce zero false updated indicators.
- **SC-003**: After any tested sequence of failed refreshes, the next successful refresh correctly identifies all changes from the last successful baseline; failures produce zero false baseline advances.
- **SC-004**: Operators can identify updated and new rows and the number of removals from the current watch view alone, both with and without styling, without consulting the prior screen.
- **SC-005**: An unchanged successful refresh clears 100% of prior markers and highlights, and a successful removal leaves zero stale rows.
- **SC-006**: All existing ordinary process-definition output examples remain unchanged; documented watch examples explain every marker and summary category without additional configuration.

## Assumptions

- The existing watch command supplies complete successful results and defines stable ordering, retry behavior, and output-mode eligibility; this feature depends on those behaviors. Bold styling is a presentation enhancement whose terminal-capability policy is defined in the plan.
- Displayed values mean the information already shown in process-definition rows, including optional statistics. Adding new business fields is outside scope.
- New rows use `+` as their indicator; bold value highlighting is reserved for changed values of keys present in both compared successful results.
- A disappeared optional value is indicated by the row marker and updated count even when no replacement value exists to render in bold.
- The zero-count summary remains visible on baseline and unchanged successful refreshes, including empty results.
- Indicators represent changes in the watched result, not proof of deployment or deletion in the underlying environment. Persistent history, notifications, additional watch modes, and configurable highlight timing are outside scope.
