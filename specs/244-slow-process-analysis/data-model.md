# Data Model: Slow Process Instance Analysis

## Analysis Request

Represents one invocation of the slow process-instance analysis command.

### Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `commandName` | string | Yes | Records whether the operator invoked `analyse` or `analyze` for output metadata and diagnostics. |
| `selectionMode` | enum | Yes | `explicit_keys` or `process_definition_search`; exactly one mode is allowed. |
| `inputKeys` | string[] | Conditional | Process-instance keys supplied through flags and stdin after parsing; required for explicit-key mode. |
| `processDefinitionSelector` | Process Definition Selector | Conditional | Required for process-definition search mode. |
| `processInstanceFilters` | Process Instance Search Filters | No | Discovery filters used only in process-definition search mode. |
| `detailFilters` | Detail Filters | No | Element and transition detail filters applied after complete timeline construction. |
| `batchSize` | int32 | No | Applies only to process-instance discovery. |
| `limit` | int32 | No | Applies only to process-instance discovery. |
| `capturedNow` | timestamp | Yes | One analysis timestamp used for active process and element duration calculations. |
| `outputMode` | enum | Yes | `human`, `json`, or `keys_only`. |

### Validation Rules

- Exactly one `selectionMode` is allowed.
- Explicit-key mode requires at least one valid key from `--key`/`-k` or stdin `-`.
- Explicit-key mode accepts at most one positional `-`.
- Explicit-key mode rejects process-definition selectors and process-instance search filters.
- Process-definition search mode requires exactly one of `bpmnProcessId` or `processDefinitionKey`.
- `batchSize` and `limit` are valid only for process-definition search discovery.
- Detail filters are valid in both selection modes.

## Process Definition Selector

Identifies the process-definition cohort used by search mode.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `bpmnProcessId` | string | Conditional | Mutually exclusive with `processDefinitionKey`. |
| `processDefinitionKey` | string | Conditional | Mutually exclusive with `bpmnProcessId`. |

## Process Instance Search Filters

Selects process instances for process-definition search mode.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `state` | enum | No | `active`, `completed`, `canceled`, `terminated`, or `all`. |
| `startDateAfter` | timestamp/date | No | Accepts precise timestamps and `YYYY-MM-DD`. |
| `startDateBefore` | timestamp/date | No | Accepts precise timestamps and `YYYY-MM-DD`. |
| `endDateAfter` | timestamp/date | No | Accepts precise timestamps and `YYYY-MM-DD`. |
| `endDateBefore` | timestamp/date | No | Accepts precise timestamps and `YYYY-MM-DD`. |
| `noIncidentsOnly` | bool | No | Excludes incident-only filtering when requested by the operator. |

### Filter Rules

- Relative-day filters are not part of this feature.
- `--incidents-only` is not part of this feature.
- Search filters do not apply to explicit-key mode.

## Detail Filters

Controls which element and transition detail rows remain visible after calculations.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `elementId` | string | No | Element rows must match when supplied. |
| `type` | string | No | Element type must match when supplied. |
| `elementState` | string | No | Element state must match when supplied. |
| `durationAfter` | duration | No | Applies independently to element durations and transition durations. |

### Filter Rules

- Detail filters never remove process-instance roots.
- Detail filters are applied after complete timeline, duration, transition, and relative indicator calculations.
- Transition rows remain visible when at least one endpoint matches active element predicates.
- Filtering never creates synthetic transitions across hidden elements.

## Analyzed Process Instance

The root result item for one selected process instance.

### Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `key` | string | Yes | Primary process-instance identity and keys-only output value. |
| `tenantId` | string | No | Preserved from process-instance selection. |
| `bpmnProcessId` | string | No | Rendered in root rows when available. |
| `processDefinitionKey` | string | No | Used for comparison grouping. |
| `processVersion` | int32 | No | Rendered in root rows when available. |
| `state` | string | No | Runtime state at selection time. |
| `startDate` | timestamp | No | Used for process duration when valid. |
| `endDate` | timestamp | No | Used for process duration when valid. |
| `parentKey` | string | No | Existing process-instance parent context. |
| `rootProcessInstanceKey` | string | No | Existing process-instance root context. |
| `incident` | bool | No | Drives existing incident marker. |
| `duration` | duration string | No | Human-readable whole duration when available. |
| `durationMillis` | int64 | No | Machine-readable whole duration when available. |
| `durationAvailable` | bool | Yes | False for terminal instances without usable whole duration. |
| `relativePercentile` | int | No | Rounded process-duration percentile when enough comparable roots exist. |
| `comparisonSampleCount` | int | No | Number of comparable root durations. |
| `timeline` | Timeline Entry[] | Yes | Ordered element and transition entries after calculation. |

### Identity & Ordering

- One selected process instance produces at most one analyzed root.
- Duplicate explicit keys are analyzed once.
- Roots with available duration sort longest to shortest.
- Roots with unavailable duration sort after all available durations.
- Tie-breaking must be deterministic across human, JSON, and keys-only modes.

## Runtime Element Timing

Represents one runtime element execution in a process-instance timeline.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `kind` | string | Yes | `element`. |
| `elementInstanceKey` | string | Yes | Stable runtime element identity. |
| `elementId` | string | Yes | BPMN element identifier. |
| `type` | string | Yes | Runtime element type. |
| `state` | string | Yes | Runtime element state. |
| `startDate` | timestamp | No | Used for duration when valid. |
| `endDate` | timestamp | No | Used for duration when valid. |
| `duration` | duration string | No | Human-readable duration when available. |
| `durationMillis` | int64 | No | Machine-readable duration when available. |
| `durationAvailable` | bool | Yes | False when timestamps cannot produce a duration. |
| `hasIncident` | bool | No | Drives compact incident marker. |
| `incidentKey` | string | No | Included in compact marker when available. |
| `relativePercentile` | int | No | Rounded element-duration percentile when enough comparable elements exist. |
| `comparisonSampleCount` | int | No | Number of comparable element measurements. |
| `processDurationShare` | int | No | Rounded percentage of root process duration when available. |

### Ordering

- Elements use the same chronological order as `c8volt get pi --with-elements`: start date ascending, then element-instance key.
- Elements are not sorted by duration.

## Transition Timing

Represents elapsed time between adjacent chronological runtime element entries.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `kind` | string | Yes | `transition`. |
| `fromElementInstanceKey` | string | Yes | Previous element instance identity. |
| `fromElementId` | string | Yes | Previous endpoint identifier shown in human output. |
| `fromElementType` | string | Yes | Previous endpoint type for comparison grouping. |
| `fromEndDate` | timestamp | Yes | Previous endpoint end timestamp. |
| `toElementInstanceKey` | string | Yes | Next element instance identity. |
| `toElementId` | string | Yes | Next endpoint identifier shown in human output. |
| `toElementType` | string | Yes | Next endpoint type for comparison grouping. |
| `toStartDate` | timestamp | Yes | Next endpoint start timestamp. |
| `duration` | duration string | Yes | Human-readable transition duration. |
| `durationMillis` | int64 | Yes | Machine-readable transition duration. |
| `relativePercentile` | int | No | Rounded transition-duration percentile when enough comparable transitions exist. |
| `comparisonSampleCount` | int | No | Number of comparable transition measurements. |
| `processDurationShare` | int | No | Rounded percentage of root process duration when available. |

### Transition Rules

- Transitions are calculated only between adjacent elements in the complete chronological timeline.
- A transition exists only when both timestamps are valid and the next start is at or after the previous end.
- Negative transition durations are never emitted.
- Missing, invalid, or filtered-out elements are never bridged.
- The arrow expresses chronological order only, not causality.

## Analysis Result

Represents the complete output payload independent of render mode.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `request` | Analysis Request | Yes | Normalized request used for analysis. |
| `capturedAt` | timestamp | Yes | Single analysis time. |
| `items` | Analyzed Process Instance[] | Yes | Frozen, ordered process-instance roots. |
| `count` | int | Yes | Number of selected process instances. |
| `empty` | bool | Yes | True when process-definition search matched no process instances. |
| `warnings` | string[] | No | Non-fatal informational notices if any are needed. |

### Output Relationships

- Human output renders `items` as root rows and detail sections plus a final count.
- JSON output renders explicit fields for roots, element timings, transition timings, comparisons, and duration shares.
- Keys-only output renders only `items[*].key`, one per line, in result ordering.
