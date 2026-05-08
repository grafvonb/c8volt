# Data Model: Resolve Incident Commands

## ResolutionTarget

Represents a key selected for resolution.

Fields:

- `key`: 16-digit Camunda key from `--key` or stdin.
- `kind`: `incident` or `processInstance`.
- `source`: `flag`, `stdin`, or `merged`.

Validation:

- Keys must pass existing key validation.
- Duplicate keys across all sources are deduplicated before scheduling work.

## IncidentResolutionAttempt

Represents the result of resolving one incident key.

Fields:

- `incidentKey`: incident key attempted.
- `processInstanceKey`: process instance key when known.
- `mutationAccepted`: whether Camunda accepted the resolution request.
- `status`: `submitted`, `confirmed`, `skipped`, `mutation_failed`, or `confirmation_failed`.
- `confirmationStatus`: observed confirmation state when waiting is enabled.
- `statusCode`: response status when available.
- `message`: optional human-readable status detail.
- `error`: failure detail when resolution or confirmation fails.

State transitions:

- `planned` when dry-run identifies an incident that would be submitted for resolution.
- `submitted` when `--no-wait` is used and the mutation is accepted.
- `confirmed` when post-mutation lookup polling observes the incident is no longer active.
- `skipped` when no resolution attempt is needed.
- `mutation_failed` when the resolution request is rejected or cannot be submitted.
- `confirmation_failed` when lookup polling times out, retries exhaust, or the incident remains active.

## ProcessInstanceResolutionResult

Represents the result for one selected process instance.

Fields:

- `processInstanceKey`: selected process instance key.
- `attemptedIncidentKeys`: initially discovered incident keys attempted for resolution.
- `resolvedIncidentKeys`: incident keys confirmed no longer active by lookup polling, or accepted with `--no-wait`.
- `skippedIncidentKeys`: incident keys skipped or none when no active incidents exist.
- `failedIncidentKeys`: incident keys that failed mutation or confirmation.
- `confirmationStatus`: aggregate confirmation state from post-mutation process-instance incident lookup polling.
- `status`: `submitted`, `confirmed`, `skipped`, `partial_failed`, or `failed`.
- `error`: failure detail when lookup, mutation, or confirmation fails.

Rules:

- The attempted set is fixed from active incidents discovered at command start.
- A process instance with no active incidents is a successful no-op result.
- Dry-run discovers the attempted set and sets mutation-submission status to false.
- Non-dry-run confirmation polls the same process-instance incident lookup path until the initially discovered incident keys are no longer active.
- Partial failure does not hide successful incident resolutions.

## ResolutionPlan

Represents the pre-mutation plan used by dry-run output and command-local safety checks.

Fields:

- `operation`: `resolveIncident` or `resolveProcessInstance`.
- `requestedKeys`: unique incident or process instance keys selected by the user.
- `dryRun`: true when the plan is rendered without mutation.
- `mutationSubmitted`: always false for dry-run plans.
- `items`: per-target planned incident or process-instance outcomes.
- `wouldResolveIncidentKeys`: incident keys that would be submitted for resolution.
- `skippedIncidentKeys`: incident keys or process-instance targets requiring no mutation.
- `errors`: lookup or validation errors that prevent mutation.

Rules:

- `resolve incident --dry-run` loads current incident state where supported before classifying planned work.
- `resolve pi --dry-run` performs process-instance incident discovery through the same lookup path as non-dry-run resolution.
- Dry-run never submits incident resolution requests and never performs confirmation polling.
- `--json --dry-run` returns the full stable plan payload.

## ResolutionResults

Aggregate command payload for either command path.

Fields:

- `items`: ordered per-target results.
- `total`: total unique targets processed.
- `submitted`: count of targets with accepted requests.
- `confirmed`: count of targets confirmed resolved.
- `skipped`: count of successful no-op targets.
- `failed`: count of targets with mutation or confirmation failures.
- `dryRun`: whether the payload represents a dry-run plan.
- `mutationSubmitted`: whether any mutation request was submitted.

Rules:

- Human and JSON output must be derived from the same result model.
- Non-zero failed count should produce command failure unless existing no-error-code configuration suppresses exit codes.
