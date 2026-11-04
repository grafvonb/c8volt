# CLI Contract: Definition Lifecycle Observation

**Feature**: [spec.md](../spec.md)
**Model**: [data-model.md](../data-model.md)

## Existing entry points

No commands, flags, aliases, or selector semantics are added. `get process-definition` and `get pd` retain their current selection, paging, sorting, tenant, and error behavior.

| Existing invocation | Lifecycle behavior |
| --- | --- |
| `get pd --key <key>` | Show supplied lifecycle on the returned human row. |
| `get pd` and existing filtered/latest listings | Show supplied lifecycle per returned definition without additional lookups. |
| Either read with `--json` | Add nonempty `state` to the existing definition object; omit unavailable state. |
| Either read with `--stat` | Show lifecycle separately before the unchanged statistics tail. |
| Either read with supported `--json --stat` | Keep lifecycle and statistics as separate definition attributes. |
| Either read with `--keys-only` | Existing key-only stream; lifecycle never contributes bytes. |
| Supported non-watch quiet modes | Preserve current quiet behavior and explicitly selected machine output. |
| Existing `--xml` read | XML behavior unchanged; no lifecycle text is inserted. |
| `--watch`, optionally with `--stat` | Current lifecycle participates in the existing human refresh and change display. |

Camunda 8.10 supplies lifecycle through its existing read support. Current 8.8/8.9 reads keep working with state omitted. Even for 8.10, absence of state is not an error and does not create an ACTIVE default.

## Human rows

Order: key, optional tenant, BPMN process ID, existing version/tag, optional lifecycle, optional existing statistics. Lifecycle has a stable semantic column even when a row lacks its value. All-empty optional columns are omitted, preserving legacy rows when state is unavailable.

Illustrative rows with an explicit tenant, supplied lifecycle, and available counts:

```text
2251799813685249 tenant orders v3 ACTIVE   [ac:5 cp:20 cx:- inc:- su:-]
2251799813685250 tenant orders v2 DRAINING [ac:- cp:12 cx:- inc:- su:2]
```

These examples retain the existing zero-as-minus and statistics label conventions. Counts and support indicators are not derived from lifecycle. The second row says deletion is waiting on unfinished work, including two suspended instances; it does not imply that resumption alone will finish draining.

Rows without state print neither a state label nor a placeholder. A mixed listing must not shift its statistics into the state column. Unfamiliar nonempty state values are shown as supplied without changing sorting or membership.

## JSON and empty results

Single lookup: retain the shared result envelope and its current definition payload. Listing: retain its current collection/page payload and nested definition objects. The only data-model addition is optional `state` on each definition.

Example definition payload fragment (not a replacement envelope/schema):

```json
{
  "key": "2251799813685250",
  "bpmnProcessId": "orders",
  "processVersion": 2,
  "state": "DRAINING"
}
```

When requested, existing `statistics` is a sibling of `state`; its current values, booleans, zero handling, and omission rules stay unchanged. A nonempty unfamiliar state is serialized unchanged. Empty/unavailable state produces no `state` property, including after an earlier successful observation had supplied it.

A returned DELETED definition remains a result. An absent explicit key retains the existing not-found error/exit behavior. Successful empty listings retain exactly one successful envelope and their established empty payload: do not change omitted fields to empty arrays or vice versa. Empty keys-only output is exactly zero bytes. JSON results are followed by EOF, with no duplicate envelope or diagnostic record on stdout.

## Watch behavior

- First successfully rendered snapshot is unmarked. Later state-only changes mark/update the row even when counts are unchanged.
- Lifecycle and statistics highlight independently. One row changing in both is one updated row.
- Omission clears a previously shown lifecycle value and marks the row updated; there is no empty bold span. Unchanged success clears earlier markers and highlighting.
- Failed refreshes and failed rendering do not replace the last successful baseline. Returned DELETED rows remain; disappearance uses the existing removed-row behavior.
- Direct ACTIVE → DELETED and unfamiliar states are valid observations. Never synthesize DELETING or require all intermediate states.
- Keep repaint → configured INFO summary → result rows ordering and existing row-marker placement. Preserve timing, retries, timeout, interruption, and warning-streak behavior.
- Keep existing rejection of watch combined with JSON, keys-only, XML, quiet, or automation before any refresh request. These rejected combinations are not new lifecycle modes.

## Streams and diagnostic configuration

Result rows, counts such as the existing `found:` result line, and selected-format payloads remain on stdout. Existing watch summaries are INFO records through the configured command logger; slow-refresh warnings are WARN; verbose timing remains gated by verbose and INFO level. Respect plain/structured formats, configured/inherited destinations, and activity-aware writers.

Remove the touched watch paths' existing private plain handler/direct stderr bypasses. Keep messages free of manual timestamp/severity prefixes. When INFO is filtered, summaries/timing disappear but rows and successful baseline commits remain. Preserve summary handler errors so an enabled summary write failure does not commit an unseen candidate. No separate/default logger fallback is allowed in that helper.

Failures retain their original error class/exit behavior, safe bounded human diagnostics, DEBUG-only raw HTTP detail with redaction, and established machine evidence. Lifecycle adds no error renderer, prompt, mutation, or extra discovery.

## Documentation and integration acceptance

Explain ACTIVE, DRAINING, DELETED, optional availability, 8.10 support, and independent instance statistics. Explain that backend deletion does not automatically cancel/resume instances, while existing `delete pd --force` explicitly cancels unfinished instances first. Neither this feature nor a returned DELETED state changes deletion-completion checks or guarantees historic-data removal.

Update the existing draining integration observation to use `get pd --key <owned-key> --json`, decode exactly one envelope, require success and the exact key/state, and bound observation using existing polling discipline. Keep independently verified cleanup/absence. Do not add a new explicit no-wait scenario, new live fixture matrix, or an implementation-time live run.
