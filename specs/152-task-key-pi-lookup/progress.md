# Ralph Progress Log

Feature: 152-task-key-pi-lookup
Started: 2026-05-01 12:35:04

## Codebase Patterns

- `cmd/get_processinstance.go` separates keyed lookup from search mode by merging `--key` and stdin keys first, then rejecting incompatible search flags before calling `GetProcessInstances`.
- Process-instance single lookup behavior is exposed through `c8volt/process.API`; internal services satisfy `internal/services/processinstance.API` with version-specific implementations and compile-time assertions.
- Native Camunda 8.8 and 8.9 generated clients expose `GetUserTaskWithResponse(ctx, userTaskKey, ...)`, and `UserTaskResult.ProcessInstanceKey` is the owning key to convert into the existing process-instance lookup flow.
- Command tests reset package-level flag globals with `resetProcessInstanceCommandGlobals`, use `httptest` capture servers for HTTP fixtures, and use helper subprocesses for command paths that intentionally exit.

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
**Commit**: Recorded in Git history for this iteration
**Files Changed**: 
- specs/152-task-key-pi-lookup/tasks.md
- specs/152-task-key-pi-lookup/progress.md
**Learnings**:
- `--task-key` should become a selector branch before search mode and should reuse existing keyed validation/rendering after resolving the process-instance key.
- The `c8volt/task` facade currently has an empty API, so the next work unit should add the task resolution seam there instead of coupling `cmd` directly to internal generated clients.
- Focused validation passed with `go test ./cmd ./c8volt/process ./internal/services/processinstance/... -count=1`.
---
