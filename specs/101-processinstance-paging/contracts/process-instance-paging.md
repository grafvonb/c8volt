# Contract: Process-Instance Paging and Overflow Handling

## Command Surface

Commands:

```bash
./c8volt get process-instance [existing search flags] [--count N] [--auto-confirm]
./c8volt cancel process-instance [existing selector flags] [--count N] [--auto-confirm]
./c8volt delete process-instance [existing selector flags] [--count N] [--auto-confirm]
```

Shared behavior:

| Input | Required behavior |
|------|-------------------|
| No `--count` override | Use the shared config default, or `1000` if no config override is set |
| `--count N` | Use `N` as the page size for the current execution |
| Search mode with more matches remaining and no `--auto-confirm` | Prompt before fetching/processing the next page |
| Search mode with more matches remaining and `--auto-confirm` | Continue automatically to the next page |
| User declines continuation prompt after processed pages exist | Stop normally and report partial completion |
| Overflow remains indeterminate after version-specific fallback | Stop and warn that more matching items may remain |

## Config Surface

| Setting | Scope | Required behavior |
|--------|-------|-------------------|
| Shared process-instance page-size config key under `app` | `get`, search-based `cancel`, search-based `delete` | Supplies the default page size whenever `--count` is not provided |

## Output Contract

Each completed page must clearly report:

- The page size used
- The current-page item count
- The cumulative processed count
- Whether more matching items remain
- Whether the command is prompting, auto-continuing, complete, partially complete, or stopping with a warning

## Version-Aware Overflow Contract

| Camunda version | Required behavior |
|----------------|-------------------|
| `8.8` | Use native search page metadata from the generated client as the preferred overflow signal |
| `8.7` | Use the repository-native fallback strategy because the current Operate response type does not expose equivalent page metadata |
| Any supported version where fallback still cannot prove exhaustion | Stop and warn instead of silently continuing or silently finishing |
| `8.9` | Remains out of scope for this feature |

## Mode Contract

| Command mode | Required behavior |
|-------------|-------------------|
| `get process-instance` search mode | Page through matching instances with the shared continuation model |
| `cancel process-instance` search mode | Search one page at a time, confirm/continue per page, then cancel only the processed page’s instances |
| `delete process-instance` search mode | Search one page at a time, confirm/continue per page, then delete only the processed page’s instances |
| Direct `--key` mode for `cancel` or `delete` | Keep current non-paged behavior unchanged |

## Validation Contract

| Rule | Required behavior |
|------|-------------------|
| Effective page size <= 0 | Reject or normalize through the existing page-size validation path before search execution |
| User declines continuation | Not an error; report partial completion |
| Final page with no more matches | Finish without another continuation prompt |
| Exact-boundary final page | Must not be misreported as overflow when the version-aware signal says no more matches remain |

## Documentation Contract

- `README.md` examples and explanations must match the shipped paging behavior.
- Generated CLI reference pages for `get process-instance`, `cancel process-instance`, and `delete process-instance` must be regenerated from updated Cobra help text.
