# Feature Specification: All-Tenants Tenant Override

**Feature Branch**: `282-all-tenants-override`

**Created**: 2026-08-30

**Status**: Draft

**GitHub Issue**: [#282](https://github.com/grafvonb/c8volt/issues/282) - feat(cli): add global --all-tenants tenant override

**Input**: GitHub issue #282 requests an explicit global `--all-tenants` option that clears configured tenant filtering for operations that may work across all tenants visible to the authenticated user, while protecting operations that require a concrete destination tenant.

## Issue Traceability

- **GitHub Issue**: #282
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/282
- **Issue Title**: feat(cli): add global --all-tenants tenant override

## Clarifications

### Session 2026-08-30

- Q: When `--all-tenants` clears a named configured tenant, how should existing human tenant-context output report that override? → A: Preserve the established broadening safeguard: show the configured tenant, emit one warning that `--all-tenants` overrides its filter, then show the unfiltered scope.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Search Across Visible Tenants Explicitly (Priority: P1)

As an operator with a tenant configured by default, I want an obvious `--all-tenants` option so that I can read, search, discover, and select resources across every tenant I am authorized to see without relying on an obscure empty tenant value.

**Why this priority**: This is the primary user value: making an existing cross-tenant discovery capability understandable and convenient while preserving authorization boundaries.

**Independent Test**: Configure a named tenant, run representative read, search, discovery, and selection commands with `--all-tenants`, and verify each command uses no tenant filter and can return resources from multiple accessible tenants.

**Acceptance Scenarios**:

1. **Given** `tenant-a` is selected through base configuration, **When** an operator runs a discovery command with `--all-tenants`, **Then** the command performs discovery without a tenant filter.
2. **Given** `tenant-a` is selected through a profile, **When** an operator runs a search command with `--all-tenants`, **Then** the profile tenant is overridden and the search is unfiltered across accessible tenants.
3. **Given** `tenant-a` is selected through the environment, **When** an operator runs a read or selection command with `--all-tenants`, **Then** the environment tenant is overridden and the operation receives an empty tenant filter.
4. **Given** the authenticated user can see resources in `tenant-a` but cannot see resources in `tenant-b`, **When** the operator uses `--all-tenants`, **Then** the command can include visible `tenant-a` resources but does not gain access to `tenant-b` resources.
5. **Given** a command already reports human tenant context for a named configured tenant, **When** `--all-tenants` clears that tenant, **Then** the output shows the configured tenant, emits one warning `--all-tenants overrides the configured tenant filter; selection is unfiltered`, and then shows the unfiltered selection scope.

---

### User Story 2 - Prevent Ambiguous Tenant Overrides (Priority: P2)

As an operator or automation author, I want explicit tenant choices to be unambiguous so that a command never has to guess whether a named tenant or all visible tenants should win.

**Why this priority**: Conflicting command-line instructions can broaden or narrow scope unexpectedly and must fail before any operational work begins.

**Independent Test**: Run the same harmless command with both `--tenant` and `--all-tenants`, including an explicitly empty tenant value, and verify every combination returns an invalid-input result without starting the command's work.

**Acceptance Scenarios**:

1. **Given** no tenant is configured, **When** an operator supplies both `--tenant tenant-a` and `--all-tenants`, **Then** the command returns an invalid-input result and performs no command work.
2. **Given** a tenant is configured from any non-command-line source, **When** the operator supplies only `--all-tenants`, **Then** the command accepts the request and overrides the configured tenant.
3. **Given** an operator explicitly supplies `--tenant ""`, **When** the operator also supplies `--all-tenants`, **Then** the command returns an invalid-input result even though both choices would otherwise produce an empty tenant filter.
4. **Given** an operator supplies neither tenant option, **When** any existing command runs, **Then** tenant resolution and command behavior remain unchanged.

---

### User Story 3 - Protect Concrete Tenant Destinations (Priority: P3)

As an operator creating or launching work, I want commands that require one destination tenant to reject `--all-tenants` so that they never silently place new work in the default tenant or an unintended tenant.

**Why this priority**: An all-tenants discovery scope has no safe single destination meaning; rejecting it prevents ambiguous creation and mutation targets.

**Independent Test**: Invoke representative create, deploy, and run operations with `--all-tenants` and verify each returns an invalid-input result before submission, while the same operations continue to work with a concrete tenant or their existing default behavior.

**Acceptance Scenarios**:

1. **Given** a deploy operation requires one destination tenant, **When** an operator supplies `--all-tenants`, **Then** the command returns an invalid-input result before deployment begins.
2. **Given** a run operation creates new work in one tenant, **When** an operator supplies `--all-tenants`, **Then** the command returns an invalid-input result before the run is submitted.
3. **Given** any create operation requires one destination tenant, **When** an operator supplies `--all-tenants`, **Then** the command does not reinterpret the option as the default tenant and performs no creation.
4. **Given** a concrete-destination command is run without `--all-tenants`, **When** its tenant is resolved, **Then** its existing named-tenant and default-tenant behavior remains unchanged.

---

### User Story 4 - Discover And Automate The New Contract (Priority: P4)

As an operator or tool author, I want command help and capability information to describe `--all-tenants` accurately so that I can discover where it is accepted, detect where it is rejected, and use it safely in scripts.

**Why this priority**: The feature replaces obscure syntax only if users and automation can discover its meaning and command scope reliably.

**Independent Test**: Inspect root and representative subcommand help, generated command documentation, and capability information; verify the option, exclusivity rule, unfiltered meaning, and concrete-destination restriction are represented consistently.

**Acceptance Scenarios**:

1. **Given** an operator views root or applicable subcommand help, **When** inherited options are displayed, **Then** `--all-tenants` is shown with a concise description of its unfiltered, visibility-bounded behavior.
2. **Given** automation inspects command capabilities, **When** a command supports or rejects the all-tenants override, **Then** the available metadata communicates the applicable option contract consistently with command execution.
3. **Given** an operator reads generated command documentation, **When** tenant options are described, **Then** the documentation explains mutual exclusivity and the restriction for concrete-destination commands.

### Edge Cases

- A named tenant may come from base configuration, a selected profile, or the environment; `--all-tenants` must override each source regardless of its normal precedence.
- An explicitly supplied empty `--tenant ""` and `--all-tenants` express equivalent filtering outcomes but remain conflicting command-line choices and must be rejected together.
- `--all-tenants` may return resources from only one tenant or no resources at all; the operation remains unfiltered because result contents do not redefine the requested scope.
- A user may have access to only one tenant; the option must not imply or grant access beyond that user's existing visibility.
- Direct resource-key operations may already rely on backend authorization rather than tenant filtering; `--all-tenants` must not weaken, bypass, or redefine that authorization behavior.
- An option inherited from the root may be written before or after the subcommand where inherited root options are normally accepted; its meaning must remain the same.
- Validation failure for conflicting tenant options or an invalid destination must occur before remote requests, prompts, progress activity, or mutation submission.
- Machine-readable, quiet, total-only, and keys-only output contracts must remain unchanged except where existing capability or tenant-context contracts explicitly carry the new option state.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The command-line interface MUST provide `--all-tenants` as a root-level option inherited by the same command scope as `--tenant`.
- **FR-002**: Supplying `--all-tenants` MUST set the effective tenant filter to empty for read, search, discovery, and selection operations.
- **FR-003**: `--all-tenants` MUST override a tenant resolved from base configuration, a selected profile, or the environment.
- **FR-004**: “All tenants” MUST mean all tenants visible to the authenticated user and MUST NOT bypass or expand existing authorization.
- **FR-005**: An explicitly supplied `--tenant` and `--all-tenants` MUST be mutually exclusive, including when the explicit tenant value is empty.
- **FR-006**: A conflict between `--tenant` and `--all-tenants` MUST return the established invalid-input result before command work begins.
- **FR-007**: When neither `--tenant` nor `--all-tenants` is supplied, existing tenant-source precedence and effective tenant behavior MUST remain unchanged.
- **FR-008**: Existing explicit empty-tenant behavior MUST remain supported when `--all-tenants` is absent.
- **FR-009**: Commands that require one concrete destination tenant, including create, deploy, and run operations, MUST reject `--all-tenants` with the established invalid-input result before submitting work.
- **FR-010**: Concrete-destination commands MUST NOT interpret `--all-tenants` as the default tenant, an arbitrary visible tenant, or an instruction to repeat creation across tenants.
- **FR-011**: Direct resource-key operations MUST retain their existing backend-authorization behavior; the all-tenants option MUST NOT activate any authorization-bypass behavior.
- **FR-012**: Where existing human tenant-context output applies, clearing a named configured tenant with `--all-tenants` MUST show the configured tenant, emit the warning `--all-tenants overrides the configured tenant filter; selection is unfiltered` exactly once, and then show the unfiltered selection scope; structured tenant context MUST report the effective unfiltered scope through its existing schema without adding command-line provenance, and quiet, total-only, and keys-only output MUST remain unchanged.
- **FR-013**: Root and applicable command help MUST explain that `--all-tenants` clears configured tenant filtering, remains bounded by user visibility, conflicts with explicit `--tenant`, and is invalid for concrete-destination operations.
- **FR-014**: Command capability information MUST be updated so tools can distinguish commands that accept the inherited all-tenants behavior from commands that reject it because they require a concrete destination.
- **FR-015**: Generated command documentation and user examples MUST show the supported all-tenants syntax and its safety constraints.
- **FR-016**: Automated acceptance coverage MUST include overrides of every configured tenant source, mutual exclusion with named and empty explicit tenant values, representative discovery operations, representative concrete-destination rejections, unchanged behavior when absent, help, capabilities, and output-mode compatibility.

### Key Entities *(include if feature involves data)*

- **Configured Tenant**: A tenant value obtained from base configuration, a selected profile, or the environment before explicit command-line choices are applied.
- **Explicit Tenant Choice**: Either a supplied `--tenant` value or the `--all-tenants` option; at most one may be present in a command invocation.
- **Effective Tenant Filter**: The tenant constraint used by a read, search, discovery, or selection operation after configuration and explicit choices are resolved; it is empty when `--all-tenants` is active.
- **Concrete Destination Tenant**: The single named or default tenant required by an operation that creates, deploys, or runs work; it cannot be represented by `--all-tenants`.
- **Visible Tenant Set**: The resources and tenants the authenticated user is already authorized to access; `--all-tenants` can broaden filtering only within this set.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In acceptance tests, 100% of representative read, search, discovery, and selection commands receive an empty tenant filter when `--all-tenants` is supplied, regardless of whether the overridden tenant came from base configuration, a profile, or the environment.
- **SC-002**: In acceptance tests, 100% of invocations that combine `--all-tenants` with an explicit named or empty `--tenant` fail as invalid input before any command work occurs.
- **SC-003**: In acceptance tests, 100% of representative create, deploy, and run commands reject `--all-tenants` before submitting work and never target the default tenant as a fallback.
- **SC-004**: All existing tenant-resolution regression scenarios pass unchanged when `--all-tenants` is absent.
- **SC-005**: Every tested all-tenants operation remains limited to resources visible to the authenticated user, with zero authorization-bypass regressions.
- **SC-006**: Root help, representative subcommand help, capability information, and generated documentation agree on the option's availability, conflict rule, unfiltered meaning, and destination restriction in 100% of reviewed cases.
- **SC-007**: In a usability review, at least 90% of operators can identify and invoke the supported all-visible-tenants behavior within 30 seconds without using `--tenant ""`.
- **SC-008**: All tested JSON, YAML, quiet, total-only, and keys-only command outputs retain their established parseability and content contracts.

## Assumptions

- Existing empty tenant values already mean no tenant filter for read, search, discovery, and selection operations; this feature provides an explicit name for that behavior rather than redefining it.
- The existing configuration precedence among base configuration, profiles, and environment remains unchanged until an explicit command-line tenant choice is applied.
- Existing command classification can distinguish operations that consume a tenant as a filter from operations that require one concrete destination.
- Existing invalid-input handling is the authoritative error classification for conflicting flags and unsupported destination scope.
- Existing tenant-context reporting remains the authoritative way to communicate effective selection scope where such reporting already exists.
- The authenticated backend remains authoritative for resource visibility and authorization across tenants.

## Out of Scope

- Removing or deprecating explicit `--tenant ""` behavior.
- Granting access to tenants or resources that the authenticated user cannot already see.
- Changing backend authorization for direct resource keys or activating an ignore-tenant authorization mode.
- Defining a bulk create, deploy, or run operation that repeats work across multiple tenants.
- Changing tenant-source precedence when no explicit command-line tenant choice is supplied.
- Adding new tenant-context output surfaces beyond integrating with those that already apply.

## Implementation Governance

- Planning, task generation, and every Ralph implementation iteration MUST read and apply `specs/ralph-implementation-rules.md`.
- Ralph MUST NOT be launched unless `--implementation-context specs/ralph-implementation-rules.md` is included in the implementation instructions.
- Commit subjects for this issue-backed work MUST use Conventional Commits format and append `#282` as the final token.
