# Feature Specification: Effective Tenant Context Before Mutations

**Feature Branch**: `283-show-tenant-context`

**Created**: 2026-08-29

**Status**: Draft

**GitHub Issue**: [#283](https://github.com/grafvonb/c8volt/issues/283) - feat(safety): show effective tenant context before mutations

**Input**: GitHub issue #283 requires operation-specific tenant context in configuration diagnostics, previews, confirmations, operational workflows, and audit reports so operators can distinguish tenant-scoped discovery, unfiltered discovery, default-tenant creation, and backend-authorized explicit-key operations before execution.

## Issue Traceability

- **GitHub Issue**: #283
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/283
- **Issue Title**: feat(safety): show effective tenant context before mutations

## Clarifications

### Session 2026-08-29

- Q: How should the command report a resolved mutation plan when at least one target's tenant is unknown? → A: Warn that some target tenants are unknown without blocking execution, and also show any warning required by the known tenant values.
- Q: How should JSON and YAML results expose tenant context when a command already returns structured output? → A: Use one common nested tenant-context object wherever tenant context applies.

### Session 2026-08-30

- Q: Which canonical grammar should tenant-context human messages use? → A: Use lower-case c8volt-style labels: `selection scope`, `creation target`, and `affected tenants`; keep `WARNING:` uppercase.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See Search And Selection Scope (Priority: P1)

As an operator, I want previews and confirmations to state whether discovery is limited to a named tenant or has no tenant filter so that I can recognize when resources from multiple accessible tenants may be selected.

**Why this priority**: An unfiltered search can produce a cross-tenant candidate set, making this the most immediate safeguard against an unexpectedly broad mutation.

**Independent Test**: Run the same search-derived mutation preview with a named tenant and with an empty tenant, then verify the first identifies the named tenant filter and the second explicitly warns that no tenant filter is applied and multiple tenants may be affected.

**Acceptance Scenarios**:

1. **Given** the effective configured tenant is `tenant-a`, **When** an operator reviews a search or selection preflight, **Then** the output states `selection scope: tenant-a only` before execution.
2. **Given** the effective configured tenant is empty, **When** an operator reviews a search or selection preflight, **Then** the output states `selection scope: unfiltered across accessible tenants` rather than describing the scope as the default tenant.
3. **Given** an unfiltered search resolves resources from one tenant only, **When** the operator reviews the mutation plan, **Then** the output still states that no tenant filter was applied because the selection semantics remain unfiltered.

---

### User Story 2 - See Creation Target Tenant (Priority: P2)

As an operator, I want create, deploy, and run commands to state the tenant where new work will be created so that I can verify the target before execution.

**Why this priority**: An empty configured tenant has a different meaning for creation than for search; displaying the target prevents operators from mistaking default-tenant creation for cross-tenant discovery.

**Independent Test**: Review create, deploy, and run execution information with a named tenant and with an empty tenant, then verify each command identifies either the named target or `<default>` before it changes operational state.

**Acceptance Scenarios**:

1. **Given** the effective configured tenant is empty, **When** an operator reviews a create, deploy, or run command before execution, **Then** the output states `creation target: default tenant`.
2. **Given** the effective configured tenant is `tenant-a`, **When** an operator reviews a create, deploy, or run command before execution, **Then** the output states `creation target: tenant-a`.
3. **Given** a non-interactive create, deploy, or run command is invoked, **When** execution information is produced in its selected output mode, **Then** the tenant target is represented without introducing an interactive prompt.

---

### User Story 3 - Understand Explicit-Key Tenant Behavior (Priority: P3)

As an administrator supplying explicit resource keys, I want the command to state that the configured tenant filter is not enforced and to show resolved resource tenants when known so that I understand the operation is governed by backend authorization.

**Why this priority**: Explicit keys intentionally follow a different tenant contract from discovery; hiding that distinction can make a direct administrative mutation appear safer or narrower than it is.

**Independent Test**: Preview an explicit-key operation with a configured tenant that differs from the resolved resource tenant, then verify the preview states that the tenant filter is not applied and identifies the resource's actual tenant when available.

**Acceptance Scenarios**:

1. **Given** an operator supplies one or more explicit resource keys, **When** the command presents information before resolution, **Then** it states `selection scope: explicit resource keys; tenant filter not applied`.
2. **Given** an explicit resource resolves to `tenant-b`, **When** the resource tenant is available in the resolved plan, **Then** the output states `affected tenants: tenant-b`.
3. **Given** an explicit resource's tenant is not available in the resolved plan, **When** the operator reviews the plan, **Then** the command does not invent, infer, or mislabel a resource tenant.
4. **Given** the configured tenant differs from the resolved resource tenant, **When** the operator reviews the plan, **Then** the command reports the actual resource tenant without rejecting the target solely for that difference.

---

### User Story 4 - Receive Cross-Tenant Mutation Warnings (Priority: P4)

As an operator, I want a prominent warning when a resolved mutation plan spans tenants so that I can stop and reassess a broad operation before confirming it.

**Why this priority**: Once a plan is resolved, the actual tenant distribution is stronger safety evidence than configuration alone and must be visible before destructive work proceeds.

**Independent Test**: Resolve a mutation plan containing resources from `tenant-a` and `tenant-b`, then verify the preview and destructive confirmation both warn that multiple tenants will be affected and identify the distinct tenants.

**Acceptance Scenarios**:

1. **Given** a resolved mutation plan contains resources from `tenant-a` and `tenant-b`, **When** the plan is shown before execution, **Then** it states `affected tenants: tenant-a, tenant-b` and prominently states `WARNING: resources from multiple tenants will be affected: tenant-a, tenant-b`.
2. **Given** multiple resolved resources all belong to `tenant-a`, **When** the plan is shown, **Then** it states `affected tenants: tenant-a` and no cross-tenant warning is emitted.
3. **Given** resolved resources contain repeated tenant values, **When** the warning is produced, **Then** each distinct known tenant appears once in a stable order.
4. **Given** some resolved resources have unknown tenant metadata, **When** the plan is shown, **Then** a non-blocking warning states that some target tenants are unknown without inventing a tenant value.
5. **Given** some resolved resources have unknown tenant metadata and known resources span multiple tenants, **When** the plan is shown, **Then** both the unknown-tenant warning and the known cross-tenant warning are emitted.

---

### User Story 5 - Trust Tenant Context Across Safety Surfaces (Priority: P5)

As an operator or automation author, I want configuration diagnostics, mutation previews, confirmations, operations workflows, and audit reports to describe tenant behavior consistently so that no safety surface contradicts what the command will do.

**Why this priority**: Consistent reporting turns the individual labels and warnings into a reliable command-wide safety contract and prevents audit records from misrepresenting unfiltered work.

**Independent Test**: Exercise representative process-instance, process-definition, job, operations, deploy, and run workflows in named, empty, explicit-key, and cross-tenant cases, then verify each applicable surface reports the same operation-specific tenant meaning while preserving its output contract.

**Acceptance Scenarios**:

1. **Given** a tenant-sensitive command is reviewed through configuration validation, connection diagnostics, dry-run, preflight, or destructive confirmation, **When** tenant context applies to that surface, **Then** the operation-specific tenant meaning is visible and consistent.
2. **Given** process-instance cancel, delete, resolve, or update; process-definition deletion; job update; or a retention, purge, repair, or smoke-test workflow, **When** it selects or mutates resources, **Then** applicable previews and confirmations expose the effective tenant context before mutation.
3. **Given** an audit report records an operation with no tenant filter, **When** the report is reviewed, **Then** it records the operation as unfiltered rather than as operating on the default tenant.
4. **Given** JSON or YAML output is selected, **When** tenant context applies to the command result, **Then** one common nested tenant-context object conveys the applicable tenant meaning without changing existing envelope semantics.
5. **Given** quiet or keys-only output is selected, **When** the command runs, **Then** quiet behavior remains quiet and keys-only output contains one key per line with no tenant labels or warnings mixed into standard output.

### Edge Cases

- An empty configured tenant means no tenant filter for search and selection, but it means the default tenant for create, deploy, and run; the same empty value must not receive one generic label across both operation types.
- A named configured tenant may differ from the actual tenant of an explicit resource; both meanings must remain distinct and the explicit resource must not be locally rejected solely for the mismatch.
- A resolved mutation plan may contain named tenants, the default tenant, missing tenant metadata, or a combination of these; warnings must use the established user-facing tenant representation and must not invent tenant values.
- Tenant metadata may become available only after resource resolution; pre-resolution messaging must not claim to know the actual resource tenant.
- Unknown tenant metadata in a resolved plan must produce a non-blocking warning, even when all known targets belong to one tenant or no target tenant is known.
- A plan may include duplicate tenant values; tenant summaries and warnings must list distinct tenants deterministically.
- An unfiltered search that happens to return resources from only one tenant remains an unfiltered search and must not be reported as tenant-scoped.
- A cross-tenant plan must remain prominent when confirmation output is compact, and must not be hidden by per-resource detail.
- Machine-readable, quiet, and keys-only modes must not receive human-oriented warning text that would invalidate their existing contracts.
- Commands that do not search, select, create, deploy, run, resolve explicit resources, or mutate tenant-associated resources must not display irrelevant tenant context.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Search and selection preflight information MUST display `selection scope: <tenant> only` when a named effective tenant limits discovery.
- **FR-002**: Search and selection preflight information MUST display `selection scope: unfiltered across accessible tenants` when the effective tenant is empty.
- **FR-003**: Search and selection reporting MUST NOT represent an empty tenant as `<default>`.
- **FR-004**: Create, deploy, and run commands MUST display `creation target: default tenant` before execution when the effective tenant is empty.
- **FR-005**: Create, deploy, and run commands MUST display `creation target: <tenant>` before execution when a named effective tenant is configured.
- **FR-006**: Explicit-key operations MUST state `selection scope: explicit resource keys; tenant filter not applied` before the configured tenant could be mistaken for an enforced filter.
- **FR-007**: Explicit-key operations MUST display `affected tenants: <tenant>` when the actual tenant is already available in the resolved plan.
- **FR-008**: Explicit-key operations MUST NOT invent or infer an actual resource tenant when that tenant is unavailable in the resolved plan.
- **FR-009**: Reporting tenant context MUST NOT change the backend-authorized behavior of explicit resource keys or impose a new local tenant restriction.
- **FR-010**: A resolved mutation plan containing resources from more than one distinct known tenant MUST produce a prominent warning that identifies every distinct known tenant affected.
- **FR-011**: Cross-tenant warning tenant values MUST be unique and presented in a deterministic order.
- **FR-012**: Tenant context MUST be visible where applicable in configuration validation, connection diagnostics, dry-run output, preflight output, and destructive mutation confirmations.
- **FR-013**: Tenant context MUST be applied where relevant to process-instance cancel, delete, resolve, and update commands; process-definition deletion; job updates; retention, purge, repair, and smoke-test workflows; and deploy and run commands.
- **FR-014**: Audit reports MUST distinguish an unfiltered all-accessible-tenant operation from an operation targeting the default tenant.
- **FR-015**: The same command execution MUST describe its tenant behavior consistently across diagnostics, preflight, confirmation, final reporting, and audit evidence where those surfaces exist.
- **FR-016**: JSON and YAML outputs MUST remain valid structured data, preserve existing envelope semantics, and use one common nested tenant-context object wherever tenant context applies.
- **FR-017**: Quiet output MUST preserve its existing suppression behavior.
- **FR-018**: Keys-only output MUST continue to emit exactly one resource key per line and no additional tenant labels, warnings, or explanatory text on standard output.
- **FR-019**: The feature MUST NOT change tenant-selection semantics or introduce an all-tenants option.
- **FR-020**: Automated acceptance coverage MUST include named tenant filters, empty search filters, default creation targets, explicit-key behavior, known resource tenants, and cross-tenant mutation plans.
- **FR-021**: User-facing help, examples, and generated documentation MUST explain operation-specific tenant meanings wherever affected command behavior is documented.
- **FR-022**: Configuration validation and connection diagnostics MUST distinguish a named configured tenant from no configured tenant and MUST NOT describe the absence of a configured tenant as a default-tenant operation.
- **FR-023**: A resolved mutation plan containing any target with unknown tenant metadata MUST produce a non-blocking unknown-tenant warning, and this warning MUST appear in addition to any cross-tenant warning required by the known targets.
- **FR-024**: Human-oriented tenant context MUST follow c8volt's operational output grammar: lower-case sentence fragments for ordinary labels and uppercase `WARNING:` only for prominent safety warnings.

### Key Entities *(include if feature involves data)*

- **Effective Tenant Context**: The tenant value resolved for one command execution, interpreted according to whether the operation searches, selects, creates, deploys, runs, or acts on explicit resources.
- **Tenant Filter**: A named tenant constraint applied to search or selection; when absent, discovery is unfiltered across resources accessible to the operator.
- **Creation Tenant Target**: The named or default tenant in which a create, deploy, or run operation will create new work.
- **Explicit Resource Target**: A resource key supplied directly by the operator and governed by backend authorization rather than the configured tenant filter.
- **Resolved Mutation Plan**: The final set of resources a mutation intends to affect, including actual resource tenant metadata when known.
- **Tenant Context Evidence**: The operation-specific labels, warnings, audit facts, and common nested tenant-context object that explain how tenant context affected or did not affect the operation.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In acceptance tests, 100% of representative search and selection previews correctly distinguish a named tenant filter from no tenant filter.
- **SC-002**: In acceptance tests, 100% of representative create, deploy, and run previews correctly distinguish a named creation tenant from the default tenant.
- **SC-003**: In acceptance tests, every explicit-key preview states that the configured tenant filter is not applied, and every already-known resource tenant is shown accurately.
- **SC-004**: In acceptance tests, every resolved plan spanning two or more known tenants produces a prominent warning listing each distinct known tenant exactly once.
- **SC-005**: Representative process-instance, process-definition, job, operations, deploy, and run workflows expose applicable tenant context before execution with no contradictory tenant descriptions across their safety surfaces.
- **SC-006**: All tested JSON and YAML results remain parseable and use the same nested tenant-context object, all tested quiet results preserve their suppression contract, and all tested keys-only results contain exactly one key per line and no tenant commentary.
- **SC-007**: Audit acceptance tests record 100% of unfiltered operations as unfiltered and never as default-tenant operations.
- **SC-008**: In operator review of the named, empty, default-target, explicit-key, and cross-tenant examples, at least 90% of reviewers correctly identify within 10 seconds which tenant or tenants may be affected before execution.
- **SC-009**: Existing tenant-selection behavior remains unchanged across all regression scenarios; only its operator-visible explanation is added or corrected.
- **SC-010**: In acceptance tests, 100% of resolved mutation plans containing unknown tenant metadata produce a non-blocking unknown-tenant warning, including plans whose known targets also span multiple tenants.

## Assumptions

- The existing effective tenant resolution from flags, environment, profiles, or base configuration remains authoritative and is not changed by this feature.
- Existing tenant-selection behavior distinguishes tenant-scoped discovery, unfiltered discovery, default-tenant creation, and backend-authorized explicit resource keys; this feature exposes those established meanings rather than redefining them.
- The established user-facing representation for the default tenant is reused when actual resource tenant metadata identifies the default tenant.
- Cross-tenant warnings are based on tenant metadata already available in the resolved plan; unavailable tenant data may remain unknown.
- Non-interactive commands remain non-interactive. Tenant context is conveyed through their selected reporting mode without adding confirmation prompts.
- Existing compatibility guarantees for structured, quiet, and keys-only output remain in force.

## Out of Scope

- Changing how tenants are selected, filtered, authorized, or targeted.
- Adding an `--all-tenants` flag or another new tenant-selection control.
- Rejecting backend-authorized explicit resource keys because they differ from the configured tenant.
- Fetching otherwise unavailable tenant metadata solely to enrich tenant-context reporting.

## Implementation Governance

- Planning, task generation, and every Ralph implementation iteration MUST read and apply `specs/ralph-implementation-rules.md`.
- Ralph MUST NOT be launched unless `--implementation-context specs/ralph-implementation-rules.md` is included in the implementation instructions.
- Commit subjects for this issue-backed work MUST use Conventional Commits format and append `#283` as the final token.
