# Data Model: Variable Destination Scope Metadata

## Existing entities and proposed change

| Entity | Fields relevant here | Change |
| --- | --- | --- |
| `domain.UserTask` | `UserTaskKey`, `ProcessInstanceKey`, `ElementInstanceKey`, tenant | None; existing source of identity |
| `domain.UserTaskVariablePlan` | Task key, element key, tenant, additions, changes, unchanged, untouched, target scope keys | Add `ProcessInstanceKey string` |
| `task.UserTaskVariablePlan` | Public counterpart of the internal plan | Add `ProcessInstanceKey string` with `json:"-"` |
| Planned addition/value | `Name`, `ScopeKey`, `Inherited`, `Value`, truncation flag | None |
| Planned change | `Name`, `ScopeKey`, `Inherited`, `Before`, `After`, truncation flag | None |
| Scope mutation target | Scope key, tenant, variables, task associations | None |
| PI command plan | `ProcessInstanceKey`, variable categories | None; existing key is sufficient |

## Relationships and propagation

One selected task belongs to one process instance and has one local element scope. Each planned write retains its actual destination scope. Several selected tasks can reference one deduplicated scope mutation target; their individual owner identities remain on their own plans.

The service builder copies the task's process identity onto the internal plan. Both domain-to-facade and facade-to-domain converters copy that field mechanically. Existing struct-copy behavior preserves it. The field is descriptive metadata only and must never select or redirect writes.

## Display rules

| Known facts | Detailed UT addition/change annotation |
| --- | --- |
| Local variable at task element scope | `(local, element:<scopeKey>)` |
| Inherited, nonempty owner key equals destination | `(inherited, pi:<scopeKey>)` |
| Inherited, nonempty owner key differs from destination | `(inherited, element:<scopeKey>)` |
| Inherited, owner key absent in legacy/incomplete plan | Existing `(inherited scope <scopeKey>)` |

PI additions and changes use `(pi:<ProcessInstanceKey>)`. Inherited bulk summaries use the corresponding typed destination alone, or the original bare key for incomplete metadata. Deduplication identity remains the raw scope key, in first-seen order.

## Invariants and validation

- Preserve existing task identity, scope ownership, payload, complete-discovery, and mutation-target validation; introduce no new precondition for process identity.
- Local additions continue to target the task element. Existing effective variables continue to target their returned owner scope.
- No added field is serialized in public plan JSON. Marshaling an otherwise identical plan with and without this metadata produces identical output.
- JSON round trips can omit display metadata; executing such plans must retain existing behavior.
- Unknown metadata must not trigger discovery or fabricate scope kind. Existing failures remain failures.
- Preserve shared-target grouping, counts, association order, map/slice isolation, and all status values.

## State transitions

No new lifecycle or persistence is introduced. Discovery produces a plan, followed by dry-run/no-op/abort or existing submission and confirmation. Metadata moves with the plan and does not alter any transition, retry, wait, or result status.
