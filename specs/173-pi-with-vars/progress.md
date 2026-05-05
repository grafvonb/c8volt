# Ralph Progress Log

Feature: 173-pi-with-vars
Started: 2026-05-05 21:28:44

## Codebase Patterns

- `cmd/get_processinstance.go` routes direct keyed lookup through `mergeAndValidateKeys`, `GetProcessInstances`, keyed-only validation, optional incident enrichment, then `listProcessInstancesView`; `--with-vars` should follow that keyed branch and preserve default output when unset.
- Process-instance command tests reset package globals in `resetProcessInstanceCommandGlobals`; any new process-instance flags need reset coverage there to avoid cross-test leakage.
- Incident enrichment is the nearest pattern: facade methods `SearchProcessInstanceIncidents` and `EnrichProcessInstancesWithIncidents` preserve item order, filter details back to the owning process-instance key, and render via a dedicated `cmd/cmd_views_processinstance_incidents.go` file with an age-meta JSON wrapper.
- Shared process-instance service contracts live in `internal/services/processinstance/api.go` plus each versioned `contract.go`; versioned service interfaces also mirror the generated client methods they call.
- v8.8/v8.9 generated Camunda clients expose `SearchVariablesWithResponse` for `/variables/search`, including filters and `truncateValues`, but `VariableResultBase` omits returned `value` and `isTruncated`; later implementation likely needs focused raw response decoding at the service boundary.
- v8.7 has an Operate generated `/variables/search` surface (`SearchVariablesForProcessInstancesWithResponse`) whose `Variable` model includes `processInstanceKey`, `scopeKey`, `value`, and `truncated`; this supports an explicit v8.7 implementation path unless later runtime validation proves otherwise.
- User-facing docs source starts in `README.md`; `docs/index.md` is synced from README by `docsgen`, and `docs/cli/` command reference is generated from Cobra via `make docs-content`.

---

## Iteration 1 - 2026-05-05 21:30:38 CEST
**User Story**: Phase 1: Setup (Shared Discovery)
**Tasks Completed**: 
- [x] T001: Inspect keyed process-instance lookup, validation, and render dispatch in `cmd/get_processinstance.go`
- [x] T002: Inspect incident enrichment facade and command patterns in `c8volt/process/client.go`, `c8volt/process/model.go`, and `cmd/cmd_views_processinstance_incidents.go`
- [x] T003: Inspect v8.8/v8.9 generated `/variables/search` request and response types in `internal/clients/camunda/v88/camunda/client.gen.go` and `internal/clients/camunda/v89/camunda/client.gen.go`
- [x] T004: Inspect v8.7 process-instance API surface and decide whether variable search is unsupported or available through an existing client in `internal/services/processinstance/v87/contract.go`
- [x] T005: Inspect process-instance command docs and generated documentation paths in `README.md`, `docs/index.md`, `docs/cli/`, and `docsgen/`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**: 
- specs/173-pi-with-vars/tasks.md
- specs/173-pi-with-vars/progress.md
**Learnings**:
- Keyed lookup is already the right insertion point for `--with-vars`; list/search variable enrichment should stay rejected until a later contract explicitly enables it.
- v8.8/v8.9 generated clients can build `/variables/search` requests but do not expose value/truncation fields through typed results.
- v8.7 should not be assumed unsupported: the Operate client has a variable search endpoint with the required value/truncation fields.
- Generated CLI docs should be regenerated with `make docs-content` after Cobra help/examples change.
---
