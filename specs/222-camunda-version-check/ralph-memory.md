# Ralph Memory

Feature: 222-camunda-version-check
Started: 2026-10-30T15:23:14Z

## Codebase Patterns

- `toolx.ParseCamundaRelease` returns numeric `uint64` major/minor and validated display text; patch overflow is rejected too. It is separate from the legacy command parser.
- `cluster.NewVersionCheck(cfg, client, log, callback)` snapshots effective config, performs no I/O at construction, and implements `httpc.BeforeRequestGate`. `Outcome()` returns `(domain.VersionCheckOutcome, bool)` only after the callback finishes. `WarningNeeded()` is independent of logger visibility.
- Install with `httpc.WithBeforeRequestGate(ctx, checker)` and wrap generated Do calls with `httpc.NewBeforeRequestClient(concreteClient)`. Without a gate, or with `httpc.BypassBeforeRequest`, Do delegates directly. Only original context cancellation prevents gated operational dispatch.
- The probe lazily calls the neutral cluster factory using a shallow client copy, refusing redirects and sharing transport/jar. Its child context has the two-second/shorter-client/parent deadline, `WithoutReadRetry`, bypass, and a fresh `BeginRequestRateAttempt` observation.

## Decisions

- T001 is the complete internal primitive only. T002 owns shared dependency/adapter wiring; T003 owns CLI activation, logging, fixture updates, and documentation. Existing CLI behavior is unchanged in this commit.
- Checker callback runs outside locks, before outcome publication and waiter notification. It must not recursively call `BeforeRequest`; it may inspect `Outcome`. Callback receives values without raw errors.
- Invalid configuration remains a bootstrap concern; lazy checker factory failures are retained as unavailable advisory outcomes. Effective version selection is never changed.

## Gotchas

- Generated topology decoding can return `json.SyntaxError` or `json.UnmarshalTypeError` before the adapter adds a domain sentinel. The checker classifies those as invalid-version outcomes without preserving raw text.
- `BeginRequestRateAttempt` allocates fresh evidence only with a positively paced HTTP service in context. Preserve that service context through CLI composition; otherwise probe pacing evidence could share an operational observation.
- `context.Background()` is not a pointer: tests comparing context identity use equality, not testify `Same`.
- All T001 HTTP fixtures use private transports; no live endpoints are accessed. The recursive-boundary test simulates adapter wrapping until T002 installs it for real.

## Reusable Commands

Run validation with inherited `C8VOLT_*`, `GO_WANT_*`, GOFLAGS/build tags, make overrides, GORACE, BASH_ENV and ENV cleared as described in quickstart.md. A temporary launcher used this iteration is `/tmp/c8volt-222-test.py`; recreate from the quickstart if absent.

- `go test ./toolx -run TestCamundaRelease -count=1`
- `go test ./internal/services/cluster/... -run TestVersionCheck -count=1`
- `go test ./internal/services/httpc/... -run TestVersionCheck -race -count=1`
- `go test ./toolx ./internal/services/cluster/... ./internal/services/httpc/... -list 'Test(CamundaRelease|VersionCheck)'`
- `go test ./toolx ./internal/services/cluster/... ./internal/services/httpc/... -race -count=1`
- `go vet ./toolx ./internal/services/cluster/... ./internal/services/httpc/...`

## Do Not Repeat

- Do not move the gate into RoundTrip: that consumes the original operation's HTTP timeout during the probe.
- Do not reuse the probe context or pacing observation for the operation, mutate the shared client's redirect policy, retry failed checks, or substitute the strict parser for legacy connection-test machine warnings.
- Do not enable CLI checking before T003 or validate against real Camunda services. Real-engine behavior remains unverified.

## Current Handoff

- Next dependency-ready work unit: T002 [US2], adding `ServiceDeps.APIClient` and migrating the 36 generated-client arguments in 33 constructors. Keep concrete HTTPClient for nested services and auth. T001 is complete; no blocking dependencies. Recheck tasks.md before selection, then prove actual adapter wiring, timeout/auth/pacing/diagnostic isolation and constructor-wide regressions. T003 remains dependent on T002.
