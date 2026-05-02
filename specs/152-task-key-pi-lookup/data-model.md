# Data Model: Resolve Process Instance From User Task Key

## User Task Key

- **Type**: User-supplied command selector value.
- **Validation**: Each value must be non-empty when `--has-user-tasks` is provided. `--has-user-tasks` may be repeated, and is mutually exclusive with `--key`, stdin key input via `-`, search filters, `--total`, and `--limit`.
- **Lifecycle**: Parsed from CLI input, each unique user task key is resolved through version-specific user-task service, then discarded after the owning process-instance key is extracted.

## Resolved User Task

- **Type**: Native Camunda v2 user-task search result.
- **Fields Used**:
  - `processInstanceKey`: required owning process-instance key.
- **Validation**: Missing or empty `processInstanceKey` is a resolution error.
- **Lifecycle**: Returned by the 8.8/8.9 tenant-aware native search and converted into a domain-level result that exposes the owning process-instance key.

## Owning Process Instance Key

- **Type**: Process-instance key string compatible with existing process-instance lookup.
- **Validation**: Must be non-empty after task resolution.
- **Lifecycle**: Passed into the same keyed process-instance lookup path used by `get pi --key`.

## Task-Key Lookup Request

- **Type**: Command invocation state.
- **Fields**:
  - `taskKeys`: user task key selectors.
  - `renderOptions`: existing single process-instance output options.
  - `version`: configured Camunda version.
- **State Transitions**:
  - `requested` -> `unsupported` on 8.7.
  - `requested` -> `task-not-found` when the user task does not exist.
  - `requested` -> `resolution-error` when the user task lacks an owning process-instance key.
  - `requested` -> `process-instance-lookup` when the owning key is available.
  - `process-instance-lookup` -> existing success or existing process-instance error behavior.

## Selector Conflict

- **Type**: Invalid command invocation.
- **Conflict Inputs**:
  - `--has-user-tasks` with `--key`
  - `--has-user-tasks` with stdin key input via `-`
  - `--has-user-tasks` with process-instance search filters
  - `--has-user-tasks` with `--total`
  - `--has-user-tasks` with `--limit`
- **Behavior**: Fail through repository-standard invalid argument handling before API resolution.
