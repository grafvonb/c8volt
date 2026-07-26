# Research: Walk PI Elements

## Decision: Reuse Existing Process-Instance Element Enrichment

Use the existing public process facade path for element enrichment: `process.API.EnrichProcessInstancesWithElements`, backed by internal process-instance enrichment that searches elements by owning process-instance key and preserves selected process-instance order.

**Rationale**: The feature extends `walk pi` with the same runtime element details already used by `get pi --with-elements`. Reusing the facade keeps command code version-neutral, preserves the unsupported Camunda 8.7 boundary, and avoids direct command-layer calls to generated clients.

**Alternatives considered**:

- Add a walk-specific element traversal service: rejected because traversal selection is already complete before enrichment and the existing enrichment behavior has the desired ownership filtering.
- Call element services directly from `cmd`: rejected by repository layering rules and would duplicate version-specific behavior.

## Decision: Enrich After Traversal Completes

Resolve the requested traversal first, convert the walked result into ordered process-instance items, then perform requested variable, incident, and element enrichment.

**Rationale**: The specification requires traversal selection, ordering, and tree relationships to remain unchanged. Enrichment after traversal makes selection behavior observable and testable, and lets validation fail before any remote enrichment work starts.

**Alternatives considered**:

- Interleave element lookup during traversal: rejected because it risks coupling enrichment failures to traversal discovery and complicates ownership/order guarantees.
- Fetch elements before traversal: rejected because the process-instance set is not known yet.

## Decision: Use The Shared Activity View Model For Combined Enrichment

Represent walked rows with `processInstanceActivityItem` so a single item can carry variables, incidents, and elements, then render sections through the existing element-aware detail formatter.

**Rationale**: The shared activity model already supports `Elements` and JSON marshalling that includes requested empty enrichment arrays. Extending walk to use this model keeps section ordering and JSON shape consistent with existing process-instance enrichment.

**Alternatives considered**:

- Add separate walk-only structs for element rows: rejected because it duplicates data mapping and increases drift risk between `get pi` and `walk pi`.
- Keep the incident-only traversal payload for element combinations: rejected because it cannot represent variables and elements together.

## Decision: Fail Whole Command On Element Lookup Failure

Any element enrichment error fails the command before human or JSON success output is rendered.

**Rationale**: c8volt favors operational proof over partial success. Operators should not mistake a partially enriched traversal for a complete diagnostic result.

**Alternatives considered**:

- Render traversal rows with warnings for failed element lookups: rejected because the issue explicitly requires failures to fail the command and avoid partially enriched success.

## Decision: Update Command Metadata And User Documentation

Add `--with-elements` to walk command help, examples, command capability metadata, README guidance, and regenerated CLI docs.

**Rationale**: The command surface changes for operators and script authors. The constitution requires documented command behavior and generated CLI reference to match shipped behavior.

**Alternatives considered**:

- Leave README unchanged and rely on CLI help only: rejected because README already documents `walk pi` enrichment usage and should remain aligned.
