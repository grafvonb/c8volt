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
c8volt resolve incident --key <incident-key> [--key <incident-key>...] [--no-wait] [--workers <n>] [--fail-fast] [--no-worker-limit]
c8volt resolve incident -
c8volt resolve inc --key <incident-key>
```

Inputs:

- Repeated `--key` flags.
- Newline-separated stdin keys when positional argument is `-`.
- Both sources may be combined and are deduplicated.

Default behavior:

- Submit resolution for each unique incident key.
- Wait until each incident is no longer active or is reported resolved.
- Report each incident independently.

`--no-wait` behavior:

- Return after accepted resolution requests.
- Mark confirmation status as skipped or not requested.

JSON payload requirements:

```json
{
  "items": [
    {
      "incidentKey": "2251799813685249",
      "mutationAccepted": true,
      "status": "confirmed",
      "confirmationStatus": "resolved",
      "statusCode": 204
    }
  ]
}
```

## Resolve Process Instance

```text
c8volt resolve process-instance --key <process-instance-key> [--key <process-instance-key>...] [--no-wait] [--workers <n>] [--fail-fast] [--no-worker-limit]
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
- Wait until the selected process instance no longer has those initially discovered active incidents.
- Report process instances with no active incidents as successful no-op results.

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
      "status": "confirmed"
    }
  ]
}
```

## Error Contract

- Unsupported Camunda versions fail before mutation.
- Invalid keys fail before mutation.
- Empty stdin fails before mutation.
- Partial failures are visible per target and should not suppress successful target output.
- Existing `--no-err-codes` behavior remains responsible for exit-code suppression when configured.
