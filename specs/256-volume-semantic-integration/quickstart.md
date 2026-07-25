# Quickstart: Volume And Semantic CLI Integration Coverage

This guide describes how the follow-up volume suite should be validated after implementation.

## Prerequisites

- Disposable Camunda clusters configured in the operator's default local c8volt configuration.
- The selected profiles are safe for destructive, high-volume test data.
- The baseline all-command integration suite from feature 255 is available.
- The operator understands that volume targets may create, mutate, retain, cancel, delete, repair, resolve, or purge suite-owned and selected disposable data.

Do not run volume targets against shared, production, customer, or protected clusters.

## Baseline Gate

Before running volume targets, run a quick baseline check:

```sh
make integration-cli-get IT_GO_TEST_FLAGS=-v
```

Expected outcome:

- the integration harness builds or reuses the c8volt binary
- default local config is used
- verbose output shows scenario names, arguments, exit codes, durations, and evidence paths
- baseline evidence is written outside `docs/`

## Volume Target Examples

Run one family volume target:

```sh
make integration-cli-get-volume IT_GO_TEST_FLAGS=-v
```

Run an ops volume target:

```sh
make integration-cli-ops-repair-volume IT_GO_TEST_FLAGS=-v IT_VOLUME_TIMEOUT=90m
```

Volume targets use `IT_VOLUME_TIMEOUT`. Baseline family targets continue to use
`IT_TIMEOUT`.

Run with a larger dataset:

```sh
C8VOLT_IT_VOLUME_COUNT=50 make integration-cli-delete-volume IT_GO_TEST_FLAGS=-v IT_VOLUME_TIMEOUT=90m
```

Expected outcome:

- the target creates or discovers suite-owned data for its own run
- evidence records the requested dataset count
- dirty-cluster unrelated data does not fail the target
- stdout/stderr logs and volume summary files are written to the evidence directory

## Critical Flag Validation

Run a target that covers destructive critical flags:

```sh
make integration-cli-update-volume IT_GO_TEST_FLAGS=-v
```

Expected outcome:

- dry-run scenarios prove no suite-owned data was mutated
- confirmed scenarios prove observable post-conditions or explicit no-wait/submitted wording
- limit, workers, fail-fast, no-worker-limit, force, auto-confirm, JSON, keys-only, and report flags are validated where the family supports them

## Pipeline Validation

Run a family that includes stdin workflows:

```sh
make integration-cli-expect-resolve-volume IT_GO_TEST_FLAGS=-v
```

Expected outcome:

- keys-only producer output is validated before use
- stdin consumers handle empty input, duplicates, malformed keys, missing keys, valid keys, and mixed input
- destructive stdin dry-run scenarios do not mutate suite-owned data

## Progress And Reporting Validation

Run an ops target:

```sh
make integration-cli-ops-purge-volume IT_GO_TEST_FLAGS=-v IT_VOLUME_TIMEOUT=90m
```

Expected outcome:

- human-mode scenarios show progress or durable progress facts
- machine-readable stdout remains clean
- ops reports are written and validated
- user-limited discovery is visible in compact output and reports
- final outcome and elapsed time are captured

## Evidence Review

Use a stable workdir when comparing repeated runs:

```sh
export C8VOLT_IT_WORKDIR=/tmp/c8volt-volume-it
make integration-cli-get-volume IT_GO_TEST_FLAGS=-v
```

Review:

- `summary.md`
- `run.json`
- `volume-<family>.json`
- `volume-data-<family>.json`
- `volume-progress-<family>.json`
- `volume-pipelines-<family>.json`
- `volume-ops-reports-<family>.json`
- `proposals-command.json`
- `proposals-embedded-bpmn.json`
- `logs/`
- `data/`

## Normal Validation

Normal repository validation remains:

```sh
make test
go test ./integration/cli -count=1
```

Expected outcome:

- normal unit validation is unaffected
- the integration package remains harmless without the integration build tag
