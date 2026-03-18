# Quickstart: Review and Refactor Cluster Service

## Goal

Implement the cluster service refactor in small, verifiable steps while preserving existing behavior and documenting any intentional low-risk capability addition.

## Prerequisites

- Go 1.25.3 available locally
- Repository dependencies installed through normal Go module workflows
- Working tree on branch `058-review-and-refactor-internal-service-cluster-api-implementation`

## Implementation Steps

1. Review `internal/services/cluster` and the supported generated cluster clients to confirm the current shared service surface and any realistic missing capability.
2. Refactor duplicated setup, response validation, and normalization logic only where the resulting code is clearer and lower risk than the current version-specific copies.
3. Preserve current factory behavior and all existing user-visible cluster topology behavior unless the coverage review supports a bounded new capability.
4. Update focused unit tests for both supported versions and factory behavior.
5. Update integration coverage if the refactor or any approved capability changes service behavior across real request paths.
6. Update `README.md` and regenerate CLI docs only if a user-visible cluster workflow changes.

## Validation Commands

```bash
go test ./internal/services/cluster/... -race -count=1
go test ./testx/... -race -count=1
make test
```

If CLI documentation changes:

```bash
make docs
```

## Completion Checklist

- Refactor stays within the existing package layout
- Supported-version cluster capability review is captured in code changes or implementation notes
- No behavioral regressions in topology fetch paths
- Added or updated tests cover preserved error and success paths
- `make test` passes
- Documentation changed only when user-visible behavior changed
