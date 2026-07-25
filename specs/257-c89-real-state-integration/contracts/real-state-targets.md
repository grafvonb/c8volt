# Contract: Real-State Targets

## Target Naming

Real-state targets must be separate from baseline and volume targets.

Initial target set:

```text
integration-cli-real-state-gaps
integration-cli-real-state-jobs
integration-cli-real-state-incidents
integration-cli-real-state-listeners
integration-cli-real-state-bpmn-error
integration-cli-real-state-retention
integration-cli-real-state-destructive
```

The target set may be split further by command family if a slice becomes too large, but the default target names above remain the stable operator entry points for this feature.

## Execution Contract

Every target must:

- run with the Go integration build tag
- use the existing `integration/cli` package
- build or reuse the c8volt binary through the existing harness
- use the default local c8volt configuration
- accept existing profile selection from the harness
- focus on Camunda 8.9 profiles for this feature
- write evidence outside `docs/`
- tolerate clean and dirty disposable clusters

Target commands should follow this shape:

```sh
make integration-cli-real-state-jobs IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-incidents IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m
```

## Independence Contract

Each target must create or discover the data it needs during its own run.

Targets must not require:

- another real-state target to run first
- a baseline or volume target to run first
- an empty cluster
- cleanup from previous runs
- exact global counts
- a custom config file

## Destructive Contract

Real-state targets may destructively mutate the selected disposable cluster.

Required behavior:

- dry-run paths must prove no mutation for scoped suite-owned candidates
- confirmed paths must verify observable post-state when the command claims completion
- no-wait paths must record that confirmation was intentionally skipped
- cleanup-failed and retained states must be reported as evidence, not hidden

## Version Contract

This feature targets Camunda 8.9 first.

Required behavior:

- selected profile and observed version must be recorded in evidence
- unsupported non-8.9 behavior must be classified before mutation
- gap artifacts must keep affected versions explicit so future minor releases can extend the suite
