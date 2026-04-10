# Feature Specification: Relative Day-Based Process-Instance Date Shortcuts

**Feature Branch**: `095-processinstance-day-filters`  
**Created**: 2026-04-10  
**Status**: Draft  
**Input**: User description: "GitHub issue #95: feat(processinstance): add relative day-based start/end shortcut flags to get, cancel, and delete"

## GitHub Issue Traceability

- **Issue Number**: 95
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/95
- **Issue Title**: feat(processinstance): add relative day-based start/end shortcut flags to get, cancel, and delete

## Clarifications

### Session 2026-04-10

- Q: How should `cancel process-instance` and `delete process-instance` behave when explicit `--key` values are combined with one or more `--start-*-days` or `--end-*-days` flags? → A: Treat the combination as an invalid command.
- Q: How should relative `--end-*-days` filters treat process instances that do not have an `endDate`? → A: Exclude instances with no `endDate` whenever `--end-date-before-days` or `--end-date-after-days` is used.
- Q: Which calendar day interpretation should relative day-based filters use? → A: Derive relative day boundaries using the configured Camunda environment's local calendar day.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Filter Get Results by Relative Day Offsets (Priority: P1)

As a CLI user, I want `c8volt get process-instance` to accept relative day-based date shortcut flags so I can find process instances from a recent or older day window without calculating calendar dates manually.

**Why this priority**: `get process-instance` is the primary read-only workflow for process-instance discovery, and adding the shortcuts there delivers the core user value with the lowest operational risk.

**Independent Test**: Can be fully tested on v8.8 by running `c8volt get process-instance` with one or more `--start-*-days` or `--end-*-days` flags and verifying that the resulting instances match the same inclusive day window that would be produced by the corresponding absolute date filters.

**Acceptance Scenarios**:

1. **Given** a v8.8 environment with process instances across multiple recent and older days, **When** the user runs `c8volt get process-instance --start-date-after-days 7`, **Then** only process instances whose start date is on or after the derived day boundary for seven days ago are returned.
2. **Given** a v8.8 environment with process instances across multiple recent and older days, **When** the user runs `c8volt get process-instance --start-date-after-days 30 --start-date-before-days 7`, **Then** only process instances whose start date falls within the inclusive derived day range are returned.
3. **Given** a v8.8 environment with completed process instances that ended on different days, **When** the user runs `c8volt get process-instance --end-date-before-days 14`, **Then** only process instances whose end date is on or before the derived boundary for fourteen days ago are returned.

---

### User Story 2 - Use the Same Relative Filters for Search-Based Cancel and Delete (Priority: P2)

As a CLI user, I want `c8volt cancel process-instance` and `c8volt delete process-instance` to support the same relative day-based shortcut flags as `get process-instance` so I can manage matching instances consistently across search-based workflows.

**Why this priority**: Reusing the same shortcut model across search-based management commands reduces mistakes and keeps the operator experience aligned with the existing date-filter rollout from issues #90 and #93.

**Independent Test**: Can be fully tested on v8.8 by running `cancel process-instance` and `delete process-instance` without explicit keys, combining existing filters with one or more relative day flags, and verifying that only instances within the derived day bounds are selected.

**Acceptance Scenarios**:

1. **Given** a v8.8 environment with active process instances started on different days, **When** the user runs `c8volt cancel process-instance --state active --start-date-before-days 30`, **Then** only active process instances whose start date is on or before the derived thirty-day boundary are selected for cancellation.
2. **Given** a v8.8 environment with completed process instances ending on different days, **When** the user runs `c8volt delete process-instance --end-date-after-days 60 --end-date-before-days 7 --auto-confirm`, **Then** only process instances whose end date falls within that inclusive derived day range are selected for deletion.
3. **Given** a user provides explicit `--key` values together with one or more relative day-based flags for cancel or delete, **When** the command is executed, **Then** the command fails with a clear invalid-combination error instead of attempting key-based selection.

---

### User Story 3 - Receive Clear Validation and Version Responses (Priority: P3)

As a CLI user, I want invalid relative-day input, conflicting filter combinations, and unsupported-version usage to fail clearly so I can trust the commands and correct problems before any search-based action occurs.

**Why this priority**: Clear validation is required to keep bulk process-instance operations safe and to preserve the behavior contract already established for the absolute date filters.

**Independent Test**: Can be fully tested by running the affected commands with negative values, conflicting absolute and relative filters, invalid derived ranges, and any new relative-day flag on v8.7, then verifying that each path fails clearly before a search-based action occurs.

**Acceptance Scenarios**:

1. **Given** a user provides a negative value to any `*-days` flag, **When** the command is executed, **Then** the command fails with a clear validation error identifying the invalid non-negative day requirement.
2. **Given** a user mixes `--start-date-after-days` with `--start-date-after` or any other relative and absolute flag pair for the same field, **When** the command is executed, **Then** the command fails with a clear validation error instead of choosing one input silently.
3. **Given** the configured Camunda version is v8.7, **When** the user supplies any of the new relative day-based flags to `get process-instance`, `cancel process-instance`, or `delete process-instance`, **Then** the command returns a clear not-implemented response through the existing error model and does not execute a date-filtered search.

### Edge Cases

- A derived date boundary that lands exactly on a process instance start date or end date must include that instance, matching the existing inclusive absolute-date semantics.
- If both `before` and `after` relative day flags are provided for the same field, the derived inclusive range must be validated after conversion and rejected when the lower bound would be later than the upper bound.
- Relative day flags must continue to compose with existing non-date search filters as additional narrowing constraints rather than replacing current behavior.
- Direct key-based behavior for `cancel process-instance` and `delete process-instance` must remain unchanged when no relative day flags are provided, but any combination of explicit `--key` values with relative day flags must be rejected.
- A process instance with no `endDate` must be excluded whenever `--end-date-before-days` or `--end-date-after-days` is used.
- Mixing an absolute date flag and a relative day flag for the same field must be rejected even if the two inputs would point to the same calendar day.
- A value of `0` days must be accepted and interpreted using the configured Camunda environment's local calendar day, matching the existing absolute date filters.
- Any use of the new relative day-based flags on v8.7 must trigger the unsupported-version response, even if the other command arguments are otherwise valid.
- Commands outside `get process-instance`, `cancel process-instance`, and `delete process-instance` remain out of scope for this feature.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST extend `c8volt get process-instance` with `--start-date-before-days`, `--start-date-after-days`, `--end-date-before-days`, and `--end-date-after-days`.
- **FR-002**: The system MUST extend `c8volt cancel process-instance` with `--start-date-before-days`, `--start-date-after-days`, `--end-date-before-days`, and `--end-date-after-days`.
- **FR-003**: The system MUST extend `c8volt delete process-instance` with `--start-date-before-days`, `--start-date-after-days`, `--end-date-before-days`, and `--end-date-after-days`.
- **FR-004**: The system MUST translate each new relative day-based flag into the corresponding existing absolute date filter introduced by issues #90 and #93 before search-based filtering is applied.
- **FR-005**: The system MUST treat `--start-date-before-days N` as an inclusive upper bound derived from the local calendar day for today minus `N` days.
- **FR-006**: The system MUST treat `--start-date-after-days N` as an inclusive lower bound derived from the local calendar day for today minus `N` days.
- **FR-007**: The system MUST treat `--end-date-before-days N` as an inclusive upper bound derived from the local calendar day for today minus `N` days.
- **FR-008**: The system MUST treat `--end-date-after-days N` as an inclusive lower bound derived from the local calendar day for today minus `N` days.
- **FR-009**: When both `before` and `after` relative day flags are provided for the same field, the system MUST combine them into an inclusive derived date range.
- **FR-010**: The system MUST accept relative day values only as valid non-negative integers.
- **FR-011**: The system MUST reject any command invocation that mixes a relative day flag and an absolute date flag for the same field with a clear validation error.
- **FR-012**: The system MUST derive all relative day-based boundaries using the configured Camunda environment's local calendar day.
- **FR-013**: The system MUST validate the derived start-date bounds and derived end-date bounds independently before any search-based action is executed.
- **FR-014**: The system MUST reject the command with a clear validation error when the derived lower bound for a field is later than the derived upper bound for that same field.
- **FR-015**: On v8.8, the system MUST apply the derived date filters to search-based process-instance selection using the same inclusive day granularity already defined for the existing absolute date filters.
- **FR-016**: On v8.8, the system MUST exclude process instances with no `endDate` whenever one or more `--end-date-before-days` or `--end-date-after-days` filters are supplied.
- **FR-017**: On v8.8, the system MUST combine the derived date filters with existing process-instance search filters without changing the current behavior of those existing filters.
- **FR-018**: The system MUST reject any `cancel process-instance` or `delete process-instance` invocation that combines explicit `--key` values with one or more relative day-based flags.
- **FR-019**: On v8.8, the system MUST preserve direct `--key` behavior for `cancel process-instance` and `delete process-instance` when none of the new relative day-based flags is used.
- **FR-020**: On v8.7, the system MUST return a clear not-implemented response through the existing error path whenever any of the new relative day-based flags is used with the affected commands.
- **FR-021**: The system MUST preserve current behavior for all affected commands when none of the new relative day-based flags is provided.
- **FR-022**: The system MUST update command help and examples for the affected commands to describe the new relative day-based shortcut flags.
- **FR-023**: The system MUST include automated coverage for happy-path conversion, inclusive boundary behavior, configured-local-day derivation, missing `endDate` exclusion, invalid or negative values, conflicting absolute and relative inputs, invalid derived ranges, invalid `--key` plus relative-day combinations, preserved direct-key behavior, and v8.7 unsupported-version responses.

### Key Entities *(include if feature involves data)*

- **Relative Day Filter Input**: A user-provided non-negative integer that expresses how many days before today a start-date or end-date boundary should be derived.
- **Derived Date Bound**: The absolute day boundary produced from a relative day filter input and used with the same inclusive semantics as the existing absolute date filters.
- **Process Instance Search Request**: The complete set of search criteria for `get process-instance`, `cancel process-instance`, or `delete process-instance`, including existing filters plus any derived date bounds.
- **Command Mode**: The distinction between search-based selection and direct key-based targeting, which determines when relative day flags participate in command behavior.
- **Version Capability Rule**: The version-specific rule that enables the feature on v8.8 and rejects it as not implemented on v8.7.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can run `get process-instance` on v8.8 with any supported relative day-based flag combination and receive results that match the equivalent derived absolute date filters.
- **SC-002**: Users can run `cancel process-instance` and `delete process-instance` on v8.8 with relative day-based flags and existing search filters, and only matching process instances are selected.
- **SC-003**: Users who apply one or more relative end-day filters on v8.8 receive results that exclude process instances with no `endDate`, matching the existing absolute end-date behavior.
- **SC-004**: Users who apply relative day-based filters on v8.8 receive results based on the configured Camunda environment's local calendar day rather than the local timezone of the machine running `c8volt`.
- **SC-005**: Every invalid negative day value, conflicting absolute-plus-relative field combination, and invalid derived range is rejected before any search-based action is executed.
- **SC-006**: Every invalid `--key` plus relative-day flag combination is rejected before any search-based or key-based action is executed.
- **SC-007**: Direct key-based `cancel process-instance` and `delete process-instance` behavior remains unchanged when relative day-based flags are not used.
- **SC-008**: Every use of the new relative day-based flags on v8.7 returns a clear not-implemented response through the existing error path.
- **SC-009**: Automated tests cover the new conversion paths, configured-local-day derivation, missing-`endDate` exclusion, validation failures, invalid `--key` combinations, preserved key-based behavior, and unsupported-version behavior across the affected commands.

## Assumptions

- The absolute date filter behavior introduced by issues #90 and #93 remains the canonical behavior, and this feature only adds convenience inputs that translate into that existing model.
- The local calendar day used for relative-day conversion is the configured Camunda environment's local day, matching the interpretation already defined for the existing absolute date filters.
- Search-based selection remains the only place where the new relative day-based filters influence cancel and delete behavior, and explicit `--key` targeting cannot be combined with those flags.
- Relative end-day shortcuts inherit the same missing-`endDate` exclusion behavior already defined for the absolute end-date filters.
- No new process-instance commands or unrelated filtering models are introduced as part of this feature.
- The three affected commands continue to use the existing command validation and shared error-model conventions.
