# Feature Specification: Report Process Definition Incident Statistics

**Feature Branch**: `042-pd-incident-stats`  
**Created**: 2026-04-21  
**Status**: Draft  
**Input**: User description: "GitHub issue #42: `get pd --stat` currently does not report incident counts correctly for process definitions with active incidents"

## GitHub Issue Traceability

- **Issue Number**: 42
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/42
- **Issue Title**: `get pd --stat` currently does not report incident counts correctly for process definitions with active incidents

## Clarifications

### Session 2026-04-21

- Q: What should the displayed "incident count" represent for `get pd --stat`? → A: Count process instances with at least one active incident.
- Q: How should `get pd --stat` render unsupported incident statistics on versions like 8.7? → A: Omit the `in:` field entirely.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Show Correct Incident Counts (Priority: P1)

As an operator inspecting process definitions, I want `get pd --stat` to show how many process instances currently have at least one active incident when the platform version can provide that number reliably so that I can see process-definition health without cross-checking separate commands.

**Why this priority**: The issue exists because the current output hides real active incidents behind `in:-` even when reliable incident-bearing process-instance information is already available elsewhere.

**Independent Test**: Run `get pd --stat` for a process definition that currently has incident-bearing process instances on a supported version and verify the incident field reports that process-instance count while the other statistics fields remain unchanged.

**Acceptance Scenarios**:

1. **Given** a supported version where reliable incident statistics are available and a process definition has process instances with active incidents, **When** the operator runs `get pd --stat`, **Then** the output includes an `in:` field whose value is the number of process instances that currently have at least one active incident.
2. **Given** a supported version where reliable incident statistics are available and a process definition has no process instances with active incidents, **When** the operator runs `get pd --stat`, **Then** the output includes an `in:0` field or the version-consistent zero-count representation rather than omitting the field.
3. **Given** a process definition where other statistics such as active and completed counts are available, **When** the incident-bearing process-instance count is added, **Then** the existing formatting and values for the other statistics fields remain unchanged.

---

### User Story 2 - Preserve Version-Specific Truthfulness (Priority: P2)

As a maintainer, I want unsupported platform versions to omit the incident field entirely rather than guessing so that the CLI never reports misleading process-definition incident-bearing instance data.

**Why this priority**: A wrong incident count is worse than no count. The issue explicitly requires version-sensitive behavior, especially for versions where the count cannot be derived reliably.

**Independent Test**: Run `get pd --stat` against a version that cannot reliably derive the number of process instances with active incidents and verify the command omits the `in:` field instead of inventing or backfilling a value.

**Acceptance Scenarios**:

1. **Given** version 8.7 cannot provide a reliable count of process instances with active incidents for a process definition, **When** the operator runs `get pd --stat`, **Then** the output omits the `in:` field rather than showing a potentially wrong number or unsupported placeholder.
2. **Given** supported and unsupported platform versions are both exercised in tests, **When** maintainers compare the outputs, **Then** only the supported versions expose the `in:` field and unsupported versions omit it.

---

### User Story 3 - Verify Version Coverage With Tests (Priority: P3)

As a maintainer, I want automated coverage for the version-specific incident-statistics behavior so that future client or service changes do not silently regress `get pd --stat`.

**Why this priority**: The feature depends on version-specific behavior and data-source differences, which are easy to break without targeted tests.

**Independent Test**: Run the relevant automated tests and verify they cover supported-version incident-bearing process-instance counts, unsupported-version fallback behavior, and preservation of the rest of the statistics output.

**Acceptance Scenarios**:

1. **Given** automated tests for supported versions, **When** the command gathers process-definition statistics, **Then** the tests confirm the reported incident value matches the number of process instances that currently have at least one active incident according to the version’s reliable statistics sources.
2. **Given** automated tests for unsupported versions, **When** the command cannot derive that process-instance count reliably, **Then** the tests confirm the `in:` field is omitted from the output.

### Edge Cases

- A process definition may have active process instances but no active incidents, and the incident field should reflect zero incident-bearing process instances rather than being omitted when the version can report the count reliably.
- A process definition may have active incidents while the existing process-definition statistics source omits that field, requiring the command to combine reliable statistics from other supported-version sources without changing the rest of the output contract.
- The reliable count of process instances with active incidents may be unavailable for one version even though other process-definition statistics remain available, and the command must preserve the other statistics fields while omitting the unsupported `in:` field.
- Temporary empty responses, partial statistics, or unavailable incident-related data must not cause the command to print an incorrect incident number or a misleading unsupported placeholder.
- Multiple active incidents on the same process instance must not inflate the displayed count beyond one for that process instance.
- Multiple process instances with incidents under the same process definition must aggregate into the displayed process-definition count consistently.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow `get pd --stat` to report how many process instances for a process definition currently have at least one active incident on platform versions where that count can be derived reliably from supported statistics or incident data.
- **FR-002**: The system MUST keep the existing `get pd --stat` output format unchanged except for including the `in:` field only when a reliable incident-bearing process-instance count is available.
- **FR-003**: For platform version 8.8, the system MUST report the current number of process instances with at least one active incident for process definitions when the command is run with `--stat`.
- **FR-004**: For platform version 8.9, the system MUST report the current number of process instances with at least one active incident for process definitions when the command is run with `--stat`.
- **FR-005**: For platform version 8.7, if the process-instance count cannot be derived reliably, the system MUST omit the `in:` field rather than displaying an inferred or incorrect number.
- **FR-006**: The reported incident value for supported versions MUST stay consistent with the reliable incident-related data available for the same process definition at the time of the command and must count each affected process instance at most once.
- **FR-007**: The system MUST preserve the existing values and formatting for the other reported process-definition statistics fields when the `in:` field is present or omitted.
- **FR-008**: The system MUST handle process definitions with zero process instances that have active incidents on supported versions without omitting the `in:` field.
- **FR-009**: The system MUST avoid implying incident-count support on versions where the required data cannot be obtained reliably by omitting the `in:` field entirely.
- **FR-010**: The system MUST add or update automated tests that cover supported-version incident-bearing process-instance counts, unsupported-version fallback behavior, and preservation of the surrounding statistics output.

### Key Entities *(include if feature involves data)*

- **Process Definition Statistics View**: The formatted `get pd --stat` output row that summarizes process-definition statistics and may include the `in:` field when incident statistics are supported.
- **Incident Count**: The current number of process instances for a process definition that each have at least one active incident when the platform version can provide it reliably.
- **Supported Version Statistics Source**: A version-appropriate source of process-definition or incident-related data that can be trusted to populate the incident-bearing process-instance count.
- **Unsupported Version Behavior**: The command behavior for platform versions where the incident count cannot be derived reliably and the `in:` field must be omitted.
- **Version-Specific Test Case**: Automated coverage that verifies the incident field behavior for one supported or unsupported platform version.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On platform versions 8.8 and 8.9, `get pd --stat` includes an `in:` field showing the current number of process instances with active incidents for a process definition.
- **SC-002**: On platform version 8.7, `get pd --stat` omits the `in:` field unless the affected-process-instance count can be derived reliably.
- **SC-003**: For supported versions, the displayed incident value matches the reliable incident-related data returned for the same process definition in automated test coverage, with each affected process instance counted at most once.
- **SC-004**: Existing non-incident statistics fields in `get pd --stat` remain unchanged in output format and meaning after the feature is implemented, regardless of whether the `in:` field is present.
- **SC-005**: Automated tests fail if a future change causes supported versions to lose the incident-bearing process-instance count or unsupported versions to display guessed counts.

## Assumptions

- Platform versions 8.8 and 8.9 expose enough reliable incident-related data to support this feature, while version 8.7 may not.
- The `in:` field in `get pd --stat` is intended to represent the number of process instances that currently have at least one active incident for the process definition at command execution time and should be omitted when unsupported.
- Existing command formatting and surrounding statistics semantics should remain stable unless the issue explicitly requires otherwise.
- The repository already has an established place to add version-focused tests for process-definition statistics behavior.
- Downstream implementation work for this feature must keep Conventional Commit formatting and append `#42` as the final token of every commit subject.
