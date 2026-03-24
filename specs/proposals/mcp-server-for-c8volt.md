# Suggested GitHub Issue Title

feat(mcp): expose c8volt operations through an mcp server

## GitHub Issue Body

## Summary

`c8volt` is a strong operations CLI, but AI agents often work better with typed tools than with shell commands and help-text interpretation. This issue adds an MCP server for `c8volt` so AI clients can use core Camunda operational capabilities directly through tool calls while reusing the existing service layer and command semantics where appropriate.

## Problem

Using the CLI from an AI agent is possible, but it still requires the model to:

- discover commands and flags
- compose shell invocations correctly
- parse command output and errors
- understand which operations are safe, read-only, or state-changing

MCP provides a better integration surface for many AI clients because it exposes typed tools directly rather than forcing everything through shell parsing.

## Goal

Add an initial MCP server for `c8volt` that exposes a focused set of high-value operations through typed tools while reusing existing business logic instead of creating a second implementation path.

## Scope

In scope:

- add a new MCP server entry point in Go
- use an established Go MCP SDK rather than inventing protocol handling manually
- expose a narrow first tool set based on existing `c8volt` capabilities
- reuse existing internal services or facade layers wherever possible
- define structured tool inputs, outputs, and error behavior
- document how the MCP layer relates to existing CLI semantics

Suggested initial tool candidates:

- get cluster information
- get process definition
- get process instance
- get process instances
- run process instance
- cancel process instance
- expect process instance state
- walk process instance family or ancestry

Out of scope:

- exposing every CLI command in the first iteration
- replacing the CLI
- introducing business operations that do not already fit existing `c8volt` capabilities

## Desired Behavior

- An MCP client can connect to `c8volt` and invoke a small set of useful operational tools without shelling out manually.
- Tool inputs are typed and narrow enough for reliable model use.
- Tool results are structured and map cleanly to the underlying service outcomes.
- The MCP server keeps protocol traffic separate from ordinary logs and avoids transport-corrupting output behavior.
- The design allows incremental addition of more tools later.

## Acceptance Criteria

- A working Go-based MCP server exists and can serve at least the initial focused tool set.
- The MCP tool handlers reuse current `c8volt` services or facades rather than reimplementing Camunda logic from scratch.
- Tool input and output contracts are documented and tested for representative success and failure paths.
- The implementation documents how MCP tool behavior aligns with existing CLI behavior, especially for waiting, confirmation, and error semantics.

## Constraints and Guidance

- Prefer a thin adapter over the existing domain and service layers.
- Keep the first tool set intentionally small and operationally valuable.
- Avoid coupling MCP design too tightly to current Cobra command structure when the underlying service abstractions are a better fit.
- Preserve room for future parity work without requiring full parity in the first issue.

## Why This Matters

If `c8volt` should be genuinely AI-agent-capable, MCP is the most direct path to a first-class AI integration surface. The CLI can remain excellent for humans, while MCP becomes the typed tool layer for AI systems.
