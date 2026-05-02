# Data Model: Graceful Orphan Parent Traversal

## Traversal Resolution Result

- **Purpose**: Represents the full outcome of an ancestry, descendants, or family traversal after the feature changes orphan-parent handling from an error-only contract to a partial-result contract.
- **Key fields**:
  - Shared result type: `TraversalResult`
  - Traversal mode: `Ancestry`, `Descendants`, or `Family`
  - Resolved keys in traversal order
  - Resolved chain of process-instance records keyed by process-instance key
  - Optional edges map for tree/family rendering
  - Missing ancestor keys (machine-readable)
  - Warning status and user-facing warning message
  - Outcome status: `Complete`, `Partial`, or `Unresolved`
- **Invariants**:
  - `Complete` has no missing ancestor keys.
  - `Partial` has at least one resolved actionable result and at least one missing ancestor key.
  - `Unresolved` has no resolved actionable result and must be treated as a normal failure by affected callers.

## Resolved Process Instance Set

- **Purpose**: Captures every process instance that was successfully loaded before traversal hit a missing ancestor boundary.
- **Key attributes**:
  - Ordered key list for rendering or follow-up expansion
  - Map of key to `ProcessInstance`
  - Root key when known
- **Invariants**:
  - Resolved keys remain usable even when an ancestor boundary is incomplete.
  - The set must not include guessed or placeholder instances for missing ancestors.

## Missing Ancestor Record

- **Purpose**: Represents a parent process-instance key that could not be resolved during upward traversal.
- **Key attributes**:
  - Missing key
  - Starting key or traversal context that encountered it
  - Whether it was the first missing ancestor boundary or one of several missing ancestors
- **Behavioral rules**:
  - Missing ancestor keys must be carried as machine-readable metadata.
  - Missing ancestor records explain why a traversal is partial but are not themselves actionable process instances.

## Partial Family Result

- **Purpose**: Represents the data consumed by `walk`, cancel/delete preflight, and indirect cleanup when family traversal can continue only with incomplete ancestry.
- **Key attributes**:
  - Family key list
  - Family edges for tree rendering
  - Family chain map
  - Missing ancestor records
  - Warning outcome
- **Invariants**:
  - Tree/list rendering must stay possible when the family result is partial.
  - Preflight callers must be able to distinguish family keys they can act on from ancestor keys they could not resolve.

## Preflight Expansion Outcome

- **Purpose**: Represents the dry-run dependency expansion used before cancel/delete and indirect process-definition cleanup.
- **Key attributes**:
  - Shared result type: `DryRunPIKeyExpansion`
  - Requested keys
  - Resolved root keys
  - Resolved collected family keys
  - Missing ancestor keys
  - Warning outcome
  - Final command success boundary (`actionable` vs `unresolved`)
- **Behavioral rules**:
  - Preflight remains actionable when at least one resolved key exists.
  - Preflight must fail normally when no process-instance data could be resolved at all.
  - Callers must be able to surface both resolved and missing keys in one consistent contract.

## Strict Single-Resource Outcome

- **Purpose**: Represents the direct lookup/state-check/waiter contract that is intentionally unchanged by this feature.
- **States**:
  - `Matched`
  - `NotFound`
  - `AbsentSatisfied` for wait flows where absence/deleted is the desired state
  - `Failure`
- **Invariants**:
  - This contract remains separate from traversal partial-result behavior.
  - Missing ancestor metadata is never injected into direct key lookups.

## Affected Command Family

- **Purpose**: Groups the user-facing flows that consume traversal or dependency-expansion results.
- **Members**:
  - `walk process-instance --parent`
  - `walk process-instance`
  - `walk process-instance --flat`
  - `cancel process-instance` keyed and paged preflight
  - `delete process-instance` keyed and paged preflight
  - Indirect process-definition deletion paths that reuse process-instance dry-run expansion
- **Invariants**:
  - Every affected command family must share the same warning metadata contract.
  - Success with warnings is allowed only when resolved actionable results exist.

## Versioned Traversal Adapter

- **Purpose**: Represents the version-specific service (`v87`, `v88`, `v89`) that delegates traversal to the shared walker while preserving each version’s existing direct lookup/state semantics.
- **Fields**:
  - Version identifier
  - Shared walker delegation status
  - Direct lookup behavior
  - State-check behavior
  - Traversal partial-result compatibility
- **Current mapping**:
  - `v8.7`: shared traversal via walker, but direct key/state lookups remain strict unsupported for tenant-unsafe seams
  - `v8.8`: shared traversal via walker, tenant-safe direct lookup/state behavior already exists
  - `v8.9`: shared traversal via walker, tenant-safe direct lookup/state behavior already exists
