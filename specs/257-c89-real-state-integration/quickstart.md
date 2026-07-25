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

## Proposal Gate

The initial scaffolding adds reserved Make targets and non-live helper checks.
Before a real-state family target is implemented, reserved Make targets fail
with a clear not-implemented message instead of producing a false pass.

Validate the initial scaffolding:

```sh
GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1
GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run 'TestRealStateTargetCatalog|TestRealStateC89ProfileClassification|TestRealStateEvidenceWritersEmitArrays|TestRealStateMachineOutputAssertions|TestRealStateProposalFallbackHelpers' -count=1 -timeout=5m
```

After implementation, run proposal-only validation first:

```sh
make integration-cli-real-state-proposals IT_GO_TEST_FLAGS=-v
```

Expected outcome:

- aggregate command proposals include all known command setup gaps
- aggregate embedded BPMN proposals include all known fixture gaps
- ops repair gaps are present in aggregate evidence
- proposal evidence is written outside `docs/`

## Real Job State

Run the jobs target:

```sh
make integration-cli-real-state-jobs IT_GO_TEST_FLAGS=-v
```

Expected outcome:

- the target creates or discovers suite-owned active jobs
- `get job` returns non-empty job rows scoped to suite-owned data
- at least one supported job mutation records before-state and after-state evidence
- no-wait or accepted paths state that confirmation was intentionally skipped

## Incidents With Related Jobs

Run the incidents target:

```sh
make integration-cli-real-state-incidents IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m
```

Expected outcome:

- the target creates or discovers active incidents
- related job evidence is present when repair or retry behavior depends on it
- repair and retry paths record candidate counts, attempted work, and final outcome
- missing setup capability is recorded as proposal evidence

## Listener And BPMN Error Paths

Run listener and BPMN error targets:

```sh
make integration-cli-real-state-listeners IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-bpmn-error IT_GO_TEST_FLAGS=-v
```

Expected outcome:

- existing embedded models are used when they can create the required state
- missing listener or BPMN error process behavior is recorded as embedded BPMN proposal evidence
- command setup gaps are recorded as command proposal evidence
- no test passes solely because the command accepted the flag

## Retention And Destructive Semantics

Run retention and destructive targets:

```sh
make integration-cli-real-state-retention IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m
make integration-cli-real-state-destructive IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m
```

Expected outcome:

- dry-run scenarios prove no mutation of scoped suite-owned candidates
- confirmed destructive scenarios verify deletion, cancellation, resolve, repair, purge, retained, cleanup-failed, or no-wait state
- mixed valid, missing, malformed, stale, and already-mutated targets produce clear partial-failure or fail-fast evidence
- ops report evidence agrees with stdout outcomes

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
