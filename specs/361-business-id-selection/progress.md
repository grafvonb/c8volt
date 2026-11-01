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
