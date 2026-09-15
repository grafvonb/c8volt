# Ralph Memory

Feature: 309-user-task-variables
Started: 2026-09-15T08:04:53Z

## Codebase Patterns

- The implemented #308 base command is `cmd/get_usertask.go`; it owns Cobra setup, key/search validation, top-level dispatch, aliases `user-tasks`/`ut`/`uts`, and delegates paging to `searchUserTasksWithPaging`.
- Active feature selection is duplicated intentionally: `.specify/feature.json` names `specs/309-user-task-variables`, while `AGENTS.md` names its `plan.md` under the active Speckit marker.

## Decisions

- No setup divergence blocks implementation. Feature #309 extends the existing #308 command; its current help text intentionally excludes variables until the later user-story tasks update behavior and documentation.
- Preserve the Go 1.26/toolchain 1.26.2 dependency set; this feature adds no dependency.

## Gotchas

- `gh issue view 309` cannot currently validate the remote issue because the configured GitHub credentials return HTTP 401. The committed feature artifacts provide the implementation contract for this iteration.

## Reusable Commands

- Base command regression: `go test ./cmd -run '^TestGetUserTask' -count=1`

## Do Not Repeat

- Do not treat the base command's current variable exclusion as a conflict; removing it is explicitly deferred to T031 after the behavior exists.

## Current Handoff

- Continue with T002 in Phase 2, adding the version-neutral enriched user-task and offset variable-page domain records without starting a user story.
