# Ralph Progress Log

Feature: 207-bpmn-selector-validation
Started: 2026-05-23 16:18:50

## Codebase Patterns

- `get pi` already validates direct `--bpmn-process-id` selectors before process-instance totals/paging with `validateProcessDefinitionSelectors`, `newPIProcessDefinitionSelectorValidationRequest`, and `handleProcessDefinitionSelectorValidationError`.
- `run pi` already validates one or more direct BPMN IDs before creation; latest mode is used unless `--pd-version` selects an exact process-definition version.
- Shared PI selector flags live in `cmd/get_processinstance_filtering.go`; `cancel pi` and `delete pi` reuse those globals, so single-selector helper additions can target the same flag set.
- `cancel pi` and `delete pi` enter search-selected mutation paths after `validatePISearchFlags`, key/date/limit checks, and `validatePISearchVersionSupport`; insert BPMN selector validation after those checks and before `processPISearchPagesWithAction` or `deleteProcessInstanceSearchPages`.
- `get incident` search mode builds `incident.Filter.ProcessDefinitionId` from `flagGetIncidentBpmnProcessID` and then calls `searchIncidentsTotal` or `searchIncidentsWithPaging`; keyed incident mode rejects search filters and should remain unchanged.
- Direct `get pd -b` and `delete pd -b` currently search process definitions directly and render an empty list or `no process definitions found to delete`; missing-selector alignment should be scoped to direct BPMN selectors before rendering/planning.
- Existing generated CLI docs describe selector validation for `get pi` and `run pi`, but `cancel pi`, `delete pi`, `get incident`, `get pd`, and `delete pd` wording still describes plain filtering or no-op behavior.
- Shared selector construction now includes single-BPMN helpers for incident and direct process-definition commands, with latest-aware mode for `get pd` and `delete pd`.
- `validateProcessDefinitionSelectorsForCommand` returns the shared local-precondition missing-selector error immediately when prompt policy forbids recovery output, while prompt-eligible callers can still pass the invalid result to the existing recovery handler.
- `stubProcessAPI` can now install incident paging callbacks and reusable failing page-search guards for command tests that need to prove selector validation happens before process-instance or incident paging.

---
## Iteration 1 - 2026-05-23 16:20:13 CEST
**User Story**: Phase 1: Setup (Shared Discovery)
**Tasks Completed**:
- [x] T001: Audit every direct `--bpmn-process-id` registration in `cmd/get_processinstance_filtering.go`, `cmd/run_processinstance.go`, `cmd/cancel_processinstance.go`, `cmd/delete_processinstance.go`, `cmd/get_incident.go`, `cmd/get_processdefinition.go`, and `cmd/delete_processdefinition.go`
- [x] T002: Inspect shared selector validation behavior and reusable gaps in `cmd/process_definition_selector_validation.go` and `cmd/process_definition_selector_validation_test.go`
- [x] T003: Inspect process-instance search-selected mutation paths in `cmd/cancel_processinstance.go`, `cmd/delete_processinstance.go`, `cmd/get_processinstance_paging.go`, and `cmd/get_processinstance_search.go`
- [x] T004: Inspect incident search filters, paging, totals, and output modes in `cmd/get_incident.go`, `cmd/get_incident_search.go`, and `cmd/get_incident_test.go`
- [x] T005: Inspect direct process-definition search/delete behavior and tests in `cmd/get_processdefinition.go`, `cmd/get_processdefinition_test.go`, `cmd/delete_processdefinition.go`, and `cmd/delete_test.go`
- [x] T006: Inspect README and generated documentation surfaces for affected command wording in `README.md`, `docs/index.md`, and `docs/cli/`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/207-bpmn-selector-validation/tasks.md
- specs/207-bpmn-selector-validation/progress.md
**Learnings**:
- Current tests intentionally document the gap for `cancel pi -b missing-process` and `delete pi -b missing-process` as direct process-instance searches returning `found: 0`; those tests will need to flip in US1.
- `cmd/process_api_stub_test.go` already provides panic-on-unexpected-call hooks for process-definition, process-instance, and incident searches, which is suitable for proving validation happens before resource paging.
- Incident tests have capture servers around `SearchIncidentsPage`; missing-selector tests can fail if an incident page request is observed before selector validation.
- Direct process-definition commands can reuse the process API selector validation shape, but `get pd` currently accepts a broader `c8volt.API`, so helper signatures may need to stay on the process-facing interface already embedded by that API.
---
---
## Iteration 2 - 2026-05-23 16:26:00 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T007: Extend or add shared single-selector request construction in `cmd/process_definition_selector_validation.go` for non-PI commands that directly accept one BPMN process ID
- [x] T008: Add reusable validation entry points that return local precondition errors without recovery prompts when command mode forbids prompting in `cmd/process_definition_selector_validation.go`
- [x] T009: Add helper tests for single-selector request construction, version/tag narrowing, and no-prompt modes in `cmd/process_definition_selector_validation_test.go`
- [x] T010: Add or update command test stubs to prove selector validation happens before resource paging by failing if search methods are called in `cmd/process_api_stub_test.go` or existing command test stubs
- [x] T011: Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*ProcessDefinitionSelector' -count=1` and fix foundational compile/test failures
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/process_definition_selector_validation.go
- cmd/process_definition_selector_validation_test.go
- cmd/process_api_stub_test.go
- specs/207-bpmn-selector-validation/tasks.md
- specs/207-bpmn-selector-validation/progress.md
**Learnings**:
- Single-selector request builders let future command changes share normalization, version/tag narrowing, and latest-mode selection without duplicating request literals.
- No-prompt command validation can return the canonical local-precondition error before recovery handling, which keeps machine and pipeline modes compact.
- Reusable paging-failure guards are now available for US1 and US2 tests that need to prove runtime resource paging is not reached after a missing selector.
---
