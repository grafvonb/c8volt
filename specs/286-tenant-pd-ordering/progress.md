# Ralph Progress Log

Feature: 286-tenant-pd-ordering
Started: 2026-08-31 13:37:23

## Iteration 9 - 2026-08-31 14:10
**Work Unit**: US1 Camunda 8.10 ordinary adapter canonical ordering
**Tasks Completed**:
- [x] T010: Add Camunda 8.10 ordinary request-sort and returned-order assertions to `internal/services/processdefinition/v810/service_test.go`
- [x] T014: Encode Camunda 8.10 ordinary backend sorting as tenant ID ASC, process definition ID ASC, version DESC, and process definition key ASC, then use the domain canonical sort for returned collections in `internal/services/processdefinition/v810/service.go`
**Tasks Remaining in Work Unit**: 0; US1 complete
**Commit**: This work-unit commit
**Files Changed**:
- internal/services/processdefinition/v810/service.go
- internal/services/processdefinition/v810/service_test.go
- specs/286-tenant-pd-ordering/tasks.md
- specs/286-tenant-pd-ordering/ralph-memory.md
- specs/286-tenant-pd-ordering/progress.md
**Learnings**:
- v8.10 ordinary search matches the v8.9 manually marshalled request-body path and now uses the same canonical ordinary sort tuple before final domain normalization.
---
## Iteration 1 - 2026-08-31 13:39
**Work Unit**: Setup baseline validation
**Tasks Completed**:
- [x] T001: Run the pre-change focused validation commands and record any baseline failures before editing, using `specs/286-tenant-pd-ordering/quickstart.md`
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- specs/286-tenant-pd-ordering/tasks.md
- specs/286-tenant-pd-ordering/ralph-memory.md
- specs/286-tenant-pd-ordering/progress.md
**Learnings**:
- Focused quickstart baseline passed for domain, shared process-definition service, v87-v810 adapters, public process facade, and command tests; the domain command currently has no matching tests to run.
---
---
## Iteration 2 - 2026-08-31 13:43
**Work Unit**: Foundational canonical process-definition comparator
**Tasks Completed**:
- [x] T002: Add table-driven canonical comparator tests for `<default>` placement, case-sensitive tenant/BPMN identity, versions 9 and 10, lexical keys `10` and `2`, empty/single collections, and shuffled inputs in `internal/domain/processdefinition_test.go`
- [x] T003: Implement and document the reusable canonical process-definition comparator and sort function `(tenantId ASC, bpmnProcessId ASC, version DESC, key ASC)` without changing unrelated sort helpers in `internal/domain/processdefinition.go`
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- internal/domain/processdefinition.go
- internal/domain/processdefinition_test.go
- specs/286-tenant-pd-ordering/tasks.md
- specs/286-tenant-pd-ordering/ralph-memory.md
- specs/286-tenant-pd-ordering/progress.md
**Learnings**:
- Domain canonical ordering now has a reusable comparator and sort function with edge-case coverage; the focused regex was updated to include the new comparator/sort tests.
---
---
## Iteration 3 - 2026-08-31 13:46
**Work Unit**: US1 service final canonical ordering
**Tasks Completed**:
- [x] T004: Add page-arrival independence and final canonical collection-order tests to `internal/services/processdefinition/search_test.go`
- [x] T015: Canonically sort the accumulated ordinary result only after service-owned page traversal while preserving exact-once membership, visitor progress metadata, and existing ordinary limit semantics in `internal/services/processdefinition/search.go`
**Tasks Remaining in Work Unit**: US1 has 11 incomplete tasks remaining (T005-T014)
**Commit**: This work-unit commit
**Files Changed**:
- internal/services/processdefinition/search.go
- internal/services/processdefinition/search_test.go
- specs/286-tenant-pd-ordering/tasks.md
- specs/286-tenant-pd-ordering/ralph-memory.md
- specs/286-tenant-pd-ordering/progress.md
**Learnings**:
- Final service results are canonical after traversal; visitor cumulative counts remain tied to selected page accumulation before final sorting.
---
---
## Iteration 4 - 2026-08-31 13:50
**Work Unit**: US1 public facade order-preservation tests
**Tasks Completed**:
- [x] T005: Add public ordinary-search and paged-search order-preservation tests to `c8volt/process/client_test.go`
**Tasks Remaining in Work Unit**: 0; US1 has 10 incomplete tasks remaining (T006-T014)
**Commit**: This work-unit commit
**Files Changed**:
- c8volt/process/client_test.go
- specs/286-tenant-pd-ordering/tasks.md
- specs/286-tenant-pd-ordering/ralph-memory.md
- specs/286-tenant-pd-ordering/progress.md
**Learnings**:
- Facade conversion preserves service slice order for ordinary search, and paged facade results inherit final normalization from `pdsvc.SearchProcessDefinitionsPages` without reordering visitor page callbacks.
---
---
## Iteration 5 - 2026-08-31 13:55
**Work Unit**: US1 command-level tenant-scope ordering tests
**Tasks Completed**:
- [x] T006: Add command-level filtered and `--all-tenants` canonical ordering tests to `cmd/get_processdefinition_test.go`
**Tasks Remaining in Work Unit**: 0; US1 has 9 incomplete tasks remaining (T007-T014)
**Commit**: This work-unit commit
**Files Changed**:
- cmd/get_processdefinition_test.go
- specs/286-tenant-pd-ordering/tasks.md
- specs/286-tenant-pd-ordering/ralph-memory.md
- specs/286-tenant-pd-ordering/progress.md
**Learnings**:
- Tenant-filtered and `--all-tenants` command listings already preserve the shared paged service canonical sequence; avoid setting `--batch-size` in command tests unless the test also resets Cobra changed-state for later package tests.
---
---
## Iteration 6 - 2026-08-31 13:59
**Work Unit**: US1 Camunda 8.7 ordinary adapter canonical ordering
**Tasks Completed**:
- [x] T007: Add Camunda 8.7 request-sort and returned-order assertions for tenant/BPMN/version/key to `internal/services/processdefinition/v87/service_test.go`
- [x] T011: Encode Operate 8.7 backend sorting as tenant ID ASC, BPMN process ID ASC, version DESC, and key ASC, then use the domain canonical sort for returned collections in `internal/services/processdefinition/v87/service.go`
**Tasks Remaining in Work Unit**: 0; US1 has 6 incomplete tasks remaining (T008-T010, T012-T014)
**Commit**: This work-unit commit
**Files Changed**:
- internal/services/processdefinition/v87/service.go
- internal/services/processdefinition/v87/service_test.go
- specs/286-tenant-pd-ordering/tasks.md
- specs/286-tenant-pd-ordering/ralph-memory.md
- specs/286-tenant-pd-ordering/progress.md
**Learnings**:
- v8.7 ordinary search can use the generic Operate `Sort` model for the full canonical tuple, while final result normalization remains local through the shared domain comparator.
---
---
## Iteration 7 - 2026-08-31 14:03
**Work Unit**: US1 Camunda 8.8 ordinary adapter canonical ordering
**Tasks Completed**:
- [x] T008: Add Camunda 8.8 ordinary request-sort and returned-order assertions to `internal/services/processdefinition/v88/service_test.go`
- [x] T012: Encode Camunda 8.8 ordinary backend sorting as tenant ID ASC, process definition ID ASC, version DESC, and process definition key ASC, then use the domain canonical sort for returned collections in `internal/services/processdefinition/v88/service.go`
**Tasks Remaining in Work Unit**: 0; US1 has 4 incomplete tasks remaining (T009-T010, T013-T014)
**Commit**: This work-unit commit
**Files Changed**:
- internal/services/processdefinition/v88/service.go
- internal/services/processdefinition/v88/service_test.go
- specs/286-tenant-pd-ordering/tasks.md
- specs/286-tenant-pd-ordering/ralph-memory.md
- specs/286-tenant-pd-ordering/progress.md
**Learnings**:
- v8.8 ordinary search uses Camunda v2 `processDefinitionKey` as the final backend tie-breaker; native latest sorting remains intentionally unchanged for later latest tasks.
---
---
## Iteration 8 - 2026-08-31 14:06
**Work Unit**: US1 Camunda 8.9 ordinary adapter canonical ordering
**Tasks Completed**:
- [x] T009: Add Camunda 8.9 ordinary request-sort and returned-order assertions to `internal/services/processdefinition/v89/service_test.go`
- [x] T013: Encode Camunda 8.9 ordinary backend sorting as tenant ID ASC, process definition ID ASC, version DESC, and process definition key ASC, then use the domain canonical sort for returned collections in `internal/services/processdefinition/v89/service.go`
**Tasks Remaining in Work Unit**: 0; US1 has 2 incomplete tasks remaining (T010, T014)
**Commit**: This work-unit commit
**Files Changed**:
- internal/services/processdefinition/v89/service.go
- internal/services/processdefinition/v89/service_test.go
- specs/286-tenant-pd-ordering/tasks.md
- specs/286-tenant-pd-ordering/ralph-memory.md
- specs/286-tenant-pd-ordering/progress.md
**Learnings**:
- v8.9 ordinary search uses a manually marshalled JSON request body, so request-sort tests decode the raw body and compare literal generated sort field values.
---
---
## Iteration 10 - 2026-08-31 14:13
**Work Unit**: US2 facade conversion and statistics association coverage
**Tasks Completed**:
- [x] T016: Add facade conversion tests proving canonical slice order and per-key statistics association survive public mapping in `c8volt/process/client_test.go`
**Tasks Remaining in Work Unit**: 0; US2 has 6 incomplete tasks remaining (T017-T022)
**Commit**: This work-unit commit
**Files Changed**:
- c8volt/process/client_test.go
- specs/286-tenant-pd-ordering/tasks.md
- specs/286-tenant-pd-ordering/ralph-memory.md
- specs/286-tenant-pd-ordering/progress.md
**Learnings**:
- Existing facade conversion preserves both the service-provided key sequence and per-key statistics association for ordinary process-definition searches with `WithStat`.
---
---
## Iteration 11 - 2026-08-31 14:17
**Work Unit**: US2 renderer output ordering parity
**Tasks Completed**:
- [x] T017: Expand the shared renderer fixture and assert identical canonical key sequences for human, JSON, keys-only, and watch rendering in `cmd/cmd_views_processdefinition_test.go`
**Tasks Remaining in Work Unit**: 0; US2 has 5 incomplete tasks remaining (T018-T022)
**Commit**: This work-unit commit
**Files Changed**:
- cmd/cmd_views_processdefinition_test.go
- specs/286-tenant-pd-ordering/tasks.md
- specs/286-tenant-pd-ordering/ralph-memory.md
- specs/286-tenant-pd-ordering/progress.md
**Learnings**:
- Process-definition renderers consume the supplied slice order consistently across human rows, JSON envelope arrays, keys-only lines, and watch refresh bodies.
---
---
## Iteration 12 - 2026-08-31 14:21
**Work Unit**: US2 service watch snapshot ordering coverage
**Tasks Completed**:
- [x] T018: Add watch-snapshot tests proving statistics-only changes retain canonical positions and broad snapshots reuse paged collection order in `internal/services/processdefinition/search_test.go`
**Tasks Remaining in Work Unit**: 0; US2 has 4 incomplete tasks remaining (T019-T022)
**Commit**: This work-unit commit
**Files Changed**:
- internal/services/processdefinition/search_test.go
- specs/286-tenant-pd-ordering/tasks.md
- specs/286-tenant-pd-ordering/ralph-memory.md
- specs/286-tenant-pd-ordering/progress.md
**Learnings**:
- Broad process-definition watch snapshots already inherit canonical ordering from the service-owned paged collector; statistics-only refresh changes update per-key statistics without changing the returned key sequence.
---
---
## Iteration 13 - 2026-08-31 14:25
**Work Unit**: US2 command watch repeated-refresh row stability coverage
**Tasks Completed**:
- [x] T019: Add repeated-refresh command tests proving count changes do not move process-definition rows in `cmd/get_processdefinition_watch_test.go`
**Tasks Remaining in Work Unit**: 0; US2 has 3 incomplete tasks remaining (T020-T022)
**Commit**: This work-unit commit
**Files Changed**:
- cmd/get_processdefinition_watch_test.go
- specs/286-tenant-pd-ordering/tasks.md
- specs/286-tenant-pd-ordering/ralph-memory.md
- specs/286-tenant-pd-ordering/progress.md
**Learnings**:
- Command watch refresh bodies preserve snapshot row order across ten statistics-only refreshes; count changes render in place on the same process-definition keys.
---
---
## Iteration 14 - 2026-08-31 14:29
**Work Unit**: US2 Camunda 8.8 with-stat ordering parity
**Tasks Completed**:
- [x] T020: Add with-stat parity coverage and keep Camunda 8.8 statistics enrichment associated by slice index without reordering in `internal/services/processdefinition/v88/service_test.go` and `internal/services/processdefinition/v88/service.go`
**Tasks Remaining in Work Unit**: 0; US2 has 2 incomplete tasks remaining (T021-T022)
**Commit**: This work-unit commit
**Files Changed**:
- internal/services/processdefinition/v88/service.go
- internal/services/processdefinition/v88/service_test.go
- specs/286-tenant-pd-ordering/tasks.md
- specs/286-tenant-pd-ordering/ralph-memory.md
- specs/286-tenant-pd-ordering/progress.md
**Learnings**:
- v8.8 ordinary with-stat search preserves the same canonical key sequence as non-stat search, with counts attached to the matching process-definition keys after final sorting.
---
---
## Iteration 15 - 2026-08-31 14:33
**Work Unit**: US2 Camunda 8.9 with-stat ordering parity
**Tasks Completed**:
- [x] T021: Add with-stat parity coverage and keep Camunda 8.9 statistics enrichment associated by slice index without reordering in `internal/services/processdefinition/v89/service_test.go` and `internal/services/processdefinition/v89/service.go`
**Tasks Remaining in Work Unit**: 0; US2 has 1 incomplete task remaining (T022)
**Commit**: This work-unit commit
**Files Changed**:
- internal/services/processdefinition/v89/service.go
- internal/services/processdefinition/v89/service_test.go
- specs/286-tenant-pd-ordering/tasks.md
- specs/286-tenant-pd-ordering/ralph-memory.md
- specs/286-tenant-pd-ordering/progress.md
**Learnings**:
- v8.9 ordinary with-stat search preserves the same canonical key sequence as non-stat search, with counts attached to the matching process-definition keys after final sorting.
---
---
## Iteration 16 - 2026-08-31 14:36
**Work Unit**: US2 Camunda 8.10 with-stat ordering parity
**Tasks Completed**:
- [x] T022: Add with-stat parity coverage and keep Camunda 8.10 statistics enrichment associated by slice index without reordering in `internal/services/processdefinition/v810/service_test.go` and `internal/services/processdefinition/v810/service.go`
**Tasks Remaining in Work Unit**: 0; US2 complete, next incomplete story is US3 starting at T023
**Commit**: This work-unit commit
**Files Changed**:
- internal/services/processdefinition/v810/service.go
- internal/services/processdefinition/v810/service_test.go
- specs/286-tenant-pd-ordering/tasks.md
- specs/286-tenant-pd-ordering/ralph-memory.md
- specs/286-tenant-pd-ordering/progress.md
**Learnings**:
- v8.10 ordinary with-stat search preserves the same canonical key sequence as non-stat search, with counts attached to the matching process-definition keys after final sorting.
---
---
## Iteration 17 - 2026-08-31 14:42
**Work Unit**: US3 shared latest traversal and reduction
**Tasks Completed**:
- [x] T023: Add service tests for complete cursor/offset traversal, page sizes 1/2/1000, exact tenant/BPMN latest grouping, tied-version lexical key choice, post-reduction limiting, and latest watch paging in `internal/services/processdefinition/search_test.go`
- [x] T030: Add the `Latest` intent to domain/public search requests and map it without changing serialized response contracts in `internal/domain/processdefinition.go`, `c8volt/process/model.go`, and `c8volt/process/convert.go`
- [x] T031: Extend service-owned traversal to collect complete latest candidates, reduce by exact `(tenantId, bpmnProcessId)` with version/key tie rules, sort canonically, apply latest limits after reduction, and route latest watch snapshots through the same path in `internal/services/processdefinition/search.go`
**Tasks Remaining in Work Unit**: 0; US3 has 12 incomplete tasks remaining (T024-T029, T032-T037)
**Commit**: This work-unit commit
**Files Changed**:
- c8volt/process/convert.go
- c8volt/process/model.go
- c8volt/process/model_test.go
- internal/domain/processdefinition.go
- internal/services/processdefinition/search.go
- internal/services/processdefinition/search_test.go
- specs/286-tenant-pd-ordering/tasks.md
- specs/286-tenant-pd-ordering/ralph-memory.md
- specs/286-tenant-pd-ordering/progress.md
**Learnings**:
- Shared latest traversal now uses complete page collection before exact tenant/BPMN reduction, so latest limits are applied to the canonical reduced collection rather than raw page arrivals.
---
