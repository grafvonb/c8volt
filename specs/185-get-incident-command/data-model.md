# Data Model: Get Incident Command

## Incident Query

Represents a user request for either direct incident-key lookup or incident search/list mode.

Fields:

- `Keys`: unique incident keys from repeated `--key` flags and optional stdin `-`.
- `Mode`: keyed lookup or search/list.
- `Filters`: incident state, error type, error message substring, process context, flow-node context, and creation-time bounds.
- `OutputMode`: default, JSON, keys-only, or total.
- `MessageLimit`: optional error message truncation limit for default output.
- `Limit` and paging settings inherited from existing get/list command conventions.

Validation rules:

- Keyed lookup cannot be combined with search/list filters.
- `--total` cannot be combined with `--json` or `--keys-only`.
- `--error-message-limit` requires default incident output.
- Date filters must parse before any remote request is issued.
- State and error type values must validate before any remote request is issued.

## Incident Filter

Represents validated search criteria.

Fields:

- `State`: `active`, `pending`, `resolved`, `migrated`, `unknown`, or `all`.
- `ErrorType`: normalized generated Camunda incident error enum value.
- `ErrorMessage`: case-insensitive substring requested by the user.
- `ProcessInstanceKey`
- `RootProcessInstanceKey`
- `ProcessDefinitionKey`
- `ProcessDefinitionID`
- `FlowNodeID`
- `FlowNodeInstanceKey`
- `CreationTimeAfter`
- `CreationTimeBefore`

Validation rules:

- Default state is `active`.
- `all` means no state filter.
- Error type validation and normalization reuse `internal/services/incidentfilter`.
- Error-message matching reuses the existing case-insensitive helper semantics.

## Incident Detail

Reusable incident record rendered by the CLI and JSON output.

Existing model:

- `internal/domain.ProcessInstanceIncidentDetail`
- `c8volt/process.ProcessInstanceIncidentDetail`

Fields used by this feature:

- `IncidentKey`
- `TenantID`
- `State`
- `ErrorType`
- `ErrorMessage`
- `CreationTime`
- `ProcessInstanceKey`
- `RootProcessInstanceKey`
- `ProcessDefinitionKey`
- `ProcessDefinitionID`
- `FlowNodeID`
- `FlowNodeInstanceKey`
- `JobKey`

Relationships:

- Incident Detail belongs to one process instance context when provided by Camunda.
- Incident Detail may reference a job key; absent job keys render as `n/a` in default output.

## Incident Search Result

Represents the final results after server-side filters, local filters, paging, and command limits.

Fields:

- `Items`: ordered incident details.
- `Total`: exact count when requested.
- `UsedBackendTotal`: true only when backend total remains exact after all filters.
- `LocalFilteringApplied`: true when message filtering or version compatibility filtering was applied locally.
- `Exhausted`: true when the service paged until no more results were available before hitting an explicit limit.

Validation rules:

- Totals must be computed after local filters.
- Local filtering must not inspect only the first page.

## Incident Output View

Represents the selected rendering contract.

Default output:

- One compact row per incident.
- Includes key, tenant, state, error type, creation time, process instance key, flow node ID, flow node instance key, job key, message, and age.
- Applies `--error-message-limit` only when explicitly requested.

JSON output:

- Preserves full incident detail fields and full error messages.
- Does not truncate `errorMessage`.

Keys-only output:

- Prints incident keys only.

Total output:

- Prints only the exact numeric count.
