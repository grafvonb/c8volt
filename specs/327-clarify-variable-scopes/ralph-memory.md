# Ralph Memory

Feature: 327-clarify-variable-scopes
Started: 2026-10-11T09:17:02Z

## Codebase Patterns

- `userTaskVariablePlanner.addTask` already owns the fetched task identity; copy process ownership there and map it mechanically through `c8volt/task/convert.go`.
- UT planned-write formatting belongs in `cmd/cmd_views_usertask_update.go`; unchanged-requested entries intentionally keep the legacy scope syntax.
- `newUpdateUserTaskOutputServer` is shared by subprocess and real-terminal command tests, so request counters must remain mutex-protected and changes require focused `-race` validation.
- PI planned-write formatting already owns `ProcessInstanceKey`; annotate additions and changes in `formatProcessInstanceVariableUpdatePlan` without expanding planning or service models.
- Successful PI no-op rendering must be mode-aware like UT rendering: JSON emits one success envelope, keys-only emits zero bytes, and human/quiet behavior stays with `renderHumanLine`.

## Decisions

- `UserTaskVariablePlan.ProcessInstanceKey` is display-only metadata. The public field uses `json:"-"`, execution still derives targets exclusively from existing scope fields, and missing metadata retains legacy inherited labels.
- Detailed local writes use `element:<key>`; known inherited roots use `pi:<key>`; known intermediate owners use `element:<key>`. Compact bulk output deduplicates raw scope keys in first-seen order before labeling.

## Gotchas

- `userTaskVariablePlanHasChanges` depends on frozen targets, so focused view fixtures that represent mutations must populate `Targets` as well as counts and per-task categories.
- Normal confirmation previews are skipped by auto-confirm and automation; use `testx.NewCmdTerminalRunner` to prove annotated interactive previews and stderr routing.

## Reusable Commands

- `go test ./internal/services/usertask/... -run 'TestPlanUserTaskVariableUpdates|TestExecuteUserTaskVariableUpdates' -count=1`
- `go test ./c8volt/task -run 'TestVariableUpdateFacade|TestLegacyTaskConstructor' -count=1`
- `go test -race ./cmd -run 'Test(UserTaskVariableUpdate|UpdateUserTask)' -count=1`
- `make docs-content && git diff --check`

## Do Not Repeat

- Do not serialize process-owner display metadata or use it to validate or redirect mutations.
- Do not apply planned-write destination syntax to unchanged or untouched values.

## Current Handoff
- Feature complete; no handoff required.
