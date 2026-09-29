# Ralph Memory

Feature: 326-update-user-task-vars
Started: 2026-09-29T15:39:19Z

## Codebase Patterns

- Supported variable adapters submit scope-local writes through `CreateElementInstanceVariablesWithResponse` with `Local` explicitly set to true, `services.RetryCamundaMutation`, and `httpc.HttpStatusErr`; the primitive performs no variable lookup or waiter call.
- Adapter tests use strict generated-client doubles whose search methods panic, proving scope writes do not perform process-instance reads or confirmation.
- User-task update planning composes `GetUserTasks` with `SearchUserTaskEffectiveVariables`; it completes every selected read before returning a plan and never calls the retained scope writer dependency.
- Planning traverses tasks in stable unique input order and requested variable names in sorted order. Scope targets retain first-encounter order, group names by scope, and carry per-task name associations for shared-target fan-out.
- Execution validates and deep-copies the complete supplied plan before I/O, submits unique targets through `pool.ExecuteSlice`, fills zero-value unscheduled slots as explicit skipped outcomes, and fans shared outcomes back to stable task order.
- Confirmation reuses complete effective-variable traversal with configured backoff bounds and requires matching name, frozen scope, complete JSON, and normalized value; no-wait and no-target execution perform no confirmation reads.
- The public task facade round-trips the frozen plan, target associations, tenant evidence, and partial outcomes mechanically; JSON-like maps and slices are recursively copied in both directions.
- `task.New` remains read-compatible and returns normalized precondition errors for update calls, while root construction uses `task.NewWithVariableUpdates` with a composed user-task and variable service.
- `update user-task` is a thin plan/confirm/execute command: it passes explicit-key admin options, executes the original frozen facade plan, and attaches tenant evidence returned by planning without rediscovery.
- Variable update commands share only payload-source decoding in `cmd/update_variables_payload.go`; the existing PI planning and mutation workflow remains otherwise unchanged.
- Canonical command additions must update the enforced command-node inventory in `specs/254-cli-debt-refactor/assessment.md`; the inventory now contains 57 paths.
- Command-count changes must stay synchronized across README prose, `docsgen/main.go`, docsgen count assertions, and the CLI assessment; generated pages alone do not update those source counts.
- User-task preview JSON uses a command-owned projection: it preserves task-level category null/empty semantics but excludes internal frozen targets and duplicate tenant context; outer envelopes carry tenant evidence.
- User-task result views count confirmed/submitted, unchanged, failed, and skipped states separately; only confirmed/submitted changed tasks appear in keys-only output.
- Shared envelope and user-task key writers propagate destination failures. User-task dispatch reports those failures on stderr without attempting a second result envelope.
- Real-terminal update coverage must use `testx.NewCmdTerminalRunner` with redirected stdout and separate stderr; the command preserves the exact default-no prompt for yes/no/default/EOF and skips prompting for dry-run, no-op, automation, and auto-confirm paths.
- The command contract is equivalent across 8.8, 8.9, and 8.10; 8.7 rejects during native task planning before transport. Scope-level functional outcomes remain verbose-only, while HTTP exchange diagnostics remain DEBUG-only.

## Decisions

- `domain.ScopeVariableUpdateResponse` is intentionally scope-neutral and records `ScopeKey`, `Accepted`, `StatusCode`, and `Status`; user-task confirmation remains owned by the later composed workflow.
- Blank scope keys and empty variable maps fail with `domain.ErrValidation` before transport. Camunda 8.7 always returns the explicit unsupported capability error without transport.
- Planning treats `(ScopeKey, Name)` as the logical identity, uses complete normalized JSON values for equality, always treats truncated values as changes, and uses the task element-instance scope only for absent names.
- The composed `usertask.VariableUpdateAPI` now publishes planning and execution; execution returns partial results alongside aggregate errors and treats acceptance as historical fact rather than success.
- Fully unchanged plans return no synthetic execution items. In mixed results, unchanged tasks use `unchanged`, skipped confirmation, and false acceptance while changed tasks retain their independent scope outcomes.

## Gotchas

- Generated `CreateElementInstanceVariablesWithResponse` calls can return a nil response with no error from a faulty transport/double; map this to `domain.ErrMalformedResponse` before reading response fields.
- Effective-variable values can be invalid JSON only when marked truncated; invalid complete values are malformed responses, while truncated fragments must remain visible as received before planning a change.
- `pool.ExecuteSlice` leaves zero-value result slots for work not started after fail-fast or cancellation; execution must materialize those slots from the frozen targets before task fan-out.

## Reusable Commands

- `go test ./internal/services/variable/... -run 'Test(UpdateScopeVariables|UpdateProcessInstanceVariables|Factory)' -count=1`
- `go test -race ./internal/services/variable/... ./internal/services/processinstance/... -count=1`
- `go test ./internal/services/usertask/... -run 'TestPlanUserTaskVariableUpdates' -count=1`
- `go test -race ./internal/services/usertask/... -run 'Test(PlanUserTaskVariableUpdates|SearchUserTaskEffectiveVariables|GetUserTasks)' -count=1`
- `go test -race ./internal/services/usertask/... -run 'Test(ExecuteUserTaskVariableUpdates|PlanUserTaskVariableUpdates|SearchUserTaskEffectiveVariables|GetUserTasks)' -count=1`
- `go test ./... -run '^$' -count=1`
- `go test ./cmd -run 'Test(UpdateUserTask|CommandCapabilityForCommand_UpdateUserTask)' -count=1`
- `go test -race ./cmd -run 'TestUpdateUserTask' -count=1`
- `go test ./cmd -run 'Test(UserTaskVariableUpdate|UpdateUserTask(OutputModes|PartialFailureJSON)|UpdateUserTaskCommand|CommandCapabilityForCommand_UpdateUserTask)' -count=1`

## Do Not Repeat

- Do not reuse `UpdateProcessInstanceVariables` for task scopes: it omits `local=true` and invokes the process-instance waiter.

## Current Handoff
- Feature complete; no handoff required.
