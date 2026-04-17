# Ralph Progress Log

Feature: 078-machine-cli-contracts
Started: 2026-04-17 18:14:53

## Codebase Patterns

- Root-level machine metadata should be derived from the existing Cobra command tree and persistent flag definitions in `cmd/root.go`, not from generated docs or a parallel registry.
- Shared machine-readable rendering should extend the existing `pickMode` / `itemView` / `listOrJSON` seams in `cmd/` and keep current public payload models intact.
- Generated docs anchors for CLI behavior come from Cobra metadata and `README.md`; feature research should record doc targets, but implementation should update source metadata first and regenerate output afterward.
- Shared discovery metadata can live on Cobra command annotations and be computed from the live command tree, which keeps command paths, inherited flags, and contract support in one repository-native source of truth.
- Foundational machine-result helpers should stay in `cmd/` while `c8volt/ferrors` only exposes the bounded failure-to-outcome mapping needed to preserve exit-code authority.

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

## Iteration 2 - 2026-04-17 18:22:37 CEST
**User Story**: Phase 2 Foundational
**Tasks Completed**:
- [x] T004: Define the shared capability and result-envelope types
- [x] T005: Implement shared command metadata and contract-support helpers
- [x] T006: Implement shared result-envelope rendering helpers and outcome-mapping utilities
- [x] T007: Add foundational regression coverage for capability metadata, outcome mapping, and exit-code alignment
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/command_contract.go
- cmd/cmd_views_contract.go
- cmd/command_contract_test.go
- cmd/capabilities_test.go
- cmd/root.go
- c8volt/ferrors/errors.go
- c8volt/ferrors/errors_test.go
- specs/078-machine-cli-contracts/tasks.md
- specs/078-machine-cli-contracts/progress.md
**Learnings**:
- Cobra required-flag annotations are sufficient to expose machine-readable required-flag metadata without introducing a second flag registry.
- Read-only versus state-changing discovery defaults can be inferred safely from the existing top-level command taxonomy, with command annotations reserved for future exceptions and explicit rollouts.
- The foundational envelope helpers can be validated before any command-family rollout by testing document generation and failure outcome mapping in isolation.
---
