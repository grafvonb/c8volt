# Ralph Memory

Feature: 222-camunda-version-check
Started: 2026-10-30T15:23:14Z

## Codebase Patterns

- `toolx.ParseCamundaRelease` returns numeric `uint64` major/minor and validated display text; patch overflow is rejected too. It is separate from the legacy command parser.
- `cluster.NewVersionCheck(cfg, client, log, callback)` snapshots effective config, performs no I/O at construction, and implements `httpc.BeforeRequestGate`. `Outcome()` returns `(domain.VersionCheckOutcome, bool)` only after the callback finishes. `WarningNeeded()` is independent of logger visibility.
- Install with `httpc.WithBeforeRequestGate(ctx, checker)`. `PrepareServiceDeps` now exposes the wrapper through additive `APIClient`; all 36 generated-client arguments in 33 constructors use it. Concrete `HTTPClient` remains unchanged for nested service construction and auth. Without a gate, or with bypass, Do delegates directly. Only original context cancellation prevents gated operational dispatch.
- The probe lazily calls the neutral cluster factory using a shallow client copy, refusing redirects and sharing transport/jar. Its child context has the two-second/shorter-client/parent deadline, `WithoutReadRetry`, bypass, and a fresh `BeginRequestRateAttempt` observation.

## Decisions

- T001 (primitive) and T002 (all shared dependency/adapter wiring) are complete. T003 owns CLI activation, logging, command fixtures, and documentation. CLI defaults remain unchanged until T003.
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

## Do Not Repeat

- Do not move the gate into RoundTrip: that consumes the original operation's HTTP timeout during the probe.
- Do not reuse the probe context or pacing observation for the operation, mutate the shared client's redirect policy, retry failed checks, or substitute the strict parser for legacy connection-test machine warnings.
- Do not enable CLI checking before T003 or validate against real Camunda services. Real-engine behavior remains unverified.

## Current Handoff

- Next dependency-ready work unit: T003 [US3], CLI activation and complete operator contract. T001 and T002 are complete; no blocking dependencies. Recheck tasks.md before selection. Install the lazy checker after auth, add the persistent opt-out and configured WARN renderer, preserve explicit connection-test JSON evidence while deduplicating human warnings, update execution/diagnostic/terminal fixtures and help/README, regenerate docs, and run the required isolated checks including the final full race suite. Preserve HTTP service context so the probe owns independent pacing evidence; do not globally disable the check in test resets.
