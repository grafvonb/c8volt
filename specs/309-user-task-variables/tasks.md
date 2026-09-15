# Tasks: Display Effective User-Task Variables

**Input**: Design documents in `specs/309-user-task-variables/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [CLI contract](contracts/cli.md), [facade/service contract](contracts/facade-service.md), and [quickstart.md](quickstart.md).

**Tests**: Explicitly required by the feature specification and issue #309. Add the listed tests before their corresponding implementation and demonstrate the expected failure, then pass them with the implementation. Keep coherent red/green changes together; do not commit a broken intermediate interface migration.

**Organization**: Shared model prerequisites precede three user-story increments. US1 includes baseline human/JSON rendering so it is usable independently; US3 adds configurable display limits and completes the compatibility matrix. This file defines future work; no task is marked complete by generation.

## Format: `[ID] [P?] [Story] Description`

- `[P]` identifies disjoint work that may run concurrently after the stated prerequisite barrier. It does not waive dependencies or authorize concurrent tests that mutate global command state.
- `[US1]`, `[US2]`, and `[US3]` map to the specification's stories.
- Paths are repository-relative. New files are named explicitly; existing helpers must be reused before introducing shared infrastructure.
- Follow `AGENTS.md` and the constitution. Ralph execution must additionally read `specs/ralph-implementation-rules.md` and stop on conflicts.
- Do not hand-edit generated clients or CLI docs. No new dependency, variable-filtering feature, mutation, recovery GET workflow, or enrichment framework is planned.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the existing feature context; the Go project is already initialized.

- [x] T001 Confirm branch `codex/309-user-task-variables`, the active paths in `.specify/feature.json` and `AGENTS.md`, and the #308 base command in `cmd/get_usertask.go`; read the linked artifacts and record any blocking divergence in `specs/309-user-task-variables/tasks.md` before implementation, preserving existing uncommitted work and dependencies in `go.mod`.

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish shared records and command fixtures. Finish T001 before this phase; finish T002–T004 before story work.

- [x] T002 [P] Add domain enriched wrappers and `UserTaskVariablePageRequest`/`UserTaskVariablePage` in `internal/domain/usertask.go`, reusing `ProcessInstanceVariable` and exact/lower-bound totals; enforce “nonnegative offset, positive size; no cursor field”, retain `HasContinuationEvidence` and raw count, and document “Item unchanged; variables initialized even when empty” and “Total equals returned item count; initialized empty items; order matches input”.
- [x] T003 [P] Add `UserTaskVariable` as the existing public process-variable alias and enriched int64-total wrappers in `c8volt/task/model.go`; preserve all variable tags and constraints verbatim: name “Backend-selected effective name; unique within each successful task collection”; value “Serialized value as received; empty strings are valid; not shortened for JSON”; variable key “Identity returned by the backend”; process key “Owning process, not a discovery filter”; scope “Actual winning scope; may differ from process-instance key”; tenant “Backend tenant metadata; do not rewrite from discovery settings”; truncation “True when the received value is reported incomplete by the backend”.
- [x] T004 [P] Add feature-specific reusable HTTP fixture support in `cmd/get_usertask_vars_test.go`, building on helpers in `cmd/get_usertask_search_test.go`; support task-keyed effective-variable pages, raw value/truncation payloads, request counters, and injected errors without changing shared global command behavior or creating a generic fixture framework.

**Checkpoint**: Domain/public records and fixture support are ready; no user-visible behavior is enabled yet.

## Phase 3: User Story 1 - Inspect Variables for Known Tasks (Priority: P1) — MVP

**Goal**: Inspect effective variables for single, multiple, and stdin task keys with complete retrieval and trustworthy errors.

**Independent Test**: Execute each keyed form and alias against supported-version fixtures with local/inherited variables, duplicate names, multiple pages, empty variables, and failures. Confirm task order, unique effective names, exact baseline human/JSON output, and no successful partial result.

### Tests for User Story 1

T005–T011 are disjoint test-authoring opportunities after foundation completion. Run each relevant test before its implementation; avoid parallel execution of global-state command tests.

- [ ] T005 [P] [US1] Add v810 native effective-variable contract tests in `internal/services/usertask/v810/variables_test.go` for route, task key, ascending name sort, offset-only page, `truncateValues=false`, raw values and both truncation fields, required metadata/value presence versus valid zero/empty values, preserved scope/tenant, and HTTP/malformed-response errors without fallback or recovery calls.
- [ ] T006 [P] [US1] Add equivalent v89 adapter contract coverage in `internal/services/usertask/v89/variables_test.go`, asserting the matching generated request types, raw decoding, error categories, and zero extra calls rather than assuming v810 tests prove version support.
- [ ] T007 [P] [US1] Add equivalent v88 adapter contract coverage in `internal/services/usertask/v88/variables_test.go`, including backend authorization for selected keys and unchanged actual scope/tenant fields.
- [ ] T008 [P] [US1] Add v87 no-request unsupported-operation coverage in `internal/services/usertask/v87/variables_test.go`, retaining native-read and legacy resolver/fallback regression cases in `internal/services/usertask/v87/native_test.go`.
- [ ] T009 [P] [US1] Add complete pagination/enrichment tests in `internal/services/usertask/variables_test.go`: exact/capped totals, retained highest lower bound, sparse pages including continuation beyond the cap, raw-count offset advancement, terminal empty pages, contradictory metadata, cancellation/overflow, identical duplicate collapse versus conflicting-name failure, stable name/task order, no process-root filtering, empty inputs, and later-page/task failures.
- [ ] T010 [P] [US1] Add facade contract tests in `c8volt/task/variables_test.go` for option mapping, `ferrors` conversion, unchanged task/variable fields, no loops or re-fetches, and the invariants “Item unchanged; variables initialized even when empty” and “Total equals returned item count; initialized empty items; order matches input”, including nil input and `items: []`/`variables: []` JSON shapes.
- [ ] T011 [P] [US1] Add keyed execution tests in `cmd/get_usertask_test.go` using T004 fixtures: all aliases, repeated/comma-separated and implicit/explicit stdin keys, merged/deduplicated order, strict mixed-found/missing input, authorized cross-discovery-tenant keys, denial/disappeared task/later-variable failures, no variables, and baseline human/JSON output with stdout/stderr separate and JSON decoder EOF.

### Implementation for User Story 1

- [ ] T012 [US1] Implement v810 `SearchUserTaskEffectiveVariablesPage` in `internal/services/usertask/v810/variables.go` and extend `internal/services/usertask/v810/contract.go` with the matching generated call; apply shared response/error helpers, offset/name-sort requests and explicit false truncation, and raw decoding where “Required `value` presence is distinguishable from a valid empty string”; validate required total/capped metadata and retain continuation evidence; update affected local client doubles and pass T005.
- [ ] T013 [P] [US1] After T012, implement the explicit v89 counterpart in `internal/services/usertask/v89/variables.go` and `internal/services/usertask/v89/contract.go`, update affected v89 client doubles, preserve raw-field/truncation/metadata semantics, and pass T006.
- [ ] T014 [P] [US1] After T012, implement the explicit v88 counterpart in `internal/services/usertask/v88/variables.go` and `internal/services/usertask/v88/contract.go`, update affected v88 client doubles, preserve raw-field/truncation/metadata semantics, and pass T007.
- [ ] T015 [P] [US1] After T012, add the established unsupported implementation in `internal/services/usertask/v87/variables.go` and signature in `internal/services/usertask/v87/contract.go`, keeping legacy resolver behavior intact and passing T008 without native variable requests.
- [ ] T016 [US1] After T012–T015, extend `internal/services/usertask/api.go` and implement complete offset retrieval plus sequential `EnrichUserTasksWithVariables` in `internal/services/usertask/variables.go`; follow all seven pagination rules in `specs/309-user-task-variables/contracts/facade-service.md`, retaining lower bounds/continuation evidence and checking cancellation/overflow; initialize empties, preserve raw counts, sort/collapse only identical records, reject conflicts, propagate errors, update affected service stubs/assertions, and pass T009.
- [ ] T017 [US1] Add the single public `EnrichUserTasksWithVariables` operation to `c8volt/task/api.go` and `c8volt/task/client.go` with mechanical mappings in `c8volt/task/convert.go`; delegate to T016, map options/errors, preserve empty arrays and returned-count semantics, update affected facade stubs/assertions, and pass T010 without altering constructors or existing lookup/search/resolver APIs.
- [ ] T018 [US1] Extract a narrow explicit-limit variable formatter into `cmd/cmd_views_variable_values.go`, retaining the PI wrapper in `cmd/cmd_views_processinstance_vars.go`; add baseline task `vars:` tree and enriched JSON views in `cmd/cmd_views_usertask_vars.go` using existing tree/row helpers, default unlimited values, backend truncation labels, `items: []`/`variables: []`, no process-age metadata, and propagated writer errors; preserve existing PI formatter tests in `cmd/cmd_views_processinstance_test.go`.
- [ ] T019 [US1] Register `--with-vars` in `cmd/get_usertask.go` and add focused selected-result dispatch in `cmd/get_usertask_vars.go`; retain strict keyed lookup before one enrichment pass, gate absent flag/effective keys-only/count/empty inputs without variable calls, honor JSON precedence and quiet/error semantics, keep the flag outside search-selector conflicts, reset new flag state in command test setup, and pass T011.
- [ ] T020 [US1] Format touched Go files and run focused adapter/service/facade/keyed-command and PI formatter checks from `specs/309-user-task-variables/quickstart.md`; verify T005–T011 pass and record actual commands/outcomes in the execution notes of `specs/309-user-task-variables/tasks.md`, including any limits on validation rather than claiming unchecked success.

**Checkpoint**: Known-task variable inspection is usable as the MVP. Search enrichment and configurable human limits remain for US2/US3; do not present the whole issue as finished or publish full-feature help yet.

## Phase 4: User Story 2 - Inspect Variables for a Bounded Search (Priority: P1)

**Goal**: Enrich exactly the selected search results while preserving limits, tenant rules, paging, and empty/error behavior.

**Independent Test**: Search fixtures with a limit inside a page and real-terminal continue/decline/EOF. Count every variable request and prove excluded tasks and empty pages are not enriched; collected output remains one result and failed retrieval remains failure.

### Tests for User Story 2

T021–T024 may be authored concurrently after US1 completes, using T004 fixtures without concurrent changes to their shared helper file.

- [ ] T021 [P] [US2] Extend `cmd/get_usertask_search_test.go` with enriched filtered/tenant search, partial-page limits, sparse task pages, complete per-task variable pages independent of task limits, and both incremental/collected paths; assert no variables for excluded tasks and no second enrichment during final summary.
- [ ] T022 [P] [US2] Extend `cmd/get_usertask_terminal_test.go` using real `testx.NewCmdTerminalRunner` stdin and separately captured stdout/stderr for unchanged yes/no/EOF prompts, inherited/configured stderr, redirected stdout, limit completion, prompt-free empty completion, auto-confirm/automation, and pure keys paging; assert no variable calls beyond accepted pages or in effective keys-only mode.
- [ ] T023 [P] [US2] Extend `cmd/get_usertask_output_test.go` with empty enriched search across human/JSON/keys/quiet and quiet+machine combinations plus supported auto-confirm/automation variants; assert exact `found: 0` newline, one empty JSON envelope followed by EOF, zero-byte keys, preserved numeric zero count, zero variable/mutation calls, and no synthetic variable entries.
- [ ] T024 [P] [US2] Extend `cmd/get_usertask_error_test.go` for enrichment failure after an earlier streamed page, collected-search failure, and writer failure; verify existing exit/error conventions, no final successful summary/envelope, no empty substitute, and unchanged task-search failures.

### Implementation for User Story 2

- [ ] T025 [US2] Integrate enrichment in `cmd/get_usertask_search.go` only after service trimming of `step.Page.Items`, using `cmd/get_usertask_vars.go` before incremental render/prompt; enrich the final selected collection once for collected modes, skip empty pages, retain summary-only completion without re-fetching, and preserve existing visitor decisions, stderr prompts, filters/tenants, and service-owned traversal while passing T021–T024.
- [ ] T026 [US2] Run focused search/output/error command tests and `TestGetUserTaskPagingTerminal` from `specs/309-user-task-variables/quickstart.md`; verify request counts, stdout/stderr purity, supported-version execution, and unchanged keyed regression behavior, and record evidence in `specs/309-user-task-variables/tasks.md`.

**Checkpoint**: Keyed and search workflows both enrich only selected tasks; errors and interactive control behavior retain the base contract.

## Phase 5: User Story 3 - Read Predictable Human and Machine Results (Priority: P2)

**Goal**: Expose human value limits and prove all formatting, output-mode, truncation, and validation contracts.

**Independent Test**: Run default/zero/positive/invalid limits with Unicode and structured values and backend truncation; compare exact output and request counts across human, JSON, keys-only, count, quiet, and mixed flags. JSON must retain received values.

### Tests for User Story 3

- [ ] T027 [P] [US3] Add display-limit and output-precedence execution tests in `cmd/get_usertask_vars_output_test.go` using existing helpers: default/zero/positive limits, negative and explicit dangling zero/positive errors before requests, JSON+keys precedence, quiet+JSON/keys, quiet-human retrieval failures, count conflicts, absent-flag unchanged output, zero-request excluded modes, and Unicode/object/array/empty/null-serialized values with single-envelope EOF checks.
- [ ] T028 [P] [US3] Add explicit-limit formatter/tree/writer regression cases in `cmd/cmd_views_usertask_vars_test.go` and preserve coverage in `cmd/cmd_views_processinstance_test.go` for exact-boundary/over-limit runes, compaction, all backend/client/both truncation labels, default-unlimited backend-incomplete values, no empty `vars:` subtree, and no shared-global flag coupling.

### Implementation for User Story 3

- [ ] T029 [US3] Register and validate `--var-value-limit` in `cmd/get_usertask.go` with the model constraint “integer, default zero, nonnegative, explicitly supplied only with `--with-vars`”; reject explicit zero without its dependency and negative values before any reads, retain existing count/search/key validation, and reset the flag in command test setup in `cmd/get_usertask_test.go`.
- [ ] T030 [US3] Pass the task display limit explicitly through `cmd/get_usertask_vars.go` and `cmd/cmd_views_usertask_vars.go` into the T018 formatter, enforcing “Positive limits apply after human structured-value compaction and count Unicode runes; ellipsis and labels are additional presentation characters” and “Neither the facade nor service receives this display limit”; leave JSON values and backend truncation flags unchanged and pass T027–T028.
- [ ] T031 [US3] Update source help/examples in `cmd/get_usertask.go` and capability/flag contract expectations in `cmd/command_contract_test.go` for both flags, all four issue examples, JSON-versus-human limits, supported versions, keys/count exclusions, and remaining out-of-scope features; remove the obsolete blanket exclusion of variable display.
- [ ] T032 [US3] Run focused user-task output/flag tests, command metadata tests, and PI variable-format regression tests from `specs/309-user-task-variables/quickstart.md`; confirm the full mode matrix and no changed default PI/task behavior, and record outcomes in `specs/309-user-task-variables/tasks.md`.

**Checkpoint**: All three stories meet their observable contracts; documentation generation and integrated validation remain.

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Complete user documentation and integrated verification after all stories pass.

- [ ] T033 Update `README.md` with effective-variable examples, full-value/default limit behavior, explicit backend truncation, supported versions, and keys/count exclusions; keep `specs/309-user-task-variables/quickstart.md` aligned with final executable examples and actual test names without claiming live checks that were not run.
- [ ] T034 Run `make docs-content` from `Makefile` after T031/T033 and review regenerated `docs/cli/c8volt_get_user-task.md` plus any dependent generated pages for both flags and examples; never hand-edit generated output and do not regenerate unrelated clients.
- [ ] T035 Review the integrated diff against `specs/309-user-task-variables/contracts/cli.md` and `specs/309-user-task-variables/contracts/facade-service.md`: verify backend loops remain internal, mode/view cohesion, no generated-client edits, all interface doubles updated, unchanged legacy resolver/tenant behavior, zero out-of-scope mutations, and full FR-001–FR-014 coverage; fix only related issues and record evidence in `specs/309-user-task-variables/tasks.md`.
- [ ] T036 Format all touched Go files, run `git diff --check`, and execute the integrated `make test` target in `Makefile` because shared facade/service interfaces and the shared formatter changed; complete applicable validation scenarios from `specs/309-user-task-variables/quickstart.md`, record actual results and environment gaps in `specs/309-user-task-variables/tasks.md`, and reuse passing checks until relevant changes invalidate them.

## Dependencies & Execution Order

### Phase and story dependencies

```text
T001
  └─ T002 + T003 + T004 (foundation)
       └─ US1: T005–T020 (known-task MVP)
            └─ US2: T021–T026 (bounded search)
                 └─ US3: T027–T032 (limits and compatibility)
                      └─ T033 → T034 → T035 → T036
```

US2 reuses US1 retrieval and views. US3 reuses both paths to validate the complete output matrix. These are explicit implementation dependencies, not promises that unfinished stories can ship independently. Each completed story has its own executable acceptance checkpoint.

### Detailed implementation barriers

- T002–T004 are independent after T001.
- T005–T011 depend on the foundation; tests define the expected behavior before their corresponding implementation. Test doubles local to those files should avoid concurrent edits to shared helpers.
- T012 follows T005. T013/T014/T015 may run together after T012 and their respective tests; they touch different version directories.
- T016 follows all adapter implementations and T009; T017 follows T016/T010; T018 follows T017 and establishes usable views; T019 follows T018/T011; T020 closes US1.
- T021–T024 follow US1 and are disjoint test-authoring tasks; T025 follows those tests, and T026 closes US2.
- T027/T028 follow US2 and may be authored together; T029 then T030 implement them; T031 follows the completed behavior; T032 closes US3.
- Final checks depend on all stories and documentation being complete. Run relevant focused checks with each implementation; T020/T026/T032 summarize evidence and close remaining coverage rather than blindly repeating successful checks.

### Parallel opportunities

Parallelize file editing only within the listed barriers. Do not run tests that manipulate command globals concurrently. No cross-story implementation concurrency is assumed because the stories share command dispatch and rendering files.

## Parallel Example: User Story 1

After T002–T004, author T005–T011 in their separate test files. After T012 establishes the adapter pattern, implement T013 (v89), T014 (v88), and T015 (v87) concurrently; merge them before the shared API/service work in T016.

## Parallel Example: User Story 2

After T020, author T021 search-selection tests, T022 terminal tests, T023 empty-output tests, and T024 error tests in separate existing files. Keep shared fixture edits in T004 complete first; integrate the behavior in T025 afterward.

## Parallel Example: User Story 3

After T026, author T027 command output-limit tests and T028 formatter/PI regression tests concurrently. Serialize T029–T031 because they share command and view integration responsibilities.

## Implementation Strategy

### MVP first

Complete setup, foundation, and US1 (T001–T020). Demonstrate single/multiple/stdin inspection with all variable pages, correct effective values, successful empty variables, and preserved failures. Baseline JSON/human rendering is included so the MVP does not depend on US3. Do not advertise unimplemented search enrichment or value-limit controls.

### Incremental delivery

Add US2 to prove search selection and real-terminal paging boundaries. Add US3 to expose configurable limits and close the full compatibility matrix. Complete documentation and integrated race validation before declaring issue #309 done. If a slice is released separately, finish the documentation tasks for that delivered scope first; release is not part of task generation.

### Validation discipline

Tests are required by this feature, not by the act of generating tasks. For executable slices, format touched Go files and run the closest relevant checks. Planning-only or documentation-only changes use document/link/diff checks. Do not run runtime suites or CLI generation solely to commit planning artifacts. Preserve request-count assertions, real-terminal coverage, and no-success-after-error checks as acceptance requirements.

## Requirement Coverage

| Requirements | Primary tasks |
| --- | --- |
| FR-001 | T011, T017, T019, T020 |
| FR-002–003 | T005–T009, T012–T016 |
| FR-004 | T011, T021–T022, T025–T026 |
| FR-005 | T019, T021–T023, T027, T030 |
| FR-006 | T018, T028, T030 |
| FR-007 | T027, T029–T030 |
| FR-008 | T003, T010–T011, T018, T023, T027, T030 |
| FR-009 | T005–T007, T012–T014, T027–T030 |
| FR-010 | T009–T011, T016–T019, T024–T025 |
| FR-011 | T019, T022–T027, T030 |
| FR-012 | T005–T008, T012–T015, T020, T026 |
| FR-013 | T031, T033–T034 |
| FR-014 | T001, T019, T031, T035–T036 |

## Execution Notes

Task generation only: implementation and runtime validation have not started. Future execution should record checks, discovered constraints, and justified plan adjustments here without changing the original scope silently.

- 2026-09-15, iteration 1 (T001): Confirmed branch `codex/309-user-task-variables`; `.specify/feature.json` and the `AGENTS.md` active-plan marker both select `specs/309-user-task-variables`; the #308 base command remains in `cmd/get_usertask.go` with keyed lookup, search, aliases, output modes, and the intentional pre-#309 variable exclusion. Read the feature specification, plan, data model, research, quickstart, contracts, #308 specification, and Ralph implementation rules. No blocking divergence was found. Preserved the existing untracked Ralph state files and left `go.mod` unchanged. Validation: `go test ./cmd -run '^TestGetUserTask' -count=1` passed.
