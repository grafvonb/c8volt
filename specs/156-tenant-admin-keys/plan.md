# Implementation Plan: Tenant Scope For Discovery And Explicit Admin Keys

**Branch**: `156-tenant-admin-keys` | **Date**: 2026-05-24 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/156-tenant-admin-keys/spec.md`

**Implementation Context**: Planning, task generation, and every Ralph implementation iteration must read and apply `specs/ralph-implementation-rules.md`. Ralph launch instructions must include `--implementation-context specs/ralph-implementation-rules.md`.

## Summary

Tenant handling must distinguish c8volt-produced candidate sets from explicit operator-supplied targets. Discovery, search, selection, create, deploy, and run flows keep applying the selected tenant where supported, while direct keys, stdin keys, IDs, and direct flag values are treated as explicit administrative input governed by Camunda backend authorization. The implementation should audit Camunda 8.8 and 8.9 process-instance, process-definition, resource, and bulk/stdin paths, remove or bypass c8volt-side tenant mismatch rejection for explicit inputs, preserve existing `<default>` tenant behavior for create/deploy/run, and update help/docs/tests to make the contract visible.

## Technical Context

**Language/Version**: Go, following the repository's current module and supported Camunda 8.7, 8.8, and 8.9 boundaries. This feature changes Camunda 8.8 and 8.9 behavior only.

**Primary Dependencies**: Cobra command layer, c8volt public facade packages, internal service interfaces, generated Camunda clients, `c8volt/foptions`, `internal/services.CallOption`, `toolx`, and existing test helpers.

**Storage**: N/A. c8volt remains a short-lived CLI client; Camunda is the runtime fact and authorization source.

**Testing**: Go tests near `cmd`, `c8volt/process`, `c8volt/resource`, `internal/services/processinstance`, `internal/services/processdefinition`, `internal/services/resource`, command contract tests, and docs generation checks where command help changes.

**Target Platform**: Cross-platform CLI binary used locally, in shells, and in automation.

**Project Type**: CLI application with command/facade/service/generated-client layering.

**Performance Goals**: Preserve existing discovery paging and keyed lookup performance. No added remote validation call should be introduced solely to enforce tenant matching for explicit admin input.

**Constraints**: Preserve mutation safety, automation behavior, output contracts, deterministic key handling, generated-client isolation, existing `<default>` tenant behavior, and explicit unsupported-version behavior. Do not rewrite Camunda 8.7 tenant behavior.

**Scale/Scope**: Tenant-sensitive process-instance, process-definition, resource, and bulk/stdin command paths for Camunda 8.8 and 8.9.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. Direct-key destructive flows keep existing dry-run, confirmation, preflight, dependency expansion, force, wait, and verification behavior while removing only stricter local tenant authorization.
- **CLI-First, Script-Safe Interfaces**: PASS. The behavior remains exposed through existing commands, flags, stdin conventions, JSON/keys-only modes, automation metadata, help, and generated docs.
- **Tests and Validation Are Mandatory**: PASS. Tasks must add tenant-scoped discovery/search coverage and explicit direct-key mismatch coverage for Camunda 8.8 or 8.9 before validation.
- **Documentation Matches User Behavior**: PASS. Help text, README tenant guidance, and generated CLI docs are in scope where tenant behavior is described.
- **Small, Compatible, Repository-Native Changes**: PASS. The design reuses existing command/facade/service boundaries and options instead of adding a new authorization system or command hierarchy.

## Project Structure

### Documentation (this feature)

```text
specs/156-tenant-admin-keys/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── tenant-admin-keys.md
├── checklists/
│   └── requirements.md
├── progress.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_processinstance*.go
├── walk_processinstance.go
├── expect_processinstance.go
├── cancel_processinstance.go
├── delete_processinstance.go
├── get_processdefinition.go
├── delete_processdefinition.go
├── get_resource.go
├── cmd_cli.go
├── cmd_stdin.go
├── *_test.go
└── command_contract_test.go

c8volt/process/
├── api.go
├── client.go
├── convert.go
└── client_test.go

c8volt/resource/
├── api.go
├── client.go
├── convert.go
└── client_test.go

internal/services/
├── calloption.go
├── common/
├── processinstance/
├── processdefinition/
└── resource/

docs/cli/
README.md
```

**Structure Decision**: Keep tenant semantics owned at the existing command/facade/service boundaries. Commands continue to distinguish search mode from explicit key/stdin/direct-ID modes, facades pass public options through, and services remain responsible for version-specific tenant filtering or direct backend-authorized lookup behavior. Documentation follows executable command help through the existing docs generation path.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design

See [data-model.md](./data-model.md), [contracts/tenant-admin-keys.md](./contracts/tenant-admin-keys.md), and [quickstart.md](./quickstart.md).

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The direct admin input contract preserves existing safety gates and keeps Camunda as the fact and authorization source.
- **CLI-First, Script-Safe Interfaces**: PASS. Help/docs and command contract checks are part of the planned validation.
- **Tests and Validation Are Mandatory**: PASS. The task plan must include targeted command/facade/service tests plus broader validation.
- **Documentation Matches User Behavior**: PASS. README and generated CLI docs are explicit deliverables where behavior text changes.
- **Small, Compatible, Repository-Native Changes**: PASS. No c8volt-side authorization layer, new persistence, or alternate command structure is introduced.

## Complexity Tracking

No constitution violations requiring justification.
