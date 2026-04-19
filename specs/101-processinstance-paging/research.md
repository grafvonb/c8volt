# Research: Version-Aware Process-Instance Paging and Overflow Handling

## Decision 1: Use one shared config key for the default process-instance page size

- **Decision**: Add one shared config value for the default page size used by `get process-instance`, search-based `cancel process-instance`, and search-based `delete process-instance`, with `--count` continuing to override it per execution.
- **Rationale**: The clarified spec requires one shared default across the three commands. The current config model already centralizes command-wide settings under `config.App`, so one additional shared setting keeps behavior predictable and avoids per-command drift.
- **Alternatives considered**:
  - Separate config keys per command: rejected because it adds operator overhead and conflicts with the clarified requirement for consistent paging behavior.
  - No config key and rely only on `--count`: rejected because the issue explicitly requires a configurable default.

## Decision 2: Keep paging orchestration in the shared command-layer process-instance search seam

- **Decision**: Implement the page loop at the command layer, extending the shared process-instance search helpers currently rooted in [`cmd/get_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go) and reused by `cancel` and `delete`.
- **Rationale**: The command layer already owns `--count`, `--auto-confirm`, prompts, keyed-vs-search mode selection, and user-facing output. Keeping paging decisions there preserves consistent CLI behavior across the three commands without pushing interactive logic into facade or service packages.
- **Alternatives considered**:
  - Implement paging loops independently in each command: rejected because it would create three variants of the same operator behavior.
  - Move prompts/continuation into facade or service code: rejected because those layers should stay transport- and command-agnostic.

## Decision 3: Use native v8.8 page metadata as the preferred overflow signal

- **Decision**: Use the v8.8 generated `ProcessInstanceSearchQueryResult.Page` metadata, especially `TotalItems`, `HasMoreTotalItems`, and pagination cursors, as the primary overflow signal for Camunda `8.8`.
- **Rationale**: The generated v8.8 client already exposes structured page metadata and cursor/offset pagination support, making this the cleanest and most truthful “API-native indicator” available in the repository today.
- **Alternatives considered**:
  - Infer overflow only from `len(items) == limit`: rejected because it throws away metadata the API already provides and cannot distinguish exact-boundary final pages as confidently.
  - Perform unconditional look-ahead fetches even on `8.8`: rejected because the native metadata should reduce unnecessary extra requests.

## Decision 4: Treat v8.7 overflow detection as a fallback path and stop/warn when it stays indeterminate

- **Decision**: Implement a version-appropriate fallback strategy for `8.7` because the current Operate search response type does not expose the same page metadata as v8.8, and stop with a warning if that fallback still cannot prove whether more matches remain.
- **Rationale**: The issue requires version-aware behavior and explicitly forbids silent continuation or silent truncation. The current `internal/services/processinstance/v87` seam can decide how much continuation evidence the Operate API provides; if it cannot provide a trustworthy answer, the safest CLI behavior is to stop and warn.
- **Alternatives considered**:
  - Assume no overflow when a v8.7 page is full: rejected because it reintroduces silent truncation risk.
  - Always auto-continue while pages remain full: rejected because it can overrun intended scope without trustworthy confirmation.

## Decision 5: Treat user-declined continuation as a successful partial completion

- **Decision**: When the user declines a continuation prompt after one or more processed pages, stop normally and report that only the processed pages were completed.
- **Rationale**: This matches the explicit clarification, preserves operator intent, and avoids misclassifying an intentional stop as an error while still making remaining matches visible.
- **Alternatives considered**:
  - Treat the decline as a command failure: rejected because intentional operator stops are not invalid usage.
  - Stop silently with no partial-completion summary: rejected because users need an explicit record of what was and was not processed.

## Decision 6: Report both current-page and cumulative counts in paging output

- **Decision**: Include both the current-page count and the cumulative processed count in operator-facing output for all three commands.
- **Rationale**: The clarification requires running-total visibility, which is especially important for multi-page bulk actions and partial-completion reporting.
- **Alternatives considered**:
  - Show only the current-page count: rejected because it obscures total impact after multiple pages.
  - Show only the cumulative count: rejected because it weakens per-page visibility and troubleshooting.

## Decision 7: Keep `get process-instance` on the same continuation model as search-based `cancel` and `delete`

- **Decision**: `get process-instance` prompts between pages unless `--auto-confirm` is set, just like the search-based write commands.
- **Rationale**: The clarified spec prefers one consistent continuation model across the three affected commands, reducing operator surprise and documentation complexity.
- **Alternatives considered**:
  - Make `get` auto-continue by default: rejected because it would create a different mental model for one member of the same feature set.
  - Make `get` stop after one page with no continuation option: rejected because it would leave the main “no silent truncation” problem unsolved.

## Decision 8: Extend tests at the command, facade, and service seams already used by process-instance work

- **Decision**: Add or update command tests for page-size resolution, prompt/auto-confirm transitions, partial completion, and output summaries; facade/service tests should cover version-specific overflow signals and continuation metadata propagation.
- **Rationale**: The observable behavior changes at the command seam, but the correctness of version-aware overflow handling depends on facade/service support as well.
- **Alternatives considered**:
  - Command tests only: rejected because they would not prove the version-specific overflow signals are surfaced correctly.
  - Service tests only: rejected because prompts, counts, and partial-completion semantics are command behavior.

## Decision 9: Update both hand-written and generated documentation

- **Decision**: Update `README.md` and command help/examples, then regenerate `docs/cli/` via the repository’s documentation commands.
- **Rationale**: The constitution requires documentation to match user-visible behavior, and the affected commands already have generated reference pages.
- **Alternatives considered**:
  - Update only Cobra help text: rejected because README and generated CLI docs are also part of the user-facing contract.
  - Hand-edit generated CLI docs: rejected because repository guidance requires regeneration instead.
