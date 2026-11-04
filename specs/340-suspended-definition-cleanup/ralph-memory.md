# Ralph Memory

Feature: 340-suspended-definition-cleanup
Started: 2026-10-23T08:10:21Z

## Codebase Patterns

- T005 keeps suspended value/support in the owned display projection, changed-field flags, ordinary flat rows and watch formatter in `cmd/cmd_views_processdefinition.go`. Unsupported values normalize to no token; supported zero uses `su:-`. Existing watch lifecycle/baseline commit rules are unchanged. `cmd/get_processdefinition_suspended_test.go` supplies exact-total HTTP fixtures reused by actual terminal watch/empty-result tests.

- `cmd/ops_purge_all_processdefinitions_unfinished_test.go` reuses the definition-delete terminal runner, now selecting either command. All fixtures are test-owned HTTP servers. The matrix checks configured/inherited destinations, plain/JSON log formats, severity/cardinality, normal/verbose/quiet, DEBUG flag/configuration, JSON EOF/partial results, and exit codes. Sparse union discovery, late unfinished evidence, mandatory drain under no-wait, and actual terminal confirmation/abort are exercised through purge.

- Definition-delete presentation now lives in `cmd/cmd_views_resource.go`; command construction/dispatch remains in `cmd/delete_processdefinition.go`, and lifecycle/progress remains in its existing focused progress file. No new command mode or mutation engine was added.
- `cmd/delete_processdefinition_unfinished_test.go` uses the existing terminal runner and HTTP fake conventions. Its configured/inherited writer matrix parses every plain/structured log record, checks error severity/cardinality and JSON EOF, and proves raw-error/direct-stderr regressions fail. The sparse suspended-force fixture exercises the full CLI → facade → v810 path without live services.

- v810 definition statistics are acquired in `internal/services/processdefinition/v810/service.go` via exact-total requests (limit 1), then capped-total traversal when necessary. New suspended and union requests reuse definition/tenant scope, ordering, and paging; the union uses the generated advanced state filter's `$in`.
- Public process statistics and ops discovery/report statistics have separate mechanical converters. Existing public active/completed/canceled/incidents JSON fields retain zero values; the new suspended fields use `omitempty`. Domain statistics omit zero values and never serialize `Unfinished`.
- Adjacent versioned tests use strict generated-client mocks. Extend `expectedProcessDefinitionStats` and `mockProcessDefinitionStats` for v810; older mocks must receive no new queries.
- Internal `ProcessInstanceFilter.States` is normalized through `common.NormalizeProcessInstanceSearchFilter` at both shared search helpers and each adapter entry point. It copies/deduplicates uppercase values, maps CANCELED to TERMINATED, rejects ambiguous or invalid plural input, and uses the existing pure SUSPENDED capability check to require v810 even for a one-value plural selector. Scalar behavior stays unchanged. Direct incident-index filtering also honors the plural selector.
- v810 search decoding retains whether `totalItems` was present and nonnegative; missing/null totals are not exact zero. The page mapper preserves advancing sparse continuation, distinguishes capped lower bounds from exact totals, and leaves unresolved metadata indeterminate.

## Decisions

- T005 is complete. Human lookup/list/watch rows append `su:` only when supported; active remains separate. Watch detects suspended-only changes, ACTIVE-to-SUSPENDED transitions, zero/support transitions and removed statistics, preserves alignment/selective bold and commits only fully displayed baselines. The definition view now suppresses empty human summaries under quiet, matching the mandatory empty-result rule and adjacent instance view; explicit JSON/keys output keeps its existing precedence. No new timing, retry, query, lifecycle or schema behavior was introduced.

- T004 is complete. Purge reuses T003's definition workflow without new discovery/cancellation mechanics. Aggregate suspended/support/unfinished fields map mechanically through the facade and report, preserving schema v1. Known family/unfinished keys are deduplicated before fallback counting; direct unfinished counts remain distinct from expanded family scope.
- Purge failures now render partial results in exactly one failure envelope and exit with the original error class. Human failures use one bounded aggregate through the configured logger, with a useful force blocker or confirmation/report failure reason. Progress counts failed completions but emits no per-key failure diagnostic; existing DEBUG transport logging/redaction stays authoritative. Report artifacts and machine errors retain backend evidence.

- T003 is complete (US1 definition milestone). Checked plans carry separate active/suspended buckets/support and authoritative unfinished counts/unique keys. Active accessors retain their meaning; unfinished accessors use max(observed count, discovered key count). Both resource and ops item converters copy additions; resource totals require complete supported observations before claiming suspended support.
- Definition deletion now uses unfinished evidence for no-force blockers, force expansion, root deduplication and required drain checks. Only `ErrUnsupported` from pure suspension capability validation selects the older ACTIVE filter. Cancellation/history operations are unchanged. No-wait retains mandatory drain; no-state-check retains read-free impact planning and its existing bypass. Newly observed work is never appended to cancellation.
- T003 also owns its changed command diagnostics: impact/deletion failures use bounded class/count summaries through the configured logger, raw machine detail is preserved, partial deletion reports appear in exactly one failure envelope, and exit classification is retained. Failure counts do not reinterpret accepted resource responses as confirmed completion. Per-key failure detail is removed only from the command progress projection; the final aggregate owns diagnostics. HTTP DEBUG stays in the existing redacted transport path.

- T001 is complete. v810 publishes statistics only after all six observations succeed. `SuspendedCountSupported` is true even for zero; internal `Unfinished` is a non-nil authoritative observation, independent of the separately sampled display buckets. T003 now consumes it for deletion planning and required verification.
- The initial exact total remains the fast path. Capped counts enumerate completely; sparse empty pages advance only with known remaining items and a fresh cursor. Empty final pages can terminate despite a cursor or capped-total flag once consistent with observed counts. Stalled/cyclic/disappearing cursors and incomplete/indeterminate empty observations return `ErrMalformedResponse`, never a partial exact count. Offset fallback remains available while traversal stays in offset mode.
- T001 delivered structured statistics; T005 completed human/watch presentation and refreshed README, source metadata and generated references through `make docs-content`.
- T002 is complete. The internal union is available with tenant/definition/options preserved and no public filter/flag addition. Both foundational tasks feed T003’s existing definition cleanup workflow.

## Gotchas

- Empty definition JSON payloads are `{}` because both `total` and `items` use `omitempty`; do not assert a zero total or synthesize an array. Quiet still retains nonempty requested definition results. Empty terminal reads make one discovery request, no statistics/mutation requests and no prompts.
- Separate pre-existing diagnostic follow-up: successful watch summaries in `processDefinitionWatchView` instantiate a plain handler on configured stderr instead of using the command-context logger. T005 leaves that summary path unchanged; terminal routing checks do not establish configured-format/level compliance for it. The existing non-watch raw-error issue below also remains outside this presentation work unit.

- Terminal confirmation formatting splits the final question onto a new line. Match that actual plain-text prompt in terminal exchanges; the unformatted single-line construction string will time out.
- Purge advertises human/JSON modes, not a keys payload. Its existing `--keys-only` behavior leaves stdout empty and suppresses progress but retains human result logs on stderr unless quiet. T004 preserves this behavior, checks zero-byte stdout and quiet/machine precedence, and does not introduce a new keys contract. Empty scope still skips confirmation/mutation and reports planned/no changes, including no-wait.

- T004 completes purge integration. Its aggregate `RequiresForce`, fallback counts/keys, progress and reports now use unfinished evidence; active totals remain active-only. Unsupported, skipped, failed, or partial observations do not claim complete suspended support.
- Existing older resource/ops fixtures now explicitly return `ErrUnsupported` for suspension capability. New suspended tests use domain service stubs directly: public statistics intentionally cannot carry internal `Unfinished`, and the old resource test adapter must not fabricate it by summing buckets.
- The CLI’s existing no-match definition selector remains an error, not a newly introduced successful empty scope. Empty *instance workload* still permits explicit definition deletion and its normal confirmation. Tests distinguish this from dry-run/no-work mutation claims; no selector/no-op contract was redesigned.

- A capped-total flag is not a next-page flag, and an end cursor alone does not prove continuation. A legacy fixture claiming a lower bound of 10000 but returning three instances was corrected to consistent capped/exact observations.
- A missing definition key must not produce an unscoped union count. The new request builder rejects it as malformed without an instance request.
- T003 rejects supported statistics with nil or negative `Unfinished` at the checked-plan boundary. Older versions intentionally keep support false and `Unfinished` nil; use their existing active observation at that boundary.
- Error-path audit: ordinary non-watch definition-read failures already flow through `handleCommandError` to `ferrors.HandleAndExit`, which logs raw `err.Error()` for human output. No diagnostic/error renderer changed in T001. This pre-existing raw-transport disclosure is a separate hardening concern, not proven compliant by the passing legacy error-envelope tests. Watch has its own compact failure diagnostics. Preserve this finding for follow-up without silently widening the current task.
- T003’s definition discovery loop rejects indeterminate completion, stalled/cyclic/disappearing cursors, missing keys and positive-count discovery with no keys. `From` advances by raw observed items, including zero through advancing sparse pages; offset fallback remains available before cursor mode. The general shared page/total helpers still have permissive completion/empty-stop rules and are not proof of authoritative cleanup discovery; T002 changes their selector validation, not their traversal framework.
- A capped total is strictly a lower bound. An empty response at that bound cannot prove completion; fresh continuation is retained, otherwise it is indeterminate. A final empty response beyond the bound can complete. The legacy v810 final-empty fixture now uses From=10001 for a capped total of 10000. Short nonempty capped pages beyond the bound remain indeterminate.

## Reusable Commands

T005 final checks (via `/tmp/c8volt-340-isolated-test.py`, clearing inherited application/live/Go flags):

- `go test ./cmd -run 'TestGetProcessDefinition(Suspended|WatchSuspended)|TestProcessDefinition(View_Suspended|WatchCandidate_SuspendedTransitions)|TestRenderProcessDefinitionWatchRefresh_WriteFailuresKeepBaseline' -count=1`
- Broader related command controls: `go test ./cmd -run '^Test(GetProcessDefinition|ProcessDefinition(View|Watch)|ListProcessDefinitionsView|RenderProcessDefinitionWatchRefresh|ValidateGetProcessDefinitionWatch|SearchProcessDefinitionsWithPagingStat|CommandCapabilityForCommand_ProcessDefinitionWatchMetadata)' -count=1`
- `make docs-content` with inherited application/test/Go/Make flags cleared; targeted gofmt, tracked/untracked whitespace review, and `git diff --check`.
- No service/cleanup reruns or full-suite/race gate were needed for this view-only change. Tests cover actual finite lookup/list and terminal watch commands against test-owned HTTP servers, plus view/lifecycle stubs; no live-engine coverage is claimed.

T004 final checks (via the existing `/tmp/c8volt-340-isolated-test.py` environment-clearing wrapper):

- `go test ./internal/services/ops ./c8volt/ops -run 'TestPurgeAllProcessDefinitions|TestClientPurgeAllProcessDefinitions' -race -count=1`
- `go test ./cmd -run '^TestOpsPurgeAllProcessDefinitions|^TestDeleteProcessDefinition(Unfinished|Suspended)|^TestOps.*SemanticProgressModeGate$' -count=1`
- `make docs-content` with inherited application/test/Go/Make flags cleared, targeted gofmt, and `git diff --check`.
- Temporary direct-stderr and raw-aggregate mutations each failed `TestOpsPurgeAllProcessDefinitionsUnfinishedDiagnosticLogging/plain-time/normal` behavior assertions and were restored. The raw mutation must retain existing imports/use to avoid confusing compile failure with behavioral detection.

T003 final checks (all through `/tmp/c8volt-340-isolated-test.py`, with inherited application/live flags cleared):

- `go test ./internal/services/processdefinition ./c8volt/resource -run 'DeleteProcessDefinition|CleanupProcessDefinition|Unfinished' -race -count=1`
- `go test ./internal/services/ops ./c8volt/ops -run 'TestPurgeAllProcessDefinitions|DeleteProcessDefinition|ConvertDeleteProcessDefinitionUnfinished' -race -count=1`
- `go test ./cmd -run 'TestDeleteProcessDefinition|TestDeleteCommands_RegressionPreservesCleanupContracts|TestDeleteHelp_DocumentsDestructiveConfirmationPaths|TestConfirmOrAbortTerminal' -count=1`
- Deliberate temporary changes to direct stderr and raw aggregate logging were each rejected by `TestDeleteProcessDefinitionUnfinishedDiagnosticLogging/plain-time/normal`; both were restored before final checks. No mutation-testing changes remain.

Use the environment-clearing `isolated_go_test` wrapper from `quickstart.md`; the actual iteration wrapper was `/tmp/c8volt-340-isolated-test.py` with the same environment rules.

- `isolated_go_test ./internal/services/processdefinition/v810 -run 'TestService_(GetProcessDefinition|SearchProcessDefinitions)' -count=1`
- `isolated_go_test ./internal/services/processdefinition/v88 ./internal/services/processdefinition/v89 -run 'TestService_(GetProcessDefinition|SearchProcessDefinitions)' -count=1`
- `isolated_go_test ./c8volt/process ./c8volt/ops -run 'ProcessDefinition|Suspended|Unfinished' -count=1`
- `isolated_go_test ./cmd -run 'TestGetProcessDefinition|TestSearchProcessDefinitionsWithPagingStat|TestCommandErrorEnvelopeProcessDefinition|TestProcessDefinitionView|TestListProcessDefinitionsView' -count=1`
- `make docs-content` (inspected: runs docsgen only); clear application/test environment and inherited Go/Make flags. Format touched Go files and run `git diff --check`.
- T002 validation: `isolated_go_test ./internal/services/common ./internal/services/processinstance/... -run 'CombinedState|NormalizeProcessInstanceSearchFilter' -count=1` (traversal/waiter/walker packages contain no matching tests).
- `isolated_go_test ./internal/services/processinstance ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 ./internal/services/processinstance/v810 -run 'Search|Suspended|CombinedState|StateFilter' -count=1`
- `isolated_go_test ./internal/services/common ./internal/domain -run 'Filter|State' -count=1`
- Consumer checks: `isolated_go_test ./c8volt/process -run 'TestClient_(SearchProcessInstances|PlanProcessInstanceMutationPages)' -count=1`; `isolated_go_test ./cmd -run 'TestGetProcessInstance(TotalOutput|PagingFlow|SuspendedSparsePaging)$' -count=1`.

## Do Not Repeat

- Do not sum active/suspended display reads for authoritative cleanup evidence, expose internal `Unfinished` on ordinary facade statistics, or issue suspended/union queries on v88/v89.
- Do not restore partial-count success on cursor failure or treat inconsistent fixtures as proof of complete enumeration.
- Do not run live/integration checks or broad unrelated suites. Passing mocks do not establish real engine, exporter, or snapshot behavior.

## Current Handoff

- Feature complete; no handoff required.
