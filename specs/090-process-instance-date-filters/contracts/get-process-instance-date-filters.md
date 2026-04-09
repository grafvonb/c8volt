# Contract: `c8volt get process-instance` Date Filters

## Command Surface

Command:

```bash
./c8volt get process-instance [existing search flags] [date filter flags]
```

New flags:

| Flag | Value | Meaning |
|------|-------|---------|
| `--start-date-after` | `YYYY-MM-DD` | Inclusive lower bound for process instance `startDate` |
| `--start-date-before` | `YYYY-MM-DD` | Inclusive upper bound for process instance `startDate` |
| `--end-date-after` | `YYYY-MM-DD` | Inclusive lower bound for process instance `endDate` |
| `--end-date-before` | `YYYY-MM-DD` | Inclusive upper bound for process instance `endDate` |

## Validity Rules

| Rule | Expected behavior |
|------|-------------------|
| Date values must be date-only input | Reject invalid values before executing search |
| `after` and `before` can be combined per field | Treat them as inclusive range bounds |
| `after > before` for the same field | Reject with validation error |
| New date flags used with `--key` | Reject because date filters apply only to list/search behavior |
| New date flags used on v8.7 | Return not-implemented error through existing error path |

## Search Semantics

| Input combination | Required behavior |
|------------------|-------------------|
| `--start-date-after D` | Return only instances with `startDate >= D` |
| `--start-date-before D` | Return only instances with `startDate <= D` |
| Both start-date flags | Return only instances with `startDate` inside inclusive range |
| `--end-date-after D` | Return only instances with `endDate >= D` |
| `--end-date-before D` | Return only instances with `endDate <= D` |
| Both end-date flags | Return only instances with `endDate` inside inclusive range |
| Any end-date filter on instance with no `endDate` | Exclude that instance from results |
| Date filters plus existing filters | Apply all constraints together as narrowing logic |

## Versioned Service Mapping

| Version | Mapping |
|---------|---------|
| `v8.8` | Map date bounds to native process-instance search filter datetime comparisons |
| `v8.7` | Reject any date-filtered request as not implemented |

## Output Expectations

- Output shape stays identical to existing `get process-instance` output modes.
- JSON output continues to use the existing serialized process instance model keys.
- The only user-visible behavioral difference is that results are narrowed by the new filters or rejected early for invalid/unsupported usage.
