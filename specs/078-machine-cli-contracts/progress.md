# Ralph Progress Log

Feature: 078-machine-cli-contracts
Started: 2026-04-17 18:14:53

## Codebase Patterns

- Root-level machine metadata should be derived from the existing Cobra command tree and persistent flag definitions in `cmd/root.go`, not from generated docs or a parallel registry.
- Shared machine-readable rendering should extend the existing `pickMode` / `itemView` / `listOrJSON` seams in `cmd/` and keep current public payload models intact.
- Generated docs anchors for CLI behavior come from Cobra metadata and `README.md`; feature research should record doc targets, but implementation should update source metadata first and regenerate output afterward.

---

## Iteration 1 - 2026-04-17 18:23:00
**User Story**: Phase 1 Setup
**Tasks Completed**:
- [x] T001: Inventory the current command tree, root flags, and machine-facing render seams
- [x] T002: Inventory representative command-family payload and outcome seams
- [x] T003: Confirm current automation-facing docs and generated CLI reference anchors
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/078-machine-cli-contracts/progress.md
- specs/078-machine-cli-contracts/research.md
- specs/078-machine-cli-contracts/tasks.md
**Learnings**:
- The feature can build discovery metadata directly from Cobra command registrations, aliases, and real flag definitions already present under `cmd/`.
- The current machine-facing seam is fragmented between read-family JSON render helpers and state-changing commands that rely on exit codes plus logs, so a shared envelope belongs in `cmd/` rather than in generic serialization helpers.
- README plus generated `docs/index.md` and `docs/cli/` already provide the documentation path needed for a future `capabilities` command without adding new doc infrastructure.
---
