# Research: Stable Tenant-Aware Process-Definition Ordering

## Decision 1: Own the canonical comparator in the domain layer

**Decision**: Define one reusable comparison in `internal/domain/processdefinition.go`: tenant ID exact text ascending, BPMN process ID exact text ascending, version numeric descending, and process-definition key exact text ascending.

**Rationale**: These fields are version-neutral domain data. One comparator prevents the four adapters, service, facade, command, statistics, and watch paths from acquiring subtly different orders.

**Alternatives considered**:

- Sort only in each renderer: rejected because JSON, keys-only, watch, and facade consumers could diverge.
- Keep the existing BPMN-ID/version helper: rejected because it interleaves tenants and lacks a deterministic key tie-breaker.
- Use case-insensitive, locale-aware, or numeric-key comparison: rejected by the clarified exact-text contract.

## Decision 2: Combine backend sorting with final local normalization

**Decision**: Send the strongest compatible sort to each Camunda API, accumulate the complete collection, then apply the shared comparator once before returning it.

**Rationale**: Backend sorting stabilizes cursor/page boundaries, while final normalization removes version-specific and page-arrival differences. Either layer alone is insufficient for the cross-version contract.

**Alternatives considered**:

- Backend sort only: rejected because available sort fields differ by Camunda version and native latest endpoints expose fewer fields.
- Local sort only: rejected because unstable backend page boundaries can skip or repeat results during traversal.
- Merge already-sorted pages: rejected as needless complexity for current bounded collections and heterogeneous backend guarantees.

## Decision 3: Use version-appropriate backend sort fields

**Decision**:

- Camunda 8.7 ordinary search: tenant ID ASC, BPMN process ID ASC, version DESC, key ASC through the generated generic Operate sort model.
- Camunda 8.8-8.10 ordinary search: tenant ID ASC, process definition ID ASC, version DESC, process definition key ASC.
- Camunda 8.8-8.10 native latest search: tenant ID ASC, process definition ID ASC, the fields accepted by that endpoint.

**Rationale**: These are the closest backend representations of the canonical key and provide stable traversal without editing generated clients. Final local sorting supplies any missing tie-break fields.

**Alternatives considered**:

- Force identical request shapes across versions: rejected because generated APIs and accepted latest-sort fields differ.
- Edit generated clients to widen the latest sort enum: rejected because repository policy requires regeneration from source schemas and the server contract still may not accept it.

## Decision 4: Make complete latest discovery service-owned

**Decision**: The version-neutral service traverses all available pages before returning latest results. Camunda 8.8-8.10 use native `isLatestVersion`; Camunda 8.7 retrieves visible versions and reduces locally by exact `(tenantID, BPMNProcessID)` identity.

**Rationale**: Existing adapter-level latest calls can stop at a single 1000-row page, and the Camunda 8.7 grouping currently omits tenant identity. Central traversal closes both correctness gaps and gives CLI, facade, and watch the same behavior.

**Alternatives considered**:

- Preserve single-page latest calls: rejected because it violates complete discovery.
- Reduce latest solely in the CLI: rejected by layering rules and would bypass facade/watch consumers.
- Disable native latest on newer versions: rejected because it would unnecessarily fetch older versions and increase API/memory cost.

## Decision 5: Express latest intent additively on the existing search request

**Decision**: Add `Latest bool` to the version-neutral and public process-definition search request models and route latest collection workflows through the normal paged service path.

**Rationale**: This reuses existing filtering, paging, conversion, factory, and error paths. It is additive and lets adapters decide native versus local mechanics without exposing version details upward.

**Alternatives considered**:

- Keep a separate public latest method backed by a separate service algorithm: rejected because it duplicates traversal and ordering behavior.
- Add a new package or collection abstraction: rejected as unnecessary for one resource and contrary to repository-native scope.
- Remove existing latest methods immediately: rejected because maintaining thin compatibility wrappers may minimize churn.

## Decision 6: Preserve order through statistics enrichment

**Decision**: Enrich definitions in their existing slice positions and exclude all statistics from the comparator.

**Rationale**: Counts are volatile annotations, not identity or ordering fields. Position-preserving enrichment guarantees `--stat` and watch refreshes do not move otherwise unchanged rows.

**Alternatives considered**:

- Re-sort after each asynchronous statistics response: rejected because response completion order is nondeterministic.
- Include counts as tie-breakers: rejected by the feature contract and would make watch output jump as runtime state changes.

## Decision 7: Keep facade conversion and renderers order-preserving

**Decision**: The public facade maps the shared service slice in sequence, and human, JSON, keys-only, and watch renderers consume that sequence without independent sorting.

**Rationale**: Ordering is a collection guarantee. Re-sorting at presentation boundaries risks output-mode drift and makes the facade contract weaker than the CLI contract.

**Alternatives considered**:

- Duplicate the comparator in public types: rejected because conversion already preserves order.
- Let tables impose their own order: rejected because JSON and keys-only consumers would not share it.

## Decision 8: Preserve direct retrieval and document the 8.7 bound

**Decision**: Leave direct-key and XML retrieval unchanged. Retain the existing Camunda 8.7 emulation ceiling of 1000 visible definitions and state it as a compatibility limit rather than silently claiming unbounded completeness.

**Rationale**: Direct retrieval is outside collection ordering. The 8.7 API compatibility path already overfetches/window-slices up to 1000; removing that server-era constraint would expand this issue substantially.

**Alternatives considered**:

- Refactor all process-definition retrieval together: rejected as unrelated risk.
- Claim identical unbounded paging on all versions: rejected because it is not operationally true for the current 8.7 adapter.

## Decision 9: Validate every contract boundary

**Decision**: Use a shared fixture matrix at domain, service, adapter, facade, command, watch, and documentation boundaries, including page sizes 1, 2, and 1000.

**Rationale**: The main failure modes are not only comparison mistakes; they include request sort encoding, cursor accumulation, latest grouping, statistics position, conversion order, and renderer parity.

**Alternatives considered**:

- Test only final human output: rejected because failures would be hard to localize and non-human output could regress.
- Rely only on cluster smoke tests: rejected because deterministic ties, shuffled pages, and volatile-stat refreshes need controlled fixtures.
