# Implementation Plan: Resolve Incident Commands

**Branch**: `181-resolve-incident-commands` | **Date**: 2026-05-08 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/181-resolve-incident-commands/spec.md`

## Summary

Add a new state-changing `c8volt resolve` root command family for resolving Camunda incidents by explicit incident key or by process instance key. The implementation extends the existing Go/Cobra CLI, keeps lookup and resolution behavior behind `internal/services/incident`, reuses process facade bulk and wait patterns for per-target reporting, and updates tests plus generated CLI documentation so the command is script-safe and operator-ready.

## Technical Context

**Language/Version**: Go, existing repository toolchain  
**Primary Dependencies**: Cobra CLI, existing generated Camunda clients, existing c8volt service/facade packages  
**Storage**: N/A, command submits Camunda state-changing requests and observes API state  
**Testing**: Go tests through targeted `go test` packages, then `make test` before commit  
**Target Platform**: CLI for local operator and automation use on supported c8volt platforms  
**Project Type**: Single Go CLI project  
**Performance Goals**: Bulk resolution should use existing worker fan-out controls and avoid unbounded polling; process-instance resolution must resolve only incidents discovered at command start  
**Constraints**: Preserve existing command behavior, support human and JSON output, fail unsupported Camunda versions before mutation, keep incident behavior out of `internal/services/processinstance`  
**Scale/Scope**: One root command family with two leaf commands, incident resolution support for Camunda versions with generated endpoints, regression coverage for affected command contracts and process-instance workflows

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Operational Proof Over Intent | PASS | Default behavior waits for incident resolution visibility; `--no-wait` is the explicit opt-out. |
| II. CLI-First, Script-Safe Interfaces | PASS | New commands use Cobra, stable flags, aliases, exit behavior, human output, JSON output, and automation metadata. |
| III. Tests and Validation Are Mandatory | PASS | Plan requires service, facade, command, contract, regression, docsgen, and validation tasks. |
| IV. Documentation Matches User Behavior | PASS | README and generated CLI docs are in scope through `make docs-content`. |
| V. Small, Compatible, Repository-Native Changes | PASS | Work follows existing command/service/facade/view patterns and avoids moving incident behavior into process-instance services. |

No constitution violations are expected. Re-check after design remains PASS because Phase 1 artifacts keep the same repository-native structure and verification requirements.

## Project Structure

### Documentation (this feature)

```text
specs/181-resolve-incident-commands/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-resolve-contract.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── resolve.go
├── resolve_incident.go
├── resolve_processinstance.go
├── cmd_views_resolve.go
├── command_contract.go
├── command_contract_test.go
└── *_test.go

c8volt/process/
├── api.go
├── bulk.go
├── client.go
├── model.go
└── client_test.go

internal/services/incident/
├── api.go
├── factory.go
├── factory_test.go
├── v87/
├── v88/
└── v89/

README.md
docs/cli/
docsgen/
```

**Structure Decision**: Use the existing single-project Go CLI structure. Command parsing and rendering live in `cmd`, public facade orchestration and result models live in `c8volt/process`, version-specific generated-client calls live in `internal/services/incident`, and generated documentation remains under `docs/cli`.

## Phase 0: Research

Research output is captured in [research.md](./research.md). Key decisions:

- Add `resolve` as a distinct root command because the issue defines incident resolution as an operator recovery action rather than a field update.
- Use generated v8.8 and v8.9 incident resolution endpoints through `internal/services/incident`; v8.7 returns an unsupported-version error before mutation.
- Confirm resolution through incident lookup for process-instance-scoped waits and a direct incident state lookup or resolution response check for incident-key waits, depending on the generated client surface.
- Model per-target command results in the process facade so human and JSON output share one data contract.

## Phase 1: Design & Contracts

Design artifacts:

- [data-model.md](./data-model.md)
- [contracts/cli-resolve-contract.md](./contracts/cli-resolve-contract.md)
- [quickstart.md](./quickstart.md)

## Complexity Tracking

No constitution violations or new parallel structures are planned.
