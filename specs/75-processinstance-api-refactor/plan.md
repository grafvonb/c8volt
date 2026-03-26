# Implementation Plan: Review and Refactor Internal Service Processinstance API Implementation

**Branch**: `75-processinstance-api-refactor` | **Date**: 2026-03-23 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/spec.md)
**Input**: Feature specification from `/specs/75-processinstance-api-refactor/spec.md`

## Summary

Refactor the internal processinstance service to reduce duplication across supported Camunda versions, preserve current create/get/search/cancel/delete/wait/walk behavior, and review generated-client coverage for one bounded missing capability that fits the existing service surface. The implementation will stay within the current API/factory/versioned-service/helper structure, prefer repository-native shared helpers over new abstraction layers, and validate the result with focused service, waiter, walker, factory, and adjacent command or integration tests before `make test`.

## Technical Context

**Language/Version**: Go 1.25.3  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...`, existing helpers in `internal/services/common`, worker utilities in `toolx/pool`  
**Storage**: N/A  
**Testing**: `go test ./... -race -count=1` via `make test`, focused `testify` unit tests in `internal/services/processinstance`, helper-level tests for waiter or walker if touched, adjacent command or fake-server-backed tests if user-visible behavior changes  
**Target Platform**: Cross-platform CLI and library use in local development and CI environments supported by Go  
**Project Type**: Go CLI application with internal service packages  
**Performance Goals**: No observable regression in process instance creation, lookup, traversal, cancellation, deletion, or wait-for-state workflows; bulk operations must preserve current worker-parallelism behavior and no-wait semantics  
**Constraints**: No package renames, no package layout changes, no crucial structural redesign, preserve current external behavior, prefer low-risk shared helpers, and allow a missing capability addition only when it fits the current API shape and defines explicit unsupported-version behavior when version coverage differs  
**Scale/Scope**: One internal service area (`internal/services/processinstance`) plus its helper packages (`waiter`, `walker`), adjacent tests, existing process-instance CLI commands, and conditional documentation updates only if user-visible behavior changes

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- `Operational proof`: Pass. Current processinstance workflows already verify creation, cancellation, deletion, and expectation behavior through returned state, polling, and tree traversal. The refactor preserves those observable completion semantics, and any added capability must define equally clear success, failure, or unsupported-version outcomes.
- `CLI compatibility`: Pass. No command names, flags, Cobra wiring, output format, or exit-code changes are planned unless a bounded processinstance capability is intentionally surfaced beyond the current internal service surface. If the work stays internal-only, CLI behavior remains unchanged.
- `Validation`: Pass. Update `internal/services/processinstance/factory_test.go`, add or expand versioned service tests for v87 and v88, add helper-level coverage for waiter or walker behavior if refactored, keep adjacent command or integration coverage aligned where processinstance semantics are already exercised, and finish with `make test`.
- `Documentation parity`: Pass with conditional update. If the feature remains internal-only, `README.md` and generated CLI docs do not change. If a new user-visible processinstance workflow, output, or unsupported-version contract appears, update `README.md` and regenerate CLI docs with `make docs`.
- `Complexity control`: Pass. Reuse the repository-native service/factory/contract/helper layout, generated clients, and `internal/services/common` helpers where they fit. Do not add dependencies. Prefer localized helper extraction, type aliases, and shared response-validation patterns over new cross-package abstraction layers.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/research.md).

- Confirm the lowest-risk refactor shape for duplicated v87 and v88 processinstance services while keeping waiter and walker responsibilities readable.
- Review generated processinstance client capabilities for both supported versions and decide whether one missing capability is worth exposing, including how unsupported-version behavior should work if only one version supports it cleanly.
- Confirm the required validation and documentation surface based on whether the final change stays internal-only or introduces a user-visible capability.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/quickstart.md)

Contracts directory is intentionally omitted for now because this feature is planned as an internal service refactor with no guaranteed external interface change. If the generated-client coverage review results in a user-visible processinstance capability or new CLI-facing contract, create a contract artifact during implementation planning before tasks are generated.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Review the supported generated processinstance clients and confirm the exact shared service surface to preserve or extend.
2. Refactor shared construction, response validation, and duplicated service control flow only where the resulting code is clearer and lower risk than the current version-specific copies.
3. Simplify waiter and walker interactions without changing current cancellation, deletion, ancestry, descendant, or family semantics.
4. Add one approved low-risk missing capability only if it fits the current API shape and has explicit unsupported-version behavior where support differs.
5. Update focused versioned service, helper, and factory tests to cover preserved behavior and any intentional capability addition.
6. Update documentation only if a new user-visible workflow, output path, or supported behavior contract appears.
7. Run targeted Go tests, regenerate docs if needed, and finish with `make test`.

## Project Structure

### Documentation (this feature)

```text
specs/75-processinstance-api-refactor/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── cancel_processinstance.go
├── delete_processinstance.go
├── expect_processinstance.go
├── get_processinstance.go
├── run_processinstance.go
├── walk_processinstance.go
└── root.go

internal/domain/
├── processinstance.go
└── state.go

internal/services/processinstance/
├── api.go
├── factory.go
├── factory_test.go
├── waiter/
│   └── waiter.go
├── walker/
│   └── walker.go
├── v87/
│   ├── bulk.go
│   ├── contract.go
│   ├── convert.go
│   └── service.go
└── v88/
    ├── bulk.go
    ├── contract.go
    ├── convert.go
    └── service.go

internal/services/common/
└── *.go

internal/clients/camunda/
├── v87/camunda/client.gen.go
├── v87/operate/client.gen.go
├── v88/camunda/client.gen.go
└── v88/operate/client.gen.go

testx/
└── ...

README.md
docs/cli/
```

**Structure Decision**: Use the existing single-project Go CLI layout and keep all work inside the current processinstance service packages, helper packages, adjacent tests, and conditional documentation files. This preserves the repository’s established factory-plus-versioned-service structure and avoids parallel abstractions.

## Post-Design Constitution Check

- `Operational proof`: Still passes. The design preserves current create, wait, cancel, delete, and traversal verification semantics, including polling-backed confirmation and family-tree handling.
- `CLI compatibility`: Still passes. No command-tree or flag changes are planned unless a bounded processinstance capability is intentionally exposed beyond the internal service layer.
- `Validation`: Still passes. The design maps directly to factory tests, new or expanded v87 and v88 service tests, helper-level waiter or walker coverage if touched, and repository-wide `make test`.
- `Documentation parity`: Still passes. Documentation remains unchanged unless the generated-client review results in a user-visible processinstance addition or output change.
- `Complexity control`: Still passes. No new dependencies, no package moves, and no non-native abstraction layers are introduced by the design.

## Complexity Tracking

No constitution violations or justified complexity exceptions are expected for this feature.
