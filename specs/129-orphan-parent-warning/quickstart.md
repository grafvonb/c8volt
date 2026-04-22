# Quickstart: Graceful Orphan Parent Traversal

## Planned Behavior

- `walk pi --parent`, `walk pi --family`, and `walk pi --family --tree` return partial traversal data plus a warning when a non-start ancestor is missing and at least one actionable process instance was resolved.
- Cancel/delete preflight and indirect cleanup expansion continue with resolved family keys while also surfacing machine-readable missing ancestor keys.
- Fully unresolved traversal remains a normal failure.
- Direct `get process-instance --key ...` remains strict.
- Wait-for-absent / wait-for-deleted behavior remains unchanged.
- The same partial-result contract applies across `v8.7`, `v8.8`, and `v8.9` because all three versioned services delegate traversal through the shared walker.

## Implementation Notes

- Start in `internal/services/processinstance/walker/walker.go` and `walker_test.go`; that is the source of the current hard-failure behavior and the best place to introduce the new structured traversal result.
- Keep facade integration in `c8volt/process/dryrun.go` thin: it should consume `TraversalResult` values and expose actionable roots/collected keys plus missing ancestors through `DryRunPIKeyExpansion`, not invent new traversal rules.
- Keep command updates focused on rendering and user-facing warnings in `cmd/walk_processinstance.go`, `cmd/cancel_processinstance.go`, and `cmd/delete_processinstance.go`.
- Preserve direct key/state/wait strictness by not widening the feature into `GetProcessInstance`, `GetProcessInstanceStateByKey`, or waiter absent/deleted success semantics.
- If operator-visible command output changes, update `README.md` and regenerate `docs/cli/` with `make docs-content`.

## Verification Focus

1. Confirm the shared walker returns partial chain/edge data plus missing ancestor metadata when a non-start parent is missing.
2. Confirm fully unresolved traversal still fails.
3. Confirm `walk` renders partial list/tree output instead of failing hard.
4. Confirm cancel/delete preflight continues when orphan children are still actionable.
5. Confirm direct `get process-instance --key` stays strict.
6. Confirm waiter absent/deleted behavior remains unchanged.
7. Confirm regression coverage exercises `v87`, `v88`, and `v89` traversal behavior.

## Completed Verification Commands

```bash
go test ./internal/services/processinstance/... -count=1
go test ./c8volt/process -count=1
go test ./cmd -count=1
make test
```

All four commands passed on 2026-04-22. `make docs-content` also ran afterward to refresh `docs/cli/` for the updated operator-facing help text.

## Manual Smoke Ideas

Use a key whose recorded parent no longer exists and compare read-only traversal with strict direct lookup:

```bash
./c8volt walk pi --key 2251799813704187 --parent
./c8volt walk pi --key 2251799813704187 --family --tree
./c8volt cancel pi --key 2251799813704187 --force --auto-confirm --no-wait
./c8volt delete pi --key 2251799813704187 --auto-confirm --no-wait
./c8volt get pi --key 2251799813704187
```

Check that:

- `walk` renders partial data and warns about missing ancestors
- cancel/delete preflight stays actionable when some keys resolve
- a fully unresolved traversal still fails
- direct `get pi --key` still returns the normal strict error when the target is missing
- there is no change to absent/deleted waiter confirmation behavior
