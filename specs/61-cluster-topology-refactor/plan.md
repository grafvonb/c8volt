# Implementation Plan: Refactor Cluster Topology Command

**Branch**: `61-cluster-topology-refactor` | **Date**: 2026-03-18 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/61-cluster-topology-refactor/spec.md)
**Input**: Feature specification from `/specs/61-cluster-topology-refactor/spec.md`

## Summary

Refactor cluster topology retrieval from the flat `c8volt get cluster-topology` command into the nested `c8volt get cluster topology` hierarchy while preserving the current execution path, output, and exit behavior. The implementation will introduce a repository-native Cobra parent command under `get`, keep the legacy command as a working deprecated compatibility alias without runtime warnings, and validate the change through focused command tests, help/docs updates, CLI doc regeneration, and `make test`.

## Technical Context

**Language/Version**: Go 1.25.3  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`  
**Storage**: N/A  
**Testing**: `go test ./... -race -count=1` via `make test`, targeted command tests in `cmd`, existing cluster service tests remain as regression coverage  
**Target Platform**: Cross-platform CLI execution in local development and CI environments supported by Go  
**Project Type**: Go CLI application  
**Performance Goals**: No observable regression in topology retrieval latency, output shape, or exit behavior between the legacy and nested command paths  
**Constraints**: Preserve current topology functionality, reuse existing Cobra wiring patterns, no runtime deprecation warning on the legacy command, update CLI docs and README only where behavior is user-visible, avoid new dependencies and parallel command structures  
**Scale/Scope**: One command area under `cmd/`, adjacent command tests, generated CLI docs under `docs/cli/`, and any matching README command examples

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- `Operational proof`: Pass. The feature keeps the same underlying topology retrieval flow and must preserve current success/failure reporting for both command paths; no new verification gap is introduced.
- `CLI compatibility`: Pass with managed compatibility. The new preferred command path is added under the existing `get` tree, while the old `cluster-topology` entry point remains functional as a compatibility alias with help/docs-only deprecation messaging.
- `Validation`: Pass. Add or update command-level tests for both paths, retain existing service-level topology coverage, and finish with `make test`.
- `Documentation parity`: Pass. Update generated CLI docs and any relevant README usage so the new hierarchy is discoverable and the legacy path is marked deprecated in documentation.
- `Complexity control`: Pass. Reuse Cobra command composition already used in the repository; no new dependencies or abstractions are required.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/61-cluster-topology-refactor/research.md).

- Confirm the lowest-risk Cobra structure for introducing `get cluster topology` without changing execution behavior.
- Confirm how to keep `cluster-topology` working as a compatibility entry point while restricting deprecation messaging to help and documentation.
- Confirm the documentation and validation surface required for a user-visible CLI hierarchy change.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/61-cluster-topology-refactor/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/61-cluster-topology-refactor/quickstart.md)
- [contracts/cli-command-contract.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/61-cluster-topology-refactor/contracts/cli-command-contract.md)

The contract artifact is included because this feature changes the public CLI command hierarchy and deprecation guidance even though the topology behavior remains unchanged.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Add the `get cluster` parent command using existing Cobra patterns.
2. Move topology command wiring under the new parent as `get cluster topology` while preserving the current execution logic.
3. Reintroduce `get cluster-topology` as a compatibility command or alias that reaches the same handler and carries deprecation messaging only in help/documentation.
4. Add or update focused command tests for help text, command routing, and preserved execution behavior.
5. Update README examples if present and regenerate CLI docs so both the new command path and the deprecated legacy path are documented correctly.
6. Run targeted Go tests, regenerate docs if needed, and finish with `make test`.

## Project Structure

### Documentation (this feature)

```text
specs/61-cluster-topology-refactor/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-command-contract.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get.go
├── get_cluster_topology.go
├── root.go
└── cmd_*test.go

internal/domain/
└── cluster.go

internal/services/cluster/
├── api.go
├── factory.go
├── common/
├── v87/
└── v88/

docs/cli/
├── c8volt_get.md
├── c8volt_get_cluster-topology.md
└── c8volt_get_cluster*.md

README.md
```

**Structure Decision**: Use the existing single-project Go CLI layout and keep changes localized to Cobra command wiring, adjacent command tests, and user-facing documentation. The topology retrieval implementation remains in the current service layer, which avoids introducing a parallel command or service structure.

## Post-Design Constitution Check

- `Operational proof`: Still passes. Both command paths are planned to execute the same topology retrieval behavior and preserve current outcome reporting.
- `CLI compatibility`: Still passes. The design adds the nested command without removing the legacy path and explicitly keeps deprecation messaging out of runtime output.
- `Validation`: Still passes. The design maps directly to command tests plus repository-wide `make test`.
- `Documentation parity`: Still passes. The design requires updating generated CLI docs and any affected README usage to match the shipped command hierarchy.
- `Complexity control`: Still passes. No new dependencies, no new service abstractions, and no non-native command-tree patterns are introduced.

## Complexity Tracking

No constitution violations or justified complexity exceptions are expected for this feature.
