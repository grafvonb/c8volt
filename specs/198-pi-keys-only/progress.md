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
