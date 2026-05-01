# Feature Specification: Tenant Discovery Command

**Feature Branch**: `151-tenant-discovery`  
**Created**: 2026-04-30  
**Status**: Draft  
**Input**: User description: "GitHub issue #151: feat(get): add tenant discovery command"

## GitHub Issue Traceability

- **Issue Number**: 151
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/151
- **Issue Title**: feat(get): add tenant discovery command

## Clarifications

### Session 2026-05-01

- Q: How should `c8volt get tenant --key <id> --filter <text>` behave? → A: Reject as invalid.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - List Tenants Compactly (Priority: P1)

As a Camunda operator, I want `c8volt get tenant` to show the tenants available to my configured environment so that I can discover tenant identifiers and names without leaving the CLI.

**Why this priority**: Tenant listing is the minimum useful tenant discovery workflow and establishes the output contract for the new command.

**Independent Test**: Can be fully tested by running `c8volt get tenant` against a configured environment with multiple tenants and verifying that only relevant non-sensitive tenant information is shown in a predictable order.

**Acceptance Scenarios**:

1. **Given** the configured environment has multiple tenants, **When** the operator runs `c8volt get tenant`, **Then** the command returns a compact human-readable list containing each tenant ID, tenant name, and description when available.
2. **Given** tenants are returned in an arbitrary upstream order, **When** the operator runs `c8volt get tenant`, **Then** the command presents tenants sorted predictably by tenant name and then tenant ID.
3. **Given** the configured environment has no visible tenants, **When** the operator runs `c8volt get tenant`, **Then** the command reports an empty tenant list without exposing sensitive information or failing unexpectedly.

---

### User Story 2 - Show One Tenant by ID (Priority: P2)

As a Camunda operator, I want `c8volt get tenant --key <tenant-id>` to show one tenant by its tenant ID so that I can inspect the relevant details for a known tenant.

**Why this priority**: Direct lookup is the natural second workflow once operators know or receive a tenant ID, and it must share the same command conventions as existing `get` commands.

**Independent Test**: Can be fully tested by running `c8volt get tenant --key tenant-a` against tenant data containing `tenant-a` and other tenants, then verifying that only `tenant-a` is returned.

**Acceptance Scenarios**:

1. **Given** tenant `tenant-a` exists, **When** the operator runs `c8volt get tenant --key tenant-a`, **Then** the command returns only details for `tenant-a`.
2. **Given** tenant `tenant-a` does not exist, **When** the operator runs `c8volt get tenant --key tenant-a`, **Then** the command returns the existing not-found style used by comparable `get` commands.
3. **Given** a tenant has optional descriptive fields omitted, **When** the operator requests that tenant by ID, **Then** the command still displays the available non-sensitive tenant details cleanly.

---

### User Story 3 - Filter Tenant Lists by Name (Priority: P3)

As a Camunda operator, I want `c8volt get tenant --filter <text>` to reduce the tenant list by tenant name so that I can quickly find tenants whose names contain a known fragment.

**Why this priority**: Filtering improves usability for larger tenant sets while keeping the command behavior intentionally simple and testable.

**Independent Test**: Can be fully tested by running `c8volt get tenant --filter demo` against tenants with matching and non-matching names, then verifying that only tenants whose names contain `demo` are returned.

**Acceptance Scenarios**:

1. **Given** tenants include names that contain `demo`, **When** the operator runs `c8volt get tenant --filter demo`, **Then** the command returns only tenants whose names contain that text.
2. **Given** no tenant names contain the filter text, **When** the operator runs `c8volt get tenant --filter demo`, **Then** the command reports an empty tenant list without treating the filter as an error.
3. **Given** the filter text includes wildcard, glob, regex, or query-language characters, **When** the operator runs the command, **Then** those characters are treated as literal text for simple contains matching.

---

### User Story 4 - Use Structured Output and Version-Aware Support (Priority: P4)

As an automation user or maintainer, I want tenant discovery to support existing JSON output and unsupported-version conventions so that scripts and users receive predictable behavior across configured Camunda versions.

**Why this priority**: Structured output and version handling make the command usable in automation and prevent misleading behavior when tenant management is unavailable.

**Independent Test**: Can be fully tested by exercising list and single-tenant modes with `--json`, supported versions, and unsupported versions, then verifying structured non-sensitive output and the existing unsupported-capability style.

**Acceptance Scenarios**:

1. **Given** matching tenants exist, **When** the operator runs `c8volt get tenant --json`, **Then** the command returns structured tenant data for all matching tenants with non-sensitive fields.
2. **Given** tenant `tenant-a` exists, **When** the operator runs `c8volt get tenant --key tenant-a --json`, **Then** the command returns structured tenant data for only `tenant-a` with non-sensitive fields.
3. **Given** the configured Camunda version does not support tenant management through this command, **When** the operator runs any tenant discovery mode, **Then** the command fails using the repository's existing unsupported-capability style.

### Edge Cases

- Tenant descriptions may be absent and must not produce malformed human-readable or JSON output.
- Multiple tenants may share the same name, so tenant ID remains visible and participates in stable sorting.
- Tenant IDs and names may differ only by case; sorting and filtering must remain deterministic.
- Combining `--key` and `--filter` must be rejected as an invalid flag combination.
- Filter text is matched as a literal substring of the tenant name; wildcard, glob, regex, and query-language syntax must not be interpreted.
- JSON output must not include credentials, tokens, secrets, authorization internals, or other sensitive data.
- Unsupported Camunda versions must fail before presenting partial or misleading tenant results.
- Existing `get` commands must keep their current behavior.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide a `c8volt get tenant` command for tenant discovery.
- **FR-002**: The system MUST support listing tenants without requiring a tenant ID.
- **FR-003**: Human-readable list output MUST include tenant ID, tenant name, and tenant description when a description is available.
- **FR-004**: Human-readable output MUST remain compact and MUST include only relevant non-sensitive tenant information.
- **FR-005**: Listed tenants MUST be sorted predictably by tenant name and then tenant ID.
- **FR-006**: The system MUST support `c8volt get tenant --key <tenant-id>` for single-tenant lookup by tenant ID.
- **FR-007**: Single-tenant lookup MUST return only the selected tenant when the tenant exists.
- **FR-008**: Single-tenant lookup for an absent tenant MUST use the existing not-found style used by comparable `get` commands.
- **FR-009**: The system MUST support `c8volt get tenant --filter <text>` for list filtering by tenant name.
- **FR-010**: Tenant filtering MUST use simple literal contains matching against tenant names.
- **FR-011**: Tenant filtering MUST NOT implement wildcard matching, glob matching, regex matching, or a query language.
- **FR-011a**: The system MUST reject combining `--key` and `--filter` as an invalid flag combination.
- **FR-012**: List mode with `--json` MUST return structured tenant data for all matching tenants.
- **FR-013**: Single-tenant mode with `--json` MUST return structured tenant data for the selected tenant.
- **FR-014**: JSON output MUST include relevant non-sensitive tenant information available to the command.
- **FR-015**: JSON output MUST NOT include sensitive information.
- **FR-016**: Tenant list, filter, single-tenant, and JSON modes MUST support the relevant global flags already supported by comparable `get` commands, including configuration, profile, authentication, Camunda version selection, output mode, and logging behavior.
- **FR-017**: Camunda versions without tenant-management support MUST fail using the repository's existing unsupported-capability style.
- **FR-018**: Existing `get` command behavior MUST remain unchanged.
- **FR-019**: Automated tests MUST cover list output, single-tenant lookup, filtering, sorting, JSON output, unsupported-version behavior, and preservation of existing `get` command behavior.
- **FR-020**: User-facing CLI documentation MUST stay in sync with the new tenant command when repository documentation generation applies.

### Key Entities *(include if feature involves data)*

- **Tenant**: A Camunda tenant visible to the configured environment, identified by tenant ID and described by non-sensitive display fields such as name and optional description.
- **Tenant List Result**: The sorted collection of tenants presented by list mode after applying any name filter.
- **Tenant Lookup Result**: The single tenant returned for a requested tenant ID, or the comparable not-found outcome when no tenant matches.
- **Tenant Name Filter**: Literal text used to include only tenants whose names contain that text.
- **Tenant Output Shape**: The human-readable or JSON representation of tenant data that excludes sensitive information.
- **Unsupported Tenant Capability**: A configured Camunda version or environment where tenant discovery cannot be supported and must use the existing unsupported-capability outcome.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `c8volt get tenant` returns all visible tenants in a stable order by name and then ID in automated coverage.
- **SC-002**: Human-readable list and single-tenant output show tenant ID and name for every returned tenant, include descriptions when available, and omit sensitive data.
- **SC-003**: `c8volt get tenant --key tenant-a` returns only tenant `tenant-a` when it exists and the comparable not-found outcome when it does not.
- **SC-004**: `c8volt get tenant --filter demo` returns only tenants whose names contain `demo` as literal text.
- **SC-005**: Automated filtering coverage demonstrates that wildcard, glob, regex, and query-language patterns are treated as literal filter text.
- **SC-006**: List and single-tenant JSON output return structured non-sensitive tenant data matching the selected mode.
- **SC-007**: Automated coverage demonstrates that unsupported Camunda versions fail with the existing unsupported-capability style.
- **SC-008**: Existing `get` command tests continue to pass after the tenant command is added.
- **SC-009**: Generated CLI documentation includes the tenant command and its supported flags when documentation generation is required.

## Assumptions

- Tenant discovery is intended for operators and automation users who already have valid configuration and authentication for the target Camunda environment.
- The tenant ID is the stable identifier used by `--key`.
- Name filtering is a convenience for list mode and does not replace direct tenant ID lookup.
- Literal contains filtering is case-sensitive unless an existing repository-wide command convention requires a different behavior.
- Empty list results are successful command outcomes, not errors.
- Unsupported-version behavior should match the repository's existing style rather than introducing a new tenant-specific error contract.
