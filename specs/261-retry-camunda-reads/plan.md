# Implementation Plan: Retry Transient Camunda Read Failures

**Branch**: `261-retry-camunda-reads` | **Date**: 2026-07-28 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/261-retry-camunda-reads/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

Add a narrow, central retry layer for transient Camunda GET/HEAD read failures so read-heavy commands can continue after temporary platform errors such as JWK lookup timeouts. The implementation should live in the shared HTTP client path used by c8volt services, reuse the existing retry/backoff style, preserve final response bodies for existing error mapping, and explicitly avoid retrying search requests or unsafe mutations in this issue.

## Technical Context

**Language/Version**: Go, using the repository's current Go module and generated Camunda 8.7/8.8/8.9 clients

**Primary Dependencies**: Standard `net/http` client/transport stack, existing `internal/services/httpc` service and transports, existing `internal/services` retry/backoff patterns, Cobra CLI bootstrap, existing `toolx/logging` logger/activity paths

**Storage**: N/A; feature is runtime HTTP resilience and operator logging only

**Testing**: Go tests with targeted `go test ./internal/services/httpc -count=1`, targeted command/service regression where useful, then `make test` for full race-enabled validation before completion

**Target Platform**: Cross-platform c8volt CLI in interactive terminals and automation environments

**Project Type**: Go CLI application with public facade packages, version-neutral internal services, and version-specific Camunda adapters sharing one HTTP client

**Performance Goals**: Successful transient recovery after a single GET/HEAD platform failure should add only bounded backoff delay; persistent failures must stop at a fixed retry budget; non-retryable requests must not incur retry overhead beyond a cheap method/status check

**Constraints**: Retry only GET/HEAD in this issue; do not retry search or other non-GET/HEAD read-like requests; do not replay mutations; preserve response bodies for final error mapping; stop promptly on context cancellation; keep logs compact and compatible with JSON, keys-only, quiet, and automation modes

**Scale/Scope**: Applies generically to commands that use the shared c8volt HTTP client for Camunda GET/HEAD reads, with the customer proof path being process-instance parent lookup during orphan detection

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The plan preserves final command outcomes and requires exhausted retries to surface the final diagnostic error rather than pretending success.
- **CLI-First, Script-Safe Interfaces**: PASS. The retry behavior is shared beneath commands and protects stdout contracts; retry information remains compact operator logging, not machine-output payload churn.
- **Tests and Validation Are Mandatory**: PASS. The plan requires focused HTTP transport tests, non-retry safety tests, cancellation/body-preservation tests, and full repository validation.
- **Documentation Matches User Behavior**: PASS. No command flags or help examples are added, so generated CLI docs should not change; a README/troubleshooting note is required if normal operator logs gain retry wording.
- **Small, Compatible, Repository-Native Changes**: PASS. The design reuses `internal/services/httpc`, existing logger/activity paths, and the existing Camunda mutation retry policy style instead of adding per-command retry branches or partial-result semantics.

## Project Structure

### Documentation (this feature)

```text
specs/261-retry-camunda-reads/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── http-read-retry-contract.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
└── root.go                         # installs shared remote command services

c8volt/
└── client.go                       # passes the shared HTTP client into facades/services

internal/services/
├── retry.go                        # existing mutation retry behavior to mirror, not replace
├── retry_test.go
└── httpc/
    ├── service.go                  # shared HTTP client service and transport installation
    ├── round_trippers.go           # existing logging/auth transports
    ├── round_trippers_test.go
    ├── httpmap.go                  # final HTTP response to domain error mapping
    └── http_read_retry_test.go     # planned focused tests for this feature

internal/services/processinstance/
├── v87/
├── v88/
└── v89/                            # proof path uses GET parent lookup through generated clients

README.md                           # update only if operator retry logging is documented
```

**Structure Decision**: Use the existing single-project Go CLI layout. The retry behavior belongs in `internal/services/httpc` because that is the central shared HTTP client path used by all generated Camunda service clients. Command packages should not grow command-specific retry branches for this issue.

## Complexity Tracking

No constitution violations are planned. The only new production concept is a small retrying HTTP transport/policy in the existing `httpc` package; it is justified because the requirement must apply generically across command families while staying below command-specific business logic.

## Phase 0: Research

Research is captured in [research.md](research.md). All planning unknowns were resolved during this phase.

## Phase 1: Design

Design artifacts:

- [data-model.md](data-model.md) defines the retry policy, retryable request, retryable failure, retry attempt, and final outcome concepts.
- [contracts/http-read-retry-contract.md](contracts/http-read-retry-contract.md) defines the operator-visible and machine-output contract for GET/HEAD retry behavior.
- [quickstart.md](quickstart.md) defines focused validation scenarios and expected outcomes.

## Constitution Check After Design

- **Operational Proof Over Intent**: PASS. The design reports success only when the final request/command succeeds and keeps exhausted retry diagnostics intact.
- **CLI-First, Script-Safe Interfaces**: PASS. The contract keeps stdout output stable and limits retry information to existing logging/error channels.
- **Tests and Validation Are Mandatory**: PASS. Quickstart covers retry success, non-retry outcomes, mutation/search safety, final body preservation, cancellation, and full validation.
- **Documentation Matches User Behavior**: PASS. Documentation impact is explicitly limited to README/troubleshooting wording if the final implementation adds visible retry logs; generated CLI docs remain unchanged unless command metadata changes unexpectedly.
- **Small, Compatible, Repository-Native Changes**: PASS. The design reuses existing `httpc` transport chaining and mutation retry style, and explicitly rejects partial-results or broad request replay.
