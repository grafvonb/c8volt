# Contract: Cancellation Across Multiple Process Families

**Requirements**: FR-009–FR-016 and FR-020–FR-023, plus the user's explicit cross-family clarification. This contract elaborates the existing specification without editing it.

## Required behavior

Both `cancel pi --bid <value> --dry-run` and execution discover all matching process instances within configured tenant scope and documented seed limits. Multiple matches may belong to different families or overlap within one family. The command must not choose the first match or process each discovery page as a separate mutation scope.

After discovery completes, apply existing cancellation ancestry-to-root and full-family traversal to every distinct seed. Deduplicate resolved roots and the affected member union globally. Each distinct family root is expanded once for planning, with exhaustive direct-child paging at every depth; no 1000-child or hidden preview cap is allowed. Different or absent BIDs on expanded relatives do not exclude them from established family scope.

Freeze the completed seed selection, resolved roots and observed expanded impact before confirmation or submission. Discovery or required expansion that fails, is interrupted, or cannot prove completeness submits zero mutations. Documented seed limits permit a bounded selected set; they do not constrain the family impact of those seeds.

Existing planned command cancellation submits roots even when the selected seed is a child. Preserve this behavior and existing force/precondition handling; do not derive new BID semantics from the single-key adapter's child-input checks.

## Full dry-run representation

The preview must retain the entire combined scope:

- `requestedKeys`: every distinct directly matched seed within the documented bound.
- `resolvedRoots`: every distinct cancellation root.
- `affectedFamilyKeys`: the full distinct observed family-member union.
- Exact selected/root/affected counts, selection BID and bound, traversal completeness, missing-ancestry evidence when incomplete, and `mutationSubmitted: false`.

Default human output reports complete counts and distinguishes selected seeds from additional affected relatives through the configured INFO logger. JSON contains the full arrays; verbose human detail lists all planned keys. Keys-only prints every affected key once. No format may silently use a truncated underlying plan or double-count families across pages. Preserve established empty representations and output-mode precedence.

## Concrete acceptance fixture

All nodes below are in tenant A. Search returns matching seeds in the order `A1`, `A2`, `B1`, with duplicate observations of `A1` and an intermediate sparse page before the last match.

```text
RA [bid: another-order]
├── A1 [bid: ORDER-123]     selected
│   └── A1a [no BID]
└── A2 [bid: ORDER-123]     selected

RB [no BID]
├── B1 [bid: ORDER-123]     selected
└── B2 [bid: different]

RC [tenant B]
└── C1 [bid: ORDER-123]     not selected in tenant A
```

For completed unbounded discovery, expected scope is:

| Set | Expected distinct members |
| --- | --- |
| Direct seeds | `A1`, `A2`, `B1` |
| Resolved roots | `RA`, `RB` |
| Affected family union | `RA`, `A1`, `A1a`, `A2`, `RB`, `B1`, `B2` |
| Additional affected members beyond direct seeds | `RA`, `A1a`, `RB`, `B2` |

Assert selected count 3, roots 2, affected count 7, and additional affected count 4. Preserve deterministic existing traversal order within the union; assertions must not impose a new sort order. `RA` is expanded once despite two selected children and repeated observations. `RC` and `C1` are excluded from selection and mutation.

Dry-run performs zero mutations and zero destructive confirmation prompts. Execution performs no mutation before all seed and expansion discovery completes; it submits each distinct root through existing cancellation behavior without duplicate submissions caused by overlapping selection. Existing transport retry policy remains unchanged. Confirmation, if eligible, refers to the completed scope. No fresh BID search runs after approval.

## Required variants

1. Put matches in more pages than the requested batch size, with empty intermediate selected pages and continuation; still reach `B1`.
2. Apply a seed limit of 2 with `A1` duplicated before `A2`: select two distinct seeds in family A and expand all four family members. Do not include family B or clip family A to two affected members. Mark discovery user-limited.
3. Add at least 1001 direct children under a selected root; require every child in the planned union, including later-page children and deeper descendants.
4. Fail or cancel a later seed page or later descendant page; require zero mutation requests, incomplete evidence and the original failure class. Test repeated cursors and unknown continuation explicitly.
5. Return no matches with authoritative exhaustion; require successful empty modes, zero prompts and zero mutations, including no-wait and dry-run.
6. Make one root succeed and another fail during execution; retain partial evidence, one compact aggregate human diagnostic or one JSON envelope, and the original nonzero failure classification.
7. Introduce another matching root after confirmation; no second BID discovery and no submission to that root. Change observed resource BIDs after planning; selection provenance remains the original criterion and observations are not rewritten to match it.

## Timing and backend cascade boundary

A dry-run command and a later execution command are separate discoveries and may produce different plans. Within one run, submitted roots and the observed plan are frozen before approval.

This is not an atomic engine snapshot. Native cancellation of a selected root may also affect descendants created concurrently beneath that root, and existing read-only post-submission verification may observe them. Keep later observations separate from the original preview. Do not promise exclusion of those descendants, add a continuous BID watcher, or add newly matching independent roots after confirmation. Normal wait/no-wait semantics remain unchanged.
