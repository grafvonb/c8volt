# Validation Guide: Request Rate Control

This guide is for the implementation phase. The feature and proposed tests are not implemented by this planning change. Only isolated fakes are permitted; no Camunda installation, deployment, live resource mutation, or cleanup is part of completion.

## Prerequisites and setup

Use Go 1.26/toolchain 1.26.2 on branch `353-request-rate-reporting`. Work from the repository root. Test fixtures must create their own HTTP servers/configuration and neutralize inherited `C8VOLT_*` settings, profile selection, credentials, and user config discovery. Subprocess helpers inherit the environment, so explicitly sanitize it in the fixtures.

Before running checks, inspect their dependencies/build constraints. Do not enable integration build tags or invoke integration/smoke targets. Clear inherited Go test flags for the commands below. Test servers must remain test-owned; a localhost Camunda installation is not a fake.

## Focused checks after implementation

Use `RequestRate` in the new acceptance test names so these commands select them. First list tests and confirm matching test functions exist in **each** affected package; an empty selection is not passing evidence.

```sh
GOFLAGS='' go test ./config ./internal/services/httpc ./internal/services ./cmd -list RequestRate
GOFLAGS='' go test ./config -run RequestRate -count=1
GOFLAGS='' go test ./internal/services/httpc -run RequestRate -count=1
GOFLAGS='' go test ./internal/services -run RequestRate -count=1
GOFLAGS='' go test ./cmd -run RequestRate -count=1
```

Run the relevant existing config, HTTP diagnostics/read-retry, mutation-retry, and affected command regressions as well. Reuse existing fixtures rather than build a parallel test framework. If a narrow facade/error mapping changes, include its closest existing classification tests.

| Scenario | Required evidence | Spec coverage |
|----------|-------------------|---------------|
| Settings and precedence | Defaults, positive/fractional rates, CLI/profile/env explicit zero, custom/zero interval; negative/non-finite/malformed/empty inputs fail before any remote request. | FR-001–002, FR-007; SC-006 |
| Shared concurrent pacing | Record delegation timestamps across workers and mixed phases/retries. Verify burst-one shared allowance, including after idle; keep responses blocked to prove pacing does not serialize in-flight calls. State the timing tolerance and check actual starts, not just elapsed completion time. | FR-003, FR-006; SC-001, SC-003 |
| Zero-rate fast path | With DEBUG off/on and different intervals, assert no limiter wait calls and no added requests. Reporter/logger blocking must not block request dispatch. | FR-004, FR-011; SC-002 |
| Local failure before dispatch | Cover already-cancelled contexts, cancellation during waiting, early insufficient deadline, and actual `http.Client.Timeout` expiry with a still-live parent context. Force the latter with a controlled waiter/timing seam if needed. Assert zero affected dispatches/count increments, local unsent reason, preserved classification and no retry callback repetition. | FR-005; SC-006 |
| Existing retry behavior | Fake backend timeout, 429 and retry-eligible 503 responses still follow existing policy/budget. Every dispatched attempt shares pacing and counts once. Local-wait state never leaks to another resource/attempt. | FR-003, FR-005–006 |
| Windows and lifetime | Controlled snapshots/ticks prove default/custom cadence, accurate elapsed rates, zero windows and concurrent boundary counts. Interval zero/non-DEBUG create no reporter. Repeated invocation, bootstrap failure, returned execution error, cancellation and double close stop reporting without waiting for the next interval. | FR-007–011; SC-003–004 |
| Logger routing | Execute with `--debug`, configured DEBUG, verbose alone, quiet combinations, plain/structured formats and configured/inherited stderr. Assert severity, destination, all three values, no duplicate summaries and no stdout diagnostics. Keep existing redaction tests. | FR-008–010, FR-013 |
| Mutation compatibility | Representative cancel, instance delete, definition delete with cleanup, and purge retain targets/order, worker settings, confirmation and results. Exercise dry-run, empty scope, no-wait, failure and interruption. | FR-012–013, FR-015; SC-005 |

For mutation/output checks, capture stdout and stderr separately through command execution. Assert exactly one JSON envelope followed by EOF, available partial results and original exit code, exact empty human wording, zero-byte empty keys output, and no empty-scope prompt or mutation. Include quiet with machine output and supported automation/auto-confirm/no-wait combinations. Use a real terminal stdin fixture where confirmation eligibility matters; retain nonempty and sparse-page regressions.

For failures, include multiple resources with repeated fake backend errors containing recognizable URL/method/body/key markers. Normal output and verbose alone must retain a bounded aggregate diagnostic and exclude raw HTTP details from both streams; quiet retains the compact error. DEBUG remains redacted. Assertions must fail if logger routing is replaced by direct stderr printing or raw aggregate errors are exposed. Existing known behavior outside this feature is not a license for unrelated repair.

## Overhead, broader checks, and docs

Add focused comparative benchmarks under `httpc` for pass-through, rate-zero/reporting-off, and rate-zero/reporting-on using fake transport and a controlled logger. Inspect allocations and relative cost without asserting zero runtime overhead or an arbitrary requests/second target. Confirm benchmarks exist before using:

```sh
GOFLAGS='' go test ./internal/services/httpc -run '^$' -bench RequestRate -benchmem
GOFLAGS='' make test
GOFLAGS='' make docs-content
git diff --check
```

The full race suite is justified by shared HTTP behavior, concurrent counting/reporting, lifecycle changes and the new dependency. Review the active Makefile before running; do not run `make integration-test*`, `make integration-cli-*`, or live-test tags. Run formatting on touched Go files and inspect regenerated documentation, root help and configuration template defaults. Do not hand-edit `docs/cli/*`.

Completion evidence should name the checks actually run, record measured benchmark costs, and link behavior to [the contract](contracts/request-rate.md). Fake tests establish the implementation contract; they do not establish a universally safe production rate or prove GOAWAY elimination.

## Planning-only validation

For this documentation-only phase, review links, requirement coverage, template completion, and whitespace. Do not run Go tests, benchmarks, or documentation generation merely to validate this plan.
