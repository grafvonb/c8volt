# Quickstart: Add Cluster License Command

## Goal

Implement `c8volt get cluster license` as a new nested cluster read command by reusing the existing cluster license service behavior, adding focused command tests, and updating user-facing documentation.

## Prerequisites

- Go 1.25.3 available locally
- Repository dependencies installed through the normal Go module workflow
- Working tree on branch `63-cluster-license`

## Implementation Steps

1. Add a `license` child command under `get cluster` using the repository's existing Cobra command style.
2. Reuse the standard CLI setup path and call `cli.GetClusterLicense(cmd.Context())`, then print the returned value with the existing JSON output helper.
3. Add or update command tests for help discovery, successful license retrieval, and failing license retrieval with subprocess-based exit assertions where needed.
4. Update `README.md` and `docs/index.md` if cluster read examples or discovery guidance should mention the new command.
5. Regenerate CLI docs with `make docs` so `docs/cli/` matches the shipped Cobra metadata.

## Validation Commands

```bash
go test ./cmd/... -race -count=1
go test ./internal/services/cluster/... -race -count=1
make docs
make test
```

## Completion Checklist

- `c8volt get cluster license` appears in `get cluster` help output
- The command prints the structured license payload for a successful response
- Failure behavior matches existing `get` command error semantics
- Targeted command tests cover help, success, and failure paths
- README and `docs/index.md` stay aligned with the new command where relevant
- Generated CLI docs include the new command page and updated parent-command listings
- `make test` passes
