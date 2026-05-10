# Ralph Progress Log

Feature: 198-pi-keys-only
Started: 2026-05-10 23:58:09

## Codebase Patterns

- `get incident` registers command-local flags in `cmd/get_incident.go`, validates them in `validateGetIncidentFlagValues`, then splits keyed lookup from search/list mode in `Run`.
- Existing incident keyed lookup merges repeated `--key` values and stdin `-` with `mergeAndValidateKeys(...).Unique()` before fetching; search filters are rejected when keyed mode is active.
- Collected incident output is centralized in `listIncidentsView`; JSON uses `renderJSONPayload`, `--keys-only` emits `IncidentKey` lines without `found:`, and human output appends `found: N`.
- Incremental incident search output is separate in `renderIncidentSearchPage`; `searchIncidentsWithPaging` prints `found:` only for incremental one-line mode, not keys-only.
- Incident command tests use `newIncidentLookupServer`, `newIncidentSearchCaptureServerWithResponses`, and `executeRootForIncidentTest*` helpers with reset hooks for global flag state.
- View tests use `newGetViewTestCommand`, `resetViewModeFlags`, and direct `listIncidentsView` calls to verify render modes.
- Docs expectations are protected by `docsgen/main_test.go` generated markdown substring checks and command capability expectations are protected in `cmd/command_contract_test.go`.
- `cancel pi` dedupes merged flag/stdin process-instance keys at command boundary with `.Unique()`, while `delete pi` currently validates merged keys without `.Unique()`.
- Incident command-local flags must be reset in `resetGetIncidentFlagState`; output-mode conflicts are validated in `validateGetIncidentFlagValues` before any lookup or search.
- Incident process-instance-key rendering can live beside `listIncidentsView` and should skip empty `ProcessInstanceKey` values while preserving duplicate non-empty values.
- Keyed incident `--pi-keys-only` output returns early from `get_incident.go` after lookup and before shared human/list rendering.
- Collected incident search/list `--pi-keys-only` output is selected in `listIncidentsView`, while incremental pages are selected in `renderIncidentSearchPage`.
- Incremental incident search should keep `found:` only for one-line human output; `--pi-keys-only` and `--keys-only` remain footer-free.
- Incident validation tests can assert pre-request failures by using a local capture server and requiring its request log stays empty.

---

---
## Iteration 1 - 2026-05-10 23:59:50 CEST
**User Story**: Phase 1: Setup
**Tasks Completed**:
- [x] T001: Inspect existing incident output and validation paths in `cmd/get_incident.go`, `cmd/get_incident_search.go`, and `cmd/cmd_views_get.go`
- [x] T002: Inspect existing incident tests and docs expectations in `cmd/get_incident_test.go`, `cmd/cmd_views_get_test.go`, `cmd/command_contract_test.go`, and `docsgen/main_test.go`
- [x] T003: Inspect delete/cancel stdin dedupe behavior in `cmd/delete_processinstance.go`, `cmd/cancel_processinstance.go`, `cmd/delete_test.go`, and `cmd/cancel_test.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/198-pi-keys-only/tasks.md
- specs/198-pi-keys-only/progress.md
**Learnings**:
- `--pi-keys-only` can be added as command-local incident state without changing global render modes; keyed lookup, collected search, and incremental search each need explicit wiring.
- Missing process-instance-key skip and duplicate preservation belong in a small incident rendering helper, not in key merge/dedupe utilities.
- `delete pi` parity with `cancel pi` is likely a one-line `.Unique()` change after merged stdin/flag validation plus focused dry-run/planning coverage.
---

---
## Iteration 2 - 2026-05-11 00:02:48 CEST
**User Story**: Phase 2: Foundational
**Tasks Completed**:
- [x] T004: Add `flagGetIncidentPIKeysOnly` registration and help text in `cmd/get_incident.go`
- [x] T005: Add local validation in `cmd/get_incident.go` rejecting `--pi-keys-only` with `--keys-only`, `--json`, `--total`, `--error-message-limit`, or `--with-no-error-message`
- [x] T006: Add a small incident process-instance-key rendering helper in `cmd/cmd_views_get.go` that emits non-empty `ProcessInstanceKey` values and preserves duplicates
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_incident.go
- cmd/cmd_views_get.go
- specs/198-pi-keys-only/tasks.md
- specs/198-pi-keys-only/progress.md
**Learnings**:
- `--pi-keys-only` validation is intentionally command-local and can reuse `pickMode()` to guard shared `--json` and `--keys-only` modes.
- The rendering helper returns `nil` to match existing view signatures even though `renderOutputLine` itself does not surface write errors.
---

---
## Iteration 3 - 2026-05-11 00:08:00 CEST
**User Story**: User Story 1 - Pipe incident matches into process-instance commands
**Tasks Completed**:
- [x] T007: Add keyed lookup and missing-process-instance-key skip tests for `--pi-keys-only` in `cmd/get_incident_test.go`
- [x] T008: Add search/list tests proving `--pi-keys-only` emits process instance keys and preserves duplicate process instance keys in `cmd/get_incident_test.go`
- [x] T009: Add view-level tests for process-instance-key rendering and duplicate preservation in `cmd/cmd_views_get_test.go`
- [x] T010: Add paging/incremental rendering coverage for `--pi-keys-only` in `cmd/get_incident_test.go`
- [x] T011: Wire keyed lookup output to the process-instance-key rendering helper in `cmd/get_incident.go`
- [x] T012: Wire collected search/list output to the process-instance-key rendering helper in `cmd/cmd_views_get.go`
- [x] T013: Wire incremental search page output to process-instance-key rendering in `cmd/get_incident_search.go`
- [x] T014: Ensure `--pi-keys-only` incremental search omits `found:` summaries while preserving existing `--keys-only` summaries in `cmd/get_incident_search.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_incident.go
- cmd/get_incident_search.go
- cmd/cmd_views_get.go
- cmd/get_incident_test.go
- cmd/cmd_views_get_test.go
- specs/198-pi-keys-only/tasks.md
- specs/198-pi-keys-only/progress.md
**Learnings**:
- `--pi-keys-only` uses the existing incident result rows and does not require service or model changes.
- Command-level tests that start local HTTP servers are skipped in this sandbox because binding `127.0.0.1` is not permitted; direct renderer coverage and `go test ./cmd` still pass.
- The incremental search footer guard should check `!flagGetIncidentPIKeysOnly` so default one-line searches still print `found:` while process-instance-key pipelines stay line-only.
---

---
## Iteration 4 - 2026-05-11 00:12:12 CEST
**User Story**: User Story 2 - Avoid ambiguous output mode combinations
**Tasks Completed**:
- [x] T015: Add validation tests for `--pi-keys-only` with `--json`, `--keys-only`, and `--total` in `cmd/get_incident_test.go`
- [x] T016: Add validation tests for `--pi-keys-only` with `--error-message-limit` and `--with-no-error-message` in `cmd/get_incident_test.go`
- [x] T017: Finalize local mutual-exclusion diagnostics for all incompatible `--pi-keys-only` combinations in `cmd/get_incident.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_incident_test.go
- specs/198-pi-keys-only/tasks.md
- specs/198-pi-keys-only/progress.md
**Learnings**:
- Existing `validateGetIncidentFlagValues` diagnostics already cover all specified `--pi-keys-only` conflicts, including message-format flags, so no production validator change was needed.
- The new tests prove local validation runs before lookup/search by configuring a test server and asserting no requests are captured.
- Validation checks passed with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestGetIncidentCommand_RejectsPIKeysOnly|TestGetIncidentCommand_RejectsJSONErrorMessageLimit|TestGetIncidentCommand_RejectsKeysOnlyErrorMessageLimit' -count=1` and `GOCACHE=/tmp/c8volt-gocache go test ./cmd -count=1`.
---
