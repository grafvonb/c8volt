# Feature Specification: Process Instance Element Enrichment

**Feature Branch**: `242-get-pi-elements`

**Created**: 2026-07-16

**Status**: Draft

**GitHub Issue**: [#242](https://github.com/grafvonb/c8volt/issues/242) - feat(get pi): add `c8volt get pi --with-elements` to show runtime element instances under process instances

**Input**: GitHub issue #242 requests a process-instance enrichment flag that attaches runtime element instances to selected process-instance results, reusing the standalone runtime element inspection capability from issue #241.

## Clarifications

### Session 2026-07-16

- Q: How should `--keys-only --with-elements` behave? → A: Reject it with a clear validation error.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Inspect Elements For One Process Instance (Priority: P1)

As a c8volt operator, I want to include runtime element instances when fetching a known process instance so I can see what that process instance executed without running a separate element query.

**Why this priority**: Keyed inspection is the smallest useful workflow and gives operators immediate execution context for a process instance they already know.

**Independent Test**: Can be fully tested by fetching one process instance with element enrichment and verifying that the process instance appears with an `elements:` section containing its runtime element rows.

**Acceptance Scenarios**:

1. **Given** a supported environment with a process instance that has runtime element instances, **When** the operator runs `c8volt get pi --key <process-instance-key> --with-elements`, **Then** the output shows the selected process instance with an `elements:` section below it.
2. **Given** a runtime element instance has no end date because it is still active, **When** the enriched process-instance output is rendered, **Then** that element row omits the end-date marker.
3. **Given** a runtime element instance has an incident, **When** the enriched process-instance output is rendered, **Then** the element row shows `inc!` when no incident key is available or `inc!:<incidentKey>` when an incident key is available.

---

### User Story 2 - Attach Elements To Process Instance Search Results (Priority: P2)

As a c8volt operator, I want element instances attached to every process instance returned by a bounded search so I can inspect multiple selected process executions in one command while preserving normal process-instance paging behavior.

**Why this priority**: Operators often investigate a filtered set of active or process-specific instances, and the enrichment must work without changing what the process-instance search selects.

**Independent Test**: Can be fully tested by running a process-instance search with `--with-elements`, a limit, and standard paging controls, then verifying that the limit and prompts apply to process instances rather than attached element rows.

**Acceptance Scenarios**:

1. **Given** several active process instances have runtime element instances, **When** the operator runs `c8volt get pi --state active --limit 5 --with-elements`, **Then** up to five process instances are returned and each returned instance includes its matching element rows.
2. **Given** process instances for a specific BPMN process identifier, **When** the operator runs `c8volt get pi -b <bpmn-process-id> --limit 5 --with-elements`, **Then** the selected process instances preserve the existing process filter behavior and each result is enriched with its elements.
3. **Given** a selected process instance contains looped or repeated BPMN elements, **When** elements are attached, **Then** each runtime execution appears as a separate element row under that process instance.

---

### User Story 3 - Combine Elements With Existing Enrichment (Priority: P3)

As a c8volt operator, I want element enrichment to work alongside variables and incidents so I can inspect all relevant process-instance details in one stable tree output.

**Why this priority**: The existing process-instance command already supports enrichment sections, and operators need the new section to compose predictably with those workflows.

**Independent Test**: Can be fully tested by running keyed process-instance lookups with `--with-elements`, `--with-vars`, and `--with-incidents` in combination and verifying section order, data completeness, and absence of duplicate sections.

**Acceptance Scenarios**:

1. **Given** a process instance has variables, incidents, and element instances, **When** the operator requests all enrichments, **Then** the output includes stable `vars:`, `incidents:`, and `elements:` sections for the same process instance.
2. **Given** only element enrichment is requested, **When** the process instance is rendered, **Then** the output includes the `elements:` section without empty variable or incident sections.
3. **Given** JSON output is requested with element enrichment, **When** process-instance results are returned, **Then** each process-instance item includes its attached element instances in the machine-readable result.

### Edge Cases

- Camunda 8.7 must return a clear unsupported-version error for `--with-elements`.
- `--with-elements` must be rejected with `--total` because totals describe process instances and cannot include enrichment sections.
- Keyed process-instance lookup with `--with-elements` must reject incompatible search-mode filters consistently with existing enrichment behavior.
- `--keys-only --with-elements` must be rejected with a clear validation error.
- `--limit` must limit process instances, not attached runtime element instances.
- `--batch-size` and interactive prompts must continue to operate on process-instance pages.
- JSON output must aggregate bounded process-instance results into one payload with attached elements.
- Element-specific filters are out of scope for this process-instance enrichment; operators should use the standalone runtime element inspection command for direct element filtering.
- Element rows must not show listener rows, job rows, summary metrics, loop-count aggregation, or the old `element:<elementId>` suffix.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST add `--with-elements` to `c8volt get process-instance` and the `pi` and `pis` aliases.
- **FR-002**: Operators MUST be able to use `--with-elements` when fetching one process instance by key.
- **FR-003**: Operators MUST be able to use `--with-elements` when listing or searching process instances with existing process-instance filters.
- **FR-004**: For every selected process instance, the system MUST attach that process instance's runtime element instances.
- **FR-005**: The feature MUST reuse the runtime element inspection behavior delivered by the standalone element command from issue #241 rather than defining a separate user-facing element lookup contract.
- **FR-006**: The feature MUST preserve existing process-instance tenant behavior when selecting process instances and attaching elements.
- **FR-007**: The feature MUST preserve existing process-instance paging semantics: `--batch-size` controls process-instance page size, `--limit` limits process instances, interactive prompts are based on process-instance pages, and bounded JSON output aggregates selected process instances into one payload.
- **FR-008**: The system MUST return a clear unsupported-version error for Camunda 8.7 when `--with-elements` is requested.
- **FR-009**: Human output MUST add an `elements:` section under each enriched process instance when `--with-elements` is used.
- **FR-010**: Element rows in human output MUST use the compact element shape with columns ordered as element instance key, type, element identifier, state, start date, optional end date, and optional incident marker.
- **FR-011**: Element row columns MUST remain aligned across rows with different type, element identifier, and state widths.
- **FR-012**: Human output MUST use the established timestamp presentation used by other c8volt command output.
- **FR-013**: Active element rows MUST omit the end-date marker when no end date exists.
- **FR-014**: Element rows with incidents MUST show exactly one incident marker: `inc!` when no incident key is available or `inc!:<incidentKey>` when an incident key is available.
- **FR-015**: Human output MUST NOT render the old `element:<elementId>` suffix for enriched element rows.
- **FR-016**: JSON output MUST include attached element instances under each enriched process-instance item.
- **FR-017**: Each attached element instance MUST preserve the fields needed for operator inspection: `elementInstanceKey`, `elementId`, `elementName`, `type`, `state`, `startDate`, `endDate`, `processInstanceKey`, `rootProcessInstanceKey`, `processDefinitionId`, `processDefinitionKey`, `tenantId`, `hasIncident`, and `incidentKey`.
- **FR-018**: `--with-elements` MUST work cleanly with `--with-vars`, `--with-incidents`, and the combination of both existing enrichment flags.
- **FR-019**: When multiple enrichment sections are requested, human output MUST render sections in a stable order: `vars:`, `incidents:`, then `elements:`.
- **FR-020**: `--with-elements` MUST be rejected with `--total`.
- **FR-021**: Keyed mode with `--with-elements` MUST reject incompatible search-mode filters consistently with existing process-instance enrichment behavior.
- **FR-022**: `--keys-only --with-elements` MUST be rejected with a clear validation error.
- **FR-023**: The feature MUST NOT add element-specific filter flags to the process-instance command.
- **FR-024**: The feature MUST NOT implement the standalone element command, element summary output, metrics output, loop-count aggregation, listener enrichment, job enrichment, execution listener rows, or task listener rows.

### Key Entities

- **Process Instance**: A selected process execution returned by keyed lookup or process-instance search. It remains the primary result item for output, paging, limits, and keys-only behavior.
- **Runtime Element Instance**: A runtime execution occurrence of a BPMN element attached beneath a selected process instance. Key attributes include element instance key, BPMN element identifier, element name, type, state, timestamps, process identifiers, tenant, and incident indicators.
- **Enrichment Section**: A named tree section rendered under a process instance, such as variables, incidents, or elements.
- **Incident Marker**: The compact human-output indicator that an attached runtime element instance has an incident, optionally including the incident key.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can inspect a known process instance and its runtime element instances in one command without running a separate element query.
- **SC-002**: For bounded process-instance searches with element enrichment, 100% of returned process-instance counts respect the requested process-instance limit regardless of how many element rows are attached.
- **SC-003**: 100% of enriched human list output renders element rows with the element identifier as an aligned positional column between type and state.
- **SC-004**: 100% of active attached element rows omit the end-date marker when no end date exists.
- **SC-005**: 100% of attached element rows with incidents show the correct compact incident marker and do not duplicate incident markers.
- **SC-006**: JSON consumers can read attached element instances under each process-instance item for keyed and list/search results.
- **SC-007**: Combined variable, incident, and element enrichment renders stable sections without missing or duplicated data.
- **SC-008**: Invalid combinations, including `--with-elements --total`, `--keys-only --with-elements`, and incompatible keyed/search filter combinations, fail before presenting misleading output.
- **SC-009**: Camunda 8.7 attempts with `--with-elements` return a clear unsupported-version result without mutating operational state.

## Assumptions

- Operators already have a configured c8volt environment and permissions sufficient to inspect process-instance and runtime element data.
- The standalone runtime element inspection capability from issue #241 is available before this feature is implemented.
- Process-instance selection, paging, tenant behavior, and existing enrichment contracts remain authoritative unless explicitly changed in this specification.
- Element-specific searching remains the responsibility of the standalone runtime element command.
- User-facing command help, README-facing examples, generated CLI documentation, and automated validation will be updated because this changes visible process-instance command behavior.
