# Implementation Plan: Slow Analysis Progress After Confirmation

**Branch**: `265-slow-analysis-progress` | **Date**: 2026-08-04 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/265-slow-analysis-progress/spec.md`

**Note**: This template is filled in by the `$speckit-plan` command; its definition describes the execution workflow.

## Summary

Fix broad `ops analyse slow-process-instances` runs that appear frozen after confirmation by adding shared durable milestone pacing to the command progress layer and using it for slow-analysis discovery and frozen-scope timeline loading. Services continue to emit structured progress events only; command code keeps ownership of human wording, activity updates, stderr routing, and output-mode safety.

## Technical Context

**Language/Version**: Go 1.26 with toolchain go1.26.2

**Primary Dependencies**: Cobra for CLI command wiring, existing `toolx/logging` activity writer/sink, existing `c8volt/ops` public progress model, existing `internal/domain` ops progress events, existing internal ops and process-instance services

**Storage**: N/A; feature changes runtime CLI progress behavior only

**Testing**: Go tests with targeted `go test ./cmd -run '<pattern>' -count=1` first, service/domain tests when shared progress state changes require them, then `make test` for full race-enabled validation

**Target Platform**: Cross-platform CLI operator terminals and automation environments supported by c8volt

**Project Type**: CLI application with public facade packages, version-neutral internal services, and version-specific Camunda adapters

**Performance Goals**: Milestone checks must be constant-time per progress event, add no extra Camunda requests, and remain lightweight for thousands of process instances and discovery pages

**Constraints**: JSON stdout remains one valid JSON document; keys-only stdout remains one key per line; quiet and automation modes remain deterministic; default human progress stays compact; verbose/debug durable detail remains available; services must not own human milestone wording or output-mode gating

**Scale/Scope**: Focused follow-up for broad `ops analyse slow-process-instances` after confirmation, covering multi-page process-instance discovery and per-process-instance runtime element or listener timeline loading

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The plan keeps progress tied to observed discovery and frozen-scope counters, and avoids timer-only lines that would imply work advanced without proof.
- **CLI-First, Script-Safe Interfaces**: PASS. Rendering and mode gating remain in `cmd`, with explicit protection for JSON, keys-only, quiet, and automation modes.
- **Tests and Validation Are Mandatory**: PASS. The plan requires focused shared progress policy tests, slow-process command progress tests, machine-output silence tests, and full validation before completion.
- **Documentation Matches User Behavior**: PASS. The implementation must review command help, README, and generated CLI docs. Updates are required only if visible wording or documented behavior changes.
- **Small, Compatible, Repository-Native Changes**: PASS. The design reuses existing structured progress events, command progress formatters, activity importance, and stderr routing instead of adding a slow-process-local policy.

## Project Structure

### Documentation (this feature)

```text
specs/265-slow-analysis-progress/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── slow-analysis-progress-contract.md
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 output from $speckit-tasks
```

### Source Code (repository root)

```text
cmd/
├── ops_progress.go
├── ops_progress_test.go
├── ops_analyse_slow_process_instances.go
├── ops_analyse_slow_process_instances_test.go
└── command_contract_test.go

c8volt/ops/
├── progress_model.go
└── convert.go

internal/domain/
├── ops_progress.go
└── ops_progress_test.go

internal/services/ops/
├── slow_process_analysis.go
└── slow_process_analysis_test.go

toolx/logging/
└── activity.go

testx/activitysink/
└── activity_sink.go

README.md
docs/
└── cli/ generated command documentation when command help changes
```

**Structure Decision**: Use the existing single-project Go CLI structure. Shared milestone pacing belongs in `cmd/ops_progress.go` because it is a command-rendering policy that depends on output mode, durable stderr permissions, and human wording. Slow-process analysis should call the shared policy from its existing progress callback in `cmd/ops_analyse_slow_process_instances.go`. Internal services should keep emitting `Preflight`, `Page`, `FrozenScope`, and `ETA` events without learning about human milestone pacing.

## Complexity Tracking

No constitution violations are planned. The only new abstraction is a small shared command progress pacing state, justified because issue #265 explicitly requires pacing policy in shared command progress code rather than a slow-process-local rule.

## Phase 0: Research

Research is captured in [research.md](research.md). All planning unknowns were resolved during this phase.

## Phase 1: Design

Design artifacts:

- [data-model.md](data-model.md) defines durable milestone, milestone pacing state, progress event inputs, progress channel behavior, and slow-analysis workflow phases.
- [contracts/slow-analysis-progress-contract.md](contracts/slow-analysis-progress-contract.md) defines the observable CLI progress contract after confirmation.
- [quickstart.md](quickstart.md) defines focused validation scenarios for default human, verbose/debug, machine-output safety, and full validation.

## Constitution Check After Design

- **Operational Proof Over Intent**: PASS. Design requires both elapsed silence and forward progress before durable default-human milestones, so visible milestones are grounded in real progress events.
- **CLI-First, Script-Safe Interfaces**: PASS. The contract states progress destinations by mode and preserves stdout safety.
- **Tests and Validation Are Mandatory**: PASS. Quickstart and contract identify focused command and shared progress tests before `make test`.
- **Documentation Matches User Behavior**: PASS. Documentation review is explicit; generated docs are updated through `make docs-content` only if command help changes.
- **Small, Compatible, Repository-Native Changes**: PASS. Design uses existing progress callbacks, formatting helpers, command channels, activity writer, and test helpers.
