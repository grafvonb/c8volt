# Implementation Plan: Report Process Definition Incident Statistics

**Branch**: `042-pd-incident-stats` | **Date**: 2026-04-21 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/spec.md)
**Input**: Feature specification from `/specs/042-pd-incident-stats/spec.md`

## Summary

Fix `get pd --stat` so the `in:` segment reports the number of process instances that currently have at least one active incident on `v8.8` and `v8.9`, while `v8.7` omits the `in:` segment entirely because the repository’s current Operate surface cannot derive that value reliably. The design keeps the existing CLI entry point and versioned processdefinition service structure, sources supported-version incident-bearing instance counts from the newer generated Camunda endpoints rather than the older process-definition element stats call alone, and updates the one-line process-definition renderer so supported zero counts print `in:0` while unsupported versions omit the segment instead of showing `in:-`.

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, shared process facade under `c8volt/process`, domain models under `internal/domain`, generated Camunda clients under `internal/clients/camunda/v88/camunda` and `v89/camunda`, Operate client under `internal/clients/camunda/v87/operate`, versioned services under `internal/services/processdefinition/{v87,v88,v89}`  
**Storage**: File-based YAML config and environment variables only; no persistent datastore changes  
**Testing**: focused `go test ./cmd -count=1`, `go test ./c8volt/process -count=1`, `go test ./internal/services/processdefinition/... -count=1`, final repository validation with `make test`  
**Target Platform**: Cross-platform CLI for local and CI use against supported Camunda `8.7`, `8.8`, and `8.9` environments  
**Project Type**: CLI  
**Performance Goals**: Keep `get pd --stat` responsive for the existing list and lookup flows, avoid broad extra fetches beyond the bounded per-definition stats enrichment already used in supported versions, and preserve current non-stat output behavior and ordering  
**Constraints**: Keep the existing `get process-definition` CLI surface and versioned service layout, preserve current non-statistics fields and formatting, omit `in:` entirely when unsupported, update README and generated CLI docs because the user-visible output contract changes, avoid inventing parallel processdefinition models unless existing shared/domain types cannot express the supported-vs-unsupported rendering distinction cleanly, and finish with `make test`  
**Scale/Scope**: `cmd/get_processdefinition.go`, [`cmd/cmd_views_get.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_get.go), process-definition command coverage in [`cmd/get_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go), shared process-definition models and conversion in [`c8volt/process/model.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/model.go), [`c8volt/process/convert.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/convert.go), and [`internal/domain/processdefinition.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/processdefinition.go), plus versioned services and tests under [`internal/services/processdefinition/v87/`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87), [`internal/services/processdefinition/v88/`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88), [`internal/services/processdefinition/v89/`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v89), and user-facing docs in [`README.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md) plus generated CLI reference under [`docs/cli/`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. The feature improves truthfulness by reporting incident-bearing process-instance counts only where the active version can prove them and by omitting the `in:` segment entirely where the repository cannot verify the value.
- **CLI-First, Script-Safe Interfaces**: Pass. The command family, flags, and invocation path remain stable; the only behavior change is a more truthful `--stat` rendering contract.
- **Tests and Validation Are Mandatory**: Pass. The plan requires focused versioned service tests, command rendering coverage, facade/model coverage where needed, documentation regeneration, and final `make test`.
- **Documentation Matches User Behavior**: Pass. The output contract for `get process-definition --stat` changes for supported and unsupported versions, so README notes and generated CLI docs must be updated in the same implementation slice.
- **Small, Compatible, Repository-Native Changes**: Pass. The design stays inside the existing command renderer, shared process-definition model, and versioned processdefinition service seams instead of introducing a new reporting subsystem.

## Project Structure

### Documentation (this feature)

```text
specs/042-pd-incident-stats/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── process-definition-statistics.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_processdefinition.go
├── cmd_views_get.go
├── get_test.go
└── cmd_options_test.go

c8volt/process/
├── api.go
├── client.go
├── client_test.go
├── convert.go
└── model.go

internal/domain/
└── processdefinition.go

internal/services/processdefinition/
├── api.go
├── factory.go
├── factory_test.go
├── v87/
│   ├── contract.go
│   ├── convert.go
│   ├── service.go
│   └── service_test.go
├── v88/
│   ├── contract.go
│   ├── convert.go
│   ├── service.go
│   └── service_test.go
└── v89/
    ├── contract.go
    ├── convert.go
    ├── service.go
    └── service_test.go

internal/clients/camunda/
├── v87/operate/client.gen.go
├── v88/camunda/client.gen.go
└── v89/camunda/client.gen.go

README.md
docs/cli/
```

**Structure Decision**: Keep the feature inside the existing `get process-definition` command path, the shared process-definition models used by `c8volt/process`, and the current `v87`/`v88`/`v89` processdefinition services. Version-specific endpoint selection belongs inside those services, while the renderer in `cmd/cmd_views_get.go` remains the only place that decides whether `in:` is shown, hidden, or rendered as `0`.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/research.md).

- Confirm which generated `v8.8` and `v8.9` endpoints provide incident-bearing process-instance counts by definition and whether they already align with the clarified meaning of `in:`.
- Confirm that the current `GetProcessDefinitionStatistics` endpoint only exposes element-level incident totals, which can overcount when one process instance has multiple active incidents.
- Confirm that `v8.7` still has no reliable repository-native way to derive the clarified count, so the command should omit `in:` there instead of showing `in:-`.
- Confirm the lowest-risk shared model adjustment that lets supported versions distinguish `in:0` from “not supported” without breaking JSON output or existing non-stat views.
- Confirm the natural regression anchors for command rendering, versioned service sourcing, and documentation updates.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/quickstart.md)
- [contracts/process-definition-statistics.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/042-pd-incident-stats/contracts/process-definition-statistics.md)

- Keep `SearchProcessDefinitions`, `SearchProcessDefinitionsLatest`, and `GetProcessDefinition` as the only command-facing service entry points; supported-version enrichment stays internal to the versioned services when `WithStat` is enabled.
- Introduce the smallest shared process-definition statistics representation that can express both a supported incident-bearing process-instance count and the unsupported case where `in:` must be omitted entirely.
- On `v8.8` and `v8.9`, source `in:` from incident-bearing process-instance statistics grouped by definition rather than from the legacy element-stat `Incidents` total alone.
- On `v8.7`, preserve the current “stats unsupported for this incident dimension” behavior by omitting `in:` while leaving the other `--stat` fields unchanged.
- Update `oneLinePD(...)` so supported zero counts render as `in:0`, supported non-zero counts render numerically, and unsupported counts omit the segment entirely without disturbing the other stat segments.
- Treat README and generated CLI reference updates as required artifacts because the visible semantics of `get process-definition --stat` change.

### Authoritative Incident Statistics Boundary

| Version | Source of `ac/cp/cx` | Source of `in:` | Rendering rule |
|--------|-----------------------|-----------------|----------------|
| `v8.7` | Existing Operate process-definition search or lookup flow without stats support | Unsupported | Omit `in:` entirely |
| `v8.8` | Existing `GetProcessDefinitionStatistics` enrichment | Incident-bearing process-instance count by definition from newer Camunda endpoint(s) | Show `in:<count>`, including `in:0` |
| `v8.9` | Existing `GetProcessDefinitionStatistics` enrichment | Incident-bearing process-instance count by definition from newer Camunda endpoint(s) | Show `in:<count>`, including `in:0` |

This table is the authoritative planning rule for later tasks. Any implementation that still shows `in:-` on supported versions, keeps `in:` on `v8.7`, or counts multiple incidents on the same process instance more than once is incomplete.

### Rendering Boundary

| Condition | Expected output behavior |
|--------|---------------------------|
| `Statistics` absent entirely | Keep current no-stat rendering without bracketed stats |
| Supported version with stats and `0` affected instances | Include `in:0` in the bracketed stats |
| Supported version with stats and `N>0` affected instances | Include `in:N` in the bracketed stats |
| Unsupported version where only `ac/cp/cx` can be shown | Keep the bracketed stats but omit the `in:` segment |

The renderer change is not optional: it is the user-visible expression of the clarified spec and must be treated as part of the same feature, not as a follow-up polish item.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Review and confirm the exact `v8.8`/`v8.9` generated endpoint shape for incident-bearing process-instance counts by definition and codify the unsupported `v8.7` boundary in focused research-backed notes.
2. Extend the shared/domain process-definition statistics model only as much as needed to distinguish a supported incident-bearing process-instance count from the unsupported case while keeping current JSON and rendering compatibility for the other statistics fields.
3. Update `internal/services/processdefinition/v88` and `v89` so `WithStat` enrichment combines the existing `ac/cp/cx` source with the newer incident-bearing process-instance count source, with focused service tests proving the clarified count semantics.
4. Preserve `v87` truthfulness by keeping `in:` unsupported there, and add regressions showing `WithStat` does not imply incident-count support on that version.
5. Update `cmd/cmd_views_get.go` and command-level tests so supported zero values print `in:0`, supported counts print numerically, and unsupported versions omit `in:` entirely while leaving the other statistics output unchanged.
6. Update README notes and regenerate CLI docs, then run focused Go tests followed by `make test`.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design reports only values the active version can verify and removes the misleading unsupported placeholder from the user-visible output.
- **CLI-First, Script-Safe Interfaces**: Still passes. The design keeps the same command and flags while tightening the output contract for automation and human operators.
- **Tests and Validation Are Mandatory**: Still passes with versioned service tests, command rendering regressions, docs regeneration, and final `make test`.
- **Documentation Matches User Behavior**: Still passes. README and generated CLI docs are explicitly part of the design because the visible `--stat` contract changes.
- **Small, Compatible, Repository-Native Changes**: Still passes. The work stays in the existing command renderer, facade model, and versioned processdefinition services.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
