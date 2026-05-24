# Contract: Ops Paged Discovery Scope

## Affected Commands

- `c8volt ops purge process-instances-with-incidents`
- `c8volt ops purge pi-with-incidents`
- `c8volt ops purge piwi`
- `c8volt ops repair incident`
- `c8volt ops repair inc`
- `c8volt ops repair process-instance`
- `c8volt ops repair pi`
- `c8volt ops purge all-process-definitions`
- `c8volt ops purge all-pds`
- `c8volt ops purge apd`

## Flag Semantics

| Flag | Contract |
|------|----------|
| `--batch-size`, `-n` | Controls per-page discovery size only. It does not cap total frozen scope. |
| `--limit`, `-l` | Caps total matching candidates frozen for preview, confirmation, mutation, automation, and reports. |
| `--dry-run` | Uses the same frozen scope as mutation would use but submits no mutation. |
| `--auto-confirm` / `--automation` | Skips interactive prompt according to existing rules while still using a single frozen scope. |

## Discovery Status Output

Affected JSON and report outputs must expose stable discovery status fields equivalent to:

```json
{
  "discovery": {
    "complete": true,
    "limited": false,
    "limit": 0,
    "batchSize": 250,
    "pages": 4,
    "candidatesFrozen": 6924
  }
}
```

When `--limit N` stops discovery:

```json
{
  "discovery": {
    "complete": false,
    "limited": true,
    "limit": 5,
    "batchSize": 250,
    "pages": 1,
    "candidatesFrozen": 5
  }
}
```

Exact field placement may follow the existing workflow result shapes, but each affected workflow must expose the same meaning for complete vs. user-limited scope.

## Human And Markdown Report Contract

Human and Markdown report output must identify one of:

- `discovery complete`
- `discovery user-limited`

The output must keep existing counts for incidents, process instances, process definitions, affected trees, skipped candidates, duplicates, and mutation results.

## Frozen Scope Contract

Interactive confirmation must display counts from the frozen candidate set. After the operator confirms, mutation must receive only the frozen candidate keys. No confirmed mutation path may perform another search-mode discovery to expand or change the approved scope.

## Compatibility Contract

- Existing keyed and stdin modes keep their current frozen-scope semantics.
- Existing unsupported-version behavior is preserved.
- Existing report schemas may be bumped if needed, but old fields must keep their current meanings.
