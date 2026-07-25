# Quickstart: All-Command Integration Suite

This guide describes the intended validation flow for the all-command integration suite after implementation.

## Prerequisites

- A disposable local Camunda cluster for each version profile being validated.
- Default local c8volt configuration available under `$HOME/.config/c8volt`.
- Existing local profiles for the selected Camunda versions.
- The operator understands that selected clusters may be mutated, purged, repaired, resolved, cancelled, or deleted by the suite.

Do not point the suite at shared, production, customer, or non-disposable clusters.

## First Harness Slice

Run the inventory and read-only profile gate:

```sh
go test -tags=integration ./integration/cli -run 'TestInventory|TestProfiles|TestReadOnlySmoke' -count=1 -timeout=10m
```

Expected outcome:

- the c8volt binary is built once
- `capabilities --json` returns 55 command nodes
- every command path has a coverage entry
- selected profiles connect and report the expected Camunda versions
- read-only commands such as `version`, `capabilities`, `config validate`, `config test-connection`, and `get cluster ...` produce evidence

## Seeded Data Slice

Run the setup-oriented slice:

```sh
go test -tags=integration ./integration/cli -run 'TestSeededData' -count=1 -timeout=20m
```

Expected outcome:

- version-matched embedded BPMN fixtures are discovered
- process definitions are deployed through c8volt commands
- process instances are started through c8volt commands
- suite-created data receives a unique run marker where supported
- keys and resource IDs are written to the evidence directory
- dirty-cluster unrelated data does not fail the slice

## Command-Family Slices

Run one family at a time while building confidence:

```sh
go test -tags=integration ./integration/cli -run 'TestGetFamily' -count=1 -timeout=20m
go test -tags=integration ./integration/cli -run 'TestWalkFamily' -count=1 -timeout=20m
go test -tags=integration ./integration/cli -run 'TestUpdateFamily' -count=1 -timeout=20m
go test -tags=integration ./integration/cli -run 'TestCancelDeleteResolveFamilies' -count=1 -timeout=30m
go test -tags=integration ./integration/cli -run 'TestOps' -count=1 -timeout=45m
```

Expected outcome:

- each covered command writes evidence
- every command-local flag in the manifest is exercised by at least one scenario
- destructive workflows run preview first where supported
- confirmed destructive workflows use disposable targets and record affected data
- direct Camunda setup fallbacks generate proposal records
- missing embedded BPMN needs generate proposal records

## Full Suite

Run the complete suite:

```sh
go test -tags=integration ./integration/cli -count=1 -timeout=60m
```

Expected outcome:

- all 55 command nodes are accounted for
- all command families have evidence
- example validation reports pass/fail results with source locations
- final summary distinguishes product failures, harness setup failures, missing command support, missing fixture support, and environment availability failures

## Rerun Evidence

Use a stable work directory to compare repeated runs:

```sh
export C8VOLT_IT_WORKDIR=/tmp/c8volt-all-command-it
go test -tags=integration ./integration/cli -count=1 -timeout=60m
```

Review:

- `summary.md`
- `coverage.json`
- `examples.json`
- `proposals-command.json`
- `proposals-embedded-bpmn.json`
- `logs/`
- `data/`
