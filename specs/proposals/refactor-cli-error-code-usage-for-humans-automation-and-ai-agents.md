# Suggested GitHub Issue Title

refactor(cli): review and refactor error code usage for humans, automation, and ai agents

## GitHub Issue Body

## Summary

Review and refactor how `c8volt` classifies, maps, and renders CLI errors so failures are easier to understand for human operators and more reliable for shell automation and AI agents.

`c8volt` already focuses on operational correctness, but that same standard should apply to failure behavior. A caller should be able to tell what kind of failure happened, whether retrying makes sense, whether the input was wrong, and whether the target system rejected the request or the CLI failed before that point.

## Problem

As `c8volt` grows, error handling can become inconsistent across command families unless the project defines a shared model for:

- error categories
- exit-code behavior
- user-facing messages
- mapping from internal sentinel errors and HTTP/API failures
- machine-facing failure semantics for scripts and AI agents

Without a clearer contract, several kinds of callers suffer:

- humans get uneven error wording and unclear next steps
- shell scripts have to infer too much from text
- AI agents cannot reliably distinguish validation problems from unsupported operations or remote infrastructure failures

This issue is about making failure behavior more intentional, more consistent, and easier to build on in future machine-readable CLI and MCP work.

## Goal

Define and implement a shared error-handling model for `c8volt` that:

- improves consistency across commands
- preserves useful operator-facing feedback
- gives automation and AI consumers a more stable failure contract

## Scope

In scope:

- review current CLI-facing error behavior across representative command families
- define a small, clear set of CLI error categories or failure classes
- define how representative internal errors map into those categories
- define how representative HTTP and remote API failures map into those categories
- standardize exit-code usage where appropriate
- standardize message structure and level of detail for representative failures
- ensure the design remains compatible with existing flags such as `--no-err-codes`
- add targeted tests for representative failure paths

Representative command families should include a mix of:

- read-only commands such as `get`
- state-changing commands such as `run`, `deploy`, `cancel`, and `delete`
- validation-heavy or expectation-oriented commands such as `expect` and `walk`

Out of scope:

- broad redesign of successful output payloads
- introducing full MCP support
- adding entirely new business capabilities unrelated to error handling

## Desired Behavior

- Similar failures are reported consistently across commands.
- Invalid user input is clearly distinguishable from remote API failure.
- Unsupported version or unsupported operation paths are classified predictably.
- Retryable infrastructure failures are distinguishable from permanent caller mistakes.
- Operator-facing messages remain understandable and actionable.
- The resulting error model is stable enough that future automation and AI features can depend on it.

## AI and Automation Considerations

This issue should explicitly treat automation and AI agents as first-class consumers of CLI failures.

The resulting design should make it easier for a machine caller to answer questions like:

- Did the command fail because the input was invalid?
- Did the command fail because the requested operation is unsupported for the configured Camunda version?
- Did the command reach the remote system and receive an error response?
- Is the failure likely retryable?
- Should the next action be to correct the command, inspect configuration, or wait and retry?

This does not require a brand-new wire protocol, but it does require consistent failure semantics and a bounded error model that can later support stronger machine-readable contracts.

## Acceptance Criteria

- A shared CLI error classification and mapping approach is defined for representative commands.
- Representative tests cover at least these failure groups:
  - invalid user input or invalid flag combinations
  - configuration or local precondition failures
  - unsupported version or unsupported operation failures
  - internal sentinel errors
  - remote HTTP or API failures
- Exit-code behavior is documented and consistent for the representative paths covered by the issue.
- Operator-facing error messages remain actionable and understandable.
- The implementation defines how the updated model interacts with existing behavior such as `--no-err-codes`.
- The final design is suitable for future machine-readable CLI contracts and AI-oriented tooling.

## Constraints and Guidance

- Prefer incremental refactoring over a broad rewrite.
- Reuse existing project patterns where they are already sound.
- Preserve externally observable behavior unless inconsistency or ambiguity is itself the problem being fixed.
- Keep the number of error categories small enough to stay understandable.
- Avoid creating separate parallel error models for humans and machines; use one coherent model that serves both.

## Why This Matters

Reliable operational tooling is defined as much by how it fails as by how it succeeds. If `c8volt` wants to be dependable for humans, scripts, and AI agents, its error behavior needs to be clear, consistent, and deliberately designed rather than incidental.
