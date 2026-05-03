# Data Model: Tasklist V1 Fallback For Task-Key Process-Instance Lookup

## Task-Key Lookup Request

- **Purpose**: Represents one invocation of `get process-instance` / `get pi` that supplies one or more `--has-user-tasks` task keys.
- **Fields**:
  - `taskKeys`: one or more positive decimal user task keys.
  - `effectiveTenant`: optional configured tenant context used by supported lookup paths.
  - `outputMode`: existing command rendering mode such as human, JSON, or keys-only.
- **Validation**:
  - Each task key must remain non-empty and positive decimal.
  - `--has-user-tasks` remains mutually exclusive with `--key`, stdin key input, process-instance search filters, `--total`, and `--limit`.
- **Lifecycle**: Parsed by command validation, resolved by the process facade through the versioned user-task service, then converted to process-instance keys for existing process-instance lookup.

## Primary User Task Lookup

- **Purpose**: Represents the existing Camunda v2 user-task lookup attempt for each task key.
- **Fields**:
  - `taskKey`: requested task key.
  - `tenantFilter`: optional effective tenant constraint.
  - `processInstanceKey`: owning process-instance key when resolution succeeds.
- **Validation**:
  - A successful result must include a usable `processInstanceKey`.
  - Empty result maps to a fallback-eligible not-found outcome.
  - Non-not-found errors must remain terminal.
- **Lifecycle**: Attempted first for every task key on Camunda 8.8 and 8.9.

## Fallback Task Lookup

- **Purpose**: Represents the Tasklist V1 lookup attempt used only after the primary lookup misses.
- **Fields**:
  - `taskKey`: requested task key.
  - `tenantFilter`: optional effective tenant constraint when supported by the fallback request.
  - `processInstanceKey`: owning process-instance key extracted from a matching fallback task.
  - `implementation`: optional upstream task implementation marker, useful for legacy compatibility assertions.
- **Validation**:
  - Fallback is allowed only after primary lookup returns not found or empty result.
  - A fallback miss produces final not-found only after the primary lookup also missed.
  - A fallback success must include exactly one matching task and a usable process-instance key.
  - Fallback auth, config, malformed response, network, and server failures are terminal errors.
- **Lifecycle**: Attempted per task key only when eligible, then discarded after the owning process-instance key is extracted.

## Owning Process Instance Key

- **Purpose**: The process-instance identifier obtained from either lookup path and passed into existing process-instance lookup.
- **Fields**:
  - `key`: positive decimal process-instance key.
  - `source`: primary lookup or fallback lookup, used internally for test assertions and logging.
- **Validation**:
  - Must be non-empty before process-instance lookup starts.
  - Must flow through existing keyed process-instance lookup so output and error behavior stay consistent.
- **Lifecycle**: Produced by user-task resolution and consumed by the existing process-instance client path.

## Resolution Outcome

- **Purpose**: Describes the observable result of resolving one task key.
- **States**:
  - `primary-resolved`: primary lookup found the task and fallback was not called.
  - `fallback-resolved`: primary lookup missed and fallback lookup found the task.
  - `not-found`: both lookup paths missed.
  - `resolution-error`: a lookup result lacked a usable process-instance key or returned ambiguous data.
  - `terminal-error`: auth, config, malformed response, network, or server error that must not be masked.
  - `unsupported-version`: Camunda 8.7 rejected the lookup before any lookup path was attempted.
- **Validation**:
  - Only `primary-resolved` and `fallback-resolved` may proceed to process-instance rendering.
  - `not-found`, `resolution-error`, `terminal-error`, and `unsupported-version` must fail clearly.
