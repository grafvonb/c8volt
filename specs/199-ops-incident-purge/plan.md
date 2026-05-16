# Implementation Plan: Ops Purge Process Instances With Incidents

**Branch**: `199-ops-incident-purge` | **Date**: 2026-05-16 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/199-ops-incident-purge/spec.md`
**Issue**: [#199](https://github.com/grafvonb/c8volt/issues/199)
**Mandatory Implementation Context**: `specs/ralph-implementation-rules.md` MUST be passed to task generation and every Ralph run as `--implementation-context specs/ralph-implementation-rules.md`.

## Summary

Add `c8volt ops purge process-instances-with-incidents` as a destructive, auditable operational cleanup workflow with alias `pi-with-incidents`. The command discovers candidate incidents through existing incident search semantics, freezes unique candidate process-instance keys, reuses the same deterministic process-instance delete plan as `delete pi`, then either reports a dry-run plan or executes confirmed deletion through the established process-instance deletion behavior. The implementation should extend the #186/#187 ops workflow/report foundation rather than shelling out to `get incident | delete pi`.

## Technical Context

**Language/Version**: Go, repository module `github.com/grafvonb/c8volt`
**Primary Dependencies**: Cobra CLI, existing c8volt public facade packages, `c8volt/incident`, `c8volt/ops`, `c8volt/process`, internal incident and ops services, internal process-instance delete planning/deletion services, generated Camunda clients only through service adapters
**Storage**: N/A for product state; optional Markdown or JSON audit report file output requested by the user
**Testing**: `go test` with command, facade, service, report, contract, and subprocess exit-code tests; final validation through targeted tests and `make test` when implementation is complete
**Target Platform**: Cross-platform CLI, with current local planning on macOS
**Project Type**: Go CLI
**Performance Goals**: Use existing incident paging, batch-size, limit, worker, fail-fast, and no-worker-limit controls; do not introduce unbounded discovery or duplicate delete submissions
**Constraints**: Keep stdout deterministic for `--automation --json`; preserve existing `get incident` and `delete pi` behavior; freeze candidate process-instance keys before delete planning; use candidate terminology in human-facing output
**Scale/Scope**: One predefined ops purge workflow with incident filters, dry-run, confirmed deletion, automation, JSON/human output, Markdown/JSON audit reporting, docs, and command contract metadata

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The workflow must discover candidates, build the existing delete plan before mutation, preserve wait/no-wait semantics, and report planned, deleted, partially failed, blocked, or failed outcomes.
- **CLI-First, Script-Safe Interfaces**: PASS. Behavior is exposed through Cobra commands, inherited root flags, shared command contracts, stable JSON/report models, automation handling, and documented exit-code classes.
- **Tests and Validation Are Mandatory**: PASS. Tasks must include command registration, incident filter validation, discovery, delete planning, destructive execution, automation JSON, reports, error classification, docs, and regression tests.
- **Documentation Matches User Behavior**: PASS. User-facing command metadata requires generated CLI docs via `make docs-content`; README-facing examples should be checked and updated when needed.
- **Small, Compatible, Repository-Native Changes**: PASS. Existing `cmd/ops_purge.go`, #186/#187 ops workflow/report code, incident facade/service search behavior, process facade, and process-instance delete planning/deletion services are the extension points.

No constitution violations are planned.

## Project Structure

### Documentation (this feature)

```text
specs/199-ops-incident-purge/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── progress.md
├── contracts/
│   └── incident-purge-cli.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── ops_purge.go
├── ops_purge_processinstances_with_incidents.go
├── ops_purge_processinstances_with_incidents_test.go
├── cmd_views_ops_purge_processinstances_with_incidents.go
├── cmd_views_ops_notices.go
├── ops_contract.go
├── ops_contract_test.go
├── command_contract.go
├── command_contract_test.go
├── get_incident.go
└── generated-doc-triggering command metadata

c8volt/
├── incident/
│   └── existing incident search facade models and conversions
├── ops/
│   ├── api.go
│   ├── client.go
│   ├── convert.go
│   ├── model.go
│   └── client_test.go
└── process/
    └── existing delete planning/deletion facade primitives

internal/services/
├── incident/
│   └── existing version-neutral incident search service and version adapters
├── ops/
│   ├── api.go
│   ├── incident_purge.go
│   └── incident_purge_test.go
└── processinstance/
    ├── api.go
    └── existing delete planning/deletion support
```

**Structure Decision**: Extend the existing ops purge command and ops facade/service layout. Incident discovery should reuse the incident search path; process-instance family scope, non-final safety, force cancel-before-delete, deletion, waiting, and worker behavior should remain owned by the existing process-instance delete planning/deletion source of truth.

## Complexity Tracking

No constitution violations or extra architectural complexity are currently justified. If implementation finds that #186/#187 shared ops helpers already cover report and workflow orchestration fully, #199-specific code should stay limited to incident selection request/result models, command wiring, discovery-to-delete orchestration, and rendering.

## Phase 0: Research

See [research.md](research.md).

## Phase 1: Design

See [data-model.md](data-model.md), [quickstart.md](quickstart.md), and [contracts/incident-purge-cli.md](contracts/incident-purge-cli.md).

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The design freezes candidate process-instance keys, validates an existing delete plan, reuses confirmation/deletion/wait behavior, and reports every final outcome.
- **CLI-First, Script-Safe Interfaces**: PASS. Human, JSON, automation, dry-run, report-file, alias, and exit-code behavior are explicit.
- **Tests and Validation Are Mandatory**: PASS. The task plan must include command, facade/service, report, automation, exit-code, docs, and regression tests before work is complete.
- **Documentation Matches User Behavior**: PASS. Generated docs refresh is required after command metadata changes.
- **Small, Compatible, Repository-Native Changes**: PASS. All planned files and APIs stay inside existing command, facade, ops service, incident service, and process-instance service boundaries.

## Implementation Notes For Ralph

- Launch Ralph only with `--implementation-context specs/ralph-implementation-rules.md`.
- Each Ralph iteration must read `specs/ralph-implementation-rules.md`, this `plan.md`, `spec.md`, `tasks.md`, and `progress.md` when present.
- Every commit subject for this feature must follow Conventional Commits and end with `#199`.
