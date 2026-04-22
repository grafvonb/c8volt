# Ralph Progress Log

Feature: 129-orphan-parent-warning
Started: 2026-04-22 11:13:42

## Codebase Patterns

- Shared traversal behavior is centralized in `internal/services/processinstance/walker/walker.go`; `walk`, dry-run expansion, and versioned services consume that seam rather than reimplementing ancestry/family logic.
- Direct single-resource semantics remain intentionally separate: `cmd/get_processinstance.go`, `internal/services/common/response.go`, and `internal/services/processinstance/waiter/waiter.go` preserve strict not-found and absent/deleted waiter behavior.
- Supported Camunda versions are selected in `internal/services/processinstance/factory.go`, and `v87`, `v88`, and `v89` all delegate traversal methods through the shared walker while keeping version-specific direct lookup behavior local to each service.
- Shared process-instance traversal contracts need a leaf package when both the parent `processinstance` package and versioned services consume them; `internal/services/processinstance/traversal` keeps result types and builders reusable without import cycles.
- Backward-compatible facade upgrades follow the existing pattern of adding structured result methods beside legacy tuple methods first, then switching command callers story by story once the shared contract is validated.
- Story-scoped behavior changes that would otherwise leak into later destructive flows should stay behind version- or caller-specific adapters first; `v87` now uses a traversal-only search adapter for walk results while dry-run expansion remains on the legacy path until the preflight story is implemented.
- Destructive preflight flows should consume `DryRunCancelOrDeletePlan` and print orphan warnings on stderr or logs, which keeps JSON stdout/report payloads stable while still surfacing missing ancestor keys to operators.
- Strict single-resource seams should document their contract where they live: `cmd/get_processinstance.go`, `internal/services/common/response.go`, and `internal/services/processinstance/waiter/waiter.go` are the canonical places to state that not-found stays strict outside traversal/preflight flows.
- CLI markdown under `docs/cli/` is generated from the current Cobra help text via `make docs-content`, so README/help wording changes should be regenerated in the same polish pass instead of hand-editing derived docs.

---

## Iteration 1 - 2026-04-22 12:03 CEST
**User Story**: Setup
**Tasks Completed**:
- [x] T001: Inventory the current orphan-parent failure path across walker, dry-run, and walk/cancel/delete command flows
- [x] T002: Confirm current strict lookup and waiter boundaries in get-process-instance, waiter, and common response helpers
- [x] T003: Confirm shared version support and traversal delegation across the process-instance factory and `v87`/`v88`/`v89` services
**Tasks Remaining in Story**: None - story complete
**Commit**: No commit - sandbox blocked git index writes
**Files Changed**:
- specs/129-orphan-parent-warning/tasks.md
- specs/129-orphan-parent-warning/progress.md
**Learnings**:
- `walker.Ancestry` is the current orphan-parent failure source; `walk` commands and `DryRunCancelOrDeleteGetPIKeys` both fail immediately on that returned error today.
- `waiter.WaitForProcessInstanceState` already treats absence as success only for desired absent/deleted-style waits, which is the strict seam the feature must not loosen.
- The version matrix is already aligned for shared traversal changes because all supported process-instance services delegate ancestry/descendants/family calls to the same walker helpers.
---

---
## Iteration 2 - 2026-04-22 11:27 CEST
**User Story**: Foundational
**Tasks Completed**:
- [x] T004: Define the authoritative orphan-parent warning, success, and failure contract
- [x] T005: Refactor the shared traversal API shape to represent partial results, missing ancestors, and unresolved outcomes
- [x] T006: Update the feature data model and quickstart guidance for the finalized traversal result contract
- [x] T007: Add foundational facade and helper seams for structured partial traversal handling
**Tasks Remaining in Story**: None - story complete
**Commit**: No commit - sandbox blocked git index writes
**Files Changed**:
- c8volt/process/api.go
- c8volt/process/client.go
- c8volt/process/client_test.go
- c8volt/process/dryrun.go
- c8volt/process/walker.go
- internal/services/common/response.go
- internal/services/processinstance/api.go
- internal/services/processinstance/traversal/result.go
- internal/services/processinstance/v87/contract.go
- internal/services/processinstance/v87/service.go
- internal/services/processinstance/v88/contract.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v89/contract.go
- internal/services/processinstance/v89/service.go
- internal/services/processinstance/walker/walker.go
- internal/services/processinstance/walker/walker_test.go
- specs/129-orphan-parent-warning/contracts/orphan-parent-traversal.md
- specs/129-orphan-parent-warning/data-model.md
- specs/129-orphan-parent-warning/plan.md
- specs/129-orphan-parent-warning/progress.md
- specs/129-orphan-parent-warning/quickstart.md
- specs/129-orphan-parent-warning/research.md
- specs/129-orphan-parent-warning/tasks.md
**Learnings**:
- The tuple-based ancestry API already carries enough orphan boundary information to seed a structured result seam once the walker preserves the partial path on orphan errors.
- A leaf `traversal` package is the repository-native way to share result contracts between `processinstance` and `v87`/`v88`/`v89` without reintroducing package cycles.
- `DryRunCancelOrDeleteGetPIKeys` can stay source-compatible while the new `DryRunCancelOrDeletePlan` carries missing-ancestor metadata for later command adoption.
---

---
## Iteration 3 - 2026-04-22 11:39 CEST
**User Story**: US1 - Inspect Partial Trees Safely
**Tasks Completed**:
- [x] T008: Add shared walker regression tests for partial ancestry, partial family traversal, and fully unresolved failure behavior
- [x] T009: Add version-aware traversal regression coverage for `v87`, `v88`, and `v89` services
- [x] T010: Add command rendering regressions for partial parent/family/tree output
- [x] T011: Implement shared partial ancestry and family result handling
- [x] T012: Thread the new traversal result contract through the process facade
- [x] T013: Render partial walk output and warnings for parent/family/tree modes
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/dryrun.go
- cmd/cmd_views_walk.go
- cmd/walk_processinstance.go
- cmd/walk_test.go
- internal/services/processinstance/v87/service.go
- internal/services/processinstance/v87/service_test.go
- internal/services/processinstance/v88/service_test.go
- internal/services/processinstance/v89/service_test.go
- internal/services/processinstance/walker/walker_test.go
- specs/129-orphan-parent-warning/tasks.md
- specs/129-orphan-parent-warning/progress.md
**Learnings**:
- `walk process-instance` can adopt the structured traversal contract without changing the default human-readable list/tree views by wrapping the existing renderers and appending warnings only when the result is partial.
- Camunda `8.7` needs a traversal-only adapter that resolves process instances through tenant-safe search for result-based walk flows, while dry-run preflight must stay on the legacy path until US2 lands.
- The broader command suite is the right guardrail for story-scoped traversal work because `cancel` and `delete` still share process-instance expansion seams that must not change early.
---

---
## Iteration 4 - 2026-04-22 11:49 CEST
**User Story**: US2 - Keep Orphan Children Actionable
**Tasks Completed**:
- [x] T014: Add dry-run expansion regressions for resolved roots, collected keys, and missing ancestors
- [x] T015: Add command preflight regressions for keyed and paged cancel/delete orphan-child scenarios
- [x] T016: Add indirect cleanup regression coverage for process-resource expansion paths
- [x] T017: Update dependency-expansion dry-run behavior to return actionable roots/collected keys plus missing ancestors
- [x] T018: Consume the shared preflight warning contract in cancel/delete keyed and paged flows
- [x] T019: Align indirect process-definition cleanup with the same preflight contract
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/client_test.go
- c8volt/process/dryrun.go
- c8volt/resource/client.go
- c8volt/resource/client_test.go
- cmd/cancel_processinstance.go
- cmd/cancel_test.go
- cmd/cmd_views_processinstance.go
- cmd/delete_processinstance.go
- cmd/delete_test.go
- cmd/process_api_stub_test.go
- specs/129-orphan-parent-warning/tasks.md
- specs/129-orphan-parent-warning/progress.md
**Learnings**:
- `DryRunCancelOrDeletePlan` is the right mutation-preflight seam for US2 because it can now reject fully unresolved expansions while still carrying partial roots, collected keys, and missing ancestor metadata for callers that remain actionable.
- Cancel/delete command warnings belong on stderr via Cobra’s error writer so preflight warnings stay visible in interactive use without corrupting JSON report payloads on stdout.
- Process-definition cleanup should consume the same structured plan as direct cancel/delete flows and act on resolved roots only, while logging the partial boundary when missing ancestors remain.
---

---
## Iteration 5 - 2026-04-22 11:57 CEST
**User Story**: US3 - Preserve Strict Single-Resource Semantics
**Tasks Completed**:
- [x] T020: Add strict direct-lookup non-regression coverage
- [x] T021: Add waiter non-regression coverage for absent/deleted semantics
- [x] T022: Add command-level non-regression coverage for strict lookup vs traversal behavior
- [x] T023: Keep direct lookup/state-check and waiter boundaries isolated from traversal changes
- [x] T024: Refresh operator-facing wording while preserving strict single-resource guidance
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- README.md
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- cmd/get_test.go
- cmd/root.go
- cmd/walk_processinstance.go
- cmd/walk_test.go
- internal/services/common/response.go
- internal/services/processinstance/v87/service_test.go
- internal/services/processinstance/v88/service_test.go
- internal/services/processinstance/v89/service_test.go
- internal/services/processinstance/waiter/waiter.go
- internal/services/processinstance/waiter/waiter_test.go
- specs/129-orphan-parent-warning/progress.md
- specs/129-orphan-parent-warning/tasks.md
**Learnings**:
- Direct `get process-instance --key` coverage is best anchored in subprocess-based command tests plus service-level not-found assertions, because the command error path exits the process rather than returning a plain Go error in-process.
- The waiter’s absent/deleted contract is still intentionally narrower than traversal behavior: `ErrNotFound` only becomes `ABSENT` when the desired state set explicitly allows it.
- The broader `go test ./cmd -count=1` suite is currently blocked in this sandbox by an unrelated IPv6 `httptest` listener panic, so US3 validation relied on targeted `cmd` tests around `get` and `walk` plus the full `internal/services/processinstance/...` suite.
---

---
## Iteration 6 - 2026-04-22 12:01 CEST
**User Story**: Final polish
**Tasks Completed**:
- [x] T025: Refresh final implementation and verification notes
- [x] T026: Run focused validation with `go test ./internal/services/processinstance/... -count=1`, `go test ./c8volt/process -count=1`, and `go test ./cmd -count=1`
- [x] T027: Run repository validation with `make test`
- [x] T028: Regenerate affected CLI docs with `make docs-content`
**Tasks Remaining in Story**: None - story complete
**Commit**: No commit - pre-existing staged and unstaged feature changes would be swept into a Phase 6-only commit
**Files Changed**:
- docs/cli/c8volt.md
- docs/cli/c8volt_get_process-instance.md
- docs/cli/c8volt_walk_process-instance.md
- docs/index.md
- specs/129-orphan-parent-warning/plan.md
- specs/129-orphan-parent-warning/progress.md
- specs/129-orphan-parent-warning/quickstart.md
- specs/129-orphan-parent-warning/tasks.md
**Learnings**:
- The full repository gate now passes in this environment; the earlier `go test ./cmd -count=1` IPv6 listener failure did not reproduce on 2026-04-22 during the final polish run.
- Phase 6 is documentation- and validation-heavy, but the commit step still needs an isolated index because the worktree currently includes pre-existing staged and unstaged feature edits outside the polish files.
---
