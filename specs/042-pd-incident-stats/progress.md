# Ralph Progress Log

Feature: 042-pd-incident-stats
Started: 2026-04-21 18:20:22

## Codebase Patterns

- `get process-definition --stat` is wired once at the command layer through `collectOptions()` and then delegated into versioned services; version branching belongs in `internal/services/processdefinition/*`, not in the CLI renderer.
- Process-definition search results are sorted centrally in the versioned services before rendering, so feature work should preserve service-owned ordering rather than re-sorting in `cmd`.
- The public `c8volt/process` facade mirrors `internal/domain` process-definition statistics almost 1:1, so model changes usually need matching domain, facade, and conversion updates together.
- Existing regression coverage is already split by concern: versioned service tests for sourcing, facade tests for option/model passthrough, and command tests for visible rendering.

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
