# Implementation Plan: Keyed Process-Instance Incident Details

**Branch**: `154-get-pi-incidents` | **Date**: 2026-05-02 | **Spec**: [spec.md](/Users/adam.boczek/.codex/worktrees/5d96/c8volt/specs/154-get-pi-incidents/spec.md)
**Input**: Feature specification from `/specs/154-get-pi-incidents/spec.md`

## Summary

Add `--with-incidents` to `c8volt get process-instance` / `get pi` for direct keyed lookup only. The implementation should keep command code thin, extend the process facade and process-instance service interfaces with tenant-aware incident search, implement supported Camunda 8.8 and 8.9 paths through generated `SearchProcessInstanceIncidentsWithResponse`, return the existing unsupported-capability style for Camunda 8.7, and render incident messages only when the new flag is requested so existing default output remains unchanged.

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, generated Camunda clients under `internal/clients/camunda/v88/camunda` and `internal/clients/camunda/v89/camunda`, existing process facade/service packages, existing render/error helpers in `cmd/`  
**Storage**: No persistent storage changes; reads current config/profile/root flag tenant through existing command bootstrap  
**Testing**: `go test`, `make test`, command tests under `cmd/`, facade tests under `c8volt/process/`, versioned service tests under `internal/services/processinstance/v87`, `v88`, and `v89`  
**Target Platform**: Cross-platform CLI for local and CI use against supported Camunda versions; incident enrichment supported for `v8.8` and `v8.9`, unsupported explicitly for `v8.7` when tenant-safe keyed enrichment is unavailable  
**Project Type**: CLI  
**Performance Goals**: Keyed lookup with `--with-incidents` performs one incident search per returned key and no incident search when the flag is omitted; multiple keys preserve existing worker behavior for process-instance lookup and may enrich results after the keyed results are known  
**Constraints**: Preserve existing keyed default output, preserve search-mode incident filters, reject `--with-incidents` outside keyed mode, avoid tenant-unsafe direct incident lookup, include configured tenant in incident search filters where supported, update generated CLI docs, finish with targeted tests and `make test`  
**Scale/Scope**: Command validation/rendering in `cmd/get_processinstance.go`, base process-instance rendering in `cmd/cmd_views_get.go`, incident-enriched rendering in `cmd/cmd_views_processinstance_incidents.go`, process public models/facade in `c8volt/process/`, domain/service interfaces in `internal/domain/processinstance.go` and `internal/services/processinstance/api.go`, versioned implementations in `internal/services/processinstance/v87`, `v88`, and `v89`, generated-client contract interfaces in `internal/services/processinstance/v88/contract.go` and `v89/contract.go`, tests in matching packages, docs under `docs/cli/` and README review

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. The feature is read-only but must prove actual incident search request construction, rendered messages, JSON shape, and unsupported-version outcomes through tests.
- **CLI-First, Script-Safe Interfaces**: Pass. The new behavior is exposed through one Cobra flag, preserves existing output when omitted, and provides JSON details for automation.
- **Tests and Validation Are Mandatory**: Pass. The issue requires command, facade, service, output, validation, tenant, and version-specific coverage plus final repository validation.
- **Documentation Matches User Behavior**: Pass. The command help and generated CLI docs must describe the keyed-only flag and incident-message purpose.
- **Small, Compatible, Repository-Native Changes**: Pass. The design extends the existing process facade/service stack rather than adding command-level generated-client calls.

## Project Structure

### Documentation (this feature)

```text
specs/154-get-pi-incidents/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── get-pi-with-incidents.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_processinstance.go
├── cmd_views_get.go
├── cmd_views_processinstance.go
├── get_processinstance_test.go
└── cmd_views_get_test.go

c8volt/process/
├── api.go
├── client.go
├── convert.go
├── model.go
└── client_test.go

internal/domain/
├── processinstance.go
└── processinstance_test.go

internal/services/processinstance/
├── api.go
├── v87/
│   ├── service.go
│   └── service_test.go
├── v88/
│   ├── contract.go
│   ├── convert.go
│   ├── service.go
│   └── service_test.go
└── v89/
    ├── contract.go
    ├── convert.go
    ├── service.go
    └── service_test.go

README.md
docs/cli/
```

**Structure Decision**: Keep incident enrichment inside the existing process-instance slice. `cmd/get_processinstance.go` owns flag validation and decides whether enrichment is requested; `cmd/cmd_views_processinstance_incidents.go` owns enriched human/JSON rendering; `c8volt/process` owns the public incident models and facade method; versioned `internal/services/processinstance` implementations own generated-client incident search and tenant filtering. Default rendering remains in `cmd/cmd_views_get.go` so ordinary process-instance output stays isolated.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/.codex/worktrees/5d96/c8volt/specs/154-get-pi-incidents/research.md).

- Confirmed `v8.8` and `v8.9` generated Camunda clients expose `SearchProcessInstanceIncidentsWithResponse(ctx, processInstanceKey, body, ...)`, `IncidentSearchQuery`, `IncidentFilter`, `IncidentResult`, and `ErrorMessage`.
- Confirmed `v8.7` exposes older general incident search methods, but existing `v87` process-instance direct lookup is explicitly not tenant-safe. This feature should return unsupported for `v8.7` rather than add a tenant-unsafe enrichment path.
- Confirmed the current public `process.ProcessInstance` model has only the boolean incident marker. Incident details should be additive and only appear in an enriched output wrapper when `--with-incidents` is requested.
- Confirmed current human output is a compact one-line row with `inc!`. Incident messages should render as indented `incident <incident-key>:` lines directly below the matching process-instance row only when requested, leaving `oneLinePI` behavior unchanged for default output.
- Confirmed JSON output currently serializes `process.ProcessInstances`. `--with-incidents --json` needs an explicit enriched shape instead of silently changing `process.ProcessInstance` for all JSON callers.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/.codex/worktrees/5d96/c8volt/specs/154-get-pi-incidents/data-model.md)
- [quickstart.md](/Users/adam.boczek/.codex/worktrees/5d96/c8volt/specs/154-get-pi-incidents/quickstart.md)
- [contracts/get-pi-with-incidents.md](/Users/adam.boczek/.codex/worktrees/5d96/c8volt/specs/154-get-pi-incidents/contracts/get-pi-with-incidents.md)

- Add public and domain incident detail models containing incident key, process-instance key, tenant ID, state/type metadata where available, and error message.
- Add a process facade method for incident lookup by process-instance key, plus a helper that enriches a keyed `ProcessInstances` result without altering default `ProcessInstances` JSON.
- Extend `internal/services/processinstance.API` with a tenant-aware process-instance incident search method.
- Extend `v88` and `v89` generated-client interfaces with `SearchProcessInstanceIncidentsWithResponse`; build request bodies with process-instance-key filtering, tenant filtering when configured, and stable page sizing.
- Keep `v87` implementation as explicit unsupported for this method because tenant-safe keyed incident enrichment cannot be guaranteed from the current service boundary.
- Add `--with-incidents` validation in keyed mode only, before any process-instance or incident lookup side effect.
- Add an enriched render path for `--with-incidents` that preserves existing default render path when the flag is absent.
- Regenerate CLI docs through `make docs-content` after help text changes.

### Version Support Matrix

| Version | Incident enrichment | Planned behavior |
|---------|---------------------|------------------|
| `v8.7` | Unsupported | Return existing unsupported-capability style for `--with-incidents` because tenant-safe keyed enrichment is unavailable |
| `v8.8` | Supported | Use generated `SearchProcessInstanceIncidentsWithResponse` and include configured tenant in incident search filter |
| `v8.9` | Supported | Match `v8.8` behavior with the `v89` generated client types |

## Phase 2: Task Planning Approach

Task generation should keep the work in independently verifiable user-story slices:

1. Prepare shared models, service/facade method contracts, and command flag validation first.
2. Deliver User Story 1 as the MVP: supported-version incident search and human-readable incident keys and messages for keyed lookup.
3. Add User Story 2 JSON enrichment with a stable machine-readable wrapper.
4. Add User Story 3 regression coverage for validation and output preservation.
5. Add User Story 4 tenant/version safeguards, docs generation, and final validation.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design requires tests that inspect actual request bodies and actual rendered output.
- **CLI-First, Script-Safe Interfaces**: Still passes. The flag is deterministic, keyed-only, and JSON-capable.
- **Tests and Validation Are Mandatory**: Still passes. Tasks include focused package tests and `make test`.
- **Documentation Matches User Behavior**: Still passes. The command help/docs tasks are explicit.
- **Small, Compatible, Repository-Native Changes**: Still passes. The change extends current process-instance boundaries and keeps generated-client details in versioned services.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
