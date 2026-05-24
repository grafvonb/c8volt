# Contract: Tenant Scope For Discovery And Explicit Admin Keys

## Affected Command Families

- `c8volt get process-instance`
- `c8volt walk process-instance`
- `c8volt expect process-instance`
- `c8volt cancel process-instance`
- `c8volt delete process-instance`
- `c8volt get process-definition`
- `c8volt delete process-definition`
- `c8volt get resource`
- Bulk and stdin-key paths that feed the affected commands
- Create, deploy, and run flows that use existing tenant and `<default>` behavior

## Tenant Semantics

| Input Source | Contract |
|--------------|----------|
| Discovery/search/list filters | Apply the selected tenant where supported by Camunda 8.8 or 8.9. |
| Search-derived cancel/delete candidates | Mutate or preview only the tenant-scoped discovered candidate set and its intended dependency scope. |
| Direct `--key` process-instance input | Treat as backend-authorized admin input. Do not reject solely because returned tenant metadata differs from selected `--tenant`. |
| Direct process-definition key or resource ID input | Treat as backend-authorized admin input. Do not reject solely because returned tenant metadata differs from selected `--tenant`. |
| Stdin keys | Treat as explicit operator-supplied admin input after existing key validation and deduplication. |
| Create/deploy/run flows | Preserve existing selected-tenant and `<default>` tenant behavior. |

## Output Contract

Returned tenant metadata may continue to appear in human, JSON, keys-only-adjacent detail, log, or report output where the command already exposes it.

For explicit admin input:

```text
selected tenant != returned tenant
Camunda returned or accepted target
=> c8volt proceeds through existing command behavior
```

For discovery-derived candidates:

```text
selected tenant = tenant-a
c8volt produced target set through tenant-scoped discovery/search
=> subsequent preview, confirmation, dependency expansion, and mutation remain tied to that tenant-scoped target meaning
```

## Error Contract

- Backend authorization, not-found, unsupported-version, and transport errors remain visible through existing error handling.
- c8volt must not convert explicit-input tenant metadata differences into local tenant mismatch errors.
- Local validation still rejects malformed keys, empty stdin, mutually exclusive flags, missing required selectors, unsafe destructive state, and unsupported automation paths according to existing command behavior.

## Documentation Contract

README tenant guidance and generated CLI docs must explain:

- `--tenant` scopes discovery, search, selection, create, deploy, and run flows where supported.
- Explicit keys, IDs, stdin keys, and direct flag values are admin input governed by Camunda backend authorization.
- c8volt may show returned tenant metadata so operators can see what the backend authorized.

## Compatibility Contract

- Behavior changes are limited to Camunda 8.8 and 8.9.
- Camunda 8.7 behavior remains unchanged.
- Existing `<default>` tenant behavior for create, deploy, and run flows remains unchanged.
