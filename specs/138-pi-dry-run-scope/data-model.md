# Data Model: Process-Instance Dry Run Scope Preview

## DryRunScopePreview

Represents one non-mutating preview of a cancel/delete process-instance command.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `operation` | string | yes | `cancel` or `delete` |
| `requestedKeys` | list of string | yes | Keys selected directly or from one search page |
| `resolvedRoots` | list of string | yes | Root keys real execution would target |
| `affectedFamilyKeys` | list of string | yes | Full affected family keys collected by dependency expansion |
| `requestedCount` | integer | yes | Count of requested keys |
| `resolvedRootCount` | integer | yes | Count of unique resolved roots |
| `affectedCount` | integer | yes | Count of unique affected family keys |
| `selectedFinalStateCount` | integer | yes | Count of selected process instances already in a final state |
| `selectedFinalState` | list of SelectedFinalState | no | Selected process-instance keys and states for instances already in final state |
| `requiresCancelBeforeDeleteCount` | integer | yes | Count of in-scope process instances that are not in final state and would need cancellation before delete |
| `requiresCancelBeforeDelete` | list of RequiresCancelBeforeDelete | no | Delete-only process-instance keys and states that would need cancellation before delete |
| `traversalOutcome` | string | yes | `complete`, `partial`, or `unresolved` |
| `scopeComplete` | boolean | yes | True only when traversal outcome is complete |
| `warning` | string | no | Warning text from dependency expansion |
| `missingAncestors` | list of MissingAncestor | no | Missing parent keys discovered during traversal |
| `mutationSubmitted` | boolean | yes | Always false for dry run |

## MissingAncestor

Represents a parent process instance that traversal referenced but could not load.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `key` | string | yes | Missing ancestor process instance key |
| `startKey` | string | yes | Requested or traversed key whose ancestry exposed the missing ancestor |

## SelectedFinalState

Represents a selected process instance that was already in a final state at dry-run planning time.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `key` | string | yes | Selected process instance key |
| `state` | string | yes | Final state such as `COMPLETED`, `CANCELED`, or `TERMINATED` |

## RequiresCancelBeforeDelete

Represents an in-scope process instance that is not in final state during a delete dry run and cannot be removed directly without cancelling first.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `key` | string | yes | In-scope process instance key |
| `state` | string | yes | Non-final state such as `ACTIVE` |

## DryRunScopeSummary

Represents an aggregated view for search-mode dry runs that process more than one page. This is the required structured output shape for multi-page search-mode dry runs.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `operation` | string | yes | `cancel` or `delete` |
| `requestedCount` | integer | yes | Total requested keys selected across processed pages |
| `resolvedRootCount` | integer | yes | Total unique roots or accumulated root count, matching command output contract |
| `affectedCount` | integer | yes | Total affected family keys reported across processed pages |
| `selectedFinalStateCount` | integer | yes | Count of unique selected process instances already in final state across processed pages |
| `selectedFinalState` | list of SelectedFinalState | no | Unique selected process instances already in final state across processed pages |
| `requiresCancelBeforeDeleteCount` | integer | yes | Count of unique in-scope process instances that are not in final state across processed pages |
| `requiresCancelBeforeDelete` | list of RequiresCancelBeforeDelete | no | Unique delete-only process instances that would need cancellation before delete |
| `traversalOutcome` | string | yes | `partial` if any page is partial, otherwise `complete` when all pages are complete |
| `scopeComplete` | boolean | yes | False if any page is partial |
| `previews` | list of DryRunScopePreview | yes | Per-page or per-selection previews |
| `mutationSubmitted` | boolean | yes | Always false for dry run |

## Validation Rules

- `mutationSubmitted` must always be false.
- `affectedCount` must equal the number of affected family keys represented for the preview.
- `resolvedRootCount` must equal the number of resolved root keys represented for the preview.
- `scopeComplete` must be false when `traversalOutcome` is `partial` or `unresolved`.
- A preview with `traversalOutcome=unresolved` and no actionable roots or family keys must fail rather than render as success.
- Missing ancestor keys must be available as structured fields whenever warning text mentions missing ancestors.
- Search-mode structured output that processes multiple pages must return one `DryRunScopeSummary` with nested `previews`; it must not return only a flat preview array or only a de-duplicated aggregate preview.

## State Transitions

```text
selected keys
  -> dependency expansion
  -> complete preview
  -> render and exit without mutation

selected keys
  -> dependency expansion
  -> partial preview with warning
  -> render and exit without mutation

selected keys
  -> dependency expansion
  -> unresolved failure
  -> return command error without mutation
```
