# Implementation Plan: Define Non-Interactive Automation Mode

**Branch**: `079-non-interactive-automation-mode` | **Date**: 2026-04-19 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/079-non-interactive-automation-mode/spec.md)
**Input**: Feature specification from `/specs/079-non-interactive-automation-mode/spec.md`

## Summary

Add one dedicated root automation flag to `c8volt`, layer it onto the existing `#78` machine-readable contract instead of creating a separate execution surface, and make supported commands behave deterministically in unattended runs. The design reuses current Cobra flags, `ferrors` exit handling, shared result-envelope helpers, and existing prompt/paging seams so automation mode can implicitly confirm supported prompts, reject unsupported commands explicitly, keep machine-readable stdout clean, and preserve `accepted` outcomes when callers combine automation mode with `--no-wait`.

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, existing machine-contract helpers in `cmd/`, existing `c8volt/ferrors`, existing JSON render helpers under `cmd/`, existing public payload models under `c8volt/process` and `c8volt/resource`  
**Storage**: N/A  
**Testing**: focused `go test ./cmd -count=1`, repository validation with `make test`, generated CLI docs via `make docs`, docs homepage sync via `make docs-content` when README changes  
**Target Platform**: Cross-platform Go CLI used in interactive terminals, AI-agent automation, and CI-style scripted runs  
**Project Type**: CLI  
**Performance Goals**: Keep automation-mode checks local to command metadata and existing render/prompt seams; avoid extra network calls or additional command-discovery passes; preserve current command runtime characteristics outside the explicit automation flag  
**Constraints**: Keep the existing Cobra command tree and human UX intact, expose automation mode through one dedicated root flag, fail explicitly for unsupported commands instead of guessing behavior, preserve current exit-code semantics through `ferrors`, keep stdout machine-safe when JSON mode is requested, reuse existing `--no-wait` accepted-work semantics instead of inventing a second async model, update README and generated docs for the user-visible automation contract, and finish with `make test`  
**Scale/Scope**: Root flags and bootstrap in [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go), prompt handling in [`cmd/cmd_cli.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_cli.go), render/envelope helpers in [`cmd/cmd_views_rendermode.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_rendermode.go) and [`cmd/cmd_views_contract.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_contract.go), command metadata in [`cmd/command_contract.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/command_contract.go), representative commands under `cmd/get_processinstance.go`, `cmd/run_processinstance.go`, `cmd/deploy_processdefinition.go`, `cmd/delete_processinstance.go`, `cmd/cancel_processinstance.go`, `cmd/expect_processinstance.go`, and `cmd/walk_processinstance.go`, plus user-facing guidance in `README.md`, `docs/index.md`, and generated docs under `docs/cli/`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. The feature strengthens unattended execution by making prompt suppression, unsupported-command rejection, and accepted-versus-confirmed outcomes explicit instead of inferred.
- **CLI-First, Script-Safe Interfaces**: Pass. The design stays entirely inside the existing Cobra CLI, adds one root flag rather than a parallel runner, and preserves script-safe exit codes and JSON behavior.
- **Tests and Validation Are Mandatory**: Pass. The plan requires focused `cmd/` coverage for representative prompting and state-changing flows, documentation regeneration where help text changes, and final `make test`.
- **Documentation Matches User Behavior**: Pass. The automation flag and its recommended invocation pattern are user-visible, so README guidance and generated CLI docs must be updated in the same work.
- **Small, Compatible, Repository-Native Changes**: Pass. The implementation extends existing command metadata, prompt helpers, and result-envelope seams rather than adding a second automation subsystem.

## Project Structure

### Documentation (this feature)

```text
specs/079-non-interactive-automation-mode/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-automation-mode-contract.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── root.go
├── cmd_cli.go
├── cmd_views_rendermode.go
├── cmd_views_contract.go
├── command_contract.go
├── capabilities.go
├── get_processinstance.go
├── run_processinstance.go
├── expect_processinstance.go
├── walk_processinstance.go
├── deploy_processdefinition.go
├── delete_processinstance.go
├── cancel_processinstance.go
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
└── index.md

README.md
```

**Structure Decision**: Keep the work inside the existing single-project Go CLI layout. The automation flag belongs at the root so every command family sees the same explicit opt-in. Command-specific behavior should be wired through current prompt/render/metadata seams in `cmd/`, and user-facing guidance stays in README plus generated `docs/cli/`.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/079-non-interactive-automation-mode/research.md).

- Choose the dedicated flag name and root-level ownership model for automation mode.
- Decide how automation support should be represented in command metadata and discovery without overloading the existing shared-envelope support signal.
- Define how supported commands inherit implicit confirmation, paging continuation, and explicit unsupported-command rejection.
- Confirm the boundary between automation mode and existing `--json`, `--no-wait`, `--quiet`, and logging behavior.
- Identify the smallest repository-native implementation seams for rollout: root flags, prompt helpers, contract metadata, representative command handlers, and docs generation paths.

### Refined Implementation Boundary

| Area | Planned responsibility | Key files |
|--------|------------------------|-----------|
| Root automation flag | Add one dedicated root flag, bind it through config resolution, and expose it consistently to command execution | `cmd/root.go` |
| Command metadata and discovery | Record which commands support automation mode and reflect that in machine discovery | `cmd/command_contract.go`, `cmd/capabilities.go` |
| Prompt and continuation control | Make supported commands auto-confirm and auto-continue under automation mode while rejecting unsupported flows explicitly | `cmd/cmd_cli.go`, `cmd/get_processinstance.go`, `cmd/delete_processinstance.go`, `cmd/cancel_processinstance.go`, `cmd/delete_processdefinition.go` |
| Result and output behavior | Preserve current exit-code semantics, keep shared result envelopes authoritative in JSON mode, and ensure stdout stays machine-safe when automation callers request JSON | `cmd/cmd_views_rendermode.go`, `cmd/cmd_views_contract.go`, representative state-changing commands |
| Documentation and tests | Describe the canonical invocation pattern and prove representative read/write behavior under automation mode | `README.md`, `docs/index.md`, `docs/cli/`, `cmd/*_test.go` |

The implementation boundary is intentionally limited to CLI-facing automation semantics built on top of the existing machine contract. It does not redesign command families, add new business operations, or replace current human-oriented flows outside the explicit automation flag.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/079-non-interactive-automation-mode/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/079-non-interactive-automation-mode/quickstart.md)
- [contracts/cli-automation-mode-contract.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/079-non-interactive-automation-mode/contracts/cli-automation-mode-contract.md)

- Use one dedicated root flag, planned as `--automation`, as the canonical opt-in for non-interactive execution.
- Reuse the existing shared JSON result envelope from `#78` instead of inventing a separate automation-only output model.
- Add explicit automation-mode support metadata per command so unsupported commands can reject the flag intentionally and discovery can report truthful support.
- Treat automation mode as implicit confirmation for supported prompt flows, but do not make commands asynchronous by default; `--no-wait` remains the explicit selector for accepted-but-not-yet-complete work.
- Keep stdout machine-safe in automation mode when JSON is requested by directing human-oriented logs and progress away from stdout.

### Authoritative Contract Boundary

| Concern | Required design contract |
|--------|---------------------------|
| Automation entry point | One dedicated root flag, planned as `--automation` |
| Support model | Supported commands opt in explicitly; unsupported commands reject automation mode instead of guessing behavior |
| Prompt behavior | Supported confirmation and continuation prompts are implicitly accepted under automation mode |
| JSON behavior | Shared result envelopes remain the machine-readable execution surface; automation mode does not replace them |
| Output channels | When JSON is requested in automation mode, stdout remains reserved for the machine-readable result |
| Async behavior | `--no-wait` remains the explicit trigger for accepted/not-yet-complete outcomes |
| Documentation | README and generated CLI docs must describe the new flag and its relationship to existing flags |
| Test strategy | Representative read/write/prompting flows must prove the automation contract without weakening human-mode behavior |

### Representative Automation Targets

| Command family | Existing seams | Automation focus |
|--------|-----------------|------------------|
| `capabilities` | `cmd/capabilities.go`, `cmd/command_contract.go` | discovery metadata for automation support and canonical invocation guidance |
| `get process-instance` | `cmd/get_processinstance.go`, `cmd/cmd_views_get.go` | paging continuation, JSON aggregation, and read-only automation behavior |
| `run process-instance` | `cmd/run_processinstance.go` | explicit `accepted` outcomes with `--no-wait`, automation + JSON compatibility |
| `deploy process-definition` | `cmd/deploy_processdefinition.go` | state-changing automation behavior with optional `--no-wait` |
| `delete process-instance` | `cmd/delete_processinstance.go` | implicit confirmation, dependency-impact prompts, unsupported-flow rejection where needed |
| `cancel process-instance` | `cmd/cancel_processinstance.go` | implicit confirmation, paging continuation, accepted outcomes with `--no-wait` |
| `expect process-instance` and `walk process-instance` | `cmd/expect_processinstance.go`, `cmd/walk_processinstance.go` | explicit supported or unsupported automation-mode behavior for representative read/observe commands |

This target set is the planning boundary for task generation. Additional commands may remain unsupported initially as long as their automation-mode rejection is explicit and discoverable.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Add the root automation flag, wire it through config/bootstrap, and define the shared helper that answers whether automation mode is active for a command run.
2. Extend command metadata and discovery so commands can report automation-mode support truthfully alongside the existing shared-envelope contract support.
3. Centralize automation-mode prompt behavior in the existing confirmation and pagination seams so supported commands implicitly confirm and continue without open-coded flag checks.
4. Update representative read/write commands to reject unsupported automation-mode use explicitly and to keep JSON-mode stdout machine-safe while preserving current human output outside the flag.
5. Add focused `cmd/` regression coverage for representative supported read and write flows, unsupported-command rejection, automation-plus-`--no-wait` accepted outcomes, and preserved human behavior without the flag.
6. Update README and command help text, regenerate `docs/cli/` with `make docs`, regenerate homepage docs with `make docs-content` if README changes, and finish with `make test`.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design makes non-interactive semantics explicit at the CLI boundary and keeps accepted-versus-confirmed outcomes observable.
- **CLI-First, Script-Safe Interfaces**: Still passes. The design remains rooted in one CLI flag, explicit metadata, current exit behavior, and structured JSON envelopes.
- **Tests and Validation Are Mandatory**: Still passes. Representative `cmd/` regression coverage plus final `make test` remain required.
- **Documentation Matches User Behavior**: Still passes. The design requires README and generated CLI docs to explain the automation flag and its interaction with existing flags.
- **Small, Compatible, Repository-Native Changes**: Still passes. The implementation extends current command metadata, prompt helpers, and render helpers rather than introducing a separate automation stack.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |

## Final Implementation Notes

- The delivered root contract is `--automation` bound through `app.automation`, with runtime support checks routed through shared command metadata instead of command-local flag guessing.
- Representative commands that currently report `automation:full` are `capabilities`, `get process-instance`, `run process-instance`, `deploy process-definition`, `delete process-instance`, and `cancel process-instance`.
- Representative observe flows `expect process-instance` and `walk process-instance` intentionally remain `automation:unsupported`, and reject `--automation` explicitly instead of falling back to interactive behavior.
- Shared prompt handling stays centralized in `cmd/cmd_cli.go`, so supported confirmation and paging-continuation prompts are implicitly accepted under automation mode without changing human-mode defaults.
- Shared result envelopes remain authoritative in JSON mode, with `accepted` reserved for explicit `--no-wait` runs and human-oriented diagnostics kept off stdout in automation JSON flows.
- User-facing automation guidance now lives in root and command help text, `README.md`, generated `docs/cli/`, and the README-synced `docs/index.md`.

## Final Validation Checklist

- Run focused automation regression coverage with `go test ./cmd -count=1`.
- Regenerate command and README-synced docs with `make docs` and `make docs-content`.
- Finish with repository validation through `make test`.
