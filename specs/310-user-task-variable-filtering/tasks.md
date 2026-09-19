# Tasks: Filter User Tasks by Local Variables

**Input**: Design documents from `specs/310-user-task-variable-filtering/`
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [CLI contract](contracts/cli.md), [facade/service contract](contracts/facade-service.md), [quickstart.md](quickstart.md)
**Branch**: `codex/310-user-task-variable-filtering`

**Tests**: Required by the feature's acceptance scenarios and plan. Add focused regression tests before the corresponding implementation. New behavior should initially fail; tests that characterize preserved behavior may already pass. Do not force production changes solely to make a compatibility task look like implementation.

**Organization**: Tasks follow the three user stories. `[P]` marks independent file work within the explicitly stated dependency wave; it does not authorize ignoring earlier prerequisites. Paths are repository-relative. Each task includes its owned files; reuse existing helpers rather than adding parallel frameworks.

## Phase 1: Setup (Shared Context)

**Purpose**: Establish the existing behavior as the baseline, without new dependencies or scaffolding.

- [x] T001 Review `specs/310-user-task-variable-filtering/plan.md`, `specs/310-user-task-variable-filtering/contracts/cli.md`, `specs/310-user-task-variable-filtering/contracts/facade-service.md`, `AGENTS.md`, and `.specify/memory/constitution.md`; confirm the current branch and record implementation validation results in `specs/310-user-task-variable-filtering/quickstart.md` as work proceeds. For Ralph execution also read `specs/ralph-implementation-rules.md`; stop on a genuine conflict. Do not change grammar, generated clients, or unrelated branch work.

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Carry the existing predicate shape through task search inputs used by every story.

- [x] T002 Add public model/conversion regression cases in `c8volt/task/model_test.go`, `c8volt/task/convert_test.go`, and `c8volt/task/search_test.go` for the additive filter field, aliases, ordered clauses, copied existence pointers including false, empty filters, unchanged returned models, and propagation through collected, visitor, and total facade calls with existing options/errors.
- [x] T003 Add `VariableFilterOperator`, `VariableFilterClause`, and `VariableFilterSet` aliases and six operator constants in `c8volt/task/model.go`, add `SearchRequest.VariableFilters` with `json:"variableFilters,omitempty"`, and add the existing domain set to `internal/domain/usertask.go`. Retain the exact data-model constraints: Name — "Required nonblank local variable name; existing parser trimming rules apply"; Operator — "Canonical existing operator; CLI `$notin` normalizes to `$notIn`"; Value — "Existing serialized value or wildcard text, without new coercion"; Exists — "Explicit true or false for existence; nil is invalid for that operator"; Source — "Existing originating flag label for diagnostics"; Clauses — "Ordered predicates, all required; empty means no variable predicate". Reuse existing domain validation; do not promise struct-valued `omitempty` removes the field when users marshal search inputs.
- [x] T004 Map all predicate fields mechanically in `c8volt/task/convert.go`, preserving order and using `toolx.CopyPtr` for optional booleans: "Do not mutate caller-owned slices or pointers; allocate mapped clauses and copy optional boolean values." Keep facade method signatures and error conversion unchanged; run the closest tests from T002.

**Checkpoint**: Additive search inputs compile, preserve caller ownership, and reach the existing internal service. No CLI flag or native filtering is enabled yet.

## Phase 3: User Story 1 — Find Tasks Using Familiar Variable Filters (Priority: P1, MVP)

**Goal**: Select tasks by local variables using the exact existing `get pi` grammar.

**Independent Test**: Run equality, existence, wildcard, advanced-clause, and ordinary-filter combinations through command execution. Verify exact native `localVariables` requests and expected fixture task identities, all three supported versions, zero filter-related variable reads, and pre-request errors. Reuse PI grammar cases rather than inventing syntax.

### Tests

- [x] T005 [P] [US1] Add parser parity cases in `cmd/get_usertask_variable_filter_test.go` using `cmd/get_processinstance_variable_filter_test.go` as the reference: all six operators, `$notin`, exists true/false, repeated flags, group order, quoted commas, supported arrays, assignment characters, wildcard escapes, malformed clauses, and exact established error conventions. Exercise both wrappers with isolated globals and prove one command's flags cannot affect the other.
- [x] T006 [P] [US1] Extend `internal/services/usertask/v88/search_test.go` with HTTP request assertions for every operator, ordered duplicate clauses, ordinary/tenant selectors, omitted empty filters, serialized null versus string-null, false existence, literal wildcard escaping, and no HTTP on invalid domain clauses or malformed/non-string membership arrays.
- [x] T007 [P] [US1] Extend `internal/services/usertask/v89/search_test.go` with the same local-variable request and invalid-input cases as T006, using the matching version's existing fixture helpers and preserving version-specific ordinary selectors.
- [x] T008 [P] [US1] Extend `internal/services/usertask/v810/search_test.go` with the same local-variable request and invalid-input cases as T006, using the matching version's existing fixture helpers and preserving version-specific ordinary selectors.

### Implementation and integration

- [x] T009 [US1] Extract explicit-input orchestration in `cmd/get_processinstance_variable_filter.go`, retain `parsePIVariableFilters` and its existing lower-level helpers unchanged in behavior, and add the task-owned wrapper in `cmd/get_usertask_variable_filter.go`. Supply independent raw slices, retain exists/var/like group order and diagnostics, and do not assign PI globals, broaden parsing, or rename unrelated helpers. Run the parser parity and existing PI parser cases from T005.
- [x] T010 [P] [US1] Add the small v88 mapper in `internal/services/usertask/v88/variable_filter.go` and attach it in `internal/services/usertask/v88/search.go`; validate the reused domain set and decode membership into string arrays before HTTP, preserve serialized scalar/pattern values and false existence, and omit empty `localVariables`. Follow the existing v88 PI mapper without importing PI services or modifying generated code; pass T006.
- [x] T011 [P] [US1] Add the matching v89 mapper in `internal/services/usertask/v89/variable_filter.go` and attach it in `internal/services/usertask/v89/search.go`, applying the same existing validation/encoding and empty-field rules; pass T007.
- [x] T012 [P] [US1] Add the matching v810 mapper in `internal/services/usertask/v810/variable_filter.go` and attach it in `internal/services/usertask/v810/search.go`, applying the same existing validation/encoding and empty-field rules; pass T008.
- [x] T013 [US1] Add command execution and request-construction cases in `cmd/get_usertask_variable_filter_test.go` and `cmd/get_usertask_test.go` for all aliases, combined selectors, local request placement, malformed/unknown clauses, explicit-key and implicit/explicit stdin-key conflicts, and zero additional task-variable/name-discovery requests. Include native supported-version cases and preserve unfiltered behavior; fixture selection is not proof of real backend missing/null semantics.
- [x] T014 [US1] Register task-owned repeatable `StringArray` flags in `cmd/get_usertask.go`, parse in argument validation, include all three flags in `hasGetUserTaskSearchFlags`, and make `newGetUserTaskSearchRequest` return `(task.SearchRequest, error)` with errors propagated by every caller. Update request-builder call sites in `cmd/get_usertask_search_test.go` and flag reset/setup helpers in `cmd/get_usertask_test.go`; do not cache parsed state globally or suppress errors. Pass T013 while retaining existing stdin handling and search dispatch.
- [x] T015 [US1] Run focused parser, task command, facade, and v88/v89/v810 request tests; add a filtered no-request unsupported-version regression in `internal/services/usertask/v87/native_test.go` and retain existing PI variable-filter tests. Record commands/results and the distinction between request-fixture evidence and live scope evidence in `specs/310-user-task-variable-filtering/quickstart.md`.

**Checkpoint**: The three search flags work end-to-end using native local matching. This is the functional MVP; US2/US3 regression gates are required before shipping the full issue.

## Phase 4: User Story 2 — Bound and Count Filtered Work Reliably (Priority: P1)

**Goal**: Preserve paging, limits, exact totals, error behavior, and script-safe results when filters are supplied.

**Independent Test**: Execute filtered searches across cursor/offset/sparse pages and capped totals, with within-page limits and each existing output mode; verify exact counts, unchanged query predicates, and prompt/result stream separation using real terminal stdin.

### Tests

- [x] T016 [P] [US2] Extend `internal/services/usertask/search_test.go` to assert unchanged predicates on initial/offset/cursor pages, sparse continuation, within-page limits, exact metadata counts, capped-total traversal, visitor stop, cancellation, malformed metadata, and later-page errors; no per-task variable calls or mutation requests may be introduced.
- [x] T017 [P] [US2] Extend `cmd/get_usertask_output_test.go` with filtered nonempty/empty human, JSON, keys-only, quiet, quiet+JSON, quiet+keys, JSON+keys, total, auto-confirm, and automation cases supported by existing validation. Capture stdout/stderr separately, assert exact human/count output, zero-byte empty keys, and decode one JSON envelope followed by EOF; assert request counts and unchanged empty payload shapes.
- [x] T018 [P] [US2] Extend `cmd/get_usertask_terminal_test.go` with filtered real-terminal-stdin cases for configured/inherited stderr, redirected stdout, exact prompt wording/defaults, continue/decline/EOF, one-key-per-line paging, sparse-page continuation, empty prompt-free completion, and unattended modes. Reuse `testx.NewCmdTerminalRunner`; no mocked-terminal or pipe-only substitutes.
- [x] T019 [P] [US2] Extend `cmd/get_usertask_error_test.go` with filtered first/later search failures and existing total/limit/output conflicts, asserting unchanged exit/error conventions and absence of an empty-success envelope or false final summary; reuse key-conflict evidence from T013 rather than duplicating it.

### Integration and validation

- [x] T020 [US2] Verify the query added in US1 flows through the existing `cmd/get_usertask_search.go` and `internal/services/usertask/search.go` unchanged for all T016–T019 scenarios; fix only demonstrated feature integration gaps in these files. Retain service-owned traversal/counting and current prompt/view helpers; if the tests pass, no production edits are required.
- [x] T021 [US2] Run the new filtered traversal/count/output/error tests and `TestGetUserTaskPagingTerminal`, including focused race coverage where appropriate, and record results in `specs/310-user-task-variable-filtering/quickstart.md`; confirm test selectors actually execute the intended cases and no backend failure is classified as a successful empty search.

**Checkpoint**: Filtered bounded searches and scripts preserve the base command's operational contract.

## Phase 5: User Story 3 — Inspect Effective Variables After Local Filtering (Priority: P2)

**Goal**: Preserve independent effective-variable display for tasks selected by local predicates.

**Independent Test**: Compare identical filtered searches with and without `--with-vars`, assert the same selected keys, and count effective-variable reads for eligible selected tasks only. Verify limits, paging stops, excluded output modes, and unchanged display errors/truncation.

### Tests

- [x] T022 [US3] Extend the existing fixtures in `cmd/get_usertask_vars_test.go` to capture local-variable search predicates alongside task-keyed effective-variable reads, exposing deterministic selected tasks with local/parent/shadowed variable examples and request counters; do not implement a second filter evaluator in the fixture or claim mocked data proves native scope semantics.
- [ ] T023 [P] [US3] Add filtered display selection tests in `cmd/get_usertask_search_test.go` using T022's fixture: same selected identities with/without display, within-page limits, sparse pages, stopped paging, and no reads for unselected tasks or duplicate reads after incremental rendering; filters themselves cause zero variable reads.
- [ ] T024 [P] [US3] Extend `cmd/get_usertask_vars_output_test.go` with filtered display tests for effective keys-only, total, empty, quiet human, quiet+JSON, and JSON-over-keys precedence; assert existing retrieval exclusions, Unicode/structured-value limits, truncation labels, full JSON values, and propagated enrichment failures.

### Integration and validation

- [ ] T025 [US3] Verify T023–T024 against `cmd/get_usertask_vars.go` and `cmd/get_usertask_search.go`, retaining the existing selected-task enrichment gate and ordering; fix only demonstrated integration gaps. Do not change `--with-vars` semantics, view schemas, backend scope resolution, or use displayed variables to filter results.
- [ ] T026 [US3] Run filtered display tests plus the existing user-task variable display regressions and record outcomes in `specs/310-user-task-variable-filtering/quickstart.md`, explicitly checking no variable reads without display or in excluded modes, and no mutation requests.

**Checkpoint**: Filtering and inspection compose without a new grammar, new output layout, or extra reads for matching.

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Deliver aligned documentation and proportionate validation without expanding scope.

- [ ] T027 Update help/examples/metadata in `cmd/get_usertask.go`, matching assertions in `cmd/command_contract_test.go`, and `README.md` to cover all six issue workflows, the existing operators/alias/encoding, local-only filtering, and independent effective display. Remove the obsolete statement that variable filtering is unavailable; do not promise client-defined missing-variable semantics for `$exists=false`.
- [ ] T028 Run `make docs-content` after T027 and review generated `docs/cli/c8volt_get_user-task.md` and `docs/index.md` for flag, grammar, example, and scope consistency; do not hand-edit generated CLI pages or accept unrelated generated changes without explanation.
- [ ] T029 Run `gofmt` on touched Go files and the relevant implementation checks from `specs/310-user-task-variable-filtering/quickstart.md`, including targeted race coverage for changed parser/command/facade/service paths and PI regressions. Reuse valid prior results; run `make test` only if actual broader shared-runtime impact or unresolved failures warrant it under `.specify/memory/constitution.md`. Record actual checks and any skipped live backend validation, without creating or mutating live fixtures.
- [ ] T030 Review the completed diff against `specs/310-user-task-variable-filtering/spec.md` and both files in `specs/310-user-task-variable-filtering/contracts/`, run `git diff --check`, and finalize evidence in `specs/310-user-task-variable-filtering/quickstart.md`; confirm no generated client edits, no task result schema changes, no new paging/filter loops, no mutation behavior, and no unrelated parser refactor.

## Dependencies & Execution Order

### Phase dependencies

```text
Setup T001
  → Foundation T002 → T003 → T004
  → US1 T005–T015
       ├→ US2 T016–T021
       └→ US3 T022–T026
  → Polish T027 → T028 → T029 → T030
```

US2 and US3 each require completed US1 selection behavior but can be validated independently of one another. Default execution follows listed priority/order. Parallel work across US2/US3 must not edit `cmd/get_usertask_search.go` simultaneously: serialize integration tasks T020 and T025.

### Within-story dependencies

- **US1 tests**: T005–T008 can be authored concurrently after T004. T009 follows T005. T010/T011/T012 each follow their matching test task (T006/T007/T008) and T004 and can run together; T009 is independent of their files. T013 follows the parser/mapping work; T014 follows T013 and updates flag registration and construction. T005's task wrapper tests may need T014 to pass if they use registered flags. T015 requires all US1 implementation and assertions passing.
- **US2**: T016–T019 can be authored concurrently after US1; T020 integrates their findings, then T021 validates.
- **US3**: T022 defines fixture additions first; T023/T024 can then run concurrently in separate files; T025 integrates, then T026 validates.
- **Polish**: Wait for all story checkpoints. Source docs precede generated docs; final checks follow changes that could invalidate prior evidence.
- Command tests manipulating Cobra globals must not use `t.Parallel`; task authoring parallelism is not permission for concurrent shared-state tests.

## Parallel Execution Examples

### User Story 1

After foundation, author T005's grammar cases and T006/T007/T008's versioned HTTP cases in separate files. Once their prerequisites exist, implement v88 (T010), v89 (T011), and v810 (T012) independently. Keep T014's command registration/request construction under one owner.

### User Story 2

After US1, T016 owns service traversal tests while T017 owns command output tests, T018 owns terminal tests, and T019 owns error tests. Integrate through T020 only after those results are available.

### User Story 3

After T022, T023 owns selection/paging display assertions in `cmd/get_usertask_search_test.go`; T024 owns output/exclusion assertions in `cmd/get_usertask_vars_output_test.go`. Neither edits the shared fixture until these tasks finish.

## Requirement Coverage

| Requirements | Tasks |
| --- | --- |
| FR-001–002: familiar flags and grammar | T005, T009, T013–T015 |
| FR-003–005: local/native matching, AND, no extra reads | T002–T004, T006–T008, T010–T015, T022–T026 |
| FR-006: invalid input and key conflicts | T005–T008, T013–T015, T019 |
| FR-007: paging/limits/counts/errors | T016, T019–T021 |
| FR-008–009: modes/empty results/prompts | T017–T021 |
| FR-010–011: display independence and exclusions | T022–T026 |
| FR-012: versions and unsupported behavior | T006–T008, T010–T012, T015 |
| FR-013: preserved unfiltered/PI behavior | T005, T009, T013–T015, T029 |
| FR-014: help/metadata/docs | T027–T028 |
| FR-015: scope boundary | T001, T025, T030 |

## Implementation Strategy

1. Complete setup and additive predicate plumbing without changing command behavior.
2. Deliver US1 as the functional MVP: operators can use familiar syntax for native local-task selection. Validate before adding more work.
3. Complete US2's operational compatibility checks and US3's optional display checks. Preserve existing production paths when tests prove they already satisfy the contract.
4. Update source and generated documentation, then perform proportionate final checks and review.

The MVP is a demonstration checkpoint, not permission to ship without required output/paging/display regressions. This task-generation command performs document validation only; the executable checks above belong to implementation. Scope remains issue #310, with no automatic implementation or commit implied by generating this file.
