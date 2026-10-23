# Feature Specification: Include Suspended Instances in Definition Cleanup

**Feature Branch**: `340-suspended-definition-cleanup`

**Created**: 2026-10-23

**Status**: Ready for planning; no implementation performed

**Input**: [Issue #340 — Include suspended instances in definition deletion and purge](https://github.com/grafvonb/c8volt/issues/340), interpreted with the user's pragmatic scope: preserve existing cancellation behavior, reuse the completed #339/#341 work, and keep separate lifecycle-display work out of this feature. The scope decisions below supersede the broader original issue wording for local specification and planning; GitHub remains unchanged.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Recognize unfinished work before deleting a definition (Priority: P1)

An operator deleting a process definition or purging selected definitions needs suspended instances to count as unfinished work, even when no instance is active. Normal deletion must not proceed on the mistaken assumption that a suspended-only definition is empty.

**Why this priority**: Correct preconditions prevent accidental omission of paused work and misleading cleanup decisions.

**Independent Test**: Use controlled observations of suspended-only, active-only, mixed, terminal-only, and empty definitions. Request normal deletion and purge without force and check the existing preflight boundaries and refusal messages.

**Acceptance Scenarios**:

1. **Given** a definition with zero active and three suspended instances, **When** deletion is requested without force and with normal state checks, **Then** it is refused before mutation and the explanation identifies the suspended unfinished work.
2. **Given** a definition with active and suspended instances, **When** its cleanup eligibility is checked, **Then** both populations contribute to unfinished work; completed and terminated instances do not.
3. **Given** a purge selection containing suspended-only definitions, **When** the existing purge preflight evaluates that selection without force, **Then** suspended work prevents progression at the same preflight boundary used for active work, without submitting mutations after that preflight refuses execution.
4. **Given** terminal-only or empty definitions, **When** normal preconditions are evaluated, **Then** suspension support introduces no new unfinished-work blocker and existing deletion eligibility remains unchanged.
5. **Given** unavailable observations or a failed required state check, **When** cleanup eligibility is evaluated, **Then** the failure remains a failure rather than being converted to zero unfinished instances.

### User Story 2 - Include suspended work in existing force cleanup (Priority: P1)

An operator using force deletion or purge needs all selected unfinished work included in the existing cleanup flow. Root cancellation and the engine's cascading behavior must continue to work as they do today.

**Why this priority**: Updating only the count would correctly block deletion but still leave force cleanup unable to handle the newly recognized work.

**Independent Test**: Exercise forced definition deletion and purge with controlled suspended roots, both mixed parent/child combinations, overlapping definition selections, and changing observations. Verify selection, existing root planning, cleanup order, and dependent-step failure behavior.

**Acceptance Scenarios**:

1. **Given** a suspended-only definition and force, **When** cleanup is planned, **Then** its suspended instances are discovered even though the active count is zero, and cleanup delegates to the existing instance operations without resuming them.
2. **Given** active and suspended instances in the same definition, including across multiple result pages, **When** force cleanup is planned, **Then** both populations are eligible under the existing tenant, selection, and limit rules. Reaching a limit or incomplete discovery must not be reported as complete cleanup.
3. **Given** an active parent with a suspended child or a suspended parent with an active child, **When** those instances enter definition cleanup, **Then** existing ancestry resolution, force escalation, and root cancellation are preserved; no separate cancellation of children is added solely because they are suspended.
4. **Given** overlapping roots across selected definitions in an existing bulk plan, **When** cleanup executes successfully without retries, **Then** the existing deduplicated root scope is preserved, including its existing disclosure of affected instances outside the original definition selection.
5. **Given** cancellation is accepted but suspended instances remain observable, **When** required cleanup verification runs, **Then** it keeps waiting within the existing budget and does not advance to dependent history or definition deletion merely because the active count is zero.
6. **Given** cancellation failure, unsuccessful cleanup results, or expiry of required verification, **When** dependent deletion would otherwise begin, **Then** it does not proceed and the existing error and partial-result behavior is retained.
7. **Given** an instance changes between ACTIVE and SUSPENDED during discovery or verification, **When** unfinished work is evaluated, **Then** changing between those two states must not make it appear finished. The workflow does not claim an atomic snapshot or exclusion of new starts.
8. **Given** additional unfinished work remains after the planned cancellation, **When** required verification runs, **Then** cleanup does not silently expand cancellation to that work or declare success; existing bounded waiting and failure handling apply.

### User Story 3 - See accurate statistics and cleanup impact (Priority: P2)

An operator needs to distinguish active execution from suspended work in definition statistics and cleanup previews, and to retain familiar output and controls when performing cleanup.

**Why this priority**: Accurate reporting makes the corrected eligibility and cleanup behavior understandable without changing established automation contracts.

**Independent Test**: Inspect definition statistics, their existing watch refreshes, dry-run previews, refusal messages, and cleanup results for a known mix of states and for older supported versions. Exercise existing output modes and opt-outs with controlled observations.

**Acceptance Scenarios**:

1. **Given** two active and three suspended instances, **When** definition statistics are requested, **Then** active remains two, suspended is reported separately as three, and cleanup treats five instances as unfinished. Existing machine-readable active fields are not repurposed to mean unfinished.
2. **Given** statistics refresh after an active instance becomes suspended, **When** an existing statistics watch refreshes, **Then** the displayed counts reflect the new observation without retaining stale values or adding definition lifecycle display as part of this feature.
3. **Given** a suspended-only or mixed scope, **When** dry-run is requested, **Then** the preview reflects suspended workload and the same applicable cleanup eligibility, with zero mutation submissions.
4. **Given** existing no-wait or state-check bypass controls, **When** they are used, **Then** their current boundaries remain intact: no-wait does not skip mandatory prerequisites of a later destructive step, and bypass does not introduce new cancellation behavior. Output distinguishes unconfirmed submission from confirmed completion wherever the current workflow does so.
5. **Given** an older supported version without suspension support, **When** ordinary definition statistics or cleanup is requested, **Then** existing behavior and capability restrictions remain intact; suspended statistics use the established unavailable-value convention rather than implying a measured zero or sending unsupported suspension requests.
6. **Given** human, JSON, keys-only, quiet, verbose, or empty-scope execution, **When** affected output is produced, **Then** established formats, stream routing, prompts, and empty-result rules remain intact apart from the necessary additive suspended-work information. A successful empty scope submits no mutations and reports no accepted work.

### Edge Cases

- Zero active instances is not evidence of zero unfinished work on a suspension-capable backend.
- Separate observations may reflect different moments. An ACTIVE-to-SUSPENDED transition must not be interpreted as termination, and failed or incomplete observations must not become a successful empty scope.
- Preserve continuation through sparse pages where the existing search contract indicates more results, plus existing tenant boundaries and selection limits.
- A selected child may resolve to a root belonging to another definition. Preserve existing scope disclosure and force rules; do not change engine-managed cascading.
- A newly started instance can race with cleanup. Keep existing bounded verification and honest failure reporting; this feature does not prevent new starts or promise transaction-wide atomicity.
- Unknown suspended statistics on older versions differ from a measured zero on a supported version.
- Accepted resource deletion may return no batch key. Preserve current nullable-response handling, polling, and final confirmation behavior.
- A suspended child can remain visible after its root appears terminal. Existing instance confirmation and the corrected definition-level unfinished check must not treat that observation as finished.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: For definition deletion and purge on suspension-capable versions, unfinished instances MUST include ACTIVE and SUSPENDED; existing terminal-state meanings MUST remain unchanged.
- **FR-002**: Definition statistics MUST expose active and suspended counts separately through existing public results and relevant human and structured statistics output, including existing statistics watch refreshes. Existing active fields MUST retain their meaning; additions MUST follow established compatibility conventions.
- **FR-003**: Normal no-force preconditions MUST account for suspended instances at the existing single-definition, bulk, and purge preflight boundaries. A refused preflight MUST submit no mutations, and required observation errors MUST NOT be treated as empty work. This does not add an all-or-nothing transaction to existing collection operations.
- **FR-004**: Force discovery MUST include both unfinished states and MUST NOT be skipped solely because the active count is zero. Preserve pagination, sparse-page continuation, tenant scope, limits, and incomplete-discovery reporting. Transitions between the two unfinished states MUST NOT alone justify omission or successful cleanup verification.
- **FR-005**: Definition cleanup MUST reuse existing instance root planning, cancellation, history deletion, force rules, and bulk root deduplication. Preserve current cleanup order, hierarchy-impact disclosure, worker limits, and fail-fast behavior. Do not introduce resumption, additional child cancellation, or a second mutation engine.
- **FR-006**: Required definition-level cleanup verification MUST consider both unfinished states. Remaining suspended work MUST NOT satisfy a zero-unfinished prerequisite, including when no-wait skips a later final confirmation.
- **FR-007**: Failed or unconfirmed mandatory cleanup prerequisites MUST prevent dependent deletion using existing failure and partial-result behavior. Retain existing wait budgets and do not silently cancel additional work encountered during verification.
- **FR-008**: State-check bypass and no-wait MUST preserve their existing read, submission, and confirmation boundaries, except that enabled unfinished-work checks now include suspension. Help and documentation MUST explain their applicable scope without promising confirmed completion when confirmation was skipped.
- **FR-009**: Preflight, dry-run, progress, errors, and results MUST account for suspended workload without relabeling it active. Preserve existing output-mode precedence, logging, prompt routing, machine-readable field meanings, and empty-scope behavior. Dry-run MUST remain mutation-free.
- **FR-010**: Camunda 8.8/8.9 behavior and capability restrictions MUST remain unchanged. Unsupported suspended statistics MUST be represented as unavailable using existing conventions, and ordinary older-version workflows MUST NOT send unsupported suspension requests.
- **FR-011**: Existing resource-deletion acceptance, nullable responses, batch polling where present, bounded final confirmation, and accepted-versus-confirmed reporting MUST be preserved. Extending definition lifecycle representation or redesigning deletion-completion evidence is outside this feature.
- **FR-012**: Affected command help and user documentation MUST describe suspended unfinished work and the preserved control semantics. Broader hardening or unrelated failures discovered during implementation MUST be reported separately rather than expanding this feature automatically.

### Key Entities

- **Process definition**: A selected definition identity with its existing tenant context and observed instance statistics.
- **Unfinished workload**: Active and suspended instance populations, with separate counts and their combined use in cleanup eligibility and verification.
- **Cleanup scope**: The selected definitions and existing planned instance roots and affected family members, including current overlap handling and disclosure.
- **Cleanup result**: Existing preview, submission, confirmation, and failure information, extended only as needed to represent suspended workload accurately.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every suspended-only and mixed no-force refusal scenario submits zero mutations after its refusing preflight; terminal-only and empty controls retain their existing eligibility.
- **SC-002**: Every eligible active or suspended instance in the complete controlled discovery scenarios contributes to cleanup selection. Zero instances are omitted solely because they are suspended or because the active count is zero.
- **SC-003**: Suspended-root and both mixed-tree cleanup scenarios use the existing root flow with zero resume actions or added child-cancellation actions; overlapping roots retain one cancellation submission per root in a successful, non-retried bulk execution.
- **SC-004**: Every persistently suspended or failed-prerequisite scenario produces zero dependent deletion submissions and never reports confirmed completion. Waiting ends within the existing configured budget.
- **SC-005**: All known-population statistics and preview scenarios preserve the exact active count and separately identify suspended work, including refreshed counts. Unsupported counts are never presented as measured zero.
- **SC-006**: All affected dry-run and empty-scope scenarios submit zero mutations, and existing opt-out, output, tenant, and older-version compatibility scenarios retain their documented outcomes.

## Assumptions

- #339 suspended-state support and the narrowed #341 suspension regressions are completed and present in the starting develop baseline. #341 preserved existing cancellation/deletion behavior; it did not deliver the broad new mutation framework assumed by the original #340 wording.
- This specification governs the pragmatic local #340 scope. It does not update the GitHub issue and does not restore discarded #341 requirements.
- General definition lifecycle display, including DRAINING in lookups, listings, statistics, and watch output, remains separate work. This feature includes suspended instance statistics, not new definition lifecycle fields or a new graceful-drain mode.
- Existing root cancellation and engine cascading remain authoritative. No new public mutation workflow, scope ledger, approval mechanism, retry design, or cross-invocation deduplication guarantee is introduced.
- New suspend/resume commands, native batch replacement, generated-client changes, server upgrades, and integration-harness redesign are out of scope.
- Implementation validation uses isolated mocks, test-owned HTTP fakes, and isolated command tests only. No real environment discovery, deployment, mutation, cleanup, or live validation is an implementation completion gate. #342 remains separate follow-up validation; the original issue's Camunda 8.10.4 live target is not an implementation prerequisite.
- This specification authorizes no implementation or test execution in the current step. No real-engine behavior is claimed as newly verified; planning must preserve that distinction.
