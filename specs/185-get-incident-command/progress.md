# Ralph Progress Log

Feature: 185-get-incident-command
Started: 2026-05-09 21:46:32

## Codebase Patterns

- Plain `get incident` now switches to search/list mode only when no `--key` flags and no stdin `-` are provided; explicit keyed mode rejects search flags before client construction.
- Incident search paging uses the same command/config batch-size source and continuation concepts as `get pi`, but advances offset by requested page size because service-local incident filtering can reduce returned page item counts.
- Public process facade additions should update `c8volt/process/model.go`, `api.go`, `convert.go`, and concrete orchestration in `client.go`; command test stubs in `cmd/process_api_stub_test.go` must also satisfy the expanded `process.API`.
- Top-level incident search now has incident-specific page/result models in the public facade and internal domain. v8.7 rejects search as unsupported, v8.8 uses the top-level search endpoint with only tenant-safe server filtering plus local filters, and v8.9 pushes safe state/error-type/process/flow-node/creation-time filters server-side while keeping root-process-instance and error-message filtering local.
- Shared incident human row formatting lives behind `incidentHumanLineWithMessageLimit`, with `incidentHumanLine` preserving the existing process-instance incident flag behavior.
- Generated Camunda v8.8 and v8.9 clients both expose top-level `SearchIncidentsWithResponse(ctx, body)` and `GetIncidentWithResponse(ctx, incidentKey)` methods. Their `IncidentFilter` types include tenant, state, error type, error message, process definition, process instance, flow-node, job, incident key, and creation time fields; `IncidentSearchQueryResult` returns `Items` plus `Page` metadata.
- `internal/services/incident.API` currently exposes direct lookup, resolution, process-instance scoped incident lookup, and wait helpers. v8.7 returns `domain.ErrUnsupported` for tenant-unsafe direct and process-instance incident lookup; v8.8 and v8.9 support direct lookup through `GetIncidentWithResponse`.
- v8.8 process-instance incident lookup intentionally avoids sending the scoped endpoint `filter` object and filters tenant/state/error type/error message locally after paging. v8.9 sends safe tenant/state/error type filters server-side and applies error-message filtering locally with `incidentfilter.ErrorMessageContains`.
- Existing incident conversion reuses `domain.ProcessInstanceIncidentDetail` / `process.ProcessInstanceIncidentDetail`; `CreationTime` is formatted as RFC3339Nano and nil job/root keys become empty strings before human rendering decides whether to show `n/a`.
- Process-instance command validation centralizes flag relationship errors in `cmd/get_processinstance_validation.go` using helpers such as `invalidFlagValuef`, `mutuallyExclusiveFlagsf`, and `missingDependentFlagsf`; incident enum validation delegates to `internal/services/incidentfilter`.
- Human/list rendering uses `pickMode`, `itemView`, `listOrJSONFlat`, `renderJSONPayload`, `renderOutputLine`, and flat row helpers in `cmd/cmd_views_get.go`; totals print only the count through `processInstanceTotalView`.
- Process-instance pagination honors command/config page size, trims pages with local `--limit`, incrementally renders one-line and keys-only modes when appropriate, auto-continues for JSON/automation, and falls back to page-by-page counting for exact totals when local filters make backend totals unsafe.
- Plain `get incident` keyed lookup is registered under `get` with aliases `incidents` and `inc`; command validation uses `silenceUsageForError` for semantic flag errors, stdin key handling reuses `readKeysIfDash`/`mergeAndValidateKeys`, and keyed facade calls go through `process.API.GetIncidents`.
- Plain incident rendering now uses `listIncidentsView` in `cmd/cmd_views_get.go`, which delegates JSON, keys-only, and human list behavior to `listOrJSON` while reusing `incidentHumanLineWithMessageLimit` for compact rows.

---

---
## Iteration 1 - 2026-05-09 21:48:08 CEST
**User Story**: Phase 1: Setup (Shared Discovery)
**Tasks Completed**:
- [x] T001: Inspect top-level incident generated client search and lookup methods in `internal/clients/camunda/v88/camunda/client.gen.go` and `internal/clients/camunda/v89/camunda/client.gen.go`
- [x] T002: Inspect current incident service methods and version behavior in `internal/services/incident/api.go`, `internal/services/incident/v87/incidents.go`, `internal/services/incident/v88/incidents.go`, and `internal/services/incident/v89/incidents.go`
- [x] T003: Inspect existing process-instance incident validation and rendering in `cmd/get_processinstance_validation.go`, `cmd/cmd_views_processinstance_incidents.go`, and `cmd/get_processinstance_test.go`
- [x] T004: Inspect existing list, paging, limit, keys-only, total, and JSON conventions in `cmd/get_processinstance.go`, `cmd/get_processinstance_paging.go`, `cmd/get_processinstance_total.go`, and `cmd/cmd_views_get.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/185-get-incident-command/tasks.md
- specs/185-get-incident-command/progress.md
**Learnings**:
- Top-level incident search can be added within the existing incident service boundary rather than command code; v8.8 compatibility should continue avoiding scoped `filter` request shapes while v8.9 can use safe server filters.
- Plain `get incident` output should reuse existing render mode helpers and shared incident row formatting rather than creating a separate rendering framework.
---

---
## Iteration 2 - 2026-05-09 21:59:57 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T005: Add incident query/filter facade models and result metadata in `c8volt/process/model.go`
- [x] T006: Extend the process facade API for keyed incident lookup and incident search/list in `c8volt/process/api.go`
- [x] T007: Extend conversion helpers for incident query/filter/result values in `c8volt/process/convert.go`
- [x] T008: Extend the incident service API with top-level incident search/list support in `internal/services/incident/api.go`
- [x] T009: Add v8.7 unsupported incident search/list tests in `internal/services/incident/v87/incidents_test.go`
- [x] T010: Add v8.8 incident search/list compatibility tests in `internal/services/incident/v88/incidents_test.go`
- [x] T011: Add v8.9 incident search/list server-filter tests in `internal/services/incident/v89/incidents_test.go`
- [x] T012: Implement v8.7 unsupported incident search/list behavior in `internal/services/incident/v87/incidents.go` and `internal/services/incident/v87/contract.go`
- [x] T013: Implement v8.8 incident search/list compatibility path in `internal/services/incident/v88/incidents.go`, `internal/services/incident/v88/convert.go`, and `internal/services/incident/v88/contract.go`
- [x] T014: Implement v8.9 incident search/list server-side filters in `internal/services/incident/v89/incidents.go`, `internal/services/incident/v89/convert.go`, and `internal/services/incident/v89/contract.go`
- [x] T015: Add factory/API compile and version selection tests for incident search/list in `internal/services/incident/factory_test.go`
- [x] T016: Add shared incident row formatting helpers reusable by process-instance and incident output in `cmd/cmd_views_processinstance_incidents.go`
- [x] T017: Add facade tests for incident query validation, service option mapping, result metadata, and unsupported-version propagation in `c8volt/process/client_test.go`
- [x] T018: Implement facade orchestration for keyed lookup and incident search/list in `c8volt/process/client.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/api.go
- c8volt/process/client.go
- c8volt/process/client_test.go
- c8volt/process/convert.go
- c8volt/process/model.go
- cmd/cmd_views_get_test.go
- cmd/cmd_views_processinstance_incidents.go
- cmd/process_api_stub_test.go
- internal/domain/processinstance.go
- internal/services/incident/api.go
- internal/services/incident/factory_test.go
- internal/services/incident/v87/contract.go
- internal/services/incident/v87/incidents.go
- internal/services/incident/v87/incidents_test.go
- internal/services/incident/v88/contract.go
- internal/services/incident/v88/incidents.go
- internal/services/incident/v88/incidents_test.go
- internal/services/incident/v89/contract.go
- internal/services/incident/v89/convert.go
- internal/services/incident/v89/incidents.go
- internal/services/incident/v89/incidents_test.go
- specs/185-get-incident-command/tasks.md
- specs/185-get-incident-command/progress.md
**Learnings**:
- Expanding `process.API` requires updating command test doubles even when no command behavior changes yet.
- v8.8 top-level incident search can keep compatibility risk low by sending only tenant filtering to the backend and applying state, message, context, and time checks locally until later pagination work broadens local filtering.
- v8.9 generated filters can represent the safe exact fields needed by the plain incident command; root process instance and error message semantics remain local.
---

---
## Iteration 3 - 2026-05-09 22:10:53 CEST
**User Story**: User Story 1 - Fetch Known Incidents
**Tasks Completed**:
- [x] T019: Add command tests for `get incident --key`, repeated `--key`, stdin `-`, deduplication, missing keys, and invalid keys in `cmd/get_incident_test.go`
- [x] T020: Add human, JSON, and keys-only incident view tests in `cmd/cmd_views_get_test.go`
- [x] T021: Add command contract expectations for `get incident`, aliases `incidents` and `inc`, and inherited get flags in `cmd/command_contract_test.go`
- [x] T022: Register `get incident` with aliases, examples, flags, and help text in `cmd/get_incident.go` and wire it from `cmd/get.go`
- [x] T023: Implement keyed lookup parsing, stdin `-` handling, key merge, validation, and facade invocation in `cmd/get_incident.go`
- [x] T024: Implement incident human, JSON, and keys-only rendering in `cmd/cmd_views_get.go` and `cmd/cmd_views_processinstance_incidents.go`
- [x] T025: Ensure keyed lookup not-found and partial lookup failures preserve existing get command exit/output conventions in `cmd/get_incident.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cmd_views_get.go
- cmd/cmd_views_get_test.go
- cmd/command_contract_test.go
- cmd/completion_test.go
- cmd/get.go
- cmd/get_incident.go
- cmd/get_incident_test.go
- cmd/get_test.go
- specs/185-get-incident-command/tasks.md
- specs/185-get-incident-command/progress.md
**Learnings**:
- Direct incident lookup can be added as a thin command layer over the facade bulk `GetIncidents` method; using `--workers 1` in command tests makes deduplication request order deterministic.
- Cobra `Args` semantic validation needs `silenceUsageForError` to preserve existing invalid-input behavior without printing full command usage.
---

---
## Iteration 4 - 2026-05-09 22:20:25 CEST
**User Story**: User Story 2 - Search Incidents By Core Fields
**Tasks Completed**:
- [x] T026: Add command tests for default active state, `--state all`, invalid states, and state output in `cmd/get_incident_test.go`
- [x] T027: Add command tests for case-insensitive `--error-type` validation and generated valid-value messages in `cmd/get_incident_test.go`
- [x] T028: Add command tests for process instance, root process instance, process definition, flow node, and flow node instance filters in `cmd/get_incident_test.go`
- [x] T029: Add facade/service tests proving server-safe filter options are passed through in `c8volt/process/client_test.go` and `internal/services/incident/v89/incidents_test.go`
- [x] T030: Add search/list flags and validation for state, error type, process context, and flow-node context in `cmd/get_incident.go`
- [x] T031: Reuse `internal/services/incidentfilter` for error type normalization and valid-value help text in `cmd/get_incident.go`
- [x] T032: Map validated search filters to facade/service options in `c8volt/process/client.go` and `internal/services/calloption.go`
- [x] T033: Preserve existing get pagination, limit, interactive, auto-confirm, and non-interactive behavior for incident search in `cmd/get_incident.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/client_test.go
- cmd/command_contract_test.go
- cmd/get_incident.go
- cmd/get_incident_search.go
- cmd/get_incident_test.go
- specs/185-get-incident-command/tasks.md
- specs/185-get-incident-command/progress.md
**Learnings**:
- Generated top-level incident search request JSON serializes simple enum filters as direct values such as `"state":"ACTIVE"` and `"errorType":"IO_MAPPING_ERROR"`, not `$eq` wrapper objects.
- Root process instance filtering remains service-local for v8.9, so command-level paging must not use the filtered item count to compute the next offset.
- Core incident search filters already flow through the facade/domain incident filter model; service call options remain scoped to existing process-instance incident enrichment behavior.
---
