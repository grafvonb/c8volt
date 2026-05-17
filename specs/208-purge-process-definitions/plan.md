# Implementation Plan: Ops Purge All Process Definitions

**Branch**: `208-purge-process-definitions` | **Date**: 2026-05-16 | **Spec**: [spec.md](spec.md)  
**Input**: Feature specification from `specs/208-purge-process-definitions/spec.md`  
**Issue**: [#208](https://github.com/grafvonb/c8volt/issues/208)  
**Mandatory Implementation Context**: `specs/ralph-implementation-rules.md` MUST be passed to task generation and every Ralph run as `--implementation-context specs/ralph-implementation-rules.md`.

## Summary

Add `c8volt ops purge all-process-definitions` as a destructive, auditable operational cleanup workflow with alias `all-pds`. The command discovers process-definition versions through the same selection semantics as `get pd`, freezes unique candidate process-definition keys, reuses the same deterministic process-definition delete preflight and deletion source of truth as `delete pd`, then either reports a dry-run plan or executes confirmed deletion through established process-definition deletion behavior. The implementation should extend the existing #186/#187/#199 ops workflow/report foundation rather than shelling out to `get pd | delete pd`.

## Technical Context

**Language/Version**: Go, repository module `github.com/grafvonb/c8volt`  
**Primary Dependencies**: Cobra CLI, existing c8volt public facade packages, `c8volt/ops`, `c8volt/processdefinition`, `c8volt/resource`, internal ops services, internal process-definition search services, internal resource delete planning/deletion services, generated Camunda clients only through service adapters  
**Storage**: N/A for product state; optional Markdown or JSON audit report file output requested by the user  
**Testing**: `go test` with command, facade, service, report, contract, and subprocess exit-code tests; final validation through targeted tests and `make test` when implementation is complete  
**Target Platform**: Cross-platform CLI, with current local planning on macOS  
**Project Type**: Go CLI  
**Performance Goals**: Use existing process-definition paging/filtering and delete worker, fail-fast, no-worker-limit, and no-wait controls; do not introduce unbounded duplicate delete submissions  
**Constraints**: Keep stdout deterministic for `--automation --json`; preserve existing `get pd` and `delete pd` behavior; freeze candidate process-definition keys before delete preflight; use candidate terminology in human-facing output  
**Scale/Scope**: One predefined ops purge workflow with process-definition filters, dry-run, confirmed deletion, automation, JSON/human output, Markdown/JSON audit reporting, docs, and command contract metadata

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The workflow must discover candidates, build the existing delete preflight before mutation, preserve wait/no-wait semantics, and report planned, deleted, partially failed, blocked, or failed outcomes.
- **CLI-First, Script-Safe Interfaces**: PASS. Behavior is exposed through Cobra commands, inherited root flags, shared command contracts, stable JSON/report models, automation handling, and documented exit-code classes.
- **Tests and Validation Are Mandatory**: PASS. Tasks must include command registration, process-definition filter validation, discovery, delete preflight, destructive execution, automation JSON, reports, error classification, docs, and regression tests.
- **Documentation Matches User Behavior**: PASS. User-facing command metadata requires generated CLI docs via `make docs-content`; README-facing examples should be checked and updated when needed.
- **Small, Compatible, Repository-Native Changes**: PASS. Existing `cmd/ops_purge.go`, #186/#187/#199 ops workflow/report code, process-definition selection code, resource facade, and process-definition delete planning/deletion services are the extension points.

No constitution violations are planned.

## Project Structure

### Documentation (this feature)

```text
specs/208-purge-process-definitions/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── progress.md
├── contracts/
│   └── all-process-definitions-purge-cli.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── ops_purge.go
├── ops_purge_all_processdefinitions.go
├── ops_purge_all_processdefinitions_test.go
├── cmd_views_ops_purge_all_processdefinitions.go
├── cmd_views_ops_notices.go
├── ops_contract.go
├── ops_contract_test.go
├── command_contract.go
├── command_contract_test.go
├── get_processdefinition.go
├── delete_processdefinition.go
└── generated-doc-triggering command metadata

c8volt/
├── ops/
│   ├── api.go
│   ├── client.go
│   ├── convert.go
│   ├── model.go
│   └── client_test.go
├── processdefinition/
│   └── existing search facade models and conversions
└── resource/
    └── existing process-definition delete planning/deletion facade primitives

internal/services/
├── ops/
│   ├── api.go
│   ├── all_process_definitions_purge.go
│   └── all_process_definitions_purge_test.go
├── processdefinition/
│   └── existing version-neutral search service and version adapters
└── resource/
    └── existing process-definition delete planning/deletion support
```

**Structure Decision**: Extend the existing ops purge command and ops facade/service layout. Process-definition discovery should reuse the `get pd` selection path; active process-instance impact, force cancel-before-delete, history cleanup, process-definition deletion, waiting, and worker behavior should remain owned by the existing process-definition delete planning/deletion source of truth in the resource layer.

## Complexity Tracking

No constitution violations or extra architectural complexity are currently justified. If implementation finds that #186/#187/#199 shared ops helpers already cover report and workflow orchestration fully, #208-specific code should stay limited to process-definition selection request/result models, command wiring, discovery-to-delete orchestration, and rendering.

## Phase 0: Research

See [research.md](research.md).

## Phase 1: Design

See [data-model.md](data-model.md), [quickstart.md](quickstart.md), and [contracts/all-process-definitions-purge-cli.md](contracts/all-process-definitions-purge-cli.md).

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The design freezes candidate process-definition keys, validates the existing delete preflight, reuses confirmation/deletion/wait behavior, and reports every final outcome.
- **CLI-First, Script-Safe Interfaces**: PASS. Human, JSON, automation, dry-run, report-file, alias, and exit-code behavior are explicit.
- **Tests and Validation Are Mandatory**: PASS. The task plan must include command, facade/service, report, automation, exit-code, docs, and regression tests before work is complete.
- **Documentation Matches User Behavior**: PASS. Generated docs refresh is required after command metadata changes.
- **Small, Compatible, Repository-Native Changes**: PASS. All planned files and APIs stay inside existing command, facade, ops service, process-definition service, and resource service boundaries.

## Implementation Notes For Ralph

- Launch Ralph only with `--implementation-context specs/ralph-implementation-rules.md`.
- Each Ralph iteration must read `specs/ralph-implementation-rules.md`, this `plan.md`, `spec.md`, `tasks.md`, and `progress.md` when present.
- Every commit subject for this feature must follow Conventional Commits and end with `#208`.
