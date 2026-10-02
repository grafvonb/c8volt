# Feature Specification: Simplify Supported Camunda Versions

**Feature Branch**: `codex/331-simplify-version-support`

**Created**: 2026-10-02

**Status**: Draft

**Input**: User description: https://github.com/grafvonb/c8volt/issues/331

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Operate with the retained supported versions (Priority: P1)

As an operator or automation author, I want c8volt to support Camunda 8.8, 8.9, and 8.10 with its existing behavior, while an unsupported version receives the ordinary unsupported-version failure. I do not need compatibility with the retired release line identified by the source issue.

**Why this priority**: Removing a release line must leave daily operations and automation reliable for everyone using a retained version.

**Independent Test**: Exercise representative read, search, deployment, mutation, and confirmation workflows on retained versions; verify default selection and generic unsupported-version handling independently of documentation cleanup.

**Acceptance Scenarios**:

1. **Given** a configuration selecting 8.8, 8.9, or 8.10, **When** an operator executes an operation already supported for that version, **Then** the operation preserves its result, tenant scope, confirmation behavior, output format, and exit behavior.
2. **Given** no explicit version selection, **When** c8volt resolves configuration, **Then** it selects 8.9.
3. **Given** an unsupported version selection, **When** a command or library consumer attempts to initialize an operation, **Then** ordinary unsupported-version handling rejects it before remote requests, mutation confirmation, or mutation submission, without silently selecting a supported version.
4. **Given** an 8.8 process-instance deletion or an 8.8/8.9 legacy task identifier lookup, **When** the operator executes that workflow, **Then** its existing component-service behavior remains available.
5. **Given** an operation restricted on a retained version, **When** it is requested on that version, **Then** its existing restriction remains enforced.

---

### User Story 2 - Maintain only necessary release support (Priority: P2)

As a maintainer, I want the retired release's implementation, compatibility behavior, fixtures, and dedicated tests removed so future work only maintains behavior that c8volt still offers.

**Why this priority**: Leaving dormant compatibility material would preserve the maintenance burden and could restore retired behavior through generation or later changes.

**Independent Test**: Review the tracked implementation, tests, fixtures, and generation inventory; run applicable generation checks and retained-version regression coverage to demonstrate that removed material is absent and cannot return through normal generation.

**Acceptance Scenarios**:

1. **Given** the completed cleanup, **When** a maintainer inspects the supported-version inventory, **Then** no identity, alias, exported compatibility symbol, adapter, generated client, fixture mapping, or runtime selection remains for the retired release.
2. **Given** behavior used solely by the retired release, **When** its callers are removed, **Then** the exclusive behavior and helpers are also removed, including special tenant defaults, capability rejection paths, and the separate dry-run traversal behavior.
3. **Given** shared test coverage with retired and retained release cases, **When** that coverage is simplified, **Then** retired cases and exclusive helpers disappear while meaningful retained-version coverage remains.
4. **Given** normal client or mock generation, **When** maintainers run it after cleanup, **Then** it cannot recreate retired-release artifacts or require their former paths.
5. **Given** an integration profile or fixture dedicated to the retired release, **When** the tracked integration inventory is reviewed, **Then** it is absent and retained-version cases still express their actual capabilities.

---

### User Story 3 - Read a consistent support policy (Priority: P2)

As an operator or contributor, I want help, examples, documentation, and project guidance to describe only the support c8volt currently provides, without obsolete compatibility instructions.

**Why this priority**: Outdated guidance can lead operators to choose an unsupported release and contributors to reintroduce removed behavior.

**Independent Test**: Review support statements and search tracked documentation, examples, configuration, and planning material for obsolete support content; compare generated help with the supported-version inventory.

**Acceptance Scenarios**:

1. **Given** command help, version output, examples, or user documentation, **When** a user checks supported versions, **Then** the material consistently identifies 8.8, 8.9, and 8.10 and the default remains 8.9.
2. **Given** tracked repository guidance and historical planning material, **When** a contributor consults it, **Then** obsolete requirements, limitations, and examples for the retired release have been removed without discarding unrelated project context.
3. **Given** the documentation cleanup, **When** the tracked documentation is reviewed, **Then** it contains no dedicated migration, deprecation, or compatibility guidance for the retired release.

### Edge Cases

- Unsupported selection through a flag, environment variable, configuration file, profile, or library consumer must follow generic validation; existing configuration precedence remains intact.
- An omitted discovery tenant must retain supported-version semantics, while operations creating tenant-owned data still choose the appropriate concrete tenant.
- A legacy component dependency required by a retained version must survive even if similar dependencies existed for the retired release.
- Missing deployment identifiers or other incomplete responses must not justify retaining an exclusively retired-release workaround; behavior shared by supported versions must remain covered.
- Shared tests may use an obsolete fixture even when the tested behavior is still needed; replace the fixture or case without losing that behavior's coverage.
- Numeric text in unrelated dependency versions, identifiers, or data may resemble the retired release. Such matches are not support references and must not be blindly rewritten.
- The repository audit includes checked-in planning material and the artifacts for this feature. These artifacts describe the retained support policy and refer to the source issue for the retired release's identity so they do not become an exception to the removal requirement.
- Git history, the external source issue, and ignored local configuration are outside the tracked-content cleanup boundary.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: c8volt MUST support exactly Camunda 8.8, 8.9, and 8.10 after this change, with 8.9 remaining the default and existing maturity labels for retained versions preserved.
- **FR-002**: c8volt MUST remove all selectable identities, aliases, exported compatibility symbols, and special compatibility handling belonging to the retired release identified in the source issue. No backward compatibility or deprecation period is required.
- **FR-003**: Unsupported version selections MUST use generic unsupported-version handling before remote requests or mutation prompts, without fallback to another release. This feature MUST NOT introduce a dedicated retired-release rejection suite.
- **FR-004**: The tracked implementation MUST contain no adapters, generated clients, runtime branches, tenant defaults, capability guards, dry-run behavior, or exclusive helpers maintained solely for the retired release.
- **FR-005**: Tracked tests and validation assets MUST contain no dedicated retired-release suites, cases, profiles, fixtures, mocks, or exclusive test helpers. Shared regression coverage MUST preserve behavior still supported.
- **FR-006**: Generation and repository validation workflows MUST neither depend on retired-release targets or paths nor recreate their artifacts.
- **FR-007**: Retained versions MUST preserve existing tenant isolation, authentication, pagination, mutation acceptance and confirmation, error classification, exit behavior, and human and machine output contracts, including empty results and interactive stream routing.
- **FR-008**: Component-service functionality still needed for 8.8 process-instance deletion and 8.8/8.9 task lookup fallback MUST remain available. Capability restrictions still applicable to retained versions MUST remain enforced.
- **FR-009**: Tracked documentation, examples, configuration guidance, command metadata, repository guidance, and planning artifacts MUST contain no obsolete support references, limitations, or requirements for the retired release. Existing project context unrelated to that release MUST be preserved.
- **FR-010**: Command help, version reporting, generated documentation, and user-facing support statements MUST agree on the retained versions and default. No dedicated retired-release migration or deprecation documentation may be added.
- **FR-011**: Completion MUST include an audit of tracked paths and content for obsolete support material, with incidental numeric matches and unrelated dependency versions distinguished from actual support references.
- **FR-012**: Delivery MUST demonstrate retained behavior through relevant targeted validation, the full project test suite, a successful build, and applicable generation boundary checks. Any unavailable validation MUST be reported rather than claimed as successful.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All supported-version discovery surfaces identify the same three retained versions, and every tested omitted-version configuration selects 8.9.
- **SC-002**: Zero retired-release support artifacts or support references remain in tracked paths and content after the final audit, excluding only incidental numeric matches and unrelated dependency versions.
- **SC-003**: All retained regression scenarios selected for affected workflows pass, including 8.8 deletion and 8.8/8.9 legacy task lookup; no previously supported behavior is lost because of the cleanup.
- **SC-004**: Normal generation produces zero retired-release artifacts and all applicable generation boundary checks pass.
- **SC-005**: Generic unsupported-version validation scenarios perform zero remote requests and present zero mutation confirmation prompts.
- **SC-006**: Maintainers can complete the required project validation with no unresolved failures attributable to this change, and users encounter no contradictory support claims in help or documentation.

## Assumptions

- The source issue identifies the single release line being retired; no other release's support or unrelated unused generated clients are removed by this feature.
- The requested compatibility break is intentional for both operators and library consumers. A subsequent plan will record its compatibility impact while user-facing documentation describes the resulting support policy.
- Retained versions continue to have distinct capabilities. This feature does not unify their implementations or expand their supported operations.
- Existing regression coverage is the baseline for supported behavior; cleanup must preserve meaningful coverage rather than maximize the number of deleted tests.
- References needed solely to describe retirement are expressed through the retained-version policy and source issue, including within this feature's own planning artifacts.
- This specification authorizes later implementation planning. Production removal, regeneration, and runtime validation belong to the implementation phase.
