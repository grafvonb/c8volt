# Implementation Plan: Ops Execute Retention Policy

**Branch**: `187-ops-retention-policy` | **Date**: 2026-05-14 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/187-ops-retention-policy/spec.md`
**Issue**: [#187](https://github.com/grafvonb/c8volt/issues/187)
**Mandatory Implementation Context**: `specs/ralph-implementation-rules.md` MUST be passed to task generation and every Ralph run as `--implementation-context specs/ralph-implementation-rules.md`.

## Summary

Add `c8volt ops execute retention-policy` as a destructive, auditable retention cleanup workflow. The command discovers process instances whose end date is older than a required retention age, freezes that seed set, reuses existing process-instance delete planning to resolve roots and affected families, then either reports a dry-run plan or executes confirmed deletion through the established process-instance deletion service. The implementation should extend the existing #186 ops workflow foundation and shared report helpers rather than shelling out to `get pi` or `delete pi`.

## Technical Context

**Language/Version**: Go, repository module `github.com/grafvonb/c8volt`
**Primary Dependencies**: Cobra CLI, existing c8volt public facade packages, `c8volt/ops`, `c8volt/process`, internal ops and process-instance services, generated Camunda clients only through service adapters
**Storage**: N/A for product state; optional Markdown or JSON audit report file output requested by the user
**Testing**: `go test` with command/facade/service/report tests; final validation through targeted tests and `make test` when implementation is complete
**Target Platform**: Cross-platform CLI, with current local planning on macOS
**Project Type**: Go CLI
**Performance Goals**: Use existing paged process-instance search, limits, batch sizing, and worker controls; do not introduce unbounded discovery loops
**Constraints**: Keep command stdout deterministic for `--automation --json`; preserve existing `get pi --end-date-older-days`, `delete pi --keys`, and delete hierarchy behavior; freeze the initially discovered retention seed set
**Scale/Scope**: One predefined ops execute workflow with validation, dry-run, confirmed deletion, automation, JSON/human output, and Markdown/JSON audit reporting

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. Cleanup must validate discovery and delete planning before mutation, verify or report deletion outcomes through existing wait/no-wait behavior, and classify final outcome as planned, deleted, partially failed, or failed.
- **CLI-First, Script-Safe Interfaces**: PASS. New behavior is exposed through Cobra commands, existing root flags, automation handling, JSON envelope patterns, stable step statuses, and report files.
- **Tests and Validation Are Mandatory**: PASS. Tasks must include command validation, discovery, planning, automation, destructive safety, report rendering, error class, docs, and regression tests.
- **Documentation Matches User Behavior**: PASS. Command metadata changes require generated CLI docs via `make docs-content`, and README-facing examples should be checked for needed updates.
- **Small, Compatible, Repository-Native Changes**: PASS. Existing `cmd/ops_execute.go`, #186 ops workflow/report code, process facade, and internal process-instance service patterns are the preferred extension points.

No constitution violations are planned.

## Project Structure

### Documentation (this feature)

```text
specs/187-ops-retention-policy/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── retention-policy-cli.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── ops_execute.go
├── ops_execute_retention_policy.go
├── ops_execute_retention_policy_test.go
├── cmd_views_ops_execute_retention_policy.go
├── cmd_views_ops_notices.go
├── ops_contract.go
├── ops_contract_test.go
├── command_contract.go
├── command_contract_test.go
└── generated-doc-triggering command metadata

c8volt/
├── ops/
│   ├── api.go
│   ├── client.go
│   ├── convert.go
│   ├── model.go
│   └── client_test.go
└── process/
    └── existing delete planning/deletion facade primitives

internal/services/
├── ops/
│   ├── api.go
│   ├── retention_policy.go
│   └── retention_policy_test.go
└── processinstance/
    ├── api.go
    ├── retention_discovery.go
    ├── retention_discovery_test.go
    └── existing delete planning/deletion support
```

**Structure Decision**: Extend the existing ops command and facade/service layout created for #186. Add a retention-policy workflow under `ops execute`, keep command code thin, and let process-instance services own search, age filtering, hierarchy/delete planning, deletion, waiting, and version-specific Camunda API behavior.

## Complexity Tracking

No constitution violations or extra architectural complexity are currently justified. If implementation finds that #186 shared ops helpers cover report and workflow orchestration fully, new #187-specific code should remain limited to retention request/result models and command wiring.

## Phase 0: Research

See [research.md](research.md).

## Phase 1: Design

See [data-model.md](data-model.md), [quickstart.md](quickstart.md), and [contracts/retention-policy-cli.md](contracts/retention-policy-cli.md).

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The design freezes discovery, validates an immutable delete plan, reuses confirmation/deletion/wait behavior, and reports every final outcome.
- **CLI-First, Script-Safe Interfaces**: PASS. Human, JSON, automation, dry-run, report-file, and exit-code behavior are explicit.
- **Tests and Validation Are Mandatory**: PASS. The task plan must include command, facade/service, report, automation, exit-code, docs, and regression tests before work is complete.
- **Documentation Matches User Behavior**: PASS. Generated docs refresh is required after command metadata changes.
- **Small, Compatible, Repository-Native Changes**: PASS. All planned files and APIs stay inside existing command, facade, ops service, and process-instance service boundaries.

## Implementation Notes For Ralph

- Launch Ralph only with `--implementation-context specs/ralph-implementation-rules.md`.
- Each Ralph iteration must read `specs/ralph-implementation-rules.md`, this `plan.md`, `spec.md`, `tasks.md`, and `progress.md` when present.
- Every commit subject for this feature must follow Conventional Commits and end with `#187`.
