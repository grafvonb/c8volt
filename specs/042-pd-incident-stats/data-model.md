# Data Model: Report Process Definition Incident Statistics

## Process Definition

- **Purpose**: Represents a deployed BPMN definition returned by `get process-definition` search and lookup flows.
- **Existing attributes**:
  - `BpmnProcessId`
  - `Key`
  - `Name`
  - `TenantId`
  - `ProcessVersion`
  - `ProcessVersionTag`
  - `Statistics`
- **Behavioral rule**:
  - The feature does not change identifier or version semantics; only the shape and sourcing of `Statistics` need planning attention.

## Process Definition Statistics

- **Purpose**: Carries the user-visible `--stat` enrichment for a process definition.
- **Existing attributes**:
  - `Active`
  - `Canceled`
  - `Completed`
  - `Incidents`
- **Planned refinement**:
  - Preserve `Active`, `Canceled`, and `Completed` as the existing element-stat counts.
  - Refine the incident-related attribute so the model can express both:
    - a supported incident-bearing process-instance count, including zero
    - an unsupported state where the renderer must omit `in:`
- **Invariants**:
  - Supported versions must count each affected process instance at most once.
  - Unsupported versions must not fabricate a zero or placeholder count.
  - The model must let the renderer distinguish `supported zero` from `unsupported`.

## Incident Statistics Support State

- **Purpose**: Captures whether the active version can supply the clarified `in:` value.
- **States**:
  - `Supported`: the model carries a verified incident-bearing process-instance count and the renderer must show `in:<count>`.
  - `Unsupported`: the model indicates that incident count support is unavailable and the renderer must omit `in:`.
- **Current mapping**:
  - `v8.8`: `Supported`
  - `v8.9`: `Supported`
  - `v8.7`: `Unsupported`

## Supported-Version Incident Count Source

- **Purpose**: Represents the newer generated-client statistics source used for the clarified `in:` value on `v8.8` and `v8.9`.
- **Relevant fields from generated clients**:
  - `ProcessDefinitionKey`
  - `ProcessDefinitionId`
  - `ProcessDefinitionVersion`
  - `ActiveInstancesWithErrorCount`
  - `TenantId`
- **Invariants**:
  - The source is grouped by process definition.
  - The source is concerned with active process instances that currently have incidents.
  - The chosen service logic must aggregate or select this source in a way that produces one final per-definition count for rendering.

## Process Definition Statistics View

- **Purpose**: The rendered `get pd --stat` line for one process definition.
- **Current visible segments**:
  - `ac:<value>`
  - `cp:<value>`
  - `cx:<value>`
  - `in:<value>`
- **Planned rules**:
  - If stats are not requested, no bracketed stats segment is shown.
  - If stats are requested on `v8.8`/`v8.9`, render `ac`, `cp`, `cx`, and `in`, with `in:0` allowed.
  - If stats are requested on `v8.7`, preserve the other available output but omit `in:` entirely.
- **Invariant**:
  - The feature must not change ordering, identifiers, or non-incident stats formatting outside the incident-segment rule.

## Versioned Processdefinition Service

- **Purpose**: Provides version-specific processdefinition search and lookup behavior behind the shared `processdefinition.API`.
- **Members**:
  - `v87.Service`
  - `v88.Service`
  - `v89.Service`
- **Behavioral rules**:
  - `v88` and `v89` may enrich `Statistics` from multiple repository-native endpoints when `WithStat` is enabled.
  - `v87` must preserve current unsupported behavior for incident count semantics instead of synthesizing a derived value through a new side architecture.

## Documentation Contract Record

- **Purpose**: Tracks the user-visible statement that must stay aligned with the implemented output.
- **Fields**:
  - Affected command: `get process-definition --stat`
  - Supported versions: `8.8`, `8.9`
  - Unsupported version behavior: `8.7 omits in:`
  - Regeneration path: `make docs-content`
- **Invariant**:
  - Documentation updates belong to the same implementation slice because the output contract changes.
