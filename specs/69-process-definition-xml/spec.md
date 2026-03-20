# Feature Specification: Add Process Definition XML Command

**Feature Branch**: `69-process-definition-xml`  
**Created**: 2026-03-20  
**Status**: Draft  
**Input**: User description: "Add c8volt get process-definition --id --xml command. Use the new GetProcessDefinitionXML service capability, expose it through the process facade, and ensure redirecting command output to a file works as expected."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Retrieve process definition XML by id (Priority: P1)

As an operator, I want to request a process definition by id in XML form so that I can inspect or reuse the exact BPMN content for that definition.

**Why this priority**: Retrieving the XML payload is the core user value described by the issue and the command is not useful without it.

**Independent Test**: Run the command with a valid process definition id and confirm that it returns the XML representation for that definition without requiring any additional manual steps.

**Acceptance Scenarios**:

1. **Given** a reachable environment and a valid process definition id, **When** the operator runs `c8volt get process-definition --id <id> --xml`, **Then** the command returns the XML content for that definition.
2. **Given** a requested process definition id that cannot be found, **When** the operator runs the XML form of the command, **Then** the command fails with a clear message and a non-success exit status.

---

### User Story 2 - Use XML output in standard CLI workflows (Priority: P2)

As an operator, I want the XML form of the command to behave predictably in shell workflows so that I can redirect the output into a file or another tool.

**Why this priority**: The issue explicitly calls out piping or redirecting output to a file, which makes stream-safe command behavior a required part of the feature.

**Independent Test**: Redirect the command output to a file and confirm the resulting file contains only the expected XML payload while error information still appears in the appropriate failure path.

**Acceptance Scenarios**:

1. **Given** a valid process definition id, **When** the operator runs `c8volt get process-definition --id <id> --xml > example.bpmn`, **Then** the created file contains the XML payload needed for later use.
2. **Given** a failure while retrieving XML, **When** the operator redirects standard output to a file, **Then** the command does not silently present the failure as a successful XML export.

---

### User Story 3 - Understand when to use XML retrieval (Priority: P3)

As an operator, I want the command help and related documentation to explain the XML option so that I can discover and use the feature without reading source code.

**Why this priority**: Once the command works, the next highest value is making the option easy to find and safe to use in normal operator workflows.

**Independent Test**: Review the command help and any affected user-facing documentation to confirm the XML option, its purpose, and its expected output behavior are clearly described.

**Acceptance Scenarios**:

1. **Given** an operator reviewing command help or usage documentation, **When** they look for process definition retrieval options, **Then** they can identify how to request XML output by id.
2. **Given** an operator learning the new command behavior, **When** they read the updated guidance, **Then** they understand that the XML form is intended for direct display or output redirection.

### Edge Cases

- What happens when the operator provides `--xml` without a valid `--id` value?
- What happens when the requested process definition exists but its XML payload is empty or unavailable?
- How does the command behave when XML retrieval fails after the operator redirects standard output to a file?
- What happens when the operator combines `--xml` with output expectations that differ from the existing non-XML process-definition view?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide a `c8volt get process-definition --id <id> --xml` workflow for retrieving a process definition in XML form.
- **FR-002**: The system MUST require an explicit process definition identifier for the XML retrieval workflow.
- **FR-003**: The system MUST return the requested process definition XML through standard command output when retrieval succeeds.
- **FR-004**: The system MUST report retrieval failures with clear user-facing error behavior and a non-success exit status.
- **FR-005**: The system MUST preserve predictable shell behavior so operators can redirect successful XML output into a file for later use.
- **FR-006**: The system MUST define how the XML workflow relates to existing process-definition retrieval behavior so operators can tell when to use the XML form.
- **FR-007**: The system MUST update user-facing command help and any affected documentation to describe the XML option and its expected output behavior.
- **FR-008**: The system MUST identify automated test coverage needed for successful XML retrieval, failure handling, and redirected-output behavior.

### Key Entities *(include if feature involves data)*

- **Process Definition XML Request**: An operator-triggered CLI request that identifies a single process definition and asks for its XML representation.
- **Process Definition XML Payload**: The XML content returned for a specific process definition and written to standard output on success.
- **XML Retrieval Failure**: A user-visible failure outcome when the requested XML cannot be retrieved, including the command's error messaging and exit result.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can retrieve the XML form of an existing process definition with a single documented CLI command.
- **SC-002**: Redirecting successful XML output to a file produces a reusable BPMN file without extra manual cleanup.
- **SC-003**: Failure cases for missing or unavailable process definitions produce a clear non-success command result rather than ambiguous partial output.
- **SC-004**: Automated tests cover the primary success path, at least one failure path, and the redirected-output workflow for the new command behavior.
- **SC-005**: Operators can identify and use the XML option through command help or updated documentation without reading source code.

## Assumptions

- The XML retrieval workflow extends the existing `get process-definition` command family rather than introducing a separate top-level command.
- The command remains scoped to fetching a single process definition by id and does not expand into broader export or bulk-download behavior.
- Successful XML retrieval writes the payload to standard output so existing shell redirection behavior can be used naturally.
- Any required internal service or facade wiring remains an implementation concern outside the specification, as long as the user-visible command contract is met.
