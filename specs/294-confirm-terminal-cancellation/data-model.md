# Data Model: Cancellation Confirmation

No new public or persisted types are introduced.

## Existing entities and relationships

| Entity | Relevant existing fields | Role and validation |
| --- | --- | --- |
| `domain.ProcessInstance` | `Key`, `State`, `ParentKey` | Identifies a member of the current family; terminal acceptance uses observed state |
| `domain.State` | ACTIVE, COMPLETED, CANCELED, TERMINATED, ABSENT, UNKNOWN | Cleanup accepts exactly the four terminal states; active/unknown do not qualify |
| Family confirmation scope | Existing discovered keys | Every included key must satisfy its wait; discovery timing and membership rules stay unchanged |
| `domain.CancelResponse` | `Ok`, `StatusCode`, `Status` | Terminal-root no-op has `Ok=true`, code 200, existing status wording; no added fields |
| `domain.Reporter` | `Key`, `Ok`, `StatusCode`, `Status` | Bulk cancellation propagates the corrected no-op success value |
| `domain.ProcessInstanceExpectationRequest` | `States`, `Incident` | Explicit expectation matching remains independent; no state/incident relaxation |

## Observation rules

| Observed state | Cancellation confirmation | Explicit state canceled |
| --- | --- | --- |
| COMPLETED | Satisfied | Not satisfied |
| CANCELED | Satisfied | Satisfied |
| TERMINATED | Satisfied | Satisfied under existing equivalence |
| ABSENT | Satisfied | Not satisfied |
| ACTIVE | Keep waiting | Keep waiting |
| UNKNOWN/unrecognized | Not satisfied | Not satisfied |

The feature does not change engine state transitions. An active instance may be observed as naturally completed, cancelled/terminated, or absent; each terminal observation satisfies cancellation cleanup. A completed member is not relabeled cancelled in stored or public data.

## Errors and no-ops

A wrapped `domain.ErrNotFound` from the cancellation precheck maps locally to absent when state checks are enabled. Other precheck errors still fail. Ordinary state getters keep their existing not-found errors. The existing waiter retains its disappearance handling during confirmation. Submission and family-discovery errors are not evidence of absence. Terminal no-op response values are corrected without adding a public evidence structure.

## Completion boundaries

Family confirmation succeeds only when every current scoped key qualifies. Successful cancellation does not prove history or process-definition deletion; existing downstream waits and failures remain authoritative. No migration, serialization change, additional storage, or expanded family model is required.
