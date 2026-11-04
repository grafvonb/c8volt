# Data Model: Process-Definition Lifecycle State

**Feature**: [spec.md](spec.md)
**Decisions**: [research.md](research.md)

## Process definition

Extend the existing `internal/domain.ProcessDefinition` and public `process.ProcessDefinition` with one additive field:

| Field | Type | Serialization | Meaning |
| --- | --- | --- | --- |
| `State` | `string` | `state,omitempty` | Lifecycle value supplied for this definition by the latest successful backend read. Empty means unavailable. |

Existing identity, version, tenant, statistics, collection, and envelope fields keep their current types and serialization. The lifecycle field is neither a selector nor a sorting key. No persistence or migration is introduced.

### Mapping and validation

1. v810 generated `ProcessDefinitionResult.State` is converted directly to a string. Preserve every nonempty supplied value, including an unfamiliar value; do not trim, normalize case, or validate against a closed set.
2. The public facade copies the domain value unchanged. The shared conversion carries it into single definitions, collections, pages, traversal results, and watch snapshots.
3. Omitted or null generated values produce the empty string and are omitted from public JSON. Do not distinguish these unavailable forms or introduce null/default placeholders.
4. v88/v89 conversion leaves state empty. Do not reject older-version reads or infer ACTIVE from a successful lookup.
5. Construct fresh values from each successful response. Do not merge an omitted state with an earlier value.
6. Statistics enrichment may update only statistics; it must neither replace nor infer lifecycle state. Canonical ordering, latest selection, key identity, and row membership remain unchanged.

### Known value meanings

| Observation | Meaning for display | What it does not establish |
| --- | --- | --- |
| `ACTIVE` | Definition is available for creating new instances. | That its existing instances are all active or healthy. |
| `DRAINING` | New instances are blocked; existing unfinished work is retaining the definition. | That the active count is positive, or that suspended work has finished. |
| `DELETED` | Definition has been removed from runtime state. | That historical data has disappeared or that the row must be hidden. |
| Other nonempty value | Preserve the supplied observation verbatim. | A known state, a failure, or a terminal-state classification. |
| Empty value | Lifecycle unavailable; omit the value. | ACTIVE, DELETED, or absence of the definition. |
| No returned definition | Existing lookup/listing/watch absence handling. | A returned DELETED observation. |

The application does not implement a definition lifecycle state machine. It may observe ACTIVE → DRAINING → DELETED, ACTIVE → DELETED, supplied → unavailable, or unfamiliar values. No path requires or synthesizes an intermediate state, including internal DELETING.

## Statistics

Existing `ProcessDefinitionStatistics` fields and availability booleans are unchanged. In particular, lifecycle is independent of `Active`, `Suspended`, completed/canceled counts, incident counts, and internal unfinished-work evidence. A fixture with `State="DRAINING"`, `Active=0`, and `Suspended=2` preserves all three values independently. Separate reads are not asserted to form an atomic snapshot.

## Human display projection

Add an owned `state string` to `processDefinitionDisplay`. The stable semantic cells are:

| Index | Value |
| --- | --- |
| 0 | Definition key |
| 1 | Tenant |
| 2 | BPMN process ID |
| 3 | Existing version/version-tag text |
| 4 | Lifecycle state, including an empty cell when unavailable |
| 5 | Existing optional statistics tail |

An all-empty lifecycle column disappears under the existing flat renderer. A mixed listing reserves its width so statistics remain aligned. A single row omits empty cells using existing compact-row behavior. No placeholder or ANSI sequence is emitted for an empty state.

## Watch comparison

Add `state bool` to `processDefinitionChangedFields`; include it in `any()` and derive it from current-versus-previous state strings. Highlight state only when that cell changed on an existing row, using existing bold capability rules. Statistics retain their own changed flags and formatting.

| Refresh outcome | Row/value handling | Baseline handling |
| --- | --- | --- |
| First successful render | Current values, no change marker | Commit first baseline |
| Successful state-only change | One updated row; highlight available changed state | Commit current observation |
| Successful stats-only change | Existing stats highlighting; state unchanged | Commit current observation |
| Successful state and stats change | One updated row; changed values styled independently | Commit current observation |
| Successful supplied-to-omitted change | Clear state, mark row updated, no empty ANSI span | Commit empty state |
| Unchanged successful refresh | Clear prior markers and highlighting | Commit unchanged values |
| Failed fetch or failed enabled summary/result write | Preserve existing failure behavior | Do not replace successful baseline |
| INFO summary suppressed by configured level | Render rows normally | Commit if remaining rendering succeeds |
| Returned DELETED row | Keep row and compare its supplied state | Commit returned row |
| Key absent from successful snapshot | Existing removal behavior | Remove absent key from baseline |

No additional stored state, counters, or watch session mechanics are introduced.
