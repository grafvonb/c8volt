# Implementation Plan: Get Incident Command

**Branch**: `185-get-incident-command` | **Date**: 2026-05-09 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/185-get-incident-command/spec.md`

## Summary

Add first-class read-only `c8volt get incident` commands for keyed incident lookup and searchable incident listing. The implementation extends the existing Go/Cobra `get` command family, reuses `c8volt/process.ProcessInstanceIncidentDetail`, `internal/services/incident`, and `internal/services/incidentfilter`, adds explicit version-aware incident search APIs, and keeps process-instance incident view/filter concepts separate from plain incident filters. The command must support human, JSON, keys-only, and exact total output with pagination-correct local filtering whenever backend semantics are not sufficient.

## Technical Context

**Language/Version**: Go, existing repository toolchain
**Primary Dependencies**: Cobra CLI, existing generated Camunda clients, `internal/services/incident`, `internal/services/incidentfilter`, existing c8volt process facade and command view helpers
**Storage**: N/A, command reads Camunda incident data and renders it locally
**Testing**: Go tests through targeted `go test` packages, then `make test` before commit
**Target Platform**: CLI for local operator and automation use on supported c8volt platforms
**Project Type**: Single Go CLI project
**Performance Goals**: Search/list mode must honor existing limit and paging conventions; local filtering must page all relevant candidates until exhausted or the explicit limit is reached; totals must be exact after all filters
**Constraints**: Preserve existing `get pi` incident semantics, avoid process-instance-only flags on `get incident`, fail unsupported versions before unsupported operations, avoid known broken v8.8 scoped request shapes, keep incident search in the incident service boundary
**Scale/Scope**: One `get` subcommand with aliases, keyed lookup, incident search filters, creation-time filters, output modifiers, documentation, generated docs, and regression coverage for existing process-instance incident behavior

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Operational Proof Over Intent | PASS | Feature is read-only; output and totals must reflect fully applied filters before reporting. |
| II. CLI-First, Script-Safe Interfaces | PASS | New command uses Cobra, stable flags, aliases, exit behavior, human output, JSON output, keys-only output, and total output. |
| III. Tests and Validation Are Mandatory | PASS | Plan requires command, service, facade, compatibility, pagination, docsgen, and regression tests. |
| IV. Documentation Matches User Behavior | PASS | README, help text, and generated CLI docs are in scope. |
| V. Small, Compatible, Repository-Native Changes | PASS | Work follows existing `get` command, process facade, incident service, view helper, and incident filter patterns. |

No constitution violations are expected. Re-check after design remains PASS because Phase 1 artifacts keep the same repository-native structure and verification requirements.

## Project Structure

### Documentation (this feature)

```text
specs/185-get-incident-command/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-get-incident-contract.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get.go
├── get_incident.go
├── get_incident_test.go
├── get_processinstance.go
├── get_processinstance_validation.go
├── get_processinstance_test.go
├── cmd_views_get.go
├── cmd_views_processinstance_incidents.go
├── cmd_views_get_test.go
├── command_contract.go
└── command_contract_test.go

c8volt/process/
├── api.go
├── client.go
├── convert.go
├── filter.go
├── model.go
└── client_test.go

internal/services/incident/
├── api.go
├── factory.go
├── v87/
├── v88/
└── v89/

internal/services/incidentfilter/
└── incidentfilter.go

README.md
docs/cli/
docsgen/
```

**Structure Decision**: Use the existing single-project Go CLI structure. Command parsing and validation live in `cmd`, public facade orchestration and result models live in `c8volt/process`, version-specific generated-client calls live in `internal/services/incident`, reusable enum/message matching remains in `internal/services/incidentfilter`, and generated documentation remains under `docs/cli`.

## Phase 0: Research

Research output is captured in [research.md](./research.md). Key decisions:

- Add `get incident` under the existing `get` root with aliases `incidents` and `inc`.
- Model keyed lookup and search/list as one incident query contract, while rejecting ambiguous keyed-plus-filter combinations locally.
- Extend the incident service API for direct incident search/list rather than placing search logic in command code.
- Use v8.9 server-side filters where semantics are safe and use v8.8 compatibility paths that avoid broken scoped `filter` request shapes.
- Treat error-message search as local case-insensitive substring filtering unless backend behavior is confirmed compatible.
- Count totals after all local filters, using backend totals only when no local post-filter changes the result set.

## Phase 1: Design & Contracts

Design artifacts:

- [data-model.md](./data-model.md)
- [contracts/cli-get-incident-contract.md](./contracts/cli-get-incident-contract.md)
- [quickstart.md](./quickstart.md)

## Complexity Tracking

No constitution violations or new parallel structures are planned.
