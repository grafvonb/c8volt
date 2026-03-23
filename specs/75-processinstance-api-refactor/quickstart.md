# Quickstart: Review and Refactor Internal Service Processinstance API Implementation

## Goal

Implement the processinstance service refactor in small, verifiable steps while preserving current create, search, wait, walk, cancel, and delete behavior and documenting any intentional low-risk capability addition.

## Prerequisites

- Go 1.25.3 available locally
- Repository dependencies installed through normal Go module workflows
- Working tree on branch `75-processinstance-api-refactor`

## Implementation Steps

1. Review `internal/services/processinstance` and the supported generated clients to confirm the current shared service surface and the most realistic missing capability candidate.
2. Refactor duplicated setup, response validation, and service control flow only where the resulting code is clearer and lower risk than the current version-specific copies.
3. Preserve current factory behavior and all existing user-visible processinstance behavior, including start confirmation, wait-for-state polling, force-cancel traversal, recursive delete handling, and family-walk semantics.
4. Keep waiter and walker semantics intact even if call sites or local helper structure become simpler.
5. Add one bounded missing capability only if it fits the current API shape; if only one supported version can expose it, define and test the unsupported-version behavior explicitly.
6. Update focused unit tests for both supported versions and factory behavior, and add helper-level waiter or walker tests when those paths are touched.
7. Update `README.md` and regenerate CLI docs only if a user-visible processinstance workflow or output changes.

## Final Validation Sequence

```bash
go test ./internal/services/processinstance/... -race -count=1
make test
```

Run the targeted processinstance regression suite first so versioned service, waiter, and walker failures stay isolated and actionable before the repository-wide suite.

## Documentation Impact Decision

The completed refactor remains internal-only:

- No command names, flags, or CLI output changes are required by default.
- `README.md` and generated docs under `docs/cli/` remain unchanged because the final implementation did not expose a new user-visible processinstance capability or alter a documented workflow.

If a future iteration or final implementation introduces a user-visible CLI change:

```bash
make docs
```

## Completion Checklist

- Refactor stays within the existing package layout
- Supported-version generated processinstance capability review is captured in code changes or implementation notes
- Waiter and walker behavior remain compatible with current callers
- No behavioral regressions in create, get, search, cancel, delete, wait, or traversal flows
- Added or updated tests cover preserved success, malformed-response, wait-state edge cases, helper invariants, and error paths
- Targeted regression proof runs `go test ./internal/services/processinstance/... -race -count=1` before the repository-wide `make test`
- `make test` passes
- Documentation remains unchanged unless implementation introduces a user-visible workflow change
