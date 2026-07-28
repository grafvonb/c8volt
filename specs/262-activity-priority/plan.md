# Implementation Plan: Preserve High-Level Activity

**Branch**: `262-activity-priority` | **Date**: 2026-07-28 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/262-activity-priority/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

Make the shared terminal activity indicator hierarchical so high-level command and workflow progress remains visible while nested HTTP requests, waiters, pollers, and service lookups run. The approach preserves existing command output contracts, promotes central progress renderers to high-value activity, treats HTTP activity as low-value fallback, and extends known Camunda endpoint labels so fallback text remains resource-aware when no higher-level activity exists.

## Technical Context

**Language/Version**: Go 1.26 with the repository's existing Cobra CLI and generated Camunda 8.7/8.8/8.9 clients

**Primary Dependencies**: Existing `toolx/logging` activity writer/sink, `internal/services/httpc` transport, Cobra command context wiring, existing internal service/facade option patterns, `testx/activitysink` for activity assertions

**Storage**: N/A; feature changes transient runtime activity behavior only

**Testing**: Go tests with targeted packages first, then `make test` for full race-enabled validation

**Target Platform**: Cross-platform CLI operator terminals and automation environments supported by c8volt

**Project Type**: CLI application with public facade packages, version-neutral internal services, and version-specific Camunda adapters

**Performance Goals**: Activity selection must remain constant-time or bounded by a small active scope stack; spinner updates must remain lightweight for commands processing 10,000+ Camunda resources; no extra Camunda requests are introduced

**Constraints**: JSON stdout remains one valid JSON document; keys-only stdout remains one key per line; quiet and automation modes remain deterministic; durable preflight, prompt, warning, verbose, debug, and final outcome wording remains stable; HTTP trace detail remains debug-only

**Scale/Scope**: Covers all command families through shared activity behavior, with representative validation for process-instance get/search, process-instance mutation, process-definition mutation, deployment, wait/expect, ops analysis or repair, and simple Camunda-backed fallback commands

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The feature preserves high-level progress and final outcome signals while nested operational verification continues. It does not weaken wait, confirmation, or done-is-done semantics.
- **CLI-First, Script-Safe Interfaces**: PASS. The feature explicitly protects JSON, keys-only, quiet, and automation modes and keeps transient activity separate from stdout contracts.
- **Tests and Validation Are Mandatory**: PASS. The plan requires close unit tests for activity hierarchy, HTTP fallback behavior, and representative command/service regressions before full validation.
- **Documentation Matches User Behavior**: PASS. The feature changes visible terminal activity wording and hierarchy; README or generated CLI docs must be reviewed and updated only where the activity contract is user-documented.
- **Small, Compatible, Repository-Native Changes**: PASS. The design reuses existing `toolx/logging`, command contexts, `httpc`, and activity sink tests rather than adding per-command bespoke progress handling.

## Project Structure

### Documentation (this feature)

```text
specs/262-activity-priority/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── activity-hierarchy-contract.md
│   └── http-activity-labels.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
toolx/logging/
├── activity.go
└── activity_test.go

testx/activitysink/
└── activity_sink.go

internal/services/httpc/
├── round_trippers.go
├── round_trippers_test.go
└── service.go

cmd/
├── root.go
├── ops_analyse_slow_process_instances.go
├── processinstance_mutation_progress.go
├── get_processinstance_paging.go
├── get_processinstance_orphan.go
├── get_processinstance_total.go
└── representative command tests

internal/services/
├── processinstance/
├── processdefinition/
├── resource/
├── incident/waiter/
└── job/waiter/

docs/
├── cli/
└── README-aligned user documentation when visible behavior is documented
```

**Structure Decision**: Use the existing single-project Go CLI structure. The hierarchy belongs in `toolx/logging` because every command receives the shared activity writer through root context wiring. HTTP fallback priority belongs in `internal/services/httpc`, and command/service code should only opt into the appropriate semantic activity level at existing centralized progress points.

## Complexity Tracking

No constitution violations are planned. The only new abstraction is hierarchical activity scope selection, justified because the defect is cross-cutting and cannot be fixed consistently by rewriting individual command messages.

## Phase 0: Research

Research is captured in [research.md](research.md). All planning unknowns were resolved during this phase.

## Phase 1: Design

Design artifacts:

- [data-model.md](data-model.md) defines activity scope, importance levels, visibility selection, output modes, endpoint label families, and representative command coverage.
- [contracts/activity-hierarchy-contract.md](contracts/activity-hierarchy-contract.md) defines the observable activity hierarchy contract.
- [contracts/http-activity-labels.md](contracts/http-activity-labels.md) defines resource-aware fallback labels for known Camunda endpoints.
- [quickstart.md](quickstart.md) defines targeted validation scenarios and broader test commands.

## Constitution Check After Design

- **Operational Proof Over Intent**: PASS. Design preserves high-level operational progress and does not alter confirmation, wait, mutation, or final outcome behavior.
- **CLI-First, Script-Safe Interfaces**: PASS. Contracts explicitly keep transient activity out of machine-oriented output modes.
- **Tests and Validation Are Mandatory**: PASS. Quickstart and contracts define activity hierarchy, fallback label, command regression, and full validation expectations.
- **Documentation Matches User Behavior**: PASS. Documentation review is part of validation; generated docs are updated only if command help or documented activity behavior changes.
- **Small, Compatible, Repository-Native Changes**: PASS. Design uses existing activity writer/sink patterns, context propagation, HTTP transport wiring, command progress renderers, and service wait scopes.
