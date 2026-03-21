# Implementation Plan: Review and Refactor Internal Service Resource API Implementation

**Branch**: `71-resource-api-refactor` | **Date**: 2026-03-21 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/spec.md)
**Input**: Feature specification from `/specs/71-resource-api-refactor/spec.md`

## Summary

Refactor the internal resource service to reduce duplication between supported Camunda versions, preserve the current deploy and delete behavior, and review generated-client coverage for a bounded missing capability that fits the existing service surface. The implementation will stay within the current API/factory/versioned-service structure, prefer small shared helpers such as repository-native payload validation over new abstractions, and validate the result with focused service and factory tests before `make test`.

## Technical Context

**Language/Version**: Go 1.25.3  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...`, existing helpers in `internal/services/common`  
**Storage**: N/A  
**Testing**: `go test ./... -race -count=1` via `make test`, focused `testify` unit tests in `internal/services/resource`, existing fake-server-backed tests where resource behavior overlaps deployment visibility flows  
**Target Platform**: Cross-platform CLI and library use in local development and CI environments supported by Go  
**Project Type**: Go CLI application with internal service packages  
**Performance Goals**: No observable regression in resource deployment, deletion, or follow-up confirmation behavior; any additional capability must match the latency and result-shape expectations of current service calls  
**Constraints**: No package renames, no package layout changes, no crucial structural redesign, preserve current external behavior, prefer low-risk shared helpers, and only add a missing capability if it exists across supported versions and fits the current service surface  
**Scale/Scope**: One internal service area (`internal/services/resource`) plus adjacent tests, shared response handling, and conditional documentation updates if a new user-visible capability is surfaced

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- `Operational proof`: Pass. The current resource workflows already return concrete deployment results or explicit delete outcomes, and the v88 deployment path includes poll-based confirmation when `--no-wait` is not set. The refactor preserves those observable completion semantics, and any added capability must expose equally clear success and failure behavior.
- `CLI compatibility`: Pass. No command names, flags, Cobra wiring, output format, or exit-code changes are planned unless a narrow resource capability is intentionally surfaced. If the work stays internal-only, CLI behavior remains unchanged.
- `Validation`: Pass. Update `internal/services/resource/v88/service_test.go`, add missing focused coverage for `internal/services/resource/v87/service.go`, keep `internal/services/resource/factory_test.go` aligned with concrete version routing, and finish with `make test`.
- `Documentation parity`: Pass with conditional update. If the feature remains internal-only, `README.md` and generated CLI docs do not change. If a new user-visible resource workflow or output path is introduced, update `README.md` and regenerate CLI docs with `make docs`.
- `Complexity control`: Pass. Reuse the repository-native service/factory/contract layout, generated clients, and `internal/services/common.RequirePayload` where it fits. Do not add dependencies. Prefer localized helper extraction or contract expansion over new cross-package abstraction layers.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/research.md).

- Confirm the lowest-risk refactor shape for duplicated v87/v88 resource services.
- Review generated resource client capabilities for both supported versions and decide whether resource lookup or content retrieval is the right bounded missing capability to expose, or whether the review should remain informational only.
- Confirm the required validation and documentation surface based on whether the final change stays internal-only or introduces a user-visible capability.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/quickstart.md)

Contracts directory is intentionally omitted for now because this feature is planned as an internal service refactor with no guaranteed external interface change. If the generated-client coverage review results in a user-visible resource capability, create a contract artifact during implementation planning before tasks are generated.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Review the supported generated resource clients and confirm the exact shared service surface to preserve or extend.
2. Refactor shared deployment multipart construction, payload validation, and delete-path handling with no behavioral change.
3. Add one approved low-risk missing capability only if it is supported across v87 and v88 and fits the current API shape.
4. Update focused unit tests for both supported versions and factory behavior.
5. Update documentation only if a new user-visible workflow or output path is introduced.
6. Run targeted Go tests, regenerate docs if needed, and finish with `make test`.

## Project Structure

### Documentation (this feature)

```text
specs/71-resource-api-refactor/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── deploy_processdefinition.go
├── delete_processdefinition.go
└── root.go

internal/domain/
└── deployment.go

internal/services/resource/
├── api.go
├── factory.go
├── factory_test.go
├── v87/
│   ├── contract.go
│   ├── convert.go
│   └── service.go
└── v88/
    ├── contract.go
    ├── convert.go
    ├── service.go
    └── service_test.go

internal/services/common/
└── response.go

internal/clients/camunda/
├── v87/camunda/client.gen.go
└── v88/camunda/client.gen.go

testx/
├── fake_server.go
├── integration87/processdefinition_test.go
└── integration88/processdefinition_test.go

README.md
docs/cli/
```

**Structure Decision**: Use the existing single-project Go CLI layout and keep all work inside the current resource service packages, adjacent tests, and conditional documentation files. This preserves the repository’s established factory-plus-versioned-service structure and avoids parallel abstractions.

## Post-Design Constitution Check

- `Operational proof`: Still passes. The design preserves current deployment and deletion verification semantics, including the v88 confirmation poll when waiting is enabled, and requires any new capability to define equally observable success and failure behavior.
- `CLI compatibility`: Still passes. No command-tree or flag changes are planned unless a bounded new resource capability is intentionally surfaced.
- `Validation`: Still passes. The design maps directly to existing v88 service tests, new or expanded v87 service tests, factory tests, and repository-wide `make test`.
- `Documentation parity`: Still passes. Documentation remains unchanged unless the generated-client review results in a user-visible addition.
- `Complexity control`: Still passes. No new dependencies, no package moves, and no non-native abstractions are introduced by the design.

## Complexity Tracking

No constitution violations or justified complexity exceptions are expected for this feature.
