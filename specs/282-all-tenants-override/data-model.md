# Data Model: All-Tenants Tenant Override

## Scope

This feature introduces no persistent data model. All entities are transient CLI state derived during command bootstrap. The effective tenant continues to use the existing configuration and public tenant-context representations.

## Entities

### All-Tenants Selection

Represents the parsed root flag and its relationship to the existing tenant flag.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `Active` | boolean | `--all-tenants` value | True only when the parsed value is true; explicit false is inactive. |
| `TenantExplicit` | boolean | Cobra/pflag changed state for `--tenant` | True even when the explicit value is empty. |
| `CommandSupport` | All-Tenants Support | resolved command annotation | Active selection is invalid when support is `rejected_concrete_destination`. |

This is command-line-only state. It is not bound to Viper and has no config-file, profile, or environment representation.

### Effective Tenant Filter

The existing `cfg.App.Tenant` value after configuration precedence, normalization, and optional override.

| State | Stored value | Meaning |
|-------|--------------|---------|
| Named | non-empty tenant ID | Filter applicable discovery to one configured tenant. |
| Unfiltered | empty string | Do not send a tenant filter; the backend returns resources visible to the authenticated identity. |
| Default destination | `<default>` produced by `TargetTenant()` | Existing creation fallback; unreachable from active `--all-tenants` because destination commands reject first. |

Invariant: when active `--all-tenants` passes validation, the effective tenant filter is empty before configuration is placed in command context or services are installed.

### Tenant Override Provenance

Private rendering state associated with the command context. It is not part of public `tenant.Context` and is never serialized in machine-readable command output.

| Field | Meaning |
|-------|---------|
| Configured tenant | Tenant selected by base config/profile/environment normalization before the all-tenants override. |
| Effective tenant | Empty when the override is applied. |
| Origin | Existing explicit-tenant origin or new all-tenants origin. |
| Broadened | True only when a named configured tenant becomes unfiltered. |

Invariant: all-tenants provenance produces override warning text only when it broadens a named configured filter. An already-empty configured filter uses ordinary unfiltered scope text without override chatter.

### All-Tenants Support

An enum-like command classification used by both runtime validation and capability output.

| Value | Runtime contract |
|-------|------------------|
| `accepted` | The inherited flag may be parsed. Tenant-sensitive discovery uses the unfiltered effective tenant; neutral/direct-key behavior remains as currently implemented. |
| `rejected_concrete_destination` | Active `--all-tenants` is invalid because the command must create or deploy into one concrete tenant. |

The default for unannotated commands is `accepted`. One resolver reads the command annotation and feeds both root validation and `CommandCapability.AllTenantsSupport`.

### Command Capability

The existing additive machine-readable command description. It gains `allTenantsSupport` while keeping document version `v1` and all current fields unchanged.

## Relationships

```text
parsed root flags
  ├── --tenant changed? ───────────────┐
  └── --all-tenants active?            │
                                       v
resolved command ── annotation ──> validation
                                       │ valid
base/profile/env/tenant configuration  v
                         normalization -> configured tenant
                                                │
                                      active override
                                                v
                                      effective tenant = ""
                                                │
                         ┌──────────────────────┴──────────────────────┐
                         v                                             v
               private provenance                            command/services
                         │                                             │
                         v                                             v
               human warning/scope                       existing request behavior
```

## State Transitions

1. Cobra parses inherited flags and resolves the executable command.
2. Root validation evaluates active all-tenants state, explicit tenant state, and command support.
3. Invalid combinations terminate as invalid input before configuration or command work.
4. Existing configuration precedence and normalization produce the configured tenant.
5. Active all-tenants selection captures provenance and changes the effective tenant to empty.
6. The effective config enters command context and existing services.
7. Applicable human renderers emit configured tenant, exact override warning, and effective scope once; protected modes suppress those lines.
8. Capability inspection reads the same command support annotation used at step 2.

## Validation Rules

| Condition | Result |
|-----------|--------|
| all-tenants inactive | Existing behavior; no new provenance or validation. |
| all-tenants active and `--tenant` explicitly changed | Invalid input, including explicit empty tenant. |
| all-tenants active and command support rejects concrete destination | Invalid input before any side effect. |
| all-tenants active and configured tenant named | Effective tenant becomes empty and warning provenance is recorded. |
| all-tenants active and configured tenant empty | Effective tenant remains empty; no override warning. |
| direct resource key with all-tenants active | Existing backend authorization behavior; no bypass. |

## Concrete-Destination Inventory

The following executable leaves MUST use `rejected_concrete_destination`:

1. `deploy process-definition`
2. `embed deploy`
3. `run process-instance`
4. `ops execute smoke-test`, including dry-run

All other commands default to `accepted`. This does not promise that every neutral command performs tenant discovery; it states that parsing the inherited flag is safe and does not change that command's established semantics.

## Serialized Impact

- Existing result envelopes: unchanged.
- Existing structured `tenantContext`: unchanged; an unfiltered selection remains `filter: "none"` with current warnings.
- Capability document: additive `allTenantsSupport` per command; version remains `v1`.
- Human output: private provenance may add the exact all-tenants warning in ordinary human modes only.
