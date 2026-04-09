# Implementation Plan: Extend Process-Instance Management Date Filters

**Branch**: `093-extend-pi-date-filters` | **Date**: 2026-04-09 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/093-extend-pi-date-filters/spec.md)
**Input**: Feature specification from `/specs/093-extend-pi-date-filters/spec.md`

## Summary

Extend the day-based start/end date filters from `c8volt get process-instance` into the existing search-driven management commands `c8volt cancel process-instance` and `c8volt delete process-instance`. The implementation reuses the current shared process-instance filter model and versioned service support added for issue `#90`, adds the missing Cobra flags plus command-layer validation for the management commands, preserves direct key workflows by rejecting `--key` with date filters, and updates tests and user-facing CLI documentation.

## Technical Context

**Language/Version**: Go 1.25.3  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...`, existing helpers in `internal/services/common`  
**Storage**: N/A  
**Testing**: `go test`, `make test`, command tests under `cmd/`, service/facade regression tests in existing process-instance packages  
**Target Platform**: CLI for local and CI use against Camunda 8.7 and 8.8 environments  
**Project Type**: CLI  
**Performance Goals**: Preserve existing search-based cancel/delete behavior and continue to use server-side process-instance filtering so matching sets up to the existing search limit do not require client-side overfetching  
**Constraints**: Reuse the issue `#90` date-filter semantics unchanged, reject explicit `--key` combined with date filters, keep v8.7 behavior as not implemented through the shared error model, preserve existing confirmation and bulk-management flows, update README and generated CLI docs for new user-visible flags  
**Scale/Scope**: Two existing commands (`cancel process-instance`, `delete process-instance`), shared command-layer process-instance search helpers, existing typed filter models and versioned services, targeted command/facade/service tests, and user-facing docs

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. These are management commands that already confirm outcomes through existing cancel/delete flows. The feature only changes how targets are selected before those flows execute, so operational guarantees remain intact as long as search selection and validation are correct.
- **CLI-First, Script-Safe Interfaces**: Pass. The work adds explicit flags to existing Cobra commands, preserves current command hierarchy, and makes invalid combinations fail clearly rather than silently ignoring user input.
- **Tests and Validation Are Mandatory**: Pass. The plan requires targeted tests for the new flag surfaces and `make test` before commit.
- **Documentation Matches User Behavior**: Pass. The feature changes visible command flags and examples, so `README.md` and regenerated CLI docs must be updated in the same unit of work.
- **Small, Compatible, Repository-Native Changes**: Pass. The plan reuses the shared process-instance filter and versioned service infrastructure already introduced for issue `#90` instead of introducing a parallel management-only filter path.

## Project Structure

### Documentation (this feature)

```text
specs/093-extend-pi-date-filters/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── process-instance-management-date-filters.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── cancel_processinstance.go
├── cancel_test.go
├── delete_processinstance.go
├── delete_test.go
├── get_processinstance.go
└── get_processinstance_test.go

c8volt/process/
├── client.go
├── client_test.go
├── convert.go
└── model.go

internal/domain/
└── processinstance.go

internal/services/processinstance/
├── api.go
├── v87/
│   ├── service.go
│   └── service_test.go
└── v88/
    ├── service.go
    └── service_test.go

README.md
docs/cli/
docs/index.md
```

**Structure Decision**: Keep the feature inside the repository’s existing CLI -> shared process filter -> versioned process-instance service path. `get_processinstance.go` remains the canonical source for date-filter parsing and shared search-filter composition, while `cancel_processinstance.go` and `delete_processinstance.go` opt into the same search filter surface and validation rules. Existing model and service files are referenced because the plan depends on their current issue `#90` behavior rather than introducing new structural layers.

## Phase 0: Research

- Confirm that issue `#90` already landed the shared process-instance filter fields, v8.8 mapping, and v8.7 rejection so `#93` can remain command-surface work rather than duplicating service logic.
- Confirm where cancel/delete currently reuse shared search helpers and where their missing flag registration and validation create the current behavior gap.
- Confirm the safest repository-native behavior for explicit `--key` combined with date filters and encode the clarified invalid-combination rule.
- Confirm documentation regeneration obligations for new cancel/delete flags and examples.

## Phase 1: Design

- Expose the four date flags on `cancel process-instance` and `delete process-instance` using the same flag variables and help text already used by `get process-instance`.
- Reuse the existing command-layer date validation helpers so invalid dates, inverted ranges, and `--key` plus date-filter combinations fail before any search or management action.
- Preserve the current search-based target selection flow by continuing to use `populatePISearchFilterOpts()` and `hasPISearchFilterFlags()` for cancel/delete when no explicit keys are provided.
- Preserve direct key workflows by rejecting any explicit `--key` usage combined with `--start-date-*` or `--end-date-*` flags.
- Add or extend tests to cover command validation, search-filter wiring, and regression expectations for cancel/delete while relying on the already-existing process-instance service coverage for versioned date-filter semantics.
- Update README examples and regenerate CLI docs so `cancel` and `delete` surfaces match shipped behavior.

## Phase 2: Task Planning Approach

- Start with command-surface changes in `cmd/` because the shared filter and service support already exist and the user-visible gap is there.
- Add validation and invalid-combination coverage next so unsafe command invocations fail before any bulk-management action.
- Reuse existing v8.7/v8.8 service behavior through targeted regression tests rather than reopening the service design unless command tests reveal a gap.
- Finish with README and generated CLI doc updates so user-facing help stays synchronized with the new flags.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design changes only pre-selection and keeps the existing cancel/delete confirmation and completion behavior untouched.
- **CLI-First, Script-Safe Interfaces**: Still passes. Additive flags and explicit invalid-combination failures preserve script safety.
- **Tests and Validation Are Mandatory**: Still passes with planned command regression coverage and `make test`.
- **Documentation Matches User Behavior**: Still passes with explicit README and generated CLI documentation updates.
- **Small, Compatible, Repository-Native Changes**: Still passes. The design relies on existing shared search helpers and issue `#90` infrastructure instead of creating a second management-only filter stack.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
