# Facade and Service Contract

## Existing public entry points

`task.API` signatures remain unchanged:

- `SearchUserTasks(ctx, request, opts...)`
- `SearchUserTasksPages(ctx, request, visitor, opts...)`
- `SearchUserTasksTotal(ctx, request, opts...)`

Each receives the additive `SearchRequest.VariableFilters` described in [data-model.md](../data-model.md). The facade performs mechanical conversion and existing `ferrors` conversion, with no parsing, paging, filtering, or variable retrieval. No new interface methods or generated client methods are required.

## Native request

The existing `SearchUserTasksPage` adapters for 8.8, 8.9, and 8.10 send `POST /v2/user-tasks/search` through `SearchUserTasksWithResponse`. Populate `filter.localVariables`, retaining all existing ordinary and tenant filters plus pagination fields.

Example for `--var 'status="approved"'`:

```json
{
  "filter": {
    "localVariables": [
      {"name": "status", "value": {"$eq": "\"approved\""}}
    ]
  }
}
```

| Normalized clause | Native `value` filter |
| --- | --- |
| `$eq` | `AdvancedStringFilter.Eq`, preserving serialized text |
| `$neq` | `AdvancedStringFilter.Neq`, preserving serialized text |
| `$exists` | `AdvancedStringFilter.Exists`, explicit boolean pointer |
| `$in` | `AdvancedStringFilter.In`, decoded JSON string array |
| `$notIn` | `AdvancedStringFilter.NotIn`, decoded JSON string array |
| `$like` | `AdvancedStringFilter.Like`, unchanged pattern |

Every clause yields one `VariableValueFilterProperty`; preserve duplicates/order and do not merge predicates. Empty clauses omit `localVariables`, preserving existing requests. Do not populate effective-variable or process-variable search fields.

Validate the existing domain filter set and build all native clauses before HTTP. Domain membership validation only checks shape, so mapping must also decode the string array and propagate errors. Test invalid direct facade requests as well as CLI inputs. Do not tighten PI validation or invent task-specific coercion.

Version-local helpers may follow the existing PI mapping code without importing PI service packages. No hand-edited generated files, raw request escape hatch, or broad common mapper extraction is necessary.

## Traversal and display boundaries

Keep `internal/services/usertask/search.go` in charge of cursor/offset advancement, sparse-page handling, caller limits, exact versus capped totals, and completion facts. Its existing query propagation carries predicates into every page. Preserve cancellation, malformed metadata, visitor failures, and backend errors.

Local filtering causes no extra requests beyond the existing search/count workflow. Optional effective-variable retrieval happens after selection through the existing enrichment workflow; it does not contribute to matching. Keep total/keys/empty exclusion and existing error behavior.

## Required contract evidence

- v88/v89/v810 HTTP fixtures assert each operator's exact request representation, null versus string-null text, wildcard escapes, false existence, combined ordinary/tenant filters, and omitted empty filter sets.
- Facade tests verify every predicate field, order, options, and errors for collected, visitor, and total calls.
- Service tests prove unchanged predicates on multiple/sparse/count pages, without per-task variable calls or mutations.
- Command execution tests assert selected identities, output modes, no-request validation failures, and optional display request counts.
- Schema/request tests establish faithful native mapping, not empirical backend missing/null behavior. Authorized live read-only fixtures may verify scope/shadowing semantics; record that separately rather than claiming mocks prove server behavior.
