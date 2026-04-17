# Implementation Plan: Add Camunda v8.9 Runtime Support

**Branch**: `110-camunda-v89-support` | **Date**: 2026-04-17 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/spec.md)
**Input**: Feature specification from `/specs/110-camunda-v89-support/spec.md`

## Summary

Add full repository-wide Camunda `v8.9` runtime support with the same command-family scope currently supported on `v8.8`, while preserving existing `v8.7` and `v8.8` behavior. The design extends the current versioned service/facade architecture by adding native `v89` implementations for cluster, process-definition, process-instance, and resource services, wiring them through the existing factories and top-level client, requiring final native `v89` paths to depend only on the generated `internal/clients/camunda/v89/camunda/client.gen.go` client, and backing the change with explicit factory coverage, at least one `v8.9` execution test per repository command family, plus required README and generated CLI documentation updates.

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...`, existing helpers in `internal/services/common`, shared facades under `c8volt/...`  
**Storage**: File-based YAML config plus environment variables; no persistent datastore changes  
**Testing**: `go test`, `make test`, factory tests under `internal/services/*/factory_test.go`, versioned service tests under `internal/services/{cluster,processdefinition,processinstance,resource}/v87` and `v88`, command regression tests under `cmd/`, plus documentation regeneration via `make docs` and `make docs-content` when help text or README changes  
**Target Platform**: Cross-platform CLI for local and CI use against Camunda 8.7, 8.8, and the new 8.9 runtime surface  
**Project Type**: CLI  
**Performance Goals**: No user-visible regression in command startup, search/paging behavior, or wait/poll flows; preserve current confirmation semantics; avoid new multi-hop fallback paths in the final native `v8.9` implementation  
**Constraints**: Keep existing Cobra command surfaces and error contracts stable, reuse repository-native versioned service and facade patterns, preserve `v8.7` and `v8.8` behavior, require repository-wide `v8.8` command-family parity on `v8.9`, allow temporary fallback only as a documented transition, require final native `v8.9` paths to use only `internal/clients/camunda/v89/camunda/client.gen.go`, require at least one explicit `v8.9` execution test per repository command family, update user-facing docs before release readiness, and finish with `make test`  
**Scale/Scope**: `toolx/version.go`, root/bootstrap version messaging in [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go), service factories under [`internal/services/cluster/factory.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/cluster/factory.go), [`internal/services/processdefinition/factory.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/factory.go), [`internal/services/processinstance/factory.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory.go), [`internal/services/resource/factory.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/factory.go), top-level client wiring in [`c8volt/client.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/client.go), new `internal/services/*/v89` packages, command families under `cmd/` that already exercise the `v8.8` service surface, and the version-support docs in [`README.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md), `docs/index.md`, and generated `docs/cli/`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. The feature is about making `v8.9` genuinely usable, so the plan requires real factory wiring, native service implementations, command execution proof, and honest docs rather than a config-only version claim.
- **CLI-First, Script-Safe Interfaces**: Pass. Existing commands, flags, output formats, and error contracts stay in place; only the supported version surface expands.
- **Tests and Validation Are Mandatory**: Pass. The plan requires factory coverage, at least one explicit `v8.9` command execution test per repository command family, preserved `v8.7`/`v8.8` coverage, docs regeneration where needed, and final `make test`.
- **Documentation Matches User Behavior**: Pass. The clarified spec makes docs a release gate, so version-support messaging in README and generated CLI docs is part of completion, not follow-up polish.
- **Small, Compatible, Repository-Native Changes**: Pass. The design extends the current `v87`/`v88` service and facade structure with parallel `v89` implementations instead of introducing a new runtime abstraction.

## Project Structure

### Documentation (this feature)

```text
specs/110-camunda-v89-support/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── v89-support.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── root.go
├── get*.go
├── deploy*.go
├── run*.go
├── expect*.go
├── cancel*.go
├── delete*.go
└── *_test.go

c8volt/
├── client.go
├── cluster/
├── process/
├── resource/
└── task/

config/
├── app.go
└── *_test.go

toolx/
└── version.go

internal/clients/camunda/v89/
├── camunda/
├── operate/
├── administrationsm/
└── tasklist/

internal/services/
├── cluster/
│   ├── api.go
│   ├── factory.go
│   ├── factory_test.go
│   ├── v87/
│   ├── v88/
│   └── v89/
├── processdefinition/
│   ├── api.go
│   ├── factory.go
│   ├── factory_test.go
│   ├── v87/
│   ├── v88/
│   └── v89/
├── processinstance/
│   ├── api.go
│   ├── factory.go
│   ├── factory_test.go
│   ├── walker/
│   ├── waiter/
│   ├── v87/
│   ├── v88/
│   └── v89/
└── resource/
    ├── api.go
    ├── factory.go
    ├── factory_test.go
    ├── v87/
    ├── v88/
    └── v89/

README.md
docs/
└── cli/
```

**Structure Decision**: Keep all work inside the repository’s current versioned-service shape. New `v89` packages should mirror the existing `v88` package boundaries and constructor patterns, factories should remain the only place that choose service versions, `c8volt/client.go` should keep the top-level wiring centralized, and command code should stay thin consumers of the facades instead of branching on version.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/research.md).

- Confirm the authoritative parity scope by mapping the repository’s current `v8.8` runtime surface to service families and command families.
- Confirm which current service families already use only the generated Camunda client and which ones still mix in other generated clients.
- Confirm whether the generated `internal/clients/camunda/v89/camunda/client.gen.go` surface contains the process-instance endpoints needed to remove the current `v88` Operate dependency from the final native `v89` path.
- Confirm the factory, facade, and documentation seams that still cap user-visible runtime support at `v8.8`.
- Confirm the best regression anchors for proving factory selection, preserved `v8.7`/`v8.8` behavior, and at least one explicit `v8.9` execution test per repository command family.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/quickstart.md)
- [contracts/v89-support.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/110-camunda-v89-support/contracts/v89-support.md)

- Extend the supported version surface at the source of truth by adding `v8.9` to `toolx.SupportedCamundaVersions()` and preserving normalization behavior that already recognizes `8.9`.
- Add native `v89` implementations for `cluster`, `processdefinition`, `processinstance`, and `resource`, following the existing `api.go` assertions, `v88` package layout, `common.PrepareServiceDeps`, and `common.EnsureLoggerAndClients` patterns.
- Keep final native `v89` paths on the generated `internal/clients/camunda/v89/camunda/client.gen.go` boundary only; any mixed-client behavior is transitional fallback and must not remain in final accepted `v89` service paths.
- Route top-level client creation and all command families through the existing service factories so `v8.9` becomes a first-class runtime choice without command-local version branching.
- Prove repository-wide `v8.8` command-family parity with at least one explicit `v8.9` execution test for each current command family, plus factory and preserved-version coverage.
- Treat README and generated CLI docs as part of the release gate, updating version-support messaging before the feature is considered complete.

### Authoritative Service and Command Boundary

| Surface | Current `v8.8` state | `v8.9` design target | Final acceptance rule |
|--------|----------------------|----------------------|-----------------------|
| Version normalization and advertised support | `NormalizeCamundaVersion()` accepts `8.9`, but supported-version lists still stop at `8.8` | Advertise `8.9` everywhere supported runtime versions are listed | User-facing version surfaces must stop saying runtime support ends at `v8.8` |
| Cluster service | Native `v88` service already uses Camunda client only | Add native `v89` service mirroring `v88` topology/license behavior | Final `v89` cluster path uses only `internal/clients/camunda/v89/camunda` |
| Process-definition service | Native `v88` service already uses Camunda client only | Add native `v89` service for search/get/xml/statistics | Final `v89` definition path uses only `internal/clients/camunda/v89/camunda` |
| Resource service | Native `v88` service already uses Camunda client only | Add native `v89` service for deploy/get/delete and keep deployment confirmation behavior intact | Final `v89` resource path uses only `internal/clients/camunda/v89/camunda` |
| Process-instance service | Native `v88` service still mixes Camunda and Operate generated clients | Add native `v89` service using only `internal/clients/camunda/v89/camunda`, including lookup, search, cancel, delete, wait, walker, and waiter composition | Final `v89` process-instance path must remove mixed-client internals and leave fallback behind |
| Top-level client and facades | `c8volt/client.go` wires cluster, process, task, and resource through service factories | Keep the same centralized wiring and let factories choose `v89` implementations | No command-local version branching |
| Repository command families | `get cluster`, `get process-definition`, `get resource`, `deploy process-definition`, `delete process-definition`, `run/get/walk/cancel/delete/expect process-instance` are already supported through `v8.8` | Each family must execute successfully under `v8.9` with the same user-facing contract | At least one explicit `v8.9` execution test per repository command family |
| Documentation surface | README, docs homepage, and generated CLI reference still describe `8.9` as recognized but not fully supported | Update version support statements, examples, and generated docs to reflect `v8.9` support | Docs are part of release readiness, not a follow-up |

This matrix is the authoritative planning boundary for later task generation. Any implementation that leaves a command family on undocumented fallback, keeps a final native `v89` path on a non-`camunda` generated client, or updates code without the matching docs/test proof is incomplete by design.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Update version-support source-of-truth surfaces so `SupportedCamundaVersions()` and root/docs messaging can truthfully include `v8.9`.
2. Add native `v89` service implementations and factory coverage for `cluster`, `processdefinition`, and `resource`, mirroring the current `v88` structure and tests.
3. Add the native `v89` process-instance service, including walker/waiter integration, using only the generated `v89` Camunda client surface and removing the need for final-path Operate usage.
4. Wire the new `v89` services through `c8volt/client.go`, shared facades, and any preserved helper seams without introducing command-local version switching.
5. Add or update tests for preserved `v8.7`/`v8.8` behavior, factory selection, and at least one explicit `v8.9` execution path per repository command family.
6. Update README, regenerate `docs/index.md` and `docs/cli/`, then finish with focused Go tests and final `make test`.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design requires factory wiring, native service implementations, command execution proof, and honest docs before `v8.9` can be called supported.
- **CLI-First, Script-Safe Interfaces**: Still passes. Commands remain stable and version selection continues to flow through config/bootstrap instead of ad hoc command changes.
- **Tests and Validation Are Mandatory**: Still passes with explicit factory, service, command, and repo-gate validation requirements.
- **Documentation Matches User Behavior**: Still passes. Version-support docs and examples are now explicitly part of release readiness.
- **Small, Compatible, Repository-Native Changes**: Still passes. The work extends the established `v87`/`v88` service pattern into `v89` rather than introducing a parallel architecture.

## Final Implementation Notes

- The repository already contains generated `v89` Camunda clients, so the main work is not code generation but service/factory adoption plus honest version-surface updates.
- Three of the four current `v88` service families already fit the final `v89` client-boundary rule naturally: `cluster`, `processdefinition`, and `resource` depend only on the generated Camunda client. `processinstance` is the main design-sensitive area because `v88` still mixes Camunda and Operate clients.
- The generated `internal/clients/camunda/v89/camunda/client.gen.go` surface includes the process-instance operations needed for a native final path, including get, search, cancel, delete, statistics, and cluster/resource/definition endpoints.
- Temporary fallback is allowed by the clarified spec as a transition tool, but the plan treats it as non-final by default. Tasks should only leave fallback in place when it is explicitly documented and scheduled for removal inside the same feature.
- User Story 2 status note: native `v89` factory routing is now in place for all four versioned service families, so there is no remaining active fallback path inside the repository runtime. The fallback rules remain as a guardrail for incomplete future work, not as a description of current behavior.

## Final Verification Notes

- Targeted validation for this feature is:
  - `go test ./internal/services/cluster/... -count=1`
  - `go test ./internal/services/processdefinition/... -count=1`
  - `go test ./internal/services/processinstance/... -count=1`
  - `go test ./internal/services/resource/... -count=1`
  - `go test ./cmd -count=1`
  - `make docs`
  - `make docs-content`
- Repository validation remains `make test`, which is required before the feature can be considered done.
- Documentation regeneration should follow command/help text and `README.md` changes so generated docs stay in sync with the runtime truth.
- User-facing version messaging is complete only when root help, `README.md`, `docs/index.md`, and generated `docs/cli/*` all agree that `8.9` is supported with the same repository command-family coverage already available on `8.8`.
- Final verification for this story should explicitly check that no user-facing surface still says `8.9` is normalization-only or that runtime support stops at `8.8`.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
