# Feature Specification: Day-Based Process Instance Date Filters

**Feature Branch**: `090-process-instance-date-filters`  
**Created**: 2026-04-06  
**Status**: Draft  
**Input**: User description: "GitHub issue #90: feat(processinstance): extend c8volt get process-instance with day-based start/end date filters"

## GitHub Issue Traceability

- **Issue Number**: 90
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/90
- **Issue Title**: feat(processinstance): extend c8volt get process-instance with day-based start/end date filters

## Clarifications

### Session 2026-04-06

- Q: How should `--end-date-*` filters treat process instances that do not have an `endDate`? → A: Exclude instances with no `endDate` from any `--end-date-*` filtered result.
- Q: Which calendar day interpretation should date-only filters use? → A: Interpret filter dates in the configured Camunda environment's local day.
- Q: Should the new date filters apply to direct process-instance lookup by ID, or only to list/search behavior? → A: Date filters are valid only for list/search usage, not ID-based lookup.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Filter by Start Date (Priority: P1)

As a CLI user, I want to narrow `c8volt get process-instance` results by process instance start date so I can find process instances that began within a retention-relevant day range.

**Why this priority**: Start-date filtering is the core user value described in the issue and delivers immediate value even without end-date filtering.

**Independent Test**: Can be fully tested by running the command with one or both start-date flags on v8.8 and verifying that only process instances whose start date falls within the inclusive bounds are returned.

**Acceptance Scenarios**:

1. **Given** a v8.8 environment with process instances that started on different calendar days, **When** the user runs `c8volt get process-instance --start-date-after 2026-01-01`, **Then** only process instances with a start date on or after 2026-01-01 are returned.
2. **Given** a v8.8 environment with process instances that started on different calendar days, **When** the user runs `c8volt get process-instance --start-date-before 2026-01-31`, **Then** only process instances with a start date on or before 2026-01-31 are returned.
3. **Given** a v8.8 environment with process instances both inside and outside a target date window, **When** the user runs `c8volt get process-instance --start-date-after 2026-01-01 --start-date-before 2026-01-31`, **Then** only process instances whose start date falls between 2026-01-01 and 2026-01-31 inclusive are returned.

---

### User Story 2 - Filter by End Date and Combine with Existing Filters (Priority: P2)

As a CLI user, I want end-date filters to work alongside existing search filters so I can narrow results without losing current command behavior.

**Why this priority**: End-date filtering is part of the requested scope, and preserving existing filter behavior avoids regression in established workflows.

**Independent Test**: Can be fully tested by running the command on v8.8 with end-date flags alone and in combination with an existing filter such as state, then verifying the result set reflects both constraints.

**Acceptance Scenarios**:

1. **Given** a v8.8 environment with completed process instances ending on different calendar days, **When** the user runs `c8volt get process-instance --end-date-after 2026-02-01`, **Then** only process instances with an end date on or after 2026-02-01 are returned.
2. **Given** a v8.8 environment with completed process instances ending on different calendar days, **When** the user runs `c8volt get process-instance --end-date-before 2026-03-31`, **Then** only process instances with an end date on or before 2026-03-31 are returned.
3. **Given** a v8.8 environment with process instances that match different states and date combinations, **When** the user runs the command with an existing filter and one or more date filters, **Then** the command continues to enforce the existing filter while also narrowing results by the provided date bounds.

---

### User Story 3 - Get Clear Errors for Unsupported or Invalid Input (Priority: P3)

As a CLI user, I want invalid date input and unsupported version usage to fail clearly so I can correct the command or understand the version limitation immediately.

**Why this priority**: Clear failure behavior is necessary for trust in the CLI and is explicitly required for both validation and v8.7 compatibility handling.

**Independent Test**: Can be fully tested by running the command with malformed dates, inverted ranges, and any new date flag on v8.7, then verifying that the returned error is clear and follows the existing command error path.

**Acceptance Scenarios**:

1. **Given** a user provides a date that is not valid date-only input, **When** the command is executed, **Then** the command fails before search execution with a clear validation error identifying the invalid value.
2. **Given** a user provides both `after` and `before` for the same field and the `after` date is later than the `before` date, **When** the command is executed, **Then** the command fails before search execution with a clear validation error describing the invalid range.
3. **Given** the configured Camunda version is v8.7, **When** the user supplies any of the new date flags, **Then** the command returns a clear not-implemented result through the existing error model and does not execute the date-filtered search.

### Edge Cases

- A process instance whose start date or end date falls exactly on the provided boundary date must be included in the results.
- A process instance with no `endDate` must be excluded whenever an `--end-date-after` or `--end-date-before` filter is supplied.
- Date-only comparisons must use the configured Camunda environment's local calendar day so results do not shift based on the machine running `c8volt`.
- Date filters must not be accepted for direct process-instance lookup by ID, because they only narrow collection searches.
- Providing only one bound for a given field must still narrow results correctly without requiring the paired bound.
- Start-date bounds and end-date bounds must be validated independently so an invalid range for one field cannot be masked by valid input for the other.
- Existing filters that already reduce the result set must continue to apply even when date filters are also present.
- On v8.7, any one of the four new flags must trigger the unsupported-version response, even if the other command arguments are otherwise valid.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST extend `c8volt get process-instance` with four new date-only flags: `--start-date-after`, `--start-date-before`, `--end-date-after`, and `--end-date-before`.
- **FR-002**: The system MUST allow the new date-only flags only for list/search behavior of `c8volt get process-instance`, not for direct lookup by ID.
- **FR-003**: The system MUST treat each `*-date-after` flag as an inclusive lower bound for its corresponding process instance date field.
- **FR-004**: The system MUST treat each `*-date-before` flag as an inclusive upper bound for its corresponding process instance date field.
- **FR-005**: When both `after` and `before` are provided for the same date field, the system MUST treat them as an inclusive range and return only results within that range.
- **FR-006**: The system MUST accept date filter values only as valid date-only input at day granularity.
- **FR-007**: The system MUST interpret date-only filter values using the configured Camunda environment's local calendar day.
- **FR-008**: The system MUST validate each date field independently before executing search behavior.
- **FR-009**: When both `after` and `before` are provided for the same date field, the system MUST reject the command if the `after` value is later than the `before` value.
- **FR-010**: When any new date filter value is invalid, the system MUST return a clear validation error before executing the search.
- **FR-011**: On v8.8, the system MUST apply the requested date filters to process-instance search results using the defined inclusive semantics.
- **FR-012**: On v8.8, the system MUST exclude process instances with no `endDate` whenever one or more `--end-date-*` filters are provided.
- **FR-013**: On v8.8, the system MUST combine date filters with existing process-instance filters as additional narrowing constraints without changing the current behavior of existing filters.
- **FR-014**: On v8.7, the system MUST return a clear not-implemented result through the existing error path whenever any of the new date flags is used.
- **FR-015**: The system MUST preserve current command behavior when none of the new date flags is provided.
- **FR-016**: The system MUST include automated coverage for valid v8.8 filtering, inclusive boundary behavior, invalid date formats, invalid ranges, and v8.7 not-implemented responses.

### Key Entities *(include if feature involves data)*

- **Process Instance Search Request**: The user-supplied set of command filters used to retrieve process instances, including existing filters and the new optional start-date and end-date bounds.
- **Date Filter Bound**: A single date-only boundary provided by the user that narrows either the start date or end date dimension using inclusive semantics.
- **Process Instance Result Set**: The returned collection of process instances after all supported filters and validation rules have been applied.
- **Version Capability Rule**: The version-specific behavior that determines whether the new date filters are supported or must return a not-implemented response.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can retrieve process instances by inclusive start-date bounds on v8.8 with correct results for lower bound only, upper bound only, and combined range inputs.
- **SC-002**: Users can retrieve process instances by inclusive end-date bounds on v8.8, with instances lacking an `endDate` excluded whenever end-date filters are used, and the result set remains narrowed correctly when existing filters are also supplied.
- **SC-003**: Every invalid date format or inverted date range submitted through the new flags is rejected before search execution with a clear validation failure.
- **SC-004**: Every use of the new date flags on v8.7 returns a clear not-implemented result through the existing error path.
- **SC-005**: Automated tests cover the new v8.8 success paths, inclusive boundary cases, validation failures, and v8.7 unsupported-version behavior.

## Assumptions

- The command continues to use the existing process-instance search flow and error-model conventions rather than introducing a separate command path.
- Date-only input uses the same calendar interpretation for all four new flags and does not require time-of-day input from users.
- The configured Camunda environment defines the local calendar day used for inclusive date comparisons.
- Existing non-date filters, including state-based filtering, remain in scope and must retain their current semantics.
- The requested feature is limited to `c8volt get process-instance` behavior and its associated test coverage.
