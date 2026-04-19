# Implementation Plan: Improve CLI Help Discovery

**Branch**: `077-cli-help-discovery` | **Date**: 2026-04-19 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/077-cli-help-discovery/spec.md)
**Input**: Feature specification from `/specs/077-cli-help-discovery/spec.md`

## Summary

Refresh the full public `c8volt` command tree so every user-visible parent command and executable leaf command explains its purpose, mutation behavior, recommended automation output mode, and relevant waiting or confirmation semantics through Cobra `Short`, `Long`, and `Example` metadata. The implementation should stay repository-native by updating command metadata in `cmd/`, regenerating `docs/cli/` through `make docs`, refreshing README-synced docs when README changes via `make docs-content`, and proving the result with focused command-help regression checks before final `make test`.

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, existing command metadata and machine-contract helpers in `cmd/`, generated CLI reference flow via `docsgen`  
**Storage**: N/A  
**Testing**: focused `go test ./cmd -count=1` for command metadata/help regressions, generated CLI docs via `make docs`, README-synced docs via `make docs-content` when README changes, final repository validation with `make test`  
**Target Platform**: Cross-platform Go CLI used interactively by operators and non-interactively by scripts, CI, and AI-assisted callers  
**Project Type**: CLI  
**Performance Goals**: Keep help and docs improvements metadata-only; do not introduce new runtime API calls or alter command execution behavior beyond small truth-preserving help clarifications; keep doc generation within the existing `make docs` and `make docs-content` flow  
**Constraints**: Cover all user-visible commands in the public Cobra tree, include both parent/group commands and executable leaf commands, refresh examples for every covered command, exclude hidden/internal commands, preserve existing command structure and terminology, keep docs generated from source metadata rather than hand-edited, and avoid broad behavioral changes unless needed to make help truthful  
**Scale/Scope**: Root and parent command metadata in [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go), family entry points such as `cmd/get.go`, `cmd/delete.go`, `cmd/deploy.go`, `cmd/run.go`, `cmd/walk.go`, `cmd/expect.go`, `cmd/config.go`, `cmd/cancel.go`, `cmd/embed.go`, command leaf files under [`cmd/`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd), capability and automation guidance in [`cmd/command_contract.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/command_contract.go) and [`cmd/capabilities.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities.go), user-facing docs in [`README.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md), [`docs/use-cases.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/use-cases.md), [`docs/index.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md), and generated CLI pages under [`docs/cli/`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. This feature improves how commands describe confirmed completion, accepted work, and destructive confirmation so user-visible help better matches operational truth instead of vague intent.
- **CLI-First, Script-Safe Interfaces**: Pass. The change stays inside the existing Cobra command tree, help metadata, and generated CLI docs; it does not create a parallel documentation or automation surface.
- **Tests and Validation Are Mandatory**: Pass. The plan requires targeted `cmd/` validation of help/metadata coverage, doc regeneration through the existing make targets, and final `make test`.
- **Documentation Matches User Behavior**: Pass. The feature is explicitly about command help and docs parity, so README and generated CLI docs move in the same unit of work whenever visible guidance changes.
- **Small, Compatible, Repository-Native Changes**: Pass. The implementation updates current command metadata and regeneration paths rather than inventing a second docs system or a new command taxonomy.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/077-cli-help-discovery/research.md).

- Confirm the full user-visible command inventory that falls inside the public Cobra tree and generated docs surface.
- Confirm the repo-native sources of truth for command descriptions, examples, mutation metadata, automation guidance, and hidden-command filtering.
- Confirm how parent/group commands and executable leaf commands should differ in help emphasis while still each receiving refreshed examples.
- Confirm the exact documentation regeneration boundaries: `make docs` for CLI reference pages and `make docs-content` when README changes should flow into the docs homepage.
- Confirm the minimum regression surface that proves help text changes remain truthful without requiring a full manual audit of generated output for every run.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/077-cli-help-discovery/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/077-cli-help-discovery/quickstart.md)
- [contracts/cli-help-contract.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/077-cli-help-discovery/contracts/cli-help-contract.md)

- Treat every user-visible command node as in scope, including root/group commands and executable leaf commands.
- Use Cobra `Short`, `Long`, and `Example` metadata as the authoritative editable source for command help and for generated `docs/cli/`.
- Exclude hidden/internal commands from the help-refresh scope by relying on the same public/discoverable command boundary the repository already uses for generated docs and discovery.
- Require refreshed examples for every covered command, with parent/group commands allowed to use navigational or chooser-oriented examples and leaf commands expected to use operational examples.
- Keep automation and machine-contract guidance aligned with the existing metadata in `cmd/command_contract.go` and user guidance already surfaced through `capabilities --json`, `--automation`, `--json`, `--auto-confirm`, and `--no-wait`.

### Authoritative Help Refresh Boundary

| Concern | Required design rule |
|--------|-----------------------|
| Public command scope | Cover every user-visible command in the public Cobra tree |
| Hidden/internal commands | Exclude them unless deliberately surfaced as public commands |
| Parent/group commands | Explain family purpose, routing, and command selection guidance |
| Leaf commands | Explain purpose, mutation semantics, automation output guidance, and realistic follow-up flows |
| Examples | Refresh examples for every covered command |
| Generated docs | Regenerate `docs/cli/` from command metadata only |
| README-synced docs | Run `make docs-content` if README changes affect docs homepage content |

### Coverage Shape

| Command layer | Expected planning focus |
|--------|-------------------------|
| Root command | Product story, discovery entry point, automation guidance, shared inherited flags |
| Parent/group commands | Explain family purpose, child-command routing, and mutation posture |
| Operational leaf commands | Add or refresh output-mode, waiting, confirmation, and verification guidance |
| Utility leaf commands | Add precise chooser-oriented examples and clarify automation or config relevance where applicable |
| Generated docs | Ensure refreshed metadata flows cleanly into `docs/cli/` and README-synced docs |

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Inventory the full user-visible Cobra tree and classify commands into root, parent/group, and executable leaf coverage so no public commands are skipped and hidden/internal commands stay out of scope.
2. Refresh top-level and parent/group command metadata first so family purpose, routing guidance, and shared automation/discovery language are consistent before leaf-command edits begin.
3. Refresh executable leaf-command `Short`, `Long`, and `Example` metadata across the public tree, reusing existing operational patterns and adding truthful guidance for `--json`, `--automation`, `--auto-confirm`, and `--no-wait` where relevant.
4. Add targeted command-level regression coverage for representative root, parent/group, and leaf help paths so future metadata changes cannot silently reintroduce vague or inconsistent guidance.
5. Update README guidance where command-help changes alter user-facing examples or discovery language, regenerate CLI docs with `make docs`, regenerate README-synced docs with `make docs-content` when README changed, and verify generated output reflects the updated metadata.
6. Finish with `make test`.

## Project Structure

### Documentation (this feature)

```text
specs/077-cli-help-discovery/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-help-contract.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── root.go
├── capabilities.go
├── command_contract.go
├── get.go
├── get_*.go
├── run.go
├── run_*.go
├── deploy.go
├── deploy_*.go
├── delete.go
├── delete_*.go
├── cancel.go
├── cancel_*.go
├── expect.go
├── expect_*.go
├── walk.go
├── walk_*.go
├── config.go
├── config_*.go
├── embed.go
├── embed_*.go
├── version.go
└── *_test.go

docs/
├── cli/
│   └── generated Markdown command reference
├── index.md
└── use-cases.md

README.md
Makefile
docsgen/
```

**Structure Decision**: Keep the work inside the existing single-project Go CLI layout. Source edits stay in `cmd/` command metadata and adjacent tests; user-facing docs stay in README plus generated docs under `docs/cli/`; documentation output is regenerated through the repository’s existing make targets rather than by hand.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design explicitly focuses on making waiting, acceptance, and confirmation semantics visible in help where behavior already exists.
- **CLI-First, Script-Safe Interfaces**: Still passes. The design preserves the existing command tree and uses source metadata plus generated docs as the one public documentation surface.
- **Tests and Validation Are Mandatory**: Still passes. Focused `cmd/` regression coverage, doc generation checks, and final `make test` remain required.
- **Documentation Matches User Behavior**: Still passes. The design requires README and generated docs to move in lockstep with visible help changes.
- **Small, Compatible, Repository-Native Changes**: Still passes. The feature remains a metadata-and-doc-generation refactor, not a behavioral rewrite.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
