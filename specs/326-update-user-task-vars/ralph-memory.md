# Ralph Memory

Feature: 326-update-user-task-vars
Started: 2026-09-29T15:39:19Z

## Codebase Patterns

- Supported variable adapters submit scope-local writes through `CreateElementInstanceVariablesWithResponse` with `Local` explicitly set to true, `services.RetryCamundaMutation`, and `httpc.HttpStatusErr`; the primitive performs no variable lookup or waiter call.
- Adapter tests use strict generated-client doubles whose search methods panic, proving scope writes do not perform process-instance reads or confirmation.

## Decisions

- `domain.ScopeVariableUpdateResponse` is intentionally scope-neutral and records `ScopeKey`, `Accepted`, `StatusCode`, and `Status`; user-task confirmation remains owned by the later composed workflow.
- Blank scope keys and empty variable maps fail with `domain.ErrValidation` before transport. Camunda 8.7 always returns the explicit unsupported capability error without transport.

## Gotchas

- Generated `CreateElementInstanceVariablesWithResponse` calls can return a nil response with no error from a faulty transport/double; map this to `domain.ErrMalformedResponse` before reading response fields.

## Reusable Commands

- `go test ./internal/services/variable/... -run 'Test(UpdateScopeVariables|UpdateProcessInstanceVariables|Factory)' -count=1`
- `go test -race ./internal/services/variable/... ./internal/services/processinstance/... -count=1`
- `go test ./... -run '^$' -count=1`

## Do Not Repeat

- Do not reuse `UpdateProcessInstanceVariables` for task scopes: it omits `local=true` and invokes the process-instance waiter.

## Current Handoff
- Continue US1 with T002: implement the complete collection-based user-task variable update planner and domain models, using the new `variable.API.UpdateScopeVariables` only as the later execution primitive.
