# CLI Contract: Ops Purge All Process Definitions

## Command

```bash
c8volt ops purge all-process-definitions [flags]
```

## Alias

```bash
c8volt ops purge all-pds [flags]
```

The alias must behave exactly like the full command. Broad aliases such as `purge-definitions` or `delete-all` must not be added.

## Grouping Command

```bash
c8volt ops purge
```

Behavior:

- Shows help/discovery output.
- Performs no cleanup directly.
- Defines no all-process-definitions-specific flags at the grouping level.

## Supported Flags And Modes

Process-definition selection flags:

- `--key`
- `--bpmn-process-id`
- `--pd-version`
- `--pd-version-tag`
- `--latest`

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
- verbose/logging/config/profile/tenant/timeout flags
- existing output modes supported by command contracts

Unsupported workflow behavior:

- Process-definition display-only flags such as `--xml` or `--stat`
- Shell composition as implementation behavior
- Direct deletion of process definitions outside the frozen candidate set

## Human Output Contract

Human output MUST follow the compact ops rhythm:

1. process-definition discovery
2. delete plan
3. deletion
4. outcome
5. report

Normal dry-run output should be shaped like:

```text
dry run: purge all process definitions
selection filters: {...}
candidate process definitions: N
invoice-process [v1: 240, v2: 180, v3: 0]
shipment-process [v1: 42, v2: 11]
delete plan: planned (candidate process definitions: N, affected process instances: A)
outcome: planned; no changes applied
```

Normal destructive output should be shaped like:

```text
purge all process definitions
selection filters: {...}
candidate process definitions: N
invoice-process [v1: 240, v2: 180, v3: 0]
delete plan: planned (candidate process definitions: N, affected process instances: A)
deletion: submitted (requests: N)
deletion confirmation: true
outcome: deleted
```

Human output MUST use candidate terminology:

- `candidate process definitions`
- `duplicate candidate process definitions`
- `affected process instances`
- `submitted process-definition deletes`

Normal human output SHOULD show BPMN process IDs with bracketed version-level affected process-instance counts. Full process-definition keys and affected process-instance keys belong in JSON/report output unless the selected output mode makes keys the primary output.

Human output MUST print `report: written <path>` after a report file is successfully written.

## JSON Output Contract

JSON output MUST use the shared command result envelope where applicable and include:

- request metadata
- process-definition discovery step
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
- process-definition selection filters used
- discovery status
- candidate process-definition count and keys
- BPMN process IDs and versions where available
- duplicate candidate process-definition summary
- delete preflight status
- affected process-instance count
- active affected process-instance count
- auto-confirm flag
- automation flag
- no-wait flag
- force flag
- fail-fast and no-worker-limit flags
- per-key delete status
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
