# Contract: Process-Instance Limit and Batch Size Flags

## Affected Commands

- `c8volt get process-instance`
- `c8volt get pi`
- `c8volt cancel process-instance`
- `c8volt cancel pi`
- `c8volt delete process-instance`
- `c8volt delete pi`

## Flags

### `--batch-size`, `-n`

- Controls per-page search fetch/process size.
- Replaces `--count` on the affected command paths.
- Keeps the previous `-n` short flag.
- May be combined with `--limit`.
- Uses the same numeric validation rules as the previous paging-size flag.

### `--limit`, `-l`

- Controls total matched process instances returned or processed across pages.
- Must be a positive integer.
- Applies only to search/list mode.
- Must be rejected with direct `--key` workflows.
- Must be rejected with `--total` because count-only output and limited detail output are mutually exclusive.

### Removed `--count`

- Must not be accepted by affected command paths.
- Must fail clearly through the repository's standard invalid-arguments behavior.
- Must not exist as a hidden alias.

## Search Execution Semantics

1. Resolve batch size from `--batch-size`, shared configuration, or default.
2. Resolve optional total match limit from `--limit`.
3. Fetch a search page using the resolved batch size.
4. Apply existing local result filters.
5. If a limit is active, truncate the filtered page to the remaining limit before rendering or destructive processing.
6. Add the returned or processed count to the cumulative limited total.
7. If the limit is reached, stop without fetching another page and without prompting for continuation.
8. If the limit is not reached, continue existing paging behavior based on overflow state and confirmation mode.

## Output Semantics

Progress output must distinguish these stop reasons:

- no more matches remained
- user aborted continuation
- configured limit was reached
- warning stop because remaining matches may be unknown

## Required Failure Cases

- `get pi --key 123 --limit 1`
- `cancel pi --key 123 --limit 1`
- `delete pi --key 123 --limit 1`
- `get pi --state active --limit 0`
- `get pi --state active --total --limit 10`
- `cancel pi --state active --limit -1`
- `delete pi --state completed --count 250`

## Documentation Contract

User-facing docs and examples for affected command paths must use:

- `--batch-size` / `-n`
- `--limit` / `-l`

They must not use:

- `--count` for affected command paths
