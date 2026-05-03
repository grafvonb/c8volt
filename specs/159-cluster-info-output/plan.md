# Implementation Plan: Improve Cluster Info Output And Version Command

**Branch**: `159-cluster-info-output` | **Date**: 2026-05-03 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/159-cluster-info-output/spec.md)  
**Input**: Feature specification from `/specs/159-cluster-info-output/spec.md`

## Summary

Improve cluster inspection commands by changing `get cluster topology` and `get cluster license` to human-readable default output, preserving structured JSON behind `--json`, adding `get cluster version`, adding `licence` as a license alias, and removing the legacy direct `get cluster-topology` command path. The implementation should stay in the existing Go/Cobra CLI layer, reuse the current cluster service topology/license calls, add focused command tests for each user-visible output path, and refresh README plus generated CLI documentation.

## Technical Context

**Language/Version**: Go 1.25.3  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`  
**Storage**: N/A  
**Testing**: Targeted `go test ./cmd -count=1`, focused cluster service regression checks if domain/service mapping changes, and final `make test`  
**Target Platform**: Cross-platform CLI execution in local development and CI environments supported by Go  
**Project Type**: Go CLI application  
**Performance Goals**: No observable extra network calls for `get cluster version`; topology/license rendering remains fast and deterministic for typical cluster sizes  
**Constraints**: Reuse existing cluster topology/license service calls and domain models; no new dependencies; remove direct legacy topology command intentionally; keep `--json` as the machine-readable compatibility path; update docs generated from Cobra metadata rather than hand-editing derived command pages  
**Scale/Scope**: Cluster command files under `cmd/`, adjacent command view helpers, command/capability tests in `cmd`, generated CLI docs under `docs/cli/`, `README.md`, `docs/index.md`, and feature artifacts under `specs/159-cluster-info-output/`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- `Operational proof`: Pass. The commands are read-only cluster inspection commands and will preserve established success/failure handling while changing only successful human-readable output.
- `CLI compatibility`: Pass with explicit breaking change. The plan intentionally removes `get cluster-topology` per clarification and documents/tests that removal; the nested `get cluster topology` path remains the supported replacement.
- `Validation`: Pass. The plan requires tests for topology tree output, topology JSON output, version default output, version broker output, flat license output, license JSON output, alias behavior, and legacy command removal.
- `Documentation parity`: Pass. README, `docs/index.md`, and generated CLI docs must reflect the new version command, human defaults, JSON escape hatch, `licence` alias, and removal of `get cluster-topology`.
- `Complexity control`: Pass. The feature reuses existing Cobra command patterns, render helpers, domain structs, and cluster service calls without adding dependencies or parallel service abstractions.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/159-cluster-info-output/research.md).

- Confirm output-mode strategy for human defaults while preserving JSON envelopes.
- Confirm the safest structure for `get cluster version` and its `--with-brokers` flag.
- Confirm removal scope for legacy `get cluster-topology`, including aliases, tests, capability metadata, docs, and examples.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/159-cluster-info-output/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/159-cluster-info-output/quickstart.md)
- [contracts/cli-command-contract.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/159-cluster-info-output/contracts/cli-command-contract.md)

The contract artifact is included because this feature changes public CLI command output, adds a new command, adds an alias, and removes a previously public command path.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Update command tests first to capture the new desired behavior and removal of the legacy command path.
2. Add cluster rendering helpers for topology tree output, license flat output, and version output using the existing cluster domain structs.
3. Rewire `get cluster topology` and `get cluster license` to choose human rendering by default and JSON rendering only when `--json` is set.
4. Add `get cluster version` under the existing `get cluster` parent with `--with-brokers`, reusing the topology service call.
5. Remove direct `get cluster-topology` command registration and its aliases/tests/docs while keeping `get cluster topology` discoverable.
6. Update capability metadata, README, docs source content, and regenerated CLI reference pages.
7. Run targeted command tests, docs generation, and final `make test`.

## Project Structure

### Documentation (this feature)

```text
specs/159-cluster-info-output/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-command-contract.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get.go
├── get_cluster.go
├── get_cluster_topology.go
├── get_cluster_license.go
├── get_cluster_version.go
├── cmd_views_cluster.go
├── get_test.go
├── root_test.go
└── command_contract_test.go

internal/domain/
└── cluster.go

internal/services/cluster/
├── api.go
├── factory.go
├── common/
├── v87/
├── v88/
└── v89/

docs/cli/
├── c8volt_get.md
├── c8volt_get_cluster.md
├── c8volt_get_cluster_topology.md
├── c8volt_get_cluster_license.md
└── c8volt_get_cluster_version.md

README.md
docs/index.md
```

**Structure Decision**: Use the existing single-project Go CLI layout. Keep service calls unchanged unless tests reveal a missing domain field, keep output formatting in `cmd` view helpers, keep command wiring in the adjacent `get_cluster*.go` files, and update generated docs through the repository docs generation target.

## Post-Design Constitution Check

- `Operational proof`: Still passes. The design keeps read-only cluster requests and verifies observable command output through command tests.
- `CLI compatibility`: Still passes with documented break. The removed direct topology command is explicit in spec, contract, tasks, tests, and documentation requirements.
- `Validation`: Still passes. Each user story maps to specific command tests plus final `make test`.
- `Documentation parity`: Still passes. Command help, README, docs homepage, and generated CLI docs are included.
- `Complexity control`: Still passes. No new dependencies, no new service abstractions, and no alternate command hierarchy are planned.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Removal of a documented legacy command | The user clarified that `c8volt get cluster-topology` should be removed as part of issue #159 | Preserving it would conflict with the clarified feature scope and keep duplicate command paths alive |
