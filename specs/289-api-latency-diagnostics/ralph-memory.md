# Ralph Memory

Feature: 289-api-latency-diagnostics
Started: 2026-09-02T04:58:25Z

## Codebase Patterns
- First incomplete work now begins at US3 T038; foundational API-latency domain/service/facade contracts T003-T013, read-only analyse implementation/evidence T014-T023, bounded active US2 command/service/facade/rendering work T024-T035, and US3 service cleanup/ownership work T036/T037/T040-T042 are complete.
- Commands own Cobra construction, local flag validation, confirmation, activity/progress setup, report-path validation, and final rendering. Follow `cmd/ops_analyse_slow_process_instances.go` for read-only analysis wiring and `cmd/ops_execute_smoketest.go` for active ops execution wiring.
- Public `c8volt/ops` facade methods are thin: map public request to `internal/domain`, delegate to `internal/services/ops.API`, map results back, and convert errors with `ferrors.FromDomain`.
- `internal/services/ops.Service` is the owning layer for stage planning, remote workflow mechanics, worker scheduling, cleanup orchestration, and progress facts. `NewWithAnalysisDependencies` already carries cluster, process-instance, process-definition, resource, job, element, version, and logger dependencies for this feature.
- Reuse `toolx/pool.ExecuteNTimes` or `ExecuteSlice` for bounded concurrent stage work; they clamp worker counts, preserve result order, respect context cancellation, optionally fail fast, and return joined errors.
- Reuse shared ops progress models from `internal/domain/ops_progress.go` and `c8volt/ops/progress_model.go`; command progress adapters must keep JSON, keys-only, quiet, and automation stdout-safe.
- Reuse `cmd/ops_report.go` report behavior exactly: format inference, `--report-format` dependency, planning overwrite protection, confirmed-mutation overwrite mode, and `0600` file writes.
- Existing `embedded/processdefinitions/C87_`, `C88_`, `C89_`, and `C810_` `SimpleUserTask.bpmn` fixtures are available; active latency should select the version-matched SimpleUserTask rather than adding a fixture.
- Version capability boundaries to preserve: v8.7 process-instance direct lookup is unsupported; full process-definition history deletion is currently supported on v8.9 or newer; v8.8 active latency cleanup must be blocked unless `--no-cleanup` is explicit.
- Foundational API-latency types live in `internal/domain/ops_api_latency.go`; public mirrors live in `c8volt/ops/model.go` and conversions in `c8volt/ops/convert.go`.
- `internal/services/ops/api_latency.go` now owns deterministic stage planning, read-only/active derived bounds, normalized-backoff visibility-attempt bounding, safe error classification, aggregate statistics, zero-baseline-safe comparisons, and deterministic finding ordering.
- `internal/services/ops/api_latency_analysis.go` owns the read-only remote workflow: topology, process-definition search, and process-instance search are primary sample calls; process-definition and process-instance keyed reads are derived calls using keys from the measured searches when available and supported.
- Read-only API-latency measurements classify per-call errors into result evidence instead of failing completed runs. Caller cancellation returns an interrupted/partial result plus the context error; unavailable keys and unsupported v8.7 process-instance keyed reads produce unavailable derived measurements.
- `cmd/ops_analyse_api_latency.go`, `cmd/cmd_views_ops_api_latency.go`, and `cmd/ops_api_latency_progress.go` implement the read-only command leaf, compact human/JSON rendering, command metadata, local validation, semantic activity, and aggregate progress-mode suppression.
- `internal/services/ops/api_latency_execute.go` now owns active API-latency preflight and immutable planning: 128-bit hex run IDs, version-matched SimpleUserTask fixture selection, topology connectivity/observed-version checks, version capability gating, zero-mutation dry-run results, and cleanup/retention block reasons.
- Active non-dry execution now deploys the selected SimpleUserTask fixture via `resourceAPI.Deploy` with `services.WithNoWait()`, creates process instances with `services.WithNoWait()`, extracts only the returned process-definition key, runs bounded closed-loop create/read/visibility stages through `toolx/pool.ExecuteNTimes`, records returned process-instance keys immediately, and classifies visibility/backpressure evidence into stage findings.
- `internal/services/ops/api_latency_cleanup.go` owns active cleanup records and the concurrency-safe ownership registry. Returned process-instance keys are recorded immediately, deduplicated, and snapshotted in deterministic sorted order; cleanup/retention consumes only these exact keys and deletes process instances before the exact process-definition key.
- Active cleanup now uses a bounded context derived with `context.WithoutCancel` so cleanup can continue after caller cancellation; the planned independent budget defaults to `poller.DefaultCompletionTimeout`, cleanup timeout leaves exact-key `unknown` records with recovery guidance, and non-timeout delete failures leave `failed` records with safe classifications.
- Explicit `--no-cleanup` retention records exact PI/PD resources as `retained`; PI recovery commands are safe, while PD recovery commands are omitted when the active version lacks complete process-definition-history cleanup support such as Camunda 8.8.
- Active facade coverage in `c8volt/ops/client_test.go` now proves active plan/fixture/cleanup plan fields, ownership keys, visibility durations/classifications, cleanup records/recovery commands, progress callback mapping, partial-result error conversion, and defensive copying across the public boundary. Public cleanup statuses in `c8volt/ops/model.go` now mirror all domain statuses (`pending`, `submitted`, `deleted`, `retained`, `failed`, `unknown`).
- `cmd/ops_execute_api_latency.go` now registers `ops execute api-latency-test` with active flags, state-changing/full/automation metadata, concrete-destination all-tenants rejection, local count/worker/keys-only/report validation, JSON mutation guardrails, active facade dispatch, aggregate progress routing, default low-level log suppression, and confirmation including `--no-cleanup`. Report-file path preflight is implemented, but actual report writing remains assigned to US4.
- Active command output now branches in `cmd/cmd_views_ops_api_latency.go`: human preview/result labels the active command, reports plan allocation parity, run ID, fixture, visibility attempt ceiling, cleanup intent/support, active primary create counts, ownership summary, visibility summary, cleanup summary, findings, and outcome without endpoint or per-key lifecycle chatter.

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
- `--report-file`/`--report-format` are currently validated on the read-only command, but actual report writing remains planned in US4 T048/T052/T053; T023 evidence intentionally covered executable read-only/default-shape, JSON stdout, invalid-budget, zero-mutation, and bounded-worker behavior without claiming report-file output is implemented.
- Facade and command preservation/rendering of partial cleanup evidence remains assigned to US3 T038/T039/T043-T046; do not treat the service-only cleanup work as completing user-visible recovery output.

## Current Handoff
- Next iteration should continue US3 at T038 by adding facade tests in `c8volt/ops/client_test.go` proving partial ownership and cleanup results survive domain error conversion.
