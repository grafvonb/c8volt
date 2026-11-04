# Implementation Plan: Optional Request Pacing and DEBUG Rate Reporting

**Branch**: `353-request-rate-reporting` | **Date**: 2026-10-24 | **Spec**: [spec.md](spec.md)

**Input**: `specs/353-request-rate-reporting/spec.md`, [issue #353](https://github.com/grafvonb/c8volt/issues/353)

## Summary

Add one shared HTTP request limiter controlled by `--request-rate` / `http.request_rate` (default `0`, disabled). Independently report actual dispatch starts at DEBUG using `http.request_rate_report_interval` (default `5s`, `0s` disables reporting). Integrate at the existing shared client so cancel, delete, and purge retain their current workflows and worker settings.

Use a burst-one limiter for positive rates and bypass it completely at zero. Reporting uses an atomic counter and a separate timer. Keep a small local-wait error observation across HTTP timeout wrapping so requests that never leave the limiter do not become retryable backend failures. Decisions and alternatives are in [research.md](research.md).

## Technical Context

**Language/Version**: Go 1.26, toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2, Viper 1.21.0, `net/http`, `slog`; add `golang.org/x/time v0.16.0` for its rate limiter.

**Storage**: Existing configuration files/profiles/environment; runtime state only, no new persisted data.

**Testing**: Go tests, testify, isolated round-tripper/HTTP fakes, CLI subprocess and terminal fixtures, race detector, focused overhead benchmarks.

**Target Platform**: Existing Go CLI platforms; no platform-specific runtime mechanism.

**Project Type**: CLI with public facades and internal Camunda service adapters.

**Performance Goals**: One allowance across concurrent requests with burst 1; no pacing wait at rate zero; no logger call on the counting path; at most one report timer per invocation; no added remote calls. Compare zero-rate overhead against pass-through rather than invent a throughput guarantee.

**Constraints**: Preserve current deadlines, retry policy for dispatched attempts, worker concurrency, resource selection, mutation order, dry runs, prompts, no-wait, and output/exit contracts. Configuration errors fail before remote work. No generated-client edits or new public facade API.

**Scale/Scope**: Thousands of targets using one invocation-owned client across existing Camunda adapters. Independent processes are not coordinated. No batch/limit flags, adaptive throttling, checkpointing, nested-pool changes, unrelated repairs, or live-environment validation.

## Constitution Check

Initial review before research: all five gates passed for the proposed narrow shared-client design. Phase 1 review below also passes; no exceptions are requested.

| Gate | Design evidence and validation |
|------|--------------------------------|
| I. Operational proof | Existing confirmation reads and no-wait semantics remain intact. A local unsent failure cannot claim acceptance or success. Fake command tests verify mutation outcomes and ordering. |
| II. Script-safe CLI | Existing flag/config resolution and configured logger are reused. No result schema changes; check separate streams, one JSON envelope, empty results, keys-only and quiet behavior. |
| III. Proportionate validation | This planning change receives document checks only. Implementation starts with focused fake tests; the shared concurrent transport and new dependency justify one full race suite afterward. |
| IV. Documentation | Update root flag metadata, configuration guidance and README examples; regenerate CLI docs with `make docs-content` during implementation. |
| V. Small native change | One HTTP wrapper/reporter and narrow root/config wiring. Reuse current retry and diagnostic paths. The dependency avoids a custom limiter; the local-wait observation addresses verified standard-library error wrapping. |

## Project Structure

### Documentation (this feature)

```text
specs/353-request-rate-reporting/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/requirements.md
└── contracts/request-rate.md
```

`tasks.md` belongs to the next `$speckit-tasks` phase and is not created here.

### Source Code (repository root)

```text
config/http.go, config/config.go           # fields, defaults, normalization, validation
cmd/root.go, cmd/root_config.go            # root flag and existing resolver binding
cmd/root_services.go                      # shared service setup/cleanup registration
cmd/root_request_rate.go                   # focused singleton-root cleanup lifecycle
internal/services/httpc/service.go         # shared wrapper and transport traversal
internal/services/httpc/request_rate*.go   # limiter, counter, reporter, local-wait evidence
internal/services/httpc/read_retry.go      # exclude locally unsent failures
internal/services/retry.go                 # per-attempt evidence before existing retry policy
config/*_test.go, cmd/*_test.go             # config and isolated command acceptance
internal/services/httpc/*_test.go          # transport, timing, accounting, lifecycle
internal/services/retry_test.go            # local wait versus backend retry regression
README.md, docs/cli/*                      # guidance and regenerated CLI reference
go.mod, go.sum                            # pinned limiter dependency
```

**Structure Decision:** Keep mechanics in `httpc`; no command-specific pacing and no changes to generated Camunda clients. Keep the small CLI cleanup lifecycle in a focused file rather than expanding root construction with several lifecycle declarations. Reuse existing aggregate error helpers if a bounded local-wait reason needs mapping; do not create a second renderer or alter schemas.

## Implementation Design

1. **Resolve and validate settings.** Add a finite nonnegative floating-point rate and duration-string interval. Reuse explicit-flag, environment, profile, and base precedence, including zero. An omitted interval defaults to `5s`; explicit empty/invalid/negative values fail. The interval is configuration-only.
2. **Admit and count requests once.** Put the shared wrapper below application read retries and above existing HTTP logging/diagnostics. Use context-aware admission only when rate is positive. Recheck cancellation, increment the optional atomic counter, then delegate. Concurrent in-flight requests remain possible; no lock is held across network I/O. Traffic using the shared client, including cookie bootstrap calls, crosses this boundary. Independent identity-provider clients share only the existing diagnostics.
3. **Preserve local failure evidence.** Return a typed local-wait cancellation/deadline error; early insufficient-deadline admission is also a deadline failure. Exclude it from read retries. When the invocation's context-carried HTTP service has positive pacing enabled, attach a fresh small observation to each mutation retry callback context and check it before classifying the callback error. At rate zero this helper is a no-op. This survives `http.Client.Timeout` replacing the original error type. Preserve errors through facade classification and existing bounded human error reporting; no raw transport chains at normal levels.
4. **Report off the request path.** Start a reporter only when the effective logger admits DEBUG and the interval is positive. Snapshot cumulative starts and actual elapsed monotonic time; report their differences through the configured logger. Include all three values in the message so plain formats remain useful. Do not include resource identifiers, URLs, bodies, or errors. Zero windows are valid; no final partial-window report is required.
5. **Own the lifetime.** Pass the invocation context from `root_services.go` through a small `httpc.WithInvocationContext(ctx)` option so the reporter observes cancellation. HTTP service close is idempotent and stops the timer promptly. Register one root Cobra finalizer, with only the current service cleanup handle, and clear it after use. Close on bootstrap failure too; avoid accumulating callbacks over repeated invocations. Existing process-exit paths need no rewrite. Direct service tests/callers own close; parallel execution of the globally stateful root remains outside scope.
6. **Document and prove compatibility.** Use the acceptance matrix in [quickstart.md](quickstart.md). Cover the representative cancel, instance-delete, definition-delete, and purge paths with fakes. Update metadata/README and regenerate docs together with implementation. No live Camunda test is required or permitted by this plan.

These are design responsibilities, not separate implementation tasks. Task generation should group implementation, fixtures, tests, formatting, and validation into cohesive compiling changes. Keep the limiter, its error handling, affected retry code, and test doubles together; do not split them into temporarily broken layers or duplicate work by user story.

## Complexity Tracking

No constitution violations. Two bounded additions are justified in research: the maintained limiter dependency replaces custom scheduling, and per-attempt local-wait evidence prevents HTTP timeout wrapping from causing unintended mutation retries. Neither introduces a new mutation workflow.
