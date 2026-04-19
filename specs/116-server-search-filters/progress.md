# Ralph Progress Log

Feature: 116-server-search-filters
Started: 2026-04-19 18:38:11

## Codebase Patterns

- `get process-instance` intentionally keeps CLI flag parsing in `populatePISearchFilterOpts()` and post-fetch narrowing in `applyPISearchResultFilters(...)`; request pushdown work should preserve that command-to-shared-filter flow instead of adding version branches in `cmd/`.
- Shared process-instance filter semantics must stay mirrored between `c8volt/process/model.go`, `internal/domain/processinstance.go`, and `c8volt/process/convert.go`; client regression coverage belongs in `c8volt/process/client_test.go`.
- Version-specific request-shape assertions already live in `internal/services/processinstance/v87/service_test.go`, `v88/service_test.go`, and `v89/service_test.go`; command paging/request helpers live in `cmd/cmd_processinstance_test.go`.

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
