# Data Model: Latest Process-Definition Search Paging

This feature changes transient request state only. It adds no persistent entity, database schema, public facade type, or CLI data model.

## Process Definition Page Request

Represents the version-neutral input already passed to one process-definition search page.

| Field | Meaning | Validation |
|---|---|---|
| `Size` | Maximum items requested for the page | Positive after existing normalization; latest direct lookup uses 1000 |
| `After` | Opaque forward continuation cursor | Empty means no cursor is available; non-empty is preserved exactly |
| `From` | Offset for ordinary searches | Retained for ordinary paging; omitted from latest-page wire requests |

The request is combined with the existing process-definition filter. `IsLatestVersion` determines whether an empty-cursor page is initial-latest or ordinary-offset mode.

## Page Modes

### Initial Latest Page

- Conditions: `IsLatestVersion` is true and `After` is empty.
- Wire fields: `limit` only.
- Forbidden wire fields: `after`, `from`, and backward-cursor fields.
- Stable context: existing filters, tenant scope, and sort by process-definition ID then tenant ID, both ascending.

### Latest Continuation Page

- Conditions: `After` is non-empty for a latest search.
- Wire fields: `after` and `limit`.
- Validation: `after` equals the preceding response's non-empty end cursor byte-for-byte.
- Forbidden wire fields: `from` and backward-cursor fields.

### Ordinary Offset Page

- Conditions: latest mode is false and `After` is empty.
- Wire fields: `from` and `limit`, including explicit offset zero.
- Forbidden wire field: `after`.
- Stable context: existing ordinary filters and version-descending/name-ascending sort remain unchanged.

## Response Continuation Cursor

An opaque, transient value read from the existing page response.

- Empty or absent: no cursor continuation request is created.
- Non-empty: copied unchanged into the next page request.
- It is never trimmed, decoded, normalized, synthesized, or exposed as a new CLI field.

## State Transitions

```text
initial latest (limit only)
    ├── non-empty end cursor and result limit not reached ──> latest continuation
    └── no non-empty end cursor or result limit reached ────> complete

latest continuation
    ├── another non-empty end cursor and limit not reached ─> latest continuation
    └── no non-empty end cursor or result limit reached ────> complete

ordinary initial/continuation ──────────────────────────────> existing offset flow
```

## Invariants

- A request never contains both `after` and `from`.
- An initial latest request never contains an empty `after` field.
- A non-empty continuation cursor is never changed.
- Version, key, filter, tenant, sort, limit, and result semantics are unchanged outside page-mode selection.
