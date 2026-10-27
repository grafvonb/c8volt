# Public Library Contract: Definition Lifecycle State

**Feature**: [spec.md](../spec.md)

## Additive result attribute

Add `State string` serialized as `state,omitempty` to the existing public `process.ProcessDefinition`. The internal domain definition carries the same optional attribute. Existing method signatures, interfaces, options, filters, factories, error conversion, and collection envelopes remain unchanged.

| Existing operation/result | Required propagation |
| --- | --- |
| `GetProcessDefinition` | Returned definition preserves state. |
| `SearchProcessDefinitions` | Each returned item preserves state. |
| `SearchProcessDefinitionsLatest` | Selected items preserve state without altering latest selection or canonical ordering. |
| `SearchProcessDefinitionsPages` | Visitor page items and final result items preserve state. |
| `CollectProcessDefinitionWatchSnapshot` | Every returned snapshot item preserves the current observation. |
| Reads using existing statistics options | State remains unchanged while statistics are enriched independently. |

`GetProcessDefinitionXML` remains unchanged. No lifecycle request method, state filter, capability error, or mutation operation is added.

## Value rules

- v810 maps the generated response string directly; facade conversion copies it unchanged.
- Known and unfamiliar nonempty values survive all conversions.
- Empty/unavailable values are empty strings in Go and absent from JSON. No pointer, custom marshaler, closed enum, or instance-state conversion is required.
- v88/v89 adapters leave the field empty and retain their current supported reads.
- Fresh responses create fresh observations; unavailable state must not inherit an earlier value.
- Returned DELETED items remain ordinary items; existing key absence, not lifecycle, determines not-found or watch removal.
- No lifecycle-specific changes to statistics calls, page advancement, tenant resolution, canonical comparisons, latest reduction, retries, or deletion-completion checks.

## Validation obligations

Adapter tests cover lookup and serialized search-body responses for ACTIVE, DRAINING, DELETED, an unfamiliar value, omitted state, and null state. Older adapter reads assert empty state. Public conversion/delegation tests cover direct, collection, page/visitor, final traversal, latest, and watch-snapshot result paths without changing their options or request counts.

A DRAINING result with zero active and two suspended instances retains state and counts. Assert the original per-scenario mock request expectations rather than impose a new fixed request budget on existing pagination/statistics behavior. No real server is required to validate this contract.
