# Ralph Progress Log

Feature: 289-api-latency-diagnostics
Started: 2026-09-02 06:58:25

## Setup Context

- Ralph context path: `specs/ralph-implementation-rules.md`
- Issue: `#289`
- Branch: `289-api-latency-diagnostics`
- Feature directory: `specs/289-api-latency-diagnostics`

## Validation Log

- 2026-09-02 07:02: prerequisite script selected `specs/289-api-latency-diagnostics` with research, data model, contracts, quickstart, and tasks available.
- 2026-09-02 07:02: setup inspection completed with `rg`, `sed`, `find`, `git branch --show-current`, and `git status --short --branch --untracked-files=all`.
- 2026-09-02 07:02: no Go tests were run because T001-T002 are context-capture tasks with no production or test code changes.

## Codebase Patterns

### Command Wiring

- Read-only analysis commands follow `cmd/ops_analyse_slow_process_instances.go`: Cobra command metadata, local argument/flag validation, `NewCli`, automation support checks, facade request construction, command-owned progress setup, facade dispatch, and renderer dispatch stay in `cmd`.
- Active ops workflows follow `cmd/ops_execute_smoketest.go`: validate local flags before `NewCli` when possible, use `shouldImplicitlyConfirm`, validate shared report paths before remote work, print tenant context before confirmation, call the facade once for execution, attempt requested reports on partial errors, then render.
- Command metadata is explicit: use `setCommandMutation`, `setContractSupport`, `setAutomationSupport`, and `setAllTenantsSupport` or `setOutputModes` where the command contract differs from defaults.

### Facade And Service Boundaries

- `c8volt/ops.API` and `c8volt/ops/client.go` expose thin public methods only. They convert through `convert.go`, delegate to `internal/services/ops.API`, map output back, and normalize errors via `ferrors.FromDomain`.
- `internal/services/ops.API` is the version-neutral workflow contract. `Service` owns cross-resource ops behavior and currently carries cluster, process-instance, incident, process-definition, resource, job, element, Camunda version, and logger dependencies through `NewWithAnalysisDependencies`.
- Backend mechanics for API latency belong under `internal/services/ops`, not `cmd` or `c8volt/ops`: stage planning, measurement loops, worker pools, derived reads, visibility polling, error classification, finding evaluation, ownership, and cleanup.

### Workers And Progress

- `toolx/pool.ExecuteNTimes` and `ExecuteSlice` are the preferred bounded worker primitives. They clamp workers to `[1,n]`, preserve ordered result slots, respect context cancellation, support fail-fast cancellation, and return joined errors.
- Progress facts use the domain/public ops progress models. Command adapters such as `cmd/ops_execute_smoketest_progress.go` translate wording-free service completion facts into semantic activity/progress while protecting JSON, keys-only, quiet, and automation stdout.
- Shared milestone pacing and progress channel policy live in `cmd/ops_progress_*`; API-latency-specific aggregate progress should move to `cmd/ops_api_latency_progress.go` once it has its own lifecycle declarations.

### Reports And Rendering

- `cmd/ops_report.go` is the shared report contract: explicit `markdown`/`json` formats, extension inference, `--report-format` requiring `--report-file`, planning-time existing-file protection, confirmed-mutation overwrite mode, and `0600` file writes.
- Ops renderers live in focused `cmd/cmd_views_ops_*.go` files. Markdown reports are rendered from structured report models; stdout JSON should use the command envelope while raw report JSON remains unwrapped.

### Fixtures, Versions, And Integration

- Version-matched `SimpleUserTask` BPMN fixtures already exist under `embedded/processdefinitions/` for C87, C88, C89, and C810. Use these for active latency instead of adding a feature-specific BPMN.
- Existing version boundaries relevant to this feature: v8.7 process-instance direct lookup is unsupported; full process-definition history deletion requires v8.9 or newer; v8.8 can own created keys but cleanup-enabled active latency must block before mutation unless `--no-cleanup` is explicit.
- Integration family coverage currently lives in `integration/cli/ops_analyse_test.go` and broader real-state/volume suites. Polish tasks must extend existing ops analyse/execute coverage and command inventory rather than adding a separate target.

## Iteration 1 - 2026-09-02 07:02
**Work Unit**: Phase 1 Setup
**Tasks Completed**:
- [x] T001: Create `specs/289-api-latency-diagnostics/progress.md` with the Ralph context path, issue `#289`, branch, validation log, and codebase-pattern sections
- [x] T002: Inspect nearest ops analyse/execute commands, facade/service seams, `toolx/pool`, SimpleUserTask fixtures, version capabilities, progress helpers, shared report helpers, command contracts, and integration suites
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- Durable setup patterns were recorded in progress and compacted into Ralph memory; no production code was changed in this iteration.
## Iteration 2 - 2026-09-02 07:17
**Work Unit**: Phase 2 Foundational
**Tasks Completed**:
- [x] T003: Add failing tests for stage ramps, minimum sample budgets, deterministic allocation, derived limits, nearest-rank percentiles, throughput, zero-baseline deltas, safe classifications, and finding order in `internal/services/ops/api_latency_test.go`
- [x] T004: Add failing facade contract tests for API-latency request/result conversion, defensive slice copying, partial-result mapping, and domain error conversion in `c8volt/ops/client_test.go`
- [x] T005: Define version-neutral request, plan, measurement, stage, finding, topology, ownership, visibility, cleanup, context, and result types with stable enums in `internal/domain/ops_api_latency.go`
- [x] T006: Extend the existing ops service interface with `AnalyseAPILatency` and `ExecuteAPILatencyTest` contracts in `internal/services/ops/api.go`
- [x] T007: Implement deterministic stage planning, normalized-backoff visibility attempt bounding, closed-loop accounting, safe error classification, statistics, comparisons, and finding evaluation in `internal/services/ops/api_latency.go`
- [x] T008: Define the matching public API-latency request/result models and intentional JSON tags in `c8volt/ops/model.go`
- [x] T009: Extend the public ops API with the two API-latency methods in `c8volt/ops/api.go`
- [x] T010: Implement mechanical domain/public conversions with defensive collection copying in `c8volt/ops/convert.go`
- [x] T011: Implement thin facade delegation, partial-result mapping, option propagation, and `ferrors.FromDomain` conversion in `c8volt/ops/client.go`
- [x] T012: Verify the existing `NewWithAnalysisDependencies` construction supplies all required services without new client wiring and add a regression assertion in `c8volt/client_test.go`
- [x] T013: Run `go test ./internal/services/ops ./c8volt/ops -run 'APILatency' -count=1` and record the passing foundational validation
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- `internal/domain/ops_api_latency.go`
- `internal/services/ops/api.go`
- `internal/services/ops/api_latency.go`
- `internal/services/ops/api_latency_test.go`
- `c8volt/ops/api.go`
- `c8volt/ops/client.go`
- `c8volt/ops/model.go`
- `c8volt/ops/convert.go`
- `c8volt/ops/client_test.go`
- `c8volt/client_test.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- API-latency foundation is pure and version-neutral; remote read-only and active workflows can reuse `PlanAPILatency`, `BuildAPILatencyStageResults`, safe classification, and deterministic findings without adding command-layer backend loops.
- Validation passed: `go test ./internal/services/ops ./c8volt/ops -run 'APILatency' -count=1`; `go test ./c8volt -run 'TestNew_V89WiresSupportedRuntime' -count=1`; `git diff --check`.
---
---
## Iteration 3 - 2026-09-02 07:42
**Work Unit**: User Story 1 partial: read-only API latency analysis command
**Tasks Completed**:
- [x] T014: Add service tests for topology/PD/PI measurement, measured-key reuse, Camunda 8.7 PI-keyed-read unavailability, empty/disappearing keys, bounded workers/counts, completed abnormal samples, cancellation, and zero mutation calls
- [x] T015: Add facade tests for read-only progress, result, unavailable measurement, partial error, and collection conversion
- [x] T016: Add command tests for registration, help/examples, defaults, aliases, invalid budgets, keys-only rejection, metadata, tenant behavior, and pre-client validation
- [x] T017: Add command tests for compact human output, one-document JSON, progress modes, stable findings, and read-only limitations
- [x] T018: Implement the read-only closed-loop service workflow using existing APIs and `toolx/pool`
- [x] T019: Add the `ops analyse api-latency` Cobra leaf with local validation, inherited contracts, facade dispatch, and semantic activity setup
- [x] T020: Implement compact read-only stage, finding, notice, limitation, and outcome rendering
- [x] T021: Centralize API-latency aggregate progress routing and JSON/automation/quiet suppression
- [x] T022: Run `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1` and record US1 validation
**Tasks Remaining in Work Unit**: 1 task remains in US1: T023 read-only quickstart scenario evidence
**Commit**: This work-unit commit
**Files Changed**:
- `README.md`
- `c8volt/ops/client_test.go`
- `c8volt/ops/model.go`
- `cmd/cmd_views_ops_api_latency.go`
- `cmd/command_contract_test.go`
- `cmd/ops_analyse_api_latency.go`
- `cmd/ops_analyse_api_latency_test.go`
- `cmd/ops_api_latency_progress.go`
- `docs/cli/c8volt_ops_analyse.md`
- `docs/cli/c8volt_ops_analyse_api-latency.md`
- `docs/cli/command-tree.md`
- `docs/index.md`
- `docsgen/main.go`
- `docsgen/main_test.go`
- `internal/services/ops/api_latency.go`
- `internal/services/ops/api_latency_analysis.go`
- `internal/services/ops/api_latency_test.go`
- `internal/services/ops/orphan_purge_test.go`
- `specs/254-cli-debt-refactor/assessment.md`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- Read-only analysis now completes bounded topology/search/keyed-read stages without mutations; fake command scenarios validate human output, JSON envelope purity, tenant context, protected progress modes, and pre-client invalid-budget rejection.
- Validation passed: `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1`; `go test ./cmd -run 'OpsWorkflowReport|CommandCapability|CapabilityDocument' -count=1`; `go test ./docsgen -count=1`; `make docs-content`; `git diff --check`.
---
---
## Iteration 4 - 2026-09-02 07:47
**Work Unit**: User Story 1: read-only quickstart evidence
**Tasks Completed**:
- [x] T023: Execute the read-only quickstart scenarios and record the zero-mutation and bounded-evidence results
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- Read-only quickstart evidence was validated through fake-server command tests and service bounds tests: invalid `--count 4 --workers 4` exits before remote work, human and JSON executions make only topology/search/keyed-read requests with no deploy/create/cancel/delete calls, JSON stdout remains one envelope, default planning still allocates stages 1/2/4 as 5/6/9, and worker high-water evidence stays within bounds.
- Report-file execution is not claimed by US1 evidence because report writing remains assigned to US4 T048/T052/T053; current read-only report flags are validated only.
- Validation passed: `go test ./cmd -run 'TestOpsAnalyseAPILatency(Default|ReadOnlyCommandRendersHuman|JSONUsesSingleEnvelope|InvalidBudgetSkipsRemote)' -count=1`; `go test ./internal/services/ops -run 'TestAPILatency(PlanDefault|ReadOnlyUsesBoundedStageWorkers|ReadOnlyMeasuresSearchesAndDerivedReads|ReadOnlyReportsUnavailableKeyedEvidence)' -count=1`; `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1`.
---
---
## Iteration 5 - 2026-09-02 07:54
**Work Unit**: User Story 2 partial: active API latency preflight and dry-run planning
**Tasks Completed**:
- [x] T024: Add service tests for run-ID generation, fixture selection, configured/observed version checks, dry-run zero mutation, active 8.7 rejection, cleanup-enabled 8.8 rejection, 8.8 no-cleanup eligibility, and 8.9/8.10 cleanup eligibility
- [x] T029: Implement active preflight, cryptographic run-ID generation, version capability matrix, existing SimpleUserTask fixture selection, and immutable execution planning
**Tasks Remaining in Work Unit**: 9 tasks remain in US2: T025-T028 and T030-T035
**Commit**: This work-unit commit
**Files Changed**:
- `internal/services/ops/api_latency.go`
- `internal/services/ops/api_latency_execute.go`
- `internal/services/ops/api_latency_test.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- Active preflight now returns zero-mutation dry-run plans with 128-bit hex run IDs, version-matched SimpleUserTask fixtures, topology-observed version checks, and explicit cleanup/retention capability gates; non-dry execution remains blocked until the bounded active stages and cleanup tasks are implemented.
- Validation passed: `go test ./internal/services/ops -run 'APILatencyActive' -count=1`; `go test ./internal/services/ops ./c8volt/ops -run 'APILatency' -count=1`; `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1`; `git diff --check`.
---
---
## Iteration 6 - 2026-09-02 08:04
**Work Unit**: User Story 2 partial: active API latency service execution and successful cleanup
**Tasks Completed**:
- [x] T025: Add service tests for exact deployed-key creation, bounded active stages, overlapping-read evidence, exact-key visibility polling, backpressure classification, active findings, and successful-path cleanup
- [x] T030: Implement bounded active stages with `toolx/pool`, exact deployed-definition creation without exporter wait, overlapping read probes, bounded exact-key visibility polling, and actual-limit accounting
- [x] T031: Implement successful-path exact-key process-instance then process-definition cleanup through existing owning services
**Tasks Remaining in Work Unit**: 6 tasks remain in US2: T026-T028 and T032-T035
**Commit**: This work-unit commit
**Files Changed**:
- `internal/services/ops/api_latency_test.go`
- `internal/services/ops/api_latency_execute.go`
- `internal/services/ops/api_latency_cleanup.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- Active non-dry service execution now uses no-wait fixture deployment, exact returned process-definition and process-instance keys, bounded stage workers, exact-key visibility evidence, safe backpressure findings, and PI-before-PD successful cleanup.
- Validation passed: `go test ./internal/services/ops -run 'TestAPILatencyActiveExecution' -count=1`; `go test ./internal/services/ops ./c8volt/ops -run 'APILatency' -count=1`; `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1`; `go test ./internal/services/ops -run 'APILatencyActive|APILatencyPlanDerivesVisibilityBound' -count=1`; `go test ./internal/services/ops -run 'TestAPILatencyActiveExecutionBoundsWorkersAndClassifiesVisibilityErrors' -race -count=1`; `git diff --check`.
---
---
## Iteration 7 - 2026-09-02 08:09
**Work Unit**: User Story 2 partial: active API latency facade conversion
**Tasks Completed**:
- [x] T026: Add facade tests for active plan, ownership, visibility, cleanup, progress, and partial-result conversion
- [x] T032: Complete active-only ownership, visibility, cleanup, and outcome conversions
**Tasks Remaining in Work Unit**: 5 tasks remain in US2: T027, T028, T033, T034, and T035
**Commit**: This work-unit commit
**Files Changed**:
- `c8volt/ops/client_test.go`
- `c8volt/ops/model.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- Active facade tests now pin active plan/fixture/cleanup, ownership, visibility, cleanup, progress, partial-result, and defensive-copy behavior; the only implementation gap found was missing public cleanup status constants for submitted, failed, unknown, and pending states.
- Validation passed: `go test ./c8volt/ops -run 'TestClientExecuteAPILatencyTest' -count=1`; `go test ./internal/services/ops ./c8volt/ops -run 'APILatency' -count=1`; `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1`; `git diff --check`.
---
---
## Iteration 8 - 2026-09-02 08:21
**Work Unit**: User Story 2 partial: active API latency command contract tests
**Tasks Completed**:
- [x] T027: Add command tests for active flags/defaults, concrete-tenant enforcement, state-changing/full/automation metadata, dry-run preview, confirmation including `--no-cleanup`, automation/auto-confirm, JSON confirmation guardrails, and report-path preflight
**Tasks Remaining in Work Unit**: 4 US2 tasks remain: T028 active output tests, T033 remaining command leaf completion, T034 active rendering, and T035 US2 validation
**Commit**: This work-unit commit
**Files Changed**:
- `cmd/ops_execute_api_latency_test.go`
- `cmd/ops_execute_api_latency.go`
- `cmd/command_contract_test.go`
- `internal/services/ops/api_latency_execute.go`
- `docs/cli/c8volt_ops_execute.md`
- `docs/cli/c8volt_ops_execute_api-latency-test.md`
- `docs/cli/command-tree.md`
- `docs/index.md`
- `docsgen/main_test.go`
- `specs/254-cli-debt-refactor/assessment.md`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- Active command tests now pin the CLI safety contract and exposed one service-path correction: process-instance create latency must pass `services.WithNoWait()` internally so creation measurement does not wait for exporter visibility.
- Validation passed: `go test ./cmd -run 'OpsExecuteAPILatency' -count=1`; `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1`; `go test ./cmd -run 'APILatency|OpsWorkflowReport|CommandCapability|CapabilityDocument' -count=1`; `go test ./docsgen -count=1`; `make docs-content`; `git diff --check`.
---
