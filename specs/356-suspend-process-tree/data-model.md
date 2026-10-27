# Data Model: Process-Instance Tree Suspension

**Feature**: [spec.md](spec.md) | **Research**: [research.md](research.md)

## Frozen planning model

Introduce corresponding internal/domain and public process models with mechanical conversion. Public models expose no generated-client or internal types.

### ProcessInstanceSuspensionPlan

| Field | Public type / JSON name | Meaning |
| --- | --- | --- |
| RequestedKeys | `typex.Keys` / `requestedKeys` | All normalized unique explicit seeds, in input order, including seeds that overlap earlier subtrees. |
| Instances | `[]ProcessInstanceSuspensionPlanItem` / `instances` | Unique discovered members in first-seen traversal order. |
| TargetKeys | `typex.Keys` / `targetKeys` | Frozen eligible mutation keys in the same order as their entries in Instances. |

### ProcessInstanceSuspensionPlanItem

| Field | Public type / JSON name | Meaning |
| --- | --- | --- |
| Key | `string` / `key` | Discovered instance identity. |
| ObservedState | existing `process.State` / `observedState` | Planning observation; never post-submission confirmation evidence. |
| Classification | `ProcessInstanceSuspensionPlanClassification` / `classification` | `eligible`, `already_suspended`, or `excluded`. |

Classification is service-owned: ACTIVE is eligible, SUSPENDED is already_suspended, all other valid observed domain states are excluded. Unknown/unrecognized state values normalize to UNKNOWN; a missing Chain record or mismatched identity is a discovery error, not an UNKNOWN instance.

Do not store redundant counters, full ProcessInstance objects, traversal edges/maps, ancestor/root metadata, rendered messages, per-member transport errors, or timestamps. Derive selected, discovered, eligible, already-suspended, and excluded counts from the frozen plan only. No new storage, opaque plan identifier, or generic plan framework is needed.

## Invariants and ownership

1. Check supported suspension capability before descendant I/O. Normalize every seed with existing suspension key validation and require nonempty input.
2. Validate local execution options before discovery using the existing first-seed dry-run. A successful local preview does not classify the seed's state.
3. Each accepted traversal must be a complete descendant result for its requested seed and include that seed. Every returned key must be locally valid and have a matching identity observation.
4. Merge seed/traversal order, retain the first observation for each key, and preserve all normalized seeds in RequestedKeys. Do not reread or reclassify overlapping members to reconcile races.
5. TargetKeys is exactly the ordered projection of Instances classified eligible. Each target appears once; parent/child identities remain distinct.
6. Successful planning exposes non-nil arrays. Any discovery, malformed-response, explicit partial/unresolved-result, or context failure returns a zero plan plus error. The caller must check error before considering a no-op.
7. The facade copies input keys, TargetKeys, RequestedKeys, and Instances; no mutable service backing arrays escape. The command copies TargetKeys before confirmation and keeps the frozen plan for final reporting.
8. The plan is not a server snapshot transaction. New descendants or later states cannot enlarge or rewrite the approved key set.
9. A zero-length TargetKeys array is valid only after successful planning. Already-suspended and excluded members remain visible in Instances.

## Execution evidence remains unchanged

Retain existing `SuspensionResult` / `SuspensionResults` and corresponding domain types. Those represent eligible execution targets only, including accepted, confirmed, failed, timed-out, interrupted, and unattempted work. Existing evidence fields and retry semantics are unchanged.

Preview uses one existing preview item per eligible key, with `status=preview`, `submission=not_submitted`, `confirmation=not_requested`, and no post-submission observation. Preview entries are formatting of a plan, not backend execution. An empty actual execution has `items: []`.

Already-suspended and excluded members never become fake accepted, confirmed, or skipped mutation items. Existing `not_attempted` remains reserved for an eligible target whose execution did not start.

## Tree-only command payload

The view owns a typed payload with:

- `items`: the existing eligible preview or mutation result array, always an array.
- `plan`: the frozen public planning model above.

The shared envelope is unchanged. The extra plan member appears only for explicit tree mode. Exact-key `SuspensionResults` and its JSON shape stay unchanged. Planning failures do not expose an executable partial plan.

## State transitions

`input -> capability/local validation -> discovering -> complete plan -> preview | successful no-op | approval -> existing collection execution -> final result`

- A failure before complete planning ends without mutation or approval.
- Dry-run ends at preview, even with no-wait.
- An empty eligible plan ends as a succeeded no-op, even with no-wait; it never passes through approval or mutation execution.
- Approval decline ends with the existing abort error and no mutation.
- Approved active targets use existing submission and bounded confirmation transitions. Accepted-but-unconfirmed failures retain acceptance; rejected/uncertain submissions are not reconciled to success from discovery or later state.

The retained lookup bound is up to 1,000 direct children per parent. No new beyond-limit completeness inference, pagination state, atomic rollback, or second discovery is introduced.
