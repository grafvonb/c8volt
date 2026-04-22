# Implementation Plan: Graceful Orphan Parent Traversal

**Branch**: `129-orphan-parent-warning` | **Date**: 2026-04-22 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/spec.md)
**Input**: Feature specification from `/specs/129-orphan-parent-warning/spec.md`

## Summary

Refactor orphan-parent handling in the shared process-instance walker and the flows that compose it so `walk`, delete preflight, cancel preflight, and indirect process-definition cleanup return structured partial results plus warnings instead of failing hard when a non-start ancestor is missing. The design keeps direct key lookup and waiter-based absent/deleted confirmation strict, introduces one shared machine-readable missing-ancestor contract across `v87`, `v88`, and `v89`, and scopes successful warning outcomes only to flows that still resolved at least one actionable process instance.

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, existing process-instance services under `internal/services/processinstance/{v87,v88,v89}`, shared helpers in `internal/services/processinstance/walker`, `internal/services/processinstance/waiter`, and facade wrappers under `c8volt/process`  
**Storage**: No persistent datastore changes; existing file-based config and in-memory traversal/preflight structures only  
**Testing**: `go test`, `make test`, command regression tests under `cmd/`, shared walker/waiter tests under `internal/services/processinstance/`, and facade-level tests under `c8volt/process`  
**Target Platform**: Cross-platform CLI for local and CI use against supported Camunda `8.7`, `8.8`, and `8.9` process-instance runtimes  
**Project Type**: CLI  
**Performance Goals**: Preserve current traversal/preflight responsiveness for normal trees, avoid repeated follow-up lookups beyond the existing ancestry/descendant traversal pattern, and keep partial-result handling deterministic without widening waiter polling behavior  
**Constraints**: Preserve existing Cobra command surfaces and render modes, keep direct `get process-instance --key` and absent/deleted waiter behavior strict, reuse the shared process-instance API instead of introducing a parallel traversal subsystem, expose one structured missing-ancestor contract across affected flows, keep warning-based success limited to cases with at least one actionable result, update docs only if the shipped operator-visible behavior changes, and finish with targeted tests plus `make test`  
**Scale/Scope**: Shared walker logic in [`internal/services/processinstance/walker/walker.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker.go), facade dry-run expansion in [`c8volt/process/dryrun.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/dryrun.go), command entry points in [`cmd/walk_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_processinstance.go), [`cmd/cancel_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go), [`cmd/delete_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go), strict lookup/wait seams in [`cmd/get_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go) and [`internal/services/processinstance/waiter/waiter.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter.go), plus versioned process-instance services under `internal/services/processinstance/{v87,v88,v89}` and relevant operator docs in [`README.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md) and generated `docs/cli/`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. The feature exists to make destructive and inspection flows report truthful partial outcomes instead of falsely failing when useful work can still continue, while keeping fully unresolved traversals and waiter confirmation strict.
- **CLI-First, Script-Safe Interfaces**: Pass. The design keeps current commands and flags intact and formalizes a deterministic machine-readable warning contract plus clear success/failure boundaries for scripted use.
- **Tests and Validation Are Mandatory**: Pass. The plan requires walker tests, version-aware service tests where behavior differs, command/facade regression coverage, targeted `go test` runs, and final `make test`.
- **Documentation Matches User Behavior**: Pass. If warning-based partial-result behavior changes what operators see in `walk`, `cancel`, `delete`, or related help text, the plan includes README and regenerated CLI docs updates.
- **Small, Compatible, Repository-Native Changes**: Pass. The design stays inside existing walker, waiter, versioned service, facade, and command layers rather than inventing a new traversal abstraction.

## Project Structure

### Documentation (this feature)

```text
specs/129-orphan-parent-warning/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── orphan-parent-traversal.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── walk_processinstance.go
├── cancel_processinstance.go
├── delete_processinstance.go
├── get_processinstance.go
├── walk_test.go
├── cancel_test.go
├── delete_test.go
└── get_processinstance_test.go

c8volt/process/
├── api.go
├── dryrun.go
├── walker.go
├── client.go
└── client_test.go

internal/services/processinstance/
├── api.go
├── factory.go
├── walker/
│   ├── walker.go
│   └── walker_test.go
├── waiter/
│   ├── waiter.go
│   └── waiter_test.go
├── v87/
│   ├── service.go
│   └── service_test.go
├── v88/
│   ├── service.go
│   └── service_test.go
└── v89/
    ├── service.go
    └── service_test.go

internal/services/common/
└── response.go

README.md
docs/cli/
```

**Structure Decision**: Keep the new behavior centered on the shared `processinstance.API` traversal contract. The walker should become the source of truth for orphan-parent partial results, the facade and commands should consume that richer contract without duplicating traversal rules, and version-specific services should only adapt where their runtime behavior around traversal, state lookup, or mutation preflight needs explicit regression coverage.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/research.md).

- Confirm where orphan-parent failure currently originates and how much partial state the walker already accumulates before returning `services.ErrOrphanedInstance`.
- Confirm which downstream flows depend on `Ancestry`, `Descendants`, or `Family`: direct `walk`, facade dry-run expansion, cancel preflight, delete preflight, indirect process-definition cleanup, and any versioned service force/delete helper that composes the walker.
- Confirm the existing strict boundaries that must not change: direct `GetProcessInstance`, `GetProcessInstanceStateByKey` waiter semantics for absent/deleted success, and current not-found rendering contracts.
- Confirm current version support in the repository: `v87`, `v88`, and `v89` all exist in the process-instance factory and must share the same orphan-parent contract where their traversal uses the shared walker.
- Confirm the best regression seams for partial-result success vs fully unresolved failure behavior across walker, facade, and command tests.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/quickstart.md)
- [contracts/orphan-parent-traversal.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/129-orphan-parent-warning/contracts/orphan-parent-traversal.md)

- Replace the current orphan-parent error-only walker outcome with a shared partial-result structure that retains resolved keys, resolved chain/edges, and machine-readable missing ancestor keys.
- Introduce shared `TraversalResult`/`DryRunPIKeyExpansion` API seams so callers can adopt the partial-result contract incrementally without breaking the current tuple-based traversal methods in one step.
- Keep the shared traversal contract version-neutral: `v87`, `v88`, and `v89` all delegate ancestry/family traversal through the same walker, so the warning/partial-result semantics should not fork by version unless a version-specific test proves an adapter need.
- Make success/failure boundaries explicit: affected traversal and preflight flows succeed with warnings only when at least one actionable result was resolved, and they fail normally when nothing was resolved.
- Preserve strict direct lookup and waiter semantics by confining the new contract to traversal and dependency-expansion callers rather than altering `GetProcessInstance` or the waiter's absent/deleted logic.
- Thread the structured warning contract through command rendering and dry-run expansion so `walk`, `cancel`, `delete`, and indirect resource cleanup can surface partial results consistently without duplicating orphan parsing logic.
- Update the active plan pointer in `AGENTS.md` so downstream Speckit steps target this feature.

### Authoritative Partial-Result Boundary

| Flow segment | Required contract |
|--------------|-------------------|
| `walker.Ancestry` when a non-start parent is missing | Return partial chain plus machine-readable missing ancestor metadata instead of only `ErrOrphanedInstance` |
| `walker.Family` / `walker.Descendants` for affected flows | Continue from the resolved root or boundary data when actionable results remain, and preserve partial edges/chain for rendering and preflight |
| `walk pi --parent` / `--family` / `--family --tree` | Render partial data and warn when ancestors are missing |
| Delete/cancel dry-run expansion | Return resolved family keys and missing ancestor keys through one shared contract; remain actionable when at least one result is resolved |
| Fully unresolved traversal | Return a normal failure rather than warning-only success |
| Direct `get process-instance --key` | Stay strict and keep normal `not found` behavior |
| Wait-for-absent / wait-for-deleted flows | Stay unchanged; absence remains success only in the waiter contract |

This matrix is the authoritative design target for later tasks. Any implementation that changes direct lookup or waiter semantics would require an explicit spec/plan update first.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Refactor the shared walker contract and tests to represent partial ancestry/family results, machine-readable missing ancestor keys, and the success/failure boundary for partially resolved vs fully unresolved traversal.
2. Update facade dry-run expansion and any shared process/resource helpers so cancel/delete preflight can consume the richer traversal contract without losing resolved actionable keys.
3. Thread the warning contract through `walk process-instance` rendering, including family tree/list output and warning presentation for partial trees.
4. Update cancel/delete command flows and any indirect process-definition cleanup path that depends on the same dry-run expansion so they keep orphan children actionable while exposing missing ancestors.
5. Add or update regression coverage across shared walker tests, facade tests, and command tests for `walk`, `cancel`, `delete`, plus strict non-regression coverage for direct `get` and waiter behavior.
6. Update README and regenerate affected CLI docs only if the final command-visible output or help text changes, then finish with focused Go tests and `make test`.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design makes command outcomes more truthful by distinguishing partial success from total failure and by surfacing exactly what remained unresolved.
- **CLI-First, Script-Safe Interfaces**: Still passes. The warning contract is explicit, machine-readable, and bounded to existing commands rather than hidden in logs or ad hoc text parsing.
- **Tests and Validation Are Mandatory**: Still passes with required shared helper, command, and final repository validation.
- **Documentation Matches User Behavior**: Still passes with a conditional docs update path if operator-visible output changes.
- **Small, Compatible, Repository-Native Changes**: Still passes. The design reuses the current API surface and command/facade layering instead of branching into a new traversal subsystem.

## Final Implementation Notes

- The shared walker is the critical seam: once it can distinguish resolved chain data from missing ancestors, the rest of the feature should consume that contract rather than recreate it.
- `c8volt/process/dryrun.go` is the main operator-impact seam after the walker because both cancel and delete preflight depend on it for actionable impact calculation.
- `v87`, `v88`, and `v89` already all expose `Ancestry`, `Descendants`, and `Family` through the shared walker, so the feature should target cross-version contract consistency with versioned regression coverage rather than version-specific behavior forks.
- Direct `GetProcessInstanceStateByKey` and waiter absent/deleted handling are explicitly out of scope for behavior change and should only receive non-regression tests.

## Final Verification Notes

- Targeted validation for this feature is:
  - `go test ./internal/services/processinstance/... -count=1`
  - `go test ./c8volt/process -count=1`
  - `go test ./cmd -count=1`
- Repository validation remains `make test`, which is required before implementation is complete.
- If docs change, regenerate CLI markdown with `make docs-content` so `README.md` and `docs/cli/` stay aligned.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
