# Contract: Evidence And Gap Reporting

> Note: The original 255 implementation generated proposal JSON files from integration tests. That runtime proposal pattern is now deprecated by `specs/integration-test-responsibility.md`. Keep this contract as historical context for 255 evidence, but do not extend generated proposal reports in new or corrected integration work.

## Evidence Directory

Each suite run writes evidence outside generated documentation. The default location should be a temporary work directory, with an explicit environment override for reruns.

Required files:

- `inventory.json`: captured command contract
- `profiles.json`: selected profile readiness and version checks
- `run.json`: run marker and suite metadata
- `coverage.json`: command coverage results
- `examples.json`: help and generated CLI example validation results
- `proposals-command.json`: legacy direct Camunda setup fallback proposals
- `proposals-embedded-bpmn.json`: legacy missing embedded model proposals
- `summary.md`: human-readable run summary
- `logs/`: per-command stdout and stderr evidence
- `data/`: selected keys, resource IDs, and run-created identifiers

## Evidence Record Shape

Each command scenario record must include:

```json
{
  "commandPath": "get process-instance",
  "scenarioName": "search by run marker",
  "profile": "local-c89",
  "camundaVersion": "8.9",
  "arguments": ["get", "process-instance", "--json"],
  "stdinPath": "",
  "stdoutPath": "logs/get-process-instance/search.stdout",
  "stderrPath": "logs/get-process-instance/search.stderr",
  "exitCode": 0,
  "startedAt": "2026-07-25T00:00:00Z",
  "finishedAt": "2026-07-25T00:00:01Z",
  "dataOwnership": ["seeded"],
  "resourceKeys": ["2251799813685250"],
  "outcome": "pass",
  "failureClass": ""
}
```

Allowed `outcome` values:

- `pass`
- `fail`
- `skipped`
- `blocked`

Allowed `failureClass` values:

- `product`
- `harness_setup`
- `missing_fixture_support`
- `missing_command_support`
- `environment_availability`

Allowed `dataOwnership` values:

- `seeded`
- `preexisting`
- `mutated`
- `retained`
- `cleanup_failed`

## Legacy Proposal Record Shape

Each direct setup or fixture gap must include:

```json
{
  "kind": "command",
  "requiredState": "listener job attached to runtime element",
  "coverageNeed": "walk process-instance --with-listeners",
  "fallbackUsed": "direct Camunda setup",
  "affectedCommands": ["walk process-instance", "get element"],
  "affectedVersions": ["8.8", "8.9"],
  "operatorValue": "Operators can create or inspect listener-oriented fixtures without direct API setup."
}
```

`kind` values:

- `command`
- `embedded_bpmn`

## Reporting Rules

- Every required command scenario must have an evidence record.
- Failed records must include a non-empty failure class.
- Destructive scenarios must record known affected keys when the CLI output provides them.
- Cleanup failures must be visible and must not overwrite the command's original result.
- Legacy proposal reports must be empty JSON arrays when no proposals are generated.
- New work should maintain setup and fixture gaps in spec-owned artifacts instead of generating proposal records from tests.
