# Implementation Plan: Define Machine-Readable CLI Contracts

**Branch**: `078-machine-cli-contracts` | **Date**: 2026-04-17 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/spec.md)
**Input**: Feature specification from `/specs/078-machine-cli-contracts/spec.md`

## Summary

Add one dedicated top-level machine-readable discovery command for `c8volt`, define a shared JSON result envelope for supported command families, and document which commands fully support, partially support, or do not yet support that contract. The design keeps the existing Cobra command tree, preserves current exit-code semantics through `c8volt/ferrors`, reuses current JSON view models as the domain payload inside the new envelope, and adds representative contract coverage plus user-facing automation documentation.

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, existing `c8volt/ferrors`, existing JSON rendering helpers under `cmd/`, existing public models under `c8volt/process` and `c8volt/resource`  
**Storage**: N/A  
**Testing**: `go test`, `make test`, command subprocess and in-process tests under `cmd/`, focused error-model tests under `c8volt/ferrors`, documentation regeneration via `make docs` and `make docs-content` when user-facing help or README content changes  
**Target Platform**: Cross-platform Go CLI execution in local shells, non-interactive automation, and CI  
**Project Type**: CLI  
**Performance Goals**: Keep discovery-command startup and JSON contract rendering negligible relative to existing command execution; avoid adding extra remote calls for discovery; preserve current command-family execution behavior while making machine parsing deterministic  
**Constraints**: Preserve the current Cobra command taxonomy, keep existing exit-code behavior authoritative, reuse existing JSON payload models instead of inventing parallel domain objects, report unsupported or limited contract support honestly in discovery, keep `--json` behavior repository-native, update README and generated CLI docs for new machine-facing behavior, and finish with `make test`  
**Scale/Scope**: Root command metadata in [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go), shared render-mode helpers in [`cmd/cmd_views_rendermode.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_rendermode.go), representative command families under `cmd/get_*`, `cmd/run_processinstance.go`, `cmd/expect_processinstance.go`, `cmd/walk_processinstance.go`, `cmd/deploy_processdefinition.go`, `cmd/delete_*`, and `cmd/cancel_*`, plus shared failure classes in [`c8volt/ferrors/errors.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors.go), README automation guidance, and generated docs under `docs/cli/`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. The feature makes machine-readable success and accepted-work states explicit without weakening current command verification semantics, especially for `--no-wait` flows where request acceptance must remain distinguishable from confirmed completion.
- **CLI-First, Script-Safe Interfaces**: Pass. The design adds a top-level CLI discovery command, preserves the existing command tree and flags, keeps exit codes stable, and formalizes structured output instead of replacing the current CLI.
- **Tests and Validation Are Mandatory**: Pass. The plan requires representative contract coverage for each in-scope command family, focused error/exit-code checks where appropriate, documentation regeneration when help text changes, and final `make test`.
- **Documentation Matches User Behavior**: Pass. The feature is user-visible for automation consumers, so README guidance and generated CLI docs must be updated in the same unit of work.
- **Small, Compatible, Repository-Native Changes**: Pass. The design reuses existing Cobra metadata, render helpers, and public JSON models rather than introducing a second command tree or a parallel automation subsystem.

## Project Structure

### Documentation (this feature)

```text
specs/078-machine-cli-contracts/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-machine-contract.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── root.go
├── version.go
├── cmd_views_rendermode.go
├── cmd_views_get.go
├── cmd_views_walk.go
├── get_*.go
├── run_processinstance.go
├── expect_processinstance.go
├── walk_processinstance.go
├── deploy_processdefinition.go
├── delete_*.go
├── cancel_*.go
└── *_test.go

c8volt/
├── ferrors/
│   ├── errors.go
│   └── errors_test.go
├── process/
│   └── model.go
└── resource/
    └── model.go

docs/
├── cli/
│   └── generated Cobra reference pages
├── index.md
└── use-cases.md

README.md
```

**Structure Decision**: Keep the work inside the existing single-project Go CLI layout. The new discovery surface belongs in `cmd/` as a normal top-level Cobra command, the shared machine-readable envelope should be built around the current command-family output models rather than replacing them, and documentation stays in the existing README plus generated `docs/cli/` flow.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/research.md).

- Confirm the dedicated top-level discovery command shape and the minimum command metadata it must expose.
- Decide how the shared result envelope layers over the current JSON payload models without breaking the existing exit-code contract.
- Resolve how current `ferrors.Class` values and `--no-wait` flows map into the four clarified outcome categories.
- Define the contract-support states for commands that are currently full, limited, or unsupported for the shared machine contract.
- Confirm the smallest repository-native extension points for implementation: Cobra command metadata, existing render helpers, shared error classes, representative command tests, and documentation generation paths.

### Refined Implementation Boundary

| Area | Planned responsibility | Key files |
|--------|------------------------|-----------|
| Discovery surface | Expose machine-readable command metadata for top-level and nested commands, flags, output modes, state-changing/read-only behavior, and contract support status | `cmd/root.go`, new top-level command under `cmd/`, selected command definitions under `cmd/` |
| Shared result envelope | Wrap supported command-family JSON responses in one common machine-readable envelope while preserving command-specific payloads | `cmd/cmd_views_rendermode.go`, selected `cmd/*` handlers, public models under `c8volt/process` and `c8volt/resource` |
| Outcome and exit alignment | Preserve `ferrors`-driven exit codes and align envelope outcomes with the current failure model and `--no-wait` semantics | `c8volt/ferrors/errors.go`, state-changing command handlers under `cmd/` |
| Representative contract coverage | Prove discovery, structured result behavior, accepted-work behavior, invalid input, remote failures, and unsupported/limited reporting for each family | `cmd/*_test.go`, `c8volt/ferrors/errors_test.go` |
| Documentation | Explain the recommended automation contract and regenerate CLI docs from command help text | `README.md`, `docs/index.md`, `docs/use-cases.md`, `docs/cli/` |

The implementation boundary for this feature is intentionally limited to CLI-facing metadata, structured output rendering, representative command-family wiring, and user-facing automation documentation. It does not add new Camunda operations or replace the current human-oriented command structure.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/quickstart.md)
- [contracts/cli-machine-contract.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/078-machine-cli-contracts/contracts/cli-machine-contract.md)

- Add one dedicated top-level discovery command instead of overloading `help` or creating only per-command schema flags.
- Reuse current public JSON payload models as the `payload` portion of a shared machine-readable result envelope instead of replacing existing domain types.
- Preserve the existing process exit code as the authoritative coarse-grained signal; the JSON envelope carries detailed machine-readable outcome information that must align with it.
- Use a command capability record with explicit contract support status so unsupported or limited commands remain visible in discovery.
- Treat `accepted` as the standard envelope outcome for state-changing flows where `--no-wait` or equivalent behavior returns before confirmation, while `succeeded` remains reserved for confirmed completion.
- Keep the implementation repository-native by centralizing contract definitions and helper behavior in `cmd/` and `c8volt/ferrors` rather than building a new parallel abstraction layer.

### Authoritative Contract Boundary

| Concern | Required design contract |
|--------|---------------------------|
| Discovery entry point | One dedicated top-level command such as `c8volt capabilities --json` |
| Discovery coverage | Include top-level and nested command paths, flags, output modes, read-only/state-changing classification, and contract support status |
| Result shape | One shared top-level envelope with a command-specific payload |
| Outcome vocabulary | Only `succeeded`, `accepted`, `invalid`, and `failed` |
| Exit behavior | Existing exit codes remain authoritative and must not contradict the envelope |
| Unsupported commands | Stay visible in discovery and are marked `limited` or `unsupported` instead of being hidden |
| Documentation | README and generated CLI docs must explain the machine contract where user-visible behavior changes |
| Test strategy | One representative contract regression per in-scope command family plus focused exit/error-alignment tests |

### Representative Command-Family Targets

| Family | Existing command seams | Contract focus |
|--------|------------------------|----------------|
| `get` | `cmd/get_processinstance.go`, `cmd/get_resource.go`, `cmd/get_processdefinition.go`, cluster get commands | confirmed `succeeded` envelope, discovery metadata, limited-vs-full reporting for output modes |
| `run` | `cmd/run_processinstance.go` | `accepted` versus `succeeded`, command-specific payload wrapping, exit alignment |
| `expect` | `cmd/expect_processinstance.go` | confirmed success and failure envelope behavior for wait-style commands |
| `walk` | `cmd/walk_processinstance.go`, `cmd/cmd_views_walk.go` | structured tree/list outputs and discovery metadata |
| `deploy` | `cmd/deploy_processdefinition.go`, `cmd/embed_deploy.go` | accepted-work semantics for `--no-wait`, deployment payload wrapping |
| `delete` | `cmd/delete_processinstance.go`, `cmd/delete_processdefinition.go` | report payload wrapping, accepted-work semantics where applicable |
| `cancel` | `cmd/cancel_processinstance.go` | report payload wrapping, accepted-work semantics where applicable |

This target set is the planning boundary for task generation. Additional commands may be documented in discovery as limited or unsupported, but these families anchor the initial contract and regression suite for this feature.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Add the dedicated top-level discovery command and define the capability record shape, including contract support status and current output-mode metadata.
2. Introduce the shared result envelope helper and wire it into one representative command from each in-scope family while reusing existing public payload models.
3. Align envelope outcomes with `ferrors` classes, exit codes, and `--no-wait` command behavior without changing current process-level semantics.
4. Add or update representative tests for discovery, `succeeded`, `accepted`, `invalid`, and `failed` scenarios across the command families in scope.
5. Update README and command help text to document the machine contract, regenerate `docs/cli/` with `make docs`, regenerate docs homepage content with `make docs-content` if README changes, and finish with `make test`.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design keeps confirmation semantics explicit and only adds a structured way to tell confirmed completion apart from accepted-but-not-yet-confirmed work.
- **CLI-First, Script-Safe Interfaces**: Still passes. The design is entirely CLI-native: a new top-level command, explicit flags/output metadata, preserved exit codes, and deterministic JSON for automation.
- **Tests and Validation Are Mandatory**: Still passes. Representative family coverage plus final `make test` remain required.
- **Documentation Matches User Behavior**: Still passes. The feature requires explicit README and generated CLI doc updates because it changes the recommended automation contract and adds a new user-visible command.
- **Small, Compatible, Repository-Native Changes**: Still passes. The design reuses existing command files, render helpers, public models, and `ferrors` rather than introducing a separate automation subsystem.

## Final Verification Notes

- Focused validation for this feature should start with:
  - `go test ./c8volt/ferrors -count=1`
  - `go test ./cmd -count=1`
- Documentation regeneration should be verified with:
  - `make docs`
  - `make docs-content`
- Repository validation remains `make test`, which is required before the feature is considered complete.

## Implementation Closure

- The final contract shape is now implemented with `c8volt capabilities --json` as the discovery entry point and shared result envelopes across the representative `get`, `run`, `expect`, `walk`, `deploy`, `delete`, and `cancel` command families.
- Validation completed on 2026-04-17 with the focused suites, docs regeneration, and repository-wide `make test`, confirming the contract rollout, generated docs, and preserved human CLI behavior remain aligned.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
