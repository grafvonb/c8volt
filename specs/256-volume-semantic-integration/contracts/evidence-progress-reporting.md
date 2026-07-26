# Contract: Evidence, Progress, And Reporting

## Evidence Files

Volume runs must write dedicated evidence files distinct from baseline behavior files.

Recommended file names:

```text
volume-<family>.json
volume-data-<family>.json
volume-progress-<family>.json
volume-pipelines-<family>.json
volume-ops-reports-<family>.json
```

The exact file set may vary by family, but each implemented volume target must produce a top-level `volume-<family>.json` summary.

## Volume Evidence Record

Each scenario record must include:

```json
{
  "commandPath": "delete process-instance",
  "scenarioName": "volume-delete-pi-dry-run-stdin",
  "profile": "local-c89",
  "camundaVersion": "8.9",
  "arguments": ["delete", "pi", "-", "--dry-run"],
  "stdinPath": "data/delete-stdin.keys",
  "stdoutPath": "logs/volume-delete-pi-dry-run-stdin.stdout",
  "stderrPath": "logs/volume-delete-pi-dry-run-stdin.stderr",
  "exitCode": 0,
  "startedAt": "2026-07-25T00:00:00Z",
  "finishedAt": "2026-07-25T00:00:02Z",
  "datasetCount": 25,
  "flagsUnderTest": ["dry-run", "stdin", "limit"],
  "outputMode": "one-line",
  "dataOwnership": ["seeded", "preexisting"],
  "resourceKeys": ["2251799813685250"],
  "postConditionChecks": ["no seeded process instance cancelled"],
  "outcome": "pass",
  "failureClass": ""
}
```

## Critical Flag Evidence

Critical flag evidence must record:

- flags under test
- expected semantic behavior
- observed semantic behavior
- post-condition check result
- affected resource count
- unsupported-version behavior where applicable

Dry-run records must include a no-mutation check. Limit records must include requested limit and observed returned count. Worker records must include requested worker mode and affected resource count. Volume records must include the requested dataset count.

## Pipeline Evidence

Pipeline evidence must record:

- producer command and output file
- consumer command and stdin file
- input shape
- duplicate count where applicable
- invalid count where applicable
- valid count where applicable
- dry-run or confirmed mutation state
- stdout cleanliness validation

Keys-only producer output must be validated before it is passed to the consumer.

## Progress Evidence

Progress evidence must record:

- command path and scenario
- human or machine mode
- whether transient activity was expected
- durable progress facts captured from logs
- final outcome line or submitted/no-wait wording
- elapsed time where available
- stdout cleanliness for machine modes

Human-mode progress can be proven through visible activity or durable verbose progress. Machine modes must prove absence of activity/progress text in stdout.

## Ops Report Evidence

Ops report evidence must validate:

- report path
- report format
- schema/version fields where present
- command and workflow identity
- profile and Camunda version context
- started/finished/duration fields
- dry-run state
- discovery completeness
- selection filters
- plan status
- execution status
- step status vocabulary
- notices and errors
- mutation accounting
- final outcome
- parity between JSON stdout and JSON report where both are produced

Markdown validation should check stable section headings and required facts, not brittle prose.

## Failure Classification

Allowed failure classes:

- `product`
- `harness_setup`
- `missing_fixture_support`
- `missing_command_support`
- `environment_availability`

Blocked setup gaps should produce skipped-prerequisite or dry-run runtime evidence when relevant. Missing command or embedded BPMN support belongs in spec-owned gap artifacts, not generated test proposal evidence.
