# Tasks: Get User Tasks by Key or Search

**Input**: Design documents from `specs/308-get-user-task/`
**Branch**: `codex/308-get-user-task` | **Issue**: #308
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [CLI contract](contracts/cli.md), [facade/service contract](contracts/facade-service.md), [quickstart.md](quickstart.md).

**Tests**: Required by FR-017, issue acceptance criteria, and the constitution. Write story tests before the corresponding behavior and verify that they expose the missing behavior; complete implementation and relevant checks in the same work unit before committing. Do not commit a knowingly failing test-only increment.

**Organization**: Shared prerequisites, then US1 keyed inspection (P1), US2 discovery/count (P1), US3 interaction/automation (P2), and final validation. US1 includes basic correct rendering to make the MVP usable; US3 completes the combined-mode and terminal contract.

## Format and Path Conventions

Each executable item uses `- [ ] Tnnn [P?] [USn?] description`. Paths are repository-relative. `[P]` identifies disjoint work within an explicitly described ready wave; prerequisites must finish first. A parallel marker is not permission to edit another task's files. Source files named below may be created if absent. No new dependencies, generated-client edits, task mutations, forms, variables, dates, custom sorting, or watch mode.

Read `AGENTS.md` and, for Ralph execution, `specs/ralph-implementation-rules.md`. Keep existing legacy resolver semantics, comments on new/touched declarations and tests, and deterministic options/errors. Run `gofmt` on touched Go files and the closest relevant tests for every implementation unit; select broader validation only for concrete cross-package, concurrency, dependency, or unresolved-regression risk under constitution v2.0.0. Documentation-only work uses diff, consistency, and local-link checks. Do not repeat passing checks solely to commit or mark tasks complete. Record commands, outcomes, and any limits in `specs/308-get-user-task/progress.md`; a skipped check is not a pass.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish the current baseline without initializing another project or changing dependency versions.

- [x] T001 Review feature artifacts, `AGENTS.md`, and `specs/ralph-implementation-rules.md`; initialize `specs/308-get-user-task/progress.md` with issue/branch, layer ownership, native-versus-legacy getter distinction, and validation evidence conventions.
- [x] T002 Run baseline `go test ./internal/services/usertask/... -count=1`, `go test ./c8volt/task -count=1`, and existing user-task resolver command tests selected from `cmd/get_processinstance_test.go`; record exact executed test names and results in `specs/308-get-user-task/progress.md`, resolving baseline failures before relying on them as regressions.

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish shared task models and compatibility checks. Finish this phase before either read workflow.

- [x] T003 Extend `internal/domain/usertask.go` with all common task, query, page, visitor, total, and completion types from `data-model.md`, retaining existing identity fields; enforce "All keys remain strings, avoiding numeric precision loss in JSON", "mutually exclusive offset/cursor positions", total kinds "exact or lower_bound", continuation "has_more/no_more/indeterminate", and completion "exhausted, limit_reached, visitor_stopped"; add model/state validation tests in `internal/domain/usertask_test.go` without changing legacy resolver transitions or behavior.
- [x] T004 Introduce matching public models and mechanical converters in `c8volt/task/model.go` and `c8volt/task/convert.go`, with mapping/JSON/copy tests in `c8volt/task/convert_test.go`: Key is "task identity, required on successful reads"; Name follows "Nullable native name becomes empty string when absent"; Assignee follows "Nullable native value becomes empty string; assignment is not a state"; CandidateUsers and CandidateGroups follow "Copy at facade boundary; empty/absent values omitted"; TenantId follows "Preserve backend metadata, including keyed reads outside discovery tenant"; preserve every field/tag in `data-model.md`, including required `state` and `processInstanceKey`, and enforce "Initialize empty collections with a non-nil empty slice" with payload `{"total":0,"items":[]}` and returned-count `Total int64`.
- [x] T005 Pin legacy resolver behavior in `internal/services/usertask/workflow_test.go` and existing `internal/services/usertask/v87/service_test.go`, `internal/services/usertask/v88/service_test.go`, `internal/services/usertask/v89/service_test.go`, and `internal/services/usertask/v810/service_test.go`: v88/v89 tenant-search/Tasklist fallback, v810 native lookup plus existing tenant/identity checks, existing v87 behavior, and input-order owning-process resolution; run these tests and record the passing foundation in `specs/308-get-user-task/progress.md`.

**Checkpoint**: Shared fields and conversions are tested; existing resolution still works. No generated structs or backend mechanics cross public boundaries.

## Phase 3: User Story 1 — Inspect Known User Tasks (Priority: P1, MVP)

**Goal**: Retrieve one or more known task keys using every accepted input form with strict errors and correct tenant metadata.

**Independent Test**: Across 8.8/8.9/8.10, all aliases and key sources return the same ordered unique collection; a missing key or denied read fails, 8.7 is unsupported for new reads, and legacy resolver outcomes remain unchanged.

### Tests for User Story 1

- [x] T006 [P] [US1] Add failing native-GET contract cases in `internal/services/usertask/v810/native_test.go`, `internal/services/usertask/v89/native_test.go`, and `internal/services/usertask/v88/native_test.go` for complete/nullable fields, authorized cross-tenant keys, no search/Tasklist fallback, 404, denied access, malformed/missing identity payloads, and transport errors; cover v87 native rejection with zero requests in `internal/services/usertask/v87/native_test.go`.
- [x] T007 [P] [US1] Add strict bulk-read cases in `internal/services/usertask/bulk_test.go` for "Explicit key deduplication is stable; worker results remain in first-input order", mixed found/missing failures, cancellation, fail-fast, worker limits/options, and an empty input returning an empty collection without requests.
- [x] T008 [P] [US1] Add facade getter/bulk delegation cases in `c8volt/task/client_test.go` for option propagation, native versus legacy method selection, required identity/error conversion, independent candidate slices, and collection shape for one or multiple keys.
- [x] T009 [P] [US1] Add command subprocess cases in `cmd/get_usertask_test.go` for canonical name/three aliases, `-k`, repeated/comma keys, flag-before-stdin order, implicit stdin and explicit `-`, trimming/blank lines, 16-digit validation of flag and stdin keys, scanner failure, empty explicit-dash errors, extra positional args, missing-key errors, and all keyed filter/limit/total conflicts with zero native reads; capture stdout/stderr separately and pin basic human/JSON/keys output.

### Implementation for User Story 1

- [x] T010 [US1] Add `GetNativeUserTask` to `internal/services/usertask/api.go` and each version's `contract.go`; implement generated direct GET with `common.RequirePayload` and native mapping in `internal/services/usertask/v810/service.go` and `internal/services/usertask/v810/convert.go`, then v89 and v88 equivalents; implement v87 unsupported in `internal/services/usertask/v87/service.go`; update affected API stubs/assertions and keep legacy `GetUserTask`, factory selection, default version, and resolver paths unchanged.
- [x] T011 [US1] Implement strict `GetUserTasks` in `internal/services/usertask/bulk.go` using stable unique keys, `toolx.DetermineNoOfWorkers`, and `toolx/pool.ExecuteSlice`; honor worker/fail-fast/context options and joined errors without returning a successful subset or adding default per-key logs; pass T007.
- [x] T012 [US1] Add public `GetUserTask` and `GetUserTasks` to `c8volt/task/api.go` and thin delegation in `c8volt/task/client.go`, using `foptions` conversion and `ferrors.FromDomain`; update impacted facade stubs without changing the constructor, root embedded API wiring, or resolver methods; pass T008.
- [x] T013 [US1] Create `cmd/cmd_views_usertask.go` with flat rows in contract column order, name-to-element fallback, intentional empty optional cells, returned-count collection envelope, keys-only lines, and `found: N` summary; choose JSON before keys before human and then suppress quiet human output, preserving the one-key collection shape; add focused writer-error/row tests in `cmd/cmd_views_usertask_test.go`.
- [x] T014 [US1] Create `cmd/get_usertask.go` with Cobra registration, canonical name/aliases, key input, reserved search flags and bounds/conflict validation, worker validation, keyed dispatch, and shared error handling; add scoped implicit piped-stdin support in `cmd/get_usertask_input.go` reusing `cmd/cmd_stdin.go` conventions without changing other commands; never read terminal stdin for inferred keys and preserve explicit `-` empty/terminal errors; search dispatch is completed in US2 and must not fake successful empty results meanwhile.
- [x] T015 [US1] Register invalid-input, read-only, shared-contract, and automation annotations in `cmd/get_usertask.go`; verify canonical alias capability metadata in `cmd/command_contract_test.go` and malformed-key error envelopes via `cmd/cmd_stdin_error_envelope_test.go`, including zero native requests and no success payload after partial bulk failure.
- [x] T016 [US1] Complete the keyed command matrix in `cmd/get_usertask_test.go` across 8.8/8.9/8.10 and v87 rejection, including configured tenant versus authorized foreign-tenant keys, denial, explicit `--state all` conflict versus default all, quiet+JSON/keys, returned candidate metadata, and nonempty implicit stdin with search-filter conflicts; ensure all US1 tests pass.
- [x] T017 [US1] Update keyed-read help/examples in `cmd/get_usertask.go`, parent help in `cmd/get.go`, and README keyed usage in `README.md`; run `make docs-content` to generate `docs/cli/c8volt_get_user-task.md` and related references, describing only implemented behavior for an MVP demonstration.
- [x] T018 [US1] Run native/bulk/facade tests and targeted `go test ./cmd -run 'TestGetUserTask|Test.*UserTask|Test.*Stdin.*Envelope' -count=1`, retain existing resolver regressions, verify formatted code and run `make test` for this MVP's shared interface and concurrent bulk changes, reusing still-valid passing evidence, and record evidence in `specs/308-get-user-task/progress.md`.

**Checkpoint**: Known-key inspection is demonstrable without search. This is an internal MVP checkpoint, not completion of issue #308 or permission to ship unfinished search behavior.

## Phase 4: User Story 2 — Discover and Count Matching User Tasks (Priority: P1)

**Goal**: Search with backend filters/tenant scope, retrieve bounded results across sparse pages, and obtain exact counts even when backend totals are capped.

**Independent Test**: A stable fixture with distinct filter values and sparse/capped pages returns the known matching keys within the overall limit and produces the exact full count with no false early completion.

### Tests for User Story 2

- [x] T019 [P] [US2] Add search request/response contract tests in `internal/services/usertask/v810/search_test.go`, `internal/services/usertask/v89/search_test.go`, and `internal/services/usertask/v88/search_test.go` for all eight explicit filters plus effective tenant, AND combinations, v810 selector unions versus v88/v89 scalars, limit/offset/cursor unions, raw counts/cursors/exact versus lower-bound totals, nullable fields, backend errors, and the identical nine-state membership; extend `internal/services/usertask/v87/native_test.go` for unsupported search with no requests.
- [x] T020 [P] [US2] Add traversal/count tests in `internal/services/usertask/search_test.go` for advancing cursor and offset fallback, empty/short intermediate pages, raw progress versus trimmed results, mid-page limits, visitor continue/stop/error, terminal probes, empty terminal after the known capped lower bound, repeated/cyclic cursor, inconsistent metadata, arithmetic overflow, cancellation, exact fast count and capped int64 fallback without collected items; pin "Count mode has no user limit or visitor and uses int64 without accumulating task objects".
- [x] T021 [P] [US2] Add facade search/visitor/count contract tests in `c8volt/task/search_test.go` for request/option conversion, copied item fields, mechanically mapped visitor actions/errors, returned versus matching counts, empty-array preservation, and `ferrors.FromDomain` classification.
- [x] T022 [P] [US2] Add command search/count tests in `cmd/get_usertask_search_test.go` for every filter, default no-filter discovery, all/case-insensitive states, invalid `assigned`, process-selector keys, empty implicit stdin, bounded/sparse discovery, tenant scope, numeric zero/exact/capped totals, and total conflicts with JSON/keys/limit; require no requests for invalid input and exact request counts for successful traversal.

### Implementation for User Story 2

- [x] T023 [US2] Add `SearchUserTasksPage` to `internal/services/usertask/api.go` and all version `contract.go` files, updating affected stubs/assertions; implement the v810 one-request page adapter in `internal/services/usertask/v810/search.go` with existing generated filter unions and payload helpers, effective tenant, raw-count/cursor/total normalization, and no local predicate filtering; v87 returns the established unsupported error in `internal/services/usertask/v87/service.go`.
- [x] T024 [P] [US2] Implement v89 one-page search in `internal/services/usertask/v89/search.go` and native mapping as needed in `internal/services/usertask/v89/convert.go`, using scalar process selectors and exact generated candidate/assignee/tenant filter properties; pass the v89 cases from T019 without changing legacy fallback behavior.
- [x] T025 [P] [US2] Implement v88 one-page search in `internal/services/usertask/v88/search.go` and native mapping as needed in `internal/services/usertask/v88/convert.go`, testing its generated scalar selectors and equality properties explicitly; pass the v88 cases from T019 without changing Tasklist fallback behavior.
- [x] T026 [US2] Implement collected search and visitor traversal in `internal/services/usertask/search.go`: prefer advancing cursors; use offset fallback with requested-size advancement for sparse empty pages and raw-length advancement for nonempty pages; separate limit trimming from raw backend progress; detect cursor cycles/overflow/inconsistent metadata; return typed exhaustion/limit/visitor-stop dispositions and do not terminate on empty/short pages with continuation evidence; pass traversal cases from T020.
- [x] T027 [US2] Implement `SearchUserTasksTotal` in `internal/services/usertask/search.go` using trustworthy exact metadata or service-owned page counting; reset/ignore collection limits, exclude interactive visitors and accumulated task slices, propagate cancellation/failures without numeric success, and avoid endless requests based solely on the global count-cap flag after terminal exhaustion; pass count cases from T020.
- [x] T028 [US2] Add `SearchUserTasks`, `SearchUserTasksPages`, and `SearchUserTasksTotal` to `c8volt/task/api.go`, implement thin delegation in `c8volt/task/client.go`, and extend `c8volt/task/convert.go` for page/visitor/result mapping; keep all advancement/counting/filtering below the facade, update impacted public stubs, and pass T021.
- [x] T029 [US2] Finish query construction and validation in `cmd/get_usertask.go` using all fields from `data-model.md`; quote and enforce BatchSize "Default/max 1000; positive at CLI boundary", Limit "0 means unlimited internally; explicitly supplied CLI limit must be positive", State "Normalized state; empty means unrestricted", "Combine predicates with AND", and "without case-folding assignees or candidates"; validate the state set "ASSIGNING, CANCELED, CANCELING, COMPLETED, COMPLETING, CREATED, CREATING, FAILED, UPDATING" with case-insensitive parsing and all omitted from the backend predicate.
- [x] T030 [US2] Implement search dispatch and the facade visitor integration in `cmd/get_usertask_search.go`, keeping cursor/offset/limit arithmetic and aggregation in services; return continue/stop from supplied metadata, silently continue sparse pages, and route completed results through `cmd/cmd_views_usertask.go`; JSON collects once and total dispatch bypasses prompting and list summaries.
- [x] T031 [US2] Add the numeric total view and completed-empty rendering integration in `cmd/cmd_views_usertask.go`, preserving exact `found: 0\n`, `{"total":0,"items":[]}` inside one successful envelope, zero-byte empty keys output, quiet-human suppression, quiet numeric totals, and output-writer failures; extend `cmd/get_usertask_search_test.go` to prove rendering adds no discovery requests.
- [x] T032 [US2] Complete per-version command coverage in `cmd/get_usertask_search_test.go` for effective tenant/default/override/all-tenant discovery options, all filters, exact/capped counts, unsupported 8.7, backend/malformed payload errors, empty and nonempty results, limit before/at/after page boundaries, and sparse pages; require correct request bodies/counts and no mutation calls.
- [x] T033 [US2] Run `go test ./internal/services/usertask/... -count=1`, `go test ./c8volt/task -count=1`, and targeted search/count command tests from `cmd/get_usertask_search_test.go`; verify v810/v89/v88 and legacy resolver suites, format changes, and record the search/count checkpoint in `specs/308-get-user-task/progress.md`.

**Checkpoint**: Keyed lookup and search/count each work through the command. US3 finishes the complete interactive and automation acceptance matrix before release.

## Phase 5: User Story 3 — Use Results Interactively and in Automation (Priority: P2)

**Goal**: Make all output combinations and interactive paging reliable without corrupting stdout or changing existing resolver behavior.

**Independent Test**: Execute nonempty/empty/error paths in every supported output combination and use real terminal stdin with separately captured stdout/stderr to prove prompt routing, continuation/stop, automation, and request counts.

### Tests for User Story 3

- [x] T034 [P] [US3] Add combined-mode execution tests in `cmd/get_usertask_output_test.go` for keyed/search human rows, missing name/assignee, JSON/keys precedence, quiet, quiet+JSON/keys, automation and auto-confirm combinations, empty outcomes, exact numeric totals including quiet, and verbose/debug/activity stream purity; decode one JSON envelope and require EOF, compare exact human output and zero-byte keys output, and assert no extra reads or mutations.
- [x] T035 [P] [US3] Add real-terminal tests in `cmd/get_usertask_terminal_test.go` using `testx.NewCmdTerminalRunner` for yes/yes, decline, unexpected answer, EOF, configured and inherited stderr destinations, redirected stdout, key lines, prompt-free empty and sparse pages, explicit-dash terminal rejection, and no inferred key-read blocking; verify prompts may occur before a supplied limit but never after it is reached and that JSON/automation/auto-confirm do not prompt.
- [x] T036 [P] [US3] Add command failure/regression cases in `cmd/get_usertask_error_test.go` for read failure after an earlier streamed page, no successful partial JSON or numeric total, caller-visible cancellation/errors, and existing `get pi --has-user-tasks` resolution/tenant/Tasklist fallback outcomes; use subprocess isolation and safe concurrent request collectors.

### Implementation for User Story 3

- [x] T037 [US3] Complete paging policy in `cmd/get_usertask_search.go` using the existing confirmation helper with `cmd.ErrOrStderr()`, contract wording/default-no/EOF behavior, terminal-stdin eligibility despite redirected stdout, automation/auto-confirm and JSON continuation, no prompt on sparse empty/completed/limit-reached pages, and truthful error propagation after earlier streamed rows; keep rendering functions in `cmd/cmd_views_usertask.go` and pass T035.
- [x] T038 [US3] Complete mode and stream handling in `cmd/cmd_views_usertask.go` and its callers in `cmd/get_usertask.go`/`cmd/get_usertask_search.go` to pass T034, selecting machine modes before quiet-human suppression, rendering one final human summary and one collected JSON envelope, retaining requested numeric totals, and restricting functional diagnostics to verbose and HTTP diagnostics to DEBUG independently.
- [x] T039 [US3] Complete error and legacy compatibility integration in `cmd/get_usertask.go`, `c8volt/task/client.go`, and the native service paths as needed to pass T036; retain the existing resolver methods in `internal/services/usertask/workflow.go` and existing version-specific getter behavior without adding fallback calls to native reads.
- [x] T040 [US3] Finish capability and automation assertions in `cmd/command_contract_test.go` for the canonical command/aliases, read-only classification, supported shared outputs and full automation, and verify no out-of-scope options or falsely advertised output combinations are introduced by `cmd/get_usertask.go`.
- [x] T041 [US3] Run `go test ./cmd -run 'Test.*UserTask.*Terminal|TestGetProcessInstanceKeysOnlyPagingTerminal' -count=1` and all new output/error/capability cases in `cmd/get_usertask_output_test.go`, `cmd/get_usertask_error_test.go`, and `cmd/command_contract_test.go`; use supported real-terminal platforms, confirm tests actually executed, and record stream/request-count evidence in `specs/308-get-user-task/progress.md`.
- [x] T042 [US3] Re-run service/facade and existing task-to-process resolver regressions plus race-enabled affected command tests, verify no global-state tests use unsafe parallel execution, and record complete US1/US2/US3 acceptance coverage and any platform limits in `specs/308-get-user-task/progress.md`.

**Checkpoint**: All user stories pass independently observable command scenarios. No successful-empty, numeric-total, or JSON result is derived from a user abort or backend failure.

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Complete user documentation and repository validation; do not add unrelated cleanup or new scope.

- [ ] T043 Update final command/parent help and examples in `cmd/get_usertask.go` and `cmd/get.go`, plus `README.md`, for all input forms, filters, states, version support, tenant behavior, count conflicts, paging, quiet/automation, and unsupported features; run `make docs-content` and inspect generated `docs/cli/c8volt_get_user-task.md` and related docs without hand edits.
- [ ] T044 Audit touched files against `AGENTS.md` and `specs/ralph-implementation-rules.md`: inventory declarations in `cmd/get_usertask.go` and `cmd/get_usertask_search.go`, verify three-or-more mode declarations remain split, keep final formatting in `cmd/cmd_views_usertask.go`, check thin facade/service ownership and comments, confirm generated clients unchanged, and record findings in `specs/308-get-user-task/progress.md`.
- [ ] T045 Execute the deterministic validation matrix in `specs/308-get-user-task/quickstart.md` against fake backends, verify each example/contract maps to tested behavior, update guide commands only if implementation names require it, and record outcomes in `specs/308-get-user-task/progress.md`; live reads remain optional and no mutation setup is required.
- [ ] T046 Verify formatting of touched Go files and run `git diff --check`; review generated docs and reconcile acceptance coverage with recorded passing checks. Run `make test` through `Makefile` for the integrated shared service/facade contracts and concurrent bulk behavior if no still-valid full-suite result covers those changes; do not rerun solely for documentation, acceptance, or commit. Record the reason for broader validation, actual outcomes, and final requirement coverage in `specs/308-get-user-task/progress.md` and update completed checkboxes in `specs/308-get-user-task/tasks.md` without marking skipped checks complete.

## Dependencies & Execution Order

### Phase Dependencies

```text
Setup T001–T002
  -> Foundation T003–T005
     -> US1 tests T006–T009 -> native/bulk/facade -> command -> MVP gate T018
        -> US2 tests T019–T022 -> adapters -> traversal/count -> facade/CLI -> T033
           -> US3 tests T034–T036 -> interaction/output/errors -> T042
              -> Documentation/audit/quickstart/full suite T043–T046
```

US2 shares command registration, models, native task mapping, and facade plumbing with US1, so the recommended implementation order is US1 then US2. US3 intentionally validates both completed read workflows. Independent testing means each story has its own observable acceptance oracle; it does not imply disjoint production files or authorization to run all stories concurrently.

### Within-Story Dependencies

- T006–T009 depend on T005 and can be authored in parallel in their disjoint files. T010 satisfies adapter contracts, T011 consumes the native API, T012 consumes bulk/native reads, and T013–T016 integrate command behavior. T017–T018 close the MVP gate.
- T019–T022 depend on the US1 gate and can be authored in parallel. T023 establishes the shared page API and v810 path; finish required signatures/temporary unsupported stubs atomically so all version assertions compile, replacing supported-version temporary stubs in T024/T025 before the adapter gate. T024/T025 are then disjoint. T026 follows all adapters, T027 extends that traversal, T028 maps it publicly, T029–T032 integrate the command, and T033 verifies the increment.
- T034–T036 depend on T033 and are disjoint test files. T037–T040 are serial because their production/contract changes overlap; T041/T042 verify the final behavior.
- Shared test scaffolding and interface-stub updates belong to the establishing task. If a parallel test task needs a shared helper not yet available, establish it first or remove the parallel marker rather than making concurrent edits.
- Expected red tests may be authored first, but execute the complete relevant implementation/test unit before marking it ready for a commit. Never claim runtime behavior from a compile failure alone.

### Parallel Opportunities and Examples

`[P]` marks only these ready waves; all other tasks follow the listed order.

| Story | Ready prerequisite | Safe parallel example |
| --- | --- | --- |
| US1 | T005 complete | T006 native contracts, T007 bulk tests, T008 facade tests, and T009 command tests use separate files |
| US2 | T018 complete | T019 adapter contracts, T020 traversal tests, T021 facade tests, and T022 command tests use separate files |
| US2 | T023 complete | T024 v89 adapter and T025 v88 adapter use separate version packages |
| US3 | T033 complete | T034 output tests, T035 terminal tests, and T036 failure/regression tests use separate files |

These are implementation scheduling examples, not instructions to start background agents or Ralph during task generation. Do not run global command-state tests with `t.Parallel()` merely because writing their files can be parallelized.

## Requirement Coverage

| Requirements | Primary tasks |
| --- | --- |
| FR-001–FR-003: grammar, input, strict keys | T006–T018 |
| FR-004–FR-008: filters, states, tenant, pages, limits | T019–T026, T028–T030, T032–T033 |
| FR-009: exact count | T020, T022, T027–T028, T031–T033 |
| FR-010: conflicts | T009, T014, T016, T022, T029 |
| FR-011–FR-013: rows, JSON/keys, empty/quiet | T004, T013, T016, T031, T034, T038 |
| FR-014: inherited modes and interaction | T034–T035, T037–T038, T040–T041 |
| FR-015–FR-016: versions, errors, legacy behavior | T005–T006, T010, T016, T019, T023–T025, T032, T036, T039, T042 |
| FR-017: execution/terminal validation | T009, T016, T022, T032, T034–T036, T041–T042, T045–T046 |
| FR-018–FR-019: docs and bounded scope | T017, T040, T043–T046 |

## Implementation Strategy

1. Complete shared setup/models without changing the legacy read path.
2. Deliver US1 as the MVP: native keyed lookup, every key input form, strict errors, correct tenant metadata, basic result modes, and its documentation/tests. Demonstrate with explicit keys; do not present incomplete search as shipped behavior.
3. Deliver US2 with version-tested backend filters, service-owned traversal, sparse-page handling, exact counts, and command-level validation.
4. Complete US3's cross-mode and real-terminal matrix, preserving the existing resolver and stream contracts.
5. Finish docs and confirm race-enabled validation covers the integrated shared contracts and concurrency, reusing still-valid results. Issue #308 is complete only when every phase is complete; use Conventional Commits with a clear scope and issue reference after required validation.

## Notes

- Task generation changes planning artifacts only. No source implementation, tests, commit, or Ralph execution is implied by this file.
- Keep the supplied contracts authoritative: explicit keys are backend-authorized admin input, search uses effective tenant, and total is an exact numeric line with the agreed conflicts.
- Do not use empty intermediate pages, user stops, missing reports, capped totals, or rendering side effects to invent successful completion.
- Preserve field/omission rules verbatim from the data model; do not redesign other resource payloads or the shared envelope.
