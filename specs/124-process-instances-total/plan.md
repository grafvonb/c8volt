# Implementation Plan: Add Process-Instance Total-Only Output

**Branch**: `124-process-instances-total` | **Date**: 2026-04-22 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/spec.md)
**Input**: Feature specification from `/specs/124-process-instances-total/spec.md`

## Summary

Add a `--total` flag to `get process-instance` so search/list usage can return only the numeric count of matching process instances without printing instance details. The design keeps the existing Cobra command and shared process-instance search path, adds a count-only branch on the `get pi` command instead of inventing a new render mode, carries backend-reported total metadata through the shared process-instance page model so `v8.8` and `v8.9` can preserve capped lower-bound totals exactly as clarified, falls back safely on `v8.7`, and updates README plus generated CLI docs to match the new user-visible flag.

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/pflag`, `github.com/spf13/viper`, `github.com/stretchr/testify`, shared process facade under `c8volt/process`, process-instance domain/service layers under `internal/domain` and `internal/services/processinstance/{v87,v88,v89}`, generated Camunda clients under `internal/clients/camunda/v88/camunda` and `internal/clients/camunda/v89/camunda`  
**Storage**: File-based YAML config plus environment variables; no persistent datastore changes  
**Testing**: focused `go test ./cmd -count=1`, `go test ./c8volt/process -count=1`, `go test ./internal/services/processinstance/... -count=1`, documentation refresh with `make docs-content`, final repository validation with `make test`  
**Target Platform**: Cross-platform CLI for local and CI use against supported Camunda `8.7`, `8.8`, and `8.9` environments  
**Project Type**: CLI  
**Performance Goals**: Keep `--total` machine-friendly and fast for search/list workflows, avoid unnecessary detail rendering, preserve current search paging behavior unless the active version already exposes a backend-reported total, and avoid forcing full page traversal on versions that already expose an authoritative or lower-bound total  
**Constraints**: Preserve strict keyed lookup behavior, keep `--total` on the list/search path rather than direct `--key` fetches, preserve the clarified rule that capped totals remain numeric lower bounds instead of triggering recounts or failures, avoid introducing a new global render mode, keep command contract and output-mode metadata coherent, update README and generated CLI docs because this is user-visible behavior, and finish with `make test`  
**Scale/Scope**: [`cmd/get_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go), [`cmd/cmd_views_get.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_get.go), shared render/contract helpers under [`cmd/`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd), public process models and conversions in [`c8volt/process/`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process), domain page models in [`internal/domain/processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/processinstance.go), versioned process-instance services/tests under [`internal/services/processinstance/`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance), plus [`README.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md) and generated CLI docs under [`docs/cli/`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. `--total` exists to report the search result count directly, so the plan explicitly preserves truthful lower-bound behavior when the backend marks totals as capped instead of pretending an exact recount happened.
- **CLI-First, Script-Safe Interfaces**: Pass. The change stays in the existing `get process-instance` Cobra command as a script-safe flag and avoids introducing a parallel command family or hidden semantics outside the CLI surface.
- **Tests and Validation Are Mandatory**: Pass. The plan requires command-level flag/behavior tests, shared model conversion coverage, versioned service page-metadata coverage, docs regeneration, and final `make test`.
- **Documentation Matches User Behavior**: Pass. This is a user-visible new flag, so README examples and generated CLI reference must be updated in the same unit of work.
- **Small, Compatible, Repository-Native Changes**: Pass. The design reuses the existing `get pi` search pipeline, shared process models, and versioned page services rather than adding a new counting subsystem or alternate command.

## Project Structure

### Documentation (this feature)

```text
specs/124-process-instances-total/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── process-instance-total-output.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_processinstance.go
├── cmd_views_get.go
├── cmd_views_rendermode.go
├── command_contract.go
├── get_processinstance_test.go
└── cmd_processinstance_test.go

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
├── v88/
│   ├── service.go
│   └── service_test.go
└── v89/
    ├── service.go
    └── service_test.go

README.md
docs/cli/
└── c8volt_get_process-instance.md
```

**Structure Decision**: Keep the feature inside the existing `get process-instance` command, shared process facade, and versioned process-instance page services. The command should remain responsible for flag validation and output selection, while shared/domain page models gain the minimal metadata needed to expose backend-reported totals consistently across versions.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/research.md).

- Confirm whether `--total` should be implemented as a command-specific flag instead of a new global render mode.
- Confirm how the current `get pi` flow renders one-line, JSON, and keys-only output so count-only behavior can remain small and compatible.
- Confirm where backend-reported totals and capped-total signals already exist in `v87`, `v88`, and `v89`, and identify the shared model gap that prevents the command from using them today.
- Confirm the correct contract boundary for `--total` versus `--key`, `--json`, `--keys-only`, and `--with-age`.
- Confirm the documentation regeneration path for CLI reference output after flag additions.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/quickstart.md)
- [contracts/process-instance-total-output.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/124-process-instances-total/contracts/process-instance-total-output.md)

- Add `--total` to `get process-instance` as a command-specific count-only flag, not as a new shared render mode.
- Keep `--total` limited to search/list workflows; direct `--key` lookups remain on the strict single-resource path and should reject `--total`.
- Treat `--total` as mutually exclusive with detail-oriented output modifiers such as `--json`, `--keys-only`, and `--with-age` so the command can keep the spec’s numeric-only contract without weakening the shared output-mode model.
- Extend the shared domain/public page model with reported-total metadata so `v8.8` and `v8.9` can pass through `totalItems` plus capped/lower-bound state, and `v8.7` can surface the best available total signal from its current response payload.
- Keep count-only logic centralized in the existing `get pi` command/search flow, using backend-reported totals when available and preserving the clarified lower-bound contract instead of forcing recounts.
- Update README and regenerate `docs/cli/c8volt_get_process-instance.md` through `make docs-content`.

### Authoritative `--total` Behavior Boundary

| Scenario | Planned behavior |
|--------|------------------|
| Search/list invocation with `--total` | Print only the numeric count result |
| Search/list invocation with zero matches | Print `0` only |
| Backend total is capped lower bound | Print the backend-reported numeric lower bound unchanged |
| Direct `--key` lookup with `--total` | Reject as invalid flag combination |
| `--total` with `--json` | Reject as invalid flag combination |
| `--total` with `--keys-only` | Reject as invalid flag combination |
| `--total` with `--with-age` | Reject as invalid flag combination |

This table is the planning boundary for later tasks. Any implementation that silently emits detail output, quietly changes JSON semantics, or invents exact recount behavior for capped totals is incomplete.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Add the new `--total` flag and validation rules in `cmd/get_processinstance.go`, including explicit incompatibility handling with direct `--key` lookups and conflicting output modifiers.
2. Extend shared page/domain/public process-instance page models and conversion seams to carry reported total metadata plus capped/lower-bound state without changing existing detail output behavior.
3. Update `v87`, `v88`, and `v89` process-instance page services/tests so page results expose the correct reported total semantics for count-only mode, including lower-bound behavior on versions that surface capped totals.
4. Add the count-only command path and focused command regressions proving numeric-only output, zero-match behavior, invalid flag combinations, and preserved default detail rendering.
5. Update README and regenerate CLI docs with `make docs-content`, then finish with focused Go tests and final `make test`.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design makes count reporting more direct while preserving the clarified rule that capped totals stay lower bounds instead of being overstated as exact counts.
- **CLI-First, Script-Safe Interfaces**: Still passes. `--total` is a small additive flag on the existing command, with explicit incompatibility rules rather than surprising output-mode overrides.
- **Tests and Validation Are Mandatory**: Still passes with command, shared-model, and versioned-service tests plus `make docs-content` and final `make test`.
- **Documentation Matches User Behavior**: Still passes. README and generated CLI docs are required deliverables because the command surface changes.
- **Small, Compatible, Repository-Native Changes**: Still passes. The design reuses current command, model, and service seams and adds only the minimum shared page metadata needed for the feature.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
