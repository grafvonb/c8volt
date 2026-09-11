# Data Model: Existing Tenant Logging State

No new data types, storage, public fields, or schema migrations are introduced.

| Existing entity | Relevant fields | Relationship and invariants |
| --- | --- | --- |
| `tenantContextHumanLine` | `Text string`, `Warn bool` | Ordered output of tenant line producers; text is forwarded unchanged, Warn selects severity |
| `tenant.Context` | Mode, filter, configured tenant, resolved tenant IDs, warnings | Existing evidence used to build human lines; no selection or evidence changes |
| Command logging context | Attached logger, configured format and level | Consumes eligible lines through the existing durable-line helper |
| Progress channel | Mode, DurableAllowed, StderrAllowed | Additional eligibility for the progress emitter only |
| Command rendered state | Existing tenant-context rendered marker | Prevents duplicate reporting across reporting opportunities |

## Transition Rules

1. Ineligible mode, absent/zero context, or already-rendered context: return with no new records or requests.
2. Eligible unrendered context: mark rendered, obtain existing ordered lines, and forward each line's Text and Warn to the durable logging helper.
3. Attached logger: WARN for Warn=true, INFO otherwise; format and threshold determine visible records. Filtering never causes raw fallback.
4. No attached logger: preserve the helper's plain line fallback to configured stderr.
5. Repeated reporting sees the existing rendered state and retains current suppression. No new reset mechanism is introduced.

## Validation Rules

- Never recompute warning classification or normalize tenant identifiers during rendering.
- Preserve line order and current producer deduplication.
- Do not attach new tenant evidence or issue requests from rendering.
- Result envelopes and nested tenantContext payloads remain unchanged.
