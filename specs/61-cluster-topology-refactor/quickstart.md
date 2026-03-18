# Quickstart: Refactor Cluster Topology Command

## Goal

Implement the nested `c8volt get cluster topology` command hierarchy while preserving current topology behavior and keeping `c8volt get cluster-topology` working as a documented deprecated path.

## Prerequisites

- Go 1.25.3 available locally
- Repository dependencies installed through the normal Go module workflow
- Working tree on branch `61-cluster-topology-refactor`

## Implementation Steps

1. Add a `cluster` parent command under `get` using the repository's existing Cobra command style.
2. Move or rebuild the topology command entry so the preferred path becomes `c8volt get cluster topology` without changing the underlying execution logic.
3. Keep `c8volt get cluster-topology` available as a compatibility command or alias that reaches the same behavior and is marked deprecated only in help/docs.
4. Add or update command tests to prove both command paths exist, preserve inherited behavior, and keep help output aligned with the migration plan.
5. Regenerate CLI docs and update README examples only where cluster-topology usage is documented.

## Validation Commands

```bash
go test ./cmd/... -race -count=1
go test ./internal/services/cluster/... -race -count=1
make test
```

If CLI docs changed:

```bash
make docs
```

## Completion Checklist

- `c8volt get cluster topology` is available in help output
- `c8volt get cluster-topology` still works without runtime deprecation output
- Both command paths reach the same topology retrieval behavior
- Targeted command tests cover the new hierarchy and compatibility path
- Generated CLI docs reflect the preferred and deprecated paths correctly
- `make test` passes
