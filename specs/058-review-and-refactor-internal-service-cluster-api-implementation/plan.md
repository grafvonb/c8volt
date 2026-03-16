# Implementation Plan: Review and Refactor Cluster Service

**Branch**: `058-review-and-refactor-internal-service-cluster-api-implementation` | **Date**: 2026-03-16 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/058-review-and-refactor-internal-service-cluster-api-implementation/spec.md)
**Input**: Feature specification from `/specs/058-review-and-refactor-internal-service-cluster-api-implementation/spec.md`

## Summary

Refactor the internal cluster service to reduce duplication between supported Camunda versions, preserve existing topology behavior, and explicitly review generated-client coverage for a low-risk missing capability. The implementation will stay within the current Go service/factory pattern, prefer small shared helpers over new abstractions, and validate behavior with focused service, factory, and integration tests before `make test`.

## Technical Context

**Language/Version**: Go 1.25.3  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...`  
**Storage**: N/A  
**Testing**: `go test ./... -race -count=1` via `make test`, existing `testify` unit tests, fake-server integration tests in `testx`  
**Target Platform**: Cross-platform CLI and library use on developer and CI environments supported by Go  
**Project Type**: Go CLI application with internal service packages  
**Performance Goals**: No observable regression for current cluster topology workflows; internal refactor should keep request/response behavior equivalent to current service calls  
**Constraints**: No package renames, no package layout changes, no crucial structural redesign, no behavioral regressions beyond an intentional small missing capability if one is added  
**Scale/Scope**: One internal service area (`internal/services/cluster`) plus adjacent tests and documentation for any user-visible change

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- `Operational proof`: Pass. The current cluster workflow already verifies the requested topology fetch through returned service results; the refactor preserves this behavior. Any added capability must keep the same observable proof model or explicitly document why verification is not applicable.
- `CLI compatibility`: Pass. No command names, flags, Cobra wiring, output format, or exit-code changes are planned unless a narrow new cluster capability is surfaced intentionally. If no CLI changes are introduced, the plan remains internal-only.
- `Validation`: Pass. Update `internal/services/cluster/v87/service_test.go`, `internal/services/cluster/v88/service_test.go`, `internal/services/cluster/factory_test.go`, and any relevant integration coverage in `testx/integration87/cluster_test.go` or `testx/integration88/cluster_test.go`. Run `make test` before completion.
- `Documentation parity`: Pass with conditional update. If the feature remains internal-only, `README.md` and generated CLI docs do not change. If a user-visible cluster capability is added, update `README.md` and regenerate CLI docs with `make docs`.
- `Complexity control`: Pass. Reuse the repository-native service/factory/contract structure and existing generated clients. Do not add dependencies. Prefer localized helper extraction or shared normalization patterns over new cross-package abstraction layers.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/058-review-and-refactor-internal-service-cluster-api-implementation/research.md).

- Confirm the lowest-risk refactor shape for duplicated v87/v88 cluster services.
- Review generated cluster client capabilities for supported versions and decide whether any missing service method is worth adding within the issue constraints.
- Confirm the required validation surface and documentation impact based on whether the final change is internal-only or user-visible.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/058-review-and-refactor-internal-service-cluster-api-implementation/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/058-review-and-refactor-internal-service-cluster-api-implementation/quickstart.md)

Contracts directory is intentionally omitted for now because the feature is planned as an internal service refactor with no guaranteed new external interface. If Phase 0 identifies a user-visible addition, create a contract artifact during implementation planning before tasks are generated.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Analyze generated cluster client coverage and confirm the exact service surface to preserve or extend.
2. Refactor shared cluster service construction and response normalization with no behavior change.
3. Add any approved low-risk missing capability using the existing versioned service pattern.
4. Update focused unit and integration tests.
5. Update documentation only if user-visible behavior changes.
6. Run `make test`, and run `make docs` if CLI docs changed.

## Project Structure

### Documentation (this feature)

```text
specs/058-review-and-refactor-internal-service-cluster-api-implementation/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_cluster_topology.go
├── get.go
└── root.go

internal/domain/
└── cluster.go

internal/services/cluster/
├── api.go
├── factory.go
├── factory_test.go
├── v87/
│   ├── contract.go
│   ├── convert.go
│   ├── service.go
│   └── service_test.go
└── v88/
    ├── contract.go
    ├── convert.go
    ├── service.go
    └── service_test.go

internal/services/common/
└── *.go

internal/clients/camunda/
├── v87/camunda/client.gen.go
└── v88/camunda/client.gen.go

testx/
├── fake_server.go
├── integration87/cluster_test.go
└── integration88/cluster_test.go

README.md
docs/cli/
```

**Structure Decision**: Use the existing single-project Go CLI layout and keep all work inside the current cluster service packages, adjacent tests, and conditional documentation files. This preserves the repository’s established factory-plus-versioned-service structure and avoids parallel abstractions.

## Post-Design Constitution Check

- `Operational proof`: Still passes. Design preserves current cluster-topology verification semantics and requires any new capability to define equivalent observable completion behavior.
- `CLI compatibility`: Still passes. No planned command-tree break; any new exposure must follow the existing `get` command conventions and Cobra wiring.
- `Validation`: Still passes. The design maps directly to existing unit, factory, and integration test locations and ends with `make test`.
- `Documentation parity`: Still passes. Documentation remains unchanged unless a new user-visible cluster command or output path is introduced.
- `Complexity control`: Still passes. No new dependencies, packages, or cross-cutting abstractions are introduced by the design.

## Complexity Tracking

No constitution violations or justified complexity exceptions are expected for this feature.
