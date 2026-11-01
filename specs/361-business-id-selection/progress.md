# Ralph Progress Log

Feature: 361-business-id-selection
Started: 2026-11-01 12:22:16

---

---
## Iteration 1 - 2026-11-01 12:43
**Work Unit**: Phase 2 Foundational — T001 trustworthy process discovery; complete
**Tasks Completed**:
- [x] T001: Add explicit discovery completion, distinct selected limits, validated raw continuation, partial evidence and reliable counted totals.
**Tasks Remaining in Work Unit**: 0; nine feature tasks remain, starting with T002 (US1).
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- c8volt/process/client.go
- c8volt/process/client_test.go
- c8volt/process/convert.go
- c8volt/process/model.go
- cmd/command_contract_test.go
- cmd/get_processinstance.go
- cmd/get_processinstance_paging_test.go
- cmd/get_processinstance_search_test.go
- docs/cli/c8volt_get_process-instance.md
- docs/index.md
- docsgen/main_test.go
- internal/domain/processinstance.go
- internal/services/processinstance/search.go
- internal/services/processinstance/search_paging.go
- internal/services/processinstance/search_test.go
- specs/361-business-id-selection/progress.md
- specs/361-business-id-selection/ralph-memory.md
- specs/361-business-id-selection/tasks.md
**Learnings**:
- Service-owned `search_paging.go` is shared by collection and total fallback. Public completion mapping is additive; errors retain partial items/counts and original classification. Direct incident lookup now counts the bound after process filtering and preserves prior loaded evidence.
- Initial failing tests reproduced duplicate keys consuming a limit and sparse pages truncating totals. New owner tests cover four completion values, unset/unknown metadata, exact/lower-bound metadata, terminal probes, raw offset/cursor progress, repeated pages/cursors, offset overflow, cancellation, visitor failure, local filters and direct-index failure. Newly added test families were listed before execution.
- Targeted process service/version-adapter and facade checks passed. Query command checks passed for selected output modes, duplicate/sparse pages, existing enrichment/tenant/exclusivity behavior and real-terminal continuation/empty-selection regressions. No platform skips appeared in the focused command run.
- New incomplete-search command tests use real terminal stdin and separately captured streams: configured/inherited logger destinations, plain-time/JSON logger formats, normal/verbose/quiet, exactly one ERROR or one JSON failure envelope followed by EOF, bounded cursor-free reasons, and original exit 1. No new output-boundary raw-error formatting or direct diagnostic writes were introduced. Existing failure flow remains service → facade `ferrors.FromDomain` → command error helper → configured logger/shared envelope → original classified exit.
- Validation used `/tmp/c8volt-361-validation.py` (the quickstart isolation wrapper), owned mocks/HTTP/PTY fixtures only. `make docs-content`, targeted `gofmt` and `git diff --check` passed. Command declarations were reviewed: only help metadata changed in production CLI code; no new lifecycle requires a mode split.
- Broader `make test PKG=./...` ran with race detection and integration tags disabled because shared discovery feeds queries and planners. All runtime cases passed; the run failed two stale wording assertions in `TestCommandCapabilityForCommand_BasicPagedReadContracts` and `TestGeneratedPagedWorkflowDocsDocumentContracts`. Both were updated and passed `go test ./cmd ./docsgen -run 'Test(CommandCapabilityForCommand_BasicPagedReadContracts|GeneratedPagedWorkflowDocsDocumentContracts)$' -race -count=1`. The full suite was not repeated for these assertion-only corrections.
- After the last runtime correction, `go test ./internal/services/processinstance/... ./c8volt/process ./cmd -run 'Test.*(ProcessInstance.*(Search|Paging)|GetProcessInstance|ConfirmationEmptySelectorResults|Search|Selection|Convert|Plan)' -race -count=1` passed, including the command package (309.016s). Targeted packages with no matching tests are not claimed as separate coverage; their full-suite cases ran in the broader check.
- Indeterminate completion cannot use synthetic offset progress as proof of observed matches. Compatible exact count-only metadata does not need rows. Mutable totals can shrink during legacy paged mutations; ordinary query prompts and selected-format JSON remain unchanged. No real-engine validation was performed or claimed.
---

---
## Iteration 2 - 2026-11-01 13:04
**Work Unit**: US1 — native process-instance and user-task business-ID search (T002 complete)
**Tasks Completed**:
- [x] T002: Native exact BID queries, shared capability/input policy, copied filters, direct-incident guards, CLI/output contracts and documentation.
**Tasks Remaining in Work Unit**: 0; eight feature tasks remain, starting with T003 (US2).
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- c8volt/ops/business_id_selection_test.go
- c8volt/ops/convert.go
- c8volt/process/api.go
- c8volt/process/business_id_selection_test.go
- c8volt/process/client.go
- c8volt/process/client_test.go
- c8volt/process/convert.go
- c8volt/process/model.go
- c8volt/resource/client_test.go
- c8volt/task/business_id_selection_test.go
- c8volt/task/convert.go
- c8volt/task/model.go
- cmd/business_id_selection_test.go
- cmd/cmd_views_business_id_search.go
- cmd/get_business_id_filter.go
- cmd/get_processinstance.go
- cmd/get_processinstance_filtering.go
- cmd/get_processinstance_search.go
- cmd/get_processinstance_validation.go
- cmd/get_usertask.go
- cmd/process_api_stub_test.go
- docs/cli/c8volt_get_process-instance.md
- docs/cli/c8volt_get_user-task.md
- docs/index.md
- internal/domain/processinstance.go
- internal/domain/usertask.go
- internal/services/common/business_id_search.go
- internal/services/common/business_id_search_test.go
- internal/services/common/processinstance_filter.go
- internal/services/ops/orphan_purge_test.go
- internal/services/processdefinition/delete_test.go
- internal/services/processinstance/api.go
- internal/services/processinstance/business_id_search_test.go
- internal/services/processinstance/dryrun_test.go
- internal/services/processinstance/search.go
- internal/services/processinstance/search_test.go
- internal/services/processinstance/v810/contract.go
- internal/services/processinstance/v810/convert.go
- internal/services/processinstance/v810/service.go
- internal/services/processinstance/v88/contract.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v89/contract.go
- internal/services/processinstance/v89/service.go
- internal/services/usertask/business_id_search_test.go
- internal/services/usertask/search.go
- internal/services/usertask/v810/search.go
- internal/services/usertask/v88/search.go
- internal/services/usertask/v89/search.go
- specs/361-business-id-selection/progress.md
- specs/361-business-id-selection/ralph-memory.md
- specs/361-business-id-selection/tasks.md
**Learnings**:
- Native scalar BID survives AND predicates, tenant scope, sparse traversal, totals, limits and enrichment. Tests cover omitted/empty/invalid UTF-8, whitespace/Unicode/commas/literal wildcards/numeric text and values beyond creation's length limit. Older adapters reject supplied input before native I/O across page, collection and total entry points.
- Both direct-incident guards are covered. Forcing the library shortcut cannot drop BID or consume the selected limit before a later-page direct-incident match. The service snapshots caller BID before invoking visitors. Process and ops filter round trips and task request conversion own independent pointers and preserve JSON absence.
- Added the thin public no-I/O validator so `get pi` rejects unsupported BID before definition-selector lookups, including otherwise-empty discovery. All implementations/test doubles compile together; custom-interface compatibility is documented.
- Initial red command tests failed on the missing --bid flag. The failure-routing regression exposed existing joined transport text on normal PI enrichment output; the BID path now emits one bounded configured ERROR with available selected counts. Tests assert separate stdout/stderr, configured/inherited writers, plain-time/structured logs, normal/verbose/quiet, redacted DEBUG by flag or config, exactly one error envelope plus EOF, original exit 2 and retained machine error evidence. No new production direct prints or raw error-string output boundaries remain.
- Task BID input validation now runs after command bootstrap and before stdin/native search. This ensures empty/key-conflict JSON failures use exactly one envelope. Tests retain nil-BID/legacy validation behavior and exercise explicit-key/stdin conflicts on both resources.
- Real-terminal tests cover prompt-free empty scopes, normal/JSON/keys/quiet combinations, quiet+machine, automation and auto-confirm. Query result/enrichment identities remain resource-owned; unexpected parent lookups are rejected by the task fixture. Existing paging/confirmation regressions also passed; no platform skips appeared.
- Newly added test families were listed before execution. Focused common/process/task service and process/task/ops facade tests passed. Broader owner regressions passed across all adapters, resource/process-definition and operations consumers. Command `Test.*(BusinessIDSelection|GetProcessInstance|GetUserTask|ProcessInstance.*(Search|Paging)|ConfirmationEmptySelectorResults)` passed (17.118s).
- `make docs-content`, changed-file gofmt verification and `git diff --check` passed. The full `make test PKG=./...` race suite passed. The final task-validation/envelope refinement was independently checked with `go test ./cmd -run 'Test.*(BusinessIDSelection|GetUserTask)' -race -count=1` after the broad run had started; it passed. All commands used the quickstart isolation wrapper, with integration tags and inherited application/helper flags disabled.
- New command declarations are limited to two scalar flag helpers and a view/error helper; existing search runners retain lifecycle ownership. No mutation mode, traversal expansion, or later story was implemented. No live-engine validation was performed or claimed.
---

---
## Iteration 3 - 2026-11-01 13:25
**Work Unit**: US2 — complete BID walk selection and traversal (T003 complete)
**Tasks Completed**:
- [x] T003: Shared collection/unique selection, exhaustive v810 child traversal, BID walk dispatch and validation, retained failure evidence, output contracts and documentation.
**Tasks Remaining in Work Unit**: 0; seven feature tasks remain, starting with T004 (US3).
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- c8volt/process/api.go
- c8volt/process/convert.go
- c8volt/process/model.go
- c8volt/process/selection.go
- c8volt/process/selection_test.go
- c8volt/process/walker.go
- c8volt/resource/client_test.go
- cmd/business_id_selection_test.go
- cmd/cmd_views_walk.go
- cmd/command_contract_test.go
- cmd/completion_test.go
- cmd/process_api_stub_test.go
- cmd/resume_processinstance.go
- cmd/suspend_processinstance.go
- cmd/walk_processinstance.go
- cmd/walk_processinstance_selection.go
- cmd/walk_processinstance_selection_test.go
- cmd/walk_test.go
- docs/cli/c8volt_resume_process-instance.md
- docs/cli/c8volt_suspend_process-instance.md
- docs/cli/c8volt_walk_process-instance.md
- docs/index.md
- docsgen/main_test.go
- internal/domain/processinstance_selection.go
- internal/services/processinstance/children_paging_test.go
- internal/services/processinstance/search.go
- internal/services/processinstance/search_paging.go
- internal/services/processinstance/searchpaging/paging.go
- internal/services/processinstance/selection.go
- internal/services/processinstance/selection_test.go
- internal/services/processinstance/traversal/result.go
- internal/services/processinstance/v810/service.go
- internal/services/processinstance/walker/walker.go
- specs/361-business-id-selection/progress.md
- specs/361-business-id-selection/ralph-memory.md
- specs/361-business-id-selection/tasks.md
**Learnings**:
- Shared selection is an internal process-service function with a thin public API delegate. Criteria, nested observations, distinct keys, completion, tenant evidence and search origin are retained. Exhausted uniqueness differs from bounded collection readiness; ambiguity, late errors and malformed missing keys cannot become successful empty walks. v88/v89 selection rejects supplied BID before search.
- Moved T001 continuation mechanics into `processinstance/searchpaging` to let v810 child lookup reuse them without importing its parent factory package. Child lookup now exhausts all pages; walker/family results preserve incomplete observations instead of discarding them. Owned service and command fixtures include 1001 direct children plus a deeper descendant (1003 total instances), sparse pages, late errors, cyclic/unknown continuation and cancellation.
- Actual command tests use real terminal stdin with separate configured streams. They cover tenant-scoped discovery followed by established key-authorized traversal, parent/children/family/flat/enrichment modes, duplicate versus ambiguous matches, empty human/JSON/keys/quiet combinations, auto-confirm, unsupported versions and selector conflicts. Empty selection performs only discovery and never walks, enriches or prompts. JSON decodes exactly once followed by EOF; failed output writes make one attempt.
- Repeated two-resource enrichment failures pass both configured/inherited logger destinations and plain-time/structured formats, compact normal/verbose/quiet ERROR records, independent DEBUG by flag/configuration with redaction, preserved exit 2 and JSON traversal evidence. Failure flow is service → facade conversion → mode-aware view/error helper → configured logger or one shared envelope → original classified exit. Changed output boundaries were reviewed for direct diagnostic prints and raw error formatting; none were introduced. This read-only story introduces no mutation summaries or confirmation lifecycle.
- Targeted service/adapter/traversal/facade checks passed with `go test ./internal/services/processinstance/... ./c8volt/process -run 'Test.*(Selection|Children|Descendants|Family|Traversal|Search|Walk|Suspend|Resume)' -count=1`. New test names were listed before their focused executions. Final `go test ./internal/services/processinstance ./c8volt/process ./cmd -run '^TestBusinessIDSelection' -race -count=1` passed after the malformed-key guard; command package took 76.214s. The added missing-key actual-command case also passed a focused race run.
- Broader `make test PKG=./...` ran with race detection and integration tags disabled because shared paging/traversal feed mutations and operations. It found a missing resource-facade test stub method, two stale child-limit metadata assertions, and two walk shell-completion regressions; all other cases passed (full command package 438.848s). Added the stub and passed `go test ./c8volt/resource -race -count=1`. Added local key/BID selector completion, updated metadata assertions and passed `go test ./cmd -run 'Test(SuspendProcessInstanceMetadata|ResumeProcessInstanceMetadata|Completion|WalkProcessInstanceSelectionCompletion|BusinessIDSelectionWalk)' -race -count=1` (27.911s). The original full invocation failed; its specific defects are resolved by these targeted reruns rather than claiming a second full-suite pass.
- `make docs-content`, `go test ./docsgen -count=1`, targeted gofmt and diff checks passed. README/help/generated docs now describe uniqueness, tenant scope, public API compatibility and complete v810 child discovery. CLI selection/completion lives in its focused mode file; views own empty/failure rendering; no backend loop was added to commands/facades.
- All checks used the quickstart environment-isolation wrapper, owned mocks/HTTP servers/PTY fixtures and offline integration-support tests. Real-engine validation was not performed; concurrent backend changes remain an unverified coverage boundary. No new user story or mutation/ops BID behavior was started.
---

---
## Iteration 4 - 2026-11-01 13:49
**Work Unit**: US3 T004 — frozen cross-family BID cancellation completed; remaining US3 actions deferred to subsequent work units
**Tasks Completed**:
- [x] T004: Deliver frozen cross-family BID cancellation, complete previews, validated execution, isolated acceptance coverage and documentation.
**Tasks Remaining in Work Unit**: 0 for T004; US3 has T005–T007 remaining, and 6 feature tasks remain overall.
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- c8volt/process/api.go
- c8volt/process/client_test.go
- c8volt/process/convert.go
- c8volt/process/model.go
- c8volt/process/selection.go
- c8volt/process/selection_test.go
- c8volt/resource/client_test.go
- cmd/cancel_processinstance.go
- cmd/cancel_processinstance_business_id.go
- cmd/cancel_processinstance_business_id_test.go
- cmd/cmd_views_processinstance_dryrun.go
- cmd/process_api_stub_test.go
- cmd/run_processinstance_business_id_test.go
- docs/cli/c8volt_cancel_process-instance.md
- docs/index.md
- internal/domain/processinstance_selection.go
- internal/services/ops/orphan_purge_test.go
- internal/services/processdefinition/delete_test.go
- internal/services/processinstance/api.go
- internal/services/processinstance/business_id_search_test.go
- internal/services/processinstance/dryrun.go
- internal/services/processinstance/dryrun_test.go
- internal/services/processinstance/search_test.go
- internal/services/processinstance/selected_cancel.go
- internal/services/processinstance/selected_cancel_test.go
- internal/services/processinstance/v810/contract.go
- internal/services/processinstance/v810/service.go
- internal/services/processinstance/v88/contract.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v89/contract.go
- internal/services/processinstance/v89/service.go
- specs/361-business-id-selection/progress.md
- specs/361-business-id-selection/ralph-memory.md
- specs/361-business-id-selection/tasks.md
**Learnings**:
- Shared service selection now composes one global ancestry/root/family plan. Duplicate/sparse discovery produces 3 seeds, 2 roots, 7 members and 4 additional relatives; limit 2 produces the complete 4-member first family. Each distinct root is expanded once. Native >1000-child paging retains all 1003 members in JSON and verbose output. Complete discovery precedes confirmation/submission; failed, cyclic, unknown or late-child discovery submits nothing.
- The new public planner/executor map copied criteria, selected observations, expansion and ordered family observations. Receiving adapters validate capability before even empty/prebuilt plans; 8.8/8.9 reject v810-created plans without HTTP. Consumption checks operation, criteria, completion/bounds, tenant/IgnoreTenant, distinct keys, parent membership/order and a selected seed in every family. Public/domain copies include nested variables and exact BID provenance; JSON round trips retain evidence. Added internal tenant-context method and all affected facade/service stubs together.
- Actual command execution with real terminal stdin covers empty normal/dry-run human/JSON/keys/quiet and supported automation/auto-confirm/no-wait combinations; exact existing empty envelopes, EOF, zero-byte keys, zero prompts/mutations and request counts pass. Nonempty approval/decline/EOF uses plain configured/inherited stderr prompts with redirected stdout. A changed post-approval discovery fixture cannot add independent roots; no BID re-search occurs. Existing non-BID paging and direct-key regressions pass.
- Successful mutation/dry-run/verbose output uses configured INFO logging with empty human stdout; both plain-time and structured formats/destinations, quiet/WARN suppression and full key membership pass. Repeated/partial root errors produce one bounded ERROR without raw HTTP details; DEBUG by flag/configuration remains redacted and independent of verbose. JSON preserves partial reports/error evidence in one envelope and exit 2; output failure makes one write attempt. Accepted-but-unconfirmed failures retain submission evidence; already-terminal no-wait roots do not claim acceptance.
- Traced failure flow: service retains partial plan/reports and original error → facade maps evidence and converts error → view emits one selected-format result → configured logger emits a bounded human failure when applicable → exitAfterRenderedResult preserves the original class. Reviewed changed direct-output/error formatting boundaries; no raw diagnostic print or error-chain rendering was added. CLI lifecycle lives in cancel_processinstance_business_id.go, final formatting remains in views, and backend loops/planning remain in services.
- New test names were listed before focused execution. Owner checks passed with `go test ./internal/services/processinstance/... ./c8volt/process -run 'Test.*(BusinessIDCancel|DryRun|Search|Selection|Traversal|Children)' -count=1`. Command/legacy checks passed with `go test ./cmd ./internal/services/processinstance ./c8volt/process -run 'Test.*(BusinessIDCancel|CancelProcessInstance|ProcessInstanceDryRun)' -count=1` (command package 6.446s). Initial fixture issues (page-request metadata, unknown pd-key flag, prompt newline, automation empty INFO expectation) were corrected and corresponding checks passed.
- Full `make test PKG=./...` passed with race detection and integration tags disabled, including cmd (452.228s), process owners, resource/ops consumers and docsgen. This breadth was justified by shared planner/converter/interface changes. The full run started before the final narrow unselected-family guard and extra verbose fixture; final `go test ./internal/services/processinstance ./c8volt/process ./cmd -run '^TestBusinessIDCancel' -race -count=1` passed on those final changes (command package 127.137s). The standalone large verbose race fixture also passed.
- Targeted gofmt, `make docs-content`, and diff checks passed. README/help/generated docs describe opaque BID, version/tenant rules, full frozen scope, seed limits, one confirmation, native cascade timing and public API compatibility. All validation used the quickstart isolation wrapper and test-owned mocks/HTTP servers/PTYs; untagged integration-support packages were reviewed as offline fixtures. No live environment was inspected or tested; concurrent real-engine cascade behavior remains unverified.
---

---
## Iteration 5 - 2026-11-01 14:09
**Work Unit**: US3 T005 — frozen-member BID deletion completed; remaining US3 actions deferred to subsequent work units
**Tasks Completed**:
- [x] T005: Deliver BID deletion from frozen members with exact-key adapter mechanics, partial execution evidence, isolated acceptance coverage and documentation.
**Tasks Remaining in Work Unit**: 0 for T005; US3 has T006–T007 remaining, and 5 feature tasks remain overall.
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- c8volt/process/api.go
- c8volt/process/client_test.go
- c8volt/process/model.go
- c8volt/process/selection.go
- c8volt/process/selection_test.go
- c8volt/resource/client_test.go
- cmd/cancel_processinstance_business_id_test.go
- cmd/cmd_views_processinstance_deletion.go
- cmd/cmd_views_processinstance_dryrun.go
- cmd/delete_processinstance.go
- cmd/delete_processinstance_business_id.go
- cmd/delete_processinstance_business_id_test.go
- cmd/process_api_stub_test.go
- docs/cli/c8volt_delete_process-instance.md
- docs/index.md
- internal/domain/processinstance.go
- internal/domain/processinstance_selection.go
- internal/services/ops/orphan_purge_test.go
- internal/services/processdefinition/delete_test.go
- internal/services/processinstance/api.go
- internal/services/processinstance/bulk_test.go
- internal/services/processinstance/business_id_search_test.go
- internal/services/processinstance/dryrun_test.go
- internal/services/processinstance/selected_cancel.go
- internal/services/processinstance/selected_delete.go
- internal/services/processinstance/selected_delete_test.go
- internal/services/processinstance/v810/contract.go
- internal/services/processinstance/v810/service.go
- internal/services/processinstance/v810/service_test.go
- internal/services/processinstance/v88/contract.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v89/contract.go
- internal/services/processinstance/v89/service.go
- specs/361-business-id-selection/progress.md
- specs/361-business-id-selection/ralph-memory.md
- specs/361-business-id-selection/tasks.md
**Learnings**:
- Extended the existing selected-family planner and validator for deletion. Execution validates receiving-adapter capability, criteria, operation, tenant context, completion and graph evidence before any shortcut. It derives stable postorder from reviewed parents, runs independent families through the existing worker pool, and stops a failed family before its ancestors. Explicit-key recursive behavior still delegates to the extracted version-specific exact submission/force/retry/wait mechanics.
- Frozen scope tests preserve the 3-seed/2-root/7-member graph, duplicate/sparse discovery and seed limits. A child introduced after terminal approval is never submitted for client deletion; backend root conflicts remain explicit partial failures. Complete >1000-child JSON/verbose previews, original provenance, defensive copies, prepared-plan rejection on both older adapters and exact-key throttling retries pass. No renderer performs discovery.
- Actual command tests cover normal/dry-run/verbose/JSON/keys/quiet, supported automation/auto-confirm/no-wait combinations, full empty payloads and zero prompts/writes, real-terminal configured/inherited approval/decline/EOF with redirected stdout, full discovery before writes, original error exits, force guards/root cancellation, mandatory no-wait prerequisites and fail-fast unattempted roots. Root items remain stable; attempted members and submission evidence preserve partial families and accepted operations whose later steps failed.
- Traced failures from service responses/errors through copied facade conversion, the mode-aware view, configured logging and classified exit. Human success/dry-run/verbose output uses INFO with empty stdout, both logger formats/destinations and quiet/WARN suppression. Repeated backend failures retain one bounded ERROR, no raw HTTP details at normal/verbose/quiet, independent redacted DEBUG and one JSON envelope followed by EOF. Output-write failures make one attempt and retain the original operation exit. The final wording distinguishes no-wait submitted members from confirmed deletions and retains unconfirmed submission evidence.
- New tests were listed before execution. Targeted `go test ./cmd ./internal/services/processinstance/... ./c8volt/process -run 'Test.*(BusinessIDDelete|BusinessIDCancel|DeleteProcessInstance|ProcessInstanceDryRun)' -count=1` passed (cmd 14.824s), including legacy deletion/cancellation regressions. Initial fixture errors (native POST deletion route and bounded precondition wording) were corrected before the successful run.
- Full `make test PKG=./...` passed with race detection and integration tags disabled (cmd 459.714s); breadth was justified by shared exact-delete mechanics across all adapters, new API implementations and family worker execution. The full run began before the final human failure-wording correction; focused `go test ./cmd -run '^TestBusinessIDDelete(Logging|ChangedFamily|WaitFailures)$' -race -count=1` passed on that final change (36.691s). No second full run was needed for that narrowly verified output change.
- Targeted gofmt, `make docs-content` and diff checks passed. README/help/generated docs describe version/tenant rules, frozen members, seed limits, force/cascade timing, partial reports and public API compatibility. The command lifecycle is in its dedicated mode file; views own formatting, services own execution, and the facade remains mechanical.
- All checks used the quickstart environment-isolation wrapper and test-owned mocks/HTTP servers/PTY fixtures. Untagged integration-support packages were reviewed as offline helpers; no live environment was inspected or tested. Concurrent real-engine behavior remains unverified. No later work unit was started.
---
