# Data Model: Effective User-Task Variables

## Reused records

`domain.UserTask` and public `task.UserTask` remain unchanged. In particular, task selection, tenant information, assignment, key order, and existing optional fields do not change.

Reuse `domain.ProcessInstanceVariable` internally. Public `task.UserTaskVariable` is an alias of `process.ProcessInstanceVariable`, retaining its JSON tags; the alias describes use by a task without introducing another wire schema.

| Field | Type / JSON name | Meaning and validation |
| --- | --- | --- |
| Name | string / `name` | Backend-selected effective name; unique within each successful task collection |
| Value | string / `value` | Serialized value as received; empty strings are valid; not shortened for JSON |
| VariableKey | string / `variableKey,omitempty` | Identity returned by the backend |
| ProcessInstanceKey | string / `processInstanceKey` | Owning process, not a discovery filter |
| ScopeKey | string / `scopeKey` | Actual winning scope; may differ from process-instance key |
| TenantId | string / `tenantId,omitempty` | Backend tenant metadata; do not rewrite from discovery settings |
| APITruncated | bool / `apiTruncated` | True when the received value is reported incomplete by the backend |

No scope filtering is applied to the effective collection. In particular, process-root-only filtering would be incorrect. Per-task name sorting is stable and ascending. Identical repeated records collapse; inconsistent records for the same name cause a malformed-response error. Raw page counts remain independent of this normalization.

## New task enrichment wrappers

Add corresponding domain and public types:

| Type | Fields | Invariants |
| --- | --- | --- |
| `VariableEnrichedUserTask` | `Item UserTask` (`item`), `Variables []UserTaskVariable` (`variables`) | Item unchanged; variables initialized even when empty |
| `VariableEnrichedUserTasks` | `Total int64` (`total`), `Items []VariableEnrichedUserTask` (`items`) | Total equals returned item count; initialized empty items; order matches input |

The domain wrapper uses `[]ProcessInstanceVariable` for its variable field. Public conversion maps every field mechanically and preserves initialized collections. No process age metadata, persistence, mutation state, or lifecycle status is added.

An empty enriched collection is `{"total":0,"items":[]}`. A returned task with no variables is represented by an entry with `"variables":[]`. The ordinary non-enriched task shape remains unchanged.

## Internal variable paging records

These records stay internal and are not additional public facade interfaces:

- `UserTaskVariablePageRequest`: `From int32` and `Size int32`; nonnegative offset, positive size; no cursor field.
- `UserTaskVariablePage`: `Items []ProcessInstanceVariable`, `Request UserTaskVariablePageRequest`, `RawItemCount int32`, `ReportedTotal UserTaskReportedTotal` (reuse exact/lower-bound semantics), and `HasContinuationEvidence bool` (including a nonempty end cursor in raw metadata).
- Raw adapter DTO: variable data plus pointer-backed required total and capped-total fields to distinguish absent metadata from valid zero/false. Recognize both raw truncation flag names using established precedence.

The adapter rejects missing required response metadata, wrong raw value types, invalid totals, and malformed bodies with the existing malformed-response error category. Required `value` presence is distinguishable from a valid empty string. The service retains the highest observed lower bound and checks request consistency, cancellation, offset progress, overflow, and total contradictions. An empty capped-total page is terminal only after satisfying that bound and without further continuation evidence. Cursor presence is evidence only; requests remain offset-only. Generated types remain inside matching version adapters.

## Display option

`--var-value-limit` is command presentation state only: integer, default zero, nonnegative, explicitly supplied only with `--with-vars`. Positive limits apply after human structured-value compaction and count Unicode runes; ellipsis and labels are additional presentation characters. Neither the facade nor service receives this display limit.

## Read lifecycle

1. Existing lookup/search selects tasks; missing explicit keys remain errors.
2. For an eligible nonempty selected collection, retrieve all effective-variable pages per task.
3. Validate, normalize identical duplicates, and sort each complete variable collection.
4. Attach variables without changing the task; convert and render only successful results.
5. On any failure, return the established error. Earlier streamed human pages may exist, but no final successful summary or successful JSON envelope may claim completion.

No durable state transitions or snapshot consistency guarantees are introduced.
