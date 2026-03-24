# Data Model: Review and Refactor CLI Error Code Usage

## CLI Error Class

- **Purpose**: Represents the bounded machine-facing and operator-facing classification for a command failure.
- **Fields**:
  - Stable class name
  - Semantic meaning
  - Default exit code
  - Retry guidance
  - Operator message intent
- **Validation Rules**:
  - Must stay small enough to remain understandable across the whole CLI.
  - Must distinguish caller mistakes, unsupported operations, local setup problems, remote failures, and internal failures.
  - Must preserve compatibility with existing `internal/exitcode` values unless the plan explicitly justifies a change.
- **Relationships**:
  - Produced by CLI Error Mapping.
  - Drives Exit Behavior Policy and Operator-Facing Failure Message.

## CLI Error Mapping

- **Purpose**: Converts command-level, service-level, domain-level, and transport-level errors into one CLI Error Class.
- **Fields**:
  - Source error or sentinel
  - Matching precedence
  - Resulting CLI error class
  - Optional wrapping or message prefix
- **Validation Rules**:
  - Must prefer sentinel and typed-error matching over raw string matching.
  - Must handle representative command validation errors, `services.ErrUnknownAPIVersion`, domain HTTP or API errors, and malformed responses.
  - Must not leave the same source failure mapped differently across commands.
- **Relationships**:
  - Consumes errors from `cmd/`, `internal/services`, `internal/domain`, and remote API handling.
  - Produces CLI Error Class instances used by `c8volt/ferrors`.

## Exit Behavior Policy

- **Purpose**: Defines the shared numeric exit outcome for each CLI Error Class.
- **Fields**:
  - Exit code constant
  - Associated CLI error class
  - `--no-err-codes` override behavior
- **Validation Rules**:
  - Must define one shared exit-code mapping for the failure classes in scope.
  - Must keep `--no-err-codes` as a final override to exit `0` without bypassing classification.
  - Must remain stable enough for scripting and future AI-oriented tooling.
- **Relationships**:
  - Consumes CLI Error Class.
  - Used by `ferrors.HandleAndExit`.

## Operator-Facing Failure Message

- **Purpose**: Represents the visible CLI failure output rendered to stderr or logs for operators.
- **Fields**:
  - Human-readable summary
  - Optional normalized class label or category signal
  - Wrapped source context
  - Suggested next-step cue when applicable
- **Validation Rules**:
  - Must stay actionable and understandable.
  - Must keep similar failures worded consistently across commands.
  - Must not imply success or ambiguity when a failure is classified.
- **Relationships**:
  - Derived from CLI Error Class plus command context.
  - Rendered by the shared CLI error layer.

## Command Failure Context

- **Purpose**: Captures where a failure originated in the command lifecycle so classification and message rendering stay consistent.
- **Fields**:
  - Command path
  - Failure stage such as pre-run configuration, flag validation, service creation, remote operation, rendering, or output write
  - Relevant resource or operation context
- **Validation Rules**:
  - Must be lightweight and derived from existing command flow.
  - Must help distinguish local precondition problems from remote-system failures.
- **Relationships**:
  - Supplies context to Operator-Facing Failure Message.
  - Helps choose the right CLI Error Class in ambiguous cases.

## CLI Command Surface

- **Purpose**: The complete set of existing CLI commands that must conform to the shared failure model.
- **Fields / Responsibilities**:
  - Root command and persistent pre-run path
  - Read-only command families such as `get`
  - State-changing command families such as `run`, `deploy`, `cancel`, and `delete`
  - Validation-heavy or expectation-oriented command families such as `expect`, `walk`, `embed`, and `config`
- **Validation Rules**:
  - All existing CLI commands must route failures through the shared model.
  - No family may silently keep incompatible ad hoc failure behavior.
- **Relationships**:
  - Produces source failures consumed by CLI Error Mapping.

## Failure Source

- **Purpose**: Represents the origin category for a failure before it is normalized into a CLI Error Class.
- **Fields**:
  - Caller input
  - Local configuration or environment
  - Unsupported capability or version
  - Remote API or transport
  - Internal implementation
  - Malformed success response
- **Validation Rules**:
  - Each representative source must have at least one stable mapping path into a CLI Error Class.
  - Retryable and permanent failures must not be conflated when the source meaning differs.
- **Relationships**:
  - Input to CLI Error Mapping.

## State Notes

- This feature does not add persistent storage or long-lived business entities.
- Relevant state transitions are classification-only: raw error source -> normalized CLI error class -> rendered operator message -> exit behavior.
- The compatibility-critical override path is: classified failure -> `--no-err-codes` -> exit code `0` with failure output still rendered.
