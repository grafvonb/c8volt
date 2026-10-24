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

## Decisions

- Iteration 1 selects only T001 (US1 transport/config/retry contract), as explicitly separated from T002's broader command acceptance work in tasks.md. No task expansion or story mixing.
- Use `golang.org/x/time v0.16.0`, burst 1, with no wrapper/limiter at rate zero. Its module requires `go 1.26.0`, accounting for the normalized Go directive; toolchain remains 1.26.2.
- A live-context admission rejection is an unsent deadline error; recheck cancellation before transport delegation. Exclude local errors from read/backend-mutation retry classification. After each mutation callback, recover recorded local failure before normal retry classification, preserving the callback result.
- No new public facade APIs, worker policy or mutation workflows. README/root metadata and generated CLI docs explain pacing and bounded local-failure diagnostics; reporting remains T003.

- Iteration 2 completes only T002 [US1]. Preserve machine error text/schema and partial reports; add compact unsent evidence to the existing human error paths. Quiet process-instance progress suppresses per-tree warnings so the final configured ERROR owns failure reporting. Cancellation summaries retain already-submitted facts when a confirmation read is locally unsent; DEBUG summaries omit raw error chains.

## Gotchas

- `TestGetViewFilesAvoidBackendOwnership` forbids every internal-service import in `cmd_views_*.go`, even a typed-error predicate. Keep that classification in `cmd/cmd_errors.go`; renderers call the command-support predicate and own bounded wording. Do not add a facade API or relax the architecture assertion.

- `http.Client.Timeout` may replace a typed transport error. For custom transports, its context timer and legacy timeout timer race: waiting on `req.Context().Done()` alone does not prove replacement. Timeout fixtures also wait on `req.Cancel`, closed after net/http marks its timeout, to deterministically exercise the lost-type case with a live parent.
- `testx` imports `services/common`, which imports `services`; importing `testx` in same-package `internal/services` tests creates an import cycle. Use a local mutex-protected timestamp slice there; other packages can use `testx.SafeSlice`.
- Real delegation timing checks allow 5ms scheduler/timer tolerance. Concurrency tests hold all responses until every start arrives, proving the limiter does not serialize in-flight requests.
- When T003 reuses the wrapper for zero-rate reporting, update `BeginRequestRateAttempt`'s positive-pacing test: wrapper existence alone will no longer prove pacing is enabled. Retain the zero-rate original-context/no-observation path.

- Reusing singleton-root fixture suites twice retains command context (including tenant-summary emission state) unless cleared. `resetRequestRateCommandContexts` clears the full tree before/after each reused suite; avoid changing production invocation lifecycle in T002.
- Purge accepts one valid long numeric `--key`, unlike the comma-separated keys on direct delete commands. Failed resource reports omit `ok` rather than emit false. Assertions must preserve these existing contracts.
- To test a locally unsent confirmation read after accepted cancellation, give the workflow a 5ms backoff timeout and pace at 20/s with the ordinary HTTP timeout. Preflight/submission/family reads complete; the confirmation's shorter deadline rejects the next read without dispatch.

## Reusable Commands

- Sanitize inherited `C8VOLT_*`, `GO_WANT_HELPER_PROCESS`, `MAKEFLAGS` and `MFLAGS`; set `GOFLAGS=''` before validation. No integration tags or live targets. The temporary launcher `/tmp/c8volt-request-rate-check.py` performs that cleanup for this session; recreate if absent.
- `GOFLAGS='' go test ./config ./internal/services/httpc ./internal/services ./cmd -list RequestRate` confirms nonempty selections in every affected package.
- `GOFLAGS='' go test ./config ./internal/services/httpc ./internal/services ./cmd -run 'RequestRate|ReadRetry|RetryCamundaMutation|APIDiagnostics|Config|Diagnostics' -count=1` covers focused behavior and existing related regressions.
- `GOFLAGS='' make test` runs the required full race suite for shared runtime/dependency changes. `GOFLAGS='' make docs-content` regenerates CLI docs and README-derived docs/index.md; `git diff --check` checks whitespace.

- `GOFLAGS='' go test ./cmd -run RequestRate -count=1` covers the reusable mutation matrices; `go test ./c8volt/ferrors -count=1` covers preserved cause/class behavior. `TestRequestRateCancellationConfirmation` covers the accepted-then-unsent distinction; `TestGetViewFilesAvoidBackendOwnership` checks the error-support/view boundary.

## Do Not Repeat

- Do not parse HTTP error strings to detect unsent attempts, allocate observations at rate zero, log transport details at normal levels, or share rate state with standalone OAuth clients.
- Do not edit generated CLI markdown manually; persistent flag width changes naturally reformat inherited option tables throughout docs/cli.
- No live-engine validation is authorized or performed. Reporting lifecycle, interval configuration, combined reporting modes and benchmarks remain T003; T002 adds no reporter or counter.

## Current Handoff

- Next dependency-ready work unit: T003 [US2], configurable DEBUG reporting, invocation cleanup, combined-mode acceptance and benchmarks. T001 and T002 are complete; recheck tasks.md before starting. No blocking dependencies.
- Reuse the shared transport and `cmd/request_rate_mutation_test.go` fixtures. The compatibility matrix injects rate through the environment; the focused subprocess runner clears inherited C8VOLT settings and takes explicit arguments/configuration. Add reporting configuration to test-owned files when extending that runner.
- Keep the positive-rate-only retry observation fast path when zero-rate reporting adds a wrapper. Reporting lifecycle/counting/interval/benchmarks remain unimplemented. Preserve bounded command diagnostics, the cause-aware facade wrapper and quiet final-error ownership.
- Final T002 validation passed: focused mutation/facade/confirmation checks, architecture ownership regression, regenerated docs, gofmt/diff checks and the complete `make test` race suite. No live-engine validation was performed.
