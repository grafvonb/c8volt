# Data Model: Extend Process-Instance Management Date Filters

## Management Search Request

- **Purpose**: Represents the full set of search-only constraints that `c8volt cancel process-instance` and `c8volt delete process-instance` may use to discover target instances when explicit keys are not provided.
- **Source models**:
  - [`c8volt/process/model.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/model.go)
  - [`internal/domain/processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/processinstance.go)
- **Relevant fields**:
  - `BpmnProcessId`
  - `ProcessVersion`
  - `ProcessVersionTag`
  - `ProcessDefinitionKey`
  - `State`
  - `ParentKey`
  - `StartDateAfter`
  - `StartDateBefore`
  - `EndDateAfter`
  - `EndDateBefore`
- **Validation rules**:
  - Search-only filters are valid only when no explicit process-instance keys are supplied.
  - All date bounds are optional and must be valid date-only input.
  - `StartDateAfter <= StartDateBefore` when both are present.
  - `EndDateAfter <= EndDateBefore` when both are present.
  - Date bounds remain additive alongside existing search filters.

## Management Command Mode

- **Purpose**: Distinguishes between the two target-selection paths used by the affected commands.
- **Modes**:
  - `DirectKeyMode`: explicit `--key` values identify the target process instances directly.
  - `SearchMode`: target process instances are discovered by search filters and then passed into the existing cancel/delete workflows.
- **Invariants**:
  - `DirectKeyMode` and date-filter flags are mutually exclusive.
  - `SearchMode` may combine date filters with existing state and process-definition filters.
  - The management action only begins after target selection succeeds.

## Date Filter Bound

- **Purpose**: Represents a single inclusive day-based lower or upper bound supplied by the CLI for process-instance `startDate` or `endDate`.
- **User-visible inputs**:
  - `--start-date-after`
  - `--start-date-before`
  - `--end-date-after`
  - `--end-date-before`
- **Derived behavior**:
  - `after` maps to an inclusive lower bound.
  - `before` maps to an inclusive upper bound.
  - Bounds follow the configured Camunda environment’s local calendar day semantics established in issue `#90`.
- **Version behavior**:
  - v8.8 uses the existing native search mapping.
  - v8.7 rejects any search request containing one or more date bounds as not implemented.

## Selected Process Instance Set

- **Purpose**: The collection of process instances returned from search mode before the management action is executed.
- **Relevant existing fields**:
  - `Key`
  - `StartDate`
  - `EndDate`
  - `State`
  - `ParentKey`
  - `TenantId`
- **Behavioral constraints**:
  - Boundary dates are inclusive.
  - Instances with missing `endDate` are excluded whenever end-date filters are present.
  - Existing non-date filters continue to narrow the same selected set.
  - An empty selected set triggers the command’s existing no-target-found failure behavior.

## Command Validation Outcome

- **Purpose**: Represents the pre-execution validation result for cancel/delete command inputs.
- **Successful state**:
  - Inputs use either explicit keys or search-only filters, but not both for date-filtered usage.
  - Date values parse as `YYYY-MM-DD`.
  - Inclusive range validation passes independently for start and end date bounds.
- **Failure states**:
  - Invalid date format.
  - Inverted start-date range.
  - Inverted end-date range.
  - Explicit `--key` combined with one or more date-filter flags.
  - Unsupported version behavior returned for a date-filtered search on v8.7.
