# CLI Contract: `c8volt ops analyse slow-process-instances`

## Command Surface

```text
c8volt ops analyse slow-process-instances [flags]
c8volt ops analyze slow-process-instances [flags]
```

Both spellings are supported and must behave the same.

## Selection Modes

Exactly one selection mode is required.

### Explicit-Key Mode

```text
c8volt ops analyse slow-process-instances --key <process-instance-key>
c8volt ops analyse slow-process-instances --key <process-instance-key> --key <process-instance-key>
c8volt ops analyse slow-process-instances -k <process-instance-key>
```

Stdin keys are accepted with one positional `-`:

```text
c8volt get pi --state active --keys-only |
  c8volt ops analyse slow-process-instances -
```

Flag keys and stdin keys may be combined:

```text
printf '2251799813694200\n' |
  c8volt ops analyse slow-process-instances --key 2251799813694100 -
```

Rules:

- Deduplicate keys before lookup.
- Validate every supplied key.
- Reject empty stdin.
- Accept no positional argument other than one `-`.
- Do not silently discard invalid, missing, or unauthorized keys.
- Do not require a process-definition selector.
- Allow keys from different process definitions.
- Preserve the established tenant contract for explicit process-instance keys.

### Process-Definition Search Mode

```text
c8volt ops analyse slow-process-instances --bpmn-process-id <bpmn-process-id>
c8volt ops analyse slow-process-instances --pd-key <process-definition-key>
```

Exactly one selector is required:

- `--bpmn-process-id`
- `--pd-key`

Supported process-instance filters:

- `--state active|completed|canceled|terminated|all`
- `--start-date-after`
- `--start-date-before`
- `--end-date-after`
- `--end-date-before`
- `--no-incidents-only`
- `--batch-size`
- `--limit`

Rules:

- Date filters accept precise timestamps and `YYYY-MM-DD`.
- Relative-day filters are not supported.
- `--incidents-only` is not supported.
- `--batch-size` and `--limit` apply only to process-instance discovery.
- A search with no matching process instances succeeds with an empty analysis result.

## Invalid Combinations

| Combination | Contract |
|-------------|----------|
| No selection mode | Reject before remote lookup with a clear validation error. |
| Both explicit keys and process-definition selector | Reject before remote lookup with a clear validation error. |
| Both `--bpmn-process-id` and `--pd-key` | Reject before remote lookup with a clear validation error. |
| Explicit keys plus process-instance search filters | Reject before remote lookup with a clear validation error. |
| More than one positional `-` | Reject before remote lookup with a clear validation error. |
| Empty stdin in explicit-key mode | Reject with a clear validation error. |
| Invalid duration value for `--duration-after` | Reject before remote lookup with a clear validation error. |

## Detail Filters

Supported in both selection modes:

- `--element-id`
- `--type`
- `--element-state`
- `--duration-after <duration>`

Rules:

- `--state` remains the process-instance state filter.
- `--element-state` filters runtime element state.
- Duration values accept unit forms such as `500ms`, `1s`, `2m`, and `1h`.
- Detail filters apply after complete timeline construction and all timing/comparison calculations.
- Element rows match only when all element predicates match.
- Transition rows remain visible when at least one endpoint matches the element predicates.
- `--duration-after` applies independently to element durations and transition durations.
- Process-instance root rows are never removed by detail filters.
- Filtering never creates synthetic transition timings across hidden elements.

## Duration Rules

### Process Instance Duration

- Valid `endDate`: `endDate - startDate`
- Active without `endDate`: `capturedAt - startDate`
- Terminal without usable `endDate`: unavailable

Ordering:

- Available process-instance durations sort longest to shortest.
- Unavailable process-instance durations sort after all available durations.
- Ties use deterministic tie-breaking.

### Element Duration

- Valid `endDate`: `endDate - startDate`
- Active without `endDate`: `capturedAt - startDate`
- Otherwise: unavailable

Terminated elements with valid timestamps are measured.

### Transition Duration

For adjacent elements in the complete chronological timeline:

```text
duration = next.startDate - previous.endDate
```

Emit a transition only when both timestamps are valid and `next.startDate >= previous.endDate`.

## Human Output

Root rows follow established `c8volt get pi` style and include available process-instance identity, tenant, BPMN process ID, version, state, timestamps, parent context, incident marker, and `dur:`.

Process instances are ordered by available duration longest to shortest, followed by unavailable durations.

Elements are listed below each process instance in chronological order:

1. `startDate` ascending
2. element-instance key as deterministic tie-breaker

Element rows include:

```text
<elementInstanceKey> <elementType> <elementId> <elementState> s:<start> [e:<end>] dur:<duration> [inc!|inc!:<incidentKey>]
```

Transition rows use:

```text
FromElement -> ToElement: <duration>
```

Rules:

- Do not add `between:` or `transition:` prefixes.
- Do not claim causality.
- Do not emit negative transitions.
- Place relative-duration indicators directly after durations.
- Do not label visual indicators with terms such as `cohort`, `peer`, `similar`, or `compared`.
- Show `PI:<percentage>` for valid process-duration shares.
- Omit relative comparison bars when fewer than three comparable measurements exist.
- End with the final process-instance count.

Empty process-definition search output:

- Shows the final process-instance count.
- Does not render root or detail rows.

## JSON Output

JSON output uses stable explicit fields. Payload fields include:

- request and captured analysis time
- ordered process-instance items
- process-instance duration and duration milliseconds
- element fields and timestamps
- element duration and duration milliseconds
- transition endpoint identifiers and timestamps
- transition duration and duration milliseconds
- relative percentile
- comparison sample count
- process-duration share

Representative payload shape:

```json
{
  "capturedAt": "2026-07-18T08:24:32Z",
  "count": 1,
  "items": [
    {
      "key": "2251799813694100",
      "tenantId": "tenant-a",
      "bpmnProcessId": "OrderProcess",
      "processDefinitionKey": "2251799813687001",
      "processVersion": 7,
      "state": "COMPLETED",
      "startDate": "2026-07-18T08:10:00Z",
      "endDate": "2026-07-18T08:24:32Z",
      "duration": "14m32s",
      "durationMillis": 872000,
      "durationAvailable": true,
      "relativePercentile": 93,
      "comparisonSampleCount": 12,
      "timeline": [
        {
          "kind": "element",
          "elementInstanceKey": "4108",
          "elementId": "ReserveStock",
          "type": "SERVICE_TASK",
          "state": "COMPLETED",
          "startDate": "2026-07-18T08:10:00.300Z",
          "endDate": "2026-07-18T08:10:04.500Z",
          "duration": "4.2s",
          "durationMillis": 4200,
          "durationAvailable": true,
          "relativePercentile": 64,
          "comparisonSampleCount": 8,
          "processDurationShare": 1
        },
        {
          "kind": "transition",
          "fromElementInstanceKey": "4108",
          "fromElementId": "ReserveStock",
          "fromElementType": "SERVICE_TASK",
          "fromEndDate": "2026-07-18T08:10:04.500Z",
          "toElementInstanceKey": "4122",
          "toElementId": "OrderFinished",
          "toElementType": "END_EVENT",
          "toStartDate": "2026-07-18T08:24:24.500Z",
          "duration": "14m20s",
          "durationMillis": 860000,
          "relativePercentile": 99,
          "comparisonSampleCount": 7,
          "processDurationShare": 99
        }
      ]
    }
  ]
}
```

Empty process-definition search JSON returns an empty `items` list and `count: 0`.

## Keys-Only Output

Keys-only output prints only process-instance keys:

```text
2251799813694100
2251799813694200
```

Rules:

- Emit unique process-instance keys.
- Emit one key per line.
- Emit keys in root result ordering.
- Print nothing for empty process-definition search results.
- Detail filters do not change which root keys are emitted.

## Unsupported Version Contract

- Camunda 8.8 and 8.9 are supported.
- Camunda 8.7 returns the established unsupported-version error.
- Unsupported-version failures must not render partial success output.

## Read-Only Contract

- The command must not mutate process instances, elements, incidents, jobs, process definitions, or variables.
- The command must not perform repair, cancellation, deletion, resolution, update, retry, or report-file generation workflows.
