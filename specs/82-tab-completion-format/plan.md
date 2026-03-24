# Implementation Plan: Fix Terminal Command Completion Suggestion Formatting

**Branch**: `82-tab-completion-format` | **Date**: 2026-03-24 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/82-tab-completion-format/spec.md)
**Input**: Feature specification from `/specs/82-tab-completion-format/spec.md`

## Summary

Fix the malformed terminal Tab-completion output by correcting `c8volt` command and completion metadata rather than changing Cobra itself. The implementation should keep the existing root command tree intact, prevent internal completion helper entries or help/usage dumps from surfacing as normal suggestions, preserve concise candidate descriptions when available, and prove the behavior with focused command-level completion regression coverage for one top-level path, one nested path, and one flag-completion path before running `make test`.

## Technical Context

**Language/Version**: Go 1.25.3  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, existing command helpers under `cmd/`  
**Storage**: N/A  
**Testing**: focused `go test` coverage in `./cmd/...` for root/help/completion behavior, then `make test`; regenerate CLI docs with `make docs` if user-visible help text changes  
**Target Platform**: Cross-platform Go CLI used in interactive shells and automation, with zsh as the reported regression example  
**Project Type**: Go CLI application using Cobra command metadata and generated CLI docs  
**Performance Goals**: No meaningful latency regression for interactive completion; completion requests should remain single-command metadata evaluations with no additional network or configuration-dependent work  
**Constraints**: Keep Cobra framework behavior unchanged, limit the fix to `c8volt` command/completion metadata and integration points, preserve the existing command hierarchy, avoid introducing a custom completion subsystem, keep utility/internal completion helpers out of normal suggestion lists, preserve concise descriptions when available, and update generated docs only through `make docs`  
**Scale/Scope**: Root command metadata in `cmd/root.go`, adjacent command registration/helpers under `cmd/`, command tests in `cmd/*_test.go`, generated CLI docs under `docs/cli/`, and README review only if current user guidance would otherwise become misleading

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- `Operational proof`: Pass. This is a CLI usability defect, so the required proof is clean, context-appropriate completion output for representative command paths rather than a hidden metadata change with no observable check.
- `CLI compatibility`: Pass. The feature preserves the existing Cobra command tree and user-facing command names while narrowing the change to `c8volt` metadata and completion integration points.
- `Validation`: Pass. The plan requires focused command-level regression tests covering one top-level path, one nested path, and one flag-completion path, followed by `make test`.
- `Documentation parity`: Pass. If help text or command visibility changes for users, the plan includes regenerating `docs/cli/` with `make docs`; `README.md` only changes if current completion guidance would otherwise be inaccurate.
- `Complexity control`: Pass. The design stays repository-native by reusing current root-command and command-metadata patterns instead of adding a parallel completion implementation or modifying Cobra.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/82-tab-completion-format/research.md).

- Confirm the lowest-risk repository-native seam for completion fixes in `c8volt` command metadata and root-command integration.
- Confirm how to keep internal completion helpers and usage-style output out of normal interactive suggestion lists without changing Cobra itself.
- Confirm the appropriate regression-test surface for completion behavior in this repository, including helper-process versus in-process command tests.
- Confirm the documentation impact for any user-visible help or completion-command behavior changes.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/82-tab-completion-format/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/82-tab-completion-format/quickstart.md)
- [contracts/cli-command-contract.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/82-tab-completion-format/contracts/cli-command-contract.md)

The contract artifact is required because this feature changes public CLI completion behavior that users experience directly in interactive shells, even though the implementation remains inside the existing Cobra-based command surface.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Identify and update the root-command or adjacent command metadata that currently causes malformed or overly verbose completion suggestions, while preserving the existing Cobra framework behavior.
2. Adjust the relevant command/completion integration so user-facing completion candidates remain visible but internal helper entries and usage/help dumps do not appear as normal suggestions.
3. Preserve concise completion descriptions where `c8volt` already provides them and ensure suggestions without descriptions still render cleanly.
4. Add focused regression tests in `cmd/` for one representative top-level path, one nested command path, and one flag-completion path, using the repository’s current root-command test patterns.
5. Update any affected help text first, regenerate CLI docs with `make docs` if user-visible command help changed, and review `README.md` only if current completion guidance becomes stale.
6. Run focused Go tests during implementation and finish with `make test`.

## Project Structure

### Documentation (this feature)

```text
specs/82-tab-completion-format/
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
├── root.go
├── cmd_checks.go
├── cmd_cli.go
├── get.go
├── delete.go
├── walk_processinstance.go
├── get_test.go
└── mutation_test.go

docs/cli/
└── c8volt.md

README.md
```

**Structure Decision**: Use the existing single-project Go CLI layout and keep the change localized to root-command metadata, adjacent command/completion integration points, and command-level regression tests. The feature intentionally avoids new packages, new dependencies, or a separate completion subsystem.

## Post-Design Constitution Check

- `Operational proof`: Still passes. The design defines success as clean completion suggestions for representative interactive flows and failure as continued noisy, internal, or malformed suggestion output.
- `CLI compatibility`: Still passes. The design keeps `c8volt` inside the current Cobra command hierarchy and explicitly avoids framework changes.
- `Validation`: Still passes. The design maps to focused `cmd/` regression tests plus final repository-wide `make test`.
- `Documentation parity`: Still passes. The design limits documentation work to command help and generated CLI docs if the public help surface changes, with README review remaining explicit.
- `Complexity control`: Still passes. The change remains metadata- and integration-focused, with no parallel completion layer and no new abstractions unless implementation reveals a narrowly scoped helper is necessary.

## Complexity Tracking

No constitution violations or justified complexity exceptions are expected for this feature.
