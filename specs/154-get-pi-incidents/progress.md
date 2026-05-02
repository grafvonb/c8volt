# Ralph Progress Log

Feature: 154-get-pi-incidents
Started: 2026-05-02 09:54:51

## Codebase Patterns

- Generated v8.8/v8.9 incident results expose BPMN flow-node metadata as `ElementId`/`ElementInstanceKey`; feature-facing models use `FlowNodeId`/`FlowNodeInstanceKey`, so conversion helpers should map element fields into flow-node fields.
- Existing public process models use `omitempty` broadly, but the incident-enriched JSON wrapper keeps `total`, `items`, `item`, `incidents`, `processInstanceKey`, and `errorMessage` non-omitempty so requested enrichment can render empty incident collections and empty messages explicitly.
- Process facade methods translate service errors through `ferr.FromDomain`, keep public/domain model mapping in `c8volt/process/convert.go`, and can compose higher-level helpers in `client.go` without leaking internal service types.
- Process-instance command validation happens after stdin/flag keys are merged and before lookup/search requests; use command-local helpers like `missingDependentFlagsf` and `mutuallyExclusiveFlagsf` so failures stay in the invalid-input path.

---

## Iteration 1 - 2026-05-02 09:56:52 CEST
**User Story**: Phase 1: Setup (Shared Infrastructure)
**Tasks Completed**:
- [x] T001: Review generated incident search shapes and record any field mismatch in `specs/154-get-pi-incidents/research.md`
- [x] T002: Add domain incident detail model in `internal/domain/processinstance.go`
- [x] T003: Add public incident detail and enriched output models in `c8volt/process/model.go`
- [x] T004: Add incident-enrichment contract notes to `specs/154-get-pi-incidents/contracts/get-pi-with-incidents.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/model.go
- internal/domain/processinstance.go
- specs/154-get-pi-incidents/contracts/get-pi-with-incidents.md
- specs/154-get-pi-incidents/progress.md
- specs/154-get-pi-incidents/research.md
- specs/154-get-pi-incidents/tasks.md
**Learnings**:
- `SearchProcessInstanceIncidentsWithResponse` exists for both generated supported clients and returns the same incident result fields needed for later service conversion.
- Focused validation passed with `go test ./internal/domain ./c8volt/process -count=1`.
---
---
## Iteration 2 - 2026-05-02 10:04:34 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T005: Extend `internal/services/processinstance.API` with tenant-aware incident lookup in `internal/services/processinstance/api.go`
- [x] T006: Extend public `process.API` with process-instance incident lookup/enrichment in `c8volt/process/api.go`
- [x] T007: Add conversion helpers between domain and public incident models in `c8volt/process/convert.go`
- [x] T008: Implement facade incident lookup and keyed result enrichment helpers in `c8volt/process/client.go`
- [x] T009: Add `--with-incidents` flag storage and registration in `cmd/get_processinstance.go`
- [x] T010: Add early validation for keyed-only `--with-incidents` usage in `cmd/get_processinstance.go`
- [x] T011: Add foundational facade tests for incident conversion/enrichment in `c8volt/process/client_test.go`
- [x] T012: Add command validation tests for `--with-incidents` without `--key` and with search-mode filters in `cmd/get_processinstance_test.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/api.go
- c8volt/process/client.go
- c8volt/process/client_test.go
- c8volt/process/convert.go
- c8volt/resource/client_test.go
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- cmd/process_api_stub_test.go
- internal/services/processinstance/api.go
- internal/services/processinstance/v87/contract.go
- internal/services/processinstance/v87/incidents.go
- internal/services/processinstance/v88/contract.go
- internal/services/processinstance/v88/incidents.go
- internal/services/processinstance/v89/contract.go
- internal/services/processinstance/v89/incidents.go
- specs/154-get-pi-incidents/progress.md
- specs/154-get-pi-incidents/tasks.md
**Learnings**:
- `--with-incidents` validation should stay independent of the later render path so unsupported or unimplemented enrichment work is not reached for invalid flag combinations.
- Versioned process-instance services now expose the incident lookup seam, with concrete supported-version request construction left to the US1 service tasks.
- Validation passed with `go test ./c8volt/process ./cmd ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1` and adjacent API checks passed with `go test ./c8volt/resource ./c8volt/variable ./c8volt/task -count=1`.
---
