# Quickstart and Validation: Bulk Job Completion

This is a validation guide for the implementation to be built from [plan.md](plan.md). The commands below are not evidence that the feature already works. No Camunda mutation or runtime test was performed during planning.

## Prerequisites

- Checkout `345-bulk-job-completion`, based on `v4.3.4`, and complete implementation.
- Use the repository Go toolchain and normal build dependencies.
- For live validation, use a disposable, authorized Camunda 8.8/8.9/8.10 environment and explicit tenant configuration with permission to read and complete its fixture jobs.
- Create a feature-specific BPMN fixture during implementation: a parent calling a child with a native user task and a CANCELING task listener of a unique type, with no worker subscribed to that type. Include multiple instances/children and a nonmatching control job. Do not rely on unreleased #341 fixtures or existing production instances.
- Record the listener's external work as intentionally unnecessary for the disposable fixture. For real recovery, the operator must separately decide how that work is handled.

All shell examples run from the repository root. See [CLI contract](contracts/cli.md) for exact selection and output semantics.

## Targeted automated validation

Test names below are required test-family names for implementation, not tests that already exist. Confirm discovery first so an empty `-run` match is not mistaken for passing coverage.

```sh
go test ./internal/services/job/... -list 'Test.*(BulkCompletion|JobSearch)'
go test ./c8volt/job -list 'Test.*BulkCompletion'
go test ./cmd -list 'Test.*(UpdateJob|JobSearch|BulkCompletion)'

go test ./internal/services/job/... -run 'Test.*(BulkCompletion|JobSearch)' -count=1
go test ./c8volt/job -run 'Test.*BulkCompletion' -count=1
go test ./cmd -run 'Test.*(UpdateJob|JobSearch|BulkCompletion)' -count=1
```

The test matrix must cover:

1. At least 1,000 controlled jobs, multiple pages, duplicate observations, sparse pages followed by matches, positive unique limit and complete empty selection. Record zero completion requests before final preparation.
2. Actual v88/v89/v810 request JSON includes named tenant, omits it for empty/all-tenant discovery, and omits it for explicit-key lookup. Include wrong-tenant rows defensively and total-search regressions.
3. Malformed/contradictory continuation, a discovery error after partial data and interrupted preparation cause zero writes. Preserve overfull-response trimming and avoid offset drift after local filtering.
4. Repeated/comma/stdin input, duplicates, empty stdin, invalid keys, predicate-only retries zero, explicit/filter rejection, unsupported version, and unrelated update-mode compatibility.
5. Races: already completed; changed filters; canceled/error-thrown; missing; read failure; 404/409 then completed; timeout then completed; timeout then only missing. Count at most one completion submission per frozen key.
6. Bounded workers, default continue, fail-fast, caller cancellation, deterministic result order and report count conservation. Verify unknown/terminal states are not mutated and retries/incidents are never repaired.
7. Dry-run, empty and nonempty execution, auto-confirm, automation, no-wait, quiet and JSON combinations. Parse exactly one envelope and require EOF; capture stderr separately. Verify partial results survive a nonzero exit.
8. Genuine terminal stdin with stdout/stderr separately captured; use `testx/cmd_terminal_runner.go`. Check configured and inherited stderr, redirected stdout, one eligible confirmation, decline behavior, and prompt-free empty completion.
9. Task-listener variable restrictions, frozen selection when completing one listener creates another, single-key legacy payload/behavior and shared capability metadata.

Use matching existing test names in addition to the new families when nearby tests have different names. Include the unsupported v87 adapter and all interface test doubles in compilation checks.

After targeted validation, run broader checks once because shared APIs and concurrent execution change:

```sh
make test
make vet
make docs-content
git diff --check
```

Run gofmt on touched Go files before testing. Review generated docs and README against the contract. Do not repeat runtime suites for subsequent prose-only changes.

## Build and safe fixture preview

```sh
go build -o /tmp/c8volt-345 .
/tmp/c8volt-345 update job --help
```

Deploy/start the disposable fixture using the repository's existing deployment/start workflow and record its root/child/job keys. Initiate cancellation once to create the pending listeners. Configure the following variables from those recorded fixture facts:

```sh
export C8VOLT_345_TENANT='replace-with-disposable-tenant'
export C8VOLT_345_LISTENER_TYPE='replace-with-unique-fixture-listener-type'
```

Preview only that scope:

```sh
/tmp/c8volt-345 update job --tenant "$C8VOLT_345_TENANT" \
  --kind TASK_LISTENER --listener-event-type CANCELING --state CREATED \
  --type "$C8VOLT_345_LISTENER_TYPE" --complete --dry-run --verbose
```

Expected: intended candidates and their tenant/process context; listener consequence; no mutations. Compare independent job/process reads before and after. Run the same preview with JSON and verify one envelope.

## Execute and independently observe

After reviewing the fixture selection:

```sh
/tmp/c8volt-345 update job --tenant "$C8VOLT_345_TENANT" \
  --kind TASK_LISTENER --listener-event-type CANCELING --state CREATED \
  --type "$C8VOLT_345_LISTENER_TYPE" --batch-size 2 \
  --complete --auto-confirm --json > /tmp/c8volt-345-result.json
```

Expected: one ordered outcome per selected fixture job. Accepted submissions have independent confirmation skipped; observed completed no-ops are distinguished. The control job remains untouched. Inspect the saved envelope and exit code separately; a partial failure still contains all frozen outcomes.

Using recorded root keys, independently inspect the hierarchy:

```sh
/tmp/c8volt-345 walk pi --key "$C8VOLT_345_ROOT_KEY" --with-elements --with-listeners
```

Set `C8VOLT_345_ROOT_KEY` to a recorded fixture root before running that read. Observe job COMPLETED and process terminal states through normal reads after exporter delay; record those observations independently of batch acceptance. This proves SC-006 without changing the command into a process waiter. A 404 alone is not proof of termination.

Repeat selection after recovery: if no new sequential listeners exist, it must succeed empty without confirmation or mutation, including with no-wait/JSON/quiet. If the fixture deliberately creates a later listener, it must be absent from the first report and selectable only on a later invocation.

## Explicit reviewed selection

Save fixture keys one per line in `/tmp/c8volt-345-reviewed-keys.txt`. Do not use production example keys from the incident report.

```sh
/tmp/c8volt-345 update job - --complete --dry-run --json \
  < /tmp/c8volt-345-reviewed-keys.txt
```

Run execution on a fresh disposable fixture to test actual completion; otherwise already-completed outcomes are expected. Repeat one key in the file and flags to prove deduplication. A malformed key or an unreadable explicit job must block the entire batch before writes.

A later selector command performs fresh discovery; dry-run does not persist a plan. An explicit saved key set preserves reviewed membership, with mandatory current-state rechecks.

## Evidence and cleanup

Record exact Camunda versions, tenant, fixture identifiers, command exit statuses, envelopes, per-key request counts in automated tests, and separate terminal hierarchy observations. Cover all three supported adapters automatically; record which engine versions were exercised live rather than claiming unrun live checks passed.

Clean up only recorded disposable fixture resources using established commands after observing their states. Do not delete history as part of bulk completion. If no live environment is available, retain SC-006 as an explicit validation gap before release.
