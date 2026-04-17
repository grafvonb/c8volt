# Data Model: Add Camunda v8.9 Runtime Support

## Supported Camunda Version Record

- **Purpose**: Captures the repository’s declared runtime support state for a Camunda version.
- **Key attributes**:
  - Version identifier: `8.7`, `8.8`, or `8.9`
  - Normalization support
  - Advertised runtime support
  - Factory support status
  - Documentation status
- **Invariants**:
  - A version must not be advertised as supported unless factories and docs agree.
  - `v8.9` support is incomplete until all required service families and command families are covered.

### Final feature state

| Version | Normalized | Advertised runtime support | Factory support | Documentation status |
|---------|------------|----------------------------|-----------------|----------------------|
| `8.7` | Yes | Supported with existing version-specific limits | Native `v87` implementations | Synced |
| `8.8` | Yes | Supported | Native `v88` implementations | Synced |
| `8.9` | Yes | Supported with `v8.8` command-family parity | Native `v89` implementations | Synced |

## Versioned Service Family

- **Purpose**: Represents one internal service family selected by config-driven version factories.
- **Members**:
  - `Cluster`
  - `ProcessDefinition`
  - `ProcessInstance`
  - `Resource`
- **Key attributes**:
  - Shared API surface
  - Factory implementation status per supported version
  - Final generated-client boundary
  - Command families that depend on the service
- **Invariants**:
  - Each service family must preserve the existing shared `api.go` contract.
  - Final native `v8.9` behavior for a service family must come from a dedicated `v89` package, not from permanent fallback.

## Native v8.9 Service Path

- **Purpose**: Represents the final accepted runtime path for one service family when `app.camunda_version` is `v8.9`.
- **Key attributes**:
  - Owning service family
  - `v89` package path
  - Generated-client contract used
  - Factory selection path
  - User-facing command families covered
- **Invariants**:
  - Final native `v8.9` paths must depend only on `internal/clients/camunda/v89/camunda/client.gen.go`.
  - Final native `v8.9` paths must preserve the existing user-facing contract of the corresponding `v8.8` command family.

## Temporary Fallback Path

- **Purpose**: Represents any documented transition-only path that preserves command behavior while native `v8.9` work is still incomplete.
- **Key attributes**:
  - Owning service family or command family
  - Reason fallback exists
  - Older-version behavior being reused
  - Planned removal condition
  - Documentation requirement
- **Invariants**:
  - Fallback may be used only as a bridge during implementation.
  - Fallback must be documented and must not remain in the final accepted native path for a service family that already follows the versioned-service architecture.

### Final feature state

- No active `v8.9` temporary fallback path remains in the repository's versioned service families.
- The fallback entity remains part of the model only as a guardrail for future incomplete version rollouts.

## Command Family Parity Record

- **Purpose**: Tracks one repository command family that must behave the same on `v8.9` as it does on `v8.8`.
- **Families in scope**:
  - Cluster metadata
  - Process-definition discovery
  - Resource lifecycle
  - Process-instance lifecycle and traversal
- **Key attributes**:
  - Backing service families
  - Minimum `v8.9` execution-test anchor
  - Whether docs/examples mention the command family
  - Whether fallback is still present
- **Invariants**:
  - Each repository command family already supported on `v8.8` must have at least one explicit `v8.9` execution test.
  - Command-family parity is incomplete if the family still depends on undocumented fallback.

### Final feature state

| Command family | Backing services | `v8.9` status |
|----------------|------------------|----------------|
| Cluster metadata | `cluster` | Native `v89` path implemented and covered |
| Process-definition discovery | `processdefinition` | Native `v89` path implemented and covered |
| Resource lifecycle | `resource`, `processdefinition` | Native `v89` path implemented and covered |
| Process-instance lifecycle and traversal | `processinstance` | Native `v89` path implemented and covered |

## Generated Client Boundary

- **Purpose**: Defines which generated client package a final native runtime path is allowed to use.
- **Key attributes**:
  - Version (`v89`)
  - Client package path
  - Service families bound to that client
  - Transitional exceptions, if any
- **Final boundary**:
  - `internal/clients/camunda/v89/camunda/client.gen.go`
- **Invariants**:
  - Final native `v8.9` paths must not mix `v87` or `v88` generated clients.
  - Final native `v8.9` paths must not rely on `v89` Operate, Tasklist, or Administration SM clients unless the design contract changes explicitly.

### Final feature state

- The accepted `v8.9` runtime path for `cluster`, `processdefinition`, `processinstance`, and `resource` stays entirely on `internal/clients/camunda/v89/camunda/client.gen.go`.

## Documentation Surface

- **Purpose**: Represents a user-visible source that communicates supported versions and command behavior.
- **Members**:
  - `README.md`
  - `docs/index.md`
  - Generated `docs/cli/*`
  - Root command help text
- **Key attributes**:
  - Current version-support statement
  - Whether it is generated or source-authored
  - Regeneration path
- **Invariants**:
  - Release readiness is blocked until all version-support surfaces describe `v8.9` support consistently.
  - Generated docs must be refreshed from the real source of truth rather than hand-edited.

### Final feature state

- `README.md`, `docs/index.md`, generated `docs/cli/*`, and root help now agree that `8.9` is supported with the same repository command-family scope already covered on `8.8`.

## Verification Slice

- **Purpose**: Represents one independently verifiable proof slice for the feature.
- **Slice types**:
  - Factory selection
  - Service behavior
  - Command execution
  - Documentation truthfulness
- **Key attributes**:
  - Owning layer
  - Validation command or test target
  - Supported-version expectation
  - Preserved older-version expectation
- **Invariants**:
  - Every service family needs factory coverage.
  - Every repository command family needs at least one explicit `v8.9` execution proof slice.
  - The repository gate remains `make test`.
