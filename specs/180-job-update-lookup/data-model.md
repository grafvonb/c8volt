# Data Model: Job Lookup And Updates

## Job Key

Identifies the job to inspect or update.

- **Value**: string key supplied by `--key`.
- **Role**: Used as the Camunda job identifier for lookup and update requests.
- **Validation**: Required for both `get job` and `update job`; missing or invalid values fail before calling Camunda.

## Job Detail

Represents the job information returned by lookup and used for retries confirmation.

- **Key**: job identifier.
- **State**: current job state when available.
- **Retries**: current retry count.
- **Deadline**: timestamp returned by Camunda when present; used for display and diagnosis, not timeout confirmation.
- **Process Instance Key**: related process instance identifier when available.
- **Element Instance Key**: related element instance identifier when available.
- **Error Fields**: error message, code, or related error metadata when present.
- **Tenant Metadata**: tenant information when available.

## Job Update Request

Represents one mutation request for a single job.

- **Job Key**: target job identifier.
- **Retries**: optional requested retry count.
- **Timeout Duration**: optional requested timeout duration converted to milliseconds.
- **Validation**: at least one of retries or timeout is required.
- **Scope**: retryBackOff, job variables, fail, complete, and BPMN error behavior are excluded.

## Job Update Plan

Represents the pre-mutation plan used by dry-run, interactive confirmation, no-op handling, and JSON output.

- **Job Key**: target job identifier.
- **Current Job Detail**: visible job state loaded through job lookup before mutation.
- **Requested Retries**: optional retry count requested by the user.
- **Retry Classification**: one of change, unchanged, or not requested.
- **Requested Timeout**: optional duration requested by the user.
- **Timeout Submission**: timeout milliseconds that would be sent when timeout is requested.
- **Material Change**: true when a mutation should be submitted; retry-only unchanged requests are not material, while timeout requests are material.
- **Dry Run**: true when the plan is rendered without mutation.
- **Mutation Submitted**: false for dry-run and no-op plans, true after an accepted mutation request.

## Job Update Plan Item

Represents one planned field-level effect.

- **Name**: retries or timeout.
- **Before**: visible current value when the value has a reliable comparison predicate.
- **After**: requested value or submitted timeout duration.
- **Status**: change, unchanged, submit, or not applicable.
- **Display Rule**: unchanged retry output is compact; timeout output shows submission intent and does not show a deadline comparison.

## Timeout Duration

Represents the user-supplied timeout input.

- **Raw Input**: string such as `60s`, `5m`, or `1h`.
- **Submitted Value**: duration converted to milliseconds for Camunda.
- **Validation**: invalid or unsupported values fail before calling Camunda.
- **Confirmation Rule**: timeout is not confirmed by comparing the returned deadline timestamp.

## Observed Deadline

Represents the timestamp returned by job lookup.

- **Source**: job lookup result.
- **Role**: displayed for diagnostics and returned where supported by output modes.
- **Non-Role**: not used as proof that a requested timeout duration was applied.

## Job Update Result

Represents the command outcome for one update request.

- **Key**: target job key.
- **Plan**: pre-mutation job update plan when available.
- **Mutation Status**: accepted, failed, or unsupported.
- **Confirmation Status**: confirmed, skipped, failed, or not applicable.
- **Submitted Fields**: retries and/or timeout fields accepted by the mutation request.
- **Confirmed Fields**: retries when retries confirmation succeeds.
- **Error**: validation, mutation, unsupported-version, or retries confirmation failure details where applicable.

## Retry Confirmation

Represents the post-mutation read-model verification path for retry updates.

- **Lookup Target**: one job key.
- **Source**: same backend behavior used by `get job --key <job-key>`.
- **Success Rule**: observed retries equal the requested retry count before waiter exhaustion.
- **Failure Rule**: waiter timeout or retry exhaustion reports confirmation failure without claiming success.

## Mutation Confirmation Gate

Represents the local decision to proceed with a planned material update.

- **Trigger**: non-dry-run material update in interactive mode.
- **Auto-Confirm Bypass**: `--auto-confirm` and automation mode proceed without prompting.
- **JSON Rule**: non-dry-run JSON material updates must use auto-confirm or automation instead of prompting.
- **No-Op Rule**: retry-only unchanged plans skip the prompt and mutation.
- **No-Wait Rule**: `--no-wait` skips post-mutation confirmation only; it does not bypass the local confirmation gate.

## Unsupported Version Error

Represents the explicit failure for unsupported Camunda 8.7 job behavior.

- **Trigger**: job update runs against a Camunda 8.7 configuration.
- **Timing**: returned before any unsupported mutation request is attempted.
- **Output**: clear unsupported-version message consistent with existing domain/facade errors.
