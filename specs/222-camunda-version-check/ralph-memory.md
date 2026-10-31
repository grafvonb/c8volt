# Ralph Memory

Feature: 222-camunda-version-check
Started: 2026-10-30T15:23:14Z

## Codebase Patterns

- `toolx.ParseCamundaRelease` returns numeric `uint64` major/minor and validated display text; patch overflow is rejected too. It is separate from the legacy command parser.
- `cluster.NewVersionCheck(cfg, client, log, callback)` snapshots effective config, performs no I/O at construction, and implements `httpc.BeforeRequestGate`. `Outcome()` returns `(domain.VersionCheckOutcome, bool)` only after the callback finishes. `WarningNeeded()` is independent of logger visibility.
- Install with `httpc.WithBeforeRequestGate(ctx, checker)`. `PrepareServiceDeps` now exposes the wrapper through additive `APIClient`; all 36 generated-client arguments in 33 constructors use it. Concrete `HTTPClient` remains unchanged for nested service construction and auth. Without a gate, or with bypass, Do delegates directly. Only original context cancellation prevents gated operational dispatch.
- The probe lazily calls the neutral cluster factory using a shallow client copy, refusing redirects and sharing transport/jar. Its child context has the two-second/shorter-client/parent deadline, `WithoutReadRetry`, bypass, and a fresh `BeginRequestRateAttempt` observation.

- `cmd/installVersionCheck` installs the invocation checker after HTTP/auth initialization, preserving the HTTP service context and independent pacing evidence. The configured logger renders typed outcomes at WARN; completed warning facts drive connection-test deduplication independently of logger filtering.
- Explicit connection tests retain the requested topology read and legacy JSON compatibility evidence, including permissive `8.9.garbage`. Human fallback validates display text separately. Actual connection failures use `handleCommandError` with a bounded reason and the original wrapped error for exit classification.

## Decisions

- T001 (primitive), T002 (shared adapter wiring), and T003 (CLI activation/operator contract) are complete. Checking is enabled by default; the persistent invocation-only `--no-version-check` flag skips it.
- Checker callback runs outside locks, before outcome publication and waiter notification. It must not recursively call `BeforeRequest`; it may inspect `Outcome`. Callback receives values without raw errors.
- Invalid configuration remains a bootstrap concern; lazy checker factory failures are retained as unavailable advisory outcomes. Effective version selection is never changed.

## Gotchas

- Generated topology decoding can return `json.SyntaxError` or `json.UnmarshalTypeError` before the adapter adds a domain sentinel. The checker classifies those as invalid-version outcomes without preserving raw text.
- `BeginRequestRateAttempt` allocates fresh evidence only with a positively paced HTTP service in context. Preserve that service context through CLI composition; otherwise probe pacing evidence could share an operational observation.
- `context.Background()` is not a pointer: tests comparing context identity use equality, not testify `Same`.
- Wiring tests execute every area/version without mocks of generated clients, including all legacy clients and nested variables. Real recursive topology now uses bypass; the earlier synthetic boundary test remains additional regression coverage.
- Cookie bootstrap uses the shared concrete client and therefore the same limiter/activity/diagnostic sequence. OAuth token refresh uses its existing separate unauthenticated concrete client, sharing diagnostics but not API pacing. Both remain outside the gate even with a gate-bearing context.
- The OAuth generated client appends `/token` to its configured base URL in fixtures. Variable-page JSON requires both `totalItems` and `hasMoreTotalItems`, even for empty results.
- Pacing evidence fixture: rate 0.45 with HTTP timeout 3s makes the next slot exceed the probe's 2s cap but fit the operation's own budget. A 4s parent then rejects a later retry, preserving earlier ambiguous/503 evidence. Unadmitted probe/retry produce no diagnostic network start.

- Existing command fake handlers serve a matching topology through `serveVersionCheckTopology`; version-sensitive helpers accept the configured release. Count the advisory in shared request/DEBUG sequences, while operation-only counters stay after the topology branch. Never globally opt out in reset helpers.
- Only four fixed-budget request-rate mutation regressions and the explicit failing-topology envelope helper opt out to isolate their original operational slots/retries. Shared-start/auth/reporting tests keep default checking enabled.
- A local process-definition terminal abort needs `--no-state-check --no-wait` to avoid the existing remote batch-capability preflight. Real-terminal abort and empty-discovery fixtures preserve prompt destinations and zero mutation calls.
- Facade error conversion normalizes an expired parent to `ferrors.ErrTimeout`; assert that public class rather than a raw context error. Local Args validation still has its existing empty stdout contract, even with `--json`.

## Reusable Commands

Run validation with inherited `C8VOLT_*`, `GO_WANT_*`, GOFLAGS/build tags, make overrides, GORACE, BASH_ENV and ENV cleared as described in quickstart.md. A temporary launcher used this iteration is `/tmp/c8volt-222-test.py`; recreate from the quickstart if absent.

- `go test ./toolx -run TestCamundaRelease -count=1`
- `go test ./internal/services/cluster/... -run TestVersionCheck -count=1`
- `go test ./internal/services/httpc/... -run TestVersionCheck -race -count=1`
- `go test ./toolx ./internal/services/cluster/... ./internal/services/httpc/... -list 'Test(CamundaRelease|VersionCheck)'`
- `go test ./toolx ./internal/services/cluster/... ./internal/services/httpc/... -race -count=1`
- `go vet ./toolx ./internal/services/cluster/... ./internal/services/httpc/...`
- `go test ./internal/services/common/... -run TestPrepareServiceDeps -count=1`
- `go test ./internal/services/cluster/... -run TestVersionCheck -race -count=1`
- `go test ./internal/services/httpc/... -run 'Test(VersionCheck|ReadRetry|RequestRate)' -race -count=1`
- `go test ./internal/services/... ./c8volt/... -count=1`
- `go vet ./internal/services/... ./c8volt/...`

- `go test ./cmd -run 'Test(VersionCheck|APIDiagnostics|RequestRate|ConfigTestConnection|BulkCompletionDiagnosticLogging|CapabilityDocument|AllCommandHelp)' -count=1`
- `go test ./internal/services/httpc/... -run 'Test(VersionCheck|ReadRetry|APIDiagnostics|RequestRate)' -race -count=1`
- `go test ./cmd -run 'Test(VersionCheck|BulkCompletionSummaryLogging|ScalarJob|UpdateProcessInstance|UpdateUserTaskAssignmentDiagnostic|UpdateUserTaskAssignmentPreflight|UpdateUserTaskAssignmentUnchanged|UpdateUserTaskAssignmentMixed|UpdateUserTaskCompletionDiagnostic|UpdateUserTaskCompletionPreflightDiagnostics|WalkProcessInstanceCommand_WithListenersJSON)' -count=1`
- `go test ./cmd -run 'Test(VersionCheckHelp|CapabilityDocumentVersionCheck|AllCommandHelp)' -count=1`
- `make docs-content`
- `make test PKG=./...` — final isolated race suite passed in iteration 3 (cmd: 405.968s).
- `git diff --check`

## Do Not Repeat

- Do not move the gate into RoundTrip: that consumes the original operation's HTTP timeout during the probe.
- Do not reuse the probe context or pacing observation for the operation, mutate the shared client's redirect policy, retry failed checks, or substitute the strict parser for legacy connection-test machine warnings.
- Do not validate implementation against real Camunda services. Real-engine reporting, deployed authentication, and external proxy/network behavior remain unverified.

## Current Handoff

- Feature complete; no handoff required.
