# Feature Specification: Keyed Process-Instance Incident Details

**Feature Branch**: `154-get-pi-incidents`  
**Created**: 2026-05-02  
**Status**: Draft  
**Input**: User description: "GitHub issue #154: feat(get-pi): add --with-incidents for keyed process instance incident messages"

## GitHub Issue Traceability

- **Issue Number**: 154
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/154
- **Issue Title**: feat(get-pi): add --with-incidents for keyed process instance incident messages

## Clarifications

### Session 2026-05-02

- Q: How should human-readable `--with-incidents` output place incident messages? → A: Print the normal process-instance row, then indented `incident <incident-key>:` lines below it.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Show Incident Messages for Keyed Lookup (Priority: P1)

As a Camunda operator diagnosing a known process instance, I want `c8volt get pi --key <process-instance-key> --with-incidents` to show incident messages next to the keyed process-instance result so that I can understand the failure without leaving the CLI.

**Why this priority**: This is the core workflow requested in the issue and removes the extra manual lookup that currently blocks fast diagnosis.

**Independent Test**: Run keyed process-instance lookup with `--with-incidents` against a fixture where the process instance has one or more incidents, then verify the normal process-instance row is returned unchanged and each incident error message appears as an indented `incident <incident-key>:` line directly below its matching process-instance row.

**Acceptance Scenarios**:

1. **Given** a keyed process instance has one active incident with an error message, **When** the operator runs `c8volt get pi --key <key> --with-incidents`, **Then** the command returns the normal process-instance row and shows the incident error message as an indented `incident <incident-key>:` line directly below it.
2. **Given** a keyed process instance has multiple incidents, **When** the operator runs the command with `--with-incidents`, **Then** the command shows all returned incident messages as indented `incident <incident-key>:` lines below that process-instance row.
3. **Given** a keyed process instance has no incidents, **When** the operator runs the command with `--with-incidents`, **Then** the command still renders the process instance successfully and does not imply that an incident message exists.

---

### User Story 2 - Consume Incident Details in JSON (Priority: P2)

As an automation author, I want JSON output for keyed process-instance lookup with `--with-incidents` to include incident details in a stable machine-readable shape so that scripts can inspect incident messages without parsing human text.

**Why this priority**: The command already supports structured output, and incident enrichment must be useful for automation as well as interactive diagnosis.

**Independent Test**: Run keyed process-instance lookup with `--json --with-incidents` against fixtures with incidents and without incidents, then verify that each returned process instance carries a predictable incident-details collection.

**Acceptance Scenarios**:

1. **Given** a keyed process instance has incident data, **When** the operator runs `c8volt get pi --key <key> --with-incidents --json`, **Then** the JSON payload includes incident details for that process instance, including the incident error message.
2. **Given** multiple `--key` values are requested, **When** JSON output is requested with `--with-incidents`, **Then** incident details remain associated with the matching process-instance key.
3. **Given** a requested process instance has no incidents, **When** JSON output is requested with `--with-incidents`, **Then** the machine-readable result represents an empty incident-details collection for that process instance.

---

### User Story 3 - Protect Existing Command Semantics (Priority: P3)

As a CLI user with existing scripts, I want `--with-incidents` to be accepted only for direct keyed lookup and to leave all existing default and search-mode behavior unchanged so that the new feature is additive and predictable.

**Why this priority**: The existing command has separate keyed and search modes. Mixing incident enrichment into search filters would create unclear semantics and risk regressions.

**Independent Test**: Exercise validation and regression cases: `--with-incidents` without `--key`, search-mode incident filters, and default keyed lookup without the new flag.

**Acceptance Scenarios**:

1. **Given** no `--key` value is provided, **When** the operator runs `c8volt get pi --with-incidents`, **Then** the command fails with a clear validation error.
2. **Given** existing search-mode filters `--incidents-only` or `--no-incidents-only` are used, **When** `--with-incidents` is omitted, **Then** their current behavior is preserved.
3. **Given** keyed lookup is run without `--with-incidents`, **When** the command renders human-readable or JSON output, **Then** the current default output shape remains unchanged.

---

### User Story 4 - Respect Tenant and Version Boundaries (Priority: P4)

As an operator in tenant-aware environments, I want incident lookups to use tenant-aware command configuration and version-safe client behavior so that the command does not expose cross-tenant or misleading incident data.

**Why this priority**: The issue explicitly calls out tenant-aware lookup paths and version-specific support boundaries. Correctness and safety matter more than opportunistic fallback behavior.

**Independent Test**: Run service-level and command-level tests that verify tenant filters are included for supported versions and unsupported-version behavior is clear for Camunda 8.7 when tenant-safe keyed incident lookup is unavailable.

**Acceptance Scenarios**:

1. **Given** a tenant is configured, **When** incident enrichment is requested for a keyed process instance, **Then** the incident lookup includes tenant filtering.
2. **Given** Camunda 8.8 or 8.9 is configured, **When** `--with-incidents` is used with `--key`, **Then** the command uses the generated Camunda incident search capability and returns matching incident details.
3. **Given** Camunda 8.7 cannot provide tenant-safe keyed incident enrichment, **When** `--with-incidents` is used with `--key`, **Then** the command fails with the repository's existing unsupported-capability style instead of falling back to tenant-unsafe direct incident lookup.

### Edge Cases

- `--with-incidents` without `--key` must fail before attempting process-instance or incident requests.
- `--with-incidents` combined with search-mode filters must be rejected because the feature is scoped to direct keyed lookup.
- Multiple `--key` values must fetch and render incident details for each returned process instance without mixing incident messages between keys.
- A process instance that has the existing incident marker but whose incident search returns no items must still render successfully with an empty incident-details collection.
- Incident results with empty error messages must not break human-readable or JSON rendering.
- Human-readable incident messages must render as indented `incident <incident-key>:` lines directly below the matching process-instance row.
- A configured tenant must be included in incident lookup filters when the incident search API supports tenant filtering.
- Camunda 8.7 must not use a tenant-unsafe direct incident lookup as a fallback for this feature.
- Existing `--incidents-only` and `--no-incidents-only` search-mode behavior must remain unchanged.
- Existing default human-readable and JSON process-instance output must remain unchanged when `--with-incidents` is omitted.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide a `--with-incidents` flag on `c8volt get process-instance` and its `get pi` alias.
- **FR-002**: The system MUST accept `--with-incidents` only when one or more `--key` values are provided.
- **FR-003**: The system MUST reject `--with-incidents` without `--key` with a clear validation error.
- **FR-004**: The system MUST reject `--with-incidents` when combined with search-mode filters, including `--incidents-only` and `--no-incidents-only`.
- **FR-005**: When `--with-incidents` is provided, the system MUST fetch incident data for each requested process-instance key that is successfully returned.
- **FR-006**: Human-readable output MUST preserve the normal process-instance row and show returned incident keys and error messages as indented `incident <incident-key>:` lines directly below the matching row.
- **FR-007**: Human-readable output MUST continue to render process instances successfully when no incident details are returned.
- **FR-008**: JSON output MUST include incident details in a machine-readable shape for each returned process instance when `--with-incidents` is provided.
- **FR-009**: JSON incident details MUST include each incident's error message when available and enough key information to associate the incident with the process instance.
- **FR-010**: Multiple requested keys MUST preserve the association between each process instance and its own incident details.
- **FR-011**: When `--with-incidents` is omitted, the command MUST preserve current human-readable and JSON output behavior.
- **FR-012**: Existing search-mode `--incidents-only` and `--no-incidents-only` behavior MUST remain unchanged.
- **FR-013**: Incident enrichment MUST use tenant-aware incident lookup paths where the configured Camunda version supports them.
- **FR-014**: If a tenant is configured and the incident search API supports tenant filtering, the incident lookup MUST include the configured tenant.
- **FR-015**: Camunda 8.8 and 8.9 support MUST use generated Camunda client incident search APIs that expose process-instance-key and tenant filters and return error messages.
- **FR-016**: Camunda 8.7 MUST return the repository's existing unsupported-capability style for `--with-incidents` when tenant-safe keyed incident enrichment cannot be implemented.
- **FR-017**: The implementation MUST NOT use a tenant-unsafe direct incident lookup as the primary incident enrichment path.
- **FR-018**: Command help and generated user-facing documentation MUST describe `--with-incidents`, its keyed-only scope, and its incident-key/message purpose.
- **FR-019**: Automated tests MUST cover validation, human-readable output, JSON output, tenant-aware request construction, multiple keys, no-incident results, default-output preservation, search-mode incident-filter preservation, and version-specific unsupported behavior.

### Key Entities *(include if feature involves data)*

- **Process Instance Lookup Result**: A process instance returned by direct keyed lookup, including existing fields such as key, state, tenant ID, and incident marker.
- **Incident Detail**: Incident data associated with a process instance, including incident key when available, process-instance key, tenant ID when available, state/type metadata when available, and error message.
- **Incident-Enriched Process Instance**: The output representation used only when `--with-incidents` is requested, combining a process instance with its incident details.
- **Tenant-Aware Incident Search**: The supported incident lookup request that filters by process-instance key and configured tenant where available.
- **Unsupported Incident Enrichment**: The version-specific outcome for environments where tenant-safe keyed incident enrichment cannot be provided.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated command tests show `c8volt get pi --key <key> --with-incidents` preserves the normal process-instance row and includes returned incident keys and error messages as indented `incident <incident-key>:` lines below it.
- **SC-002**: Automated command tests show `c8volt get pi --key <key> --with-incidents --json` includes machine-readable incident details with error messages.
- **SC-003**: Automated validation tests show `--with-incidents` without `--key` fails with a clear validation error.
- **SC-004**: Automated regression tests show existing default keyed output is unchanged when `--with-incidents` is omitted.
- **SC-005**: Automated regression tests show search-mode `--incidents-only` and `--no-incidents-only` behavior remains unchanged.
- **SC-006**: Automated tests show multiple keys keep incident details attached to the correct process-instance key.
- **SC-007**: Automated tests show process instances without returned incidents still render successfully.
- **SC-008**: Service-level tests for supported versions verify incident search requests include process-instance-key filtering and configured tenant filtering.
- **SC-009**: Version-specific tests verify Camunda 8.7 returns the existing unsupported-capability style for this flag when tenant-safe enrichment is unavailable.
- **SC-010**: Help output or generated CLI documentation includes `--with-incidents` and states that it applies to keyed process-instance lookup.

## Assumptions

- The feature is intended for direct diagnosis of known process-instance keys, not for broad search-mode incident reporting.
- Existing process-instance keyed lookup remains the source of truth for whether a requested key exists.
- Incident details are supplemental output and must not change the command's process-instance selection semantics.
- Tenant safety has priority over partial version support; unsupported behavior is preferred to tenant-unsafe fallback.
- Documentation generated from command metadata should be regenerated through the repository's existing docs path after command metadata changes.
