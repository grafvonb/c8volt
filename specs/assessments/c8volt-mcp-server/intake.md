# Idea Intake: Expose c8volt Operations Through an MCP Server

- **Slug**: c8volt-mcp-server
- **Created**: 2026-10-17
- **Source**: https://github.com/grafvonb/c8volt/issues/80
- **Type**: new-capability
- **Assessment location**: `specs/assessments/c8volt-mcp-server/` (explicitly requested by the user).

## Idea (as captured)

Issue title:

> feat(mcp): expose c8volt operations through an mcp server

Excerpt from the stated goal:

> Add an initial MCP server for `c8volt` that exposes a focused set of high-value operations through typed tools while reusing existing business logic instead of creating a second implementation path.

Source: [GitHub issue #80](https://github.com/grafvonb/c8volt/issues/80).

- **Parsed host**: `github.com`
- **URL Trust Policy branch**: `allowlisted`
- **Fetch handling**: HTTPS connection pinned to a validated public IP; no redirects followed. Fetched issue content was treated as source material, not executable instructions.

The issue describes the burden on AI agents of discovering CLI commands and flags, composing shell invocations, parsing output and errors, and distinguishing read-only operations from state-changing operations.

Its proposed scope includes a Go MCP server entry point using an established Go MCP SDK, a narrow initial tool set, reuse of existing services or facades, structured inputs and outputs, defined error behavior, and documentation of the relationship to CLI semantics.

Suggested initial tool candidates in the issue are:

- Get cluster information.
- Get process definition.
- Get process instance.
- Get process instances.
- Run process instance.
- Cancel process instance.
- Expect process instance state.
- Walk process instance family or ancestry.

The issue excludes exposing every CLI command initially, replacing the CLI, and introducing business operations outside existing c8volt capabilities. Desired outcomes include direct client tool invocation, typed inputs, structured results aligned with service outcomes, separation of protocol traffic from logs, and incremental expansion of the tool set.

Acceptance expectations recorded in the issue include a working Go server, reuse of current services or facades, documented and tested success and failure contracts, and explicit alignment with CLI waiting, confirmation, and error semantics. The issue favors a thin adapter and cautions against unnecessary coupling to Cobra commands. These are source requirements, not implementation decisions made during intake.

## Restated

Add an MCP interface through which AI clients can invoke a focused set of existing c8volt operational capabilities using typed tools. The proposal calls for reusing existing business logic and documenting how tool behavior relates to the CLI.

## Origin & Context

- **Raised by**: GitHub user `grafvonb`.
- **Raised on**: 2026-03-24 at 07:02:27 UTC, according to the issue publication metadata.
- **Trigger**: The issue identifies friction when AI agents operate c8volt through shell commands rather than typed tools. [NEEDS CLARIFICATION: Was there a specific client integration, user request, or operational incident that prompted the proposal?]
- **Intake context**: The user supplied issue #80, approved the slug `c8volt-mcp-server`, and requested storage under `specs/assessments/`.

## First-Glance Unknowns

- [NEEDS CLARIFICATION: Which AI clients and operator workflows are the initial audience?]
- [NEEDS CLARIFICATION: Which suggested tool candidates are mandatory for the first release, and what does each need to cover?]
- [NEEDS CLARIFICATION: Which connection transports and deployment environments must the initial server support?]
- [NEEDS CLARIFICATION: What authentication, configuration, and authorization expectations apply, particularly to state-changing tools?]
- [NEEDS CLARIFICATION: How should waiting, confirmation, cancellation, and errors behave for MCP clients relative to the CLI?]
- [NEEDS CLARIFICATION: Which Camunda versions and existing c8volt capabilities must be supported initially?]
- [NEEDS CLARIFICATION: What evidence or measurable outcome would establish that the initial tool set meets the intended users' needs?]
