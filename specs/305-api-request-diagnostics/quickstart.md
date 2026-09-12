# Quickstart Validation: API Request Diagnostics

## Prerequisites

- Work from the repository root on `codex/305-api-request-diagnostics` with Go 1.26/toolchain go1.26.2 and repository dependencies available.
- Implement the design before expecting the new diagnostics from existing `--verbose`. The named `APIDiagnostics` tests below are planned acceptance tests, not existing passing tests at planning time.
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

## 1. Check implementation coverage and run focused validation

Add acceptance tests under the shared `TestAPIDiagnostics...` naming prefix in httpc, OAuth and cmd during implementation. Confirm the test lists contain the intended cases before running them; a successful command with no matching tests is not validation.

```sh
go test ./internal/services/httpc -list APIDiagnostics
go test ./internal/services/auth/oauth2 -list APIDiagnostics
go test ./cmd -list APIDiagnostics
go test ./internal/services/httpc -run 'TestAPIDiagnostics' -count=1
go test ./internal/services/auth/oauth2 -run 'TestAPIDiagnostics' -count=1
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
