# Data Model: Process Instance Incident Expectation

## Process Instance Selection

- **Purpose**: Identifies the process instances monitored by `expect process-instance`.
- **Fields**:
  - `keys`: one or more process-instance keys supplied by `--key` or stdin `-`.
  - `source`: direct flag input or stdin pipeline.
- **Validation**:
  - At least one key must be present after merging flag and stdin inputs.
  - Existing key validation and deduplication rules remain authoritative.

## Process Instance

- **Purpose**: Represents the observed process-instance record evaluated by the expectation command.
- **Fields**:
  - `key`: stable process-instance key.
  - `state`: lifecycle state used by existing state expectations.
  - `incident`: boolean marker exposed by the domain/public process-instance model.
- **State Rules**:
  - Present process instances can satisfy incident expectations.
  - Missing process instances can satisfy `--state absent` only through existing absent semantics.
  - Missing process instances never satisfy `--incident false`.

## Expectation Request

- **Purpose**: Captures the requested conditions for each selected process instance.
- **Fields**:
  - `states`: zero or more requested process-instance states.
  - `incident`: optional requested incident boolean.
- **Validation**:
  - At least one of `states` or `incident` is required.
  - `incident` accepts exactly `true` or `false` when provided.
  - When both fields are present, both must match before success.

## Expectation Report

- **Purpose**: Describes the result of waiting for a selected process instance.
- **Fields**:
  - `key`: process-instance key.
  - `state`: observed state when available.
  - `incident`: observed incident marker when available.
  - `ok`: whether all requested expectations were satisfied.
  - `status`: human-readable status suitable for logs and command output.
- **Lifecycle**:
  - Starts as waiting while any requested condition is unmet.
  - Becomes successful only when every selected process instance satisfies all requested expectations.
  - Becomes failed through existing timeout, context cancellation, invalid input, or backend error paths.
