# Implementation Plan: Process Instance Element Enrichment

**Branch**: `235-get-pi-elements` | **Date**: 2026-07-16 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/235-get-pi-elements/spec.md`

## Summary

Add `--with-elements` to `c8volt get process-instance` and aliases `pi`/`pis` so keyed and list/search process-instance output can include runtime element instances beneath each selected process instance. The implementation should reuse the standalone runtime element capability from `c8volt/element` and `internal/services/element`, add process facade/service enrichment models that preserve process-instance ordering, and extend the existing process-instance activity renderer so `vars:`, `incidents:`, and `elements:` compose in stable order without command-layer element lookup logic.

## Technical Context

**Language/Version**: Go, repository current module toolchain

**Primary Dependencies**: Cobra command tree, existing process and element facades, `internal/services/processinstance` enrichment helpers, `internal/services/element` runtime element search, generated Camunda v8.8/v8.9 clients already wrapped by the element service, shared process-instance activity rendering helpers, docs generator

**Storage**: N/A; read-only runtime inspection with no persisted local state

**Testing**: Targeted Go tests for `cmd`, `c8volt/process`, `internal/services/processinstance`, and docs metadata; then `make docs-content` and `make test` before merge

**Target Platform**: Multi-platform CLI binary for c8volt operators and automation

**Project Type**: Go CLI with public facade and internal Camunda service adapters

**Performance Goals**: Preserve process-instance paging behavior; `--limit` caps selected process instances only; enrichment performs one runtime element search per selected process instance and must not make element count affect process-instance paging

**Constraints**: No mutation; no element-specific process-instance filter flags; no standalone element command work; `--with-elements` rejects `--total` and `--keys-only`; keyed mode rejects incompatible search filters; Camunda 8.7 returns unsupported-version errors from the reused element service path; command layer must not call generated clients or versioned services directly

**Scale/Scope**: One existing get command extended with one enrichment flag; process facade and internal enrichment model expansion; renderer and JSON payload expansion; command metadata, README, generated docs, and tests

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The feature is read-only and reports success only after selected process instances and requested element details are loaded or after a clear validation/unsupported-version error.
- **CLI-First, Script-Safe Interfaces**: PASS. Behavior is exposed through the existing `get process-instance` command, validation errors, stable human tree output, and shared JSON envelopes.
- **Tests and Validation Are Mandatory**: PASS. The plan requires service, facade, command validation, renderer, JSON, docs, paging, and unsupported-version tests.
- **Documentation Matches User Behavior**: PASS. `--with-elements` changes visible command behavior, so README and generated CLI docs must be updated from command metadata.
- **Small, Compatible, Repository-Native Changes**: PASS. The design extends existing process-instance enrichment and renderer paths and reuses the already implemented element service/facade.

## Project Structure

### Documentation (this feature)

```text
specs/235-get-pi-elements/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli.md
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 output from /speckit-tasks
```

### Source Code (repository root)

```text
c8volt/
├── client.go            # Wire element service into process facade dependencies
└── process/
    ├── api.go           # Add element enrichment facade method
    ├── client.go        # Delegate to internal process-instance enrichment
    ├── convert.go       # Map enriched element domain/facade models
    ├── model.go         # Public element-enriched process-instance types
    └── client_test.go

internal/
├── domain/
│   ├── element.go
│   └── processinstance_enrichment.go
└── services/
    └── processinstance/
        ├── enrichment.go
        └── enrichment_test.go

cmd/
├── get_processinstance.go
├── get_processinstance_enrichment.go
├── get_processinstance_search.go
├── get_processinstance_validation.go
├── get_processinstance_test.go
├── cmd_views_processinstance_activity.go
├── cmd_views_get_test.go
├── process_api_stub_test.go
└── command_contract_test.go

README.md
docsgen/
```

**Structure Decision**: Use the established single-project CLI layout. The process facade remains the command-facing API for `get pi`; it receives an element service dependency and delegates element attachment to `internal/services/processinstance`. Runtime element lookup/search mechanics stay in `internal/services/element`. `cmd` only validates `--with-elements`, requests process enrichment through the facade, and renders the shared activity output.

## Architecture Grounding

- Architecture extension status: installed.
- Architecture memory status: `.specify/memory/architecture.md` and all five 4+1 view files are present.
- Decision: reuse existing architecture memory without refresh. This feature stays inside documented command/facade/service/generated-client boundaries and extends existing process-instance enrichment behavior.

## Ralph Implementation Context

- Every Ralph implementation iteration MUST receive `--implementation-context specs/ralph-implementation-rules.md`.
- Do not launch Ralph unless the launcher instructions include that implementation context.
- Each Ralph work unit must complete only one story or validation slice and must not stage or commit until validation passes.
- Commit subjects must use Conventional Commits and end with `#242`.

## Ralph Iteration 1 Setup Review

- Reviewed `spec.md`, `contracts/cli.md`, and `specs/ralph-implementation-rules.md`; no implementation conflict was found for the planned command, facade, service, or renderer boundaries.
- The Ralph launch context requirement is present here and in `tasks.md`: `--implementation-context specs/ralph-implementation-rules.md`.
- Commit-policy note: this plan records GitHub issue #242, while the iteration resolved commit policy is `commit.issue: auto`; the iteration must follow the resolved policy for the actual commit subject.

## Phase 0: Research

See [research.md](research.md). All planning unknowns are resolved.

## Phase 1: Design & Contracts

- Data model: [data-model.md](data-model.md)
- CLI contract: [contracts/cli.md](contracts/cli.md)
- Quickstart and verification scenarios: [quickstart.md](quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The contract defines observable outcomes for keyed enrichment, list/search enrichment, combined sections, validation failures, JSON payloads, and unsupported Camunda 8.7 behavior.
- **CLI-First, Script-Safe Interfaces**: PASS. The CLI contract defines valid commands, invalid combinations, output modes, section order, and machine-readable payload shape.
- **Tests and Validation Are Mandatory**: PASS. The quickstart names targeted command, facade, service, docs, and full validation commands.
- **Documentation Matches User Behavior**: PASS. Documentation updates and `make docs-content` are part of the validation path.
- **Small, Compatible, Repository-Native Changes**: PASS. The design extends existing process and element packages without introducing a parallel command family or alternate generated-client path.

## Complexity Tracking

No constitution violations or justified extra complexity.
