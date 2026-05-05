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
- Shared facade API changes require updating test stubs in `cmd/process_api_stub_test.go` and `c8volt/resource/client_test.go` even when the runtime command behavior is not wired yet.
- v8.8/v8.9 variable raw decoding should read `value` from the response body and accept both `isTruncated` and legacy `truncated` markers when converting to `APITruncated`.

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
---
## Iteration 2 - 2026-05-05 21:37:57 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T006: Add `ProcessInstanceVariable`, `VariableEnrichedProcessInstance`, and `VariableEnrichedProcessInstances` facade models in `c8volt/process/model.go`
- [x] T007: Add matching domain variable models in `internal/domain/processinstance.go`
- [x] T008: Add process-instance service API method signatures for searching process-instance variables in `internal/services/processinstance/api.go`, `internal/services/processinstance/v87/contract.go`, `internal/services/processinstance/v88/contract.go`, and `internal/services/processinstance/v89/contract.go`
- [x] T009: Add process facade API methods for searching and enriching process instances with variables in `c8volt/process/api.go`
- [x] T010: Add domain/facade conversion helpers for process-instance variables in `c8volt/process/convert.go`
- [x] T011: Add v8.8/v8.9 variable conversion helpers and raw value/truncation decoding support in `internal/services/processinstance/v88/convert.go` and `internal/services/processinstance/v89/convert.go`
- [x] T012: Run targeted foundational validation
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/api.go
- c8volt/process/client.go
- c8volt/process/client_test.go
- c8volt/process/convert.go
- c8volt/process/model.go
- c8volt/resource/client_test.go
- cmd/process_api_stub_test.go
- internal/domain/processinstance.go
- internal/services/processinstance/api.go
- internal/services/processinstance/v87/contract.go
- internal/services/processinstance/v87/variables.go
- internal/services/processinstance/v88/contract.go
- internal/services/processinstance/v88/convert.go
- internal/services/processinstance/v88/service_test.go
- internal/services/processinstance/v88/variables.go
- internal/services/processinstance/v89/contract.go
- internal/services/processinstance/v89/convert.go
- internal/services/processinstance/v89/service_test.go
- internal/services/processinstance/v89/variables.go
- specs/173-pi-with-vars/tasks.md
- specs/173-pi-with-vars/progress.md
**Learnings**:
- The foundational interface expansion compiles cleanly with placeholder service methods; user-visible command behavior remains unwired for the next work unit.
- v8.8/v8.9 generated typed variable results still omit value/truncation fields, so service implementation should decode from raw response bodies before mapping variables.
- Validation passed with `GOCACHE=/tmp/c8volt-gocache go test ./c8volt/process ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -run 'Test.*Variable|Test.*API' -count=1` and a compile-only check for `./cmd ./c8volt/resource`.
---
