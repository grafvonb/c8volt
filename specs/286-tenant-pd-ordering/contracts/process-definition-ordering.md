# Contract: Process-Definition Collection Ordering

## Scope

This contract applies to process-definition collections returned by `c8volt get process-definition` and its established aliases, including filtered and `--all-tenants` discovery, `--latest`, `--stat`, watch snapshots, human output, JSON output, and keys-only output.

It does not apply to direct retrieval by process-definition key or XML retrieval.

## Canonical Order

Before presentation, every collection MUST compare definitions in this exact priority:

| Priority | Field | Direction | Comparison |
|----------|-------|-----------|------------|
| 1 | Tenant ID | Ascending | Exact, case-sensitive text |
| 2 | BPMN process ID | Ascending | Exact, case-sensitive text |
| 3 | Version | Descending | Numeric |
| 4 | Process-definition key | Ascending | Exact opaque text, never numeric |

The displayed default tenant identifier `<default>` is compared as ordinary exact text. It has no special placement.

## Paging

- The backend request MUST use the strongest compatible stable sort before cursor/page traversal.
- The service MUST traverse all available pages for an unbounded collection request, subject to documented version limits.
- Every ordinary matching definition MUST appear exactly once.
- A final canonical sort MUST occur after page accumulation.
- Page sizes 1, 2, and 1000 MUST yield the same ordered key sequence for the same matching data.

## Latest

- A latest group is the exact, case-sensitive pair `(tenant ID, BPMN process ID)`.
- `--latest` MUST return at most one newest matching definition per represented group.
- If the greatest version is tied, exact-text ascending key order selects the deterministic representative.
- Camunda 8.8-8.10 MUST retain native latest filtering and complete the available page traversal.
- Camunda 8.7 MUST reduce visible versions locally by the same group and tie rules.
- The final latest collection MUST use the canonical order.

## Output Modes

| Mode | Contract |
|------|----------|
| Human | Rows appear in canonical order; no new paging or ordering diagnostics are added. |
| JSON | Existing schema/envelope is unchanged; array order matches the canonical human sequence. |
| Keys-only | Prints one existing process-definition key per line and nothing else, in canonical order. |
| Watch | Every snapshot uses canonical order; statistics-only changes do not move rows. |

Renderers MUST preserve the service-provided sequence rather than define independent comparators.

## Statistics

- `--stat` enriches the same ordered definitions in place.
- Active-instance counts, incident counts, and all other volatile statistics MUST NOT affect ordering.
- The ordered key sequence with and without statistics MUST match for the same collection.
- Existing unsupported-statistics behavior on Camunda 8.7 remains unchanged.

## Filtering and Tenant Visibility

- Existing filters and selectors change membership only, not the ordering contract.
- Tenant-filtered and `--all-tenants` collections use the same comparator.
- Authenticated visibility and authorization remain unchanged.
- Tenant IDs and BPMN process IDs that differ only by case remain distinct.

## Version Compatibility

- Equivalent data available on Camunda 8.7, 8.8, 8.9, and 8.10 MUST produce the same canonical sequence.
- Adapters MAY use different generated request structures and supported sort fields.
- Final version-neutral normalization MUST hide those backend ordering differences.
- Camunda 8.7 retains its existing emulated search ceiling of 1000 visible definitions; completeness claims for 8.7 apply within that compatibility window.
- No generated Camunda client is edited for this feature.

## Documentation

Command source metadata, README guidance, and generated CLI documentation MUST state the same field priority and directions. Generated CLI docs MUST be refreshed from command metadata rather than hand-edited.

## Regression Guarantees

- Direct-key retrieval content, validation, authorization, errors, and output remain unchanged.
- XML retrieval content, validation, authorization, errors, and output remain unchanged.
- Existing aliases, flags, exit codes, human columns, JSON fields/envelope, and keys-only shape remain unchanged.
