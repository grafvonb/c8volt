## Codebase Patterns

- `get process-instance --total` stays command-local: `cmd/get_processinstance.go` should ask `SearchProcessInstancesPage` for the first page, trust `ReportedTotal` only when no command-local post-filtering is active, and otherwise keep counting through the existing paged search seam instead of widening the shared render-mode contract.
- `cmd/get_processinstance.go` owns process-instance search validation, keyed-vs-search branching, pagination orchestration, and the final handoff to `listProcessInstancesView`, so command-specific output changes belong there before shared render helpers are widened.
- `cmd/cmd_views_get.go` keeps process-instance detail rendering mode-agnostic through `listOrJSON`, with `--with-age` as a narrow JSON/one-line decoration; count-only output should short-circuit before this renderer to avoid creating a new global render mode.
- Shared process-instance page models in `internal/domain/processinstance.go` and `c8volt/process/model.go` now expose optional `ReportedTotal{Count, Kind}` metadata alongside `Request`, `Items`, and `OverflowState`, and `c8volt/process/convert.go` mirrors that seam directly so command code can stay version-agnostic once services populate it.
- Versioned search services already see backend total signals: `v87` trims Operate results with an optional `payload.Total`, while `v88` and `v89` compute overflow from `Page.TotalItems` and `HasMoreTotalItems`; the command layer currently loses that data because services only return `OverflowState`.
- Existing regression anchors are already close to the needed feature seams: `cmd/get_processinstance_test.go` covers command help and search request behavior, `cmd/cmd_processinstance_test.go` provides reusable process-instance search fixtures, `c8volt/process/client_test.go` covers cross-version page conversion, and versioned service tests assert paging metadata behavior around `OverflowState` and capped totals.
- `ReportedTotal=nil` is the shared unavailable signal, while `ReportedTotal.Kind` uses `exact` and `lower_bound`; that keeps absence distinct from a real numeric zero total and avoids inventing a third enum state in the model.
- The command contract should keep `--total` as a visible flag while discovery output modes stay limited to shared render choices (`one-line`, `json`, `keys-only`); the count-only branch remains command-local rather than a new render mode.
- `cmd/get_processinstance.go` remains the single source for help synopsis/examples and visible flag descriptions, and the capabilities document reflects that same flag description verbatim, so help-text regressions should cover both command help and capability metadata.
- `make docs-content` intentionally regenerates `docs/cli/*` and syncs `docs/index.md` from `README.md`, so README changes that affect user-facing docs should expect both files to move together in the same iteration.

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
## Iteration 4 - 2026-04-22 22:34:23 CEST
**User Story**: User Story 2 - Preserve Existing List Behavior
**Tasks Completed**:
- [x] T013: Add command regressions for invalid `--total` combinations and preserved default output behavior
- [x] T014: Add service-level regressions proving reported-total metadata stays consistent without changing non-`--total` page behavior
- [x] T015: Enforce `--total` validation rules for `--key`, `--json`, `--keys-only`, and `--with-age`
- [x] T016: Keep command contract and output-mode metadata coherent for the new flag without introducing a new global render mode
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cmd_processinstance_test.go
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- cmd/get_test.go
- internal/services/processinstance/v87/service_test.go
- internal/services/processinstance/v88/service_test.go
- internal/services/processinstance/v89/service_test.go
- specs/124-process-instances-total/progress.md
- specs/124-process-instances-total/tasks.md
**Learnings**:
- Validation for `--total` belongs in `validatePISearchFlags()` for render-related conflicts, while the `--key` conflict stays on the keyed lookup branch because only that branch knows whether search mode was bypassed.
- Default one-line output should continue to flow through `listProcessInstancesView`; the safest regression is to prove reported-total metadata does not collapse normal output into count-only mode.
- Capability metadata already models `--total` correctly as a flag, so the key regression is preventing it from being treated as a new shared output mode.
---
## Iteration 5 - 2026-04-22 22:38:00 CEST
**User Story**: User Story 3 - Understand the New Flag Quickly
**Tasks Completed**:
- [x] T017: Add or update command-help regressions for the new `--total` flag text
- [x] T018: Update user-facing command documentation and examples for `--total`
- [x] T019: Regenerate CLI reference output with `make docs-content`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- README.md
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- cmd/get_test.go
- docs/cli/c8volt_get_process-instance.md
- docs/index.md
- specs/124-process-instances-total/progress.md
- specs/124-process-instances-total/tasks.md
**Learnings**:
- Help discoverability for `--total` needs to stay aligned across Cobra help text, capabilities metadata, README examples, and generated CLI docs to avoid conflicting automation guidance.
- `make docs-content` is the repository-native regeneration path for this story because it refreshes the command reference and the README-backed docs homepage together.
---
## Iteration 6 - 2026-04-22 22:53:00 CEST
**User Story**: Phase 6: Polish & Cross-Cutting Concerns
**Tasks Completed**:
- [x] T020: Refresh final implementation and verification notes
- [x] T021: Run focused validation with `go test ./c8volt/process -count=1`, `go test ./internal/services/processinstance/... -count=1`, and `go test ./cmd -count=1`
- [x] T022: Run repository validation with `make test`
**Tasks Remaining in Story**: None - story complete
**Commit**: Not yet recorded in Git history for this iteration
**Files Changed**:
- specs/124-process-instances-total/plan.md
- specs/124-process-instances-total/progress.md
- specs/124-process-instances-total/quickstart.md
- specs/124-process-instances-total/tasks.md
**Learnings**:
- The focused feature suites were already clean, and the full repository gate (`make test`) also passed, so the feature is ready by the task-plan definition.
- The remaining open work after Ralph stopped was validation bookkeeping rather than implementation risk.
---
