# CLI Command Contract: Shared Failure Semantics

## Scope

This contract defines the expected public CLI failure behavior for all existing `c8volt` commands covered by the shared error-model refactor.

## Shared Failure Rules

- All existing commands must classify failures through one shared CLI error model.
- Similar failures must produce the same failure class and exit-code behavior regardless of command family.
- Failure output must remain understandable to operators and stable enough for scripts and AI agents to reason about.

## Failure Classes

- **Invalid input**: Command usage, invalid flags, mutually exclusive flags, missing dependent flags, or invalid flag values must fail before any backend request and use the shared invalid-input behavior.
- **Local precondition or configuration failure**: Missing configuration, service-construction failures, or local environment problems must be distinguishable from remote API failures.
- **Unsupported capability**: Unsupported Camunda version or unsupported operation paths must be classified predictably rather than as generic runtime failures.
- **Remote or infrastructure failure**: Remote HTTP, transport, timeout, rate-limit, or availability failures must be distinguishable from permanent caller mistakes.
- **Conflict or not found**: Known conflict and missing-resource cases must preserve dedicated non-success behavior.
- **Internal or malformed response failure**: Unexpected internal errors and malformed success payloads must not be rendered as successful or ambiguous results.

## Exit-Code Rules

- One shared exit-code mapping must be used for the failure classes in scope.
- Existing `internal/exitcode` values remain the compatibility baseline unless an explicitly documented implementation change justifies otherwise.
- `--no-err-codes` must force exit code `0` without turning the failure into a successful command outcome in message rendering.

## Output Rules

- Failure output must describe the failure consistently enough that operators can tell whether to correct input, inspect configuration, or retry later.
- Commands must not rely on one-off wording to communicate whether the remote system was reached.
- A malformed `200 OK` or equivalent success response must not produce empty or successful-looking output.

## Command Surface Rules

- The contract applies to root pre-run failures and all existing command families, including `get`, `run`, `deploy`, `cancel`, `delete`, `expect`, `walk`, `embed`, `config`, and related nested subcommands.
- Existing Cobra command structure, names, and flag propagation rules remain unchanged by this contract.

## Testable Acceptance Signals

- Equivalent failure causes in different command families return the same exit-code class.
- Invalid input failures happen before backend work and use the shared invalid-input behavior.
- Unsupported version or unsupported operation failures are distinguishable from transport or availability failures.
- `--no-err-codes` returns exit code `0` while preserving failure output.
- A representative malformed success response produces a classified failure rather than empty success output.
