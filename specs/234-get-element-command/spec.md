# Feature Specification: Runtime Element Instance Command

**Feature Branch**: `240-get-element-command`

**Created**: 2026-07-16

**Status**: Draft

**Input**: User description: "https://github.com/grafvonb/c8volt/issues/241"

## Clarifications

### Session 2026-07-16

- Q: How should `--key` and search filters interact when selectors are combined? → A: Search filters combine with AND semantics; `--key` is mutually exclusive with search filters.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Fetch One Runtime Element Instance (Priority: P1)

As a c8volt operator, I want to fetch a runtime element instance by its element instance key so I can inspect the exact BPMN element execution record that appeared during a process run.

**Why this priority**: Direct lookup is the smallest useful slice and lets operators verify a known element instance without scanning unrelated runtime data.

**Independent Test**: Can be fully tested by requesting one known element instance key and verifying that the command returns one matching element instance with its inspection fields.

**Acceptance Scenarios**:

1. **Given** a Camunda 8.8 or 8.9 environment with an existing element instance, **When** the operator runs `c8volt get element --key <element-instance-key>`, **Then** the command returns exactly that runtime element instance.
2. **Given** a Camunda 8.7 environment, **When** the operator runs `c8volt get element --key <element-instance-key>`, **Then** the command returns a clear unsupported-version error and does not claim success.

---

### User Story 2 - Search Runtime Elements For Operational Inspection (Priority: P2)

As a c8volt operator, I want to search runtime element instances by process, BPMN element, state, type, process definition, and BPMN process identifier so I can understand which elements were entered during execution.

**Why this priority**: Search is the core operational workflow for diagnosing process execution, including loops and multi-instance behavior where a single BPMN element may have multiple runtime entries.

**Independent Test**: Can be fully tested by searching for a process instance that has multiple element instances and verifying that all matching runtime rows are listed using standard paging controls.

**Acceptance Scenarios**:

1. **Given** a process instance with runtime element instances, **When** the operator runs `c8volt get element --pi-key <process-instance-key>`, **Then** the command lists element instances for that process instance.
2. **Given** a BPMN element that ran more than once in a process instance, **When** the operator searches by that BPMN element identifier, **Then** the command shows separate rows with the same BPMN element identifier and different element instance keys.
3. **Given** available element instances, **When** the operator searches with `--element-id`, `--state`, `--type`, `--pd-key`, or `--bpmn-process-id`, **Then** the command returns only element instances matching the selected criteria.
4. **Given** multiple search filters in one request, **When** the operator runs `c8volt get element` with those filters, **Then** the command returns only element instances matching all supplied filters.
5. **Given** an unfiltered search, **When** the operator runs `c8volt get element` with standard paging controls, **Then** the command follows the existing `get` command paging conventions without requiring a custom all-results flag.

---

### User Story 3 - Consume Element Results In Standard Output Modes (Priority: P3)

As a c8volt operator or automation author, I want runtime element results in the same human, JSON, keys-only, and total output modes used by other `get` commands so I can inspect results interactively or script against stable output.

**Why this priority**: Output consistency keeps the new command predictable for existing operators and enables automation without custom parsing rules.

**Independent Test**: Can be fully tested by running equivalent element searches with human output, `--json`, `--keys-only`, and `--total`, then verifying each mode follows the expected contract.

**Acceptance Scenarios**:

1. **Given** a search with two matching element instances, **When** the operator uses default human output, **Then** the command prints compact aligned rows with the primary key first and a final `found: 2` line.
2. **Given** matching element instances, **When** the operator uses `--keys-only`, **Then** the command prints only element instance keys, one per line.
3. **Given** matching element instances, **When** the operator uses `--total`, **Then** the command prints only the numeric total.
4. **Given** matching element instances, **When** the operator uses `--json`, **Then** the command returns a stable machine-readable payload containing `total` and `items`.

### Edge Cases

- Camunda 8.7 does not support runtime element instance inspection and must return a clear unsupported-version error.
- Direct lookup by `--key` must not be combined with search filters.
- Searches with no matches must return empty output appropriate to the selected mode and must not be reported as failures.
- Active element instances without an end date must omit the end-time marker in compact human output.
- Element instances with incidents must visibly show exactly one incident marker in compact human output: `inc!` when no incident key is available, or `inc!:<incidentKey>` when an incident key is available.
- Paging controls must respect `--batch-size` and `--limit` so bounded searches do not return more rows than requested.
- JSON output must aggregate bounded search results into one payload, while interactive human and keys-only output may stream page by page according to existing `get` behavior.
- Unfiltered search must use existing paging conventions and must not introduce a custom `--all` flag.
- Listener and job details must not appear as element instance rows.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide a read-only `c8volt get element` command for inspecting Camunda runtime element instances.
- **FR-002**: The command MUST represent runtime element instances entered during execution, not static BPMN diagram elements.
- **FR-003**: The command MUST support Camunda 8.8 and Camunda 8.9 runtime element instance inspection.
- **FR-004**: The command MUST return a clear unsupported-version error for Camunda 8.7.
- **FR-005**: Operators MUST be able to fetch one element instance with `--key <element-instance-key>`.
- **FR-006**: Operators MUST be able to search element instances by process instance key using `--pi-key <process-instance-key>`.
- **FR-007**: Operators MUST be able to search element instances by BPMN element identifier using `--element-id <bpmn-element-id>`.
- **FR-008**: Operators MUST be able to search element instances by runtime state using `--state <state>`.
- **FR-009**: Operators MUST be able to search element instances by element type using `--type <type>`.
- **FR-010**: Operators MUST be able to search element instances by process definition key using `--pd-key <process-definition-key>`.
- **FR-011**: Operators MUST be able to search element instances by BPMN process identifier using `--bpmn-process-id <bpmn-process-id>`.
- **FR-012**: When multiple search filters are supplied, the command MUST return only element instances that match all supplied filters.
- **FR-013**: Direct lookup with `--key` MUST be mutually exclusive with search filters.
- **FR-014**: The command MUST support standard `get` command controls for `--batch-size`, `--limit`, `--total`, `--json`, and `--keys-only`.
- **FR-015**: The command MUST NOT add a custom `--all` flag; unfiltered search MUST follow existing `get` command paging behavior.
- **FR-016**: Each returned element instance MUST preserve the fields needed for operator inspection: `elementInstanceKey`, `elementId`, `elementName`, `type`, `state`, `startDate`, `endDate`, `processInstanceKey`, `rootProcessInstanceKey`, `processDefinitionId`, `processDefinitionKey`, `tenantId`, `hasIncident`, and `incidentKey`.
- **FR-017**: Human output MUST use the existing compact `get` command grammar: aligned rows, primary key first, stable short tags, and a final `found: N` line for list output.
- **FR-018**: Human output MUST avoid normal-mode request, cursor, backend target, and per-page lifecycle details.
- **FR-019**: Human output MUST use c8volt's established timestamp presentation for start and end dates.
- **FR-020**: Human output MUST show exactly one incident marker for an element instance with an incident: `inc!` when no incident key is available, or `inc!:<incidentKey>` when an incident key is available.
- **FR-021**: Human output MUST NOT render both `inc!` and `inc!:<incidentKey>` for the same element instance row.
- **FR-022**: `--keys-only` output MUST print only element instance keys, one per line.
- **FR-023**: `--total` output MUST print only the numeric total.
- **FR-024**: `--json` output MUST return a stable machine-readable payload with `total` and `items`.
- **FR-025**: `--limit` MUST cap returned element instances across pages.
- **FR-026**: The command MUST preserve repeated runtime executions as separate element instance rows, including looped or multi-instance executions with the same BPMN element identifier.
- **FR-027**: The feature MUST NOT implement `c8volt get pi --with-elements`.
- **FR-028**: The feature MUST NOT add element summary output, metrics output, loop-count aggregation, execution listener rows, task listener rows, listener enrichment, or job enrichment.

### Key Entities

- **Runtime Element Instance**: A single execution occurrence of a BPMN element within a process run. Key attributes include element instance key, BPMN element identifier, element name, type, state, timestamps, process instance identifiers, process definition identifiers, tenant, and incident indicators.
- **Process Instance**: A running or completed process execution that can contain multiple runtime element instances.
- **Process Definition**: The deployed process definition associated with returned element instances, identified by key and BPMN process identifier.
- **Incident Marker**: The user-visible indication that an element instance is associated with an incident, optionally including a specific incident key.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can fetch a known element instance by key in one command and see the matching runtime element details with no additional lookup steps.
- **SC-002**: Operators can list all element instances for a process instance with standard paging controls and verify looped BPMN elements as distinct runtime rows.
- **SC-003**: 100% of supported output modes for this command match the documented mode contract: human list, JSON, keys-only, and total.
- **SC-004**: 100% of unsupported Camunda 8.7 attempts return a clear unsupported-version result without mutating operational state.
- **SC-005**: For element instances with incidents, human output makes incident presence visible in the compact row in every applicable list result.
- **SC-006**: The feature is considered complete only when tests cover supported Camunda 8.8 and 8.9 behavior, unsupported Camunda 8.7 behavior, command validation, human output, JSON output, keys-only output, totals, paging, and incident markers.

## Assumptions

- Operators already have a configured c8volt environment and permissions sufficient to inspect process runtime data.
- The command should follow the established `get incident` and `get pi` user experience for paging, output modes, validation, and concise human rows.
- The standalone element command is the only user-facing feature in this story; later process-instance enrichment can reuse the capability but is out of scope here.
- Exact total values may be used when available; otherwise the command may count page by page according to existing `get` command behavior.
- User-facing documentation and command reference updates are required because this introduces a new command.
