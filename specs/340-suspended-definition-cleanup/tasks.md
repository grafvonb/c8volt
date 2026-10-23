# Tasks: Include Suspended Instances in Definition Cleanup

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [contract](contracts/definition-cleanup.md), [quickstart.md](quickstart.md)
**Branch / Issue**: `340-suspended-definition-cleanup` / #340
**Status**: Generated implementation backlog; all work is pending. No implementation or runtime validation performed.

## Format and execution rules

Each checkbox is one cohesive, compiling work unit with implementation, affected fixtures/converters, tests, formatting and relevant documentation. Labels identify the primary story owner; the coverage matrix records shared acceptance coverage without duplicating implementation. File paths are repository-relative.

Read active artifacts, AGENTS.md and `specs/ralph-implementation-rules.md` as part of each unit. Follow Ralph's one-story-or-one-non-story-phase scope; do not combine foundational and story tasks in one iteration. Context reading, branch checks, review, and progress/memory maintenance are responsibilities, not tasks.

Use isolated tests only. Before execution inspect test dependencies and clear inherited flags/application configuration using [quickstart.md](quickstart.md). No integration tags/targets, live environments, smoke runs, cluster discovery, deployments, or cleanup. Do not restart existing tests. No generated-client changes, new mutation engine, resumption, extra child cancellation, or DRAINING display work.

Tests are required by the specification. Within each unit, add/extend the relevant failing regression before the behavior change, then implement and run targeted checks. When new fields are needed for a test to compile, introduce the minimal compiling model/fixture shape first; a compile error is not the required behavioral regression. Retain passing controls; do not manufacture failures or split tests into standalone tasks. Run gofmt on touched Go files, document actually executed checks, and broaden only for a concrete risk or unresolved failure.

For user-visible changes, update source metadata and README with that unit and regenerate relevant CLI docs using `make docs-content` after verifying the target. Do not hand-edit generated references. All issue-associated commit subjects end in ` #340`; Ralph additionally follows its coordinated-commit rules.

## Phase 1: Setup

The repository, branch, toolchain, and fixtures already exist. No setup task is needed. Do not add dependencies, scaffold packages, provision services, or create a new test harness.

## Phase 2: Foundational — observed counts and combined selection

**Goal**: Supply the two reusable capabilities required by cleanup without changing mutation orchestration.

- [ ] T001 [P] Implement separate suspended statistics and authoritative unfinished observations in `internal/domain/processdefinition.go` and `internal/services/processdefinition/v810/service.go`, with public suspended/support fields and mechanical mappings in `c8volt/process/model.go`, `c8volt/process/convert.go`, and `c8volt/ops/convert.go`; extend adjacent service/facade tests and their existing statistics fixtures, preserve older adapters, format and run the targeted statistics/conversion checks described below.

**T001 acceptance and constraints**:

- Add `Suspended int64` with `suspended,omitempty`, `SuspendedCountSupported bool` with `suspendedCountSupported,omitempty`, and internal `Unfinished *int64` with `json:"-"`, absent from facade statistics. Preserve all existing buckets and meanings.
- Apply the model's exact rules: “A supported result with nil internal Unfinished is invalid for cleanup.” “For 8.10, publish a statistics result only after all required observations succeed, including the new suspended and combined queries.” “Older versions leave suspension support false, issue no new suspended/union query, and retain their existing active-only cleanup count.” Boundary rejection of missing evidence is completed by T003; T001 establishes the evidence contract.
- Query SUSPENDED separately for display and ACTIVE/SUSPENDED together using the generated advanced filter for the authoritative count. Do not sum separate bucket reads. Reuse current exact-total optimization and capped-count traversal; incomplete/stalled combined-count enumeration returns an error rather than a partial exact count. Preserve genuine final empty pages and offset fallback; capped-total metadata is not a next-page flag and a cursor alone does not establish continuation.
- Extend `internal/services/processdefinition/v810/service_test.go`, matching v88/v89 service tests, `c8volt/process/convert_test.go` and affected ops conversion/client tests. Cover stable active=2/suspended=3/unfinished=5, supported zero, unsupported omission, differing observations during state transitions, union/read failures, exact versus capped counts, cursor stalls and no-stat request counts. No public API signature or report schema version change.
- Run the smallest matching statistics tests in `./internal/services/processdefinition/v810`, older adapter controls, and affected `./c8volt/process` / `./c8volt/ops` conversion tests with `-count=1`. Document additive structured statistics semantics in README. Human/watch presentation is T005; avoid claiming it is delivered here.

- [ ] T002 [P] Implement the internal combined-state selector in `internal/domain/processinstance.go`, validation in `internal/services/processinstance/search.go` and existing v88/v89/v810 adapter search entry points, and v810 request mapping in `internal/services/processinstance/v810/convert.go`; include fake-backed filter/capability/page-normalization regressions in adjacent tests, preserve scalar queries, format and run targeted instance-search checks.

**T002 acceptance and constraints**:

- Add internal `States` using the existing domain states collection, without adding a CLI flag or public facade filter field. Keep model constraints verbatim: “Singular and nonempty plural selection are mutually exclusive; reject ambiguity before I/O.” “Normalize and deduplicate the plural values and validate every requested state. Do not silently drop unsupported values or broaden a query.”
- Map the 8.10 union to the existing generated `$in` shape. Older adapters reject unsupported plural input before I/O; existing scalar searches and tenant/options remain intact. Update all affected validation entry points together so shared service helpers do not validate only singular State.
- Preserve authoritative continuation information needed by cleanup: empty pages with genuine advancing continuation cannot be normalized to complete solely because they are empty. Test final empty, sparse advancing, capped total, cursor-only ambiguity, and indeterminate metadata. Keep fixes local to the affected mapping/normalization; no generic traversal framework.
- Extend `internal/services/processinstance/search_test.go`, v810 `service_test.go`/`convert_test.go` and matching older adapter tests. Assert request bodies contain both states with definition and tenant criteria, unsupported/ambiguous input sends zero requests, and old scalar requests retain their shapes.
- Run matching isolated search/state-filter tests in `./internal/services/processinstance/...` with `-count=1`, using narrow test patterns first. No generated client edits and no user documentation for a new filter, since none is exposed.

**Checkpoint**: Both tasks leave the project compiling. Combined observation and selection are available, with all relevant fixtures updated; mutation behavior is not switched piecemeal.

## Phase 3: User Story 1 — Recognize unfinished work (P1)

**Goal**: Make definition deletion correctly evaluate paused work and keep its existing forced path coherent with that evaluation. Shared force mechanics are implemented here once, then reused by US2's purge integration.

**Independent test criteria**: Direct single/bulk definition deletion refuses suspended-only and mixed no-force scopes before their existing mutation boundary; terminal/empty controls remain eligible. Force requests use the same unfinished scope through planning and mandatory verification. US1's purge-entry acceptance is completed by T004, not claimed at this checkpoint.

- [ ] T003 [US1] Deliver coherent unfinished-work planning and deletion in `internal/services/processdefinition/delete.go`, additive plan fields/accessors in `internal/domain/resource_workflow.go` and `c8volt/resource/model.go`, mechanical mappings in `c8volt/resource/convert.go` and `c8volt/ops/convert.go`, and affected delete-command preflight/views in `cmd/delete_processdefinition.go`; include adjacent service/facade/command regressions, updated help/README/generated docs, formatting and targeted isolated validation in this same unit.

**T003 acceptance and constraints**:

- Depends on T001 and T002. Add the plan fields specified by [data-model.md](data-model.md): suspended count/support, authoritative unfinished count and unique unfinished keys, with their exact public tags. Preserve active count/key fields and `ActiveProcessInstances()` semantics. Carry additions through plan totals and both resource/ops item converters; do not defer a converter that would discard required evidence.
- Enforce “The internal nil-unfinished validation happens before producing a successful checked plan.” “Separate active keys are populated only from items observed ACTIVE, never by copying the union.” “Existing `StateCheckSkipped` at the plan boundary remains the signal that counters were not observed.” “Older-version plans populate unfinished from their existing active observation.” The separate unfinished accessor must not understate known discovered keys.
- Use authoritative unfinished count in enabled preconditions, force impact expansion, and required drain verification together. Select the union through existing pure capability validation; only unsupported capability selects older ACTIVE behavior. Do not switch a blocker to unfinished while leaving its forced cleanup ACTIVE-only.
- Page union discovery completely, deduplicate keys, retain tenant/options, continue authoritative advancing sparse pages, and error on stalled/indeterminate incomplete traversal. Feed results into current root planning and bulk unique-root scope. Preserve hierarchy impact disclosure, force escalation, cancellation dispatch, worker limits, fail-fast, instance-history order and resource-deletion behavior.
- Keep required verification before dependent history/definition deletion even with no-wait. Preserve no-state-check's skipped/read-free plan and existing bypass; no new state reads in skipped mode. Newly observed work during verification is not added to cancellation. Failure, persistent suspension and timeout stop dependent operations; accepted nullable responses, batch/final confirmation and existing partial results remain unchanged.
- Extend `internal/services/processdefinition/delete_test.go`, `c8volt/resource/client_test.go` and `cmd/delete_test.go`, `cmd/delete_processdefinition_response_test.go` and existing adjacent command fixtures. Cover no-force suspended-only/mixed, active/terminal/empty controls, suspended root, both mixed trees, overlapping roots, transition/late-work observations, sparse/incomplete discovery, missing union evidence, read/cancel/drain/history failure, no-wait/bypass combinations and older-version restrictions. Assert request ordering and zero resume/added child cancellation; one root dispatch per root in successful non-retried overlap cases.
- Correct active-only blockers/previews/progress to unfinished wording with available breakdown; preserve result modes and logger routing. Apply the shared output-validation obligations below, reuse fixtures, update affected source metadata/README, regenerate docs and run matching definition-delete service/facade/command tests from [quickstart.md](quickstart.md). Do not refactor instance cancellation or add an all-or-nothing collection contract.

**Checkpoint**: Definition deletion is internally consistent, including its existing forced recovery. Purge's own aggregate gate/report adaptation remains T004; do not advertise the entire feature as complete.

## Phase 4: User Story 2 — Include suspended work in force cleanup (P1)

**Goal**: Integrate the shared definition behavior into purge without a second cleanup engine, completing both purge eligibility and force behavior.

**Independent test criteria**: Purge refuses suspended-only/mixed no-force scopes without mutation, and force purge includes their existing root scope, preserves overlapping-root deduplication, and stops dependent work after failure/unconfirmed observations, including no-wait.

- [ ] T004 [US2] Integrate unfinished workload through `internal/services/ops/all_process_definitions_purge.go`, `internal/domain/ops_all_process_definitions_purge.go`, `c8volt/ops/model.go` and `c8volt/ops/convert.go`, then update `cmd/ops_purge_all_processdefinitions.go`, `cmd/cmd_views_ops_purge_all_processdefinitions.go`, `cmd/ops_report_purge_all_processdefinitions.go` and `cmd/ops_purge_all_processdefinitions_progress.go`; include the corresponding isolated service/facade/command/report tests, source help/README/generated docs and formatting in the unit.

**T004 acceptance and constraints**:

- Depends on T003. Add suspended/unfinished totals/support following [data-model.md](data-model.md); preserve existing active totals/tags. Enforce: “Carry suspended support in aggregates; a skipped check is not an observed zero.” “RequiresForce uses unfinished workload, not active-only workload.” “Preserve the existing report schema version and result/envelope shapes; these are additive fields.”
- Use unfinished counts/keys for affected-count fallback and affected/blocked report-key extraction where expanded family evidence is absent. Keep candidate count, direct unfinished instances, and expanded affected family distinct. Do not add counts across overlapping families instead of existing key deduplication.
- Keep existing definition candidate selection/freeze, latest/version behavior, tenant boundaries, limit reporting and existing execution controls. Invoke the shared definition workflow from T003; do not copy its cancellation/discovery logic or add a purge bypass flag. Update no-force preflight at both service and CLI entry points before prompting/mutation.
- Extend `internal/services/ops/all_process_definitions_purge_test.go`, `c8volt/ops/client_test.go`, `cmd/ops_purge_all_processdefinitions_test.go`, and `cmd/ops_purge_all_processdefinitions_progress_test.go`. Use the existing no-force, overlapping-root, execution-controls, frozen-limit and older-capability fixtures, adding suspended-only/mixed variants. Cover failed/unconfirmed prerequisites, additional work, no-wait, dry-run, empty/terminal controls, partial results, report field retention, keys, support/zero, and count agreement across Markdown/JSON/human output.
- Apply shared output-validation obligations below, including real terminal tests where prompts/empty paths are affected. Update existing progress wording from active to unfinished without new lifecycle events or chatter. Regenerate affected CLI docs from metadata and run matching isolated purge service/facade/command tests with `-count=1`; no live integration harness changes.

**Checkpoint**: US1 and US2 definition/purge acceptance is covered through the same cleanup implementation; remaining US3 work is human statistics/watch presentation.

## Phase 5: User Story 3 — See accurate statistics and cleanup impact (P2)

**Goal**: Display separate suspended statistics consistently in existing lookup/list/watch output. Cleanup reports are already updated alongside their behavior in T003/T004 rather than deferred here.

**Independent test criteria**: Known active/suspended populations render accurately in human and structured results; suspended-only watch changes refresh; zero/unavailable transitions clear stale data. Existing non-stat, keys-only, quiet, empty and older-version output remains valid.

- [ ] T005 [US3] Complete suspended statistics presentation in `cmd/cmd_views_processdefinition.go` and applicable metadata in `cmd/get_processdefinition.go`, with static/watch/terminal regressions in `cmd/cmd_views_processdefinition_test.go`, `cmd/get_processdefinition_watch_test.go` and existing adjacent terminal fixtures; update README and generated definition CLI documentation, format and run targeted isolated command checks.

**T005 acceptance and constraints**:

- Depends on T001 for statistics; normal execution order follows T004 for a single finished cleanup feature. Add compact `su:` only when supported and use existing zero-as-dash formatting. Preserve active fields and existing support/omission rules; do not expose internal Unfinished on ordinary statistics.
- Update the display projection, equality/change identity, changed-fields flags, flat rows, watch statistics formatting and support transitions together. Cover supported nonzero/zero, unsupported omission, suspended-only changes, ACTIVE-to-SUSPENDED refresh, clearing removed statistics, unchanged snapshots, highlighting/alignment and baseline ownership. Reuse watch lifecycle in `cmd/get_processdefinition_watch.go`; no new timing/retry logic or DRAINING display.
- Exercise actual lookup/list/watch command paths in addition to view tests, with test-owned dependencies. Verify human, JSON, keys-only, quiet and empty behavior plus no-stat request preservation. Use the shared output obligations for changed paths; preserve logger routing and no raw HTTP data. Tests and documentation must distinguish unavailable statistics from an observed zero.
- Run the narrow matching definition view/watch/command test patterns in `./cmd` with `-count=1`. Update statistics help/README, regenerate docs and review generated changes. Do not rerun unrelated cleanup checks unless this diff invalidates them.

**Checkpoint**: All three stories are complete after their actual isolated checks pass. No real-engine claim is made.

## Phase 6: Polish and cross-cutting concerns

No standalone polish task: formatting, docs, diagnostics, output-mode regressions and review belong to the owning units above. At each completion verify the changed path against [contract](contracts/definition-cleanup.md); preserve real-engine limitations in recorded evidence. Do not manufacture a final broad test or live-validation task.

### Shared output-validation obligations (part of affected units)

- Trace service error → facade conversion → command result/error handling → configured logger → exit. Review changed direct writes and error formatting. Results stay on stdout, plain prompts on configured stderr, diagnostics on the configured logger.
- For changed error paths, capture both streams and assert configured/inherited destination and severity, no duplicate emission, repeated-error aggregation with distinctive URL/method/body/key, no raw HTTP details at normal/verbose levels, compact errors in quiet, exactly one JSON envelope with EOF, partial results and original failure exit. Check DEBUG via flag and configured level with redaction and both plain/structured logs. Assertions must fail for a logger bypass or normal-level transport leak.
- For affected empty/prompt paths, use real terminal stdin and separate captures. Assert exact human empty text, zero-byte keys, one successful JSON envelope, no empty-scope confirmation/mutation, quiet+machine precedence, supported automation/auto-confirm/no-wait, and nonempty/sparse/abort/error controls. Reuse existing fixtures rather than creating another harness.
- Do not label renderer-only tests or a stderr substring as execution-path proof. Extend only the affected coverage required by AGENTS.md; preserve completed evidence without repeatedly running passing checks.

## Dependencies and execution order

```text
T001 (statistics) ─┬─> T005 (statistics UI)
                  └─> T003 (definition workflow) ─> T004 (purge)
T002 (selector) ─────> T003
```

T005 technically needs only T001, but default/Ralph order is T001, T002, T003, T004, T005. T003 depends on both foundations; T004 depends on T003. Tasks from different stories/phases are not combined in a Ralph iteration. Existing files, models and fixtures remain compiling after every unit.

**Story completion**: US1's definition milestone is T003, with its purge acceptance completed in T004. US2's shared mechanics are implemented in T003 and its purge integration/acceptance completes in T004. US3's data/report foundations are covered by T001/T003/T004 and its remaining presentation completes in T005. This is shared implementation ownership, not three independent engines.

## Parallel opportunities and examples

Only T001 and T002 are marked `[P]`: their production/test owners are different definition/statistics versus instance-selector files and neither requires the other's code. Parallel execution is optional; do not create tasks or spawn agents merely to exploit it. Merge/validate their results before T003; avoid shared progress-file writes if using separate workers.

- **US1**: No parallel implementation tasks. T003 keeps models, converters, enabled gates and forced cleanup together. Independent fake-backed service/facade checks may be run together only after verifying they do not share mutable state.
- **US2**: No parallel implementation tasks. T004's service and report changes share the same models and command fixtures; splitting them risks inconsistent counts. Reuse the shared T003 workflow.
- **US3**: No parallel implementation tasks. The static/watch projection and changed-field logic belong together in T005. Its technical prerequisite is T001; default execution remains sequential to avoid concurrent command/docs edits with T003/T004.

## Coverage and counts

| Requirement | Owning tasks |
| --- | --- |
| FR-001 unfinished classification | T001, T003, T004 |
| FR-002 statistics/output/watch | T001, T005 |
| FR-003 no-force preconditions | T003, T004 |
| FR-004 union discovery/completeness | T002, T003, T004 |
| FR-005 existing cleanup/root behavior | T003, T004 |
| FR-006 mandatory unfinished verification | T001, T003, T004 |
| FR-007 failure/partial result preservation | T003, T004 |
| FR-008 no-wait/bypass semantics | T003, T004 |
| FR-009 output/dry-run/empty contracts | T003, T004, T005 |
| FR-010 older-version compatibility | T001, T002, T003, T004, T005 |
| FR-011 existing resource lifecycle | T003, T004 |
| FR-012 help/documentation/scope | T001, T003, T004, T005 |

SC-001 maps to T003/T004; SC-002 to T002/T003/T004; SC-003 and SC-004 to T003/T004; SC-005 to T001/T003/T004/T005; SC-006 to the affected compatibility/output checks in all units.

**Total**: 5 tasks: 2 foundational, 1 US1, 1 US2, 1 US3; 0 artificial setup/polish tasks. All are pending.

## Implementation strategy and consolidation review

All five tasks, T001–T005, are required to complete #340. Start with the independent foundations, then complete definition cleanup, purge integration, and statistics/watch presentation. Intermediate checkpoints describe implementation progress only; they are not a reduced release scope, and T005 is not optional.

The plan's three broad groups are refined into five manageable units. Statistics acquisition and display are separated because they have independently testable observation and presentation outcomes and to retain story ordering. The internal selector is separately testable before mutation wiring. Definition cleanup and purge remain separate because their owners and observable entry points differ. Do not split the coupled checks/force-discovery/drain changes inside T003, or peel tests/models/converters off into extra tasks. The two foundations use distinct owners and have distinct request contracts, so merging them solely to reduce count would enlarge an iteration without removing duplicate work.

**Consolidation result (2026-10-23)**: 5 → 5 tasks. No merge removes meaningful duplicate work without enlarging a cohesive unit or obscuring story ownership. Corrected the delete-command test path to existing fixtures and separated the actual T005 dependency from the preferred execution order in the graph. All acceptance coverage and safe intermediate behavior are retained. There is no live-validation gate, no implementation evidence yet, and no authorization to run Ralph from this task-generation command.
