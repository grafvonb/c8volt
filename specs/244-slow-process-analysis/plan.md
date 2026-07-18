# Implementation Plan: Slow Process Instance Analysis

**Branch**: `244-slow-process-analysis` | **Date**: 2026-07-18 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/244-slow-process-analysis/spec.md`

**Note**: This feature folder and branch label are aligned with GitHub issue #244.

## Summary

Add a read-only `c8volt ops analyse slow-process-instances` command, with American spelling alias `analyze`, that analyzes selected process instances, calculates whole-process, element, and transition durations, ranks selected roots by measured duration, and renders stable human, JSON, and keys-only outputs. The implementation should add an ops facade/service capability that orchestrates process-instance discovery, keyed lookup, runtime element enrichment, timing calculations, filtering, and comparison indicators below the command layer while reusing existing process-instance and runtime element service capabilities.

## Technical Context

**Language/Version**: Go, using the repository's current Go module configuration

**Primary Dependencies**: Cobra command tree, existing ops command family, process facade/service, element facade/service, `internal/services/ops`, `internal/domain` models, `toolx` duration/key helpers, `typex.Keys`, process-instance and element render helpers, command contract metadata, README and docs generator

**Storage**: N/A; read-only runtime inspection with no persisted local state

**Testing**: Go tests close to changed packages; targeted `go test ./cmd ./c8volt/ops ./internal/services/ops -count=1`, targeted process/element service tests if shared contracts change, then `make docs-content` and `make test` before merge

**Target Platform**: Cross-platform CLI for c8volt operators and automation

**Project Type**: Go CLI with public facade and internal Camunda service adapters

**Performance Goals**: Preserve bounded process-instance discovery controls; `--limit` and `--batch-size` apply only to process-instance discovery; explicit keys and selected timeline details are never truncated by discovery limits; duration and comparison calculations are based on one frozen selection and complete unfiltered timelines

**Constraints**: Read-only command; supports Camunda 8.8 and 8.9; Camunda 8.7 returns unsupported-version errors; no generated-client or versioned-service calls from `cmd`; no listener/job analysis, BPMN path reconstruction, mutation, repair, report-file generation, or unrelated filters; keys-only output remains one key per line and nothing else

**Scale/Scope**: One new ops analysis command plus facade/domain/internal ops service models and renderers; reuses existing process-instance search/key lookup and element search capabilities; adds command metadata, README/docs updates, CLI contracts, and focused tests for validation, timing, filtering, output modes, unsupported versions, and empty results

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The feature is read-only and claims success only after selection, element inspection, timing, and output generation complete, or after clear validation/unsupported-version errors.
- **CLI-First, Script-Safe Interfaces**: PASS. Behavior is exposed through stable Cobra commands, validation errors, human output, JSON output, and keys-only output suitable for pipelines.
- **Tests and Validation Are Mandatory**: PASS. The plan requires command, facade, service, renderer, JSON, keys-only, validation, docs, and unsupported-version tests.
- **Documentation Matches User Behavior**: PASS. The command, flags, aliases, examples, and output contracts are user-facing and require README plus generated CLI docs updates.
- **Small, Compatible, Repository-Native Changes**: PASS. The design extends existing ops, process, element, command contract, and renderer patterns without adding a parallel architecture.

## Project Structure

### Documentation (this feature)

```text
specs/244-slow-process-analysis/
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
├── client.go            # Wire ops service dependencies if element/process access is expanded
└── ops/
    ├── api.go           # Public analysis facade contract
    ├── client.go        # Thin facade over internal ops service
    ├── convert.go       # Public/domain analysis model mapping
    ├── model.go         # Public analysis request/result types
    └── client_test.go

internal/
├── domain/
│   ├── ops_slow_process_analysis.go
│   ├── processinstance.go
│   └── element.go
└── services/
    └── ops/
        ├── api.go
        ├── slow_process_analysis.go
        └── slow_process_analysis_test.go

cmd/
├── ops_analyse_slow_process_instances.go
├── ops_analyse_slow_process_instances_test.go
├── cmd_views_ops_slow_process_analysis.go
├── cmd_views_ops_slow_process_analysis_test.go
├── ops_contract_test.go
└── command_contract_test.go

README.md
docsgen/
```

**Structure Decision**: Use the established single-project CLI layout. `cmd` owns flags, aliases, validation, command metadata, and rendering only. `c8volt/ops` exposes thin public models and delegates to `internal/services/ops`. `internal/services/ops` owns selection discovery, frozen-set orchestration, runtime element lookup coordination, duration calculations, detail filtering, comparison sample calculation, and deterministic ordering by calling existing process-instance and element service capabilities. Version-specific Camunda behavior remains in the existing process-instance and element service adapters.

## Architecture Grounding

- Architecture extension status: installed.
- Architecture memory status: `.specify/memory/architecture.md` and all five 4+1 view files are present.
- Decision: reuse existing architecture memory without refresh. This feature stays inside documented command/facade/service/generated-client boundaries and extends the existing ops command family plus existing process-instance and runtime element inspection services.

## Ralph Implementation Context

- Every Ralph implementation iteration MUST receive `--implementation-context specs/ralph-implementation-rules.md`.
- Do not launch Ralph unless the launcher instructions include that implementation context.
- Each Ralph work unit must complete only one story or validation slice and must not stage or commit until validation passes.
- Commit subjects must use Conventional Commits and end with `#244`.

## Phase 0: Research

See [research.md](research.md). All planning unknowns are resolved.

## Phase 1: Design & Contracts

- Data model: [data-model.md](data-model.md)
- CLI contract: [contracts/cli.md](contracts/cli.md)
- Quickstart and verification scenarios: [quickstart.md](quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The contract defines observable outcomes for keyed analysis, process-definition discovery, empty selections, unsupported versions, timing calculations, filtering, output modes, and read-only behavior.
- **CLI-First, Script-Safe Interfaces**: PASS. The CLI contract defines valid commands, aliases, selectors, invalid combinations, stdin behavior, human output, JSON payload shape, and keys-only output.
- **Tests and Validation Are Mandatory**: PASS. The quickstart names targeted command, facade, internal service, docs, and full validation commands.
- **Documentation Matches User Behavior**: PASS. Documentation updates and `make docs-content` are part of the validation path.
- **Small, Compatible, Repository-Native Changes**: PASS. Design artifacts keep the work scoped to existing `cmd`, `c8volt/ops`, `internal/domain`, `internal/services/ops`, and existing process/element service ownership boundaries.

## Complexity Tracking

No constitution violations or justified extra complexity.
