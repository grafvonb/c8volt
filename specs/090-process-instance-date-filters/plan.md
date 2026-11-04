# Implementation Plan: Day-Based Process Instance Date Filters

**Branch**: `090-process-instance-date-filters` | **Date**: 2026-04-06 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/090-process-instance-date-filters/spec.md)
**Input**: Feature specification from `/specs/090-process-instance-date-filters/spec.md`

## Summary


## Technical Context

**Language/Version**: Go 1.25.3  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...`, existing helpers in `internal/services/common`  
**Storage**: N/A  
**Testing**: `go test`, `make test`, command tests under `cmd/`, service tests under `internal/services/processinstance/...`  
**Target Platform**: CLI for local and CI use against Camunda 8.8 environments
**Project Type**: CLI  
**Performance Goals**: Preserve current `get process-instance` search behavior and server-side filtering efficiency for result sets up to the existing search limit of 1000 items  
**Scale/Scope**: Single command family (`get process-instance`), shared process-instance filter models, versioned v8.8 service search paths, related command/service tests, and user-facing docs

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. This is a read/list command, so the required operational guarantee is accurate result narrowing and clear unsupported-version behavior rather than post-action polling. The plan keeps filtering in the normal search flow and verifies outcomes with command and service tests.
- **CLI-First, Script-Safe Interfaces**: Pass. The feature is exposed as explicit Cobra flags on the existing `get process-instance` command, preserves current command structure, and keeps machine-readable output behavior unchanged apart from narrowed result sets.
- **Tests and Validation Are Mandatory**: Pass. The plan adds targeted command validation tests, v8.8 service tests, and requires `make test` before commit.
- **Documentation Matches User Behavior**: Pass. The feature changes user-visible flags and behavior, so `README.md` and generated CLI docs under `docs/cli/` must be updated via `make docs-content` and `make docs`.
- **Small, Compatible, Repository-Native Changes**: Pass. The work stays within existing command/filter/service patterns and versioned process-instance service split without introducing new subsystems.

## Project Structure

### Documentation (this feature)

```text
specs/090-process-instance-date-filters/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── get-process-instance-date-filters.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_processinstance.go
└── get_processinstance_test.go

c8volt/process/
├── client.go
├── model.go
└── client_test.go

internal/domain/
└── processinstance.go

internal/services/processinstance/
├── api.go
└── v88/
    ├── contract.go
    ├── service.go
    └── service_test.go

README.md
docs/cli/
```

**Structure Decision**: Use the repository’s existing CLI -> facade -> domain filter -> versioned service flow. Command validation and flag wiring stay in `cmd/`, shared filter shape changes live in `c8volt/process` and `internal/domain`, version-specific search behavior stays in `internal/services/processinstance/v88`, and user-facing flag documentation is updated in `README.md` plus regenerated `docs/cli/`.

## Phase 0: Research

- Confirm whether native v8.8 search filters support date-range comparisons and how to express inclusive bounds.
- Confirm where command-level validation should live to preserve current `--key`/filter exclusivity and shared error behavior.
- Confirm documentation/regeneration obligations for new CLI flags.

## Phase 1: Design

- Extend the process-instance filter models with date range fields that can represent inclusive lower and upper bounds for both start and end date searches.
- Add command flags and validation helpers for date-only parsing, inclusive range validation, and incompatibility with direct ID lookup.
- Map v8.8 date filters into native generated-client datetime filter properties using inclusive `$gte`/`$lte` request fields and configured-environment day semantics.
- Cover behavior with command tests and versioned service tests before updating docs.

## Phase 2: Task Planning Approach

- Start with shared filter model changes because both command wiring and service implementations depend on them.
- Implement command validation next so incorrect inputs fail before service execution.
- Finish with tests and documentation regeneration so user-visible output matches shipped behavior.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. No new asynchronous semantics are introduced.
- **CLI-First, Script-Safe Interfaces**: Still passes. The design uses additive flags and explicit validation failures.
- **Tests and Validation Are Mandatory**: Still passes with planned command and service coverage plus `make test`.
- **Documentation Matches User Behavior**: Still passes with explicit README and generated CLI docs updates.
- **Small, Compatible, Repository-Native Changes**: Still passes. The design reuses existing models and versioned services instead of adding a parallel date-filter subsystem.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
