# Facade and Service Contract: User Task Reads

## Public facade (`c8volt/task`)

Extend the existing API without changing its constructor or resolver signatures. Proposed method contracts:

| Method | Inputs after context | Result |
| --- | --- | --- |
| GetUserTask | key string, facade options | UserTask, error |
| GetUserTasks | keys typex.Keys, wantedWorkers int, facade options | UserTasks, error |
| SearchUserTasks | SearchRequest, facade options | UserTasks, error |
| SearchUserTasksPages | SearchRequest, SearchPageVisitor, facade options | SearchPagesResult, error |
| SearchUserTasksTotal | SearchRequest, facade options | int64, error |

All options use `foptions.FacadeOption`; public signatures contain no generated client or internal domain/service types. Public models follow [data-model.md](../data-model.md). The existing root `c8volt.API` embeds `task.API`, making the new methods available through the current client wiring.

Map options with `MapFacadeOptionsToCallOptions`, copy/map values in `convert.go`, and map errors through `ferrors.FromDomain`. For page visitors, map a domain step to the public step, call the visitor, and map its action back. Do not count, trim, retry, choose cursors, schedule workers, apply tenant filters, or reinterpret continuation in the facade.

The public single getter calls the new service native getter. Multi-key methods return ordered unique requested tasks, never a successful subset after a read failure. Existing `ResolveProcessInstanceKeyFromUserTask` and `ResolveProcessInstanceKeysFromUserTasks` retain their prior delegation path and behavior.

## Versioned adapter API (`internal/services/usertask`)

Keep legacy `GetUserTask(context, key, ...CallOption)` unchanged. Add:

| Method | Responsibility |
| --- | --- |
| GetNativeUserTask(context, key, ...CallOption) | Native direct GET, common payload validation, task mapping; no discovery tenant comparison and no Tasklist fallback |
| SearchUserTasksPage(context, UserTaskSearchQuery, UserTaskPageRequest, ...CallOption) | One native filtered search request; map effective tenant and filter unions; normalize items and page facts |

Each existing version contract and compile-time assertion must remain satisfied. Update impacted test stubs when the interface expands. Preserve factory behavior and public constructor inputs. Do not create a second service framework or use runtime type assertions to guess capabilities.

| Version | Native new methods | Filter differences | Legacy resolver |
| --- | --- | --- | --- |
| 8.7 | Explicit existing unsupported domain error; no new HTTP task call | None | Preserve current behavior |
| 8.8 | Generated native GET/search | Process selectors use scalar pointers; assignment/candidate/tenant equality uses generated filter properties | Tenant-scoped search plus Tasklist fallback preserved |
| 8.9 | Generated native GET/search | Same relevant selector shape as 8.8 | Tenant-scoped search plus Tasklist fallback preserved |
| 8.10 | Generated native GET/search | PI/PD keys and processDefinitionId use generated filter unions | Native getter's existing tenant/identity checks preserved for resolver |

BPMN process ID maps to `processDefinitionId`. All search filters are backend predicates; no local candidate/assignee filtering or enrichment requests. Map all nine supported states; compare version-specific generated enums in adapter tests. Keep generated files unchanged.

## Shared service functions

Implement sibling free functions `GetUserTasks`, `SearchUserTasks`, `SearchUserTasksPages`, and `SearchUserTasksTotal` in the usertask service area, taking the existing API or a narrow compile-time subset where useful. This follows `ResolveProcessInstanceKeys` ownership without a new workflow constructor.

- Bulk: apply call options at the owner, stable-deduplicate keys, use existing worker-count policy and `pool.ExecuteSlice`, pass context and relevant options onward, preserve ordering and joined error/fail-fast behavior. No per-key informational chatter by default.
- Traversal: initialize a limit page, normalize options once for the workflow, propagate relevant options to each adapter call, and prefer advancing end cursors. Fallback to offset when metadata requires continuation but no cursor is available. Keep raw progress distinct from selected item counts. Limit trimming and stop disposition are service-owned.
- Sparse pages: continue while an advancing cursor or unexhausted metadata indicates more. Do not infer exhaustion from an empty/short intermediate page. Advance empty offset pages by the requested size; nonempty pages by raw length. Check offset overflow and repeated/cyclic cursor before repeating a request.
- Completion: trustworthy exact totals can prove exhaustion; capped totals cannot. A final nonempty cursor can require a terminal probe. The global count-cap flag alone does not prove a next page: after traversing the known lower bound, an empty response with no advancing cursor and no other continuation evidence ends the fallback count. Before that bound, remaining population evidence requires progress across sparse pages. Inconsistent/no-progress metadata yields an existing classified read error rather than success or an endless loop.
- Count: ignore/reset collection limit internally, never invoke an interactive visitor, return a trustworthy exact total directly or count matching pages to authoritative completion using int64. Keep only current page and progress state. Errors return no usable numeric result.
- Collection: store selected items for ordinary collection/visitor results; provide count, pages, and typed completion disposition. Initialize empty item slices deliberately. JSON rendering occurs only after successful collection.

Use existing transport timeout/retry/auth/logging policy. Cancellation propagates; no new retry framework, snapshot guarantee, persistence, or background goroutines. Version-specific HTTP mechanics stay in adapters, and all shared mechanics remain below the facade.

## Contract tests

Pin serialized filters and page unions per version; scalar versus v810 union representations are not assumed interchangeable. Cover nil response payloads, HTTP failures, native 404, authorization failures, tenant metadata, optional values, sparse and capped pages, cursor progress/offset fallback, and request counts. Test facade option propagation, copying, visitor actions/errors, and ferrors conversion. Retain all legacy resolver and fallback tests unchanged in expected behavior.
