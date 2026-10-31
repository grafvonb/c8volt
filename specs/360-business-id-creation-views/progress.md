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

---
## Iteration 3 - 2026-10-31 13:30
**Work Unit**: US2 — T003 process-instance inspection and observed mutation plans complete; US2 partial
**Tasks Completed**:
- [x] T003: Preserve observed process business IDs across inspection/enrichment/walks, mutation plans and definition-deletion impact, with isolated execution/logger tests and generated documentation.
**Tasks Remaining in Work Unit**: T003 complete; 3 tasks remain in US2 and overall (T004–T006).
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- c8volt/ops/convert.go
- c8volt/ops/convert_test.go
- c8volt/process/convert.go
- c8volt/process/convert_test.go
- c8volt/process/model.go
- c8volt/resource/business_id_test.go
- c8volt/resource/convert.go
- cmd/cancel_processinstance.go
- cmd/cancel_processinstance_selector_test.go
- cmd/cmd_confirmation_terminal_test.go
- cmd/cmd_views_processinstance.go
- cmd/cmd_views_processinstance_activity.go
- cmd/cmd_views_processinstance_business_id_test.go
- cmd/cmd_views_processinstance_dryrun.go
- cmd/cmd_views_processinstance_resume_tree.go
- cmd/cmd_views_processinstance_suspend_tree.go
- cmd/cmd_views_processinstance_update.go
- cmd/cmd_views_resource.go
- cmd/cmd_views_walk.go
- cmd/delete_processdefinition.go
- cmd/delete_processinstance.go
- cmd/delete_processinstance_selector.go
- cmd/delete_processinstance_selector_test.go
- cmd/get_processinstance.go
- cmd/processinstance_business_id_test.go
- cmd/resume_processinstance.go
- cmd/resume_processinstance_tree_test.go
- cmd/run_processinstance_business_id_test.go
- cmd/suspend_processinstance.go
- cmd/suspend_processinstance_tree_test.go
- cmd/update_processinstance.go
- cmd/update_processinstance_variables.go
- cmd/walk_processinstance.go
- docs/cli/c8volt_cancel_process-instance.md
- docs/cli/c8volt_delete_process-definition.md
- docs/cli/c8volt_delete_process-instance.md
- docs/cli/c8volt_get_process-instance.md
- docs/cli/c8volt_resume_process-instance.md
- docs/cli/c8volt_suspend_process-instance.md
- docs/cli/c8volt_update_process-instance.md
- docs/cli/c8volt_walk_process-instance.md
- docs/index.md
- internal/domain/processinstance_resumption_plan.go
- internal/domain/processinstance_suspension_plan.go
- internal/domain/processinstance_variable.go
- internal/services/processdefinition/delete_test.go
- internal/services/processinstance/dryrun.go
- internal/services/processinstance/observation_business_id_test.go
- internal/services/processinstance/resume_plan.go
- internal/services/processinstance/suspend_plan.go
- internal/services/processinstance/update.go
- internal/services/processinstance/update_validation.go
- specs/360-business-id-creation-views/progress.md
- specs/360-business-id-creation-views/ralph-memory.md
- specs/360-business-id-creation-views/tasks.md
**Learnings**:
- Initial new tests reproduced the missing operations identity mapping and human row column. Also found the separate resource facade mapper and observation-reconstructing variable/suspension/resumption plan models. Added copied optional IDs across those boundaries and froze dry-run identity pointers without new requests or parent fallback.
- Preserved nil and observed empty values, age metadata, requested listener-section presence, independent parent/child facts, exact opaque JSON values, safely quoted human IDs, and existing cancellation-plan field casing. Aggregate delete reconstruction now retains both final-state and cancellation-blocker observations. Key/status-only and local exact-key previews remain read-free.
- Actual command/real-terminal matrices covered get single/list/enriched views, family/flat/parent/children walks, cancel/delete/update/tree/definition previews, successful tree execution, empty discovery, no-wait/automation/auto-confirm controls, quiet+machine, INFO/WARN, plain-time/JSON handlers, configured/inherited destinations and separate streams. No terminal platform skip occurred. Requests are counted and mutations guarded where previews/no-ops apply.
- Corrected existing output defects exposed by these tests: enriched keyed inspection and family walking now honor keys-only; empty key paths emit zero bytes; empty cancel/delete summaries use configured INFO. Two deliberate negative substitutions (empty summary to stdout, impact details directly to stderr) failed assertions. Production sources were restored and final focused tests passed.
- Nonzero BusinessID inventory confirmed in cmd, process/ops/resource facades, core process-instance and process-definition services. All targeted checks and full affected non-command package suites passed. BusinessID service/facade race checks passed; final command race matrix including existing mutation logging, redaction, terminal and definition-failure regressions passed (42.734s).
- Full command regression run (328.731s) completed with only two new-test/help issues: ancestry keys are child-to-parent, and Long/Short help forbids “one key per line.” Corrected the assertion and moved exact formatting guidance to README. Corrected ancestry passed targeted/race checks; final focused observation/help/ownership selection passed (9.535s). All other tests in that broad run passed; the entire command suite was not repeated solely for the final help-prose edit.
- Error/output review: no adapter/facade error classification, partial-result or exit path changed. New identity output uses observed data and configured INFO; empty paths remain mode-aware, shared envelopes decode once with EOF, and no raw error formatting/direct diagnostic print was added. Existing repeated-failure/DEBUG redaction matrices passed. New backend work is limited to copying observed fields; no new mode lifecycle, service calls in renderers, generated-client changes, or discovery requests were introduced.
- Targeted gofmt, git diff --check, sanitized go build ./... and make docs-content passed. Reviewed eight generated CLI references plus the README-derived index. All validation used mocks/test-owned HTTP and terminal fixtures. No live engine was tested; full repository race validation remains T006.
---
