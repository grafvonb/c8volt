# Feature Specification: Process Definition Watch Repaint

**Feature Branch**: `268-pd-watch-repaint`

**Created**: 2026-08-05

**Status**: Draft

**Input**: User description: "GitHub issue #268: fix(get): make process-definition watch repaint like Linux watch"

## Clarifications

### Session 2026-08-05

- Q: For default human `--watch`, how should slow-refresh warnings repeat? → A: Warn once per continuous slow-refresh streak, then reset after an on-time refresh.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Watch Repaints One Live View (Priority: P1)

As an operator monitoring process definitions, I want watch mode to refresh a single terminal view instead of appending repeated snapshot blocks, so I can observe current state without scrolling through stale output.

**Why this priority**: This is the primary defect and the main operator value: watch mode should behave like a live monitor rather than a log stream.

**Independent Test**: Run process-definition watch mode for multiple refreshes and verify that the terminal presents one refreshed view whose result body matches the non-watch human output for the same lookup.

**Acceptance Scenarios**:

1. **Given** an operator runs process-definition watch mode, **When** multiple refresh cycles complete, **Then** the visible terminal view is repainted rather than extended with repeated result blocks.
2. **Given** an operator compares a watched refresh with the same non-watch process-definition lookup, **When** both commands display human output, **Then** the watched result body contains the same process-definition rows and summary information as the non-watch command.
3. **Given** watch mode is refreshing, **When** a new snapshot is shown, **Then** the result body does not include watch-only labels such as "snapshot 1", "snapshot 2", or "found" unless those labels are already part of the normal non-watch output.

---

### User Story 2 - Slow Refreshes Are Clear Without Noise (Priority: P2)

As an operator using broad process-definition lookups, I want to know when a refresh takes longer than the configured interval, so I can understand why updates are slower without losing the result view.

**Why this priority**: Slow refreshes are expected for broad lookups and statistics, and operators need an actionable signal without cluttering the monitored output.

**Independent Test**: Run watch mode against a lookup that takes longer than the configured interval and verify that the operator receives a clear warning while result rows remain focused on process-definition data.

**Acceptance Scenarios**:

1. **Given** a watched refresh takes longer than the configured interval, **When** the refresh completes, **Then** the operator is warned that collection exceeded the interval.
2. **Given** slow refreshes continue in default human mode, **When** additional refreshes complete, **Then** warnings remain concise and are not repeated in a way that obscures the result view.
3. **Given** an operator asks for verbose output, **When** refreshes complete, **Then** more detailed per-refresh timing or status information is available outside the result body.

---

### User Story 3 - Watch Keeps Human-Only Boundaries (Priority: P3)

As an automation author or CLI user, I want incompatible machine-oriented output modes to remain rejected with watch mode, so command behavior stays predictable and script-safe.

**Why this priority**: This preserves existing command contracts while improving human watch behavior.

**Independent Test**: Combine process-definition watch mode with each incompatible output or automation mode and verify that the command rejects the combination before producing watched output.

**Acceptance Scenarios**:

1. **Given** an operator combines watch mode with JSON output, **When** the command is invoked, **Then** it rejects the combination and does not start watching.
2. **Given** an operator combines watch mode with keys-only, XML, quiet, or automation output, **When** the command is invoked, **Then** it rejects each combination consistently.
3. **Given** process-definition commands are run without watch mode, **When** human or machine-oriented output is requested, **Then** existing non-watch output remains unchanged.

### Edge Cases

- A refresh takes longer than the configured watch interval.
- Multiple consecutive refreshes exceed the configured interval.
- A refresh fails after prior successful refreshes.
- The terminal cannot be cleared or repainted in the current environment.
- A user combines watch mode with JSON, keys-only, XML, quiet, or automation output.
- The watched lookup includes broad selection or statistics that produce longer-running result collection.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The process-definition watch view MUST repaint a single terminal view on each refresh instead of appending a new result block below previous refreshes.
- **FR-002**: Each watch refresh MUST display the same human-readable process-definition result body that the equivalent non-watch command displays for the same selection and options.
- **FR-003**: The default watch result body MUST NOT include watch-only labels, counters, or summary rows such as snapshot numbering unless the equivalent non-watch command already includes that text.
- **FR-004**: Watch mode MUST remain available only for human-readable output.
- **FR-005**: Watch mode MUST reject combinations with JSON output, keys-only output, XML output, quiet mode, and automation mode before watch output begins.
- **FR-006**: Watch refresh cycles MUST never overlap; a new refresh MUST begin only after the previous refresh has completed.
- **FR-007**: The command MUST measure the duration of each watch refresh.
- **FR-008**: If a refresh duration exceeds the configured watch interval, the command MUST warn the operator that the refresh exceeded the interval.
- **FR-009**: Slow-refresh warnings SHOULD remain outside the process-definition result body whenever the terminal environment allows it.
- **FR-010**: Default human watch mode MUST warn once per continuous slow-refresh streak, suppress repeated warnings while refreshes continue exceeding the interval, and allow a new warning after a refresh completes within the interval.
- **FR-011**: Verbose watch mode MUST provide more detailed timing or refresh-status information than default human mode.
- **FR-012**: Existing non-watch process-definition output MUST remain unchanged for human, JSON, keys-only, XML, quiet, and automation modes.

### Key Entities *(include if feature involves data)*

- **Process Definition Result Body**: The human-readable rows and summaries produced by a process-definition lookup.
- **Watch Refresh Cycle**: One complete collection and rendering pass during watch mode, including duration measurement.
- **Slow Refresh Warning**: An operator-facing notice that a refresh exceeded the configured interval.
- **Output Mode Selection**: The user's selected presentation mode, used to determine whether watch mode is allowed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In an interactive terminal, operators can observe at least three watch refreshes while the visible result area contains only the latest process-definition view.
- **SC-002**: For the same process-definition lookup, the default watched result body matches the non-watch human result body with no additional watch-specific rows in 100% of tested scenarios.
- **SC-003**: Slow refreshes that exceed the configured interval produce a clear operator warning within one completed refresh cycle.
- **SC-004**: Consecutive refreshes never overlap during validation runs where collection time exceeds the configured interval.
- **SC-005**: All incompatible machine-oriented mode combinations are rejected consistently before watch output begins.
- **SC-006**: Existing non-watch output checks continue to pass with no intentional output changes.

## Assumptions

- The feature applies to both `get process-definition` and its `get pd` alias.
- "Normal human output" means the output shown by the equivalent command without `--watch` using the same selectors and display options.
- Slow-refresh warnings may use concise status text in default mode and more detailed timing in verbose mode.
- If terminal repaint support is unavailable, the command should still avoid misleading watch-specific result rows and should communicate limitations clearly.
- Documentation and generated CLI references will be updated during implementation planning if user-visible command behavior or examples change.
