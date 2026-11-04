# Contract: Native Business-ID Search

**Requirements**: FR-001–FR-008, FR-018–FR-024 | **Model**: [data-model.md](../data-model.md)

## Input and version behavior

`--bid <value>` is one optional scalar exact-match criterion. Omitted input preserves existing behavior. Explicit empty input is invalid; valid UTF-8 strings are preserved without trimming, splitting, case changes, key parsing or process-variable interpretation. Literal commas and wildcard characters have no special meaning. Do not impose creation-specific length limits on search.

Native support is available in the existing v810 adapter and future adapters only when explicitly implemented. v88/v89 return the existing unsupported-capability classification for supplied BID and make no search/mutation request with a discarded criterion. Keep the default configured version unchanged. Public library input obeys the same semantic contract as CLI input.

## Covered query surfaces

| Surface | BID behavior | Combination rules |
| --- | --- | --- |
| `get pi` | Match native process-instance business ID | AND with existing supported PI predicates; preserve default state and tenant scope |
| `get user-task` | Match native stored task business ID | AND with task predicates, including process-instance key and variables; no join to current PI identity |
| Public process filter/search | Same exact criterion | Preserve page/total/enrichment semantics and selector validation |
| Public task search | Same exact task criterion | Preserve paging, totals and variable enrichment |

Explicit resource-key input, including stdin where supported, remains exclusive with search mode. A user-task `--pi-key` is a task-search predicate and can combine with BID; it is not explicit task-key mode. Do not widen other existing selector combinations.

## Native serialization

Set `filter.businessId` using the exact string alternative of `StringFilterProperty`; JSON wire shape is a scalar string. Keep all other predicates and tenant selection in the same native filter. The PI custom filter must remain nonempty when BID is its only criterion. No generated-client changes are required.

For example, the relevant fragment for a literal comma-containing ID is:

```json
{"filter":{"businessId":"ORDER-123,ORDER-456"}}
```

This fragment shows one predicate, not a complete request or an instruction to call a live endpoint.

## Direct-incident filtering

A supplied BID makes the direct-incident-index shortcut ineligible both in the CLI and in service normalization, even when a library caller explicitly requests that shortcut. Use native PI selection and existing local direct-incident checks. Do not first cap incident-selected candidates and then postfilter BID. Totals use native exact metadata only when all predicates are compatible; otherwise count through service-owned filtered discovery.

## Paging and results

Retain the criterion on every page and enrichment path. Page size is not a total selection cap. Sparse intermediate pages with continuation must not terminate traversal. Distinct selection and raw-page progress are separate; duplicate observations cannot consume a distinct-key bound. Unknown or cyclic continuation cannot prove exhausted results.

Keep existing human result rows, JSON envelope, total-count and keys-only conventions. Empty results follow the existing mode-specific payload; keys-only is zero bytes. An older task with no stored BID does not acquire its process instance's current value. No extra PI lookup is added merely to filter or display a task BID.

## Required isolated evidence

Inspect actual serialized custom PI and native task requests for omitted, BID-only, combined, literal opaque and unsupported-version cases. Exercise both direct-incident entry boundaries, sparse pages, duplicate keys, limits, totals and enrichment through actual command/public service paths. Native task fixtures must deliberately differ from their current PI business ID. Include explicit-key/stdin conflicts and pointer-copy/JSON-description round trips.
