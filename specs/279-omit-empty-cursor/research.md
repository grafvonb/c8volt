# Research: Omit Empty Initial Search Cursor

## Decision 1: Use the Generated Limit-Only Page Variant

**Decision**: For an initial latest process-definition search, construct each generated `SearchQueryPageRequest` with its existing `LimitPagination` variant and the requested positive limit.

**Rationale**: All v8.8, v8.9, and v8.10 generated clients expose `FromLimitPagination`, and the repository already uses this pattern in the corresponding tenant adapters. Its serialized form is exactly `{ "limit": ... }`, with neither `after` nor `from`.

**Alternatives considered**: A cursor-forward page with a nil or empty cursor was rejected because v8.8 and v8.9 require a value field and serialize the defect. Hand-built raw page JSON was rejected because the generated union already models the required shape. Editing generated clients was rejected as unnecessary and contrary to repository guidance.

## Decision 2: Use Three-Way Adapter-Local Page Selection

**Decision**: In each v8.8, v8.9, and v8.10 page builder, select cursor-forward pagination only when `After` is non-empty; otherwise select limit-only pagination for latest searches and offset pagination for ordinary searches.

**Rationale**: The defect comes from the current `preferCursor || After != ""` branch, which mistakes latest-search intent for evidence that a continuation cursor exists. A three-way branch expresses the actual protocol states without changing domain or public types.

**Alternatives considered**: Making every initial request limit-only was rejected because ordinary searches must preserve explicit offset zero. Moving the selection into commands or facades was rejected because backend wire differences belong in version-specific adapters. A new shared generic abstraction was rejected because generated union types differ by version and only three small existing builders are affected.

## Decision 3: Preserve Shared Traversal and Established Result Limits

**Decision**: Keep the version-neutral `ProcessDefinitionPageRequest`, `SearchProcessDefinitionsPages`, and direct latest-search result behavior unchanged. Rely on existing shared traversal to pass a non-empty response `EndCursor` into the next request and stop at the established limit or completion state.

**Rationale**: Shared traversal already preserves a real cursor unchanged and has focused coverage. The issue is request serialization, not public pagination API design or expansion of the direct latest lookup beyond its established 1000-item request.

**Alternatives considered**: Redesigning the facade or latest-search API to expose new paging state was rejected as out of scope. Duplicating traversal loops in all version adapters was rejected because it would add mechanics unrelated to the defect.

## Decision 4: Verify the Wire Shape Per Supported Adapter

**Decision**: Test serialized request JSON for all three adapters across initial latest, latest continuation, and ordinary offset cases, while retaining assertions for latest filters and stable sort order.

**Rationale**: Generated union accessors can decode overlapping object shapes and therefore do not alone prove that forbidden fields were absent on the wire. Per-version serialization tests also protect the small generated-type differences between v8.8/v8.9 and v8.10.

**Alternatives considered**: One shared domain test was rejected because it cannot observe adapter JSON. Testing only v8.9 was rejected because the same builder defect exists in v8.8 and v8.10. Broad command-only tests were rejected as too far from the serialization boundary.

## Decision 5: Keep the CLI and Documentation Contract Unchanged

**Decision**: Do not change commands, flags, prompts, output fields, success wording, README content, or generated CLI documentation.

**Rationale**: Operators invoke the same BPMN-ID and latest-definition workflows. Only an invalid backend request field is removed. Feature-local design documentation is sufficient for this internal compatibility correction.

**Alternatives considered**: Adding a cursor flag or exposing paging details was rejected because it would leak backend mechanics and expand the user contract. Regenerating unchanged CLI docs was rejected as noise.

## Decision 6: Combine Automated Serialization Proof with Live H2 Verification

**Decision**: Run close per-version tests and the full race-enabled suite, then manually prove the previously failing ten-instance BPMN-ID workflow on a disposable Camunda 8 Run 8.9.17 default-H2 cluster.

**Rationale**: Unit tests deterministically prove the request shape for every supported adapter. The exact RDBMS failure depends on an externally provisioned Camunda environment that the repository does not own, so the constitution requires an explicit operational path rather than pretending it is fully reproduced by local mocks.

**Alternatives considered**: Requiring `make integration-test-all` was rejected as the primary proof because it is broad, destructive, and includes unrelated real-state families. Existing focused Go integration scenarios exercise BPMN-ID validation but do not currently prove a count of ten. A new cluster provisioner was rejected as outside this defect's scope.
