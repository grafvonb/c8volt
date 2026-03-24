# Suggested GitHub Issue Title

feat(cli): add machine-readable command contracts for ai and automation

## GitHub Issue Body

## Summary

`c8volt` already works well for human operators and basic scripting, but AI agents need stronger command contracts than humans do. The CLI should make it easy for an agent to discover which commands are safe to call, which flags are supported, what output shape to expect, and how to distinguish success, partial success, validation errors, and remote API failures.

This issue is about strengthening the CLI contract for machine consumers without changing the human-oriented command model.

## Problem

Today, an AI agent can call `c8volt`, but it still has to infer too much from help text, mixed log output, one-line output modes, and command-specific behavior. That raises the chance of brittle tool use, incorrect follow-up actions, or poor recovery after errors.

The main gaps are:

- machine-readable output is not consistently positioned as the canonical automation contract
- success and error results are not described as a stable contract across commands
- agents cannot easily discover command capabilities and output expectations without reading prose help
- different command families expose useful automation behavior, but that behavior is not expressed as an explicit contract

## Goal

Define and implement a consistent machine-readable CLI contract that AI agents and other automation can rely on across the main `c8volt` command families.

## Scope

In scope:

- define a stable structured output contract for command success cases where `--json` is supported
- define a stable structured error contract or an equivalent machine-parseable error strategy
- document which command families support which output modes
- make command validation failures precise and actionable for non-human callers
- add a discovery mechanism for machine consumers, such as a `capabilities`, `schema`, or similar command that can emit JSON
- cover representative high-value command families such as `get`, `run`, `expect`, `walk`, `deploy`, `delete`, and `cancel`

Out of scope:

- adding new Camunda business operations unrelated to machine contract behavior
- redesigning the existing human CLI taxonomy
- introducing MCP in this issue

## Desired Behavior

- An agent can call a discovery command and learn which top-level and nested commands exist, which flags they accept, which output modes they support, and whether they are read-only or state-changing.
- An agent can prefer one structured output mode and receive stable machine-readable results for the supported commands.
- Validation errors tell the caller exactly what was wrong and how to correct the call.
- State-changing commands clearly distinguish request acceptance from confirmed completion, especially for operations that wait by default.
- The machine contract reuses existing project patterns and keeps the current CLI structure intact.

## Acceptance Criteria

- A machine-readable discovery surface exists and can be consumed without scraping human help text.
- At least one representative command from each major family covered by this issue has tests proving the machine-readable contract.
- The implementation defines how machine consumers should identify:
  - successful completion
  - accepted-but-not-yet-confirmed work when relevant
  - user input or validation errors
  - remote API or infrastructure failures
- Existing human-friendly usage remains supported.
- Documentation explains the recommended contract for automation and AI consumers.

## Constraints and Guidance

- Prefer incremental changes over a broad rewrite.
- Reuse existing Cobra command structure and current output patterns where possible.
- Preserve externally observable behavior for current human users unless the change is clearly documented and justified.
- Favor a contract that can later be reused by an MCP adapter instead of inventing a parallel model that would need to be translated twice.

## Why This Matters

This creates the foundation for reliable AI use of `c8volt`. Without a stable machine contract, improvements to help text or future MCP support will still rest on a brittle CLI layer.
