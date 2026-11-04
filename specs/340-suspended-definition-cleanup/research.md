# Research: Suspended Instances in Definition Cleanup

**Date**: 2026-10-23
**Baseline**: `dfacfa09` on `340-suspended-definition-cleanup`
**Scope authority**: [spec.md](spec.md), including its narrowed scope and isolated-validation boundary.

Research used repository source, generated client definitions, and existing tests. Two focused read-only research agents examined cleanup and statistics/output ownership. No tests, live services, or GitHub mutations were used. No external technology choice or dependency upgrade is needed.

## 1. Extend the existing workflow

**Decision**: Keep orchestration in `internal/services/processdefinition/delete.go` and `internal/services/ops/all_process_definitions_purge.go`. Replace active-only decisions with unfinished-work decisions while retaining existing instance operations, cleanup order, root deduplication, workers, and opt-outs.

**Evidence**: `completeProcessDefinitionDeleteImpact` skips expansion when `ActiveProcessInstances()` is zero; `listActiveProcessInstancesForDefinition` filters ACTIVE; both drain waiters reload active counts. Purge's `buildAllProcessDefinitionsPurgeDeletePlan` and affected-count fallback also use active only. Bulk cleanup already combines unique roots before cancellation and history deletion.

**Rationale**: The missing state is in selection and verification. #341's completed work preserves the existing cancellation behavior; it does not require a new engine.

**Alternatives considered**: New mutation framework, shared execution ledger, new approval protocol, preemptive changes to instance cancellation. Rejected as unnecessary and outside the agreed scope.

## 2. Use one unfinished predicate for safety decisions

**Decision**: On 8.10, use `state in [ACTIVE, SUSPENDED]` for authoritative unfinished counts and force discovery. Do not use the sum of separately sampled display buckets as the zero-work gate.

**Evidence**: `internal/domain/processinstance.go` currently has a singular `State` filter. Generated 8.10 `AdvancedProcessInstanceStateFilter` already supports `$in`; no regeneration is needed. The 8.10 definition adapter already owns state-count requests and exact-total/capped-total handling in `countProcessInstancesForDefinitionSearch`.

**Rationale**: With separate ACTIVE and SUSPENDED reads, a SUSPENDED-to-ACTIVE transition between reads can make both counts zero. A combined predicate retains membership across either direction of transition. It does not provide snapshot isolation against starts, completion, or exporter lag.

**Design**: Add an internal-only `States` selector to the domain instance filter, leaving the public facade filter and CLI unchanged. Validate mutually exclusive singular/plural selection, normalize/deduplicate explicit states, and reject unsupported multi-state input before I/O in older adapters. Update service search validation to inspect all requested states, not just `State`. Map plural input to the existing generated advanced filter in v810. Definition cleanup selects this union only when existing suspension capability validation permits it; an unsupported capability uses the unchanged singular ACTIVE path, while other errors propagate.

**Alternatives considered**: Two state searches plus reconciliation; scanning all historical states and filtering locally; new public search APIs. Rejected because the generated combined predicate is available and more focused.

## 3. Keep authoritative counts internal to definition statistics

**Decision**: Add an internal nullable `Unfinished` count to domain definition statistics, excluded from serialization. The v810 adapter observes it with the combined predicate inside the existing statistics acquisition. Public statistics expose separate active/suspended buckets only. Cleanup plans carry an additive unfinished count for decisions and reporting.

**Evidence**: `getProcessDefinitionDeletePlanBase` and both existing drain waiters already use definition metadata with `WithStat`. Definition API signatures need no change. `SearchProcessInstancesTotal` exists as a free helper, but moving the checks there would require extra instance-service plumbing through existing private wait paths.

**Rationale**: Keeping the current ownership and `WithStat` boundary avoids a new counting API and preserves no-state-check's read-free preview. A pointer distinguishes absent evidence from an observed zero. On a suspension-capable result, missing or failed union observation is an error, never a fallback to arithmetic. Older adapters retain active-only acquisition; the existing active count supplies their unfinished count at the cleanup boundary.

**Cost**: A v810 statistics acquisition adds one suspended bucket query and one union query; existing capped-total fallback may page. No-stat reads and no-state-check previews must acquire neither. Public statistics watch therefore also incurs these queries when statistics are enabled; this is the simple reuse choice, with no cache or new public option.

**Alternatives considered**: A public unfinished-statistics schema, a new service method, or summing display counts. The first two broaden interfaces; the last can produce false zero evidence.

## 4. Preserve completeness of the changed searches

**Decision**: Reuse the current page and exact-total conventions. Exact backend totals can answer count queries directly. A capped total requires complete enumeration or an error. Union force discovery visits all pages and deduplicates keys before existing root planning.

**Evidence**: Current cleanup discovery errors on an empty page before inspecting its continuation cursor. The v810 instance overflow classifier treats empty pages as complete early. The definition statistics fallback returns a partial total on cursor stalls/disappearance. These behaviors need focused handling only where they could undermine the new cleanup predicate.

**Rules**: An empty page with authoritative continuation and an advancing cursor must continue. Missing/repeated continuation with no safe progress, or indeterminate completion, cannot become a successful complete cleanup scope. `HasMoreTotalItems` is a capped-total indicator, not by itself a next-page flag; an end cursor alone also does not prove more results. Preserve genuine final empty pages and existing offset fallback. Exercise the specific metadata combinations using fakes; do not introduce a general pagination framework or redesign unrelated definition candidate discovery.

**Alternatives considered**: Returning partial union counts as exact or treating every empty page as final. Rejected because both can falsely satisfy a destructive prerequisite. Broad traversal hardening is excluded.

## 5. Add fields without changing active semantics

**Decision**: Follow the incident-count support convention: add `Suspended` and `SuspendedCountSupported` to public/domain definition statistics. Add suspended and unfinished counts plus unfinished keys to cleanup plans; retain all existing active fields as active-only. Aggregate counts and report projections must carry the new values mechanically.

**Evidence**: `internal/domain/processdefinition.go`, `c8volt/process/model.go`, and `cmd/cmd_views_processdefinition.go` already model incident support separately and omit its human bucket when unsupported. Plan and statistics conversions are duplicated in `c8volt/ops/convert.go` as well as the resource/process facades.

**Rationale**: Supported zero and unsupported data must be distinguishable. A backend failure is not a zero or a capability downgrade. A stable fixture's unfinished count equals active plus suspended, but separate observations during change need not be simultaneous; the union count remains authoritative for cleanup.

**Human output**: Append compact `su:` within existing statistics output only when supported; use existing `zeroAsMinus` formatting. Include value and support in watch comparison/projection so a suspended-only change refreshes and stale data clears. Use unfinished wording with a compact active/suspended breakdown in cleanup summaries and blockers. No new default lifecycle column.

**Alternatives considered**: Repurposing active fields, presenting unsupported zero, changing report schema version for additive fields, or adding DRAINING here. Rejected for compatibility or scope reasons.

## 6. Preserve controls and validation boundaries

**Decision**: Preserve full-history capability checks (8.9+), no-state-check's existing bypass, no-wait's final-confirmation opt-out, nullable delete responses, batch polling, and existing partial-result handling. Required unfinished draining remains before dependent deletion even with no-wait. New work encountered by verification is not silently added to cancellation.

**Evidence**: `PreviewDeleteProcessDefinitionsWithWorkers` returns before statistics/discovery when state checks are skipped. Existing force cleanup calls cancellation, mandatory drain verification, instance history deletion, then definition deletion. Purge's CLI exposes no new state-check bypass for this feature.

**Validation**: Isolated fake-client, HTTP, command, and terminal tests only. Documentation-only planning gets document checks, not runtime tests. The original GitHub live target is not an implementation gate; #342 remains separate.

All design unknowns are resolved for Phase 1. Remaining limits are observational backend timing and real-engine behavior, neither of which is claimed proven by this plan.
