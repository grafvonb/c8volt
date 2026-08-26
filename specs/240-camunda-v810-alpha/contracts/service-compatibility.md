# Contract: Alpha Service Compatibility

## Boundary rules

- Every version-aware factory has an explicit `V810Alpha` case.
- Every alpha adapter implements the existing version-neutral service API.
- Alpha adapters may import only `internal/clients/camunda/v810alpha/camunda` among generated Camunda clients.
- No alpha adapter imports Operate, Tasklist, or Administration SM clients.
- Domain models, public facades, CLI rendering, paging, retry, polling, and confirmation contracts remain version-neutral.
- Unsupported mutations return the shared unsupported classification before an external mutation.

## Required matrix

| Service family | Planned alpha outcome | Special verification |
|----------------|-----------------------|----------------------|
| Batch operations | Native alpha adapter | Batch lookup/search type and state conversion |
| Cluster | Native alpha adapter | Topology and license response conversion |
| Element instances | Native alpha adapter | Existing search/get behavior; new wait-state endpoint remains out of scope |
| Incidents | Native alpha adapter | Search/get/resolve and pre/post-mutation behavior |
| Jobs | Native alpha adapter | Search/update/complete/fail/throw contracts; new business-ID capability remains out of scope |
| Process definitions | Native alpha adapter | Search/get/XML/statistics and alpha search-filter shapes |
| Process instances | Native alpha adapter | Create/search/get/cancel/delete/walk/wait and alpha business-ID fields ignored unless already represented |
| Resources | Native alpha adapter | Deploy/get/delete plus 8.10 eventual-consistency retry; new resource search/content-binary coverage out of scope |
| Tenants | Native alpha adapter | Alpha identifier type conversions and existing tenant operations |
| User tasks | Native alpha adapter, unified API only | Direct keyed GET, tenant validation, no Tasklist fallback or job-based task claim |
| Variables | Native alpha adapter | Search/get/update behavior and alpha filter shapes |

## Capability contract

History-safe process-definition deletion is supported for `V89` and `V810Alpha`. Both direct process-definition deletion and all-process-definitions purge consume the same named predicate. V87 and V88 retain their existing unsupported result.

No generic version ordering is introduced. Future stable 8.10 support must explicitly join each capability set.

## New 8.10 surfaces

The generated client may contain agent-instance, form, wait-state, expanded resource/job, business-ID, and administration surfaces. Their presence does not add c8volt commands or public facade methods in this feature. They remain out of scope unless required to preserve an existing command contract.

## Verification

Each row requires:

- compile-time interface assertion
- factory selection test for `V810Alpha`
- constructor/client-boundary test
- at least one representative success test
- relevant error and malformed-response tests
- explicit unsupported test where a shared API operation cannot be fulfilled
- source import audit proving no older generated client is referenced
