# Contract: Process-Instance Dry Run Scope Preview

## Command Surface

| Command | Required flag | Required behavior |
|---------|---------------|-------------------|
| `cancel process-instance` / `cancel pi` | `--dry-run` | Preview the same roots and affected family keys that real cancellation would use, then exit without confirmation, cancel requests, or wait polling |
| `delete process-instance` / `delete pi` | `--dry-run` | Preview the same roots and affected family keys that real deletion would use, then exit without confirmation, delete requests, cancel-before-delete requests, or wait polling |

## Scope Calculation Contract

Dry run must call the same dependency-expansion path as real preflight:

1. Collect direct `--key`, stdin, or search-page selected keys.
2. Resolve ancestry from selected children to roots.
3. Resolve descendants/family keys from each root.
4. Deduplicate roots and affected family keys using the existing plan semantics.
5. Return complete or partial scope data, or fail if no actionable scope resolves.

The command is incomplete if dry run calculates roots or affected family keys through a separate traversal path.

## Human Output Contract

Human-readable dry-run output must show:

- selected process instance count
- process-instance tree count for the requested operation
- total process instances in scope
- count of selected process instances already in final state when applicable
- for delete, count of process instances not in final state, their states, and whether `--force` would cancel them before delete
- whether the resolved scope is complete or partial
- selected keys when practical for the existing output mode
- root process-instance tree keys
- in-scope process-instance keys or a clear affected-scope representation
- warnings and missing ancestor keys when applicable

## Structured Output Contract

Structured output must include fields equivalent to:

```json
{
  "operation": "cancel",
  "requestedKeys": ["child-1"],
  "resolvedRoots": ["root-1"],
  "affectedFamilyKeys": ["root-1", "child-1"],
  "requestedCount": 1,
  "resolvedRootCount": 1,
  "affectedCount": 2,
  "selectedFinalStateCount": 0,
  "selectedFinalState": [],
  "requiresCancelBeforeDeleteCount": 1,
  "requiresCancelBeforeDelete": [{"key": "root-1", "state": "ACTIVE"}],
  "traversalOutcome": "complete",
  "scopeComplete": true,
  "warning": "",
  "missingAncestors": [],
  "mutationSubmitted": false
}
```

For partial traversal, `traversalOutcome` must be `partial`, `scopeComplete` must be false, and `missingAncestors` must include machine-readable keys.

For search-mode dry runs that process multiple pages, structured output must return one aggregate summary with nested per-page previews. The aggregate summary must expose total requested count, resolved root count, affected count, count/details for selected instances already in final state, count/details for delete instances requiring cancellation before delete, overall traversal outcome, overall scope completeness, and `mutationSubmitted=false`; each nested preview must retain the page-level requested keys, roots, affected family keys, details for selected instances already in final state, delete cancel-before-delete details, traversal outcome, warnings, and missing ancestors.

## Non-Mutation Contract

When `--dry-run` is set, the command must not call:

- `CancelProcessInstances`
- `DeleteProcessInstances`
- force-cancel-before-delete mutation paths
- confirmation prompts
- wait or polling paths

Tests should make unexpected mutation calls fail loudly.

## Search and Paging Contract

Search-mode dry run must reuse the same page selection path as real search-based cancel/delete:

- page size and limit behavior must match the real command path
- each selected page must run the same dependency expansion as real preflight
- partial page traversal warnings must remain visible
- structured output must preserve aggregate totals and nested per-page data for automation to inspect the previewed scope

## Orphan-Parent Contract

| Situation | Required dry-run behavior |
|-----------|---------------------------|
| Full ancestry/family resolves | Render complete preview and exit successfully |
| Missing ancestor but actionable roots/family keys resolve | Render partial preview, warning text, and missing ancestor keys; exit successfully |
| No actionable roots or family keys resolve | Fail consistently with current unresolved/orphan handling |

## Documentation Contract

The feature is incomplete until:

- command help advertises `--dry-run` on cancel/delete process-instance
- README includes at least one dry-run example or explanation for destructive preview
- generated CLI docs are refreshed through `make docs-content`
