# Quickstart: Add Camunda v8.9 Runtime Support

## Implemented Behavior

- `app.camunda_version: v8.9` becomes a fully supported runtime option rather than a normalization-only value.
- Repository command families already supported on `v8.8` remain available on `v8.9`.
- `cluster`, `processdefinition`, `processinstance`, and `resource` factories select native `v89` implementations.
- Final native `v8.9` paths use only the generated `internal/clients/camunda/v89/camunda/client.gen.go` client surface.
- Existing `v8.7` and `v8.8` behavior remains unchanged.
- README and generated CLI documentation reflect `v8.9` support before release readiness is claimed.

## Implementation Notes

- Start at the version source of truth: update `toolx.SupportedCamundaVersions()` and the root/help surfaces that still claim support stops at `v8.8`.
- During the foundational phase, keep runtime truth separate from advertised support: `toolx` can list `v8.9` while factory error messages continue to use implemented-version helpers until native constructors are wired.
- Mirror the existing `v88` structure when creating `v89` packages under `internal/services/{cluster,processdefinition,processinstance,resource}/v89`.
- Reuse `common.PrepareServiceDeps` and `common.EnsureLoggerAndClients` in new constructors so config, HTTP client, and logger behavior stays consistent.
- Keep version selection in the existing factories and `c8volt/client.go`; avoid adding command-local version branches.
- Treat `processinstance` as the most sensitive service family because its current `v88` implementation still mixes Camunda and Operate generated clients.
- Use the generated `v89` Camunda client endpoints for process-instance get/search/cancel/delete/state-oriented flows so the final native path can satisfy the strict client-boundary rule.
- Treat any temporary fallback as documented transition-only behavior, not as a final acceptance state.
- Update command help text first, then regenerate docs rather than hand-editing generated CLI reference files.

## Verification Focus

1. Confirm `toolx` and root/bootstrap surfaces advertise `v8.9` as supported once the runtime implementation exists.
2. Confirm the foundational checkpoint creates compile-safe `v89` service packages and shared API assertions before any factory starts selecting them.
3. Confirm each versioned service factory selects a native `v89` implementation when `v8.9` is configured.
4. Confirm native `v89` cluster, process-definition, resource, and process-instance paths depend only on the generated `v89` Camunda client.
5. Confirm `v8.7` and `v8.8` behavior remains unchanged.
6. Confirm at least one explicit `v8.9` command execution path succeeds for each repository command family.
7. Confirm README and generated docs no longer claim runtime support stops at `v8.8`.

## Foundational Checkpoint

- `internal/services/{cluster,processdefinition,processinstance,resource}/v89` exists with constructor scaffolds and version-local generated-client contracts.
- Shared `internal/services/*/api.go` files recognize the `v89` API surface without yet asserting runtime selection.
- Factory errors still report implemented versions until User Story 1 wires real `v89` constructors into selection.
- Release readiness remains blocked until command parity, docs, and `make test` all pass.

## Suggested Verification Commands

```bash
go test ./internal/services/cluster/... -count=1
go test ./internal/services/processdefinition/... -count=1
go test ./internal/services/processinstance/... -count=1
go test ./internal/services/resource/... -count=1
go test ./cmd -count=1
make docs
make docs-content
make test
```

Run the focused service and command suites first to isolate factory/runtime regressions, regenerate documentation after help or README updates, then run `make test` as the repository gate.

## Manual Smoke Ideas

Use a temp config that sets `app.camunda_version: v8.9`, then verify one command path from each repository family:

```bash
./c8volt --config /tmp/c8volt-v89.yaml get cluster topology
./c8volt --config /tmp/c8volt-v89.yaml get process-definition --latest
./c8volt --config /tmp/c8volt-v89.yaml get resource --id resource-id-123
./c8volt --config /tmp/c8volt-v89.yaml run process-instance --bpmn-process-id order-process
./c8volt --config /tmp/c8volt-v89.yaml get process-instance --key 2251799813685255
```

Check that:

- the command succeeds on `v8.9` with the same user-facing contract currently expected on `v8.8`
- no command family still depends on undocumented fallback behavior
- supported-version messaging in help and docs matches the runtime truth
