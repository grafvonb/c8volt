# Feature Specification: Runtime Listener Jobs Under Elements

**Feature Branch**: `249-runtime-listener-jobs`

**Created**: 2026-07-23

**Status**: Draft

**Input**: User description: "GitHub issue #249 - feat(ops): show runtime listener jobs under elements"

## Clarifications

### Session 2026-07-23

- Q: How should listener jobs that cannot be matched to an element instance key be represented? → A: Omit unmatched listener jobs from enriched output and document that only element-matched jobs are shown.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Inspect Listener Jobs For A Specific Element (Priority: P1)

An operator investigating a process instance can request listener details for an element-oriented view and see runtime listener jobs directly under the element that owns them.

**Why this priority**: This is the core operator problem: listener-related retries, failures, and incidents are hard to diagnose when listener jobs are separated from the element that created them.

**Independent Test**: Can be fully tested by viewing a known element instance with runtime listener jobs using listener enrichment and confirming the listener rows appear under the matching element.

**Acceptance Scenarios**:

1. **Given** an element instance with one or more runtime listener jobs, **When** the operator requests that element with listener enrichment, **Then** the output shows those listener jobs nested under the matching element.
2. **Given** a process instance with multiple element instances and listener jobs on only one element, **When** the operator requests element views for that process instance with listener enrichment, **Then** listener jobs appear only under their owning element.
3. **Given** an element with no runtime listener jobs, **When** the operator requests listener enrichment, **Then** the output clearly shows no listener jobs without implying an error.

---

### User Story 2 - Correlate Listener Jobs In Process Instance Element Views (Priority: P1)

An operator viewing a process instance with elements can opt in to listener details and correlate runtime listener jobs with the element timeline without changing default output.

**Why this priority**: Process-instance views are a primary investigation path, and listener context must be visible there without surprising users or scripts that depend on existing output.

**Independent Test**: Can be fully tested by comparing process-instance output with and without listener enrichment and verifying that only the enriched output includes listener details under elements.

**Acceptance Scenarios**:

1. **Given** a process instance with element details and runtime listener jobs, **When** the operator requests elements and listener enrichment, **Then** each listener job appears under the matching element row.
2. **Given** the same process instance, **When** the operator does not request listener enrichment, **Then** existing human and structured output remain unchanged.
3. **Given** listener enrichment is requested without an element context, **When** the operator runs the command, **Then** the command fails with a clear validation message explaining the missing element context.

---

### User Story 3 - Preserve Walk Tree Readability With Listener Details (Priority: P2)

An operator walking a process-instance tree can include listener details while preserving the existing process tree shape across default, parent, children, and flat walk modes.

**Why this priority**: Walk views are used for relationship-oriented investigations; listener details must enrich element blocks without making child process instances look like listener children.

**Independent Test**: Can be fully tested by running each supported walk mode with elements and listener enrichment and confirming listeners stay inside the owning process instance's element block.

**Acceptance Scenarios**:

1. **Given** a process tree with listener jobs on elements, **When** the operator requests a default walk with elements and listeners, **Then** listener rows remain visually inside the owning process instance's elements block.
2. **Given** the operator requests parent, children, or flat walk modes with elements and listeners, **When** the walk output is rendered, **Then** the existing process tree order and grouping are preserved.
3. **Given** the operator requests listener enrichment with keys-only walk output, **When** the command is validated, **Then** it fails with a clear validation message because listener rows require element context.

---

### User Story 4 - Include Listener Context In Slow Process Analysis (Priority: P3)

An operator analyzing a slow process instance can opt in to listener details so listener-related work is visible near the elements being analyzed.

**Why this priority**: Slow-process investigations benefit from listener context, but this is a secondary path after direct element and process-instance inspection.

**Independent Test**: Can be fully tested by running slow-process analysis for a process instance with listener jobs and confirming listener details appear under matching elements only when requested.

**Acceptance Scenarios**:

1. **Given** a slow-process analysis target with listener jobs on element execution, **When** the operator requests listener enrichment, **Then** listener jobs appear under the relevant elements in the analysis output.
2. **Given** listener enrichment is not requested, **When** slow-process analysis is run, **Then** the existing analysis output remains unchanged.

### Edge Cases

- Listener enrichment is requested in a command mode that has no element context.
- Listener enrichment is requested with keys-only output where nested listener rows cannot be represented.
- A supported command returns elements but no matching runtime listener jobs.
- A runtime listener job cannot be matched to an element instance key and is omitted from listener-enriched output.
- The connected environment does not support runtime listener job lookup.
- Multiple listener jobs are attached to the same element.
- Listener jobs include both execution listener and user task listener kinds.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The product MUST provide `--with-listeners` as the opt-in listener enrichment control for element-based process investigation views.
- **FR-002**: The product MUST support listener enrichment for `get element -k <element-instance-key>`.
- **FR-003**: The product MUST support listener enrichment for `get element --pi-key <process-instance-key>`.
- **FR-004**: The product MUST support listener enrichment for `get pi -k <process-instance-key> --with-elements`.
- **FR-005**: The product MUST support listener enrichment for `walk pi -k <process-instance-key> --with-elements` in default, parent, children, and flat modes.
- **FR-006**: The product MUST support listener enrichment for `ops analyse slow-process-instances -k <process-instance-key>`.
- **FR-007**: The product MUST require an element context before listener enrichment can be requested.
- **FR-008**: The product MUST reject listener enrichment with keys-only output where listener details cannot be represented.
- **FR-009**: Runtime listener jobs MUST be associated with the element instance that owns them when a matching element instance key is available.
- **FR-010**: Listener details MUST distinguish execution listener jobs from user task listener jobs.
- **FR-011**: Listener details MUST include operator-relevant fields needed to identify the job, listener kind, listener event, worker or handler type, current state, retry count, and available job failure fields such as error code and error message.
- **FR-012**: Human-readable output MUST nest listener details below the element row that owns them.
- **FR-013**: Walk output MUST keep listener rows inside the owning process instance's element block and preserve the existing process-instance tree structure.
- **FR-014**: Structured output MUST include listener arrays under element objects only when listener enrichment is requested.
- **FR-015**: Structured output MUST continue to distinguish "listeners not requested" from "listeners requested but none found".
- **FR-016**: Existing output without listener enrichment MUST remain unchanged for supported commands and modes.
- **FR-017**: If runtime listener lookup is unsupported in the connected environment, the product MUST fail using the established unsupported-version error style for nearby operator commands.
- **FR-018**: Validation failures MUST be clear enough for an operator to understand which additional context or mode change is required.
- **FR-019**: Runtime listener jobs without a matching element instance key MUST be omitted from listener-enriched output.

### Key Entities *(include if feature involves data)*

- **Process Instance**: A running or historical workflow execution that operators inspect, analyze, or walk.
- **Element Instance**: A BPMN element execution within a process instance; listener jobs are nested under this entity when ownership can be determined.
- **Runtime Listener Job**: A runtime job created for an execution listener or user task listener, including identity, kind, event, type, state, retries, and failure-related status.
- **Listener-Enriched Element View**: An operator-facing view where requested listener jobs are attached to their matching element instances.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In 100% of supported element-oriented views, an operator can request listener enrichment and see matching runtime listener jobs under their owning elements.
- **SC-002**: In 100% of invalid listener-enrichment combinations, the command fails before producing misleading output and explains the missing or incompatible element context.
- **SC-003**: Existing human and structured outputs without listener enrichment remain byte-for-byte stable for the affected command scenarios covered by validation.
- **SC-004**: For a process instance with listener jobs, an operator can identify the owning element, listener kind, state, and retry count from a single enriched output in under 30 seconds.
- **SC-005**: Structured-output consumers can reliably distinguish listener details not requested from listener details requested with zero matching jobs in all affected element views.

## Assumptions

- The target users are operators and automation authors investigating Camunda process execution behavior.
- Listener enrichment remains opt-in because it may require extra runtime lookups.
- Only runtime listener jobs attached to element execution are in scope.
- Static or global BPMN listener configuration is out of scope.
- Filtering, sorting, or ranking listeners by duration is out of scope.
- When a runtime listener job cannot be matched to an element instance, it is omitted because listener-enriched output is limited to element-owned listener jobs.
- Human-readable output may omit an empty listeners block when no listener jobs exist, while structured output preserves the established requested-but-empty convention.
