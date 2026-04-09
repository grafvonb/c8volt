# Contract: Process-Instance Management Date Filters

## Command Surface

Commands:

```bash
./c8volt cancel process-instance [existing selector flags] [date filter flags]
./c8volt delete process-instance [existing selector flags] [date filter flags]
```

New flags on both commands:

| Flag | Value | Meaning |
|------|-------|---------|
| `--start-date-after` | `YYYY-MM-DD` | Inclusive lower bound for process instance `startDate` |
| `--start-date-before` | `YYYY-MM-DD` | Inclusive upper bound for process instance `startDate` |
| `--end-date-after` | `YYYY-MM-DD` | Inclusive lower bound for process instance `endDate` |
| `--end-date-before` | `YYYY-MM-DD` | Inclusive upper bound for process instance `endDate` |

## Validity Rules

| Rule | Expected behavior |
|------|-------------------|
| Date values must be date-only input | Reject invalid values before search or management begins |
| `after` and `before` can be combined per field | Treat them as inclusive range bounds |
| `after > before` for the same field | Reject with validation error |
| Any date flag combined with explicit `--key` | Reject with invalid-combination error |
| Date flags used on v8.7 search path | Return not-implemented error through existing error path |

## Search-Based Selection Semantics

| Input combination | Required behavior |
|------------------|-------------------|
| `--start-date-after D` | Select only instances with `startDate >= D` |
| `--start-date-before D` | Select only instances with `startDate <= D` |
| Both start-date flags | Select only instances with `startDate` inside inclusive range |
| `--end-date-after D` | Select only instances with `endDate >= D` |
| `--end-date-before D` | Select only instances with `endDate <= D` |
| Both end-date flags | Select only instances with `endDate` inside inclusive range |
| Any end-date filter on instance with no `endDate` | Exclude that instance from the selected set |
| Date filters plus existing filters | Apply all constraints together as narrowing logic |

## Management Behavior

| Command mode | Required behavior |
|--------------|-------------------|
| Search mode without explicit keys | Search for matching process instances first, then pass the selected keys into the existing cancel/delete workflow |
| Explicit key mode without date filters | Keep current key-based behavior unchanged |
| Explicit key mode with date filters | Reject before any search, cancellation, or deletion occurs |
| Search mode with no matches | Fail with the command’s existing no-target-found behavior |

## Output Expectations

- Cancel/delete output shape stays identical to the current command output and confirmation flow.
- JSON and text output formats do not change.
- The only user-visible difference is which instances are selected or whether invalid/unsupported input is rejected before action begins.
