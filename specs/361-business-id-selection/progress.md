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
