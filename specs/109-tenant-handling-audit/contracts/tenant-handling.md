# Contract: Tenant Handling Across Tenant-Aware Commands

## Tenant Resolution Contract

All tenant-aware commands must use one effective tenant context per execution:

| Source | Required behavior |
|--------|-------------------|
| `--tenant` flag | Wins when explicitly provided |
| Environment | Supplies tenant when no explicit flag overrides it |
| Profile | Supplies tenant when neither flag nor environment does |
| Base config | Supplies tenant when no higher-precedence source does |

That effective tenant context must be passed consistently into every internal lookup performed by the command flow.

## Supported Tenant Mismatch Contract

| Situation | Required behavior |
|-----------|-------------------|
| Resource exists in selected tenant | Return it normally |
| Resource is absent from selected tenant | Return `not found` |
| Resource exists only in another tenant on a supported tenant-safe path | Return the same `not found` outcome as an absent resource |

Supported tenant mismatch must not reveal whether the resource exists in another tenant.

## Unsupported Version Contract

| Version/operation state | Required behavior |
|-------------------------|-------------------|
| `v8.8` operation has a tenant-safe upstream path | Use it |
| `v8.7` operation can be made tenant-safe through current generated-client semantics | Use it |
| `v8.7` operation cannot be made tenant-safe through current generated-client semantics | Return an explicit unsupported outcome for that exact operation or flow segment |
| `v8.9` planning scope without repository implementation | Record as audit/follow-up only; do not claim current runtime support |

Unsupported behavior must be as narrow as possible and must not be widened to the whole command family unless every relevant segment is unsafe.

## Authoritative Operation Boundary

The feature must use the following operation-level contract as the authoritative design target:

| Operation segment | `v8.8` target contract | `v8.7` target contract | Current boundary note |
|------------------|------------------------|------------------------|-----------------------|
| Direct get by key | Must resolve through a tenant-safe upstream path and return tenant-safe `not found` on mismatch | Supported only if a tenant-safe upstream equivalent exists; otherwise explicit unsupported outcome | Current direct-get endpoints appear unscoped in both versions and must not remain the authoritative tenant seam |
| State check by key | Must inherit the same tenant-safe contract as direct lookup | Supported only where the lookup path can stay tenant-safe; otherwise explicit unsupported outcome | Wait and mutation preflight depend on this seam |
| Search by filter | Must include effective tenant context in the upstream request | Must include effective tenant context in the upstream request | Search is the safest current upstream tenant seam in both supported versions |
| Direct children lookup | Must be derived from tenant-safe search | Must be derived from tenant-safe search | `ParentKey` searches can stay supported when the search path is tenant-safe |
| Ancestry / descendants / family | Must compose only tenant-safe lookup steps | Must fail only on the exact unsafe segment when a required lookup step cannot be tenant-safe | Mixed-flow helpers must not reintroduce unsafe direct-get behavior |
| Wait / polling | Must inherit tenant-safe `not found` or matched outcomes from the backing state-check path | Must fail explicitly only when the backing state-check path is unsafe | Wait must not become a side channel for cross-tenant existence |
| Cancel / delete preflight | Must validate targets through tenant-safe lookup/state-check paths before mutating | Must fail explicitly only when the required validation segment is unsafe | Mutation must stop before acting on cross-tenant targets |

`v8.9` remains outside the runtime contract for this repository until a `processinstance` service implementation exists; its only binding requirement in this feature is honest audit documentation.

## Mixed-Flow Contract

| Flow segment | Required behavior |
|-------------|-------------------|
| Direct get | Must be tenant-safe or explicitly unsupported |
| Search | Must include effective tenant context through supported upstream filter semantics |
| State check / wait | Must inherit the same tenant-safe contract as the lookup path it depends on |
| Ancestry / descendants / family | Must not cross tenant boundaries |
| Cancel / delete | Must not act on targets that are only reachable through cross-tenant leakage |

Commands must not mix a tenant-safe search path with an unsafe direct-get follow-up that reintroduces cross-tenant exposure.

## Command-Family Coverage Contract

Every tenant-aware command family in scope must receive explicit regression coverage for:

- matching tenant
- wrong tenant
- non-existing tenant
- default tenant
- explicit `--tenant`
- environment-derived tenant
- profile-derived tenant
- base-config-derived tenant
- supported vs unsupported version behavior where relevant

## Repository Implementation Seams

The main implementation seams for this contract are:

- `cmd/root.go` and `config/` for resolving the effective tenant context
- `c8volt/process/client.go` for forwarding the shared process-instance contract
- `internal/services/processinstance/v87/service.go`
- `internal/services/processinstance/v88/service.go`
- `internal/services/processinstance/walker/walker.go`
- `internal/services/processinstance/waiter/waiter.go`

No command may add a parallel tenant-filtering layer that compensates for an unsafe service lookup by hiding cross-tenant data after retrieval.
