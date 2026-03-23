# Data Model: Review and Refactor Internal Service Processinstance API Implementation

## Processinstance Service Surface

- **Purpose**: Exposes process-instance-related operations to the rest of the application through one version-selected entry point.
- **Fields / Responsibilities**:
  - Supported operations exposed by the shared API
  - Version selection based on configured Camunda version
  - Stable result and error behavior across supported versions
  - Shared contracts for create, lookup, search, cancellation, deletion, waiting, and traversal
- **Relationships**:
  - Delegates to one Versioned Processinstance Behavior implementation
  - Uses Waiter Behavior for polling-based state confirmation
  - Uses Walker Behavior for ancestry and descendant traversal
  - Returns Processinstance Result, Creation Result, Cancel Result, Delete Result, or State Result values

## Versioned Processinstance Behavior

- **Purpose**: Implements processinstance operations for one supported Camunda version while adapting version-specific generated-client behavior into stable domain results.
- **Fields / Responsibilities**:
  - Camunda client dependency for create, get, search, and cancel operations
  - Operate client dependency for search, get, and delete-related operations where applicable
  - Config dependency for tenant and backoff settings
  - Logger dependency
  - Request construction, payload validation, result normalization, and version-specific behavior guards
- **Validation Rules**:
  - Must reject malformed or nil success payloads consistently
  - Must preserve current create, cancel, delete, and wait semantics
  - Must keep version-specific differences explicit where they are part of the current contract
  - If a new capability is supported in only one version, must return a defined unsupported-version error for the other supported version
- **Relationships**:
  - Consumes Generated Processinstance Capability operations from one generated client version
  - Produces Processinstance Result values and any approved additional bounded capability result

## Waiter Behavior

- **Purpose**: Polls processinstance state until a desired state set is reached or a timeout, error, or retry limit stops the wait.
- **Fields / Responsibilities**:
  - Process instance key or key set
  - Desired states
  - Backoff policy from configuration
  - Worker count and fail-fast behavior for bulk waits
- **Validation Rules**:
  - Must treat configured context deadlines and retry limits as authoritative
  - Must preserve absent-state handling for not-found responses
  - Must return clear status text for success, timeout, context cancellation, and transport failures
- **Relationships**:
  - Depends on the Processinstance Service Surface through the `PIWaiter` contract
  - Produces State Response or State Responses values

## Walker Behavior

- **Purpose**: Traverses parent-child processinstance relationships to build ancestry, descendant, or family views used by cancellation, deletion, and walk workflows.
- **Fields / Responsibilities**:
  - Start key or root key
  - Edge map keyed by parent processinstance key
  - Chain map keyed by processinstance key
  - Visited-node tracking for cycle protection
- **Validation Rules**:
  - Must detect cycles and return `services.ErrCycleDetected`
  - Must preserve orphan detection behavior when parents are missing beyond the starting node
  - Must keep traversal deterministic for the same fetched child order
- **Relationships**:
  - Depends on the Processinstance Service Surface through the `PIWalker` contract
  - Supplies family and descendant context to cancel and delete operations

## Processinstance Result

- **Purpose**: Represents the normalized domain view of a process instance returned by get, search, walk, and state-check flows.
- **Fields**:
  - Process instance key
  - BPMN process identifier
  - Process definition key and version
  - Parent key and parent flow-node key
  - Start and end timestamps
  - Tenant identifier
  - State
  - Incident flag
  - Variables map
- **Validation Rules**:
  - Missing or malformed success payloads must not silently produce false positives
  - Parent and state fields must preserve current normalization semantics across versions
- **Relationships**:
  - Used by waiter, walker, cancellation, deletion, and search workflows

## Processinstance Creation Result

- **Purpose**: Carries the externally visible result of a processinstance creation request.
- **Fields**:
  - Process instance key
  - BPMN process identifier
  - Process definition key and version
  - Tenant identifier
  - Variables
  - Start timestamp
  - Start confirmation timestamp when wait-based confirmation is enabled
- **Validation Rules**:
  - Empty or nil success payloads are treated as malformed responses
  - `StartConfirmedAt` is only present when polling confirms the started state
- **Relationships**:
  - Derived from version-specific create responses and optional waiter confirmation

## Cancellation and Deletion Outcomes

- **Purpose**: Represent the externally observable outcome of cancel and delete operations.
- **Fields**:
  - Boolean success indicator
  - HTTP status code
  - Human-readable status message
  - Associated processinstance family context for cancel flows when needed
- **Validation Rules**:
  - Must preserve current terminal-state shortcuts, dry-run behavior, force behavior, and no-wait behavior
  - Delete behavior must preserve existing recursive family handling and active-instance safeguards
- **Relationships**:
  - Depend on Processinstance Result, Waiter Behavior, and Walker Behavior

## Generated Processinstance Capability

- **Purpose**: Represents a processinstance-related generated-client operation already available in a supported client and eligible for service-layer exposure.
- **Fields / Responsibilities**:
  - Operation name
  - Version availability
  - Result shape
  - Unsupported-version behavior if support differs
  - Risk assessment for exposure
- **Validation Rules**:
  - Must fit current service boundaries without package or layout change
  - Must have a clear test strategy and stable error behavior
  - If added in only one version, must define the unsupported-version path explicitly
- **Relationships**:
  - Reviewed against the Processinstance Service Surface during implementation

## State Notes

- This feature does not add new long-lived persistence or persistent state machines.
- Relevant execution outcomes are success, transport failure, HTTP failure, malformed success payload, wait timeout, context cancellation, cycle or orphan traversal failure, unsupported version selection, and defined unsupported-version capability errors.
