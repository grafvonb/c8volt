# Ralph Progress Log

Feature: 360-business-id-creation-views
Started: 2026-10-31 12:43:38

---


---
## Iteration 1 - 2026-10-31 12:50
**Work Unit**: US1 — T001 public business-ID creation complete; story partial (CLI T002 remains)
**Tasks Completed**:
- [x] T001: Deliver business-ID-aware public single, sequential and counted creation, native mappings, validation and successful partial results.
**Tasks Remaining in Work Unit**: T001 complete; 1 task remains in US1 (T002). Five feature tasks remain overall.
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- c8volt/process/bulk.go
- c8volt/process/business_id_test.go
- c8volt/process/client.go
- c8volt/process/convert.go
- c8volt/process/convert_test.go
- c8volt/process/model.go
- docs/index.md
- internal/domain/business_id.go
- internal/domain/business_id_test.go
- internal/domain/processinstance.go
- internal/services/processinstance/bulk.go
- internal/services/processinstance/bulk_test.go
- internal/services/processinstance/business_id_test.go
- internal/services/processinstance/v810/business_id_test.go
- internal/services/processinstance/v810/convert.go
- internal/services/processinstance/v810/convert_test.go
- internal/services/processinstance/v810/service.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v88/service_test.go
- internal/services/processinstance/v89/service.go
- internal/services/processinstance/v89/service_test.go
- specs/360-business-id-creation-views/progress.md
- specs/360-business-id-creation-views/ralph-memory.md
- specs/360-business-id-creation-views/tasks.md
**Learnings**:
- Initial regression tests failed as expected: sequential errors lost prior successes; counted errors exposed zero-value slots. The validator test first failed because its implementation was absent. All final checks passed.
- Confirmed nonzero BusinessID inventory in every changed area. Targeted domain/service/public creation checks passed; the same service/public selection passed under the race detector. A final public BusinessID race rerun passed after test assertion cleanup.
- Full affected-package tests passed for internal/domain, internal/services/processinstance/... and c8volt/process; sanitized `go build ./...` passed. All dependencies were strict mocks or test-owned HTTP servers; no live engine or integration test was run. Full repository race validation remains T006's gate.
- HTTP tests assert generated request bodies for both selectors, omission/exact values, Unicode boundaries, zero mutation on invalid/unsupported input, response-versus-observation identity, failed waiting, and actual conflict preservation through public calls. Bulk tests cover snapshots, bounded out-of-order completion, stable partial order, fail-fast and cancellation without fabricated entries.
- Error-path review: adapters return original domain/transport errors; collection services retain successful evidence; facades convert both results and errors. No command files, direct diagnostic prints or error-format logging were added. T002 owns remaining command rendering/logger/exit corrections and their complete execution matrix.
- Targeted gofmt and `git diff --check` passed. `make docs-content` passed; generated differences are README-derived public API guidance and build metadata in docs/index.md, with no CLI reference changes.
---

---
## Iteration 2 - 2026-10-31 13:13
**Work Unit**: US1 — T002 CLI business-ID creation complete; US1 complete
**Tasks Completed**:
- [x] T002: Deliver scalar `run pi --bid`, dedicated creation views, successful partial results, original exits and configured diagnostic logging with command tests and documentation.
**Tasks Remaining in Work Unit**: 0 in US1; four feature tasks remain (T003–T006, US2).
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- cmd/cmd_json_assertions_test.go
- cmd/cmd_views_processinstance_run.go
- cmd/ops_explicit_large_work_progress.go
- cmd/run_processinstance.go
- cmd/run_processinstance_business_id_test.go
- cmd/run_processinstance_logging_test.go
- cmd/run_test.go
- docs/cli/c8volt_run_process-instance.md
- docs/index.md
- specs/360-business-id-creation-views/progress.md
- specs/360-business-id-creation-views/ralph-memory.md
- specs/360-business-id-creation-views/tasks.md
**Learnings**:
- Initial new regression failed for the missing `--bid` flag. Final isolated targeted selection `go test ./cmd -run 'TestRunProcessInstance|TestRunHelp|Test.*BusinessID' -count=1` passed, as did its focused race run. BusinessID inventory contains five acceptance suites plus the subprocess helper.
- Full `go test ./cmd -count=1` initially exposed two additional assertions: old quiet output expected a row, and command help forbids literal stream terminology. Updated the quiet assertion, kept stream guidance in README, simplified help, and passed the affected tests plus the full targeted creation selection. The final full command suite passed (400.923s).
- Real terminal stdin with redirected stdout exercised configured/inherited logger destinations, plain-time/JSON log formats, INFO/WARN/quiet, both DEBUG entry points and redaction, partial/zero-success results, exact keys, single-envelope EOF, original conflict/unsupported/invalid-input exits, NoErrCodes and one attempted write on renderer failure. No platform skip occurred in the new terminal matrix.
- Request assertions cover both selectors/aliases, exact opaque and omitted input, empty and over-limit input, a UTF-16 boundary, repeated counted input, unsupported versions before mutation, observed/native/absent IDs, sequential/concurrent partial failures, fail-fast and read counts. Existing selector/worker, tenant, observed-state, no-wait and activity regressions passed.
- Negative mutation checks separately replaced human logging with direct stdout, aggregate ERROR logging with direct stderr, and the bounded message with raw joined errors. Each failed assertions; all production files were restored before final validation.
- Error-path review: adapter errors and successful siblings survive service/facade conversion; command output retains the original class and partial payload; final configured ERROR uses only supported counts and a bounded reason. Unreturned work is not called a backend rejection. Writer failure cannot cause a second JSON envelope or override an operation error. Creation progress clears raw failure detail and suppresses duplicate failure warnings while preserving verbose facts and independent redacted DEBUG transport diagnostics.
- Creation formatting/sorting lives in the new dedicated view file; command changes add scalar input and dispatch only, with no new mode lifecycle or backend orchestration. No service, generated-client, shared-pool or default-version change was needed.
- Targeted gofmt, `git diff --check` and `make docs-content` passed. Generated differences are the creation CLI reference and README-derived index/build metadata. All execution used isolated test-owned fakes; real-engine behavior remains unverified. Full repository race validation remains assigned to T006.
---
