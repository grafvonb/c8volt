# Feature Specification: Walk Process-Instance Incident Details

**Feature Branch**: `157-walk-pi-incidents`  
**Created**: 2026-05-02  
**Status**: Draft  
**Input**: User description: "GitHub issue #157: feat(walk-pi): add --with-incidents for keyed process instance walk output"

## GitHub Issue Traceability

- **Issue Number**: 157
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/157
- **Issue Title**: feat(walk-pi): add --with-incidents for keyed process instance walk output

## Clarifications

### Session 2026-05-02

- Q: What should happen if incident enrichment fails for a walked process instance? → A: Fail the whole command.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Show Incident Messages While Walking (Priority: P1)

As a Camunda operator diagnosing a process-instance tree, I want `c8volt walk pi --key <process-instance-key> --with-incidents` to show incident messages for walked process instances so that I can inspect failures without running a separate lookup for each instance.

**Why this priority**: This is the core value requested by the issue and makes keyed process-instance walks immediately useful for incident diagnosis.

**Independent Test**: Run a keyed walk against fixture data where at least one walked process instance has incident details, then verify normal walk ordering remains intact and each incident error message is rendered under its matching process-instance row.

**Acceptance Scenarios**:

1. **Given** a walked process-instance tree contains one instance with an incident error message, **When** the operator runs `c8volt walk pi --key <key> --with-incidents`, **Then** the command walks the tree and shows the incident error message under the matching process instance.
2. **Given** multiple walked instances have incidents, **When** the operator runs the command with `--with-incidents`, **Then** all returned incident messages remain associated with the correct process-instance key.
3. **Given** a walked process instance has no returned incidents, **When** the operator runs the command with `--with-incidents`, **Then** the instance still renders successfully without implying that an incident message exists.

---

### User Story 2 - Consume Walk Incident Details in JSON (Priority: P2)

As an automation author, I want JSON walk output with `--with-incidents` to include incident details in a stable machine-readable shape so that scripts can inspect process-instance tree incidents without parsing human text.

**Why this priority**: Walk output already supports JSON and the feature must be useful for scripted diagnostics as well as interactive CLI use.

**Independent Test**: Run keyed walk JSON output with incident fixture data and verify each walked item includes a predictable incident-details collection for its process-instance key.

**Acceptance Scenarios**:

1. **Given** a walked process instance has incident data, **When** the operator runs `c8volt walk pi --key <key> --with-incidents --json`, **Then** the JSON payload includes incident details for that process instance, including the incident error message.
2. **Given** multiple walked process instances have incidents, **When** JSON output is requested with `--with-incidents`, **Then** incident details remain attached to the matching process-instance item.
3. **Given** a walked process instance has no incidents, **When** JSON output is requested with `--with-incidents`, **Then** that item represents an empty incident-details collection.

---

### User Story 3 - Preserve Walk Traversal Semantics (Priority: P3)

As a CLI user with existing walk workflows, I want `--with-incidents` to be additive and keyed-only so that current traversal behavior, ordering, parent/child relationships, tree rendering, tenant boundaries, and default output remain unchanged when the flag is omitted.

**Why this priority**: The walk command is used to reason about process relationships. Incident enrichment must not change what is walked or how existing scripts interpret default output.

**Independent Test**: Exercise default walk modes and validation cases with and without `--with-incidents`, then verify traversal keys, ordering, warnings, and default output match current behavior unless incident enrichment is explicitly requested.

**Acceptance Scenarios**:

1. **Given** `--with-incidents` is omitted, **When** a user runs any supported keyed walk mode, **Then** existing human-readable and JSON output remain unchanged.
2. **Given** `--with-incidents` is used without a valid `--key`, **When** command validation runs, **Then** the command fails with a clear validation error before performing lookups.
3. **Given** a family walk has missing ancestor warnings or partial traversal results, **When** `--with-incidents` is used, **Then** existing traversal warnings and partial-result semantics are preserved.

---

### User Story 4 - Respect Tenant and Version Boundaries (Priority: P4)

As an operator in tenant-aware environments, I want walk incident enrichment to reuse tenant-aware incident lookup behavior so that `walk --key --with-incidents` does not expose cross-tenant data or silently rely on unsupported version behavior.

**Why this priority**: The issue explicitly requires tenant-aware lookup paths and version-specific support boundaries. Safety and clear unsupported behavior matter more than opportunistic fallback behavior.

**Independent Test**: Run facade, command, and service-level tests proving supported versions include tenant filtering and Camunda 8.7 fails with the existing unsupported-capability style when tenant-safe enrichment is unavailable.

**Acceptance Scenarios**:

1. **Given** a tenant is configured, **When** incident enrichment is requested for walked process instances, **Then** incident lookup includes tenant filtering where the Camunda request body supports it.
2. **Given** Camunda 8.8 or 8.9 is configured, **When** `walk pi --key <key> --with-incidents` is used, **Then** the command uses the generated Camunda process-instance incident search capability.
3. **Given** Camunda 8.7 cannot provide tenant-safe keyed incident enrichment, **When** `--with-incidents` is used with walk, **Then** the command fails with the repository's existing unsupported-capability style instead of falling back to tenant-unsafe direct incident lookup.

### Edge Cases

- `--with-incidents` without a usable `--key` must fail before attempting process-instance or incident requests.
- Incident lookups must run only for process instances actually returned by the selected walk mode.
- Multiple walked instances with incidents must keep each incident collection associated with the matching process-instance key.
- Partial family results and missing ancestor warnings must still render warnings after incident enrichment.
- Tree rendering must keep existing branch layout and attach incident messages without changing traversal order.
- `--keys-only --with-incidents` must be rejected with a clear validation error because key-only output cannot carry incident details.
- If any requested incident lookup fails after traversal succeeds, the command must fail instead of rendering a partially enriched traversal.
- JSON output must preserve traversal metadata such as mode, outcome, root key, keys, edges, warnings, and missing ancestors.
- Incident results with empty error messages must not break human-readable or JSON rendering.
- Tenant filtering must be included in incident lookup when a tenant is configured and supported.
- Camunda 8.7 must not use a tenant-unsafe direct incident lookup as a fallback for this feature.
- Existing default walk output must remain unchanged when `--with-incidents` is omitted.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide a `--with-incidents` flag on `c8volt walk process-instance` and its `walk pi` alias.
- **FR-002**: The system MUST accept `--with-incidents` only for keyed process-instance walks using `--key`.
- **FR-003**: The system MUST reject `--with-incidents` without a usable `--key` with a clear validation error.
- **FR-004**: When `--with-incidents` is provided, the system MUST fetch incident data only for walked process instances returned by the selected walk mode.
- **FR-005**: Human-readable output MUST preserve current walk ordering and show returned incident keys and error messages directly under the matching process-instance row.
- **FR-006**: Human-readable output MUST continue to render process instances successfully when no incident details are returned.
- **FR-007**: JSON output MUST include incident details in a machine-readable shape for each walked process instance when `--with-incidents` is provided.
- **FR-008**: JSON incident details MUST include each incident's error message when available and enough key information to associate the incident with the process instance.
- **FR-009**: Multiple walked process instances MUST preserve the association between each process instance and its own incident details.
- **FR-010**: When `--with-incidents` is omitted, the command MUST preserve current human-readable, key-only, tree, and JSON output behavior.
- **FR-011**: Incident enrichment MUST preserve current traversal behavior, ordering, parent/child relationships, root/start key semantics, partial-result warnings, and tenant boundaries.
- **FR-012**: The system MUST reject `--keys-only --with-incidents` with a clear validation error.
- **FR-013**: Incident enrichment MUST use the tenant-aware incident lookup/model/output conventions introduced for issue #154 where they apply.
- **FR-014**: If a tenant is configured and the incident search API supports tenant filtering, incident lookups MUST include the configured tenant.
- **FR-015**: Camunda 8.8 and 8.9 support MUST use generated Camunda client incident search APIs that expose the process-instance-key path parameter and return incident messages.
- **FR-016**: Incident search request bodies MUST avoid redundant `filter.processInstanceKey` values when the process-instance key is already scoped by the generated API path and that redundancy would be rejected by Camunda.
- **FR-017**: If incident lookup fails for any walked process instance while `--with-incidents` is requested, the command MUST fail instead of rendering partially enriched output.
- **FR-018**: Camunda 8.7 MUST return the repository's existing unsupported-capability style for `--with-incidents` when tenant-safe keyed incident enrichment cannot be implemented.
- **FR-019**: The implementation MUST NOT use a tenant-unsafe direct incident lookup as the primary incident enrichment path.
- **FR-020**: Command help and generated user-facing documentation MUST describe `--with-incidents`, its keyed-walk scope, and its incident-message purpose.
- **FR-021**: Automated tests MUST cover validation, human-readable output, JSON output, tenant-aware request construction, incident lookup failure, no-incident results, default-output preservation, traversal preservation, key-only rejection, and version-specific unsupported behavior where applicable.

### Key Entities *(include if feature involves data)*

- **Traversal Result**: A keyed walk result containing mode, outcome, root key, ordered process-instance keys, parent/child edges, process-instance items, warnings, and missing ancestor metadata.
- **Walked Process Instance**: A process instance returned by the selected walk mode, including existing fields such as key, parent key, state, tenant ID, and incident marker.
- **Incident Detail**: Incident data associated with a process instance, including incident key when available, process-instance key, tenant ID when available, state/type metadata when available, and error message.
- **Incident-Enriched Traversal Item**: The output representation used only when `--with-incidents` is requested, combining a walked process instance with its incident details.
- **Incident-Enriched Traversal Result**: The traversal payload used only when incident enrichment is requested, preserving traversal metadata while adding per-item incident details.
- **Tenant-Aware Incident Search**: The supported incident lookup request that filters by configured tenant where available and uses the process-instance-key scoped generated client operation.
- **Unsupported Incident Enrichment**: The version-specific outcome for environments where tenant-safe keyed incident enrichment cannot be provided.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated command tests show `c8volt walk pi --key <key> --with-incidents` preserves walk ordering and includes returned incident keys and error messages under the matching process-instance rows.
- **SC-002**: Automated command tests show `c8volt walk pi --key <key> --with-incidents --json` includes machine-readable incident details with error messages for walked items.
- **SC-003**: Automated validation tests show `--with-incidents` without a usable `--key` fails with a clear validation error.
- **SC-004**: Automated regression tests show existing default walk output is unchanged when `--with-incidents` is omitted.
- **SC-005**: Automated regression tests show existing traversal modes, tree output, ordering, warnings, and parent/child relationships remain unchanged.
- **SC-006**: Automated tests show multiple walked process instances keep incident details attached to the correct process-instance key.
- **SC-007**: Automated tests show process instances without returned incidents still render successfully.
- **SC-008**: Service or facade tests for supported versions verify incident search requests include configured tenant filtering and do not redundantly send a rejected process-instance-key filter.
- **SC-009**: Automated command or facade tests show incident lookup failure causes the command to fail rather than render partial incident data.
- **SC-010**: Version-specific tests verify Camunda 8.7 returns the existing unsupported-capability style for this flag when tenant-safe enrichment is unavailable.
- **SC-011**: Help output or generated CLI documentation includes `--with-incidents` and states that it applies to keyed process-instance walks.

## Assumptions

- The feature is intended for keyed process-instance walks only; broad search-mode incident reporting remains out of scope.
- Existing traversal results remain the source of truth for which process instances receive incident enrichment.
- Incident details are supplemental output and must not change the command's traversal selection semantics.
- Human-readable incident messages should follow the issue #154 convention by rendering directly below the matching process-instance row.
- JSON enrichment should reuse the issue #154 incident detail model where practical while preserving walk-specific traversal metadata.
- Tenant safety has priority over partial version support; unsupported behavior is preferred to tenant-unsafe fallback.
- Documentation generated from command metadata should be regenerated through the repository's existing docs path after command metadata changes.
