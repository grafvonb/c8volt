# Implementation Plan: Process Definition Watch Mode

**Branch**: `258-process-definition-watch` | **Date**: 2026-08-04 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/258-process-definition-watch/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

Add human-output-only `--watch` support to `get process-definition` / `get pd` / `get pds` so operators can repeatedly observe process-definition snapshots until interrupted, timed out, or stopped by unrecoverable failure. The implementation will keep `cmd` responsible for flags, validation, output-mode selection, and rendering; put reusable fixed-interval watch-loop mechanics in `toolx`; keep process-definition snapshot collection and paging below the facade in `internal/services/processdefinition`; reject machine-oriented watch combinations before lookup work; and preserve existing non-watch behavior and machine-output contracts.

## Technical Context

**Language/Version**: Go 1.26 with the repository's existing Cobra CLI and generated Camunda 8.7/8.8/8.9 clients

**Primary Dependencies**: Cobra command wiring, Viper-backed config, existing root `--timeout` and command-local backoff config, existing `toolx/logging` activity writer/sink, public `c8volt/process` facade, internal `processdefinition` services, generated docs tooling

**Storage**: N/A; feature is runtime CLI behavior and structured command output only

**Testing**: Go tests with targeted package runs first, then `make test` for full race-enabled repository validation

**Target Platform**: Cross-platform CLI operator terminals and automation environments supported by c8volt

**Project Type**: CLI application with public facade packages, version-neutral internal services, and version-specific Camunda adapters

**Performance Goals**: Default watch cadence is 1 second; each snapshot uses existing paged process-definition discovery without duplicate command-local page loops; activity/status work must stay lightweight for broad all-process-definition snapshots

**Constraints**: Non-watch command behavior remains unchanged; watch mode is human-output only; `--watch` with `--json`, `--keys-only`, `--xml`, `--quiet`, or `--automation` is rejected before lookup work; verbose/debug remain human diagnostics; generated Camunda clients are not hand-edited

**Scale/Scope**: First feature slice covers process-definition lookup only, including broad missing-selector watch across all process definitions, BPMN/latest selectors, key lookup, stat-style output, paged snapshots, retry budget reset, retry exhaustion, interrupt handling, incompatible machine-output validation, and documentation updates

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. Watch mode reports one snapshot only after the selected lookup for that tick completes; interrupted or timed-out watch sessions must distinguish normal watch termination from lookup failure.
- **CLI-First, Script-Safe Interfaces**: PASS. The design exposes stable flags, makes watch explicitly human-output only, and preserves existing stdout contracts by rejecting JSON, keys-only, quiet, automation, and XML watch combinations.
- **Tests and Validation Are Mandatory**: PASS. The plan requires focused tests in `toolx`, `internal/services/processdefinition`, `c8volt/process`, `cmd`, docs generation, and full `make test` before completion.
- **Documentation Matches User Behavior**: PASS. Command help, command metadata, README-facing examples, and generated CLI docs must describe watch behavior, interval default, retry/timeout behavior, and output-mode contracts.
- **Small, Compatible, Repository-Native Changes**: PASS. The plan reuses Cobra command structure, root config/backoff behavior, existing activity/output helpers, facade mapping patterns, and service-owned paging. The only new reusable helper is the watch loop required for future read-only command reuse.

## Project Structure

### Documentation (this feature)

```text
specs/258-process-definition-watch/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-watch-contract.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
toolx/
├── watch/
│   ├── watch.go
│   └── watch_test.go
└── logging/
    └── activity.go

internal/domain/
└── processdefinition.go

internal/services/processdefinition/
├── api.go
├── search.go
├── search_test.go
├── v87/
├── v88/
└── v89/

c8volt/process/
├── api.go
├── client.go
├── convert.go
├── model.go
└── client_test.go

cmd/
├── get.go
├── get_processdefinition.go
├── get_processdefinition_test.go
├── cmd_views_get.go
├── cmd_views_rendermode.go
├── command_contract.go
└── command_contract_test.go

docs/
└── cli/ generated command documentation

README.md
docsgen/
```

**Structure Decision**: Use the existing single-project Go CLI structure. `toolx/watch` owns reusable interval/retry/cancellation mechanics without process-definition knowledge. `internal/services/processdefinition` owns complete snapshot paging. `c8volt/process` remains a thin facade that maps public request/result types. `cmd/get_processdefinition.go` owns flags, validation, mode selection, and rendering because stdout/stderr contracts are command UX.

## Complexity Tracking

No constitution violations are planned. The only new abstraction is a reusable watch loop, justified because the issue explicitly asks for watch mechanics reusable by other read-only commands and because embedding retry/interval/cancellation loops directly in `cmd/get_processdefinition.go` would make later command adoption inconsistent.

## Phase 0: Research

Research is captured in [research.md](research.md). All planning unknowns were resolved during this phase.

## Phase 1: Design

Design artifacts:

- [data-model.md](data-model.md) defines watch session, snapshot request/result, interval, retry budget, termination reason, and output-mode contracts.
- [contracts/cli-watch-contract.md](contracts/cli-watch-contract.md) defines the observable CLI behavior for flags, human snapshots, retry/timeout, and incompatible output combinations.
- [quickstart.md](quickstart.md) defines focused validation scenarios and expected outcomes.

## Constitution Check After Design

- **Operational Proof Over Intent**: PASS. Contracts require snapshots to represent completed current lookups and require termination status to distinguish interrupt/timeout from lookup failure.
- **CLI-First, Script-Safe Interfaces**: PASS. The CLI contract explicitly defines human-only watch behavior, JSON/keys-only/quiet/automation/XML rejection, verbose diagnostics, and non-watch compatibility behavior.
- **Tests and Validation Are Mandatory**: PASS. Quickstart covers focused `toolx`, service/facade, command, docs, and full validation commands.
- **Documentation Matches User Behavior**: PASS. Documentation updates are required for all new user-facing flags and watch output contracts, with generated docs refreshed from command metadata.
- **Small, Compatible, Repository-Native Changes**: PASS. Design reuses service-owned paging, existing process facade conversion, root timeout/backoff configuration, and command view helpers while keeping generated clients untouched.
