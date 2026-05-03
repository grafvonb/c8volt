# Data Model: Keyed Process-Instance Incident Details

## Process Instance Lookup Result

Existing public process-instance result returned by direct keyed lookup.

**Fields used by this feature**:

- `key`: Process-instance key used to associate incident details.
- `tenantId`: Tenant identifier shown in output and used to keep request/response context tenant-aware.
- `incident`: Existing boolean marker that indicates whether the process instance has an incident.
- Existing display fields such as BPMN process ID, version, state, start date, end date, and parent key remain unchanged.

## Incident Detail

Supplemental incident data returned only when incident enrichment is requested.

**Fields**:

- `incidentKey`: Incident identifier when returned by the generated client.
- `processInstanceKey`: Process-instance key associated with the incident.
- `tenantId`: Tenant ID associated with the incident when returned.
- `state`: Incident state when returned.
- `errorType`: Incident error type when returned.
- `errorMessage`: Human-facing diagnostic message for the incident.
- `flowNodeId`: Flow node ID when returned.
- `flowNodeInstanceKey`: Flow node instance key when returned.
- `jobKey`: Job key when returned.

**Validation rules**:

- `processInstanceKey` must match the process instance being enriched.
- Empty `errorMessage` values are allowed and must not break rendering.
- Missing optional metadata fields must be omitted from JSON rather than rendered as misleading placeholder values.

## Incident-Enriched Process Instance

Output wrapper used only for `--with-incidents`.

**Fields**:

- `item`: The original process-instance lookup result.
- `incidents`: Incident details associated with `item.key`.

**Relationships**:

- One process instance can have zero or more incident details.
- Incident details must not be associated with a different process-instance key.

## Incident-Enriched Process Instances

Collection wrapper used for multiple keys and JSON output.

**Fields**:

- `total`: Count of returned process instances.
- `items`: Collection of incident-enriched process-instance items.

**Rules**:

- The item order follows the existing keyed lookup result order.
- Each item carries its own incident details, including an empty collection when enrichment was requested and no incidents were returned.
