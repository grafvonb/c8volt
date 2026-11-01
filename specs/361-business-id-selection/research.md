# Research: Business-ID Filters and Process Selection

**Date**: 2026-11-01 | **Feature**: [spec.md](spec.md) | **Issue**: #361

Research used the checked-in source, generated contracts, existing tests, constitution, and repository/Ralph rules. No live environment was inspected or exercised. Proposed identifiers below describe implementation work, not existing capabilities.

## 1. Native exact filters and optional presence

**Decision**: Add copied optional `BusinessId *string` criteria to public/domain process filters, task search requests, and slow-analysis filters. Preserve omission versus explicit empty input; reject explicit empty input and invalid UTF-8 without trimming, normalization, splitting, or interpreting wildcard characters. Do not import creation's 256 UTF-16-unit limit into search.

**Evidence**: `internal/clients/camunda/v810/camunda/provenance.json` pins 8.10.3 and commit `3961b79c3fdaf43c48691ed77672d6db42d0abcc`. Its `client.gen.go` defines `ProcessInstanceFilter.BusinessId` and `UserTaskFilter.BusinessId` as `*StringFilterProperty`; `StringFilterProperty0` is the exact string alternative. Search fields do not use the creation `BusinessId` constrained type. The task description notes availability for tasks created on 8.10 onward.

**Rationale**: Native resource-level matching retains backend task snapshots and tenant predicates. `toolx.CopyPtr` prevents a caller changing a frozen criterion after conversion. Use the existing filter formatting helpers with safely quoted opaque text.

**Alternatives considered**: Variable filters, key-list parsing, advanced string operators, process-to-task joins, and creation-length validation would change the requested semantics. Generated clients already expose the native fields; regeneration is unnecessary.

## 2. Custom process payload and shortcut eligibility

**Decision**: Extend the custom `processInstanceFilter` and `isEmpty()` in `internal/services/processinstance/v810/convert.go`, and construct the exact native string union in `v810/service.go:SearchForProcessInstancesPage`. Add the task field in `internal/services/usertask/v810/search.go`. A supplied BID disables the direct-incident-index route at both `cmd/get_processinstance_search.go:canSearchProcessInstancesViaDirectIncidentIndex` and service request normalization in `internal/services/processinstance/search.go`.

**Rationale**: The PI adapter serializes a custom body; changing only generated types or facade models would silently drop BID. The incident index cannot express BID and applies limits before loading process instances. Filtering those limited candidates afterward can miss later valid matches. Normal PI search followed by existing direct-incident filtering preserves AND semantics and service-owned totals.

**Alternatives considered**: A command-only route guard leaves library callers unsafe. Local BID filtering after incident selection is not equivalent to native PI selection. No generic capability framework is needed.

## 3. Capability checks before every shortcut

**Decision**: v88/v89 PI and task adapters reject every non-nil BID criterion with `domain.ErrUnsupported` before a request; v810 validates the exact input before body construction. Add a no-I/O `ValidateProcessInstanceSearchFilter` method to the internal PI API and all adapters for fresh search, shared selection and frozen process-plan consumption; update implementations and stubs together. Ops also checks its configured `Service.version` at entry, before empty/frozen-key shortcuts, using one narrow shared BID search-input policy. Nil input leaves older behavior intact. No extra backend probe is added solely to validate capability.

**Evidence**: Versioned PI page searches and task `search.go` own request construction. `internal/services/ops/api.go:Service` already carries `version`. Retention and orphan workflows can bypass discovery when `DiscoveredKeys != nil`; an adapter-only check does not cover that path.

**Rationale**: Both fresh discovery and confirmed/frozen execution must reject unsupported input, including a v810-built or caller-supplied BID plan passed to a v88/v89 executor without fresh search. Shared input validation can accept the already known version; it must not become a general feature registry or introduce version checks into facades. Do not use state support as an indirect BID capability test.

**Alternatives considered**: CLI-only validation misses public callers; forcing a redundant search just to validate a frozen request adds latency and can alter scope.

## 4. Explicit completion and one process discovery engine

**Decision**: Extend the process search result with a completion disposition, following the existing user-task traversal pattern: exhausted, documented limit reached, visitor stopped, and incomplete evidence accompanying errors. Centralize continuation validation, raw-page advancement, cursor-cycle protection, and distinct-key accounting in the process service. Reuse that traversal for a narrow selection API and mutation planner; do not implement a resolver per command.

**Evidence**: `internal/services/processinstance/search.go:SearchProcessInstancesPages` currently cannot distinguish exhaustion from visitor stop and stops on any overflow other than `has_more`. Its total fallback stops on an empty page. `internal/domain/processinstance.go:ProcessInstanceSearchPagesResult` contains only items, limit, and pages. `internal/services/usertask/search.go:walkUserTaskSearchPages` already demonstrates validated metadata, terminal probes, sparse continuation, and cursor-cycle checks.

**Rationale**: Empty intermediate pages and duplicate observations cannot prove completion or consume a distinct-instance limit. Prefer cursors, otherwise advance by raw backend progress. Unknown continuation must be resolved by trustworthy metadata or a supported terminal probe, or fail as incomplete; it must not become success. Preserve existing ordinary query prompt policy, while BID mutations require a complete bounded selection and walk uniqueness requires exhaustion.

**Alternatives considered**: Inferring completeness from a slice length, first page, reported lower-bound total, or absent error is unsafe. Reusing the task package directly would create an inappropriate resource dependency; reuse its pattern inside the process service.

## 5. Cross-family cancellation requires full planning

**Decision**: Introduce a full-scope process selection/planning entry point in `internal/services/processinstance`, exposed through a thin process facade. Collect all distinct selected seeds before expansion, run the existing ancestry-to-root and descendants rules over the union, deduplicate roots and affected keys globally, and freeze one plan before confirmation. The BID path must not mutate inside a page visitor.

**Evidence**: `cmd/cancel_processinstance_selector.go` currently performs execution inside the `PlanProcessInstanceMutationPages` visitor. `internal/services/processinstance/dryrun.go` expands each page independently. `DryRunCancelOrDeletePlan` already deduplicates ancestry roots and descendant keys within one invocation. The current page-summary renderer sums impact counts and can double-count overlap.

**Rationale**: This directly implements the user's clarification: matches from different families, multiple seeds within one family, duplicates across pages, and relatives with other/no BIDs must yield one accurate cancellation scope. Passing the combined selection through the existing expansion once reuses its root semantics. Existing planned cancellation submits resolved roots; do not invent a new `--force` requirement solely because a selected seed is a child.

**Alternatives considered**: Merely registering `--bid` on the current paged mutation path can mutate before a later discovery error. Summing per-page previews or deduplicating only at rendering time cannot prevent duplicate submissions.

## 6. Complete traversal and the boundary of a frozen plan

**Decision**: Make v810 direct-child discovery exhaustive before traversal reports complete. Reuse service-owned paging, protect against nonadvancing cursors/offsets, and propagate partial/unresolved traversal rather than accepting a fixed-size list. Freeze submitted roots/keys and observed expanded impact; execution may recheck state and authorization but must not rediscover BID matches or silently rebuild submitted scope. Cancellation can reuse frozen-root bulk execution. Deletion requires a frozen-member executor preserving observed parent/child order and factoring existing exact-key submission/state/force/wait mechanics without recursive target discovery.

**Evidence**: `internal/services/processinstance/v810/service.go:GetDirectChildrenOfProcessInstance` currently fetches one list capped at 1000. `internal/services/processinstance/traversal` treats a nil child-fetch error as complete. Cancellation also re-observes family state when waiting after submission. In contrast, `DeleteProcessInstance` re-runs `Descendants` and recursively submits child deletions, so simply passing frozen roots to existing bulk deletion would admit new client-selected targets.

**Rationale**: A full dry-run cannot omit the 1001st sibling. Read-only post-submission verification is compatible with frozen mutation targets. Root cancellation is a native cascading operation: a frozen client plan is not a transactional snapshot or a promise that the backend cannot affect descendants created concurrently under an already targeted root. Document this limit without introducing continuous BID discovery.

**Alternatives considered**: Clipping descendants to the seed limit or a page size misrepresents impact. Reusing recursive deletion unchanged breaks frozen scope; a newly appearing child must instead remain unsubmitted, with any resulting backend conflict reported. A server-side snapshot/lock and continuous chasing of new matches are outside this feature.

## 7. Selection cardinality and command-specific scope

**Decision**: A narrow shared `ResolveProcessInstanceSelection` contract accepts collection or unique selection. Collection returns distinct matches within the documented bound. Unique selection may stop immediately upon finding a second distinct key and return an ambiguity precondition error; it may succeed with one key only after authoritative exhaustion. Zero exhausted matches produce successful empty output.

**Evidence**: Walk currently marks `--key` required and its enrichment validators also require that flag. Suspend/resume defaults target exact keys; their tree planners expand descendants, not ancestors/siblings. Existing variable-update planning and resolve execution have their own scope and action rules.

**Rationale**: Resolve BID into existing operation inputs, then reuse each operation's existing plan/execution machinery. Keep tenant-scoped discovery separate from explicit-key administrative options. Remove key-only walk validation only where a proven BID-selected seed supplies the same capability.

**Alternatives considered**: Choosing the first match, resolving a common root as uniqueness, or universally applying cancellation-family expansion would change established behavior.

## 8. Operations carry provenance separately from execution targets

**Decision**: Extend process-filter mappings in both `c8volt/process/convert.go` and `c8volt/ops/convert.go`, including reverse report mappings. Carry original selection provenance separately from operation-specific frozen executable plans. Add copied plan inputs to repair, retention, and orphan workflows so confirmation reuses observed scope rather than passing only `DiscoveredKeys` and rebuilding expansion. BID-bearing frozen-key input without completed plan evidence is invalid; legacy non-BID inputs retain their established behavior.

**Evidence**: `cmd/ops_repair.go:opsRepairConfirmedRequestFromPlan` changes mode to keyed and clears `ProcessInstanceSelection`; retention/orphan command confirmation preserves only discovered keys, and service execution rebuilds delete plans. Existing domain ops types already contain discovery, delete/repair plans, tenant evidence, and audit fields that can be reused.

**Rationale**: Reports must retain the selection used even after execution switches to frozen keys. Keep repair incident/job/variable targets tied to the reviewed repair plan. Validate plan operation, criteria, tenant scope, and readiness on consumption; clone collections at facade and plan boundaries. Plans are in-memory run artifacts, not signed tokens or a new report-import execution feature.

**Alternatives considered**: Clearing filters loses provenance; retaining a live selector for execution rediscovers work; a generic workflow engine or mutable global plan cache adds unnecessary scope.

## 9. Ops-specific filters, limits, and report paths

**Decision**: Add BID to the separate slow-analysis filter and its CLI/service validation; allow BID alone or combined with one definition selector. Preserve keyed/search exclusivity and update selection-mode/progress descriptions. For BID analysis defer preflight confirmation until an actual selected record exists, before enrichment, preserving all-empty sparse discovery without prompts. Preserve repair/retention/analysis limits on distinct matching seeds. Align orphan purge with its documented limit on distinct matching child candidates inspected before orphan classification.

**Evidence**: `cmd/ops_analyse_slow_process_instances_validation.go` and `internal/services/ops/slow_process_analysis.go` currently require a definition selector. The service's preflight can request confirmation on an empty first page unless its total is exactly zero. `internal/services/processinstance/orphan_discovery.go` stops on an empty page, counts accumulated orphan matches, and deduplicates only afterward. Existing test `TestDiscoverOrphanProcessInstancesLimitCountsOrphansNotCandidates` encodes that behavior, while command help and this spec explicitly describe inspected child candidates.

**Rationale**: Call out the orphan limit correction as an intentional compatibility change, update the conflicting test, and test a non-orphan candidate consuming the bound. Preserve report JSON schemas and add optional `businessId` to the existing selection filters; escape BID text for human/Markdown descriptions. Keep originally observed resource IDs independent.

**Alternatives considered**: Silently adopting the orphan-count behavior contradicts the current documented contract. Correcting only a BID code path would make the same flag count different units depending on selector.

## 10. Complete scope presentation and diagnostic separation

**Decision**: Render one globally deduplicated cancellation preview using the existing payload's requested keys, resolved roots, affected family keys, completeness, and mutation-submitted fields. Add optional selection provenance without replacing the envelope. Compact default human output reports full counts and expansion facts; JSON and verbose human detail expose every planned key without an implicit preview cap. Keys-only retains its established meaning with deduplicated keys.

**Evidence**: `cmd/cmd_views_processinstance_dryrun.go:newProcessInstanceDryRunPreview` already contains full arrays, and its single-preview verbose renderer lists keys. The paged summary adds counts instead of taking a set union. `cmd/cmd_views_contract.go` and `cmd/cmd_views_rendermode.go` own envelope, exit, and configured logging helpers. `cmd/run_processinstance_logging_test.go` provides a recent isolated multi-failure/output precedent.

**Rationale**: “Full scope” means no hidden truncation in the plan, truthful compact counts, and complete inspectable membership; it does not require noisy default per-key lifecycle logs. Partial discovery must remain marked incomplete even when an evidence preview exists. One error envelope preserves original error details for machines, while normal/verbose human errors use bounded aggregate reasons.

**Alternatives considered**: Raw errors on stderr bypass configured logging; a second error renderer duplicates JSON; rendering that fetches more data changes the reviewed scope.

## 11. Validation and delivery boundaries

**Decision**: Use adjacent unit tests, generated-client mocks, owned HTTP servers, subprocess output checks, and real terminal stdin fixtures. Verify the test entry points and clear inherited live-test build/config switches before execution. Start with targeted suites; run one full race suite after shared discovery/plan changes are assembled because their consumers span commands, process services, and ops. Planning itself runs document checks only.

**Evidence**: `Makefile:test` runs `go test $(PKG) -race -count=1`; the integration targets are separate and explicitly environment-backed. `testx` and existing terminal/output suites already support isolated execution. `make docs-content` runs the local docs generator from command metadata.

**Rationale**: Runnable quickstart checks must never designate or discover a live target. Documentation and diagnostic tests belong with each behavior unit, not a late polish phase. No new dependency or generic framework is required.

**Alternatives considered**: Live smoke tests, integration build tags, or manual cancellation against a cluster violate the implementation boundary. New tests merely mirroring struct assignments are insufficient to prove execution-path safety.

## Resolution status

All planning unknowns have a repository-grounded decision. The cross-family clarification is captured in this research, the plan, contracts, and validation fixtures; `spec.md` remains unchanged as requested. Real-engine race/cascade behavior remains unverified by this planning workflow and is not an implementation completion gate.
