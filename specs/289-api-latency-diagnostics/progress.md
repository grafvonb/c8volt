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
---
## Iteration 9 - 2026-09-02 08:28
**Work Unit**: User Story 2: active API latency command output and validation
**Tasks Completed**:
- [x] T028: Add command tests for compact active preview/result output, plan/execution count parity, run identity, fixture, visibility, findings, successful cleanup, and no low-level per-key chatter
- [x] T033: Validate the `ops execute api-latency-test` Cobra leaf with preview, confirmation, JSON guardrails, facade dispatch, tenant context, and shared report-path planning
- [x] T034: Extend API-latency views with active preview, write/read/visibility stages, ownership summary, successful cleanup, and outcome rendering
- [x] T035: Run focused US2 API-latency validation and record the passing result
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- `cmd/ops_execute_api_latency_test.go`
- `cmd/cmd_views_ops_api_latency.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- Active human output now uses active-specific labels and sample math, exposes bounded plan/run/fixture/visibility/cleanup summaries, and keeps endpoint/per-key lifecycle chatter out of compact output.
- Validation passed: `go test ./cmd -run 'TestOpsExecuteAPILatencyDryRunRendersActivePreview|TestRenderOpsExecuteAPILatencyHumanRendersActiveResult' -count=1`; `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1`; `go test ./cmd -run 'APILatency|OpsWorkflowReport|CommandCapability|CapabilityDocument' -count=1`; `git diff --check`.
---
---
## Iteration 10 - 2026-09-02 08:35
**Work Unit**: User Story 3 partial: exact-key ownership registry and cleanup ordering
**Tasks Completed**:
- [x] T036: Add service tests for concurrency-safe immediate ownership registration, stable key ordering/deduplication, exact-key-only cleanup authority, PI-before-PD order, and absence of BPMN-ID or tenant-wide cleanup discovery
- [x] T040: Implement the concurrency-safe exact-key ownership registry and terminal cleanup-record accounting
**Tasks Remaining in Work Unit**: 9 US3 tasks remain: T037-T039 and T041-T046
**Commit**: This work-unit commit
**Files Changed**:
- `internal/services/ops/api_latency_cleanup.go`
- `internal/services/ops/api_latency_execute.go`
- `internal/services/ops/api_latency_test.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- Active ownership registration is now service-owned through a mutex-protected registry that immediately records returned keys, deduplicates them, snapshots deterministic exact-key evidence, and keeps cleanup PI-before-PD without BPMN-ID or tenant-wide cleanup discovery.
- Validation passed: `go test ./internal/services/ops -run 'TestAPILatencyActiveOwnershipRegistryDeduplicatesAndCleansExactKeys' -count=1`; `go test ./internal/services/ops -run 'TestAPILatencyActiveOwnershipRegistryDeduplicatesAndCleansExactKeys' -race -count=1`; `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1`; `go test ./internal/services/ops -run 'APILatencyActive' -race -count=1`; `git diff --check`.
---
---
## Iteration 11 - 2026-09-02 08:44
**Work Unit**: User Story 3 partial: cleanup after incomplete active runs
**Tasks Completed**:
- [x] T037: Add service tests for cleanup after stage failure, request timeout, visibility exhaustion, canceled caller, independent cleanup timeout, partial cleanup, recovery commands, 8.8 retention, and no-cleanup distinction
- [x] T041: Extend cleanup orchestration with `context.WithoutCancel`, a bounded completion context, exact PI-before-PD deletion, and joined partial errors
- [x] T042: Implement intentional retention, remaining-resource classification, and exact-key manual recovery guidance
**Tasks Remaining in Work Unit**: 6 US3 tasks remain: T038, T039, and T043-T046
**Commit**: This work-unit commit
**Files Changed**:
- `internal/services/ops/api_latency.go`
- `internal/services/ops/api_latency_cleanup.go`
- `internal/services/ops/api_latency_test.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- Active cleanup now receives an independent bounded context after caller cancellation, records timeout remainders as `unknown`, joins cleanup errors with execution errors, and omits unsupported process-definition recovery commands for explicit Camunda 8.8 retention.
- Validation passed: `go test ./internal/services/ops -run 'APILatencyActiveCleanup|APILatencyActiveTimeoutAndVisibility|APILatencyActiveNoCleanup' -count=1`; `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1`; `go test ./internal/services/ops -run 'APILatencyActive' -race -count=1`; `git diff --check`.
---
---
## Iteration 12 - 2026-09-02 08:48
**Work Unit**: User Story 3 partial: facade preservation of partial cleanup evidence
**Tasks Completed**:
- [x] T038: Add facade tests proving partial ownership/cleanup results survive domain error conversion
**Tasks Remaining in Work Unit**: 5 US3 tasks remain: T039 and T043-T046
**Commit**: This work-unit commit
**Files Changed**:
- `c8volt/ops/client_test.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- Active facade regression coverage now proves timeout-normalized partial execution still returns exact ownership plus deleted, unknown, and failed cleanup records with safe classifications and manual recovery commands.
- Validation passed: `go test ./c8volt/ops -run 'TestClientExecuteAPILatencyTestPreservesPartialCleanupEvidence' -count=1`; `go test ./c8volt/ops -run 'APILatency' -count=1`; `go test ./internal/services/ops ./c8volt/ops -run 'APILatency' -count=1`; `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1`; `git diff --check`.
---
---
## Iteration 13 - 2026-09-02 09:02
**Work Unit**: User Story 3 partial: active command cancellation and cleanup recovery output
**Tasks Completed**:
- [x] T039: Add command tests for scoped interrupt cancellation, retained cleanup output, exact recovery guidance, established nonzero error envelope, and no-cleanup confirmation coverage
- [x] T043: Install scoped signal-aware cancellation for the active execution window
- [x] T045: Render retained resources, cleanup attempts/results, remaining exact keys, recovery guidance, and partial/interrupted outcomes compactly
**Tasks Remaining in Work Unit**: 2 US3 tasks remain: T044 service/facade outcome preservation and T046 final race validation record
**Commit**: This work-unit commit
**Files Changed**:
- `cmd/ops_execute_api_latency_test.go`
- `cmd/ops_execute_api_latency.go`
- `cmd/ops_execute_api_latency_signal.go`
- `cmd/cmd_views_ops_api_latency.go`
- `internal/services/ops/api_latency_test.go`
- `internal/services/ops/smoke_test_test.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- Active command cancellation is now scoped to the mutation window and exact cleanup recovery output remains compact by listing only retained, failed, and unknown resources.
- The race gate exposed a shared test-stub race in `stubSmokeTestClusterAPI.topologyCalls`; making that counter atomic allowed `go test ./internal/services/ops ./cmd -run 'APILatency' -race -count=1` to pass.
- Validation passed: `go test ./cmd -run 'TestOpsExecuteAPILatency(InterruptContextScopesActiveExecution|CleanupFailureUsesJSONErrorEnvelope)|TestRenderOpsExecuteAPILatencyHumanRendersRetainedAndRecoveryResources' -count=1`; `go test ./cmd -run 'APILatency|OpsWorkflowReport|CommandCapability|CapabilityDocument' -count=1`; `go test ./internal/services/ops ./cmd -run 'APILatency' -race -count=1`; `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1`; `git diff --check`.
---
---
## Iteration 14 - 2026-09-02 09:09
**Work Unit**: User Story 3: active terminal outcome and evidence preservation
**Tasks Completed**:
- [x] T044: Preserve interrupted, partial, failed, and completed-retained outcomes while returning available service/facade evidence
- [x] T046: Run cancellation/cleanup race-focused tests and record US3 validation
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- `c8volt/ops/client_test.go`
- `internal/services/ops/api_latency_execute.go`
- `internal/services/ops/api_latency_test.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- Deploy call errors were being classified in setup measurements but not returned from `deployAPILatencyFixture`; active execution could continue into create calls after a failed deploy.
- Failed deploy responses that still include an exact process-definition key now preserve ownership and attempt exact cleanup while returning a failed outcome and the original deploy error.
- Validation passed: `go test ./internal/services/ops -run 'TestAPILatencyActiveDeployErrorWithReturnedKeyCleansExactDefinition' -count=1`; `go test ./c8volt/ops -run 'TestClientExecuteAPILatencyTestPreservesTerminalOutcomeEvidence' -count=1`; `go test ./internal/services/ops ./c8volt/ops -run 'APILatency' -count=1`; `go test ./internal/services/ops ./cmd -run 'APILatency' -race -count=1`; `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1`; `git diff --check`.
---
---
## Iteration 15 - 2026-09-02 09:14
**Work Unit**: User Story 4 partial: API latency renderer regression coverage
**Tasks Completed**:
- [x] T047: Add human and JSON renderer tests for stable schema/context, ordered stages/classifications/findings, read-only active-field omission, active evidence fields, compact wording, and the five-second render budget
**Tasks Remaining in Work Unit**: 9 US4 tasks remain: T048-T056
**Commit**: This work-unit commit
**Files Changed**:
- `cmd/ops_analyse_api_latency_test.go`
- `cmd/ops_execute_api_latency_test.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- Existing `renderOpsAPILatencyResult` behavior already satisfied the new stable read-only/active human and JSON renderer assertions; US4 should continue with report behavior at T048.
- Validation passed: `go test ./cmd -run 'TestRenderOps(Analyse|Execute)APILatencyStableHumanAndJSON' -count=1`; `go test ./cmd -run 'APILatency' -count=1`; `git diff --check`.
---
---
## Iteration 16 - 2026-09-02 09:28
**Work Unit**: User Story 4 partial: API latency report tests and shared report writing
**Tasks Completed**:
- [x] T048: Add report tests for Markdown/JSON inference, explicit override, dependent flags, `0600` files, missing parents, preserve/overwrite policy, raw JSON payload, Markdown parity, report-written line, and partial-result preservation
- [x] T052: Implement command-specific Markdown and raw JSON report rendering by reusing `cmd/ops_report.go` and API-latency view helpers
- [x] T053: Wire shared report flag/path/write ordering, format inference, partial report attempts, and active mutation write mode into both API-latency commands
**Tasks Remaining in Work Unit**: 6 US4 tasks remain: T049-T051 and T054-T056
**Commit**: This work-unit commit
**Files Changed**:
- `c8volt/ops/model.go`
- `cmd/cmd_views_ops_api_latency.go`
- `cmd/ops_analyse_api_latency.go`
- `cmd/ops_analyse_api_latency_test.go`
- `cmd/ops_execute_api_latency.go`
- `cmd/ops_execute_api_latency_test.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- API-latency reports can use the public result as the raw JSON report payload while Markdown renders compact parity from the same model; active report overwrite must be based on returned exact ownership evidence.
- Validation passed: `go test ./cmd -run 'TestOps(Analyse|Execute)APILatency.*Report|TestOpsExecuteAPILatencyDefaultsAndValidation' -count=1`; `go test ./cmd -run 'APILatency' -count=1`; `go test ./cmd -run 'APILatency|OpsWorkflowReport|CommandCapability|CapabilityDocument' -count=1`; `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1`; `git diff --check`.
---
---
## Iteration 17 - 2026-09-02 09:37
**Work Unit**: User Story 4 partial: API latency output safety and progress coverage
**Tasks Completed**:
- [x] T049: Add output-safety and progress tests covering tokens, authorization headers, secrets, variables, payloads, raw response bodies, unbounded errors, JSON/automation silence, quiet failures, and verbose/debug detail
**Tasks Remaining in Work Unit**: 5 US4 tasks remain: T050, T051, and T054-T056
**Commit**: This work-unit commit
**Files Changed**:
- `cmd/ops_analyse_api_latency_test.go`
- `cmd/ops_execute_api_latency_test.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- API-latency command tests now prove OAuth token/header/client-secret material, ignored process-variable/business-payload fields, raw upstream bodies, and long upstream detail strings do not enter completed command output or reports; active JSON automation stays silent on stderr and verbose/debug aggregate progress remains stderr-only.
- Validation passed: `go test ./cmd -run 'TestOps(Analyse|Execute)APILatency(OutputSafety|JSONAutomationOutputSafety|ProgressModeGate|QuietFailure)' -count=1`; `go test ./cmd -run 'APILatency' -count=1`; `go test ./cmd -run 'APILatency|OpsWorkflowReport|CommandCapability|CapabilityDocument' -count=1`; `git diff --check`.
---
---
## Iteration 18 - 2026-09-02 09:52
**Work Unit**: User Story 4 partial: API latency subprocess exit-envelope coverage
**Tasks Completed**:
- [x] T050: Add subprocess tests proving completed abnormal evidence exits successfully while invalid, incomplete, report-failed, and requested-cleanup-failed runs use the established nonzero error envelope
**Tasks Remaining in Work Unit**: 4 US4 tasks remain: T051, T054, T055, and T056
**Commit**: This work-unit commit
**Files Changed**:
- `cmd/ops_analyse_api_latency_test.go`
- `cmd/ops_execute_api_latency_test.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- API-latency subprocess coverage now pins completed-abnormal success, invalid local input, canceled/incomplete read-only execution, report write failure, and requested active cleanup failure through real process exit behavior and shared JSON envelopes.
- Validation passed: `go test ./cmd -run 'TestOps(AnalyseAPILatency(CompletedAbnormalSubprocessExitsSuccessfully|InvalidSubprocessUsesJSONErrorEnvelope|IncompleteSubprocessUsesJSONErrorEnvelope|ReportFailureSubprocessUsesJSONErrorEnvelope)|ExecuteAPILatencyRequestedCleanupFailureSubprocessUsesJSONErrorEnvelope)$' -count=1`; `go test ./cmd -run 'APILatency|OpsWorkflowReport|CommandCapability|CapabilityDocument' -count=1`; `git diff --check`.
---
---
## Iteration 19 - 2026-09-02 09:57
**Work Unit**: User Story 4 partial: shared API latency renderer completion
**Tasks Completed**:
- [x] T051: Complete shared compact human and stable command-envelope JSON rendering from the one API-latency result model
**Tasks Remaining in Work Unit**: 3 US4 tasks remain: T054 safe context/limitations/notices, T055 exit mapping, and T056 US4 validation
**Commit**: This work-unit commit
**Files Changed**:
- `cmd/cmd_views_ops_api_latency.go`
- `cmd/ops_analyse_api_latency_test.go`
- `cmd/ops_execute_api_latency_test.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- Compact API-latency stage rows now show prior-stage p50 and throughput deltas from the shared result model, while JSON rendering is pinned to the existing shared success envelope for both command leaves.
- Validation passed: `go test ./cmd -run 'TestRenderOps(Analyse|Execute)APILatencyStableHumanAndJSON' -count=1`; `go test ./cmd -run 'APILatency|OpsWorkflowReport|CommandCapability|CapabilityDocument' -count=1`; `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1`; `git diff --check`.
---
---
## Iteration 20 - 2026-09-02 10:03
**Work Unit**: User Story 4 partial: safe API latency context, notices, and limitations
**Tasks Completed**:
- [x] T054: Attach safe build/profile/tenant/version context and fixed limitations/notices to both API-latency command results and views
**Tasks Remaining in Work Unit**: 2 US4 tasks remain: T055 exit mapping and T056 US4 validation
**Commit**: This work-unit commit
**Files Changed**:
- `cmd/ops_analyse_api_latency.go`
- `cmd/ops_execute_api_latency.go`
- `cmd/cmd_views_ops_api_latency.go`
- `cmd/ops_analyse_api_latency_test.go`
- `cmd/ops_execute_api_latency_test.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- API-latency commands now use one shared enrichment hook to backfill safe schema/build/profile/tenant/version context and carry plan notices/limitations into the final stdout/report payload without duplication.
- Validation passed: `go test ./cmd -run 'TestAttachOpsAPILatencyResultContextPreservesSafeEvidence|TestRenderOps(Analyse|Execute)APILatencyStableHumanAndJSON' -count=1`; `go test ./cmd -run 'APILatency|OpsWorkflowReport|CommandCapability|CapabilityDocument' -count=1`; `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1`; `git diff --check`.
---
---
## Iteration 21 - 2026-09-02 10:08
**Work Unit**: User Story 4: API latency exit mapping and validation
**Tasks Completed**:
- [x] T055: Map completed-abnormal results to success and incomplete/report/cleanup failures to existing command errors without a new partial stdout envelope
- [x] T056: Run the focused US4 command validation and record results
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- `cmd/ops_api_latency_exit.go`
- `cmd/ops_analyse_api_latency.go`
- `cmd/ops_execute_api_latency.go`
- `cmd/ops_analyse_api_latency_test.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- API-latency command exits now key off terminal result outcomes: planned/completed/completed-retained render as successes, and incomplete outcomes continue through the established command error envelope with reports written first when requested.
- Validation passed: `go test ./cmd -run 'TestOpsAPILatencyCommandErrorMapsOutcomes|TestOpsAnalyseAPILatency(CompletedAbnormalSubprocessExitsSuccessfully|IncompleteSubprocessUsesJSONErrorEnvelope|ReportFailureSubprocessUsesJSONErrorEnvelope)|TestOpsExecuteAPILatencyRequestedCleanupFailureSubprocessUsesJSONErrorEnvelope' -count=1`; `go test ./cmd -run 'APILatency|OpsWorkflowReport|CommandCapability|CapabilityDocument' -count=1`; `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1`; `git diff --check`.
---
---
## Iteration 22 - 2026-09-02 10:15
**Work Unit**: Phase 7 polish: command capability and inventory coverage
**Tasks Completed**:
- [x] T057: Update command capability/family assertions and the expected inventory from 55 to 57
**Tasks Remaining in Work Unit**: 8 Phase 7 tasks remain: T058-T065
**Commit**: This work-unit commit
**Files Changed**:
- `cmd/capabilities_test.go`
- `cmd/command_contract_test.go`
- `integration/cli/all_commands_test.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- API-latency command discovery is now pinned in command capability tests, ops family assertions, all-tenants metadata, and the integration command coverage manifest at an inventory count of 57.
- Validation passed: `go test ./cmd -run 'TestCommandCapabilityForCommand_Ops(Analyse|Execute)APILatencyContract|TestCapabilitiesCommand_JSONIncludes(AllTenantsSupport|OpsRootMetadata)|TestAllTenantsSupportForCommand_ConcreteDestinationInventory|TestCommandCapabilityForCommand_IncludesAllTenantsSupport|TestCapabilityDocumentForRoot_CoversCLIDebtAssessment' -count=1`; `go test -tags integration ./integration/cli -run 'TestCommandInventory' -count=1`; `go test ./cmd -run 'CommandCapability|CapabilityDocument|CapabilitiesCommand' -count=1`; `git diff --check`.
---
---
## Iteration 23 - 2026-09-02 10:22
**Work Unit**: Phase 7 polish: read-only API latency volume coverage
**Tasks Completed**:
- [x] T058: Extend read-only human, JSON, report, bounds, and seeded dirty-state coverage
**Tasks Remaining in Work Unit**: 7 Phase 7 tasks remain: T059-T065
**Commit**: This work-unit commit
**Files Changed**:
- `integration/cli/volume_ops_analyse_test.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- Read-only API-latency volume coverage now runs against seeded dirty-state data, validates compact human output plus Markdown reporting, and validates JSON/report parity with deterministic stage and derived-request bounds.
- Validation passed: `go test -tags integration ./integration/cli -run 'TestVolumeOpsAnalyseFamily|TestVolumeOwnershipClassification|TestProposal' -count=1`; `go test -tags integration ./integration/cli -run 'TestVolumeOpsAnalyseFamily|TestCommandInventory' -count=1`; `git diff --check`.
---
---
## Iteration 24 - 2026-09-02 10:46
**Work Unit**: Phase 7 polish: active API latency volume coverage
**Tasks Completed**:
- [x] T059: Extend active dry-run selected-version coverage and disposable 8.9/8.10 confirmed-cleanup evidence
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- `integration/cli/volume_ops_execute_test.go`
- `internal/services/calloption.go`
- `internal/services/ops/api_latency.go`
- `internal/services/ops/api_latency_cleanup.go`
- `internal/services/ops/api_latency_test.go`
- `internal/services/processinstance/v89/service.go`
- `internal/services/processinstance/v89/service_test.go`
- `internal/services/processinstance/v810/service.go`
- `internal/services/processinstance/v810/service_test.go`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- Active API-latency integration now proves selected-version dry-run planning and cleanup-capable confirmed execution with exact ownership, visibility, and JSON/report parity.
- Live C89 cleanup exposed exporter-lag-sensitive descendant prechecks, active-history delete conflicts, and short absent-wait exhaustion; API-latency cleanup now submits exact no-wait PI deletes, cancels exact active fixture instances on conflict, and retries exact PD deletion on transient conflicts.
- Validation passed: `go test ./internal/services/ops -run 'TestAPILatencyActive(PlanDerivesVisibilityBound|ExecutionUsesExactReturnedKeysAndCleansUp|InstanceCleanupCancelsActiveHistoryConflict|DefinitionCleanupRetriesConflict)' -count=1`; `go test ./internal/services/processinstance/v89 ./internal/services/processinstance/v810 -run 'TestService_CancelAndDeleteProcessInstance/ExactDeleteBypassesDescendantLookup' -count=1`; `go test ./internal/services/ops ./c8volt/ops ./cmd -run 'APILatency' -count=1`; `go test ./internal/services/ops ./cmd -run 'APILatency' -race -count=1`; `go test -tags integration ./integration/cli -run 'TestVolumeOpsExecuteFamily|TestCommandInventory' -count=1`; `git diff --check`.
---
---
## Iteration 25 - 2026-09-02 10:51
**Work Unit**: Phase 7 polish: API latency README and ops-index discoverability
**Tasks Completed**:
- [x] T060: Add command discoverability, safety distinction, and representative examples to `README.md` and `docs/ops/index.md`
**Tasks Remaining in Work Unit**: 5 Phase 7 tasks remain: T061-T065
**Commit**: This work-unit commit
**Files Changed**:
- `README.md`
- `docs/ops/index.md`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- API-latency docs discoverability now distinguishes read-only zero-mutation analysis from confirmed active testing with cleanup; focused operator guide pages remain the next docs task.
- Validation passed: `git diff --check`; generated reference targets for both API-latency commands exist.
---
---
## Iteration 26 - 2026-09-02 10:56
**Work Unit**: Phase 7 polish: API latency operator guidance
**Tasks Completed**:
- [x] T061: Add focused operator guidance in `docs/ops/analyse-api-latency.md` and `docs/ops/execute-api-latency-test.md`
**Tasks Remaining in Work Unit**: 4 Phase 7 tasks remain: T062-T065
**Commit**: This work-unit commit
**Files Changed**:
- `docs/ops/analyse-api-latency.md`
- `docs/ops/execute-api-latency-test.md`
- `docs/ops/index.md`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- API-latency operator guidance now has focused read-only and active playbooks covering safety, stage bounds, report behavior, active version gates, confirmation, cleanup, retention, and recovery without hand-editing generated CLI docs.
- Validation passed: `git diff --check`.
---
---
## Iteration 27 - 2026-09-02 10:59
**Work Unit**: Phase 7 polish: generated API latency CLI documentation
**Tasks Completed**:
- [x] T062: Run `make docs-content` and review generated API-latency CLI pages and docs index without hand-editing generated content
**Tasks Remaining in Work Unit**: 3 Phase 7 tasks remain: T063-T065
**Commit**: This work-unit commit
**Files Changed**:
- `docs/index.md`
- `specs/289-api-latency-diagnostics/tasks.md`
- `specs/289-api-latency-diagnostics/ralph-memory.md`
- `specs/289-api-latency-diagnostics/progress.md`
**Learnings**:
- `make docs-content` left the generated API-latency leaf pages unchanged and refreshed `docs/index.md` with API-latency discovery, generated-reference links, command count 57, and active all-tenants wording.
- Validation passed: `make docs-content`; reviewed `docs/cli/c8volt_ops_analyse_api-latency.md`, `docs/cli/c8volt_ops_execute_api-latency-test.md`, and `docs/index.md`; `git diff --check`.
---
