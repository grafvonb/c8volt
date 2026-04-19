# Ralph Progress Log

Feature: 077-cli-help-discovery
Started: 2026-04-19 13:07:21

## Codebase Patterns

- Keep public CLI discovery language anchored in `cmd/root.go` and `cmd/capabilities.go`; later command-family help should reuse the same `--json` and `automation:full` terminology instead of inventing local phrasing.
- Treat `isDiscoverableCommand` as the single public-versus-hidden boundary for machine-readable coverage and regression tests; hidden completion plumbing and `help` should stay out of both capability docs and summary assertions.
- Shared help assertions can live in `cmd/root_test.go` and be reused across package tests because the repository keeps common CLI execution helpers in package-level `_test.go` files instead of a dedicated helper package.

---

## Iteration 1 - 2026-04-19 13:30:00
**User Story**: Setup - command inventory and validation alignment
**Tasks Completed**:
- [x] T001: Capture the full user-visible command coverage inventory and batching notes in research.md
- [x] T002: Align the implementation and validation checklist for the public command tree in quickstart.md
**Tasks Remaining in Story**: None - story complete
**Commit**: No commit - repository validation is blocked in this sandbox by `httptest` listener bind failures on `[::1]`
**Files Changed**:
- specs/077-cli-help-discovery/progress.md
- specs/077-cli-help-discovery/quickstart.md
- specs/077-cli-help-discovery/research.md
- specs/077-cli-help-discovery/tasks.md
**Learnings**:
- The feature should reuse the live `capabilities --json` traversal as the public-command inventory source because it already applies the discoverable-command boundary.
- The current public scope is root plus 28 discoverable command paths: 10 group nodes and 18 executable leaves.
- Validation needs two layers: focused `go test ./cmd -count=1` during metadata edits and `make test` before a commit, with docs regeneration only when the corresponding sources change.
- Full `make test` currently fails in this environment before reaching code-specific assertions because existing tests in `cmd`, `internal/services/auth/cookie`, and `internal/services/cluster/v87`/`v88` cannot bind local `httptest` listeners.
---

## Iteration 2 - 2026-04-19 13:16:12 CEST
**User Story**: Foundational - shared discovery language and regression seams
**Tasks Completed**:
- [x] T003: Refresh the shared top-level discovery and automation guidance baseline in `cmd/root.go` and `cmd/capabilities.go`
- [x] T004: Strengthen public-versus-hidden command coverage assertions in `cmd/command_contract_test.go`
- [x] T005: Add shared help-output regression helpers for public command metadata in `cmd/root_test.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/capabilities.go
- cmd/capabilities_test.go
- cmd/command_contract_test.go
- cmd/root.go
- cmd/root_test.go
- specs/077-cli-help-discovery/progress.md
- specs/077-cli-help-discovery/tasks.md
**Learnings**:
- The root help baseline should explicitly route humans to `c8volt <group> --help` and machine callers to `c8volt capabilities --json`; later command help should build on that split instead of restating it differently per file.
- Public capability summaries are easier to review when they say hidden/internal commands are excluded up front, which also gives the tests a stable negative assertion surface.
- Repository-wide validation now passes in this environment, so future completed work units can include the required commit after `make test`.
---
