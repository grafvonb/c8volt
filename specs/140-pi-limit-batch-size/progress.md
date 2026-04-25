# Ralph Progress Log

Feature: 140-pi-limit-batch-size
Started: 2026-04-25 15:48:48

## Codebase Patterns

- Shared process-instance search paging lives in `cmd/get_processinstance.go`; `flagGetPISize`, `resolvePISearchSize`, `newPISearchPageRequest`, `searchProcessInstancesWithPaging`, and `processPISearchPagesWithAction` are the main seams for batch-size and total-limit behavior.
- Process-instance flag validation should accept the current `*cobra.Command` when it needs `Flags().Changed(...)`; referencing package-level command variables from shared validation creates Go initialization cycles.
- Removed process-instance flags should use `useInvalidInputFlagErrors` on the affected leaf commands so Cobra parse failures map to the repository invalid-input exit model.
- `cancel process-instance` and `delete process-instance` register their own command flags but reuse get-side process-instance search globals and paging helpers for search-mode destructive workflows.
- Multi-page process-instance tests use `newProcessInstanceSearchCaptureServerWithResponses`, `decodeCapturedPISearchPages`, `decodeCapturedTopLevelPISearchPages`, and `safeSlice` helpers to assert request paging, continuation prompts, and destructive side effects.
- Command test helpers reset package-level Cobra flag globals with `resetProcessInstanceCommandGlobals`; future flag additions need reset coverage to avoid cross-test leakage.
- Generated CLI docs under `docs/cli/` currently mirror Cobra command metadata and should be regenerated from command source after help/examples change, not hand-edited as the source of truth.

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
