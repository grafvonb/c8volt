# Feature Specification: Extend Process-Instance Management Date Filters

**Feature Branch**: `093-extend-pi-date-filters`  
**Created**: 2026-04-09  
**Status**: Draft  
**Input**: User description: "GitHub issue #93: feat(processinstance): extend date-range filters from #90 to search-based process-instance management commands"

## GitHub Issue Traceability

- **Issue Number**: 93
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/93
- **Issue Title**: feat(processinstance): extend date-range filters from #90 to search-based process-instance management commands

## Clarifications

### Session 2026-04-09

- Q: How should `cancel process-instance` and `delete process-instance` behave when explicit `--key` values are combined with `--start-date-*` or `--end-date-*` flags? → A: Treat the combination as an invalid command.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Cancel by Date-Filtered Search (Priority: P1)

As a CLI user, I want `c8volt cancel process-instance` to accept the same day-based start and end date filters as `get process-instance` so I can cancel matching instances without collecting keys manually first.

**Why this priority**: Date-filtered cancellation is the main operational outcome of the follow-up issue and delivers immediate value for retention and cleanup workflows.

**Independent Test**: Can be fully tested by running `c8volt cancel process-instance` on v8.8 without explicit keys, using one or more date filters, and verifying that only the instances matching the combined search criteria are selected for cancellation.

**Acceptance Scenarios**:

1. **Given** a v8.8 environment with active process instances started on different calendar days, **When** the user runs `c8volt cancel process-instance --state active --start-date-before 2026-03-31`, **Then** only active process instances whose start date is on or before 2026-03-31 are selected for cancellation.
2. **Given** a v8.8 environment with process instances ending on different calendar days, **When** the user runs `c8volt cancel process-instance --end-date-after 2026-01-01 --end-date-before 2026-01-31`, **Then** only process instances whose end date falls within that inclusive date window are selected for cancellation.
3. **Given** a user provides one or more explicit `--key` values together with one or more date filters, **When** the cancel command is executed, **Then** the command fails with a clear invalid-combination error instead of attempting key-based cancellation.

---

### User Story 2 - Delete by Date-Filtered Search (Priority: P2)

As a CLI user, I want `c8volt delete process-instance` to support the same date-range filters during search-based selection so I can delete the correct subset of process instances with the same filtering model used elsewhere.

**Why this priority**: Deletion is another search-driven process-instance management flow already aligned with `get process-instance`, so consistent filtering reduces mistakes and extra command chaining.

**Independent Test**: Can be fully tested by running `c8volt delete process-instance` on v8.8 without explicit keys, combining existing filters with start-date and end-date bounds, and verifying that only matching instances are selected for deletion.

**Acceptance Scenarios**:

1. **Given** a v8.8 environment with completed process instances in multiple date ranges, **When** the user runs `c8volt delete process-instance --state completed --end-date-after 2026-01-01 --end-date-before 2026-01-31 --auto-confirm`, **Then** only completed process instances whose end date falls within the inclusive January 2026 window are selected for deletion.
2. **Given** a v8.8 environment with instances that differ by BPMN process ID, version metadata, and date range, **When** the user runs the delete command with existing filters and one or more date filters, **Then** the selected instances satisfy all provided filters together.
3. **Given** a user supplies no explicit keys and the combined filters match no process instances, **When** the delete command is executed, **Then** the command fails with the same no-matching-instances behavior used by the existing search-based delete flow.

---

### User Story 3 - Preserve Validation and Version Limits (Priority: P3)

As a CLI user, I want the new management-command date filters to fail clearly for invalid input and unsupported versions so I can trust the command results and correct problems quickly.

**Why this priority**: Clear validation and version handling prevents unsafe bulk-management actions and preserves the behavioral contract already introduced in issue #90.

**Independent Test**: Can be fully tested by running the cancel and delete commands with malformed dates, inverted ranges, and date flags on v8.7, then verifying that both commands fail through the existing error path before any management action occurs.

**Acceptance Scenarios**:

1. **Given** a user provides an invalid date-only value to any of the new flags, **When** either search-based management command is executed, **Then** the command fails before searching with a clear validation error.
2. **Given** a user provides both `after` and `before` for the same field and the lower bound is later than the upper bound, **When** either command is executed, **Then** the command fails before searching with a clear invalid-range error.
3. **Given** the configured Camunda version is v8.7, **When** the user supplies any of the new date flags to `cancel process-instance` or `delete process-instance`, **Then** the command returns a clear not-implemented result through the existing error model and does not perform a date-filtered search.

### Edge Cases

- A process instance whose start date or end date falls exactly on the provided boundary date must be included in the selected result set.
- A process instance with no `endDate` must be excluded whenever an `--end-date-after` or `--end-date-before` filter is used during search-based selection.
- Date-only comparisons must use the same calendar-day interpretation already established for issue #90 so management commands and `get process-instance` do not diverge.
- Search-based selection must continue to combine existing filters and new date filters as narrowing constraints rather than replacing current filter behavior.
- Direct key-based invocation must reject any combination of explicit `--key` values with `--start-date-*` or `--end-date-*` flags so users do not assume the date filters narrowed the keyed selection.
- If search-based filters match no process instances, the command must fail without attempting cancellation or deletion.
- On v8.7, any one of the four new date flags must trigger the unsupported-version response even if the remaining command arguments are otherwise valid.
- Commands that are not already search-based process-instance management flows, such as `expect`, `walk`, and `run`, remain out of scope for this feature.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST extend `c8volt cancel process-instance` with `--start-date-after`, `--start-date-before`, `--end-date-after`, and `--end-date-before`.
- **FR-002**: The system MUST extend `c8volt delete process-instance` with `--start-date-after`, `--start-date-before`, `--end-date-after`, and `--end-date-before`.
- **FR-003**: The system MUST apply the new date filters only within the existing search-based selection behavior of the cancel and delete commands.
- **FR-004**: The system MUST reject any command invocation that combines explicit process-instance keys with one or more `--start-date-*` or `--end-date-*` flags.
- **FR-005**: The system MUST treat each `*-date-after` flag as an inclusive lower bound for its corresponding date field.
- **FR-006**: The system MUST treat each `*-date-before` flag as an inclusive upper bound for its corresponding date field.
- **FR-007**: When both `after` and `before` are provided for the same date field, the system MUST treat them as an inclusive range.
- **FR-008**: The system MUST accept date filter values only as valid date-only input at day granularity.
- **FR-009**: The system MUST validate start-date bounds and end-date bounds independently before performing search-based selection.
- **FR-010**: When both `after` and `before` are provided for the same field and `after` is later than `before`, the system MUST reject the command with a clear validation error before any search or management action occurs.
- **FR-011**: When any new date filter value is invalid, the system MUST return a clear validation error before any search or management action occurs.
- **FR-012**: On v8.8, the system MUST apply the requested date filters to process-instance search results for both cancel and delete command flows using the same inclusive semantics defined in issue #90.
- **FR-013**: On v8.8, the system MUST exclude process instances with no `endDate` whenever one or more `--end-date-*` filters are supplied during search-based selection.
- **FR-014**: On v8.8, the system MUST combine date filters with existing process-instance filters, including state and process-definition filters, as additional narrowing constraints without changing current filter behavior.
- **FR-015**: On v8.7, the system MUST return a clear not-implemented result through the existing error path whenever any of the new date flags is used with `cancel process-instance` or `delete process-instance`.
- **FR-016**: The system MUST preserve current command behavior when none of the new date flags is provided.
- **FR-017**: The system MUST update command help and examples for the affected management commands to show the new filter-based selection behavior.
- **FR-018**: The system MUST keep `c8volt expect process-instance`, `c8volt walk process-instance`, and `c8volt run process-instance` out of scope for this feature.
- **FR-019**: The system MUST include automated coverage for v8.8 success paths, inclusive boundary behavior, empty search results, invalid date formats, invalid ranges, invalid `--key` plus date-filter combinations, and v8.7 not-implemented responses for both affected commands.

### Key Entities *(include if feature involves data)*

- **Management Search Request**: The user-supplied filter set used by `cancel process-instance` or `delete process-instance` when explicit keys are not provided.
- **Date Filter Bound**: A single date-only boundary that narrows the process-instance start date or end date using inclusive semantics.
- **Selected Process Instance Set**: The collection of process instances returned by the search flow after all existing filters and new date bounds are applied.
- **Management Command Mode**: The distinction between search-based selection and direct key-based targeting, which determines when the new date filters affect behavior.
- **Version Capability Rule**: The version-specific rule that allows the new date filters on v8.8 and rejects them as not implemented on v8.7.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can cancel process instances on v8.8 by combining existing search filters with inclusive start-date and end-date bounds, and only matching instances are selected.
- **SC-002**: Users can delete process instances on v8.8 by combining existing search filters with inclusive start-date and end-date bounds, and only matching instances are selected.
- **SC-003**: Any attempt to combine explicit process-instance keys with one or more date filters is rejected before any search, cancellation, or deletion occurs.
- **SC-004**: Every invalid date format or inverted date range submitted through the new flags is rejected before any search, cancellation, or deletion occurs.
- **SC-005**: Every use of the new date flags on v8.7 through the affected commands returns a clear not-implemented result through the existing error path.
- **SC-006**: Automated tests cover the new cancel and delete command success paths, validation failures, no-match behavior, inclusive boundary cases, and v8.7 unsupported-version handling.

## Assumptions

- The date-filter semantics defined for issue #90 are the canonical behavior and must be reused unchanged for the affected management commands.
- The affected commands continue to use their current search-and-manage workflow rather than introducing a separate bulk-management model.
- Existing confirmation, dry-run validation, and key-collection behavior remain in place, except that explicit `--key` usage cannot be combined with the new date filters.
- Only commands that already support filter-based process-instance discovery are included in scope for this feature.
- The configured Camunda environment defines the local calendar day used for date-only comparisons.
