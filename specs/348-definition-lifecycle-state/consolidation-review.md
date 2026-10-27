# Task Consolidation Review: Definition Lifecycle State

**Date**: 2026-10-27
**Feature**: `348-definition-lifecycle-state`
**Reviewed list**: [tasks.md](tasks.md)
**Context**: [spec.md](spec.md), [plan.md](plan.md), design contracts, research, quickstart, repository guidance and constitution v2.0.0.

## Outcome

Consolidated the fresh draft from **4 to 3 tasks**, preserving one acceptance owner per story. No implementation was started, no checked tasks or progress record existed, and no completed evidence was moved into unfinished work. Only `tasks.md` and this review were saved by the consolidation hook. No historical review section needed relocation.

## Code-grounded boundary review

| Original boundary or shared behavior | Decision and evidence |
| --- | --- |
| T001 model/adapter/facade → T002 ordinary CLI read | Merge into final T001. `v810.fromProcessDefinitionResult` and `process.fromDomainProcessDefinition` are mechanical field copies; lookup and search already share conversion. `processDefinitionDisplay`, `flatRow`, and `formatProcessDefinitionWatchColumn` supply the existing common row path. No new service API, pagination or orchestration is needed. Combining removes a model-only handoff and delivers the optional string, omission and output evidence in one focused read unit. Existing adapter/facade/command fixtures keep validation practical; the existing draining observer is a bounded source change with compile-only validation. |
| Original T002 ordinary output → T003 statistics | Keep as final T001 → T002. Build the shared column exactly once in T001. `definitionStatisticsFixture` in `cmd/get_processdefinition_suspended_test.go` separately controls definition and count requests, and `retrieveProcessDefinitionStats` in `v810/service.go` enriches existing definitions. T002 owns the distinct proof that DRAINING survives active=0/suspended=2 and absent/unsupported/error statistics with unchanged count requests. Folding its matrix into T001 would expand the already complete lookup/list/mode/observer validation; it also crosses the explicit US1/US2 Ralph story scope. T002 must not rebuild the projection or introduce new statistics mechanics. |
| Original T003 statistics → T004 watch | Keep as final T002 → T003. Watch introduces temporal candidate/commit behavior in `renderProcessDefinitionWatchRefresh` and `processDefinitionWatchBaseline`, plus terminal styling, handler errors and timing diagnostics. These require deterministic refresh/failure sequences and PTY/configured-logger execution beyond finite statistics fixtures. Combining would cross story scope and make a single iteration cover both finite count evidence and the larger failure/terminal matrix. |
| Original T002 and T004 share the display file | Keep shared column addition/index adjustment together in final T001; final T003 adds comparison flags and state highlighting. `flatRow` currently puts statistics at index 4 and `formatProcessDefinitionWatchColumn` dispatches on that index, so separating those edits would break existing watch/stat rendering. Stateful comparison is a genuine later acceptance increment, not a second renderer implementation. |
| Proposed split of watch comparison, logger fixes and tests | Reject. `processDefinitionWatchView` currently constructs a private handler and returns its write errors; the caller commits the baseline only after rendering succeeds. `renderGetProcessDefinitionWatchRefreshStatus` currently bypasses the logger for slow warnings and timing. `renderHumanLine` is void, while `logging.FromContext` returns a fallback plus an error. Final T003 must preserve summary failure propagation, filtered-INFO success, configured logging and repaint→summary→rows together. Splitting would leave misleading baseline or logger evidence between tasks. Reuse existing write-failure, timeout, summary-order, slow-streak and PTY tests rather than introduce a new runner. |
| Tests, docs, integration observer as separate work | Keep attached to their behavior. The ordinary read observer in `TestRealStateProcessDefinitionDraining` belongs to final T001; independent API cleanup evidence remains. Each story owns source help/README and `make docs-content` when its behavior changes. No final test-only, cleanup, setup, context-reading, progress or speculative parallel task is added. |

The strongest additional merge would combine final T001 and T002 because the shared renderer supports both ordinary and statistics output. It is rejected for the concrete validation and story-scope reasons above; shared production work remains in T001 and T002 only fills independent enrichment evidence and operator explanation. Package separation alone is not the justification.

## ID mapping

| Draft ID | Final ID | Story |
| --- | --- | --- |
| T001 | T001 | US1 |
| T002 | T001 | US1 |
| T003 | T002 | US2 |
| T004 | T003 | US3 |

The draft had no progress or external task references found by repository Markdown searches for this feature's tasks path/qualified IDs, so fresh sequential renumbering was safe. All local references, coverage ownership, checkpoints and strategy text were repaired. Dependencies are T001 → T002 → T003. No `[P]` markers remain: shared projection, command fixtures and source docs make parallel implementation unsuitable. Independent validation examples are retained without creating extra tasks or requesting agent execution.

## Coverage and consistency checks performed

- Compared every draft acceptance obligation with the final list: optional string constraints and exact preservation; omitted/null/older versions; shared converter/body decoding; lookup/list/page/latest/watch propagation; mode/empty/error/request contracts; statistics independence and failures; all watch transitions and baseline outcomes; configured logging, DEBUG/redaction and failure exits; terminal checks; documentation and existing draining observer.
- Verified ownership of all **18 FRs** and **6 success criteria** in the coverage table. Story labels match the specification priorities: US1 P1, US2 P1, US3 P2.
- Programmatically verified three sequential unchecked task IDs, one per story, checklist format, path-bearing task descriptions, valid task references, dependency chain, and absence of checked progress.
- Checked relative Markdown links and trailing whitespace across the feature artifacts; `git diff --check` passed. Reviewed no-new-request/no-inference/no-new-command boundaries and the absence of explicit no-wait test additions.
- Parsed extension configuration and confirmed this consolidation is the only mandatory after-tasks hook. Optional commit and Ralph launch are left for user selection.

## Limitations

Application tests, generators, integration binaries and live environments were not run for this documentation-only change. Test names/fixtures and production ownership were inspected, not executed. Implementation must audit its selected helpers and inherited configuration before running the targeted commands in the quickstart. Compile-only integration checking does not prove live engine transitions, timing or cleanup. No unresolved planning uncertainty was identified.
