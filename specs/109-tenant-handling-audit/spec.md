# Feature Specification: Harden Tenant Handling Across Tenant-Aware Commands

**Feature Branch**: `109-tenant-handling-audit`  
**Created**: 2026-04-17  
**Status**: Draft  
**Input**: User description: "GitHub issue #109: bug(tenant): audit, simplify, and harden tenant handling across all tenant-aware commands"

## GitHub Issue Traceability

- **Issue Number**: 109
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/109
- **Issue Title**: bug(tenant): audit, simplify, and harden tenant handling across all tenant-aware commands

## Clarifications

### Session 2026-04-17

- Q: How should `v8.9` be handled for this feature scope? → A: Audit `v8.9` too, but only change it if the tenant-handling bug or contract gap is actually present.
- Q: What should tenant mismatch look like on supported tenant-safe operations? → A: Treat tenant mismatch as not found.
- Q: How broad should unsupported behavior be in `v8.7` when only part of a command flow cannot be made tenant-safe? → A: Only the specific `v8.7` operation or flow segment that cannot be tenant-safe should be reported as unsupported.
- Q: How broad should the audit and new regression coverage be across tenant-aware commands? → A: Audit all tenant-aware command families and add new regression coverage for all of them.
- Q: How should derived tenant sources be covered in regression testing? → A: Cover env, profile, and base config separately.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Keep Tenant-Scoped Lookups Safe (Priority: P1)

As a CLI operator, I want every tenant-aware command to respect the tenant context I selected so that commands never return or act on resources from a different tenant.

**Why this priority**: Cross-tenant leakage is a correctness and isolation bug that can expose the existence of resources outside the selected tenant context.

**Independent Test**: Execute direct lookup commands with a selected tenant against keys that belong to the same tenant, a different tenant, and the default tenant, then verify that only in-tenant resources are surfaced.

**Acceptance Scenarios**:

1. **Given** a tenant-aware command runs with a selected tenant and the requested resource belongs to that same tenant, **When** the command performs its lookup, **Then** it returns the resource successfully within that tenant context.
2. **Given** a tenant-aware command runs with a selected tenant and the requested resource belongs to a different tenant, **When** the command performs its lookup, **Then** it returns the same not-found outcome used for resources absent from the selected tenant context.
3. **Given** a tenant-aware command runs with a selected tenant and the requested resource exists only in the default tenant, **When** the selected tenant does not match the default tenant, **Then** the command does not surface the default-tenant resource as a valid match.

---

### User Story 2 - Preserve Tenant Boundaries Through Multi-Step Flows (Priority: P2)

As a CLI operator, I want traversal, wait, cancel, delete, and mixed lookup flows to preserve tenant boundaries through every internal step so that follow-up operations cannot cross tenants unintentionally.

**Why this priority**: A command may look correct at entry but still become unsafe if later searches, traversal steps, or state checks stop applying the same tenant context.

**Independent Test**: Run representative walk, ancestry, descendants, wait, cancel, and delete flows with tenant-selected resources and cross-tenant keys, then verify that every follow-up lookup preserves the same tenant contract end-to-end.

**Acceptance Scenarios**:

1. **Given** a traversal-oriented command starts from a resource within the selected tenant, **When** it resolves parents, children, descendants, or family relationships, **Then** every returned related resource also belongs to the selected tenant context.
2. **Given** a traversal-oriented command starts from a key that belongs to another tenant, **When** the command attempts mixed direct and follow-up lookups, **Then** it does not reveal cross-tenant relationships or descendants.
3. **Given** a tenant-aware wait, cancel, or delete flow uses a key outside the selected tenant context, **When** the flow performs status checks or follow-up operations, **Then** the command fails safely without acting on the cross-tenant resource.
4. **Given** a command family is tenant-aware, **When** the feature audit is completed, **Then** that command family has been reviewed for tenant correctness and has explicit regression coverage for the relevant tenant-handling paths.

---

### User Story 3 - Make Version Behavior Explicit and Predictable (Priority: P3)

As a maintainer, I want tenant-safe behavior defined consistently across supported platform versions so that the CLI either enforces tenant correctness or clearly reports when a version cannot support the requested tenant-safe operation.

**Why this priority**: Version-specific ambiguity makes future maintenance risky and can leave users with behavior that appears safe but is not.

**Independent Test**: Review supported version-specific command paths and regression coverage, then verify that versions with tenant-safe support enforce the same contract and versions without equivalent support report a clear unsupported outcome instead of using a misleading partial fallback.

**Acceptance Scenarios**:

1. **Given** a supported platform version can enforce tenant-safe behavior for a tenant-aware operation, **When** the command runs with a selected tenant, **Then** it applies the same tenant-safety contract as other supported versions.
2. **Given** a platform version cannot support an equivalent tenant-safe operation through the supported upstream behavior, **When** the operator requests that operation with tenant scoping, **Then** the CLI reports that the specific operation or flow segment is not supported for that version and scenario without unnecessarily blocking other tenant-safe behavior in the same command family.
3. **Given** maintainers review tenant-aware command behavior across supported versions, **When** they inspect the specification and regression coverage, **Then** they can identify one explicit contract for supported behavior, not-found behavior, unsupported-version behavior, and the rule that version 8.9 is audited for parity and only changed when a real gap is present.

### Edge Cases

- A selected tenant may be supplied through a flag, environment variable, profile, or base configuration, but the resulting effective tenant context must remain identical across all internal calls made by one command execution.
- Regression coverage must prove tenant correctness separately for environment-derived, profile-derived, and base-config-derived tenant context rather than treating non-flag sources as one interchangeable case.
- A direct key-based lookup must not reveal that a resource exists in another tenant by returning partial details, related resources, or a different failure shape than the tenant-safe not-found contract.
- Multi-step flows that begin with a cross-tenant key must not continue into descendant, ancestry, wait, cancel, delete, or state-check operations as though the initial lookup succeeded.
- The default tenant must follow the same isolation rules as named tenants and must not bypass tenant checks simply because it is the implicit fallback context.
- Commands that combine direct retrieval and follow-up searches must not apply a weaker tenant rule in one step than in another.
- When a supported platform version cannot guarantee tenant-safe behavior for a specific operation, the CLI must fail with an explicit unsupported outcome for that specific operation or flow segment rather than using local post-filtering to appear correct.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST apply the effective tenant context consistently across every internal lookup performed by a tenant-aware command.
- **FR-002**: The system MUST ensure direct key-based tenant-aware lookups do not return resources from a different tenant than the selected tenant context.
- **FR-003**: When tenant-safe lookup is supported for an operation, the system MUST treat tenant mismatch as though the resource is not found within the selected tenant context.
- **FR-003a**: When tenant mismatch occurs on a supported tenant-safe operation, the system MUST return the same user-facing not-found contract used for an actually absent resource in the selected tenant context.
- **FR-004**: The system MUST preserve the same tenant contract across commands that combine direct retrieval with follow-up searches, traversal, wait logic, state checks, cancel flows, and delete flows.
- **FR-005**: The system MUST ensure traversal-oriented commands do not surface parent, child, ancestry, descendant, or family data from outside the selected tenant context.
- **FR-006**: The system MUST avoid correcting tenant behavior through local post-filtering of cross-tenant data that was already retrieved.
- **FR-007**: The system MUST centralize tenant-safe behavior around the supported upstream tenant-scoping contract so that command-specific ad hoc tenant logic is reduced or removed.
- **FR-008**: The system MUST define one clear user-facing outcome for tenant mismatch when tenant-safe behavior is supported, and that outcome MUST be not found so cross-tenant resource existence is not leaked.
- **FR-009**: The system MUST review tenant-aware command paths across direct lookup, search, traversal, wait, cancel, delete, and mixed-flow operations instead of limiting the fix to one command.
- **FR-009a**: The system MUST audit all tenant-aware command families in scope for tenant correctness instead of limiting the review to only the commands explicitly named in the issue description.
- **FR-010**: The system MUST prioritize correct tenant-safe behavior for platform version 8.8 where supported upstream behavior exists.
- **FR-011**: The system MUST only change platform version 8.7 behavior when equivalent tenant-safe behavior is truly supported for the affected operation.
- **FR-012**: When an affected operation cannot be made tenant-safe for platform version 8.7, the system MUST return a clear user-facing unsupported outcome for that specific operation or flow segment and tenant scenario.
- **FR-012b**: The system MUST avoid reporting an entire command family as unsupported in platform version 8.7 when only a narrower operation or flow segment lacks tenant-safe support.
- **FR-012a**: The system MUST audit affected tenant-aware operations on platform version 8.9 for parity with the tenant-handling contract and only change version 8.9 behavior where the same contract gap is actually present.
- **FR-013**: Regression coverage MUST cover matching-tenant, wrong-tenant, non-existing-tenant, and default-tenant scenarios for affected tenant-aware commands.
- **FR-013a**: New or updated regression coverage MUST be added for all audited tenant-aware command families rather than only the paths that end up changing behavior.
- **FR-014**: Regression coverage MUST cover both explicit `--tenant` usage and tenant context derived from other supported configuration sources.
- **FR-014a**: Regression coverage MUST cover environment-derived, profile-derived, and base-config-derived tenant context as separate cases rather than collapsing them into one generic derived-tenant scenario.
- **FR-015**: Regression coverage MUST cover mixed flows that combine direct lookup and search behavior so tenant consistency is verified end-to-end.
- **FR-016**: The feature MUST simplify tenant-related behavior so maintainers can reason about one shared tenant-handling contract instead of multiple divergent command-specific rules.

### Key Entities *(include if feature involves data)*

- **Effective Tenant Context**: The tenant scope a command resolves before performing any tenant-aware operation, regardless of how that tenant was selected.
- **Tenant-Aware Command Flow**: A direct or multi-step command path that reads, traverses, waits on, mutates, or otherwise interacts with tenant-scoped resources.
- **Tenant-Safe Lookup Contract**: The expected behavior for a tenant-aware operation that prevents resources from another tenant from being surfaced or acted on.
- **Cross-Tenant Resource**: A resource that exists outside the selected tenant context and therefore must not be returned or affected by a tenant-aware command.
- **Unsupported Version Scenario**: A version-specific operation where equivalent tenant-safe behavior cannot be guaranteed and must therefore be reported explicitly instead of approximated.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Representative tenant-aware direct lookup commands behave identically for matching-tenant requests and do not return resources for wrong-tenant requests in automated regression coverage.
- **SC-001a**: Automated regression coverage shows that wrong-tenant requests on supported operations return the same not-found outcome as genuinely absent resources in the selected tenant context.
- **SC-002**: Representative mixed lookup and traversal flows demonstrate in automated tests that no parent, child, ancestor, descendant, wait, cancel, or delete step crosses the selected tenant boundary.
- **SC-003**: Automated coverage includes matching tenant, wrong tenant, non-existing tenant, default tenant, explicit `--tenant`, and derived-tenant scenarios for the command paths in scope.
- **SC-003a**: Every tenant-aware command family in scope has explicit regression coverage for its relevant tenant-handling behavior by the end of the feature.
- **SC-003b**: Automated coverage demonstrates tenant-correct behavior separately for environment-derived, profile-derived, and base-config-derived tenant selection across the command families in scope.
- **SC-004**: Platform version 8.8 command paths in scope enforce the documented tenant-safe contract for affected operations.
- **SC-005**: Any platform version 8.7 operation or flow segment that cannot guarantee equivalent tenant-safe behavior produces a clear unsupported outcome in covered scenarios instead of a misleading partial success.
- **SC-005b**: Regression coverage demonstrates that unsupported `v8.7` outcomes are scoped to the unsafe operation or flow segment rather than unnecessarily blocking separate tenant-safe behavior.
- **SC-005a**: Platform version 8.9 command paths in scope are audited against the same tenant-handling contract, and only the paths with a confirmed gap require behavioral change or added regression coverage.
- **SC-006**: Maintainers can identify one explicit tenant-handling contract in the specification and supporting regression coverage without needing to infer different rules for different command flows.

## Assumptions

- Tenant-safe behavior should be implemented only where the supported upstream behavior can enforce it correctly.
- When tenant-safe behavior is available, treating tenant mismatch as not found is the safest default because it avoids leaking cross-tenant resource existence.
- The default tenant is part of the same tenant-handling contract and should not receive looser isolation behavior than named tenants.
- The feature is intended to correct tenant-handling behavior across all relevant tenant-aware commands, not just the originally observed walk flow.
- The audit and regression work should cover every tenant-aware command family in scope, even if some families ultimately need only confirmation and tests rather than behavior changes.
- Version-specific behavior may differ, but the user-facing contract must still make supported and unsupported scenarios explicit.
- Platform version 8.9 should be reviewed for the same tenant-handling risks, but changes there are only required if the audit finds the same bug or contract gap.
- Unsupported `v8.7` behavior should be as narrow as possible and limited to the exact operation or flow segment that cannot be made tenant-safe.
- Derived tenant sources should be treated as distinct regression cases because tenant propagation bugs may differ across environment, profile, and base-config resolution paths.
