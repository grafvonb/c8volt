# Contract: Process-Instance Search Filter Pushdown

## Shared Filter Contract

The shared process-instance search filter must distinguish between:

| Semantic | Contract |
|--------|----------|
| Exact parent match | Use `ParentKey` equality |
| Parent presence | Use optional `HasParent=*bool(true)` |
| Parent absence | Use optional `HasParent=*bool(false)` |
| Incident presence | Use optional `HasIncident=*bool(true)` |
| Incident absence | Use optional `HasIncident=*bool(false)` |

`ParentKey` equality must remain separate from parent presence semantics.

## Version Capability Contract

| Version | `roots-only` / `children-only` | `incidents-only` / `no-incidents-only` | `orphan-children-only` |
|--------|---------------------------------|-----------------------------------------|-------------------------|
| `v8.7` | Client-side fallback | Client-side fallback | Client-side fallback |
| `v8.8` | Request-side pushdown | Request-side pushdown | Client-side fallback |
| `v8.9` | Request-side pushdown | Request-side pushdown | Client-side fallback |

Unsupported versions must not silently approximate request-side semantics with a misleading partial request.

## Request Construction Contract

| Semantic | Supported request rule |
|--------|-------------------------|
| Root-only | Encode parent-key absence through the generated advanced filter shape |
| Children-only | Encode parent-key presence through the generated advanced filter shape |
| Incidents-only | Encode `hasIncident=true` |
| No-incidents-only | Encode `hasIncident=false` |
| Parent-key equality | Keep existing explicit parent-key equality encoding |
| Orphan children | Do not encode in the initial request |

## Paging Contract

For supported versions:

- page totals and continuation prompts must reflect the filtered server result set
- no broad first page should be fetched when the request can already encode the filter

For fallback versions:

- the final visible results must still honor the CLI flags
- continuation behavior may remain bounded by the fetched pre-filter page because that is the truthful version limitation

## Audit Contract For Other `get` Commands

- The feature must explicitly inspect other `get` command families for equivalent server-capable late-filtering seams.
- If no other qualifying seam is found, the feature must record that no-addition conclusion explicitly in planning or task artifacts rather than leaving the audit implicit.

## Regression Contract

The feature is incomplete unless tests prove:

- shared filter mapping carries the new optional semantics correctly
- `v8.8` request bodies include the pushed-down predicates
- `v8.9` request bodies include the pushed-down predicates
- `v8.7` request bodies omit unsupported pushed-down predicates
- command-level paging behavior shows the supported-version improvement
- `orphan-children-only` remains client-side
