# CLI Contract: Ops Purge Orphan Process Instances

## Command

```bash
c8volt ops purge orphan-process-instances [flags]
```

## Grouping Command

```bash
c8volt ops purge
```

Behavior:

- Shows help/discovery output.
- Performs no cleanup directly.
- Defines no orphan-process-instance-specific flags at the grouping level.

## Required Flags And Modes

Inherited/root behavior:

- `--automation`
- `--auto-confirm`
- `--json`
- existing output modes supported by command contracts
- existing worker/backoff/config flags where applicable

Workflow behavior:

- `--dry-run`
- `--report-file <path>`
- `--report-format markdown|json`
- compatible process-instance search filters from orphan-child discovery, including process-definition selectors, state, date bounds, parent/child-compatible filters, batch size, and limit where existing behavior supports them

## Human Output Contract

Human output MUST include:

- selection filters used
- discovered orphan child process-instance count
- discovered keys when appropriate for existing output mode
- deletion plan status
- deletion execution status when mutation is attempted
- final outcome
- clear no-target message when no orphan child process instances are found

## JSON Output Contract

JSON output MUST use the shared command result envelope where applicable and include:

- request metadata
- discovery step
- deletion plan step
- deletion execution step when applicable
- final outcome
- errors array when errors occurred

Step statuses MUST use shared ops vocabulary:

- `planned`
- `skipped`
- `submitted`
- `confirmed`
- `confirmation_failed`
- `failed`
- existing shared `blocked` status may be used for local precondition failures where appropriate

## Report File Contract

`--report-file` writes an audit report after cleanup finishes or after a failure that occurs after discovery.

`--report-format` behavior:

- explicit `markdown` renders Markdown
- explicit `json` renders deterministic JSON
- omitted format infers `.json`, `.md`, or `.markdown`
- omitted format defaults to Markdown when extension does not identify a supported format

Report content MUST include:

- schema/version
- command name
- started timestamp
- finished timestamp
- duration
- dry-run flag
- c8volt version when available
- configured Camunda version
- safe profile/config identity
- selection filters used
- orphan discovery status
- discovered count
- discovered process-instance keys
- delete requested flag
- auto-confirm flag
- automation flag
- per-key or per-batch delete status
- all errors encountered
- final outcome: `planned`, `deleted`, `partially_failed`, or `failed`

## Error Contract

- `--dry-run` never prompts and never mutates.
- `--automation` without `--auto-confirm` fails before mutation when deletion targets exist.
- Discovery errors are reported in the normal command error path and in requested report files when a report can be built.
- Report write failures are surfaced as command failures after cleanup outcome is known.
