# Data Model: Define Non-Interactive Automation Mode

## Automation Mode Flag

- **Purpose**: Represents the single dedicated CLI flag that opts a caller into the canonical non-interactive automation contract.
- **Fields**:
  - `name`: planned as `automation`
  - `scope`: root persistent flag
  - `enabled`: boolean effective state for the current command execution
- **Invariants**:
  - The flag is opt-in and does not change current human behavior unless explicitly set.
  - The flag is the canonical entry point for the automation contract.
  - The flag does not imply `--no-wait`.

## Automation Support Status

- **Purpose**: Indicates whether a command explicitly supports automation-mode execution.
- **Allowed values**:
  - `full`
  - `unsupported`
- **Invariants**:
  - `full` means the command has defined automation-mode prompt, output, and failure behavior.
  - `unsupported` means the command must reject `--automation` explicitly.
  - Shared result-envelope support and automation-mode support are related but distinct.

## Automation Invocation

- **Purpose**: Captures the effective execution context when a caller uses the automation flag.
- **Fields**:
  - `automation`: whether the automation flag is active
  - `json`: whether the shared machine-readable result surface is requested
  - `noWait`: whether the caller explicitly requested accepted/not-yet-complete execution
  - `quiet`: whether output suppression is requested
  - `verbose`: whether extra progress or diagnostics are requested
- **Invariants**:
  - `automation=true` never implies `noWait=true`.
  - `automation=true` with `json=true` must keep stdout machine-safe.
  - `automation=true` with `noWait=true` uses `accepted` semantics for supported state-changing commands.

## Automation Capability Record

- **Purpose**: Extends command discovery with automation-mode truthfulness for one command path.
- **Fields**:
  - `path`: canonical Cobra command path
  - `automationSupport`: `full` or `unsupported`
  - `automationNotes`: optional explanation when support is limited by scope or intentionally rejected
- **Invariants**:
  - Discovery must not claim a command supports automation mode unless runtime behavior matches the contract.
  - Unsupported commands remain discoverable.

## Prompt Policy

- **Purpose**: Defines how interactive gates behave under automation mode.
- **Fields**:
  - `confirmation`: whether a supported confirmation prompt is implicitly accepted
  - `continuation`: whether a supported paging continuation prompt is implicitly accepted
  - `unsupportedBehavior`: explicit rejection behavior for unsupported commands
- **Invariants**:
  - Supported prompts are implicitly accepted in automation mode.
  - Unsupported commands reject automation mode instead of falling back to interactive behavior.
  - Human-mode behavior stays unchanged when automation mode is not active.

## Output Channel Policy

- **Purpose**: Defines how machine-readable output and human-oriented diagnostics are separated.
- **Fields**:
  - `stdoutRole`: machine-readable result channel when JSON is requested
  - `stderrRole`: log/progress/error diagnostic channel
  - `suppressibleDiagnostics`: human-oriented progress or informational output that may be omitted
- **Invariants**:
  - With `automation=true` and `json=true`, stdout contains only the machine-readable result.
  - Human-oriented logs and progress do not pollute stdout in automation JSON runs.
  - Non-JSON human-oriented output remains available outside the machine-readable path.

## Execution Outcome State

- **Purpose**: Reuses the existing top-level result-envelope vocabulary for automation-mode runs.
- **Allowed values**:
  - `succeeded`
  - `accepted`
  - `invalid`
  - `failed`
- **Invariants**:
  - `succeeded` is reserved for confirmed completion.
  - `accepted` is used only when a supported command intentionally returns before confirmation, such as with `--no-wait`.
  - `invalid` and `failed` continue to align with `ferrors`-based exit behavior.

## Unsupported Automation Invocation

- **Purpose**: Represents a rejected attempt to use automation mode on a command that has not opted in.
- **Fields**:
  - `command`: canonical command path
  - `class`: repository-native failure class, expected to map to unsupported execution
  - `message`: actionable explanation for the caller
  - `suggestion`: optional remediation, such as removing `--automation` or choosing a supported command
- **Invariants**:
  - The rejection is explicit and immediate.
  - The rejection does not fall back to interactive behavior.
  - JSON mode should still return a machine-readable `failed` envelope for the rejection.

## Representative Command Coverage

- **Purpose**: Identifies the initial rollout boundary for automation-mode implementation and tests.
- **Current variants**:
  - `capabilities`
  - `get process-instance`
  - `run process-instance`
  - `deploy process-definition`
  - `delete process-instance`
  - `cancel process-instance`
  - `expect process-instance`
  - `walk process-instance`
- **Invariants**:
  - At least one representative read flow and one representative write flow must be covered.
  - Unsupported commands may remain in the discovery surface as long as rejection behavior is explicit.
  - The rollout boundary stays small enough for incremental implementation and testing.
