# Research: Process Instance Element Enrichment

## Decision: Reuse `c8volt/element` and `internal/services/element` for runtime element lookup

**Rationale**: The standalone runtime element command from issue #241 is present in the repository and already owns Camunda 8.8/8.9 lookup/search behavior plus Camunda 8.7 unsupported-version behavior. Reusing it keeps element search, paging, conversion, and version-specific behavior below the command layer as required by the repository layering rules.

**Alternatives considered**:

- Add process-instance-specific element lookup in `cmd`: rejected because command code must not own backend search mechanics or call versioned services/generated clients.
- Duplicate element search logic in `internal/services/processinstance`: rejected because it would fork the standalone element contract and increase maintenance risk.

## Decision: Add process-instance element enrichment as a process facade/service capability

**Rationale**: `get pi` already obtains variables and incidents through `process.API` enrichment methods. Adding `EnrichProcessInstancesWithElements` keeps command orchestration uniform and lets the internal process-instance enrichment helper preserve per-process association and ordering.

**Alternatives considered**:

- Have `cmd/get_processinstance.go` call both `process.API` and `element.API`: rejected because the command layer would own multi-step enrichment aggregation and would multiply branch logic.
- Expose element enrichment only as a standalone command composition pattern: rejected because the spec requires `get pi --with-elements` as one command.

## Decision: Extend the existing activity renderer instead of adding a separate element tree renderer

**Rationale**: `cmd/cmd_views_processinstance_activity.go` already renders process-instance rows followed by `vars:` and `incidents:` sections in stable order. Extending the activity item with elements allows `vars:`, `incidents:`, and `elements:` to compose without separate renderers for every flag combination.

**Alternatives considered**:

- Add a dedicated element-enriched renderer only for `--with-elements`: rejected because combined `--with-vars --with-incidents --with-elements` would still need shared aggregation.
- Render elements with the standalone element list view under each process instance: rejected because child tree indentation and process-instance-scoped alignment need a process-instance activity renderer.

## Decision: Sort attached elements by `startDate`, then `elementInstanceKey`

**Rationale**: The issue recommends this deterministic order, and it keeps repeated or looped BPMN executions readable without aggregating them. Sorting belongs in the enrichment service so human and JSON output receive the same stable item order.

**Alternatives considered**:

- Preserve backend order: rejected because backend ordering is not guaranteed enough for stable tests and repeated-loop readability.
- Sort by element identifier: rejected because it hides runtime execution order.

## Decision: Reject `--keys-only --with-elements`

**Rationale**: Clarification selected a clear validation error. Existing `--keys-only` behavior with variable/incident enrichment is inconsistent across keyed and paged paths; this feature should not add another ambiguous enrichment/key-only combination.

**Alternatives considered**:

- Output process-instance keys only: rejected by clarification because it would silently discard the requested enrichment.
- Output element keys: rejected because `get pi --keys-only` should not switch identity domains.

## Decision: Keep `--limit`, `--batch-size`, and prompts scoped to process instances

**Rationale**: The spec states that process-instance selection remains authoritative. Element enrichment is attached after process instances are selected; element counts must not affect process-instance limits or page prompts.

**Alternatives considered**:

- Let element count affect `--limit`: rejected because it would change existing process-instance paging semantics.
- Add element-specific enrichment filters or limits: rejected as out of scope; direct element filtering belongs to `get element` / `get ei`.

## Decision: Treat Camunda 8.7 unsupported behavior as an enrichment failure

**Rationale**: The element service already returns unsupported errors for 8.7. `get pi --with-elements` should surface that error clearly rather than render process instances without requested details.

**Alternatives considered**:

- Render process instances and omit `elements:` on 8.7: rejected because it would claim success while failing the requested enrichment.
- Hide the flag for 8.7: rejected because command availability is stable and runtime capability errors are the repository pattern.
