# Progress: Ops Repair Workflows

## Traceability

- Issue: #183
- Branch: `183-ops-repair-workflows`
- Mandatory implementation context: `specs/ralph-implementation-rules.md`
- Commit subject suffix: `#183`

## Clarification Gate

- 2026-05-17: No critical ambiguities detected worth formal clarification before planning. The issue defines command targets, input modes, frozen target behavior, job applicability, variable update scope, dry-run, reports, architecture constraints, and out-of-scope behavior.

## Codebase Patterns

- `cmd/ops_repair.go` is a grouping command only: `Use: "repair"`, `cobra.NoArgs`, help-only `RunE`, no target `--key`, no direct workflow execution, and target-specific subcommands must own their own key/filter semantics.
- `get incident` already owns the repair-relevant incident selector shape: repeated `--key`, stdin `-`, `mergeAndValidateKeys(...).Unique()`, `validateKeys`, `hasGetIncidentSearchModeFlags`, `populateGetIncidentSearchFilter`, `SearchIncidentsPage`, and keyed/search mutual exclusion before remote calls.
- `internal/domain.ProcessInstanceIncidentDetail` already carries the incident freeze data repair needs, including incident key, process-instance key, root key, flow-node and element-instance keys, error context, state, tenant, process definition data, and optional `JobKey`.
- `internal/services/incident.API` exposes incident primitives repair should compose: exact lookup, paged search, process-instance incident search, incident resolution, and resolution wait/confirmation methods.
- `get pi` keeps keyed process-instance lookup separate from search mode, rejects ambiguous keyed-plus-filter combinations, uses shared process-definition/date filter registration, and applies relationship/incident filters through `applyPISearchResultFilters` after each backend page where needed.
- Process-instance incident selectors already distinguish `--incidents-only` from `--direct-incidents-only`; direct incident enrichment uses incident filters such as state, error type, and error message only when the corresponding selector is active.
- `cmd/update_processinstance_variables.go` is the source pattern for repair variable parsing: `--vars` and `--vars-file` are mutually exclusive, payloads must decode as JSON objects, and previews compare normalized decoded values against current process-scope variables.
- `internal/services/processinstance.UpdateProcessInstancesVariables` dedupes process-instance keys with `typex.Keys.Unique()`, chooses workers through `toolx.DetermineNoOfWorkers`, executes with `pool.ExecuteSlice`, and delegates single-scope confirmation to `UpdateProcessInstanceVariables`.
- `update job` builds a local pre-mutation plan, validates JSON mutation guardrails, treats retry confirmation separately from timeout submission, rejects timeout updates for non-active jobs, and uses `--dry-run`, `--auto-confirm`, or `--automation` for non-interactive JSON mutation output.
- Job service support is versioned: v88/v89 implement search-by-key and update through Camunda v2 generated clients with retry confirmation via `job/waiter`; v87 returns explicit unsupported domain errors for get/update.
- `c8volt/client.go` already constructs a job service and public `JobAPI`, but `opsvc.NewWithWorkflowDependencies` currently receives cluster, process-instance, incident, process-definition, and resource services only; repair must add job service injection there instead of having ops reach through the public facade.
- `cmd/ops_contract.go` defines shared ops report status and file handling but does not yet include `not_applicable`; repair must add that status consistently in command, facade, and domain vocabularies.
- `ops purge process-instances-with-incidents` is the closest command workflow pattern: validate flags before `NewCli`, validate report paths before mutation, build a public ops request with started time and command metadata, use dry-run planning before interactive confirmation, write reports on success and post-discovery failures, and render through the shared command envelope for machine output.
- `cmd/cmd_views_ops_purge_processinstances_with_incidents.go` keeps compact human output separate from deterministic JSON/report rendering; verbose mode reveals key lists while compact output reports counts and hidden-key guidance.
- `cmd/ops_repair.go` already defines the grouping command and must remain free of top-level target `--key` semantics.
- `cmd/ops_purge_processinstances_with_incidents.go` demonstrates incident-filtered ops workflow flags, report handling, pre-mutation planning, confirmation, automation metadata, and deterministic rendering patterns.
- `internal/services/ops/api.go` currently injects process-instance, incident, process-definition, resource, and cluster services; this feature must add job service injection for repair without bypassing resource services.
- `c8volt/ops/model.go` already has shared workflow statuses, but repair requires adding `not_applicable`.
- Existing incident primitives live in `internal/services/incident`; process-instance search and variable updates live in `internal/services/processinstance`; job lookup and update live in `internal/services/job`.

## Validation Notes

- Planning artifacts only so far; no implementation validation has been run.

---
## Iteration 1 - 2026-05-17 17:21:56 CEST
**User Story**: Phase 1: Setup (Shared Discovery)
**Tasks Completed**:
- [x] T001: Inspect existing ops repair grouping behavior in `cmd/ops_repair.go` and record target-specific command constraints
- [x] T002: Inspect incident command/filter patterns in `cmd/get_incident.go`, `cmd/get_incident_search.go`, `internal/services/incident/api.go`, and `internal/domain/incident.go`
- [x] T003: Inspect process-instance search and variable update patterns in `cmd/get_processinstance*.go`, `cmd/update_processinstance_variables.go`, and `internal/services/processinstance/variables.go`
- [x] T004: Inspect job lookup/update patterns in `cmd/update_job.go`, `c8volt/job`, and `internal/services/job`
- [x] T005: Inspect ops report and automation patterns in `cmd/ops_contract.go`, `cmd/ops_purge_processinstances_with_incidents.go`, and `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`
- [x] T006: Record mandatory Ralph context, issue traceability, and discovered implementation patterns
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/183-ops-repair-workflows/tasks.md
- specs/183-ops-repair-workflows/progress.md
**Learnings**:
- No conflict was found between `specs/ralph-implementation-rules.md` and the active feature artifacts.
- The first implementation iteration should start with foundational models and dependency wiring, especially adding job service injection to the ops workflow boundary.
- Repair command implementation should reuse existing selector validation and report patterns instead of creating parallel parsing, output, or file-writing behavior.
---
