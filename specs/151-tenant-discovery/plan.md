# Implementation Plan: Tenant Discovery Command

**Branch**: `151-tenant-discovery` | **Date**: 2026-04-30 | **Spec**: [spec.md](/Users/adam.boczek/.codex/worktrees/38de/c8volt/specs/151-tenant-discovery/spec.md)
**Input**: Feature specification from `/specs/151-tenant-discovery/spec.md`

## Summary

Add `c8volt get tenant` as a read-only tenant discovery command with list, single-tenant lookup, literal name filtering, JSON output, generated CLI docs, and version-aware unsupported behavior. The implementation should follow the existing command/facade/internal-service layering used by `get process-instance`: command code remains thin, a new tenant facade exposes domain-safe tenant data, versioned internal tenant services wrap the generated Camunda tenant client methods for `v8.8` and `v8.9`, and `v8.7` reports the repository's existing unsupported-capability style because its generated clients do not expose tenant-management search/get APIs.

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/v88/camunda` and `internal/clients/camunda/v89/camunda`, existing render/error helpers in `cmd/`, existing service/facade patterns under `internal/services/*` and `c8volt/*`  
**Storage**: No persistent storage changes; command reads configuration/profile/authentication through existing root/global flag handling  
**Testing**: `go test`, `make test`, command regression tests under `cmd/`, facade tests under `c8volt/tenant/`, versioned service tests under `internal/services/tenant/v87`, `v88`, and `v89`  
**Target Platform**: Cross-platform CLI for local and CI use against supported Camunda versions; tenant discovery supported for `v8.8` and `v8.9`, unsupported explicitly for `v8.7`  
**Project Type**: CLI  
**Performance Goals**: Tenant listing should perform one tenant search request per command execution for normal list/filter output, single-tenant lookup should perform one tenant get request on supported versions, and local sort/filter overhead should be linear in returned tenant count  
**Constraints**: Preserve existing `get` command behavior, keep command code out of generated-client details, avoid sensitive fields in human or JSON output, use literal contains filtering only, sort by name then tenant ID, use existing unsupported and not-found error mapping, update generated CLI docs if command help changes, finish with targeted tests and `make test`  
**Scale/Scope**: New public tenant facade under `c8volt/tenant/`, new internal domain model in `internal/domain/tenant.go`, new versioned service package under `internal/services/tenant/`, command and view files under `cmd/`, root facade wiring in `c8volt/client.go` and `c8volt/contract.go`, generated-client use in `internal/clients/camunda/v88/camunda` and `v89/camunda`, tests in `cmd/`, `c8volt/tenant/`, and `internal/services/tenant/`, docs in `docs/cli/` and README guidance if generated docs or examples change

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. The command is read-only, and tests must prove actual list, lookup, filter, JSON, sorting, and unsupported-version outcomes rather than merely wiring help text.
- **CLI-First, Script-Safe Interfaces**: Pass. The feature adds a Cobra subcommand with existing global flags, supports JSON for automation, and uses stable error/output contracts.
- **Tests and Validation Are Mandatory**: Pass. The plan requires close command, facade, and versioned service tests plus final `make test`.
- **Documentation Matches User Behavior**: Pass. The command changes user-facing help and docs, so generated CLI documentation must be refreshed and any README examples updated if needed.
- **Small, Compatible, Repository-Native Changes**: Pass. The design mirrors existing `cluster`, `process`, and `resource` facade/service layering instead of calling generated clients directly from `cmd`.

## Project Structure

### Documentation (this feature)

```text
specs/151-tenant-discovery/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── tenant-command.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get.go
├── get_tenant.go
├── cmd_views_get.go
├── get_tenant_test.go
└── process_api_stub_test.go or tenant-specific test stubs

c8volt/
├── client.go
├── contract.go
└── tenant/
    ├── api.go
    ├── client.go
    ├── convert.go
    ├── model.go
    └── client_test.go

internal/domain/
├── tenant.go
└── tenant_test.go

internal/services/tenant/
├── api.go
├── factory.go
├── factory_test.go
├── v87/
│   ├── service.go
│   └── service_test.go
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
docs/cli/
```

**Structure Decision**: Add a small tenant domain/facade/service slice that mirrors existing repository-native service families. `cmd/get_tenant.go` should use the public facade and render helpers, while `internal/services/tenant/{v88,v89}` own generated-client requests and conversions. `v87` remains an explicit unsupported service so the factory and command can report the existing unsupported-capability style without special branching in command code.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/.codex/worktrees/38de/c8volt/specs/151-tenant-discovery/research.md).

- Confirm generated tenant APIs exist for `v8.8` and `v8.9`: `SearchTenants` and `GetTenant` with `TenantSearchQueryRequest`, `TenantFilter`, `TenantSearchQuerySortRequest`, and `TenantResult`.
- Confirm `v8.7` generated clients do not expose equivalent tenant-management search/get methods and should therefore fail explicitly through the unsupported-capability path.
- Confirm the command should keep filtering simple: local `strings.Contains` on tenant name after service retrieval is sufficient and intentionally treats wildcard/glob/regex syntax as literal text.
- Confirm sort behavior should be deterministic in the CLI/facade layer even if upstream sorting is requested, using name then tenant ID so tests do not depend on upstream response order.
- Confirm output should include only tenant ID, name, and description because generated tenant results include exactly those non-sensitive display fields for the discovery command's supported versions.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/.codex/worktrees/38de/c8volt/specs/151-tenant-discovery/data-model.md)
- [quickstart.md](/Users/adam.boczek/.codex/worktrees/38de/c8volt/specs/151-tenant-discovery/quickstart.md)
- [contracts/tenant-command.md](/Users/adam.boczek/.codex/worktrees/38de/c8volt/specs/151-tenant-discovery/contracts/tenant-command.md)

- Add `internal/domain.Tenant` plus sort/filter helpers close to the domain or facade conversion code.
- Add `internal/services/tenant.API` with list/search and get-by-ID operations. `v88` and `v89` call generated Camunda tenant APIs; `v87` returns `domain.ErrUnsupported` wrapped with a concise tenant-management operation description.
- Add `c8volt/tenant.API` and facade methods that map domain errors through `c8volt/ferrors` and return JSON-ready public tenant models.
- Wire the tenant facade into `c8volt.API` and `c8volt.New` alongside existing cluster/process/resource/task facades.
- Add `cmd/get_tenant.go` with `--key` and `--filter`, command aliases only if they match existing naming conventions, automation support, read-only mutation classification, JSON support, and validation that `--key` and `--filter` are not combined unless implementation deliberately treats `--filter` as list-only and rejects it in keyed mode.
- Extend `cmd/cmd_views_get.go` with tenant human output and list/JSON rendering that shows only tenant ID, name, and optional description.
- Update generated CLI docs through `make docs-content` after command help is implemented.

### Version Support Matrix

| Version | Tenant list | Tenant lookup | Planned behavior |
|---------|-------------|---------------|------------------|
| `v8.7` | Unsupported | Unsupported | Return existing unsupported-capability style for tenant discovery operations |
| `v8.8` | Supported through generated `SearchTenants` | Supported through generated `GetTenant` | Convert tenant results to the shared tenant model |
| `v8.9` | Supported through generated `SearchTenants` | Supported through generated `GetTenant` | Match `v8.8` behavior unless generated field shapes differ |

## Phase 2: Task Planning Approach

Task generation should keep user-story slices independently verifiable:

1. Build the shared tenant model, internal tenant API/factory, public facade API, and root wiring first.
2. Deliver User Story 1 as the MVP: supported-version tenant list with sorting, compact human output, command tests, facade tests, and service tests.
3. Add User Story 2 keyed lookup with not-found handling and JSON-ready single result behavior.
4. Add User Story 3 local literal name filtering and tests proving wildcard/glob/regex/query text is not interpreted.
5. Add User Story 4 JSON and unsupported-version coverage, generated docs, and final validation.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. Each story has explicit command/service tests and quickstart scenarios that prove the actual CLI outcomes.
- **CLI-First, Script-Safe Interfaces**: Still passes. The command uses existing output modes and global flags and avoids a new filtering language.
- **Tests and Validation Are Mandatory**: Still passes. Tasks must include command, facade, versioned service, docs generation, targeted `go test`, and `make test`.
- **Documentation Matches User Behavior**: Still passes. The plan includes generated CLI documentation and README review.
- **Small, Compatible, Repository-Native Changes**: Still passes. The new tenant slice follows existing facade/service boundaries and avoids command-level generated-client calls.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
