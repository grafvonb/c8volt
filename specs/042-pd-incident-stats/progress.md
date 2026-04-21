# Ralph Progress Log

Feature: 042-pd-incident-stats
Started: 2026-04-21 18:20:22

## Codebase Patterns

- `get process-definition --stat` is wired once at the command layer through `collectOptions()` and then delegated into versioned services; version branching belongs in `internal/services/processdefinition/*`, not in the CLI renderer.
- Process-definition search results are sorted centrally in the versioned services before rendering, so feature work should preserve service-owned ordering rather than re-sorting in `cmd`.
- The public `c8volt/process` facade mirrors `internal/domain` process-definition statistics almost 1:1, so model changes usually need matching domain, facade, and conversion updates together.
- Existing regression coverage is already split by concern: versioned service tests for sourcing, facade tests for option/model passthrough, and command tests for visible rendering.
- The shared process-definition stats seam can carry version capability through the model itself; `IncidentCountSupported` belongs beside `Incidents` in both domain and facade models so later rendering stays version-agnostic.

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
