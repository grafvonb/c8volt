# Data Model: Runtime Listener Jobs Under Elements

## Process Instance

Represents one selected workflow execution in process-instance, walk, or slow-analysis output.

**Fields**:

- `key`: Unique process-instance key.
- `tenantId`: Tenant associated with the returned process instance.
- `bpmnProcessId`: BPMN process identifier.
- `processDefinitionKey`: Process definition key.
- `processVersion`: Process definition version.
- `state`: Runtime or terminal process-instance state.
- `startDate`: Process-instance start timestamp.
- `endDate`: Optional process-instance end timestamp.
- `parentProcessInstanceKey` / `parentKey`: Parent relationship where available.
- `rootProcessInstanceKey`: Root process-instance key.
- `incident`: Process-instance incident marker.

**Validation rules**:

- Listener enrichment must not add, remove, or reorder selected process instances.
- Keyed command modes preserve backend-authorized admin input behavior and returned tenant metadata.
- Existing output without listener enrichment must remain unchanged.

## Runtime Element Instance

Represents one BPMN element execution under a process instance.

**Fields**:

- `elementInstanceKey`: Unique element instance key.
- `elementId`: BPMN element identifier.
- `elementName`: Optional display name.
- `type`: Element type.
- `state`: Runtime element state.
- `startDate`: Element start timestamp.
- `endDate`: Optional element end timestamp.
- `processInstanceKey`: Owning process-instance key.
- `rootProcessInstanceKey`: Root process-instance key.
- `processDefinitionId`: BPMN process identifier from runtime data.
- `processDefinitionKey`: Process definition key.
- `tenantId`: Tenant associated with the element.
- `hasIncident`: Whether the element has an incident marker.
- `incidentKey`: Optional direct incident key.
- `listeners`: Optional listener jobs requested for this element.

**Validation rules**:

- Only elements whose `processInstanceKey` matches the selected process instance may be attached to that process-instance row.
- Elements are displayed in stable runtime order, with deterministic tie-breaking by `elementInstanceKey`.
- If listener enrichment is not requested, `listeners` must be absent from structured element output.
- If listener enrichment is requested, `listeners` must be present on each element as an array, including an empty array when no listener jobs match.

## Runtime Listener Job

Represents a runtime job created by an execution listener or user task listener and attached to an owning element.

**Fields**:

- `jobKey`: Unique job key.
- `kind`: Listener job kind, either `EXECUTION_LISTENER` or `TASK_LISTENER`.
- `listenerEventType`: Listener event type such as `START`, `END`, `COMPLETING`, or related Camunda listener event values.
- `type`: Job type used by workers.
- `state`: Current job state.
- `retries`: Remaining retries.
- `worker`: Optional worker name.
- `deadline`: Optional job deadline.
- `processInstanceKey`: Owning process-instance key.
- `elementInstanceKey`: Owning element instance key when available.
- `elementId`: BPMN element identifier.
- `tenantId`: Tenant associated with the job.
- `errorCode`: Optional job error code.
- `errorMessage`: Optional job error message.

**Validation rules**:

- Only jobs with kind `EXECUTION_LISTENER` or `TASK_LISTENER` are listener jobs for this feature.
- A listener job must be attached only when its `elementInstanceKey` matches a selected element's `elementInstanceKey`.
- Listener jobs without a matching element instance are omitted from listener-enriched output.
- Listener jobs are sorted deterministically within each element, preferring available runtime order and falling back to `jobKey`.

## Listener-Enriched Process Instance

Represents one selected process instance plus element rows with optional listener jobs.

**Fields**:

- `item`: Selected process instance.
- `elements`: Requested runtime elements; may be empty.
- `variables`: Optional process-instance-scope variables in shared activity views.
- `incidents`: Optional incident details in shared activity views.

**Validation rules**:

- Listener enrichment requires element enrichment or an element command context.
- Human detail sections remain ordered as existing activity sections, with listener rows nested inside the `elements:` section rather than as a sibling section.
- Walk traversal metadata and row order must remain unchanged by listener enrichment.

## Slow Process Analysis Timeline Entry

Represents one slow-analysis element or transition detail row.

**Fields**:

- Existing element and transition timing fields remain unchanged.
- `listeners`: Optional listener jobs requested for element timeline rows.

**Validation rules**:

- Listener jobs attach only to element timeline entries, not transition entries.
- Default and full-timeline analysis output must remain unchanged when listener enrichment is not requested.
- Structured output includes listener arrays on element timeline entries only when requested.
