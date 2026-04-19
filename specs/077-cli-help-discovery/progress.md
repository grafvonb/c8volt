# Ralph Progress Log

Feature: 077-cli-help-discovery
Started: 2026-04-19 13:07:21

## Codebase Patterns

- Keep public CLI discovery language anchored in `cmd/root.go` and `cmd/capabilities.go`; later command-family help should reuse the same `--json` and `automation:full` terminology instead of inventing local phrasing.
- Treat `isDiscoverableCommand` as the single public-versus-hidden boundary for machine-readable coverage and regression tests; hidden completion plumbing and `help` should stay out of both capability docs and summary assertions.
- Shared help assertions can live in `cmd/root_test.go` and be reused across package tests because the repository keeps common CLI execution helpers in package-level `_test.go` files instead of a dedicated helper package.
- In `cmd` tests that reuse the shared Cobra tree, reset the full command tree with `testx.ResetCommandTreeFlags` rather than only root persistent flags before executing `--help`; subcommand help flags stay sticky across in-process runs otherwise.
- Cobra top-level completion descriptions are fed directly from each command's `Short` text, so any public wording refresh for group commands needs matching completion regression updates in `cmd/completion_test.go`.
- The process-instance test helpers must scrub bound globals after `testx.ResetCommandTreeFlags(root)` because Cobra `StringSlice` defaults can repopulate slice-backed globals with phantom `[]` values during in-process runs.

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

## Iteration 3 - 2026-04-19 13:56:00 CEST
**User Story**: Partial progress on US1 - root and family-entry discovery guidance
**Tasks Completed**:
- [x] T006: Add root and parent/group help regression coverage in `cmd/root_test.go` and `cmd/get_test.go`
- [x] T007: Add discovery and configuration help regression coverage in `cmd/capabilities_test.go` and `cmd/config_test.go`
- [x] T008: Refresh root and family-entry help text in `cmd/root.go`, `cmd/get.go`, `cmd/config.go`, `cmd/embed.go`, and `cmd/version.go`
**Tasks Remaining in Story**: 3
**Commit**: No commit - partial progress
**Files Changed**:
- cmd/capabilities_test.go
- cmd/completion_test.go
- cmd/config.go
- cmd/config_test.go
- cmd/embed.go
- cmd/get.go
- cmd/get_test.go
- cmd/root.go
- cmd/root_test.go
- cmd/version.go
- specs/077-cli-help-discovery/progress.md
- specs/077-cli-help-discovery/tasks.md
**Learnings**:
- Top-level shell completion descriptions are another public help surface fed directly by Cobra `Short` text, so family-entry wording changes need matching regression updates in `cmd/completion_test.go`.
- Parent/group commands can carry chooser-oriented examples without promising unsupported automation behavior, which keeps the full-tree coverage requirement compatible with current runtime semantics.
- `go test ./cmd -count=1` is a useful story-slice gate here because help, completion, and command-contract regressions all consume the same command metadata.
---

## Iteration 4 - 2026-04-19 15:05:00 CEST
**User Story**: US1 - Choose The Right Command Path
**Tasks Completed**:
- [x] T009: Refresh discovery, configuration, and cluster-read examples in `cmd/config_show.go`, `cmd/get_cluster.go`, `cmd/get_cluster_license.go`, and `cmd/get_cluster_topology.go`
- [x] T010: Refresh read-oriented retrieval guidance in `cmd/get_processdefinition.go`, `cmd/get_processinstance.go`, `cmd/get_resource.go`, and `cmd/get_variable.go`
- [x] T011: Refresh embedded-resource and version command examples in `cmd/embed_list.go`, `cmd/embed_export.go`, and `cmd/version.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: No commit - `make test` fails on an existing `cmd` sandbox listener panic in `TestDeployProcessDefinitionCommand_TenantFlagOverridesEnvProfileAndConfig`
**Files Changed**:
- cmd/config_show.go
- cmd/config_test.go
- cmd/embed_export.go
- cmd/embed_list.go
- cmd/embed_test.go
- cmd/get_cluster.go
- cmd/get_cluster_license.go
- cmd/get_cluster_topology.go
- cmd/get_processdefinition.go
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- cmd/get_resource.go
- cmd/get_test.go
- cmd/get_variable.go
- cmd/version.go
- cmd/version_test.go
- specs/077-cli-help-discovery/progress.md
- specs/077-cli-help-discovery/tasks.md
**Learnings**:
- Read-oriented command help works better when examples stay command-focused instead of embedding long sample payloads; this keeps generated docs scannable while preserving the shared `--json` guidance.
- `executeRootForProcessInstanceTest` needs a full command-tree flag reset before help assertions, because subcommand `--help` state persists on the shared Cobra tree and can short-circuit later tests.
- Targeted help regressions for the updated commands pass, but repository validation is currently blocked by an unrelated IPv6 `httptest` bind panic in `cmd/deploy_test.go` under this sandbox.
---

## Iteration 5 - 2026-04-19 16:10:00 CEST
**User Story**: US2 - Understand Confirmation And Completion Semantics
**Tasks Completed**:
- [x] T012: Add state-changing help regression coverage in `cmd/run_test.go`, `cmd/deploy_test.go`, `cmd/delete_test.go`, and `cmd/cancel_test.go`
- [x] T013: Add wait-and-verification help regression coverage in `cmd/expect_test.go` and `cmd/walk_test.go`
- [x] T014: Refresh run and deploy command-family semantics in `cmd/run.go`, `cmd/run_processinstance.go`, `cmd/deploy.go`, `cmd/deploy_processdefinition.go`, and `cmd/embed_deploy.go`
- [x] T015: Refresh cancel and delete command-family confirmation guidance in `cmd/cancel.go`, `cmd/cancel_processinstance.go`, `cmd/delete.go`, `cmd/delete_processinstance.go`, and `cmd/delete_processdefinition.go`
- [x] T016: Refresh verification and tree-inspection follow-up guidance in `cmd/expect.go`, `cmd/expect_processinstance.go`, `cmd/walk.go`, and `cmd/walk_processinstance.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cancel.go
- cmd/cancel_processinstance.go
- cmd/cancel_test.go
- cmd/completion_test.go
- cmd/delete.go
- cmd/delete_processdefinition.go
- cmd/delete_processinstance.go
- cmd/delete_test.go
- cmd/deploy.go
- cmd/deploy_processdefinition.go
- cmd/deploy_test.go
- cmd/embed_deploy.go
- cmd/expect.go
- cmd/expect_processinstance.go
- cmd/expect_test.go
- cmd/get_processinstance_test.go
- cmd/run.go
- cmd/run_processinstance.go
- cmd/run_test.go
- cmd/walk.go
- cmd/walk_processinstance.go
- cmd/walk_test.go
- specs/077-cli-help-discovery/progress.md
- specs/077-cli-help-discovery/tasks.md
**Learnings**:
- Waiting and destructive-confirmation help needs to distinguish accepted work from confirmed completion explicitly; otherwise `--no-wait` examples read as stronger guarantees than the runtime actually provides.
- Completion regressions are part of the same surface area as help refreshes because Cobra shells render `Short` text directly in interactive suggestions.
- `go test ./cmd -count=1` only became stable for process-instance helpers after re-scrubbing bound globals post-reset, which prevents slice-backed flags from injecting phantom `[]` keys into search-driven cancel/delete flows.
---
