# Ralph Progress Log

Feature: 140-pi-limit-batch-size
Started: 2026-04-25 15:48:48

## Codebase Patterns

- Help-output omission checks can use `assertCommandHelpOutput(..., omits)` for command help tests, while get process-instance help tests currently use `executeRootForProcessInstanceTest` with direct `require.NotContains` assertions.
- Process-instance search request page size is asserted through the captured request page `limit` field; combined `--batch-size --limit` tests should use a batch size larger than the limit to prove per-page size and total cap remain independent.
- Shared process-instance search paging lives in `cmd/get_processinstance.go`; `flagGetPISize`, `resolvePISearchSize`, `newPISearchPageRequest`, `searchProcessInstancesWithPaging`, and `processPISearchPagesWithAction` are the main seams for batch-size and total-limit behavior.
- Process-instance flag validation should accept the current `*cobra.Command` when it needs `Flags().Changed(...)`; referencing package-level command variables from shared validation creates Go initialization cycles.
- Removed process-instance flags should use `useInvalidInputFlagErrors` on the affected leaf commands so Cobra parse failures map to the repository invalid-input exit model.
- `cancel process-instance` and `delete process-instance` register their own command flags but reuse get-side process-instance search globals and paging helpers for search-mode destructive workflows.
- Multi-page process-instance tests use `newProcessInstanceSearchCaptureServerWithResponses`, `decodeCapturedPISearchPages`, `decodeCapturedTopLevelPISearchPages`, and `safeSlice` helpers to assert request paging, continuation prompts, and destructive side effects.
- Command test helpers reset package-level Cobra flag globals with `resetProcessInstanceCommandGlobals`; future flag additions need reset coverage to avoid cross-test leakage.
- Generated CLI docs under `docs/cli/` currently mirror Cobra command metadata and should be regenerated from command source after help/examples change, not hand-edited as the source of truth.
- Total `--limit` enforcement belongs after local process-instance filters and before rendering or destructive page actions; progress summaries should count the limited page subset, not the raw backend page.
- The limit-reached stop condition is represented as its own process-instance continuation state so paging can stop without prompting while verbose progress remains distinct from no-more-matches and warning stops.

---
## Iteration 1 - 2026-04-25 16:57:33 CEST
**User Story**: Phase 1 Setup (Shared Infrastructure)
**Tasks Completed**:
- [x] T001: Review existing process-instance paging helpers and identify shared limit insertion points
- [x] T002: Review command test helpers for multi-page process-instance fixtures
- [x] T003: Review affected docs references for `--count`, `--batch-size`, and `--limit`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/140-pi-limit-batch-size/tasks.md
- specs/140-pi-limit-batch-size/progress.md
**Learnings**:
- The focused validation command `go test ./cmd -run 'Test(GetProcessInstancePagingFlow|CancelProcessInstanceCommand_SearchPagingPromptFlow|DeleteProcessInstanceCommand_SearchPagingPromptFlow)$' -count=1` passed.
- Current user-facing docs and generated command references still use `--count`; later documentation work should wait until Cobra help text changes are in place.
- Limit insertion should happen after existing local result filters and before rendering or cancel/delete page actions so total limiting matches the feature contract.
---

---
## Iteration 2 - 2026-04-25 17:16:01 CEST
**User Story**: Phase 2 Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T004: Replace affected `--count` registrations with `--batch-size` while preserving `-n`
- [x] T005: Add shared `--limit` / `-l` flag storage and registration for affected process-instance commands
- [x] T006: Add validation for positive `--limit`, `--limit` with `--key`, `--limit` with `--total`, and updated `--batch-size` flag checks
- [x] T007: Add command tests for removed `--count`, invalid `--limit`, and `--limit` with `--key` across get/cancel/delete
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cancel.go
- cmd/cancel_processinstance.go
- cmd/cancel_test.go
- cmd/delete.go
- cmd/delete_processinstance.go
- cmd/delete_test.go
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- specs/140-pi-limit-batch-size/tasks.md
- specs/140-pi-limit-batch-size/progress.md
**Learnings**:
- `go test ./cmd -run 'Test(GetProcessInstanceCommand_RejectsInvalidLimitAndRemovedCountFlags|CancelProcessInstanceCommand_RejectsInvalidLimitAndRemovedCountFlags|DeleteProcessInstanceCommand_RejectsInvalidLimitAndRemovedCountFlags|ResolvePISearchSize|GetProcessInstanceSearchScaffold_UsesTempConfigAndCapturesSearchRequest|GetProcessInstancePagingFlow|CancelProcessInstanceCommand_SearchPagingPromptFlow|DeleteProcessInstanceCommand_SearchPagingPromptFlow)$' -count=1` passed after adding command-local flag parse normalization.
- `go test ./cmd -count=1` passed after updating parent cancel/delete help examples from `--count` to `--batch-size`.
- Cobra unknown-flag parse errors defaulted to exit code 1 until the affected leaf commands installed an invalid-input flag error function.
---

---
## Iteration 3 - 2026-04-25 17:26:42 CEST
**User Story**: User Story 1 - Limit Search Results Across Pages
**Tasks Completed**:
- [x] T008: Add `get process-instance` multi-page `--limit` tests
- [x] T009: Add search-based `cancel process-instance` multi-page `--limit` tests
- [x] T010: Add search-based `delete process-instance` multi-page `--limit` tests
- [x] T011: Add remaining-limit calculation and page truncation helpers
- [x] T012: Apply limited page results to `get process-instance` aggregation and incremental rendering
- [x] T013: Apply limited page keys to search-based cancel/delete page actions
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- cmd/cancel_test.go
- cmd/delete_test.go
- specs/140-pi-limit-batch-size/tasks.md
- specs/140-pi-limit-batch-size/progress.md
**Learnings**:
- `go test ./cmd -run 'Test(GetProcessInstancePagingFlow|CancelProcessInstanceCommand_SearchPagingLimitFlow|DeleteProcessInstanceCommand_SearchPagingLimitFlow)$' -count=1 -v` passed with cross-page limit coverage for get/cancel/delete.
- `go test ./cmd -count=1` passed after applying the shared limit-reached continuation state.
- Destructive search tests still observe one confirmation call under `--auto-confirm` because the first-page destructive confirmation function is invoked with implicit confirmation; continuation prompting remains skipped when the limit is reached.
---

---
## Iteration 4 - 2026-04-25 17:35:44 CEST
**User Story**: User Story 2 - Distinguish Batch Size From Total Limit
**Tasks Completed**:
- [x] T014: Add `get process-instance` tests for `--batch-size`, `-n`, and combined `--batch-size --limit`
- [x] T015: Add search-based cancel/delete tests for combined `--batch-size --limit`
- [x] T016: Update page-size resolution to use `--batch-size` flag change detection and preserve shared config/default behavior
- [x] T017: Update examples and worker help text to use batch-size terminology
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- cmd/cancel_processinstance.go
- cmd/cancel_test.go
- cmd/delete_processinstance.go
- cmd/delete_test.go
- specs/140-pi-limit-batch-size/tasks.md
- specs/140-pi-limit-batch-size/progress.md
**Learnings**:
- `go test ./cmd -run 'Test(GetProcessInstancePagingFlow|CancelProcessInstanceCommand_SearchPagingBatchSizeLimitFlow|DeleteProcessInstanceCommand_SearchPagingBatchSizeLimitFlow|ResolvePISearchSize)$' -count=1` passed.
- `go test ./cmd -count=1` passed.
- The existing page-size resolver already keys off `Flags().Changed("batch-size")`; this iteration added focused behavior coverage and command examples showing `--batch-size` and `--limit` together.
---

---
## Iteration 5 - 2026-04-25 17:39:53 CEST
**User Story**: User Story 3 - Reject Ambiguous or Invalid Flag Combinations
**Tasks Completed**:
- [x] T018: Add explicit help/parse regression coverage proving `--count` is absent or rejected on affected command paths
- [x] T019: Add regression coverage confirming `--total` with `--limit` is rejected as mutually exclusive
- [x] T020: Ensure `--limit` validation is search-mode-only and rejects `--total` combinations
- [x] T021: Align standard invalid-argument handling for removed and invalid flags without adding `--count` aliases
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_processinstance_test.go
- cmd/cancel_test.go
- cmd/delete_test.go
- specs/140-pi-limit-batch-size/tasks.md
- specs/140-pi-limit-batch-size/progress.md
**Learnings**:
- `go test ./cmd -run 'Test(GetProcessInstanceHelp_DocumentsPagingAndAutomationSurface|CancelHelp_DocumentsConfirmationAndNoWaitSemantics|DeleteHelp_DocumentsDestructiveConfirmationPaths|GetProcessInstanceCommand_RejectsInvalidLimitAndRemovedCountFlags|CancelProcessInstanceCommand_RejectsInvalidLimitAndRemovedCountFlags|DeleteProcessInstanceCommand_RejectsInvalidLimitAndRemovedCountFlags)$' -count=1` passed.
- `go test ./cmd -count=1` passed.
- Existing validation in `validatePISearchFlags` already rejects `--total --limit`, while keyed-mode validation rejects `--key --limit` before command execution.
- The affected process-instance leaf commands already use `useInvalidInputFlagErrors`, so removed `--count` parse failures stay on the repository invalid-input path without aliases.
---
