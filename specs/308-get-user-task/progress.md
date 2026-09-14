# Ralph Progress Log

Feature: 308-get-user-task
Started: 2026-09-13 13:20:06

---

## Codebase Patterns

- Issue/branch: GitHub issue #308 on `codex/308-get-user-task`.
- Layer ownership: `cmd` owns CLI input, metadata, prompting, and rendering; `c8volt/task` owns stable public models and thin delegation; `internal/domain` and `internal/services/usertask` own version-neutral state and workflows; version packages own generated-client differences.
- Native-versus-legacy getter: retain legacy `GetUserTask` resolver behavior and add separate native direct-read methods with no Tasklist fallback or discovery-tenant post-filter.
- Validation evidence: record exact commands and outcomes, run closest package tests first, run `gofmt` for touched Go files, and select broader validation by concrete risk under constitution v2.0.0. The integrated feature warrants `make test` for shared contracts and concurrency; documentation-only refreshes use lightweight checks, and a commit alone never triggers a rerun.

---
## Iteration 1 - 2026-09-13 13:28
**Work Unit**: Phase 1 Setup (Shared Infrastructure)
**Tasks Completed**:
- [x] T001: Review feature and repository context and initialize durable progress conventions.
- [x] T002: Establish user-task service, facade, command-resolver, and repository-wide test baselines.
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- specs/308-get-user-task/tasks.md
- specs/308-get-user-task/ralph-memory.md
- specs/308-get-user-task/progress.md
**Learnings**:
- `go test ./internal/services/usertask/... -count=1` passed all usertask packages; `go test ./c8volt/task -count=1` passed with `[no test files]`.
- `go test ./cmd -run 'TestGetProcessInstanceCommand_(HasUserTasks|RejectsHasUserTasks)|TestGetProcessInstanceHelp_DocumentsHasUserTasksLookup' -count=1` passed the existing resolver command selection.
- `make test` (`go test ./... -race -count=1`) passed; the `cmd` package completed in 369.987s.
---

## Planning Refresh - 2026-09-14

**Scope**: Documentation only, after rebasing the two planning/baseline commits onto `develop` at `a9aef2c3`. No implementation task completed in this refresh.

- Updated `AGENTS.md` to select this feature's plan. Aligned spec, plan, tasks, quickstart, and durable Ralph guidance with constitution v2.0.0: targeted checks first, full-suite validation justified by shared contracts/concurrency, and no runtime tests solely for documentation or commits.
- Preserved the original iteration and baseline outcomes above as historical evidence; they do not claim validation of new feature behavior or the rebased runtime. T001–T002 remain complete; resume at T003.
- Reviewed spec, plan, task coverage, data model, research, CLI/service contracts, and quickstart for agreement on native versus legacy reads, supported versions, tenant handling, keyed conflicts, sparse paging, exact counts, output modes, and terminal acceptance. No local blocking inconsistency identified after the guidance refresh; implementation still must prove these contracts.
- Corrected the durable commit guidance to require an explicit #308 reference when automatic issue inference misses the branch prefix.
- Validation: `git diff --check` passed. A read-only Python structural check passed for all 18 local Markdown links, all 46 unique sequential task IDs, unchanged completion flags (only T001/T002), 19 functional requirement IDs, 7 success-criterion IDs, and the active-plan target. Reviewed the documentation diff and remaining validation wording.
- Live issue verification remains pending: `gh issue view 308 --json title,body,state,url` returned HTTP 401 (Bad credentials). The retained requirements were not newly verified against the live issue.
- Runtime tests and CLI documentation generation were not run: this change affects planning guidance only, with no executable code or command metadata changes.

---
## Iteration 1 - 2026-09-14 10:00
**Work Unit**: Phase 2 Foundational (T003 domain user-task models)
**Tasks Completed**:
- [x] T003: Add and validate version-neutral task, query, page, visitor, total, continuation, and completion models.
**Tasks Remaining in Work Unit**: 2 (T004–T005)
**Commit**: This work-unit commit
**Files Changed**:
- internal/domain/usertask.go
- internal/domain/usertask_test.go
- specs/308-get-user-task/tasks.md
- specs/308-get-user-task/ralph-memory.md
- specs/308-get-user-task/progress.md
**Learnings**:
- String-backed task identity fields preserve values beyond JavaScript's exact integer range, and page-position plus closed-enum validation makes later traversal state rejectable before use.
- `go test ./internal/domain -run 'TestUserTask' -count=1`, `go test ./internal/domain -count=1`, and `go test ./internal/services/usertask/... -count=1` passed; `go test ./... -run '^$' -count=1` compiled every package successfully; `git diff --check` passed.
---

---
## Iteration 1 - 2026-09-14 13:30
**Work Unit**: Phase 2 Foundational (T004 public user-task models and converters)
**Tasks Completed**:
- [x] T004: Add matching public task/search/page models and copy-safe mechanical converters.
**Tasks Remaining in Work Unit**: 1 (T005)
**Commit**: This work-unit commit
**Files Changed**:
- c8volt/task/model.go
- c8volt/task/convert.go
- c8volt/task/convert_test.go
- specs/308-get-user-task/tasks.md
- specs/308-get-user-task/ralph-memory.md
- specs/308-get-user-task/progress.md
**Learnings**:
- Required task identity fields and collection fields retain stable JSON names while nullable native strings become omitted empty public values; candidate and result slices are independently owned across the facade boundary.
- `go test ./c8volt/task -count=1`, `go test ./... -run '^$' -count=1`, and `git diff --check` passed; the compile-only repository check confirmed the new public types do not break package consumers.
---

---
## Iteration 2 - 2026-09-14 13:33
**Work Unit**: Phase 2 Foundational (T005 legacy resolver compatibility)
**Tasks Completed**:
- [x] T005: Pin shared and version-specific legacy user-task resolver behavior.
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- internal/services/usertask/workflow_test.go
- internal/services/usertask/v88/service_test.go
- internal/services/usertask/v89/service_test.go
- internal/services/usertask/v810/service_test.go
- specs/308-get-user-task/tasks.md
- specs/308-get-user-task/ralph-memory.md
- specs/308-get-user-task/progress.md
**Learnings**:
- Legacy ownership resolution preserves task input order; v88/v89 Tasklist fallback rejects tenant and returned-key mismatches, while v810 rejects native returned-key mismatches.
- `go test ./internal/services/usertask/... -count=1` and `git diff --check` passed, covering the existing v87 unsupported path and all retained primary/fallback resolver regressions.
---

---
## Iteration 3 - 2026-09-14 13:42
**Work Unit**: US1 Inspect Known User Tasks (native direct-read contract and adapters)
**Tasks Completed**:
- [x] T006: Add native direct-read contract coverage for supported adapters and zero-request V87 rejection.
- [x] T010: Add and implement `GetNativeUserTask` across the version-neutral and versioned service contracts.
**Tasks Remaining in Work Unit**: 11 (T007–T009 and T011–T018)
**Commit**: This work-unit commit
**Files Changed**:
- internal/services/usertask/api.go
- internal/services/usertask/workflow_test.go
- internal/services/usertask/v87/contract.go
- internal/services/usertask/v87/service.go
- internal/services/usertask/v87/native_test.go
- internal/services/usertask/v88/contract.go
- internal/services/usertask/v88/convert.go
- internal/services/usertask/v88/service.go
- internal/services/usertask/v88/service_test.go
- internal/services/usertask/v88/native_test.go
- internal/services/usertask/v89/contract.go
- internal/services/usertask/v89/convert.go
- internal/services/usertask/v89/service.go
- internal/services/usertask/v89/service_test.go
- internal/services/usertask/v89/native_test.go
- internal/services/usertask/v810/contract.go
- internal/services/usertask/v810/convert.go
- internal/services/usertask/v810/service.go
- internal/services/usertask/v810/native_test.go
- specs/308-get-user-task/tasks.md
- specs/308-get-user-task/ralph-memory.md
- specs/308-get-user-task/progress.md
**Learnings**:
- Direct reads must validate key, state, and owning process identity while retaining authorized foreign-tenant metadata; the legacy resolver remains on its existing search/fallback or tenant-checked path.
- `go test ./internal/services/usertask/... -count=1`, `go test ./... -run '^$' -count=1`, and `git diff --check` passed; the first focused run identified and then resolved the expected legacy test-stub interface update.
- The prior work-unit subject was repaired from `feat(ralph): pin legacy user task resolver behavior #308` to `feat(ralph): pin legacy user task resolver behavior` because `issue: auto` cannot infer from the `codex/` branch prefix.
---

---
## Iteration 4 - 2026-09-14 13:47
**Work Unit**: US1 Inspect Known User Tasks (strict ordered native bulk reads)
**Tasks Completed**:
- [x] T007: Add strict bulk-read coverage for ordering, failures, cancellation, fail-fast, workers/options, and empty input.
- [x] T011: Implement strict service-owned native user-task bulk reads.
**Tasks Remaining in Work Unit**: 9 (T008–T009 and T012–T018)
**Commit**: This work-unit commit
**Files Changed**:
- internal/services/usertask/bulk.go
- internal/services/usertask/bulk_test.go
- specs/308-get-user-task/tasks.md
- specs/308-get-user-task/ralph-memory.md
- specs/308-get-user-task/progress.md
**Learnings**:
- The shared pool preserves indexed result order and joins worker failures; the strict workflow must discard those result slots on any error and separately detect a context canceled before work was scheduled.
- `go test ./internal/services/usertask -run '^TestGetUserTasks' -count=10`, `go test -race ./internal/services/usertask/... -count=1`, and `git diff --check` passed.
---

---
## Iteration 5 - 2026-09-14 13:53
**Work Unit**: US1 Inspect Known User Tasks (public facade native and bulk reads)
**Tasks Completed**:
- [x] T008: Add facade getter and bulk delegation coverage for selection, options, mapping, collection shape, slice ownership, and errors.
- [x] T012: Add thin public native getter and strict bulk-read delegation with facade option and error conversion.
**Tasks Remaining in Work Unit**: 7 (T009 and T013–T018)
**Commit**: This work-unit commit
**Files Changed**:
- c8volt/task/api.go
- c8volt/task/client.go
- c8volt/task/client_test.go
- cmd/process_api_stub_test.go
- specs/308-get-user-task/tasks.md
- specs/308-get-user-task/ralph-memory.md
- specs/308-get-user-task/progress.md
**Learnings**:
- The public single getter selects only the native adapter method, while the public bulk getter preserves stable service-owned deduplication and ordering; neither changes legacy resolver delegation.
- `go test ./c8volt/task -count=1`, `go test -race ./c8volt/task -count=1`, repository-wide `go test ./... -run '^$' -count=1`, the targeted existing resolver command tests, and `git diff --check` passed.
- The focused facade slice did not require repeating `make test`; race-enabled facade behavior plus repository-wide interface compilation covered its concrete risk, while the integrated MVP gate remains T018.
---

---
## Iteration 6 - 2026-09-14 14:07
**Work Unit**: US1 Inspect Known User Tasks (keyed CLI input, rendering, metadata, and execution matrix)
**Tasks Completed**:
- [x] T009: Add subprocess coverage for aliases, key sources, strict validation, conflicts, failures, and basic output modes.
- [x] T013: Add mode-aware user-task collection views with stable rows, fallback display, quiet handling, and writer errors.
- [x] T014: Register the keyed command, reserved search controls, scoped implicit stdin, validation, and native bulk dispatch.
- [x] T015: Register and verify invalid-input, read-only, full-contract, automation, alias, and stdin error-envelope metadata.
- [x] T016: Complete supported-version, V87, tenant, denial, candidate, quiet-machine, and partial-failure command coverage.
**Tasks Remaining in Work Unit**: 2 (T017–T018)
**Commit**: This work-unit commit
**Files Changed**:
- cmd/cmd_stdin_error_envelope_test.go
- cmd/cmd_views_usertask.go
- cmd/cmd_views_usertask_test.go
- cmd/command_contract_test.go
- cmd/get_usertask.go
- cmd/get_usertask_input.go
- cmd/get_usertask_test.go
- specs/254-cli-debt-refactor/assessment.md
- specs/308-get-user-task/tasks.md
- specs/308-get-user-task/ralph-memory.md
- specs/308-get-user-task/progress.md
**Learnings**:
- The new command can reuse shared key validation while keeping optional implicit stdin isolated; an empty implicit stream reaches the explicitly unsupported pre-US2 search path instead of fabricating an empty result.
- `go test ./internal/services/usertask/... -count=1`, `go test ./c8volt/task -count=1`, and `go test ./cmd -run 'TestGetUserTask|Test.*UserTask|Test.*Stdin.*Envelope' -count=1` passed the complete retained US1 service/facade/command slice.
- `go test ./cmd -count=1` passed after synchronizing the live command inventory assessment; repository-wide `go test ./... -run '^$' -count=1` and `git diff --check` also passed.
---

---
## Iteration 7 - 2026-09-14 14:27
**Work Unit**: US1 Inspect Known User Tasks (keyed MVP documentation and integrated validation)
**Tasks Completed**:
- [x] T017: Document keyed reads in command and parent help and README, then regenerate CLI references.
- [x] T018: Validate native, bulk, facade, keyed command, legacy resolver, formatting, docs generation, and the full race-enabled repository suite.
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- cmd/completion_test.go
- cmd/get.go
- cmd/get_test.go
- cmd/get_usertask.go
- docs/cli/c8volt_get.md
- docs/cli/c8volt_get_user-task.md
- docs/cli/command-tree.md
- docs/index.md
- docsgen/main_test.go
- specs/254-cli-debt-refactor/assessment.md
- specs/308-get-user-task/tasks.md
- specs/308-get-user-task/ralph-memory.md
- specs/308-get-user-task/progress.md
**Learnings**:
- Keyed-only help now distinguishes implemented reads from reserved US2 search flags, and the parent keeps its compatibility-sensitive short summary while adding user-task discovery in long help and examples.
- Initial `make test` identified stale parent-help and 55-command inventory assertions; focused command/docsgen regressions passed after synchronization, and the repeated `make test` passed with `cmd` completing in 390.615s.
- Targeted `go test ./internal/services/usertask/... -count=1`, `go test ./c8volt/task -count=1`, `go test ./cmd -run 'TestGetUserTask|Test.*UserTask|Test.*Stdin.*Envelope' -count=1`, and the explicit legacy resolver command pattern all passed before the full gate.
---

---
## Iteration 8 - 2026-09-14 14:41
**Work Unit**: US2 Discover and Count Matching User Tasks (native one-page search contracts and adapters)
**Tasks Completed**:
- [x] T019: Add versioned search request/response contract coverage and zero-request V87 rejection.
- [x] T023: Extend service contracts and implement the V810 page adapter plus V87 unsupported behavior.
- [x] T024: Implement the V89 scalar-selector page adapter.
- [x] T025: Implement the V88 scalar-selector page adapter.
**Tasks Remaining in Work Unit**: 11 (T020–T022 and T026–T033)
**Commit**: This work-unit commit
**Files Changed**:
- c8volt/task/client_test.go
- internal/services/usertask/api.go
- internal/services/usertask/bulk_test.go
- internal/services/usertask/v87/contract.go
- internal/services/usertask/v87/native_test.go
- internal/services/usertask/v87/service.go
- internal/services/usertask/v88/contract.go
- internal/services/usertask/v88/search.go
- internal/services/usertask/v88/search_test.go
- internal/services/usertask/v89/contract.go
- internal/services/usertask/v89/search.go
- internal/services/usertask/v89/search_test.go
- internal/services/usertask/v810/contract.go
- internal/services/usertask/v810/search.go
- internal/services/usertask/v810/search_test.go
- internal/services/usertask/v810/service_test.go
- internal/services/usertask/workflow_test.go
- specs/308-get-user-task/tasks.md
- specs/308-get-user-task/ralph-memory.md
- specs/308-get-user-task/progress.md
**Learnings**:
- Generated v8.10 process selectors require equality unions, while v8.8/v8.9 expose scalar selector pointers; the task-state and string predicates remain generated equality unions on all supported versions.
- `go test ./internal/services/usertask/... -count=1`, `go test ./c8volt/task -count=1`, repository-wide compile-only tests, and `git diff --check` passed for the expanded service interface and adapter contracts.
---
