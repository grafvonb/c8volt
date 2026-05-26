# Implementation Plan: Ops Paged Discovery Scope

**Branch**: `228-ops-paged-discovery` | **Date**: 2026-05-24 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/228-ops-paged-discovery/spec.md`

**Implementation Context**: Planning, task generation, and every Ralph implementation iteration must read and apply `specs/ralph-implementation-rules.md`. Ralph launch instructions must include `--implementation-context specs/ralph-implementation-rules.md`.

## Summary

Affected ops workflows currently freeze a single bounded discovery result before dry-run, confirmation, or mutation. The fix is to move complete-by-default paged discovery into the service layer for incident, process-instance, and process-definition candidate discovery, record whether each frozen scope is complete or user-limited, and render that status consistently in human, JSON, Markdown report, automation, help, and documentation paths. Existing retention and orphan discovery loops are the closest repository pattern.

## Technical Context

**Language/Version**: Go, following the repository's current module and supported Camunda 8.7, 8.8, and 8.9 boundaries.

**Primary Dependencies**: Cobra command layer, c8volt public facade packages, internal service interfaces, generated Camunda clients, `toolx`, `toolx/pool`, and existing test helpers.

**Storage**: N/A. c8volt remains a short-lived CLI client; Camunda is the runtime fact source.

**Testing**: Go tests near `internal/services/ops`, affected resource service packages, command tests under `cmd`, command contract tests, and docs generation checks where command help changes.

**Target Platform**: Cross-platform CLI binary used locally, in shells, and in automation.

**Project Type**: CLI application with service/facade layering.

**Performance Goals**: Discovery must process all matching pages without arbitrary first-page truncation and must stop promptly when an explicit `--limit` is reached.

**Constraints**: Preserve mutation safety, automation behavior, existing command taxonomy, generated-client isolation, deterministic output order, and explicit Camunda version compatibility.

**Scale/Scope**: Multi-page operational populations for process instances, incidents, and process definitions. `--batch-size` remains page-size tuning; `--limit` is the only explicit total-scope cap.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The plan preserves dry-run, confirmation, frozen scope reuse, mutation confirmation, and report semantics.
- **CLI-First, Script-Safe Interfaces**: PASS. Changes flow through existing commands, flags, output modes, JSON contracts, automation support, and help text.
- **Tests and Validation Are Mandatory**: PASS. Tasks must add multi-page service and command coverage plus frozen-scope reuse tests before validation.
- **Documentation Matches User Behavior**: PASS. Help text, README/docs examples, and generated CLI docs are in scope where command text changes.
- **Small, Compatible, Repository-Native Changes**: PASS. The design reuses existing service-owned discovery loops and avoids parallel command or resource structures.

## Project Structure

### Documentation (this feature)

```text
specs/228-ops-paged-discovery/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── ops-paged-discovery.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
internal/domain/
├── ops_incident_purge.go
├── ops_repair.go
└── ops_all_process_definitions_purge.go

internal/services/incident/
├── api.go
├── lookup.go
└── v87|v88|v89/

internal/services/processdefinition/
├── api.go
└── v87|v88|v89/

internal/services/ops/
├── incident_purge.go
├── repair.go
├── all_process_definitions_purge.go
└── *_test.go

c8volt/ops/
├── model.go
├── convert.go
└── client_test.go

cmd/
├── ops_purge_processinstances_with_incidents.go
├── ops_repair_incident.go
├── ops_repair_processinstance.go
├── ops_purge_all_processdefinitions.go
├── cmd_views_ops_*.go
├── ops_*_test.go
└── command_contract_test.go

docs/ops/
README.md
docs/cli/
```

**Structure Decision**: Keep discovery ownership in `internal/services` and domain/facade data shapes, with commands only passing flags and rendering returned scope/completeness. Process-instance paging can reuse existing page APIs. Incident discovery can reuse `SearchIncidentsPage`. Process-definition discovery needs a page-capable service boundary because existing `SearchProcessDefinitions` only returns one bounded result set.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design

See [data-model.md](./data-model.md), [contracts/ops-paged-discovery.md](./contracts/ops-paged-discovery.md), and [quickstart.md](./quickstart.md).

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. Frozen scope is a first-class result shape and confirmed mutation consumes only frozen keys.
- **CLI-First, Script-Safe Interfaces**: PASS. Output contract changes are additive fields/statuses on existing ops report models.
- **Tests and Validation Are Mandatory**: PASS. The task plan must include targeted and broad Go validation plus generated docs validation.
- **Documentation Matches User Behavior**: PASS. Documentation and generated docs are explicit deliverables.
- **Small, Compatible, Repository-Native Changes**: PASS. No new command hierarchy or persistence layer is introduced.

## Complexity Tracking

No constitution violations requiring justification.
