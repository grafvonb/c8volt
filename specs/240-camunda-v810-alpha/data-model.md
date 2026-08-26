# Data Model: Experimental Camunda 8.10 Alpha Support

## Runtime Support Record

Represents one selectable Camunda runtime identity.

| Field | Description | Validation |
|-------|-------------|------------|
| Canonical identifier | Stable internal and displayed identity | `8.10-alpha` for this feature |
| Accepted aliases | Inputs normalized to the canonical identity | Every alpha alias must contain `alpha` |
| Stability class | Stable or experimental | `experimental` |
| Default | Whether omission selects this runtime | Must remain `false` |
| Baseline | Exact upstream release validated for this identity | `8.10.0-alpha4` |
| Fixture family | Embedded fixture prefix used by this runtime | Explicitly `C89_` |

### State transitions

```text
unknown input -> rejected
explicit alpha alias -> normalized experimental target -> configured runtime
missing input -> existing V88 default
experimental target -> stable 8.10 target   # future issue only
```

## Generated Client Artifact

Represents the checked-in unified client used only by alpha adapters.

| Field | Value or rule |
|-------|---------------|
| Runtime target | `v810alpha` |
| Package role | Unified Camunda product v2 client |
| Source baseline | `8.10.0-alpha4` |
| Ownership | Alpha adapters only |
| Publication | Generated to a temporary file, then published after guards pass |

### Invariants

- The artifact must not be hand-edited.
- It must not import or overwrite another Camunda version's artifact.
- Regeneration from recorded inputs must be deterministic apart from approved generator formatting changes.

## Provenance Record

Machine-readable metadata stored beside the generated client.

| Field | Required value |
|-------|----------------|
| `repository` | `https://github.com/camunda/camunda.git` |
| `tag` | `8.10.0-alpha4` |
| `commit` | `4a76f06c9df8ad0a6c64fe88b92c618c2ebf2ef6` |
| `sourceSpec` | `zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml` |
| `mutations` | Ordered list of all applied mutation scripts |
| `generator` | Generator name and exact version |
| `command` | Canonical repository-relative reproduction command |

### Invariants

- All seven fields are non-empty.
- Tag and commit must resolve to the same upstream object.
- The output destination implied by the command must be the alpha client only.

## Service Compatibility Record

Represents the result of evaluating one version-aware service family against the alpha contract.

| Field | Description |
|-------|-------------|
| Family | One of the eleven current factory-owned service areas |
| Shared contract | Version-neutral service behavior required by facades |
| Alpha adapter | Dedicated version-specific implementation |
| Client boundary | Alpha unified generated client only |
| Compatibility outcome | Supported, version-neutral reuse, or unsupported capability |
| Special handling | Type conversion, eventual consistency, or removed legacy fallback |
| Verification anchor | Factory and adapter tests for the family |

### Invariants

- Every version-aware factory has exactly one explicit alpha outcome.
- No alpha adapter imports a v8.7, v8.8, or v8.9 generated client.
- Unsupported mutations fail before an external mutation is attempted.

## Capability Record

Represents a named operation-level compatibility decision.

| Field | Description |
|-------|-------------|
| Name | Stable semantic capability name |
| Supported runtimes | Explicit runtime identity set |
| Unsupported error | Existing domain unsupported classification and operator wording |
| Call sites | Commands or workflows guarded by this decision |

The first required record is history-safe process-definition deletion, supported by `V89` and `V810Alpha` and consumed by direct deletion plus all-process-definitions purge.

## Documentation Support Record

Represents one user-facing support claim.

| Field | Description |
|-------|-------------|
| Stable versions | `8.7`, `8.8`, `8.9` |
| Experimental versions | `8.10-alpha` |
| Experimental baseline | `8.10.0-alpha4` |
| Default runtime | Existing `8.8` default |
| Source | Authored command metadata, README, or generation guide |
| Generated derivatives | CLI reference and documentation index |

### Invariants

- Stable and experimental versions are never presented as one undifferentiated stability claim.
- Machine-readable additions are additive and preserve existing envelope and field types.
- Generated documentation is refreshed from authored sources.

## Smoke Evidence Record

Represents one execution of the pinned-alpha live validation.

| Field | Description |
|-------|-------------|
| Baseline observed | Gateway/runtime version returned by the environment |
| Authentication | Connection/authentication result |
| Discovery | Topology result |
| Read workflow | Command and observed result |
| Mutation workflow | Submitted mutation and confirmation result |
| Unsupported probe | Rejected target or unsupported capability and failure classification |
| Timestamp and environment | Reproduction context without secrets |

The record is complete only when all five behavioral checks have explicit pass/fail outcomes. A missing environment is reported as not run and does not satisfy release readiness.
