# Contract: Process-Instance Relative Day Filters

## Command Surface

Commands:

```bash
./c8volt get process-instance [existing search flags] [relative day flags]
./c8volt cancel process-instance [existing selector flags] [relative day flags]
./c8volt delete process-instance [existing selector flags] [relative day flags]
```

New flags on all three commands:

| Flag | Value | Meaning |
|------|-------|---------|
| `--start-date-older-days` | non-negative integer | Inclusive lower bound for `startDate`, derived from today minus `N` days |
| `--start-date-newer-days` | non-negative integer | Inclusive upper bound for `startDate`, derived from today minus `N` days |
| `--end-date-older-days` | non-negative integer | Inclusive lower bound for `endDate`, derived from today minus `N` days |
| `--end-date-newer-days` | non-negative integer | Inclusive upper bound for `endDate`, derived from today minus `N` days |

## Validity Rules

| Rule | Expected behavior |
|------|-------------------|
| Relative day values must be non-negative integers | Reject invalid or negative values before any search-based action |
| `before` and `after` can be combined per field | Treat them as inclusive derived range bounds |
| Derived lower bound is later than derived upper bound | Reject with validation error |
| Relative flag mixed with absolute date flag for the same field | Reject with validation error |
| Any relative day flag combined with explicit `--key` on cancel/delete | Reject with invalid-combination error |
| Relative day flags used on v8.7 | Return not-implemented error through the existing error path |

## Derivation Semantics

| Input combination | Required behavior |
|------------------|-------------------|
| `--start-date-older-days N` | Behave as the equivalent of `startDate >= today - N days` |
| `--start-date-newer-days N` | Behave as the equivalent of `startDate <= today - N days` |
| Both start-day flags | Behave as an inclusive derived start-date range |
| `--end-date-older-days N` | Behave as the equivalent of `endDate >= today - N days` |
| `--end-date-newer-days N` | Behave as the equivalent of `endDate <= today - N days` |
| Both end-day flags | Behave as an inclusive derived end-date range |
| Any relative day filter | Use the configured Camunda environment’s local calendar day for derivation |
| Any relative end-day filter on instance with no `endDate` | Exclude that instance from the result or selected set |

## Command-Mode Behavior

| Command mode | Required behavior |
|--------------|-------------------|
| `get process-instance` search mode | Return only instances matching the derived absolute date bounds and existing filters |
| `cancel`/`delete` search mode | Search for matching instances first, then pass selected keys into the existing management workflow |
| `cancel`/`delete` direct key mode without relative flags | Keep current behavior unchanged |
| `cancel`/`delete` direct key mode with relative flags | Reject before any search, cancellation, or deletion occurs |

## Output Expectations

- Output formats for `get`, `cancel`, and `delete` remain identical to current command behavior.
- JSON serialization stays on the existing process-instance model keys.
- The only user-visible behavior changes are the new shortcut flags, their validation failures, and the narrowed or selected result set produced through the existing absolute date-filter model.
