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
- `cmd/get_processinstance_variable_filter.go` owns CLI variable-filter grammar; it preserves top-level comma semantics, quoted values, arrays, and `$notin` alias normalization before facade mapping.
- Variable-search flags use `StringArray`-style Cobra binding so pflag does not split comma-sensitive values before the parser.
- Shared variable filter intent flows `cmd` -> `c8volt/process.ProcessInstanceFilter.VariableFilters` -> `internal/domain.ProcessInstanceFilter.VariableFilters`; versioned request builders should consume the domain clauses later.
- Native process-instance variable request mapping lives in focused `internal/services/processinstance/v88/variable_filter.go` and `internal/services/processinstance/v89/variable_filter.go`; later operator stories should extend those switch statements rather than adding command-layer request code.
- `--var` equality uses the same parser path as advanced `--var` clauses: `name=value` normalizes to `$eq` while preserving serialized quote characters and comma-containing values.
- v8.8 and v8.9 native equality mapping should set `AdvancedStringFilter.Eq` directly from the preserved domain clause value; command, facade, and service layers must not parse or reserialize that value.
- `--var-like` uses the same parser path with a `$like` default operator; wildcard strings, question marks, and escaped wildcard characters are preserved as raw native pattern values.
- v8.8 and v8.9 native like mapping should set `AdvancedStringFilter.Like` directly from the preserved domain clause value.
- Advanced `$in` and `$notIn` clauses are locally validated as JSON string arrays while preserving the original serialized clause text through command, facade, and domain layers.
- v8.8 and v8.9 generated `AdvancedStringFilter.In` and `NotIn` fields require `[]string`, so versioned `variable_filter.go` files parse validated array text at native request construction time.
- `validatePISearchVersionSupport` owns local configured-version gates for search-only flags before selector validation or remote search starts.
- v8.7 process-instance search must reject non-empty `VariableFilters` before building the Operate request body, preserving existing non-variable search behavior.
- v8.8 and v8.9 native variable filters compose with the existing tenant filter in the same process-instance search request body.

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
---
## Iteration 2 - 2026-05-25 22:45:59 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T009: Add failing parser tests for variable clause splitting, quoting, arrays, and operator normalization in a focused `cmd/get_processinstance_variable_filter_test.go`
- [x] T010: Add failing domain filter string and validation tests for variable filters in `internal/domain/processinstance_test.go`
- [x] T011: Add failing process facade mapping tests for variable filters in `c8volt/process/client_test.go`
- [x] T012: Add version-neutral variable filter clause and filter set fields in `internal/domain/processinstance.go`
- [x] T013: Add public process variable filter models and conversions in `c8volt/process/model.go` and `c8volt/process/convert.go`
- [x] T014: Add command parser scaffolding for `--var-exists`, `--var`, and `--var-like` in a focused `cmd/get_processinstance_variable_filter.go`
- [x] T015: Wire parsed variable filters into `populatePISearchFilterOpts` and search validation in the appropriate `cmd/get_processinstance_*.go` files
- [x] T016: Run targeted compile and parser validation for `cmd`, `c8volt/process`, and `internal/domain`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_processinstance.go
- cmd/get_processinstance_filtering.go
- cmd/get_processinstance_validation.go
- cmd/get_processinstance_variable_filter.go
- cmd/get_processinstance_variable_filter_test.go
- cmd/get_processinstance_test.go
- c8volt/process/model.go
- c8volt/process/convert.go
- c8volt/process/client_test.go
- internal/domain/processinstance.go
- internal/domain/processinstance_test.go
- specs/139-pi-variable-search/tasks.md
- specs/139-pi-variable-search/progress.md
**Learnings**:
- Variable filters should use `StringArray`-style flag binding in future story work so the command parser, not pflag, owns comma-sensitive splitting.
- The foundational parser can validate shape and preserve serialized values without introducing service request construction or generated-client dependencies.
- `hasPISearchFilterFlags` now treats parsed variable-filter globals as search-mode selectors, preserving existing keyed-mode incompatibility checks once flags are registered.
---
---
## Iteration 3 - 2026-05-25 22:52:00 CEST
**User Story**: User Story 1 - Find Instances By Variable Existence
**Tasks Completed**:
- [x] T017: Add command parser and validation tests for `--var-exists customerId` and `--var-exists payload,email` in `cmd/get_processinstance_variable_filter_test.go`
- [x] T018: Add command execution tests for `get pi --var-exists` request flow in `cmd/get_processinstance_test.go`
- [x] T019: Add v8.8 native request construction tests for `$exists=true` filters in `internal/services/processinstance/v88/service_test.go`
- [x] T020: Add v8.9 native request construction tests for `$exists=true` filters in `internal/services/processinstance/v89/service_test.go`
- [x] T021: Register `--var-exists` flag and help text in `cmd/get_processinstance.go`
- [x] T022: Implement `--var-exists` parsing and validation in `cmd/get_processinstance_variable_filter.go`
- [x] T023: Map existence clauses through process facade and domain filters in `c8volt/process/convert.go` and `internal/domain/processinstance.go`
- [x] T024: Implement native existence request mapping for Camunda 8.8 in `internal/services/processinstance/v88/service.go` and `internal/services/processinstance/v88/variable_filter.go`
- [x] T025: Implement native existence request mapping for Camunda 8.9 in `internal/services/processinstance/v89/service.go`, `internal/services/processinstance/v89/convert.go`, and `internal/services/processinstance/v89/variable_filter.go`
- [x] T026: Verify US1 with targeted tests for `cmd`, `c8volt/process`, `internal/domain`, `internal/services/processinstance/v88`, and `internal/services/processinstance/v89`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- cmd/get_processinstance_variable_filter_test.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v88/service_test.go
- internal/services/processinstance/v88/variable_filter.go
- internal/services/processinstance/v89/convert.go
- internal/services/processinstance/v89/service.go
- internal/services/processinstance/v89/service_test.go
- internal/services/processinstance/v89/variable_filter.go
- specs/139-pi-variable-search/tasks.md
- specs/139-pi-variable-search/progress.md
**Learnings**:
- `StringArrayVar` lets `--var-exists payload,email` reach the repository parser intact while repeated flags append independent raw inputs.
- The generated v8.8 and v8.9 variable filter shape can reuse `AdvancedStringFilter{Exists: ...}` inside `VariableValueFilterProperty.Value`.
- v8.9 still needs the local `processInstanceFilter` JSON mirror extended when adding generated-client fields that the service body marshals manually.
---
---
## Iteration 4 - 2026-05-25 22:58:11 CEST
**User Story**: User Story 2 - Find Instances By Variable Equality
**Tasks Completed**:
- [x] T027: Add parser tests for equality shorthand, repeated `--var`, and quoted comma values in `cmd/get_processinstance_variable_filter_test.go`
- [x] T028: Add command execution tests for equality filters in `cmd/get_processinstance_test.go`
- [x] T029: Add v8.8 native request tests for `$eq` equality filters in `internal/services/processinstance/v88/service_test.go`
- [x] T030: Add v8.9 native request tests for `$eq` equality filters in `internal/services/processinstance/v89/service_test.go`
- [x] T031: Register `--var` flag and equality examples in `cmd/get_processinstance.go`
- [x] T032: Implement `name=value` equality shorthand parsing in `cmd/get_processinstance_variable_filter.go`
- [x] T033: Preserve quoted values and comma-containing values in parser logic in `cmd/get_processinstance_variable_filter.go`
- [x] T034: Map equality clauses through process facade and domain filters in `c8volt/process/convert.go` and `internal/domain/processinstance.go`
- [x] T035: Implement native equality request mapping for Camunda 8.8 in `internal/services/processinstance/v88/`
- [x] T036: Implement native equality request mapping for Camunda 8.9 in `internal/services/processinstance/v89/`
- [x] T037: Verify US2 with targeted tests for `cmd`, `c8volt/process`, `internal/domain`, `internal/services/processinstance/v88`, and `internal/services/processinstance/v89`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- cmd/get_processinstance_variable_filter_test.go
- internal/services/processinstance/v88/service_test.go
- internal/services/processinstance/v88/variable_filter.go
- internal/services/processinstance/v89/service_test.go
- internal/services/processinstance/v89/variable_filter.go
- specs/139-pi-variable-search/tasks.md
- specs/139-pi-variable-search/progress.md
**Learnings**:
- `StringArrayVar` registration for `--var` preserves comma-sensitive equality inputs so `splitPIVariableClauses` can distinguish top-level delimiters from commas inside serialized values.
- Equality shorthand was already represented through facade and domain filters by the foundational plumbing; US2 only needed command registration plus native `$eq` request serialization in v8.8 and v8.9.
- Validation passed with targeted parser, command, facade, domain, and versioned service tests, followed by `go test ./cmd ./c8volt/process ./internal/domain ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1`.
---
---
## Iteration 5 - 2026-05-25 23:04:10 CEST
**User Story**: User Story 3 - Search With Like Patterns
**Tasks Completed**:
- [x] T038: Add parser tests for `--var-like`, `*`, `?`, and escaped wildcard values in `cmd/get_processinstance_variable_filter_test.go`
- [x] T039: Add command execution tests for like filters in `cmd/get_processinstance_test.go`
- [x] T040: Add v8.8 native request tests for `$like` filters in `internal/services/processinstance/v88/service_test.go`
- [x] T041: Add v8.9 native request tests for `$like` filters in `internal/services/processinstance/v89/service_test.go`
- [x] T042: Register `--var-like` flag and wildcard examples in `cmd/get_processinstance.go`
- [x] T043: Implement `--var-like` shorthand parsing in `cmd/get_processinstance_variable_filter.go`
- [x] T044: Preserve wildcard and escaped wildcard values in parser logic in `cmd/get_processinstance_variable_filter.go`
- [x] T045: Map like clauses through process facade and domain filters in `c8volt/process/convert.go` and `internal/domain/processinstance.go`
- [x] T046: Implement native like request mapping for Camunda 8.8 in `internal/services/processinstance/v88/`
- [x] T047: Implement native like request mapping for Camunda 8.9 in `internal/services/processinstance/v89/`
- [x] T048: Verify US3 with targeted tests for `cmd`, `internal/services/processinstance/v88`, and `internal/services/processinstance/v89`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- cmd/get_processinstance_variable_filter_test.go
- internal/services/processinstance/v88/service_test.go
- internal/services/processinstance/v88/variable_filter.go
- internal/services/processinstance/v89/service_test.go
- internal/services/processinstance/v89/variable_filter.go
- specs/139-pi-variable-search/tasks.md
- specs/139-pi-variable-search/progress.md
**Learnings**:
- `--var-like` can reuse the existing variable value parser path; the story only needed flag registration plus tests that prove `*`, `?`, and `\*` are preserved.
- Generated `AdvancedStringFilter.Like` fields accept the preserved pattern string directly in v8.8 and v8.9 request builders.
- Validation passed with `GOCACHE=/private/tmp/c8volt-go-build go test ./cmd ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1`.
---
---
## Iteration 6 - 2026-05-25 23:10:44 CEST
**User Story**: User Story 4 - Use Advanced Native Operators
**Tasks Completed**:
- [x] T049: Add parser tests for `$eq`, `$neq`, `$exists`, `$in`, `$notIn`, `$like`, and `$notin` in `cmd/get_processinstance_variable_filter_test.go`
- [x] T050: Add parser tests for invalid operators, malformed booleans, and malformed arrays in `cmd/get_processinstance_variable_filter_test.go`
- [x] T051: Add command execution tests for advanced operators in `cmd/get_processinstance_test.go`
- [x] T052: Add v8.8 native request tests for advanced operators in `internal/services/processinstance/v88/service_test.go`
- [x] T053: Add v8.9 native request tests for advanced operators in `internal/services/processinstance/v89/service_test.go`
- [x] T054: Add advanced operator parsing and `$notin` normalization in `cmd/get_processinstance_variable_filter.go`
- [x] T055: Add local validation for operator value shape in `cmd/get_processinstance_variable_filter.go`
- [x] T056: Extend domain/facade variable filter conversion for advanced operators in `internal/domain/processinstance.go` and `c8volt/process/convert.go`
- [x] T057: Implement native advanced operator mapping for Camunda 8.8 in `internal/services/processinstance/v88/`
- [x] T058: Implement native advanced operator mapping for Camunda 8.9 in `internal/services/processinstance/v89/`
- [x] T059: Verify US4 with targeted parser, command, and versioned service tests
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_processinstance_variable_filter.go
- cmd/get_processinstance_variable_filter_test.go
- cmd/get_processinstance_test.go
- c8volt/process/client_test.go
- internal/services/processinstance/v88/variable_filter.go
- internal/services/processinstance/v88/service_test.go
- internal/services/processinstance/v89/variable_filter.go
- internal/services/processinstance/v89/service_test.go
- specs/139-pi-variable-search/tasks.md
- specs/139-pi-variable-search/progress.md
**Learnings**:
- `$notin` normalization already lived in the command parser; US4 extended coverage to all advanced operators and made malformed JSON arrays fail locally before remote search.
- Native v8.8/v8.9 request construction should parse `$in` and `$notIn` arrays only at the generated-client boundary because the intermediate model intentionally preserves serialized values.
- Validation passed with `go test ./cmd ./c8volt/process ./internal/domain ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1`.
---
---
## Iteration 7 - 2026-05-25 23:15:33 CEST
**User Story**: User Story 5 - Preserve Version And Tenant Contracts
**Tasks Completed**:
- [x] T060: Add 8.7 unsupported command tests for each new variable-search flag in `cmd/get_processinstance_test.go`
- [x] T061: Add v8.7 service unsupported tests for variable filters in `internal/services/processinstance/v87/service_test.go`
- [x] T062: Add tenant preservation tests for variable filters in `cmd/get_processinstance_test.go` or `internal/services/processinstance/v88/service_test.go`
- [x] T063: Add regression tests proving existing 8.7 `get pi` searches without variable filters still behave as before in `cmd/get_processinstance_test.go`
- [x] T064: Add local version support validation for variable-search flags in `cmd/get_processinstance*.go`
- [x] T065: Add explicit 8.7 unsupported handling for domain filters with variable clauses in `internal/services/processinstance/v87/`
- [x] T066: Preserve tenant filter composition with variable filters in `internal/services/processinstance/v88/` and `internal/services/processinstance/v89/`
- [x] T067: Ensure no Operate fallback is used for variable-search paths in `internal/services/processinstance/v87/`, `v88/`, and `v89/`
- [x] T068: Verify US5 with targeted command and versioned service tests
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_processinstance_validation.go
- cmd/get_processinstance_test.go
- internal/services/processinstance/v87/service.go
- internal/services/processinstance/v87/service_test.go
- internal/services/processinstance/v88/service_test.go
- specs/139-pi-variable-search/tasks.md
- specs/139-pi-variable-search/progress.md
**Learnings**:
- Command-level 8.7 rejection prevents variable-search flags from reaching selector validation, paging, or remote search code.
- The v8.7 service-level guard protects facade callers that supply domain variable filters directly, while strict service tests prove no Operate request is made.
- Existing v8.9 variable-filter tests already asserted tenant composition; v8.8 now has matching coverage for tenant plus native variable filters.
- Validation passed with `GOCACHE=/private/tmp/c8volt-go-build go test ./cmd ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1`.
---
