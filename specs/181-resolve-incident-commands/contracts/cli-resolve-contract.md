# CLI Contract: Resolve Incident Commands

## Root Command

```text
c8volt resolve
```

Behavior:

- State-changing root command.
- Includes shared backoff and automation-compatible behavior.
- Lists `incident` and `process-instance` children in help and capability output.

## Resolve Incident

```text
c8volt resolve incident --key <incident-key> [--key <incident-key>...] [--dry-run] [--no-wait] [--workers <n>] [--fail-fast] [--no-worker-limit]
c8volt resolve incident -
c8volt resolve inc --key <incident-key>
```

Inputs:

- Repeated `--key` flags.
- Newline-separated stdin keys when positional argument is `-`.
- Both sources may be combined and are deduplicated.

Default behavior:

- Submit resolution for each unique incident key.
- Wait by polling incident lookup through the incident service until each incident is no longer active.
- Report each incident independently.

`--dry-run` behavior:

- Load current incident state for each unique supplied incident key where supported.
- Report which incident keys would be submitted for resolution.
- Submit no mutation and perform no confirmation polling.
- Include `dryRun: true` and `mutationSubmitted: false` in JSON output.

`--no-wait` behavior:

- Return after accepted resolution requests.
- Mark confirmation status as skipped and do not run post-mutation lookup polling.

JSON payload requirements:

```json
{
  "items": [
    {
      "incidentKey": "2251799813685249",
      "mutationAccepted": true,
      "status": "confirmed",
      "confirmationStatus": "resolved",
      "statusCode": 204,
      "dryRun": false,
      "mutationSubmitted": true
    }
  ]
}
```

JSON dry-run payload requirements:

```json
{
  "operation": "resolveIncident",
  "dryRun": true,
  "mutationSubmitted": false,
  "items": [
    {
      "incidentKey": "2251799813685249",
      "status": "planned",
      "wouldResolve": true
    }
  ]
}
```

## Resolve Process Instance

```text
c8volt resolve process-instance --key <process-instance-key> [--key <process-instance-key>...] [--dry-run] [--no-wait] [--workers <n>] [--fail-fast] [--no-worker-limit]
c8volt resolve process-instance -
c8volt resolve pi --key <process-instance-key>
```

Inputs:

- Repeated `--key` flags.
- Newline-separated stdin keys when positional argument is `-`.
- Both sources may be combined and are deduplicated.

Default behavior:

- Discover active incidents for each selected process instance at command start.
- Resolve the discovered incident set.
- Wait by polling the same process-instance incident lookup path used for discovery until the selected process instance no longer has those initially discovered active incidents.
- Report process instances with no active incidents as successful no-op results.

`--dry-run` behavior:

- Discover active incidents for each selected process instance at command start.
- Report the incident keys that would be resolved.
- Report process instances with no active incidents as successful no-op planned results.
- Submit no mutation and perform no confirmation polling.
- Include `dryRun: true` and `mutationSubmitted: false` in JSON output.

JSON payload requirements:

```json
{
  "items": [
    {
      "processInstanceKey": "2251799813685250",
      "attemptedIncidentKeys": ["2251799813685249"],
      "resolvedIncidentKeys": ["2251799813685249"],
      "skippedIncidentKeys": [],
      "failedIncidentKeys": [],
      "confirmationStatus": "resolved",
      "status": "confirmed",
      "dryRun": false,
      "mutationSubmitted": true
    }
  ]
}
```

JSON dry-run payload requirements:

```json
{
  "operation": "resolveProcessInstance",
  "dryRun": true,
  "mutationSubmitted": false,
  "items": [
    {
      "processInstanceKey": "2251799813685250",
      "attemptedIncidentKeys": ["2251799813685249"],
      "status": "planned"
    }
  ]
}
```

## Error Contract

- Unsupported Camunda versions fail before mutation.
- Invalid keys fail before mutation.
- Empty stdin fails before mutation.
- Dry-run lookup failures are reported before mutation and no resolution requests are submitted.
- `--json --verbose` fails before lookup or mutation, including dry-run mode.
- Partial failures are visible per target and should not suppress successful target output.
- Existing `--no-err-codes` behavior remains responsible for exit-code suppression when configured.
