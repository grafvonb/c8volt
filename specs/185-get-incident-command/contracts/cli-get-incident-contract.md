# CLI Contract: Get Incident Command

## Command Names

```text
c8volt get incident
c8volt get incidents
c8volt get inc
```

Behavior:

- Read-only incident lookup and search command.
- Does not expose process-instance view extender or process-instance incident result filter flags.
- Follows existing `get` command output, paging, limit, automation, and exit behavior conventions.

## Keyed Lookup

```text
c8volt get incident --key <incident-key>
c8volt get incident --key <incident-key> --key <incident-key>
c8volt get incident -
c8volt get pi --with-incidents --keys-only | c8volt get incident -
```

Inputs:

- Repeated `--key` flags.
- Newline-separated stdin keys when positional argument is `-`.
- Both sources may be combined and are deduplicated.

Default behavior:

- Fetch each unique incident key through the incident facade/service.
- Return a not-found style error for missing keys.
- Reject search/list filters when keyed lookup is selected.

Output modifiers:

- `--json` returns machine-readable incident details.
- `--keys-only` prints only incident keys.
- `--error-message-limit` applies only to human output.

## Search/List

```text
c8volt get incident
c8volt get incident --state active
c8volt get incident --state resolved
c8volt get incident --state all
c8volt get incident --error-type io_mapping_error
c8volt get incident --error-message "intentional incident"
c8volt get incident --process-instance-key 2251799813748687
c8volt get incident --root-process-instance-key 2251799813748682
c8volt get incident --process-definition-key 2251799813687048
c8volt get incident --process-definition-id C89_SimpleUserTaskWithIncident_Process
c8volt get incident --flow-node-id SimpleUserTaskWithIncident_UserTask
c8volt get incident --flow-node-instance-key 2251799813748691
c8volt get incident --creation-time-after 2026-05-08T00:00:00Z
c8volt get incident --creation-time-before 2026-05-09T00:00:00Z
```

Filters:

- `--state active|pending|resolved|migrated|unknown|all`
- `--error-type <incident-error-type>`
- `--error-message <substring>`
- `--process-instance-key <key>`
- `--root-process-instance-key <key>`
- `--process-definition-key <key>`
- `--process-definition-id <bpmn-process-id>`
- `--flow-node-id <id>`
- `--flow-node-instance-key <key>`
- `--creation-time-after <date-or-timestamp>`
- `--creation-time-before <date-or-timestamp>`

State semantics:

- Default is `active`.
- `all` means no state filter.

Error type semantics:

- User input is case-insensitive.
- Values are validated against generated Camunda incident error enum values.
- Internally normalized values match the generated enum representation.

Error message semantics:

- User-visible behavior is case-insensitive substring matching.
- Backend `$like` is used only if compatible with that behavior.
- Otherwise c8volt pages candidates and filters locally.

## Output Modes

Human row shape:

```text
<incidentKey> <tenantId> <errorType> <state> j:<jobKey|n/a> <creationTime> (<age>) <bpmn-process-id> pi:<processInstanceKey> root:<rootProcessInstanceKey> fn:<flowNodeId> fni:<flowNodeInstanceKey> m:<errorMessage>
```

JSON shape:

```json
{
  "items": [
    {
      "incidentKey": "2251799813817616",
      "tenantId": "tenant-a",
      "state": "ACTIVE",
      "errorType": "JOB_NO_RETRIES",
      "errorMessage": "No retries left",
      "creationTime": "2026-05-08T10:15:00Z",
      "processInstanceKey": "2251799813748687",
      "flowNodeId": "SimpleUserTaskWithIncident_UserTask",
      "flowNodeInstanceKey": "2251799813748691",
      "jobKey": "2251799813748700"
    }
  ]
}
```

Keys-only output:

```text
2251799813817616
2251799813817617
```

Total output:

```text
2
```

## Error Contract

- `--key` lookup combined with search filters fails before remote calls.
- `--total --json` fails before remote calls.
- `--total --keys-only` fails before remote calls.
- `--error-message-limit` with non-human output fails before remote calls.
- Invalid `--state` values fail with valid values in the message.
- Invalid `--error-type` values fail with generated valid values in the message.
- Invalid creation-time values fail before remote calls.
- Unsupported Camunda versions fail before unsupported incident requests.
- v8.8 compatibility paths avoid known broken scoped `filter` request shapes.
