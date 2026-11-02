# Data Model: Business-ID Filters and Process Selection

**Feature**: [spec.md](spec.md) | **Design**: [plan.md](plan.md)

These are implementation design contracts. Proposed types and fields are additive unless an explicit compatibility correction is stated. Public models expose public facade types, never generated clients or internal domain types.

## Business-ID criterion

| Owner/model | Addition | Mapping and invariant |
| --- | --- | --- |
| `internal/domain.ProcessInstanceFilter` | `BusinessId *string` | nil means omitted; pointer to empty is invalid |
| `c8volt/process.ProcessInstanceFilter` | `BusinessId *string`, `json:"businessId,omitempty"` | Copy in both directions, including duplicated ops converters |
| `internal/domain.UserTaskSearchQuery` | `BusinessId *string` | Native task-level exact matching |
| `c8volt/task.SearchRequest` | `BusinessId *string`, optional JSON field | Copy in `toDomainSearchRequest`; do not trim |
| Public/domain `SlowProcessAnalysisProcessInstanceSearchFilters` | `BusinessId *string`, optional public JSON field | Copy in both directions and into ordinary PI search filters |
| v810 custom PI filter | `BusinessId *camundav810.StringFilterProperty` | Exact scalar variant; contributes to `isEmpty()` |
| v810 native task filter | Existing generated `BusinessId` field | Exact scalar variant; no process-instance identity join |

Validation order: reject unsupported versions for supplied BID, then reject explicit empty/invalid UTF-8 on supported paths. CLI can reject local empty input early; direct library calls still receive authoritative validation. Preserve all valid string bytes, including leading/trailing spaces, commas, Unicode, numeric-looking values and literal wildcard characters. No creation-length constraint is introduced. Values in text descriptions are escaped/quoted; serialized selection values remain exact.

The proposed internal PI API method `ValidateProcessInstanceSearchFilter` performs no I/O and is implemented by all adapters. Fresh searches and frozen process-plan executors invoke it against the receiving adapter; a prebuilt plan cannot bypass an older version's rejection. Ops validates its known service version before its own bypass paths as well.

## Process discovery evidence

Extend the existing `ProcessInstanceSearchPagesResult` with a typed completion disposition using the user-task precedent. Mirror the additive public result field mechanically. Existing `Items`, `Limit`, and `Pages` remain meaningful.

| Disposition | Meaning | Allowed use |
| --- | --- | --- |
| `exhausted` | Authoritative exhaustion established for the query | Empty result, collection selection, or uniqueness proof |
| `limit_reached` | A documented distinct-selection bound was reached without error | Complete bounded collection; never uniqueness proof |
| `visitor_stopped` | Caller chose to stop early | Query partial results only; not mutation-ready or uniquely resolved |
| `incomplete` | Failure, interruption, malformed/nonprogressing continuation, or unresolved completion | Evidence only, returned with original error; never mutation-ready |

An unset/unknown disposition is not success. Keep raw page counts/cursor progress separate from selected counts, preserve lower-bound versus exact total metadata, and validate continuation before declaring completion. Errors retain partial evidence where the contract exposes it; they never become empty success. For existing ordinary query consumers, additive completion evidence must not change interactive continuation policy accidentally.

## Proposed process selection request and result

`ProcessInstanceSelectionRequest` contains the existing `ProcessInstanceSearchRequest` plus cardinality `collection` or `unique_start`. Its search filter carries BID, other predicates, configured tenant scope through options, ordering and any supported seed limit.

`ProcessInstanceSelection` contains:

- A copied search criterion/provenance snapshot; operation-specific eligibility filters must remain visible where they are applied.
- Ordered distinct selected observations and keys, preserving each observed resource's own `BusinessId` or nil.
- Completion disposition, page count, documented bound and whether it was reached.
- Existing tenant evidence and selection origin (`search` versus actual explicit-key input) sufficient to prevent premature ignore-tenant options.

Deduplicate by technical key, not BID or root. Two distinct keys sharing a BID remain two selections. Bound counting occurs after applicable seed filtering and key deduplication; raw traversal progression is unaffected. Unique-start never treats a limit as exhaustive; it returns a precondition/ambiguity error on a second distinct key and otherwise requires exhaustion.

## Proposed selected mutation plan

`ProcessInstanceSelectedMutationRequest` identifies cancel/delete and contains the selection request plus existing expansion worker/options controls. `ProcessInstanceSelectedMutationPlan` carries:

| Field group | Meaning |
| --- | --- |
| Operation and original selection | Copied criteria, selected observations/keys, source and discovery completion |
| Existing `DryRunPIKeyExpansion` | Distinct roots, affected keys, final/non-final observations, missing ancestry, warnings, outcome and tenant evidence |
| Family membership and deletion order | Root-to-observed-key and parent/child associations retained from traversal, plus the existing children-before-parent order for exact planned deletion; no new reads needed to reconstruct scope |
| Readiness | Derived from valid collection completion plus complete required expansion; not trusted solely from a caller boolean |

Do not reuse a single completeness boolean for both global search exhaustion and bounded execution readiness. A limit-reached seed selection can be complete within its bound while additional matches exist globally; expansion for those selected seeds must still complete without clipping. Missing or unresolved traversal makes the plan non-executable.

Copy pointer values, slices, maps and operation options crossing facade/plan boundaries. Reject operation/criteria/tenant mismatches when consuming a plan; preserve backend authorization and action-time state checks. The plan is not a permission token, persistent lock, or imported report format. Execution uses the frozen roots/targets, never a fresh BID search.

## Existing operation plans retained

| Operation | Frozen executable scope | Provenance that must survive |
| --- | --- | --- |
| Cancel | Resolved roots plus reviewed observed family impact | BID-selected seeds versus expanded members, bound/completion and tenant evidence |
| Delete | Frozen affected members and observed children-before-parent order; exact planned submission without recursive live discovery | BID-selected seeds versus expanded members, bound/completion and tenant evidence |
| Variable update | Existing per-instance variable-update plans and payloads | Selected instance observations and original criteria; no automatic family expansion |
| Resolve | Existing resolution PI/family scope and any incident work advertised for confirmation | PI selector and reviewed incident/job/variable targets; no incident-native BID input |
| Suspend/resume default | Exact selected distinct instances | Search origin and exact-scope selection |
| Suspend/resume tree | Existing descendant plan `TargetKeys` and state classification | Original seeds versus descendants; no ancestors/siblings |
| Walk | One exhausted, distinct starting key or successful empty result | Original criterion and unique-selection evidence |

## Typed ops plan handoff

Add an optional copied `FrozenPlan` input to public/domain repair, retention and orphan requests, with a dedicated operation-specific plan type built from existing discovery, frozen-set and action-plan types. Mark executable plan input `json:"-"` where request models are serialized to prevent recursive/duplicated audit documents. Keep exposed report criteria and plan evidence in their established fields.

- Repair plan: original `ProcessInstanceSelection`, selection mode, tenant evidence, frozen PI/incident/job/variable scope and reviewed repair actions. Confirmation must not erase criteria or rediscover newly appearing incidents into execution.
- Retention plan: original selection and fixed derived retention boundary, discovery evidence, `RetentionDeletePlan`, roots and affected scope, frozen deletion membership/order, skips/blockers and relevant execution controls.
- Orphan plan: original selection, distinct child candidates inspected, resulting orphan seeds, completed expansion, reviewed delete membership/order and tenant evidence.
- Bare `DiscoveredKeys` is not proof of a complete BID plan. Reject BID plus an unproven frozen-key bypass before execution; preserve existing non-BID semantics.

For a normal one-shot library invocation without a supplied plan, the service performs selection and planning before mutation. For command confirmation, planning returns the typed scope and execution consumes that same copied scope. Keep the same capability/input checks on both entry paths.

Retention/orphan use the same frozen-member deletion executor as BID delete. A child appearing after approval is not added to client delete requests; changed backend conditions produce the established explicit failure/partial result. Native root cancellation during existing force handling retains its documented cascade boundary.

## Slow-analysis selection mode

Retain `explicit_keys` and `process_definition_search` for existing requests. Add `process_instance_search` for BID-only requests. Definition+BID retains definition-search mode and combines filters with AND. Both search modes share discovery; all progress, validation, selector summaries and serialization recognize the additive mode. No new Markdown report capability is introduced where none exists.

## Reports and command payloads

Preserve existing JSON envelope and payload shapes, omission/null/empty-array rules, schema identifiers and machine error evidence. Add optional `businessId` to existing filter objects. Add optional selection metadata to BID preview payloads so the selector is separate from observed resource identity; legacy nil metadata remains omitted.

Cancellation preview retains `requestedKeys`, `resolvedRoots`, `affectedFamilyKeys`, their counts, `traversalOutcome`, `scopeComplete`, missing ancestry, and `mutationSubmitted`. Counts use global distinct sets, not summed pages. `scopeComplete` refers to required expansion; selection completion/bound metadata separately identifies user-limited discovery. Full arrays are never truncated for display. Empty completed cases use each command's existing empty payload, not invented report entries.

## Lifecycle

```text
supplied input -> capability/value/selector validation
              -> tenant-scoped discovery
              -> exhausted OR documented bounded selection
              -> operation-specific planning/complete expansion
              -> copied frozen plan
              -> dry-run output OR confirmation/automation
              -> existing action-time checks and frozen-target submission
              -> accepted (no-wait) OR verified/partial/failed outcomes
```

Any discovery/expansion failure goes to non-executable failure with evidence. Authoritative empty discovery goes directly to successful empty output without planning side effects, confirmation or mutation. User abort is its existing abort outcome, never an empty-selection success. Read-only post-submission verification may observe later backend facts; those facts do not rewrite the original plan or trigger new BID discovery.
