# Data Model: Update User-Task Variables

Domain types belong in `internal/domain/usertask_update.go`; public equivalents belong in `c8volt/task/model.go` and convert mechanically. These are transient operation values, not stored records. Copy mutable maps/slices across facade boundaries using existing helpers.

## VariableUpdatePlan

- `RequestedKeys`: unique task keys in input order.
- `UserTasks`: one task plan per requested key, including tasks with no changes.
- `Targets`: unique scope payloads in first-encounter order; only additions and changes.
- Counts: requested tasks, tasks with changes, additions, changes, unchanged requested values, untouched values. Variable category counts describe task-variable observations, following the PI preview; target write counts are separate internal execution facts and MUST NOT replace these counts.
- `MutationSubmitted`: false for every plan/preview/no-op.
- Tenant evidence: observed actual tenants and unknown count for existing tenant-context rendering.

A plan is executable only after all task and variable reads succeeded. An empty payload yields a valid zero-change plan for valid selected tasks. Empty CLI key input remains invalid. Freeze task identity and scope/name targets; execute never recomputes them from current reads.

## UserTaskVariablePlan

- `UserTaskKey`, `ElementInstanceKey`, actual `TenantId`.
- `Additions`, `Changes`, `UnchangedRequested`, `Untouched`: stable name-sorted collections.
- Each entry has `Name`, `ScopeKey`, inherited/local classification, and either `Value` or `Before`/`After`.
- `APITruncated` identifies incomplete before-values; preserve the received value and label it rather than suggesting completeness.
- Target references associate planned task changes with their unique scope payloads.

Existing requested names keep the scope returned by effective-variable discovery. Absent names use the selected task's element scope. Complete normalized values establish equality; truncated values never do. Null is a value, not deletion. Invalid or absent required scope identity is a planning error.

## ScopeVariableUpdateTarget

- `ScopeKey`: target identity; validate with existing key rules.
- `Variables`: nonempty map of planned additions/changes for this scope.
- Task associations: task keys and names that depend on this target.
- Tenant evidence: consistent observed context; no tenant-based rediscovery.

Logical identity is `(ScopeKey, Name)`. Merge identical requests; reject conflicting requested values or contradictory identity/tenant evidence before writes. Group all names for one scope into one request with local semantics. Retry attempts follow existing mutation policy and are not additional logical targets. Same names in different scopes remain independent.

## ScopeVariableUpdateOutcome

- `ScopeKey`, names, acceptance fact, optional HTTP status/message, optional error.
- Submission state: submitted, mutation_failed, or skipped (never started).
- No task variable confirmation takes place inside the scope writer.

## UserTaskVariableUpdateResult

Retain PI result field vocabulary where meaningful: `key` (user-task key), `status`, `mutationAccepted`, `confirmationStatus`, `message`, `error`, and `variables`. Add `scopes` only to represent the necessary multiple-scope outcome facts.

- `status`: `confirmed`, `submitted`, `mutation_failed`, `confirmation_failed`, `unchanged`, or `skipped`.
- `mutationAccepted`: true if any required scope write for this task was accepted. This is a historical fact, not a success predicate; inspect status and scopes for completeness.
- `confirmationStatus`: `confirmed`, `failed`, or `skipped`; unchanged tasks use `skipped` and `mutationAccepted=false`.
- `scopes`: outcomes for this task's planned changed scopes, retaining partial acceptance and failures. Do not assign one arbitrary HTTP status to a task with several scope responses.
- `variables`: requested payload, copied at boundaries.

Results retain input task order. A shared scope outcome appears consistently in every dependent task result. A task with no changes has `unchanged`; failed work never becomes an unchanged result.

## State and aggregation rules

1. Planning failure: return error; no executable plan and no writes.
2. Valid plan with no targets: successful no-op; preview facts only, no synthetic execution reports.
3. Dry-run: same plan; no writes regardless of no-wait.
4. Submitted scope writes: retain all outcomes including skipped targets after fail-fast/cancellation.
5. Any task scope failure: task is `mutation_failed`, retaining accepted scopes; any skipped required scope with no direct failure makes it `skipped`, also retaining accepted work. Neither is successful.
6. All required scopes accepted: `submitted` with no-wait; otherwise confirm every requested value/scope through complete task effective-variable lookup and return `confirmed` or `confirmation_failed`.
7. No rollback and no rescoping after confirmation. New local shadowing, disappearance, or authorization loss can prevent confirmation even after a successful write.

Return aggregate results alongside errors. Successful task totals include confirmed/submitted/unchanged as appropriate, while changed-task update summaries exclude unchanged tasks; failed/skipped work never inflates successful totals. Use explicit counts in views, not the PI `OK()` predicate blindly.
