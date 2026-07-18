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
- US2 process-definition search uses the existing process-instance `SearchForProcessInstancesPage` API, freezes unique roots before timing analysis, records `DiscoveredScopeStatus`, and then reuses explicit-key root duration sorting.
- Slow-analysis command date filters normalize RFC3339, c8volt timestamp, and `YYYY-MM-DD` bounds to RFC3339Nano before facade delegation; date-only upper bounds expand to the end of the UTC day.
- Supported process-instance v8.8/v8.9 search adapters now accept RFC3339 bounds in addition to date-only values, keeping ops-normalized search filters compatible with generated request builders.
- US3 timeline analysis uses `pisvc.EnrichProcessInstancesWithElements` after root selection is frozen, calculates element rows and adjacent transition rows from the complete chronological element list, then applies detail filters without creating bridged transitions.
- Slow-analysis human detail rendering prints an `elements:` section under each root, compact element rows with `dur:`/`inc!`, and transition rows in `A -> B: duration` form; keys-only output remains root-only and ignores detail filters.

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
- US1 keyed analysis validates selector combinations in Cobra `Args`, then reads stdin after CLI bootstrap, merges keys through `mergeAndValidateKeys`, deduplicates with `typex.Keys.Unique()`, and delegates a normalized explicit-key request to the ops facade.
- Internal slow-analysis keyed selection uses `pisvc.LookupProcessInstance` for tenant-safe lookup, returns unsupported for Camunda 8.7 before remote lookup, captures one analysis timestamp, measures active roots against that timestamp, treats terminal roots without usable end timestamps as unavailable, and sorts available durations longest-to-shortest with unavailable roots last.
- Human US1 rendering reuses `flatRowPIWithTimezone` for root identity fields and appends `dur:<value>` or `dur:-`; keys-only still emits one root key per line.
- `--batch-size` and `--limit` are process-definition discovery-only flags; explicit-key mode rejects them when explicitly set and never truncates keyed roots.
- `--incidents-only` remains unsupported and unregistered; `--no-incidents-only` maps to `HasIncident=false` in process-instance discovery.
- Runtime element ordering is delegated to existing process-instance enrichment semantics: start date ascending, then element-instance key; keep this when adding JSON or comparison indicators.
- Detail filters are post-calculation visibility filters: element predicates must match all supplied predicates, transitions remain visible when either original endpoint matches the active predicates, and `--duration-after` applies to measured detail row durations only.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./cmd ./c8volt/ops ./internal/services/ops -run '^$' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/ops ./internal/services/ops -run 'TestOpsAnalyse|TestCommandContractOpsAnalyseSlowProcessInstances|TestClientAnalyseSlowProcessInstances|TestSlowProcessAnalysisFixtures' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/ops ./internal/services/ops -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./internal/services/ops -run 'TestSlowProcessAnalysis' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./c8volt/ops -run 'TestClientAnalyseSlowProcessInstances' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestOpsAnalyseSlowProcessInstances|TestRenderOpsSlowProcessAnalysis' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/ops ./internal/services/ops ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1`

## Do Not Repeat
- Do not re-open a separate investigation for the Phase 1 artifact consistency check unless later specs change; the current artifacts are aligned.

## Current Handoff
- Next iteration should start US4 tasks T044-T053 only. Add comparison indicators, JSON payload coverage, keys-only assertions, command/docs metadata, and README examples without changing US3's complete-timeline-before-filtering behavior or root ordering.
