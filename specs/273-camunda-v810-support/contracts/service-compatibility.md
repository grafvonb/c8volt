# Contract: Camunda V810 Service Compatibility

## Boundary Rules

- Every version-aware factory has an explicit `toolx.V810` case.
- Every V810 adapter implements the existing version-neutral API.
- V810 adapters import only `internal/clients/camunda/v810/camunda` among generated clients.
- V810 adapters import no v87-v89 service package/client and no Operate, Tasklist, or Administration SM client.
- Conversion stays inside each `v810` package.
- `cmd/` and `c8volt/` import no generated clients/versioned implementations.
- Paging, polling, retries, traversal, waiters, confirmation, errors, facades, and rendering remain version-neutral.
- Unsupported mutations return shared unsupported classification before external mutation.

## Required Matrix

| Service family | V810 outcome | Special verification |
|----------------|--------------|----------------------|
| Batch operations | Native unified adapter | Search/get type/state conversion |
| Cluster | Native unified adapter | Topology, broker, license, gateway version |
| Element instances | Native unified adapter | Existing get/search and paging |
| Incidents | Native unified adapter | Get/search/resolve and confirmation |
| Jobs | Native unified adapter | Existing get/search/mutation contracts |
| Process definitions | Native unified adapter | Search/get/XML/statistics/delete/filter shapes |
| Process instances | Native unified adapter | Create/search/get/cancel/delete/walk/wait; V810 variables |
| Resources | Native unified adapter | Deploy/get/delete and visibility confirmation |
| Tenants | Native unified adapter | Identifier conversion/existing operations |
| User tasks | Unified only | No Tasklist fallback; explicit unavailable outcome |
| Variables | Native unified adapter | Search/get/update and value conversion |

New 8.10 schemas do not add commands/public APIs unless an existing workflow requires them.

## Capability and Fixture Contracts

- Full process-definition history deletion includes V89 and V810 through one named predicate; V87/V88 retain pre-mutation rejection.
- No generic version ordering is introduced.
- V810 production embedded/smoke fixtures map explicitly to `C89_` while reports remain `8.10`.
- No C810 fixture or integration selection is added.

## Verification Per Family

Each row requires a compile-time assertion, V810 factory test, local generated-client boundary test, representative success/error/malformed tests, mutation/confirmation proof where applicable, explicit unsupported-before-mutation proof where needed, and source scan rejecting old/removed clients.

The top-level `c8volt.New` test proves ten direct factories plus the nested variable factory form a complete V810 client.
