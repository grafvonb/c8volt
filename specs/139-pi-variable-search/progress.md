# Progress: Native Process Instance Variable Search

## Traceability

- GitHub Issue: #139
- Feature Branch: `139-pi-variable-search`
- Implementation Context: `specs/ralph-implementation-rules.md`

## Codebase Patterns

- `cmd/get_processinstance.go` owns the process-instance command registration, help text, flags, high-level command flow, and command metadata.
- Adjacent `cmd/get_processinstance_*` files own search flag validation, filter population, paging, totals, enrichment, and selector validation.
- `populatePISearchFilterOpts` is the command-layer handoff from validated `get pi` flags to `process.ProcessInstanceFilter`; new search flags should enter the facade through that path.
- `hasPISearchFilterFlags` controls keyed-mode incompatibility checks, so new search-only filters must participate there to stay mutually exclusive with explicit `--key` lookups.
- `c8volt/process` should stay thin: public filter models map to `internal/domain` filter models and then delegate to `internal/services/processinstance`.
- `internal/domain.ProcessInstanceFilter` is the current version-neutral search filter passed into process-instance services.
- Domain and facade `ProcessInstanceFilter.String()` methods use `toolx.Append*Field` helpers for stable debug output; variable filters need the same rendering style.
- `internal/services/processinstance/v88` and `v89` own generated-client request construction for supported native process-instance search.
- Generated v8.8 and v8.9 `ProcessInstanceFilter` types already expose `Variables *[]VariableValueFilterProperty`, where each variable clause carries `name` plus a full `StringFilterProperty` value supporting `$eq`, `$neq`, `$exists`, `$in`, `$notIn`, and `$like`.
- `internal/services/processinstance/v87` already carries explicit unsupported paths for version gaps and must reject the new variable-search flags instead of falling back.
- `cmd/update_processinstance_variables.go` parses JSON payloads locally and validates one source flag before backend calls; variable-search parser errors should follow the same `invalidFlagValuef`/local validation style.
- Variable display and update confirmation treat `scopeKey` as direct process-instance scope when matching process-scope variables; docs for search must keep that direct-scope wording.
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

---
## Iteration 1 - 2026-05-25 22:37:13 CEST
**User Story**: Phase 1: Setup (Shared Discovery)
**Tasks Completed**:
- [x] T001: Inspect current `get pi` search flags, validation, and filter population in `cmd/get_processinstance.go` and adjacent `cmd/get_processinstance_*.go`
- [x] T002: Inspect process facade filter mapping in `c8volt/process/api.go`, `c8volt/process/model.go`, `c8volt/process/convert.go`, and `c8volt/process/client.go`
- [x] T003: Inspect current domain process-instance filters in `internal/domain/processinstance.go`
- [x] T004: Inspect process-instance service interfaces and versioned search request construction in `internal/services/processinstance/api.go`, `internal/services/processinstance/v87/`, `internal/services/processinstance/v88/`, and `internal/services/processinstance/v89/`
- [x] T005: Inspect generated Camunda variable/process-instance search request types in `internal/clients/camunda/v88/` and `internal/clients/camunda/v89/`
- [x] T006: Inspect existing variable display and update parsing patterns in `cmd/get_processinstance*.go`, `cmd/update_processinstance*.go`, and `internal/services/processinstance/variables.go`
- [x] T007: Inspect command contract and docs generation expectations in `cmd/command_contract_test.go`, `README.md`, `docsgen/`, and `docs/cli/`
- [x] T008: Record discovered ownership notes in `specs/139-pi-variable-search/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/139-pi-variable-search/tasks.md
- specs/139-pi-variable-search/progress.md
**Learnings**:
- `get pi` search mode builds a public facade filter after selector validation; variable search should be blocked from keyed mode by adding it to the existing filter-flag detection.
- v8.8 uses generated `camundav88.ProcessInstanceFilter` directly, while v8.9 uses a local JSON mirror of the generated filter shape before posting with `SearchProcessInstancesWithBodyWithResponse`.
- Both generated v8.8 and v8.9 process-instance filters include native `variables` clauses, so implementation can stay in the versioned service adapters without editing generated clients.
- Command contract tests already have a focused process-instance variable flag test and docs are regenerated from Cobra metadata with `make docs-content`.
---
