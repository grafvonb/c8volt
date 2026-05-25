# Progress: Element Terminology Standardization

## Codebase Patterns

- `cmd/get_incident.go` currently owns incident search flag grammar, including legacy `--flow-node-id` and `--fni-key`.
- `cmd/cmd_views_processinstance_incidents.go` currently renders incident context labels with `fn` and `fni`.
- Public incident fields currently live in `c8volt/incident/model.go` and map through `c8volt/incident/convert.go`.
- Public process parent context currently appears in `c8volt/process/model.go`, `c8volt/process/convert.go`, `c8volt/ops/convert.go`, and `c8volt/resource/convert.go`.
- v8.8/v8.9 generated Camunda clients already expose `elementId`, `elementInstanceKey`, and `parentElementInstanceKey` for many v2 paths; older generated Operate clients still contain `FlowNode*` fields.
- README and generated docs currently contain `--flow-node-id` and flow-node wording for incident filters.

## Planning Notes

- Clarification gate completed with no formal questions; issue #233 explicitly defines canonical names, forbidden aliases, and adapter boundary rules.
- Architecture memory was reused because the existing command/facade/domain/service/generated-client and docs-generation boundaries already cover this feature.
- Ralph launch must include `--implementation-context specs/ralph-implementation-rules.md`.

## Implementation Status

- Speckit specification: complete.
- Clarification: complete; no questions asked.
- Architecture grounding: complete; existing memory reused.
- Planning artifacts: complete.
- Tasks: complete.
