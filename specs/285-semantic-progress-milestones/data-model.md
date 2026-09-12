# Data Model: Semantic Progress Milestones

The feature adds in-memory progress facts and command state only. It does not add persistent storage or alter result/report schemas.

## Progress Scope

Represents one finite operator-visible phase with one frozen unit of work.

| Field | Type | Rules |
| --- | --- | --- |
| Phase | string | Stable machine-facing phase identifier; human wording remains in command configuration. |
| Core resource | string | Unit counted by `completed/total`, such as process-definition, process-instance-root, repair, or smoke-test-stage. |
| Total | integer | Exact frozen total when available; never changes during the scope. |
| Affected resource | optional string | Unit for cumulative affected values, shown only when coverage is trustworthy. |
| Affected coverage complete | boolean | Must be true before affected values are eligible for rendering. |
| Started at | time | Set when real work begins, after any destructive confirmation. |
| Output policy | Output Policy | Fixed for the scope. |

### Invariants

- Discovery and mutation use separate scopes.
- A destructive mutation scope starts only after confirmation.
- `total` is never inferred from discovery page size.
- Nested HTTP, wait, and batch activity cannot replace the workflow activity.

## Completion Fact

One wording-free service fact emitted for an item that actually finished its configured boundary.

| Field | Type | Rules |
| --- | --- | --- |
| Phase | string | Selects the owning progress scope. |
| Identity | string | Stable operator-useful key or stage identity; may be omitted only when the producer cannot prove one. |
| Disposition | enum | `submitted`, `confirmed`, or `failed`. |
| Failure detail | optional string | Existing error detail for immediate diagnostics; never contains a command-rendered sentence. |
| Affected count | optional pointer to integer | Pointer distinguishes trustworthy zero from unavailable. Must be a non-negative per-completion delta. |

### Invariants

- Exactly one fact is emitted per executed item/stage.
- An unscheduled fail-fast item emits no fact.
- `submitted` means request acceptance without operational confirmation.
- `confirmed` means the existing command/service confirmation contract completed.
- `failed` is emitted on the same callback that records the failed item.
- Facts may arrive concurrently and out of input order.

## Progress Aggregate

Command-owned state derived from completion facts.

| Field | Type | Initial value | Update |
| --- | --- | --- | --- |
| Completed | integer | 0 | Increment once for every accepted completion fact. |
| Failed | integer | 0 | Increment when disposition is `failed`. |
| Affected | integer | 0 | Add the fact delta only while affected coverage remains valid. |
| Affected valid | boolean | Scope capability | Becomes false permanently if any fact lacks a trustworthy delta. |
| Last identity | optional string | empty | Replace with the most recently ingested identity for verbose rendering. |

### Invariants

- `0 <= failed <= completed`.
- When a total is known, `completed <= total`.
- Completed, failed, and affected never decrease.
- Affected output is omitted for the entire scope unless coverage is complete.
- Aggregation order does not change final result ordering.

## Durable Milestone State

Command-owned pacing and finish state.

| Field | Meaning |
| --- | --- |
| Last informational time | Scope start or time of the previous paced aggregate line. |
| Durable activated | True after the first paced information line or failure warning. |
| Dirty | Aggregate changed since the last durable line. |
| Finished | Makes finish/close idempotent. |

### State transitions

```text
created
  -> active on workflow activity start
  -> active + durable on first >=10s completion or any failure
  -> active + clean after each durable line
  -> active + dirty after later completion
  -> finished after optional single final flush and activity stop
```

A clean scope that finishes before durable activation produces no durable line.

## Output Policy

Derived from existing command flags and render mode.

| Mode | Transient aggregate | Paced aggregate | Per-item durable | Failure warning | Final flush |
| --- | --- | --- | --- | --- | --- |
| Default human | yes | yes | no | yes | if activated and dirty |
| Verbose/debug | yes | no | yes | yes | not needed when every fact was durable |
| Quiet | no | no | no | yes | only as a warning when failed context remains; never for success-only progress |
| Automation | no | no | no | no | no |
| JSON | no | no | no | no | no |
| Keys-only | no | no | no | no | no |

## Relationships

- A Progress Scope owns exactly one Progress Aggregate, Durable Milestone State, Output Policy, and workflow activity lifetime.
- A Progress Scope consumes zero or more Completion Facts.
- A Completion Fact belongs to one phase/scope and changes the aggregate exactly once.
- Durable lines are projections of the aggregate; they are not authoritative result data.
