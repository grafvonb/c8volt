# Implementation Plan: Version-Aware Process-Instance Paging and Overflow Handling

**Branch**: `101-processinstance-paging` | **Date**: 2026-04-12 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/101-processinstance-paging/spec.md)
**Input**: Feature specification from `/specs/101-processinstance-paging/spec.md`

## Summary

Add shared paging-aware search orchestration to `c8volt get process-instance`, `c8volt cancel process-instance`, and `c8volt delete process-instance` so search mode no longer silently truncates at the first `1000` matches. The design keeps the existing command -> facade -> versioned service path, introduces one shared config key for the default process-instance page size, uses version-aware overflow signals from the underlying APIs, preserves current direct-key workflows, and makes continuation behavior explicit through the existing interactive/`--auto-confirm` confirmation model.

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, existing process-instance command helpers in `cmd/`, facade types in `c8volt/process`, config model in `config/`, versioned services in `internal/services/processinstance/v87` and `internal/services/processinstance/v88`, generated Camunda clients under `internal/clients/camunda/...`  
**Storage**: N/A  
**Testing**: `go test`, `make test`, command regression tests under `cmd/`, facade regression tests in `c8volt/process`, versioned service tests under `internal/services/processinstance/v87` and `internal/services/processinstance/v88`  
**Target Platform**: CLI for local and CI use against Camunda 8.7 and 8.8 environments  
**Project Type**: CLI  
**Performance Goals**: Preserve the existing default fetch size of `1000`, avoid silent truncation, avoid unbounded client-side overfetching, and continue processing one page at a time with predictable user-visible progress  
**Constraints**: Reuse the current Cobra command surfaces and shared process-instance search helper path, keep direct `--key` workflows unchanged, use one shared config key for the default page size, keep continuation behavior consistent across `get`/search-based `cancel`/search-based `delete`, treat user-declined continuation as a non-error partial completion, stop and warn if overflow cannot be determined after the version-appropriate fallback, update `README.md` plus generated CLI docs in the same change  
**Scale/Scope**: Three existing commands, shared command-layer search/filter helpers, one facade search entry point, two versioned process-instance search implementations, targeted command/facade/service tests, and user-facing documentation for process-instance command behavior

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. The feature is specifically about preventing silent partial results/actions, and the design makes continuation and indeterminate overflow states explicit instead of reporting success for unseen matches.
- **CLI-First, Script-Safe Interfaces**: Pass. The feature stays on existing Cobra commands, uses explicit prompts or `--auto-confirm`, keeps output/exit behavior intentional, and introduces one stable shared config key instead of command-specific hidden rules.
- **Tests and Validation Are Mandatory**: Pass. The plan requires command, facade, and versioned service regression coverage where behavior changes surface, plus `make test` before commit.
- **Documentation Matches User Behavior**: Pass. The feature changes visible command behavior, prompts, defaults, and config, so `README.md`, Cobra help, and generated CLI docs must all be updated together.
- **Small, Compatible, Repository-Native Changes**: Pass. The design extends the current process-instance command/service seams instead of introducing a new paging subsystem or parallel command hierarchy.

## Project Structure

### Documentation (this feature)

```text
specs/101-processinstance-paging/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── process-instance-paging.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_processinstance.go
├── get_processinstance_test.go
├── cancel_processinstance.go
├── cancel_test.go
├── delete_processinstance.go
├── delete_test.go
├── cmd_processinstance_test.go
└── root.go

config/
├── app.go
└── config.go

c8volt/process/
├── api.go
├── client.go
└── client_test.go

internal/services/processinstance/
├── api.go
├── v87/
│   ├── contract.go
│   ├── service.go
│   └── service_test.go
└── v88/
    ├── contract.go
    ├── service.go
    └── service_test.go

README.md
docs/cli/
├── c8volt_get_process-instance.md
├── c8volt_cancel_process-instance.md
└── c8volt_delete_process-instance.md
```

**Structure Decision**: Keep the feature in the repository’s current command orchestration path. `cmd/get_processinstance.go` already owns the shared process-instance search flags and validation helpers reused by `cancel_processinstance.go` and `delete_processinstance.go`; that remains the canonical seam for page-size resolution, continuation prompts, and output summaries. The facade and versioned service layers stay responsible for search execution and version-specific overflow signals rather than introducing a separate paging-specific package.

## Phase 0: Research

- Confirm the best repository-native home for the shared page-size default and bind it through the existing Viper-backed config flow.
- Confirm the exact version-aware overflow signals available from the generated clients: v8.8 native search page metadata versus v8.7 Operate search behavior without equivalent response metadata.
- Confirm where search-mode `cancel` and `delete` can be extended from one-shot `SearchProcessInstances(..., consts.MaxPISearchSize)` calls into a shared page loop without changing direct-key flows.
- Confirm the safest CLI behavior for partial completion, cumulative progress reporting, and indeterminate-overflow stop/warn behavior.
- Confirm the documentation and regression seams needed to keep `README.md`, generated CLI docs, and process-instance command tests aligned with the shipped behavior.

## Phase 1: Design

- Add one shared config field for the default process-instance search page size and resolve page size in order: `--count` override, shared config key, fallback constant `1000`.
- Extend the command-layer process-instance search helpers so `get`, search-based `cancel`, and search-based `delete` all use the same paging orchestrator and operator-facing progress summary logic.
- Preserve current direct-key behavior by keeping paging in search mode only; keyed `cancel` and `delete` remain outside this loop.
- Teach the versioned service layer to surface enough paging metadata for the command layer to decide whether more matches remain:
  - v8.8 uses the generated `SearchQueryPageResponse` metadata and its native pagination primitives.
  - v8.7 uses a version-appropriate follow-up strategy because the Operate response type does not expose equivalent page metadata; if the fallback still cannot prove exhaustion, the command stops and warns.
- Keep continuation behavior consistent across all three commands: prompt when more matches remain and `--auto-confirm` is false, auto-continue when `--auto-confirm` is true, and treat user-declined continuation as a non-error partial completion.
- Update operator-facing output so each page reports the page size used, current-page count, cumulative processed count, whether more matches remain, and whether the next step is prompting, auto-continuing, or stopping with a warning.
- Update README examples/help text and regenerate CLI docs for the new paging/config behavior.

## Phase 2: Task Planning Approach

- Start with shared config and command-orchestration seams because all three affected commands depend on those decisions.
- Add version-specific service/facade support for overflow metadata next so the shared command loop can make correct continuation decisions.
- Add command and lower-level regression tests before documentation work so the output and stop/continue semantics are locked down early.
- Finish with README/help/doc regeneration and final `make test` validation.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design explicitly prevents silent truncation, preserves existing completion guarantees for write commands, and treats indeterminate overflow as a stop-and-warn case instead of a silent success.
- **CLI-First, Script-Safe Interfaces**: Still passes. The design uses existing commands, shared config, explicit prompts, and deterministic auto-confirm behavior without introducing incompatible command surfaces.
- **Tests and Validation Are Mandatory**: Still passes with planned command, facade, and versioned service coverage plus `make test`.
- **Documentation Matches User Behavior**: Still passes with explicit `README.md` updates and regenerated CLI reference docs.
- **Small, Compatible, Repository-Native Changes**: Still passes. The design extends current helpers and versioned services rather than inventing a new standalone paging abstraction.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
