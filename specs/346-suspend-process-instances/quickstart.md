# Validation Guide: Suspend Process Instances

**Feature**: [spec.md](spec.md) | **CLI contract**: [contracts/cli.md](contracts/cli.md) | **Service contract**: [contracts/facade-service.md](contracts/facade-service.md)

This guide is for implementation validation using isolated dependencies. The tests named below are required implementation coverage, not tests claimed to exist or pass at planning time. No running Camunda environment is needed or authorized.

## Prerequisites and boundary

- Use the repository Go version/toolchain (`go 1.26.0`, toolchain `go1.26.2`) and existing dependencies.
- Read `AGENTS.md`, `specs/ralph-implementation-rules.md`, this feature's design artifacts, and eventual tasks before implementation.
- Inspect selected test entry points and build constraints before running them. Never run integration-tag tests, `make integration-test*`, `make integration-cli-*`, live fixtures, demo targets, or environment-backed validation helpers.
- Keep `GOFLAGS` empty and `GOENV=off` for the commands below so inherited/persisted flags cannot enable integration tags indirectly. Review Make overrides before using `make test`; do not inherit a different `PKG`, test target, or makefile.
- Test-owned subprocess environments must strip inherited application/profile/credential selectors and provide their own temporary config, explicit fake-server URL, no-auth settings where appropriate, and temporary config-home directories. Do not inspect or select a live profile to make tests work.
- Use generated-client mocks, `testx.NewIPv4Server`/test-owned HTTP servers, `testx.RunCmdSubprocessSeparate`, and existing real-terminal helpers. No fixture may provision services or leave resources outside test-owned temporary storage.

## Planning-only checks

For this planning change, review the changed Markdown, local references, requirement coverage, and whitespace. Do not run Go tests or regenerate CLI docs merely to validate these documents. Record actual document checks separately from future runtime coverage.

## 1. Service submission and collection checks

Implement service tests with a `TestProcessInstanceSuspension` prefix in the owning service/adapter/waiter packages. First confirm the expected tests exist; a no-tests-matched run is not evidence:

```sh
env GOENV=off GOFLAGS= go test ./internal/services/processinstance/... -list 'ProcessInstanceSuspension'
env GOENV=off GOFLAGS= go test ./internal/services/processinstance/... -run 'ProcessInstanceSuspension' -count=1
```

Required assertions:

- v810 sends `{}` to exactly the selected single-instance suspension path and accepts only a valid 204 without a call error. Cover malformed/nil responses, unexpected success status, and a call error with a partial response.
- Single and collection v88/v89 calls reject capability before any request, including dry-run and empty library collections. Test through version factories as well as direct service stubs.
- Dry-run and no-wait produce zero confirmation reads; dry-run also produces zero mutation requests. Empty v810 library collections preserve initialized empty results without acceptance claims.
- Selecting child only, parent only, or both never calls hierarchy/search/relative mutation endpoints. Duplicate flags/input equivalents become one operation per identical key, in first-input order.
- Already-suspended, terminal, terminating, missing, and unauthorized responses remain rejected. A state change or available SUSPENDED response cannot reconcile rejection into success; failed submissions cause zero confirmation reads.
- Retry sequences include 429/RESOURCE_EXHAUSTED then acceptance, ambiguous transport then rejection, ambiguous transport then acceptance, canceled retry backoff, unsent local pacing, earlier uncertainty followed by an unsent attempt, and HTTP-timeout wrapping of pacing failure. Keep retry timing deterministic and bounded with existing test seams or immediate Retry-After responses.
- Collection results retain every key under mixed success/failure, fail-fast, cancellation before scheduling, and cancellation with work in flight. Assert original causes remain discoverable and no accepted evidence becomes a zero-value skipped item.

## 2. Confirmation checks

Use scripted reads and short test-owned deadlines, not real engine time or broad sleep-based checks. Test the [state machine](data-model.md#confirmation-state-machine):

- Accepted → ACTIVE → SUSPENDED confirms with one logical submission and bounded reads.
- Accepted → each true terminal state stops immediately, reports conflict-class failure, and makes no further reads.
- ABSENT, UNKNOWN, timestamp-only, or unrecognized states cannot confirm. A typed not-found continues observation; an authorization/transport error whose message contains `404` or `not found` must not be mistaken for absence.
- A first read failure yields `hasObservation=false`; an UNKNOWN read followed by failure yields `hasObservation=true` and `lastObservedState=UNKNOWN`.
- Timeout, attempt exhaustion, caller interruption, and read authorization failure retain accepted status code/evidence and the previous state/date when present.
- A mismatched instance key cannot confirm the requested target. Existing SUSPENDED mapping may confirm without a date; a date with ACTIVE never confirms.
- The configured deadline constrains in-flight reads/retries, not only time between polls. The waiter has no mutation callback and cannot resubmit.

## 3. Facade and compatibility checks

Implement facade tests under the same behavioral prefix and use stub internal services:

```sh
env GOENV=off GOFLAGS= go test ./c8volt/process -list 'ProcessInstanceSuspension'
env GOENV=off GOFLAGS= go test ./c8volt/process -run 'ProcessInstanceSuspension' -count=1
```

Verify option/worker/context propagation, mechanical field conversion including no-observation and actual UNKNOWN, collection copies/order, partial results returned with converted errors, and unsupported capability on public single/collection/dry-run calls. Include an isolated factory-composed facade test for each supported configured version so a stub cannot hide a missing guard.

Run the closest existing cancellation, forced-deletion, suspended-state/date, and expectation regressions after affected code changes. Inspect actual test names before selecting patterns. Interface additions must compile all affected concrete stubs in process/resource, process-definition deletion, ops, and adapter test packages together. Do not create standalone tasks that temporarily break compilation.

## 4. CLI execution and output checks

Implement command tests with a `TestSuspendProcessInstance` prefix, including output, diagnostics, capability, and terminal subtests:

```sh
env GOENV=off GOFLAGS= go test ./cmd -list 'SuspendProcessInstance'
env GOENV=off GOFLAGS= go test ./cmd -run 'SuspendProcessInstance' -count=1
```

Run the actual command path with fake dependencies and capture stdout/stderr separately. Cover canonical/alias names, repeated/comma-separated flags, stdin plus flags, duplicates, invalid/missing keys, empty/blank stdin, version guards, preview, no-wait, default waiting, mixed outcomes, fail-fast, abort, and the worker controls.

| Mode/scenario | Required evidence |
| --- | --- |
| Human normal/default waiting | Exact compact INFO summary on the configured logger destination, empty stdout and no default per-key chatter; true success only after SUSPENDED. |
| Human dry-run, with/without no-wait | Exact INFO preview wording on the configured logger destination, empty stdout; no mutation or read requests and no prompt. |
| JSON success, preview, accepted, partial failure | Decode exactly one shared envelope, then require EOF; validate per-key fields, payload and error class, and real subprocess exit. |
| JSON+quiet | Same explicitly requested envelope; no informational logger records. |
| Keys-only / keys-only+quiet | Exact selected preview or accepted execution keys in order, one per line; zero bytes for no qualifying keys. Failed confirmation still returns accepted keys and the failure exit. |
| JSON+keys-only | JSON wins, subject to normal JSON validation rules. |
| JSON+verbose or JSON without unattended/preview eligibility | Existing local invalid-input behavior, no mutation. |
| Empty/invalid input across modes | No mutation, no confirmation prompt, no accepted claim; correct error envelope/logging and invalid-input exit. |
| Auto-confirm/automation/no-wait combinations | Existing flag precedence and prompt eligibility, exact request counts, truthful envelope outcome. |

### Diagnostic routing regression checks

Start with `cmd/update_element_logging_test.go` and its configured logger/subprocess helpers. Use repeated multi-key failures containing distinctive URL, method, body, key, token/cookie, and cursor markers. Prove all of the following:

- Exactly one compact aggregate ERROR record at normal, verbose-only, and quiet levels; correct severity and configured/inherited destination. Assert record structure and exact bounded wording, not a substring on stderr.
- Raw HTTP markers are absent from both streams in human/keys modes; normal/verbose output cannot leak report errors. Existing machine-readable error detail is preserved in JSON rather than stripped as a human-output fix.
- JSON errors have one envelope, partial results, expected failure exit, and no duplicate human failure diagnostic.
- Both `--debug` and configured debug level independently expose the existing redacted HTTP diagnostic records with verbose disabled. Test actual existing quiet-over-debug precedence; do not expect quiet to be overridden.
- Plain/time and structured logger formats both carry useful compact messages. Verify secrets remain redacted and diagnostic records never enter stdout, including under JSON output.
- Machine-result writer failure does not trigger a second renderer or replace an existing operation failure exit. Human summaries do not write to the result destination.
- Success, preview, and verbose rows are INFO logger records, with empty human stdout. Verify plain/structured formats and configured/inherited destinations, plus suppression at configured WARN and under quiet. Direct writes to either stream must fail the assertions.

The assertions must fail if the aggregate logger call is replaced with a direct stderr print, if raw joined errors are logged at normal level, or if partial payloads are discarded.

### Real-terminal interaction checks

Reuse the PTY patterns in `cmd/update_element_terminal_test.go` and `testx/cmd_terminal_runner.go`, with actual terminal stdin and separately redirected stdout/stderr. Exercise approval, decline/default/EOF, configured and inherited stderr, redirected stdout, human/keys/quiet-keys, and prompt suppression under dry-run/auto-confirm/automation. Empty stdin is a pipe-input case; missing selection with terminal stdin must also terminate without a prompt. Verify no confirmation path runs before capability/input validation and no mutations follow abort.

## 5. Documentation, fixture syntax, and final integration of code

After command metadata/help changes are implemented:

```sh
env GOENV=off GOFLAGS= make docs-content
bash -n integration/scripts/run-c810-suspended.sh
git diff --check
```

Review generated CLI documentation and README for the permissions, independent parent/child scope, active-only eligibility, retained runtime resources, in-flight workers/external effects, rejected completion/failure, repeat execution after resumption, backpressure, and acceptance/confirmation distinction. Inspect the fixture diff to ensure suspension setup uses c8volt while existing scenarios/cleanup remain unchanged. Do not execute or source it.

Run `gofmt` on touched Go files. After targeted checks pass, run the ordinary full race suite once for the integrated implementation because the new concurrent collection path and widened facade/service interfaces span several packages:

```sh
env GOENV=off GOFLAGS= make test
```

Before this command, confirm the repository `test` target still expands to `go test ./... -race -count=1` without live tags or overridden targets. If only prose changes follow a passing runtime check, do not repeat it. If the environment prevents a check, report it as unavailable, not passed.

## Evidence to retain

Record checks actually run, selected cases, request counts and prohibited paths, envelope/exit results, logger destination/severity/redaction evidence, real-terminal coverage, and any platform skips. Real-engine behavior remains unverified by this workflow; an updated integration fixture is not live validation evidence.
