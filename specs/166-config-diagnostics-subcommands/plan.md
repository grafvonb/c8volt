# Implementation Plan: Config Diagnostics Subcommands

**Branch**: `166-config-diagnostics-subcommands` | **Date**: 2026-05-04 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/166-config-diagnostics-subcommands/spec.md`

## Summary

Split the current `c8volt config show` diagnostics flags into dedicated `config validate`, `config template`, and `config test-connection` subcommands while preserving `config show`, `config show --validate`, and `config show --template`. Extract reusable config validation and template rendering helpers so legacy flags and new subcommands share behavior, then implement `test-connection` as a config-first diagnostic that validates local configuration, logs the loaded config source at `INFO`, reuses the existing cluster topology facade/renderer for the remote connection proof, and warns only on major/minor Camunda version mismatches.

## Technical Context

**Language/Version**: Go, repository current module toolchain  
**Primary Dependencies**: Cobra command tree, Viper-backed configuration resolver, existing `config.Config` validation/template/sanitization methods, c8volt cluster facade, existing cluster topology renderer and error/logging helpers  
**Storage**: N/A  
**Testing**: Go tests through targeted `go test ./cmd` and final `make test`  
**Target Platform**: CLI on the repository's supported platforms  
**Project Type**: Go CLI  
**Performance Goals**: Validation and template subcommands complete without remote network calls; `test-connection` performs configuration validation before one topology retrieval attempt  
**Constraints**: Preserve legacy `config show` behavior and flags; do not shell out to existing commands internally; use existing error handling and topology rendering; make loaded config source visible at `INFO` without `--debug`; compare Camunda versions by major/minor only  
**Scale/Scope**: One command family under `c8volt config`, with docs/help/test updates for the new subcommands and compatibility shortcuts

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. `test-connection` reports success only after local validation and cluster topology retrieval succeed.
- **CLI-First, Script-Safe Interfaces**: PASS. The plan adds explicit subcommands while preserving existing flags, exit behavior, and documented command paths.
- **Tests and Validation Are Mandatory**: PASS. The plan requires command-level tests for compatibility, validation, template output, connection success/failure, logging, version comparison, docs/help, targeted Go tests, and `make test`.
- **Documentation Matches User Behavior**: PASS. README and generated CLI docs are in scope because the command surface changes.
- **Small, Compatible, Repository-Native Changes**: PASS. The design reuses the Cobra command layout, config resolver, cluster facade, topology renderer, and existing error/logging helpers.

## Project Structure

### Documentation (this feature)

```text
specs/166-config-diagnostics-subcommands/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-config-diagnostics.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── config.go
├── config_show.go
├── config_test.go
├── get_cluster_topology.go
├── cmd_views_cluster.go
├── cmd_cli.go
└── root.go

config/

c8volt/
└── cluster/

internal/
└── services/
    └── cluster/

README.md
docs/
docsgen/
```

**Structure Decision**: Keep command wiring under the existing `configCmd`. Add small `config validate`, `config template`, and `config test-connection` Cobra commands in `cmd/` next to `config_show.go`; extract shared local helpers from `config_show.go` for validation and template rendering so compatibility flags and new subcommands cannot drift. Use the existing `NewCli`, `GetClusterTopology`, and `renderClusterTopologyTree` flow for the remote proof instead of introducing a separate cluster client path.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

- Data model: [data-model.md](./data-model.md)
- CLI contract: [contracts/cli-config-diagnostics.md](./contracts/cli-config-diagnostics.md)
- Quickstart and verification scenarios: [quickstart.md](./quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The contract requires `test-connection` to validate config before remote calls and to render topology only after a successful topology response.
- **CLI-First, Script-Safe Interfaces**: PASS. All command paths, legacy shortcuts, exit-code expectations, and warning semantics are documented.
- **Tests and Validation Are Mandatory**: PASS. The task list will include close command tests, help/discovery tests, docs regeneration, targeted Go tests, and `make test`.
- **Documentation Matches User Behavior**: PASS. README and generated CLI documentation updates are mandatory because users discover this feature through command help and docs.
- **Small, Compatible, Repository-Native Changes**: PASS. The design avoids parallel config or cluster abstractions and keeps the command family consistent with existing Cobra patterns.

## Complexity Tracking

No constitution violations or additional complexity exceptions are required.
