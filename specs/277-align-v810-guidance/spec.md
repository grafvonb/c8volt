# Feature Specification: Align Camunda 8.10 Guidance

**Feature Branch**: `277-align-v810-guidance`

**Created**: 2026-08-19

**Status**: Draft

**Input**: GitHub issue #277: "docs: align Camunda 8.10 guidance with #273 and #275"

## GitHub Issue Traceability

- **Issue Number**: 277
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/277
- **Issue Title**: docs: align Camunda 8.10 guidance with #273 and #275
- **Authoritative Predecessors**: Issue #273 defines the V810 compatibility line; issue #275 finalizes its embedded-definition selection with native C810 definitions.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Maintain Version-Aware Features Correctly (Priority: P1)

As a c8volt maintainer, I want repository guidance to describe Camunda 8.10 alongside all other supported compatibility lines so that future version-aware work consistently considers V810 services, generated contracts, capabilities, and tests.

**Why this priority**: Guidance that stops at Camunda 8.9 can cause otherwise correct changes to omit the newest supported runtime and silently create incomplete cross-version behavior.

**Independent Test**: Review every normative maintainer section that enumerates supported Camunda versions or version-specific components and verify that it includes V87, V88, V89, and V810, identifies V810 as the newest supported runtime, and identifies V89 as the default.

**Acceptance Scenarios**:

1. **Given** a maintainer is adding version-aware behavior, **When** they consult the repository guidance, **Then** they are instructed to evaluate and test V810 as well as V87, V88, and V89.
2. **Given** a maintainer is locating version-specific behavior, **When** they consult the architecture map, **Then** it identifies the native V810 service-adapter and generated-contract lines.
3. **Given** V810 is the newest supported runtime, **When** the guidance describes support and defaults, **Then** it distinguishes newest support from the V89 default.

---

### User Story 2 - Interpret Gateway Compatibility Consistently (Priority: P1)

As an operator or reviewer, I want the Camunda 8.10 gateway compatibility requirements to distinguish a release-line match from a diagnostic non-match so that documentation does not promise a hard failure where the established behavior reports a warning.

**Why this priority**: Ambiguous use of the word "reject" can lead operators, testers, and future implementers to infer conflicting command behavior from otherwise consistent requirements.

**Independent Test**: Evaluate the documented outcomes for matching 8.10, patch-level 8.10, prerelease 8.10, different major/minor, empty, and unparseable gateway versions and verify that each has one unambiguous result consistent across normative guidance.

**Acceptance Scenarios**:

1. **Given** V810 is configured and the gateway reports `8.10`, an `8.10.x` patch, or an 8.10 prerelease, **When** compatibility is evaluated, **Then** the documentation classifies the gateway as belonging to the configured 8.10 release line.
2. **Given** V810 is configured and the gateway reports a different major/minor release, **When** compatibility is evaluated, **Then** the documentation classifies it as a non-match and requires the established mismatch diagnostic without introducing a new mandatory command failure.
3. **Given** V810 is configured and the gateway version is empty or unparseable, **When** compatibility is evaluated, **Then** the documentation states that compatibility cannot be verified and requires the established unrecognizable-version diagnostic without introducing a new mandatory command failure.

---

### User Story 3 - Distinguish Active C810 Guidance from History (Priority: P1)

As a c8volt maintainer, I want active Camunda 8.10 guidance to select native C810 embedded definitions while preserving clearly labeled historical records of the former C89 fallback so that current work follows one production-fixture decision without losing implementation history.

**Why this priority**: Issue #275 deliberately replaced only the original fixture decision from issue #273. Mixing active and historical statements could restore an invalid fallback or lead maintainers to rewrite useful delivery records.

**Independent Test**: Inspect every active issue #273 design artifact and every retained historical C89 record, verifying that active guidance permits only C810 selection while historical records remain preserved and visibly superseded by issue #275.

**Acceptance Scenarios**:

1. **Given** a maintainer reviews active V810 fixture guidance, **When** they identify the selected embedded family, **Then** only native C810 definitions are presented as valid.
2. **Given** a historical completed record describes the former V810-to-C89 mapping, **When** it is reviewed, **Then** it remains recognizable as historical and is not presented as an active alternative.
3. **Given** an operator lists, exports, deploys, or smoke-selects embedded definitions for V810, **When** documentation describes the expected family, **Then** it identifies C810 and never authorizes C89 fallback.

---

### User Story 4 - See One Coherent Operator Contract (Priority: P2)

As a c8volt operator, I want all operator-facing guidance to present one coherent Camunda 8.10 support contract so that I can configure the correct identity, understand its active source baseline, and know that V89 is the current default.

**Why this priority**: The primary behavior is already delivered, but consistent operator guidance reduces configuration mistakes and prevents prerelease provenance from being confused with a selectable compatibility identity.

**Independent Test**: Review the operator-facing version and configuration guidance and verify that it consistently presents the canonical identity, accepted aliases, V89 default, active prerelease baseline, in-place baseline replacement model, and applicable embedded-definition family.

**Acceptance Scenarios**:

1. **Given** an operator wants to configure Camunda 8.10, **When** they consult the documentation, **Then** they see canonical `8.10` and the aliases `810`, `v810`, and `v8.10`.
2. **Given** the active V810 generated contracts originate from a prerelease, **When** an operator reviews version information, **Then** the prerelease source is disclosed separately from the configured `8.10` identity.
3. **Given** no Camunda version is configured, **When** an operator reviews the documented default, **Then** V89 is the stated default.
4. **Given** a later 8.10 baseline is adopted, **When** an operator reviews the compatibility model, **Then** it remains one in-place V810 identity rather than creating a new selectable prerelease or patch identity.

### Edge Cases

- An active design artifact mentions C89 only as the behavioral source from which C810 definitions were derived; that statement must not be interpreted as permitting C89 runtime selection.
- A completed task or progress record accurately describes the former C89 fallback; it remains historical rather than being rewritten as if C810 had been the original implementation.
- A gateway reports `8.10.0-alpha4`; it is an 8.10 release-line match even though that full source tag is not an accepted configuration identity.
- A gateway reports `8.10.7`; it is an 8.10 release-line match even though patch-specific configuration inputs remain unsupported.
- A gateway reports another minor line, an empty value, or malformed text; the result is a diagnostic non-match, not a newly introduced hard failure.
- A future final 8.10 baseline replaces the prerelease source; the operator identity, aliases, service-family identity, and default remain unchanged.
- Guidance for stable Camunda 8.7, 8.8, and 8.9 must remain accurate while V810 is added to enumerations.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Normative maintainer guidance MUST identify V87, V88, V89, and V810 as the supported version-specific runtime lines.
- **FR-002**: Normative maintainer guidance MUST identify V810 as the newest supported runtime while stating that V89 is the default when no version is configured.
- **FR-003**: Version-aware development guidance MUST require maintainers to evaluate factory selection, owned behavior, capability availability, and automated coverage for V810 whenever a change applies to supported runtime lines.
- **FR-004**: Architecture guidance MUST identify a native V810 service-adapter line and a native V810 generated-contract line alongside their V87, V88, and V89 counterparts.
- **FR-005**: Guidance MUST preserve the boundary between version-neutral contracts and version-specific runtime behavior when adding V810 to the architecture description.
- **FR-006**: The canonical operator-facing Camunda 8.10 compatibility identity MUST remain `8.10`.
- **FR-007**: Operator guidance MUST identify `8.10`, `810`, `v810`, and `v8.10` as the complete accepted alias set for V810.
- **FR-008**: Operator guidance MUST treat alpha, release-candidate, final-release, and patch source identifiers as baseline provenance rather than additional selectable compatibility identities.
- **FR-009**: Operator guidance MUST disclose the active V810 source baseline and its prerelease status separately from the configured `8.10` identity.
- **FR-010**: Guidance MUST state that later 8.10 source baselines replace the active V810 baseline in place without changing the operator identity, aliases, or service-family identity.
- **FR-011**: Gateway compatibility requirements MUST classify observed `8.10`, `8.10.x`, and 8.10 prerelease values as matches for configured V810.
- **FR-012**: Gateway compatibility requirements MUST classify a different major/minor release as a non-match that produces the established mismatch diagnostic.
- **FR-013**: Gateway compatibility requirements MUST classify an empty or unparseable observed release as unverifiable and require the established unrecognizable-version diagnostic.
- **FR-014**: Gateway non-match requirements MUST NOT imply a new mandatory command failure where the established contract provides diagnostics.
- **FR-015**: All active Camunda 8.10 fixture guidance MUST select native C810 embedded definitions for listing, export, deployment, and smoke workflows.
- **FR-016**: Active guidance MUST NOT permit V810 to select C89 definitions as a compatibility fallback.
- **FR-017**: Statements that describe C89 as the behavioral source for C810 definitions MUST distinguish derivation from runtime selection.
- **FR-018**: Historical completed tasks, progress records, and implementation memory that describe the former C89 fallback MUST remain preserved as historical records.
- **FR-019**: Retained historical C89 records MUST remain clearly superseded by the native C810 decision from issue #275.
- **FR-020**: Guidance MUST continue to state that live Camunda 8.10 integration infrastructure is outside the delivered support scope.
- **FR-021**: The refinement MUST change only the omitted-version fallback from V88 to V89; it MUST NOT otherwise change runtime behavior, compatibility identities, embedded process definitions, or stable-version behavior.
- **FR-022**: The final guidance set MUST contain no contradictory active statement about V810 identity, gateway release-line results, default selection, source-baseline status, or embedded-definition selection.

### Key Entities

- **V810 Compatibility Line**: The single operator-facing Camunda 8.10 identity, its accepted aliases, version-specific behavior, and relationship to the V89 default.
- **Active V810 Baseline**: The one upstream Camunda 8.10 source state from which current compatibility artifacts originate; it may be replaced in place without creating another compatibility identity.
- **Gateway Release-Line Result**: One of match, diagnostic mismatch, or unverifiable diagnostic, determined from the configured and observed major/minor release lines.
- **C810 Embedded Definition Family**: The native Camunda 8.10 process definitions selected by embedded listing, export, deployment, and smoke workflows.
- **Active Guidance**: Normative documentation that directs current maintainer or operator behavior and must reflect the final #273/#275 support model.
- **Historical Implementation Record**: A completed task, progress entry, or implementation-memory statement that records an earlier delivered state without defining current behavior.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of normative maintainer sections that enumerate supported runtime lines include V87, V88, V89, and V810, with zero statements naming V89 as the newest supported runtime.
- **SC-002**: 100% of normative architecture references to version-specific services and generated contracts include the V810 line.
- **SC-003**: All six gateway cases—plain 8.10, 8.10 patch, 8.10 prerelease, different major/minor, empty, and unparseable—have one unambiguous documented outcome matching the established diagnostic contract.
- **SC-004**: 100% of active fixture-selection statements map V810 to C810, with zero active statements permitting a C89 fallback.
- **SC-005**: 100% of retained historical C89 implementation records remain preserved and clearly distinguishable from active guidance.
- **SC-006**: 100% of reviewed operator-facing version summaries agree on the canonical identity, four accepted aliases, V89 default, active baseline status, and in-place replacement model.
- **SC-007**: A maintainer using only normative repository guidance can correctly identify all four supported runtime lines, the newest supported line, the default line, and the V810 fixture family on the first review.
- **SC-008**: Repository review finds only the intended V88-to-V89 fallback change, with zero other runtime-behavior, compatibility-identity, embedded-definition, stable-version, or live-integration scope changes.
- **SC-009**: All documentation consistency checks applicable to the affected guidance complete with zero contradictions or failures.

## Assumptions

- Issue #273 remains authoritative for the V810 compatibility identity, aliases, baseline lifecycle, native runtime behavior, and live-integration boundary; this refinement intentionally supersedes its former V88 default with V89.
- Issue #275 remains authoritative for native C810 embedded-definition selection and supersedes only the former V810-to-C89 fixture mapping.
- Existing runtime behavior remains authoritative except for the explicitly requested V88-to-V89 fallback promotion, which requires updated factory regression coverage.
- Historical delivery records are valuable audit evidence and should be labeled through existing supersession context rather than rewritten.
- Primary operator documentation already contains much of the correct V810 contract and should change only where a consistency review finds a concrete contradiction or omission.

### Dependencies

- The final specifications and contracts from issues #273 and #275 remain available as the authoritative source for the refined guidance.
- Existing documentation consistency and generation paths remain available to validate affected user-facing material.
- Existing gateway diagnostic and embedded-selection behavior remains unchanged during the refinement.

### Out of Scope

- Runtime, service, command, configuration, or generated-client behavior changes other than the V88-to-V89 omitted-version fallback.
- New Camunda compatibility identities or configuration aliases.
- Updating the active Camunda 8.10 source baseline.
- Restoring V810-to-C89 embedded-definition fallback.
- Modifying C87, C88, C89, or C810 process definitions.
- Adding live Camunda 8.10 integration environments, profiles, fixtures, scripts, targets, or real-state scenarios.
- Rewriting completed task descriptions, progress logs, or implementation-memory records to conceal their historical state.
