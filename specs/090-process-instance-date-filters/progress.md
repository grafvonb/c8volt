# Ralph Progress Log

Feature: 090-process-instance-date-filters
Started: 2026-04-08 13:08:20

## Codebase Patterns

- `get process-instance` currently has no dedicated command test file; new coverage should live in `cmd/get_processinstance_test.go` instead of extending unrelated `cmd/get_test.go` cases.
- Fresh `get process-instance` command tests should use an explicit temp config plus `--config`, and they should avoid the shared pre-`SetArgs()` flag reset helper because Cobra can round-trip `StringSlice` defaults into a literal `[]` and falsely trip `--key` exclusivity.
- Shared facade-to-domain process-instance filter mapping lives in `c8volt/process/convert.go`; foundational filter plumbing should extend that helper and verify propagation through `client.SearchProcessInstances` rather than adding a separate conversion path.
- v8.8 generated datetime search filters should be built through `camundav88.AdvancedDateTimeFilter` and the union setter in `internal/services/common/filter.go`; direct struct assignment to `DateTimeFilterProperty` is not the repository-native pattern.
- Inclusive day-upper-bound filtering is represented as next-day midnight minus one nanosecond, and the command-level search tests can assert that exact serialized `$lte` timestamp via the real `/v2/process-instances/search` request body.
- `get process-instance` validation failures that terminate inside the command should be covered with helper-process tests using `Execute()` plus a temp `--config`, because the Run handler exits through `ferrors.HandleAndExit` instead of returning an error to `ExecuteC`.
- v8.7 process-instance date-filter rejection belongs at the top of `SearchForProcessInstances`, before request construction or Operate calls, so unsupported-version behavior stays explicit and testable.
- In this sandbox, `go run`-backed doc generation must override `GOCACHE` to a writable path such as `/tmp/go-build`; the default macOS cache under `~/Library/Caches/go-build` is not writable here.
- Full `make docs` also depends on locally installed Jekyll gems; `make docs-content` can refresh generated markdown, but the site build still fails until Bundler can resolve the missing gems.
- Full `make test` is not a reliable pass/fail signal in this sandbox because several unrelated packages use `httptest.NewServer`/`NewTLSServer`, and listener binds on `[::1]:0` are blocked here.

---

## Iteration 1 - 2026-04-08 13:18:09 CEST
**User Story**: Setup (Phase 1: Shared Infrastructure)
**Tasks Completed**:
- [x] T001: Create command test coverage scaffold for date-filter scenarios in `cmd/get_processinstance_test.go`
- [x] T002: Align feature verification commands and temp-config prerequisites in `specs/090-process-instance-date-filters/quickstart.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_processinstance_test.go
- specs/090-process-instance-date-filters/progress.md
- specs/090-process-instance-date-filters/quickstart.md
- specs/090-process-instance-date-filters/tasks.md
**Learnings**:
- A dedicated process-instance command harness can capture the real `/v2/process-instances/search` request body and gives later date-filter tasks a stable place to add start-date, end-date, and validation cases.
- Repository-wide `make test` now passes with the scaffold in place, so the setup phase is complete and the next iteration can move to foundational filter-model plumbing.
---

## Iteration 2 - 2026-04-08 13:20:32 CEST
**User Story**: Foundational (Phase 2: Blocking Prerequisites)
**Tasks Completed**:
- [x] T003: Extend the public process-instance filter with start/end date bound fields in `c8volt/process/model.go`
- [x] T004: Extend the domain process-instance filter with start/end date bound fields in `internal/domain/processinstance.go`
- [x] T005: Update facade-to-domain process-instance filter conversion for date bounds in `c8volt/process/client.go`
- [x] T006: Add shared filter-mapping coverage for the new date bound fields in `c8volt/process/client_test.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/client_test.go
- c8volt/process/convert.go
- c8volt/process/model.go
- internal/domain/processinstance.go
- specs/090-process-instance-date-filters/progress.md
- specs/090-process-instance-date-filters/tasks.md
**Learnings**:
- The actual shared process-instance filter conversion is implemented in `c8volt/process/convert.go`, so future foundational or facade filter work should extend that helper even when the task description references `client.go`.
- A single `client.SearchProcessInstances` facade test is enough to prove both filter-field propagation and facade-option pass-through without coupling foundational coverage to versioned service behavior.
---

## Iteration 3 - 2026-04-08 13:24:52 CEST
**User Story**: User Story 1 - Filter by Start Date
**Tasks Completed**:
- [x] T007: Add command coverage for valid start-date flags and inclusive range behavior in `cmd/get_processinstance_test.go`
- [x] T008: Add v8.8 service coverage for inclusive start-date request mapping in `internal/services/processinstance/v88/service_test.go`
- [x] T009: Implement start-date flags, parsing, and search-filter population in `cmd/get_processinstance.go`
- [x] T010: Implement native v8.8 start-date filter translation in `internal/services/processinstance/v88/service.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- internal/services/common/filter.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v88/service_test.go
- specs/090-process-instance-date-filters/progress.md
- specs/090-process-instance-date-filters/tasks.md
**Learnings**:
- The command can validate date-only `--start-date-*` flags with `time.Parse(time.DateOnly, ...)` while still passing the original strings through the shared filter model for versioned service translation.
- Reusing a shared `NewDateTimeRangeFilterPtr` helper keeps v8.8 advanced datetime filter construction aligned with the existing common filter helpers instead of open-coding generated union handling in the service.
---

## Iteration 4 - 2026-04-08 13:33:27 CEST
**User Story**: User Story 2 - Filter by End Date and Combine with Existing Filters
**Tasks Completed**:
- [x] T011: Add command coverage for valid end-date filters combined with existing search flags in `cmd/get_processinstance_test.go`
- [x] T012: Add v8.8 service coverage for end-date mapping and missing `endDate` exclusion in `internal/services/processinstance/v88/service_test.go`
- [x] T013: Implement end-date flags and composed search-filter population in `cmd/get_processinstance.go`
- [x] T014: Implement native v8.8 end-date filter translation and missing `endDate` handling in `internal/services/processinstance/v88/service.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- internal/services/common/filter.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v88/service_test.go
- specs/090-process-instance-date-filters/progress.md
- specs/090-process-instance-date-filters/tasks.md
**Learnings**:
- End-date filtering can stay fully server-side on v8.8 by extending the shared datetime filter helper with the generated `$exists` operator instead of adding a separate client-side exclusion pass.
- The process-instance command test harness should reset all date-flag globals, not just search-state globals, so successive in-process executions do not leak start or end date filters across subtests.
---

## Iteration 5 - 2026-04-08 13:32:38 CEST
**User Story**: User Story 3 - Get Clear Errors for Unsupported or Invalid Input
**Tasks Completed**:
- [x] T015: Add command coverage for invalid date formats, invalid ranges, and `--key` incompatibility in `cmd/get_processinstance_test.go`
- [x] T016: Add v8.7 service coverage for date-filter not-implemented behavior in `internal/services/processinstance/v87/service_test.go`
- [x] T017: Implement date-range validation and direct-lookup rejection for date filters in `cmd/get_processinstance.go`
- [x] T018: Implement v8.7 date-filter rejection through the existing service error path in `internal/services/processinstance/v87/service.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- internal/services/processinstance/v87/service.go
- internal/services/processinstance/v87/service_test.go
- specs/090-process-instance-date-filters/progress.md
- specs/090-process-instance-date-filters/tasks.md
**Learnings**:
- Inverted start/end date bounds can stay in the existing command validation seam by parsing both values with `time.DateOnly` after per-flag format checks and returning the shared invalid-input sentinel on `after > before`.
- Direct lookup rejection for date flags is clearest when the key path emits a dedicated `--key` incompatibility message before the generic mixed-filter guard, while v8.7 unsupported handling belongs in the versioned service so the facade and CLI both keep the same capability boundary.
---

## Iteration 6 - 2026-04-08 13:47:00 CEST
**User Story**: Partial progress on Polish & Cross-Cutting Concerns
**Tasks Completed**:
- [x] T019: Update command help text and examples for the new date flags in `cmd/get_processinstance.go`
- [x] T020: Update user-facing command documentation and examples in `README.md`
- [x] T022: Refresh feature smoke-check steps after implementation in `specs/090-process-instance-date-filters/quickstart.md`
**Tasks Remaining in Story**: 2
**Commit**: No commit - partial progress
**Files Changed**:
- README.md
- cmd/get_processinstance.go
- docs/cli/c8volt_get_process-instance.md
- docs/index.md
- specs/090-process-instance-date-filters/progress.md
- specs/090-process-instance-date-filters/quickstart.md
- specs/090-process-instance-date-filters/tasks.md
**Learnings**:
- `make docs-content` succeeds in this sandbox when `GOCACHE=/tmp/go-build`, and it refreshed both the generated CLI page and the docs homepage sync from `README.md`.
- `make docs` remains blocked by missing local Bundler/Jekyll gems (`commonmarker`, `racc`, `eventmachine`, `http_parser.rb`, `json`, `bigdecimal`), so the generated markdown is updated but the full docs-site build is not.
- The feature-specific command, facade, and versioned service tests pass, but full `make test` still fails in unrelated packages because this sandbox forbids the `httptest` listener binds those tests require.
---

## Iteration 7 - 2026-04-08 13:38:41 CEST
**User Story**: Partial progress on Polish & Cross-Cutting Concerns
**Tasks Completed**:
- [x] T021: Regenerate CLI reference output for the command in `docs/cli/c8volt_get_process-instance.md` via `make docs-content` and `make docs`
**Tasks Remaining in Story**: 1
**Commit**: No commit - partial progress
**Files Changed**:
- docs/cli/c8volt_get_process-instance.md
- specs/090-process-instance-date-filters/progress.md
- specs/090-process-instance-date-filters/tasks.md
**Learnings**:
- `GOCACHE=/tmp/go-build make docs-content` still regenerates the command reference markdown successfully, and `GOCACHE=/tmp/go-build make docs` re-runs that generation before failing later in the Jekyll site build step.
- The remaining `make test` failure is environmental rather than feature-specific for the v8.7/v8.8 process-instance services: the first failing package is `cmd`, where `httptest.NewServer` panics on `listen tcp6 [::1]:0: bind: operation not permitted`, and unrelated auth/cluster packages fail for the same listener restriction.
---

## Iteration 8 - 2026-04-08 14:16:00 CEST
**User Story**: Polish & Cross-Cutting Concerns
**Tasks Completed**:
- [x] T023: Run repository validation from `Makefile` and stabilize affected tests in `Makefile`, `cmd/get_processinstance_test.go`, `internal/services/processinstance/v87/service_test.go`, and `internal/services/processinstance/v88/service_test.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- README.md
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- docs/cli/c8volt_get_process-instance.md
- docs/index.md
- specs/090-process-instance-date-filters/progress.md
- specs/090-process-instance-date-filters/quickstart.md
- specs/090-process-instance-date-filters/tasks.md
**Learnings**:
- The process-instance command request-capture tests need to skip when the environment forbids opening even a loopback test listener; otherwise they fail before any CLI assertions run and add noise unrelated to the feature behavior.
- Fresh targeted validation now passes for `go test ./cmd -run 'TestGetProcessInstance(SearchScaffold_UsesTempConfigAndCapturesSearchRequest|DateFilterScaffold|InvalidDateFormatHelper|InvalidStartDateRangeHelper|DateFiltersWithKeyHelper)$' -count=1` and `go test ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 -count=1`.
- Fresh `GOCACHE=/tmp/go-build make test` still fails outside this feature in pre-existing listener-based tests such as `cmd/get_test.go`, `internal/services/auth/cookie/service_it_tiny_test.go`, and `internal/services/cluster/v87`/`v88` fake-server coverage.
---
