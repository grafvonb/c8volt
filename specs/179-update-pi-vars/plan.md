# Implementation Plan: Process Instance Variable Updates

**Branch**: `179-update-pi-vars` | **Date**: 2026-05-07 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/179-update-pi-vars/spec.md`

## Summary

Add a new `c8volt update` command family with `update process-instance` / `update pi` for setting process-instance-scope variables on existing process instances. Reuse existing process-instance key parsing, worker/fail-fast/no-worker-limit controls, command activity, output rendering, and waiter/backoff conventions; add a facade/service update path backed by the generated Camunda 8.8/8.9 element-instance variable endpoint; reject 8.7 before mutation; and confirm requested variable values through the existing `get pi --with-vars` lookup path unless `--no-wait` is supplied.

## Technical Context

**Language/Version**: Go, repository current module toolchain  
**Primary Dependencies**: Cobra command tree, existing command metadata helpers, process facade, process-instance service API, generated Camunda v8.8/v8.9 clients, process-instance variable lookup/enrichment, waiter/backoff and bulk worker helpers, shared rendering/error helpers  
**Storage**: N/A  
**Testing**: Go tests through targeted `go test ./cmd ./c8volt/process ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1`, docs generation checks, and final `make test`  
**Target Platform**: CLI on the repository's supported platforms  
**Project Type**: Go CLI  
**Performance Goals**: Update and confirmation work should fan out through existing worker controls for multi-key updates, avoid duplicate work for duplicate keys, and poll only requested variable names during confirmation  
**Constraints**: Preserve existing `run --vars` and `get process-instance --with-vars` behavior; require JSON object `--vars`; reject Camunda 8.7 before mutation; use process instance key as `elementInstanceKey`; compare normalized JSON values; support `--no-wait`; keep output script-safe in human and JSON modes  
**Scale/Scope**: One new root command family, one process-instance update leaf command and alias, facade/domain/service additions for update results and variable update calls, tests for command validation/output/service behavior, README/help/generated docs updates, and Speckit/Ralph artifacts

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The default command path confirms requested variables through the existing variable read path before reporting confirmed success; `--no-wait` explicitly reports submitted/accepted status without claiming visibility.
- **CLI-First, Script-Safe Interfaces**: PASS. The command uses stable Cobra flags, existing automation support checks, human/JSON output, deterministic validation, and existing bulk controls.
- **Tests and Validation Are Mandatory**: PASS. The plan requires command tests, facade tests, versioned service tests, docs generation checks, targeted Go tests, and final `make test`.
- **Documentation Matches User Behavior**: PASS. The new root command, alias, examples, selectors, `--vars`, `--no-wait`, and version support are user-visible and require README/help/generated docs updates.
- **Small, Compatible, Repository-Native Changes**: PASS. The design extends existing command, facade, service, waiter, metadata, and docs patterns instead of adding a parallel orchestration layer.

## Project Structure

### Documentation (this feature)

```text
specs/179-update-pi-vars/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-update-process-instance-vars.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── update.go
├── update_processinstance.go
├── update_processinstance_test.go
├── command_contract_test.go
├── cmd_views_processinstance.go
├── get_processinstance.go
├── get_processinstance_test.go
├── run_processinstance.go
└── root.go

c8volt/
└── process/
    ├── api.go
    ├── client.go
    ├── client_test.go
    └── model.go

internal/
├── domain/
│   ├── processinstance.go
│   └── processinstance_test.go
└── services/
    └── processinstance/
        ├── api.go
        ├── factory.go
        ├── factory_test.go
        ├── waiter/
        │   ├── waiter.go
        │   └── waiter_test.go
        ├── v87/
        │   ├── contract.go
        │   ├── service.go
        │   └── service_test.go
        ├── v88/
        │   ├── contract.go
        │   ├── variables.go
        │   └── service_test.go
        └── v89/
            ├── contract.go
            ├── variables.go
            └── service_test.go

README.md
docs/
docsgen/
```

**Structure Decision**: Add `cmd/update.go` as the root command and `cmd/update_processinstance.go` as the process-instance leaf, mirroring existing `run` and `delete` command registration. Extend `c8volt/process` for facade-level variable update and confirmation result models. Extend `internal/services/processinstance` for the update API, with v8.8/v8.9 implemented via generated `CreateElementInstanceVariables...` methods and v8.7 returning the shared unsupported-version error. Reuse existing command rendering and docs generation paths rather than creating new output or docs infrastructure.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

- Data model: [data-model.md](./data-model.md)
- CLI contract: [contracts/cli-update-process-instance-vars.md](./contracts/cli-update-process-instance-vars.md)
- Quickstart and verification scenarios: [quickstart.md](./quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The contract separates submitted vs confirmed results and makes confirmation failure a per-key outcome.
- **CLI-First, Script-Safe Interfaces**: PASS. The contract defines valid/invalid invocations, JSON object validation, key selection, human/JSON output, and automation-compatible behavior.
- **Tests and Validation Are Mandatory**: PASS. The task list will include failing-first tests per user story, versioned service tests, command metadata tests, docs generation, targeted Go tests, and final `make test`.
- **Documentation Matches User Behavior**: PASS. Documentation work is part of the feature contract and task list.
- **Small, Compatible, Repository-Native Changes**: PASS. The design reuses existing key, worker, waiter, facade, service, rendering, and generated-client patterns.

## Complexity Tracking

No constitution violations or additional complexity exceptions are required.
