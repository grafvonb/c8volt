# Data Model: Version-Aware Process-Instance Paging and Overflow Handling

## Process-Instance Page Size

- **Purpose**: Represents the effective page size for a single search-based process-instance command execution.
- **Sources**:
  - Shared config key under `app`
  - `--count` command-line override
  - Fallback constant `1000`
- **Resolution rules**:
  - `--count` takes precedence for the current command execution.
  - If `--count` is absent, the shared config value is used.
  - If neither is available or valid, the effective page size is `1000`.
- **Validation rules**:
  - Must be a positive integer.
  - Must continue to respect any server-enforced maximum size.

## Process-Instance Search Page

- **Purpose**: Represents one fetched page of matching process instances before the command decides whether to continue.
- **Source models**:
  - [`c8volt/process/api.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/api.go)
  - [`internal/services/processinstance/api.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/api.go)
- **Core data**:
  - Matching process instances for the current page
  - Effective page size
  - Current-page item count
  - Cumulative processed count
- **Behavioral rules**:
  - A page can be terminal even when its size equals the requested page size if the version-specific overflow signal says no more matches remain.
  - A page can end in an indeterminate-overflow stop/warn state when the version fallback cannot prove whether more matches exist.

## Overflow Signal

- **Purpose**: Represents the command’s version-aware answer to the question “are there more matching items beyond this page?”
- **States**:
  - `NoMore`: current page is known to be the final page
  - `HasMore`: additional matching items are known to remain
  - `Indeterminate`: additional matching items may remain, but the command cannot prove it after the version-specific fallback
- **Version behavior**:
  - `8.8`: derive from native search page metadata exposed by the generated Camunda client
  - `8.7`: derive from the service-layer fallback strategy because the current Operate response type does not expose equivalent page metadata
- **Invariant**:
  - `Indeterminate` must stop the command with a warning rather than silently finish or silently continue.

## Continuation State

- **Purpose**: Represents what the command does after each completed page when the overflow signal is evaluated.
- **States**:
  - `Prompt`: ask the user whether to continue because more matches are known to remain and `--auto-confirm` is false
  - `AutoContinue`: fetch/process the next page because more matches are known to remain and `--auto-confirm` is true
  - `Completed`: stop because no more matches remain
  - `PartialComplete`: stop because the user declined continuation after one or more pages
  - `WarningStop`: stop because overflow stayed indeterminate after the version-specific fallback
- **Invariant**:
  - `PartialComplete` is a non-error outcome and must retain the counts for already processed pages.

## Process-Instance Paging Summary

- **Purpose**: Represents the operator-facing status information emitted after each page and at the end of the command.
- **Required fields**:
  - Effective page size used
  - Current-page item count
  - Cumulative processed count
  - Whether more matching items remain
  - Whether the command is prompting, auto-continuing, partially complete, complete, or stopping with a warning
- **Scope**:
  - Applies to `get process-instance`
  - Applies to search-based `cancel process-instance`
  - Applies to search-based `delete process-instance`

## Command Mode

- **Purpose**: Preserves the distinction between direct key-based targeting and search-based selection.
- **States**:
  - `Search mode`: paging behavior applies
  - `Direct key mode`: existing non-paged behavior applies
- **Invariants**:
  - Direct `--key` workflows stay unchanged.
  - Paging applies only when the command is operating on search results.

## Version Paging Capability

- **Purpose**: Captures the repository-visible difference between the two supported API versions.
- **States**:
  - `v8.8 native metadata`: search results include page metadata that can drive overflow decisions directly
  - `v8.7 fallback`: search results require a service-side fallback strategy to infer whether more matches remain
- **Invariant**:
  - The command must never describe `8.9` as supported by this feature.
