# Quickstart: Harden Tenant Handling Across Tenant-Aware Commands

## Implemented Behavior

- All tenant-aware command families use one effective tenant context per execution.
- Supported tenant mismatches behave exactly like `not found`.
- `v8.8` uses tenant-safe generated search/query paths as the preferred baseline for direct-get-adjacent behavior.
- `v8.7` preserves tenant-aware behavior only where the current generated-client/upstream contract can enforce it safely; keyed direct lookup and keyed state-check remain explicitly unsupported, while search-backed flows stay usable.
- `8.9` remains acknowledged in planning and config normalization, but current repository runtime support still stops at `v8.8` for process-instance services.
- Regression coverage is added for every tenant-aware command family and for explicit flag, env, profile, and base-config tenant sources separately.

## Implementation Notes

- Start from the versioned process-instance services under `internal/services/processinstance/`, not from ad hoc command patches.
- Treat the active command surface as five audited families: keyed/search `get process-instance`, walker-backed `walk pi`, mutation-plus-preflight `cancel pi`, mutation-plus-preflight `delete pi`, and creation-plus-confirmation `run pi`.
- Audit every place that composes `GetProcessInstance`, `GetProcessInstanceStateByKey`, `SearchForProcessInstances`, `GetDirectChildrenOfProcessInstance`, walker ancestry/descendants, and waiter polling.
- Prefer upstream tenant-safe search/query calls over direct retrieval endpoints when the direct call cannot express tenant scope.
- Keep unsupported `v8.7` behavior narrow by failing the exact unsafe step instead of disabling the whole command family.
- Keep the planning honest about version support: `toolx` normalizes `8.9`, but the current process-instance factory and runtime tests still support only `v8.7` and `v8.8`.
- Preserve existing repository conventions: versioned service factories, `common.PrepareServiceDeps`, shared response helpers, command tests with explicit temp config files, and final `make test`.

## Verification Focus

1. Confirm `v8.8` direct-get-adjacent flows no longer surface cross-tenant resources when a tenant is selected.
2. Confirm walker and waiter flows inherit the same tenant-safe contract as the service methods they call.
3. Confirm wrong-tenant lookups on supported paths return the same `not found` outcome as genuinely absent resources.
4. Confirm `v8.7` unsupported outcomes are scoped to the exact unsafe operation or flow segment.
5. Confirm `v8.7` search-backed flows still work where the upstream contract is tenant-safe.
6. Confirm explicit `--tenant`, environment-derived tenant, profile-derived tenant, and base-config-derived tenant all propagate identically through tenant-aware flows.
7. Confirm command-family regressions cover `get`, `walk`, `cancel`, `delete`, and `run`, with command-specific anchors already present in `cmd/get_processinstance_test.go`, `cmd/walk_test.go`, `cmd/cancel_test.go`, `cmd/delete_test.go`, and `cmd/run_test.go`.

## Suggested Verification Commands

```bash
go test ./internal/services/processinstance/... -count=1
go test ./cmd -count=1
go test ./config -count=1
make test
```

Run the three focused suites first to isolate tenant-handling regressions by layer, then run `make test` as the repository gate required by project policy.

## Manual Smoke Ideas

Use one temp config file with a base tenant plus at least one profile override, then exercise the same key through multiple tenant sources:

```bash
./c8volt --config /tmp/c8volt.yaml --tenant tenant-a get process-instance --key 2251799813720823
C8VOLT_APP_TENANT=tenant-a ./c8volt --config /tmp/c8volt.yaml walk pi --key 2251799813720823 --tree
./c8volt --config /tmp/c8volt.yaml --profile support cancel pi --key 2251799813720823
./c8volt --config /tmp/c8volt.yaml delete pi --key 2251799813720823
```

Check that:

- matching-tenant resources still work
- wrong-tenant requests look identical to `not found` on supported paths
- unsupported `v8.7` behavior is explicit and narrowly scoped
- `v8.7` search-backed flows still succeed where the upstream tenant filter is authoritative
- no traversal, wait, cancel, or delete path crosses tenant boundaries silently
