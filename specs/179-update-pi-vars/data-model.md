# Data Model: Process Instance Variable Updates

## Process Instance Key

Identifies the process instance to update.

- **Value**: string key supplied by `--key` or stdin `-`.
- **Role**: Used as the Camunda `elementInstanceKey` for process-instance-scope variable updates.
- **Validation**: Must pass existing key validation and deduplication behavior.

## Variable Update Payload

Represents the JSON object supplied through `--vars`.

- **Raw Input**: CLI string value.
- **Parsed Value**: top-level JSON object.
- **Variable Names**: top-level property names that must be confirmed after mutation.
- **Variable Values**: JSON-compatible values, including nested objects, arrays, strings, numbers, booleans, and null.
- **Validation**: Missing, malformed, or non-object input fails before mutation.

## Update Target Set

Represents the unique process instances selected for one command invocation.

- **Sources**: repeated `--key` values and newline-separated stdin keys when `-` is supplied.
- **Deduplication**: Existing key merge and deduplication rules remove duplicates before mutation.
- **Worker Behavior**: Existing worker, fail-fast, and no-worker-limit settings control multi-key execution where applicable.

## Update Result

Represents the per-process-instance command outcome.

- **Key**: target process instance key.
- **Mutation Status**: accepted, failed, or unsupported.
- **Confirmation Status**: confirmed, skipped, failed, or not applicable.
- **Error**: validation, mutation, unsupported-version, or confirmation failure details where applicable.
- **Output**: Rendered independently in human and JSON output.

## Confirmation Lookup

Represents the post-mutation read-model verification path.

- **Lookup Target**: one process instance key.
- **Source**: same backend path used by `get process-instance --key <key> --with-vars`.
- **Checked Names**: only variable names supplied in the request payload.
- **Success Rule**: every requested name is visible with the requested normalized JSON value before waiter exhaustion.

## Normalized Variable Value

Represents canonical comparison between requested and observed variable values.

- **Requested Value**: parsed JSON value from `--vars`.
- **Observed Value**: returned process-instance variable value decoded from the existing variable lookup model.
- **Comparison Rule**: semantically equal JSON values match even when raw string formatting differs.

## Unsupported Version Error

Represents the explicit failure for Camunda 8.7.

- **Trigger**: update process-instance command runs against a Camunda 8.7 configuration.
- **Timing**: returned before any mutation request is attempted.
- **Output**: clear unsupported-version message consistent with existing domain/facade errors.
