# Ralph Memory

Feature: 326-update-user-task-vars
Started: 2026-09-29T15:39:19Z

## Codebase Patterns

- Supported variable adapters submit scope-local writes through `CreateElementInstanceVariablesWithResponse` with `Local` explicitly set to true, `services.RetryCamundaMutation`, and `httpc.HttpStatusErr`; the primitive performs no variable lookup or waiter call.
- Adapter tests use strict generated-client doubles whose search methods panic, proving scope writes do not perform process-instance reads or confirmation.
- User-task update planning composes `GetUserTasks` with `SearchUserTaskEffectiveVariables`; it completes every selected read before returning a plan and never calls the retained scope writer dependency.
- Planning traverses tasks in stable unique input order and requested variable names in sorted order. Scope targets retain first-encounter order, group names by scope, and carry per-task name associations for shared-target fan-out.

## Decisions

- `domain.ScopeVariableUpdateResponse` is intentionally scope-neutral and records `ScopeKey`, `Accepted`, `StatusCode`, and `Status`; user-task confirmation remains owned by the later composed workflow.
- Blank scope keys and empty variable maps fail with `domain.ErrValidation` before transport. Camunda 8.7 always returns the explicit unsupported capability error without transport.
- Planning treats `(ScopeKey, Name)` as the logical identity, uses complete normalized JSON values for equality, always treats truncated values as changes, and uses the task element-instance scope only for absent names.
- The composed `usertask.VariableUpdateAPI` intentionally publishes only planning in T002; add execution only when T003 implements validation, submission, confirmation, and partial outcomes together.

## Gotchas

- Generated `CreateElementInstanceVariablesWithResponse` calls can return a nil response with no error from a faulty transport/double; map this to `domain.ErrMalformedResponse` before reading response fields.
- Effective-variable values can be invalid JSON only when marked truncated; invalid complete values are malformed responses, while truncated fragments must remain visible as received before planning a change.

## Reusable Commands

- `go test ./internal/services/variable/... -run 'Test(UpdateScopeVariables|UpdateProcessInstanceVariables|Factory)' -count=1`
- `go test -race ./internal/services/variable/... ./internal/services/processinstance/... -count=1`
- `go test ./internal/services/usertask/... -run 'TestPlanUserTaskVariableUpdates' -count=1`
- `go test -race ./internal/services/usertask/... -run 'Test(PlanUserTaskVariableUpdates|SearchUserTaskEffectiveVariables|GetUserTasks)' -count=1`
- `go test ./... -run '^$' -count=1`

## Do Not Repeat

- Do not reuse `UpdateProcessInstanceVariables` for task scopes: it omits `local=true` and invokes the process-instance waiter.

## Current Handoff
- Continue US1 with T003: extend the composed service with validated frozen-plan execution, shared scope outcome fan-out, fail-fast worker behavior, and scope-aware confirmation in `update.go` and `update_wait.go`.
