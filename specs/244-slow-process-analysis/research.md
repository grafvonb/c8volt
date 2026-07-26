# Research: Slow Process Instance Analysis

## Decision: Add slow-run analysis as an ops facade and internal ops service capability

**Rationale**: The command is an operational workflow that combines process-instance selection, runtime element enrichment, timing calculations, filtering, and output preparation. Keeping the orchestration in `internal/services/ops` preserves the command/facade/service layering while matching the existing ops workflow package.

**Alternatives considered**:

- Put orchestration in `cmd`: rejected because command code must not own backend workflow mechanics, pagination loops, lookup orchestration, or comparison calculations.
- Add the workflow to `c8volt/process`: rejected because the feature is an ops analysis workflow rather than a general process-instance retrieval or enrichment primitive.

## Decision: Reuse process-instance and runtime element services for data access

**Rationale**: Existing process-instance services own keyed lookup, tenant-safe search, paging, and process filters. Existing element services own runtime element lookup/search and Camunda 8.7 unsupported behavior. Reusing those services avoids duplicate generated-client access and keeps version-specific API differences in their existing adapters.

**Alternatives considered**:

- Add generated-client calls in a new ops version adapter: rejected because process-instance and element services already expose the needed version-neutral capabilities.
- Re-query through CLI commands such as `get pi` and `get element`: rejected because facade/service workflows should compose APIs directly instead of shelling out or parsing command output.

## Decision: Treat explicit-key mode as a frozen keyed set after validation and deduplication

**Rationale**: Explicit keys are the primary resources in keyed mode and may come from flags, stdin, or both. Deduplicating before lookup avoids duplicate work while preserving a single result per key. Rejecting invalid, missing, unauthorized, or empty stdin inputs prevents misleading partial analysis.

**Alternatives considered**:

- Silently drop invalid keys: rejected because operators need a trustworthy read-only analysis and scripts need deterministic failures.
- Require process-definition selectors in keyed mode: rejected because keyed investigation may intentionally span process definitions.

## Decision: Treat process-definition search mode as discovery followed by frozen analysis

**Rationale**: Process-definition search mode compares a meaningful operational cohort. Freezing selected process instances before element inspection makes duration and comparison calculations stable even if runtime state changes while the command is executing.

**Alternatives considered**:

- Inspect elements while discovery is still paging: rejected because comparison indicators and duration shares need one coherent selected set.
- Apply `--limit` to details after discovery: rejected because the spec scopes discovery controls to process-instance selection only.

## Decision: Sort unavailable process-instance durations after measured durations

**Rationale**: Clarification selected measured durations first. This preserves slowest-first usefulness for human, JSON, and keys-only modes while keeping terminal instances with unavailable whole duration visible and deterministically ordered.

**Alternatives considered**:

- Sort unavailable durations first: rejected because unavailable does not mean slow and would obscure measured slow runs.
- Reject unavailable terminal durations: rejected because the spec requires terminal process instances to remain analyzable when selected.

## Decision: Empty process-definition searches are successful empty analyses

**Rationale**: Clarification selected successful empty output. This matches normal read-only search behavior and gives scripts predictable no-match semantics across human, JSON, and keys-only modes.

**Alternatives considered**:

- Fail empty searches: rejected because no matches is not an invalid request.
- Prompt to broaden search: rejected because this command must remain pipeline-friendly and automation-safe.

## Decision: Build complete timelines before filtering

**Rationale**: Element durations, transition timings, comparison samples, and process-duration shares must be calculated from complete unfiltered timelines. Applying filters only to rendered details prevents synthetic transition timings and keeps relative indicators stable.

**Alternatives considered**:

- Filter elements before transition calculation: rejected because it would create misleading gaps and artificial adjacent pairs.
- Recalculate comparisons after filters: rejected because indicators must describe the frozen selection rather than the currently visible subset.

## Decision: Use explicit comparison scopes for relative indicators

**Rationale**: Process, element, and transition indicators compare different scopes: same process-definition key for roots, same process-definition key plus element ID/type for elements, and same process-definition key plus from/to element IDs/types for transitions. Keeping scopes explicit makes tests and JSON fields unambiguous.

**Alternatives considered**:

- Compare all selected timings together: rejected because roots, elements, and transitions measure different concepts.
- Compare by BPMN process ID only: rejected because process-definition key captures the deployed definition version used by the selected run.

## Decision: Keep human output compact and JSON explicit

**Rationale**: Human output should remain scan-friendly and follow the issue's compact `dur:`, bar, `PI:`, and arrow conventions. JSON output should use explicit field names for the same data so automation does not depend on positional parsing.

**Alternatives considered**:

- Make human output label every indicator: rejected because the spec requires label-free compact comparison placement.
- Mirror compact human tokens in JSON: rejected because structured output should expose stable named fields.

## Decision: Treat Camunda 8.7 as an unsupported-version analysis failure

**Rationale**: Runtime element inspection is supported through the existing element service for Camunda 8.8 and 8.9; 8.7 should surface the established unsupported-version error rather than rendering incomplete analysis.

**Alternatives considered**:

- Render only process-instance roots on 8.7: rejected because it would claim success while omitting required timeline analysis.
- Hide the command for 8.7 configs: rejected because command availability is stable and runtime capability errors are the repository pattern.
