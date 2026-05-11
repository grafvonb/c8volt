# Progress: Ops Purge Orphan Process Instances

**Issue**: https://github.com/grafvonb/c8volt/issues/186  
**Feature**: `186-ops-orphan-cleanup`  
**Mandatory Implementation Context**: `specs/ralph-implementation-rules.md`

## Ralph Rules

- Every Ralph launch must include `--implementation-context specs/ralph-implementation-rules.md`.
- Every implementation iteration must read and apply `specs/ralph-implementation-rules.md`.
- Each iteration must complete only the current Ralph work unit.
- Do not stage or commit unless the Ralph workflow reaches its commit step and validation passes.
- Commit subjects must follow Conventional Commits and end with `#186`.

## Codebase Patterns

- `cmd/ops.go`, `cmd/ops_execute.go`, and `cmd/ops_repair.go` already define discovery-only grouping commands and shared ops foundation from issue #197.
- This feature should add `cmd/ops_purge.go` for destructive cleanup workflows while preserving `ops execute` for non-purge playbooks such as smoke tests.
- `cmd/ops_contract.go` already defines shared ops workflow step statuses and report-format primitives.
- `cmd/ops_contract_test.go` protects the shared ops step status vocabulary and report-format inference; extend it when report behavior changes.
- `cmd/get_processinstance_filtering.go` owns process-instance search filter conversion and local orphan-child filtering through `FilterProcessInstanceWithOrphanParent`.
- `cmd/get_processinstance_paging.go` owns shared process-instance search paging, limits, continuation states, progress output, and automation-aware continuation behavior.
- `cmd/delete_processinstance.go` owns existing process-instance delete dry-run planning, destructive confirmation, and deletion submission through the process facade.
- `deleteProcessInstancesWithPlanAndRender` can validate/delete a frozen key set while deferring dry-run rendering, which matches the orphan purge plan orchestration need.
- `c8volt/process/dryrun.go` exposes thin facade methods over `internal/services/processinstance` delete/cancel dry-run planning.
- `internal/services/processinstance` owns version-neutral process-instance service contracts and versioned service implementations.
- No `c8volt/ops` or `internal/services/ops` packages exist yet; the foundational work unit must create them and wire `c8volt/client.go` plus `c8volt/contract.go`.
- `cmd/command_contract.go` records mutation, contract, automation, output-mode, and required-flag metadata through Cobra annotations consumed by capabilities tests.
- Generated CLI docs live under `docs/cli/` and must be refreshed through `make docs-content`.

## Validation Log

- Planning artifacts created on 2026-05-11.

---
## Iteration 1 - 2026-05-11 20:40:24 CEST
**User Story**: Phase 1: Setup (Shared Infrastructure)
**Tasks Completed**:
- [x] T001: Record mandatory Ralph context and issue traceability in `specs/186-ops-orphan-cleanup/progress.md`
- [x] T002: Inspect existing ops foundation from issue #197, process-instance search, process-instance delete, command contract, and docs generation patterns; record reusable discoveries in `specs/186-ops-orphan-cleanup/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/186-ops-orphan-cleanup/tasks.md
- specs/186-ops-orphan-cleanup/progress.md
**Learnings**:
- Issue traceability is explicit in `spec.md`, `plan.md`, and `tasks.md`; every commit subject for this feature must end with `#186`.
- The current work unit is tracking/setup only; no production code changes were needed before foundational ops APIs.
- Existing process-instance delete helpers already support dry-run planning, confirmation, and mutation against an immutable key set, so later orphan purge work should reuse those semantics instead of duplicating delete mechanics.
- Validation passed with `git diff --check` and `go test ./cmd -run 'TestOps|TestCommandContract' -count=1`.
---
