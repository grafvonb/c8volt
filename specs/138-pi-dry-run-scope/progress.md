# Ralph Progress Log

Feature: 138-pi-dry-run-scope
Started: 2026-04-25 22:36:37

## Codebase Patterns

- Dry-run command helpers wrap facade planning failures as `<operation> validation` errors and return before rendering, prompting, or mutating when the expansion is unresolved.
- Human dry-run output always lists missing ancestor keys directly; the verbose/count behavior only applies to real preflight warnings rendered before mutation.
- `processPISearchPagesWithAction` carries both mutation reports and dry-run previews; search-mode dry-run callers suppress per-page rendering and render one aggregate summary after pagination completes.
- Dry-run search paging should not prompt for continuation; it still applies `limitPIPageItems` before dependency planning so `--batch-size` and `--limit` constrain the selected keys before preview aggregation.
- `cancelProcessInstancesWithPlan` and `deleteProcessInstancesWithPlan` already call `DryRunCancelOrDeletePlan` before confirmation and mutation; dry-run behavior should branch after this shared plan is computed.
- Process-instance preflight warnings are rendered through `printDryRunExpansionWarning`, with verbose mode controlling whether missing ancestor keys are listed or counted.
- Command tests use `stubProcessAPI` for focused helper coverage and `require` assertions from `testify`; unexpected process API calls panic unless a test installs a handler.
- Structured command results use the shared result envelope in JSON mode; dry-run previews should render a succeeded envelope because no mutation or waiter state is submitted.
- `process.MissingAncestor` has exported Go field names without JSON tags, so command dry-run payloads use a local DTO to expose `key` and `startKey`.
- Keyed dry-run support branches in the shared cancel/delete helper after `DryRunCancelOrDeletePlan` mapping and before warning, confirmation, mutation, or wait-adjacent report rendering.
- Process-instance command tests must reset `flagDryRun` in `resetProcessInstanceCommandGlobals` because cancel and delete share the package-level command flag.

---

---
## Iteration 1 - 2026-04-25 22:39:16 CEST
**User Story**: Phase 1: Setup (Shared Infrastructure)
**Tasks Completed**:
- [x] T001: Review the dry-run scope contract against current process and command helpers
- [x] T002: Extend dry-run mutation guard support in cmd/process_api_stub_test.go
- [x] T003: Add shared cancel dry-run preview fixtures/assertions in cmd/cancel_test.go
- [x] T004: Add shared delete dry-run preview fixtures/assertions in cmd/delete_test.go
**Tasks Remaining in Story**: None - story complete
**Commit**: No commit - sandbox blocked writes to `.git/index.lock`
**Files Changed**:
- cmd/process_api_stub_test.go
- cmd/cancel_test.go
- cmd/delete_test.go
- specs/138-pi-dry-run-scope/tasks.md
- specs/138-pi-dry-run-scope/progress.md
**Learnings**:
- The feature contract maps cleanly onto existing facade data: requested keys come from command selection, roots and affected family keys come from `DryRunPIKeyExpansion`, warnings and missing ancestors are already structured.
- The setup helpers intentionally assert JSON-decoded payload maps so later tasks can define concrete dry-run view structs without coupling Phase 1 to their final Go type names.
---

---
## Iteration 2 - 2026-04-25 22:43:44 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T005: Define the shared process-instance dry-run preview payload and aggregate payload
- [x] T006: Implement human-readable dry-run rendering
- [x] T007: Implement structured dry-run rendering support
- [x] T008: Refactor cancel preflight to compute a shared plan result
- [x] T009: Refactor delete preflight to compute a shared plan result
- [x] T010: Add focused cancel dry-run preview payload mapping coverage
- [x] T011: Add focused delete dry-run preview payload mapping coverage
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cmd_views_processinstance_dryrun.go
- cmd/cancel_processinstance.go
- cmd/delete_processinstance.go
- cmd/get_processinstance.go
- cmd/cancel_test.go
- cmd/delete_test.go
- specs/138-pi-dry-run-scope/tasks.md
- specs/138-pi-dry-run-scope/progress.md
**Learnings**:
- The cancel/delete helpers can share one plan-to-preview mapper while keeping existing confirmation and mutation behavior unchanged.
- Search aggregation can build on `processInstancePageActionResult.DryRunPreview` in a later story without changing the current report collection contract.
- Staging failed because the environment cannot write inside `.git`; a later environment with Git metadata write access must create the work-unit commit.
---

---
## Iteration 3 - 2026-04-25 22:50:14 CEST
**User Story**: User Story 1 - Preview Keyed Destructive Scope
**Tasks Completed**:
- [x] T012: Add keyed cancel dry-run test for child-to-root escalation
- [x] T013: Add keyed cancel dry-run test for full-family scope and zero mutation calls
- [x] T014: Add keyed delete dry-run test for child-to-root escalation
- [x] T015: Add keyed delete dry-run test for full-family scope and zero mutation calls
- [x] T016: Register `--dry-run` on cancel process-instance
- [x] T017: Render and return keyed cancel dry-run previews before confirmation or mutation
- [x] T018: Register `--dry-run` on delete process-instance
- [x] T019: Render and return keyed delete dry-run previews before confirmation, force-cancel, mutation, or wait behavior
- [x] T020: Run focused keyed dry-run validation
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cancel_processinstance.go
- cmd/delete_processinstance.go
- cmd/cancel_test.go
- cmd/delete_test.go
- cmd/get_processinstance_test.go
- specs/138-pi-dry-run-scope/tasks.md
- specs/138-pi-dry-run-scope/progress.md
**Learnings**:
- Keyed dry-run can reuse the shared preview payload directly and return an empty report list so the caller can exit without rendering mutation reports.
- Focused mutation guards are enough to prove the shared helper does not call cancel/delete APIs when `flagDryRun` is set.
- Validation passed with `go test ./cmd -run 'Test(Cancel|Delete).*DryRun' -count=1` and `go test ./cmd -run 'Test(Cancel|Delete)ProcessInstance' -count=1`.
---

---
## Iteration 4 - 2026-04-26 06:04:43 CEST
**User Story**: User Story 2 - Preview Search-Based and Paged Scope
**Tasks Completed**:
- [x] T021: Add search-based cancel dry-run test across multiple pages with aggregate structured output and nested per-page previews
- [x] T022: Add search-based delete dry-run test across multiple pages with aggregate structured output and nested per-page previews
- [x] T023: Add search dry-run test covering `--batch-size` and `--limit` page selection behavior for cancel
- [x] T024: Add search dry-run test covering `--batch-size` and `--limit` page selection behavior for delete
- [x] T025: Extend cancel search dry run so each selected page contributes dry-run scope without mutation
- [x] T026: Extend delete search dry run so each selected page contributes dry-run scope without mutation
- [x] T027: Preserve search progress and limit-reached behavior for dry-run pages
- [x] T028: Implement structured search dry-run output as an aggregate summary with nested per-page previews
- [x] T029: Run focused search/paged dry-run validation
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cancel_processinstance.go
- cmd/delete_processinstance.go
- cmd/get_processinstance.go
- cmd/process_api_stub_test.go
- cmd/cancel_test.go
- cmd/delete_test.go
- specs/138-pi-dry-run-scope/tasks.md
- specs/138-pi-dry-run-scope/progress.md
**Learnings**:
- Search-mode dry run needs to collect page previews separately from mutation reports so JSON mode emits one valid shared result envelope.
- The page orchestration already performs limit truncation before invoking the page action, so dry-run page tests can assert the planned keys directly.
- Validation passed with `GOCACHE=/tmp/c8volt-go-build go test ./cmd -run 'Test(Cancel|Delete).*DryRun.*Search|Test.*ProcessInstance.*DryRun.*Paged' -count=1` and `GOCACHE=/tmp/c8volt-go-build go test ./cmd -run 'Test(Cancel|Delete).*DryRun|Test(Cancel|Delete)ProcessInstance' -count=1`.
---

---
## Iteration 5 - 2026-04-26 06:08:00 CEST
**User Story**: User Story 3 - Preserve Orphan-Parent Warning Behavior
**Tasks Completed**:
- [x] T030: Add cancel dry-run partial orphan-parent test with warning and missing ancestor keys
- [x] T031: Add delete dry-run partial orphan-parent test with warning and missing ancestor keys
- [x] T032: Add unresolved orphan dry-run failure test for cancel
- [x] T033: Add unresolved orphan dry-run failure test for delete
- [x] T034: Confirm facade partial and unresolved dry-run coverage remains aligned
- [x] T035: Ensure dry-run human output includes partial scope warning and missing ancestor keys
- [x] T036: Ensure dry-run structured output includes traversalOutcome, scopeComplete, warning, and missingAncestors
- [x] T037: Ensure cancel dry run returns unresolved expansion failures without mutation
- [x] T038: Ensure delete dry run returns unresolved expansion failures without mutation
- [x] T039: Run focused orphan-parent dry-run validation
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cancel_test.go
- cmd/delete_test.go
- specs/138-pi-dry-run-scope/tasks.md
- specs/138-pi-dry-run-scope/progress.md
**Learnings**:
- Partial orphan dry runs can be verified through the command helper by asserting the rendered warning, direct missing ancestor keys, partial traversal outcome, and `mutationSubmitted=false` preview state.
- Unresolved orphan dry runs are represented as facade planning errors; the command helper returns the wrapped validation error before any confirmation prompt, render, or cancel/delete mutation.
- Existing facade tests in `c8volt/process/client_test.go` already cover both partial structured expansion and unresolved no-actionable-scope failure.
- Validation passed with `GOCACHE=/tmp/c8volt-go-build go test ./cmd ./c8volt/process -run 'DryRun.*Orphan|DryRun.*Partial|DryRunCancelOrDelete' -count=1` and `GOCACHE=/tmp/c8volt-go-build go test ./cmd -run 'Test(Cancel|Delete).*DryRun' -count=1`.
---
