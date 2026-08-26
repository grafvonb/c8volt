# Contract: Latest Process-Definition Search Requests

## Scope

This contract defines c8volt's outbound Camunda request shape for process-definition search paging in the v8.8, v8.9, and v8.10 adapters. It does not change the c8volt CLI, facade API, response schema, or generated clients.

## Page Selection Matrix

| Search state | Required page JSON | Forbidden page fields |
|---|---|---|
| Initial latest page | `{ "limit": <positive-size> }` | `after`, `from`, `before` |
| Latest continuation with cursor `C` | `{ "after": "C", "limit": <positive-size> }` | `from`, `before` |
| Ordinary page at offset `N` | `{ "from": N, "limit": <positive-size> }` | `after`, `before` |

The same matrix applies to Camunda 8.8, 8.9, and 8.10. A patch or prerelease within one configured compatibility line does not alter it.

## Latest Search Invariants

- The filter continues to set `isLatestVersion` to true.
- BPMN process ID, tenant ID, exact filters already present in the request, and the established result limit are preserved.
- Sort remains `processDefinitionId ASC`, followed by `tenantId ASC`.
- An empty or absent response end cursor does not produce a cursor continuation request.
- A non-empty response end cursor is opaque and is returned unchanged as the next request's `after` value.
- Paging stops under the existing completion and result-limit rules.

## Preserved Selection Contracts

- Ordinary process-definition searches retain offset-and-limit pagination, including explicit offset zero.
- Exact-version process-definition selection is unchanged.
- Process-definition-key selection is unchanged.
- BPMN-ID process-instance creation continues to validate all selectors before creating any instances.
- Validation failure continues to prevent partial creation.

## CLI Compatibility

- No command or alias changes.
- No flag, default, prompt, or validation wording changes.
- No human, JSON, or keys-only output changes.
- No exit-code changes.
- No new cursor information appears in ordinary output; diagnostics remain governed by existing verbose behavior.

## Verification Contract

For each affected adapter, automated tests must inspect serialized JSON and prove:

1. Initial latest request: positive `limit`, absent `after`, absent `from`.
2. Latest continuation: exact non-empty `after`, positive `limit`, absent `from`.
3. Ordinary request: expected `from` and `limit`, absent `after`.
4. Latest filters and stable sort are unchanged.

Operational verification must additionally run the BPMN-ID selector workflow against Camunda 8 Run 8.9.17 default H2/RDBMS and create ten instances without the former process-definition search 500 error.
