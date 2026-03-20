# Implementation Plan: Add Process Definition XML Command

**Branch**: `69-process-definition-xml` | **Date**: 2026-03-20 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/69-process-definition-xml/spec.md)
**Input**: Feature specification from `/specs/69-process-definition-xml/spec.md`

## Summary

Add `c8volt get process-definition --key <id> --xml` as a focused extension of the existing process-definition retrieval flow. The implementation will expose the already-supported versioned `GetProcessDefinitionXML` service capability through the public process facade, gate the CLI flag so XML retrieval is only used in single-item mode, print the raw XML payload directly to standard output for redirect-safe workflows, and validate the change with command tests, facade/service regression coverage where needed, regenerated CLI docs, and `make test`.

## Technical Context

**Language/Version**: Go 1.25.3  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...`  
**Storage**: N/A  
**Testing**: `go test ./... -race -count=1` via `make test`, focused command tests in `cmd`, and targeted service or facade regression tests near the changed packages  
**Target Platform**: Cross-platform CLI execution in local development and CI environments supported by Go  
**Project Type**: Go CLI application  
**Performance Goals**: No observable regression for existing `get process-definition` behaviors; XML retrieval should stream directly to stdout without additional transformation or cleanup steps  
**Constraints**: Reuse the current Cobra command path, preserve existing non-XML behaviors, require explicit single-definition selection for XML mode, keep stdout safe for shell redirection, avoid new dependencies, update generated CLI docs through `make docs`, and keep validation compatible with `make test`  
**Scale/Scope**: One existing CLI command in `cmd/`, the public process facade in `c8volt/process/`, existing processdefinition services in `internal/services/processdefinition/`, adjacent command/service tests, generated docs under `docs/cli/`, and any matching README usage notes

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- `Operational proof`: Pass. This feature is a read-only retrieval workflow, so the relevant proof is accurate success/failure signaling. The command will only report success after XML retrieval succeeds and will preserve non-success exit behavior on retrieval errors.
- `CLI compatibility`: Pass. The change stays within the existing `c8volt get process-definition` command, adds a single repository-native flag, and keeps the existing default list/detail render paths unchanged when `--xml` is not used.
- `Validation`: Pass. Add or update command-level tests for XML success and failure behavior, ensure redirected-output expectations are covered, keep existing versioned service coverage in place or extend it minimally where public wiring changes, and finish with `make test`.
- `Documentation parity`: Pass. Update command help text first, regenerate `docs/cli/` with `make docs`, and update `README.md` only if process-definition retrieval examples or usage notes would otherwise become stale.
- `Complexity control`: Pass. The plan reuses the already-available service capability and existing Cobra command instead of introducing a new command tree, output subsystem, or dependency.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/69-process-definition-xml/research.md).

- Confirm the lowest-risk way to expose existing versioned XML retrieval through the public process facade.
- Confirm the CLI contract for `--xml`, especially how it interacts with `--key`, list filters, and global render flags such as `--json` and `--keys-only`.
- Confirm the safest output behavior for redirect-friendly XML export without disturbing current non-XML rendering helpers.
- Confirm the minimum documentation and validation surface for a user-visible command flag addition.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/69-process-definition-xml/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/69-process-definition-xml/quickstart.md)
- [contracts/cli-command-contract.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/69-process-definition-xml/contracts/cli-command-contract.md)

The contract artifact is included because this feature changes the public CLI interface and defines a new raw-output mode that operators will use directly in shell workflows.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Extend the public process facade interface and client implementation to expose process-definition XML retrieval through the existing service layer.
2. Add `--xml` handling to `c8volt get process-definition`, including focused flag validation that requires `--key` and rejects incompatible list or render combinations.
3. Add a raw XML output path that writes the payload directly to stdout while keeping the current list/detail rendering helpers unchanged for non-XML requests.
4. Add or update focused tests in `cmd/` for XML success, failure, and redirect-safe output behavior, plus any missing regression coverage for the public facade wiring.
5. Update command help text, regenerate CLI docs, and adjust README usage only if user-facing examples need the new XML retrieval path.
6. Run targeted Go tests during implementation, regenerate docs when help text changes, and finish with `make test`.

## Project Structure

### Documentation (this feature)

```text
specs/69-process-definition-xml/
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
├── get_processdefinition.go
├── cmd_views_get.go
└── get_test.go

c8volt/
├── client.go
└── process/
   ├── api.go
   └── client.go

internal/services/processdefinition/
├── api.go
├── factory.go
├── v87/
│  └── service_test.go
└── v88/
   └── service_test.go

docs/cli/
└── c8volt_get_process-definition.md

README.md
```

**Structure Decision**: Use the existing single-project Go CLI layout and keep changes localized to the current process-definition command, the public process facade, and the already-supported versioned service surface. The feature intentionally avoids new packages or parallel command structures.

## Post-Design Constitution Check

- `Operational proof`: Still passes. The design keeps success tied to completed XML retrieval and preserves existing error-driven exit behavior for failures.
- `CLI compatibility`: Still passes. The design adds one opt-in flag under the current command and leaves existing list/detail flows untouched unless `--xml` is explicitly requested.
- `Validation`: Still passes. The design maps to focused command tests, minimal adjacent regression coverage, generated docs refresh, and repository-wide `make test`.
- `Documentation parity`: Still passes. The design requires help-text updates first and generated CLI docs in the same change set, with README changes scoped to affected examples only.
- `Complexity control`: Still passes. The design reuses existing service methods and Cobra patterns rather than introducing a general-purpose raw-rendering framework or separate XML command.

## Complexity Tracking

No constitution violations or justified complexity exceptions are expected for this feature.
