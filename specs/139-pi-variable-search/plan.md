# Implementation Plan: Native Process Instance Variable Search

**Branch**: `139-pi-variable-search` | **Date**: 2026-05-25 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/139-pi-variable-search/spec.md`

## Summary


## Technical Context

**Language/Version**: Go, repository current module toolchain
**Primary Dependencies**: Cobra command tree, existing `get pi` search/paging helpers, command contract metadata, `c8volt/process` facade, `internal/domain` process-instance filters, `internal/services/processinstance` API and v8.8/v8.9 adapters, generated Camunda clients, shared filter constructors, docs generator
**Storage**: N/A
**Target Platform**: Multi-platform CLI binary
**Project Type**: Go CLI
**Performance Goals**: Preserve current process-instance search paging behavior and avoid extra per-process-instance variable lookups for native variable search

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The feature is read-only and requires tests proving returned search behavior, native request construction, and unsupported-version closure.
- **CLI-First, Script-Safe Interfaces**: PASS. The plan covers flags, help, command metadata, JSON/keys-only-safe behavior, local validation, and deterministic unsupported errors.
- **Tests and Validation Are Mandatory**: PASS. The plan requires parser, command, facade/domain, version adapter, docs, and regression validation before implementation is complete.
- **Documentation Matches User Behavior**: PASS. README-facing examples and generated CLI docs are required because the command surface changes.
- **Small, Compatible, Repository-Native Changes**: PASS. The work reuses the existing `get pi` command, process facade, domain filter, versioned service adapters, docs generator, and test helpers instead of introducing a parallel command or service family.

## Project Structure

### Documentation (this feature)

```text
specs/139-pi-variable-search/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── progress.md
├── contracts/
│   └── cli-pi-variable-search.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_processinstance.go
├── get_processinstance_*.go
├── get_processinstance_test.go
├── command_contract_test.go
└── cmd_views_*.go

c8volt/process/
├── api.go
├── client.go
├── convert.go
├── model.go
└── client_test.go

internal/
├── domain/processinstance.go
├── services/processinstance/api.go
├── services/processinstance/v88/
└── services/processinstance/v89/

README.md
docs/cli/
docs/index.md
```


## Architecture Grounding

- Architecture extension status: installed.
- Architecture memory status: `.specify/memory/architecture.md` and all five 4+1 view files are present.
- Decision: reuse existing architecture memory without refresh. Issue #139 stays within documented command contract, facade/domain/service, generated-client isolation, version-gating, and docs-generation boundaries. It adds a new process-instance search filter surface but does not change runtime topology, ownership boundaries, or deployment assumptions.

## Ralph Implementation Context

- Every implementation iteration MUST receive `--implementation-context specs/ralph-implementation-rules.md`.
- Do not launch Ralph unless the launcher instructions include that implementation context.
- Each Ralph work unit must complete only one story or validation slice and must not stage or commit until validation passes.
- Commit subjects must use Conventional Commits and end with `#139`.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

- Data model: [data-model.md](./data-model.md)
- CLI contract: [contracts/cli-pi-variable-search.md](./contracts/cli-pi-variable-search.md)
- Quickstart and verification scenarios: [quickstart.md](./quickstart.md)

## Post-Design Constitution Check

- **CLI-First, Script-Safe Interfaces**: PASS. The contract covers flag grammar, local parse failures, help text, command metadata, JSON/keys-only compatibility, and documentation.
- **Tests and Validation Are Mandatory**: PASS. The design requires targeted package tests, generated docs refresh, and broader validation before commit readiness.
- **Documentation Matches User Behavior**: PASS. Documentation updates are source-driven and generated outputs are refreshed.
- **Small, Compatible, Repository-Native Changes**: PASS. The design rejects a parallel command family, Operate fallback, command-layer generated-client access, and facade-owned backend request construction.

## Complexity Tracking

No constitution violations or complexity exceptions are required.
