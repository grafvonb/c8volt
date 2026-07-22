# Implementation Plan: Slow Process Timeline Readability

**Branch**: `248-slow-process-timeline` | **Date**: 2026-07-22 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/248-slow-process-timeline/spec.md`

**Note**: This feature folder and branch label are aligned with GitHub issue #248.

## Summary

Improve the existing read-only `c8volt ops analyse slow-process-instances` human output so the default view shows a compact `slowest elements:` hotspot summary instead of the full chronological timeline, while adding `--with-full-timeline` to restore the complete human timeline on demand. The implementation should keep the existing analysis service payload complete, preserve JSON and keys-only output exactly, and focus changes in command flags, command metadata, human rendering, README/generated docs, and close command/renderer tests.

## Technical Context

**Language/Version**: Go, using the repository's current Go module configuration

**Primary Dependencies**: Cobra command tree, existing slow-process analysis command, `cmd/cmd_views_ops_slow_process_analysis.go`, existing `c8volt/ops` public models, existing `internal/domain` and `internal/services/ops` slow-process analysis models, command contract metadata, README and docs generator

**Storage**: N/A; read-only runtime analysis presentation with no persisted local state

**Testing**: Go tests close to changed packages; targeted `go test ./cmd -run 'Test.*SlowProcessAnalysis|TestCommandContractOpsAnalyseSlowProcessInstances|TestOpsContract' -count=1`, `go test ./docsgen -count=1` when docs metadata changes, then `make docs-content` and `make test` before merge

**Target Platform**: Cross-platform CLI for c8volt operators and automation

**Project Type**: Go CLI with public facade and internal service layers already implemented for slow-process analysis

**Performance Goals**: Preserve existing analysis work and avoid extra remote lookups; default human summarization should operate on the already-calculated visible timeline for each root and scale linearly with timeline row count

**Constraints**: Read-only behavior; supports the existing Camunda 8.8 and 8.9 analysis path; Camunda 8.7 behavior remains unchanged; JSON output and keys-only output remain unchanged; `--with-full-timeline` affects human rendering only; existing selection, filtering, duration calculation, and ordering semantics remain unchanged

**Scale/Scope**: One focused presentation change to an existing command, including flag/help examples, human renderer summary selection, hidden-row summary, renderer/command contract tests, README updates, and generated CLI docs

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The feature is read-only and changes only how completed analysis results are presented; success remains tied to completed analysis/rendering or existing validation errors.
- **CLI-First, Script-Safe Interfaces**: PASS. The behavior is exposed through a stable Cobra flag and preserves machine-safe JSON and keys-only contracts.
- **Tests and Validation Are Mandatory**: PASS. The plan requires focused renderer, command flag, command contract, docs metadata, JSON stability, and keys-only stability tests.
- **Documentation Matches User Behavior**: PASS. The new flag and changed default human output are user-facing and require README plus generated CLI documentation updates.
- **Small, Compatible, Repository-Native Changes**: PASS. The plan reuses the existing slow-process analysis command, facade models, service payload, and renderer patterns without adding a new architecture.

## Project Structure

### Documentation (this feature)

```text
specs/248-slow-process-timeline/
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
cmd/
├── ops_analyse_slow_process_instances.go          # Add --with-full-timeline flag and help/examples
├── ops_analyse_slow_process_instances_test.go     # Validate flag request/render-mode parsing and output-mode isolation
├── cmd_views_ops_slow_process_analysis.go         # Default hotspot summary and full-timeline dispatch
├── cmd_views_ops_slow_process_analysis_test.go    # Human summary, full timeline, hidden count, JSON/keys-only stability
├── command_contract_test.go                       # Command metadata and flag contract coverage
└── ops_contract_test.go                           # Ops command family contract coverage where relevant

c8volt/ops/
├── model.go                                      # Prefer unchanged; only extend if a render-mode field is truly needed
└── convert.go                                    # Prefer unchanged; preserve JSON contract if touched

internal/
└── domain/
    └── ops_slow_process_analysis.go              # Prefer unchanged; service payload remains complete

README.md
docsgen/
docs/cli/                                         # Regenerated by make docs-content
```

**Structure Decision**: Use the existing single-project CLI layout. Keep selection, duration calculation, detail filtering, comparisons, and complete timeline construction in the already implemented ops service. Keep this feature in `cmd` unless implementation discovers a narrow need to carry an internal human-rendering preference through existing command-local state. Do not change generated Camunda clients or versioned service adapters.

## Architecture Grounding

- Architecture extension status: installed.
- Architecture memory status: `.specify/memory/architecture.md` and 4+1 view files are present.
- Decision: reuse existing architecture memory without refresh. This feature is a presentation and CLI-contract refinement inside the established `cmd` boundary, building on the service/facade work from `specs/244-slow-process-analysis`.

## Ralph Implementation Context

- Every Ralph implementation iteration MUST receive `--implementation-context specs/ralph-implementation-rules.md`.
- Do not launch Ralph unless the launcher instructions include that implementation context.
- Each Ralph work unit must complete only one story or validation slice and must not stage or commit until validation passes.
- Commit subjects must use Conventional Commits and end with `#248`.

## Phase 0: Research

See [research.md](research.md). All planning unknowns are resolved.

## Phase 1: Design & Contracts

- Data model: [data-model.md](data-model.md)
- CLI contract: [contracts/cli.md](contracts/cli.md)
- Quickstart and verification scenarios: [quickstart.md](quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The contract defines observable default summary, full-timeline, hidden-row, and output-mode stability outcomes.
- **CLI-First, Script-Safe Interfaces**: PASS. The CLI contract documents the new flag, default human behavior, full-timeline behavior, and unchanged machine modes.
- **Tests and Validation Are Mandatory**: PASS. The quickstart names targeted renderer/command/contract/docs tests plus full project validation.
- **Documentation Matches User Behavior**: PASS. Documentation updates and `make docs-content` are part of the validation path.
- **Small, Compatible, Repository-Native Changes**: PASS. Design artifacts keep implementation scoped to the existing command and renderer, with service/domain changes explicitly avoided unless proven necessary.

## Complexity Tracking

No constitution violations or justified extra complexity.
