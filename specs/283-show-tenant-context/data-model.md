# Data Model: Effective Tenant Context Before Mutations

## TenantContext

One immutable description of tenant semantics and resolved tenant evidence for a command execution or frozen plan.

| Field | Type | Required | Rules |
|---|---|---:|---|
| `mode` | enum | yes | `configuration`, `discovery`, `creation`, or `explicit_keys` |
| `filter` | enum | yes | `named`, `none`, `not_applied`, or `not_applicable`; must match `mode` |
| `configuredTenantId` | string | no | Raw named effective tenant; omitted when configuration is empty |
| `targetTenantId` | string | no | Creation destination only; named value or `<default>` |
| `resolvedTenantIds` | array of string | yes | Unique known actual tenants in ascending lexical order; empty array when none are known |
| `unknownTargetCount` | integer | yes | Count of unique affected targets whose tenant metadata is unavailable; never negative |
| `crossTenant` | boolean | yes | True exactly when `resolvedTenantIds` contains more than one value |
| `warnings` | array of `TenantContextWarning` | no | Stable-code warnings derived deterministically from the other fields |

### Mode and filter invariants

| Mode | Valid filter | Additional invariant |
|---|---|---|
| `configuration` | `named` or `none` | Describes configured discovery scope only; never synthesizes `<default>` |
| `discovery` | `named` or `none` | `named` requires `configuredTenantId`; `none` emits `unfiltered_selection` |
| `creation` | `not_applicable` | Requires `targetTenantId`; empty configuration maps to `<default>` only here |
| `explicit_keys` | `not_applied` | Configured tenant may be retained as context but never restricts or rejects a key |

`resolvedTenantIds` records actual known resource metadata. `<default>` is valid only when supplied by an existing target/plan model or used as the creation target; an empty resource tenant is unknown, not default.

## TenantContextWarning

| Field | Type | Rules |
|---|---|---|
| `code` | enum | `unfiltered_selection`, `multiple_tenants`, or `unknown_target_tenants` |
| `message` | string | Stable human-readable explanation generated from the context |

Warnings are ordered `unfiltered_selection`, `multiple_tenants`, then `unknown_target_tenants`. Multiple-tenant and unknown warnings can coexist. No warning changes command eligibility, exit status, or confirmation policy.

## TenantEvidenceAccumulator

Internal service helper used while constructing a plan.

| State | Meaning |
|---|---|
| `seenTargets` | Unique resource keys already counted; prevents duplicate plan/traversal entries from inflating unknown counts |
| `knownTenants` | Set of non-empty tenant IDs from affected resources |
| `unknownTargets` | Number of unique affected keys with empty/unavailable tenant metadata |

### Operations

1. `Add(key, tenantID)` ignores a duplicate key.
2. A non-empty `tenantID` is inserted into `knownTenants`.
3. An empty `tenantID` increments `unknownTargets`.
4. `Snapshot()` sorts known tenant IDs, sets `crossTenant = len(knownTenants) > 1`, and returns immutable resolved evidence.
5. Page/workflow aggregation merges by target key, not raw counts, so the same dependency cannot be counted twice.

Legacy PI dry-run paths that expose only affected keys produce unknown evidence for those keys rather than making a lookup or inventing a tenant.

## Relationships

```text
Effective configuration ──interpreted by command──┐
                                                  ├── TenantContext ──> human preview/confirmation
Frozen mutation or ops plan ──aggregated by service┘                 ├──> ResultEnvelope.tenantContext
                                                                     └──> AuditReport.tenantContext
```

- Process-instance expansion carries a snapshot built from traversal-chain resources before metadata is discarded.
- Paged PI mutation plans merge step snapshots into one confirmation context.
- Process-definition deletion aggregates plan-item tenants and nested PI cancellation evidence.
- Job update uses the current job already read for planning.
- PI variable update uses tenant-bearing variables already read; absence remains unknown.
- Retention, purge, repair, and smoke-test workflows aggregate their frozen plan/result objects.
- Deploy/run creation uses the effective target before execution and may enrich resolved evidence from the returned object afterward.

## Public and Serialized Forms

`internal/domain.TenantContext` is the canonical service value. `c8volt/tenant.Context` is the public form and carries both `json` and `yaml` tags. Facade conversion is field-for-field and returns a copy of slices. Command-specific view types reference the public form; they do not create parallel warning or enum types.

The value appears:

- as optional `tenantContext` beside `payload` in full shared result envelopes;
- as optional `tenantContext` in applicable raw structured views;
- as optional `tenantContext` at the root of affected ops audit reports;
- as `tenantContext` in the sanitized `config show` YAML document.

## Lifecycle

1. **Base**: after configuration resolution, command creates the appropriate mode/filter/target context.
2. **Resolving**: service planning adds actual known/unknown evidence from resources already read.
3. **Frozen**: before preview or confirmation, command combines base semantics with the plan snapshot and derives warnings.
4. **Rendered**: the same frozen value is used by human preflight, confirmation, structured result, and audit enrichment.
5. **Completed**: returned resource evidence may fill previously unavailable target tenant for final reporting, but it must not rewrite what was shown as known before confirmation.

For paged workflows, confirmation context is computed from all pages frozen for that confirmation boundary. If a workflow confirms per page, each page context is accurate for that page and the final report merges all executed pages.

## Validation Rules

- Reject invalid mode/filter combinations in constructors/tests; serialized contexts originate only from constructors.
- Never include an empty string in `resolvedTenantIds`.
- Sort and deduplicate tenant IDs before warning construction.
- Count unknowns by unique affected key.
- Set `crossTenant` from known tenant IDs only.
- Do not reject an explicit target when `configuredTenantId` differs from a resolved tenant.
- Do not perform a network lookup solely to reduce `unknownTargetCount`.
