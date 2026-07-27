# Implementation Plan: Ops-Scale Preflight And Progress UX

**Branch**: `259-ops-scale-progress-ux` | **Date**: 2026-07-27 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/259-ops-scale-progress-ux/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

Add a shared ops-scale preflight and progress UX for c8volt commands that can process thousands of Camunda resources. The implementation should reuse existing page metadata and the existing activity writer/sink, begin with `ops analyse slow-process-instances` as the proof workflow, and keep command renderers responsible for prompts and output-mode safety while services own pagination, enrichment, and mutation mechanics.

## Technical Context

**Language/Version**: Go, using the repository's current Go module and existing generated Camunda 8.7/8.8/8.9 clients

**Primary Dependencies**: Cobra for CLI command wiring, Viper-backed configuration, existing `toolx/logging` activity sink/writer, existing `toolx/pool`, existing internal service/facade packages

**Storage**: N/A; feature is runtime CLI behavior and structured command output only

**Testing**: Go tests with targeted package runs first, then `make test` for the full race-enabled repository validation

**Target Platform**: Cross-platform CLI for operator terminals and automation environments supported by c8volt

**Project Type**: CLI application with public facade packages, version-neutral internal services, and version-specific Camunda adapters

**Performance Goals**: Preflight must avoid duplicate full discovery when first-page metadata is sufficient; progress must stay lightweight enough for 10,000+ selected resources; enrichment progress must not add unbounded fan-out or per-resource output noise

**Constraints**: JSON stdout remains one valid JSON document; keys-only stdout remains one key per line; quiet and automation modes remain deterministic; default human output stays compact; HTTP endpoint traces remain debug-only; destructive workflows preserve frozen-scope and confirmation semantics

**Scale/Scope**: First proof slice covers `ops analyse slow-process-instances` discovery and runtime element enrichment at thousands of process instances; follow-up slices normalize destructive ops workflows and basic high-volume inspection commands for process instances, incidents, jobs, elements, and process definitions

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The plan requires preflight, frozen-scope progress, and final counts to reflect actual observed or frozen command scope. Destructive workflows must retain confirmation against the frozen scope.
- **CLI-First, Script-Safe Interfaces**: PASS. The design keeps user-visible behavior in Cobra command renderers and explicitly protects JSON, keys-only, quiet, and automation output.
- **Tests and Validation Are Mandatory**: PASS. The plan requires targeted command, facade/service, renderer, and activity-sink tests plus `make test` before completion.
- **Documentation Matches User Behavior**: PASS. Help text, generated CLI docs, and README updates are required because this changes visible preflight and progress behavior.
- **Small, Compatible, Repository-Native Changes**: PASS. The plan reuses `cmd` renderers, `toolx/logging`, service-owned traversal, existing reported-total metadata, and existing ops discovery scope models before adding any new shared abstraction.

## Project Structure

### Documentation (this feature)

```text
specs/259-ops-scale-progress-ux/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-progress-contract.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_processinstance_paging.go
├── get_processinstance_search.go
├── ops_analyse_slow_process_instances.go
├── cmd_views_ops_slow_process_analysis.go
├── cancel_processinstance.go
├── delete_processinstance.go
└── ops_* command renderers and tests

c8volt/
├── process/
├── incident/
├── job/
├── element/
└── ops/

internal/domain/
├── processinstance.go
├── incident.go
├── job.go
├── element.go
├── processdefinition.go
└── ops*.go

internal/services/
├── processinstance/
├── incident/
├── job/
├── element/
├── processdefinition/
└── ops/

toolx/logging/
└── activity.go

testx/activitysink/
└── activity_sink.go

docs/
└── cli/ generated command documentation
```

**Structure Decision**: Use the existing single-project Go CLI structure. Shared progress concepts belong in version-neutral domain/facade/service shapes where they describe command work, while final wording, prompts, stderr/activity routing, and output-mode gating remain in `cmd`. The first proof implementation should touch `internal/services/ops/slow_process_analysis.go`, its public facade mapping in `c8volt/ops`, and `cmd/ops_analyse_slow_process_instances.go`/renderer tests before broader command-family expansion.

## Complexity Tracking

No constitution violations are planned. The only new abstraction justified during planning is a shared ops-scale progress/preflight model because the requirement spans many command families and existing ad hoc activity strings cannot represent total certainty, preflight consequences, page progress, frozen-scope progress, and ETA consistently.

## Phase 0: Research

Research is captured in [research.md](research.md). All planning unknowns were resolved during this phase.

## Phase 1: Design

Design artifacts:

- [data-model.md](data-model.md) defines preflight scope, total certainty, page progress, frozen-scope progress, ETA, progress channels, and coverage targets.
- [contracts/cli-progress-contract.md](contracts/cli-progress-contract.md) defines the user-facing CLI/activity contract for human, verbose, JSON, keys-only, quiet, and automation modes.
- [quickstart.md](quickstart.md) defines validation scenarios and expected outcomes.

## Constitution Check After Design

- **Operational Proof Over Intent**: PASS. Design distinguishes candidate scope, frozen scope, and final affected resources, and requires progress/final output to reflect the scope actually used.
- **CLI-First, Script-Safe Interfaces**: PASS. The CLI contract explicitly states stdout and prompt/progress behavior by output mode.
- **Tests and Validation Are Mandatory**: PASS. Quickstart identifies focused tests for activity sink, renderer contracts, command output modes, and slow-process fake-volume behavior before full validation.
- **Documentation Matches User Behavior**: PASS. Documentation updates are called out as required follow-up tasks for command help and generated docs.
- **Small, Compatible, Repository-Native Changes**: PASS. Design reuses existing activity sink, page metadata, discovery scope status, command renderers, and service-owned paging.
