# CLI Contract: Tenant Context

## Structured object

Every applicable structured surface uses this object and these field names:

```json
{
  "tenantContext": {
    "mode": "discovery",
    "filter": "none",
    "resolvedTenantIds": ["tenant-a", "tenant-b"],
    "unknownTargetCount": 1,
    "crossTenant": true,
    "warnings": [
      {
        "code": "unfiltered_selection",
        "message": "selection scope: unfiltered across accessible tenants"
      },
      {
        "code": "multiple_tenants",
        "message": "affected tenants: tenant-a, tenant-b"
      },
      {
        "code": "unknown_target_tenants",
        "message": "tenant metadata is unknown for 1 target"
      }
    ]
  }
}
```

`resolvedTenantIds`, `unknownTargetCount`, and `crossTenant` are always present. Optional identifier fields are omitted rather than emitted as empty strings.

### Mode examples

Named discovery:

```json
{
  "mode": "discovery",
  "filter": "named",
  "configuredTenantId": "tenant-a",
  "resolvedTenantIds": ["tenant-a"],
  "unknownTargetCount": 0,
  "crossTenant": false
}
```

Default creation:

```json
{
  "mode": "creation",
  "filter": "not_applicable",
  "targetTenantId": "<default>",
  "resolvedTenantIds": [],
  "unknownTargetCount": 0,
  "crossTenant": false
}
```

Explicit key with a configured/resolved mismatch:

```json
{
  "mode": "explicit_keys",
  "filter": "not_applied",
  "configuredTenantId": "tenant-a",
  "resolvedTenantIds": ["tenant-b"],
  "unknownTargetCount": 0,
  "crossTenant": false
}
```

## Shared result envelope

Full-contract JSON adds the optional common object without changing `payload`:

```json
{
  "outcome": "succeeded",
  "command": "cancel pi",
  "tenantContext": {
    "mode": "explicit_keys",
    "filter": "not_applied",
    "configuredTenantId": "tenant-a",
    "resolvedTenantIds": ["tenant-b"],
    "unknownTargetCount": 0,
    "crossTenant": false
  },
  "payload": {}
}
```

`outcome`, `class`, `command`, `payload`, and `detail` retain their existing meanings and shapes. Commands without applicable tenant semantics omit `tenantContext`. Limited/raw JSON views include the same object at their result root.

## Human contract

Render the applicable semantic line before the mutation or its confirmation:

- Named discovery: `selection scope: tenant-a only`
- Unfiltered discovery: `selection scope: unfiltered across accessible tenants`
- Named creation: `creation target: tenant-a`
- Default creation: `creation target: default tenant`
- Explicit keys: `selection scope: explicit resource keys; tenant filter not applied`
- One resolved tenant: `affected tenants: tenant-b`
- Multiple resolved tenants, warning-level: `affected tenants: tenant-a, tenant-b`
- Configured tenant before override: `configured tenant: tenant-a`
- Named-to-empty override warning: `--tenant "" overrides the configured tenant filter; selection is unfiltered`
- Named-to-different override information: `--tenant "tenant-b" overrides configured tenant filter`
- Empty-to-named override information: `--tenant "tenant-a" sets the tenant filter`
- Unknown metadata message, singular: `tenant metadata is unknown for 1 target`
- Unknown metadata message, plural: `tenant metadata is unknown for N targets`

Tenant IDs are unique and lexically sorted. Warning messages do not embed `WARN` or `WARNING:`; the output channel supplies severity exactly once. The cross-tenant warning uses the same `affected tenants: ...` summary instead of printing a second message with the same tenant IDs, remains visible in compact confirmation text, and is followed by the unknown warning when both apply.

## Output-mode matrix

| Surface | Contract |
|---|---|
| Human | Render semantic line and resolved evidence through existing human/warning channels before mutation. |
| JSON full contract | Emit one parseable document; place the common object beside the unchanged envelope payload. |
| JSON limited/raw | Emit one parseable document with the common object at the view root. |
| YAML | No new mode. `config show` remains one valid sanitized YAML document and includes the common object. |
| Quiet | Emit no new human tenant lines or warnings; existing machine result behavior remains unchanged. |
| Keys-only | Emit exactly one key per line on stdout and no tenant context or warnings there. |
| Ops Markdown report | Render operation-specific tenant fields and warnings; never label unfiltered work as default-tenant work. |
| Ops JSON report | Include the common root object; legacy `tenantId` is omitted for unfiltered discovery. |

## Command-family application

| Family | Base mode | Resolved evidence |
|---|---|---|
| `config validate`, effective `config show`, `config test-connection` | configuration | none required |
| PI cancel/delete/resolve selected by filters | discovery | frozen PI mutation plan |
| PI cancel/delete/resolve/update by explicit key | explicit keys | frozen PI/incident/variable plan |
| Process-definition delete by selector | discovery | definition plan items and cancellation plan |
| Process-definition delete by explicit key | explicit keys | definition plan items and cancellation plan |
| Job update | explicit keys | current job already loaded by plan |
| Deploy and run | creation | target before call; returned resource for final result |
| Retention and search-based purge | discovery | frozen workflow plan |
| Explicit repair/purge inputs | explicit keys | frozen incident/resource plan |
| Smoke test | creation for created resources | deployment/run result; cleanup evidence stays attached to the same audit |

## Audit compatibility

Affected reports keep their current `*.v1` schema identifiers and add `tenantContext`. The optional legacy `tenantId` field is deprecated:

- named discovery: it may contain the named filter for compatibility;
- creation: it may contain the target tenant for compatibility;
- unfiltered discovery: it must be omitted;
- explicit keys: it must be omitted unless it has an existing, unambiguous documented meaning independent of the configured filter.

New and updated readers must prefer `tenantContext`. Markdown must use the human contract above instead of a generic `Tenant` field.

## Behavioral invariants

- Context is evidence only; tenant selection, backend requests, and authorization do not change.
- Explicit keys continue to use backend authorization and are not locally rejected on tenant mismatch.
- Unknown metadata is non-blocking and never converted to `<default>`.
- A single known tenant plus unknown targets is not marked cross-tenant, but it does carry the unknown warning.
- An unfiltered search remains unfiltered even when every resolved target has one tenant.
- No backend call is made solely to enrich the object.
