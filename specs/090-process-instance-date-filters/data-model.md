# Data Model: Day-Based Process Instance Date Filters

## Process Instance Search Request

- **Purpose**: Represents the full set of user-supplied constraints for `c8volt get process-instance` list/search behavior.
- **Source models**:
  - [`c8volt/process/model.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/model.go)
  - [`internal/domain/processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/processinstance.go)
- **Existing fields**:
  - `Key`
  - `BpmnProcessId`
  - `ProcessVersion`
  - `ProcessVersionTag`
  - `ProcessDefinitionKey`
  - `State`
  - `ParentKey`
- **New fields to add**:
  - `StartDateAfter`
  - `StartDateBefore`
  - `EndDateAfter`
  - `EndDateBefore`
- **Validation rules**:
  - New fields are optional and only valid for list/search usage.
  - Each value must be a valid date-only input.
  - `StartDateAfter <= StartDateBefore` when both are present.
  - `EndDateAfter <= EndDateBefore` when both are present.
  - New date fields remain additive alongside existing search filters.

## Date Filter Bound

- **Purpose**: Represents an inclusive day-granularity lower or upper bound supplied by the CLI.
- **User-visible inputs**:
  - `--start-date-after`
  - `--start-date-before`
  - `--end-date-after`
  - `--end-date-before`
- **Derived behavior**:
  - `after` maps to inclusive lower bound.
  - `before` maps to inclusive upper bound.
  - Bounds are interpreted in the configured Camunda environment’s local calendar day.
- **Transformation rules**:
  - For v8.8, convert day inputs into datetime filters suitable for generated-client `$gte`/`$lte` comparisons.
  - For v8.7, the presence of any date bound short-circuits to not implemented.

## Version Capability Rule

- **Purpose**: Determines whether a search request using the new date fields is executable.
- **States**:
  - `v8.8`: request proceeds with native date filtering.
  - `v8.7`: request fails with not-implemented error when any date field is present.
- **Invariants**:
  - Requests without new date fields keep current behavior on both versions.
  - Capability handling remains inside the existing versioned process-instance service split.

## Process Instance Result Set

- **Purpose**: Returned process instances after all supported filters are applied.
- **Relevant existing fields**:
  - `Key`
  - `StartDate`
  - `EndDate`
  - `State`
  - `ParentKey`
  - `TenantId`
- **Behavioral constraints introduced by this feature**:
  - Instances with missing `endDate` are excluded whenever end-date filters are present.
  - Boundary dates are inclusive.
  - Existing non-date filters still narrow the same result set.

## Command Validation Outcome

- **Purpose**: Represents the pre-execution validation result for new CLI inputs.
- **Successful state**:
  - All date values parse.
  - Range checks pass.
  - `--key` is not combined with any search-only date filter.
- **Failure states**:
  - Invalid date format.
  - Inverted date range for start date or end date.
  - Direct lookup attempted with search-only date filters.
  - Unsupported version path for any date-filtered search on v8.7.
