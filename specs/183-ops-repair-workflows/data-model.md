# Data Model: Ops Repair Workflows

## Repair Request

Captures one operator invocation for either repair target.

- **Command name**: `ops repair incident` or `ops repair process-instance`
- **Target type**: incident or process-instance
- **Discovery mode**: keyed, stdin keys, or search filters
- **Input keys**: explicit incident keys or process-instance keys
- **Selection filters**: incident or process-instance filter values
- **Bulk controls**: workers, fail-fast, no-worker-limit, batch-size, limit
- **Execution controls**: dry-run, auto-confirm, automation, output mode
- **Repair controls**: variables, variable file, retries, job timeout
- **Report controls**: report file and report format
- **Started at**: UTC timestamp for report duration

## Frozen Repair Set

Immutable discovery output produced before mutation.

- **Incident keys**: deduped incidents selected for repair
- **Process-instance keys**: selected or associated process instances
- **Expanded family keys**: present only when a process-instance workflow uses family scope
- **Job keys**: related job keys where present
- **Variable scopes**: deduped process-instance keys that will receive variables
- **Original incident context**: state, error type, error message, flow node id, element instance key, root key, and creation time when known
- **Discovery filters**: the filter values used to produce the set

## Repair Plan Item

Per-incident plan entry.

- **Incident key**: required
- **Process-instance key**: required when known
- **Root process-instance key**: optional
- **Job key**: optional
- **Variable scope key**: optional
- **Requested variable names**: deduped variable names
- **Requested retries**: omitted, default `1`, or explicit `0`
- **Requested timeout**: optional duration
- **Variable update status**: planned, skipped, submitted, confirmed, blocked, failed
- **Retry update status**: planned, skipped, not_applicable, submitted, confirmed, failed
- **Timeout update status**: planned, skipped, not_applicable, submitted, confirmed, failed
- **Resolution status**: planned, skipped, submitted, confirmed, confirmation_failed, blocked, failed
- **Notices**: semantic context such as no related job
- **Errors**: per-incident failures

## Variable Scope Update

Represents one deduped process-instance variable mutation and confirmation.

- **Scope key**: process-instance key
- **Variable names**: requested names only
- **Payload**: normalized JSON object
- **Dependent incident keys**: incidents that must wait for this update
- **Status**: planned, submitted, confirmed, blocked, or failed
- **Errors**: parsing, submission, or confirmation failures

## Job Applicability

Per-incident job repair decision.

- **Incident key**: incident being repaired
- **Job key**: present for job-backed incidents
- **Retry applicability**: applicable, skipped, or not_applicable
- **Timeout applicability**: applicable, skipped, or not_applicable
- **Reason**: human and machine-readable explanation when not applicable

## Repair Result

Aggregate workflow output.

- **Request**: sanitized repair request
- **Discovery**: frozen repair set and discovery status
- **Plan**: repair plan items
- **Variable updates**: per-scope update results
- **Job updates**: per-incident retry and timeout results
- **Resolution**: per-incident resolution and confirmation results
- **Remaining incident summary**: post-repair active or unresolved context
- **Outcome**: planned, repaired, partially_failed, or failed
- **Notices**: workflow-level semantic notices
- **Errors**: workflow-level errors

## Audit Report

Structured report model rendered as Markdown or JSON.

- **Schema version**
- **Command name**
- **Started and finished timestamps**
- **Duration**
- **Dry-run flag**
- **c8volt version**
- **Configured Camunda version**
- **Safe profile identity**
- **Discovery mode, filters, and input keys**
- **Frozen incident and process-instance keys**
- **Root keys, flow node ids, and element instance keys when known**
- **Job keys and job applicability**
- **Original incident state and error context**
- **Requested variable names and per-scope update status**
- **Requested retries and retry update status**
- **Requested timeout and timeout update status**
- **Incident resolution and confirmation status**
- **Remaining incident summary**
- **Skipped and not-applicable steps**
- **Notices, errors, and final outcome**

## State Transitions

1. **planned**: Discovery and validation completed without mutation, usually for dry-run.
2. **submitted**: A variable, job, or incident mutation request was accepted.
3. **confirmed**: The requested observable state was confirmed.
4. **blocked**: A required predecessor step failed, so dependent repair did not proceed.
5. **not_applicable**: The step does not apply to the incident, such as job repair for a non-job incident.
6. **failed**: The step or workflow failed.
7. **partially_failed**: At least one repair target failed while another target was planned, submitted, or confirmed.
