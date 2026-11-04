# Quickstart and Validation: Bulk Job Completion

Implementation checks use only isolated mocks, test-owned HTTP fake servers and local subprocess/terminal tests. Follow `AGENTS.md`; do not contact an installed Camunda environment, discover Kubernetes targets, enable integration build tags, deploy BPMN or clean up real resources.

The user corrected this scope on 2026-10-20. The previous live deployment/recovery instructions have been removed. Historical observations remain in [validation.md](validation.md) and `progress.md`; they are not commands to repeat or prerequisites for completion.

## Prerequisites

Use the repository Go toolchain on `345-bulk-job-completion`. No Camunda profile, cluster, tenant credentials or live target is required. Test helpers generate temporary configuration pointing at their own fake servers. Clear inherited `GOFLAGS` so integration tags cannot be enabled indirectly; inspect any overridden test target before running it. Do not invoke `make integration-test*` or `make integration-cli-*`.

These are validation commands, not a requirement to repeat passing suites after documentation-only changes. Review [validation.md](validation.md) for existing evidence and run only checks justified by changes.

## Targeted automated validation

Confirm test discovery first so an empty `-run` match is not mistaken for passing coverage. These families now exist; previous iteration evidence is in `progress.md`.

```sh
env GOFLAGS= go test ./internal/services/job/... -list 'Test.*(BulkCompletion|JobSearch)'
env GOFLAGS= go test ./c8volt/job -list 'Test.*BulkCompletion'
env GOFLAGS= go test ./cmd -list 'Test.*(UpdateJob|JobSearch|BulkCompletion)'

env GOFLAGS= go test ./internal/services/job/... -run 'Test.*(BulkCompletion|JobSearch)' -count=1
env GOFLAGS= go test ./c8volt/job -run 'Test.*BulkCompletion' -count=1
env GOFLAGS= go test ./cmd -run 'Test.*(UpdateJob|JobSearch|BulkCompletion)' -count=1
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

Use matching existing test names in addition to the new families when nearby tests have different names. Compile the v88/v89/v810 adapters and all interface test doubles. Keep command-level 8.7 rejection checks; there is no v87 adapter to compile or restore.

After targeted validation, run broader checks once because shared APIs and concurrent execution change:

```sh
env GOFLAGS= make test
env GOFLAGS= make vet
env GOFLAGS= make docs-content
git diff --check
```

Run gofmt on touched Go files before testing. Review generated docs and README against the contract. Do not repeat runtime suites for subsequent prose-only changes.

The default race runtime sleeps before each helper subprocess exits. If the full
command package reaches Go's ten-minute timeout from that cumulative delay, use
`env GOFLAGS= GORACE=atexit_sleep_ms=0 make test`; this still runs the full suite with `-race`.
Record both the timeout and the adjusted run, rather than labeling the initial
run successful.

## Representative listener scenario (SC-006)

```sh
env GOFLAGS= GORACE=atexit_sleep_ms=0 go test ./cmd -run '^TestBulkCompletionAcceptanceListenerHierarchy$' -race -count=1 -v
```

The fake server supplies three matching `TASK_LISTENER` / `CANCELING` jobs from distinct process-instance keys and a control with a different type. The CLI must preserve candidate context, preview without submissions, complete each matching job once and leave the control untouched. All requests go to a test-owned loopback server; unexpected process/repair/history endpoints fail the test. Accepted responses must show independent confirmation skipped rather than inferred process termination.

The test runs through all three adapters. It does not execute a BPMN engine or prove that a real process hierarchy terminates. That limitation is explicitly outside implementation acceptance.

For bulk failure output, `TestBulkCompletionDiagnosticLogging` exercises repeated HTTP rejections through isolated command subprocesses on all three adapters. It verifies configured logger formats and destinations, compact normal/verbose/quiet errors, partial JSON evidence and exit codes, and HTTP diagnostics enabled independently by flag or configured debug level. `TestBulkCompletionFailureSummaryBounds` covers bounded mixed reasons and untrusted diagnostic text; the terminal stream suite covers logged aborts with unchanged plain prompts.

## Static reference fixtures

[fixtures/README.md](fixtures/README.md) describes the retained BPMN examples. They are static references from the superseded live-validation plan, not deployment inputs for implementation. They may be parsed as local XML; no deployment, worker or environment cleanup is needed.

## Completion evidence

Record the targeted test command/results, prior passing full race-suite/vet/docs evidence, and local artifact checks in [validation.md](validation.md). Keep the recorded original timeout as well as the successful adjusted race run. Distinguish simulated API responses from real-engine observations. Complete the revised T009 after these isolated checks pass; an unavailable live environment is not a blocker.
