# Implementation Plan: Relative Day-Based Process-Instance Date Shortcuts

**Branch**: `095-processinstance-day-filters` | **Date**: 2026-04-10 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/095-processinstance-day-filters/spec.md)
**Input**: Feature specification from `/specs/095-processinstance-day-filters/spec.md`

## Summary


## Technical Context

**Language/Version**: Go 1.25.3  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, existing process-instance command helpers in `cmd/`, shared facade/domain filters in `c8volt/process` and `internal/domain`, generated Camunda clients under `internal/clients/camunda/...`, existing versioned services under `internal/services/processinstance/...`  
**Storage**: N/A  
**Testing**: `go test`, `make test`, command tests under `cmd/`, shared facade regression tests in `c8volt/process`, versioned service tests under `internal/services/processinstance/v88`
**Target Platform**: CLI for local and CI use against Camunda 8.8 environments
**Project Type**: CLI  
**Performance Goals**: Preserve existing server-side process-instance filtering behavior and avoid client-side overfetching while supporting relative-day convenience inputs across result sets up to the current search limit  
**Scale/Scope**: Three existing commands (`get process-instance`, `cancel process-instance`, `delete process-instance`), shared search-helper validation and filter population in `cmd/get_processinstance.go`, existing shared filter models and versioned process-instance services, targeted command/facade/service tests, and user-facing CLI documentation

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. `get process-instance` remains a read/list path and `cancel`/`delete` already own operational completion behavior. This feature changes only how date-based selection inputs are translated before the existing search or action flows run.
- **CLI-First, Script-Safe Interfaces**: Pass. The feature is exposed as additive Cobra flags on existing commands, keeps command hierarchy unchanged, and makes invalid or unsupported combinations fail explicitly instead of silently choosing one input.
- **Tests and Validation Are Mandatory**: Pass. The plan requires targeted command, facade, and versioned service coverage where behavior changes surface, plus `make test` before commit.
- **Documentation Matches User Behavior**: Pass. The feature adds user-visible flags and examples to existing commands, so `README.md`, generated CLI docs, and command help must stay synchronized.
- **Small, Compatible, Repository-Native Changes**: Pass. The design intentionally reuses the current CLI -> shared process-instance filter -> versioned service path from issues `#90` and `#93` rather than creating new filter types or command-specific shortcuts.

## Project Structure

### Documentation (this feature)

```text
specs/095-processinstance-day-filters/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── process-instance-relative-day-filters.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_processinstance.go
├── get_processinstance_test.go
├── cancel_processinstance.go
├── cancel_test.go
├── delete_processinstance.go
├── delete_test.go
└── cmd_processinstance_test.go

c8volt/process/
├── client.go
├── client_test.go
└── model.go

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
├── c8volt_get_process-instance.md
├── c8volt_cancel_process-instance.md
└── c8volt_delete_process-instance.md
```

**Structure Decision**: Keep the feature inside the repository’s existing command search-helper seam. `cmd/get_processinstance.go` remains the canonical home for shared process-instance search flag parsing, search-filter detection, and validation helpers already used by `cancel_processinstance.go` and `delete_processinstance.go`. Shared filter shape stays in `c8volt/process/model.go` and `internal/domain/processinstance.go`, while version-specific date behavior stays in `internal/services/processinstance/v88/service.go`. User-visible documentation changes stay in `README.md` plus regenerated files under `docs/cli/`.

## Phase 0: Research

- Confirm the safest repository-native place to derive relative day inputs into absolute date bounds while preserving existing `validatePISearchFlags()`, `hasPISearchFilterFlags()`, and `populatePISearchFilterOpts()` behavior used across `get`, `cancel`, and `delete`.
- Confirm the configured-environment local-day rule and explicit `--key` incompatibility should remain shared across all three commands.
- Confirm documentation and test obligations for new relative day flags on already-documented commands.

## Phase 1: Design

- Add shared relative-day flag variables and command registration for `--start-date-newer-days`, `--start-date-older-days`, `--end-date-newer-days`, and `--end-date-older-days` across `get process-instance`, `cancel process-instance`, and `delete process-instance`.
- Reuse the existing command-layer validation seam to parse non-negative integer day values, reject mixed absolute-plus-relative filters for the same field, reject invalid derived ranges, and reject explicit `--key` combined with relative day filters before any search-based action occurs.
- Derive relative day shortcuts into the existing absolute date bound fields used by `process.ProcessInstanceFilter` so downstream facade and service layers continue to operate on the canonical absolute filter model.
- Update README examples and regenerate CLI docs so the three command surfaces advertise the new shortcut flags consistently with the shipped command help.

## Phase 2: Task Planning Approach

- Start with shared command-helper and flag-surface changes because all three affected commands depend on the same search validation and filter composition path.
- Add or update tests around derived-bound conversion and invalid-combination behavior next so regressions are caught before any service or documentation work.
- Reuse the already-existing absolute date filter plumbing in facade and versioned services, extending lower-level tests only where relative-day conversion must prove it lands on the canonical absolute behavior.
- Finish with README and generated CLI documentation updates, then run `make test` for final validation.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design changes selection input translation only and keeps existing cancel/delete completion flows untouched.
- **CLI-First, Script-Safe Interfaces**: Still passes. The design uses additive flags, explicit validation failures, and shared command wiring rather than silent fallback behavior.
- **Tests and Validation Are Mandatory**: Still passes with planned command, facade, and versioned service coverage plus `make test`.
- **Documentation Matches User Behavior**: Still passes with explicit README updates and regenerated CLI reference pages.
- **Small, Compatible, Repository-Native Changes**: Still passes. The design reuses the current absolute date-filter model and command helpers instead of creating a relative-day-only subsystem.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
