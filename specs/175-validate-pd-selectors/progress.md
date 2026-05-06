# Ralph Progress Log

Feature: 175-validate-pd-selectors
Started: 2026-05-06 12:15:57

## Codebase Patterns

- Process-definition selector failures now route through `processDefinitionSelectorValidationError`, which keeps the existing local-precondition error text but can optionally ask interactive human users whether to list visible process definitions.
- Selector listing prompt eligibility is stricter than shared confirmation: it is disabled for `--auto-confirm`, `--json`, `--keys-only`, `--automation`, and non-TTY stdin/stdout as checked by `processDefinitionSelectorInteractiveTerminalFn`.
- Accepted selector listing prompts search visible process definitions with an empty `process.ProcessDefinitionFilter` and render through `listProcessDefinitionsView`, preserving the existing flat process-definition rows and `found: <n>` summary.
- Process-definition selector validation now lives in `cmd/process_definition_selector_validation.go` with request/result structs, per-BPMN `process.ProcessDefinitionFilter` construction, distinct-ID normalization, any-version versus latest validation modes, and no-prompt local-precondition diagnostics.
- `get pi` search mode now runs shared process-definition selector validation before `--total` or paged process-instance search; keyed and user-task lookup paths still bypass the BPMN preflight.
- Command tests can reuse `stubProcessAPI` for process-definition search callbacks; facade tests can authorize `SearchProcessDefinitionsLatest` through the extended `stubProcessDefinitionAPI` instead of panicking on latest searches.
- Shared process-instance selector flags are registered by `registerPISharedProcessDefinitionFilterFlags` and stored in `flagGetPIBpmnProcessID`, `flagGetPIProcessVersion`, and `flagGetPIProcessVersionTag`.
- `populatePISearchFilterOpts` maps BPMN process ID, process-definition version, version tag, and process-definition key into `process.ProcessInstanceFilter`; `validatePISearchFlags` keeps `--pd-key` mutually exclusive with BPMN/version/tag selectors and requires BPMN ID for version/tag selectors.
- `get pi` search mode calls `searchProcessInstancesWithPaging`; `cancel pi` and `delete pi` reuse `populatePISearchFilterOpts` before `processPISearchPagesWithAction` or `deleteProcessInstanceSearchPages`, while keyed paths bypass search filters.
- `run pi` supports multiple `--bpmn-process-id` values, rejects `--pd-version` with multiple BPMN IDs, and constructs all `process.ProcessInstanceData` before create calls.
- Process-definition visibility checks should use the `process.API` facade: `SearchProcessDefinitions` and `SearchProcessDefinitionsLatest` convert `process.ProcessDefinitionFilter` through `toDomainProcessDefinitionFilter`; the current facade test stub panics on latest searches unless extended for that path.
- Existing process-definition recovery listings can reuse `runSearchProcessDefinitions` or `listProcessDefinitionsView`; `listOrJSONFlat` renders one-line rows with `found: <n>`, JSON payloads, or keys-only IDs from the same data.
- Prompt behavior is centralized around `confirmCmdOrAbortFn`, `shouldImplicitlyConfirm`, `automationModeEnabled`, `flagViewAsJson`, and `flagViewKeysOnly`; non-TTY stdin auto-confirms existing prompts, so selector listing prompt eligibility will need an explicit no-prompt policy for non-TTY and machine modes.

---

---
## Iteration 1 - 2026-05-06 12:17:33 CEST
**User Story**: Phase 1: Setup (Shared Discovery)
**Tasks Completed**:
- [x] T001: Inspect shared process-instance BPMN/version/tag flags and search filter construction in `cmd/get_processinstance.go`
- [x] T002: Inspect process-instance search/mutation paging paths in `cmd/get_processinstance.go`, `cmd/cancel_processinstance.go`, and `cmd/delete_processinstance.go`
- [x] T003: Inspect `run pi` BPMN process ID and version validation in `cmd/run_processinstance.go`
- [x] T004: Inspect process-definition search facade and conversion behavior in `c8volt/process/api.go`, `c8volt/process/client.go`, `c8volt/process/convert.go`, and `c8volt/process/model.go`
- [x] T005: Inspect process-definition list rendering and tests in `cmd/get_processdefinition.go`, `cmd/cmd_views_get.go`, and `cmd/get_test.go`
- [x] T006: Inspect automation, JSON, keys-only, and prompt helpers in `cmd/` before adding the listing offer
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/175-validate-pd-selectors/tasks.md
- specs/175-validate-pd-selectors/progress.md
**Learnings**:
- The first implementation should insert selector validation only in BPMN search/start paths so keyed lookups and non-BPMN searches retain existing behavior.
- The process-definition listing offer can reuse the existing flat process-definition renderer, but prompt eligibility must be stricter than `confirmCmdOrAbort` because current non-TTY confirmation returns nil.
---

---
## Iteration 2 - 2026-05-06 12:22:02 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T007: Add shared process-definition selector validation request/result types and helper skeletons in `cmd/process_definition_selector_validation.go`
- [x] T008: Add helper logic that maps process-instance BPMN/version/tag flags to `process.ProcessDefinitionFilter` in `cmd/process_definition_selector_validation.go`
- [x] T009: Add helper logic that validates one or more BPMN process IDs through `process.API.SearchProcessDefinitions` or `SearchProcessDefinitionsLatest` in `cmd/process_definition_selector_validation.go`
- [x] T010: Add reusable missing-selector diagnostic formatting and no-prompt error behavior in `cmd/process_definition_selector_validation.go`
- [x] T011: Add process facade tests proving `SearchProcessDefinitions` receives BPMN process ID, version, and version tag filters in `c8volt/process/client_test.go`
- [x] T012: Add command unit tests for selector-to-filter construction and missing selector formatting in `cmd/process_definition_selector_validation_test.go`
- [x] T013: Run targeted foundational tests
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/process_definition_selector_validation.go
- cmd/process_definition_selector_validation_test.go
- cmd/process_api_stub_test.go
- c8volt/process/client_test.go
- specs/175-validate-pd-selectors/tasks.md
- specs/175-validate-pd-selectors/progress.md
**Learnings**:
- Missing-selector diagnostics should be local precondition errors because the selector can be absent, filtered by version/tag/tenant, or invisible to credentials rather than syntactically invalid.
- The helper treats `ProcessDefinitions.Total > 0` or non-empty `Items` as a visible match so both facade-produced and hand-built test responses are supported.
- `run pi` can use the same validator with `processDefinitionSelectorValidationLatest` when exact `--pd-version` is absent; search/mutation paths should keep the default any-version mode.
---

---
## Iteration 3 - 2026-05-06 12:28:33 CEST
**User Story**: User Story 1 - Detect Missing Selectors Before Empty Results
**Tasks Completed**:
- [x] T014: Add command test for `get pi --bpmn-process-id <missing>` failing before process-instance search in `cmd/get_processinstance_test.go`
- [x] T015: Add command test for visible process definition with zero matching process instances preserving `found: 0` in `cmd/get_processinstance_test.go`
- [x] T016: Add command test proving `--pd-version`, `--pd-version-tag`, and tenant options are included in validation context in `cmd/get_processinstance_test.go`
- [x] T017: Invoke shared selector validation before `get pi` process-instance search when `--bpmn-process-id` is set in `cmd/get_processinstance.go`
- [x] T018: Ensure successful validation allows existing `searchProcessInstancesWithPaging` and `found: 0` behavior to continue unchanged in `cmd/get_processinstance.go`
- [x] T019: Run targeted `get pi` selector tests
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- specs/175-validate-pd-selectors/tasks.md
- specs/175-validate-pd-selectors/progress.md
**Learnings**:
- Full command-path selector tests use local HTTP listeners and are skipped in this sandbox because TCP bind is unavailable; `go test ./cmd` still passes here and will exercise those tests where local listeners are allowed.
- The validation call can run unconditionally in the `get pi` search branch because an empty BPMN selector normalizes to no IDs and performs no process-definition facade calls.
---

---
## Iteration 4 - 2026-05-06 12:33:41 CEST
**User Story**: User Story 2 - Provide Safe Human and Automation Diagnostics
**Tasks Completed**:
- [x] T020: Add human-output diagnostic tests for a single missing selector
- [x] T021: Add human-output diagnostic tests for multiple missing selectors
- [x] T022: Add tests proving `--json`, `--automation`, `--keys-only`, and non-TTY execution do not prompt
- [x] T023: Add test proving accepted interactive listing uses existing process-definition list rendering
- [x] T024: Implement prompt eligibility checks for visible-definition listing
- [x] T025: Implement interactive listing through existing process-definition search/list view helpers
- [x] T026: Ensure structured and automation modes return clear errors without prompt text
- [x] T027: Run targeted selector prompt/diagnostic tests and fix regressions
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_processinstance.go
- cmd/process_definition_selector_validation.go
- cmd/process_definition_selector_validation_test.go
- specs/175-validate-pd-selectors/tasks.md
- specs/175-validate-pd-selectors/progress.md
**Learnings**:
- Recovery listing should reuse the existing process-definition flat list renderer directly instead of duplicating output formatting.
- Prompt eligibility needs its own TTY hook because the shared confirmation helper auto-confirms non-TTY stdin, which would make recovery listing unsafe for piped or automated execution.
- `--auto-confirm` should suppress selector recovery prompts so unattended commands do not emit extra listing output after validation failures.
---
