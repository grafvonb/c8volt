# Progress: Native Process Instance Variable Search

## Traceability

- GitHub Issue: #139
- Feature Branch: `139-pi-variable-search`
- Implementation Context: `specs/ralph-implementation-rules.md`

## Codebase Patterns

- `cmd/get_processinstance.go` owns the process-instance command registration, help text, flags, high-level command flow, and command metadata.
- Adjacent `cmd/get_processinstance_*` files own search flag validation, filter population, paging, totals, enrichment, and selector validation.
- `c8volt/process` should stay thin: public filter models map to `internal/domain` filter models and then delegate to `internal/services/processinstance`.
- `internal/domain.ProcessInstanceFilter` is the current version-neutral search filter passed into process-instance services.
- `internal/services/processinstance/v88` and `v89` own generated-client request construction for supported native process-instance search.
- `internal/services/processinstance/v87` already carries explicit unsupported paths for version gaps and must reject the new variable-search flags instead of falling back.
- Generated CLI docs under `docs/cli/` are regenerated with `make docs-content`; do not hand-edit them.

## Architecture Grounding

- Architecture extension installed.
- Architecture memory reused without refresh.
- Relevant boundaries: command contract, facade/domain/service layering, generated-client isolation, version gating, docs generation, and script-safe output.

## Clarification Gate

- No critical ambiguities detected worth formal clarification on 2026-05-25.

## Ralph Discipline

- Each Ralph iteration must implement only one work unit.
- Each iteration must receive `--implementation-context specs/ralph-implementation-rules.md`.
- Do not stage or commit unless validation for the work unit passes.
- Commit subjects must use Conventional Commits and end with `#139`.
