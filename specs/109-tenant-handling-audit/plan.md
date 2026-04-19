# Implementation Plan: Harden Tenant Handling Across Tenant-Aware Commands

**Branch**: `109-tenant-handling-audit` | **Date**: 2026-04-17 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/spec.md)
**Input**: Feature specification from `/specs/109-tenant-handling-audit/spec.md`

## Summary

Audit and harden every tenant-aware command family so tenant scope is enforced consistently across direct get, search, walk, ancestry, descendants, wait, cancel, and delete flows. The design keeps tenant correctness in the versioned process-instance service layer and shared walker/waiter helpers, prefers upstream tenant-safe generated client paths over local post-filtering, defines supported-path tenant mismatch as a tenant-safe `not found` outcome, scopes unsupported `v8.7` behavior to the exact unsafe operation, and requires new regression coverage across all tenant-aware command families for explicit flag, env, profile, and base-config tenant sources.

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...`, existing helpers in `internal/services/common`, process-instance walker/waiter helpers under `internal/services/processinstance/`  
**Storage**: File-based YAML config plus environment variables; no persistent datastore changes  
**Testing**: `go test`, `make test`, command regression tests under `cmd/`, versioned service tests under `internal/services/processinstance/v87` and `v88`, walker/waiter tests under `internal/services/processinstance/`, config precedence tests under `config/` for derived tenant sources  
**Target Platform**: Cross-platform CLI for local and CI use against supported Camunda 8.7 and 8.8 environments, with explicit planning note that `toolx` normalizes `8.9` but the current process-instance factory does not yet expose a `v89` implementation  
**Project Type**: CLI  
**Performance Goals**: No user-visible regression in command startup or existing paging/wait loops, no extra cross-tenant fetch-and-filter passes, deterministic tenant-safe `not found` behavior for supported mismatches, and unchanged command throughput for normal in-tenant flows  
**Constraints**: Preserve existing Cobra command surfaces and shared `cmd` bootstrap error handling, reuse repository-native versioned service patterns, avoid client-side post-filtering of cross-tenant results, keep unsupported `v8.7` outcomes scoped to the exact unsafe operation, audit all tenant-aware command families, add regression coverage for flag/env/profile/base-config tenant sources, update user-facing tenant docs if behavior changes, and finish with `make test`  
**Scale/Scope**: Root tenant resolution in [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go) and `config/`, process-instance command families under [`cmd/get_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go), [`cmd/walk.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk.go), [`cmd/cancel.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel.go), [`cmd/delete.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete.go), [`cmd/run.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run.go), and related commands that call the process facade, the shared facade under `c8volt/process/`, versioned services under `internal/services/processinstance/v87` and `v88`, walker/waiter helpers, and the relevant operator docs in [`README.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md) plus generated `docs/cli/`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. The feature is specifically about preventing false tenant-safe success, so the plan requires end-to-end tenant enforcement in the actual service and helper flows rather than intent-only flag resolution.
- **CLI-First, Script-Safe Interfaces**: Pass. Existing commands and flags stay in place, and supported mismatches resolve to a deterministic tenant-safe `not found` contract while unsupported `v8.7` segments return clear CLI-safe failures.
- **Tests and Validation Are Mandatory**: Pass. The plan requires new regression coverage across all tenant-aware command families, versioned service tests, derived-tenant-source coverage, and final `make test`.
- **Documentation Matches User Behavior**: Pass. Any user-visible tenant behavior changes will be reflected in the relevant tenant guidance in `README.md` and regenerated CLI docs.
- **Small, Compatible, Repository-Native Changes**: Pass. The design reuses the existing process-instance factory, versioned service layout, shared walker/waiter helpers, and `internal/services/common` utilities rather than introducing a new tenant-resolution subsystem.

## Project Structure

### Documentation (this feature)

```text
specs/109-tenant-handling-audit/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── tenant-handling.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── root.go
├── get_processinstance.go
├── walk.go
├── cancel.go
├── delete.go
├── run.go
├── expect.go
├── *_test.go
└── helpers used by tenant-aware command families

config/
├── app.go
├── config.go
└── *_test.go

c8volt/process/
├── api.go
├── client.go
├── convert.go
└── *_test.go

internal/services/common/
├── deps.go
├── response.go
└── filter.go

internal/services/processinstance/
├── api.go
├── factory.go
├── factory_test.go
├── v87/
│   ├── contract.go
│   ├── convert.go
│   ├── bulk.go
│   ├── service.go
│   └── service_test.go
├── v88/
│   ├── contract.go
│   ├── convert.go
│   ├── bulk.go
│   ├── service.go
│   └── service_test.go
├── walker/
│   ├── walker.go
│   └── walker_test.go
└── waiter/
    ├── waiter.go
    └── waiter_test.go

README.md
docs/cli/
```

**Structure Decision**: Keep the feature centered on the existing process-instance service abstraction. Tenant selection continues to originate in shared config/bootstrap, but tenant correctness is enforced in the versioned `internal/services/processinstance` implementations and the shared walker/waiter flows that compose them. Command families under `cmd/` should remain thin consumers of the facade instead of introducing command-specific tenant filtering or version branching.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/research.md).

- Confirm which generated-client calls are already tenant-safe in `v88`, especially `SearchProcessInstancesWithResponse`, and which direct-get paths still bypass tenant filters.
- Confirm the current `v87` split between Operate direct-get and search behavior and whether any direct key-based tenant-safe retrieval can be implemented without local post-filtering.
- Confirm how walker, waiter, cancel, and delete flows currently compose `GetProcessInstance`, `GetProcessInstanceStateByKey`, `GetDirectChildrenOfProcessInstance`, and family traversal so unsupported behavior can be scoped to exact operations instead of entire command families.
- Confirm the current support boundary for `8.9`: the version parser accepts it, but the process-instance factory currently supports only `8.7` and `8.8`, which must be reflected honestly in planning and tasking.
- Confirm the existing test seams for tenant-aware command families and where explicit flag, env, profile, and base-config tenant coverage should live.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/quickstart.md)
- [contracts/tenant-handling.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/109-tenant-handling-audit/contracts/tenant-handling.md)

- Keep the tenant contract in the versioned process-instance services: supported mismatches must behave like `not found`, unsupported `v8.7` segments must fail explicitly, and no command should patch correctness with local cross-tenant filtering.
- Make `v88` the preferred tenant-safe baseline by routing direct-get-adjacent behavior through tenant-safe generated search/query paths when the direct retrieval endpoint cannot enforce tenant scope.
- Keep unsupported `v8.7` behavior as narrow as possible by mapping unsupported outcomes to the specific direct-get, state-check, wait, or traversal segment that lacks an equivalent tenant-safe upstream path.
- Audit `8.9` as a planning concern only: current repo support stops at `v88`, so planning must either treat `8.9` as future follow-up work or as a no-code audit note until a `v89` service exists.
- Add regression coverage across every tenant-aware command family, with explicit cases for `--tenant`, environment-derived tenant, profile-derived tenant, and base-config-derived tenant selection.
- Update agent context and user-facing tenant docs so future implementation work stays aligned with the clarified contract.

### Authoritative Boundary Matrix

| Operation segment | `v8.8` plan | `v8.7` plan | `v8.9` plan |
|------------------|-------------|-------------|-------------|
| Direct get and state-by-key | Replace unscoped direct-key authority with tenant-safe lookup semantics before treating these segments as supported | Keep supported only if an equivalent tenant-safe upstream path exists; otherwise mark the exact segment unsupported | Audit only; no runtime implementation in current repo |
| Search and direct-children search | Keep as the primary tenant-safe baseline with effective tenant injected into upstream filters | Keep supported where the generated search path can enforce tenant scope | Audit only; no runtime implementation in current repo |
| Ancestry / descendants / family / wait | Compose only through tenant-safe lookup and state seams; mismatch stays `not found` | Allow only the segments whose dependencies are tenant-safe; fail the exact unsafe segment otherwise | Audit only; no runtime implementation in current repo |
| Cancel / delete preflight and follow-up | Require tenant-safe validation before mutation or wait-dependent confirmation | Require tenant-safe validation where available; otherwise report explicit unsupported outcome for the unsafe preflight or polling segment | Audit only; no runtime implementation in current repo |

This matrix is the authoritative planning boundary for `T005` and later tasks. Any implementation that exceeds it must first change the documented contract rather than quietly broadening or narrowing support in code.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Inventory every tenant-aware command family and map each flow to its underlying process-instance service calls, highlighting direct-get, state-check, wait, ancestry, descendants, cancel, and delete behavior.
2. Refactor the `v88` process-instance service so direct lookup and mixed flows use tenant-safe generated client paths and return tenant-safe `not found` outcomes for mismatches.
3. Audit `v87` flow by flow, implementing tenant-safe behavior only where the upstream generated client can support it and adding explicit unsupported outcomes for the exact unsafe segments.
4. Thread the clarified tenant contract through walker/waiter and command handling so unsupported behavior stays narrowly scoped and mixed flows cannot cross tenant boundaries.
5. Add new regression coverage for all tenant-aware command families, including explicit `--tenant`, environment-derived, profile-derived, and base-config-derived tenant cases, plus supported-vs-unsupported version behavior.
6. Update README and regenerate affected CLI docs if the final behavior or operator guidance changes, then finish with targeted Go tests and `make test`.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design focuses on service-layer and traversal-layer proof of tenant correctness, not just on printing the selected tenant in debug logs.
- **CLI-First, Script-Safe Interfaces**: Still passes. The contract stays CLI-visible and deterministic: supported mismatch means tenant-safe `not found`, unsupported `v8.7` segments fail clearly, and command surfaces remain stable.
- **Tests and Validation Are Mandatory**: Still passes with full command-family regression coverage, versioned service tests, and final `make test`.
- **Documentation Matches User Behavior**: Still passes with planned README and generated CLI doc updates if command-visible tenant behavior changes.
- **Small, Compatible, Repository-Native Changes**: Still passes. The design keeps all changes inside current `cmd`, `c8volt/process`, and `internal/services/processinstance` patterns without introducing parallel service hierarchies.

## Final Implementation Notes

- The shipped tenant contract now matches the design boundary: `v8.8` direct-get-adjacent behavior resolves through tenant-safe search-backed lookup/state flows, while `v8.7` keeps only the segments that can remain tenant-safe without post-filtering.
- Shared mixed-flow helpers now inherit the same contract instead of redefining it: walker ancestry/descendants and waiter polling compose the versioned process-instance service methods directly, so tenant mismatch remains `not found` on supported paths and explicit unsupported on the exact unsafe `v8.7` seam.
- User-facing guidance is aligned with the runtime truth: README and generated CLI help explicitly state that `8.9` is normalized in config but process-instance runtime support in this repository still stops at `v8.8`.

## Final Verification Notes

- Targeted validation for this feature is:
  - `go test ./internal/services/processinstance/... -count=1`
  - `go test ./cmd -count=1`
  - `go test ./config -count=1`
- Repository validation remains `make test`, which is required before the final polish work unit can be considered complete.
- Existing dirty-worktree context must be respected during validation; only feature artifacts and intentional code changes should be staged for the final polish commit.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
