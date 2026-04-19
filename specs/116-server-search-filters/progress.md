# Ralph Progress Log

Feature: 116-server-search-filters
Started: 2026-04-19 18:38:11

## Codebase Patterns

- `get process-instance` intentionally keeps CLI flag parsing in `populatePISearchFilterOpts()` and post-fetch narrowing in `applyPISearchResultFilters(...)`; request pushdown work should preserve that command-to-shared-filter flow instead of adding version branches in `cmd/`.
- Shared process-instance filter semantics must stay mirrored between `c8volt/process/model.go`, `internal/domain/processinstance.go`, and `c8volt/process/convert.go`; client regression coverage belongs in `c8volt/process/client_test.go`.
- Version-specific request-shape assertions already live in `internal/services/processinstance/v87/service_test.go`, `v88/service_test.go`, and `v89/service_test.go`; command paging/request helpers live in `cmd/cmd_processinstance_test.go`.
- Shared optional filter semantics should be represented as `*bool` fields so unset callers preserve current request shapes while later request builders can distinguish explicit `true` and `false` pushdown intent.
- Supported parent-presence pushdown should continue to prefer explicit `ParentKey` equality when present and otherwise encode `HasParent` through the generated `parentProcessInstanceKey` `"$exists"` filter; incident presence remains a plain `hasIncident` boolean on `v8.8` and `v8.9`.

---

## Iteration 1 - 2026-04-19 19:10 CEST
**User Story**: Setup (Shared Infrastructure)
**Tasks Completed**:
- [x] T001: Inventory the current late-filtering seam and supported-version request capabilities
- [x] T002: Confirm the shared filter-model seams and existing mapping coverage
- [x] T003: Inspect the existing request-capture and paging regression anchors
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/116-server-search-filters/progress.md
- specs/116-server-search-filters/research.md
- specs/116-server-search-filters/tasks.md
**Learnings**:
- `v8.7` request construction is still equality-only around `searchProcessInstancesRequest(...)`, so parent/incident presence semantics must stay on fallback there.
- `v8.8` and `v8.9` already have request-capture seams that can absorb the new pushdown assertions without introducing new test scaffolding.
- The command paging regressions should extend existing helpers in `cmd/cmd_processinstance_test.go` and `cmd/get_processinstance_test.go` rather than duplicating capture servers.
---

## Iteration 2 - 2026-04-19 18:44 CEST
**User Story**: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T004: Define the authoritative pushdown contract and version matrix
- [x] T005: Extend the shared process-instance filter model for parent-presence and incident-presence semantics
- [x] T006: Update public-to-domain filter mapping and shared client coverage for the new filter fields
- [x] T007: Refresh planning artifacts to reflect the finalized shared filter vocabulary
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/model.go
- internal/domain/processinstance.go
- c8volt/process/convert.go
- c8volt/process/client_test.go
- specs/116-server-search-filters/contracts/process-instance-search-filters.md
- specs/116-server-search-filters/research.md
- specs/116-server-search-filters/plan.md
- specs/116-server-search-filters/data-model.md
- specs/116-server-search-filters/quickstart.md
- specs/116-server-search-filters/tasks.md
- specs/116-server-search-filters/progress.md
**Learnings**:
- The shared pushdown vocabulary fits the existing facade and service seam cleanly as optional `*bool` fields, so later command and versioned-service work can express roots, children, and incident presence without overloading `ParentKey`.
- `c8volt/process/client.go` already routes both list and page searches through `toDomainProcessInstanceFilter(...)`, so one mapping update plus facade regression coverage protects both call paths.
- The foundational artifact set now uses the same authoritative vocabulary, which should keep the `v8.8` and `v8.9` request-builder tasks from drifting on field semantics.
---
## Iteration 3 - 2026-04-19 18:52 CEST
**User Story**: User Story 1 - Return only matching process instances per page
**Tasks Completed**:
- [x] T008: Add shared filter-mapping regression coverage for parent-presence and incident-presence semantics
- [x] T009: Add `v8.8` request-capture tests for parent-presence and incident-presence pushdown
- [x] T010: Add `v8.9` request-capture tests for parent-presence and incident-presence pushdown
- [x] T011: Add command paging regressions for supported-version filtered page behavior
- [x] T012: Translate supported list-mode flags into the shared request-capable filter fields
- [x] T013: Implement `v8.8` request-side encoding for parent-presence and incident-presence filters
- [x] T014: Implement `v8.9` request-side encoding for parent-presence and incident-presence filters
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/client_test.go
- cmd/cmd_processinstance_test.go
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- internal/services/common/filter.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v88/service_test.go
- internal/services/processinstance/v89/convert.go
- internal/services/processinstance/v89/service.go
- internal/services/processinstance/v89/service_test.go
- specs/116-server-search-filters/progress.md
- specs/116-server-search-filters/tasks.md
**Learnings**:
- The existing command paging diagnostics already become accurate once supported filters narrow the fetched page server-side, so the user-visible fix is primarily request construction plus regression coverage rather than a prompt-format change.
- `v8.8` can reuse shared `internal/services/common` filter builders for advanced parent-key existence filters, while `v8.9` needs the same logic in its local JSON-union helpers.
- Keeping `applyPISearchResultFilters(...)` in place after adding request pushdown preserves behavior while supported versions effectively no-op those same filters against already narrowed pages.
---
