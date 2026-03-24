# Suggested GitHub Issue Title

feat(cli): define an explicit non-interactive automation mode for ai agents

## GitHub Issue Body

## Summary

AI agents need a predictable way to run `c8volt` without getting stuck on prompts, ambiguous progress output, or command-specific interaction rules. This issue introduces a deliberate non-interactive automation mode built on top of existing CLI behavior so agents and CI-style callers can run commands safely and predictably.

## Problem

`c8volt` already has some automation-friendly flags such as `--auto-confirm`, `--json`, and quiet or verbose logging options. However, an agent still has to know which combination of flags to use, which commands may prompt, and how logs or confirmation behavior interact with machine-readable output.

That creates several risks:

- agents may forget to disable prompts for state-changing commands
- logs may be mixed with output in ways that complicate parsing or follow-up decisions
- different commands may feel automation-friendly but not yet behave as one coherent non-interactive contract

## Goal

Define a clear, documented, and testable non-interactive mode that AI agents and other automation can rely on when invoking `c8volt`.

## Scope

In scope:

- define what non-interactive mode means for `c8volt`
- determine whether the mode is represented by one new flag, a documented flag combination, or a small shared execution contract
- ensure prompts, confirmation flows, and progress output behave predictably in that mode
- ensure the mode works cleanly with the recommended structured output path
- add targeted tests for representative state-changing commands

Out of scope:

- designing an MCP server
- broad command redesign
- replacing current human-friendly interactive behavior

## Desired Behavior

- A caller can opt into one clear automation mode for non-interactive execution.
- State-changing commands do not block on prompts in that mode.
- Output behavior in that mode is documented and consistent enough for machine consumers.
- If a command cannot safely proceed in non-interactive mode, the failure is explicit and actionable.
- Existing human workflows remain available outside the automation mode.

## Acceptance Criteria

- The project defines and documents the intended non-interactive behavior for representative read and write commands.
- Representative prompting commands have tests proving they do not block in the supported automation mode.
- Structured output remains usable in non-interactive mode.
- The implementation reuses existing flags and conventions where practical instead of introducing an unnecessary parallel UX.

## Constraints and Guidance

- Favor one easy-to-remember automation contract over a scattered set of implicit rules.
- Reuse current project patterns around confirmation, waiting, and error handling.
- Keep the design compatible with future MCP use, where tools will also need deterministic non-interactive semantics.

## Why This Matters

Agents are only useful if they can execute safely without supervision for well-defined tasks. A first-class non-interactive mode reduces the risk of hangs, bad parsing, and accidental misuse of state-changing commands.
