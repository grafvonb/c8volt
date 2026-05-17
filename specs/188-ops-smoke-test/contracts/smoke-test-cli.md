# CLI Contract: ops execute smoke-test

## Command

```bash
c8volt ops execute smoke-test [flags]
```

`c8volt ops execute` remains a grouping command and does not execute the smoke test directly.

## Flags

| Flag | Type | Default | Behavior |
| --- | --- | --- | --- |
| `--count` | positive integer | `1` | Number of process instances to create from the deployed smoke-test definition |
| `-n` | positive integer | `1` | Shorthand for `--count` |
| `--workers` | integer | existing command default | Worker count passed to reusable creation/deletion behavior |
| `--no-worker-limit` | boolean | `false` | Allows worker settings beyond the normal limit where existing lower-level behavior supports it |
| `--fail-fast` | boolean | `false` | Stops worker-backed operations according to existing fail-fast behavior |
| `--no-cleanup` | boolean | `false` | Retains created process instances and deployed process definition |
| `--dry-run` | boolean | `false` | Plans only and submits no mutations |
| `--automation` | boolean | `false` | Non-interactive mode for agents/scripts |
| `--auto-confirm` | boolean | `false` | Human/script convenience confirmation flag |
| `--no-wait` | boolean | `false` | Applies existing no-wait behavior during cleanup |
| `--report-file` | path | empty | Writes an audit report when supplied |
| `--report-format` | `markdown` or `json` | inferred | Report format; requires `--report-file` |

## Human Output Contract

Human output must show:

- connectivity validation
- selected fixture
- deployment result
- process-instance creation count and created keys
- traversal status
- cleanup skipped, blocked, submitted, confirmed, failed, or passed state
- final outcome
- `report: written <path>` when a report is written

Human output should stay compact and follow the existing ops rhythm. Reports and JSON keep full details even when human output suppresses low-value detail.

## JSON Output Contract

JSON output must use the shared command result envelope where applicable and include structured per-step data:

- `request`
- `plan`
- `fixture`
- `deployment`
- `run`
- `walk`
- `cleanup`
- `report`
- `outcome`
- `errors`

For `--automation --json`, stdout must remain deterministic machine-readable JSON. Progress and logs must go to stderr or be suppressed according to existing patterns.

## Report Contract

`--report-file` writes a report if the command succeeds or fails after planning starts.

Format inference:

- `.json` -> JSON
- `.md` or `.markdown` -> Markdown
- otherwise Markdown

Report fields:

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
- selected fixture file
- BPMN process ID
- deployment status
- deployed process-definition key
- deployed process-definition version when available
- requested process-instance count
- created process-instance count
- created process-instance keys
- per-instance run status
- per-instance walk status
- traversal summary for each created process instance
- cleanup requested flag
- no-cleanup flag
- process-instance cleanup status
- process-definition cleanup eligibility status
- process-definition cleanup status
- auto-confirm flag
- automation flag
- no-wait flag
- all errors encountered
- final outcome

## Error And Exit Behavior

- Static CLI shape errors, such as invalid `--count`, unsupported `--report-format`, or `--report-format` without `--report-file`, use existing invalid-input helpers and `exitcode.InvalidArgs`.
- Runtime/local precondition failures discovered after planning or preflight, such as missing embedded fixture, report overwrite blockers, aborts, unsupported local workflow state, or unsafe cleanup blockers, use `localPreconditionError` and `exitcode.Error` where applicable.
- Cleanup blockers should be represented in JSON and reports even when the command exits with an error.

## Examples

```bash
c8volt ops execute smoke-test --dry-run
c8volt ops execute smoke-test -n 5
c8volt ops execute smoke-test --count 5
c8volt ops execute smoke-test --no-cleanup
c8volt ops execute smoke-test --dry-run --report-file smoke-test.md
c8volt ops execute smoke-test --count 10 --automation --json --report-file smoke-test.json --report-format json
```
