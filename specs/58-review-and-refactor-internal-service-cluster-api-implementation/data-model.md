# Data Model: Review and Refactor Cluster Service

## Cluster Service Surface

- **Purpose**: Exposes cluster-related operations to the rest of the application through one version-selected entry point.
- **Fields / Responsibilities**:
  - Supported operations exposed by the service surface
  - Version selection based on configured Camunda version
  - Stable error and result behavior across supported versions
- **Relationships**:
  - Delegates to one Versioned Cluster Behavior implementation
  - Returns Cluster Topology Result values to callers

## Versioned Cluster Behavior

- **Purpose**: Implements cluster operations for one supported Camunda version while adapting version-specific generated-client behavior into stable domain results.
- **Fields / Responsibilities**:
  - Client dependency for the supported version
  - Logger dependency
  - Config dependency
  - Response validation and normalization rules
- **Validation Rules**:
  - Must reject nil or malformed success responses
  - Must preserve current error mapping semantics for transport and HTTP failures
  - Must keep response translation behavior consistent with the domain model
- **Relationships**:
  - Consumes Supported Cluster Capability operations from one generated client version
  - Produces Cluster Topology Result values and, if adopted, any additional approved cluster result type

## Supported Cluster Capability

- **Purpose**: Represents a cluster-related operation already available in the supported generated clients and eligible for service-layer exposure.
- **Fields / Responsibilities**:
  - Operation name
  - Availability across supported versions
  - Expected result shape
  - Risk assessment for exposure
- **Validation Rules**:
  - Must exist in every supported version before it can become part of the stable shared service surface
  - Must fit current service boundaries without package or layout change
  - Must have a clear testing path and preserved error behavior
- **Relationships**:
  - Reviewed against the Cluster Service Surface during implementation

## Cluster Topology Result

- **Purpose**: Carries cluster metadata returned by topology retrieval.
- **Fields**:
  - Broker list
  - Cluster size
  - Gateway version
  - Partitions count
  - Replication factor
- **Validation Rules**:
  - Empty or nil success payloads are treated as malformed responses
  - Version-specific generated responses must normalize into one stable domain shape
- **Relationships**:
  - Contains one or more Broker records

## Broker

- **Purpose**: Represents one broker in the cluster topology result.
- **Fields**:
  - Host
  - Node identifier
  - Port
  - Version
  - Partition list
- **Validation Rules**:
  - Missing optional fields normalize safely without changing current behavior
- **Relationships**:
  - Contains zero or more Partition records

## Partition

- **Purpose**: Represents one partition assigned to a broker.
- **Fields**:
  - Partition identifier
  - Role
  - Health
- **Validation Rules**:
  - Version-specific values normalize into stable domain enums or strings already used by the domain model

## State Notes

- This feature does not add long-lived persistence or new lifecycle states.
- The only relevant transitions are request execution outcomes: success, transport failure, HTTP failure, malformed success payload, and unsupported version selection.
