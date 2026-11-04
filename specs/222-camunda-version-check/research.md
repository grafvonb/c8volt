# Research: Advisory Camunda Version Check

**Date**: 2026-10-30 | **Source**: [spec.md](spec.md)

Repository inspection and two read-only research agents resolved the unknowns below. No runtime tests or live-service requests were run during planning. Dependency choices use the versions already pinned in `go.mod`.

## 1. Request boundary and timeout isolation

**Decision**: Introduce a context-scoped gate consulted by a small internal `Do(*http.Request)` wrapper before the underlying `http.Client.Do`. Add `APIClient` to `internal/services/common.ServiceDeps`, retaining its concrete `HTTPClient`. `PrepareServiceDeps` supplies the wrapper; absent an invocation gate, it delegates directly.

**Rationale**: `cmd/root_services.go` owns one invocation's HTTP service, but commands can construct multiple facades. Checking in `NewCli` is premature for local previews, validation, and declined confirmations. Checking inside `RoundTrip` starts the ordinary `http.Client.Timeout` before the advisory read: a short request timeout can expire solely because the advisory runs. A pre-Do gate avoids that error without changing redirect, response-body, or request-timeout semantics for operational requests.

**Integration evidence**: There are 36 generated-client `WithHTTPClient(deps.HTTPClient)` arguments across 33 versioned `service.go` files, covering 11 areas and v88/v89/v810, plus legacy Operate/Tasklist clients. Switch those arguments to `deps.APIClient`. Retain the three process-instance constructors' concrete `deps.HTTPClient` arguments when constructing nested variable services. Existing generated clients accept a Do interface. Public `c8volt.WithHTTPClient`, factory signatures, and generated code remain unchanged.

**Alternatives considered**: Root pre-run checking makes local-only paths perform network work; per-command/per-facade checks duplicate policy and miss internal requests; a RoundTripper check steals operational timeout budget; resetting deadlines inside transport would change body/redirect semantics; changing all public client signatures is unnecessary.

## 2. Policy owner and gateway source

**Decision**: Put the checker in existing `internal/services/cluster/version_check.go`. Use `cluster.New` and its version-neutral `GetClusterTopology` to read `domain.Topology.GatewayVersion`. Keep context-gate plumbing and the pre-Do wrapper in `internal/services/httpc`. Wire the checker as an invocation support service after auth initialization; command code never calls generated clients or versioned adapters.

**Rationale**: `internal/services/cluster/{v88,v89,v810}/service.go` already retrieves topology and maps gateway version. `cluster/common/topology.go` already imports `httpc`; therefore `httpc` must not import `cluster`. A callback/interface gate in `httpc` with a concrete checker in `cluster` keeps the dependency direction valid and gives existing owners the new behavior.

**Alternatives considered**: A new versioncheck service package is unnecessary; raw HTTP in cmd duplicates endpoint/error mapping; selecting an adapter from the observed version violates the issue. Using a global checker would leak results between invocations.

## 3. Attempt lifetime, recursion, authentication, and pacing

**Decision**: One checker per invocation, initialized without requests. Concurrent first requests share one terminal outcome, including failure; waiting callers can cancel independently. Use a mutex-protected started flag and completion channel, with no locks held across network I/O. The winning caller performs the bounded probe synchronously; no timer loop, background monitor, or per-request goroutine is needed.

The probe gets a private derived context with gate bypass, `httpc.WithoutReadRetry`, and a fresh `httpc.BeginRequestRateAttempt` observation. Derive its deadline from the caller, capped at two seconds and any shorter positive configured HTTP timeout. Cancel only that derived context. The original request keeps its original context and ordinary HTTP timeout; an expired parent still prevents dispatch. Close the original body if the wrapper returns early without delegating, following HTTP client ownership expectations.

Use a shallow copy of the invocation HTTP client for the probe, sharing transport, cookie jar, authentication editor, limiter, and diagnostics, while refusing HTTP redirects for this advisory request. A redirect is an unavailable lookup, not a second gateway lookup. Do not mutate the shared client's timeout or redirect policy. The probe context bypasses the gate when the topology adapter reaches the common wrapper.

**Rationale**: `cmd/root_services.go` initializes authentication before publishing remote services. Auth bootstrap and token refresh use the original concrete HTTP client, outside the operational wrapper. The existing chain is Auth → ReadRetry → RequestRate → Log/Diagnostics → transport. `WithoutReadRetry` already disables safe-read retry for advisory contexts. `BeginRequestRateAttempt` replaces the observation pointer under positive pacing, preventing probe admission failure from contaminating a mutation's submission evidence. The shared limiter still counts and paces actual probe dispatches.

**Alternatives considered**: Separate unauthenticated/unpaced clients lose established controls; reusing the original request's observation can misclassify actual work; reusing the probe context cancels later work; a bare recursive once callback deadlocks; detached unbounded goroutines outlive the invocation.

**Compatibility consequence**: The advisory adds a request-rate slot and up to two seconds of check waiting. Ordinary pacing may add its own existing delay after the check. The two-second bound applies to the check, not total command duration. No existing workflow deadline is extended.

## 4. Parsing and error classification

**Decision**: Add a small pure helper in `toolx/camunda_release.go` for bounded release parsing, used by the new checker and safe human fallback rendering in `cmd/config_diagnostics.go`. Accept optional `v`/`V`, numeric major/minor, optional numeric patch, and conventional ASCII prerelease/build suffixes; compare numeric major/minor only. Accept surrounding spaces, but reject control characters, negative/overflowing components, empty components, extra numeric components, and values exceeding 128 bytes. Preserve a validated trimmed display value for mismatch warnings. Use no external semver dependency.

**Rationale**: Existing `parseMajorMinorVersion` in `cmd/config_diagnostics.go` handles ordinary patch and prerelease versions but also accepts negative components and arbitrary trailing material. It cannot safely supply normal-level diagnostic text unchanged. Preserve its existing comparison/classification path for explicit connection-test machine warnings, including permissive historical inputs, rather than silently changing JSON evidence. The new strict helper is a distinct advisory/safe-display contract, not a replacement for legacy machine classification. Existing tests require `8.10`, `8.10.7`, and `8.10.0-alpha4` to match configured 8.10; preserve those cases and existing alias normalization in `toolx`/configuration.

Classify errors through typed causes: unauthorized/forbidden → `access denied`; deadline/network timeout → `timeout`; malformed or absent version → `invalid gateway version`; cancellation → `canceled`; other network/backend/redirect failures → `unavailable`. Do not build warning reasons from `err.Error()`. `httpc.HttpStatusErr` wraps domain sentinels but includes method, URL, and body in its text; only classification is safe at WARN. Call topology without verbose workflow options, and keep detailed HTTP visibility in the existing redacted DEBUG path.

**Alternatives considered**: Float/string comparison mishandles release lines; unrestricted parser reuse permits unsafe values; replacing legacy machine classification would change existing JSON evidence; echoing backend error strings violates diagnostic rules. The narrow separate safe parser is justified by these different compatibility contracts.

## 5. CLI integration and duplicate diagnostics

**Follow-up (2026-10-31)**: The CLI-only decision below is superseded by the
requested persisted `app.no_version_check` setting. Existing `config.App` profile
overlay and reflective environment binding already preserve explicitly configured
false values through `ResolveEffectiveConfig`. Bind the existing flag to this
field and read only the resolved value in the invocation installer; no separate
precedence mechanism or profile schema is needed.


**Decision**: Add persistent boolean `--no-version-check`, default false, without adding a persisted configuration key. Keep flag translation and checker installation in a focused `cmd/root_version_check.go`, called from `installRemoteCommandServices` after auth. Supply the configured command logger through a compact outcome renderer in `cmd/cmd_views_version_check.go`. The checker returns typed facts to that renderer exactly once; only ordinary parent cancellation can propagate to operational dispatch, never the advisory failure itself.

`config test-connection` is remote despite being in `bypassRootBootstrap`: it calls `installRemoteCommandServices` explicitly. Its operational topology read remains separate. Preserve its actual-topology-derived JSON `warnings` values. Suppress its legacy WARN loop when the automatic checker already produced a warning outcome, regardless of logger filtering. If the advisory matched but the subsequent actual result differs, retain one explicit compatibility warning. When the new check is disabled, preserve explicit connection-test compatibility evaluation. Bound/sanitize any touched human fallback for unsafe actual-version values without changing existing machine warning evidence.

**Rationale**: `cmd/root.go` resolves quiet before debug; quiet sets ERROR level. Normal logger filtering therefore suppresses WARN without disabling the probe. Existing `config_test.go` asserts connection-test warnings in JSON and on stderr, so duplicate suppression requires intentional assertion updates rather than deleting machine evidence.

**Alternatives considered**: Excluding all bootstrap-bypass commands misses remote work; removing connection-test warnings changes result contracts; suppressing warnings based on logger visibility loses behavior under filtering; globally suppressing explicit diagnostics after a silent match would hide a changed second observation.

## 6. Validation and fixture compatibility

**Decision**: Implement isolated unit and command/subprocess tests at the changed boundaries, followed by the full race-enabled unit suite because common request wiring and concurrent callers are affected. Run no runtime tests for the planning-only diff.

**Evidence**: `root_api_diagnostics_test.go`, `root_request_rate_test.go`, `request_rate_mutation_test.go`, and `update_job_batch_logging_test.go` provide stream, logger, request-count, pacing, and partial-result fixtures. Strict existing fake handlers may reject `/v2/topology`, and counters may assume the first request is operational. Update affected fixtures to serve a matching advisory result and account for its start; use explicit opt-out only in tests deliberately isolating unrelated behavior. Never globally disable checking in reset helpers.

`testx` subprocess helpers inherit environment. Clear inherited `C8VOLT_*`, helper-process markers, GOFLAGS/build tags, and make overrides before tests, and use test-owned config/server addresses. `Makefile:test` runs `go test ./... -race -count=1` by default; integration sources are build-tag gated. No integration targets, live tags, credentials, clusters, or engine fixtures are part of this plan.

**Alternatives considered**: Renderer-only assertions cannot detect logger bypass or incorrect timing; happy-path tests miss error-envelope duplication; live testing violates the implementation boundary.

## Resolution status

All research questions are resolved. The two-second timeout remains the explicit specification assumption. The implementation design requires no new dependency, generated API change, public facade contract change, or live-environment gate. [plan.md](plan.md) records the resulting design and validation scope.
