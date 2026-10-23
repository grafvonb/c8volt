# Ralph Progress Log

Feature: 340-suspended-definition-cleanup
Started: 2026-10-23 10:10:21

---


---
## Iteration 1 - 2026-10-23 10:24
**Work Unit**: Phase 2 (Foundational), T001 — completed statistics observations and projections; phase partially complete
**Tasks Completed**:
- [x] T001: Separate suspended statistics, authoritative unfinished counts, complete capped-count traversal, public/ops mappings, adjacent regressions and structured-statistics documentation.
**Tasks Remaining in Work Unit**: T001 has none; Phase 2 still has T002, which was not selected in this iteration. Four feature tasks remain.
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- c8volt/ops/client_test.go
- c8volt/ops/convert.go
- c8volt/ops/convert_test.go
- c8volt/process/convert.go
- c8volt/process/convert_test.go
- c8volt/process/model.go
- cmd/get_processdefinition.go
- docs/cli/c8volt_get_process-definition.md
- docs/index.md
- internal/domain/processdefinition.go
- internal/services/processdefinition/v810/service.go
- internal/services/processdefinition/v810/service_test.go
- internal/services/processdefinition/v88/service_test.go
- internal/services/processdefinition/v89/service_test.go
- specs/340-suspended-definition-cleanup/progress.md
- specs/340-suspended-definition-cleanup/ralph-memory.md
- specs/340-suspended-definition-cleanup/tasks.md
**Learnings**:
- Behavioral regressions first failed for missing suspended support and for successful partial counts on stalled/disappearing cursors and incomplete empty observations; both now pass. A temporary facade compile failure from the public model's different JSON tags was corrected before validation.
- Passed with the quickstart-equivalent environment-clearing wrapper: `go test ./internal/services/processdefinition/v810 -run 'TestService_(GetProcessDefinition|SearchProcessDefinitions)' -count=1`; `go test ./internal/services/processdefinition/v88 ./internal/services/processdefinition/v89 -run 'TestService_(GetProcessDefinition|SearchProcessDefinitions)' -count=1`; `go test ./c8volt/process ./c8volt/ops -run 'ProcessDefinition|Suspended|Unfinished' -count=1`.
- Broadened to the existing command consumers because the statistics model and help metadata are shared: `go test ./cmd -run 'TestGetProcessDefinition|TestSearchProcessDefinitionsWithPagingStat|TestCommandErrorEnvelopeProcessDefinition|TestProcessDefinitionView|TestListProcessDefinitionsView' -count=1` passed. All selected dependencies were mocks, stubs, or test-owned HTTP/subprocess/terminal fixtures; inherited application/live flags were cleared.
- `make docs-content` passed after target inspection; generated changes are limited to the definition command reference and README-derived index/build provenance. Touched Go files pass gofmt; tracked and new-file whitespace checks and `git diff --check` pass.
- Covered stable 2/3/5 observations, supported zero, transitions with inconsistent display sums, exact totals, capped cursor/offset enumeration, sparse advancing pages, final empties, cursor stalls/cycles/disappearance, missing identity, query/HTTP/payload failures, old-version omission, no-stat requests, and process/ops JSON projections.
- Reviewed service → facade normalization → command handler → logger/envelope → exit. New traversal errors are bounded malformed-response errors; no raw error formatting or direct diagnostic writes were added. The existing non-watch raw-error human renderer is recorded separately in memory; passing legacy envelope checks are not claimed as diagnostic compliance. No command execution-mode declarations were added.
- No live checks ran. Real-engine behavior remains unverified. Cleanup selection/gates and human/watch suspended presentation remain pending tasks.
---

---
## Iteration 2 - 2026-10-23 10:35
**Work Unit**: Phase 2 (Foundational), T002 — completed combined-state selector and page normalization; foundational phase complete
**Tasks Completed**:
- [x] T002: Internal plural selection, shared/adapter validation, v810 union mapping, authoritative page metadata, adjacent regression tests and targeted isolated validation.
**Tasks Remaining in Work Unit**: None in Phase 2; T003, T004 and T005 remain in later stories.
**Commit**: This work-unit commit
**Files Changed**:
- internal/domain/processinstance.go
- internal/services/common/processinstance_filter.go
- internal/services/common/processinstance_filter_test.go
- internal/services/processinstance/search.go
- internal/services/processinstance/search_test.go
- internal/services/processinstance/v810/convert.go
- internal/services/processinstance/v810/convert_test.go
- internal/services/processinstance/v810/service.go
- internal/services/processinstance/v810/service_test.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v88/service_test.go
- internal/services/processinstance/v89/service.go
- internal/services/processinstance/v89/service_test.go
- specs/340-suspended-definition-cleanup/progress.md
- specs/340-suspended-definition-cleanup/ralph-memory.md
- specs/340-suspended-definition-cleanup/tasks.md
**Learnings**:
- Behavioral regressions first failed because plural selectors were ignored and sparse/missing/capped page metadata was normalized to complete; the implemented selectors and metadata classifications now pass. Invalid or ambiguous input and older-version plural requests send zero instance or incident requests.
- Passed with the quickstart-equivalent `/tmp/c8volt-340-isolated-test.py` environment-clearing wrapper: `go test ./internal/services/common ./internal/services/processinstance/... -run 'CombinedState|NormalizeProcessInstanceSearchFilter' -count=1`. The traversal/waiter/walker packages had no matching tests and are not claimed as exercised.
- Passed broader relevant controls: `go test ./internal/services/processinstance ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 ./internal/services/processinstance/v810 -run 'Search|Suspended|CombinedState|StateFilter' -count=1`; `go test ./internal/services/common ./internal/domain -run 'Filter|State' -count=1`.
- Because shared search entry points and v810 page metadata affect existing consumers, also passed `go test ./c8volt/process -run 'TestClient_(SearchProcessInstances|PlanProcessInstanceMutationPages)' -count=1` and `go test ./cmd -run 'TestGetProcessInstance(TotalOutput|PagingFlow|SuspendedSparsePaging)$' -count=1`, including the existing isolated terminal paging regression.
- Coverage includes union serialization, definition/tenant scope and ignore-tenant/cursor options, normalization/deduplication and caller slice ownership, all-member validation, older-version rejection, scalar/empty/canceled controls, direct incident-index selection, exact/final/sparse/capped metadata, stalled or missing sparse cursors, and absent/null/negative totals. Corrected the legacy capped final-empty fixture to place its observed count beyond the strict lower bound.
- Formatted touched Go files; diff and whitespace checks pass. No command declarations, facade fields, generated clients, output renderers or diagnostic boundaries changed. No user-facing filter was added, so CLI docs regeneration is not required for T002. New validation errors preserve domain classification and do not format transport errors.
- Shared permissive traversal is not claimed as safe cleanup discovery. T003 must consume the normalized single-page facts in its own existing workflow loop, reject incomplete/indeterminate scope and track actual observed counts through sparse pages; this is recorded in memory.
- No live validation ran; real-engine/exporter/snapshot behavior remains unverified. Cleanup orchestration and statistics presentation remain later tasks.
---
