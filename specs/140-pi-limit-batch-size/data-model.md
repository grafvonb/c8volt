# Data Model: Process-Instance Limit and Batch Size Flags

## Process-Instance Batch Size

Represents the per-page fetch or processing size used by search-driven process-instance commands.

- **Source**: `--batch-size`, `-n`, shared process-instance page-size configuration, or default maximum.
- **Validation**: Positive value within the server-supported maximum, using the same rules as the previous paging-size flag.
- **Applies To**: Search-mode `get process-instance`, `cancel process-instance`, and `delete process-instance`.
- **Does Not Mean**: Total number of process instances to return or process.

## Process-Instance Match Limit

Represents the maximum total number of matched process instances that may be returned or processed during one command execution.

- **Source**: `--limit` or `-l`.
- **Validation**: Positive integer when provided.
- **Applies To**: Search/list mode only.
- **Rejected With**: Direct `--key` workflows.
- **Rejected With**: `--total`, which remains count-only output rather than limited detail output.

## Remaining Limit Window

Represents how many additional matched process instances may still be returned or processed before the configured limit is satisfied.

- **Initial Value**: The configured process-instance match limit.
- **Updated By**: The number of limited, filtered process instances returned or processed per page.
- **Completion Rule**: When it reaches zero, no later pages are fetched and no continuation prompt is shown.

## Limited Page Result

Represents the page subset that is safe to render or process after applying the remaining limit window.

- **Input**: A backend search page after existing process-instance local filters are applied.
- **Output**: Up to the remaining limit's number of process instances.
- **Consumers**: `get` rendering/aggregation, cancel planning, delete planning.

## Limit Stop Reason

Represents the user-facing stop state when execution ends because the configured total match limit was reached.

- **Displayed In**: Verbose one-line progress output and any existing progress summary surface.
- **Distinct From**: No more matches, user-aborted continuation, and warning stop.
