# Implementation Plan: API Latency Diagnostics

**Branch**: `289-api-latency-diagnostics` | **Date**: 2026-09-01 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/289-api-latency-diagnostics/spec.md`
**Issue**: [#289](https://github.com/grafvonb/c8volt/issues/289)
**Mandatory Implementation Context**: `specs/ralph-implementation-rules.md` MUST be passed to task generation and every Ralph run as `--implementation-context specs/ralph-implementation-rules.md`.

## Summary

Add `c8volt ops analyse api-latency` for strictly read-only latency evidence and `c8volt ops execute api-latency-test` for a confirmed, bounded active test. Both commands use the existing ops command/facade/service boundaries, closed-loop worker pool, progress conventions, command envelope, and shared Markdown/JSON report machinery. The active workflow reuses the existing version-matched `SimpleUserTask` fixture, records only exact keys returned by Camunda as cleanup authority, and preflights version capabilities before mutation. No new report framework, worker framework, fixture, configuration surface, or generated-client change is planned.

## Technical Context

**Language/Version**: Go 1.26 (`go 1.26`, toolchain `go1.26.2`)
**Primary Dependencies**: Cobra 1.10.2, existing `c8volt/ops` facade, `internal/services/ops`, version-neutral process-definition/process-instance/resource services, `toolx/pool`, shared ops progress/report helpers, standard library `crypto/rand`, `context`, and `time`
**Storage**: No durable product state; optional user-requested Markdown or JSON report files and ephemeral Camunda fixture resources owned by one active run
**Testing**: Go `testing` and Testify; focused domain/service/facade/command tests, existing CLI contract/inventory and integration-volume suites, `make docs-content`, `git diff --check`, then `make test`
**Target Platform**: Cross-platform CLI against supported Camunda 8.7-8.10 profiles; current planning environment is macOS
**Project Type**: Go CLI with public facade and versioned backend adapters
**Performance Goals**: Never exceed the requested logical primary-sample budget, derived-request bound, or worker ceiling; render the final in-memory summary within 5 seconds after the last measurement or cleanup attempt, excluding report-destination latency
**Constraints**: Read-only mode must never mutate; inherited `--timeout` remains an HTTP per-request timeout; searches and keyed reads retain their existing retry differences; active execution is unavailable on 8.7 because exact ownership keys are not returned, requires `--no-cleanup` on 8.8, and supports cleanup by exact keys on 8.9/8.10; raw upstream bodies and secrets must not enter diagnostic models or reports
**Scale/Scope**: Two command leaves, one shared API-latency result model with active-only evidence, deterministic stages from 1 to `--workers`, default count 20/workers 4, five reportable measurement families, findings, owned-resource cleanup, docs, and extensions to existing test suites

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. Every conclusion is tied to measured stage evidence, safe response classes, topology evidence, or explicit unavailable/limited states. Findings remain hypotheses and include the next investigation.
- **CLI-First, Script-Safe Interfaces**: PASS. The two Cobra leaves reuse inherited flags, the standard command envelope, established mutation confirmation, automation behavior, progress suppression, and shared report handling.
- **Tests and Validation Are Mandatory**: PASS. The design requires deterministic statistics, concurrency/count safety, zero-mutation, version gates, cleanup-after-cancellation, secret-safety, report, command-contract, docs, and integration coverage.
- **Documentation Matches User Behavior**: PASS. Command source metadata remains authoritative; implementation regenerates `docs/cli` and updates the ops index, README examples, and two focused operator guides.
- **Small, Compatible, Repository-Native Changes**: PASS. The plan extends existing domain, ops service/facade, command, renderer, fixture, report, progress, and integration patterns without new packages or third-party dependencies.

No constitution violations are planned.

## Project Structure

### Documentation (this feature)

```text
specs/289-api-latency-diagnostics/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/
│   └── requirements.md
├── contracts/
│   └── api-latency-cli.md
└── tasks.md                       # Created later by $speckit-tasks
```

### Source Code (repository root)

```text
cmd/
├── ops_analyse_api_latency.go
├── ops_analyse_api_latency_test.go
├── ops_execute_api_latency.go
├── ops_execute_api_latency_test.go
├── cmd_views_ops_api_latency.go
├── ops_contract.go                # Reuse report/output contracts
├── ops_report.go                  # Reuse format/path/write behavior
├── ops_progress_mode.go           # Reuse activity/progress modes
└── ops_semantic_progress.go       # Reuse semantic progress reporting

c8volt/ops/
├── api.go
├── client.go
├── convert.go
├── model.go
└── client_test.go

internal/domain/
└── ops_api_latency.go

internal/services/ops/
├── api.go
├── api_latency.go
└── api_latency_test.go

embedded/processdefinitions/
├── C87_SimpleUserTask.bpmn        # Read-only repository context only
├── C88_SimpleUserTask.bpmn
├── C89_SimpleUserTask.bpmn
└── C810_SimpleUserTask.bpmn

integration/cli/
├── all_commands_test.go
├── volume_ops_analyse_test.go
├── volume_ops_execute_test.go
└── examples_test.go

docs/ops/
├── index.md
├── analyse-api-latency.md
└── execute-api-latency-test.md
```

**Structure Decision**: Keep Cobra construction, static validation, confirmation, activity selection, report-file handling, and final wording under `cmd/`. Extend the existing `c8volt/ops` facade with thin request/result conversion. Put stage planning, timing, classification, statistics, findings, visibility polling, ownership tracking, and cleanup orchestration in `internal/services/ops`. Continue to use the existing version-neutral services and capability interfaces for remote operations; do not call generated clients or version packages from the command or facade. Add a focused progress/mode file only if implementation crosses the repository's three-declaration cohesion threshold.

## Complexity Tracking

No constitution violations or parallel abstractions are justified. In particular, the feature adds no new report contract, no OpenAPI artifact, no custom worker pool, no BPMN fixture, and no new versioned ops adapters. Scoped signal-aware cancellation for the active leaf is the smallest addition needed to attempt cleanup after an operator interrupt; cleanup itself remains in the ops service and uses an independent bounded context.

## Phase 0: Research

See [research.md](research.md).

## Phase 1: Design

See [data-model.md](data-model.md), [quickstart.md](quickstart.md), and [contracts/api-latency-cli.md](contracts/api-latency-cli.md).

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The data model separates observations, classifications, findings, limitations, and final execution status; abnormal evidence does not become an execution failure or a definitive root-cause claim.
- **CLI-First, Script-Safe Interfaces**: PASS. The contract defines compact human output, one-document JSON, rejection of irrelevant keys-only mode, non-interactive JSON mutation guardrails, exact shared report behavior, and established error-envelope behavior.
- **Tests and Validation Are Mandatory**: PASS. The quickstart includes focused tests, command-contract and docs validation, the full race suite, and disposable-cluster checks without fragile real-time thresholds.
- **Documentation Matches User Behavior**: PASS. The plan updates source help first and regenerates CLI docs; operator guides document safety, version limits, report formats, and the meaning of bounded evidence.
- **Small, Compatible, Repository-Native Changes**: PASS. One shared feature model and one combined CLI contract cover both modes; active-only fields are optional rather than duplicated into a second framework.

## Implementation Notes For Ralph

- Launch Ralph only with `--implementation-context specs/ralph-implementation-rules.md`.
- Each Ralph iteration must read `specs/ralph-implementation-rules.md`, this `plan.md`, `spec.md`, `tasks.md`, and `progress.md` when present.
- Preserve the established `--workers/-w`, `--count/-n`, output, confirmation, automation, tenant, timeout, progress, and report contracts.
- Every commit subject for this feature must follow Conventional Commits and end with `#289`.
