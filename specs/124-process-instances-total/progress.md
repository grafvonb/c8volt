## Codebase Patterns

- `cmd/get_processinstance.go` owns process-instance search validation, keyed-vs-search branching, pagination orchestration, and the final handoff to `listProcessInstancesView`, so command-specific output changes belong there before shared render helpers are widened.
- `cmd/cmd_views_get.go` keeps process-instance detail rendering mode-agnostic through `listOrJSON`, with `--with-age` as a narrow JSON/one-line decoration; count-only output should short-circuit before this renderer to avoid creating a new global render mode.
- Shared process-instance page models in `internal/domain/processinstance.go` and `c8volt/process/model.go` currently expose only `Request`, `Items`, and `OverflowState`, and `c8volt/process/convert.go` mirrors that seam directly, so any total metadata must be added in both layers and conversion helpers together.
- Versioned search services already see backend total signals: `v87` trims Operate results with an optional `payload.Total`, while `v88` and `v89` compute overflow from `Page.TotalItems` and `HasMoreTotalItems`; the command layer currently loses that data because services only return `OverflowState`.
- Existing regression anchors are already close to the needed feature seams: `cmd/get_processinstance_test.go` covers command help and search request behavior, `cmd/cmd_processinstance_test.go` provides reusable process-instance search fixtures, `c8volt/process/client_test.go` covers cross-version page conversion, and versioned service tests assert paging metadata behavior around `OverflowState` and capped totals.

---
## Iteration 1 - 2026-04-22 22:15:51 CEST
**User Story**: Phase 1: Setup (Shared Infrastructure)
**Tasks Completed**:
- [x] T001: Inventory the current `get process-instance` render/validation flow and count-related response seams
- [x] T002: Confirm the shared page-model and conversion seams for reported totals
- [x] T003: Confirm the command, service, and docs regression anchors for `--total`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/124-process-instances-total/tasks.md
- specs/124-process-instances-total/progress.md
**Learnings**:
- `searchProcessInstancesWithPaging` already centralizes list-mode pagination, so `--total` can stay on the existing search path without introducing a parallel fetch flow.
- The service layer already distinguishes exact-vs-capped totals implicitly via `TotalItems` and `HasMoreTotalItems` in `v8.8`/`v8.9`, but the shared page model drops that distinction before it reaches `cmd/`.
- `README.md` and generated CLI docs currently have no `--total` coverage, and the repository-standard regeneration path is `make docs-content`.
---
