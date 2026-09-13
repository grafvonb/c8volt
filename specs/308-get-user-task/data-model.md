# Data Model: User Task Reads

## UserTask

Extend `internal/domain.UserTask` and introduce the corresponding public `task.UserTask`. Generated types do not cross the adapter boundary. Existing resolver identity fields retain their semantics.

| Go field | Type | Public JSON | Source / rule |
| --- | --- | --- | --- |
| Key | string | `key` | Native `userTaskKey`; task identity, required on successful reads |
| State | string | `state` | Native state; uppercase backend value |
| Name | string | `name,omitempty` | Nullable native name becomes empty string when absent |
| ElementId | string | `elementId,omitempty` | BPMN task element ID; display fallback for absent name |
| ElementInstanceKey | string | `elementInstanceKey,omitempty` | Runtime element identity |
| Assignee | string | `assignee,omitempty` | Nullable native value becomes empty string; assignment is not a state |
| CandidateUsers | []string | `candidateUsers,omitempty` | Copy at facade boundary; empty/absent values omitted |
| CandidateGroups | []string | `candidateGroups,omitempty` | Copy at facade boundary; empty/absent values omitted |
| ProcessInstanceKey | string | `processInstanceKey` | Owning process instance |
| ProcessDefinitionKey | string | `processDefinitionKey,omitempty` | Owning process definition |
| ProcessDefinitionId | string | `processDefinitionId,omitempty` | BPMN process ID; input flag is `--bpmn-process-id` |
| TenantId | string | `tenantId,omitempty` | Preserve backend metadata, including keyed reads outside discovery tenant |

All keys remain strings, avoiding numeric precision loss in JSON. No dates, forms, variables, custom headers, business IDs, or mutation state are added. Missing mandatory identity fields or invalid successful payloads use the existing invalid-response/error policy rather than producing a fabricated task. Legacy resolver checks and fallback conversions remain unchanged.

Supported state set on the checked-in 8.8/8.9/8.10 contracts: ASSIGNING, CANCELED, CANCELING, COMPLETED, COMPLETING, CREATED, CREATING, FAILED, UPDATING. CLI parsing is case-insensitive; `all` maps to no state predicate. No transitions are performed or inferred by this read feature.

## UserTaskSearchQuery / public SearchRequest

| Field | Type | Meaning |
| --- | --- | --- |
| ProcessInstanceKey | string | Exact owning instance predicate |
| ProcessDefinitionKey | string | Exact definition predicate |
| BpmnProcessId | string | Backend processDefinitionId predicate |
| ElementId | string | BPMN task element predicate |
| State | string | Normalized state; empty means unrestricted |
| Assignee | string | Exact assignee predicate |
| CandidateUser | string | Exact member candidate-user predicate |
| CandidateGroup | string | Exact member candidate-group predicate |
| BatchSize | int32 | Default/max 1000; positive at CLI boundary |
| Limit | int32 | 0 means unlimited internally; explicitly supplied CLI limit must be positive |

Tenant scope comes from existing call options and configuration, not a second query tenant field. Combine predicates with AND. CLI changed-flag validation preserves the distinction between an omitted default and an explicitly supplied selector. Keyed lookup is a separate API operation, not a search query mode. All key inputs use current command key validation; string predicate values follow neighboring trimming conventions without case-folding assignees or candidates.

## Collection

Public `UserTasks` contains `Total int64` (`total`) and `Items []UserTask` (`items`), neither omitted. `Total` is the number returned after limit/visitor stop, not the global backend total. Initialize empty collections with a non-nil empty slice so the new command's empty payload is exactly `{"total":0,"items":[]}`. Preserve that shape across facade conversion and all key cardinalities. This is a new task payload; shared envelopes and unrelated resource schemas remain unchanged.

## Service page and traversal state

Use task-specific domain types with public equivalents only where needed by the page-visitor facade:

- `UserTaskPageRequest`: `From int32`, `Size int32`, `After string`; mutually exclusive offset/cursor positions. An empty initial position uses a limit request. This is backend workflow state, not a CLI flag surface.
- `UserTaskReportedTotal`: `Count int64`, `Kind` = exact or lower_bound. Normalize `hasMoreTotalItems` once in the adapter. Never confuse lower_bound with a count that can be printed as exact.
- `UserTaskSearchPage`: items, original request, raw item count, end cursor, reported total, and typed continuation state (has_more/no_more/indeterminate). Raw item count and request progress precede user-limit trimming.
- `UserTaskSearchPageStep`: the selected page, cumulative selected count, and limit-reached flag. CLI observes these facts; it does not advance positions or derive completion.
- Visitor action: continue or stop, plus an error channel for writer/interaction failure. Facade adapters map actions mechanically.
- `UserTaskSearchPagesResult`: selected items/count, pages read, and completion disposition (exhausted, limit_reached, visitor_stopped). Internal traversal failure returns an error. Completion disposition is workflow metadata, not an addition to the CLI collection payload.

Prefer existing equivalent typed page/total enums where ownership is suitable; do not reuse a helper that equates empty pages with exhaustion. Keep candidate/result slices independently owned at public boundaries.

## Invariants

1. Explicit key deduplication is stable; worker results remain in first-input order. Missing keys or backend failures never become successful partial collections.
2. Search tenant filtering is backend-applied; native keyed reads do not compare returned tenant to configured discovery tenant.
3. Each next request must advance its cursor or offset. Repeated/cyclic cursor, incompatible metadata, or arithmetic overflow returns a classified error.
4. Sparse pages with continuation are not empty-success results. Exact totals must be consistent with observed progress; capped totals never stop counting at the cap.
5. Limits count selected tasks across pages. Count mode has no user limit or visitor and uses int64 without accumulating task objects.
6. Cancellation, failures, and user stops cannot establish an exact full count. Only completed discovery or trustworthy exact backend metadata can.
7. No stored state or migrations are introduced; acceptance counting uses a stable backend fixture and does not promise snapshot isolation.
