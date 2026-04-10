# Data Model: Relative Day-Based Process-Instance Date Shortcuts

## Relative Day Filter Input

- **Purpose**: Represents the user-facing integer shortcut supplied through the new `*-days` flags before it is translated into the canonical absolute date filter model.
- **User-visible inputs**:
  - `--start-date-after-days`
  - `--start-date-before-days`
  - `--end-date-after-days`
  - `--end-date-before-days`
- **Validation rules**:
  - Each value is optional and must be a non-negative integer.
  - A relative day input may not be combined with the corresponding absolute date flag for the same field.
  - Relative day inputs participate only in search-based selection flows.
- **Derived behavior**:
  - `after` inputs become inclusive lower bounds.
  - `before` inputs become inclusive upper bounds.
  - Day offsets are interpreted using the configured Camunda environment’s local calendar day.

## Derived Date Bound

- **Purpose**: Represents the absolute date-only boundary produced from a relative day input and passed into the existing absolute date-filter pipeline.
- **Canonical fields**:
  - `StartDateAfter`
  - `StartDateBefore`
  - `EndDateAfter`
  - `EndDateBefore`
- **Transformation rules**:
  - `N` days means today minus `N` calendar days in the configured Camunda environment.
  - Derived bounds must preserve the same inclusive semantics already used by issues `#90` and `#93`.
  - If both derived bounds exist for the same field, the lower bound must be less than or equal to the upper bound.

## Process Instance Search Request

- **Purpose**: Represents the full filter set used by `get process-instance`, `cancel process-instance`, and `delete process-instance` when operating in search mode.
- **Source models**:
  - [`c8volt/process/model.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/model.go)
  - [`internal/domain/processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/processinstance.go)
- **Relevant existing fields**:
  - `Key`
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
- **New behavior introduced by this feature**:
  - Relative day inputs are normalized into the existing absolute date fields before the filter reaches facade or service layers.
  - Existing non-date filters remain additive and continue to narrow the same result set.
  - `cancel` and `delete` continue to use explicit `--key` as an alternate mode instead of combining it with search filters.

## Command Mode

- **Purpose**: Represents the branch between direct key-based targeting and search-based selection for process-instance commands.
- **States**:
  - `Search mode`: no explicit keys supplied, so existing search filters plus derived date bounds are applied.
  - `Direct key mode`: one or more explicit keys supplied without relative day-based filters.
- **Invariants**:
  - Any explicit `--key` plus relative day-based flag combination is invalid.
  - Relative day filters affect only search mode.
  - Existing direct key workflows remain unchanged when relative day flags are absent.

## Version Capability Rule

- **Purpose**: Determines whether a search request using derived date bounds is executable for the configured Camunda version.
- **States**:
  - `v8.8`: request proceeds using the existing native inclusive date filter mapping.
  - `v8.7`: request fails with the repository-native not-implemented error when any derived date bound is present.
- **Invariants**:
  - Requests without relative day flags keep current behavior on both versions.
  - Version-specific handling remains inside the existing versioned process-instance service split.

## Process Instance Result Set

- **Purpose**: Represents the returned or selected process instances after all supported filters and command-mode rules are applied.
- **Relevant existing fields**:
  - `Key`
  - `StartDate`
  - `EndDate`
  - `State`
  - `ParentKey`
  - `TenantId`
- **Behavioral constraints introduced by this feature**:
  - Derived bounds are inclusive at both boundaries.
  - Instances with missing `endDate` are excluded whenever relative end-day filters are present.
  - Search-based cancel/delete actions operate on the same selected process-instance set produced by the shared filter pipeline.
