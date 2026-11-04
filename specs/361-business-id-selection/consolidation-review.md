# Implementation task sizing review

**Date**: 2026-11-01

**Feature**: #361, Business-ID Filters and Process Selection

**Result**: [tasks.md](tasks.md) reduced from **14 to 10 unchecked implementation tasks** after direct review of the code and requirements. This review was performed without invoking the consolidation skill, as requested by the user. It supersedes the earlier assessment that retained all 14 tasks.

## Why the previous assessment was too conservative

The previous review treated different native payloads, resource eligibility and test scenarios as sufficient reasons for separate tasks. Those differences must remain explicit in the code and tests, but several do not require separate implementation iterations. It also described established planners and paging helpers as if integrating BID meant rebuilding those mechanisms.

The revised split distinguishes new algorithms/lifecycles from adapting existing behavior. Each merged task still contains implementation, affected interfaces/stubs, command execution coverage, formatting and documentation. No requirements were removed to reach a number. Ten is the current practical decomposition, not a proven minimum or an iteration-time guarantee.

## Merges and code evidence

| Previous tasks | Current task | Concrete basis and work avoided |
| --- | --- | --- |
| T002 + T003: PI and user-task native filters | T002: native BID queries | `internal/services/usertask/v810/search.go:userTaskSearchRequest` already builds equality filters using `newStringEqFilterPtr`; `usertask/search.go:walkUserTaskSearchPages` already handles raw progress, sparse pages, cursor cycles and metadata validation. Task-level BID is another native field plus presence/version checks and mapping. The PI custom `isEmpty` filter and direct-incident guards remain explicit, but both resources can implement one exact-value/presence policy and its adapter matrix in one focused search task. Avoid two policy/documentation passes. Keep task-snapshot identity, AND predicates and both actual command paths tested. |
| T004 + T005: child paging and BID walk/resolver | T003: complete selection and walk | `v810/service.go:GetDirectChildrenOfProcessInstance` is a small wrapper around `SearchForProcessInstances(..., 1000, ...)`. Once T001 delivers authoritative paging, replacing that cap is reuse of the engine, not another traversal framework. Implement it with the resolver and first walk consumer so unique selection and complete family traversal become one usable read-only outcome. Existing walkers remain the relationship engine. Preserve >1000 siblings, deeper descendants, sparse/error pages, duplicate/ambiguous seeds and tenant-option tests. |
| T008 + T010: update and suspend/resume selection | T006: adapt existing process plans | `processinstance/update.go:PlanProcessInstanceVariableUpdates` already freezes copied payloads and root-variable observations; its executor already consumes that plan. `suspend_plan.go` and `resume_plan.go` already freeze classified descendant `TargetKeys`. Their scope/failed-discovery tests already exist. The new work is connecting one completed BID selection, origin/capability evidence and empty-scope handling to these established workflows. Do that once as a same-story integration unit. Keep exact update scope, exact default state controls and descendant-only tree behavior distinct; no generic mutation framework or reimplementation of these planners. Retain every command's execution/logging/PTY tests. |
| T012 + T013: retention and orphan purge | T009: frozen cleanup handoff | Both ops services call existing discovery, build a delete plan, then call `pisvc.DeleteProcessInstances` on roots. Both command confirmation paths retain only `DiscoveredKeys`, causing replanning. Once T005 supplies frozen-member deletion, the principal change is the same complete typed-plan handoff, early version checks, provenance/report copying and executor consumption. Implement both together rather than establish and then retrofit that handoff twice. Retention's fixed date/state policy and orphan's candidate-limit correction remain explicit acceptance cases; neither requires a new deletion engine. |

The cleanup merge is the largest combined unit. Its scope is limited to two adapters of the already-completed deletion lifecycle, their existing eligibility rules and their existing report schemas. It must not absorb T005's executor implementation, add a generic ops framework or weaken per-command failure coverage.

## Remaining boundaries

- **T001, discovery protocol:** `SearchProcessInstancesPages` currently returns zero-value results on errors, does not deduplicate before limits, and breaks on visitor stop or non-HasMore state without explicit completion evidence. Correcting raw progression, unknown continuation, total counting and partial evidence affects ordinary query behavior too. Keep this actual shared algorithm correction separate from native field wiring; do not bundle it into an oversized all-search task.
- **T004 versus T005, cancellation versus deletion:** T004 owns global family planning, full preview and frozen root submission. `v810/service.go:DeleteProcessInstance` currently calls `Descendants`, recursively deletes children, handles conflict-triggered cancellation, retries and waits for absence. T005 must extract exact-target submission without enlarging the reviewed membership. That is a new execution lifecycle, unlike the existing-plan adapters in T006. Sharing the family plan removes duplication without merging the two lifecycles.
- **T007, resolution:** `incident/resolve.go:ResolveProcessInstanceIncidentsOperation` discovers active incidents during execution. Preserving reviewed incident actions needs an actual plan/execution boundary and partial/wait semantics; it is not just another instance-key adapter for T006.
- **T008, repair:** `ops/repair.go:finishProcessInstanceIncidentRepair` composes variable updates, incident work, job applicability, worker execution and remaining-incident evidence. `finishDryRunIncidentRepair` builds a separate preview. Reusing T007 mechanics does not eliminate repair's own complete-plan handoff. Keep this US4 operation separate from US3 resolution and from cleanup.
- **T010, slow analysis:** a separate filter/mode model and delayed confirmation before enrichment require a read-only lifecycle change. Sparse unknown-total empty discovery must never prompt. This does not share the cleanup executor or repair actions, so combining them would only batch unrelated work.

## Every original adjacent boundary reconsidered

| Original boundary | Decision |
| --- | --- |
| T001 / T002 | Retain discovery algorithm versus native-field integration; existing user-task paging makes the alternative T002+T003 merge better bounded. |
| T002 / T003 | Merge native query delivery. |
| T003 / T004 | Keep US1 query delivery separate from US2 relationship/cardinality work; merge each with its closer partner instead. |
| T004 / T005 | Merge exhaustive child lookup with resolver/walk delivery. |
| T005 / T006 | Keep the read-only selector/walk outcome separate from global mutation-plan construction, confirmation and submission. |
| T006 / T007 | Retain new family-plan lifecycle versus new exact-member deletion lifecycle. |
| T007 / T008 | Do not couple recursive deletion replacement to existing variable-plan integration. |
| T008 / T009 | Keep new incident plan/execution separation distinct; merge T008 with the nonadjacent existing-plan adapters T010 instead. |
| T009 / T010 | Same reasoning: incident freezing is substantive new service behavior; state controls reuse existing planners. |
| T010 / T011 | Keep existing process-plan integration separate from repair's incident/job/variable orchestration and its primary story. |
| T011 / T012 | Repair and cleanup do not share executable action plans; combine cleanup with its orphan counterpart instead. |
| T012 / T013 | Merge the two frozen cleanup handoffs, preserving different eligibility/limit cases. |
| T013 / T014 | Keep destructive cleanup and read-only delayed-preflight analysis separate. |

Nonadjacent sharing is explicit: T002 owns the common value/version policy; T003 owns the single collection/unique resolver; T004 owns global family planning; T005 owns frozen deletion reused by T009; T007 supplies reviewed incident mechanics to T008. Each task includes its own thin facade, conversions, CLI integration and tests. No duplicated selector, delete recursion or single-item-first implementation is planned. All merges stay inside one primary user story. US5 acceptance remains embedded in each delivery unit.

## ID mapping and progress

The draft had no completed task or feature progress record. Other feature artifacts contained no task-ID references; the old IDs remain meaningful in prior conversation messages and are mapped below. IDs were renumbered to keep executable order sequential.

| Original ID(s) | New ID | Outcome |
| --- | --- | --- |
| T001 | T001 | Reliable process discovery |
| T002, T003 | T002 | Native PI and task queries |
| T004, T005 | T003 | Selection and complete walk |
| T006 | T004 | Cross-family cancellation |
| T007 | T005 | Frozen-member deletion |
| T008, T010 | T006 | Update and state-control integration |
| T009 | T007 | Reviewed incident resolution |
| T011 | T008 | Frozen repair |
| T012, T013 | T009 | Frozen retention/orphan cleanup |
| T014 | T010 | Slow analysis |

New counts: foundation 1, US1 1, US2 1, US3 4, US4 3, US5 0 standalone. All ten remain unchecked. The MVP is T001–T002. Functional dependencies, graph, ownership table, checkpoints and counts use the new IDs. Shared files still favor serial implementation; existing isolated validation opportunities remain. No separate setup, test-only, documentation-only or polish task was introduced.

## Preservation and checks

Preserved every original task's implementation and validation detail under its new owner, including quoted model constraints. The merged tasks identify one outcome and keep action-specific details as subordinate instructions. Compared original task bodies to the revised text after ID mapping, rather than relying only on a task count.

The ownership table still covers FR-001–FR-024 and SC-001–SC-008. Cross-family cancellation retains the three-seed/two-root/seven-member fixture, deduplication across pages/families, complete dry-run arrays, sparse/late-error behavior, tenant exclusion, seed limits independent of expansion, >1000 children and no new selected roots after approval. Frozen deletion's new-child failure, orphan candidate-limit correction, typed ops plans and logger/JSON/PTY/version obligations remain.

Checked sequential IDs, primary-story labels, acyclic earlier-task dependencies, updated references, exact preservation of original task bodies after mapping, local Markdown links, fence balance and whitespace. Specification, plan, contracts, other supporting feature artifacts and application code were not edited. The prior review's facade/test path corrections remain incorporated; its no-merge conclusion is superseded.

Only tasks.md and this review were revised. No application tests, builds, live-environment checks, commits, hooks or Ralph launches were performed. Implementation effort remains an estimate from inspected code; actual execution must preserve every acceptance obligation even where tasks are grouped.
