# Feature Specification: Complete Smoke-Test Definition Cleanup

**Feature Branch**: `codex/333-fix-smoke-test-cleanup`

**Created**: 2026-10-10

**Status**: Draft

**Input**: User description: "https://github.com/grafvonb/c8volt/issues/333 — fix smoke-test cleanup leaving deployed dependency definitions behind"

**Source note**: Based on the issue title and body drafted in the originating conversation and the observed reproduction. Reading the published issue failed because GitHub authentication returned bad credentials and the web fetch was unavailable; correspondence with the published issue remains unverified.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Clean Up the Complete Deployment (Priority: P1)

As an operator, I want a successful smoke test to remove all eligible definitions it deployed so that running a diagnostic does not leave fixture resources in my tenant.

**Why this priority**: The current command reports passed after removing only the main fixture definition, leaving its two dependencies behind.

**Independent Test**: Run the default smoke test in a tenant with no definitions, then list definitions and verify that none of the three fixture definitions remain.

**Acceptance Scenarios**:

1. **Given** an empty target tenant, **When** the default smoke test deploys `C89_SimpleUserTask`, `C89_SimpleParent`, and `C89_MultipleSubProcessesParent` and completes successfully, **Then** the created process-instance families and all three definitions are removed and cleanup is confirmed.
2. **Given** a supported fixture version with multiple deployed definitions, **When** cleanup succeeds, **Then** every distinct definition returned by that deployment is accounted for and no eligible deployed definition remains.
3. **Given** unrelated definitions or older versions in the same or another tenant, **When** smoke-test cleanup completes, **Then** those unrelated resources remain unchanged.

---

### User Story 2 - Preserve Unsafe Definitions and Explain Partial Cleanup (Priority: P1)

As an operator, I want cleanup to preserve definitions that fail existing safety checks and to report incomplete cleanup accurately so that I can act on retained resources without risking unrelated workloads.

**Why this priority**: Expanding cleanup must preserve resource ownership and make the final outcome trustworthy.

**Independent Test**: Exercise a deployment of three definitions with one blocked by unrelated instances, then separately with a deletion failure; verify resource-level outcomes and the aggregate result.

**Acceptance Scenarios**:

1. **Given** one deployed dependency has unrelated process instances, **When** cleanup evaluates the deployment, **Then** that dependency is retained with the safety reason recorded, and the command does not claim complete cleanup.
2. **Given** some definition deletions succeed and another fails, **When** the workflow finishes, **Then** completed deletions remain reported as successful, the failed or unattempted definitions are accounted for, and the final outcome indicates partial failure using established command conventions.
3. **Given** cleanup retains or fails to remove a definition, **When** human output or a requested audit report is inspected, **Then** the cleanup outcome and retained-resource evidence agree across formats.

---

### User Story 3 - Preserve Existing Execution Controls (Priority: P2)

As an operator or automation author, I want existing smoke-test controls to keep their meaning while covering the complete deployment.

**Why this priority**: The fix must remain usable by existing scripts and diagnostic workflows.

**Independent Test**: Exercise no-cleanup, no-wait, dry-run, and unattended execution with a three-definition deployment and inspect mutations and reported status.

**Acceptance Scenarios**:

1. **Given** `--no-cleanup`, **When** the smoke test completes, **Then** it submits no cleanup mutations and accounts for all retained deployed definitions.
2. **Given** `--no-wait`, **When** eligible cleanup requests are accepted, **Then** all eligible deployed definitions are included in the submission scope and output distinguishes submission from confirmed removal.
3. **Given** `--dry-run`, **When** the preview completes, **Then** it describes cleanup of the complete fixture deployment without submitting any mutation or claiming removal.
4. **Given** supported auto-confirm or automation execution, **When** the workflow runs, **Then** the existing prompt and exit behavior is preserved while cleanup covers all deployed definitions.

### Edge Cases

- A dependency definition is blocked even though the main fixture is eligible for deletion.
- A cleanup safety check or deletion fails after another definition was removed.
- Multiple returned deployment entries refer to the same definition key; cleanup must not duplicate deletion for that identity.
- Deployment evidence is missing a usable definition identity; the workflow must not infer ownership from a name or claim full cleanup without evidence.
- Another tenant or an older version contains a definition with the same BPMN process ID.
- A definition is already absent when cleanup checks it; preserve established deletion semantics and report only supported confirmation facts.
- No-wait cleanup is accepted but its eventual completion is unknown.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The workflow MUST account for every distinct process definition returned by its fixture deployment, including dependencies, while retaining the main fixture identity used to start instances.
- **FR-002**: Default cleanup MUST include all eligible definitions in that deployment scope after the established process-instance cleanup stage succeeds.
- **FR-003**: Cleanup MUST target exact deployed definition identities and preserve tenant boundaries; matching a BPMN process ID alone MUST NOT authorize deleting other definitions or versions.
- **FR-004**: Every definition considered for deletion MUST undergo the existing applicable cleanup safety checks. Definitions blocked by unrelated instances MUST be retained with a reason.
- **FR-005**: The workflow MUST distinguish confirmed removal, submitted but unconfirmed deletion, safety-based retention, failed cleanup, and unattempted cleanup for the deployed definitions.
- **FR-006**: The final outcome MUST reflect the combined cleanup result. Full cleanup MUST NOT be claimed while any deployed definition is retained, failed, unattempted, or unconfirmed. Existing cleanup-skipped and partial-failure conventions MUST remain meaningful.
- **FR-007**: Human summaries, machine-readable output, and requested Markdown or JSON audit reports MUST consistently account for the full deployment and identify retained or failed definitions with available identity and reason evidence. Default human output MUST remain compact; detailed diagnostics MUST follow existing verbosity conventions.
- **FR-008**: `--no-cleanup` MUST preserve all deployed definitions and created instances without submitting cleanup mutations, and report that retention accurately.
- **FR-009**: `--no-wait` MUST preserve existing eligibility and submission semantics across all eligible definitions and MUST NOT describe accepted requests as confirmed removal.
- **FR-010**: Dry-run MUST remain mutation-free and describe the complete cleanup scope. Existing confirmation, automation, quiet, output-stream, and supported output-mode contracts MUST be preserved.
- **FR-011**: The corrected behavior MUST apply to every supported smoke-test fixture version, without changing fixture selection or process-instance creation and family-cleanup semantics.
- **FR-012**: Missing or ambiguous deployment identity evidence MUST NOT authorize broader discovery-based deletion or produce a misleading complete-cleanup claim.
- **FR-013**: User-facing help, examples, and relevant documentation MUST accurately describe cleanup of all deployed fixture definitions and the distinction between retained, submitted, and removed resources.

### Key Entities *(include if feature involves data)*

- **Smoke-Test Deployment Scope**: The distinct definition identities returned by one fixture deployment, including the main runnable fixture and its dependencies, with available tenant and version evidence.
- **Definition Cleanup Outcome**: The eligibility, attempted action, confirmation status, and retention or failure reason for one deployed definition.
- **Aggregate Cleanup Outcome**: The combined result of instance cleanup and all definition cleanup outcomes used in summaries and audit reports.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A successful default smoke test in an initially empty tenant leaves zero fixture definitions and zero created process instances after confirmed cleanup, for every supported fixture version.
- **SC-002**: In safety and failure acceptance scenarios, 100% of deployed definitions are accounted for and zero unrelated definitions or process instances are deleted beyond established family-cleanup scope.
- **SC-003**: Every acceptance scenario with a retained, failed, unattempted, or unconfirmed definition avoids a claim of complete confirmed cleanup in every requested output format.
- **SC-004**: No-cleanup submits zero cleanup mutations; dry-run submits zero mutations; no-wait reports accepted cleanup without claiming confirmed removal.
- **SC-005**: An operator can determine whether cleanup completed and identify any retained or failed definition and its available reason from the command result or requested audit report, without examining low-level HTTP diagnostics.

## Assumptions

- Issue #333 corresponds to the smoke-test cleanup issue drafted in the originating conversation; verification of the published issue remains pending.
- Existing resource deletion, confirmation, safety, tenant, and failure-handling behavior remains authoritative; this feature expands the tracked cleanup scope rather than redesigning those policies.
- Deployment response evidence defines the candidate scope; it does not bypass existing safety checks or authorize deleting resources merely because they share fixture names.
- Repairing leftovers from prior runs, changing fixture contents, adding new flags, and redesigning recovery after failures before the existing cleanup stage are outside scope.
- Requested report compatibility and any necessary representation changes will be assessed during planning; existing fields must not silently acquire contradictory meanings.
