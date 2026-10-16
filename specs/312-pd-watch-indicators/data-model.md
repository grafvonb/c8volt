# Data Model: Watch Presentation State

All entities are private command presentation types. No persistent storage, public facade fields, domain changes, or machine-output schema changes are introduced. Names below describe responsibilities rather than mandatory Go identifiers.

## Displayed Definition

- `key`: stable definition identity used for matching.
- Ordered plain display fields: key, tenant, BPMN process ID, combined version/tag.
- Optional statistics presence and display tokens: active, completed, canceled, and incident count when supported.
- Display tokens use the existing formatting rules, including `-` for zero.

Projection owns its values and does not retain mutable statistics pointers. Optional-field absence is distinct from a present value. Compare only tokens that affect visible output; metadata that is never displayed is excluded. Version and tag are one existing display field. Statistics retain individual token identity even though assembled into one tail column.

## Watch Baseline

- `initialized`: distinguishes no successful refresh from a successful empty refresh.
- `byKey`: the prior successfully displayed definitions, indexed by key.

Lifetime is one watch invocation. The snapshot collector already supplies unique keys; this feature does not create a separate duplicate-resolution policy. Map iteration never determines output order.

## Candidate Watch View

- Ordered rows corresponding exactly to the current snapshot order.
- Per-row marker: blank, `*`, or `+`.
- Per-row changed-field mask identifying visible values eligible for bold.
- Counts: updated, new, removed, and current row count for the existing footer.
- Candidate next baseline containing only current definitions.

Invariants:

- An uninitialized baseline yields blank markers and all-zero counts.
- A current key absent from an initialized baseline is new and cannot also be updated.
- A key present in both is updated iff at least one displayed field differs.
- Multiple changed fields count as one updated definition.
- Prior keys absent from current rows count as removed and produce no retained row.
- Bold masks apply only to updated rows and currently visible changed values. A removed optional field still triggers `*` but has no replacement token to bold.
- Current row count is independent of change counts and preserves the established `found: N` meaning.

## Transitions

| Prior state | Event | Visible result | Next baseline |
| --- | --- | --- | --- |
| Uninitialized | Collection fails | Existing failure handling only | Uninitialized |
| Uninitialized | Complete result renders successfully | Blank markers, zero changes | Initialized with current keys, even if empty |
| Initialized | Collection fails or returns partial data with error | Existing failure handling; no new comparison | Unchanged |
| Initialized | Complete result and successful rendering | Delta against prior baseline | Current displayed values |
| Either | Repaint or view write fails | Existing error path; no acknowledged refresh | Unchanged |
| Initialized | Identical successful result | Blank markers, zero changes, no bold | Equivalent current values |
| Initialized, nonempty | Successful empty result | Removed count equals previous key count; no rows | Initialized empty |
| Any | Session ends | Existing stop behavior | State discarded |

Construct the candidate without mutating the prior baseline. Write the view through an error-returning watch-specific path, then commit the candidate baseline. No additional discovery requests are made for rendering or counting changes.
