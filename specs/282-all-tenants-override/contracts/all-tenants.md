# CLI Contract: `--all-tenants`

## Flag Grammar

```text
c8volt --all-tenants <command> [command options]
c8volt <command> --all-tenants [command options]
```

`--all-tenants` is a root persistent boolean flag. Cobra's inherited-flag behavior permits it wherever other root persistent flags are accepted. Its default is false.

The flag is intentionally command-line-only:

- no config-file key;
- no profile key;
- no environment-variable alias;
- no change to the existing `--tenant` binding.

Explicit `--all-tenants=false` is equivalent to absence.

## Precedence and Resolution

When active, the flag overrides the tenant produced by the established order of base config, active profile, environment, and normalization. It does not participate as another Viper source.

| Resolved configured tenant | Active flag | Effective tenant filter |
|----------------------------|-------------|-------------------------|
| named tenant | false | named tenant |
| empty | false | empty |
| named tenant from base/profile/environment | true | empty |
| Camunda 8.7 normalized `<default>` | true | empty |
| already empty | true | empty |

For applicable discovery requests, empty means the tenant filter field is omitted. Results remain limited to resources visible to the authenticated identity.

## Invalid Input Contract

Validation occurs after flag parsing and help bypass, but before configuration loading, service installation, file/stdin access, prompts, activity indicators, report creation, or remote requests.

| Condition | Error text | Error class / exit |
|-----------|------------|--------------------|
| `--all-tenants` active plus explicitly changed `--tenant`, including `--tenant ""` | `--tenant cannot be combined with --all-tenants` | `invalid_input`, exit 2 |
| active flag on a concrete-destination command | `--all-tenants cannot be used with <command>; this command requires a concrete destination tenant` | `invalid_input`, exit 2 |

`<command>` is the stable command path used by existing command error/help conventions. Structured error modes use the existing command envelope and error classification; usage remains silenced according to current invalid-input behavior.

## Command Support Contract

| Command class | Support state | Behavior |
|---------------|---------------|----------|
| Search/list/discovery, mixed discovery-plus-mutation, and tenant-aware ops | `accepted` | Existing tenant selection is cleared before execution. |
| `deploy process-definition` | `rejected_concrete_destination` | Reject before file inspection or deployment work. |
| `embed deploy` | `rejected_concrete_destination` | Reject before embedded resource access, deployment, or optional run. |
| `run process-instance` | `rejected_concrete_destination` | Reject before input/prompt/activity/request work. |
| `ops execute smoke-test` | `rejected_concrete_destination` | Reject before planning or execution, including dry-run. |
| Direct resource-key command | `accepted` | Existing key lookup and backend authorization are unchanged; the flag grants no new access. |
| Tenant-neutral command | `accepted` | Flag parsing is safe; command semantics remain unchanged. |

The runtime validator and capability generator MUST read the same command annotation. Unannotated commands default to `accepted`.

## Human Output Contract

When the configured tenant is named and active all-tenants broadens it, ordinary human tenant context renders in this order:

```text
configured tenant: <tenant>
warning: --all-tenants overrides the configured tenant filter; selection is unfiltered
selection scope: unfiltered across accessible tenants
```

The required warning text is exactly:

```text
--all-tenants overrides the configured tenant filter; selection is unfiltered
```

The existing renderer controls presentation prefixes and durable-progress severity. The message is rendered once per command execution. If the configured tenant was already empty, the override warning and configured-tenant provenance are omitted; ordinary unfiltered scope remains.

The existing explicit `--tenant ""` warning and wording are unchanged.

## Output-Mode Isolation

| Mode | Override warning |
|------|------------------|
| Ordinary human output | Visible when a named filter was broadened. |
| Durable human progress/report | Visible once through existing warning routing. |
| JSON | Suppressed; output remains parseable. |
| YAML | Suppressed; output remains parseable. |
| keys-only | Suppressed; exactly one key per line. |
| total-only | Suppressed; only the established total output. |
| quiet | Suppressed. |

Public structured `tenantContext` is unchanged. It continues to represent effective unfiltered selection with `filter: "none"` and existing warning semantics, without command-line provenance.

## Capability Contract

Each command capability gains one additive field:

```json
{
  "path": "search process-instance",
  "allTenantsSupport": "accepted"
}
```

or:

```json
{
  "path": "run process-instance",
  "allTenantsSupport": "rejected_concrete_destination"
}
```

The inherited flag contract continues to appear with:

- name: `all-tenants`
- type: boolean
- default: false
- scope: inherited/root persistent

Capability document version remains `v1` because the field is additive and no existing field meaning changes. Human capability/help output must communicate the same per-command support state.

## Authorization and Compatibility

- “All tenants” means no tenant filter within the authenticated caller's visibility; it is not tenant enumeration and not an authorization bypass.
- No mapping to `WithIgnoreTenant` or equivalent behavior is allowed.
- Existing backend 403/404 and resource-visibility behavior remains authoritative for direct-key operations.
- Existing invocations without active `--all-tenants` are unchanged across Camunda 8.7, 8.8, 8.9, and 8.10.
- No public facade, service contract, adapter, request schema, or generated client changes are part of this contract.
