# Implementation Plan: get incident process-instance key output

**Branch**: `198-pi-keys-only` | **Date**: 2026-05-10 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/198-pi-keys-only/spec.md`

## Summary

Add a command-local `--pi-keys-only` output mode to `c8volt get incident` so incident lookup/search results can feed process-instance commands. The implementation stays inside the existing Go/Cobra incident command and view helpers, preserves duplicate process instance keys in incident output, rejects ambiguous output combinations locally, updates docs/contracts, and includes a small local cleanup to dedupe `delete pi` flag/stdin keys at the same command boundary already used by `cancel pi`.

## Technical Context

**Language/Version**: Go, existing repository toolchain  
**Primary Dependencies**: Cobra CLI, existing command render-mode helpers, `c8volt/incident` result models, existing incident search/list service and command tests  
**Storage**: N/A, command reads Camunda incident data and renders line-oriented output  
**Testing**: Go tests through targeted command packages, docs generation tests, then `make test` before commit  
**Target Platform**: CLI for local operators and automation on supported c8volt platforms  
**Project Type**: Single Go CLI project  
**Performance Goals**: `--pi-keys-only` must use existing incident lookup/search paths without additional remote calls and must preserve existing paging/limit behavior  
**Constraints**: Preserve existing `--keys-only` incident-key output, preserve duplicate process instance keys for `--pi-keys-only`, reject incompatible modes before remote calls, keep delete dedupe cleanup local and non-observable except for duplicate planning counts  
**Scale/Scope**: One command-local flag, render/validation updates, focused command/view/docs tests, generated CLI docs, README example, and one small `delete pi` dedupe cleanup

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Operational Proof Over Intent | PASS | Feature is read-only output selection; destructive downstream safety is handled by predictable line output and local validation. |
| II. CLI-First, Script-Safe Interfaces | PASS | Work adds a script-safe flag to an existing command and keeps incompatible output modes explicit. |
| III. Tests and Validation Are Mandatory | PASS | Plan requires command, view, paging, validation, docs, and delete cleanup tests. |
| IV. Documentation Matches User Behavior | PASS | README and generated CLI docs are in scope for the new pipeline flag. |
| V. Small, Compatible, Repository-Native Changes | PASS | Uses existing incident command, render helpers, validation style, and process-instance key dedupe pattern. |

Post-design re-check remains PASS. No new service boundary, dependency, or command hierarchy is planned.

## Project Structure

### Documentation (this feature)

```text
specs/198-pi-keys-only/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-get-incident-pi-keys-only.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_incident.go
├── get_incident_search.go
├── get_incident_test.go
├── cmd_views_get.go
├── cmd_views_get_test.go
├── command_contract_test.go
├── delete_processinstance.go
└── delete_test.go

docsgen/
└── main_test.go

docs/cli/
├── c8volt_get.md
├── c8volt_get_incident.md
└── index.md

README.md
AGENTS.md
```

**Structure Decision**: Use the existing single-project Go CLI structure. Flag registration and validation stay in `cmd/get_incident.go`; collected output rendering stays in `cmd/cmd_views_get.go`; incremental search rendering stays in `cmd/get_incident_search.go`; documentation remains in README and generated CLI docs; the optional delete cleanup stays in `cmd/delete_processinstance.go`.

## Phase 0: Research

Research output is captured in [research.md](./research.md). Key decisions:

- Add a command-local boolean flag rather than a new global render mode.
- Render process instance keys from the same incident result items already used by default, JSON, and `--keys-only` output.
- Preserve duplicate process instance keys in `get incident --pi-keys-only` output.
- Skip incident rows without process instance keys in `--pi-keys-only` output.
- Reject incompatible output modifiers with local validation before remote calls.
- Align `delete pi` duplicate stdin handling with `cancel pi` only at the command boundary.

## Phase 1: Design & Contracts

Design artifacts:

- [data-model.md](./data-model.md)
- [contracts/cli-get-incident-pi-keys-only.md](./contracts/cli-get-incident-pi-keys-only.md)
- [quickstart.md](./quickstart.md)

## Complexity Tracking

No constitution violations or new parallel structures are planned.
