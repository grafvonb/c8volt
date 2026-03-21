# Data Model: Review and Refactor Internal Service Resource API Implementation

## Resource Service Surface

- **Purpose**: Exposes resource-related operations to the rest of the application through one version-selected entry point.
- **Fields / Responsibilities**:
  - Supported operations exposed by the service surface
  - Version selection based on configured Camunda version
  - Stable error and result behavior across supported versions
- **Relationships**:
  - Delegates to one Versioned Resource Behavior implementation
  - Returns Deployment Result values today and may return Resource Metadata if one bounded lookup capability is added

## Versioned Resource Behavior

- **Purpose**: Implements resource operations for one supported Camunda version while adapting version-specific generated-client behavior into stable domain results.
- **Fields / Responsibilities**:
  - Resource generated client dependency for the supported version
  - Optional process-definition client dependency used for deployment confirmation
  - Logger dependency
  - Config dependency
  - Multipart request construction, response validation, and normalization rules
- **Validation Rules**:
  - Must reject nil or malformed success responses
  - Must preserve current transport, HTTP, and polling error semantics
  - Must keep version-specific behavior differences explicit where they are part of the current contract
- **Relationships**:
  - Consumes Generated Resource Capability operations from one generated client version
  - Produces Deployment Result values and, if adopted, Resource Metadata values

## Generated Resource Capability

- **Purpose**: Represents a resource-related operation already available in the supported generated clients and eligible for service-layer exposure.
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
  - Reviewed against the Resource Service Surface during implementation

## Deployment Result

- **Purpose**: Carries the externally visible result of a resource deployment request.
- **Fields**:
  - Deployment key
  - Tenant identifier
  - Deployment units
- **Validation Rules**:
  - Empty or nil success payloads are treated as malformed responses
  - Version-specific generated responses must normalize into one stable domain shape already used by callers
- **Relationships**:
  - Contains zero or more Deployment Unit records

## Deployment Unit

- **Purpose**: Represents one unit created or reported during a deployment result.
- **Fields**:
  - Process definition deployment details
- **Validation Rules**:
  - Optional nested process-definition data must normalize safely without changing existing behavior
- **Relationships**:
  - Contains one Process Definition Deployment record when the deployment unit represents a process definition

## Process Definition Deployment

- **Purpose**: Represents process-definition details surfaced from a resource deployment.
- **Fields**:
  - Process definition identifier
  - Process definition key
  - Process definition version
  - Resource name
  - Tenant identifier
- **Validation Rules**:
  - Missing fields normalize to the repository’s current zero-value behavior
- **Relationships**:
  - Embedded within a Deployment Unit

## Potential Resource Metadata

- **Purpose**: Represents the most likely bounded missing capability if the coverage review approves a resource lookup operation.
- **Fields**:
  - Resource identifier
  - Resource key
  - Resource name
- **Validation Rules**:
  - Only becomes part of the stable surface if both supported versions can return the same practical shape
  - Response validation must follow the same malformed-payload rules as the existing deploy path
- **Relationships**:
  - Derived from version-specific generated `ResourceResult` payloads

## State Notes

- This feature does not add long-lived persistence or new lifecycle states.
- The relevant execution outcomes are success, transport failure, HTTP failure, malformed success payload, poll timeout or poll failure for confirmed deployments, and unsupported version selection.
