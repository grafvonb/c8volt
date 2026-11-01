# Data Model: Business-ID Creation and Observations

**Feature**: [spec.md](spec.md)
**Decisions and provenance**: [research.md](research.md)

## Representation

Use `BusinessId *string` consistently with existing `BpmnProcessId`/`TenantId` naming. Public and bespoke JSON models use `json:"businessId,omitempty"`. Domain-only types need no new serialization policy. Native v810 types already have the optional field.

| Value | Creation meaning | Observed-result meaning |
| --- | --- | --- |
| nil | Omitted; do not submit field | Not supplied by backend; omit from public JSON |
| Pointer to empty string | Invalid before mutation | Preserve the exact backend fact, including empty string in JSON |
| Pointer to nonempty string | Exact opaque ID; validate before submission | Preserve exact backend value without creation validation |

Copy pointers using `toolx.CopyPtr` when converting mutable public/domain models or freezing plans. Never fill an absent value from an input, parent, sibling, or related process instance. No changes to existing envelope keys, array/null conventions, or listener-section presence.

## Entities and changed fields

| Entity | Owner/types | Change and relationship |
| --- | --- | --- |
| Creation intent | `internal/domain.ProcessInstanceData`, `c8volt/process.ProcessInstanceData` | Add optional BusinessId. Retain existing definition selectors, version, tenant, and variables. Counted creation repeats the exact input value. |
| Native creation result | `internal/domain.ProcessInstanceCreation` | Add optional BusinessId from the creation response; after successful waiting, replace it with the observed instance value, including nil. |
| Process observation | `internal/domain.ProcessInstance`, `c8volt/process.ProcessInstance` | Add optional BusinessId; embedded observations carry it into traversals, enrichment, mutation impact, and operations. |
| Job observation | `internal/domain.Job`, `c8volt/job.Job` | Add optional BusinessId from the job response. Do not consult the owning process instance. |
| User-task observation | `internal/domain.UserTask`, `c8volt/task.UserTask` | Add optional BusinessId; preserve it in variable-enriched task results. |
| Runtime listener observation | `internal/domain.RuntimeListenerJob`; public counterparts in `c8volt/process`, `c8volt/element`, `c8volt/ops` | Add optional BusinessId copied from the source job. Preserve `*[]RuntimeListenerJob` collection semantics. |
| Completion candidate | Domain/public `CompletionCandidate` | Add optional BusinessId from observed job data; maintain compatible shapes and explicit pointer copying in both conversion directions and immutable execution plans. This does not expose assignment during job completion. |
| Slow-analysis process observation | Domain/public `SlowProcessAnalysisProcessInstance` | Add optional BusinessId from its source instance. Related listener observations retain independent IDs. |
| Bespoke preview/output rows | Process-instance dry-run entries and existing custom scalar report lists | Include optional BusinessId only when observed data exists; key/status-only facts remain key/status-only. |

## Creation validation

- Nil is valid and omitted. Supplied empty strings and malformed UTF-8 are invalid input.
- Maximum: 256 UTF-16 code units. Each Unicode code point at or below U+FFFF counts once; a supplementary code point counts twice. Preserve all accepted code points exactly, including whitespace and commas.
- Use a small validator in `internal/domain/business_id.go`; v810 services enforce it before each mutation. Snapshot the optional value before counted scheduling; version-neutral bulk helpers must not preempt the older adapters' unsupported classification with value validation. No new string normalization, key parser, variable mapping, or third-party dependency.
- v88/v89 reject non-nil input as unsupported before mutation. v810 enforces input validity before native request construction. Existing creation without this field retains its version behavior.
- Preserve backend uniqueness decisions. No local uniqueness storage, preflight lookup, suffix generation, or retry with an altered ID.

## Conversion map

| Boundary | Required sites |
| --- | --- |
| v810 process requests/results | `internal/services/processinstance/v810/convert.go`: both `toProcessInstanceCreationInstruction` branches, `fromCreateProcessInstanceResult`, `fromProcessInstanceResult` |
| Process public boundary | `c8volt/process/convert.go`: `toProcessInstanceData`, `fromDomainProcessInstanceCreation`, `fromDomainProcessInstance`, `toDomainProcessInstance` |
| Job/task public boundary | `internal/services/job/v810/convert.go:fromJobSearchResult`, `c8volt/job/client.go:fromDomainJob`; `internal/services/usertask/v810/convert.go:fromUserTaskResult`, `c8volt/task/convert.go:fromDomainUserTask` |
| Runtime listeners | `internal/domain/job.go:RuntimeListenerJobFromJob`; `fromDomainRuntimeListenerJob` in process, element, and operations facade converters |
| Operations duplication | `c8volt/ops/convert.go:fromDomainProcessInstance`, `fromDomainDryRunPIKeyExpansion`, and `fromDomainSlowProcessAnalysisProcessInstance` |
| Completion plans | `internal/services/job/completion/prepare.go`, `execution_plan.go`, `c8volt/job/convert.go` in both directions |
| Slow-analysis root | `internal/services/ops/slow_process_analysis.go:slowProcessAnalysisRootFromProcessInstance` |

## Outcome lifecycle

1. Parse optional intent without losing presence; validate it before mutation.
2. Native submission returns backend creation data or an error. A returned business ID is an observed response fact, not an echo of input.
3. With no-wait, return accepted creation data under the existing acceptance contract.
4. With ordinary waiting, preserve the current visibility waiter and observable states; successful waiting supplies the observed business ID, state, and date. It does not prove process execution completion.
5. Sequential and counted creation retain successful items when another item fails. Preserve their existing ordering and error classification. Remove failed/unattempted zero-value pool slots from returned successful collections at the service boundary, using the worker's successful-return evidence; do not infer success from the input or requested count.
6. A wait error remains a failure under the existing contract. This feature preserves prior successful items without inventing success for the failed attempt or redesigning uncertain-submission tracking.

No persistence schema, migration, new state machine, business-ID index, or independent lifecycle is introduced. A backend resource may gain a new ID later, but previously created related snapshots retain their own values.
