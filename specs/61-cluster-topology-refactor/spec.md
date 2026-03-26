# Feature Specification: Refactor Cluster Topology Command

**Feature Branch**: `61-cluster-topology-refactor`  
**Created**: 2026-03-18  
**Status**: Draft  
**Input**: User description: "https://github.com/grafvonb/c8volt/issues/61"

## Clarifications

### Session 2026-03-18

- Q: How should the deprecated `c8volt get cluster-topology` command communicate deprecation during the migration period? → A: Document and mark it as deprecated in help and documentation only, with no runtime warning.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Use the new nested command path (Priority: P1)

As an operator, I want to run cluster topology retrieval through `c8volt get cluster topology` so that the command structure matches the rest of the CLI without changing the result I receive.

**Why this priority**: The issue's primary goal is to move the topology command into a clearer command hierarchy while preserving existing behavior.

**Independent Test**: Run the new command path against a valid environment and confirm it succeeds with the same user-visible outcome, output shape, and exit behavior as the current topology command.

**Acceptance Scenarios**:

1. **Given** an operator can retrieve cluster topology today, **When** they run `c8volt get cluster topology`, **Then** the command completes successfully and returns the same observable topology information as the existing command path.
2. **Given** the current topology command would return an error for invalid input or execution failure, **When** the operator runs `c8volt get cluster topology` under the same conditions, **Then** the new command path reports the same class of failure with equivalent exit semantics.

---

### User Story 2 - Keep the legacy command working during migration (Priority: P2)

As an existing user, I want `c8volt get cluster-topology` to keep working for now so that I can upgrade without immediately changing my scripts.

**Why this priority**: The issue explicitly requires backward compatibility through a deprecated but still functional legacy command.

**Independent Test**: Run the legacy command after the change and confirm it still works, remains documented as deprecated in help or user-facing documentation, and produces the same observable result as the new command path.

**Acceptance Scenarios**:

1. **Given** an existing script calls `c8volt get cluster-topology`, **When** it runs after the refactor, **Then** it still completes with the same functional result as before.
2. **Given** an operator reviews CLI help or affected documentation, **When** they look up the legacy command, **Then** they can see that `c8volt get cluster topology` is the preferred replacement.

---

### User Story 3 - Understand the supported command behavior (Priority: P3)

As a contributor or operator, I want command help and documentation to reflect the new hierarchy and the deprecated alias so that I know which command path should be used going forward.

**Why this priority**: A command-tree refactor is only useful if people can discover the new path and understand the transition plan.

**Independent Test**: Review help output and affected user-facing documentation to confirm the new command path is discoverable and the legacy path is documented as deprecated where relevant.

**Acceptance Scenarios**:

1. **Given** a user explores the `get cluster` command area, **When** they inspect command help or related documentation, **Then** `topology` is shown in the expected hierarchy.
2. **Given** the deprecated command remains available, **When** users review affected help text or documentation, **Then** they can identify the recommended replacement command without reading source code.

### Edge Cases

- The new nested command path must preserve current output and exit behavior for both successful and failing topology retrievals.
- The deprecated command must remain functional even if users call it from existing scripts or automation that do not inspect CLI help or updated documentation.
- Help and usage text must not imply that the legacy command has already been removed while it is still intentionally supported.
- If no user-facing documentation currently mentions the legacy command, the feature must still make the preferred command path discoverable through CLI help.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST expose cluster topology retrieval through the user-visible command path `c8volt get cluster topology`.
- **FR-002**: The system MUST preserve the current observable behavior of cluster topology retrieval, including successful output, failure behavior, and exit semantics, when invoked through the new command path.
- **FR-003**: The system MUST keep `c8volt get cluster-topology` working during the deprecation period.
- **FR-004**: The system MUST clearly indicate in CLI help and affected documentation that `c8volt get cluster-topology` is deprecated and identify `c8volt get cluster topology` as the replacement command.
- **FR-005**: The system MUST ensure the deprecated and replacement command paths execute the same topology retrieval capability rather than diverging in user-visible behavior.
- **FR-006**: The system MUST define the user-visible command behavior, output, and exit semantics for both the new command path and the deprecated compatibility path, including that the legacy path does not emit a runtime deprecation warning.
- **FR-007**: The system MUST make the new command hierarchy discoverable through CLI help or equivalent user-facing command guidance.
- **FR-008**: The system MUST update user-facing documentation in the same change whenever existing documentation or examples are affected by the new command hierarchy or deprecation messaging.
- **FR-009**: The system MUST define how maintainers verify preserved behavior and deprecation behavior through automated tests and observable CLI outcomes.
- **FR-010**: The system MUST add or update automated tests to cover the new command path, the deprecated compatibility path, and the preserved topology behavior.
- **FR-011**: The system MUST keep the feature bounded to command-structure refactoring and migration guidance without changing the underlying topology functionality.

### Key Entities *(include if feature involves data)*

- **Cluster Topology Command Path**: The user-visible CLI path that triggers topology retrieval and determines how the capability is discovered in help output.
- **Deprecated Command Alias**: The legacy command entry point that remains temporarily supported while directing users toward the new hierarchy.
- **Topology Retrieval Outcome**: The observable result of running the command, including output content, failure messaging, and process exit behavior.

## Assumptions

- The underlying topology retrieval logic already exists and does not need a functional redesign for this issue.
- The deprecation period continues beyond this change, so the legacy command should remain usable rather than acting as a hard failure.
- Existing automation depends on the legacy command potentially continuing to return the same successful or failing result as before.
- Deprecation guidance is intended to appear in CLI help and affected documentation rather than in runtime command output.
- Documentation updates are only required where the changed command structure or deprecation path is user-visible today.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can retrieve cluster topology successfully through `c8volt get cluster topology` without learning any source-code details.
- **SC-002**: Existing automation using `c8volt get cluster-topology` continues to complete with the same functional result during the deprecation period.
- **SC-003**: Maintainers can verify both command paths and their deprecation expectations through automated tests and documented CLI behavior in a single review session.
- **SC-004**: Users reviewing CLI help or affected documentation can identify `c8volt get cluster topology` as the preferred command path on first read.
