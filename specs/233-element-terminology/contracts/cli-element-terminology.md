# CLI Contract: Element Terminology Standardization

## Traceability

- GitHub Issue: #233
- GitHub URL: https://github.com/grafvonb/c8volt/issues/233
- Feature: `233-element-terminology`

## Incident Filter Flags

Affected commands must expose these canonical flags where incident search filters are supported:

- `--element-id <id>`
- `--element-instance-key <key>`

Affected commands must reject these legacy public flags as unknown:

- `--flow-node-id`
- `--fni-key`

Affected command surfaces:

- `c8volt get incident`
- `c8volt ops repair incident`
- `c8volt ops purge process-instances-with-incidents`

## JSON Fields

Incident JSON must use:

- `elementId`
- `elementInstanceKey`

Process-instance JSON must use:

- `parentElementInstanceKey`

Public JSON must not expose:

- `flowNodeId`
- `flowNodeInstanceKey`
- `parentFlowNodeInstanceKey`

## Human Output

Incident human rows must use:

- `e:<elementId>`
- `ei:<elementInstanceKey>`

Incident human rows must not use:

- `fn:`
- `fni:`

## Documentation And Metadata

Command help, command contract metadata, README examples, docs under `docs/cli/`, docs under `docs/ops/`, and generated index content must use the canonical element terminology after regeneration.

## Adapter Boundary

Generated clients and version-specific adapters may reference generated legacy `FlowNode*` names only when translating upstream wire shapes into canonical c8volt fields. Public command, facade, domain output, docs, and tests must not treat those names as public contract.
