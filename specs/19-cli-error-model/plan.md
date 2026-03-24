# Implementation Plan: Review and Refactor CLI Error Code Usage

**Branch**: `19-cli-error-model` | **Date**: 2026-03-24 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/19-cli-error-model/spec.md)
**Input**: Feature specification from `/specs/19-cli-error-model/spec.md`

## Summary

Define one shared CLI error classification and exit-code mapping across all existing `c8volt` commands by extending the existing `c8volt/ferrors` boundary, normalizing command, service, and domain failures into a bounded set of CLI classes, and sweeping command handlers so they all route failures through the same model. The implementation will preserve Cobra structure, existing exit-code constants where possible, and `--no-err-codes` compatibility while adding subprocess-based regression tests plus scripting-doc updates for the user-visible failure contract.

## Technical Context

**Language/Version**: Go 1.25.3  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, existing `c8volt/ferrors`, `internal/exitcode`, `internal/domain`, `internal/services`, and command packages under `cmd/`  
**Storage**: N/A  
**Testing**: `go test` with `testify`, subprocess-based CLI tests in `cmd/`, focused shared-error tests near `c8volt/ferrors`, final validation via `make test`  
**Target Platform**: Cross-platform Go CLI execution in local development, shell automation, and CI environments  
**Project Type**: Go CLI application with internal service packages  
**Performance Goals**: No user-noticeable regression in command startup or failure-path latency; classification overhead must remain negligible compared with existing config loading, HTTP setup, and command execution costs  
**Constraints**: Preserve existing command tree and flag layouts, keep `--no-err-codes` compatible, prefer current exit-code constants, avoid introducing a parallel CLI error framework, and update user-facing scripting docs when shipped behavior changes  
**Scale/Scope**: Shared error handling in `c8volt/ferrors`, exit-code definitions, command validation sentinels, root pre-run failure handling, all existing command families under `cmd/`, focused CLI and helper tests, and documentation updates in `README.md` and `docs/index.md` with conditional `docs/cli/` regeneration

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- `Operational proof`: Pass. This feature changes failure behavior, not success verification semantics. The design keeps existing commands honest about failure origin and preserves non-success reporting instead of weakening outcome proof.
- `CLI compatibility`: Pass with controlled user-visible change. Command names, subcommands, and flag propagation remain unchanged, while failure classification and exit semantics become more consistent. `--no-err-codes` stays compatible by overriding the final numeric exit result rather than bypassing classification.
- `Validation`: Pass. The plan requires focused subprocess-based CLI tests for failure paths, shared-error helper coverage where useful, and final `make test`.
- `Documentation parity`: Pass. `README.md` and `docs/index.md` will be updated for scripting and error-code behavior. `docs/cli/` regeneration is conditional on help-text changes because those pages are sourced from Cobra metadata.
- `Complexity control`: Pass. Reuse `c8volt/ferrors`, existing sentinels, existing exit codes, and the current Cobra command tree. Do not add new dependencies or a parallel framework.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/19-cli-error-model/research.md).

- Confirm the lowest-risk shared boundary for CLI failure classification and exit mapping.
- Define the bounded CLI error classes and how existing domain, service, and command sentinels should map into them.
- Confirm the testing strategy for `os.Exit`-driven failure paths and the exact user-facing documentation impact.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/19-cli-error-model/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/19-cli-error-model/quickstart.md)
- [cli-command-contract.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/19-cli-error-model/contracts/cli-command-contract.md)

The contract is required for this feature because the shared error model changes a public CLI behavior surface relied on by operators and automation.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Define the final shared CLI error classes, normalized sentinel mapping, and exit-code policy inside `c8volt/ferrors` and adjacent shared constants or helpers.
2. Update root pre-run, command validation, and command execution paths under `cmd/` so all existing command families route failures through the shared model consistently.
3. Add focused subprocess-based CLI regression tests for representative failure categories across root, get, run, deploy, cancel, delete, expect, walk, embed, and config commands, plus direct shared-error tests where they improve isolation.
4. Update `README.md` and `docs/index.md` to document the shipped failure contract, and regenerate `docs/cli/` only if Cobra help text changes.
5. Run targeted tests first, then finish with `make test`.

## Project Structure

### Documentation (this feature)

```text
specs/19-cli-error-model/
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
├── cmd_cli.go
├── cmd_errors.go
├── config_show.go
├── get*.go
├── run*.go
├── deploy*.go
├── cancel*.go
├── delete*.go
├── expect*.go
├── walk*.go
├── embed*.go
└── root.go

c8volt/
└── ferrors/
    └── errors.go

internal/
├── domain/
│   └── errors.go
├── exitcode/
│   └── exitcode.go
└── services/
    └── errors.go

README.md
docs/
├── index.md
└── cli/
```

**Structure Decision**: Use the existing single-project Go CLI layout and keep the refactor inside the current command packages, shared CLI error package, exit-code package, and adjacent documentation files. This preserves the repository’s established Cobra-first structure and avoids introducing a parallel command or error framework.

## Post-Design Constitution Check

- `Operational proof`: Still passes. The design keeps failure behavior explicit without weakening command outcome verification.
- `CLI compatibility`: Still passes. The only intended user-visible change is more consistent failure semantics; command names, flags, and command-tree structure remain intact, and `--no-err-codes` compatibility is preserved.
- `Validation`: Still passes. The design maps directly to subprocess-based CLI regression tests, focused helper tests, and final `make test`.
- `Documentation parity`: Still passes. The design includes required scripting-doc updates and conditional generated-doc regeneration.
- `Complexity control`: Still passes. The plan reuses existing repository-native packages and patterns, introduces no new dependencies, and avoids architectural churn.

## Complexity Tracking

No constitution violations or justified complexity exceptions are expected for this feature.
