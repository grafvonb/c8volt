# Feature Specification: Extend Release Validation to Camunda 8.10

**Feature Branch**: `338-extend-810-validation`

**Created**: 2026-10-14

**Status**: Draft

**Input**: [GitHub issue #338](https://github.com/grafvonb/c8volt/issues/338) — test(integration): extend release validation to Camunda 8.10

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Trust the requested release validation result (Priority: P1)

As a release maintainer, I want to require validation of a selected Camunda minor and see exactly which checks ran, so a successful result cannot conceal missing profiles or skipped live coverage.

**Why this priority**: A passing result without the requested coverage is misleading release evidence.

**Independent Test**: Evaluate runs with complete, missing, mismatched, skipped, and interrupted coverage and verify that only complete required coverage can pass.

**Acceptance Scenarios**:

1. **Given** release validation requests 8.10 and only 8.9 profiles are selected, **When** validation starts, **Then** it fails the requested-version requirement before mutation and identifies the missing target.
2. **Given** a selected profile claims 8.10 but the connected cluster reports another minor, **When** readiness is checked, **Then** no mutation is submitted to that profile and the mismatch is reported.
3. **Given** an 8.10-only run, **When** any required live family is skipped or has no execution evidence, **Then** the aggregate result is unsuccessful and names the missing family.
4. **Given** a run with known prerequisite gaps, **When** the report is produced, **Then** it separates executed, skipped, blocked, and failed checks and distinguishes live post-state proof, submission-only proof, and dry runs for each version.
5. **Given** a required check fails or a run stops early, **When** available evidence is summarized, **Then** failures and missing results remain visible and cannot be replaced by a successful aggregate result.

---

### User Story 2 - Exercise real operational state on 8.10 (Priority: P1)

As an operator validating a release, I want the existing baseline, volume, and real-state workflows to exercise the selected version using appropriate process models, so I can assess real behavior on 8.10 without losing older coverage.

**Why this priority**: The seven live real-state families currently exclude 8.10, and cascade scenarios contain version-specific identities.

**Independent Test**: Select an 8.10.3 disposable cluster and assess execution evidence for all seven families, then verify that existing 8.9 real-state and 8.8/8.9 baseline and volume scenarios remain selectable.

**Acceptance Scenarios**:

1. **Given** a ready 8.10 profile, **When** real-state validation runs, **Then** jobs, incidents, listeners, BPMN errors, retention, destructive post-state, and parent/child cascades are selected for that version rather than excluded by an 8.9-only gate.
2. **Given** cascade scenarios for 8.9 and 8.10, **When** their fixtures are selected, **Then** each uses process identities appropriate to its version and verifies the resulting parent/child state.
3. **Given** an unavailable scenario prerequisite, **When** that scenario is assessed, **Then** its exact blocked proof and affected version are recorded; preview or submission evidence is not relabeled as completed state verification.
4. **Given** an existing 8.9 real-state profile or an 8.8/8.9 baseline or volume profile, **When** the applicable suite is selected, **Then** its previously supported coverage remains available.
5. **Given** unrelated data exists in the disposable cluster, **When** suite-owned resource assertions run, **Then** unrelated resources do not create false failures or substitute for the required evidence.

---

### User Story 3 - Verify batch mutation outcomes (Priority: P2)

As a release maintainer, I want batch-backed mutations on 8.10 to demonstrate completion, failure accounting, and resulting resource state, so successful submission is not mistaken for completed work.

**Why this priority**: Batch response and lifecycle differences can affect everyday cancellation and deletion workflows.

**Independent Test**: Assess successful and failing batch-backed mutation scenarios, their normal waiting and no-wait output, and controlled lifecycle cases that do not depend on catching a fleeting live state.

**Acceptance Scenarios**:

1. **Given** a batch-backed mutation that completes successfully, **When** normal waiting returns success, **Then** accounting and follow-up resource state support the reported outcome.
2. **Given** a mutation with failed operations, **When** its outcome is reported, **Then** failure counts and affected resources are visible and it is not reported as fully successful.
3. **Given** the same workflow uses `--no-wait`, **When** submission succeeds, **Then** output describes acceptance without claiming observed completion; subsequent validation records post-state proof separately.
4. **Given** a controlled process-definition deletion response with `DRAINING`, or a batch response with an omitted optional operation type, **When** compatibility checks evaluate it, **Then** the documented 8.10 lifecycle and optional-field contract is asserted without requiring a live run to observe that particular response.

---

### User Story 4 - Verify tenant boundaries (Priority: P2)

As an operator of a tenant-aware deployment, I want validation to prove discovery and mutations respect tenant boundaries, so a release cannot claim isolation based only on tenant listing.

**Why this priority**: Cross-tenant effects are operationally significant and require explicit positive and negative evidence.

**Independent Test**: Prepare distinguishable resources in two tenants and validate scoped discovery, scoped mutation, and supported all-tenant operations, including separately identified physical-tenant prerequisites.

**Acceptance Scenarios**:

1. **Given** resources in two tenants, **When** discovery is scoped to one tenant, **Then** expected resources are found and resources exclusive to the other tenant are absent.
2. **Given** a tenant-scoped mutation, **When** post-state is checked, **Then** intended resources changed and control resources in the other tenant retain their prior state.
3. **Given** an authorized supported `--all-tenants` workflow, **When** it runs, **Then** the expected cross-tenant scope and outcomes are verified without treating single-tenant evidence as sufficient.
4. **Given** physical-tenant coverage lacks required infrastructure or permissions, **When** coverage is assessed, **Then** the missing prerequisite and blocked proof are reported; the environment is not labeled as successfully validated for that coverage.

---

### User Story 5 - Use a coherent aggregate validation workflow (Priority: P2)

As a maintainer, I want aggregate checks, version evidence, supporting guards, and instructions to agree, so I can reproduce validation without discovering omitted checks or obsolete baseline assumptions.

**Why this priority**: Reliable release decisions require both live evidence and consistent supporting validation.

**Independent Test**: Review aggregate check selection and supporting assets against the requested version and verify deliberate failures in inventory, examples, or baseline identity are surfaced.

**Acceptance Scenarios**:

1. **Given** the documented aggregate workflow, **When** checks are selected, **Then** inventory, coverage-manifest, profile, read-only smoke, and example validation are included alongside family suites, with any inapplicable check explicitly explained.
2. **Given** 8.10.3 is the pinned baseline, **When** generation provenance and repository-boundary guards are assessed, **Then** they validate the current baseline and permit intended 8.10 coverage while retaining meaningful protections.
3. **Given** a maintainer follows the integration instructions, **When** choosing the aggregate workflow or the separate shell runner, **Then** supported versions, prerequisites, scope, and result limitations are explicit and 8.10 is supported by both entry points.

### Edge Cases

- No selected profile, an unreachable profile, an unsupported minor, or a version that cannot be verified: report the cause; required release coverage cannot pass.
- Mixed 8.9/8.10 profiles: evidence and completeness are evaluated per version; success on one cannot satisfy the other.
- A supported older version lacks a newer capability: retain an explicit unsupported classification without removing existing applicable coverage.
- A required family runs only dry-run checks: do not describe it as fully live-covered or satisfy a required post-state assertion with its exit code.
- A known accepted prerequisite gap occurs: show it separately; it cannot waive an entire required family or count as completed proof.
- Batch responses omit optional metadata or expose transient states: controlled checks remain repeatable and are labeled separately from live evidence.
- Two selected tenant profiles actually resolve to the same tenant: reject them as proof of two-tenant isolation.
- A run is interrupted, evidence is missing, or cleanup fails: preserve available results and cleanup status without claiming missing checks passed or hiding the original failure.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Maintainers MUST be able to request release validation for a specific supported Camunda minor, including 8.10, while retaining existing non-release family selection.
- **FR-002**: Release validation MUST fail when the requested minor has no usable selected profile, when a required live family is absent or skipped, or when required evidence is missing.
- **FR-003**: Every selected mutation target MUST have its connected cluster version verified against its expected supported minor before mutation. A mismatch or unverifiable version MUST block mutations to that target.
- **FR-004**: Real-state coverage MUST support both 8.9 and 8.10 for all seven families: jobs, incidents, listeners, BPMN errors, retention, destructive post-state, and parent/child cascades.
- **FR-005**: Baseline and volume coverage MUST remain reusable for 8.8, 8.9, and 8.10; existing 8.9 real-state coverage MUST remain available. This feature MUST NOT imply new 8.8 real-state support.
- **FR-006**: Fixture selection, including cascade process identities, MUST follow the selected version and reuse suitable existing fixtures. Missing fixtures MUST be reported with the proof they block.
- **FR-007**: Reports MUST identify the requested minor, actual observed server version, selected profile, scenario and family, execution outcome, evidence level, and any prerequisite or failure reason. Results from different versions MUST remain distinguishable.
- **FR-008**: Execution outcomes MUST distinguish executed success, skipped, blocked, failed, and unsupported checks. Evidence levels MUST distinguish live post-state verification, submission-only verification, dry-run verification, and controlled compatibility verification.
- **FR-009**: Required coverage MUST be defined before execution. Accepted prerequisite gaps MUST identify the affected scenario, version, missing prerequisite, and blocked proof; unexpected gaps MUST NOT be silently accepted, and no gap may turn absent proof into success.
- **FR-010**: Existing gaps MUST be reassessed per version, including activated-job timeout updates, catchable BPMN errors and their setup, listener variants, orphan candidates, durable standalone resolution, and repair partial failures. Unsupported or unavailable prerequisites MUST remain explicit.
- **FR-011**: Focused 8.10 batch-backed mutation scenarios MUST verify completion accounting, failure reporting, and observable resource state for normal waiting, and distinguish accepted submission from completion for `--no-wait`.
- **FR-012**: Compatibility validation MUST cover process-definition deletion entering `DRAINING` separately from batch-operation states, and omitted optional batch operation type according to the documented 8.10 contract. Repeatable controlled checks MUST cover responses that cannot reliably be induced or observed live, without claiming those checks are live proof.
- **FR-013**: Tenant scenarios MUST use distinguishable resources in two distinct tenants and verify scoped discovery, scoped mutation, negative isolation assertions, and supported `--all-tenants` behavior.
- **FR-014**: Physical-tenant coverage MUST state its required environment and permissions, expected assertions, and actual evidence. Missing infrastructure MUST be reported as a gap and MUST prevent a success claim for required physical-tenant coverage.
- **FR-015**: The aggregate workflow MUST include common inventory, coverage-manifest, profile, read-only smoke, and example validation alongside applicable family suites. Inapplicability MUST be explicit, and failed required common checks MUST prevent release success.
- **FR-016**: Proposal version lists, real-state coverage artifacts, target descriptions, integration instructions, and the separate shell release runner MUST reflect 8.10 support and distinguish their respective scopes.
- **FR-017**: Baseline provenance and repository-boundary guards MUST agree with the pinned 8.10.3 baseline and intended integration coverage, preserving generation integrity and protections for unaffected supported versions.
- **FR-018**: Validation MUST use explicitly selected disposable targets, preserve existing command output contracts, and tolerate unrelated cluster data. Evidence MUST distinguish fixture/setup limitations, unavailable environments, harness failures, and observed product failures.
- **FR-019**: Interrupted or incomplete runs MUST retain available evidence and remain unsuccessful for required missing coverage. Cleanup results MUST be reported separately from scenario results.
- **FR-020**: Documentation MUST explain how to select the target version, required profiles, known prerequisite gaps, evidence levels, and the difference between partial exploratory runs and release validation.

### Key Entities

- **Validation Request**: Requested minor, selected profiles, release or exploratory intent, required coverage, and known accepted scenario gaps.
- **Validation Target**: A selected disposable environment with expected minor, observed server version, readiness, tenant identity, and relevant permissions.
- **Coverage Family**: A group of related checks with version applicability and required evidence expectations.
- **Scenario Evidence**: A recorded scenario outcome, proof level, relevant resource identities, observed state, and failure or prerequisite explanation.
- **Prerequisite Gap**: A missing capability, fixture, or environment that blocks identified proof for specified versions without representing a passing check.
- **Fixture Set**: Version-appropriate process models and distinguishable resources used to establish operational state and tenant controls.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All seven live real-state families are selected for an 8.10-only release-validation request; every deliberately absent or skipped required family produces an unsuccessful result.
- **SC-002**: Every mismatched or unverifiable target in readiness validation receives zero mutation submissions.
- **SC-003**: Every reported scenario identifies its version, outcome, and evidence level; zero dry-run, submission-only, or controlled checks are described as live post-state proof.
- **SC-004**: Successful batch scenarios have matching completion accounting and observed resource state; all deliberately failed operations remain visible in failure evidence, and no-wait responses make zero unsupported completion claims.
- **SC-005**: Two-tenant scenarios prove both intended changes and unchanged control resources; every unavailable required tenant environment is recorded as a gap rather than successful isolation proof.
- **SC-006**: Existing applicable 8.8/8.9 baseline and volume coverage and 8.9 real-state coverage remain selectable, with no coverage removed solely to enable 8.10.
- **SC-007**: All five common validation categories are accounted for in the aggregate workflow, and all supporting baseline references and guards used for release validation agree with 8.10.3.
- **SC-008**: A maintainer can determine from the run summary whether the requested release coverage is complete and identify every remaining prerequisite gap without inspecting individual command logs.

## Assumptions

- The initial live baseline is Camunda 8.10.3. Selection is by supported minor; evidence records the actual server version rather than asserting that every later patch is certified.
- Maintainers provide existing local profiles, disposable clusters, credentials, and tenant permissions. Provisioning a Camunda platform or acquiring tenant infrastructure is outside this feature.
- Release validation is stricter than exploratory family runs. A documented, predeclared scenario-level prerequisite gap may remain visible within a running family, but cannot waive an entire required family, substitute for required proof, or authorize a claim of complete coverage for the blocked scenario.
- Physical-tenant deployment prerequisites and exact expected observations are established during planning from the supported platform contract. Logical tenant evidence alone does not certify physical isolation.
- This is integration-harness work. New business ID functionality, Camunda application UI testing, and unrelated production changes are excluded. Product defects discovered through validation are reported separately rather than silently expanding implementation scope.
- Live execution is a separate validation activity on explicitly selected disposable environments. Creating this specification does not run live tests.
