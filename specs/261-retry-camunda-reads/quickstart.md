# Quickstart: Retry Transient Camunda Read Failures

## Prerequisites

- Go toolchain available for the repository
- c8volt dependencies already downloaded
- Current branch: `261-retry-camunda-reads`

## Focused Validation

Run HTTP retry package tests:

```bash
go test ./internal/services/httpc -count=1
```

Expected outcome:

- GET retry after transient 500 succeeds when a later attempt succeeds.
- HEAD follows the same retry rules.
- 404 is not retried.
- POST search is not retried.
- Mutation methods are not retried by the generic read layer.
- Final response body remains readable after exhausted retries.
- Context cancellation stops retry sleep promptly.

## Existing Retry Regression

Run existing mutation retry tests to confirm behavior was not replaced or weakened:

```bash
go test ./internal/services -run 'RetryCamundaMutation' -count=1
```

Expected outcome:

- Existing Camunda mutation retry tests remain green.
- Resource-exhausted mutation retry behavior remains owned by the existing mutation helper.

## Customer Scenario Regression

Add or run a regression that simulates the observed process-instance parent lookup path:

```bash
go test ./cmd ./internal/services/processinstance/... -run 'Orphan|Retry' -count=1
```

Expected outcome:

- A transient GET failure during parent process-instance lookup is retried.
- The orphan-child total/discovery command continues when the retry succeeds.
- A true not-found parent is still treated as an orphan signal.

## Full Validation

Before completion, run:

```bash
make test
```

Expected outcome:

- Full repository tests pass with race detection.
- No generated CLI documentation changes are required unless command metadata changes.
- If normal retry logging is user-visible, README or troubleshooting documentation describes the compact retry behavior.
