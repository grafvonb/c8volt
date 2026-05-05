# Implementation Plan: Process Instance Incident Expectation

**Branch**: `170-process-incident-expect` | **Date**: 2026-05-05 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/170-process-incident-expect/spec.md`

## Summary

Add `--incident true|false` to `c8volt expect process-instance` and `c8volt expect pi` while preserving existing `--state`, stdin key pipelining, read-only classification, and automation unsupported behavior. Extend the existing expectation validation and process-instance wait path so incident expectations can run alone or compose with state expectations, then update help, generated docs, and close tests across command validation, facade mapping, waiter behavior, and versioned service contracts.

## Technical Context

**Language/Version**: Go, repository current module toolchain  
**Primary Dependencies**: Cobra command tree, existing process facade, process-instance service API, service waiter, shared command rendering/error helpers, generated Camunda clients already mapped into domain process-instance models  
**Storage**: N/A  
**Testing**: Go tests through targeted `go test ./cmd`, `go test ./c8volt/process`, `go test ./internal/services/processinstance/waiter`, versioned process-instance service packages, and final `make test`  
**Target Platform**: CLI on the repository's supported platforms  
**Project Type**: Go CLI  
**Performance Goals**: Incident expectation polling should use the existing wait cadence and worker controls; adding incident checks must not add extra lookups beyond the selected process-instance fetches needed by the current wait loop  
**Constraints**: Preserve `--state absent`, canceled/terminated compatibility, stdin `-` key handling, shared envelope behavior, read-only command classification, and automation unsupported behavior; accept exactly `true` and `false` for `--incident`; missing instances must not satisfy `--incident false`  
**Scale/Scope**: One expectation command family under `c8volt expect`, with docs/help/test updates for incident expectation behavior

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The feature waits for observed process-instance incident state before reporting success.
- **CLI-First, Script-Safe Interfaces**: PASS. The plan extends an existing flag-based CLI workflow, preserves stdin composition, and keeps failure modes script-safe.
- **Tests and Validation Are Mandatory**: PASS. The plan requires close command tests, facade tests, waiter/service tests, docs/help checks, targeted Go tests, and `make test`.
- **Documentation Matches User Behavior**: PASS. Help text, README/docs examples, and generated CLI documentation are in scope because the command surface changes.
- **Small, Compatible, Repository-Native Changes**: PASS. The design reuses the existing Cobra command, facade, service API, and waiter path instead of adding a parallel flow.

## Project Structure

### Documentation (this feature)

```text
specs/170-process-incident-expect/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-expect-process-incident.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── expect.go
├── expect_processinstance.go
├── expect_test.go
├── process_api_stub_test.go
└── command_contract_test.go

c8volt/
└── process/
    ├── api.go
    ├── bulk.go
    ├── client.go
    ├── model.go
    └── client_test.go

internal/
├── domain/
└── services/
    └── processinstance/
        ├── api.go
        ├── waiter/
        │   ├── waiter.go
        │   └── waiter_test.go
        ├── v87/
        ├── v88/
        └── v89/

README.md
docs/
docsgen/
```

**Structure Decision**: Keep user-facing wiring in `cmd/expect_processinstance.go`, add any public facade types/methods in `c8volt/process`, extend the internal service API and waiter in `internal/services/processinstance`, and let versioned services delegate to the shared waiter as they do for state waits. Update docs through repository generation rather than hand-editing generated command docs where possible.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

- Data model: [data-model.md](./data-model.md)
- CLI contract: [contracts/cli-expect-process-incident.md](./contracts/cli-expect-process-incident.md)
- Quickstart and verification scenarios: [quickstart.md](./quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The contract requires success only after every selected present instance matches the requested incident expectation, and explicitly rejects absence as success for `--incident false`.
- **CLI-First, Script-Safe Interfaces**: PASS. The CLI contract covers accepted values, missing expectation failure, stdin `-`, combined flags, help output, and existing compatibility semantics.
- **Tests and Validation Are Mandatory**: PASS. The task list will include failing-first tests for command validation, pipelining, waiter behavior, facade mapping, versioned service contract conformance, targeted Go tests, docs generation, and `make test`.
- **Documentation Matches User Behavior**: PASS. Help, README/docs examples, and generated CLI docs are mandatory because users discover the new flag through command documentation.
- **Small, Compatible, Repository-Native Changes**: PASS. The design extends current waiter/service/facade contracts without introducing a second process-instance polling abstraction.

## Complexity Tracking

No constitution violations or additional complexity exceptions are required.
