# Implementation Plan: Process-Instance Limit and Batch Size Flags

**Branch**: `140-pi-limit-batch-size` | **Date**: 2026-04-25 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/140-pi-limit-batch-size/spec.md)
**Input**: Feature specification from `/specs/140-pi-limit-batch-size/spec.md`

## Summary

Add total-match limiting to search-driven process-instance commands and rename the current per-page `--count` flag to `--batch-size`. The implementation should extend the existing shared process-instance paging helpers in `cmd/get_processinstance.go`, preserve direct `--key` behavior, apply `--limit` consistently across `get`, search-based `cancel`, and search-based `delete`, reject `--total` combined with `--limit`, and refresh command help plus generated documentation so operators can distinguish total limits from per-batch fetch sizing.

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, existing command helpers in `cmd/`, process facade in `c8volt/process`, config model in `config/`, versioned process-instance services in `internal/services/processinstance/*`, generated docs pipeline via repository make targets  
**Storage**: N/A  
**Testing**: `go test`, focused package tests under `cmd/`, final `make test`; docs regeneration through `make docs-content` when command metadata changes  
**Target Platform**: CLI for local and CI use against supported Camunda process-instance search flows  
**Project Type**: CLI  
**Performance Goals**: Avoid fetching or processing pages after `--limit` is satisfied; avoid processing more matched instances than the configured limit; preserve current page-size defaults and configured defaults under the new `--batch-size` name  
**Constraints**: Reuse existing shared process-instance paging orchestration; keep direct key workflows non-paged; reject `--limit` with `--key`; reject `--limit` with `--total`; remove `--count` without aliasing on affected command paths; keep `-n` for batch size; add `-l` for limit; update README and generated CLI docs in the same change  
**Scale/Scope**: Three command surfaces (`get`, `cancel`, `delete` process-instance), shared command-level paging helpers, focused command tests, README examples, generated CLI docs, and help text

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. The feature prevents accidental over-processing by stopping once a user-defined total limit is reached and requires progress output to distinguish limit, no-more-matches, and user-aborted stops.
- **CLI-First, Script-Safe Interfaces**: Pass. Behavior is exposed through explicit Cobra flags and standard validation. Removed `--count` fails through normal invalid-argument handling instead of a hidden alias.
- **Tests and Validation Are Mandatory**: Pass. The plan requires focused command tests for cross-page limit behavior, invalid combinations, and mode-specific behavior, plus final repository validation.
- **Documentation Matches User Behavior**: Pass. The feature changes visible flags and examples, so README and generated CLI docs must be updated from command metadata.
- **Small, Compatible, Repository-Native Changes**: Pass. The design extends existing process-instance paging helpers rather than introducing a new paging subsystem.

## Project Structure

### Documentation (this feature)

```text
specs/140-pi-limit-batch-size/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── process-instance-limit-flags.md
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

README.md
docs/cli/
├── c8volt_get_process-instance.md
├── c8volt_cancel_process-instance.md
└── c8volt_delete_process-instance.md
```

**Structure Decision**: Keep implementation in the current command-layer paging path. `cmd/get_processinstance.go` already owns shared process-instance search flags, page-size resolution, paging continuation, progress summaries, and search-page processing helpers reused by search-based `cancel` and `delete`. That remains the canonical place for `--batch-size`, `--limit`, truncation to the remaining limit, validation against `--key` and `--total`, and limit-reached stop reporting.

## Phase 0: Research

- Confirm `--batch-size` can replace the existing `--count` flag registration on all affected command paths without preserving an alias.
- Confirm the current `flagGetPISize` and `resolvePISearchSize` path is the right repository-native place to preserve per-batch behavior under the new flag name.
- Confirm `--limit` should be command-layer behavior because the limit applies after search and local filters, before rendering or destructive page actions.
- Confirm `--total` and `--limit` should be rejected together because count-only output and limited detail output are separate modes.
- Confirm search-page loops can stop without prompt when the limit is reached by extending the shared progress/continuation state rather than duplicating loops per command.
- Confirm docs should be updated from Cobra examples/help and generated via existing docs targets after command metadata changes.

## Phase 1: Design

- Rename the affected command registrations from `--count` to `--batch-size`, keep `-n`, and update related help text such as worker descriptions.
- Add a shared positive integer limit flag with `--limit` and `-l` to the affected search-driven process-instance command paths.
- Validate `--limit` before execution: positive integer only, rejected with direct key mode, and rejected when combined with `--total`.
- Extend shared search-page helpers so each page is truncated to the remaining limit before results are rendered or keys are passed to cancel/delete planning.
- Track a limit-reached completion state in progress summaries so verbose one-line output can distinguish limit reached from no more matches, user-aborted continuation, and indeterminate warning stop.
- Ensure `get` aggregation, incremental one-line/keys-only rendering, `--json`, `--automation`, and `--auto-confirm` all use the same remaining-limit calculation.
- Ensure search-based `cancel` and `delete` never plan or process more keys than the remaining limit for that page.
- Let removed `--count` fail through Cobra's unknown flag handling for affected command paths; do not add deprecated aliases.
- Update README examples and regenerate CLI reference docs after command help/examples change.

## Phase 2: Task Planning Approach

- Start with shared flag registration and validation to establish the new public contract.
- Add tests around removed `--count`, invalid `--limit`, `--limit` with `--key`, and `--limit` with `--total` before adding limit truncation behavior.
- Implement limit handling in the shared get/search-page action helpers so all affected commands use one path.
- Add command regression coverage for each render/confirmation mode required by the issue.
- Finish with docs updates, generated docs refresh, and repository validation.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design stops before extra pages are fetched or extra process instances are acted on once the limit is reached.
- **CLI-First, Script-Safe Interfaces**: Still passes. The new flags are explicit, scriptable, and validated through established command behavior.
- **Tests and Validation Are Mandatory**: Still passes with required command coverage plus final `make test`.
- **Documentation Matches User Behavior**: Still passes with README updates and generated CLI docs refresh.
- **Small, Compatible, Repository-Native Changes**: Still passes. The work stays inside the existing command-layer paging helpers and docs generation path.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
