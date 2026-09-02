# Ralph Progress Log

Feature: 289-api-latency-diagnostics
Started: 2026-09-02 06:58:25

---

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

---
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
---
---
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
