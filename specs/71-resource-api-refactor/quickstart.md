# Quickstart: Review and Refactor Internal Service Resource API Implementation

## Goal

Implement the resource service refactor in small, verifiable steps while preserving current deploy and delete behavior and documenting any intentional low-risk capability addition.

## Prerequisites

- Go 1.25.3 available locally
- Repository dependencies installed through normal Go module workflows
- Working tree on branch `71-resource-api-refactor`

## Implementation Steps

1. Review `internal/services/resource` and the supported generated resource clients to confirm the current shared service surface and the most realistic missing capability candidate.
2. Refactor duplicated setup, multipart request creation, payload validation, and delete-path handling only where the resulting code is clearer and lower risk than the current version-specific copies.
3. Preserve current factory behavior and all existing user-visible resource behavior, including the v88 deployment confirmation poll and the current consistent-delete behavior.
4. Add one bounded missing capability only if it exists across both supported versions and fits the current API shape without broadening the feature unnecessarily.
5. Update focused unit tests for both supported versions and factory behavior, adding missing v87 service coverage as needed.
6. Update `README.md` and regenerate CLI docs only if a user-visible resource workflow changes.

## Validation Commands

```bash
go test ./internal/services/resource/... -race -count=1
make test
```

## Documentation Impact

The completed refactor remains internal-only:

- The final capability addition was `resource.Get` in the internal service layer.
- No command names, flags, CLI output, or operator workflow changed.
- `README.md` and generated docs under `docs/cli/` therefore remain unchanged for this feature.

If a future iteration introduces a user-visible CLI workflow change:

```bash
make docs
```

## Completion Checklist

- Refactor stays within the existing package layout
- Supported-version generated resource capability review is captured in code changes or implementation notes
- No behavioral regressions in deploy, delete, or wait-for-confirmation paths
- Added or updated tests cover preserved success, malformed-response, and error paths
- Targeted regression proof runs `go test ./internal/services/resource/... -race -count=1` before the repository-wide `make test`
- `make test` passes
- Documentation remains unchanged here because the final `Get` addition stayed internal-only
