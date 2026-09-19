# CLI Contract: Local User-Task Variable Filters

## Flags and grammar

Existing `get user-task`, `user-tasks`, `ut`, and `uts` gain three repeatable `StringArray` flags, all unset by default:

| Flag | Existing grammar reused |
| --- | --- |
| `--var-exists` | Comma-separated names, each requiring existence |
| `--var` | `name=value` or `name.$operator=value` |
| `--var-like` | `name=pattern`, using existing native wildcard semantics |

The parser is the existing PI parser with explicit inputs, not a new grammar. Supported canonical operators are `$eq`, `$neq`, `$exists`, `$in`, `$notIn`, `$like`; `$notin` is the existing alias. Preserve repeated flags, supported quoted commas, string-array operands, assignment splitting, error messages, wildcard escaping, and all-clauses-match behavior. Preserve its existing group order: exists flags, then var flags, then like flags. Do not broaden object/array acceptance, auto-quote unquoted text, or convert values into new representations.

Examples (single quotes protect shell dollar expansion):

```sh
c8volt get ut --var 'status="approved"'
c8volt get ut --var-exists payload,email
c8volt get ut --var-like 'email=*@example.com'
c8volt get ut --var 'status.$neq="failed",active.$exists=false'
c8volt get ut --var 'kind.$in=["a","b"],segment.$notin=["c","d"]'
c8volt get ut --assignee alice --var 'status="approved"' --limit 20
c8volt get ut --var 'status="approved"' --total
c8volt get ut --var 'status="approved"' --with-vars
```

Membership examples deliberately match existing PI syntax; do not reinterpret their values. All predicates target local task variables. Parent-only values cannot satisfy positive local matches; shadowed names are evaluated locally. Negative/existence predicates retain backend semantics without adding client-side missing-variable logic.

## Validation and selection

- Parse during command argument validation and return established invalid-flag errors before requests.
- Treat any explicitly supplied variable flag as a search selector for both explicit-key and stdin-key conflicts; preserve existing key-input discovery and error wording.
- Combine predicates with existing task selectors and tenant restrictions. Never discard contradictory predicates or silently change AND to OR.
- Keep existing `--total`, `--limit`, and output-mode conflicts; no new display dependency.
- Native search/count uses all predicates on every page, including sparse pages and capped-total traversal.
- Camunda 8.8/8.9/8.10 use existing native task search. Version 8.7 remains unsupported. Invalid/unsupported filters and backend failures cannot produce unfiltered fallback.

## Output and display

| Mode | Result contract | Effective-variable reads with `--with-vars` |
| --- | --- | --- |
| Human | Existing task rows, optional existing vars subtree, `found: N` | Selected tasks only |
| JSON | Exactly one existing successful collection envelope | Selected tasks only; received values preserved |
| Effective keys-only | One key per line; zero bytes when empty | None |
| Total | Existing exact numeric count line, including `0` | None |
| Quiet human | Existing suppressed output; errors remain failures | Existing requested enrichment behavior |
| Quiet JSON | Existing successful JSON envelope | Selected tasks only |
| Empty result | Existing command-appropriate empty payload/text | None |

Without `--with-vars`, all filter-only modes make zero per-task variable reads. JSON precedence over keys-only remains unchanged. Display limits/truncation labels do not affect filtering or JSON values. Do not change view structs, add filter echoes, or expose extra lifecycle details.

Interactive paging retains terminal-stdin eligibility even with redirected stdout, stderr destination inheritance/configuration, prompt wording/defaults, EOF/decline behavior, and auto-confirm/automation behavior. Empty completed scopes add no prompts. Later errors retain the existing failure path; a partially rendered human search must not emit a false successful final summary.
