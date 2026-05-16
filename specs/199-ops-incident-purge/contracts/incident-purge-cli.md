# CLI Contract: Ops Purge Process Instances With Incidents

## Command

```bash
c8volt ops purge process-instances-with-incidents [flags]
```

## Alias

```bash
c8volt ops purge pi-with-incidents [flags]
```

The alias must behave exactly like the full command. The alias `incident-pis` must not be added.

## Grouping Command

```bash
c8volt ops purge
```

Behavior:

- Shows help/discovery output.
- Performs no cleanup directly.
- Defines no incident-purge-specific flags at the grouping level.

## Supported Flags And Modes

Incident selection flags:

- `--key`: incident key
- `--state`
- `--error-type`
- `--error-message`
- `--bpmn-process-id`
- `--pd-key`
- `--pi-key`
- `--root-key`
- `--flow-node-id`
- `--fni-key`
- `--creation-time-after`
- `--creation-time-before`
- `--batch-size`
- `--limit`

Workflow and inherited behavior:

- `--dry-run`
- `--report-file <path>`
- `--report-format markdown|json`
- `--workers`
- `--no-worker-limit`
- `--fail-fast`
- `--no-wait`
- `--force`
- `--automation`
- `--auto-confirm`
- `--json`
- verbose/logging/config flags
- existing output modes supported by command contracts

Unsupported workflow behavior:

- Incident display-only flags such as `--pi-keys-only`, `--total`, `--error-message-limit`, or `--with-no-error-message`
- Shell composition as implementation behavior
- Direct incident deletion or resolution behavior

## Human Output Contract

Human output MUST follow the compact ops rhythm:

1. incident discovery
2. delete plan
3. deletion
4. outcome
5. report

Normal dry-run output should be shaped like:

```text
dry run: purge process-instances with incidents
selection filters: {...}
candidate incidents: N
candidate process instances: M
delete plan: planned (candidate process instances: M, roots: R, affected process instances: A)
outcome: planned; no changes applied; use --verbose to list process-instance keys
```

Normal destructive output should be shaped like:

```text
purge process-instances with incidents
selection filters: {...}
candidate incidents: N
candidate process instances: M
delete plan: planned (candidate process instances: M, roots: R, affected process instances: A)
deletion: submitted (requests: R)
deletion confirmation: true
outcome: deleted
```

Human output MUST use candidate terminology:

- `candidate incidents`
- `candidate process instances`
- `duplicate candidate process instances`
- `resolved root keys`
- `affected process-instance keys`

Detailed key lists SHOULD be shown only in verbose output unless the selected output mode makes keys the primary output.

Human output MUST print `report: written <path>` after a report file is successfully written.

## JSON Output Contract

JSON output MUST use the shared command result envelope where applicable and include:

- request metadata
- incident discovery step
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
- `blocked`
- `failed`

`--automation --json` MUST keep stdout deterministic machine-readable JSON. Progress, diagnostics, and logs belong on stderr or should be suppressed according to existing patterns.

## Report File Contract

`--report-file` writes an audit report after cleanup finishes or after a failure where report data can be built.

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
- incident selection filters used
- discovery status
- candidate incident count and keys
- candidate process-instance count and keys
- duplicate candidate process-instance summary
- skipped incident summary
- resolved root count and keys
- affected process-instance count and keys
- final-state selected keys and states
- non-final affected keys and states
- missing ancestors and traversal warnings
- auto-confirm flag
- automation flag
- no-wait flag
- force flag
- fail-fast and no-worker-limit flags
- per-key or per-batch delete status
- all errors and notices encountered
- final outcome: `planned`, `deleted`, `partially_failed`, or `failed`

Existing report files MUST be preserved for dry-run, aborted, unconfirmed, or locally blocked runs unless overwrite is already allowed by destructive confirmation or non-interactive confirmation.

## Error Contract

- Invalid flag values, forbidden flag combinations, and missing dependent flags fail locally with invalid-input behavior and `exitcode.InvalidArgs`.
- Existing report files discovered as runtime/local precondition failures after planning or preflight use `localPreconditionError` and `exitcode.Error`.
- `--dry-run` never prompts and never mutates.
- `--automation --json` is supported without `--auto-confirm` for this command when command metadata declares full automation support.
- No-target runs submit no delete request and should not be reported as destructive failure.
- Discovery errors are reported in the normal command error path and in requested report files when a report can be built.
- Report write failures are surfaced as command failures after cleanup outcome is known.
