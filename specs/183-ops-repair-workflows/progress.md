# Progress: Ops Repair Workflows

## Traceability

- Issue: #183
- Branch: `183-ops-repair-workflows`
- Mandatory implementation context: `specs/ralph-implementation-rules.md`
- Commit subject suffix: `#183`

## Clarification Gate

- 2026-05-17: No critical ambiguities detected worth formal clarification before planning. The issue defines command targets, input modes, frozen target behavior, job applicability, variable update scope, dry-run, reports, architecture constraints, and out-of-scope behavior.

## Codebase Patterns

- Dry-run repair planning now flows through one internal service helper after target freezing, so explicit incident, incident search, explicit process-instance, and process-instance search modes share the same planned variable, job, resolution, and remaining-summary statuses without calling mutation primitives.
- Repair report options are accepted and validated during dry-run planning, with `--report-format` requiring `--report-file`; report paths are preserved and formats are inferred for output, but report file writing remains a separate US6 concern.
- Compact repair human output uses `report: planned <path> (<format>)` for requested report previews so it does not imply a report file was written before US6 report rendering exists.
- Repair variable flags reuse the existing `update pi` JSON object parser through a narrow repair wrapper, so `--vars` and `--vars-file` keep the same mutual exclusion and parse failure semantics on both repair targets.
- Process-instance variable confirmation now verifies only requested process-scope variable names after JSON normalization, using `SearchProcessInstanceVariables` so formatting and numeric representation differences do not fail valid confirmations.
- Ops repair applies variable payloads once per unique process-instance scope before incident mutation, records dependent incident keys per scope, and blocks dependent incident resolution when a scope update fails or confirmation fails.
- Verbose repair output includes deduped variable scope rows plus per-incident variable step status; compact output reports only the variable scope count to keep operator output scan-friendly.
- `ops repair process-instance` reuses the `get pi` process-definition/date/state/relationship validation globals and helper functions, but owns separate target keys, retries, timeout, and repair-specific search-mode validation.
- Process-instance repair search mode is explicit: `--incidents-only` or `--direct-incidents-only` is required before remote selection, and `--key`/stdin input is rejected with search filters before mutation.
- Internal process-instance repair first freezes selected process-instance keys, discovers direct active incidents through the incident service, dedupes incidents by key in selected-process-instance order, and then routes the frozen incidents through the shared incident repair execution path.
- The public repair request now carries `DirectIncidentsOnly` so the command can preserve the selected direct-incident semantics across the facade boundary without encoding CLI-only state in process filters.
- Adding the process-instance repair child affects docsgen expectations: generated `ops repair` docs should list both concrete repair targets while still omitting target-specific flags from the grouping page.
- `ops repair incident` search mode owns repair-specific incident filter globals and reuses the same validation helpers as `get incident`: state/error/date validation, key-shaped context validation, batch-size/limit guardrails, and keyed-plus-filter rejection before remote mutation.
- Filtered incident repair uses `DiscoveryMode=search` to call `incident.SearchIncidents` once with `Limit` taking precedence over `BatchSize`, freezes that returned set, and then reuses the explicit incident repair execution path without re-querying or expanding scope.
- Dry-run search repair still performs discovery and plan construction, but leaves job updates and incident resolution uncalled; human output shows the filter summary before frozen counts and verbose key/step rows.
- `ops repair incident` now owns explicit incident repair input semantics: repeated `--key`, stdin `-`, local key validation, `--retries` defaulting to `1`, optional `--job-timeout`, and inherited bulk controls.
- Explicit incident repair freezes incident details with `incident.GetIncidents` before any job or resolution mutation, then derives incident, process-instance, root, job, and variable-scope key sets from the frozen details.
- Repair job steps are per incident: missing `JobKey` yields `not_applicable`; `--retries 0` yields `skipped`; retry updates can be confirmed through job service primitives; timeout updates submit without retry confirmation semantics.
- Repair command output uses a dedicated `cmd/cmd_views_ops_repair.go` renderer and shared result envelopes for JSON, keeping command orchestration separate from human/JSON rendering.
- Adding a concrete child under an ops grouping command changes generated capabilities and docs expectations: the grouping command becomes `limited` contract support while remaining automation-unsupported and flag-free.
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
- `internal/domain/ops_repair.go` now owns the shared repair request/result/report model, while `c8volt/ops/model.go` mirrors only public JSON-facing repair types.
- `internal/services/ops.NewWithRepairDependencies` injects the job service for repair while keeping `NewWithWorkflowDependencies` available for older workflow tests and non-repair callers.
- Foundational repair service methods currently validate process-instance, incident, and job dependencies and return a planned result skeleton; later user stories should replace the skeleton with concrete discovery and mutation behavior without changing the facade shape.
- Public repair facade methods in `c8volt/ops/client.go` remain thin: convert public models, delegate to internal ops, map domain errors, and map results back through `c8volt/ops/convert.go`.

## Validation Notes

- Iteration 2 foundational validation passed with targeted ops/facade tests and full `go test ./... -count=1` using `GOCACHE=/private/tmp/go-build-cache` for sandbox-compatible cache writes.
- Iteration 6 US4 validation passed with the exact targeted command `GOCACHE=/private/tmp/go-build-cache go test ./cmd ./internal/services/ops ./internal/services/processinstance -run 'TestOpsRepair.*Var|Test.*Variable' -count=1` plus broader related package validation `GOCACHE=/private/tmp/go-build-cache go test ./cmd ./c8volt/process ./c8volt/ops ./internal/services/ops ./internal/services/processinstance -count=1`.
- Iteration 7 US5 validation passed with `GOCACHE=/private/tmp/go-build-cache go test ./cmd ./internal/services/ops -run 'TestOpsRepair.*DryRun' -count=1`, dry-run rendering validation, and broader related package validation `GOCACHE=/private/tmp/go-build-cache go test ./cmd ./internal/services/ops -count=1`.

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
---
## Iteration 2 - 2026-05-17 17:32:20 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T007: Add internal repair request/result/domain models and `not_applicable` workflow status
- [x] T008: Add public repair request/result models and `not_applicable` workflow status mapping
- [x] T009: Extend internal ops service API and constructors to accept job service dependency
- [x] T010: Extend public ops facade API with repair workflow methods
- [x] T011: Implement repair model conversions
- [x] T012: Implement thin public repair facade methods
- [x] T013: Wire job service into ops facade construction
- [x] T014: Add foundational ops repair model/conversion tests
- [x] T015: Add internal repair workflow constructor/dependency tests
- [x] T016: Mark foundational tasks complete and record validation notes
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/client.go
- c8volt/ops/api.go
- c8volt/ops/client.go
- c8volt/ops/client_test.go
- c8volt/ops/convert.go
- c8volt/ops/model.go
- internal/domain/ops_orphan_purge.go
- internal/domain/ops_repair.go
- internal/services/ops/api.go
- internal/services/ops/repair.go
- internal/services/ops/repair_test.go
- specs/183-ops-repair-workflows/tasks.md
- specs/183-ops-repair-workflows/progress.md
**Learnings**:
- `not_applicable` belongs in the shared ops workflow step vocabulary because repair job steps need it for mixed job-backed and non-job incidents.
- Keeping repair service methods as planned skeletons lets later user stories add explicit incident and process-instance behavior without changing public method signatures.
- Validation run: `go test ./c8volt/ops ./internal/services/ops -run 'TestClientRepairIncidentsMapsServiceBoundary|TestRepairWorkflows' -count=1`; `GOCACHE=/private/tmp/go-build-cache go test ./c8volt ./c8volt/ops ./internal/services/ops -count=1`; `GOCACHE=/private/tmp/go-build-cache go test ./... -count=1`.
---
---
## Iteration 3 - 2026-05-17 17:48:42 CEST
**User Story**: User Story 1 - Repair Explicit Incidents
**Tasks Completed**:
- [x] T017: Add command tests for `ops repair incident --help`, no top-level parent `--key`, explicit `--key`, stdin `-`, and invalid key failures
- [x] T018: Add internal service tests for frozen explicit incident keys and mixed job-backed/non-job repair planning
- [x] T019: Add facade tests for explicit incident repair request conversion and error mapping
- [x] T020: Add command contract metadata tests for `ops repair incident`
- [x] T021: Add `ops repair incident` Cobra command, explicit key flags, stdin key handling, and validation
- [x] T022: Implement explicit incident repair planning and target freezing
- [x] T023: Implement per-incident job applicability and default retry planning using job service primitives
- [x] T024: Implement incident resolution and confirmation delegation through incident service primitives
- [x] T025: Add human and JSON rendering for explicit incident repair results
- [x] T026: Set mutation, contract, output-mode, and automation metadata for `ops repair incident`
- [x] T027: Run targeted validation
- [x] T028: Mark US1 tasks complete and record validation notes
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/ops/client_test.go
- cmd/capabilities_test.go
- cmd/cmd_views_ops_repair.go
- cmd/command_contract_test.go
- cmd/ops_contract.go
- cmd/ops_contract_test.go
- cmd/ops_repair_incident.go
- cmd/ops_repair_incident_test.go
- docsgen/main_test.go
- internal/services/ops/repair.go
- internal/services/ops/repair_test.go
- specs/183-ops-repair-workflows/tasks.md
- specs/183-ops-repair-workflows/progress.md
**Learnings**:
- Explicit incident repair can stay inside the existing layering by having the command call the public ops facade and the internal ops workflow compose incident and job services.
- The frozen target set must be built before calling job or resolution mutations; later filtered discovery should reuse the same frozen-set construction path rather than mutate during paging.
- Validation run: `GOCACHE=/private/tmp/go-build-cache go test ./cmd ./c8volt/ops ./internal/services/ops -run 'TestOpsRepairIncident|TestCommandContract' -count=1`; `GOCACHE=/private/tmp/go-build-cache go test ./cmd ./c8volt/ops ./internal/services/ops -count=1`; `GOCACHE=/private/tmp/go-build-cache go test ./docsgen -run TestGeneratedOpsDocsDocumentGroupingCommands -count=1`; `GOCACHE=/private/tmp/go-build-cache go test ./... -count=1`.
---
---
## Iteration 4 - 2026-05-17 17:59:40 CEST
**User Story**: User Story 2 - Discover Incidents With Filters
**Tasks Completed**:
- [x] T029: Add command tests for incident filter flags, keyed-plus-filter rejection, batch-size/limit validation, and dry-run output
- [x] T030: Add internal service tests for filtered incident discovery, frozen target sets, and no expansion to newly created incidents
- [x] T031: Add command rendering tests for dry-run incident repair rows and JSON
- [x] T032: Reuse `get incident` filter parsing and validation patterns for repair incident search mode
- [x] T033: Implement incident-filter discovery and frozen repair set construction
- [x] T034: Implement dry-run behavior that discovers and validates without variable, job, or incident mutations
- [x] T035: Render dry-run discovery filters, frozen keys, job applicability, retry/timeout requests, and resolution targets
- [x] T036: Run targeted validation
- [x] T037: Mark US2 tasks complete and record validation notes
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cmd_views_ops_repair.go
- cmd/cmd_views_ops_repair_test.go
- cmd/ops_repair_incident.go
- cmd/ops_repair_incident_test.go
- internal/services/ops/repair.go
- internal/services/ops/repair_test.go
- specs/183-ops-repair-workflows/tasks.md
- specs/183-ops-repair-workflows/progress.md
**Learnings**:
- Search-mode repair deliberately treats explicit search flags as required, so the default active state does not turn a bare `ops repair incident` invocation into a mutation target.
- The existing incident search helper already captures version-specific paging and local filtering behavior; repair only needs to choose the bounded size and freeze the returned details.
- Validation run: `GOCACHE=/private/tmp/go-build-cache go test ./cmd ./internal/services/ops -run 'TestOpsRepairIncident|TestRenderOpsRepairIncident|TestRepairIncidentsSearchMode' -count=1`; `GOCACHE=/private/tmp/go-build-cache go test ./cmd ./internal/services/ops -run 'TestOpsRepairIncident' -count=1`; `GOCACHE=/private/tmp/go-build-cache go test ./cmd ./internal/services/ops -count=1`; `GOCACHE=/private/tmp/go-build-cache go test ./... -count=1`.
---
---
## Iteration 5 - 2026-05-17 18:11:48 CEST
**User Story**: User Story 3 - Repair Incidents From Process Instances
**Tasks Completed**:
- [x] T038: Add command tests for `ops repair process-instance --help`, explicit keys, stdin `-`, invalid keys, and keyed-plus-filter rejection
- [x] T039: Add command tests requiring `--incidents-only` or `--direct-incidents-only` in process-instance search mode
- [x] T040: Add internal service tests for process-instance selection, incident discovery, deduped incident keys, and deterministic reporting
- [x] T041: Add command contract metadata tests for `ops repair process-instance`
- [x] T042: Add `ops repair process-instance` Cobra command, explicit key flags, stdin key handling, PI filter flags, and validation
- [x] T043: Reuse `get pi` search filter validation and incident-bearing selector behavior
- [x] T044: Implement process-instance selection and active incident discovery
- [x] T045: Deduplicate incident keys and route process-instance repair through the shared incident repair workflow
- [x] T046: Render process-instance repair discovery, frozen incidents, duplicate handling, and final result
- [x] T047: Set mutation, contract, output-mode, and automation metadata for `ops repair process-instance`
- [x] T048: Run targeted validation
- [x] T049: Mark US3 tasks complete and record validation notes
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/ops/convert.go
- c8volt/ops/model.go
- cmd/cmd_views_ops_repair.go
- cmd/command_contract_test.go
- cmd/ops_repair_processinstance.go
- cmd/ops_repair_processinstance_test.go
- docsgen/main_test.go
- internal/domain/ops_repair.go
- internal/services/ops/orphan_purge_test.go
- internal/services/ops/repair.go
- internal/services/ops/repair_test.go
- specs/183-ops-repair-workflows/tasks.md
- specs/183-ops-repair-workflows/progress.md
**Learnings**:
- Process-instance repair can preserve layering by having the command pass PI selectors to the public ops facade while the internal ops workflow composes process-instance lookup/search and incident discovery primitives.
- Direct incident selection needs a durable request bit in addition to the process-instance filter, because the existing process filter only represents incident-bearing state, not direct-only semantics.
- Validation run: `GOCACHE=/private/tmp/go-build-cache go test ./cmd ./internal/services/ops -run 'TestOpsRepairProcessInstance|TestCommandContract' -count=1`; `GOCACHE=/private/tmp/go-build-cache go test ./cmd ./c8volt/ops ./internal/services/ops -count=1`; `GOCACHE=/private/tmp/go-build-cache go test ./docsgen -run TestGeneratedOpsDocsDocumentGroupingCommands -count=1`; `GOCACHE=/private/tmp/go-build-cache go test ./... -count=1`.
---
---
## Iteration 6 - 2026-05-17 18:24:51 CEST
**User Story**: User Story 4 - Apply Shared Variable Updates Safely
**Tasks Completed**:
- [x] T050: Add command tests for `--vars`, `--vars-file`, parse failures, and variable-scope output
- [x] T051: Add internal service tests for variable scope dedupe, normalized confirmation, and blocked dependent incidents
- [x] T052: Add process-instance facade tests for requested variable confirmation behavior
- [x] T053: Reuse existing process-instance variable parsing and validation for repair variable flags
- [x] T054: Add requested variable confirmation in process-instance service primitives
- [x] T055: Implement deduped variable scope update planning and execution
- [x] T056: Block incident resolution for incidents dependent on failed variable scopes
- [x] T057: Render variable scopes, requested names, confirmation status, and blocked incidents
- [x] T058: Run targeted validation
- [x] T059: Mark US4 tasks complete and record validation notes
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/client_test.go
- cmd/cmd_views_ops_repair.go
- cmd/ops_repair_incident.go
- cmd/ops_repair_incident_test.go
- cmd/ops_repair_processinstance.go
- cmd/ops_repair_processinstance_test.go
- cmd/ops_repair_variables.go
- internal/services/ops/orphan_purge_test.go
- internal/services/ops/repair.go
- internal/services/ops/repair_test.go
- internal/services/processinstance/variables.go
- specs/183-ops-repair-workflows/tasks.md
- specs/183-ops-repair-workflows/progress.md
**Learnings**:
- Repair variable scope is consistently the process-instance key; deduped scope updates can be applied before shared incident repair without changing the public repair model shape.
- The process-instance variable service can provide normalized requested-name confirmation centrally, so repair does not need to duplicate JSON comparison logic.
- Validation run: `GOCACHE=/private/tmp/go-build-cache go test ./cmd ./internal/services/ops ./internal/services/processinstance -run 'TestOpsRepair.*Var|Test.*Variable' -count=1`; `GOCACHE=/private/tmp/go-build-cache go test ./cmd ./c8volt/process ./c8volt/ops ./internal/services/ops ./internal/services/processinstance -count=1`.
---
---
## Iteration 7 - 2026-05-17 18:31:52 CEST
**User Story**: User Story 5 - Preview Repair Without Mutation
**Tasks Completed**:
- [x] T060: Add dry-run no-mutation service tests covering variables, jobs, and incident resolution
- [x] T061: Add dry-run command tests for both repair targets with report options
- [x] T062: Add dry-run rendering tests for planned report path/format and mixed job applicability
- [x] T063: Normalize dry-run planning behavior across repair discovery modes
- [x] T064: Ensure command planning validates report paths without implying mutations
- [x] T065: Ensure dry-run JSON includes frozen targets, variable scopes, job keys, applicability, retry/timeout requests, and resolution targets
- [x] T066: Run targeted validation
- [x] T067: Mark US5 tasks complete and record validation notes
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cmd_views_ops_repair.go
- cmd/cmd_views_ops_repair_test.go
- cmd/ops_repair_incident.go
- cmd/ops_repair_incident_test.go
- cmd/ops_repair_processinstance.go
- cmd/ops_repair_processinstance_test.go
- internal/services/ops/repair.go
- internal/services/ops/repair_test.go
- specs/183-ops-repair-workflows/tasks.md
- specs/183-ops-repair-workflows/progress.md
**Learnings**:
- Dry-run can produce a complete mutation preview by reusing the same frozen incident plan builder after discovery and stopping before variable, job, or incident mutation services are invoked.
- Report options need to be represented in the repair request before US6 writes files, because dry-run JSON and human previews must show the planned report path and resolved format.
- Validation run: `GOCACHE=/private/tmp/go-build-cache go test ./cmd ./internal/services/ops -run 'TestOpsRepair.*DryRun' -count=1`; `GOCACHE=/private/tmp/go-build-cache go test ./cmd ./internal/services/ops -count=1`.
---
