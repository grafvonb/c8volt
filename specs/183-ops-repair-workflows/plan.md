# Implementation Plan: Ops Repair Workflows

**Branch**: `183-ops-repair-workflows` | **Date**: 2026-05-17 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/183-ops-repair-workflows/spec.md`
**Issue**: [#183](https://github.com/grafvonb/c8volt/issues/183)
**Mandatory Implementation Context**: `specs/ralph-implementation-rules.md` MUST be passed to task generation and every Ralph run as `--implementation-context specs/ralph-implementation-rules.md`.

## Summary

Add concrete `c8volt ops repair incident` and `c8volt ops repair process-instance` workflows under the existing `ops repair` grouping command from issue #197. The workflows discover explicit or filtered repair targets, freeze the initial target set, handle job-backed and non-job incidents in the same run, optionally update process-instance scoped variables, apply job retry/timeout changes only when a related job exists, resolve incidents, confirm clearance, and render deterministic output plus optional Markdown or JSON audit reports. Implementation must compose existing incident, process-instance, job, and ops service capabilities through the repository layering rather than calling generated Camunda clients from command or ops orchestration code.

## Technical Context

**Language/Version**: Go, repository module `github.com/grafvonb/c8volt`
**Primary Dependencies**: Cobra CLI, existing `c8volt/ops`, `c8volt/incident`, `c8volt/process`, and `c8volt/job` facade patterns; internal ops, incident, process-instance, and job services; generated Camunda clients only through versioned service adapters
**Storage**: N/A for product state; optional report files written through existing ops report helpers
**Testing**: `go test` with focused command, facade, internal ops service, incident/process-instance/job primitive tests; final validation with targeted suites and `make test` when implementation is complete
**Target Platform**: Cross-platform CLI for interactive operators and unattended automation
**Project Type**: Go CLI
**Performance Goals**: Use existing paging, batch-size, worker, fail-fast, and no-worker-limit controls; freeze discovered targets once and avoid unbounded incident chasing
**Constraints**: Preserve existing `get incident`, `get pi`, `update pi`, `update job`, `resolve incident`, and `resolve pi` behavior; keep stdout deterministic for JSON/automation; do not call generated Camunda clients from command, facade, or ops workflow code; initial variable repair scope is the process-instance key
**Scale/Scope**: Two repair targets with keyed mode, stdin keys, filtered discovery, dry-run, mixed job applicability, variable updates, confirmation, JSON/human output, automation support, and Markdown/JSON audit reports

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. Repair reports success only after resolution and confirmation unless dry-run or an existing explicit no-wait-style behavior is applied in later design.
- **CLI-First, Script-Safe Interfaces**: PASS. Behavior is exposed through target-specific Cobra commands, stable flags, existing output modes, deterministic JSON/automation behavior, and audit reports.
- **Tests and Validation Are Mandatory**: PASS. Tasks must include command, facade, service, report rendering, failure-mode, and docs validation before stories are marked complete.
- **Documentation Matches User Behavior**: PASS. Command metadata changes require generated CLI docs via `make docs-content`, and README-facing examples must stay aligned if updated.
- **Small, Compatible, Repository-Native Changes**: PASS. The design extends existing ops, incident, process-instance, job, report, flag, and view patterns without introducing a parallel orchestration stack.

No constitution violations are planned.

## Project Structure

### Documentation (this feature)

```text
specs/183-ops-repair-workflows/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── ops-repair-cli.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── ops_repair.go
├── ops_repair_incident.go
├── ops_repair_processinstance.go
├── ops_repair_incident_test.go
├── ops_repair_processinstance_test.go
├── cmd_views_ops_repair.go
├── ops_contract.go
├── ops_contract_test.go
├── command_contract_test.go
└── docs-triggering command metadata

c8volt/
├── client.go
└── ops/
    ├── api.go
    ├── client.go
    ├── convert.go
    ├── model.go
    └── client_test.go

internal/services/
├── ops/
│   ├── api.go
│   ├── repair.go
│   └── repair_test.go
├── incident/
│   ├── api.go
│   ├── lookup.go
│   ├── resolve.go
│   └── version package tests as needed
├── processinstance/
│   ├── api.go
│   ├── variables.go
│   └── search or discovery tests as needed
└── job/
    ├── api.go
    └── version package tests as needed

README.md
docs/cli/
docsgen/
```

**Structure Decision**: Extend the existing single-project Go CLI structure. Command files own flags, validation, confirmation, output dispatch, and command metadata. `c8volt/ops` owns public request/result models and thin facade conversion. `internal/services/ops` owns repair workflow orchestration, target freezing, step aggregation, report construction, and dependency coordination. Incident lookup/search/resolve/confirmation, process-instance search/variable updates, and job lookup/update behavior stay in their owning service packages.

## Complexity Tracking

No constitution violations are planned. The only deliberate complexity is the `internal/services/ops` workflow boundary, which is justified because repair coordinates multiple resource services, freezes targets, dedupes variable scopes, and produces a structured report. Resource-specific API behavior must remain in the owning service packages.

## Phase 0: Research

See [research.md](research.md).

## Phase 1: Design

See [data-model.md](data-model.md), [quickstart.md](quickstart.md), and [contracts/ops-repair-cli.md](contracts/ops-repair-cli.md).

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The design includes discovery, immutable repair plan, mutation submission, resolution confirmation, final context, and explicit dry-run planning.
- **CLI-First, Script-Safe Interfaces**: PASS. Human, JSON, automation, dry-run, keyed, search, and report behaviors are explicit.
- **Tests and Validation Are Mandatory**: PASS. Task generation must include focused behavior tests and relevant validation commands for each story.
- **Documentation Matches User Behavior**: PASS. Generated docs refresh is required after command metadata changes.
- **Small, Compatible, Repository-Native Changes**: PASS. All planned code paths are scoped to existing command, facade, service, and report boundaries.

## Implementation Notes For Ralph

- Launch Ralph only with `--implementation-context specs/ralph-implementation-rules.md`.
- Each Ralph iteration must read `specs/ralph-implementation-rules.md`, this `plan.md`, `spec.md`, `tasks.md`, and `progress.md` when present.
- Complete only the current Ralph work unit in each iteration.
- Every commit subject for this feature must follow Conventional Commits and end with `#183`.
