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
- `cmd/get_processinstance_filtering.go` owns process-instance search filter conversion and local orphan-child filtering through `FilterProcessInstanceWithOrphanParent`.
- `cmd/get_processinstance_paging.go` owns shared process-instance search paging, limits, continuation states, progress output, and automation-aware continuation behavior.
- `cmd/delete_processinstance.go` owns existing process-instance delete dry-run planning, destructive confirmation, and deletion submission through the process facade.
- `c8volt/process/dryrun.go` exposes thin facade methods over `internal/services/processinstance` delete/cancel dry-run planning.
- `internal/services/processinstance` owns version-neutral process-instance service contracts and versioned service implementations.
- Generated CLI docs live under `docs/cli/` and must be refreshed through `make docs-content`.

## Validation Log

- Planning artifacts created on 2026-05-11.
