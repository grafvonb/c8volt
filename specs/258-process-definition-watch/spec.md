# Feature Specification: Process Definition Watch Mode

**Feature Branch**: `258-process-definition-watch`

**Created**: 2026-08-04

**Status**: Draft

**GitHub Issue**: [#258](https://github.com/grafvonb/c8volt/issues/258) - feat(get): add watch mode for process-definition lookup

**Input**: User description: "https://github.com/grafvonb/c8volt/issues/258"

## Issue Traceability

- **GitHub Issue**: #258
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/258
- **Issue Title**: feat(get): add watch mode for process-definition lookup

## Clarifications

### Session 2026-08-04

- Q: What default refresh interval should watch mode use when `--watch-interval` is omitted? -> A: 1s
- Q: What should `get process-definition --watch` do when no selector is provided? -> A: Watch all process definitions
- Q: What retry budget should watch mode use when no retry flag is provided? -> A: Existing command retry default
- Q: Should `--watch` support machine-readable output modes? -> A: No, human output only

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Watch Process Definitions Until Visible (Priority: P1)

As a Camunda operator, I want `get process-definition` lookups to repeat automatically so I can watch for process-definition deployment visibility without manually re-running the command.

**Why this priority**: The core value is continuous observation. Operators need a reliable way to wait for process definitions to appear, update, or change while preserving the existing one-shot lookup behavior for normal use.

**Independent Test**: Run `c8volt get process-definition --watch` against a controlled process-definition population, add or change a matching definition during the run, and verify that repeated snapshots show the current matching state until the command is interrupted or times out.

**Acceptance Scenarios**:

1. **Given** a matching process definition exists, **When** the operator runs `c8volt get process-definition --watch`, **Then** c8volt immediately prints a snapshot and continues printing later snapshots at the configured cadence.
2. **Given** no matching process definition is visible yet, **When** the operator runs `c8volt get process-definition --bpmn-process-id <id> --latest --watch`, **Then** c8volt reports successful empty snapshots for the selected BPMN process ID until a matching definition appears or the run ends.
3. **Given** the operator does not provide a selector, **When** the operator runs `c8volt get process-definition --watch`, **Then** c8volt watches all process definitions and each snapshot contains the current complete paged result set.
4. **Given** watch mode is running, **When** the operator interrupts it or an existing timeout limit is reached, **Then** c8volt exits cleanly without claiming a failed lookup solely because the watch ended.

---

### User Story 2 - Control Watch Cadence And Retry Tolerance (Priority: P2)

As a Camunda operator, I want to choose how often process-definition snapshots are refreshed and how many transient failures are tolerated so I can balance freshness, load, and reliability.

**Why this priority**: Watch mode must be useful in real operational environments where short polling can be helpful, but excessive retries or tight intervals can create noisy output or unnecessary load.

**Independent Test**: Run watch mode with different positive interval values and retry settings, inject transient lookup failures, and verify that snapshot cadence and consecutive failure handling match the selected controls.

**Acceptance Scenarios**:

1. **Given** the operator provides `--watch-interval 2s`, **When** watch mode runs, **Then** snapshots are refreshed approximately every 2 seconds after the immediate first snapshot.
2. **Given** the operator omits `--watch-interval`, **When** watch mode runs, **Then** snapshots are refreshed approximately every 1 second after the immediate first snapshot.
3. **Given** the operator provides an invalid, zero, or negative watch interval, **When** the command starts, **Then** c8volt rejects the command with a clear validation error before performing the lookup.
4. **Given** transient lookup failures occur and no retry flag is provided, **When** failures remain within the existing command retry default, **Then** c8volt reports the failure appropriately and continues watching; after a successful snapshot, the consecutive failure budget resets.
5. **Given** consecutive transient failures exceed the configured retry budget, **When** watch mode can no longer continue safely, **Then** c8volt stops with a clear error and a non-success exit status.

---

### User Story 3 - Preserve Script-Safe Output By Rejecting Machine Modes (Priority: P3)

As an automation author, I want watch mode to reject machine-oriented output combinations so JSON, keys-only, quiet, and automation output contracts remain simple and script-safe.

**Why this priority**: c8volt is used both interactively and in automation. Forbidding machine-oriented watch combinations keeps `--json` as one stable document per command invocation and avoids ambiguous live-stream contracts.

**Independent Test**: Run watch scenarios with default human and verbose output, then run `--watch` with JSON, keys-only, quiet, automation, and XML combinations and verify incompatible combinations fail before lookup work.

**Acceptance Scenarios**:

1. **Given** JSON output is requested with watch mode, **When** the command starts, **Then** c8volt rejects the combination before lookup work and explains that watch is human-output only.
2. **Given** keys-only output is requested with watch mode, **When** the command starts, **Then** c8volt rejects the combination before lookup work and preserves the keys-only contract for non-watch invocations.
3. **Given** quiet or automation mode is requested with watch mode, **When** the command starts, **Then** c8volt rejects the combination before lookup work because watch requires human snapshot output.
4. **Given** verbose output is selected without machine-oriented modes, **When** watch mode runs, **Then** additional durable context may appear without changing the human snapshot result rows.

### Edge Cases

- Watch mode is combined with XML output; c8volt must reject the combination because XML output represents a single artifact rather than a human watch display.
- Watch mode is combined with JSON, keys-only, quiet, or automation output; c8volt must reject the combination because watch mode is a human-output feature.
- Watch mode is combined with a process-definition key lookup; c8volt may allow it, including with stat-style output, because key-level observations are useful when a specific definition is known.
- Watch mode is run without a selector; c8volt must watch all process definitions and each snapshot must include the complete paged result set visible for that broad lookup.
- The selected process-definition population spans multiple result pages; each watch snapshot must represent a complete current snapshot rather than only the first page.
- The matching process-definition population changes between snapshots; c8volt must report each snapshot independently and avoid implying that previous results remain current.
- A snapshot has zero results; output must remain valid for the selected mode and make the empty result understandable in human mode.
- The operator interrupts watch mode while a lookup is in progress; c8volt must stop promptly and avoid partial human snapshot output.
- A timeout or retry limit ends watch mode after one or more successful snapshots; c8volt must distinguish normal completion from lookup failure.
- Existing non-watch invocations must continue to behave exactly as before, including validation, output, exit status, and selector requirements.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `get process-definition` and its established aliases MUST support a watch mode that repeatedly emits process-definition lookup snapshots until interrupted, timed out, or stopped by an unrecoverable error.
- **FR-002**: Watch mode MUST emit the first snapshot immediately before waiting for the refresh interval.
- **FR-003**: Watch mode MUST use a 1-second refresh interval when the operator does not provide `--watch-interval`.
- **FR-004**: Watch mode MUST support an operator-configurable refresh interval.
- **FR-005**: The refresh interval MUST accept positive duration values and MUST reject invalid, zero, or negative values with a clear validation error.
- **FR-006**: Watch mode MUST reuse the command's existing lookup selectors and output mode selection wherever the combination remains meaningful.
- **FR-007**: Non-watch process-definition behavior MUST remain unchanged for selectors, validation, output, exit status, and errors.
- **FR-008**: Watch mode MUST reject XML output because XML output is a single artifact, not a human watch display.
- **FR-009**: Watch mode MUST allow process-definition key lookup when the existing selector is otherwise valid, including compact stat-style observations.
- **FR-010**: In watch mode only, a missing selector MUST watch all process definitions rather than using the fatal selector diagnostic that applies to one-shot mode.
- **FR-011**: Each watch snapshot MUST include all matching process definitions available through the selected lookup at the time of the snapshot, including results that require paging.
- **FR-012**: When no retry flag is provided, watch mode MUST use the existing command retry default.
- **FR-013**: Transient lookup failures during watch mode MUST be counted consecutively against the configured retry budget.
- **FR-014**: The consecutive retry budget MUST reset after any successful snapshot.
- **FR-015**: When consecutive transient failures exceed the configured retry budget, watch mode MUST stop with a clear error and non-success exit status.
- **FR-016**: Existing timeout controls that are meaningful for repeated read-only observation MUST limit the overall watch run.
- **FR-017**: Watch mode MUST reject JSON, keys-only, quiet, and automation-oriented output combinations before lookup work.
- **FR-018**: Human watch output MUST be compact, snapshot-oriented, and understandable without exposing low-level request details by default.
- **FR-019**: Progress, retry, timeout, and status messages MUST stay within the human watch experience and MUST NOT alter non-watch machine output contracts.
- **FR-020**: Verbose mode MAY provide additional durable watch context, but default human output MUST remain compact and scan-friendly.
- **FR-021**: Non-watch JSON, keys-only, quiet, and automation-oriented modes MUST preserve existing deterministic output guarantees.
- **FR-022**: Watch-mode incompatibility errors MUST be clear, local validation errors emitted before any process-definition lookup.
- **FR-023**: Watch mode MUST stop promptly when the operator interrupts the command.
- **FR-024**: Watch mode MUST be structured so other read-only commands can adopt the same operator-facing watch semantics in future work.
- **FR-025**: Process-definition snapshot behavior MUST be available through the public process-definition command surface rather than requiring operators to use lower-level implementation concepts.
- **FR-026**: User-facing help, command metadata, README-facing documentation, and generated CLI documentation MUST describe watch mode as human-output only, the default 1-second interval, the interval control, incompatible output combinations, retry/timeout behavior, and default retry behavior.
- **FR-027**: Automated tests MUST cover default interval behavior, default retry behavior, watch cadence validation, immediate first snapshot, interrupt or timeout completion, retry budget reset, retry budget exhaustion, paged snapshots, and unchanged non-watch behavior.
- **FR-028**: Automated tests MUST cover human, verbose, JSON-rejection, keys-only-rejection, quiet-rejection, automation-rejection, XML-rejection, key lookup, stat-style, and broad missing-selector watch scenarios.

### Key Entities *(include if feature involves data)*

- **Watch Session**: A bounded or user-interrupted observation run that repeatedly evaluates the selected process-definition lookup.
- **Watch Snapshot**: One complete result emitted by a watch session for the current selected process-definition lookup.
- **Snapshot Selector**: The set of command selectors that define which process definitions belong in each snapshot, such as BPMN process ID, latest selection, key selection, or broad search.
- **Watch Interval**: The operator-selected time between snapshot attempts after the immediate first snapshot.
- **Retry Budget**: The allowed number of consecutive transient snapshot failures before watch mode stops.
- **Output Mode Contract**: The selected result format and channel rules for human watch output and rejected machine-oriented watch combinations.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In controlled watch-mode tests, 100% of runs emit the first snapshot before waiting for the configured interval.
- **SC-002**: When `--watch-interval` is omitted, watch mode refreshes snapshots approximately every 1 second in 100% of default-cadence tests.
- **SC-003**: Invalid, zero, and negative watch intervals are rejected before lookup work begins in 100% of validation tests.
- **SC-004**: Existing non-watch process-definition tests continue to pass with no changed output, selector validation, or exit behavior.
- **SC-005**: Paged process-definition populations are fully represented in every watch snapshot in 100% of paged snapshot tests.
- **SC-006**: Consecutive transient failures stop watch mode only after the default or configured retry budget is exceeded, and a successful snapshot resets that budget in 100% of retry tests.
- **SC-007**: JSON, keys-only, quiet, automation, and XML output combinations are rejected with watch mode before lookup work in 100% of incompatible-output tests.
- **SC-008**: Non-watch JSON, keys-only, quiet, and automation tests continue to pass with existing deterministic output guarantees.
- **SC-009**: Human and verbose watch-mode tests produce compact snapshot-oriented output with no low-level endpoint, request, cursor, or per-page lifecycle detail in default human output.
- **SC-010**: Help text, README-facing documentation, and generated CLI docs accurately describe watch as human-output only, all watch flags, retry/timeout behavior, and incompatible combinations before the feature is considered ready.

## Assumptions

- The primary users are Camunda operators who already use `c8volt get process-definition`, `get pd`, or `get pds`; automation authors continue to use non-watch machine-readable invocations.
- Watch mode is read-only observation; it does not deploy, mutate, or delete process definitions.
- Existing retry and timeout controls remain the preferred operator-facing controls when their meanings are clear for watch mode.
- Empty snapshots are valid watch results for explicit selectors because operators may be waiting for a deployment or visibility change.
- "Snapshot" means the best current complete result for the selected lookup at one point in time; c8volt does not guarantee that the Camunda population remains stable between snapshots.
- Future reuse by other read-only commands is a design goal, but this feature proves the behavior only for process-definition lookup.

## Out of Scope

- Watch mode for mutating commands.
- Watch mode for every read-only command in the first feature slice.
- Live terminal dashboards, interactive filtering, or cursor-navigation interfaces.
- Streaming XML snapshots.
- Streaming JSON or keys-only watch snapshots.
- Changing process-definition search semantics unrelated to repeated observation.
- Exposing low-level request, cursor, endpoint, or paging diagnostics in default human output.
