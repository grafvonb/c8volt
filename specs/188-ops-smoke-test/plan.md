# Implementation Plan: Ops Execute Smoke Test

**Branch**: `188-ops-smoke-test` | **Date**: 2026-05-17 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/188-ops-smoke-test/spec.md`
**Issue**: [#188](https://github.com/grafvonb/c8volt/issues/188)
**Mandatory Implementation Context**: `specs/ralph-implementation-rules.md` MUST be passed to task generation and every Ralph run as `--implementation-context specs/ralph-implementation-rules.md`.

## Summary

Add `c8volt ops execute smoke-test` as a high-level, auditable operational workflow. The command validates the configured profile and cluster connectivity, selects the embedded multiple-subprocess BPMN fixture for the configured Camunda version, deploys it, starts one or more process instances, walks each created process-instance family, optionally cleans up the created instances and deployed process definition, and emits compact human output, deterministic JSON, and optional Markdown/JSON audit reports. The implementation should extend the existing #186/#187/#199 ops workflow/report foundation and lower-level service primitives rather than shelling out to existing CLI commands.

## Technical Context

**Language/Version**: Go, repository module `github.com/grafvonb/c8volt`
**Primary Dependencies**: Cobra CLI, existing `c8volt/ops`, `c8volt/process`, `c8volt/resource`, internal ops services, internal resource/processinstance/processdefinition services, embedded BPMN fixtures, generated Camunda clients only through service adapters
**Storage**: N/A for product state; optional Markdown or JSON audit report file output requested by the user
**Testing**: `go test` with command, facade, service, report, contract, subprocess exit-code, and docs-generation tests; final validation through targeted tests and `make test` when implementation is complete
**Target Platform**: Cross-platform CLI, with current local planning on macOS
**Project Type**: Go CLI
**Performance Goals**: Reuse existing process-instance creation worker controls for `--count > 1`; avoid unbounded search for cleanup eligibility; keep JSON stdout deterministic for automation
**Constraints**: Preserve existing `config test-connection`, `embed deploy`, `run pi`, `walk pi`, `delete pi`, and `delete pd` behavior; command code must stay thin; ops workflow must not own resource-specific API logic, waiter/polling, traversal, or deletion internals
**Scale/Scope**: One predefined ops execute workflow with dry-run, versioned fixture selection, deployment, process-instance creation, traversal, optional cleanup, no-cleanup retention, automation, JSON/human output, Markdown/JSON audit reporting, docs, and command contract metadata

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The workflow proves connectivity, deployment, run, traversal, and cleanup outcomes or clearly reports planned, skipped, blocked, failed, partially failed, passed, or cleanup-skipped states.
- **CLI-First, Script-Safe Interfaces**: PASS. Behavior is exposed through Cobra commands, inherited root flags, stable command contracts, deterministic JSON envelopes, report files, and automation-safe confirmation handling.
- **Tests and Validation Are Mandatory**: PASS. Tasks must include command registration, validation, dry-run, fixture selection, deployment, run/walk, cleanup, no-cleanup, automation JSON, reports, error class, docs, and lower-level regression tests.
- **Documentation Matches User Behavior**: PASS. User-facing command metadata requires generated CLI docs via `make docs-content`; README-facing examples should be checked and updated when needed.
- **Small, Compatible, Repository-Native Changes**: PASS. Existing `cmd/ops_execute.go`, ops workflow/report helpers, `c8volt/ops`, internal ops services, resource services, process-instance services, and process-definition deletion services are the extension points.

No constitution violations are planned.

## Project Structure

### Documentation (this feature)

```text
specs/188-ops-smoke-test/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── progress.md
├── contracts/
│   └── smoke-test-cli.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── ops_execute.go
├── ops_execute_smoke_test.go
├── ops_execute_smoke_test_test.go
├── cmd_views_ops_execute_smoke_test.go
├── cmd_views_ops_notices.go
├── ops_contract.go
├── ops_contract_test.go
├── command_contract.go
├── command_contract_test.go
├── config_test_connection.go
├── embed_deploy.go
├── run_processinstance.go
├── walk_processinstance.go
├── delete_processinstance.go
└── delete_processdefinition.go

c8volt/
├── ops/
│   ├── api.go
│   ├── client.go
│   ├── convert.go
│   ├── model.go
│   └── client_test.go
├── process/
│   └── existing run, walk, dry-run, and delete facade primitives
└── resource/
    └── existing embedded fixture deployment and process-definition delete facade primitives

internal/services/
├── ops/
│   ├── api.go
│   ├── smoke_test.go
│   └── smoke_test_test.go
├── processinstance/
│   ├── api.go
│   ├── bulk.go
│   ├── dryrun.go
│   ├── walker/
│   └── existing delete planning/deletion support
├── processdefinition/
│   ├── api.go
│   └── delete.go
└── resource/
    └── existing deployment support

embedded/processdefinitions/
├── C87_MultipleSubProcessesParent.bpmn
├── C88_MultipleSubProcessesParent.bpmn
└── C89_MultipleSubProcessesParent.bpmn
```

**Structure Decision**: Extend the existing ops execute command and ops facade/service layout. The ops workflow should orchestrate steps and aggregate reports, while connectivity validation, embedded fixture loading/deployment, process-instance creation, traversal, process-instance deletion, and process-definition deletion remain owned by existing lower-level packages or new primitives added to those owning packages.

## Complexity Tracking

No constitution violations or extra architectural complexity are currently justified. If implementation discovers a missing primitive, add it to the owning resource/process service or facade instead of expanding the ops command into resource-specific logic. If existing report helpers already cover a behavior, reuse them directly and keep smoke-test-specific code limited to request/result models, workflow orchestration, and rendering.

## Phase 0: Research

See [research.md](research.md).

## Phase 1: Design

See [data-model.md](data-model.md), [quickstart.md](quickstart.md), and [contracts/smoke-test-cli.md](contracts/smoke-test-cli.md).

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The design includes read-only planning, explicit mutation steps, traversal proof, cleanup eligibility, reportable step statuses, and final outcomes.
- **CLI-First, Script-Safe Interfaces**: PASS. Human, JSON, automation, dry-run, no-cleanup, report-file, and exit-code behavior are explicit.
- **Tests and Validation Are Mandatory**: PASS. The task plan must include command, facade/service, report, automation, exit-code, docs, and regression tests before work is complete.
- **Documentation Matches User Behavior**: PASS. Generated docs refresh is required after command metadata changes.
- **Small, Compatible, Repository-Native Changes**: PASS. All planned files and APIs stay inside existing command, facade, ops service, resource service, process-instance service, and process-definition service boundaries.

## Implementation Notes For Ralph

- Launch Ralph only with `--implementation-context specs/ralph-implementation-rules.md`.
- Each Ralph iteration must read `specs/ralph-implementation-rules.md`, this `plan.md`, `spec.md`, `tasks.md`, and `progress.md` when present.
- Every commit subject for this feature must follow Conventional Commits and end with `#188`.
