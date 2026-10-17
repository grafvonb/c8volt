# Feature Specification: Suspended Process-Instance State

**Feature Branch**: `339-suspended-instance-state`

**Created**: 2026-10-17

**Status**: Draft

**Input**: [Issue #339 — Support suspended process-instance state](https://github.com/grafvonb/c8volt/issues/339)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Recognize suspended instances accurately (Priority: P1)

As an operator or consumer of the public process interface, I need suspended instances to retain their distinct state in instance results and state reports so that paused execution is not mistaken for an unknown or finished process.

**Why this priority**: Reliable state representation is the foundation for interpreting every existing read and expectation operation.

**Independent Test**: Read an instance whose observed state is suspended through existing instance and state-lookup operations; confirm that every result identifies it as suspended and nonterminal.

**Acceptance Scenarios**:

1. **Given** an instance in `SUSPENDED`, **When** an instance result or state report is returned through the public interface or CLI, **Then** its state remains `SUSPENDED` rather than `UNKNOWN`.
2. **Given** `SUSPENDED`, `suspended`, or a mixed-case spelling, **When** existing state parsing and normalization are applied, **Then** each identifies the same suspended state.
3. **Given** a suspended instance, **When** its state is classified, **Then** it is nonterminal and does not imply execution completion.
4. **Given** existing canceled/terminated aliases, absent or unknown states, or an unrecognized backend state, **When** those values are interpreted, **Then** their established compatibility and fallback behavior remains unchanged.

### User Story 2 - Select and wait for suspended instances (Priority: P1)

As an operator, I need existing state selectors and expectations to support suspension so I can find paused work and verify that suspension has been observed without changing the meaning of active or terminal states.

**Why this priority**: Incorrect selection or expectation results can mislead operational decisions and automation.

**Independent Test**: Use suspended, active-only, and all-state selections over a known mixture of states, then evaluate suspended and terminal expectations against a suspended instance.

**Acceptance Scenarios**:

1. **Given** a configured Camunda 8.10 environment containing active and suspended instances, **When** suspended-state filtering is requested, **Then** only suspended instances matching the other selectors are returned.
2. **Given** the same environment, **When** active-only selection is requested, **Then** suspended instances are excluded; **When** existing all-state selection is used, **Then** suspended instances are eligible alongside other states.
3. **Given** an expectation of `SUSPENDED`, **When** suspension is observed, **Then** the expectation succeeds and its report preserves `SUSPENDED`.
4. **Given** a terminal-state expectation, **When** suspension is observed, **Then** it remains unsatisfied and follows the existing wait and timeout behavior.
5. **Given** configured Camunda 8.8 or 8.9, **When** a suspended filter or expectation is explicitly requested through either the CLI or public interface, **Then** the established unsupported-capability error is returned before an unsupported request is sent, without broadening or remapping the selection.
6. **Given** an existing state-selection command, **When** an operator consults its help, validation feedback, or shell completion, **Then** suspended-state selection and its version limitation are represented consistently.

### User Story 3 - Confirm creation when execution is suspended (Priority: P2)

As an operator creating an instance, I need an observed suspended instance to confirm that creation succeeded even though execution has not completed.

**Why this priority**: Creation must not time out merely because a newly created instance is already suspended.

**Independent Test**: Make the first successful creation observation a suspended instance and verify creation visibility succeeds without claiming execution completion.

**Acceptance Scenarios**:

1. **Given** a newly created instance whose first observed state is suspended, **When** creation visibility is checked, **Then** creation is confirmed without requiring a later active or terminal state.
2. **Given** an instance that is not yet observable, a read failure, or an existing terminal-state observation, **When** creation visibility is checked, **Then** existing timeout, error, and terminal-observation behavior is preserved.

### User Story 4 - Read current suspension metadata (Priority: P2)

As an operator or automation author, I need the reported suspension timestamp to reflect the current response so that refreshed output does not show obsolete suspension information.

**Why this priority**: Accurate metadata supports diagnosis without adding noise or implying a suspension history that is not available.

**Independent Test**: Read an instance with a suspension timestamp, then refresh it with a response lacking that timestamp, including in existing watch output where relevant.

**Acceptance Scenarios**:

1. **Given** a response containing `suspendedDate`, **When** the instance is exposed through the public interface or relevant structured output, **Then** the timestamp is preserved using established timestamp formatting.
2. **Given** a response without a suspension timestamp, **When** the instance is exposed, **Then** existing absent-value conventions apply.
3. **Given** a previously displayed timestamp, **When** a subsequent response lacks it, **Then** refreshed or watch output clears or omits the value according to those conventions and does not retain stale metadata.
4. **Given** an instance with suspension metadata, **When** default human output is rendered, **Then** it remains compact and does not add suspension timestamp detail solely for this feature.

### Edge Cases

- A mixed set of desired states that explicitly includes suspension on 8.8 or 8.9 must produce the unsupported-capability error; other desired states must not hide the unsupported selection.
- Unfiltered reads and older-version defaults retain their existing behavior; a general read must not be rejected merely because suspension support exists in the shared state vocabulary.
- Empty suspended searches retain the established output-mode contract, including one successful envelope where supported and zero-byte keys-only output.
- A suspended instance with no timestamp is still suspended; state classification must not depend on timestamp presence.
- A timestamp disappearing between refreshes does not establish a cumulative duration or a historical suspension interval.
- Suspension observed during a terminal wait does not end that wait successfully; existing cancellation, error, and timeout behavior remains in effect.
- Existing output-mode precedence, quiet behavior, aliases, and unknown-state handling must remain unchanged.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST recognize `SUSPENDED` as a distinct, nonterminal process-instance state across existing internal and public state representations, case-insensitive parsing, and normalization.
- **FR-002**: Instance results, state lookup reports, and expectation reports MUST preserve `SUSPENDED` without converting it to another state.
- **FR-003**: The system MUST preserve existing `CANCELED`/`TERMINATED` compatibility, `ABSENT`/`UNKNOWN` semantics, and handling of unrecognized backend states.
- **FR-004**: Existing process-instance operations that accept state filters or expectations MUST accept suspension on Camunda 8.10. Suspended searches MUST select the corresponding suspended state without broadening other criteria.
- **FR-005**: Active-only filtering MUST exclude suspended instances, while existing all-state selection MUST include them when other selection criteria match.
- **FR-006**: Observed suspension MUST satisfy a suspended-state expectation and MUST NOT satisfy a terminal-state expectation. Existing waiting, timeout, cancellation, and error behavior MUST be preserved.
- **FR-007**: Explicit suspended-state filters or expectations on configured Camunda 8.8 or 8.9 MUST fail through the established unsupported-capability error in both CLI and public entry points before an unsupported request is sent. The system MUST NOT silently broaden selection or map suspension to another state.
- **FR-008**: Unfiltered reads and older-version defaults MUST retain their established behavior.
- **FR-009**: Creation visibility MUST treat an observed suspended instance as evidence of successful creation, separately from execution completion, while preserving existing timeout, failure, and terminal-observation handling.
- **FR-010**: Instance representations and relevant structured output MUST preserve nullable `suspendedDate` using established timestamp and absent-value conventions.
- **FR-011**: A response without a suspension timestamp MUST clear or omit any earlier timestamp in refreshed and watch results. The system MUST NOT infer cumulative suspension duration or historical periods from this field.
- **FR-012**: Default human output MUST remain compact. Existing human, structured, keys-only, quiet, and empty-result contracts MUST remain compatible except for the intended recognition of suspension and relevant structured metadata.
- **FR-013**: Affected validation messages, command help and metadata, shell completions, and user documentation MUST describe the supported state consistently and identify the older-version restriction.

### Key Entities

- **Process-instance state**: The observed lifecycle classification, including suspension as a distinct nonterminal state and the existing aliases and fallback meanings.
- **Process instance**: An identified instance with its current state and optional suspension timestamp.
- **State selection or expectation**: Requested state criteria used to find instances or verify an observed state, subject to configured-version capabilities.
- **State report**: The observed state and existing expectation outcome exposed to an operator or public-interface consumer.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every suspended instance in the acceptance scenarios retains its state across instance results, state lookup, and expectation reports; none is mislabeled unknown or terminal.
- **SC-002**: Across a known mixture of active, suspended, and terminal instances, suspended-only and active-only selections return exactly their matching populations, and all-state selection omits no instance because it is suspended.
- **SC-003**: Every explicit suspended selector or expectation tested on each older supported version fails as unsupported before an unsupported request is sent, with zero silent fallbacks.
- **SC-004**: Creation succeeds in every controlled scenario where the first valid observation is suspended, without waiting for a state transition or claiming execution completion.
- **SC-005**: Every present-to-absent timestamp refresh scenario removes the obsolete value; zero stale timestamps remain in subsequent results.
- **SC-006**: Operators can identify suspension and the supported-version boundary from existing help and result output without consulting implementation details; all existing alias, unknown-state, and output-contract regression scenarios retain their expected outcomes.

## Assumptions

- Feature semantics apply to the Camunda 8.10 release line; Camunda 8.10.4 is the required integration-validation server target.
- This feature has no prerequisite within the issue series. It provides representation and read/expectation behavior for #340 and #341; combined integration validation belongs to #342.
- Existing formatting, absence conventions, capability errors, authentication, and selection behavior remain authoritative unless a requirement above explicitly extends them.
- The current generated client already supports suspension. Its exact baseline and delivery constraints are recorded in [planning-context.md](planning-context.md) for the next phase.
- New suspend/resume commands or public mutation workflows, process-definition lifecycle states or statistics, mutation eligibility, cancellation/deletion orchestration, purge discovery/cleanup, new suspension-date filters or sorting flags, unrelated coverage, client regeneration, and server upgrades are out of scope.
