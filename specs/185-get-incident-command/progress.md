# Ralph Progress Log

Feature: 185-get-incident-command
Started: 2026-05-09 21:46:32

## Codebase Patterns

- Generated Camunda v8.8 and v8.9 clients both expose top-level `SearchIncidentsWithResponse(ctx, body)` and `GetIncidentWithResponse(ctx, incidentKey)` methods. Their `IncidentFilter` types include tenant, state, error type, error message, process definition, process instance, flow-node, job, incident key, and creation time fields; `IncidentSearchQueryResult` returns `Items` plus `Page` metadata.
- `internal/services/incident.API` currently exposes direct lookup, resolution, process-instance scoped incident lookup, and wait helpers. v8.7 returns `domain.ErrUnsupported` for tenant-unsafe direct and process-instance incident lookup; v8.8 and v8.9 support direct lookup through `GetIncidentWithResponse`.
- v8.8 process-instance incident lookup intentionally avoids sending the scoped endpoint `filter` object and filters tenant/state/error type/error message locally after paging. v8.9 sends safe tenant/state/error type filters server-side and applies error-message filtering locally with `incidentfilter.ErrorMessageContains`.
- Existing incident conversion reuses `domain.ProcessInstanceIncidentDetail` / `process.ProcessInstanceIncidentDetail`; `CreationTime` is formatted as RFC3339Nano and nil job/root keys become empty strings before human rendering decides whether to show `n/a`.
- Process-instance command validation centralizes flag relationship errors in `cmd/get_processinstance_validation.go` using helpers such as `invalidFlagValuef`, `mutuallyExclusiveFlagsf`, and `missingDependentFlagsf`; incident enum validation delegates to `internal/services/incidentfilter`.
- Human/list rendering uses `pickMode`, `itemView`, `listOrJSONFlat`, `renderJSONPayload`, `renderOutputLine`, and flat row helpers in `cmd/cmd_views_get.go`; totals print only the count through `processInstanceTotalView`.
- Process-instance pagination honors command/config page size, trims pages with local `--limit`, incrementally renders one-line and keys-only modes when appropriate, auto-continues for JSON/automation, and falls back to page-by-page counting for exact totals when local filters make backend totals unsafe.

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
