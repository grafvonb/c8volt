# Implementation Plan: Process Instance Incident List Output

**Branch**: `171-pi-incident-list-output` | **Date**: 2026-05-05 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/171-pi-incident-list-output/spec.md`

## Summary

Extend `c8volt get pi --with-incidents` from keyed-only enrichment to list/search output while preserving keyed behavior, paging, and JSON shape. Reuse the existing process-instance incident enrichment facade and human rendering helpers, add human-only message truncation, shorten the incident line prefix for both `get pi` and `walk pi`, and update validation/tests/docs so the new list behavior is script-safe.

## Technical Context

**Language/Version**: Go, repository current module toolchain  
**Primary Dependencies**: Cobra command tree, existing process facade, process-instance service API, incident enrichment helpers, process-instance paging helpers, shared rendering/error helpers, generated Camunda clients already mapped into domain process-instance models  
**Storage**: N/A  
**Testing**: Go tests through targeted `go test ./cmd`, `go test ./c8volt/process`, relevant versioned process-instance service packages if touched, docs generation checks, and final `make test`  
**Target Platform**: CLI on the repository's supported platforms  
**Project Type**: Go CLI  
**Performance Goals**: Incident enrichment should run only for the process instances selected for output, preserve page and `--limit` boundaries, and avoid extra incident lookups when `--with-incidents` is not set  
**Constraints**: Preserve output without `--with-incidents`; preserve keyed `get pi --with-incidents` behavior except human prefix shortening; keep `--with-incidents` invalid with `--total`; make `--incident-message-limit` human-only, non-negative, and dependent on `--with-incidents`; keep JSON messages full  
**Scale/Scope**: One process-instance get command family, shared incident human rendering used by get and walk, command docs/help, and close tests for list rendering, validation, JSON, truncation, and paging compatibility

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The feature renders incident details only after direct incident lookup has enriched the listed process instances, and it explicitly represents indirect markers when direct lookup is empty.
- **CLI-First, Script-Safe Interfaces**: PASS. The plan extends existing flags and output modes, preserves exit behavior, keeps JSON full fidelity, and adds validation for unsafe flag combinations.
- **Tests and Validation Are Mandatory**: PASS. The plan requires command/view/facade regression coverage, docs generation, targeted Go tests, and final `make test`.
- **Documentation Matches User Behavior**: PASS. Help text, README examples, and generated CLI docs must change because `--with-incidents` list/search behavior and the new limit flag are user-visible.
- **Small, Compatible, Repository-Native Changes**: PASS. The design reuses existing enrichment, paging, rendering, and service API paths instead of introducing a parallel incident lookup flow.

## Project Structure

### Documentation (this feature)

```text
specs/171-pi-incident-list-output/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-get-process-instance-incidents.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_processinstance.go
├── get_processinstance_test.go
├── cmd_views_processinstance_incidents.go
├── cmd_views_get_test.go
├── cmd_views_walk_incidents.go
├── walk_test.go
└── command_contract_test.go

c8volt/
└── process/
    ├── client.go
    ├── client_test.go
    └── model.go

internal/
└── services/
    └── processinstance/
        ├── api.go
        ├── v87/
        ├── v88/
        └── v89/

README.md
docs/
docsgen/
```

**Structure Decision**: Keep command validation and mode selection in `cmd/get_processinstance.go`; keep human and JSON incident rendering in `cmd/cmd_views_processinstance_incidents.go` and the shared `incidentHumanLine` helper used by walk rendering; reuse facade-level enrichment in `c8volt/process` for keyed and list/search outputs; update README/generated docs through the existing documentation path.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

- Data model: [data-model.md](./data-model.md)
- CLI contract: [contracts/cli-get-process-instance-incidents.md](./contracts/cli-get-process-instance-incidents.md)
- Quickstart and verification scenarios: [quickstart.md](./quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The contract requires direct incident lines to be associated with the listed process instance and indirect markers to get explicit notes plus one follow-up warning.
- **CLI-First, Script-Safe Interfaces**: PASS. The contract defines valid and invalid flag combinations, human vs JSON behavior, truncation, paging compatibility, and unchanged default output.
- **Tests and Validation Are Mandatory**: PASS. The task list will include failing-first command/view tests, facade regression coverage where needed, docs generation, targeted `go test ./cmd ./c8volt/process`, and `make test`.
- **Documentation Matches User Behavior**: PASS. User-facing command examples and generated CLI documentation are in scope.
- **Small, Compatible, Repository-Native Changes**: PASS. The design extends the current keyed enrichment and renderer rather than adding new service abstractions unless tests reveal a missing reusable boundary.

## Complexity Tracking

No constitution violations or additional complexity exceptions are required.
