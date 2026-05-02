# Data Model: Walk Process-Instance Incident Details

## Traversal Result

Represents the process-instance walk output before optional incident enrichment.

**Fields**:

- `mode`: selected traversal mode (`parent`, `children`, or `family`)
- `outcome`: traversal outcome (`complete`, `partial`, or `unresolved`)
- `startKey`: process-instance key supplied through `--key`
- `rootKey`: root process-instance key when known
- `keys`: ordered process-instance keys returned by traversal
- `edges`: parent-to-child key mapping for tree/family output
- `chain`: process-instance details keyed by process-instance key
- `missingAncestors`: missing ancestor metadata for partial family/parent outcomes
- `warning`: human-readable traversal warning when present

**Validation rules**:

- Incident enrichment must not reorder `keys`.
- Incident enrichment must not add process instances that are absent from `keys` and `chain`.
- Partial and warning metadata must survive enrichment unchanged.
- If incident lookup fails for any key, no incident-enriched traversal result is valid for rendering.

## Walked Process Instance

Represents one process instance returned by the selected walk mode.

**Key attributes**:

- `key`
- `parentKey`
- `parentProcessInstanceKey`
- `parentFlowNodeInstanceKey`
- `state`
- `tenantId`
- `incident`
- existing process-definition and timing fields

**Validation rules**:

- Existing default JSON and human output must remain unchanged when enrichment is not requested.
- The process-instance key is the join key for incident details.

## Incident Detail

Reuses issue #154 incident detail shape.

**Key attributes**:

- `incidentKey`
- `processInstanceKey`
- `tenantId`
- `state`
- `errorType`
- `errorMessage`
- `flowNodeId`
- `flowNodeInstanceKey`
- `jobKey`
- `rootProcessInstanceKey`
- `processDefinitionKey`
- `processDefinitionId`

**Validation rules**:

- Incident details must be attached to the matching `processInstanceKey`.
- Empty incident result sets must render as empty collections in JSON.
- Empty error messages must not break rendering.

## Incident-Enriched Traversal Item

Combines one walked process instance with its incident details.

**Fields**:

- `item`: walked process instance
- `incidents`: incident detail collection for `item.key`

**Validation rules**:

- `incidents` must contain only details for `item.key`.
- The item must come from the traversal result, not from incident search results.

## Incident-Enriched Traversal Result

Walk JSON payload used only when `--with-incidents` is requested.

**Fields**:

- `mode`
- `outcome`
- `rootKey`
- `keys`
- `edges`
- `items`: ordered collection of incident-enriched traversal items
- `missingAncestors`
- `warning`

**Validation rules**:

- Metadata fields must match the un-enriched traversal result.
- Item order must match `keys`.
- The payload must use the repository's existing shared JSON envelope behavior.
- The payload is emitted only after all requested incident lookups succeed.

## Incident Enrichment Failure

Represents the all-or-nothing failure state for requested incident enrichment.

**Validation rules**:

- A failed incident lookup for any walked process instance must fail the command.
- Failed incident lookups must not be represented as empty incident collections.
- Failed incident lookups must not render a partial human or JSON traversal.

## Unsupported Incident Enrichment

Represents the explicit unsupported result for versions where tenant-safe incident enrichment is unavailable.

**Validation rules**:

- Camunda 8.7 must fail with the existing unsupported-capability style.
- Unsupported behavior must occur only when enrichment is requested.
