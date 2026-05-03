# Ralph Progress Log

Feature: 162-tasklist-v1-fallback
Started: 2026-05-03 11:10:33

## Codebase Patterns

- Tasklist V1 fallback wiring can be introduced as optional versioned service dependencies first; v88/v89 constructors already support option-based generated client replacement without changing command callers.
- Generated Tasklist V1 search requests do not expose a task-id request field; fallback builders currently use available generated search fields (`implementation`, `pageSize`, `processInstanceKey`, `tenantIds`) while response conversion maps `id`, `processInstanceKey`, and `tenantId` into `domain.UserTask`.
- v88 and v89 Tasklist V1 generated clients expose identical `TaskSearchRequest` and `TaskSearchResponse` shapes for fallback work: `TaskSearchRequest` includes `PageSize`, `Implementation`, `ProcessInstanceKey`, and `TenantIds`; `TaskSearchResponse` includes `Id`, `Implementation`, `ProcessInstanceKey`, `TaskState`, and `TenantId`.
- Existing v88 and v89 user-task service tests use package-local mock Camunda clients with `SearchUserTasksWithResponse`, `WithClientCamunda`, `requireUserTaskSearchBody`, and `newHTTPResponse`; fallback tests should mirror that pattern with Tasklist-specific mocks instead of network servers.
- Command tests use `newIPv4Server` and route assertions over `/v2/user-tasks/search` plus `/v2/process-instances/<key>` to prove request order and rendering; fallback command tests should add `/v1/tasks/search` route expectations while preserving direct keyed render comparisons.
- Current help and generated/user docs explicitly say there is no Tasklist or Operate fallback in `cmd/get_processinstance.go`, `README.md`, `docs/index.md`, and `docs/cli/c8volt_get_process-instance.md`.
- Versioned v88/v89 services can construct generated Tasklist V1 clients from `config.APIs.Tasklist.BaseURL`, while unit tests that bypass config normalization should inject Tasklist mocks with `WithClientTasklist`.
- Tasklist fallback command tests can rely on config normalization to derive `/v1/tasks/search` from the same test server root used for `/v2/user-tasks/search` and `/v2/process-instances/<key>`.

---

---
## Iteration 1 - 2026-05-03 11:12:02 CEST
**User Story**: Phase 1: Setup
**Tasks Completed**:
- [x] T001: Review Tasklist V1 generated request/response fields in `internal/clients/camunda/v88/tasklist/client.gen.go` and `internal/clients/camunda/v89/tasklist/client.gen.go`
- [x] T002: Review current task-key lookup tests and helper server routes in `internal/services/usertask/v88/service_test.go`, `internal/services/usertask/v89/service_test.go`, and `cmd/get_processinstance_test.go`
- [x] T003: Confirm current help/docs wording that mentions no Tasklist fallback in `cmd/get_processinstance.go`, `README.md`, and generated files under `docs/`
**Tasks Remaining in Story**: None - story complete
**Commit**: No commit - Git staging failed because `.git/index.lock` could not be created in this sandbox (`Operation not permitted`)
**Files Changed**:
- specs/162-tasklist-v1-fallback/tasks.md
- specs/162-tasklist-v1-fallback/progress.md
**Learnings**:
- Both Tasklist generated clients expose identical fallback search fields across v88 and v89, so next implementation can keep versioned code symmetrical.
- Existing service tests already map empty primary v2 search results to `domain.ErrNotFound` and malformed primary task results to `domain.ErrMalformedResponse`.
- Existing command tests already assert request order for primary user-task lookup and keyed process-instance rendering, which is the right place to prove fallback route ordering.
- Focused setup validation passed with `go test ./internal/services/usertask/v88 ./internal/services/usertask/v89 ./cmd -run 'GetUserTask|HasUserTasks|Help' -count=1`.
- Local Git metadata is not writable in this session, so staging and committing must be retried outside the sandbox or after permissions are fixed.
---

---
## Iteration 2 - 2026-05-03 11:16:51 CEST
**User Story**: Phase 2: Foundational
**Tasks Completed**:
- [x] T004: Add Tasklist V1 client interfaces and constructor options for v88 fallback in `internal/services/usertask/v88/contract.go` and `internal/services/usertask/v88/service.go`
- [x] T005: Add Tasklist V1 client interfaces and constructor options for v89 fallback in `internal/services/usertask/v89/contract.go` and `internal/services/usertask/v89/service.go`
- [x] T006: Add fallback Tasklist result conversion helpers in `internal/services/usertask/v88/convert.go` and `internal/services/usertask/v89/convert.go`
- [x] T007: Add fallback request builder helpers that include task key, page size, and effective tenant when available in `internal/services/usertask/v88/service.go` and `internal/services/usertask/v89/service.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- internal/services/usertask/v88/contract.go
- internal/services/usertask/v88/service.go
- internal/services/usertask/v88/convert.go
- internal/services/usertask/v89/contract.go
- internal/services/usertask/v89/service.go
- internal/services/usertask/v89/convert.go
- specs/162-tasklist-v1-fallback/tasks.md
- specs/162-tasklist-v1-fallback/progress.md
**Learnings**:
- Optional Tasklist clients let subsequent fallback behavior tests inject Tasklist-specific mocks without forcing fallback construction into the current command path.
- Tasklist search request generation has no dedicated task-id filter, so later implementation should verify the issue's intended Tasklist lookup shape before relying on `/v1/tasks/search` for exact task-key matching.
- Focused validation passed with `go test ./internal/services/usertask/v88 ./internal/services/usertask/v89 -count=1`.
---

---
## Iteration 3 - 2026-05-03 11:24:47 CEST
**User Story**: User Story 1 - Resolve Legacy Task Keys Through Fallback
**Tasks Completed**:
- [x] T008: Add v89 service test for primary miss followed by Tasklist fallback success
- [x] T009: Add v88 service test for primary miss followed by Tasklist fallback success
- [x] T010: Add command integration test for fallback-resolved process-instance output
- [x] T011: Add command integration test for fallback-resolved JSON output matching direct keyed JSON
- [x] T012: Implement v89 Tasklist fallback search after primary not-found
- [x] T013: Implement v88 Tasklist fallback search after primary not-found
- [x] T014: Wire v89 Tasklist client construction from `config.APIs.Tasklist.BaseURL`
- [x] T015: Wire v88 Tasklist client construction from `config.APIs.Tasklist.BaseURL`
- [x] T016: Verify fallback-resolved command behavior
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- internal/services/usertask/v88/service.go
- internal/services/usertask/v88/service_test.go
- internal/services/usertask/v89/service.go
- internal/services/usertask/v89/service_test.go
- cmd/get_processinstance_test.go
- specs/162-tasklist-v1-fallback/tasks.md
- specs/162-tasklist-v1-fallback/progress.md
**Learnings**:
- Primary lookup remains the first path; Tasklist fallback is gated with `errors.Is(err, domain.ErrNotFound)` and is not reached for primary success or malformed primary results.
- Fallback service tests need explicit Tasklist mocks because direct service test configs are not normalized like CLI configs.
- Validation passed with `go test ./internal/services/usertask/v88 ./internal/services/usertask/v89 ./cmd -run 'HasUserTasks|GetUserTask' -count=1` and `go test ./internal/services/usertask/v88 ./internal/services/usertask/v89 ./cmd -count=1`.
---
