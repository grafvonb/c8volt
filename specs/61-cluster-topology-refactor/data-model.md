# Data Model: Refactor Cluster Topology Command

## Get Cluster Parent Command

- **Purpose**: Groups cluster-related read operations under the existing `get` command tree.
- **Fields / Responsibilities**:
  - Command use string: `cluster`
  - Help text that describes cluster-related retrieval commands
  - Inherited flags and standard `get` command behavior
- **Relationships**:
  - Lives under `get`
  - Owns the `Topology Subcommand`

## Topology Subcommand

- **Purpose**: Exposes cluster topology retrieval at the preferred user-visible path `c8volt get cluster topology`.
- **Fields / Responsibilities**:
  - Command use string: `topology`
  - Help text for topology retrieval
  - Shared execution path to fetch and print topology
- **Validation Rules**:
  - Must preserve existing output shape
  - Must preserve existing error handling and exit behavior
  - Must honor inherited root and `get` flags exactly as the legacy command does
- **Relationships**:
  - Child of the `Get Cluster Parent Command`
  - Shares execution with the `Deprecated Cluster Topology Command`
  - Returns a `Topology Retrieval Outcome`

## Deprecated Cluster Topology Command

- **Purpose**: Preserves compatibility for users and scripts that still call `c8volt get cluster-topology`.
- **Fields / Responsibilities**:
  - Command use string: `cluster-topology`
  - Existing aliases that remain supported unless explicitly removed during implementation review
  - Help and documentation deprecation messaging pointing to `c8volt get cluster topology`
- **Validation Rules**:
  - Must remain executable during the deprecation period
  - Must not emit a runtime deprecation warning
  - Must route to the same topology retrieval behavior as the preferred command path
- **Relationships**:
  - Child of `get`
  - Shares execution with the `Topology Subcommand`

## Topology Retrieval Outcome

- **Purpose**: Represents the observable result of running either command path.
- **Fields**:
  - Standard output payload containing topology information
  - Error output when retrieval fails
  - Process exit status
- **Validation Rules**:
  - Preferred and deprecated command paths must produce equivalent outcomes for the same inputs
  - Any documentation examples must reflect the preferred command path and deprecation notes accurately

## State Notes

- This feature does not add persistence or new business entities.
- The relevant state transitions are limited to command discovery, successful topology retrieval, and existing failure outcomes already supported by the topology workflow.
