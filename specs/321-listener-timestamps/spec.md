# Feature Specification: Consistent Listener Timestamps

**Feature Branch**: `codex/321-listener-timestamps`

**Created**: 2026-09-16

**Status**: Draft

**Input**: User description: "https://github.com/grafvonb/c8volt/issues/321 — fix(cli): show listener start and end timestamps consistently"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Distinguish Listener Creation, End, and Deadline (Priority: P1)

An operator inspecting listener jobs can tell when a job was created and when it ended, without mistaking an activation deadline for completion time.

**Why this priority**: Showing a deadline on a completed job while omitting its actual timestamps misrepresents the listener timeline during troubleshooting.

**Independent Test**: Inspect an element with listener jobs in completed, activated, canceled, and other non-active states, with distinct creation, end, and deadline values, and verify the meaning and visibility of each timestamp.

**Acceptance Scenarios**:

1. **Given** a completed listener with creation time, end time, and a retained deadline, **When** the operator runs `get element --with-listeners`, **Then** its row shows creation time as `s:` and end time as `e:`, and does not show `d:`.
2. **Given** an `ACTIVATED` listener with creation time and a deadline but no end time, **When** its row is displayed, **Then** it shows `s:` and `d:` and omits `e:`.
3. **Given** a canceled or other non-active listener with a retained deadline, **When** its row is displayed, **Then** it omits `d:` and shows only the available creation and end timestamps.
4. **Given** a listener with only one or none of its timestamps available, **When** its row is displayed, **Then** unavailable timestamp tags are omitted, available timestamps retain their meaning, and no deadline is substituted for an end time.

---

### User Story 2 - Read the Same Timeline Across Investigation Commands (Priority: P1)

An operator switching between element inspection, process-instance inspection, process-tree walking, and slow-process analysis sees the same listener timestamp semantics and familiar time formatting.

**Why this priority**: Operators must be able to correlate the same job across investigation views without reinterpreting its timestamps.

**Independent Test**: View the same listener data through all four affected commands using the existing timezone and timestamp display settings and compare timestamp tags and values.

**Acceptance Scenarios**:

1. **Given** the same listener data, **When** the operator uses `get element --with-listeners`, `get process-instance --with-elements --with-listeners`, `walk process-instance --with-elements --with-listeners`, or `ops analyse slow-process-instances --with-listeners`, **Then** all four commands apply the same `s:`, `e:`, and activated-only `d:` rules.
2. **Given** an existing timezone or timestamp display setting, **When** listener timestamps are shown in any affected command, **Then** they follow that setting and the existing timestamp formatting conventions.
3. **Given** unchanged process and element data, **When** the operator requests listener details after this change, **Then** process and element durations and slow-process analysis results remain unchanged.
4. **Given** an operator consulting command documentation, **When** they read the listener timestamp guidance, **Then** it explains that `s:` is job creation rather than worker execution start, `e:` is job end, and `d:` is an activated-job deadline, with a completed-listener example.

---

### User Story 3 - Retain Available Timestamps for Programmatic Consumers (Priority: P2)

A user consuming job data programmatically receives the available creation and end timestamps, including in listener-enriched JSON results, and can distinguish missing data from recorded times.

**Why this priority**: Correct human rendering depends on retaining the data, and automation needs the same facts for timeline analysis.

**Independent Test**: Retrieve jobs with distinct creation and end times through the public job interface and supported JSON output, then repeat with each timestamp absent and with a supported environment that does not supply those fields.

**Acceptance Scenarios**:

1. **Given** job data containing creation and end times, **When** it is retrieved through the public job interface or included in an affected command's supported JSON output, **Then** both times are preserved as the same instants, with JSON fields named `creationTime` and `endTime`.
2. **Given** one or both timestamps are unavailable, including due to connected-version differences, **When** otherwise supported job or listener retrieval succeeds, **Then** available data remains accessible, absent timestamps are omitted from JSON, and no fabricated timestamp is returned.
3. **Given** a non-active job with a recorded deadline, **When** its data is returned programmatically, **Then** the existing deadline field is preserved; hiding `d:` applies to human listener rows.

### Edge Cases

- Completed or canceled jobs retain a deadline that differs from the actual end time.
- An activated job has no deadline, or unexpectedly has an end time: omit missing tags and show supplied creation/end times without inventing state-dependent values.
- Only creation time or only end time is present; neither timestamp depends on the other being available.
- A listener has no timestamps at all: retain its ordinary identifying and status fields without empty tags or placeholder times.
- A job has a missing, unfamiliar, or any non-`ACTIVATED` state: suppress the human deadline tag.
- Execution listeners and user task listeners obey identical timestamp rules.
- A supported connected version lacks one or both fields: render available information without manufacturing timestamps or failing solely because those fields are absent.
- No listener jobs are found, or listener lookup itself is unsupported: preserve existing empty-result and unsupported-version behavior.
- Timezone offsets and date boundaries must not change the represented instants or introduce a separate listener-only formatting convention.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Job data MUST retain available creation and end timestamps from retrieval through all intermediate representations to the public job interface and listener-enriched results, without changing the represented instants.
- **FR-002**: Human listener rows MUST label job creation time with `s:` and job end time with `e:` whenever those values are available. `s:` MUST NOT be described as worker execution start.
- **FR-003**: Human listener rows MUST show an available activation deadline with `d:` only when the job state is `ACTIVATED`; all other states MUST omit that tag.
- **FR-004**: Missing timestamps MUST be omitted independently. The product MUST NOT substitute a deadline, current time, or another timestamp for a missing creation or end time.
- **FR-005**: The rules MUST apply consistently to both listener kinds across the four commands listed in User Story 2 and their existing listener-supported modes.
- **FR-006**: Listener timestamps MUST use the existing timezone controls, precision, and timestamp formatting conventions. When multiple timestamp tags are present, they MUST appear in `s:`, `e:`, `d:` order.
- **FR-007**: Supported JSON job representations MUST expose available timestamps as `creationTime` and `endTime`, omit unavailable values, and preserve existing fields and surrounding output contracts, including recorded deadlines regardless of state.
- **FR-008**: Missing timestamp fields in a supported connected version MUST NOT cause otherwise supported retrieval to fail. Existing unsupported listener-lookup behavior MUST remain unchanged.
- **FR-009**: Process and element duration calculations, analysis outcomes, listener selection and grouping, and behavior without listener enrichment MUST remain unchanged, except for the additive job timestamp fields in programmatic results.
- **FR-010**: User-facing command guidance and affected generated CLI documentation MUST explain the timestamp meanings, conditional deadline visibility, and omission of unavailable timestamps, and include a completed-listener example.

### Key Entities *(include if feature involves data)*

- **Listener Job**: An execution-listener or user-task-listener job associated with an element, with a state and independently optional creation time, end time, and activation deadline.
- **Listener Timeline**: The recorded creation and end instants shown to an operator; the activation deadline is a separate deadline, not evidence of completion or worker execution start.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In all four affected commands, 100% of acceptance cases show available creation/end timestamps under the correct tags and show zero deadline tags for non-active listeners.
- **SC-002**: For completed, activated, canceled, and other non-active listener cases, users can identify every available creation/end time directly from the row without treating a deadline as completion evidence.
- **SC-003**: Across presence/absence combinations for creation and end timestamps, 100% of programmatic acceptance cases preserve supplied instants and return zero fabricated timestamps.
- **SC-004**: Comparing identical process and element inputs before and after the change yields zero differences in their calculated durations or slow-process analysis outcomes.
- **SC-005**: All four commands' affected documentation describes the same timestamp grammar, and every displayed listener timestamp follows existing timezone and formatting settings in the acceptance cases.

## Assumptions

- Scope is the listener timestamp correction in issue #321; new commands, new timestamp flags, worker execution timing, and changes to duration calculations are excluded.
- Existing listener lookup availability and supported command modes remain authoritative. Missing timestamp fields do not imply that listener lookup itself is supported on an otherwise unsupported version.
- Creation and end timestamps are independently optional facts supplied by the connected environment; this feature does not infer lifecycle times.
- The activation deadline remains available in existing programmatic job data even when hidden in a non-active human listener row.
- Standalone job human-output redesign is outside scope; shared public job data gains available timestamps as required by the issue.
- Documentation regeneration belongs to implementation when command source guidance changes; this specification establishes the required documentation outcome.
