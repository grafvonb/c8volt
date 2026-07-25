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

Validated ops execute slice:

```sh
make integration-cli-ops-execute-volume IT_GO_TEST_FLAGS=-v
```

Evidence path:

- `/var/folders/jc/60f5tdds44d2v3b4fc0xs5700000gp/T/c8volt-all-command-it-722503529`

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

## Current Implementation Validation

Validation recorded for the initial `integration-cli-get-volume` slice:

```sh
GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run 'TestVolumeTargetCatalog|TestVolumeOwnershipClassification' -count=1 -timeout=5m
make integration-cli-get-volume IT_GO_TEST_FLAGS=-v
```

Observed result:

- non-integration package guard passed
- integration compile-only check passed
- local volume helper tests passed
- `integration-cli-get-volume` passed against `kind-camunda-platform-local-c89`
- evidence path from the passing run: `/var/folders/jc/60f5tdds44d2v3b4fc0xs5700000gp/T/c8volt-all-command-it-4290819661`

Validation recorded for the `integration-cli-walk-volume` slice:

```sh
GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run 'TestVolumeTargetCatalog|TestVolumeOwnershipClassification' -count=1 -timeout=5m
make integration-cli-walk-volume IT_GO_TEST_FLAGS=-v
git diff --check -- Makefile integration/README.md integration/cli specs/256-volume-semantic-integration
```

Observed result:

- non-integration package guard passed
- integration compile-only check passes
- local volume helper tests passed
- `integration-cli-walk-volume` passed against `kind-camunda-platform-local-c89`
- evidence includes `volume-walk.json`, `volume-data-walk.json`, `volume-progress-walk.json`, `volume-pipelines-walk.json`, and `volume-ops-reports-walk.json`
- evidence path from the passing run: `/var/folders/jc/60f5tdds44d2v3b4fc0xs5700000gp/T/c8volt-all-command-it-3597956928`

Validation recorded for the `integration-cli-deploy-embed-run-volume` slice:

```sh
GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run 'TestVolumeTargetCatalog|TestVolumeOwnershipClassification' -count=1 -timeout=5m
make integration-cli-deploy-embed-run-volume IT_GO_TEST_FLAGS=-v
git diff --check -- Makefile integration/README.md integration/cli specs/256-volume-semantic-integration
```

Observed result:

- non-integration package guard passed
- integration compile-only check passed
- local volume helper tests passed
- `integration-cli-deploy-embed-run-volume` passed against `kind-camunda-platform-local-c89`
- evidence includes `volume-deploy-embed-run.json`, `volume-data-deploy-embed-run.json`, `volume-progress-deploy-embed-run.json`, `volume-pipelines-deploy-embed-run.json`, and `volume-ops-reports-deploy-embed-run.json`
- proposal evidence files were written as empty arrays because this slice did not require direct Camunda setup or new embedded BPMN fixtures
- evidence path from the passing run: `/var/folders/jc/60f5tdds44d2v3b4fc0xs5700000gp/T/c8volt-all-command-it-3323686246`

Validation recorded for the `integration-cli-update-volume` slice:

```sh
GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run 'TestVolumeTargetCatalog|TestVolumeOwnershipClassification' -count=1 -timeout=5m
make integration-cli-update-volume IT_GO_TEST_FLAGS=-v
git diff --check -- Makefile integration/README.md integration/cli specs/256-volume-semantic-integration
```

Observed result:

- non-integration package guard passed
- integration compile-only check passed
- local volume helper tests passed
- `integration-cli-update-volume` passed against `kind-camunda-platform-local-c89`
- evidence includes `volume-update.json`, `volume-data-update.json`, `volume-progress-update.json`, `volume-pipelines-update.json`, and `volume-ops-reports-update.json`
- proposal evidence records update-job and richer variable-shape setup gaps
- evidence path from the passing run: `/var/folders/jc/60f5tdds44d2v3b4fc0xs5700000gp/T/c8volt-all-command-it-1158420946`

Validation recorded for the `integration-cli-cancel-volume` slice:

```sh
GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run 'TestVolumeTargetCatalog|TestVolumeOwnershipClassification' -count=1 -timeout=5m
make integration-cli-cancel-volume IT_GO_TEST_FLAGS=-v
git diff --check -- Makefile integration/README.md integration/cli specs/256-volume-semantic-integration
```

Observed result:

- non-integration package guard passed
- integration compile-only check passed
- local volume helper tests passed
- `integration-cli-cancel-volume` passed against `kind-camunda-platform-local-c89`
- evidence includes `volume-cancel.json`, `volume-data-cancel.json`, `volume-progress-cancel.json`, `volume-pipelines-cancel.json`, and `volume-ops-reports-cancel.json`
- evidence path from the passing run: `/var/folders/jc/60f5tdds44d2v3b4fc0xs5700000gp/T/c8volt-all-command-it-4165489772`

Validation recorded for the `integration-cli-delete-volume` slice:

```sh
GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run 'TestVolumeTargetCatalog|TestVolumeOwnershipClassification' -count=1 -timeout=5m
make integration-cli-delete-volume IT_GO_TEST_FLAGS=-v
git diff --check -- Makefile integration/README.md integration/cli specs/256-volume-semantic-integration
```

Observed result:

- non-integration package guard passed
- integration compile-only check passed
- local volume helper tests passed
- `integration-cli-delete-volume` passed against `kind-camunda-platform-local-c89`
- evidence includes `volume-delete.json`, `volume-data-delete.json`, `volume-progress-delete.json`, `volume-pipelines-delete.json`, and `volume-ops-reports-delete.json`
- evidence path from the passing run: `/var/folders/jc/60f5tdds44d2v3b4fc0xs5700000gp/T/c8volt-all-command-it-4106919776`

Validation recorded for the `integration-cli-expect-resolve-volume` slice:

```sh
GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run 'TestVolumeTargetCatalog|TestVolumeOwnershipClassification|TestCommandProposalRegistrationRecordsDirectCamundaSetup' -count=1 -timeout=5m
make integration-cli-expect-resolve-volume IT_GO_TEST_FLAGS=-v
git diff --check -- Makefile integration/README.md integration/cli specs/256-volume-semantic-integration
```

Observed result:

- non-integration package guard passed
- integration compile-only check passed
- local volume helper and proposal tests passed
- `integration-cli-expect-resolve-volume` passed against `kind-camunda-platform-local-c89`
- evidence includes `volume-expect-resolve.json`, `volume-data-expect-resolve.json`, `volume-progress-expect-resolve.json`, `volume-pipelines-expect-resolve.json`, `volume-ops-reports-expect-resolve.json`, and `proposals-command.json`
- proposal evidence records that state-only `expect process-instance` JSON rows should carry stable key/ok identity fields like incident expectation rows
- evidence path from the passing run: `/var/folders/jc/60f5tdds44d2v3b4fc0xs5700000gp/T/c8volt-all-command-it-3582324578`

Validation recorded for the `integration-cli-ops-analyse-volume` slice:

```sh
GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run 'TestVolumeTargetCatalog|TestVolumeOwnershipClassification|TestCommandProposalRegistrationRecordsDirectCamundaSetup' -count=1 -timeout=5m
make integration-cli-ops-analyse-volume IT_GO_TEST_FLAGS=-v
git diff --check -- Makefile integration/README.md integration/cli specs/256-volume-semantic-integration
```

Observed result:

- non-integration package guard passed
- integration compile-only check passed
- local volume helper and proposal tests passed
- `integration-cli-ops-analyse-volume` passed against `kind-camunda-platform-local-c89`
- evidence includes `volume-ops-analyse.json`, `volume-data-ops-analyse.json`, `volume-progress-ops-analyse.json`, `volume-pipelines-ops-analyse.json`, `volume-ops-reports-ops-analyse.json`, `proposals-command.json`, and `proposals-embedded-bpmn.json`
- proposal evidence records deterministic slow-duration and listener fixture gaps
- evidence path from the passing run: `/var/folders/jc/60f5tdds44d2v3b4fc0xs5700000gp/T/c8volt-all-command-it-1000040702`
