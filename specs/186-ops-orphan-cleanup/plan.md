# Implementation Plan: Ops Purge Orphan Process Instances

**Branch**: `186-ops-orphan-cleanup` | **Date**: 2026-05-11 | **Spec**: [spec.md](spec.md)  
**Input**: Feature specification from `specs/186-ops-orphan-cleanup/spec.md`  
**Issue**: [#186](https://github.com/grafvonb/c8volt/issues/186)  
**Mandatory Implementation Context**: `specs/ralph-implementation-rules.md` MUST be passed to task generation and every Ralph run as `--implementation-context specs/ralph-implementation-rules.md`.

## Summary

Add `c8volt ops purge orphan-process-instances` as a destructive operational cleanup workflow. Issue #197 already created the `ops`, `ops execute`, `ops repair`, and shared workflow contract foundation; this feature adds the missing `ops purge` grouping command and the first purge workflow. The command discovers orphan child process instances using the established process-instance search and orphan-filter semantics, freezes the discovered key set, validates the deletion plan through existing delete planning behavior, optionally reports dry-run/audit details, and deletes only the discovered keys when destructive execution is explicitly confirmed. The implementation should extend the existing `ops` command foundation and shared process-instance command/service primitives rather than shelling out to `get pi` or `delete pi`.

## Technical Context

**Language/Version**: Go, repository module `github.com/grafvonb/c8volt`  
**Primary Dependencies**: Cobra CLI, existing c8volt public facade packages, internal process-instance services, generated Camunda clients only through service adapters  
**Storage**: N/A for product state; optional audit report file output requested by the user  
**Testing**: `go test` with package-level command/facade/service tests; final validation through targeted tests and `make test` when implementation is complete  
**Target Platform**: Cross-platform CLI, with current local planning on macOS  
**Project Type**: Go CLI  
**Performance Goals**: Use existing paged process-instance search and worker configuration; do not introduce unbounded discovery loops  
**Constraints**: Keep command stdout deterministic for JSON/automation; preserve existing `get pi --orphan-children-only --keys-only` and `delete pi --key` behavior; operate on the initially discovered orphan set only  
**Scale/Scope**: One predefined ops workflow with dry-run, confirmed delete, automation, JSON/human output, and Markdown/JSON audit reporting  

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. Cleanup must validate discovery/deletion plan before mutation and report confirmed, failed, skipped, or planned outcomes.
- **CLI-First, Script-Safe Interfaces**: PASS. New behavior is exposed through Cobra commands, existing root flags, JSON envelope patterns, automation handling, and stable report/status tokens.
- **Tests and Validation Are Mandatory**: PASS. Tasks must include command, facade/service, report rendering, automation, and docs validation plus targeted and broader Go tests.
- **Documentation Matches User Behavior**: PASS. Command metadata changes require generated CLI docs via `make docs-content`, and README-facing behavior should be checked for needed updates.
- **Small, Compatible, Repository-Native Changes**: PASS. Existing `cmd/ops*.go`, `cmd/delete_processinstance.go`, process facade, and internal process-instance service patterns are the preferred extension points.

No constitution violations are planned.

## Project Structure

### Documentation (this feature)

```text
specs/186-ops-orphan-cleanup/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── orphan-cleanup-cli.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── ops_execute.go
├── ops_purge.go
├── ops_purge_orphan_processinstances.go
├── ops_purge_orphan_processinstances_test.go
├── ops_contract.go
├── ops_contract_test.go
├── cmd_views_ops_purge_orphan_processinstances.go
├── command_contract_test.go
└── docs-triggering command metadata

c8volt/
├── client.go
└── ops/
    ├── api.go
    ├── client.go
    ├── convert.go
    ├── model.go
    └── client_test.go

internal/services/
├── ops/
│   ├── api.go
│   ├── orphan_purge.go
│   └── orphan_purge_test.go
└── processinstance/
    ├── api.go
    ├── orphan_discovery.go
    └── orphan_discovery_test.go
```

**Structure Decision**: Extend the existing CLI and process-instance service layout. Add an ops facade/service only as a thin workflow orchestration boundary for the high-level playbook; keep resource-specific discovery, filtering, deletion planning, and deletion execution in `internal/services/processinstance` or existing process facade methods.

## Complexity Tracking

No constitution violations or extra architectural complexity are currently justified. If implementation finds that the existing process facade already provides enough primitives, the proposed `internal/services/ops` layer can remain minimal or be omitted in favor of a thin `c8volt/ops` facade that delegates directly to process service-owned orchestration.

## Phase 0: Research

See [research.md](research.md).

## Phase 1: Design

See [data-model.md](data-model.md), [quickstart.md](quickstart.md), and [contracts/orphan-cleanup-cli.md](contracts/orphan-cleanup-cli.md).

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The design includes discovery, immutable delete plan, confirmation gate, per-step status, and final outcome.
- **CLI-First, Script-Safe Interfaces**: PASS. Human, JSON, automation, dry-run, and report-file behaviors are explicit.
- **Tests and Validation Are Mandatory**: PASS. The task plan must include behavior tests before marking stories complete.
- **Documentation Matches User Behavior**: PASS. Generated docs refresh is required after command metadata changes.
- **Small, Compatible, Repository-Native Changes**: PASS. All files and APIs are scoped to existing command, facade, and service boundaries.

## Implementation Notes For Ralph

- Launch Ralph only with `--implementation-context specs/ralph-implementation-rules.md`.
- Each Ralph iteration must read `specs/ralph-implementation-rules.md`, this `plan.md`, `spec.md`, `tasks.md`, and `progress.md` when present.
- Every commit subject for this feature must follow Conventional Commits and end with `#186`.
