# Contract: Orphan-Parent Traversal and Preflight Handling

## Shared Traversal Contract

Affected traversal and dependency-expansion flows must use one shared outcome contract:

| Situation | Required behavior |
|-----------|-------------------|
| Full ancestry/family resolves | Return normal traversal result with no warning |
| Non-start ancestor missing and at least one actionable result resolved | Return partial result, machine-readable missing ancestor keys, and a user-facing warning |
| No process-instance data resolved at all | Return a normal failure, not warning-based success |

The contract must not require each command to rediscover orphan handling independently.

The shared implementation seam may expose the contract through result objects such as `TraversalResult` and `DryRunPIKeyExpansion`, as long as legacy tuple-based callers can be migrated incrementally without changing the contract itself.

## Machine-Readable Warning Contract

Every affected traversal/preflight flow must expose the same structured warning semantics:

| Field | Contract |
|-------|----------|
| Resolved keys | Keys that were actually resolved and remain actionable |
| Resolved chain/edges | The partial traversal data that can still be rendered or consumed |
| Missing ancestor keys | Machine-readable parent keys that could not be resolved |
| Warning message | Human-facing explanation that the tree/family result is incomplete |
| Outcome status | `complete`, `partial`, or `unresolved` |

Warning text alone is insufficient.

## Success Boundary Contract

| Outcome | Required command behavior |
|---------|----------------------------|
| `complete` | Normal success |
| `partial` with at least one actionable result | Success with warning |
| `unresolved` | Normal failure |

This success boundary applies to:

- `walk pi --parent`
- `walk pi --family`
- `walk pi --family --tree`
- cancel preflight and force-root expansion
- delete preflight and indirect process-definition cleanup expansion

## Strict Non-Regression Contract

The feature must not change these behaviors:

| Flow | Required preserved behavior |
|------|-----------------------------|
| `get process-instance --key ...` | Still returns the normal strict `not found` error when the target is missing |
| Wait for `absent` / `deleted` | Still treats disappearance as success only in the waiter contract |
| Other direct single-resource lookup paths | Remain strict and do not emit traversal warning metadata |

## Version Contract

| Version | Traversal expectation |
|---------|-----------------------|
| `v8.7` | Shared walker partial-result behavior applies to traversal/preflight; existing strict direct lookup/state constraints remain unchanged |
| `v8.8` | Shared walker partial-result behavior applies to traversal/preflight |
| `v8.9` | Shared walker partial-result behavior applies to traversal/preflight |

The feature is incomplete if one supported version still fails hard on orphan-parent traversal while the others return the new shared partial-result contract.

## Rendering Contract

For human-oriented walk output:

- partial ancestry/family results must still render the resolved data
- tree rendering must continue when family edges exist for the resolved subset
- the warning must clearly state that the tree is incomplete because one or more parent process instances were not found

For machine-readable output:

- resolved keys and missing ancestor keys must be distinguishable without parsing warning text
- partial vs unresolved outcomes must be explicit

## Preflight Contract

For cancel/delete and indirect cleanup expansion:

- resolved family keys remain actionable
- missing ancestor keys remain visible to the caller
- the command must not fail solely because an ancestor is missing when actionable results exist
- the command must still fail normally when nothing could be resolved

## Regression Contract

The feature is incomplete unless tests prove:

- walker ancestry/family returns partial data plus missing ancestor metadata when a non-start parent is missing
- fully unresolved traversal still fails
- `walk` commands render partial results and warnings
- cancel/delete preflight remains actionable with orphan children when some keys resolve
- direct key lookup remains strict
- waiter absent/deleted behavior remains unchanged
- `v87`, `v88`, and `v89` all honor the shared traversal contract
