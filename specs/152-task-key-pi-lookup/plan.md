# Implementation Plan: Resolve Process Instance From User Task Key

**Branch**: `152-task-key-pi-lookup` | **Date**: 2026-04-30 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/152-task-key-pi-lookup/spec.md`

## Summary

Add repeatable `--has-user-tasks` as a lookup selector on `get process-instance` / `get pi`. For Camunda 8.8 and 8.9, resolve owning process-instance keys through tenant-aware native generated Camunda v2 user-task search, then pass those resolved keys into the existing keyed process-instance lookup and rendering path. Camunda 8.7 must reject `--has-user-tasks` explicitly, and the command must reject combinations with other selectors, search filters, `--total`, or `--limit`.

## Technical Context

**Language/Version**: Go, repository current module toolchain  
**Primary Dependencies**: Cobra command tree, generated Camunda v2 clients, existing c8volt facade and internal service packages  
**Storage**: N/A  
**Testing**: Go tests through targeted `go test` packages and final `make test`  
**Target Platform**: CLI on the repository's supported platforms  
**Project Type**: Go CLI  
**Performance Goals**: Task-key lookup performs one tenant-aware user-task search per user task key plus the existing keyed process-instance lookup; no fallback API fan-out  
**Constraints**: Use only native Camunda v2 user-task search for 8.8/8.9; reject 8.7; do not call Tasklist or Operate for user-task resolution; preserve existing output shape after resolution  
**Scale/Scope**: One or more user task keys resolve to owning process instances per command invocation

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The command must only report success after resolving the user task and fetching/rendering the owning process instance through existing lookup behavior.
- **CLI-First, Script-Safe Interfaces**: PASS. The change is exposed as a Cobra flag with explicit mutual-exclusion validation and stable output behavior.
- **Tests and Validation Are Mandatory**: PASS. The plan requires versioned service tests, command validation tests, output tests, docs contract tests, targeted Go tests, and `make test`.
- **Documentation Matches User Behavior**: PASS. Help, README, and generated CLI docs are included in scope.
- **Small, Compatible, Repository-Native Changes**: PASS. The implementation reuses the existing process-instance command and service/facade patterns; no parallel command family is introduced.

## Project Structure

### Documentation (this feature)

```text
specs/152-task-key-pi-lookup/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-get-pi-task-key.md
└── tasks.md
```

### Source Code (repository root)

```text
c8volt/
├── client.go
├── process/
│   ├── api.go
│   └── client.go
└── task/
    └── api.go

cmd/
├── get_processinstance.go
├── get_processinstance_test.go
└── cmd_views_processinstance.go

internal/
├── domain/
│   └── usertask.go
└── services/
    ├── processinstance/
    │   └── api.go
    └── usertask/
        ├── api.go
        ├── factory.go
        ├── v87/
        ├── v88/
        └── v89/

docsgen/
└── main.go

README.md
docs/cli/
```

**Structure Decision**: Keep command behavior in `cmd/get_processinstance.go`, expose any needed facade method through the existing public client surface, and place version-specific native user-task lookup in `internal/services/usertask` so process-instance lookup remains responsible for fetching/rendering the final process instance.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

- Data model: [data-model.md](./data-model.md)
- CLI contract: [contracts/cli-get-pi-task-key.md](./contracts/cli-get-pi-task-key.md)
- Quickstart and verification scenarios: [quickstart.md](./quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The design requires tests that assert the user-task call resolves before process-instance rendering and that 8.7 does not perform fallback calls.
- **CLI-First, Script-Safe Interfaces**: PASS. The CLI contract defines accepted commands, invalid combinations, exit behavior, and output preservation.
- **Tests and Validation Are Mandatory**: PASS. The task plan will include command-level tests, version service tests, docs tests, generated-doc validation, targeted Go tests, and `make test`.
- **Documentation Matches User Behavior**: PASS. README/help/generated docs are explicit deliverables.
- **Small, Compatible, Repository-Native Changes**: PASS. The design extends existing service and facade layers without introducing a separate command family.

## Complexity Tracking

No constitution violations or additional complexity exceptions are required.
