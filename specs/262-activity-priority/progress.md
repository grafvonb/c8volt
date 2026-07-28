# Ralph Progress Log

Feature: 262-activity-priority
Started: 2026-07-28 11:17:49

---
## Iteration 1 - 2026-07-28 11:19
**Work Unit**: Phase 1 Setup (Shared Discovery)
**Tasks Completed**:
- [x] T001: Reviewed current shared activity writer behavior and nested-scope tests in `toolx/logging/activity.go` and `toolx/logging/activity_test.go`
- [x] T002: Reviewed current HTTP fallback activity behavior and label tests in `internal/services/httpc/round_trippers.go` and `internal/services/httpc/round_trippers_test.go`
- [x] T003: Reviewed reusable activity sink test support in `testx/activitysink/activity_sink.go`
- [x] T004: Reviewed representative command progress emitters in `cmd/get_processinstance_paging.go`, `cmd/get_processinstance_total.go`, `cmd/get_processinstance_orphan.go`, `cmd/processinstance_mutation_progress.go`, and `cmd/ops_analyse_slow_process_instances.go`
- [x] T005: Reviewed service-level bulk and waiter activity call sites in `internal/services/processinstance/bulk.go`, `internal/services/processinstance/variables.go`, `internal/services/processinstance/waiter/waiter.go`, `internal/services/processdefinition/delete.go`, `internal/services/incident/waiter/waiter.go`, `internal/services/job/waiter/waiter.go`, `internal/services/resource/v88/service.go`, and `internal/services/resource/v89/service.go`
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- specs/262-activity-priority/tasks.md
- specs/262-activity-priority/ralph-memory.md
- specs/262-activity-priority/progress.md
**Learnings**:
- Activity is currently single-message/ref-count based; durable details were compacted into `ralph-memory.md` for Phase 2.
---
---
## Iteration 2 - 2026-07-28 11:24
**Work Unit**: Phase 2 Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T006: Added activity importance values, scoped activity metadata, and priority selection rules in `toolx/logging/activity.go`
- [x] T007: Added backward-compatible helper APIs for priority-aware start and update operations in `toolx/logging/activity.go`
- [x] T008: Added unit tests for priority ordering, equal-priority tie breaking, scope stop fallback, idempotent stop behavior, and disabled writer behavior in `toolx/logging/activity_test.go`
- [x] T009: Updated activity sink test support to record priority-aware starts and updates while preserving existing callers in `testx/activitysink/activity_sink.go`
- [x] T010: Added tests for context helper compatibility and priority-aware activity helper routing in `toolx/logging/activity_test.go`
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- toolx/logging/activity.go
- toolx/logging/activity_test.go
- testx/activitysink/activity_sink.go
- specs/262-activity-priority/tasks.md
- specs/262-activity-priority/ralph-memory.md
- specs/262-activity-priority/progress.md
**Learnings**:
- Priority-aware context helpers can stay additive; hidden lower-priority updates must not redraw the selected higher-priority scope.
---
---
## Iteration 3 - 2026-07-28 11:37
**Work Unit**: User Story 1 - Keep Workflow Progress Visible
**Tasks Completed**:
- [x] T011: Added activity hierarchy contract tests for workflow-over-wait and workflow-over-http examples in `toolx/logging/activity_test.go`
- [x] T012: Added process-instance mutation activity tests for delete and cancel progress not being overwritten by nested loads or waits in `cmd/cancel_test.go` and `cmd/delete_test.go`
- [x] T013: Added bulk run activity tests for process-instance creation progress not being overwritten by individual create or confirmation activity in `cmd/run_test.go`
- [x] T014: Added process-instance get/search progress tests for paging, totals, and orphan discovery activity stability in `cmd/get_processinstance_test.go`, `cmd/cmd_paging_totals_test.go`, and `cmd/get_test.go`
- [x] T015: Added ops slow-process activity tests proving runtime element search activity does not replace analysis progress in `cmd/ops_analyse_slow_process_instances_test.go`
- [x] T016: Added service waiter nesting tests for wait activity staying below workflow and above HTTP in `internal/services/processinstance/waiter/waiter_test.go`, `internal/services/incident/waiter/waiter_test.go`, and `internal/services/job/waiter/waiter_test.go`
- [x] T017: Promoted process-instance paging and search activity to workflow importance in `cmd/get_processinstance_paging.go`
- [x] T018: Promoted process-instance total and orphan discovery activity to workflow importance in `cmd/get_processinstance_total.go` and `cmd/get_processinstance_orphan.go`
- [x] T019: Promoted process-instance mutation preflight, frozen-scope, and execution progress to workflow importance in `cmd/processinstance_mutation_progress.go`
- [x] T020: Promoted ops slow-process discovery, analysis, and timeline progress to workflow importance in `cmd/ops_analyse_slow_process_instances.go`
- [x] T021: Marked service-level process-instance bulk create, get, cancel, delete, and variable update scopes as batch importance in `internal/services/processinstance/bulk.go` and `internal/services/processinstance/variables.go`
- [x] T022: Marked process-definition delete impact and delete execution scopes as batch importance in `internal/services/processdefinition/delete.go`
- [x] T023: Marked deployment confirmation scopes as batch importance in `internal/services/resource/v88/service.go` and `internal/services/resource/v89/service.go`
- [x] T024: Marked process-instance, incident, and job waiter scopes as wait importance in `internal/services/processinstance/waiter/waiter.go`, `internal/services/incident/waiter/waiter.go`, and `internal/services/job/waiter/waiter.go`
- [x] T025: Marked shared poller activity as wait importance in `toolx/poller/poller.go`
- [x] T026: Ran targeted US1 validation from `specs/262-activity-priority/quickstart.md`
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- cmd/cancel_test.go
- cmd/cmd_paging_totals_test.go
- cmd/delete_test.go
- cmd/get_processinstance_orphan.go
- cmd/get_processinstance_paging.go
- cmd/get_processinstance_test.go
- cmd/get_processinstance_total.go
- cmd/get_test.go
- cmd/ops_analyse_slow_process_instances.go
- cmd/ops_analyse_slow_process_instances_test.go
- cmd/processinstance_mutation_progress.go
- cmd/run_test.go
- internal/services/incident/waiter/waiter.go
- internal/services/incident/waiter/waiter_test.go
- internal/services/job/waiter/waiter.go
- internal/services/job/waiter/waiter_test.go
- internal/services/processdefinition/delete.go
- internal/services/processdefinition/delete_test.go
- internal/services/processinstance/bulk.go
- internal/services/processinstance/bulk_test.go
- internal/services/processinstance/variables.go
- internal/services/processinstance/waiter/waiter.go
- internal/services/processinstance/waiter/waiter_test.go
- internal/services/resource/v88/service.go
- internal/services/resource/v88/service_test.go
- internal/services/resource/v89/service.go
- internal/services/resource/v89/service_test.go
- toolx/logging/activity_test.go
- toolx/poller/poller.go
- toolx/poller/poller_test.go
- specs/262-activity-priority/tasks.md
- specs/262-activity-priority/ralph-memory.md
- specs/262-activity-priority/progress.md
**Learnings**:
- Workflow, batch, and wait priorities can be migrated at existing central activity emitters without changing durable output contracts; HTTP fallback remains the next story.
---
---
## Iteration 4 - 2026-07-28 11:51
**Work Unit**: User Story 2 - Preserve Useful Fallback Activity
**Tasks Completed**:
- [x] T027: Extended HTTP label table tests for all known labels in `specs/262-activity-priority/contracts/http-activity-labels.md` inside `internal/services/httpc/round_trippers_test.go`
- [x] T028: Added HTTP fallback priority tests proving HTTP activity uses fallback importance and remains available without higher scopes in `internal/services/httpc/round_trippers_test.go`
- [x] T029: Added representative simple command activity tests for cluster, tenant, resource, incident, job, variable, element-instance, and user-task operations
- [x] T030: Started HTTP transport activity with HTTP fallback importance in `internal/services/httpc/round_trippers.go`
- [x] T031: Added resource-aware fallback labels for deployment, license, resource deletion, batch operation, element-instance, variable, user-task, and tenant endpoints in `internal/services/httpc/round_trippers.go`
- [x] T032: Normalized version-prefixed and legacy Camunda paths consistently for fallback labels in `internal/services/httpc/round_trippers.go`
- [x] T033: Kept generic fallback wording for unknown HTTP methods and paths only when no known label matches in `internal/services/httpc/round_trippers.go`
- [x] T034: Ran targeted US2 validation from `specs/262-activity-priority/quickstart.md`
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- cmd/get_cluster_topology.go
- cmd/get_element_test.go
- cmd/get_incident_test.go
- cmd/get_job_test.go
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- cmd/get_processinstance_variable_filter_test.go
- cmd/get_tenant_test.go
- cmd/get_test.go
- internal/services/httpc/round_trippers.go
- internal/services/httpc/round_trippers_test.go
- specs/262-activity-priority/tasks.md
- specs/262-activity-priority/ralph-memory.md
- specs/262-activity-priority/progress.md
**Learnings**:
- HTTP fallback labels now cover the contract table centrally; command tests assert representative simple operations keep carrying HTTP-priority fallback activity through the command context.
---
