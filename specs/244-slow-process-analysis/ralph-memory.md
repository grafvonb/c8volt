# Ralph Memory

Feature: 244-slow-process-analysis
Started: 2026-07-18T10:01:26Z

## Codebase Patterns
- Ops command files define Cobra flags and local validation in `cmd`, call `NewCli`, require automation support, build a public facade request, delegate, then render through `cmd_views_*`; command metadata is set with `setCommandMutation`, `setContractSupport`, `setAutomationSupport`, and `setOutputModes`.
- Stdin key pipelines are standardized through `validateOptionalDashArg`, `readKeysIfDash`, `mergeAndValidateKeys`, `validateKeys`, and `typex.Keys.Unique()`.
- `c8volt/ops` is a thin facade: `api.go` exposes public methods, `client.go` maps facade options with `options.MapFacadeOptionsToCallOptions`, delegates to `internal/services/ops`, converts results in `convert.go`, and maps errors with `ferrors.FromDomain`.
- `internal/services/ops.Service` owns cross-resource workflow orchestration and dependency fields; existing constructors wire process-instance, incident, process-definition, resource, job, cluster, and version dependencies.
- Process-instance services expose `GetProcessInstances`, tenant-safe lookup via search, and paged search. Element services expose runtime element lookup plus search/page by process instance, element ID, state, type, process definition key, and BPMN process ID.
- Slow-analysis foundation now has version-neutral domain models in `internal/domain/ops_slow_process_analysis.go`, public facade models/converters in `c8volt/ops`, and a read-only `ops analyse/analyze slow-process-instances` command scaffold with full machine-contract metadata.

## Decisions
- No implementation conflicts were found between `spec.md`, `plan.md`, `contracts/cli.md`, and `specs/ralph-implementation-rules.md`.
- Ralph launch instructions already include `--implementation-context specs/ralph-implementation-rules.md` in both `plan.md` and `tasks.md`.
- The foundational service method returns an explicit unsupported error until US1/US2 implement keyed and search selection behavior; this keeps the API compile-safe without claiming analysis success.

## Gotchas
- `cmd/ops.go` sets the root ops command as state-changing; the new slow-analysis child must explicitly set read-only metadata.
- `get pi` has `--incidents-only`, but slow-analysis must not register or advertise that flag; only `--no-incidents-only` is in scope.
- Empty stdin is already rejected by `readKeysIfDash`; reuse this behavior for explicit-key analysis.
- Phase 1 was setup-only inspection. Do not create a commit containing only `tasks.md`, `ralph-memory.md`, and `progress.md`.
- `c8volt.New` now passes `esvc.API` into `opsvc.NewWithAnalysisDependencies`; older ops constructors still allow nil element API for existing tests.
- `ops analyse` is the canonical parent command and `ops analyze` is its alias; capability discovery uses the canonical `ops analyse slow-process-instances` path.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./cmd ./c8volt/ops ./internal/services/ops -run '^$' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/ops ./internal/services/ops -run 'TestOpsAnalyse|TestCommandContractOpsAnalyseSlowProcessInstances|TestClientAnalyseSlowProcessInstances|TestSlowProcessAnalysisFixtures' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/ops ./internal/services/ops -count=1`

## Do Not Repeat
- Do not re-open a separate investigation for the Phase 1 artifact consistency check unless later specs change; the current artifacts are aligned.

## Current Handoff
- Next iteration should start US1 tasks T015-T023. Implement explicit-key analysis using the existing stdin/key helpers, facade delegation, tenant-safe process-instance lookup, unsupported 8.7 behavior, captured analysis time, root duration sorting, and keyed human rendering. Keep backend workflow mechanics in `internal/services/ops`.
