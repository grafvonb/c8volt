# Contract: Volume Targets

## Target Naming

Volume targets must be separate from baseline family targets and use a `-volume` suffix.

Required targets:

```text
integration-cli-get-volume
integration-cli-walk-volume
integration-cli-update-volume
integration-cli-cancel-volume
integration-cli-delete-volume
integration-cli-expect-resolve-volume
integration-cli-deploy-embed-run-volume
integration-cli-ops-analyse-volume
integration-cli-ops-execute-volume
integration-cli-ops-purge-volume
integration-cli-ops-repair-volume
```

If implementation proceeds incrementally, a smaller aggregate ops target may be introduced first, but final coverage must remain family-addressable.

## Execution Contract

Every target must:

- run with the Go integration build tag
- use the existing `integration/cli` package
- build or reuse the c8volt binary through the existing harness
- use default local c8volt configuration only
- accept the existing profile selection mechanism
- write evidence outside `docs/`
- be safe to run in any order against clean or dirty disposable clusters

Target commands should follow this shape:

```sh
make integration-cli-get-volume
make integration-cli-get-volume C8VOLT_IT_GO_TEST_FLAGS=-v
make integration-cli-get-volume C8VOLT_IT_VOLUME_TIMEOUT=90m
```

## Dataset Size Contract

Volume targets must use a conservative default dataset count and allow an environment override through `C8VOLT_IT_VOLUME_COUNT`.

Rules:

- Default size must be large enough to prove multi-record behavior.
- Paging scenarios must create more matching suite-owned records than one requested page.
- Limit scenarios must create more matching suite-owned records than the requested limit.
- Dataset count must be recorded in evidence.
- Increasing dataset count must not require code changes.

## Independence Contract

Every target must seed or discover the data it needs during its own run.

Targets must not require:

- another volume target to run first
- a baseline family target to run first
- an empty cluster
- successful cleanup from previous runs
- stable global process-definition versions or global counts

## Validation Contract

Each volume target must produce pass/fail evidence for:

- command family under test
- selected profiles and observed versions
- dataset count and created identifiers
- critical flags exercised
- output modes exercised
- dirty-cluster data ownership classification
- setup gaps and proposal records

Failures must identify whether the issue is product behavior, harness setup, missing command support, missing fixture support, or environment availability.
