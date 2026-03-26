# Feature Specification: Add Resource Get Command By Id

**Feature Branch**: `73-get-resource-id`  
**Created**: 2026-03-21  
**Status**: Draft  
**Input**: User description: "https://github.com/grafvonb/c8volt/issues/73"

## GitHub Issue Traceability

- **Issue Number**: `73`
- **Issue URL**: `https://github.com/grafvonb/c8volt/issues/73`
- **Issue Title**: `Add c8volt get resource --id command`

## Clarifications

### Session 2026-03-21

- Q: What should successful `c8volt get resource --id` output contain? → A: Show the normal resource details/object for that single resource.
- Q: How should the command behave when `--id` is missing? → A: `--id` is required and missing it is a validation error before any lookup.
- Q: Should raw resource content retrieval be part of this feature? → A: No, this feature is scoped only to metadata/details lookup.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Retrieve one resource by id (Priority: P1)

As an operator, I want to fetch a single resource by id from the CLI so that I can inspect the deployed resource content without building my own API call.

**Why this priority**: The issue is specifically about adding the missing `get resource --id` command, so direct resource lookup is the core value without expanding into separate raw-content retrieval behavior.

**Independent Test**: Run `c8volt get resource --id <id>` against an environment containing the requested resource and confirm the command returns that resource in the expected command output shape.

**Acceptance Scenarios**:

1. **Given** a reachable environment and an existing resource id, **When** the operator runs `c8volt get resource --id <id>`, **Then** the command returns the matching resource as the normal single-resource details/object output.
2. **Given** a requested resource id that does not exist, **When** the operator runs `c8volt get resource --id <id>`, **Then** the command fails with a clear message and a non-success exit status.

---

### User Story 2 - Reuse existing get-command conventions (Priority: P2)

As an operator, I want the new resource lookup command to behave like the existing `c8volt get ...` commands so that I can use it without learning a different flag style or output pattern.

**Why this priority**: The command is most useful when it fits naturally into the established CLI family instead of introducing a surprising workflow for one resource type.

**Independent Test**: Compare the new resource command with existing `get` subcommands and verify that invocation, help text, required-flag validation, and output/error conventions remain consistent.

**Acceptance Scenarios**:

1. **Given** an operator familiar with the existing `c8volt get` command family, **When** they inspect help or run the resource command, **Then** the command uses the same general structure and argument conventions as related get commands.
2. **Given** a command failure caused by missing required input or backend lookup errors, **When** the operator runs the resource command, **Then** the command follows the existing CLI error and exit-code behavior used by related get commands.

---

### User Story 3 - Discover the new resource lookup workflow (Priority: P3)

As an operator, I want the command help and documentation to mention resource lookup by id so that I can find and use the new command without reading implementation code or issue history.

**Why this priority**: Once the command exists, the next highest-value outcome is making it discoverable through the same user-facing surfaces that document other CLI commands.

**Independent Test**: Review command help and generated CLI documentation to confirm the resource lookup command is described clearly enough for a user to find and run it.

**Acceptance Scenarios**:

1. **Given** an operator exploring available get commands, **When** they read `c8volt get` help or the resource command help, **Then** they can identify that resource lookup by id is supported.
2. **Given** updated CLI reference documentation, **When** the operator looks for resource retrieval commands, **Then** they can see the required `--id` usage and the purpose of the command.

### Edge Cases

- The command must fail validation before any lookup when the operator runs `c8volt get resource` without providing `--id`.
- What happens when the operator provides an empty, malformed, or whitespace-only id value?
- How does the command behave when the underlying resource lookup succeeds at the transport level but returns no resource payload?
- How does the command behave when the selected Camunda version does not support the expected resource response shape?
- Raw resource content export or download behavior is out of scope for this feature.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide a `c8volt get resource --id <id>` CLI workflow for retrieving a single resource by id.
- **FR-002**: The system MUST require an explicit `--id` resource identifier for the resource lookup workflow.
- **FR-003**: The system MUST return the requested resource as the normal single-resource details/object output when lookup succeeds.
- **FR-004**: The system MUST surface not-found and lookup-failure outcomes with clear user-facing error behavior and a non-success exit status.
- **FR-005**: The system MUST reuse the existing internal resource retrieval capability already available through the shared resource service instead of requiring a separate operator workflow.
- **FR-006**: The system MUST preserve established `c8volt get` command conventions for command structure, help text, and exit behavior.
- **FR-007**: The system MUST fail validation before attempting any lookup when `--id` is missing, empty, or otherwise invalid.
- **FR-008**: The system MUST update user-facing command help and generated CLI reference documentation to describe the new resource lookup command and its required identifier input.
- **FR-009**: The system MUST define automated test coverage for successful lookup, invalid input handling, and at least one failing resource retrieval path.
- **FR-010**: The system MUST keep raw resource content retrieval or export behavior out of scope for this feature.

### Key Entities *(include if feature involves data)*

- **Resource Lookup Request**: An operator-triggered CLI request that identifies a single resource id and asks for the corresponding deployed resource.
- **Resource Output**: The normal single-resource details/object representation returned by the command when a resource lookup succeeds.
- **Resource Lookup Failure**: A user-visible error outcome covering missing ids, not-found resources, malformed success payloads, and backend retrieval failures.

## Assumptions

- The feature extends the existing `c8volt get` command family rather than creating a new top-level command group.
- The shared `internal/services/resource` API added in issue `#71` is the intended service entry point for this CLI workflow.
- Resource lookup remains scoped to fetching a single resource by id and does not include broader list, search, or export behavior.
- Raw resource content retrieval is intentionally deferred to a separate follow-up if needed.
- The final command output should follow the formatting conventions already used by comparable `get` commands in this repository.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can retrieve an existing resource by id with a single documented CLI command.
- **SC-002**: Missing or invalid id input produces a clear non-success command result instead of ambiguous or partial output.
- **SC-003**: At least one automated test covers the primary success path and at least one automated test covers a failing lookup or invalid-input path.
- **SC-004**: The new command is discoverable through command help and generated CLI documentation without requiring users to read source code.
- **SC-005**: The feature remains bounded to the resource-by-id workflow and does not broaden into unrelated resource command redesign.
