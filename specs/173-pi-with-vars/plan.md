# Implementation Plan: Process Instance Variable Output

**Branch**: `173-pi-with-vars` | **Date**: 2026-05-05 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/173-pi-with-vars/spec.md`

## Summary

Add `c8volt get pi --key <key> --with-vars` for keyed process-instance inspection. The implementation should reuse the existing keyed process-instance lookup, enrichment facade, and one-line rendering patterns, add a process-instance variable model and service lookup, query Camunda `/variables/search` for variables whose `processInstanceKey` and `scopeKey` both match the selected key, sort variables by name, and render human and JSON output with stable truncation metadata. Human output keeps received values full by default and only applies CLI shortening when `--var-value-limit <chars>` is provided.

## Technical Context

**Language/Version**: Go, repository current module toolchain  
**Primary Dependencies**: Cobra command tree, existing process facade, process-instance service packages, generated Camunda v8.8/v8.9 clients, raw JSON decoding where generated variable result structs omit value metadata, shared rendering/error helpers, docs generation path  
**Storage**: N/A  
**Testing**: Go tests through targeted `go test ./cmd`, `go test ./c8volt/process`, versioned `internal/services/processinstance` tests, docs generation checks, and final `make test`  
**Target Platform**: CLI on the repository's supported platforms  
**Project Type**: Go CLI  
**Performance Goals**: Variable enrichment should run only for explicitly selected keyed process instances, avoid variable lookups when `--with-vars` is not set, request process-scope variables directly from the API, and preserve stable output ordering without excessive in-memory work  
**Constraints**: Preserve default `get pi` behavior; limit `--with-vars` to keyed process-instance lookup in this iteration; fail clearly instead of rendering partial enrichment on lookup errors; use `--var-value-limit <chars>` only as an opt-in human display limit; keep JSON values as received; account for generated client gaps around value/truncation fields  

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The feature fetches process instances first, then enriches only after variable search returns process-scope variables, with failures surfaced clearly.
- **CLI-First, Script-Safe Interfaces**: PASS. The plan extends an existing command with explicit flags, predictable validation, human output, and stable JSON output.
- **Tests and Validation Are Mandatory**: PASS. The plan requires command/view/facade/service tests plus docs generation and final repository validation.
- **Documentation Matches User Behavior**: PASS. `--with-vars`, `--var-value-limit`, JSON behavior, and unsupported combinations are user-visible and require README/generated docs updates.
- **Small, Compatible, Repository-Native Changes**: PASS. The design mirrors existing incident enrichment and service conversion patterns instead of introducing a parallel command or unrelated abstraction.

## Project Structure

### Documentation (this feature)

```text
specs/173-pi-with-vars/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-get-process-instance-vars.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_processinstance.go
├── get_processinstance_test.go
├── cmd_views_processinstance_vars.go
├── cmd_views_get_test.go
└── command_contract_test.go

c8volt/
└── process/
    ├── api.go
    ├── client.go
    ├── client_test.go
    ├── convert.go
    └── model.go

internal/
├── domain/
│   └── processinstance.go
└── services/
    └── processinstance/
        ├── api.go
        ├── v88/
        └── v89/

README.md
docs/
docsgen/
```

**Structure Decision**: Keep command validation and mode selection in `cmd/get_processinstance.go`; add variable-specific human/JSON rendering beside the existing incident renderer; add facade-level variable enrichment to `c8volt/process`; add service-level variable search to the process-instance service packages; use raw JSON response handling only where generated v8.8/v8.9 models omit returned value/truncation fields.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

- Data model: [data-model.md](./data-model.md)
- CLI contract: [contracts/cli-get-process-instance-vars.md](./contracts/cli-get-process-instance-vars.md)
- Quickstart and verification scenarios: [quickstart.md](./quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The design requires service and facade tests that prove process-scope filtering, per-key association, and lookup failure handling.
- **CLI-First, Script-Safe Interfaces**: PASS. The contract defines valid/invalid flag combinations, human value display, truncation markers, JSON fidelity, and unchanged default output.
- **Tests and Validation Are Mandatory**: PASS. The task list will include failing-first tests, targeted service/facade/command runs, docs generation, and `make test`.
- **Documentation Matches User Behavior**: PASS. The quickstart and tasks include README and generated CLI documentation updates for all new flags and examples.
- **Small, Compatible, Repository-Native Changes**: PASS. The design extends existing process-instance command and enrichment boundaries, with focused raw response decoding only to compensate for generated-client field omissions.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Focused raw variable search response decoding | v8.8/v8.9 generated `VariableResultBase` exposes identity/scope fields but not the returned value or truncation state required by the issue | Relying only on generated structs would make JSON/human value output impossible; full client regeneration is larger and riskier for this feature slice unless implementation confirms the generated source has already been updated |
