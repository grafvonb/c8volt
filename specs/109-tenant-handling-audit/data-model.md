# Data Model: Harden Tenant Handling Across Tenant-Aware Commands

## Effective Tenant Context

- **Purpose**: Represents the tenant scope a command resolves before invoking any tenant-aware process-instance operation.
- **Key attributes**:
  - Resolved tenant value
  - Winning source: `Flag`, `Environment`, `Profile`, or `BaseConfig`
  - Version context: `8.7`, `8.8`, or future `8.9` planning note
  - Whether the tenant is the implicit default or an explicit named tenant
- **Invariants**:
  - One command execution has one effective tenant context.
  - Every internal lookup in a tenant-aware flow must use the same effective tenant context.

## Tenant-Aware Operation

- **Purpose**: Represents a single process-instance operation that may expose or act on tenant-scoped resources.
- **Operation types**:
  - `DirectGet`
  - `Search`
  - `StateCheck`
  - `GetChildren`
  - `Ancestry`
  - `Descendants`
  - `Family`
  - `Wait`
  - `Cancel`
  - `Delete`
- **Key attributes**:
  - Owning command family
  - Backing service version
  - Upstream generated-client call
  - Tenant-safety capability: `Supported` or `Unsupported`
- **Invariants**:
  - Tenant-safe support must be determined per operation, not assumed for the whole command family.
  - Commands that compose multiple operations must preserve the strictest tenant contract among those operations.

## Tenant-Safe Lookup Outcome

- **Purpose**: Represents the externally visible result of a tenant-aware read path.
- **States**:
  - `Matched`: resource exists within the selected tenant context
  - `NotFound`: resource is absent or belongs to a different tenant on a supported tenant-safe path
  - `Unsupported`: the version/operation pair cannot guarantee tenant-safe behavior
- **Behavioral rules**:
  - `NotFound` must be indistinguishable for absent-resource and wrong-tenant cases on supported paths.
  - `Unsupported` must be scoped to the exact unsafe operation or flow segment.

## Version Support Record

- **Purpose**: Captures the tenant-safety status of each supported versioned service.
- **Fields**:
  - Version identifier
  - Repository support status
  - Preferred tenant-safe lookup strategy
  - Unsupported-operation notes
- **Current repository mapping**:
  - `8.7`: supported service exists, search-backed flows remain tenant-safe, but keyed direct lookup and keyed state-check seams stay explicitly unsupported where no tenant-safe upstream equivalent exists
  - `8.8`: supported service exists and search-backed lookup/state behavior is the tenant-safe authority for direct-get-adjacent flows
  - `8.9`: version normalizes in config/tooling, but no process-instance service implementation exists yet, so process-instance runtime support is still unavailable

## Tenant-Aware Command Family

- **Purpose**: Groups the CLI entry points that rely on tenant-scoped process-instance behavior.
- **Members**:
  - `get process-instance`
  - `walk pi`
  - `cancel pi`
  - `delete pi`
  - `run pi` and follow-up wait/state flows
  - Any additional command that composes process-instance direct get, search, walker, or waiter behavior
- **Invariants**:
  - Every command family in scope must receive explicit regression coverage.
  - A command family may contain both supported and unsupported operation segments depending on version.

## Derived Tenant Source Case

- **Purpose**: Represents how tenant selection entered the command when `--tenant` was not used.
- **Source variants**:
  - `Environment`
  - `Profile`
  - `BaseConfig`
- **Behavioral rules**:
  - Each variant must be tested separately.
  - Tenant propagation must be identical across all internal calls regardless of which derived source supplied the tenant.

## Traversal Chain

- **Purpose**: Represents the root, ancestry path, descendant set, and family graph used by walker-based flows.
- **Key attributes**:
  - Start key
  - Root key
  - Parent-child edges
  - Loaded process-instance records
  - Tenant context applied to all traversal steps
- **Invariants**:
  - Traversal output must remain root-inclusive and preserve explicit `nil` leaf edges where existing walker behavior depends on that shape.
  - Traversal must not cross the selected tenant boundary.

## Mutation Guard

- **Purpose**: Represents the preconditions checked before cancel or delete operates on a process instance.
- **Key attributes**:
  - Target key
  - Current tenant-safe lookup outcome
  - Current state or absence state
  - Version support decision
  - Whether `--force` applies
- **Behavioral rules**:
  - Mutation must not proceed when the target can only be seen through a cross-tenant leak.
  - Unsupported tenant safety in `v87` must stop the unsafe segment before mutation occurs.
