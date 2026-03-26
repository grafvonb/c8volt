# Feature Specification: Review and Refactor CLI Error Code Usage

**Feature Branch**: `19-cli-error-model`  
**Created**: 2026-03-24  
**Status**: Draft  
**Input**: User description: "https://github.com/grafvonb/c8volt/issues/19"

## GitHub Issue Traceability

- **Issue Number**: `19`
- **Issue URL**: `https://github.com/grafvonb/c8volt/issues/19`
- **Issue Title**: `refactor(cli): review and refactor error code usage for humans, automation, and ai agents`

## Clarifications

### Session 2026-03-24

- Q: Should this feature define one shared exit-code mapping for the representative failure classes in scope, or defer exact exit-code mapping to a later follow-up? → A: Define one shared exit-code mapping for the representative failure classes in scope.
- Q: Should this feature apply the new error model to all existing CLI commands now, or to a clearly named representative subset that becomes the rollout standard later? → A: Apply the new error model to all existing CLI commands in this feature.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Understand failures quickly as an operator (Priority: P1)

As a human operator using `c8volt`, I want command failures to be classified and explained consistently so that I can quickly understand what went wrong and what to do next.

**Why this priority**: Clear, actionable failure feedback is the most immediate user value and the foundation for all other consumers of CLI errors.

**Independent Test**: Reviewers can trigger representative validation, configuration, unsupported-operation, and remote-failure paths and confirm that each failure is presented with a consistent category and understandable next-step guidance.

**Acceptance Scenarios**:

1. **Given** a command fails before reaching the remote system because the caller supplied invalid input, **When** the CLI reports the failure, **Then** the message clearly distinguishes caller error from remote-system failure.
2. **Given** a command reaches the remote system and receives an error response, **When** the CLI reports the failure, **Then** the message makes it clear that the command reached the target system and did not fail only because of local validation.

---

### User Story 2 - React correctly in automation and AI flows (Priority: P2)

As an automation or AI caller, I want CLI failures to map to a small, stable set of failure classes and exit behaviors so that I can decide whether to retry, correct input, inspect configuration, or stop.

**Why this priority**: Machine callers depend on predictable semantics rather than ad hoc wording, and the issue explicitly names automation and AI agents as first-class consumers.

**Independent Test**: Reviewers can inspect representative failure paths and confirm that the resulting failure classes and exit behavior distinguish invalid input, unsupported operations, local precondition failures, internal failures, and remote failures.

**Acceptance Scenarios**:

1. **Given** a command fails because the configured Camunda version does not support the requested operation, **When** the CLI reports the failure, **Then** the failure is classified predictably as unsupported rather than as a generic runtime problem.
2. **Given** a command fails because of a retryable infrastructure or remote availability problem, **When** the CLI reports the failure, **Then** the failure is distinguishable from a permanent caller mistake.

---

### User Story 3 - Maintain one coherent error model across command families (Priority: P3)

As a maintainer, I want representative command families to share one deliberate error model so that refactoring and future machine-readable work can build on a consistent contract instead of one-off behaviors.

**Why this priority**: The work needs to stay incremental and maintainable, and a bounded shared model reduces future drift across the CLI.

**Independent Test**: Reviewers can compare representative `get`, `run`, `deploy`, `cancel`, `delete`, `expect`, and `walk` failure paths and confirm that similar failures map to the same documented categories and message patterns.

**Acceptance Scenarios**:

1. **Given** two different command families encounter the same failure class, **When** the CLI reports each failure, **Then** both paths use the same category and consistent operator-facing structure.
2. **Given** the project retains compatibility behavior such as `--no-err-codes`, **When** the error model is applied, **Then** that existing behavior continues to interact predictably with the new classification approach.

### Edge Cases

- A command may fail before contacting the target system because of invalid flags, missing configuration, or other local preconditions that must stay distinct from remote errors.
- Some operations may be unsupported only for certain configured Camunda versions and must not be misclassified as transient infrastructure failures.
- Internal sentinel errors and remote HTTP or API failures may produce similar operator-visible symptoms but need different failure semantics for scripts and AI agents.
- Existing flags such as `--no-err-codes` may alter what is rendered to the operator without removing the need for a stable underlying error classification approach.
- Representative command families may currently use slightly different message shapes for similar failures, requiring convergence without a broad redesign of successful outputs.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST define a shared CLI error classification approach for representative command families.
- **FR-002**: The system MUST keep the number of failure classes small enough to be understandable by human operators and reliable for machine callers.
- **FR-003**: The system MUST distinguish invalid user input and invalid flag combinations from configuration or local precondition failures.
- **FR-004**: The system MUST classify unsupported version and unsupported operation paths predictably.
- **FR-005**: The system MUST define how representative internal sentinel errors map into the shared CLI error classes.
- **FR-006**: The system MUST define how representative remote HTTP or API failures map into the shared CLI error classes.
- **FR-007**: The system MUST distinguish likely retryable infrastructure or remote availability failures from permanent caller mistakes.
- **FR-008**: The system MUST define one shared exit-code mapping for the representative failure classes covered by this feature.
- **FR-009**: The system MUST standardize the operator-facing message structure and level of detail for the representative failures covered by this feature.
- **FR-010**: The system MUST preserve understandable and actionable operator-facing feedback while improving consistency.
- **FR-011**: The system MUST remain compatible with existing behavior such as `--no-err-codes`.
- **FR-012**: The system MUST define and implement the shared error model across all existing CLI commands, including read-only, state-changing, and validation-heavy workflows.
- **FR-013**: The system MUST add or update targeted automated tests for invalid-input, local-precondition, unsupported-operation, internal, and remote-failure paths across the covered command surface.
- **FR-014**: The system MUST keep the change bounded to incremental refactoring rather than a broad redesign of successful output payloads or unrelated business capabilities.
- **FR-015**: The system MUST leave the resulting error model stable enough to support future machine-readable CLI contracts and AI-oriented tooling.

### Key Entities *(include if feature involves data)*

- **CLI Error Class**: A bounded failure category that explains what kind of problem occurred and how callers should interpret it.
- **Operator-Facing Failure Message**: The human-readable failure output that communicates the category, context, and next-step guidance.
- **Exit Behavior**: The command termination outcome used by shell automation and other callers to react to a failure.
- **CLI Command Surface**: The full set of existing CLI commands whose failure behavior must align with the shared error model.
- **Failure Source**: The origin of a failure, such as caller input, local configuration, unsupported capability, internal logic, or remote system response.

## Assumptions

- The current CLI already has enough representative failure paths to define a shared model without introducing unrelated new capabilities.
- Existing successful output behavior remains out of scope unless a failure-semantics change requires minimal documentation updates.
- Compatibility expectations for current operators remain important, so the work should favor consistent classification and messaging over disruptive behavioral changes.
- Future automation and AI tooling will benefit more from one coherent error model than from separate human-only and machine-only failure semantics.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All existing CLI commands classify similar failures consistently according to the shared error model.
- **SC-002**: Automated tests cover at least invalid input, local precondition, unsupported operation or version, internal sentinel, and remote failure groups across the command surface in scope.
- **SC-003**: Reviewers can determine from the documented model whether a representative failure is caused by caller input, local setup, unsupported capability, internal logic, or remote system response.
- **SC-004**: Reviewers can determine from the resulting failure contract whether a representative failure is more likely to require correction, inspection, or retry.
- **SC-005**: The updated model defines one shared exit-code mapping for the representative failure classes in scope while remaining compatible with existing behavior such as `--no-err-codes`.
