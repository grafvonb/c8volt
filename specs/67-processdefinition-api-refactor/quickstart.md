# Quickstart: Review and Refactor Internal Service Processdefinition API Implementation

## Goal

Implement the processdefinition service refactor in small, verifiable steps while preserving current behavior and documenting any approved low-risk capability addition from the generated-client review.

## Prerequisites

- Go 1.25.3 available locally
- Repository dependencies installed through normal Go module workflows
- Working tree on branch `67-processdefinition-api-refactor`

## Implementation Steps

1. Review `internal/services/processdefinition` and the supported generated processdefinition clients to confirm the current shared service surface and whether XML retrieval is the right bounded capability candidate.
2. Refactor duplicated setup, response validation, and latest-result handling only where the resulting code is clearer and lower risk than the current version-specific copies.
3. Preserve current factory behavior and all existing user-visible processdefinition behavior unless the coverage review supports one bounded new capability.
4. Update focused unit tests for both supported versions and factory behavior.
5. Update README and regenerate CLI docs only if a user-visible processdefinition workflow changes.

## Validation Commands

```bash
go test ./internal/services/processdefinition/... -race -count=1
go test ./internal/services/processdefinition -race -count=1
make test
```

If CLI documentation changes:

```bash
make docs
```

## Completion Checklist

- Refactor stays within the existing package layout
- Supported-version processdefinition capability review is reflected in code changes or implementation notes
- No behavioral regressions in search, latest, and get processdefinition paths
- Existing version-specific statistics behavior remains covered and preserved
- Added or updated tests cover preserved error and success paths
- `make test` passes
- Documentation changed only when user-visible behavior changed
