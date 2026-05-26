# Data Model: Job Ops Workflow Primitives

## Job Detail

Represents the observed job record used by keyed lookup, list/search output, dry-run plans, and mutation confirmation.

**Fields**:

- `key`: unique Camunda job key.
- `state`: job lifecycle state.
- `retries`: remaining retry count.
- `deadline`: activation deadline when present.
- `type`: job type.
- `worker`: worker name when present.
- `kind`: job kind, such as BPMN element or listener job.
- `listenerEventType`: listener event type for listener jobs.
- `processInstanceKey`: associated process instance key.
- `elementInstanceKey`: associated element instance key.
- `elementId`: BPMN element ID.
- `errorCode`: failed job or BPMN error code when present.
- `errorMessage`: failed job or BPMN error message when present.
- `tenantId`: tenant identifier when available.

**Validation Rules**:

- Job keys and numeric keys must parse as non-empty Camunda keys before remote calls.
- Job terminology uses `elementId` and `elementInstanceKey`; no new `flowNode` or `fni` aliases are valid for jobs.

## Job Query

Represents either exact keyed lookup or list/search mode.

**Fields**:

- `key`: optional exact job key.
- `state`: optional job state filter.
- `type`: optional job type filter.
- `processInstanceKey`: optional process instance key filter from `--pi-key`.
- `elementInstanceKey`: optional element instance key filter.
- `elementId`: optional BPMN element ID filter.
- `worker`: optional worker filter.
- `retries`: optional exact retry count filter.
- `kind`: optional job kind filter.
- `listenerEventType`: optional listener event type filter.
- `limit`: maximum number of jobs returned in list/search mode.

**Validation Rules**:

- `key` mode and list/search filters are mutually exclusive.
- `limit` must be positive and follow existing command limit conventions.
- enum-like values must be normalized or rejected before remote calls.

## Job Update Request

Represents the existing retry/timeout update path.

**Fields**:

- `key`: target job key.
- `retries`: optional retry count update.
- `timeout`: optional timeout duration.
- `timeoutMillis`: timeout converted to milliseconds.
- `dryRun`: whether mutation is preview-only.
- `noWait`: whether confirmation polling is skipped.
- `autoConfirm`: whether interactive confirmation is bypassed.
- `automation`: whether automation mode is active.

**Validation Rules**:

- At least one of `retries` or `timeout` is required for update mode.
- retry count must be non-negative.
- timeout must be a positive duration.
- update mode cannot be combined with `--throw-bpmn-error` or `--complete`.

## Job Worker Outcome

Represents one mutually exclusive worker outcome mutation.

**Variants**:

- `technicalFailure`: submits a job failure.
- `bpmnError`: throws a modeled BPMN error.
- `completion`: completes the job.

**Common Fields**:

- `key`: target job key.
- `message`: optional operator message where supported.
- `variables`: optional JSON object for failure, BPMN error, or completion variables where supported by the upstream request.
- `dryRun`: whether mutation is preview-only.
- `noWait`: whether the command returns after accepted submission where applicable.
- `autoConfirm`: whether interactive confirmation is bypassed.
- `automation`: whether automation mode is active.

**Technical Failure Fields**:

- `retries`: retry count left after failure.
- `retryBackoff`: optional positive duration converted to milliseconds.

**BPMN Error Fields**:

- `errorCode`: required modeled BPMN error code.

**Completion Fields**:

- `variables`: optional completion variables; omitted variables complete with no additional variables.

**Validation Rules**:

- exactly one worker outcome variant may be selected.
- `technicalFailure` requires the failure fields needed by the command contract.
- `bpmnError` requires a non-empty error code.
- variable payloads must parse as JSON objects before remote calls.
- worker outcome modes cannot be combined with timeout updates; `bpmnError` and `completion` cannot be combined with retry updates.

## Mutation Plan

Represents the dry-run and pre-confirmation payload for state-changing job operations.

**Fields**:

- `key`: target job key.
- `current`: observed job detail when available.
- `mode`: update, technical failure, BPMN error, or completion.
- `items`: compact list of planned changes or outcome details.
- `materialChange`: whether a mutation would be submitted.
- `dryRun`: whether this plan is preview-only.
- `mutationSubmitted`: whether a real mutation was submitted.

**State Transitions**:

- `planned` -> `dry-run complete` when `--dry-run` is supplied.
- `planned` -> `aborted` when interactive confirmation is declined.
- `planned` -> `submitted` when Camunda accepts mutation.
- `submitted` -> `confirmed` only for retry update confirmation where the read model observes requested retries.
- `submitted` -> `confirmation skipped` for timeout and worker outcome paths without a stable confirmation predicate.
- `submitted` -> `confirmation failed` when retry confirmation exhausts.
- `planned` -> `mutation failed` when submission fails.

## Unsupported Capability Error

Represents explicit pre-mutation failure for unsupported Camunda versions or missing upstream support.

**Fields**:

- `version`: configured Camunda version.
- `operation`: requested job operation.
- `message`: user-facing unsupported capability explanation.

**Validation Rules**:

- v8.7 unsupported paths must fail before mutation.
- unsupported errors must preserve machine-readable error behavior through the shared command contract.
