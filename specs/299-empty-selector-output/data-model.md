# Data Model: Empty Selector Result Output

This feature changes presentation only. It introduces no persisted entity, public type, migration, or service contract.

## Selector scope

- **Source**: Existing mutation planning result, especially `RequestedCount` and its successful return.
- **Meaning**: Number of process instances selected across completed discovery, not the number on one page or the number of mutation reports.
- **Validation**: Only successful planning with `RequestedCount == 0` enters the empty-result path. Invalid selectors, discovery errors, aborted nonempty operations, and sparse intermediate pages do not qualify merely because a collection is empty.
- **Relationship**: The operation (`delete` or `cancel`) and execution type (normal or dry-run) select the corresponding output payload.

## Shared result envelope

Reuse `ResultEnvelope[T]` from `cmd/command_contract.go`.

| Field | Empty-result value |
| --- | --- |
| `outcome` | `succeeded`, including normal execution with no-wait |
| `command` | Canonical `delete process-instance` or `cancel process-instance` |
| `payload` | Existing empty report object or aggregate dry-run summary |
| `tenantContext` | Existing attached context when available; no additional lookup |
| `class`, `detail` | Absent for successful empty discovery |

A completed no-op does not use `accepted`, and it does not create mutation evidence.

## Normal report payload

Reuse `process.DeleteReports` or `process.CancelReports` from `c8volt/process/model.go`.

- `Items`: Zero reports. Existing `json:"items,omitempty"` serialization omits this field, giving `{}`.
- No synthetic success report, instance key, or mutation status is added.
- Normal empty reports have no `mutationSubmitted` field; successful outcome plus the empty report object expresses the no-op within the existing contract.

## Dry-run aggregate payload

Reuse `processInstanceDryRunSummary` from `cmd/cmd_views_processinstance_dryrun.go`, initialized by `newProcessInstanceDryRunSummary(operation, nil)`.

| Fields | Value |
| --- | --- |
| `operation` | `delete` or `cancel` |
| `requestedCount`, `resolvedRootCount`, `affectedCount` | `0` |
| `selectedFinalStateCount`, `requiresCancelBeforeDeleteCount` | `0` |
| `selectedFinalState`, `requiresCancelBeforeDelete`, `missingAncestors`, `previews` | `null`, preserving the existing constructor's nil-slice serialization |
| `traversalOutcome` | `complete` |
| `scopeComplete` | `true` |
| `warning` | Empty string |
| `mutationSubmitted` | `false` |

Do not fabricate one empty page preview or normalize collection serialization globally.

## State transitions

Successful discovery with zero selected instances transitions directly to completed no-op rendering and successful return. There is no confirmation, mutation submission, wait, or retry transition. Successful nonempty discovery retains its existing preview/confirmation/mutation lifecycle. Existing errors and operator aborts retain their existing handling.
