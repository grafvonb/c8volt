# Quickstart Validation: API Request Diagnostics

## Prerequisites

- Work from the repository root on `codex/305-api-request-diagnostics` with Go 1.26/toolchain go1.26.2 and repository dependencies available.
- API request diagnostics are implemented through the existing `--verbose` flag. The named `APIDiagnostics` tests below are the runnable acceptance suite.
- Default automated validation uses local HTTP/TLS fixtures and temporary configuration, including authentication and cancellation endpoints. It requires no live Camunda credentials and performs no real mutations.
- Linux/macOS terminal checks use the existing `testx.NewCmdTerminalRunner`; unsupported platforms must explicitly report that limitation.

See [the output contract](contracts/api-diagnostics.md) for field/redaction rules and [the data model](data-model.md) for lifecycle semantics.

## Baseline before implementation

On 2026-09-12, branch `codex/305-api-request-diagnostics` passed the required pre-implementation baseline:

```sh
go test ./internal/services/httpc ./internal/services/auth/... ./cmd -run 'ReadRetry|ProcessInstanceConfirmationTerminal|RootHelp' -count=1
```

The shared HTTP retry tests and command root-help/real-terminal confirmation tests passed. Auth packages without matching baseline test names reported `[no tests to run]`; no package failed. Review of the feature artifacts, constitution and `specs/ralph-implementation-rules.md` found no conflicts.

## User Story 1 command/bootstrap validation

On 2026-09-12, the diagnostic test inventories for `internal/services/httpc`, `internal/services/auth/oauth2`, and `cmd` all listed matching `APIDiagnostics` tests. The US1 gate passed:

```sh
go test ./internal/services/httpc ./internal/services/auth/oauth2 ./cmd -run 'TestAPIDiagnostics|RootHelp' -count=1
```

The command cases cover inherited `--verbose` placement, auth-none, OAuth token and cookie-login bootstrap exchanges, unchanged read stdout/request counts, help without exchanges, and verbose-off/debug-only/quiet/restrictive-INFO filtering. After updating pre-feature verbose assertions to distinguish safe API records from command progress text, the full race-enabled repository gate also passed:

```sh
make test
```

## User Story 2 safety and operational validation

On 2026-09-12, the sanitizer, OAuth, cookie and real-terminal diagnostic test inventories all contained their expected acceptance cases. The US2 gate passed without live service access or live mutations:

```sh
go test ./internal/services/httpc ./internal/services/auth/... ./cmd -run 'TestAPIDiagnostics|ProcessInstanceConfirmationTerminal' -count=1
```

Adversarial cases cover AWS/Google signed URLs, multiply encoded names and values, repeated safe and unsafe values, known-secret reflection across query/header/context fields, response cookies reflected in allowed identifiers, arbitrary headers and errors, malformed metadata, control injection and fuzz seeds. Cookie-login and OAuth fixtures verify that credentials and token/cookie reflections are absent while safe correlation identifiers remain. Real-terminal cancellation cases execute with local fixtures only and prove that abort, EOF and empty scopes submit no mutations. The full race-enabled repository gate also passed:

```sh
make test
```

## User Story 3 failure, concurrency and compatibility validation

On 2026-09-12, the US3 race gate passed with lifecycle, retry, redirect, trace/concurrency, writer-failure, subprocess-invocation and supplied-client tests discovered across the requested packages:

```sh
go test ./internal/services/httpc ./internal/services/auth/... ./c8volt ./cmd -run 'APIDiagnostics|ReadRetry|ProcessInstanceConfirmationTerminal|RootHelp' -race -count=1
```

The cases preserve partial request/response counts and original body errors, retain typed or unambiguous trace-backed failure phases, omit conflicting phase evidence, keep frozen records immutable, and give separate process invocations independent stderr destinations and sequences. The repository-wide race suite also passed after the OAuth timeout boundary was made deterministic from the elapsed request deadline:

```sh
make test
```

## 1. Check implementation coverage and run focused validation

Acceptance tests use the shared `TestAPIDiagnostics...` naming prefix in httpc, OAuth, cookie auth, c8volt client wiring and cmd. Confirm the test lists contain the intended cases before running them; a successful command with no matching tests is not validation.

```sh
go test ./internal/services/httpc -list APIDiagnostics
go test ./internal/services/auth/oauth2 -list APIDiagnostics
go test ./internal/services/auth/cookie -list APIDiagnostics
go test ./c8volt -list APIDiagnostics
go test ./cmd -list APIDiagnostics
go test ./internal/services/httpc -run 'TestAPIDiagnostics' -count=1
go test ./internal/services/auth/... -run 'TestAPIDiagnostics' -count=1
go test ./c8volt -run 'TestAPIDiagnostics' -count=1
go test ./cmd -run 'TestAPIDiagnostics' -count=1
```

For preservation comparisons, keep existing verbose and logger settings identical and toggle only the observer through test wiring; separately test user-facing verbose/quiet gating. The test fixtures should create fresh clients/configuration for each enabled/disabled run and provide deterministic response bodies and operation state. Assertions must observe the real transport, not replace the facade with a mock.

Expected proof:

- With verbose enabled and INFO admitted, successful, failed, timed-out, canceled and retried exchanges yield one record each at the observed boundary. Redirects and token requests share the same invocation sequence.
- A controlled server sends informational headers, then final headers, then delayed body bytes. Only final headers end `headers`; `body` ends at EOF/error/early Close. Assert total=headers+body before rounding using controlled synchronization and tolerances, not fragile exact sleep comparisons. Test absent headers/body on pre-header failures, zero body for known bodyless responses, evidence-backed failure phases and numeric status retained on body failure.
- A persistent connection is reused; unperformed phase fields are absent. Inject trace callback sequences to validate overlapping connections and late callbacks independently of live DNS timing.
- Early close and retry-discarded bodies remain incomplete. No extra Read/Close calls occur. Request and response byte counts match delegated reads, including bytes returned with an error.
- GET retries and mutation submissions/polling are identical to baseline. OAuth does not recursively authenticate or acquire a new retry policy; cookie startup is covered.
- Seeded secrets, signed URL parameters, payloads and arbitrary error text never appear, including a response Set-Cookie value reflected in X-Request-ID. Safe keys/tenants/profile identities and query values remain available. Quoted controls never introduce extra records.

## 2. Verify real command execution and streams

The cmd acceptance matrix runs `get process-definition` and `cancel process-instance` through root flag parsing, bootstrap, authentication and real local HTTP handlers. Run:

```sh
go test ./cmd -run 'TestAPIDiagnosticsCommand|TestAPIDiagnosticsTerminal' -count=1
```

For each supported mode, compare an enabled run to a disabled run against reset fixtures:

| Mode / scenario | Expected result |
| --- | --- |
| Normal output | Exact same stdout; diagnostics only on stderr |
| JSON, including quiet + JSON | One unchanged result envelope, decoder then EOF |
| Keys-only, including quiet + keys-only | Exactly one key per stdout line, zero bytes for no keys |
| Quiet | Existing quiet result behavior; INFO diagnostics suppressed, including with verbose |
| Log formats and levels | Plain/plain-time/text/JSON preserve the safe message and configured timestamp/source behavior; verbose-off, debug-only and filtered INFO produce no diagnostic messages |
| Configured child stderr | All diagnostic records reach that writer |
| Inherited root stderr | All diagnostic records reach inherited writer |
| Read sparse/empty discovery | Same page requests and continuation behavior; no synthetic records |
| Cancellation with auto-confirm / automation | Same submitted requests and verified outcomes |
| Cancellation with no-wait | Same accepted-state semantics; diagnostics do not imply completed mutation |
| Real terminal accepts / aborts | Same prompt wording and eligibility, stderr prompts, no result contamination |
| Empty mutation selection | No prompt/mutation, unchanged empty result; only genuine discovery records |
| Writer fails | Existing command/HTTP outcome retained, no stdout fallback |

Capture stdout and stderr separately. Use the real terminal runner for stdin while keeping stdout piped; mocked terminal detection and pipe-only input are insufficient. Exercise both explicit and inherited stderr, plus activity rendering, to catch bootstrap routing regressions.

## 3. Concurrency, compatibility and resource checks

```sh
go test ./internal/services/httpc ./internal/services/auth/... ./cmd -run 'APIDiagnostics|ReadRetry|ProcessInstanceConfirmationTerminal|RootHelp' -race -count=1
go test ./internal/services/httpc -run '^$' -bench APIDiagnostics -benchmem
```

Require no races, duplicate records or interleaved lines during reordered concurrent completions and read/close/trace overlap. Use independent collectors or subprocesses for separate-invocation tests; do not run the shared global Cobra root concurrently in-process. Cover representative v8.7–v8.10/Operate/Tasklist requests to prove all supported client factories retain the supplied transport.

Benchmarks compare disabled/enabled diagnostics using the existing logger configured with a discard writer, small responses and streamed large bodies. Report allocations and runtime; metadata memory must not grow with payload size. Blocking stderr behavior is documented, not hidden behind an unbounded queue.

On 2026-09-12, the benchmark gate passed on Darwin/arm64 (Apple M3 Pro):

```text
BenchmarkAPIDiagnostics/disabled/small-body-12             316.0 ns/op      746 B/op    7 allocs/op
BenchmarkAPIDiagnostics/disabled/streaming-body-12       77980 ns/op        744 B/op    7 allocs/op
BenchmarkAPIDiagnostics/enabled/small-body-12             3018 ns/op       4311 B/op   84 allocs/op
BenchmarkAPIDiagnostics/enabled/streaming-body-12        83502 ns/op       4421 B/op   87 allocs/op
```

The streaming case generated and consumed 8 MiB without retaining a payload buffer. Enabled metadata allocation remained about 4.3 KiB per exchange (a 110-byte difference between the 32-byte and 8 MiB cases), so memory did not scale with payload size and no history, queue or payload buffering appeared. Runtime increased with bytes consumed, as expected; no unsupported latency threshold is claimed.

## 4. Optional manual read-only smoke check

With an existing valid configuration for a development Camunda instance:

```sh
make build
./bin/c8volt --help
./bin/c8volt get pd --help
./bin/c8volt --config ./config.yaml get pd --stat --verbose > /tmp/c8volt-diagnostics-results.txt 2> /tmp/c8volt-diagnostics-stderr.txt
./bin/c8volt --config ./config.yaml get pd --stat --quiet --json --verbose > /tmp/c8volt-diagnostics-results.json 2> /tmp/c8volt-diagnostics-quiet.txt
```

Use the actual local configuration path if different. Inspect the result and stderr files separately: only stderr contains `api #` records; quiet suppresses INFO diagnostics, so the quiet smoke check must contain no diagnostic messages. A live changing dataset is unsuitable for byte-for-byte baseline comparisons; use the deterministic automated fixtures for that guarantee. Mutation proof comes from the local cancellation tests, not a live purge or resource creation workflow.

## 5. Documentation and final delivery gates

After formatting touched Go files, update README and root command help/examples, then run:

```sh
make docs-content
make test
git diff --check
```

Inspect generated CLI documentation for existing verbose behavior and README guidance for timing, incomplete transfers, retained identifiers, redaction, quiet/stderr behavior and the observation boundary. Do not hand-edit generated CLI pages. Record test outcomes and any platform limitations before committing; full race-enabled `make test` is mandatory for implementation delivery.

## Final delivery validation

On 2026-09-12, iteration 17 formatted every Go file changed from the `origin/develop` merge base; `gofmt` produced no diff. All automated quickstart test-list and focused acceptance commands passed, including the real command/terminal suite and the race-enabled diagnostics, retry, authentication and root-help gate. The benchmark gate also passed on Darwin/arm64 (Apple M3 Pro): enabled allocation was 4311 B/op for the 32-byte body and 4424 B/op while generating and consuming the 8 MiB stream, confirming payload-size-independent metadata.

`make docs-content` completed and refreshed only the generated homepage build metadata. The final `make test` race suite and `git diff --check` passed. No platform limitation applied to the real-terminal tests.

The FR-001–FR-011 diff review found diagnostics confined to root bootstrap, the shared `internal/services/httpc` transport boundary and OAuth collector sharing. There are no new flags, configuration keys or dependencies; no production facade, individual command or generated-client diagnostic edits; and no extra network requests, body consumption or mutation behavior. Acceptance coverage verifies request counts and bodies, stdout and exit preservation, prompt and mutation behavior, quiet/logger routing, timing and partial evidence, retries, redaction, concurrency and invocation isolation. README, root metadata and regenerated documentation cover the required operator guidance.

Recovery validation on 2026-09-13: `make test` passed with exit code 0 outside the sandbox after the initial sandboxed run was blocked from binding local fixture sockets. `git diff --check` passed. No production code changes were needed for recovery.
