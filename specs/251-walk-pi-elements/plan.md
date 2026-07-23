# Implementation Plan: Walk PI Elements

**Branch**: `251-walk-pi-elements` | **Date**: 2026-07-22 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/251-walk-pi-elements/spec.md`

## Summary

Add `--with-elements` to `walk process-instance` / `walk pi` so keyed process-instance traversals can show runtime element instances below each walked process-instance row. The implementation will reuse the existing process-instance element enrichment facade and shared activity rendering model, then extend the walk-specific orchestration, validation, human output, JSON output, command metadata, and documentation.

## Technical Context

**Language/Version**: Go 1.26 with toolchain go1.26.2

**Primary Dependencies**: Cobra for CLI commands; existing c8volt public facades under `c8volt/process`; internal services under `internal/services/processinstance`; shared helpers under `toolx` and `testx`

**Storage**: N/A; read-only CLI enrichment over existing Camunda runtime data

**Testing**: Go unit and command tests with `go test`; full repository validation through `make test`

**Target Platform**: Cross-platform command-line binary for c8volt-supported operator environments

**Project Type**: CLI application with public facade APIs and internal Camunda service adapters

**Performance Goals**: Element lookup happens after traversal and performs one enrichment pass over the already walked process instances without changing traversal selection; no extra element lookups when `--with-elements` is absent or validation fails

**Constraints**: Preserve existing walk behavior without `--with-elements`; reject invalid flag combinations before remote enrichment; fail rather than render partially enriched success output; keep command-layer code out of generated Camunda clients and version-specific service implementations

**Scale/Scope**: Keyed walk traversals across family, children, parent, and flat modes; supports combination with existing `--with-vars` and `--with-incidents`; excludes element filters and other enrichment types

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The command remains read-only and reports enriched traversal output only after traversal and requested enrichment complete successfully. Validation and enrichment failures must produce errors instead of partial success.
- **CLI-First, Script-Safe Interfaces**: PASS. The feature is exposed through an explicit flag, stable human section ordering, stable JSON payload shape, and pre-enrichment validation for incompatible keys-only output.
- **Tests and Validation Are Mandatory**: PASS. Planning requires targeted `cmd` tests for human, JSON, validation, unsupported-version, and unchanged default behavior, plus `make test` before commit/merge.
- **Documentation Matches User Behavior**: PASS. User-visible flag/help/output changes require command metadata updates, README alignment, and regenerated CLI docs with `make docs-content`.
- **Small, Compatible, Repository-Native Changes**: PASS. The plan reuses existing facade enrichment and activity rendering structures; no new abstraction or generated client edits are planned.

## Project Structure

### Documentation (this feature)

```text
specs/251-walk-pi-elements/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── walk-pi-elements-cli.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── walk_processinstance.go                 # walk flag registration, validation, orchestration
├── cmd_views_processinstance_activity.go   # activity item merge and element-aware detail formatting
├── cmd_views_walk_incidents.go             # walk path/tree rendering and traversal JSON payloads
├── command_contract.go                     # command capability metadata source
├── command_contract_test.go                # discoverability and command contract assertions
├── walk_test.go                            # walk behavior, validation, output, and version tests
└── get_processinstance_enrichment.go       # reusable enrichment activity helpers

c8volt/process/
├── api.go                                  # public process facade contract
├── client.go                               # facade delegation to internal services
└── model.go                                # public JSON models for process instances and elements

internal/services/processinstance/
└── enrichment.go                           # existing internal element enrichment behavior

README.md                                  # user-facing examples and behavior notes
docs/cli/                                  # regenerated CLI reference from command metadata
```

**Structure Decision**: Use the existing single Go CLI project layout. Command wiring and rendering remain in `cmd`, facade calls remain through `c8volt/process`, and internal enrichment remains under `internal/services/processinstance`.

## Complexity Tracking

No constitution violations or justified complexity exceptions.

## Phase 0: Research

Research decisions are recorded in [research.md](./research.md). All technical context unknowns are resolved; no `NEEDS CLARIFICATION` markers remain.

## Phase 1: Design & Contracts

Design artifacts:

- [data-model.md](./data-model.md)
- [contracts/walk-pi-elements-cli.md](./contracts/walk-pi-elements-cli.md)
- [quickstart.md](./quickstart.md)

Post-design Constitution Check:

- **Operational Proof Over Intent**: PASS. Contracts require traversal and enrichment to complete before rendering success, and lookup failures to fail the command.
- **CLI-First, Script-Safe Interfaces**: PASS. CLI flag, validation, human output, and JSON contract are specified.
- **Tests and Validation Are Mandatory**: PASS. Quickstart defines targeted command tests and full validation.
- **Documentation Matches User Behavior**: PASS. Quickstart includes README and generated docs validation.
- **Small, Compatible, Repository-Native Changes**: PASS. Data model and contracts reuse current public models and rendering concepts.
