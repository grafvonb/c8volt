# Suggested GitHub Issue Title

docs(cli): improve ai-oriented help and command discoverability

## GitHub Issue Body

## Summary

`c8volt` help text is already useful for human operators, but AI agents benefit from more explicit guidance about when to use a command, which output mode to prefer, and what the safest follow-up action is. This issue focuses on improving help and generated CLI documentation so the existing command surface becomes easier for both humans and agents to discover and use correctly.

## Problem

Even with a good command structure, AI agents often make mistakes when help text does not explicitly answer questions like:

- which output mode is best for automation
- whether a command is read-only or state-changing
- whether a command waits for confirmed completion by default
- which follow-up command should be used to verify or inspect results
- which flags are mutually exclusive or required together

Those details are often present in implementation or examples, but not expressed consistently enough in command help to serve as reliable operating guidance.

## Goal

Make `c8volt` help, examples, and generated CLI docs significantly better for AI-assisted and automation-heavy usage without turning the CLI into an AI-only product.

## Scope

In scope:

- improve Cobra `Short`, `Long`, and `Example` text for the most important command families
- explicitly describe recommended output modes for automation
- explicitly describe default waiting behavior and non-interactive usage where relevant
- include copy-pasteable examples that work well for agent planning and recovery
- propagate those improvements into generated docs under `docs/cli/`

Out of scope:

- changing the underlying command behavior unless a small clarification fix is needed
- adding new protocol surfaces such as MCP
- solving full machine contracts in this issue

## Desired Help Improvements

- Each important command explains when it should be used.
- State-changing commands call out whether they wait for confirmation and how to disable or tune that behavior.
- Commands that support `--json` clearly recommend it for automation and AI use.
- Mutual exclusivity and flag dependencies are surfaced in help, not only in runtime errors.
- Examples show realistic operational flows, including read-only inspection followed by a state-changing action and then verification.

## Acceptance Criteria

- The selected command set has updated help text and examples that are materially more useful for automation and AI consumers.
- Generated CLI docs are refreshed from command metadata rather than hand-edited.
- At least one command in each major family covered by this issue documents:
  - recommended output mode
  - whether the command changes state
  - whether the command waits by default
  - a realistic follow-up verification command where relevant
- Documentation remains aligned with actual behavior.

## Constraints and Guidance

- Keep the existing CLI style and terminology.
- Prefer concise, precise guidance over marketing language.
- Reuse existing repository conventions for generated docs.
- If a help improvement reveals ambiguous or inconsistent behavior, record the behavior gap clearly instead of hiding it in prose.

## Why This Matters

Better help text will not replace a formal machine contract, but it will make the current CLI much easier for AI agents, scripts, and humans to use safely and correctly while broader machine-facing work is still evolving.
