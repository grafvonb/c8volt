# CLI Contract: Ops Repair Workflows

## Command Surface

| Command | Purpose | Mutation |
| --- | --- | --- |
| `c8volt ops repair` | Grouping command for repair/remediation workflows | None directly |
| `c8volt ops repair incident` | Repair incidents selected by explicit keys, stdin keys, or incident filters | State-changing unless `--dry-run` |
| `c8volt ops repair process-instance` | Repair active incidents associated with selected process instances | State-changing unless `--dry-run` |

## Shared Flags

| Flag | Applies To | Contract |
| --- | --- | --- |
| `--dry-run` | both targets | Discover and validate only; submit no mutations |
| `--vars <json-object>` | both targets | Parse and validate like `update pi`; update process-instance scopes |
| `--vars-file <path>` | both targets | Parse and validate like `update pi`; update process-instance scopes |
| `--retries <n>` | both targets | Default is `1` for job-backed incidents; `0` skips retry restoration |
| `--job-timeout <duration>` | both targets | Applies only when an incident has a related job key |
| `--workers <n>` | both targets | Maximum concurrent workers for applicable bulk work |
| `--fail-fast` | both targets | Stop scheduling new work after first error where supported |
| `--no-worker-limit` | both targets | Use all queued work as workers when `--workers` is unset |
| `--batch-size <n>` | search modes | Page size for discovery |
| `--limit <n>` | search modes | Maximum matching records to inspect |
| `--auto-confirm` | both targets | Skip interactive confirmation when mutation would otherwise prompt |
| `--automation` | both targets | Enable unattended behavior and deterministic machine contracts |
| `--report-file <path>` | both targets | Write audit report |
| `--report-format markdown|json` | both targets | Select report rendering format |

## Incident Repair Input

Incident repair accepts:

- repeated `--key <incident-key>`
- stdin keys with positional `-`
- incident filters from `get incident`: `--state`, `--error-type`, `--error-message`, `--pi-key`, `--root-key`, `--pd-key`, `--bpmn-process-id`, `--flow-node-id`, `--fni-key`, `--creation-time-after`, `--creation-time-before`, `--batch-size`, and `--limit`

Contract:

- Keyed mode and search-filter mode are mutually exclusive.
- Key validation fails before remote mutation.
- Discovery freezes incident keys before repair starts.

## Process-Instance Repair Input

Process-instance repair accepts:

- repeated `--key <process-instance-key>`
- stdin keys with positional `-`
- process-instance filters from `get pi`: `--bpmn-process-id`, `--pd-key`, `--pd-version`, `--pd-version-tag`, `--state`, `--parent-key`, `--roots-only`, `--children-only`, `--incidents-only`, `--direct-incidents-only`, `--start-date-after`, `--start-date-before`, `--start-date-older-days`, `--start-date-newer-days`, `--end-date-after`, `--end-date-before`, `--end-date-older-days`, `--end-date-newer-days`, `--batch-size`, and `--limit`

Contract:

- Keyed mode and search-filter mode are mutually exclusive.
- Search mode requires `--incidents-only` or `--direct-incidents-only`.
- Discovery freezes process-instance keys and deduped incident keys before repair starts.

## Output Contract

Human output must be compact and scan-friendly. JSON output must be deterministic under `--automation --json`.

Required structured fields:

- command name
- dry-run flag
- discovery mode
- discovery filters or input keys
- frozen incident keys
- process-instance keys
- variable scopes and requested variable names
- job keys where present
- job applicability per incident
- requested retries and timeout
- per-step statuses
- notices
- errors
- final outcome
- report path and format when requested

## Report Contract

Report formats:

- `json`: structured report model
- `markdown`: Markdown rendering of the structured report model

Report outcomes:

- `planned`
- `repaired`
- `partially_failed`
- `failed`

Step statuses:

- `planned`
- `skipped`
- `not_applicable`
- `submitted`
- `confirmed`
- `confirmation_failed`
- `blocked`
- `failed`

## Error Contract

- Invalid key values fail before remote mutation.
- Mutually exclusive keyed and search modes fail before remote mutation.
- Process-instance search without an incident-bearing selector fails before remote mutation.
- Variable parse or validation failures fail before mutation.
- Variable update failure blocks dependent incident resolution.
- Missing related job keys do not fail repair; job steps are `not_applicable`.
- Report write failures are surfaced with the workflow result when possible.
