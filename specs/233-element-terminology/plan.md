# Implementation Plan: Element Terminology Standardization

**Branch**: `233-element-terminology` | **Date**: 2026-05-25 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/233-element-terminology/spec.md`

## Summary

Standardize public incident and process-context contracts on Camunda v2 element terminology. Replace public `flowNode*` and compact `fn`/`fni` naming with `elementId`, `elementInstanceKey`, `parentElementInstanceKey`, `e:`, and `ei:` across CLI flags, JSON models, command views, ops workflows, tests, README, and generated CLI docs. Generated legacy clients may still contain `FlowNode*` names, but those names must remain contained inside generated code or version-specific adapter mappings. Planning, tasks, and Ralph implementation MUST include `--implementation-context specs/ralph-implementation-rules.md`.

## Technical Context

**Language/Version**: Go, repository current module toolchain
**Primary Dependencies**: Cobra command tree, command contract helpers, existing incident/process/ops facades, `internal/domain` incident and process models, `internal/services/incident` and process-instance adapters, generated Camunda v8.7/v8.8/v8.9 clients, shared render/error helpers, docs generator
**Storage**: N/A
**Testing**: Targeted Go tests for `cmd`, `c8volt/incident`, `c8volt/process`, `c8volt/ops`, `c8volt/resource`, `internal/services/incident`, and process-instance consumers; generated docs refresh through `make docs-content`; broader validation with `make test` before commit readiness
**Target Platform**: Multi-platform CLI binary
**Project Type**: Go CLI
**Performance Goals**: Preserve current incident search, process-instance lookup, walk, repair, and purge behavior without adding extra remote discovery or pagination work
**Constraints**: Intentional breaking public contract cleanup; no transitional aliases; generated clients are not hand-edited; legacy generated names are allowed only at generated-client and versioned adapter boundaries; command output must follow `specs/ralph-implementation-rules.md` operator UX rules; README and generated CLI docs must stay synchronized with changed behavior
**Scale/Scope**: Incident filtering flags, incident JSON/human rendering, process-instance parent context fields, ops repair/purge incident filters and summaries, public facade/domain models, command contract metadata, docs, and tests

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The change preserves existing read/mutation workflows and adds regression checks proving old public names are gone and canonical names work.
- **CLI-First, Script-Safe Interfaces**: PASS. Public flags, JSON fields, keys-only behavior, human output, command metadata, and generated docs are in scope.
- **Tests and Validation Are Mandatory**: PASS. The plan requires command, facade, service, renderer, contract, docs, and regression tests before implementation is complete.
- **Documentation Matches User Behavior**: PASS. README and generated CLI docs are required outputs for changed command surfaces.
- **Small, Compatible, Repository-Native Changes**: PASS. The work stays inside existing incident, process, ops, facade, service, renderer, and docs-generation boundaries; compatibility is intentionally broken only for the named public aliases.

## Project Structure

### Documentation (this feature)

```text
specs/233-element-terminology/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── progress.md
├── contracts/
│   └── cli-element-terminology.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_incident.go
├── get_incident_test.go
├── get_processinstance_test.go
├── walk_processinstance_test.go
├── ops_repair_incident*.go
├── ops_purge_processinstances_with_incidents*.go
├── cmd_views_processinstance_incidents.go
├── cmd_views_*.go
└── command_contract_test.go

c8volt/
├── incident/
├── process/
├── ops/
└── resource/

internal/
├── domain/
├── services/incident/
└── services/processinstance/

README.md
docs/cli/
docs/ops/
docs/index.md
```

**Structure Decision**: Reuse the existing command/facade/domain/service/generated-client layering. `cmd/get_incident.go` and ops command files own public flag grammar and validation. `cmd/cmd_views_processinstance_incidents.go` and nearby view files own human rendering. Public facade models under `c8volt/incident`, `c8volt/process`, `c8volt/ops`, and `c8volt/resource` own JSON-facing field names. Internal domain and service packages own canonical semantic fields and versioned mappings. Generated clients remain untouched.

## Architecture Grounding

- Architecture extension status: installed.
- Architecture memory status: `.specify/memory/architecture.md` and all five 4+1 view files are present.
- Decision: reuse existing architecture memory without refresh. Issue #233 applies already documented boundaries: command contract stability, facade/domain/service layering, generated-client isolation, docs generation, and script-safe output. It changes public terminology, not architecture ownership or runtime topology.

## Ralph Implementation Context

- Every implementation iteration MUST receive `--implementation-context specs/ralph-implementation-rules.md`.
- Do not launch Ralph unless the launcher instructions include that implementation context.
- Each Ralph work unit must complete only one story or validation slice and must not stage or commit until validation passes.
- Commit subjects must use Conventional Commits and end with `#233`.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

- Data model: [data-model.md](./data-model.md)
- CLI contract: [contracts/cli-element-terminology.md](./contracts/cli-element-terminology.md)
- Quickstart and verification scenarios: [quickstart.md](./quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The contract requires tests proving canonical flags work, old flags fail locally, JSON fields are renamed, human output labels change, and docs are regenerated.
- **CLI-First, Script-Safe Interfaces**: PASS. The design covers help text, command metadata, JSON, human output, generated docs, README examples, and automation-safe flag behavior.
- **Tests and Validation Are Mandatory**: PASS. The design requires targeted package tests plus docs generation and broader validation before task completion.
- **Documentation Matches User Behavior**: PASS. Documentation updates are source-driven and generated outputs are refreshed.
- **Small, Compatible, Repository-Native Changes**: PASS. The design rejects adapter bypasses, command-layer generated-client access, and compatibility aliases.

## Complexity Tracking

No constitution violations or complexity exceptions are required.
