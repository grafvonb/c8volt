# Data Model: Add Cluster License Command

## Cluster License Subcommand

- **Purpose**: Exposes cluster license retrieval at the user-visible path `c8volt get cluster license`.
- **Fields / Responsibilities**:
  - Command use string: `license`
  - Help text that describes cluster license retrieval
  - Shared command execution pattern used by neighboring `get cluster` commands
- **Validation Rules**:
  - Must be reachable under `get cluster`
  - Must preserve inherited root and `get` flags
  - Must be the only new user-visible command path introduced for this feature
- **Relationships**:
  - Child of the `Get Cluster Parent Command`
  - Produces a `Cluster License Result`

## Get Cluster Parent Command

- **Purpose**: Groups cluster-related read operations under `get`.
- **Fields / Responsibilities**:
  - Command use string: `cluster`
  - Help text listing supported cluster read operations
  - Standard inherited flag handling from `get` and root commands
- **Relationships**:
  - Child of `get`
  - Parent of `Topology Subcommand`
  - Parent of `Cluster License Subcommand`

## Cluster License Result

- **Purpose**: Represents the structured license information printed by the new command.
- **Fields**:
  - `LicenseType`
  - `ValidLicense`
  - `ExpiresAt` when provided by the connected Camunda version
  - `IsCommercial` when provided by the connected Camunda version
- **Validation Rules**:
  - Required fields must remain aligned with the existing domain model returned by the cluster service
  - Optional fields may be absent and must not be invented by the command
  - Successful output must remain compatible with the CLI's standard structured JSON printing behavior
- **Relationships**:
  - Returned by the internal cluster service
  - Printed by the `Cluster License Subcommand`

## License Retrieval Failure

- **Purpose**: Represents the observable failure outcome when license retrieval does not succeed.
- **Fields**:
  - Error output routed through the existing CLI error handler
  - Process exit status derived from current `ferrors.HandleAndExit` behavior
- **Validation Rules**:
  - Must preserve current error semantics for transport failures, non-success HTTP responses, and malformed responses
  - Must remain consistent with other `get` command failure behavior
- **Relationships**:
  - Produced by the `Cluster License Subcommand`
  - Derived from existing cluster service and CLI error-handling paths

## State Notes

- This feature adds no new persistence, workflow state machine, or stored entities.
- Relevant state transitions are limited to command discovery, successful cluster license retrieval, and existing failure outcomes already supported by the cluster service.
