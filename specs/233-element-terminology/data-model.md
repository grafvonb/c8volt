# Data Model: Element Terminology Standardization

## Incident Context

**Purpose**: Public incident location data used by CLI filters, JSON payloads, human rows, ops repair, and ops purge workflows.

**Canonical Fields**:
- `elementId`: BPMN element identifier associated with the incident.
- `elementInstanceKey`: Element instance key associated with the incident.

**Validation Rules**:
- Empty values remain omitted where current output omits empty context.
- Public JSON must not include `flowNodeId` or `flowNodeInstanceKey`.
- Public filters must accept `--element-id` and `--element-instance-key`.
- Public filters must not accept `--flow-node-id` or `--fni-key`.

## Process Instance Context

**Purpose**: Public parent execution context used by process-instance inspection, walk output, and incident-enriched process views.

**Canonical Fields**:
- `parentElementInstanceKey`: Parent element instance key for process-instance relationship context.

**Validation Rules**:
- Empty values remain omitted where current output omits empty parent context.
- Public JSON must not include `parentFlowNodeInstanceKey`.

## Adapter Mapping Boundary

**Purpose**: Internal translation point where generated Camunda or Operate wire fields are converted into c8volt canonical domain fields.

**Allowed Legacy Inputs**:
- Generated client fields or types containing `FlowNode*`.
- Version-specific adapter code that reads those generated fields.

**Forbidden Public Outputs**:
- Public c8volt models, JSON tags, CLI flags, command help, command metadata, README, generated docs, and human output must not expose flow-node names or `fn`/`fni` labels.

## State Transitions

This feature does not introduce new runtime states. It renames public selection and representation fields while preserving existing incident search, process-instance lookup, walk, repair, and purge behavior.
