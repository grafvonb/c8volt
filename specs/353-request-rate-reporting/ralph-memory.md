# Ralph Memory

Feature: 353-request-rate-reporting
Started: 2026-10-24T17:30:57Z

## Codebase Patterns

- Root persistent flags bind through `cmd/root_config.go`'s `resolverBindings`; reflection-based configuration/profile overlays already preserve explicit zero. Viper weak decoding needs a raw scalar check to reject empty strings and booleans for a numeric rate.
- Shared transport order is auth → read retry → optional request rate → logging → diagnostics → underlying transport. Both traversal helpers in `httpc/service.go` must understand each wrapper. Separate OAuth clients receive only the shared diagnostic collector.
- `RetryCamundaMutation` callbacks propagate their callback context to generated clients. `BeginRequestRateAttempt` creates fresh atomic local-failure evidence only when the context-carried HTTP service has positive pacing; `Failure()` is nil-safe. Deadline/cancellation identities survive via `RequestRateWaitError.Unwrap` and existing facade conversion.
- `config show` renders YAML even with the root JSON flag; use YAML decoding for effective-config assertions. Root command tests must reset singleton flags and capture configured output streams separately.

- `ferrors.wrap` must preserve typed request-rate causes through its existing inspectable error wrapper, including context identity and selected outer-class precedence. Ordinary errors retain the original class-only contract. Bounded command reasons use typed evidence, never error-string parsing.
- Mutation acceptance lives in `cmd/request_rate_mutation_test.go`: reuse existing suites with `C8VOLT_HTTP_REQUEST_RATE`, then add shared-start, local deadline, interruption and accepted-cancellation confirmation tests. `runRequestRateMutation` captures the configured/inherited writer separately from process stderr, so even formatted direct writes are detected.

- `http.request_rate_report_interval` is a configuration-only duration string: omitted `5s`, explicit `0s` disables, explicit empty remains invalid through configured-key-aware HTTP normalization. Raw Config/HTTP test literals that bypass normalization must supply the validated interval.
- Reporter state lives in `httpc`: optional atomic cumulative counter, one cancellable ticker, count/time deltas using current monotonic snapshot time, and one complete DEBUG message. Dispatch neither logs nor acquires a reporting lock; rate-zero measurement has no limiter.
- `cmd/root_request_rate.go` registers one Cobra finalizer and holds only the current service close callback. `installRemoteCommandServices` clears stale ownership, passes the invocation context and registers close before authentication. Failed initialization closes immediately; returned invocations consume the handle. Direct DEBUG service tests own close.

## Decisions

- Iteration 1 selects only T001 (US1 transport/config/retry contract), as explicitly separated from T002's broader command acceptance work in tasks.md. No task expansion or story mixing.
- Use `golang.org/x/time v0.16.0`, burst 1, with no wrapper/limiter at rate zero. Its module requires `go 1.26.0`, accounting for the normalized Go directive; toolchain remains 1.26.2.
- A live-context admission rejection is an unsent deadline error; recheck cancellation before transport delegation. Exclude local errors from read/backend-mutation retry classification. After each mutation callback, recover recorded local failure before normal retry classification, preserving the callback result.
- No new public facade APIs, worker policy or mutation workflows. README/root metadata and generated CLI docs explain pacing and bounded local-failure diagnostics; T003 subsequently adds reporting.

- Iteration 2 completes only T002 [US1]. Preserve machine error text/schema and partial reports; add compact unsent evidence to the existing human error paths. Quiet process-instance progress suppresses per-tree warnings so the final configured ERROR owns failure reporting. Cancellation summaries retain already-submitted facts when a confirmation read is locally unsent; DEBUG summaries omit raw error chains.

- Iteration 3 completes T003 [US2] as one work unit: reporting configuration, accounting/lifecycle, combined mutation acceptance, docs and benchmarks. No follow-up task or story mixing. All feature tasks are complete.
- Keep existing output schemas and mutation workflows unchanged. Report only actual dispatched attempts, including failed responses/retries; unsent work does not count. Empty windows are valid, no final partial window is required, and zero interval leaves existing HTTP DEBUG diagnostics enabled.

## Gotchas

- `TestGetViewFilesAvoidBackendOwnership` forbids every internal-service import in `cmd_views_*.go`, even a typed-error predicate. Keep that classification in `cmd/cmd_errors.go`; renderers call the command-support predicate and own bounded wording. Do not add a facade API or relax the architecture assertion.

- `http.Client.Timeout` may replace a typed transport error. For custom transports, its context timer and legacy timeout timer race: waiting on `req.Context().Done()` alone does not prove replacement. Timeout fixtures also wait on `req.Cancel`, closed after net/http marks its timeout, to deterministically exercise the lost-type case with a live parent.
- `testx` imports `services/common`, which imports `services`; importing `testx` in same-package `internal/services` tests creates an import cycle. Use a local mutex-protected timestamp slice there; other packages can use `testx.SafeSlice`.
- Real delegation timing checks allow 5ms scheduler/timer tolerance. Concurrency tests hold all responses until every start arrives, proving the limiter does not serialize in-flight requests.
- The reporting wrapper exists at rate zero when DEBUG and a positive interval enable counting. `BeginRequestRateAttempt` checks the limiter, not wrapper presence, retaining the zero-rate original-context/no-observation path.

- Reusing singleton-root fixture suites twice retains command context (including tenant-summary emission state) unless cleared. `resetRequestRateCommandContexts` clears the full tree before/after each reused suite; avoid changing production invocation lifecycle in T002.
- Purge accepts one valid long numeric `--key`, unlike the comma-separated keys on direct delete commands. Failed resource reports omit `ok` rather than emit false. Assertions must preserve these existing contracts.
- To test a locally unsent confirmation read after accepted cancellation, give the workflow a 5ms backoff timeout and pace at 20/s with the ordinary HTTP timeout. Preflight/submission/family reads complete; the confirmation's shorter deadline rejects the next read without dispatch.

- `testing/synctest.Wait` does not protect a buffer read against a future tick's write. Reuse `diagnosticCallbackWriter` for synchronized capture; always cancel/join the reporter on test failure. The initial full race run caught this fixture race, fixed before the final passing full suite.
- Root reporting tests release their fake response only after the first one-start summary is captured, so an early zero window cannot accidentally end the command before its request is reported. Report-message assertions parse elapsed duration and verify measured precision and configured logger framing.

## Reusable Commands

- Sanitize inherited `C8VOLT_*`, `GO_WANT_HELPER_PROCESS`, `MAKEFLAGS` and `MFLAGS`; set `GOFLAGS=''` before validation. No integration tags or live targets. The temporary launcher `/tmp/c8volt-request-rate-check.py` performs that cleanup for this session; recreate if absent.
- `GOFLAGS='' go test ./config ./internal/services/httpc ./internal/services ./cmd -list RequestRate` confirms nonempty selections in every affected package.
- `GOFLAGS='' go test ./config ./internal/services/httpc ./internal/services ./cmd -run 'RequestRate|ReadRetry|RetryCamundaMutation|APIDiagnostics|Config|Diagnostics' -count=1` covers focused behavior and existing related regressions.
- `GOFLAGS='' make test` runs the required full race suite for shared runtime/dependency changes. `GOFLAGS='' make docs-content` regenerates CLI docs and README-derived docs/index.md; `git diff --check` checks whitespace.

- `GOFLAGS='' go test ./cmd -run RequestRate -count=1` covers the reusable mutation matrices; `go test ./c8volt/ferrors -count=1` covers preserved cause/class behavior. `TestRequestRateCancellationConfirmation` covers the accepted-then-unsent distinction; `TestGetViewFilesAvoidBackendOwnership` checks the error-support/view boundary.

- `GOFLAGS='' go test ./internal/services/httpc -run '^$' -bench RequestRate -benchmem` selects the dispatch microbenchmark. Iteration 3 on Darwin arm64 / Apple M3 Pro: pass-through 1.950 ns/op, zero/reporting-off 1.959 ns/op, zero/reporting-on 4.180 ns/op; all 0 B/op and 0 allocs/op. These measure the optional boundary with a fake delegate, not real engine capacity or zero total overhead.
- `GOFLAGS='' go test ./internal/services/httpc -run 'RequestRateReporterCadence|RequestRateWindows|RequestRateConcurrentWindow' -race -count=10` passed after synchronizing the virtual-clock capture. Final `make test` passed across all packages after the last executable test change; final docs generation, formatting and whitespace checks passed.

## Do Not Repeat

- Do not parse HTTP error strings to detect unsent attempts, allocate observations at rate zero, log transport details at normal levels, or share rate state with standalone OAuth clients.
- Do not edit generated CLI markdown manually; persistent flag width changes naturally reformat inherited option tables throughout docs/cli.
- No live-engine validation is authorized or performed. Do not infer production capacity or GOAWAY elimination from fake tests or dispatch microbenchmarks.

## Current Handoff

- Feature complete; no handoff required.
