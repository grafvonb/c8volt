# Implementation Plan: Tasklist V1 Fallback For Task-Key Process-Instance Lookup

**Branch**: `162-tasklist-v1-fallback` | **Date**: 2026-05-03 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/162-tasklist-v1-fallback/spec.md`

## Summary

Extend the existing `get process-instance` / `get pi --has-user-tasks` lookup so Camunda 8.8 and 8.9 keep using the current Camunda v2 user-task search first, then use Tasklist API V1 only when that primary lookup returns a not-found style miss. A successful fallback extracts the task's owning process-instance key and reuses the existing keyed process-instance lookup and rendering path. Camunda 8.7 remains explicitly unsupported, non-not-found primary/fallback failures remain visible, and help/docs must stop claiming that no Tasklist fallback exists.

## Technical Context

**Language/Version**: Go, repository current module toolchain  
**Primary Dependencies**: Cobra command tree, generated Camunda v2 clients, generated Tasklist V1 clients, existing c8volt process facade and versioned internal service packages  
**Storage**: N/A  
**Testing**: Go tests through targeted `go test` packages and final `make test`  
**Target Platform**: CLI on the repository's supported platforms  
**Project Type**: Go CLI  
**Performance Goals**: Primary-resolved task keys perform no fallback calls; primary misses perform at most one Tasklist V1 search per task key before existing process-instance lookup  
**Constraints**: Keep v2 lookup first; call Tasklist V1 only for primary not-found/empty results; preserve 8.7 unsupported behavior; preserve direct process-instance output shape after resolution; do not mask auth/config/server failures as not found  
**Scale/Scope**: One or more user task keys resolve to owning process instances per command invocation, with each key independently eligible for fallback

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. Success is only reported after task ownership is resolved and the owning process instance is fetched/rendered through the existing lookup path.
- **CLI-First, Script-Safe Interfaces**: PASS. The command surface and `--has-user-tasks` selector stay stable; only its lookup contract changes for primary misses.
- **Tests and Validation Are Mandatory**: PASS. The plan requires versioned service tests, command-level regression tests, documentation/help tests, targeted Go tests, and `make test`.
- **Documentation Matches User Behavior**: PASS. Help, README, and generated CLI docs are explicitly in scope because current text says no fallback exists.
- **Small, Compatible, Repository-Native Changes**: PASS. The design extends existing versioned user-task services and generated-client patterns without adding a parallel command family.

## Project Structure

### Documentation (this feature)

```text
specs/162-tasklist-v1-fallback/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-get-pi-tasklist-fallback.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_processinstance.go
├── get_processinstance_test.go
└── get_processinstance_user_tasks.go

c8volt/
└── process/
    ├── api.go
    ├── client.go
    └── client_test.go

internal/
├── clients/
│   └── camunda/
│       ├── v88/
│       │   ├── camunda/
│       │   └── tasklist/
│       └── v89/
│           ├── camunda/
│           └── tasklist/
├── domain/
└── services/
    └── usertask/
        ├── api.go
        ├── factory.go
        ├── v87/
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
docs/
docsgen/
```

**Structure Decision**: Keep `cmd/get_processinstance.go` and the process facade as the caller-facing contract. Implement fallback resolution inside `internal/services/usertask/v88` and `internal/services/usertask/v89` by adding Tasklist V1 client dependencies next to the current Camunda v2 client dependency. Leave `internal/services/usertask/v87` as unsupported.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

- Data model: [data-model.md](./data-model.md)
- CLI contract: [contracts/cli-get-pi-tasklist-fallback.md](./contracts/cli-get-pi-tasklist-fallback.md)
- Quickstart and verification scenarios: [quickstart.md](./quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The design verifies resolution path, fallback gating, and final process-instance rendering behavior instead of relying on internal intent.
- **CLI-First, Script-Safe Interfaces**: PASS. The CLI selector remains unchanged and error contracts are documented for scripts.
- **Tests and Validation Are Mandatory**: PASS. The task list will include service, command, docs, generated-docs, targeted Go tests, and `make test`.
- **Documentation Matches User Behavior**: PASS. Documentation changes are required because user-visible fallback behavior changes.
- **Small, Compatible, Repository-Native Changes**: PASS. The design reuses generated clients, existing config normalization, existing domain errors, and existing process-instance lookup.

## Complexity Tracking

No constitution violations or additional complexity exceptions are required.
