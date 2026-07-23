# Data Model: Walk PI Elements

## Walked Process Instance

Represents one process instance selected by a walk traversal.

**Fields**:

- `key`: Unique process-instance key.
- `tenantId`: Tenant associated with the returned process instance.
- `processDefinitionId`: BPMN process definition identifier.
- `processDefinitionKey`: Process definition key.
- `processDefinitionVersion`: Process definition version.
- `processDefinitionVersionTag`: Optional process definition version tag.
- `state`: Runtime or terminal process-instance state.
- `startDate`: Process-instance start timestamp.
- `endDate`: Optional process-instance end timestamp.
- `parentProcessInstanceKey`: Optional parent process-instance key.
- `rootProcessInstanceKey`: Root process-instance key.
- `incident`: Process-instance incident marker.

**Validation rules**:

- Walk selection must be keyed.
- Selected items must preserve traversal order.
- Enrichment must not add, remove, or reorder selected process instances.

## Runtime Element Instance

Represents one runtime BPMN element attached to a walked process instance.

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
- `processDefinitionId`: BPMN process definition identifier.
- `processDefinitionKey`: Process definition key.
- `tenantId`: Tenant associated with the element.
- `hasIncident`: Whether the element has an incident marker.
- `incidentKey`: Optional direct incident key.

**Validation rules**:

- Only elements whose `processInstanceKey` matches the walked process-instance key may be attached to that row.
- Elements are displayed in stable runtime order, with deterministic tie-breaking by `elementInstanceKey`.
- Missing optional fields must not produce misleading placeholders in human output.

## Traversal Result

Represents the selected walk result before enrichment.

**Fields**:

- `mode`: Traversal mode, one of family, children, or parent.
- `outcome`: Completion status for traversal.
- `rootKey`: Root process-instance key when available.
- `keys`: Ordered selected process-instance keys.
- `edges`: Parent-child relationships for tree rendering.
- `missingAncestors`: Missing ancestor markers for partial traversal warnings.
- `warning`: Optional traversal warning.

**Validation rules**:

- Element enrichment must preserve `mode`, `outcome`, `rootKey`, `keys`, `edges`, `missingAncestors`, and `warning`.
- Family tree rendering must keep detail sections and child process instances as siblings under the correct process-instance row.

## Activity Item

Represents one walked process instance plus requested enrichment sections.

**Fields**:

- `item`: The walked process instance.
- `variables`: Optional process-instance-scope variables.
- `incidents`: Optional direct incident details or empty array when requested.
- `elements`: Optional runtime element instances or empty array when requested.

**Validation rules**:

- If `--with-elements` is requested, JSON output must include an `elements` array for each activity item, even when empty.
- Human detail sections must render in this order when present: `vars:`, `incidents:`, `elements:`.
- `--keys-only` must not be combined with `--with-elements`.
