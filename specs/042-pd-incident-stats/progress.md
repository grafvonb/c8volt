# Ralph Progress Log

Feature: 042-pd-incident-stats
Started: 2026-04-21 18:20:22

## Codebase Patterns

- `get process-definition --stat` is wired once at the command layer through `collectOptions()` and then delegated into versioned services; version branching belongs in `internal/services/processdefinition/*`, not in the CLI renderer.
- Process-definition search results are sorted centrally in the versioned services before rendering, so feature work should preserve service-owned ordering rather than re-sorting in `cmd`.
- The public `c8volt/process` facade mirrors `internal/domain` process-definition statistics almost 1:1, so model changes usually need matching domain, facade, and conversion updates together.
- Existing regression coverage is already split by concern: versioned service tests for sourcing, facade tests for option/model passthrough, and command tests for visible rendering.
- The shared process-definition stats seam can carry version capability through the model itself; `IncidentCountSupported` belongs beside `Incidents` in both domain and facade models so later rendering stays version-agnostic.
- The `v88` and `v89` process-definition client interfaces are also reused in resource-service tests, so widening those interfaces requires keeping the resource test doubles compiling in the same iteration.

---

## Iteration 1 - 2026-04-21 18:36 CEST
**User Story**: Phase 1: Setup
**Tasks Completed**:
- [x] T001: Inventory the current `get process-definition --stat` flow
- [x] T002: Confirm the supported-version generated-client seams for incident-bearing process-instance counts
- [x] T003: Confirm the existing process-definition stats model and regression anchors
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/042-pd-incident-stats/research.md
- specs/042-pd-incident-stats/tasks.md
- specs/042-pd-incident-stats/progress.md
**Learnings**:
- `v8.8` and `v8.9` already centralize stats enrichment in `retrieveProcessDefinitionStats(...)`, which is the narrowest place to swap incident semantics without disturbing CLI entry points.
- The generated clients expose `GetProcessInstanceStatisticsByDefinitionWithResponse` plus `ActiveInstancesWithErrorCount`, while the current implementation still relies on the older `ProcessElementStatisticsResult.Incidents` field.
- The current shared stats model cannot represent `supported zero` versus `unsupported omission`, so the next phase needs a model-level support signal instead of renderer-side version checks.
---

## Iteration 2 - 2026-04-21 18:25 CEST
**User Story**: Phase 2: Foundational
**Tasks Completed**:
- [x] T004: Define the authoritative incident-statistics contract and version matrix
- [x] T005: Extend the shared process-definition statistics model to represent supported incident counts versus unsupported omission
- [x] T006: Update public-to-domain conversion and shared client coverage for the refined stats model
- [x] T007: Refresh planning artifacts to reflect the finalized model vocabulary and rendering rules
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/client_test.go
- c8volt/process/convert.go
- c8volt/process/model.go
- internal/domain/processdefinition.go
- specs/042-pd-incident-stats/contracts/process-definition-statistics.md
- specs/042-pd-incident-stats/data-model.md
- specs/042-pd-incident-stats/plan.md
- specs/042-pd-incident-stats/progress.md
- specs/042-pd-incident-stats/quickstart.md
- specs/042-pd-incident-stats/research.md
- specs/042-pd-incident-stats/tasks.md
**Learnings**:
- The smallest repository-native support seam is `Incidents` plus `IncidentCountSupported`; it preserves existing count fields while giving later service and renderer work an explicit unsupported state.
- A single facade test on `GetProcessDefinition` is enough to prove `supported zero` survives domain-to-public conversion without forcing service behavior changes in this foundational slice.
---

## Iteration 3 - 2026-04-21 18:33 CEST
**User Story**: User Story 1 - Show Correct Incident Counts
**Tasks Completed**:
- [x] T008: Add `v8.8` service tests proving `WithStat` uses incident-bearing process-instance count semantics
- [x] T009: Add `v8.9` service tests proving `WithStat` uses incident-bearing process-instance count semantics
- [x] T010: Add command rendering regressions for supported non-zero and supported zero incident counts
- [x] T011: Implement supported-version incident-count enrichment in `v8.8`
- [x] T012: Implement supported-version incident-count enrichment in `v8.9`
- [x] T013: Update the process-definition renderer to show `in:<count>` and `in:0` on supported versions
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cmd_views_get.go
- cmd/get_test.go
- internal/services/processdefinition/v88/contract.go
- internal/services/processdefinition/v88/service.go
- internal/services/processdefinition/v88/service_test.go
- internal/services/processdefinition/v89/contract.go
- internal/services/processdefinition/v89/service.go
- internal/services/processdefinition/v89/service_test.go
- internal/services/resource/v88/service_test.go
- internal/services/resource/v89/service_test.go
- specs/042-pd-incident-stats/tasks.md
- specs/042-pd-incident-stats/progress.md
**Learnings**:
- The exact supported-version contract is easier to satisfy by deduplicating active incident `ProcessInstanceKey` values per definition than by relying on the generated error-hash aggregation endpoints, which would still overcount across multiple distinct incident errors.
- Renderer behavior can stay version-agnostic once the services set `IncidentCountSupported=true`; the command only needs to decide whether to append the `in:` segment, not which platform version it is talking to.
- `make test` is a useful final gate here because the process-definition client interface is shared outside the story’s direct package, and widening it surfaced compile-only fallout in resource-service test doubles.
---
