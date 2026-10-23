# Ralph Memory

Feature: 340-suspended-definition-cleanup
Started: 2026-10-23T08:10:21Z

## Codebase Patterns

- v810 definition statistics are acquired in `internal/services/processdefinition/v810/service.go` via exact-total requests (limit 1), then capped-total traversal when necessary. New suspended and union requests reuse definition/tenant scope, ordering, and paging; the union uses the generated advanced state filter's `$in`.
- Public process statistics and ops discovery/report statistics have separate mechanical converters. Existing public active/completed/canceled/incidents JSON fields retain zero values; the new suspended fields use `omitempty`. Domain statistics omit zero values and never serialize `Unfinished`.
- Adjacent versioned tests use strict generated-client mocks. Extend `expectedProcessDefinitionStats` and `mockProcessDefinitionStats` for v810; older mocks must receive no new queries.
- Internal `ProcessInstanceFilter.States` is normalized through `common.NormalizeProcessInstanceSearchFilter` at both shared search helpers and each adapter entry point. It copies/deduplicates uppercase values, maps CANCELED to TERMINATED, rejects ambiguous or invalid plural input, and uses the existing pure SUSPENDED capability check to require v810 even for a one-value plural selector. Scalar behavior stays unchanged. Direct incident-index filtering also honors the plural selector.
- v810 search decoding retains whether `totalItems` was present and nonnegative; missing/null totals are not exact zero. The page mapper preserves advancing sparse continuation, distinguishes capped lower bounds from exact totals, and leaves unresolved metadata indeterminate.

## Decisions

- T001 is complete. v810 publishes statistics only after all six observations succeed. `SuspendedCountSupported` is true even for zero; internal `Unfinished` is a non-nil authoritative observation, independent of the separately sampled display buckets. Cleanup does not consume it until T003.
- The initial exact total remains the fast path. Capped counts enumerate completely; sparse empty pages advance only with known remaining items and a fresh cursor. Empty final pages can terminate despite a cursor or capped-total flag once consistent with observed counts. Stalled/cyclic/disappearing cursors and incomplete/indeterminate empty observations return `ErrMalformedResponse`, never a partial exact count. Offset fallback remains available while traversal stays in offset mode.
- Human/watch suspended presentation remains T005. Iteration 1 documented the delivered structured statistics in README and command metadata and refreshed generated references through `make docs-content`.
- T002 is complete. The internal union is available with tenant/definition/options preserved and no public filter/flag addition. Both foundational tasks are now ready for T003; mutation orchestration has not changed.

## Gotchas

- A capped-total flag is not a next-page flag, and an end cursor alone does not prove continuation. A legacy fixture claiming a lower bound of 10000 but returning three instances was corrected to consistent capped/exact observations.
- A missing definition key must not produce an unscoped union count. The new request builder rejects it as malformed without an instance request.
- Supported statistics with nil `Unfinished` must be rejected by the cleanup boundary in T003. Older versions intentionally keep support false and `Unfinished` nil; use their existing active observation at that boundary.
- Error-path audit: ordinary non-watch definition-read failures already flow through `handleCommandError` to `ferrors.HandleAndExit`, which logs raw `err.Error()` for human output. No diagnostic/error renderer changed in T001. This pre-existing raw-transport disclosure is a separate hardening concern, not proven compliant by the passing legacy error-envelope tests. Watch has its own compact failure diagnostics. Preserve this finding for follow-up without silently widening the current task.
- For T003, use the definition workflow's existing direct page loop and reject indeterminate completion, stalled/cyclic/disappearing cursors, and incomplete scope. `From` must reflect actual observed raw items, not requested page size after an empty cursor page. The general shared page/total helpers still have permissive completion/empty-stop rules and are not proof of authoritative cleanup discovery; T002 changes their selector validation, not their traversal framework.
- A capped total is strictly a lower bound. An empty response at that bound cannot prove completion; fresh continuation is retained, otherwise it is indeterminate. A final empty response beyond the bound can complete. The legacy v810 final-empty fixture now uses From=10001 for a capped total of 10000. Short nonempty capped pages beyond the bound remain indeterminate.

## Reusable Commands

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

- Next dependency-ready work is T003 (US1, Phase 3): coherent unfinished planning, discovery, gates, force cleanup and required verification, including plan/facade/CLI projections and isolated regression evidence. T001 and T002 are complete; recheck selection against `tasks.md`. T004 depends on T003; T005 remains pending in the preferred order. No blocking prerequisite is known.
