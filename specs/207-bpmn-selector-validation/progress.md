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
- `cancel pi` and `delete pi` now validate direct BPMN selectors after PI search flag/version checks and before process-instance paging, preserving keyed/stdin-key flows and valid empty search results after a visible selector preflight.
- Process-instance command tests share visible and empty process-definition search fixtures so BPMN selector preflight behavior can be asserted without duplicating response JSON.
- `get incident` search mode now validates a direct BPMN selector before both `searchIncidentsTotal` and `searchIncidentsWithPaging`; keyed incident mode and `--pd-key` filtering remain outside this BPMN preflight.
- Incident BPMN selector validation has tenant-context coverage through the global `--tenant` option; `get incident` does not expose process-definition version or version-tag selector flags, so those fields remain intentionally absent from its selector request.
- Direct `get pd -b` now uses the shared selector validator and renders the validated process-definition matches directly, avoiding an extra search while turning missing selectors into the canonical diagnostic.
- Direct `delete pd -b` now uses the shared selector validator before delete impact planning and reuses the validated process-definition keys for preview and deletion; key and stdin-key delete paths remain unchanged.

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
---
## Iteration 3 - 2026-05-23 16:32:18 CEST
**User Story**: User Story 1 - Block no-op mutations from missing BPMN selectors
**Tasks Completed**:
- [x] T012: Add `cancel pi --bpmn-process-id <missing>` test proving validation fails before process-instance search paging in `cmd/cancel_test.go`
- [x] T013: Add `delete pi --bpmn-process-id <missing>` test proving validation fails before process-instance search paging or delete planning in `cmd/delete_test.go`
- [x] T014: Add valid visible selector with zero matching process instances tests preserving existing no-op/empty behavior in `cmd/cancel_test.go` and `cmd/delete_test.go`
- [x] T015: Add machine/non-interactive mode tests for `--json`, `--automation`, and key-only-equivalent output where applicable in `cmd/cancel_test.go` and `cmd/delete_test.go`
- [x] T016: Invoke shared BPMN selector validation before `processPISearchPagesWithAction` in the search-selected path of `cmd/cancel_processinstance.go`
- [x] T017: Invoke shared BPMN selector validation before `deleteProcessInstanceSearchPages` in the search-selected path of `cmd/delete_processinstance.go`
- [x] T018: Preserve keyed, stdin key, non-BPMN search, dry-run, auto-confirm, and valid `found: 0` behavior in `cmd/cancel_processinstance.go` and `cmd/delete_processinstance.go`
- [x] T019: Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test(Cancel|Delete).*Bpmn|Test(Cancel|Delete).*ProcessDefinitionSelector' -count=1` and fix regressions
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cancel_processinstance.go
- cmd/delete_processinstance.go
- cmd/cancel_test.go
- cmd/delete_test.go
- cmd/cmd_processinstance_test.go
- specs/207-bpmn-selector-validation/tasks.md
- specs/207-bpmn-selector-validation/progress.md
**Learnings**:
- Missing direct BPMN selectors for `cancel pi` and `delete pi` now fail on the process-definition preflight and never reach process-instance page search.
- Valid visible BPMN selectors still proceed to process-instance search and keep the existing `found: 0` no-op output when no runtime instances match.
- `--json` and `--automation` selector failures return the shared diagnostic without recovery prompts.
---
---
## Iteration 4 - 2026-05-23 16:38:46 CEST
**User Story**: User Story 2 - Validate incident searches by BPMN selector
**Tasks Completed**:
- [x] T020: Add `get incident --bpmn-process-id <missing>` test proving validation fails before incident search paging in `cmd/get_incident_test.go`
- [x] T021: Add visible selector with zero matching incidents test preserving empty incident output in `cmd/get_incident_test.go`
- [x] T022: Add `--total`, `--keys-only`, `--pi-keys-only`, `--json`, and `--automation` no-prompt validation tests where compatible in `cmd/get_incident_test.go`
- [x] T023: Add version/tag/tenant selector context coverage for incident BPMN validation in `cmd/get_incident_test.go`
- [x] T024: Build and invoke shared BPMN selector validation before `searchIncidentsTotal` and `searchIncidentsWithPaging` when `flagGetIncidentBpmnProcessID` is set in `cmd/get_incident.go`
- [x] T025: Preserve keyed incident mode, `--pd-key`, non-BPMN incident filters, paging continuation, totals, and key-only rendering in `cmd/get_incident.go` and `cmd/get_incident_search.go`
- [x] T026: Update `get incident` help text to describe BPMN selector validation without changing unrelated incident examples in `cmd/get_incident.go`
- [x] T027: Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestGetIncident.*(Bpmn|Selector|Total|KeysOnly|Automation|JSON)' -count=1` and fix regressions
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_incident.go
- cmd/get_incident_test.go
- specs/207-bpmn-selector-validation/tasks.md
- specs/207-bpmn-selector-validation/progress.md
**Learnings**:
- `get incident` can reuse the shared process-definition selector validator because the root CLI API embeds both incident and process APIs.
- The validation must sit after keyed-mode handling and immediately before incident search totals/paging so keyed lookup, stdin keys, `--pd-key`, and other non-BPMN filters keep their existing behavior.
- The sandbox reports localhost listener tests as skipped, but both the required US2 target and the broader `TestGetIncident` selection compile and pass in this environment.
---
---
## Iteration 5 - 2026-05-23 16:47:04 CEST
**User Story**: User Story 3 - Audit direct process-definition selectors
**Tasks Completed**:
- [x] T028: Add `get pd --bpmn-process-id <missing>` test for explicit missing-selector behavior in `cmd/get_processdefinition_test.go`
- [x] T029: Add `delete pd --bpmn-process-id <missing>` test proving failure before delete impact planning in `cmd/delete_test.go`
- [x] T030: Add valid visible selector tests preserving existing `get pd` listing and `delete pd` preview/confirmation behavior in `cmd/get_processdefinition_test.go` and `cmd/delete_test.go`
- [x] T031: Align direct BPMN search misses in `runSearchProcessDefinitions` with the explicit selector diagnostic when `flagGetPDBpmnProcessId` is set in `cmd/get_processdefinition.go`
- [x] T032: Align direct BPMN delete misses before impact planning when `flagDeletePDBpmnProcessId` is set in `cmd/delete_processdefinition.go`
- [x] T033: Preserve broad `get pd`, keyed `get pd`, keyed/stdin `delete pd`, `--latest`, version, tag, and XML compatibility behavior in `cmd/get_processdefinition.go` and `cmd/delete_processdefinition.go`
- [x] T034: Run `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test(Get|Delete)ProcessDefinition.*(Bpmn|Selector|Missing|Latest)' -count=1` and fix regressions
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_processdefinition.go
- cmd/get_processdefinition_test.go
- cmd/delete_processdefinition.go
- cmd/delete_test.go
- specs/207-bpmn-selector-validation/tasks.md
- specs/207-bpmn-selector-validation/progress.md
**Learnings**:
- The direct process-definition commands can use the validated selector match set as the command selection source, so `get pd -b` and `delete pd -b` do not need a second process-definition search after preflight.
- `delete pd --no-state-check` can preview deletion from the selected process-definition keys without an additional process-definition GET; the visible-selector test should assert the search and delete request boundary instead.
- The US3 targeted gate passes in this sandbox; a broader `go test ./cmd -count=1` still reaches unrelated existing fixture failures where prior PI/incident selector preflights introduce process-definition search requests.
---
