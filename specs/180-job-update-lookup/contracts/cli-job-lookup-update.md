# CLI Contract: Job Lookup And Updates

## Scope

This contract covers user-visible behavior for:

- `c8volt get job`
- `c8volt update job`

## Valid Invocations

```bash
c8volt get job --key <job-key>
c8volt --json get job --key <job-key>
c8volt update job --key <job-key> --retries 3
c8volt update job --key <job-key> --timeout 5m
c8volt update job --key <job-key> --retries 3 --timeout 5m
c8volt update job --key <job-key> --retries 3 --dry-run
c8volt update job --key <job-key> --timeout 5m --dry-run
c8volt update job --key <job-key> --retries 3 --no-wait
c8volt --json update job --key <job-key> --retries 3 --dry-run
c8volt --json update job --key <job-key> --retries 3 --auto-confirm
```

## Invalid Invocations

```bash
c8volt get job
c8volt update job
c8volt update job --key <job-key>
c8volt update job --key <job-key> --retries invalid
c8volt update job --key <job-key> --timeout invalid
c8volt --json update job --key <job-key> --retries 3
c8volt --json --verbose update job --key <job-key> --retries 3 --dry-run
c8volt update job --key <job-key> --retries 3 # with Camunda 8.7 config
```

Expected behavior:

- Missing `--key` fails validation before calling Camunda.
- Missing update flags for `update job` fails validation before calling Camunda.
- Invalid retry counts fail validation before calling Camunda.
- Invalid timeout durations fail validation before calling Camunda.
- Non-dry-run JSON mutations fail before lookup or mutation unless `--auto-confirm` or automation mode is supplied.
- `--json --verbose` fails before lookup or mutation for `update job`, including dry-run mode.
- Camunda 8.7 job updates fail with an unsupported-version error before mutation.

## Lookup Behavior

`get job --key <job-key>` searches for the supplied job key.

When a matching job exists, output includes available job details:

- key;
- state;
- retries;
- deadline when present;
- process instance key;
- element instance key;
- error fields when present;
- tenant metadata when available.

When no matching job exists, output reports a not-found outcome suitable for human-readable and JSON modes.

## Mutation Behavior

For Camunda 8.8 and 8.9, `update job` submits one mutation request for the supplied job key.

Supported changes:

- `--retries <count>`;
- `--timeout <duration>`;
- both flags together in one request.

Unsupported changes:

- retryBackOff;
- job variables;
- fail job;
- complete job;
- throw BPMN error;
- bulk updates from filters.

## Planning And Dry-Run Behavior

Before a material mutation, `update job` loads the current job through the same lookup behavior as `get job --key <job-key>` and builds a plan.

The plan distinguishes:

- retries that would change, for example `retries: 1 -> 3`;
- retries already matching the request, for example `retries: 3 (unchanged)`;
- timeout submission intent, for example `timeout: submit 5m`;
- whether mutation was submitted.

With `--dry-run`:

- the command renders the plan and submits no mutation;
- no interactive confirmation prompt is shown;
- retry-only unchanged requests report that there is nothing to update;
- timeout requests are shown as submitted intent, not as deadline before/after comparison.

## Interactive Confirmation Behavior

For material non-dry-run updates in interactive mode:

- the command renders a compact plan;
- the command asks for confirmation before mutation;
- `--auto-confirm` and automation mode bypass the prompt;
- retry-only unchanged requests skip both the prompt and mutation;
- `--no-wait` skips post-mutation confirmation only and does not bypass this local confirmation gate.

## Default Confirmation Behavior

Unless `--no-wait` is supplied:

- updates with `--retries` wait until `get job --key <job-key>` observes the requested retry count;
- updates with both `--retries` and `--timeout` confirm the retry count only;
- timeout-only updates return submitted/accepted output after mutation acceptance and do not perform deadline confirmation;
- retry confirmation timeout or retry exhaustion reports confirmation failure.

## No-Wait Behavior

With `--no-wait`:

- the command returns after the mutation request is accepted;
- output reports submitted/accepted status;
- no retry confirmation is attempted;
- mutation failures are still reported.

## Human Output

Human output must be compact and distinguish:

- job found;
- job not found;
- dry-run plan;
- no-op update plan;
- submitted/accepted without confirmation;
- confirmed retry update;
- mutation failure;
- confirmation failure;
- unsupported version.

For timeout-only updates, human output must not imply confirmed deadline state.

Plan output must stay compact. Summary lines should use short counts and avoid phrases such as `mutation submitted: false`; dry-run output should state that no changes were applied.

## JSON Output

JSON output must be script-safe and include enough fields to distinguish:

- job key;
- lookup found or not found;
- mutation accepted status;
- confirmation status;
- submitted fields;
- confirmed fields where applicable;
- skipped or not-applicable confirmation;
- dry-run status and mutation-submitted status;
- plan items for retries and timeout;
- error details.

For timeout-only updates, JSON output must represent the timeout as submitted, not confirmed.

`--json --dry-run` returns the full stable plan payload. `--verbose` must not change JSON shape or add human detail; `--json --verbose` is rejected for `update job`.

## Command Metadata

`update job` must be marked:

- state-changing;
- contract-supported where the repository exposes command contracts;
- automation-compatible following existing metadata patterns.

`get job` must remain read-only while supporting human-readable and JSON output.
