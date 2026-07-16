# Data Model: Runtime Element Instance Command

## Runtime Element Instance

Represents one runtime execution occurrence of a BPMN element.

### Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `elementInstanceKey` | string | Yes | Stable primary key for lookup, keys-only output, and first human column. |
| `elementId` | string | Yes | BPMN element identifier from the process model. |
| `elementName` | string | No | Human-readable BPMN element name when available. |
| `type` | string | Yes | Camunda element type such as `SERVICE_TASK`, `END_EVENT`, or `MULTI_INSTANCE_BODY`. |
| `state` | string | Yes | Runtime lifecycle state such as `ACTIVE`, `COMPLETED`, or `TERMINATED`. |
| `startDate` | timestamp string | No | Rendered in human output through existing timestamp helpers. |
| `endDate` | timestamp string | No | Omitted from human output when absent. |
| `processInstanceKey` | string | Yes | Owning process instance key. |
| `rootProcessInstanceKey` | string | No | Root process instance key for child executions when available. |
| `processDefinitionId` | string | No | BPMN process identifier associated with the element instance. |
| `processDefinitionKey` | string | No | Process definition key associated with the element instance. |
| `tenantId` | string | No | Tenant context for multi-tenant clusters. |
| `hasIncident` | bool | Yes | Drives the compact incident marker in human rows. |
| `incidentKey` | string | No | Uses `inc!:<incidentKey>` when present; otherwise incident rows use `inc!`. |

### Identity & Uniqueness

- `elementInstanceKey` uniquely identifies a runtime element instance.
- Multiple rows may share the same `elementId` when loops or multi-instance behavior execute the same BPMN element more than once.
- `elementId` is not a unique runtime identifier and must not be used as the keys-only output.

### State Rules

- Active instances may have no `endDate`.
- Completed or terminated instances may include an `endDate`.
- Unknown or newly introduced Camunda states should be preserved as received for JSON and compact output unless local validation rejects the operator-provided search value.

## Element Search Request

Represents the read-only search criteria supplied by `c8volt get element`.

### Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `processInstanceKey` | string | No | From `--pi-key`; must be a valid key. |
| `elementId` | string | No | From `--element-id`. |
| `state` | string | No | From `--state`; validated against supported element states where practical. |
| `type` | string | No | From `--type`; normalized/validated consistently with generated enum values where practical. |
| `processDefinitionKey` | string | No | From `--pd-key`; must be a valid key. |
| `bpmnProcessId` | string | No | From `--bpmn-process-id`. |
| `batchSize` | int32 | No | Positive page size up to the existing backend maximum. |
| `limit` | int32 | No | Positive cap across pages when supplied. |

### Validation Rules

- Search filters combine with AND semantics.
- `--key` direct lookup is mutually exclusive with every search filter.
- `--total` is mutually exclusive with `--json` and `--keys-only`.
- `--limit` must be positive when explicitly supplied.
- `--batch-size` must be positive and within the existing maximum.

## Element Search Page

Represents one bounded page of element search results.

### Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `items` | Runtime Element Instance[] | Yes | Trimmed to requested page/limit boundaries. |
| `request` | Page Request | Yes | Captures effective `from`, cursor, and/or size used by the service. |
| `overflowState` | enum | Yes | `no_more`, `has_more`, or project-equivalent state. |
| `reportedTotal` | Reported Total | No | Exact or lower-bound total when supplied by Camunda. |

## Element Search Result

Represents collected element search output for JSON and non-incremental rendering.

### Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `total` | int32 | Yes | Number of collected items in bounded JSON/list output. |
| `items` | Runtime Element Instance[] | Yes | Stable machine-readable element payload. |

## Relationships

- A process instance has zero or more runtime element instances.
- A process definition has zero or more runtime element instances across process instances.
- A runtime element instance may have zero or one directly reported incident marker.
- A future process-instance enrichment story may attach runtime element instances to process-instance output, but this feature only exposes standalone element lookup/search.
