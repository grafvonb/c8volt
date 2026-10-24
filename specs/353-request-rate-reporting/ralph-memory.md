# Ralph Memory

Feature: 353-request-rate-reporting
Started: 2026-10-24T17:30:57Z

## Codebase Patterns

- Root persistent flags bind through `cmd/root_config.go`'s `resolverBindings`; reflection-based configuration/profile overlays already preserve explicit zero. Viper weak decoding needs a raw scalar check to reject empty strings and booleans for a numeric rate.
- Shared transport order is auth → read retry → optional request rate → logging → diagnostics → underlying transport. Both traversal helpers in `httpc/service.go` must understand each wrapper. Separate OAuth clients receive only the shared diagnostic collector.
- `RetryCamundaMutation` callbacks propagate their callback context to generated clients. `BeginRequestRateAttempt` creates fresh atomic local-failure evidence only when the context-carried HTTP service has positive pacing; `Failure()` is nil-safe. Deadline/cancellation identities survive via `RequestRateWaitError.Unwrap` and existing facade conversion.
- `config show` renders YAML even with the root JSON flag; use YAML decoding for effective-config assertions. Root command tests must reset singleton flags and capture configured output streams separately.

## Decisions

- Iteration 1 selects only T001 (US1 transport/config/retry contract), as explicitly separated from T002's broader command acceptance work in tasks.md. No task expansion or story mixing.
- Use `golang.org/x/time v0.16.0`, burst 1, with no wrapper/limiter at rate zero. Its module requires `go 1.26.0`, accounting for the normalized Go directive; toolchain remains 1.26.2.
- A live-context admission rejection is an unsent deadline error; recheck cancellation before transport delegation. Exclude local errors from read/backend-mutation retry classification. After each mutation callback, recover recorded local failure before normal retry classification, preserving the callback result.
- No new diagnostic printing, public facade APIs, worker policy, or mutation workflow changes. README/root metadata and generated CLI docs explain pacing; reporting remains T003.

## Gotchas

- `http.Client.Timeout` may replace a typed transport error. For custom transports, its context timer and legacy timeout timer race: waiting on `req.Context().Done()` alone does not prove replacement. Timeout fixtures also wait on `req.Cancel`, closed after net/http marks its timeout, to deterministically exercise the lost-type case with a live parent.
- `testx` imports `services/common`, which imports `services`; importing `testx` in same-package `internal/services` tests creates an import cycle. Use a local mutex-protected timestamp slice there; other packages can use `testx.SafeSlice`.
- Real delegation timing checks allow 5ms scheduler/timer tolerance. Concurrency tests hold all responses until every start arrives, proving the limiter does not serialize in-flight requests.
- When T003 reuses the wrapper for zero-rate reporting, update `BeginRequestRateAttempt`'s positive-pacing test: wrapper existence alone will no longer prove pacing is enabled. Retain the zero-rate original-context/no-observation path.

## Reusable Commands

- Sanitize inherited `C8VOLT_*`, `GO_WANT_HELPER_PROCESS`, `MAKEFLAGS` and `MFLAGS`; set `GOFLAGS=''` before validation. No integration tags or live targets. The temporary launcher `/tmp/c8volt-request-rate-check.py` performs that cleanup for this session; recreate if absent.
- `GOFLAGS='' go test ./config ./internal/services/httpc ./internal/services ./cmd -list RequestRate` confirms nonempty selections in every affected package.
- `GOFLAGS='' go test ./config ./internal/services/httpc ./internal/services ./cmd -run 'RequestRate|ReadRetry|RetryCamundaMutation|APIDiagnostics|Config|Diagnostics' -count=1` covers focused behavior and existing related regressions.
- `GOFLAGS='' make test` runs the required full race suite for shared runtime/dependency changes. `GOFLAGS='' make docs-content` regenerates CLI docs and README-derived docs/index.md; `git diff --check` checks whitespace.

## Do Not Repeat

- Do not parse HTTP error strings to detect unsent attempts, allocate observations at rate zero, log transport details at normal levels, or share rate state with standalone OAuth clients.
- Do not edit generated CLI markdown manually; persistent flag width changes naturally reformat inherited option tables throughout docs/cli.
- Do not claim T002's complete mutation/output matrix or T003's reporting/benchmarks from T001's transport and root tests. No live-engine validation is authorized or performed.

## Current Handoff

- Next dependency-ready work unit: T002 [US1], mutation-command compatibility and compact local-wait diagnostics. T001 is complete and validated; recheck tasks.md before starting. No blocking dependencies.
- Reuse T001's request-rate helpers and existing command/terminal/subprocess fixtures. Trace local admission and repeated backend failures through facade conversion, configured logger, selected-format rendering and exit; keep the full T002 matrix in scope without unrelated repairs.
- T003 [US2] remains blocked on T002. Reporting, invocation cleanup lifecycle, interval configuration and benchmarks are unimplemented; do not mix them into the next US1 iteration.
