# Quickstart: C89 Real-State Semantic Integration Coverage

This guide describes how the real-state suite should be validated after implementation.

## Prerequisites

- A disposable Camunda 8.9 cluster is configured in the default local c8volt configuration.
- The selected profile is safe for destructive integration testing.
- The 255 baseline all-command suite and 256 volume suite are available.
- The operator understands that real-state targets may create, mutate, delete, cancel, resolve, repair, purge, retain, or leave cleanup-failed data in the selected cluster.

Do not run real-state targets against shared, production, customer, or protected clusters.

## Baseline Gate

Run a quick baseline target before starting real-state validation:

```sh
make integration-cli-get IT_GO_TEST_FLAGS=-v
```

Expected outcome:

- the integration harness builds or reuses the c8volt binary
- default local config is used
- verbose output shows scenario names, arguments, exit codes, durations, and evidence paths
- unrelated dirty cluster data does not fail the baseline target

## Gap Boundary Gate

The gap target is non-destructive and validates the spec-owned boundary for
missing setup capabilities and embedded BPMN assets. It keeps `gaps.md` and the
coverage matrix structurally complete without generating runtime backlog
proposal files.

Validate the initial scaffolding:

```sh
GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run 'TestRealStateTargetCatalog|TestRealStateC89ProfileClassification|TestRealStateEvidenceWritersEmitArrays|TestRealStateMachineOutputAssertions|TestRealStateGapFamily|TestRealStateGapArtifactDocumentsCurrentPrerequisites' -count=1 -timeout=5m
```

Run gap validation before destructive real-state slices:

```sh
make integration-cli-real-state-gaps IT_GO_TEST_FLAGS=-v
```

Expected outcome:

- `gaps.md` includes all known command setup and embedded BPMN fixture gaps that block deeper live coverage
- `follow-ups.md` consolidates the remaining 255/256/257 command, embedded BPMN, ops, output, and pipeline follow-ups into issue-sized groups
- every open gap includes blocked proof, affected commands, affected versions, and runtime behavior until closed
- the coverage matrix uses explicit evidence statuses for every priority topic
- runtime tests do not generate backlog proposal JSON files
- gap artifacts stay outside `docs/`

## Real Job State

Run the jobs target:

```sh
make integration-cli-real-state-jobs IT_GO_TEST_FLAGS=-v
```

Expected outcome:

- the target creates or discovers suite-owned Camunda 8.9 service-task jobs
- `get job` returns non-empty job rows scoped to suite-owned data
- `update job --retries` is confirmed against real state
- `update job --fail` and `--no-wait` return accepted/submitted evidence
- `update job --timeout` produces clean dry-run plan evidence for created jobs
- `gaps.md` records the remaining activated-job setup gap for confirmed timeout mutation

## Incidents With Related Jobs

Run the incidents target:

```sh
make integration-cli-real-state-incidents IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m
```

Expected outcome:

- the target creates a suite-owned job-backed active incident by failing a real job
- `get incident` returns a related `jobKey` scoped to the suite-owned process instance
- `get job --key` verifies the related failed job state
- `ops repair incident --dry-run` reports candidate and related-job counts

## Listener And BPMN Error Paths

Run listener and BPMN error targets:

```sh
make integration-cli-real-state-listeners IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-bpmn-error IT_GO_TEST_FLAGS=-v
```

Expected outcome:

- the listener target uses `C89_SimpleServiceTask.bpmn` to create real execution-listener jobs
- `get element --with-listeners`, `walk process-instance --with-listeners`, and `ops analyse slow-process-instances --with-listeners` include the suite-owned listener job key
- JSON listener scenarios keep stdout machine-safe; traversal JSON is run without `--automation` because traversal commands do not support automation mode
- the BPMN error target records clean `update job --throw-bpmn-error --dry-run` evidence and verifies the job is unchanged afterward
- missing confirmed BPMN error behavior is skipped or reported as prerequisite-missing at runtime and tracked in `gaps.md`
- no test passes solely because the command accepted the flag

## Retention And Destructive Semantics

Run retention and destructive targets:

```sh
make integration-cli-real-state-retention IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m
make integration-cli-real-state-destructive IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m
```

Expected outcome:

- the retention target creates fresh completed suite-owned process instances, finds non-empty `ops execute retention-policy --retention-days 0` candidate sets, proves dry-run leaves candidates completed, and proves confirmed deletion makes report-frozen keys absent
- the destructive target creates active suite-owned process instances and incident-bearing process instances, then proves cancel/delete dry-run safety and confirmed post-state
- incident-selected purge proves exact `--inc-key` frozen selection, dry-run safety, confirmed report parity, and absent post-state for the suite-owned target
- resolve scenarios prove dry-run and confirmed submission on real self-recreating incidents; durable incident clearing is asserted through `ops repair` because the current embedded model recreates a resolution-only incident
- mixed-target scenarios prove malformed and missing process-instance inputs fail before mutation, stale deleted targets report not found, already-terminated targets stay dry-run safe, and `resolve incident` keeps JSON machine output clean for partial and fail-fast no-wait failures
- remaining process-definition purge, orphan purge, durable standalone resolve, and ops repair-specific mixed failure behavior stays tracked in `gaps.md` and `coverage-matrix.md`
- ops report evidence agrees with stdout outcomes for the implemented retention, repair, and incident-selected purge slices

## Normal Validation

Normal repository validation remains:

```sh
go test ./integration/cli -count=1
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m
make test
```

Expected outcome:

- the non-integration package remains harmless
- integration tests compile with the build tag
- normal unit and race validation are unaffected by real-state targets

## Final Feature Validation

Before closing the feature, run every real-state target independently so order
dependencies and dirty-cluster assumptions stay visible:

```sh
make integration-test-real-state IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m
```

The aggregate target expands to the independent slices below:

```sh
make integration-cli-real-state-gaps IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-jobs IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-incidents IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-listeners IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-bpmn-error IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-retention IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m
make integration-cli-real-state-destructive IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m
```

To run the baseline, volume, and real-state suites in sequence:

```sh
make integration-test-all IT_GO_TEST_FLAGS=-v IT_VOLUME_TIMEOUT=90m IT_REAL_STATE_TIMEOUT=90m
```

Generated CLI docs are required only when command help metadata, aliases, flags,
or examples change. The current 257 real-state suite changes integration tests,
runtime JSON failure handling, and spec-owned validation artifacts; it does not
change command help/example metadata, so `make docs-content` is not required.

Use `specs/257-c89-real-state-integration/follow-ups.md` as the source for
future issue/spec creation. The 255 and 256 artifacts remain historical context,
but remaining setup, embedded BPMN, ops-report, output, and pipeline gaps should
be carried forward from the 257 roadmap.

Finish with:

```sh
git diff --check -- Makefile integration/README.md integration/cli specs/257-c89-real-state-integration specs/integration-test-responsibility.md
```
