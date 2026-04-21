# Research: Report Process Definition Incident Statistics

## Decision 1: Treat `in:` as incident-bearing process-instance count, not raw incident total

- **Decision**: Define `in:` as the number of process instances for the process definition that currently have at least one active incident, counting each affected process instance at most once.
- **Rationale**: The clarified spec explicitly chose process-instance count semantics. The generated `v8.8` and `v8.9` clients expose incident-process-instance statistics grouped by definition with `ActiveInstancesWithErrorCount`, which already matches that meaning more closely than the existing element-stat `Incidents` field.
- **Alternatives considered**:
  - Keep using raw incident totals from process-definition element statistics: rejected because one process instance can have multiple active incidents, which would overcount relative to the clarified contract.
  - Keep the meaning ambiguous and document it loosely: rejected because that would weaken tests and leave the renderer contract unstable.

## Decision 2: Keep `ac`, `cp`, and `cx` on the existing process-definition statistics endpoint

- **Decision**: Continue sourcing active, completed, and canceled counts from the existing `GetProcessDefinitionStatistics` response in `v8.8` and `v8.9`, and only replace the source for `in:`.
- **Rationale**: The current versioned services already enrich `Statistics` from `ProcessDefinitionElementStatisticsQueryResult`, and the issue only reports the incident count as wrong. Reusing the existing source for the other fields keeps the implementation small and avoids unnecessary churn.
- **Alternatives considered**:
  - Replace the whole stats pipeline with only the newer process-instance statistics endpoints: rejected because that would broaden the change without a need for the other fields.
  - Stop showing `ac`, `cp`, and `cx` on unsupported paths to simplify consistency: rejected because the issue explicitly asks to preserve the other fields.

## Decision 3: Use incident statistics grouped by definition on `v8.8` and `v8.9`

- **Decision**: Plan to source the supported-version `in:` value from the newer incident/process-instance statistics endpoints grouped by process definition rather than from `SearchIncidents`.
- **Rationale**: The generated `v8.8` and `v8.9` clients expose `GetProcessInstanceStatisticsByDefinitionWithResponse`, and its result carries `ActiveInstancesWithErrorCount` grouped by process definition. That is a closer direct fit to the clarified requirement than fetching raw incidents and deduplicating process-instance keys manually through `SearchIncidents`.
- **Alternatives considered**:
  - Use `SearchIncidents` and deduplicate `ProcessInstanceKey` client-side: rejected as a fallback option only if the grouped endpoint proves unusable, because it increases response volume and pushes aggregation work into repository code.
  - Keep the older `GetProcessDefinitionStatistics` endpoint alone: rejected because its `Incidents` field reflects incident totals at the element aggregation layer, not the clarified process-instance count.

## Decision 4: Keep `v8.7` unsupported for `in:`

- **Decision**: Treat the `v8.7` incident count as unsupported and omit the `in:` segment entirely on that version.
- **Rationale**: The current `v87` processdefinition service rejects `WithStat` outright through `ensureStatsSupported(...)`, and the repository’s `v87` Operate client/service flow does not expose the newer grouped incident-process-instance statistics seam used by `v8.8` and `v8.9`. The clarified spec prefers omission over a guessed or placeholder count.
- **Alternatives considered**:
  - Approximate the value by pulling incidents and deduplicating process-instance keys through a new `v87` side path: rejected because the plan should remain repository-native and low-risk unless a reliable existing surface already exists.
  - Keep `in:-` as the unsupported marker: rejected because clarification explicitly chose omitting `in:` entirely.

## Decision 5: The shared process-definition statistics model needs a supported-vs-unsupported distinction

- **Decision**: Extend the shared/domain process-definition statistics representation with `IncidentCountSupported bool` so it can distinguish “incident count is supported and equals zero” from “incident count is unsupported and should not render.”
- **Rationale**: Today the renderer shows `in:` whenever `Statistics != nil`, and `zeroAsMinus(...)` turns `0` into `-`. That means the existing `Incidents int64` field alone cannot express either `in:0` on supported versions or omission on unsupported versions without additional state.
- **Alternatives considered**:
  - Keep `Incidents int64` as-is and special-case version in the renderer: rejected because the renderer should not need to know API version; support should flow from the enriched model.
  - Omit `Statistics` entirely on unsupported versions: rejected because `ac`, `cp`, and `cx` would disappear with it.

## Decision 6: The renderer must stop treating zero as “missing” for supported incident counts

- **Decision**: Update `oneLinePD(...)` so the incident segment is formatted independently from `zeroAsMinus(...)`, allowing `in:0` when supported and omission when unsupported.
- **Rationale**: The current renderer uses `zeroAsMinus(stats.Incidents)` inside a fixed `"[ac:%s cp:%s cx:%s in:%s]"` format, which produces `in:-` for zero and leaves no way to omit the segment while still showing `ac`, `cp`, and `cx`.
- **Alternatives considered**:
  - Reuse `zeroAsMinus(...)` and accept `in:-` for zero: rejected because the clarified spec expects `in:0` on supported versions with no affected process instances.
  - Replace all zero formatting for `ac`, `cp`, and `cx`: rejected because the issue asks to preserve the other field behavior.

## Decision 7: Documentation updates are required, not optional

- **Decision**: Plan to update user-facing docs in the same implementation slice and regenerate CLI reference output.
- **Rationale**: The constitution requires README and generated docs to match user-visible command behavior. `get process-definition --stat` output changes in a way that matters to both humans and automation readers, so the docs should reflect the supported-version and unsupported-version behavior.
- **Alternatives considered**:
  - Leave docs untouched because the command name and flags do not change: rejected because the output contract changes materially.
  - Update only generated docs: rejected because the constitution explicitly calls out README and relevant generated docs together.

## Confirmed Current Seams

- [`cmd/get_processdefinition.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processdefinition.go) is the sole CLI entry point for `get process-definition`, and `flagGetPDWithStat` controls whether versioned services are enriched through `WithStat`.
- [`cmd/cmd_views_get.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_get.go) owns the visible one-line `get pd` rendering through `oneLinePD(...)`.
- [`internal/services/processdefinition/v88/service.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/service.go) and [`internal/services/processdefinition/v89/service.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v89/service.go) already enrich process definitions with stats when `WithStat` is enabled.
- [`internal/services/processdefinition/v87/service.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/service.go) currently rejects `WithStat`, making `v8.7` the natural unsupported boundary.

## Setup Inventory: Existing `get process-definition --stat` Flow

- `cmd/get_processdefinition.go` exposes `--stat` through `flagGetPDWithStat`, and `collectOptions()` in `cmd/cmd_options.go` forwards that flag into the facade as `options.WithStat()`.
- `c8volt/process/client.go` passes facade options straight through to the versioned processdefinition services, so the CLI does not branch on version when stats are requested.
- `internal/services/processdefinition/v88/service.go` and `internal/services/processdefinition/v89/service.go` both call `retrieveProcessDefinitionStats(...)` from their search and lookup paths when `WithStat` is enabled, which means supported-version incident enrichment belongs inside those service implementations.
- `internal/services/processdefinition/v87/service.go` blocks `WithStat` in `ensureStatsSupported(...)`, so any unsupported-version behavior change must be implemented deliberately rather than assumed from existing v8.8/v8.9 logic.
- `cmd/cmd_views_get.go` renders the final one-line output in `oneLinePD(...)` and currently prints a fixed `[ac:%s cp:%s cx:%s in:%s]` segment whenever `Statistics != nil`, using `zeroAsMinus(...)` for all four fields.

## Setup Inventory: Shared Stats Model Constraint

- `internal/domain/processdefinition.go` and `c8volt/process/model.go` both currently model process-definition statistics with only four scalar counters: `Active`, `Canceled`, `Completed`, and `Incidents`.
- `c8volt/process/convert.go` maps that domain structure straight through to the public facade model with no extra support-state metadata.
- Because the renderer only sees `Statistics != nil` and `Incidents int64`, the current shared model cannot distinguish `supported zero` from `unsupported omission`, which confirms the planned model refinement is necessary before renderer logic can stay version-agnostic.

## Setup Inventory: Existing Regression Anchors

- `internal/services/processdefinition/v88/service_test.go` and `internal/services/processdefinition/v89/service_test.go` already cover `WithStat` enrichment via `GetProcessDefinitionStatisticsWithResponse`, so they are the primary seams for proving the supported-version incident source changes without regressing `ac`, `cp`, or `cx`.
- `internal/services/processdefinition/v87/service_test.go` already asserts that `WithStat` is unsupported on `v8.7`, so it is the anchor for keeping the unsupported boundary truthful.
- `cmd/get_test.go` is the command-level regression seam for `get process-definition` help text and rendered output, and `cmd/cmd_views_get.go` contains the concrete `oneLinePD(...)` formatter that those tests need to protect.
- `c8volt/process/client_test.go` already proves the facade forwards `WithStat`, making it the right place to protect any shared model or conversion changes that affect public process-definition statistics.

## Generated-Client Capability Notes

- `internal/clients/camunda/v88/camunda/client.gen.go` and `internal/clients/camunda/v89/camunda/client.gen.go` both expose `GetProcessInstanceStatisticsByDefinitionWithResponse`.
- The grouped result type includes `ActiveInstancesWithErrorCount`, which directly expresses “number of active process instances that currently have an incident with the specified error hash code.”
- Those same clients also expose `SearchIncidentsWithResponse`, which is a viable fallback or supplemental lookup path if the grouped endpoint needs error-specific fan-out, but it is not the first-choice source for the clarified contract.
- The existing `GetProcessDefinitionStatisticsWithResponse` path still returns `ProcessElementStatisticsResult` with `Incidents int64`, which reflects incident totals at the element-stat layer and is therefore not sufficient on its own for the clarified semantics.

## Regression Anchors

- [`internal/services/processdefinition/v88/service_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/service_test.go) and [`internal/services/processdefinition/v89/service_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v89/service_test.go) already cover `WithStat` enrichment and are the right place to prove the new incident-bearing process-instance count behavior.
- [`internal/services/processdefinition/v87/service_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/service_test.go) already proves `WithStat` is unsupported and should be updated to reflect the final `v8.7` truthfulness contract.
- [`cmd/get_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go) is the natural command-level anchor for `get process-definition --stat` rendering expectations across supported and unsupported versions.
- [`c8volt/process/client_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go) is the right shared-facade regression seam if the public process-definition statistics model changes.
