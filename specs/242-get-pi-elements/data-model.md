# Data Model: Process Instance Element Enrichment

## Process Instance

Represents the primary `get pi` result item selected by key or process-instance search.

### Fields Used By This Feature

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `key` | string | Yes | Primary process-instance identity. Controls paging, `--limit`, keyed lookup, and JSON item ownership. |
| `tenantId` | string | No | Preserved from existing process-instance lookup/search behavior. |
| `bpmnProcessId` | string | No | Rendered in process-instance row and used by existing process filters. |
| `processVersion` | int32 | No | Rendered in process-instance row. |
| `processVersionTag` | string | No | Preserved where already supported. |
| `state` | string | No | Existing process-instance lifecycle state. |
| `startDate` | timestamp string | No | Existing process-instance row timestamp. |
| `endDate` | timestamp string | No | Existing process-instance row timestamp when present. |
| `rootProcessInstanceKey` | string | No | Existing process tree context. |
| `parentKey` | string | No | Existing process tree context. |
| `incident` | bool | No | Existing marker for process-instance tree incident state. |

### Identity & Selection Rules

- Process instances remain the primary result items.
- `--limit`, `--batch-size`, interactive prompts, and `found: N` counts are based on process instances only.
- Element enrichment must not add or remove selected process instances.

## Runtime Element Instance

Represents one runtime execution occurrence of a BPMN element attached under a selected process instance.

### Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `elementInstanceKey` | string | Yes | Stable element identity and first element row column. |
| `elementId` | string | Yes | BPMN element identifier shown between `type` and `state`. |
| `elementName` | string | No | Preserved for JSON consumers when available. |
| `type` | string | Yes | Runtime element type such as `START_EVENT`, `SERVICE_TASK`, or `END_EVENT`. |
| `state` | string | Yes | Runtime element state such as `ACTIVE` or `COMPLETED`. |
| `startDate` | timestamp string | No | Rendered with `s:` in human output. |
| `endDate` | timestamp string | No | Rendered with `e:` only when present. |
| `processInstanceKey` | string | Yes | Must match the owning process instance key for attachment. |
| `rootProcessInstanceKey` | string | No | Preserved from runtime element record. |
| `processDefinitionId` | string | No | BPMN process identifier for JSON consumers. |
| `processDefinitionKey` | string | No | Process definition key for JSON consumers. |
| `tenantId` | string | No | Tenant context from the runtime element record. |
| `hasIncident` | bool | Yes | Drives compact incident marker. |
| `incidentKey` | string | No | Uses `inc!:<incidentKey>` when present. |

### Identity & Ordering

- `elementInstanceKey` uniquely identifies a runtime element instance.
- Multiple attached rows may share the same `elementId` for looped or repeated BPMN execution.
- Attached elements are sorted by `startDate`, then `elementInstanceKey`.
- Elements whose `processInstanceKey` does not match the selected process instance are ignored during attachment.

### State Rules

- Active element instances may have no `endDate`; human output omits `e:` in that case.
- Completed or ended element instances may include `endDate`.
- Incident marker rules:
  - No marker when `hasIncident` is false.
  - `inc!` when `hasIncident` is true and `incidentKey` is empty.
  - `inc!:<incidentKey>` when `hasIncident` is true and `incidentKey` is present.
  - Never render both markers for one element row.

## Element-Enriched Process Instance

Represents one process instance plus the runtime elements attached to it.

### Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `item` | Process Instance | Yes | The selected process instance. |
| `elements` | Runtime Element Instance[] | Yes | Runtime elements owned by `item.key`; empty slice when none are returned. |

### Relationships

- One process instance has zero or more runtime element instances.
- Element attachment is by `Runtime Element Instance.processInstanceKey == Process Instance.key`.
- Element attachment preserves the order of selected process instances.

## Combined Activity Item

Represents the command view model for process-instance detail sections.

### Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `item` | Process Instance | Yes | Process-instance row. |
| `variables` | Process Instance Variable[] | No | Existing `--with-vars` section. |
| `incidents` | Process Instance Incident Detail[] | No | Existing `--with-incidents` section. |
| `elements` | Runtime Element Instance[] | No | New `--with-elements` section. |
| `showIncidents` | bool | Yes | Existing marker to control indirect incident section rendering. |

### Section Ordering

When sections are present, human output renders:

1. `vars:`
2. `incidents:`
3. `elements:`

Missing or empty sections are omitted unless existing incident marker behavior requires an indirect incident note.

## Validation Rules

- `--with-elements --total` is invalid.
- `--keys-only --with-elements` is invalid.
- `--with-elements` with keyed lookup and search-mode filters is invalid.
- `--limit` applies only to process instances.
- Element-specific filters are not accepted on `get pi`; use `get element` / `get ei` instead.
- Camunda 8.7 returns unsupported-version errors when element enrichment is requested.
