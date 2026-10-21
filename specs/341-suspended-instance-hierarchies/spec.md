# Feature Specification: Verify Suspended Instance Cancellation and Deletion

**Feature Branch**: `341-suspended-instance-hierarchies`

**Created / Revised**: 2026-10-21

**Status**: Ready for task planning; no implementation performed

**Input**: The user’s 2026-10-21 scope decision: preserve existing cancellation behavior and verify suspension-specific cases. This specification and its plan supersede the broader original issue description for feature #341.

## User Scenarios & Testing

### User Story 1 — Cancel suspended instances through the existing workflow (Priority: P1)

An operator cancels a suspended instance or a mixed active/suspended tree using the same root cancellation and force rules already used by c8volt.

**Why this priority**: Suspension adds another unfinished state; operators need confidence that the working cancellation flow handles it without a replacement workflow.

**Independent Test**: Cancel a suspended root and both mixed parent/child combinations using controlled observations. Verify the existing root request and completion behavior without resuming any instance.

**Acceptance scenarios**:

1. Given a suspended root, cancellation uses the existing root operation without resumption. In normal waiting mode, success requires the existing terminal-or-absent observations for the discovered family.
2. Given an active root with a suspended child, or a suspended root with an active child, existing root cancellation handles the tree. No additional child cancellation or listener bypass is introduced.
3. Given a child selection, existing force requirements and ancestry escalation remain unchanged.
4. Given a CLI selection containing a root and its child, the existing planner produces one root and dispatches one cancellation for that root in a successful, non-retried execution.
5. Given accepted cancellation whose observed family remains suspended, normal waiting ends within the existing budget without claiming completion. Existing no-wait behavior may report submission without confirmation.

### User Story 2 — Preserve existing deletion safeguards for suspended instances (Priority: P1)

An operator deletes suspended-instance history through the current force-deletion recovery workflow, without changing deletion order or the meaning of existing controls.

**Why this priority**: Suspended instances remain unfinished; tests should prove the existing safeguards apply to them.

**Independent Test**: Attempt deletion with and without force, then simulate a deletion conflict followed by cancellation and terminal observations. Verify the existing request sequence and failure handling.

**Acceptance scenarios**:

1. Given suspended scope without force, CLI planning refuses deletion before mutation; preserve the existing direct-service conflict behavior as well.
2. Given suspended scope with force, preserve descendant-first deletion and cancellation recovery when deletion encounters unfinished execution. Do not require canceling the entire tree before the first deletion attempt.
3. Given conflict recovery, the existing terminal-state prerequisite must succeed before deletion is retried, including with no-wait selected.
4. Given cancellation failure or observations remaining suspended until the wait expires, no dependent deletion retry is submitted and no completed deletion is reported.
5. Given normal waiting or no-wait, preserve the existing final deletion-confirmation behavior. Existing terminal, absent, and unknown-state behavior is unchanged.

### User Story 3 — Keep existing controls and make suspension discoverable (Priority: P2)

Operators retain familiar controls and output while command help correctly identifies the existing suspended-state selector.

**Why this priority**: Regression coverage must not become a reason to change working behavior or public contracts.

**Independent Test**: Check affected command help and exercise relevant existing dry-run, force, no-wait, and no-state-check cases alongside the suspension regressions.

**Acceptance scenarios**:

1. Cancel/delete state help includes suspended with its Camunda 8.10 applicability; older versions retain their existing unsupported-state handling.
2. Existing dry-run previews submit no mutations, and existing opt-outs retain their established read/submission boundaries.
3. Public signatures, result fields, error classifications, output formats, prompts, tenant boundaries, retention eligibility, and 8.8/8.9 behavior remain unchanged.

### Edge Cases

- Suspension is independent between a parent and its call-activity children; cover both mixed combinations.
- A delayed observation can still show a suspended child after the root appears terminal during confirmation. Normal waiting must not treat that observed suspended child as finished.
- A persistently suspended observation does not prove a listener fault. Preserve bounded failure handling without inventing a diagnosis.
- Overlapping CLI selections test the existing planner. This does not create a new deduplication guarantee for direct library collections or separate invocations.
- Existing successful terminal/absent cases and unknown-state failures remain regression controls, not a new reconciliation design.

## Requirements

### Functional Requirements

- **FR-001**: Preserve suspended-as-unfinished classification and existing root cancellation, force escalation, and engine-managed cascading; never add resumption or listener bypass.
- **FR-002**: Verify suspended roots and both mixed active/suspended tree combinations through the existing workflow.
- **FR-003**: Verify existing CLI root deduplication for overlapping selections without expanding its contract to other entry points.
- **FR-004**: Preserve normal cancellation confirmation, bounded waiting, and accepted-versus-confirmed behavior when observations remain suspended.
- **FR-005**: Preserve non-force refusal, descendant-first deletion, and conflict-triggered force cancellation recovery. Preserve the terminal prerequisite before a deletion retry, including with no-wait.
- **FR-006**: Preserve existing opt-outs, retries, terminal/absent/unknown handling, public results, output, authorization, retention, and 8.8/8.9 behavior.
- **FR-007**: Correct cancel/delete state-selector help to include the existing suspended option and its version applicability; keep related user documentation consistent.
- **FR-008**: Limit any production fix to a demonstrated suspension-specific failure captured by a focused regression. Passing behavior requires no production rewrite. General hardening findings belong in separate work.

### Key Entities

- **Selected instance**: An existing process-instance identity and observed state.
- **Root and family**: Existing ancestry and discovered relatives used by current planning and confirmation.
- **Operation result**: Existing submission, success, and failure information; no new outcome model is required.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Suspended-root and both mixed-tree scenarios complete through the existing root flow with zero resume or additional child-cancellation actions.
- **SC-002**: An overlapping CLI root/child selection produces one planned root and one cancellation submission in the successful non-retried scenario.
- **SC-003**: Persistently suspended confirmation scenarios never report completion; failed force-recovery prerequisites produce zero deletion retries, including with no-wait.
- **SC-004**: Successful force recovery retains the established request order, and non-force CLI refusal submits zero mutations.
- **SC-005**: Affected help lists the suspended selector accurately; relevant existing compatibility checks pass without changing public contracts or older-version behavior.

## Assumptions

- Suspended-state support from #339 is already present on the inspected baseline.
- Camunda supports cancellation while suspended and cascading termination from the root. The engine owns this behavior; c8volt does not reproduce it.
- Tests use controlled observations to verify client behavior. They do not prove real broker, listener, or exporter timing.
- This revision authorizes specification and planning changes only. Implementation begins in a later workflow.

## Scope and Validation Boundary

#341 owns focused suspension regressions, the small help correction, and only minimal fixes justified by those regressions. #340 owns definition-level ACTIVE-plus-SUSPENDED discovery, counts, deletion checks, and purge orchestration; it should reuse existing instance operations and does not require a new #341 mutation framework. Any relevant failure found here is a concrete dependency to assess, not a blanket redesign prerequisite. #342 remains separate validation context.

Out of scope: new lifecycle packages, execution ledgers, public outcome fields, deletion gates, retry/reconciliation redesign, changed deletion order, changed opt-out read counts, broader direct-library deduplication, traversal/pagination hardening, new suspend/resume commands, retention changes, generated-client changes, and server upgrades.

Implementation validation uses isolated mocks, test-owned HTTP servers, and isolated command tests only. No live target designation, discovery, deployment, mutation, or cleanup is an implementation completion gate. Any real-engine exercise requires a separate explicit request for a specific target. Existing historical live evidence remains historical; no new runtime or live validation was performed for this revision.
