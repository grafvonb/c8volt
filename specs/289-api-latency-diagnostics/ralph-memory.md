# Ralph Memory

Feature: 289-api-latency-diagnostics
Started: 2026-09-02T04:58:25Z

## Codebase Patterns
- First incomplete work now begins at foundational tests T003/T004; setup T001-T002 recorded repository patterns only.
- Commands own Cobra construction, local flag validation, confirmation, activity/progress setup, report-path validation, and final rendering. Follow `cmd/ops_analyse_slow_process_instances.go` for read-only analysis wiring and `cmd/ops_execute_smoketest.go` for active ops execution wiring.
- Public `c8volt/ops` facade methods are thin: map public request to `internal/domain`, delegate to `internal/services/ops.API`, map results back, and convert errors with `ferrors.FromDomain`.
- `internal/services/ops.Service` is the owning layer for stage planning, remote workflow mechanics, worker scheduling, cleanup orchestration, and progress facts. `NewWithAnalysisDependencies` already carries cluster, process-instance, process-definition, resource, job, element, version, and logger dependencies for this feature.
- Reuse `toolx/pool.ExecuteNTimes` or `ExecuteSlice` for bounded concurrent stage work; they clamp worker counts, preserve result order, respect context cancellation, optionally fail fast, and return joined errors.
- Reuse shared ops progress models from `internal/domain/ops_progress.go` and `c8volt/ops/progress_model.go`; command progress adapters must keep JSON, keys-only, quiet, and automation stdout-safe.
- Reuse `cmd/ops_report.go` report behavior exactly: format inference, `--report-format` dependency, planning overwrite protection, confirmed-mutation overwrite mode, and `0600` file writes.
- Existing `embedded/processdefinitions/C87_`, `C88_`, `C89_`, and `C810_` `SimpleUserTask.bpmn` fixtures are available; active latency should select the version-matched SimpleUserTask rather than adding a fixture.
- Version capability boundaries to preserve: v8.7 process-instance direct lookup is unsupported; full process-definition history deletion is currently supported on v8.9 or newer; v8.8 active latency cleanup must be blocked unless `--no-cleanup` is explicit.

## Decisions
- For this Ralph run, the prerequisite script selected `specs/289-api-latency-diagnostics`; the AGENTS Speckit block still names an older active plan and should not override the checked `FEATURE_DIR`.

## Gotchas
- Task file requires commit subjects for this feature to be Conventional Commits ending in `#289`; the current branch prefix also supports issue auto-inference.
- Avoid command-local worker pools, page traversal, or backend loops. Those are completion-blocking layering shortcuts under `specs/ralph-implementation-rules.md`.
- A command progress helper crossing three mode/lifecycle declarations needs a focused file such as `cmd/ops_api_latency_progress.go`.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./internal/services/ops ./c8volt/ops -run 'APILatency' -count=1`
- `go test ./cmd -run 'APILatency|OpsWorkflowReport|CommandCapability|CapabilityDocument' -count=1`
- `git diff --check`

## Do Not Repeat
- Do not create a new report framework, worker framework, fixture, generated client, or versioned latency adapter for this feature.
- Do not hand-edit generated CLI docs under `docs/cli`; update command metadata and run `make docs-content` when command behavior exists.

## Current Handoff
- Next iteration should start at T003/T004 in Phase 2. Add failing foundational tests first, beginning with `internal/services/ops/api_latency_test.go` for deterministic planning/statistics/classification and `c8volt/ops/client_test.go` for facade conversion/error contracts; do not start US1 yet.
