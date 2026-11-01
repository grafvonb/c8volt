# Implementation Plan: Business-ID Filters and Process Selection

**Branch**: `361-business-id-selection` | **Date**: 2026-11-01 | **Spec**: [spec.md](spec.md)

**Input**: `specs/361-business-id-selection/spec.md`, issue #361, and the user's explicit cross-family cancellation clarification. The specification is unchanged; that clarification is made concrete in this plan and [cancellation contract](contracts/cancellation.md).

## Summary

Add exact native BID filtering to process-instance and user-task search on Camunda 8.10+, then use a shared process-service selection contract for all eight covered process commands and four operations. Preserve tenant scope, predicate combinations, native task identity, operation-specific expansion, and established output/error behavior. Generated clients already expose the required search fields; no regeneration or new dependency is planned.

BID cancellation needs more than a new flag: fully discover the bounded seed set, traverse every selected family, deduplicate the global scope, and freeze a complete plan before confirmation and mutation. Fix the fixed-size child lookup and ambiguous discovery-completion handling used by this path. Render complete compact scope counts, with all planned keys available in JSON and verbose detail. Preserve current non-BID page-by-page mutation UX rather than silently converting all commands to a new lifecycle.

Research decisions and exact source anchors are in [research.md](research.md). No unresolved technical clarification remains.

## Technical Context

**Language/Version**: Go 1.26.0; toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2, pflag 1.0.10, Viper 1.21.0, generated Camunda v810 clients pinned to 8.10.3, testify 1.11.1, standard JSON/logging, `typex.Keys`, `toolx` copying/pool helpers, and current process traversal/planning services. No new dependency.

**Storage**: Invocation-local copied filters and frozen plans; existing operation JSON/Markdown reports. No persistent selector index, new database, lock service, or plan cache.

**Testing**: Adjacent unit tests, generated-client mocks, owned HTTP servers, subprocesses, and real-terminal fixtures. Documentation checks only during this planning phase.

**Target Platform**: Existing CLI/library platforms. Native BID support in v810; explicit rejection in v88/v89 without changing the default version.

**Project Type**: CLI plus public Go facade library over version-neutral service contracts and versioned adapters.

**Performance Goals**: One service-owned discovery traversal; distinct-key accumulation proportional to selected work; each distinct family root expanded once per plan, with existing bounded worker controls. Memory is O(distinct selected plus observed affected instances), necessary for truthful frozen scope. Native matching avoids unfiltered full scans. No invented latency SLA or hidden family-size cap.

**Constraints**: No mutation before complete bounded discovery and complete required traversal; no premature tenant bypass; exact opaque values; no extra reads during rendering; stable machine envelopes and existing wait/exit semantics; configured logging; isolated-only validation.

**Scale/Scope**: Eight process commands, user-task search, four operations, three version adapters for capability policy, duplicated process/ops filter conversions, and existing report formats. Direct BID selection for jobs/incidents/elements, incident-selected ops, definition deletion/global purge, late assignment, and new version support remain excluded.

## Constitution Check

Initial gates were evaluated against the specification and repository before detailed design. The completed design is re-evaluated below.

| Principle | Initial gate | Post-design result |
| --- | --- | --- |
| I. Operational proof over intent | PASS: completed discovery and existing mutation verification required | PASS: explicit discovery disposition, exhaustive traversal, immutable submitted targets, no-wait acceptance kept distinct; native cascade race boundary documented |
| II. CLI-first, script-safe interfaces | PASS: additive scalar input on existing commands | PASS: stable envelopes, one-error rendering, mode-aware no-ops, tenant/selector exclusivity; intentional compatibility corrections documented |
| III. Proportionate validation | PASS: this phase is documentation-only | PASS: targeted owned-fixture checks per work unit, one later broad race suite for shared discovery/planning changes, no live gate |
| IV. Documentation matches behavior | PASS: new selector and traversal semantics need documentation | PASS: metadata, README, examples, and regenerated CLI docs accompany behavior, including removal of stale v810 1000-child caveats |
| V. Small, compatible, repository-native changes | PASS: existing owners and helpers reused | PASS: narrow process selection/plan contracts and typed ops plan inputs; no framework, per-command resolver, or generated-client hand edits |

No unjustified gate violation remains. Read-only re-observation after submission is permitted verification; it must not add new BID-selected mutation roots. No live-engine behavior is claimed verified by this plan.

## Project Structure

### Documentation (this feature)

```text
specs/361-business-id-selection/
├── spec.md                         # preserved
├── checklists/requirements.md      # preserved specification review
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    ├── search.md
    ├── selection.md
    ├── cancellation.md
    └── operations-output.md
```

`tasks.md` belongs to the next phase and is not generated here.

### Source Code (repository root)

```text
cmd/
├── get_processinstance_{filtering,search}.go
├── get_usertask*.go
├── {cancel,delete}_processinstance_selector.go
├── {cancel,delete}_processinstance_business_id.go     # proposed BID lifecycle files
├── {walk,update,resolve,suspend,resume}_processinstance*.go
├── ops_{repair_processinstance,repair,execute_retention_policy,purge_orphan_processinstances}.go
├── ops_analyse_slow_process_instances*.go
├── cmd_views_processinstance*.go
├── cmd_views_ops_*.go
└── ops_report*.go
c8volt/
├── process/{api,model,convert,client,dryrun}.go
├── process/selection.go                             # proposed thin delegates
├── task/{model,convert}.go
└── ops/{api,model,convert,client}.go
internal/
├── domain/{processinstance,processinstance_traversal,usertask,ops_*}.go
├── domain/processinstance_selection.go              # proposed selection/plan types
├── services/common/                                # narrow shared version/input policy
└── services/
    ├── processinstance/{search,dryrun,orphan_discovery,retention_discovery}.go
    ├── processinstance/selection.go                 # proposed selection and plan composition
    ├── processinstance/{traversal,walker,v88,v89,v810}/
    ├── usertask/{search.go,v88/,v89/,v810/}
    ├── incident/                                   # existing resolution workflow
    └── ops/{repair,retention_policy,orphan_purge,slow_process_analysis}.go
README.md
docs/cli/                                            # generated from metadata only
```

**Structure Decision**: Backend paging, eligibility, expansion, plan validation and execution remain in internal services. Facades only copy/map/delegate/convert errors. CLI code owns flags, mode dispatch, confirmation and views. New BID lifecycle logic belongs in focused selector/mode files when it has three or more declarations or its own state, rather than enlarging base command constructors. Paths marked proposed do not yet exist.

## Phase 0 — Research

Resolved native exact-match types, custom PI payload handling, task snapshot semantics, both direct-incident shortcut boundaries, old-version and frozen-key bypass gates, completion and duplicate-limit defects, cross-family planning, the 1000-child cap, command-specific scopes, operation confirmation handoffs, report propagation, and the orphan limit compatibility discrepancy. [research.md](research.md) records decisions, alternatives, and source evidence.

## Phase 1 — Design

### Filter path and capability policy

Use optional `BusinessId *string` criteria with `json:"businessId,omitempty"` in public filters, and equivalent internal fields. Populate them only when `cmd.Flags().Changed("bid")`; preserve the raw scalar value. Reject an explicit empty value and invalid UTF-8 at authoritative service boundaries. Do not apply creation-specific length limits or treat a comma as a list separator.

Update both directions of process filter conversion in both process and ops facades, task request conversion, slow-analysis filter conversion, descriptions, and report copies. Use `toolx.CopyPtr` at input/plan boundaries. v810 PI custom payload and `isEmpty()` must retain a BID-only filter. Use the scalar exact alternative of the generated string union for PI and task native requests.

Add a no-I/O `ValidateProcessInstanceSearchFilter` contract to the internal PI API and each adapter, reusing existing normalization/state validation plus the narrow BID policy. Invoke it at page-search/shared-selection entry and before consuming any BID-bearing frozen process plan, including empty or externally supplied plans. v88/v89 reject supplied BID before HTTP; a plan prepared with v810 cannot bypass that guard when passed to an older adapter. Update all adapter implementations and affected test doubles together. Ops also validates version/input before any existing `DiscoveredKeys`, frozen-plan, or empty-result branch using its known version. Both command and service disable the incident-index optimization for BID, falling back to native PI search plus existing direct-incident checks. This prevents a caller-provided shortcut flag from dropping the predicate.

### Shared selection and complete traversal

Add proposed `ResolveProcessInstanceSelection` as a narrow process service function with a thin public facade equivalent. It reuses the existing process search engine rather than implementing another page loop. Extend that engine's result with explicit completion evidence, validate metadata, preserve raw-page progress separately from selected count, continue through sparse pages, detect nonadvancing cursors, and deduplicate before a selected-instance limit is consumed. Exact totals remain usable only for fully native compatible predicates; local filtering uses complete counted traversal.

Collection mode accepts authoritative exhaustion or a documented bound reached without error. A bound caps distinct seeds, never the family expansion. Unique-start mode succeeds only after exhausted discovery leaves exactly one distinct key; a second distinct key is enough to fail ambiguity immediately. A limit or early stop is never uniqueness evidence. Preserve partial evidence with original errors and mark it non-executable.

Change v810 `GetDirectChildrenOfProcessInstance` to exhaust direct-child pages before declaring traversal complete; do not truncate at 1000 or at the BID seed limit. Propagate child-page failures, unsupported continuation, missing ancestry and partial traversal. Reuse existing traversal and worker controls; scope this correction to the shared paths needed by the feature and retain older-version behavior unless the same shared fix necessarily applies, with regression tests.

### Frozen cross-family cancellation and deletion

Add proposed `PlanProcessInstanceSelectedMutation` to compose collection selection with `DryRunCancelOrDeletePlan` over the complete seed union. Retain selected observations, ordered distinct keys, roots, affected keys, tenant evidence, selection criteria, and explicit completion. Preserve root-to-family associations needed to explain overlap. Validate that discovery and required traversal completed before the plan can execute.

For BID cancel/delete, use dedicated CLI dispatch into this full-plan path. Do not call mutations from `PlanProcessInstanceMutationPages` visitors. Validated cancellation execution submits only frozen distinct roots through existing bulk cancellation. Deletion needs a plan-aware executor over frozen affected members and their observed parent/child order: the current `DeleteProcessInstances`/`DeleteProcessInstance` path recursively discovers children again and cannot be reused unchanged. Factor and reuse exact-key delete submission, state/force handling, retry and wait mechanics without a recursive lookup that adds targets. Preserve existing children-before-parent ordering and preconditions. A newly appearing child must not receive an unplanned client delete request; any resulting backend conflict is an explicit failure. Genuine action-time changes may fail or shrink successful outcomes, never silently enlarge the client-selected scope.

The [cancellation contract](contracts/cancellation.md) defines a multi-family fixture and exact expected union. Dry-run and execution each independently discover their scope. The submitted plan is frozen within a run; native cascading root actions and later verification can observe concurrently created descendants. This design promises no new BID discovery after approval, not an atomic backend snapshot.

### Other process commands

- `walk pi`: replace unconditional required-key metadata with key/BID exclusivity and one-of validation. Enrichment validators accept a proven BID-selected seed. Use existing family/ancestry/descendant traversal and existing empty-result views.
- `update pi`: select once and pass all distinct keys to existing variable plan/execution methods; preserve each selected instance's variable scope and mutation payload. No automatic family expansion.
- `resolve pi`: select and expand the existing process resolution scope once; reuse incident service mechanics. Any incident/job/variable targets represented in a confirmation plan must also be frozen before execution, extending planning inside the owning incident service if required. Do not add a direct incident BID selector.
- `suspend/resume pi`: resolve a collection and reuse exact-key bulk execution by default. `--tree` reuses existing descendant-only plans and their frozen targets; no cancellation-style ancestor/sibling expansion. Empty selections bypass keyed input errors and use successful empty views.

Do not pass `collectExplicitPIAdminInputOptions` into BID discovery: it appends ignore-tenant behavior intended for explicit keys. Carry selection origin or separate discovery and execution options through shared helpers. Preserve actual explicit-key authorization; after scoped discovery, existing key-based traversal/action authorization and tenant evidence remain deliberate, tested transitions.

### Operations and audit provenance

Reuse the shared process selector while retaining each operation's local eligibility: active-incident repair, retention's derived end-date boundary, and orphan-child checks. Add BID to slow analysis's separate filters and both CLI/service validators. Use an additive `process_instance_search` mode for BID-only analysis; retain `process_definition_search` for existing definition selections, including definition+BID. Update progress and selector summaries for both search modes. For BID analysis, defer preflight confirmation until an actual matching record has been selected; continue sparse empty pages without prompting, and confirm before enrichment or expensive follow-on processing. Preserve wording, routing, automation and abort behavior. Unknown totals on an empty first page cannot justify a prompt for a selection that ultimately exhausts empty.

Introduce typed, copied `FrozenPlan` inputs for the three mutating ops using existing discovery/repair/delete-plan types. A confirmed execution retains original selection provenance separately from executable targets. It does not clear BID or rebuild an expanded plan from only discovered keys. Validate operation, criteria, tenant context and plan readiness before consuming it; preserve action-time checks and failure evidence. Frozen plan input is invocation-local and excluded from recursive request/report serialization. Report fields continue to carry the selection and observed plan through their current schemas.

Repair must preserve its reviewed incident/job/variable work as well as selected PIs. Retention and orphan execution consume reviewed roots/affected scope through the same frozen-member deletion executor, not the existing recursively rediscovering delete path. Slow analysis remains read-only and uses the same trustworthy discovery status. Filter descriptions and all supported JSON/Markdown paths carry the exact criterion without assigning it to related resource observations.

Align orphan `--limit` with the documented unit of distinct matching child candidates inspected before orphan classification, including non-orphans. Update the contradictory existing orphan-count test and document this intentional correction. Other limits remain matching seeds frozen for repair, retention, or analysis; page size remains a per-request control.

### Output, failure flow, and documentation

Render BID cancellation as one complete deduplicated preview, reusing existing requested/root/affected fields and adding optional selection provenance. Default human INFO output presents full counts and expansion/completeness facts; verbose detail and JSON expose the entire planned key union. Existing keys-only meaning remains unchanged, with no duplicate lines and zero bytes for no keys. Do not sum per-page counts to infer unique impact. Do not create a new report family or change existing null/empty conventions.

Trace every changed failure: adapter/service error plus partial plan/outcomes → facade preserves both and converts error → command view renders one selected-format result → configured logger emits one bounded aggregate human error where applicable → `exitAfterRenderedResult` retains the original failure class. Never call a second error renderer after an envelope. A successful no-op uses a successful envelope even under no-wait; absence of submitted work cannot become accepted.

Human summaries, dry-run and verbose outcomes use `renderHumanLine` with the configured command logger; diagnostics never use direct stderr prints. Normal/verbose failures omit raw HTTP details; redacted DEBUG remains independent of verbose. Prompts remain plain configured stderr text. Update metadata, README, examples and generated docs in behavior-delivery units, including full-scope inspection, version support, bounds, ambiguity, no-op behavior and native cascade timing. Run `make docs-content`; do not hand-edit generated docs.

### Compatibility decisions

Additive optional fields preserve absent-field output and existing workflows without BID. New public facade methods require implementing test doubles to change with their interfaces; keep those changes in the same compiling work unit. Document external custom implementations and unkeyed struct literal impacts.

BID mutation uses full planning rather than legacy paged mutation/confirmation, intentionally. Existing non-BID prompt policy remains intact. BID slow-analysis preflight is deferred past sparse empty pages to honor prompt-free empty completion. Corrected shared v810 traversal can expose previously omitted descendants and removes the documented 1000-child limitation; help and tests must reflect this broader but intended traversal. Orphan limits now follow the documented candidate unit, including for non-BID calls to the corrected shared discovery. The slow-analysis mode addition is additive and must be covered by serialization consumers.

## Validation Strategy

[quickstart.md](quickstart.md) contains isolated implementation validation commands and scenario expectations. This turn checks documents only.

Require actual command execution for filters, modes, errors and prompts. Cover native payloads, conversion copies/round trips, both direct-incident bypass attempts, version rejection before requests, BID-only slow analysis, tenant isolation, sparse and duplicate pages, unknown/cyclic continuation, >1000-child traversal, unique walk, and all per-command scope rules. Full cancellation fixtures must prove global union, one expansion per root, no mutation until discovery/expansion complete, and no rediscovery after confirmation.

Use output matrices for human/verbose/JSON/keys/quiet and supported automation/auto-confirm/no-wait/dry-run combinations. Decode exactly one JSON envelope and require EOF. Exercise repeated distinctive backend errors with configured/inherited logger destinations, plain and structured formats, WARN/quiet suppression, flag/config DEBUG and redaction, and original exit codes. Assertions must fail on direct-print routing or raw-error substitutions. Real terminal stdin must prove empty scopes never prompt, while nonempty approval/decline/EOF behavior is unchanged.

Run targeted tests with each behavior unit, then one full race suite after the shared runtime changes pass; the broad run is justified by cross-package search, traversal, and planning changes. No live integration tests, environment discovery, cluster probes, provisioning, or manual engine validation are part of implementation. Preserve any unverified engine behavior as a coverage limitation.

## Work-Unit Guidance for Task Generation

Generate cohesive compiling outcomes with fixtures, tests, formatting, documentation and targeted checks included. Suggested boundaries, not a tasks list:

1. **Native search and query delivery (US1)**: optional models, all required converters, adapter policy/native payloads, shortcut guards, PI/task CLI flags and exclusivity, totals/enrichment tests and docs. Keep field/interface/converter/test-double changes together; split by resource only if each resulting slice is independently complete.
2. **Shared selection and unique walking (US2)**: explicit completion and distinct limits, narrow selection facade, complete v810 child traversal, walk validation/enrichment, isolated selection/large-family tests and affected help. These are reused by later mutation paths.
3. **Frozen cancellation/deletion (US3)**: full-plan composition and validated execution, cross-family dedup and complete previews, late-discovery failure safety, focused CLI lifecycle files, full output/terminal matrix and docs. Build collection behavior once; do not deliver a single-match shortcut first. If practical size requires splitting cancel/delete, retain shared plan contracts in the first cohesive slice.
4. **Other process selection (US3)**: update and resolve retain their own plan scope, suspend/resume exact/tree integration, and source-aware options. Split along distinct operation planners if needed; each unit must include its complete failure/output path and tests.
5. **Maintenance operations (US4)**: repair, retention and orphan selection, typed frozen confirmation plans, provenance/report round trips, compatibility correction for orphan limits, and execution tests. Split repair from cleanup if necessary; do not split a confirmed-plan input from its consumers.
6. **Slow-process analysis (US4)**: separate filter/mode changes, BID-only and combined discovery, progress/view mappings and isolated command tests.

US5 compatibility, diagnostics, empty results and documentation are obligations within each outcome, not delayed polish tasks. Context reading, branch checks, commits, and progress tracking are work-unit responsibilities. Review adjacent units for consolidation during task generation; these separations reflect independently verifiable query, uniqueness, mutation, maintenance and analysis outcomes. A count above ten requires review, not reduced coverage or oversized units. Ralph iterations must keep the permitted single-primary-story scope.

## Complexity Tracking

No constitution exception is required. A copied frozen scope and a narrow selection facade are necessary because the existing per-page callbacks cannot establish completeness before mutation. Reusing existing search/traversal/bulk services is simpler than a generic selector framework, persistent plan registry, or separate resolver per command. Typed ops plan handoff corrects the specific confirmation gap without redesigning unrelated operations.
