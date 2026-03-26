# Data Model: Add Resource Get Command By Id

## Resource Lookup Request

- **Purpose**: Represents one operator-triggered CLI request to fetch a single deployed resource by identifier.
- **Fields**:
  - `id`: required resource identifier supplied through `--id`
  - inherited CLI call options from root and `get` command flags
- **Validation Rules**:
  - `id` must be provided through `--id`
  - empty or whitespace-only values are rejected before lookup
  - the request remains single-item only; no list, batch, or raw-content mode is part of this feature
- **Relationships**:
  - Consumed by the new `get resource` Cobra command
  - Routed through the `c8volt/resource` facade to the internal resource service

## Resource Output

- **Purpose**: The public single-resource object/details representation returned to CLI rendering on successful lookup.
- **Fields**:
  - `id`
  - `key`
  - `name`
  - `tenantId`
  - `version`
  - `versionTag`
- **Validation Rules**:
  - maps from the existing internal domain `Resource` shape
  - successful lookup must produce this normal object/details form rather than raw resource content
  - zero-value fields follow the repository’s existing normalization rules
- **Relationships**:
  - Produced by the `c8volt/resource` facade
  - Rendered by a `cmd` view helper consistent with other single-item get commands

## Resource Lookup Failure

- **Purpose**: Captures the user-visible failure outcomes for single-resource lookup.
- **Variants**:
  - validation error for missing or invalid `--id`
  - transport or client error while calling the backend
  - not-found response from the backend
  - malformed-response error when the backend reports success but omits the expected payload
  - unsupported-version selection surfaced through existing factory/service behavior
- **Validation Rules**:
  - must preserve normal CLI non-success exit semantics
  - malformed-response handling must remain distinct from not-found handling
- **Relationships**:
  - Originates in command validation, facade mapping, or versioned service calls

## Resource Facade Model

- **Purpose**: Public `c8volt/resource` model that exposes resource lookup data without leaking internal domain packages into command code.
- **Fields**:
  - mirrors the fields of `internal/domain.Resource` needed for output
- **Validation Rules**:
  - should stay minimal and compatible with current resource package patterns
  - should not introduce raw-content fields in this feature
- **Relationships**:
  - mapped from `internal/domain.Resource`
  - returned by a new method on `c8volt/resource.API`
  - included in the aggregate `c8volt.API`

## State Notes

- This feature adds no persistence and no long-lived lifecycle state.
- The relevant execution outcomes are validation failure, lookup success, backend not found, transport failure, malformed success payload, and unsupported-version service selection failure.
