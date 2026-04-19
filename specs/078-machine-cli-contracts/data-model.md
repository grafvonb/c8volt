# Data Model: Define Machine-Readable CLI Contracts

## Capability Document

- **Purpose**: Represents the full response from the dedicated top-level discovery command.
- **Fields**:
  - `command`: canonical discovery command identifier
  - `version`: contract version string for machine consumers
  - `commands`: ordered list of command capability records
- **Invariants**:
  - The document is available without scraping human help text.
  - Nested commands are represented explicitly rather than inferred by the caller.
  - Every listed command includes contract support status.

## Command Capability Record

- **Purpose**: Describes one top-level or nested command path that automation can invoke.
- **Fields**:
  - `path`: canonical command path, such as `get process-instance`
  - `aliases`: supported aliases for the path
  - `summary`: concise command description
  - `mutation`: whether the command is `read_only` or `state_changing`
  - `contractSupport`: `full`, `limited`, or `unsupported`
  - `outputModes`: supported output-mode records
  - `flags`: supported flag records
  - `children`: nested command capability records when applicable
- **Invariants**:
  - `path` matches the Cobra command tree.
  - `mutation` must stay consistent with the observable behavior of the command.
  - `contractSupport` must not overstate actual envelope support.

## Flag Contract

- **Purpose**: Captures one flag that matters to machine consumers for discovery and invocation.
- **Fields**:
  - `name`: long flag name
  - `shorthand`: optional short form
  - `type`: flag value type
  - `required`: whether the flag is required
  - `repeated`: whether the flag accepts multiple values
  - `description`: concise command-safe description
- **Invariants**:
  - Discovery metadata must reflect the real Cobra flag definition.
  - Root flags inherited by a command remain discoverable for that command path.

## Output Mode Contract

- **Purpose**: Describes one output mode exposed by a command.
- **Fields**:
  - `name`: canonical mode name such as `json`, `one-line`, `keys-only`, or `tree`
  - `supported`: whether the mode is supported on that command
  - `machinePreferred`: whether automation should prefer the mode
  - `notes`: optional limitations or caveats
- **Invariants**:
  - `json` is the machine-preferred mode when the shared machine contract is fully supported.
  - Unsupported modes must be reported explicitly when they affect automation behavior.

## Result Envelope

- **Purpose**: Represents the shared top-level machine-readable contract returned by supported commands.
- **Fields**:
  - `outcome`: one of `succeeded`, `accepted`, `invalid`, or `failed`
  - `class`: optional detailed failure or semantic class, derived from repository-native behavior such as `ferrors.Class`
  - `command`: canonical command path that produced the result
  - `payload`: command-specific JSON payload
  - `detail`: optional machine-readable error or validation details
- **Invariants**:
  - The envelope shape is shared across all commands with `contractSupport=full`.
  - `payload` preserves the command family’s domain-specific shape.
  - The envelope must not contradict the process exit code.

## Outcome Category

- **Purpose**: Gives automation one small, stable vocabulary for top-level command results.
- **Allowed values**:
  - `succeeded`
  - `accepted`
  - `invalid`
  - `failed`
- **Invariants**:
  - `succeeded` is reserved for confirmed successful completion.
  - `accepted` is reserved for successfully requested but not yet confirmed state-changing work.
  - `invalid` is reserved for caller-correctable input and validation failures.
  - `failed` covers non-validation execution failures, including local preconditions, unsupported cases, conflicts, timeouts, unavailable services, malformed responses, and internal errors.

## Failure Detail

- **Purpose**: Carries structured machine-readable detail for `invalid` and `failed` outcomes.
- **Fields**:
  - `message`: concise human-readable detail
  - `class`: repository-native detailed class, such as `invalid_input` or `unavailable`
  - `suggestion`: optional caller action when a correction is possible
- **Invariants**:
  - Validation failures should provide actionable correction guidance when available.
  - Failure detail supplements but does not replace the process exit code.

## Contract Support Status

- **Purpose**: Indicates how much of the shared machine contract a command supports today.
- **Allowed values**:
  - `full`
  - `limited`
  - `unsupported`
- **Invariants**:
  - `full` means the command exposes the shared result envelope in machine mode.
  - `limited` means some machine-readable behavior exists, but the full shared contract is not yet implemented.
  - `unsupported` means the command is discoverable but should not be treated as part of the shared machine contract yet.

## Process-Level Signal

- **Purpose**: Represents the existing command exit code that remains authoritative for scripts and automation.
- **Fields**:
  - `exitCode`: integer exit status
  - `resolvedBy`: shared exit resolver such as `ferrors.ResolveExitCode`
- **Invariants**:
  - The result envelope must align with the exit code.
  - The feature must not redefine process success or failure independently from the exit code.

## Representative Payload Families

- **Purpose**: Groups the current command-specific payloads that will sit inside the shared result envelope.
- **Current variants**:
  - `process.ProcessInstances` and `process.ProcessInstance`
  - `process.ProcessDefinitions` and `process.ProcessDefinition`
  - `process.CancelReports`
  - `process.DeleteReports`
  - `process.StateReports`
  - `resource.Resource`
  - `resource.ProcessDefinitionDeployment`
  - build/version info maps
- **Invariants**:
  - The envelope may wrap different payload families, but the top-level machine contract stays stable.
  - Existing JSON tags on the payload models remain authoritative for payload field names.
