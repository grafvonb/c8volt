# Implementation Plan: Add Cluster License Command

**Branch**: `63-cluster-license` | **Date**: 2026-03-18 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/63-cluster-license/spec.md)
**Input**: Feature specification from `/specs/63-cluster-license/spec.md`

## Summary

Add the nested `c8volt get cluster license` command under the existing `get cluster` hierarchy by reusing the repository's current Cobra patterns and the already-available internal cluster license service. The implementation will keep the change narrowly scoped to command wiring, shared command execution behavior, focused `cmd` tests, and user-facing documentation updates plus CLI doc regeneration.

## Technical Context

**Language/Version**: Go 1.25.3  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`  
**Storage**: N/A  
**Testing**: `go test ./cmd/... -race -count=1`, existing `go test ./internal/services/cluster/... -race -count=1`, and repository-wide `make test`  
**Target Platform**: Cross-platform CLI execution in local development and CI environments supported by Go  
**Project Type**: Go CLI application  
**Performance Goals**: No observable regression in command startup or license retrieval behavior relative to existing `get cluster` commands; successful output and failure semantics remain consistent with current cluster-read workflows  
**Constraints**: Add only the nested `c8volt get cluster license` command, reuse existing cluster service behavior, preserve inherited flag handling, avoid new dependencies or abstractions, update README and generated CLI docs when command behavior becomes user-visible  
**Scale/Scope**: One cluster-read command area under `cmd/`, adjacent command tests, generated CLI docs under `docs/cli/`, and matching command examples in `README.md` and `docs/index.md`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- `Operational proof`: Pass. The command is read-only and reuses the existing cluster license retrieval flow, so the feature preserves current success/failure reporting without introducing a new verification gap.
- `CLI compatibility`: Pass. The feature adds one nested subcommand under the existing `get cluster` hierarchy, preserves standard Cobra flag propagation, and intentionally does not introduce a parallel legacy `cluster-license` path.
- `Validation`: Pass. The plan adds focused command tests for help, success, and failure paths, relies on existing service-level license coverage as regression protection, and finishes with `make test`.
- `Documentation parity`: Pass. The plan requires updating README usage where cluster reads are documented and regenerating CLI reference pages from Cobra metadata.
- `Complexity control`: Pass. The design reuses the existing command tree, domain model, service API, and output pattern without adding dependencies or non-native abstractions.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/63-cluster-license/research.md).

- Confirm the lowest-risk Cobra shape for adding `license` under `get cluster`.
- Confirm the safest way to reuse the existing cluster license service and domain output without introducing new command-specific translation logic.
- Confirm the validation and documentation surface required for a new user-visible `get cluster` command.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/63-cluster-license/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/63-cluster-license/quickstart.md)
- [contracts/cli-command-contract.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/63-cluster-license/contracts/cli-command-contract.md)

The contract artifact is included because this feature introduces a new public CLI command and needs a stable, testable description of command behavior, output expectations, help visibility, and documentation obligations.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Add a `license` subcommand beneath the existing `get cluster` command using the same Cobra style as neighboring cluster read commands.
2. Reuse the current CLI construction path and internal cluster service call so the new command prints the existing `domain.License` payload through the standard JSON helper.
3. Add or update command tests for `get --help`, `get cluster --help`, `get cluster license --help`, successful license retrieval, and failing license retrieval with exit-code assertions.
4. Update README and `docs/index.md` examples where the new cluster read command should be discoverable, then regenerate CLI docs with `make docs`.
5. Run targeted Go tests, regenerate docs if changed, and finish with `make test`.

## Project Structure

### Documentation (this feature)

```text
specs/63-cluster-license/
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
├── get_cluster.go
├── get_cluster_topology.go
├── root.go
└── get_test.go

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
├── c8volt_get_cluster.md
└── c8volt_get_cluster*.md

README.md
docs/index.md
```

**Structure Decision**: Use the existing single-project Go CLI layout and keep changes localized to Cobra command wiring, adjacent command tests, and user-facing documentation. The license retrieval implementation remains in the existing cluster service layer and domain model so the feature does not create a parallel command or service structure.

## Post-Design Constitution Check

- `Operational proof`: Still passes. The command remains a read-only cluster retrieval path using established success and failure reporting behavior.
- `CLI compatibility`: Still passes. The design introduces only the nested `get cluster license` path and keeps flag propagation, help structure, and output conventions aligned with current `get` commands.
- `Validation`: Still passes. The design maps directly to focused command tests, existing cluster service regression coverage, and repository-wide `make test`.
- `Documentation parity`: Still passes. The design explicitly includes README and generated CLI doc updates for the new user-visible command.
- `Complexity control`: Still passes. No new dependencies, no extra command hierarchy, and no new abstraction layer are required.

## Complexity Tracking

No constitution violations or justified complexity exceptions are expected for this feature.
