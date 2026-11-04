# Data Model: Suspended Definition Cleanup

This is an additive extension of existing models. No storage, generated client, lifecycle enum, or new public mutation API is introduced. See [research.md](research.md) for decisions and [contracts/definition-cleanup.md](contracts/definition-cleanup.md) for observable behavior.

## Definition statistics

Owners: `internal/domain/processdefinition.go`, `c8volt/process/model.go`; converters in `c8volt/process/convert.go` and `c8volt/ops/convert.go`.

| Field | Type / representation | Meaning |
| --- | --- | --- |
| Existing `Active`, `Completed`, `Canceled`, `Incidents` | Existing types/tags | Unchanged buckets; Active never includes suspended |
| `Suspended` | `int64`, `suspended,omitempty` | Observed suspended count when supported |
| `SuspendedCountSupported` | `bool`, `suspendedCountSupported,omitempty` | Distinguishes supported zero from unavailable statistics |
| Internal `Unfinished` | `*int64`, `json:"-"`; absent from facade statistics | Authoritative union observation for cleanup, not a sum of independently sampled buckets |

For 8.10, publish a statistics result only after all required observations succeed, including the new suspended and combined queries. A supported result with nil internal Unfinished is invalid for cleanup. Older versions leave suspension support false, issue no new suspended/union query, and retain their existing active-only cleanup count. No-state-check produces the existing skipped plan, not synthetic authoritative zero statistics.

Independent observations can disagree during concurrent state changes; do not force reconciliation by rewriting active or suspended values. Stable-fixture expectations remain exact. The combined count controls cleanup eligibility and verification.

## Internal instance search selector

Owner: `internal/domain/processinstance.go` with versioned request mapping/validation.

- Preserve singular `State` and all public facade filters.
- Add internal `States` of the existing domain states collection type for combined selection.
- Singular and nonempty plural selection are mutually exclusive; reject ambiguity before I/O.
- Normalize and deduplicate the plural values and validate every requested state. Do not silently drop unsupported values or broaden a query.
- v810 maps the union to the existing generated `$in` representation. Older adapters reject unsupported plural requests; definition cleanup itself keeps using singular ACTIVE on those versions.
- Definition key, effective tenant, and existing options remain part of the same request. Do not implement a local state filter that discards continuation information.

No new CLI flag or facade filter field is required.

## Definition cleanup plan item

Owners: `internal/domain/resource_workflow.go`, `c8volt/resource/model.go`, associated resource and ops converters.

Preserve existing identity, active count/keys, cancellation plan, and warnings. Add:

| Field | Type / JSON name on public plan | Meaning |
| --- | --- | --- |
| `SuspendedProcessInstanceCount` | `int64` / `suspendedProcessInstanceCount,omitempty` | Separate observed suspended count |
| `SuspendedCountSupported` | `bool` / `suspendedCountSupported,omitempty` | Availability of the suspended count |
| `UnfinishedProcessInstanceCount` | `int64` / `unfinishedProcessInstanceCount,omitempty` | Authoritative unfinished observation used by enabled checks |
| `UnfinishedProcessInstanceKeys` | Existing key-slice convention / `unfinishedProcessInstanceKeys,omitempty` | Unique active or suspended keys discovered for force planning |

Keep `ActiveProcessInstances()` active-only. Add a separate unfinished accessor with the existing count/key fallback convention; after discovery it must not understate known unfinished keys. Separate active keys are populated only from items observed ACTIVE, never by copying the union. Existing `StateCheckSkipped` at the plan boundary remains the signal that counters were not observed.

The internal nil-unfinished validation happens before producing a successful checked plan. No extra public availability flag for the combined count is necessary. Older-version plans populate unfinished from their existing active observation. Existing no-state-check plans retain zero-value fields and the skip signal.

The existing `CancellationPlan` receives union-derived keys and continues to represent root/family expansion. Root and affected deduplication remain where they are today. Do not infer that a missing union key means confirmed absence after mutation.

## Totals and purge reports

Owners: public resource plan totals; `internal/domain/ops_all_process_definitions_purge.go`, `c8volt/ops/model.go`, and conversions.

- Preserve all existing active totals and field names.
- Add corresponding suspended/unfinished totals to resource plan totals and purge delete-plan projections where previews/reports need them. Public names follow existing lower-camel-case conventions: `suspendedProcessInstances` / `unfinishedProcessInstances` on resource totals, `suspendedProcessInstanceCount` / `unfinishedProcessInstanceCount` on purge plans.
- Carry suspended support in aggregates; a skipped check is not an observed zero. A normal workflow uses one configured backend; any aggregate containing unavailable suspended observations must not claim complete suspended statistics.
- `RequiresForce` uses unfinished workload, not active-only workload. Affected-count fallback and report key collection use unfinished counts/keys where no expanded family is available.
- Preserve the existing report schema version and result/envelope shapes; these are additive fields. No new outcome enum, submission semantics, or DRAINING state is introduced.

## State and validity rules

ACTIVE and SUSPENDED are both unfinished. Moving between them does not complete cleanup. Existing completed/terminated/absent rules remain in the instance layer; definition lifecycle is separate and unchanged.

Combined discovery/counting is an observation, not a transaction. Exact totals may be used; lower-bound totals cannot be labeled exact. Incomplete enumeration or required read errors must stop dependent work. Current terminal prerequisites, wait deadlines, no-wait distinction, and no-state-check bypass remain authoritative.
