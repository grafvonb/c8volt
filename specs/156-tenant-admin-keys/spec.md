# Feature Specification: Tenant Scope For Discovery And Explicit Admin Keys

**Feature Branch**: `156-tenant-admin-keys`

**Created**: 2026-05-24

**Status**: Draft

**Input**: GitHub issue [#156](https://github.com/grafvonb/c8volt/issues/156), "refactoring(tenant): clarify tenant scope for discovery vs explicit admin keys"

## Issue Traceability

- **GitHub Issue**: #156
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/156
- **Issue Title**: refactoring(tenant): clarify tenant scope for discovery vs explicit admin keys

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Preserve Tenant-Scoped Discovery Boundaries (Priority: P1)

As a Camunda operator, I want discovery and search flows to keep using the selected tenant where supported so that c8volt-produced candidate sets do not accidentally broaden into unrelated cross-tenant resources.

**Why this priority**: Discovery-derived candidate sets are the highest-risk boundary because c8volt, not the operator, produced the target population that later read or mutation flows may consume.

**Independent Test**: Configure a Camunda 8.8 or 8.9 test path with tenant-scoped search results, run process-instance discovery or a search-derived destructive preview with `--tenant tenant-a`, and verify only the tenant-scoped candidate set is selected and carried forward.

**Acceptance Scenarios**:

1. **Given** `--tenant tenant-a` and process instances across tenants, **When** an operator runs a search or list command where tenant filtering is supported, **Then** c8volt discovers only resources visible through the tenant-scoped search result.
2. **Given** `--tenant tenant-a` and a search-derived `cancel pi` or `delete pi` flow, **When** c8volt freezes the candidate set, **Then** the operation uses the tenant-scoped discovered candidates and their intended dependency scope only.
3. **Given** dependency expansion is required for a search-derived destructive flow, **When** expansion completes, **Then** c8volt does not add unrelated cross-tenant resources that were not part of the intended discovered scope.

---

### User Story 2 - Treat Explicit Process-Instance Keys As Admin Input (Priority: P2)

As a Camunda administrator, I want direct process-instance keys to be governed by Camunda backend authorization, even when `--tenant` names a different tenant, so that c8volt remains an admin tool instead of adding its own stricter tenant block.

**Why this priority**: Direct key commands are common operational escape hatches; blocking them locally changes the admin contract and can prevent legitimate backend-authorized work.

**Independent Test**: Use a Camunda 8.8 or 8.9 test path where `--tenant tenant-a` is selected and the backend returns or accepts an explicit tenant-b process-instance key, then verify c8volt does not reject the command solely because returned tenant metadata differs.

**Acceptance Scenarios**:

1. **Given** `--tenant tenant-a`, **When** an operator runs `get pi --key <tenant-b-key>` and Camunda returns the resource, **Then** c8volt displays the returned resource instead of rejecting it solely for tenant mismatch.
2. **Given** `--tenant tenant-a`, **When** an operator runs `walk pi --key <tenant-b-key>` or `expect pi --key <tenant-b-key>`, **Then** c8volt treats the key as explicit admin input and relies on Camunda backend authorization.
3. **Given** `--tenant tenant-a`, **When** an operator runs `cancel pi --key <tenant-b-key>` or `delete pi --key <tenant-b-key>`, **Then** existing preflight, dry-run, confirmation, dependency expansion, and safety behavior still apply without adding a stricter local tenant authorization boundary.

---

### User Story 3 - Align Explicit Definition, Resource, And Stdin Inputs (Priority: P3)

As a Camunda administrator, I want process-definition keys, resource IDs, stdin keys, and explicit flag values to follow the same backend-authorized admin-input contract so that tenant behavior is consistent across resource families.

**Why this priority**: Mixed semantics across command families create surprising failures and can make documentation inaccurate even after process-instance behavior is corrected.

**Independent Test**: Run explicit process-definition, resource, and stdin-key command paths with a selected tenant that differs from returned metadata and verify c8volt either proceeds when Camunda authorizes the request or reports the backend error without a separate tenant mismatch rejection.

**Acceptance Scenarios**:

1. **Given** `--tenant tenant-a`, **When** an operator runs `get pd --key`, `get pd --xml`, or `delete pd --key` for a backend-authorized resource associated with another tenant, **Then** c8volt does not reject solely because selected tenant and returned metadata differ.
2. **Given** `--tenant tenant-a`, **When** an operator runs `get resource --id` or process-definition deletion uses resource deletion for explicit inputs, **Then** the direct resource ID is treated as explicit admin input.
3. **Given** explicit stdin keys, **When** a bulk command consumes those keys, **Then** c8volt treats them as operator-supplied admin targets rather than search-derived tenant-scoped candidates.

---

### User Story 4 - Document Tenant Semantics For Operators (Priority: P4)

As an operator or automation author, I want help text and user-facing documentation to explain that `--tenant` scopes discovery/search and creation flows while explicit keys remain backend-authorized admin operations so that I can predict command behavior before running broad or destructive workflows.

**Why this priority**: The behavior change is only usable if operators can understand the distinction without reverse-engineering tests or source code.

**Independent Test**: Inspect relevant help and generated documentation for tenant-sensitive commands and verify the discovery-vs-explicit-key contract is stated consistently.

**Acceptance Scenarios**:

1. **Given** an operator reads help for tenant-sensitive search, discovery, create, deploy, or run flows, **When** `--tenant` behavior is described, **Then** the text explains that tenant scoping applies to those flows where supported.
2. **Given** an operator reads help or documentation for direct key, direct ID, or stdin-key flows, **When** tenant behavior is described, **Then** the text explains that explicit inputs are backend-authorized admin operations.
3. **Given** documentation is regenerated from command metadata, **When** generated docs are reviewed, **Then** they match the executable help text.

### Edge Cases

- Camunda 8.8 and 8.9 may expose different tenant filtering support; each supported version must preserve its current backend-compatible discovery behavior.
- Camunda 8.7 tenant behavior is out of scope and must not be rewritten as part of this feature.
- Returned resource metadata may identify a different tenant than the selected `--tenant`; explicit input flows may display that returned tenant for operator visibility.
- Search-derived destructive flows may still perform dependency expansion, but expansion must remain tied to the intended discovered scope.
- Direct keys supplied by flags and direct keys supplied through stdin must be treated consistently as explicit operator input.
- Existing `<default>` tenant behavior for create, deploy, and start flows must be preserved.
- Backend authorization failures must remain backend failures, not be converted into c8volt-side tenant mismatch errors.
- Automation and JSON output paths must remain deterministic and must not introduce interactive prompts.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Tenant-scoped discovery and search flows MUST continue to include the selected tenant where Camunda 8.8 or 8.9 supports tenant filtering.
- **FR-002**: Candidate sets produced by tenant-scoped discovery or search MUST NOT cause c8volt to broaden subsequent read, preview, cancellation, deletion, or dependency-expansion scope into unrelated cross-tenant resources.
- **FR-003**: Direct user-supplied process-instance keys MUST be treated as explicit admin input governed by Camunda backend authorization.
- **FR-004**: Direct stdin process-instance keys MUST be treated as explicit admin input governed by Camunda backend authorization.
- **FR-005**: Direct process-definition keys and direct resource IDs MUST be treated as explicit admin input governed by Camunda backend authorization.
- **FR-006**: c8volt MUST NOT reject explicit keys, stdin keys, IDs, or flag values solely because the selected `--tenant` differs from returned resource metadata.
- **FR-007**: Destructive direct-key commands MAY keep existing preflight, dry-run, confirmation, dependency expansion, force, and verification behavior, but those checks MUST NOT impose a stricter tenant authorization boundary than Camunda for explicit input.
- **FR-008**: Search-derived `cancel pi` and `delete pi` flows MUST continue to act on the tenant-scoped discovered candidate set and its intended dependency scope.
- **FR-009**: `get pi --key`, `walk pi --key`, `expect pi --key`, `cancel pi --key`, and `delete pi --key` MUST follow the explicit admin-input contract for Camunda 8.8 and 8.9.
- **FR-010**: `get pd --key`, `get pd --xml`, `delete pd --key`, `get resource --id`, and resource deletion used by process-definition deletion MUST follow the explicit admin-input contract for Camunda 8.8 and 8.9.
- **FR-011**: Create, deploy, and run flows MUST preserve existing `<default>` tenant behavior.
- **FR-012**: User-facing help and documentation MUST explain that `--tenant` scopes discovery, search, selection, create, deploy, and run flows where supported, while explicit keys, IDs, stdin keys, and direct flag values remain backend-authorized admin input.
- **FR-013**: Tests MUST cover at least one tenant-scoped discovery/search-derived flow for Camunda 8.8 or 8.9.
- **FR-014**: Tests MUST cover at least one explicit direct-key flow for Camunda 8.8 or 8.9 where selected tenant and returned metadata differ.
- **FR-015**: The implementation MUST NOT rewrite Camunda 8.7 tenant behavior as part of this issue.

### Key Entities *(include if feature involves data)*

- **Selected Tenant**: The operator-provided tenant context from `--tenant`, used to scope discovery, search, selection, create, deploy, and run flows where supported.
- **Discovery-Derived Candidate Set**: Targets produced by c8volt from tenant-scoped search or discovery and carried into previews, confirmations, dependency expansion, and mutations.
- **Explicit Admin Input**: Direct keys, IDs, stdin keys, and flag values supplied by the operator and governed by Camunda backend authorization.
- **Returned Tenant Metadata**: Tenant information returned by Camunda for an observed resource, displayed or logged for operator visibility without becoming a local authorization block for explicit input.
- **Dependency Scope**: Additional process-instance or resource scope included by existing safety behavior for destructive commands.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Tenant-scoped discovery/search tests for Camunda 8.8 or 8.9 show that 100% of c8volt-produced candidates come from the tenant-scoped result set for the tested flow.
- **SC-002**: Search-derived `cancel pi` or `delete pi` tests show that the frozen candidate set and intended dependency scope do not include unrelated cross-tenant resources.
- **SC-003**: Explicit direct-key tests for process instances show that c8volt performs no tenant mismatch rejection when Camunda returns or accepts the explicit target.
- **SC-004**: Explicit process-definition or resource tests show the same backend-authorized admin-input contract for direct keys or IDs.
- **SC-005**: Existing create, deploy, and run tests that cover `<default>` tenant behavior continue to pass without behavior changes.
- **SC-006**: Help text and regenerated documentation contain the discovery-vs-explicit-input tenant contract for relevant command families.
- **SC-007**: The closest relevant automated tests for changed command, facade, service, and documentation surfaces pass.

## Assumptions

- Operators using direct keys or stdin keys are intentionally performing administrative operations and expect Camunda backend authorization to be authoritative.
- Tenant-scoped discovery remains a c8volt responsibility because c8volt produces the candidate set before the operator confirms or automation consumes it.
- Existing command safety behavior remains valuable and should be preserved unless it currently enforces a local tenant mismatch block for explicit input.
- Version-specific differences between Camunda 8.8 and 8.9 should remain explicit and tested near the owning service or command paths.
- User-facing documentation is generated from command metadata where applicable, so command help and docs must be updated through the repository's generation path.

## Implementation Governance

- Planning, task generation, and every Ralph implementation iteration MUST read and apply `specs/ralph-implementation-rules.md`.
- Ralph MUST NOT be launched unless `--implementation-context specs/ralph-implementation-rules.md` is included in the implementation instructions.
- Commit subjects for this issue-backed work MUST use Conventional Commits format and append `#156` as the final token.
