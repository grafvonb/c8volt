# Ralph Progress Log

Feature: 152-task-key-pi-lookup
Started: 2026-05-01 12:35:04

## Codebase Patterns

- Public facades map internal service errors with `c8volt/ferrors.FromDomain` and convert facade options through `options.MapFacadeOptionsToCallOptions`.
- Service factories select version implementations from `cfg.App.CamundaVersion` and return `services.ErrUnknownAPIVersion` with `toolx.ImplementedCamundaVersionsString()` for unsupported configured versions.
- Versioned services expose generated-client injection through `WithClientCamunda` option helpers and assert both shared service APIs and generated client contracts at compile time.
- `cmd/get_processinstance.go` separates keyed lookup from search mode by merging `--key` and stdin keys first, then rejecting incompatible search flags before calling `GetProcessInstances`.
- Process-instance single lookup behavior is exposed through `c8volt/process.API`; internal services satisfy `internal/services/processinstance.API` with version-specific implementations and compile-time assertions.
- Native Camunda 8.8 and 8.9 generated clients expose `GetUserTaskWithResponse(ctx, userTaskKey, ...)`, and `UserTaskResult.ProcessInstanceKey` is the owning key to convert into the existing process-instance lookup flow.
- Command tests reset package-level flag globals with `resetProcessInstanceCommandGlobals`, use `httptest` capture servers for HTTP fixtures, and use helper subprocesses for command paths that intentionally exit.
- Task-key process-instance lookup should resolve through `c8volt/task.API.ResolveProcessInstanceKeyFromUserTask`, then call the existing `GetProcessInstances` keyed path so rendering stays owned by `listProcessInstancesView`.
- Native user-task services should use `common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)` for generated Camunda response handling, matching other native service implementations.
- Native user-task services let `common.RequirePayload` map missing-task 404s to `domain.ErrNotFound`, and treat an empty resolved `ProcessInstanceKey` as `domain.ErrMalformedResponse`.
- Task-key conflict validation runs after merging `--key` and stdin keys but before version support and API calls, so command tests can assert zero HTTP requests for invalid selector/search combinations.
- v8.7 user-task lookup returns a domain unsupported error directly from `internal/services/usertask/v87`, with no Tasklist or Operate client construction.

---

---
## Iteration 1 - 2026-05-01 12:36:38 CEST
**User Story**: Phase 1: Setup
**Tasks Completed**: 
- [x] T001: Review the existing keyed-vs-search flow and validation
- [x] T002: Review process-instance facade and internal service contracts
- [x] T003: Review generated native user-task client signatures for 8.8 and 8.9
- [x] T004: Review command test helpers and HTTP fixture patterns
**Tasks Remaining in Story**: None - story complete
**Commit**: No commit - sandbox blocked `.git` writes
**Files Changed**: 
- specs/152-task-key-pi-lookup/tasks.md
- specs/152-task-key-pi-lookup/progress.md
**Learnings**:
- `--task-key` should become a selector branch before search mode and should reuse existing keyed validation/rendering after resolving the process-instance key.
- The `c8volt/task` facade currently has an empty API, so the next work unit should add the task resolution seam there instead of coupling `cmd` directly to internal generated clients.
- Focused validation passed with `go test ./cmd ./c8volt/process ./internal/services/processinstance/... -count=1`.
---

---
## Iteration 2 - 2026-05-01 12:42:36 CEST
**User Story**: Phase 2: Foundational
**Tasks Completed**: 
- [x] T005: Add a user-task domain type exposing the owning process-instance key
- [x] T006: Add internal user-task service API, factory, and version assertions
- [x] T007: Add public facade task API method for resolving a process-instance key from a user task
- [x] T008: Wire the user-task service into the c8volt client construction
- [x] T009: Add test doubles or helper seams for task-key command tests
**Tasks Remaining in Story**: None - story complete
**Commit**: No commit - sandbox blocked `.git` writes
**Files Changed**: 
- internal/domain/usertask.go
- internal/services/usertask/api.go
- internal/services/usertask/factory.go
- internal/services/usertask/v87/contract.go
- internal/services/usertask/v87/service.go
- internal/services/usertask/v88/contract.go
- internal/services/usertask/v88/service.go
- internal/services/usertask/v89/contract.go
- internal/services/usertask/v89/service.go
- c8volt/task/api.go
- c8volt/task/client.go
- c8volt/client.go
- cmd/process_api_stub_test.go
- specs/152-task-key-pi-lookup/tasks.md
- specs/152-task-key-pi-lookup/progress.md
**Learnings**:
- The foundational seam can stay independent of command flag behavior by exposing `task.API.ResolveProcessInstanceKeyFromUserTask` and wiring it through the root c8volt client.
- Camunda 8.7 task-key lookup should remain native-only and unsupported; the v87 user-task service does not construct Tasklist or Operate clients.
- Broad `go test ./cmd` execution is blocked in this sandbox because `httptest` cannot bind localhost ports; `GOCACHE=/tmp/c8volt-go-build go test ./cmd -run '^$' -count=1` passed as a compile check, and `GOCACHE=/tmp/c8volt-go-build go test ./c8volt ./c8volt/task ./internal/services/usertask/... -count=1` passed.
- The sandbox also blocks writes inside `.git`, so this completed work unit could not be staged or committed from Codex.
---

---
## Iteration 3 - 2026-05-01 12:50:11 CEST
**User Story**: User Story 1 - Lookup Process Instance By User Task Key
**Tasks Completed**: 
- [x] T010: Add 8.8 native user-task service success test resolving processInstanceKey
- [x] T011: Add 8.9 native user-task service success test resolving processInstanceKey
- [x] T012: Add command success test for `get pi --task-key` on 8.8
- [x] T013: Add command success test for `get pi --task-key` on 8.9
- [x] T014: Implement 8.8 native `GetUserTaskWithResponse` resolution and conversion
- [x] T015: Implement 8.9 native `GetUserTaskWithResponse` resolution and conversion
- [x] T016: Add `--task-key` flag registration and happy-path branch before search mode
- [x] T017: Reuse existing process-instance keyed lookup and `listProcessInstancesView` rendering after task-key resolution
- [x] T018: Run focused US1 validation
**Tasks Remaining in Story**: None - story complete
**Commit**: No commit - sandbox blocked `.git` writes
**Files Changed**: 
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- internal/services/usertask/v88/convert.go
- internal/services/usertask/v88/service.go
- internal/services/usertask/v88/service_test.go
- internal/services/usertask/v89/convert.go
- internal/services/usertask/v89/service.go
- internal/services/usertask/v89/service_test.go
- specs/152-task-key-pi-lookup/tasks.md
- specs/152-task-key-pi-lookup/progress.md
**Learnings**:
- `--task-key` now resolves the owning process-instance key through the task facade and immediately reuses existing keyed process-instance lookup and rendering.
- The full requested US1 command `GOCACHE=/tmp/c8volt-go-build go test ./cmd ./c8volt/task ./internal/services/usertask/v88 ./internal/services/usertask/v89 -count=1` is blocked by an unrelated `httptest` IPv6 bind failure in `TestDeployProcessDefinitionCommand_TenantFlagOverridesEnvProfileAndConfig`.
- Passing validation: `GOCACHE=/tmp/c8volt-go-build go test ./internal/services/usertask/v88 ./internal/services/usertask/v89 -count=1`, `GOCACHE=/tmp/c8volt-go-build go test ./cmd -run 'TestGetProcessInstanceCommand_TaskKeyLookupUsesNativeUserTaskAndKeyedProcessInstance' -count=1`, and `GOCACHE=/tmp/c8volt-go-build go test ./cmd -run '^$' -count=1 && GOCACHE=/tmp/c8volt-go-build go test ./c8volt/task ./internal/services/usertask/v88 ./internal/services/usertask/v89 -count=1`.
- As in the previous iteration, `git add -A` failed with `.git/index.lock: Operation not permitted`, so no commit could be created.
---

---
## Iteration 4 - 2026-05-01 12:55:15 CEST
**User Story**: User Story 2 - Reject Unsupported Or Ambiguous Task-Key Requests
**Tasks Completed**: 
- [x] T019: Add 8.7 unsupported task-key service test
- [x] T020: Add command validation tests for `--task-key` with `--key` and stdin key input
- [x] T021: Add command validation tests for `--task-key` with process-instance search filters
- [x] T022: Add command validation tests for `--task-key` with `--total` and `--limit`
- [x] T023: Implement explicit 8.7 unsupported behavior for task-key lookup
- [x] T024: Add task-key mutual-exclusion validation for `--key`, stdin `-`, filters, `--total`, and `--limit`
- [x] T025: Ensure validation rejects conflicts before native user-task or process-instance lookup calls
- [x] T026: Run focused US2 validation
**Tasks Remaining in Story**: None - story complete
**Commit**: No commit - sandbox blocked `.git` writes
**Files Changed**: 
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- internal/services/usertask/v87/service.go
- internal/services/usertask/v87/service_test.go
- specs/152-task-key-pi-lookup/tasks.md
- specs/152-task-key-pi-lookup/progress.md
**Learnings**:
- `validatePITaskKeyMode` can reject key/stdin, search filters, `--total`, and `--limit` after stdin key merging while still preventing all user-task and process-instance HTTP calls.
- `GOCACHE=/tmp/c8volt-go-build go test ./internal/services/usertask/v87 -count=1` and `GOCACHE=/tmp/c8volt-go-build go test ./cmd -run 'TestGetProcessInstanceCommand_(RejectsTaskKeyConflictsBeforeLookup|TaskKeyUnsupportedOnV87)' -count=1` passed.
- The requested `GOCACHE=/tmp/c8volt-go-build go test ./cmd ./c8volt/task ./internal/services/usertask/... -count=1` remains blocked by the unrelated sandbox `httptest` IPv6 bind failure in `TestDeployProcessDefinitionCommand_TenantFlagOverridesEnvProfileAndConfig`; `GOCACHE=/tmp/c8volt-go-build go test ./cmd -run '^$' -count=1 && GOCACHE=/tmp/c8volt-go-build go test ./c8volt/task ./internal/services/usertask/... -count=1` passed.
- `git add -A` failed with `.git/index.lock: Operation not permitted`, so this completed work unit could not be committed from Codex.
---

---
## Iteration 5 - 2026-05-01 13:02:12 CEST
**User Story**: User Story 3 - Preserve Existing Single Lookup Rendering Options
**Tasks Completed**: 
- [x] T027: Add command test proving `get pi --task-key=<task-key> --json` matches direct keyed JSON shape
- [x] T028: Add command tests for `--task-key` with valid single lookup render flags such as `--with-age` and `--keys-only`
- [x] T029: Add missing user task and missing processInstanceKey tests
- [x] T030: Add command test preserving process-instance not-found behavior after task-key resolution
- [x] T031: Normalize missing task and missing processInstanceKey errors through repository error conventions
- [x] T032: Preserve existing direct keyed render option flow after task-key resolution
- [x] T033: Run focused US3 validation
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**: 
- cmd/get_processinstance_test.go
- internal/services/usertask/v88/service.go
- internal/services/usertask/v88/service_test.go
- internal/services/usertask/v89/service.go
- internal/services/usertask/v89/service_test.go
- specs/152-task-key-pi-lookup/tasks.md
- specs/152-task-key-pi-lookup/progress.md
**Learnings**:
- Task-key lookup continues to use the existing `GetProcessInstances` plus `listProcessInstancesView` flow, so `--json`, `--with-age`, `--keys-only`, and resolved process-instance not-found behavior match direct keyed lookup.
- The requested `GOCACHE=/tmp/c8volt-go-build go test ./cmd ./internal/services/usertask/v88 ./internal/services/usertask/v89 -count=1` remains blocked by the unrelated sandbox `httptest` IPv6 bind failure in `TestDeployProcessDefinitionCommand_TenantFlagOverridesEnvProfileAndConfig`.
- Passing validation: `GOCACHE=/tmp/c8volt-go-build go test ./cmd -run 'TestGetProcessInstanceCommand_TaskKey(JSONMatchesDirectKeyedJSON|PreservesSingleLookupRenderFlags|PreservesResolvedProcessInstanceNotFound|LookupUsesNativeUserTaskAndKeyedProcessInstance)' -count=1`, `GOCACHE=/tmp/c8volt-go-build go test ./cmd -run '^$' -count=1`, and `GOCACHE=/tmp/c8volt-go-build go test ./internal/services/usertask/v88 ./internal/services/usertask/v89 -count=1`.
---
