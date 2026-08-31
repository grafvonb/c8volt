# Feature Specification: Stable Tenant-Aware Process-Definition Ordering

**Feature Branch**: `286-tenant-pd-ordering`

**Created**: 2026-08-31

**Status**: Draft

**GitHub Issue**: [#286](https://github.com/grafvonb/c8volt/issues/286) - feat(cli): add stable tenant-aware process-definition ordering

**Input**: GitHub issue #286 requests one stable, tenant-aware ordering contract for process-definition collections returned by `c8volt get process-definition` and its aliases.

## Issue Traceability

- **GitHub Issue**: #286
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/286
- **Issue Title**: feat(cli): add stable tenant-aware process-definition ordering
- **Related Issue**: [#282](https://github.com/grafvonb/c8volt/issues/282), which made unfiltered cross-tenant discovery explicit through `--all-tenants`

## Clarifications

### Session 2026-08-31

- Q: When tenant ID, BPMN process ID, and version are equal, how should process-definition keys be compared to break the tie? → A: Exact text ascending.
- Q: When the default tenant appears with named tenants, where should its process-definition group be placed? → A: Use exact tenant-ID ascending order.
- Q: Should tenant IDs and BPMN process IDs that differ only by letter case remain distinct groups and use case-sensitive ordering? → A: Case-sensitive identity and ordering.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Scan Definitions by Tenant and Process (Priority: P1)

As a Camunda operator, I want process-definition listings grouped consistently by tenant and BPMN process so that I can scan cross-tenant results without definitions from unrelated tenants or processes being interleaved.

**Why this priority**: This is the feature's primary value. Cross-tenant listings are difficult to understand unless related definitions stay together and their versions appear in a predictable order.

**Independent Test**: Populate at least two tenants with at least two BPMN process IDs and multiple versions, run a process-definition listing, and verify the rows are ordered by exact tenant ID text ascending, exact BPMN process ID text ascending, version descending, and exact process-definition key text ascending for otherwise equal rows.

**Acceptance Scenarios**:

1. **Given** visible definitions belong to multiple tenants and BPMN process IDs, **When** an operator lists all visible process definitions, **Then** rows are grouped first by tenant ID in ascending order and then by BPMN process ID in ascending order within each tenant.
2. **Given** one tenant contains several versions of the same BPMN process, **When** the definitions are listed, **Then** the newest version appears first and older versions follow in descending numeric version order.
3. **Given** two returned definitions have the same tenant ID, BPMN process ID, and version, **When** they are listed, **Then** their process-definition keys establish a deterministic exact-text ascending order without numeric interpretation.
4. **Given** only one tenant or one BPMN process is represented, **When** definitions are listed, **Then** the same canonical ordering rules apply without requiring `--all-tenants`.

---

### User Story 2 - Keep Ordering Consistent Across Views (Priority: P2)

As an operator or automation author, I want the same process-definition collection to retain the same row order in every supported output view so that adding statistics or changing presentation does not change the meaning or stability of the result.

**Why this priority**: Stable ordering must be a property of the collection, not a visual coincidence in one renderer. This protects scripts and prevents watch rows from moving when only volatile statistics change.

**Independent Test**: Query the same controlled definition population with and without statistics and through human, JSON, and keys-only output; compare the ordered process-definition keys and verify every supported view produces the same sequence.

**Acceptance Scenarios**:

1. **Given** statistics are available for a process-definition collection, **When** the operator repeats the same listing with and without `--stat`, **Then** both results contain process definitions in the same order.
2. **Given** the same collection is rendered as human, JSON, or keys-only output, **When** its process-definition keys are read in output order, **Then** all three sequences are identical.
3. **Given** watch mode refreshes an unchanged collection while statistics change, **When** a new snapshot is displayed, **Then** the process-definition rows remain in the same order.
4. **Given** a watch snapshot's definition membership or canonical sort fields change, **When** the next snapshot is displayed, **Then** rows move only as required by the canonical ordering contract.

---

### User Story 3 - Preserve Complete and Compatible Discovery (Priority: P3)

As an operator, I want stable ordering to remain correct for paged and latest-only searches across every supported cluster version so that I receive a complete, predictable collection without regressions to direct retrieval.

**Why this priority**: The ordering is useful only if it applies to the complete discovered collection and behaves consistently across supported environments while existing single-resource workflows remain intact.

**Independent Test**: Retrieve the same multi-tenant, multi-process, multi-version population with several discovery page sizes and with `--latest`; repeat on supported cluster versions, then verify complete canonical ordering, latest grouping, and unchanged single-key and XML retrieval.

**Acceptance Scenarios**:

1. **Given** matching definitions span multiple discovery pages, **When** the collection is returned, **Then** every matching definition appears exactly once in canonical order regardless of page boundaries or selected page size.
2. **Given** multiple tenants contain the same BPMN process ID, **When** the operator requests `--latest`, **Then** the newest matching version is selected independently for each tenant and BPMN process ID group.
3. **Given** equivalent definition populations on Camunda 8.7, 8.8, 8.9, and 8.10, **When** they are listed, **Then** they follow the same canonical ordering contract.
4. **Given** an operator retrieves a definition by key or requests its XML, **When** the request completes, **Then** the existing single-resource content, validation, and output behavior are unchanged.

### Edge Cases

- The result set is empty or contains one definition; output remains valid and no artificial grouping information is added.
- The default tenant and named tenants appear together; the displayed `<default>` identifier participates in exact ascending tenant-ID comparison with no special first-or-last placement rule.
- Tenant IDs or BPMN process IDs differ only by character case; they remain distinct groups, and their exact identifiers determine a stable, case-sensitive ascending order.
- Two definitions share tenant ID, BPMN process ID, and version; exact ascending process-definition key text resolves the tie deterministically without interpreting keys as numbers.
- Version values such as 9 and 10 are ordered numerically, so version 10 appears before version 9.
- Definitions arrive in different orders from separate discovery pages or repeated requests; the final collection order remains canonical and independent of arrival order.
- Statistics are unavailable on a supported cluster version; the existing unsupported-statistics behavior remains unchanged, while ordinary listings still use canonical ordering.
- A watch refresh changes only active-instance or incident counts; row order remains unchanged because statistics are not ordering fields.
- Filters narrow the result to a tenant, BPMN process, version, or version tag; every resulting collection still follows the same ordering contract.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Every process-definition collection returned by `get process-definition` and its established aliases MUST use one canonical ordering before presentation.
- **FR-002**: The canonical ordering MUST compare exact tenant ID text in ascending order as its first field; the displayed `<default>` identifier MUST use the same comparison without special placement.
- **FR-003**: Tenant IDs and BPMN process IDs MUST retain exact, case-sensitive identity; within one exact tenant ID, the canonical ordering MUST compare exact BPMN process ID text in case-sensitive ascending order as its second field.
- **FR-004**: Within one tenant and BPMN process ID group, the canonical ordering MUST compare process-definition version numerically in descending order as its third field.
- **FR-005**: Definitions equal on tenant ID, BPMN process ID, and version MUST compare process-definition keys as exact text in ascending order as the deterministic final tie-breaker; keys MUST NOT be interpreted numerically for ordering.
- **FR-006**: Canonical ordering MUST apply to both tenant-filtered and unfiltered cross-tenant discovery, including invocations using `--all-tenants`.
- **FR-007**: Human, JSON, and keys-only output MUST consume the same canonically ordered collection and expose the same process-definition sequence.
- **FR-008**: Adding `--stat` MUST enrich definitions without changing their canonical order.
- **FR-009**: Volatile statistics, including active process-instance or incident counts, MUST NOT participate in default ordering.
- **FR-010**: Watch snapshots MUST use canonical ordering, and changing statistics alone MUST NOT move rows between snapshots.
- **FR-011**: `--latest` MUST identify groups by the combination of exact, case-sensitive tenant ID and exact, case-sensitive BPMN process ID, selecting the newest matching version independently within each group.
- **FR-012**: Results spanning multiple discovery pages MUST be returned exactly once per matching definition and in canonical order, independent of page size, page boundaries, or arrival order.
- **FR-013**: The canonical ordering contract MUST be consistent across Camunda 8.7, 8.8, 8.9, and 8.10 for the process-definition data available on each version.
- **FR-014**: Single-key process-definition retrieval and XML retrieval MUST retain their existing content, validation, authorization, output, and error behavior.
- **FR-015**: Existing process-definition filters and selectors MUST retain their meaning; this feature changes only collection ordering and tenant-aware latest grouping.
- **FR-016**: Default human output MUST remain compact and MUST NOT add endpoint, request, cursor, page-lifecycle, or per-key ordering diagnostics.
- **FR-017**: Command help, source metadata, README examples, and generated command documentation MUST describe the canonical collection order where the listing contract is documented.
- **FR-018**: Automated acceptance coverage MUST include multiple tenants, multiple BPMN process IDs, multiple versions, deterministic key ties, pagination, statistics parity, output-mode parity, latest grouping, watch stability, supported cluster versions, and unchanged single-key and XML retrieval.

### Key Entities *(include if feature involves data)*

- **Process Definition**: A deployed BPMN model version identified by tenant ID, BPMN process ID, numeric version, and process-definition key, with optional statistics used only for enrichment.
- **Canonical Process-Definition Collection**: A complete set of matching process definitions ordered by exact case-sensitive tenant ID text ascending, exact case-sensitive BPMN process ID text ascending, version descending, and exact key text ascending.
- **Tenant and Process Group**: Definitions sharing both a tenant ID and BPMN process ID; version ordering and latest selection operate within this boundary.
- **Process-Definition Statistics**: Volatile information associated with a definition, such as active-instance or incident counts; it does not define collection order.
- **Watch Snapshot**: One complete observed process-definition collection whose rows use the canonical order for that refresh.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In acceptance datasets containing at least 2 tenants, 3 BPMN process IDs, and 3 versions of one process, 100% of returned rows follow exact case-sensitive tenant text ascending, exact case-sensitive BPMN process text ascending, version descending, and exact key text ascending order.
- **SC-002**: For every tested collection, ordered process-definition key sequences match exactly across human, JSON, and keys-only output.
- **SC-003**: For every statistics-capable test environment, ordered process-definition key sequences match exactly with and without `--stat`.
- **SC-004**: Across page sizes of 1, 2, and a size larger than the result set, 100% of tested searches return the same ordered keys with zero skipped or duplicated definitions.
- **SC-005**: In every tested `--latest` result, exactly one newest matching definition is selected per represented tenant and BPMN process ID group.
- **SC-006**: Across Camunda 8.7, 8.8, 8.9, and 8.10 compatibility tests, equivalent input populations produce the same canonical process-definition sequence.
- **SC-007**: In watch tests where only statistics change across at least 10 consecutive refreshes, zero process-definition rows move position.
- **SC-008**: All existing single-key and XML retrieval regression scenarios pass without changes to returned content or user-visible behavior.
- **SC-009**: In an operator review using a mixed-tenant result set, at least 90% of participants correctly identify tenant groups, process groups, and newest versions on their first scan.
- **SC-010**: User-facing help and generated documentation describe the ordering contract consistently in 100% of reviewed process-definition listing references.

## Assumptions

- Tenant IDs and BPMN process IDs use exact, case-sensitive identity and ascending text comparison; values that differ only by letter case remain distinct groups. Process-definition keys are opaque identifiers compared as exact text in ascending order, without numeric interpretation.
- The default tenant is represented as `<default>` in process-definition results and participates in exact tenant-ID ascending order without a special first-or-last grouping rule.
- Process-definition versions are numeric and are compared numerically rather than as text.
- Statistics parity applies where statistics are supported; Camunda 8.7 retains its existing behavior of rejecting or not providing native statistics.
- Existing authenticated visibility remains authoritative; stable ordering does not broaden which tenants or definitions an operator may see.
- Each watch refresh represents a complete current collection, so rows may move when definitions are added, removed, or their canonical sort fields genuinely differ.

## Out of Scope

- Adding an operator-selectable sort option or changing the canonical direction through flags.
- Introducing statistics-based ordering such as `--sort active`; that may be considered as a separate feature.
- Changing which statistics are collected or displayed.
- Changing tenant selection, tenant authorization, or the meaning of `--all-tenants`.
- Changing the behavior of resource collections other than process definitions.
- Changing single-key process-definition or XML retrieval.
- Adding low-level paging or ordering diagnostics to default human output.

## Implementation Governance

- Planning, task generation, and every Ralph implementation iteration MUST read and apply `specs/ralph-implementation-rules.md`.
- Ralph MUST NOT be launched unless `--implementation-context specs/ralph-implementation-rules.md` is included in the implementation instructions.
- Commit subjects for this issue-backed work MUST use Conventional Commits format and append `#286` as the final token.
