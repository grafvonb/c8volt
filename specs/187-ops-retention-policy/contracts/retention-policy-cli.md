# CLI Contract: Ops Execute Retention Policy

## Command

```bash
c8volt ops execute retention-policy [flags]
```

## Grouping Command

```bash
c8volt ops execute
```

Behavior:

- Shows help/discovery output.
- Performs no retention cleanup directly.
- Defines no retention-policy-specific flags at the grouping level.

## Required Flags And Modes

Required workflow flag:

- `--retention-days <days>`: non-negative integer mapped to existing relative end-date selection semantics equivalent to `--end-date-older-days <days>`

Inherited/root behavior:

- `--automation`
- `--auto-confirm`
- `--json`
- verbose/logging/config flags
- existing output modes supported by command contracts

Workflow behavior:

- `--dry-run`
- `--report-file <path>`
- `--report-format markdown|json`
- compatible process-instance search filters: `--bpmn-process-id`, `--pd-key`, `--pd-version`, `--pd-version-tag`, `--state`, `--parent-key`, `--roots-only`, `--children-only`, `--incidents-only`, `--no-incidents-only`, `--limit`, and `--batch-size`
- compatible delete execution controls: `--workers`, `--no-worker-limit`, `--fail-fast`, `--no-wait`, `--no-state-check`, and `--force` only if it preserves existing `delete pi` semantics

Unsupported workflow behavior:

- Explicit process-instance `--key` selection
- Shell composition as implementation behavior
- Camunda native retention policy or batch deletion APIs

## Human Output Contract

Human output MUST follow the compact ops rhythm:

1. retention discovery
2. delete plan
3. deletion
4. outcome
5. report

Human output MUST include:

- retention days and derived boundary when available
- selection filters used
- discovered retention seed count
- resolved root count
- affected process-instance count
- duplicate handling summary
- non-final affected instance count
- missing ancestor or traversal notices when relevant
- deletion execution status when mutation is attempted
- final outcome
- clear no-target message when no retention-eligible process instances are found
- `report: written <path>` when a report file is written

Detailed key lists SHOULD be shown only in verbose output unless the selected output mode makes keys the primary output.

## JSON Output Contract

JSON output MUST use the shared command result envelope where applicable and include:

- request metadata
- discovery step
- delete plan step
- deletion execution step when applicable
- final outcome
- notices
- errors array when errors occurred

Step statuses MUST use shared ops vocabulary:

- `planned`
- `skipped`
- `submitted`
- `confirmed`
- `confirmation_failed`
- `failed`
- existing shared blocked/local-precondition status may be used where appropriate

`--automation --json` MUST keep stdout deterministic machine-readable JSON. Progress, diagnostics, and logs belong on stderr or should be suppressed according to existing patterns.

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
- tenant id when available
- retention days
- derived end-date boundary when available
- selection filters used
- discovery status
- discovered retention seed count and keys
- resolved root count and keys
- affected process-instance count and keys
- duplicate handling summary
- final-state selected keys and states
- non-final affected keys and states
- missing ancestors and traversal warnings
- auto-confirm flag
- automation flag
- no-wait flag
- no-state-check flag
- force flag when supported
- per-key or per-batch delete status
- all errors encountered
- final outcome: `planned`, `deleted`, `partially_failed`, or `failed`

Existing report files MUST be preserved for dry-run, aborted, unconfirmed, or locally blocked runs unless overwrite is already allowed by destructive confirmation or non-interactive confirmation.

## Error Contract

- Missing, negative, or non-integer `--retention-days` fails locally with invalid-input behavior and `exitcode.InvalidArgs`.
- Unsupported explicit `--key` selection fails locally with invalid-input behavior and `exitcode.InvalidArgs`.
- `--dry-run` never prompts and never mutates.
- `--automation --json` is supported without `--auto-confirm` for this command when command metadata declares full automation support.
- Runtime/local precondition failures after planning or preflight use `localPreconditionError` and `exitcode.Error`.
- Discovery errors are reported in the normal command error path and in requested report files when a report can be built.
- Report write failures are surfaced as command failures after cleanup outcome is known.
