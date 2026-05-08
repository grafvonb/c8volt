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

- `submitted` when `--no-wait` is used and the mutation is accepted.
- `confirmed` when waiting observes the incident no longer active or resolved.
- `skipped` when no resolution attempt is needed.
- `mutation_failed` when the resolution request is rejected or cannot be submitted.
- `confirmation_failed` when polling times out, retries exhaust, or the incident remains active.

## ProcessInstanceResolutionResult

Represents the result for one selected process instance.

Fields:

- `processInstanceKey`: selected process instance key.
- `attemptedIncidentKeys`: initially discovered incident keys attempted for resolution.
- `resolvedIncidentKeys`: incident keys confirmed resolved or accepted with `--no-wait`.
- `skippedIncidentKeys`: incident keys skipped or none when no active incidents exist.
- `failedIncidentKeys`: incident keys that failed mutation or confirmation.
- `confirmationStatus`: aggregate confirmation state for the selected process instance.
- `status`: `submitted`, `confirmed`, `skipped`, `partial_failed`, or `failed`.
- `error`: failure detail when lookup, mutation, or confirmation fails.

Rules:

- The attempted set is fixed from active incidents discovered at command start.
- A process instance with no active incidents is a successful no-op result.
- Partial failure does not hide successful incident resolutions.

## ResolutionResults

Aggregate command payload for either command path.

Fields:

- `items`: ordered per-target results.
- `total`: total unique targets processed.
- `submitted`: count of targets with accepted requests.
- `confirmed`: count of targets confirmed resolved.
- `skipped`: count of successful no-op targets.
- `failed`: count of targets with mutation or confirmation failures.

Rules:

- Human and JSON output must be derived from the same result model.
- Non-zero failed count should produce command failure unless existing no-error-code configuration suppresses exit codes.
