# Implementation Plan: Add Resource Get Command By Id

**Branch**: `73-get-resource-id` | **Date**: 2026-03-21 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/73-get-resource-id/spec.md)
**Input**: Feature specification from `/specs/73-get-resource-id/spec.md`

## Summary

Add a new `c8volt get resource --id <id>` command that retrieves a single resource through the existing internal resource service capability introduced by issue `#71`, returns the normal single-resource object/details view, and preserves the repository’s established Cobra help, validation, rendering, and error-handling patterns. The implementation will stay repository-native by extending the existing `c8volt/resource` facade, adding one `get` subcommand plus a dedicated resource view helper, keeping empty-success responses as malformed-response errors via the current shared payload validation behavior, and validating the result with targeted command and facade/service tests before `make test`.

## Technical Context

**Language/Version**: Go 1.25.3  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...`, existing helpers in `internal/services/common`, existing facade packages under `c8volt/...`  
**Storage**: N/A  
**Testing**: focused `go test` for `./cmd/...`, `./c8volt/resource/...`, and `./internal/services/resource/...`, then `make test`; regenerate CLI docs with `make docs` if command help changes land  
**Target Platform**: Cross-platform Go CLI used interactively and in automation on developer machines and CI  
**Project Type**: Go CLI application with facade packages over internal versioned services  
**Performance Goals**: Match the latency and output expectations of existing single-item `get` commands with no extra polling or bulk fan-out; add no new network round trips beyond the single resource lookup  
**Constraints**: Preserve current command-tree conventions, require `--id` validation before lookup, keep raw resource-content retrieval out of scope, reuse existing domain/resource conversion paths, avoid package layout changes, and keep malformed success payloads as errors rather than empty success  
**Scale/Scope**: One new CLI path, one facade capability addition in `c8volt/resource`, shared resource domain/model wiring, command rendering helpers, tests, and user-facing documentation generated from Cobra metadata

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- `Operational proof`: Pass. The command is a read-only lookup, so observable completion is the returned single resource object or a clear failure path. No implicit success-without-payload behavior is allowed.
- `CLI compatibility`: Pass. The feature extends the existing `get` tree with a repository-native Cobra subcommand, preserves flag-driven behavior, and uses the same failure/exit handling style as current commands.
- `Validation`: Pass. The design includes targeted tests for command validation and success rendering, facade wiring, and versioned service error normalization, followed by `make test`.
- `Documentation parity`: Pass. Because this is user-visible CLI behavior, command help and generated `docs/cli/` output must be updated in the same change. `README.md` should be reviewed and updated if it references supported `get` workflows in a way that would now be incomplete.
- `Complexity control`: Pass. The plan reuses the current facade/service/view split, adds no dependencies, and avoids broader resource-command redesign or raw-content export behavior.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/73-get-resource-id/research.md).

- Confirm the repository-native command shape for a single-item `get resource` workflow.
- Resolve the deferred clarification about `200 OK` responses with no JSON payload by aligning the plan to existing service behavior.
- Confirm where facade, model, and rendering changes belong so this capability fits the current `c8volt` layering.
- Confirm the documentation and validation surface for a new CLI-visible `get` command.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/73-get-resource-id/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/73-get-resource-id/quickstart.md)
- [contracts/cli-command-contract.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/73-get-resource-id/contracts/cli-command-contract.md)

The contract is required because this feature introduces a new public CLI command path whose flags, output behavior, and documentation surface must remain stable and testable.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Extend the `c8volt/resource` facade API and models to expose single-resource retrieval without changing existing resource workflows.
2. Add the `get resource` Cobra command, required `--id` flag validation, and a normal single-resource view helper consistent with other `get` commands.
3. Wire the command through the existing CLI/facade/service stack and preserve malformed-response error handling from the underlying services.
4. Add or update focused tests for command validation, success rendering, facade mapping, and version-specific service response handling.
5. Update command help text and regenerate CLI docs with `make docs`; update `README.md` only if its user-facing guidance would otherwise be stale.
6. Run targeted Go tests, regenerate docs, and finish with `make test`.

## Project Structure

### Documentation (this feature)

```text
specs/73-get-resource-id/
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
├── get_resource.go
├── get_test.go
└── cmd_views_get.go

c8volt/
├── contract.go
└── resource/
    ├── api.go
    ├── client.go
    └── model.go

internal/domain/
└── resource.go

internal/services/resource/
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

README.md
docs/cli/
```

**Structure Decision**: Use the existing single-project Go CLI layout and implement the feature by extending the current `get` command tree, the `c8volt/resource` facade, existing resource domain/model types, and adjacent tests. This keeps the change small and compatible with the repository’s current layering.

## Post-Design Constitution Check

- `Operational proof`: Still passes. The design defines success as a returned single-resource object and failure as validation, not-found, transport, or malformed-response errors, with no ambiguous empty-success behavior.
- `CLI compatibility`: Still passes. The command remains inside the existing `get` hierarchy, uses a required flag instead of bespoke positional parsing, and follows existing help and exit-code patterns.
- `Validation`: Still passes. The design maps to targeted command/facade/service tests plus final `make test`.
- `Documentation parity`: Still passes. The plan explicitly includes command help updates and generated CLI docs regeneration in the same change.
- `Complexity control`: Still passes. No new dependencies, no package moves, and no raw-content workflow are introduced.

## Complexity Tracking

No constitution violations or justified complexity exceptions are expected for this feature.
