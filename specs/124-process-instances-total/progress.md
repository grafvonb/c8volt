## Codebase Patterns

- `get process-instance --total` stays command-local: `cmd/get_processinstance.go` should ask `SearchProcessInstancesPage` for the first page, trust `ReportedTotal` only when no command-local post-filtering is active, and otherwise keep counting through the existing paged search seam instead of widening the shared render-mode contract.
- `cmd/get_processinstance.go` owns process-instance search validation, keyed-vs-search branching, pagination orchestration, and the final handoff to `listProcessInstancesView`, so command-specific output changes belong there before shared render helpers are widened.
- `cmd/cmd_views_get.go` keeps process-instance detail rendering mode-agnostic through `listOrJSON`, with `--with-age` as a narrow JSON/one-line decoration; count-only output should short-circuit before this renderer to avoid creating a new global render mode.
- Shared process-instance page models in `internal/domain/processinstance.go` and `c8volt/process/model.go` now expose optional `ReportedTotal{Count, Kind}` metadata alongside `Request`, `Items`, and `OverflowState`, and `c8volt/process/convert.go` mirrors that seam directly so command code can stay version-agnostic once services populate it.
- Versioned search services already see backend total signals: `v87` trims Operate results with an optional `payload.Total`, while `v88` and `v89` compute overflow from `Page.TotalItems` and `HasMoreTotalItems`; the command layer currently loses that data because services only return `OverflowState`.
- Existing regression anchors are already close to the needed feature seams: `cmd/get_processinstance_test.go` covers command help and search request behavior, `cmd/cmd_processinstance_test.go` provides reusable process-instance search fixtures, `c8volt/process/client_test.go` covers cross-version page conversion, and versioned service tests assert paging metadata behavior around `OverflowState` and capped totals.
- `ReportedTotal=nil` is the shared unavailable signal, while `ReportedTotal.Kind` uses `exact` and `lower_bound`; that keeps absence distinct from a real numeric zero total and avoids inventing a third enum state in the model.

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
## Iteration 2 - 2026-04-22 22:21:25 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T004: Define the authoritative `--total` command contract, invalid combinations, and lower-bound total rule
- [x] T005: Extend the shared process-instance page model for reported totals and exact-vs-lower-bound semantics
- [x] T006: Update public/domain conversion seams and shared client coverage for the new page metadata
- [x] T007: Refresh the feature data model and quickstart guidance for the finalized reported-total vocabulary
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/client_test.go
- c8volt/process/convert.go
- c8volt/process/model.go
- internal/domain/processinstance.go
- specs/124-process-instances-total/contracts/process-instance-total-output.md
- specs/124-process-instances-total/data-model.md
- specs/124-process-instances-total/plan.md
- specs/124-process-instances-total/progress.md
- specs/124-process-instances-total/quickstart.md
- specs/124-process-instances-total/research.md
- specs/124-process-instances-total/tasks.md
**Learnings**:
- The smallest stable seam is an optional `ReportedTotal` pointer on `ProcessInstancePage`; it lets future command logic distinguish unavailable totals from zero-match totals without adding a separate availability flag.
- `c8volt/process/client_test.go` is sufficient to lock the public conversion contract for reported totals before versioned services start populating the new metadata.
- The shared model can absorb exact-vs-lower-bound semantics now without changing existing service paging behavior, which keeps the foundational work unit isolated from the first user story.
---
## Iteration 3 - 2026-04-22 22:28:04 CEST
**User Story**: User Story 1 - Return Count Only
**Tasks Completed**:
- [x] T008: Add command regressions for numeric-only `--total` output and zero-match behavior
- [x] T009: Add shared page-metadata conversion coverage for count-only output
- [x] T010: Add the `--total` flag and search-mode count-only command path
- [x] T011: Populate reported-total metadata from `v87`, `v88`, and `v89` search page responses
- [x] T012: Keep non-`--total` detail rendering unchanged while short-circuiting count-only output before the existing list renderer
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/client_test.go
- cmd/cmd_views_get.go
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- internal/services/processinstance/v87/service.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v89/service.go
- specs/124-process-instances-total/progress.md
- specs/124-process-instances-total/tasks.md
**Learnings**:
- Count-only output can stay out of the shared render-mode model by adding a small command-local `processInstanceTotalView` and returning before `listProcessInstancesView`.
- `ReportedTotal` is enough to skip extra paging on the first page for normal `--total` searches, while command-local fallback counting remains necessary when post-search filtering could change the visible total.
- The existing focused package tests already cover the safest validation boundary for this story: `./cmd`, `./c8volt/process`, and `./internal/services/processinstance/...`.
---
