# Implementation Plan: Process Definition Watch Repaint

**Branch**: `268-pd-watch-repaint` | **Date**: 2026-08-05 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/268-pd-watch-repaint/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

Fix `get process-definition --watch` / `get pd --watch` so it behaves like a live terminal watch view: each refresh repaints one visible result, the result body matches normal non-watch human output, watch-specific snapshot labels disappear, slow refreshes are detected without noisy default warnings, and machine-oriented watch combinations remain rejected before lookup work.

## Technical Context

**Language/Version**: Go 1.26 with the repository's existing Cobra CLI and generated Camunda 8.7/8.8/8.9 clients

**Primary Dependencies**: Cobra command wiring, existing render-mode helpers in `cmd`, reusable watch runner in `toolx/watch`, Viper-backed config, public `c8volt/process` facade, internal `processdefinition` services, generated docs tooling

**Storage**: N/A; feature changes runtime CLI rendering and command status behavior only

**Testing**: Go tests with targeted `cmd` and `toolx/watch` runs first, generated-doc validation, then `make test` for full race-enabled repository validation

**Target Platform**: Cross-platform CLI operator terminals and script-safe non-watch command invocations supported by c8volt

**Project Type**: CLI application with public facade packages, version-neutral internal services, and version-specific Camunda adapters

**Performance Goals**: Watch refresh collection remains non-overlapping; slow refreshes are detected when a refresh exceeds the configured interval; repaint/status overhead is negligible compared with process-definition collection

**Constraints**: Non-watch output remains unchanged; watch mode remains human-only; `--watch` with `--json`, `--keys-only`, `--xml`, `--quiet`, or `--automation` is rejected before lookup work; generated Camunda clients are not hand-edited; default result stdout must stay compact and free of low-level endpoint, cursor, or per-page lifecycle detail

**Scale/Scope**: This follow-up slice covers only process-definition watch rendering/status behavior, including broad and filtered snapshots, direct key snapshots, statistics output where already supported, slow-refresh warning streaks, docs/help metadata, and existing command contract tests

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. Each visible watch view represents one completed refresh; slow-refresh warnings are emitted only after measured refresh completion; cancellation, timeout, and retry exhaustion remain explicit.
- **CLI-First, Script-Safe Interfaces**: PASS. The design preserves flag names and exit behavior, keeps watch human-only, and keeps JSON, keys-only, XML, quiet, and automation contracts finite by rejecting those combinations.
- **Tests and Validation Are Mandatory**: PASS. The plan requires focused command/watch tests, docs regeneration when metadata changes, and full `make test` before completion.
- **Documentation Matches User Behavior**: PASS. README, command help, command metadata, and generated CLI docs must be updated from the command source so "repaint" replaces the old "repeated snapshot blocks" wording.
- **Small, Compatible, Repository-Native Changes**: PASS. The plan reuses the existing process-definition snapshot path and watch runner, keeping changes scoped to command rendering/status plus targeted tests/docs.

## Project Structure

### Documentation (this feature)

```text
specs/268-pd-watch-repaint/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-watch-repaint-contract.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
toolx/watch/
├── watch.go
└── watch_test.go

cmd/
├── get_processdefinition.go
├── get_processdefinition_test.go
├── cmd_views_get.go
├── cmd_views_rendermode.go
├── command_contract.go
└── command_contract_test.go

c8volt/process/
├── api.go
├── client.go
├── convert.go
└── model.go

internal/services/processdefinition/
├── search.go
└── search_test.go

README.md
docs/cli/
docsgen/
```

**Structure Decision**: Use the existing single-project Go CLI structure. `toolx/watch` continues to own serial tick timing and cancellation mechanics. `internal/services/processdefinition` and `c8volt/process` remain the snapshot collection path. `cmd` owns repaint, result-body rendering, incompatible flag validation, slow-refresh status, help text, and command contract metadata because those are operator-visible CLI concerns.

## Complexity Tracking

No constitution violations are planned. No new service or facade abstraction is required; any new helper should be narrowly scoped to terminal repaint/status behavior unless the existing renderer already has the right shape.

## Phase 0: Research

Research is captured in [research.md](research.md). All planning unknowns were resolved during this phase.

## Phase 1: Design

Design artifacts:

- [data-model.md](data-model.md) defines the watch refresh, repainted view, slow-refresh warning streak, and output-mode compatibility concepts.
- [contracts/cli-watch-repaint-contract.md](contracts/cli-watch-repaint-contract.md) defines the observable CLI contract for repaint, result body parity, slow-refresh warnings, verbose diagnostics, and incompatible output modes.
- [quickstart.md](quickstart.md) defines targeted validation scenarios and expected outcomes.

## Constitution Check After Design

- **Operational Proof Over Intent**: PASS. Contracts require completed refreshes before rendering and measured slow-refresh warnings; no success message is claimed before observable output exists.
- **CLI-First, Script-Safe Interfaces**: PASS. The CLI contract preserves deterministic machine-output behavior by keeping watch human-only and rejecting incompatible modes before lookup work.
- **Tests and Validation Are Mandatory**: PASS. Quickstart covers focused command tests, watch-loop timing tests if needed, docs regeneration, and full repository validation.
- **Documentation Matches User Behavior**: PASS. Documentation updates are part of the design, and generated docs are refreshed from command metadata rather than hand-edited.
- **Small, Compatible, Repository-Native Changes**: PASS. The implementation path changes only the necessary command-facing behavior and reuses the existing snapshot, paging, facade, and watch-loop code.
