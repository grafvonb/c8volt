# Implementation Plan: Opt-in API Request Diagnostics

**Branch**: `codex/305-api-request-diagnostics` | **Date**: 2026-09-12 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/305-api-request-diagnostics/spec.md`; source issue #305.

## Summary

Extend the existing `--debug` behavior to log compact technical evidence for actual HTTP exchanges on configured stderr: `api #45 GET /v2/topology: status=200 total=180ms headers=170ms body=10ms dns=8ms tcp=24ms tls=61ms`, followed by available safe context. Keep issue #305; implement a generic interceptor without workflow-specific instrumentation or narrative diagnosis. Implement one reusable observer in `internal/services/httpc`, below retries, with transparent body wrappers and composed HTTP trace hooks. Share an invocation-scoped collector across the main client and the separate OAuth token client. Preserve command results, authentication, retries, body consumption and mutation verification.

## Technical Context

**Language/Version**: Go 1.26, toolchain go1.26.2 from go.mod.

**Primary Dependencies**: Existing Cobra 1.10.2, pflag, configuration/Viper, standard `net/http`, `net/http/httptrace`, `io`, `sync`, `time`; no additional production dependencies.

**Storage**: None. Per-invocation collector and per-active-exchange metadata only; no payload retention or historical reports.

**Testing**: Standard Go tests with testify, httptest/testx IPv4 servers, existing command subprocess/terminal runner; race detector and full `make test` gate.

**Target Platform**: Existing Go CLI targets. Real-terminal validation uses existing Linux/macOS testx support, with explicit unsupported-platform reporting elsewhere.

**Project Type**: CLI with public Go facade and version-specific internal services.

**Performance Goals**: No additional network requests, eager body reads, payload buffers or background polling. Metadata/counter work per exchange and observed read; memory independent of body size. Disabled path installs no observer. Add benchmark evidence for enabled/disabled small and streamed responses rather than inventing a product latency target.

**Constraints**: Current invocation only; stderr-only DEBUG records subject to quiet and configured level filtering; no generated-client or individual command observation; original transport/body errors and ownership preserved; no secret/body dumps. Synchronous stderr backpressure can add output latency, as with existing logging; it is not included in reported exchange duration.

**Scale/Scope**: Existing API-backed commands and current Camunda 8.7–8.10, Operate and Tasklist clients; authentication exchanges included. One record per observable RoundTrip invocation, including explicit retries and redirects. Internal network-transport retransmissions are not separate observable exchanges. No new analysis command or result-envelope schema.

## Constitution Check

*Initial gates evaluated before design; re-evaluated against the completed design.*

| Principle / gate | Initial | Post-design evidence |
| --- | --- | --- |
| Operational proof over intent | Pass | Observer does not change mutation submission, polling, confirmation or success reporting; cancellation tests assert the same requests and final outcome. |
| CLI-first, script-safe | Pass | Existing debug opt-in and configured logger; stdout and exit outcomes preserved; configured/inherited stderr and quiet combinations have execution tests. |
| Tests and validation mandatory | Pass | Transport, auth and command tests followed by `make test` are required implementation gates; terminal tests prove stream routing. |
| Documentation matches behavior | Pass | Root help/examples and README updates, then `make docs-content`; generated docs are not edited manually. |
| Small, compatible, repository-native | Pass | Extend shared httpc service and bootstrap, reuse activity writer and testx; no new dependency or facade mechanics. |
| Repository layering and cohesion | Pass | Observation/redaction lives in httpc; CLI owns flag/bootstrap only; no version adapter or generated-client changes. |

There are no exceptions or unresolved design gates. These are design assessments, not claims that implementation tests have already passed.

## Project Structure

### Documentation (this feature)

```text
specs/305-api-request-diagnostics/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── tasks.md
├── quickstart.md
├── checklists/requirements.md
└── contracts/api-diagnostics.md
```

`tasks.md` defines the 32 implementation and validation tasks for this design.

### Source Code (repository root)

```text
cmd/
├── root.go                         # existing debug help and effective stderr selection
├── root_services.go                # explicit invocation diagnostic options
├── root_api_diagnostics_test.go    # new bootstrap + command execution tests
├── root_test.go                    # inherited flag/help/docs expectations
└── cmd_confirmation_terminal_test.go # extend real-terminal stream coverage
internal/services/httpc/
├── service.go                      # collector option and shared wrapping helper
├── round_trippers.go               # existing Log/Auth chain integration only
├── diagnostics.go                  # new collector and RoundTripper
├── diagnostics_body.go             # new transparent body observation
├── diagnostics_record.go           # new data snapshot and line formatting
├── diagnostics_redaction.go        # new centralized sanitization
└── diagnostics*_test.go            # lifecycle, trace, security, concurrency tests
internal/services/auth/oauth2/
├── service.go                      # attach collector to separate token client
└── diagnostics_test.go             # token traffic, recursion and secret tests
README.md                           # operator examples and semantics
```

**Structure Decision**: Keep backend observation in the existing httpc service package. New focused files separate body lifecycle, safe record formatting and sanitization. Root wiring is shared bootstrap, so it belongs in existing root files; individual command runners, facade APIs, version adapters and generated clients need no diagnostic logic. Reuse `toolx/logging/activity.go` and `testx` without broad refactoring. Generated CLI pages are regenerated during implementation.

## Phase 0 — Research Results

[research.md](research.md) resolves transport placement, OAuth coverage, stderr ownership, lifecycle, timing, size, security and failure-policy questions. Timing uses monotonic timestamps around delegate RoundTrip and observed body termination for total/headers/body; composed httptrace callbacks supply DNS/TCP/TLS and connection reuse. The exact compact grammar, failure-phase evidence and unavailable-field rules are fixed in the contract. All design questions are resolved. Key repository findings are the separate OAuth client and retry responses closed without draining.

## Phase 1 — Design

1. **Bootstrap**: Reuse resolved logger configuration and invocation logger. Resolve executing `cmd.ErrOrStderr()` before activity wrapping and create the normal configured logger with that writer. Pass the logger, profile/tenant and redaction inputs before authenticator initialization. No new flag or logging configuration. Install no diagnostic observer when the logger filters DEBUG.
2. **Service construction**: Add a `WithDiagnostics` option and helper in httpc that installs the observer under the current LogTransport only when enabled. One collector owns the sequence and retains the existing invocation logger. Both API and OAuth attachment use one helper that places diagnostics beneath LogTransport, or directly around a bare transport, while preserving an already attached observer. OAuth shares the collector without copying auth/retry behavior. Diagnostics alone owns exchange logs; LogTransport retains only activity and explicitly configured request dumps. Remove the legacy calling start line entirely. Do not mutate `http.DefaultTransport`, unrelated clients or global output state.
3. **Observation**: Clone the request to attach a composed trace and counting body while preserving headers, context, GetBody, timeout and transport selection. Delegate reads and closes without buffering or added calls. Synchronize callback state, freeze terminal snapshots once, and serialize redacted records after releasing state locks. Preserve existing trace callbacks and optional body capabilities where applicable; test unusual Read/Close returns and replay paths.
4. **Terminal events**: Emit on transport error, known no-body response, EOF, non-EOF read error, or early close. A closed retry response is incomplete unless completion was observed. Unknown/no terminal event means no fabricated completion. Late trace events do not mutate emitted records. Request completion and response completion are separate evidence.
5. **Safety and output**: Follow [the contract](contracts/api-diagnostics.md). Never call arbitrary error formatting or request dumps. Known secrets remain private redaction inputs; only safe values enter records. Emit the complete safe ASCII message through `log.Debug`, after releasing observation locks. Required fields belong in the message because PlainHandler ignores structured attributes. Reuse existing timestamp/source/plain/plain-time/text/JSON framing and synchronization; do not add a logger or redesign handlers. Prompts stay plain text on configured stderr. Writer failure does not change request/body results or spill to stdout.
6. **Compatibility**: Diagnostics do not enable body dumps, change environment/config precedence or change existing result models. Existing logging settings govern diagnostics, including quiet suppression and format selection; the redaction guarantee applies to the diagnostic message. Keep authentication failure handling and mutation verification intact.

## Validation Strategy

- **httpc unit/integration**: Controlled trace callbacks and local servers for final headers versus delayed body, informational headers that must not end the headers interval, DNS/connect/TLS, reused connections, overlapping connect samples, existing trace composition, bodyless/empty/partial/error bodies, exact delegated Read/Close results, request replay and GetBody, decompression, timeout/cancellation, redirects, and failure before status. Assert one record and no extra reads/closes/requests.
- **Retry regression**: Existing GET/HEAD retry policy and higher-level mutation attempts remain unchanged. Assert a record for each boundary invocation and incomplete evidence for discarded undrained responses. Do not promise visibility into transport-internal retransmissions.
- **Logger compatibility**: Cover plain, plain-time, text and JSON logging, timestamp/source options, default, verbose-only, debug-only, debug plus quiet and restrictive configured levels. Required fields survive in message text across handlers; JSON logs remain separate from command JSON stdout.
- **Security**: Table-driven URL/userinfo/query/header/error cases, repeated/encoded query keys, credential reflection, signed URL families, control characters, payload-bearing values and malformed metadata. Fuzz sanitizer and formatter. Verify safe operational identifiers survive and seeded secrets/bodies never appear. Include a cookie-login response that reflects its Set-Cookie value in X-Request-ID and require the reflected value to be omitted or redacted.
- **Concurrency**: Parallel requests with reordered completion, simultaneous read/close/trace callbacks, two independent collectors and writers, and repeated root invocations after resets. Race detector must pass; records must remain intact. Do not execute the global Cobra singleton concurrently in-process; use separate subprocess invocations when testing CLI concurrency.
- **Command paths**: Run real root bootstrap and `get process-definition` plus `cancel process-instance` against deterministic local fixtures. Compare enabled/disabled stdout, exit code, HTTP requests and confirmation behavior across normal/JSON/keys-only/quiet and supported combinations of automation/auto-confirm/no-wait. JSON must decode as one envelope then EOF; keys-only must be exact lines or zero bytes for empty results. Include configured leaf stderr, inherited root stderr, read pagination, empty scope, aborts and failures.
- **Auth/version coverage**: OAuth token fetch/cache use, cookie initialization, auth-none, and representative requests from each supported version/service family all reach the collector. Tokens use the same invocation sequence with no recursive authorization or added token retries.
- **Terminal behavior**: Extend the existing real-terminal runner scenarios to confirm stderr routing, prompts and activity coexist, stdout stays clean, and empty scopes remain prompt-free. Capture streams separately.
- **Delivery gates**: Targeted `go test` by changed package, gofmt on touched Go files, README/root metadata update, `make docs-content`, then `make test` before commit/merge. Inspect generated diffs for the inherited flag. See [quickstart.md](quickstart.md) for runnable commands.

## Complexity Tracking

No constitution violations require an exception. The response-body wrapper and separate OAuth attachment are necessary to observe the requested evidence without changing payload consumption or authentication. No new abstraction layer, background queue or dependency is introduced.
