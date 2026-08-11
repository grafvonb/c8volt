# CLI Watch Contract: Process Definition Watch Mode

## Scope

This contract defines observable behavior for `c8volt get process-definition --watch` and aliases `get pd --watch` / `get pds --watch`.

## Flags

| Flag | Contract |
|------|----------|
| `--watch` | Repeats the selected process-definition lookup until interrupted, timed out, retry-exhausted, or fatally failed. |
| `--watch-interval <duration>` | Sets the fixed interval between snapshot attempts after the immediate first snapshot. Must be a positive duration. |
| `--backoff-timeout` | Limits the overall watch run when configured for the command. |
| `--backoff-max-retries` | Controls consecutive transient failure tolerance. A successful snapshot resets the consecutive failure count. |
| `--json` | Incompatible with `--watch`; reject before lookup work begins. |
| `--keys-only` | Incompatible with `--watch`; reject before lookup work begins. |
| `--xml` | Incompatible with `--watch`; reject before lookup work begins. |
| `--quiet` | Incompatible with `--watch`; reject before lookup work begins. |
| `--automation` | Incompatible with `--watch`; reject before lookup work begins. |
| `--key` | Compatible with `--watch`; each snapshot observes the selected key. |
| `--stat` | Compatible with `--watch` where existing stat support applies. |
| `--batch-size` | Controls page size for broad or filtered search snapshots; does not cap total snapshot rows. |
| `--verbose` | Compatible with `--watch`; may add durable human context without changing snapshot rows. |

## Default Behavior

- `--watch` emits the first snapshot immediately.
- If `--watch-interval` is omitted, the interval is `1s`.
- If no selector is provided, watch mode observes all process definitions visible to the command context.
- Watch mode is a human-output feature only.
- JSON, keys-only, XML, quiet, and automation combinations fail before lookup work.
- Non-watch invocations keep existing selector validation and output behavior.
- Empty snapshots are successful watch results.

## Snapshot Contract

Each snapshot is the complete result of the selected lookup at one point in time.

Rules:

- Broad and filtered search snapshots include all pages needed for the current result.
- Key snapshots contain the selected process definition when it is visible or fail according to existing key lookup behavior.
- Latest snapshots apply the existing latest-version selector on every tick.
- Stat snapshots apply the existing stat behavior and unsupported-version handling on every tick.
- Snapshot ordering follows existing non-watch process-definition list ordering.
- A changed population between snapshots is reported as separate current snapshots; prior snapshots are not revised.

## Output Channel Rules

| Mode | stdout | stderr/activity | Required behavior |
|------|--------|-----------------|-------------------|
| Default human | Snapshot result rows and human summary | Transient activity and retry/status context allowed | Output is compact and snapshot-oriented; low-level endpoint/cursor detail is omitted. |
| Verbose human | Snapshot result rows and human summary | Durable watch context and progress allowed | May show interval, tick, retry, or page context without changing result rows. |
| Debug | Selected output plus debug logging according to existing rules | Debug logs may include low-level detail | Debug detail stays out of default human output. |
| JSON | N/A | N/A | `--json --watch` is rejected before lookup work. |
| Keys-only | N/A | N/A | `--keys-only --watch` is rejected before lookup work. |
| XML | N/A | N/A | `--xml --watch` is rejected before lookup work. |
| Quiet | N/A | N/A | `--quiet --watch` is rejected before lookup work. |
| Automation | N/A | N/A | `--automation --watch` is rejected before lookup work. |

## Incompatible Output Contract

Watch mode is intentionally not a machine-output stream.

Rules:

- Reject `--json --watch` because JSON output remains one stable document per command invocation.
- Reject `--keys-only --watch` because keys-only output remains a finite pipeline format.
- Reject `--xml --watch` because XML output remains a single artifact.
- Reject `--quiet --watch` because watch requires visible human snapshots.
- Reject `--automation --watch` because automation mode is non-interactive and deterministic.
- Rejections must happen before process-definition lookup work.
- Rejection errors must be clear local validation errors.

## Human Watch Contract

Human output may include compact snapshot boundaries and existing process-definition rows.

Example shape:

```text
snapshot 1:
2251799813685255 tenant invoice v3
found: 1

snapshot 2:
2251799813685255 tenant invoice v3
2251799813685256 tenant payment v1
found: 2
```

Rules:

- Snapshot boundaries are human-only.
- Empty snapshots should be understandable, for example `found: 0`.
- Default human output must not include endpoint names, request bodies, raw cursors, or per-page lifecycle detail.

## Retry And Termination Contract

- Transient lookup failures increment the consecutive retry count.
- Successful snapshots reset the consecutive retry count.
- Failure status is reported away from result stdout when stdout carries data.
- Retry exhaustion stops watch mode with a clear error and non-success exit status.
- Operator interruption stops promptly and does not claim lookup failure solely because watch ended.
- Timeout stops the session according to existing command timeout behavior.

## Documentation Contract

Help and generated docs must describe:

- `--watch`
- `--watch-interval`
- Default `1s` interval
- Broad missing-selector behavior
- Human-output-only watch behavior
- JSON incompatibility
- Keys-only incompatibility
- XML incompatibility
- Quiet incompatibility
- Automation incompatibility
- Default retry behavior and retry budget reset
- Timeout behavior
- `--batch-size` behavior for broad watch snapshots
