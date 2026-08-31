# Data Model: Stable Tenant-Aware Process-Definition Ordering

## Process Definition

A version-neutral representation of one deployed BPMN process definition.

| Field | Type | Role | Validation / semantics |
|-------|------|------|------------------------|
| Tenant ID | string | Primary canonical sort field and latest-group field | Compared exactly and case-sensitively. The displayed default value `<default>` is an ordinary value for ordering. |
| BPMN process ID | string | Secondary canonical sort field and latest-group field | Compared exactly and case-sensitively. |
| Version | integer | Tertiary canonical sort field and latest selection | Compared numerically; greater values sort first. |
| Process-definition key | string | Final deterministic sort field and resource identity | Opaque text; compared exactly and never parsed numerically. |
| Version tag and metadata | existing types | Filters/display | Do not participate in canonical ordering. |
| Statistics | optional statistics | Enrichment/display | Do not participate in canonical ordering or identity. |

No new persistence is introduced. Objects are populated from existing Camunda API responses.

## Canonical Sort Key

The tuple:

```text
(tenantID ASC, bpmnProcessID ASC, version DESC, processDefinitionKey ASC)
```

Invariants:

1. Text comparisons use Go's exact bytewise string ordering and are case-sensitive.
2. `<default>` receives no special first/last handling.
3. Version is numeric, so version 10 precedes version 9.
4. Key is opaque text, so `"10"` precedes `"2"` lexically.
5. Statistics, API arrival order, page size, and output mode cannot affect the result.
6. Applying the comparator repeatedly is deterministic and idempotent.

## Tenant and Process Group Key

The exact tuple:

```text
(tenantID, bpmnProcessID)
```

It identifies the boundary within which versions belong to the same latest-selection group. Values that differ only by letter case are different groups.

## Latest Selection

For every represented tenant/process group:

1. Select the greatest numeric version.
2. If multiple definitions share that version, select the lowest exact-text process-definition key.
3. Emit at most one definition for the group.
4. Canonically sort all selected definitions.

Camunda 8.8-8.10 may perform the version filtering natively, but the returned collection still receives final canonical sorting. Camunda 8.7 performs selection locally over the visible compatibility window.

## Search Request

The existing version-neutral and public request models retain their current filter and paging fields and gain one additive field:

| Field family | Purpose | Ordering interaction |
|--------------|---------|----------------------|
| Existing filters | Tenant, BPMN ID, version, version tag, and current selectors | Narrow membership only; they do not change the comparator. |
| Existing page/cursor fields | Traverse backend pages | Backend sort stabilizes traversal; final order ignores arrival order. |
| Existing limit | Bound requested results | Ordinary semantics remain unchanged. With latest intent, apply after complete latest reduction and sorting. |
| `Latest bool` | Request one newest definition per exact tenant/process group | Selects native latest on 8.8-8.10 and local reduction on 8.7. |

## Page Metadata

Existing backend cursor/page metadata remains adapter-owned. The shared collector validates progress using the established service conventions and must stop at the terminal page without emitting cursor diagnostics in default output.

## Canonical Process-Definition Collection

An in-memory sequence of matching definitions after traversal and optional latest reduction.

Invariants:

- Every ordinary matching definition returned across the traversal appears exactly once in its canonical position.
- Only latest selection intentionally removes older definitions.
- Page size and page boundaries do not alter membership or order.
- Public conversion and rendering preserve sequence order.

## Collection State Transition

```text
filtered request
    -> version adapter request with compatible backend sort
    -> page accumulation
    -> optional latest reduction
    -> canonical sort
    -> optional latest-result limit
    -> optional position-preserving statistics enrichment
    -> facade conversion
    -> human / JSON / keys-only / watch presentation
```

For ordinary non-latest searches, existing limit behavior is retained and the latest-reduction and latest-limit stages are skipped.

## Process-Definition Statistics

Existing volatile values such as active instances and incidents are attached to a process definition without changing identity or collection position. Camunda 8.7 keeps its current unsupported-statistics behavior.

## Watch Snapshot

One complete canonical collection captured during a watch refresh. A snapshot may change membership or positions only when definitions or canonical sort fields change. Statistics-only changes update cells in place.

## Lifecycle and Persistence

There is no stored entity lifecycle, schema migration, cache, or new configuration. Each command or watch refresh independently constructs a collection from current authenticated API visibility.
