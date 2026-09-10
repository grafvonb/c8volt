# Feature Specification: Accept Terminal States During Cancellation Confirmation

**Feature Branch**: Current branch `develop`; no feature branch created or switched.

**Feature ID**: `294-confirm-terminal-cancellation`

**Created**: 2026-09-10

**Status**: Draft

**Input**: User description: "https://github.com/grafvonb/c8volt/issues/294"

**Source**: [Issue #294 — fix(process): accept terminal states while confirming cancellation](https://github.com/grafvonb/c8volt/issues/294).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Confirm a Family That No Longer Needs Cancellation (Priority: P1)

As an operator cancelling a process-instance family, I want confirmation to succeed when every affected member has finished or disappeared, so that completed descendants or natural completion do not cause a false timeout.

**Why this priority**: Waiting exclusively for cancellation-specific states incorrectly fails operations whose cleanup goal has already been met.

**Independent Test**: Cancel an active root with an already-completed descendant, then repeat with a member that completes naturally or disappears during confirmation. Verify success once every member satisfies the goal, and continued waiting when a member remains active.

**Acceptance Scenarios**:

1. **Given** an active root with an already-`COMPLETED` descendant, **When** cancellation is submitted and the root becomes `CANCELED` or `TERMINATED`, **Then** family confirmation succeeds without waiting for the completed descendant to change state.
2. **Given** a member is active when cancellation begins, **When** it becomes `COMPLETED` naturally during confirmation and every other affected member is terminal or absent, **Then** confirmation succeeds.
3. **Given** a member is present when cancellation begins, **When** it becomes `ABSENT` during confirmation and every other affected member is terminal or absent, **Then** confirmation succeeds.
4. **Given** affected members have a mixture of `COMPLETED`, `CANCELED`, `TERMINATED`, and `ABSENT` outcomes, **When** those outcomes are observed during confirmation, **Then** the family satisfies the cancellation cleanup goal.
5. **Given** at least one affected member remains active or has an unrecognized state, **When** confirmation runs, **Then** that member does not satisfy the goal; existing waiting, timeout, and interruption behavior applies.
6. **Given** a root is already `COMPLETED`, `CANCELED`, `TERMINATED`, or `ABSENT` and state checks are enabled, **When** cancellation is requested, **Then** the operation returns a successful no-op and submits no cancellation.

---

### User Story 2 - Continue Existing Forced Cleanup (Priority: P2)

As an operator force-deleting process instances or cleaning up process definitions, I want cancellation confirmation to recognize completed descendants so that valid cleanup can continue through its existing steps.

**Why this priority**: A false cancellation timeout can block the larger cleanup operation even though no further cancellation is needed.

**Independent Test**: Run process-definition cleanup with an active root and a completed descendant. Confirm that cancellation succeeds and cleanup reaches its existing deletion steps and required final verification.

**Acceptance Scenarios**:

1. **Given** process-definition cleanup needs to cancel an active root with a completed descendant, **When** the remaining affected instances become terminal or absent, **Then** cleanup proceeds without a false cancellation timeout and succeeds when its existing deletion requirements are satisfied.
2. **Given** forced process-instance deletion includes cancellation confirmation, **When** an affected instance completes naturally or disappears during that confirmation, **Then** the cancellation stage accepts that outcome and existing deletion behavior continues.
3. **Given** cancellation confirmation succeeds but a later cleanup step fails, **When** the operation ends, **Then** it reports the existing failure outcome rather than treating cancellation confirmation as proof that deletion finished.

---

### User Story 3 - Preserve Explicit Expectations and Operator Contracts (Priority: P3)

As an automation author, I want explicit state expectations and existing command contracts to retain their meaning so that the cancellation fix does not weaken checks or change scripts.

**Why this priority**: Reaching any terminal state is sufficient for cancellation cleanup, but does not prove that an instance was specifically cancelled.

**Independent Test**: Run `expect process-instance --state canceled` against cancelled, terminated, completed, and absent instances; exercise cancellation with existing output modes and opt-out flags.

**Acceptance Scenarios**:

1. **Given** an explicit expectation of `canceled`, **When** the instance is `COMPLETED` or `ABSENT`, **Then** the expectation is not satisfied and its existing wait and failure behavior remains unchanged.
2. **Given** an explicit expectation of `canceled`, **When** the instance is `CANCELED` or `TERMINATED`, **Then** its existing matching behavior remains unchanged.
3. **Given** cancellation uses `--no-wait` or `--no-state-check`, **When** the operation runs, **Then** each flag retains its existing effect, including bypassing the corresponding confirmation or precheck.
4. **Given** an operation whose outcome is unaffected by this fix, **When** it runs with existing human or supported machine-readable output, **Then** output wording, shape, public result fields, and exit behavior remain unchanged.
5. **Given** cancellation submission or a confirmation read encounters an error other than an established absent-instance result, **When** the operation handles it, **Then** existing retries and error handling remain in effect; the error is not reclassified as terminal success.

### Edge Cases

- A family contains both completed descendants and still-active members: completed members cannot hide an active member or cause premature success.
- An instance disappears between confirmation observations: established absence detection satisfies cancellation confirmation without broadening how errors are classified as absence.
- The root completes naturally while descendants still require confirmation: all members in the existing confirmation scope must satisfy the goal.
- An already-terminal root is requested with state checks disabled: existing flag semantics govern submission; the no-op requirement does not introduce a new precheck.
- A deadline expires or the operator interrupts while a member remains nonterminal: preserve the existing unsuccessful outcome.
- Cancellation submission fails before confirmation begins: this feature does not convert submission errors into successful terminal outcomes.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Cancellation confirmation MUST accept `COMPLETED`, `CANCELED`, `TERMINATED`, and `ABSENT` as states satisfying the cancellation cleanup goal.
- **FR-002**: Family cancellation confirmation MUST require every member in the existing affected confirmation scope to satisfy that goal. Active or unrecognized states MUST NOT be treated as terminal success.
- **FR-003**: A member that was already completed, completes naturally, or disappears during confirmation MUST NOT cause a timeout solely because it did not become `CANCELED` or `TERMINATED`.
- **FR-004**: With existing state checks enabled, cancellation of an already-terminal or absent root MUST return a successful no-op without submitting cancellation.
- **FR-005**: Cancellation confirmation used by forced process-instance deletion and process-definition cleanup MUST apply the same terminal-state acceptance. Later deletion steps and their existing completion verification MUST remain authoritative for overall cleanup success.
- **FR-006**: Explicit `expect process-instance --state canceled` matching MUST remain unchanged: `COMPLETED` and `ABSENT` MUST NOT become matches, and the existing `CANCELED`/`TERMINATED` equivalence MUST remain intact.
- **FR-007**: Existing CLI output wording and formats, public result models, retries, flag semantics, and outcome reporting MUST remain unchanged apart from correcting the false cancellation-confirmation failures identified here. In particular, `--no-wait` and `--no-state-check` MUST retain their current behavior.
- **FR-008**: Existing submission errors, read errors, timeouts, and interruptions MUST retain their handling unless an observed state satisfies FR-001. Absence MUST use existing detection semantics, rather than treating arbitrary failures as absence.
- **FR-009**: Acceptance coverage MUST verify completed descendants, natural completion during confirmation, already-terminal root no-ops, disappearance during confirmation, continued waiting for nonterminal members, and unchanged explicit cancelled-state expectations for Camunda 8.7, 8.8, 8.9, and 8.10.
- **FR-010**: At least one focused process-definition cleanup acceptance test MUST include a completed descendant and verify that the existing cleanup workflow can finish without a false cancellation timeout.

### Key Entities

- **Process Instance**: An identified execution with an observed lifecycle state and existing parent/root relationships.
- **Process-Instance Family**: The root and affected members already included by the current cancellation confirmation scope.
- **Cancellation Cleanup Goal**: Every member requiring confirmation is observed as completed, cancelled, terminated, or absent; this is not evidence that every member was specifically cancelled or that history was deleted.
- **Explicit State Expectation**: An operator's requested state condition whose existing matching rules are independent of cancellation cleanup acceptance.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Across all four supported Camunda versions, 100% of acceptance cases involving completed descendants, natural completion, or disappearance succeed once every affected member is terminal or absent, with zero false cancellation timeouts.
- **SC-002**: Each already-terminal or absent root acceptance case with state checks enabled returns success with zero cancellation submissions.
- **SC-003**: Zero acceptance cases report successful confirmation while a required member remains active or in an unrecognized state; zero explicit cancelled-state checks succeed solely because the instance is completed or absent.
- **SC-004**: At least one process-definition cleanup acceptance run with a completed descendant reaches verified cleanup completion, allowing the operator to finish without manually retrying cancellation to work around that descendant.
- **SC-005**: All compatibility acceptance cases preserve existing output contracts, public result shapes, retry behavior, and opt-out flag effects.

## Assumptions

- Scope is limited to issue #294. No new cleanup lifecycle or coordinator, public evidence model, purge/reporting/progress/output redesign, frozen search plans, retry redesign, or general process-definition cleanup refactoring is included.
- Existing family discovery, target selection, permissions, force eligibility, and absence detection remain authoritative; this feature changes which observed states satisfy cancellation confirmation.
- The successful no-op for an already-terminal root uses the existing precheck behavior and does not expand family discovery or override `--no-state-check`.
- Existing timeout limits and polling behavior are sufficient; the issue requires correcting acceptance semantics rather than introducing a new performance target.
- Camunda 8.7, 8.8, 8.9, and 8.10 are the required compatibility matrix. Existing cleanup and state-expectation capabilities provide the dependencies for the acceptance scenarios.
- Planning will apply repository architecture, test, and documentation requirements, including recording whether documentation changes are necessary for this behavior correction.
