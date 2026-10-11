# Ralph Memory

Feature: 327-clarify-variable-scopes
Started: 2026-10-11T09:17:02Z

## Codebase Patterns

- `userTaskVariablePlanner.addTask` already owns the fetched task identity; copy process ownership there and map it mechanically through `c8volt/task/convert.go`.
- UT planned-write formatting belongs in `cmd/cmd_views_usertask_update.go`; unchanged-requested entries intentionally keep the legacy scope syntax.
- `newUpdateUserTaskOutputServer` is shared by subprocess and real-terminal command tests, so request counters must remain mutex-protected and changes require focused `-race` validation.

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
- T002 / US2 is next: annotate process-instance additions and changes with the selected `pi:<ProcessInstanceKey>`, retain the completed UT behavior, extend PI command and real-terminal coverage, update PI help/README, and regenerate docs.
