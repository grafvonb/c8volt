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

---
## Iteration 4 - 2026-10-31 13:40
**Work Unit**: US2 — T004 job/user-task observations and frozen completion candidates complete; US2 partial
**Tasks Completed**:
- [x] T004: Preserve observed job and user-task business IDs through native/facade mappings, ordinary/enriched views, scalar update plans and frozen completion candidates/outcomes, with isolated tests and regenerated documentation.
**Tasks Remaining in Work Unit**: T004 complete; 2 tasks remain in US2 and overall (T005–T006).
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- c8volt/job/business_id_test.go
- c8volt/job/client.go
- c8volt/job/convert.go
- c8volt/job/model.go
- c8volt/task/business_id_test.go
- c8volt/task/convert.go
- c8volt/task/model.go
- cmd/cmd_views_job.go
- cmd/cmd_views_usertask.go
- cmd/get_job.go
- cmd/get_job_search.go
- cmd/get_usertask.go
- cmd/job_task_business_id_test.go
- cmd/update_job.go
- cmd/update_job_plan.go
- docs/cli/c8volt_get_job.md
- docs/cli/c8volt_get_user-task.md
- docs/cli/c8volt_update_job.md
- docs/index.md
- internal/domain/job.go
- internal/domain/job_completion.go
- internal/domain/usertask.go
- internal/services/job/completion/business_id_test.go
- internal/services/job/completion/execution_plan.go
- internal/services/job/completion/prepare.go
- internal/services/job/v810/business_id_test.go
- internal/services/job/v810/convert.go
- internal/services/usertask/v810/business_id_test.go
- internal/services/usertask/v810/convert.go
- specs/360-business-id-creation-views/progress.md
- specs/360-business-id-creation-views/ralph-memory.md
- specs/360-business-id-creation-views/tasks.md
**Learnings**:
- The first command regression reproduced missing job identity. Added optional domain/public observations and matching candidate fields, v810 native copies, explicit facade pointer copies, preparation/execution snapshots and quoted view columns. Older adapters retain absence, including when a fixture supplies a native field. No generated-client, version-default, service signature or completion-assignment change.
- Command tests exposed an additional inward task mapper used by effective-variable enrichment and both incremental/nonincremental empty job summaries. Copied identity into `toDomainUserTask`; empty quiet queries now suppress `found: 0` through view-owned formatting. Nonempty query, keys, totals and machine payload contracts remain unchanged.
- New native/facade/plan tests cover nil, observed empty and opaque values, mutation after preparation, both candidate conversion directions and newer execution observations without changing frozen identity. Command fixtures cover all three versions, single/list/enriched reads, scalar retry/no-op/completion previews, bulk previews, accepted/partial completion, one-envelope EOF and original conflict exits. Requests are counted, unrelated reads rejected, and completion bodies are checked for no business-ID assignment.
- Real-terminal execution matrices cover configured/inherited destinations, plain-time/JSON loggers, INFO/WARN and quiet suppression, empty selections under automation/auto-confirm/no-wait, zero-byte empty keys, preserved quiet+machine output, no confirmation/mutation for empty work, and separately captured stdout/stderr. Existing sparse-page, totals, explicit-key, abort/EOF and failure regressions passed.
- Full affected service/facade suites passed: `go test ./internal/services/job/... ./internal/services/usertask/... ./c8volt/job ./c8volt/task -count=1`. Nonzero BusinessID inventory verified in completion service, v810 job/task adapters, both facades and cmd. Targeted command tests passed; broader job/task/complete/scalar/help regression selection passed (38.726s).
- Race selection covering all BusinessID tests plus existing bulk diagnostic/summary/terminal and scalar outcome-error tests passed in all six selected areas (cmd 37.791s). Existing diagnostic matrices exercise both DEBUG entry points and redaction independently of verbose. All validation used sanitized environments, mocks or owned HTTP/terminal fixtures; no live environment was accessed. Full repository race validation remains T006.
- Negative checks used Go overlays only: direct stderr scalar summary output and raw joined completion-error logging each failed the new strict assertions. Production files were never substituted. Reviewed service error → facade conversion → partial envelope/logging → exit: error classes and machine evidence remain unchanged, human failures retain one bounded aggregate ERROR, and new identity output uses INFO. No new raw error formatting or direct diagnostic prints were introduced.
- Command cohesion review found one new final-summary view helper and no new mode lifecycle; backend changes only copy observation data. Targeted gofmt, `git diff --check`, sanitized `go build ./...` and `make docs-content` passed. Reviewed generated job/task/update references and README-derived index. No follow-up task expansion was needed.
---

---
## Iteration 5 - 2026-10-31 13:46
**Work Unit**: US2 T005 — independent listener and slow-analysis observations; completed
**Tasks Completed**:
- [x] T005: Preserve listener and slow-analysis business IDs through domain/facade copies, query views, command acceptance tests, and documentation.
**Tasks Remaining in Work Unit**: 0; T006 remains in US2 for the next iteration.
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- c8volt/element/convert.go
- c8volt/element/listener_business_id_test.go
- c8volt/element/model.go
- c8volt/ops/convert.go
- c8volt/ops/listener_business_id_test.go
- c8volt/ops/model.go
- c8volt/process/convert.go
- c8volt/process/listener_business_id_test.go
- c8volt/process/model.go
- cmd/cmd_views_element.go
- cmd/cmd_views_ops_slow_process_analysis.go
- cmd/cmd_views_processinstance_activity.go
- cmd/get_element.go
- cmd/get_processinstance.go
- cmd/listener_business_id_test.go
- cmd/ops_analyse_slow_process_instances.go
- cmd/walk_processinstance.go
- docs/cli/c8volt_get_element.md
- docs/cli/c8volt_get_process-instance.md
- docs/cli/c8volt_ops_analyse_slow-process-instances.md
- docs/cli/c8volt_walk_process-instance.md
- docs/index.md
- internal/domain/job.go
- internal/domain/job_test.go
- internal/domain/ops_slow_process_analysis.go
- internal/services/ops/slow_process_analysis.go
- internal/services/ops/slow_process_analysis_test.go
- specs/360-business-id-creation-views/progress.md
- specs/360-business-id-creation-views/ralph-memory.md
- specs/360-business-id-creation-views/tasks.md
**Learnings**:
- The initial command regression reproduced a dropped listener ID on Camunda 8.10. Added optional listener IDs to domain and all three facade models with copied pointers, and copied slow-analysis root IDs at service/facade boundaries and the reconstructed human PI row. All three listener row builders reuse the existing safely quoted column. No generated-client or service signature change and no new backend requests.
- Native-to-command matrices cover element, process activity, family walk, default slow summaries and full timelines on 8.8/8.9/8.10. Root `new` and independent opaque, nil, and empty listener IDs coexist. Tests assert exact values/absence, safe single-line quoting, separate streams, one JSON envelope plus EOF, requested-empty/unrequested sections, existing quiet behavior, keys-only and fixed request counts. Missing/empty slow roots remain independent of populated listeners.
- Domain, service-root and facade tests mutate source pointers after conversion and preserve collection pointer semantics. Full affected domain/ops-service/process/element/ops-facade suites passed. The command regression selection recorded in ralph-memory.md passed (11.287s), including existing activity, element, slow-analysis, listener, progress, validation and help checks.
- New BusinessID inventory is nonzero in all six changed areas. Targeted race checks passed in all six packages (cmd 2.147s). Listener/enrichment regressions passed in the two base element/process-instance service packages; nested packages reported no matching tests and are not counted as exercised by that selection. Repository `go build ./...` passed. All validation used the quickstart's sanitized environment and mocks/owned HTTP fixtures; no live environment was accessed. Final full repository race validation remains T006.
- Reviewed the output boundary: changed rows are query results using existing stdout renderers; no mutation summaries, diagnostics, prompts, empty-execution or error handling branches were changed. The existing slow failure path remains service error → facade error conversion → configured command error/envelope helper → classified exit. No new direct diagnostic print, raw error formatting, or logger bypass was introduced; existing query progress/error regressions remain in the selected suite.
- Source metadata and README now describe independent listener/root identity; `make docs-content` regenerated four CLI references and the README-derived index. Reviewed generated differences, ran targeted gofmt and `git diff --check`, and confirmed no new production command declarations or secondary lifecycle were introduced. T006 remains separate because custom operation report representations and full-feature validation are its cohesive outcome; no task expansion was needed.
---
