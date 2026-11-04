# Data Model: Process-Instance Resumption

All structures are in-memory workflow evidence. Public types live in `c8volt/process`; domain equivalents live in `internal/domain`. No storage migration, generated-client type exposure, or changes to existing suspension schemas.

## Selection and plan

`ProcessInstanceResumptionPlan` has:

| Field / JSON name | Type | Invariant |
| --- | --- | --- |
| `RequestedKeys` / `requestedKeys` | `typex.Keys` | Trimmed 16-digit unique seeds in first-input order; preserve overlapping explicit seeds |
| `Instances` / `instances` | `[]ProcessInstanceResumptionPlanItem` | Unique discovered members in first-seen traversal order, including seeds |
| `TargetKeys` / `targetKeys` | `typex.Keys` | Exactly the ordered projection of eligible `Instances`; frozen before approval |

Each item contains `Key` / `key` (string), `ObservedState` / `observedState` (existing domain/public State), and `Classification` / `classification` (resumption-specific string enum).

| Classification | Rule |
| --- | --- |
| `eligible` | Observed SUSPENDED during complete discovery |
| `excluded` | Every other state; preserve known state, normalize unrecognized values to UNKNOWN |

No already-resumed class or inferred previous-operation ownership. Active/terminal parents remain traversable. Counts are derived from the plan, not separately persisted: selected = requested keys, scope = instances, eligible = target keys, excluded = scope minus eligible.

Successful plans initialize all arrays, including empty target arrays. Failure returns zero plan plus error and is never a successful empty plan. Validate traversal outcome, mode, start/root identity, missing ancestry, seed presence, key syntax, and matching per-key observations before accepting each subtree. Reject partial/unresolved or malformed results even when they concern overlapping members. Do not claim completeness beyond the existing direct-child lookup limit.

## Per-key outcome

Public `ResumptionResult` corresponds to domain `ProcessInstanceResumptionResult`.

| Field / JSON name | Type | Meaning |
| --- | --- | --- |
| `Key` / `key` | string | Normalized execution target |
| `Status` / `status` | resumption status | Workflow disposition, not engine state |
| `Submission` / `submission` | submission enum | Cumulative submission evidence |
| `Confirmation` / `confirmation` | confirmation enum | Observation evidence independent of submission |
| `StatusCode` / `statusCode,omitempty` | int | Last valid submission HTTP status, never read status |
| `HasObservation` / `hasObservation` | bool | Distinguishes no read evidence from observed UNKNOWN |
| `LastObservedState` / `lastObservedState,omitempty` | State | Last valid matching observation; absent when none |
| `SuspendedDate` / `suspendedDate,omitempty` | string | Latest successful observation's supplied date; no synthesis |
| `Error` / `error,omitempty` | string | Machine evidence; never rendered directly as human diagnostics |

Enums follow existing workflow vocabulary using new resumption-specific types/constants:

- Status: `preview`, `accepted`, `confirmed`, `failed`, `timed_out`, `interrupted`, `not_attempted`.
- Submission: `not_submitted`, `accepted`, `rejected`, `unknown`.
- Confirmation: `not_requested`, `unconfirmed`, `confirmed`.

## Transition and evidence rules

| Event | Status | Submission | Confirmation | Result / error |
| --- | --- | --- | --- | --- |
| Local dry-run | preview | not_submitted | not_requested | No state verification or mutation |
| Queued work never starts | not_attempted | not_submitted | not_requested | Retain key and bounded skipped-work evidence |
| Local validation/admission failure | failed, timed_out, or interrupted as classified | Preserve prior submission evidence; not_submitted only if no attempt was dispatched | not_requested | Preserve original failure |
| Explicit rejection without earlier ambiguity | failed | rejected | not_requested | Preserve submission status and error |
| Ambiguous response/transport outcome | failed/timed_out/interrupted | unknown | not_requested | Later rejection cannot erase ambiguity |
| Valid empty 204, no-wait | accepted | accepted | not_requested | No confirmation reads |
| Accepted, awaiting observation | accepted while in progress | accepted | unconfirmed | Acceptance survives subsequent failure |
| Matching ACTIVE or COMPLETED within bound | confirmed | accepted | confirmed | Success; retain actual observed state |
| Matching CANCELED or TERMINATED | failed | accepted | unconfirmed | Stop with conflict; distinct cancellation reason |
| Bound/attempt limit reached | timed_out | accepted | unconfirmed | Preserve last observation or no-observation state |
| Caller canceled after acceptance | interrupted | accepted | unconfirmed | Preserve last observation and acceptance |
| Read authorization/other failure | failed (or timeout/interruption class) | accepted | unconfirmed | Preserve prior successful observation |

SUSPENDED, ABSENT, and UNKNOWN do not confirm; bounded reads may continue according to existing policy. A not-found read records ABSENT. Wrong-key observations fail as malformed; reads completing after the deadline cannot confirm. No successful mutation response means no confirmation workflow. A valid later acceptance may resolve earlier submission ambiguity; confirmation polling never triggers mutation retries.

Each successful matching observation replaces state/date evidence; an omitted date clears the previous date. A failed read preserves the last successful observation, except the explicit not-found mapping records ABSENT and clears date evidence. Local admission failure during a retry preserves prior rejected/unknown submission evidence and its last valid status; it cannot reset a previously dispatched attempt to not_submitted. Counts overlap deliberately: accepted-but-unconfirmed failures count as both accepted and failed. Do not infer disjoint totals from acceptance and failure counts.

## Collections and rendering payloads

`ResumptionResults` / `ProcessInstanceResumptionResults` wraps `Items` / `items`, always a nonnil array in rendered results. Exactly one slot per normalized execution target, in stable order, survives fail-fast or interruption. Parent and child targets are independent; worker completion order does not reorder results.

- Exact-key payload: `{ "items": [...] }`.
- Tree payload: `{ "items": [...], "plan": { "requestedKeys": [...], "instances": [...], "targetKeys": [...] } }`.
- Tree dry-run: one preview item per eligible key; planning states remain in `plan.instances`, not post-submission observation fields.
- Successful empty tree: `items: []`, `targetKeys: []`, with discovered exclusions retained in the plan. No invented result item.
- Planning/input failure: use the existing error-envelope contract; never manufacture a successful complete plan.

Converters copy slices and map fields/enums mechanically. Public models cannot expose generated or internal domain types. Error conversion returns results alongside `ferrors.FromDomain(err)` without discarding evidence.
