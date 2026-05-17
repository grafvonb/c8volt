# Progress: Ops Execute Smoke Test

## Current State

- Feature artifacts created from GitHub issue #188.
- Clarification gate completed with no critical ambiguities worth formal questioning.
- Planning artifacts generated with mandatory Ralph implementation context recorded.

## Codebase Patterns

- Production Go source files cannot use filenames ending in `_test.go`; smoke-test implementation files should use non-test suffixes such as `_model.go` or `_service.go` so normal package builds include them.
- `cmd/ops_execute.go` owns the `ops execute` grouping command and should remain a grouping command.
- Existing concrete ops commands use `cmd/ops_execute_retention_policy.go`, `cmd/ops_purge_processinstances_with_incidents.go`, and `cmd/ops_purge_all_processdefinitions.go` as command/report/confirmation patterns.
- `cmd/ops_contract.go` owns shared workflow status and report file helpers.
- `cmd/cmd_views_ops_notices.go` owns shared compact notice rendering.
- `c8volt/ops` is the public facade boundary for ops workflows.
- `internal/services/ops` owns high-level ops workflow orchestration and should delegate resource-specific behavior.
- Process-instance creation concurrency already exists under `internal/services/processinstance/bulk.go`.
- Process-instance traversal already exists under `internal/services/processinstance/walker` and related facade/command paths.
- Process-instance delete planning and deletion behavior should remain the source of truth for cleanup.
- Process-definition deletion behavior lives under `internal/services/processdefinition`.
- Embedded multiple-subprocess fixtures already exist for C87, C88, and C89 under `embedded/processdefinitions/`.

## Ralph Notes

- Implement only the current work unit per iteration.
- Read and apply `specs/ralph-implementation-rules.md` every iteration.
- Commit subjects must use Conventional Commits and end with `#188`.

---
## Iteration 1 - 2026-05-17 08:18:11 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T003: Define internal smoke-test request/result domain models
- [x] T004: Define public ops smoke-test request/result models
- [x] T005: Extend public ops facade API for smoke-test execution
- [x] T006: Extend internal ops service interface for smoke-test execution
- [x] T007: Implement public/internal smoke-test model conversions
- [x] T008: Implement thin public ops facade smoke-test method
- [x] T009: Add foundational ops facade wiring tests for smoke-test execution
- [x] T010: Add foundational internal ops service validation tests for smoke-test request shape
- [x] T011: Mark Phase 2 tasks complete and record validation notes
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- internal/domain/ops_smoke_test_model.go
- c8volt/ops/model.go
- c8volt/ops/api.go
- c8volt/ops/convert.go
- c8volt/ops/client.go
- c8volt/ops/client_test.go
- internal/services/ops/api.go
- internal/services/ops/smoke_test_service.go
- internal/services/ops/smoke_test_test.go
- specs/188-ops-smoke-test/tasks.md
- specs/188-ops-smoke-test/progress.md
**Learnings**:
- The ops facade pattern is thin delegation through `options.MapFacadeOptionsToCallOptions`, `ferrors.FromDomain`, and mechanical domain/public converters in `c8volt/ops/convert.go`.
- Foundational smoke-test service validation now fails before workflow mutation for invalid count, unsupported report format, and report format without a report file.
- Validation run: `go test ./c8volt/ops -run 'TestClientExecuteSmokeTest' -count=1`; `go test ./internal/services/ops -run 'TestExecuteSmokeTest' -count=1`; `go test ./c8volt/ops ./internal/services/ops -count=1`; `go test ./cmd -run '^$' -count=1`.
---
